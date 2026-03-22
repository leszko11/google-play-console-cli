package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
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
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
}

type importOptions struct {
	FromDir            string
	Dir                string
	Track              string
	VersionCode        int64
	PackageName        string
	WriteProjectConfig bool
}

type diffOptions struct {
	FromDir     string
	PackageName string
	Track       string
	VersionCode int64
	Output      string
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

type diffResult struct {
	SourceDir                string   `json:"sourceDir"`
	PackageName              string   `json:"packageName"`
	Track                    string   `json:"track"`
	VersionCode              int64    `json:"versionCode,omitempty"`
	HasDiff                  bool     `json:"hasDiff"`
	TrackFound               bool     `json:"trackFound"`
	LiveReleaseName          string   `json:"liveReleaseName,omitempty"`
	ListingLocaleCount       int      `json:"listingLocaleCount"`
	RemoteListingLocaleCount int      `json:"remoteListingLocaleCount"`
	ReleaseNotesLocaleCount  int      `json:"releaseNotesLocaleCount"`
	ChangeCount              int      `json:"changeCount"`
	Changes                  []change `json:"changes"`
}

type change struct {
	Scope   string `json:"scope"`
	Target  string `json:"target"`
	Field   string `json:"field,omitempty"`
	Action  string `json:"action"`
	Live    any    `json:"live,omitempty"`
	Desired any    `json:"desired,omitempty"`
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

type localeDiff struct {
	Locale  string
	Listing gpc.ListingUpdate
	Images  map[string][]string
}

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ListListings(ctx context.Context, packageName, editID string) ([]gpc.ListingInfo, error)
	ListImages(ctx context.Context, packageName, editID, language, imageType string) ([]gpc.ImageInfo, error)
	ListTracks(ctx context.Context, packageName, editID string) ([]gpc.TrackInfo, error)
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
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewClient == nil {
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (Client, error) {
			return gpc.NewClient(ctx, creds)
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
	}
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
			newFastlaneDiffCommand(deps),
			newFastlaneImportCommand(deps),
		},
	}
}

func newFastlaneDiffCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts diffOptions
	fs.StringVar(&opts.FromDir, "from-dir", "", "Fastlane metadata root (`fastlane`, `fastlane/metadata`, or `fastlane/metadata/android`)")
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Track, "track", "production", "Track name for live release note comparison")
	fs.Int64Var(&opts.VersionCode, "version-code", 0, "Preferred Fastlane changelog version code (falls back to default.txt)")
	fs.StringVar(&opts.Output, "output", "", "Output format: json, table, markdown, yaml")

	return &ffcli.Command{
		Name:      "diff",
		ShortHelp: "Compare Fastlane metadata against live Play listing and changelog state",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			resolved, err := validateDiffOptions(opts)
			if err != nil {
				return err
			}
			locales, err := scanFastlaneAndroidDir(resolved.FromDir, resolved.Track, resolved.VersionCode)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
			})
			if err != nil {
				return err
			}
			defer cancel()

			result, err := runFastlaneDiff(ctx, requestCtx, client, resolved, locales)
			if err != nil {
				return err
			}
			return writeDiffResult(deps.Stdout, resolved.Output, result)
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

