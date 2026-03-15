package migrate

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
	"gopkg.in/yaml.v3"
)

var fastlaneSingleImageFiles = map[string]struct{}{
	"icon":           {},
	"featureGraphic": {},
	"promoGraphic":   {},
	"tvBanner":       {},
}

var fastlaneScreenshotDirs = map[string]struct{}{
	"phoneScreenshots":     {},
	"sevenInchScreenshots": {},
	"tenInchScreenshots":   {},
	"tvScreenshots":        {},
	"wearScreenshots":      {},
}

type Deps struct {
	Stdout io.Writer
	Stderr io.Writer
}

type importOptions struct {
	FromDir            string
	Dir                string
	Track              string
	VersionCode        int64
	PackageName        string
	WriteProjectConfig bool
}

type importResult struct {
	SourceDir        string            `json:"sourceDir"`
	Dir              string            `json:"dir"`
	Track            string            `json:"track"`
	VersionCode      int64             `json:"versionCode,omitempty"`
	Status           string            `json:"status"`
	ListingLocales   int               `json:"listingLocales"`
	ChangelogLocales int               `json:"changelogLocales"`
	ImagesCopied     int               `json:"imagesCopied"`
	Files            map[string]string `json:"files,omitempty"`
}

type localeImport struct {
	Locale                string
	Title                 string
	ShortDescription      string
	FullDescription       string
	ImageFiles            []imageCopy
	VersionedOrDefaultLog string
}

type imageCopy struct {
	Source string
	Target string
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "migrate",
		ShortHelp: "Import or transform metadata from other tool layouts",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newFastlaneCommand(deps),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return deps
}

func newFastlaneCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "fastlane",
		ShortHelp: "Migrate Fastlane metadata into gpc workspace layout",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newFastlaneImportCommand(deps),
		},
	}
}

func newFastlaneImportCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts importOptions
	fs.StringVar(&opts.FromDir, "from-dir", "", "Fastlane metadata root (`fastlane`, `fastlane/metadata`, or `fastlane/metadata/android`)")
	fs.StringVar(&opts.Dir, "dir", "", "Target gpc workspace directory")
	fs.StringVar(&opts.Track, "track", "production", "Track name for imported changelogs")
	fs.Int64Var(&opts.VersionCode, "version-code", 0, "Preferred Fastlane changelog version code (falls back to default.txt)")
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name to persist into .gpc.yaml")
	fs.BoolVar(&opts.WriteProjectConfig, "write-project-config", false, "Write .gpc.yaml with local listing/changelog defaults")

	return &ffcli.Command{
		Name:      "import",
		ShortHelp: "Import Fastlane metadata into the local gpc listing/changelog layout",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(_ context.Context, _ []string) error {
			resolved, err := validateImportOptions(opts)
			if err != nil {
				return err
			}
			locales, err := scanFastlaneAndroidDir(resolved.FromDir, resolved.Track, resolved.VersionCode)
			if err != nil {
				return err
			}
			result, err := runImport(resolved, locales)
			if err != nil {
				return err
			}
			return shared.WriteJSON(deps.Stdout, result)
		},
	}
}

func validateImportOptions(opts importOptions) (importOptions, error) {
	opts.FromDir = strings.TrimSpace(opts.FromDir)
	if opts.FromDir == "" {
		return importOptions{}, shared.UsageErrorf("--from-dir is required")
	}
	opts.Dir = strings.TrimSpace(opts.Dir)
	if opts.Dir == "" {
		return importOptions{}, shared.UsageErrorf("--dir is required")
	}
	opts.Track = strings.TrimSpace(opts.Track)
	if opts.Track == "" {
		opts.Track = "production"
	}
	if opts.VersionCode < 0 {
		return importOptions{}, shared.UsageErrorf("--version-code must be zero or greater")
	}
	opts.PackageName = strings.TrimSpace(opts.PackageName)
	if opts.WriteProjectConfig && opts.PackageName == "" {
		return importOptions{}, shared.UsageErrorf("--package-name is required when --write-project-config is set")
	}
	resolvedSource, err := resolveFastlaneAndroidDir(opts.FromDir)
	if err != nil {
		return importOptions{}, err
	}
	opts.FromDir = resolvedSource
	return opts, nil
}

func resolveFastlaneAndroidDir(root string) (string, error) {
	root = strings.TrimSpace(root)
	candidates := []string{
		root,
		filepath.Join(root, "android"),
		filepath.Join(root, "metadata", "android"),
	}
	for _, candidate := range candidates {
		if isFastlaneAndroidDir(candidate) {
			return candidate, nil
		}
	}
	return "", shared.UsageErrorf("--from-dir must point to a Fastlane metadata directory containing locale folders")
}

func isFastlaneAndroidDir(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		localeDir := filepath.Join(path, entry.Name())
		if fileExists(filepath.Join(localeDir, "title.txt")) ||
			fileExists(filepath.Join(localeDir, "short_description.txt")) ||
			fileExists(filepath.Join(localeDir, "full_description.txt")) ||
			dirExists(filepath.Join(localeDir, "changelogs")) ||
			dirExists(filepath.Join(localeDir, "images")) {
			return true
		}
	}
	return false
}

func scanFastlaneAndroidDir(root, track string, versionCode int64) ([]localeImport, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read Fastlane metadata directory: %w", err)
	}
	locales := make([]localeImport, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		localePath := filepath.Join(root, entry.Name())
		locale, err := scanFastlaneLocale(entry.Name(), localePath, track, versionCode)
		if err != nil {
			return nil, err
		}
		locales = append(locales, locale)
	}
	sort.Slice(locales, func(i, j int) bool { return locales[i].Locale < locales[j].Locale })
	if len(locales) == 0 {
		return nil, fmt.Errorf("no Fastlane locale directories found in %s", root)
	}
	return locales, nil
}