func validateDiffOptions(opts diffOptions) (diffOptions, error) {
	opts.FromDir = strings.TrimSpace(opts.FromDir)
	if opts.FromDir == "" {
		return diffOptions{}, shared.UsageErrorf("--from-dir is required")
	}
	var err error
	opts.PackageName, err = shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return diffOptions{}, err
	}
	opts.Track, err = shared.ResolveDefaultTrack(opts.Track)
	if err != nil {
		return diffOptions{}, err
	}
	opts.Track = strings.TrimSpace(opts.Track)
	if opts.Track == "" {
		opts.Track = "production"
	}
	if opts.VersionCode < 0 {
		return diffOptions{}, shared.UsageErrorf("--version-code must be zero or greater")
	}
	if _, err := resolveOutput(opts.Output); err != nil {
		return diffOptions{}, err
	}
	resolvedSource, err := resolveFastlaneAndroidDir(opts.FromDir)
	if err != nil {
		return diffOptions{}, err
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

func runFastlaneDiff(parentCtx, requestCtx context.Context, client Client, opts diffOptions, locales []localeImport) (diffResult, error) {
	desiredLocales := toLocaleDiffs(locales)
	desiredNotes := toReleaseNotes(locales)
	result := diffResult{
		SourceDir:               opts.FromDir,
		PackageName:             opts.PackageName,
		Track:                   opts.Track,
		ListingLocaleCount:      len(desiredLocales),
		ReleaseNotesLocaleCount: len(desiredNotes),
		Changes:                 []change{},
	}
	if opts.VersionCode > 0 {
		result.VersionCode = opts.VersionCode
	}

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return result, fmt.Errorf("failed to create edit: %w", err)
	}
	defer func() {
		_ = deleteEdit(parentCtx, client, opts.PackageName, edit.ID)
	}()

	remoteListings, err := client.ListListings(requestCtx, opts.PackageName, edit.ID)
	if err != nil {
		return result, fmt.Errorf("failed to list live listings: %w", err)
	}
	result.RemoteListingLocaleCount = len(remoteListings)

	remoteByLocale := make(map[string]gpc.ListingInfo, len(remoteListings))
	for _, listing := range remoteListings {
		remoteByLocale[strings.TrimSpace(listing.Language)] = listing
	}

	changes := make([]change, 0)
	for _, locale := range desiredLocales {
		remote, exists := remoteByLocale[locale.Locale]
		if !exists {
			changes = append(changes, change{
				Scope:   "listing",
				Target:  locale.Locale,
				Action:  "create_locale",
				Desired: locale.Listing,
			})
			delete(remoteByLocale, locale.Locale)
			continue
		}

		localeChanges, err := compareListingLocale(requestCtx, client, opts.PackageName, edit.ID, locale, remote)
		if err != nil {
			return result, err
		}
		changes = append(changes, localeChanges...)
		delete(remoteByLocale, locale.Locale)
	}

	remainingLocales := make([]string, 0, len(remoteByLocale))
	for locale := range remoteByLocale {
		remainingLocales = append(remainingLocales, locale)
	}
	sort.Strings(remainingLocales)
	for _, locale := range remainingLocales {
		changes = append(changes, change{
			Scope:  "listing",
			Target: locale,
			Action: "remote_only_locale",
			Live:   remoteByLocale[locale],
		})
	}

	tracks, err := client.ListTracks(requestCtx, opts.PackageName, edit.ID)
	if err != nil {
		return result, fmt.Errorf("failed to list live tracks: %w", err)
	}
	for _, track := range tracks {
		if strings.TrimSpace(track.Name) != opts.Track {
			continue
		}
		result.TrackFound = true
		if len(track.Releases) > 0 {
			result.LiveReleaseName = track.Releases[0].Name
		}
		if len(desiredNotes) > 0 {
			changes = append(changes, compareReleaseNotes(opts.Track, track.Releases, desiredNotes)...)
		}
		break
	}
	if !result.TrackFound && len(desiredNotes) > 0 {
		changes = append(changes, change{
			Scope:   "track",
			Target:  opts.Track,
			Field:   "releaseNotes",
			Action:  "create",
			Desired: sliceOrNil(desiredNotes),
		})
	}

	sortChanges(changes)
	result.Changes = changes
	result.ChangeCount = len(changes)
	result.HasDiff = len(changes) > 0
	return result, nil
}

func toLocaleDiffs(locales []localeImport) []localeDiff {
	out := make([]localeDiff, 0, len(locales))
	for _, locale := range locales {
		images := map[string][]string{}
		for _, image := range locale.ImageFiles {
			normalized := filepath.ToSlash(image.Target)
			parts := strings.Split(normalized, "/")
			if len(parts) < 2 || parts[0] != "images" {
				continue
			}
			if len(parts) == 2 {
				imageType := strings.TrimSuffix(parts[1], filepath.Ext(parts[1]))
				images[imageType] = []string{image.Source}
				continue
			}
			imageType := parts[1]
			images[imageType] = append(images[imageType], image.Source)
		}
		for imageType := range images {
			sort.Strings(images[imageType])
		}
		out = append(out, localeDiff{
			Locale: locale.Locale,
			Listing: gpc.ListingUpdate{
				Title:            locale.Title,
				ShortDescription: locale.ShortDescription,
				FullDescription:  locale.FullDescription,
			},
			Images: images,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Locale < out[j].Locale })
	return out
}

func toReleaseNotes(locales []localeImport) []gpc.LocalizedText {
	notes := make([]gpc.LocalizedText, 0, len(locales))
	for _, locale := range locales {
		text := strings.TrimSpace(locale.VersionedOrDefaultLog)
		if text == "" {
			continue
		}
		notes = append(notes, gpc.LocalizedText{
			Language: locale.Locale,
			Text:     text,
		})
	}
	return normalizeNotes(notes)
}

func compareListingLocale(ctx context.Context, client Client, packageName, editID string, locale localeDiff, remote gpc.ListingInfo) ([]change, error) {
	changes := make([]change, 0)
	if remote.Title != locale.Listing.Title {
		changes = append(changes, change{
			Scope:   "listing",
			Target:  locale.Locale,
			Field:   "title",
			Action:  "update",
			Live:    remote.Title,
			Desired: locale.Listing.Title,
		})
	}
	if remote.ShortDescription != locale.Listing.ShortDescription {
		changes = append(changes, change{
			Scope:   "listing",
			Target:  locale.Locale,
			Field:   "shortDescription",
			Action:  "update",
			Live:    remote.ShortDescription,
			Desired: locale.Listing.ShortDescription,
		})
	}
	if remote.FullDescription != locale.Listing.FullDescription {
		changes = append(changes, change{
			Scope:   "listing",
			Target:  locale.Locale,
			Field:   "fullDescription",
			Action:  "update",
			Live:    remote.FullDescription,
			Desired: locale.Listing.FullDescription,
		})
	}

	imageTypes := sortedImageTypes(locale.Images)
	for _, imageType := range imageTypes {
		remoteImages, err := client.ListImages(ctx, packageName, editID, locale.Locale, imageType)
		if err != nil {
			return nil, fmt.Errorf("failed to list live images for %q/%q: %w", locale.Locale, imageType, err)
		}
		liveHashes := normalizeRemoteImages(remoteImages)
		desiredHashes, err := hashFiles(locale.Images[imageType])
		if err != nil {
			return nil, fmt.Errorf("failed to hash Fastlane images for %q/%q: %w", locale.Locale, imageType, err)
		}
		if !reflect.DeepEqual(liveHashes, desiredHashes) {
			changes = append(changes, change{
				Scope:   "images",
				Target:  locale.Locale,
				Field:   imageType,
				Action:  "replace",
				Live:    liveHashes,
				Desired: desiredHashes,
			})
		}
	}

	return changes, nil
}

func compareReleaseNotes(trackName string, releases []gpc.TrackReleaseInfo, desiredNotes []gpc.LocalizedText) []change {
	if len(releases) == 0 {
		return []change{{
			Scope:   "track",
			Target:  trackName,
			Field:   "releaseNotes",
			Action:  "create",
			Desired: sliceOrNil(desiredNotes),
		}}
	}

	liveNotes := normalizeNotes(releases[0].ReleaseNotes)
	if reflect.DeepEqual(liveNotes, desiredNotes) {
		return nil
	}
	return []change{{
		Scope:   "track",
		Target:  trackName,
		Field:   "releaseNotes",
		Action:  "update",
		Live:    sliceOrNil(liveNotes),
		Desired: sliceOrNil(desiredNotes),
	}}
}

func deleteEdit(ctx context.Context, client Client, packageName, editID string) error {
	cleanupCtx, cleanupCancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
	defer cleanupCancel()
	if err := client.DeleteEdit(cleanupCtx, packageName, editID); err != nil {
		return fmt.Errorf("failed to clean up edit: %w", err)
	}
	return nil
}

func writeDiffResult(out io.Writer, output string, payload diffResult) error {
	format, err := resolveOutput(output)
	if err != nil {
		return err
	}
	switch format {
	case "json":
		return shared.WriteJSON(out, payload)
	case "yaml":
		return shared.WriteYAML(out, payload)
	case "table":
		return writeDiffTable(out, payload)
	case "markdown":
		return writeDiffMarkdown(out, payload)
	default:
		return shared.UsageErrorf("unsupported output format %q", format)
	}
}

func writeDiffTable(out io.Writer, result diffResult) error {
	status := "no-diff"
	if result.HasDiff {
		status = "diff"
	}
	lines := []struct {
		key   string
		value string
	}{
		{"STATUS", status},
		{"PACKAGE", result.PackageName},
		{"SOURCE", result.SourceDir},
		{"TRACK", result.Track},
		{"TRACK_FOUND", strconv.FormatBool(result.TrackFound)},
		{"CHANGE_COUNT", strconv.Itoa(result.ChangeCount)},
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(out, "%s\t%s\n", line.key, line.value); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(out, "SCOPE\tTARGET\tFIELD\tACTION\tLIVE\tDESIRED"); err != nil {
		return err
	}
	for _, entry := range result.Changes {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\n", entry.Scope, entry.Target, entry.Field, entry.Action, formatValue(entry.Live), formatValue(entry.Desired)); err != nil {
			return err
		}
	}
	return nil
}

func writeDiffMarkdown(out io.Writer, result diffResult) error {
	status := "no-diff"
	if result.HasDiff {
		status = "diff"
	}
	if err := shared.WriteMarkdownTable(out, []string{"field", "value"}, [][]string{
		{"status", status},
		{"package", result.PackageName},
		{"source", result.SourceDir},
		{"track", result.Track},
		{"trackFound", strconv.FormatBool(result.TrackFound)},
		{"changeCount", strconv.Itoa(result.ChangeCount)},
	}); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	rows := make([][]string, 0, len(result.Changes))
	for _, entry := range result.Changes {
		rows = append(rows, []string{entry.Scope, entry.Target, entry.Field, entry.Action, formatValue(entry.Live), formatValue(entry.Desired)})
	}
	return shared.WriteMarkdownTable(out, []string{"scope", "target", "field", "action", "live", "desired"}, rows)
}

func resolveOutput(local string) (string, error) {
	output := shared.ResolveOutput(local)
	switch output {
	case "json", "table", "markdown", "yaml":
		return output, nil
	default:
		return "", shared.UsageErrorf("output must be json, table, markdown, or yaml")
	}
}

func formatValue(v any) string {
	if v == nil {
		return "-"
	}
	if typed, ok := v.(string); ok {
		if typed == "" {
			return "-"
		}
		return typed
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(raw)
}

func sortChanges(changes []change) {
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Scope != changes[j].Scope {
			return changes[i].Scope < changes[j].Scope
		}
		if changes[i].Target != changes[j].Target {
			return changes[i].Target < changes[j].Target
		}
		if changes[i].Field != changes[j].Field {
			return changes[i].Field < changes[j].Field
		}
		return changes[i].Action < changes[j].Action
	})
}