func scanFastlaneLocale(locale, dir, track string, versionCode int64) (localeImport, error) {
	title, err := readRequiredFile(filepath.Join(dir, "title.txt"))
	if err != nil {
		return localeImport{}, fmt.Errorf("%s: %w", locale, err)
	}
	shortDesc, err := readRequiredFile(filepath.Join(dir, "short_description.txt"))
	if err != nil {
		return localeImport{}, fmt.Errorf("%s: %w", locale, err)
	}
	fullDesc, err := readRequiredFile(filepath.Join(dir, "full_description.txt"))
	if err != nil {
		return localeImport{}, fmt.Errorf("%s: %w", locale, err)
	}
	images, err := collectFastlaneImages(locale, filepath.Join(dir, "images"))
	if err != nil {
		return localeImport{}, err
	}

	changelog := ""
	changelogDir := filepath.Join(dir, "changelogs")
	if versionCode > 0 {
		versionPath := filepath.Join(changelogDir, strconv.FormatInt(versionCode, 10)+".txt")
		if fileExists(versionPath) {
			changelog, err = readOptionalFile(versionPath)
			if err != nil {
				return localeImport{}, fmt.Errorf("%s: %w", locale, err)
			}
		}
	}
	if changelog == "" {
		defaultPath := filepath.Join(changelogDir, "default.txt")
		if fileExists(defaultPath) {
			changelog, err = readOptionalFile(defaultPath)
			if err != nil {
				return localeImport{}, fmt.Errorf("%s: %w", locale, err)
			}
		}
	}

	return localeImport{
		Locale:                locale,
		Title:                 title,
		ShortDescription:      shortDesc,
		FullDescription:       fullDesc,
		ImageFiles:            images,
		VersionedOrDefaultLog: changelog,
	}, nil
}

func collectFastlaneImages(locale, dir string) ([]imageCopy, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("%s: read images directory: %w", locale, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s: images must be a directory", locale)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("%s: read images directory: %w", locale, err)
	}
	copies := make([]imageCopy, 0)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		source := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if _, ok := fastlaneScreenshotDirs[entry.Name()]; !ok {
				continue
			}
			files, err := os.ReadDir(source)
			if err != nil {
				return nil, fmt.Errorf("%s: read image directory %q: %w", locale, entry.Name(), err)
			}
			for _, imageFile := range files {
				if imageFile.IsDir() || strings.HasPrefix(imageFile.Name(), ".") {
					continue
				}
				copies = append(copies, imageCopy{
					Source: filepath.Join(source, imageFile.Name()),
					Target: filepath.Join("images", entry.Name(), imageFile.Name()),
				})
			}
			continue
		}
		imageType := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if _, ok := fastlaneSingleImageFiles[imageType]; !ok {
			continue
		}
		copies = append(copies, imageCopy{
			Source: source,
			Target: filepath.Join("images", entry.Name()),
		})
	}
	sort.Slice(copies, func(i, j int) bool { return copies[i].Target < copies[j].Target })
	return copies, nil
}

func runImport(opts importOptions, locales []localeImport) (importResult, error) {
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return importResult{}, fmt.Errorf("create target directory: %w", err)
	}

	result := importResult{
		SourceDir: opts.FromDir,
		Dir:       opts.Dir,
		Track:     opts.Track,
		Status:    "imported",
		Files:     map[string]string{},
	}
	if opts.VersionCode > 0 {
		result.VersionCode = opts.VersionCode
	}

	for _, locale := range locales {
		localeDir := filepath.Join(opts.Dir, "listing", locale.Locale)
		if err := writeBytes(filepath.Join(localeDir, "title.txt"), []byte(locale.Title)); err != nil {
			return importResult{}, err
		}
		if err := writeBytes(filepath.Join(localeDir, "short-description.txt"), []byte(locale.ShortDescription)); err != nil {
			return importResult{}, err
		}
		if err := writeBytes(filepath.Join(localeDir, "full-description.txt"), []byte(locale.FullDescription)); err != nil {
			return importResult{}, err
		}
		result.ListingLocales++

		for _, image := range locale.ImageFiles {
			target := filepath.Join(localeDir, image.Target)
			if err := copyFile(image.Source, target); err != nil {
				return importResult{}, err
			}
			result.ImagesCopied++
		}

		if strings.TrimSpace(locale.VersionedOrDefaultLog) != "" {
			target := filepath.Join(opts.Dir, "changelog", opts.Track, locale.Locale+".txt")
			if err := writeBytes(target, []byte(locale.VersionedOrDefaultLog)); err != nil {
				return importResult{}, err
			}
			result.ChangelogLocales++
		}
	}

	if opts.WriteProjectConfig {
		cfgPath := filepath.Join(opts.Dir, ".gpc.yaml")
		if err := writeProjectConfig(cfgPath, opts.PackageName); err != nil {
			return importResult{}, err
		}
		result.Files["projectConfig"] = cfgPath
	}

	if len(result.Files) == 0 {
		result.Files = nil
	}
	return result, nil
}

func writeProjectConfig(path, packageName string) error {
	cfg := map[string]any{
		"package-name":  packageName,
		"listing-dir":   "./listing",
		"changelog-dir": "./changelog",
	}
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal project config: %w", err)
	}
	return writeBytes(path, raw)
}

func readRequiredFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("missing required file %s", filepath.Base(path))
		}
		return "", err
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", fmt.Errorf("required file %s is empty", filepath.Base(path))
	}
	return text, nil
}

func readOptionalFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func copyFile(source, target string) error {
	raw, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read %s: %w", source, err)
	}
	return writeBytes(target, raw)
}

func writeBytes(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