func sortedImageTypes(images map[string][]string) []string {
	keys := make([]string, 0, len(images))
	for key := range images {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeRemoteImages(images []gpc.ImageInfo) []string {
	out := make([]string, 0, len(images))
	for _, image := range images {
		switch {
		case strings.TrimSpace(image.SHA256) != "":
			out = append(out, strings.ToLower(strings.TrimSpace(image.SHA256)))
		case strings.TrimSpace(image.SHA1) != "":
			out = append(out, "sha1:"+strings.ToLower(strings.TrimSpace(image.SHA1)))
		default:
			out = append(out, "id:"+strings.TrimSpace(image.ID))
		}
	}
	sort.Strings(out)
	return out
}

func hashFiles(paths []string) ([]string, error) {
	hashes := make([]string, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(raw)
		hashes = append(hashes, hex.EncodeToString(sum[:]))
	}
	sort.Strings(hashes)
	return hashes, nil
}

func normalizeNotes(values []gpc.LocalizedText) []gpc.LocalizedText {
	out := append([]gpc.LocalizedText(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Language != out[j].Language {
			return out[i].Language < out[j].Language
		}
		return out[i].Text < out[j].Text
	})
	return out
}

func sliceOrNil[T any](values []T) any {
	if len(values) == 0 {
		return nil
	}
	return values
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
