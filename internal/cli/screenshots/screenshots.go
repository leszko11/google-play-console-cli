package screenshots

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/edits"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ValidateEdit(ctx context.Context, packageName, editID string) error
	CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) (gpc.EditInfo, error)
	DeleteAllImages(ctx context.Context, packageName, editID, language, imageType string) ([]gpc.ImageInfo, error)
	UploadImage(ctx context.Context, packageName, editID, language, imageType, imagePath string) (gpc.ImageInfo, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
}

type syncOptions struct {
	PackageName       string
	Dir               string
	Confirm           bool
	DryRun            bool
	Output            string
	AutoFixDraftTrack bool
}

type syncResult struct {
	PackageName             string   `json:"packageName"`
	Dir                     string   `json:"dir"`
	Status                  string   `json:"status"`
	LocaleCount             int      `json:"localeCount"`
	ImageTypeCount          int      `json:"imageTypeCount"`
	ImageUploadCount        int      `json:"imageUploadCount"`
	ReplaceCount            int      `json:"replaceCount"`
	PlannedActions          []string `json:"plannedActions,omitempty"`
	Committed               bool     `json:"committed"`
	CleanupPerformed        bool     `json:"cleanupPerformed"`
	ChangesNotSentForReview bool     `json:"changesNotSentForReview,omitempty"`
	CommitRetried           bool     `json:"commitRetried,omitempty"`
	DraftTrackAutoFixed     bool     `json:"draftTrackAutoFixed,omitempty"`
}

type localeScreenshots struct {
	Locale string
	Images map[string][]string
}

type LocaleScreenshots = localeScreenshots

var screenshotDirAliases = map[string]string{
	"phone":                  "phoneScreenshots",
	"phonescreenshots":       "phoneScreenshots",
	"phone-screenshots":      "phoneScreenshots",
	"seven-inch":             "sevenInchScreenshots",
	"seven_inch":             "sevenInchScreenshots",
	"seveninchscreenshots":   "sevenInchScreenshots",
	"seven-inch-screenshots": "sevenInchScreenshots",
	"ten-inch":               "tenInchScreenshots",
	"ten_inch":               "tenInchScreenshots",
	"teninchscreenshots":     "tenInchScreenshots",
	"ten-inch-screenshots":   "tenInchScreenshots",
	"tv":                     "tvScreenshots",
	"tvscreenshots":          "tvScreenshots",
	"tv-screenshots":         "tvScreenshots",
	"wear":                   "wearScreenshots",
	"wearscreenshots":        "wearScreenshots",
	"wear-screenshots":       "wearScreenshots",
}

var screenshotImageTypes = map[string]struct{}{
	"phoneScreenshots":     {},
	"sevenInchScreenshots": {},
	"tenInchScreenshots":   {},
	"tvScreenshots":        {},
	"wearScreenshots":      {},
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "screenshots",
		ShortHelp: "Manage screenshot-only sync workflows",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newSyncCommand(deps),
		},
	}
}

func ScanScreenshotsDir(root string) ([]LocaleScreenshots, error) {
	return scanScreenshotsDir(root)
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

func newSyncCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts syncOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Dir, "dir", "", "Screenshot directory root")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm committing the edit (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Create and validate the edit, then delete it instead of mutating Play")
	fs.StringVar(&opts.Output, "output", "", "Output format: json")
	fs.BoolVar(&opts.AutoFixDraftTrack, "auto-fix-draft-track", false, "If a draft-app commit fails because the internal track has a completed release, rewrite that internal release to draft and retry")

	return &ffcli.Command{
		Name:      "sync",
		ShortHelp: "Sync screenshots from a directory",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if strings.TrimSpace(opts.Output) != "" && shared.ResolveOutput(opts.Output) != "json" {
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(opts.Output))
			}

			var err error
			opts, err = validateSyncOptions(opts)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
				Upload:     true,
			})
			if err != nil {
				return err
			}
			defer cancel()

			locales, err := scanScreenshotsDir(opts.Dir)
			if err != nil {
				return err
			}

			return runSync(ctx, requestCtx, client, deps.Stdout, opts, locales)
		},
	}
}

func validateSyncOptions(opts syncOptions) (syncOptions, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return syncOptions{}, err
	}
	opts.PackageName = pkg
	opts.Dir, err = shared.ResolveProjectPath(opts.Dir, func(cfg config.ProjectConfig) string { return cfg.ScreenshotsDir })
	if err != nil {
		return syncOptions{}, err
	}
	opts.Dir = strings.TrimSpace(opts.Dir)
	if opts.Dir == "" {
		return syncOptions{}, shared.UsageErrorf("--dir is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return syncOptions{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}
	return opts, nil
}

func runSync(parentCtx, requestCtx context.Context, client Client, out io.Writer, opts syncOptions, locales []localeScreenshots) error {
	result := syncResult{
		PackageName: opts.PackageName,
		Dir:         opts.Dir,
		Status:      "failed",
		LocaleCount: len(locales),
	}
	for _, locale := range locales {
		result.ImageTypeCount += len(locale.Images)
	}
	result.PlannedActions = buildPlannedActions(locales)

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return fmt.Errorf("failed to create edit: %w", err)
	}

	fail := func(err error) error {
		cleanupCtx, cleanupCancel := shared.ContextWithTimeout(parentCtx, shared.ActiveGlobalFlags().Timeout)
		cleanupErr := client.DeleteEdit(cleanupCtx, opts.PackageName, edit.ID)
		cleanupCancel()
		if cleanupErr == nil {
			result.CleanupPerformed = true
		}
		_ = shared.WriteJSON(out, result)
		return err
	}

	if opts.DryRun {
		if err := client.ValidateEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
			return fail(explainDraftAppValidationError(fmt.Errorf("failed to validate edit: %w", err)))
		}
		if err := client.DeleteEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
			return fail(fmt.Errorf("failed to delete dry-run edit: %w", err))
		}
		result.CleanupPerformed = true
		result.Status = "dry-run"
		return shared.WriteJSON(out, result)
	}

	for _, locale := range locales {
		imageTypes := sortedImageTypes(locale.Images)
		for _, imageType := range imageTypes {
			paths := locale.Images[imageType]
			for _, path := range paths {
				if err := edits.ValidateImageUploadFile(imageType, path); err != nil {
					return fail(err)
				}
			}

			if _, err := client.DeleteAllImages(requestCtx, opts.PackageName, edit.ID, locale.Locale, imageType); err != nil {
				return fail(fmt.Errorf("failed to replace images for %q/%q: %w", locale.Locale, imageType, err))
			}
			result.ReplaceCount++

			for _, path := range paths {
				if _, err := client.UploadImage(requestCtx, opts.PackageName, edit.ID, locale.Locale, imageType, path); err != nil {
					return fail(fmt.Errorf("failed to upload image %q for %q/%q: %w", path, locale.Locale, imageType, err))
				}
				result.ImageUploadCount++
			}
		}
	}

	if err := client.ValidateEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
		return fail(explainDraftAppValidationError(fmt.Errorf("failed to validate edit: %w", err)))
	}

	commit, err := shared.CommitEditWithOptions(requestCtx, client, opts.PackageName, edit.ID, shared.EditCommitOptions{
		AutoFixDraftTrack: opts.AutoFixDraftTrack,
	})
	if err != nil {
		return fail(fmt.Errorf("failed to commit edit: %w", err))
	}
	result.ChangesNotSentForReview = commit.ChangesNotSentForReview
	result.CommitRetried = commit.RetriedWithChangesNotSentForReview
	result.DraftTrackAutoFixed = commit.DraftTrackAutoFixed
	result.Committed = true
	result.Status = "committed"
	return shared.WriteJSON(out, result)
}

func explainDraftAppValidationError(err error) error {
	if !shared.IsDraftAppError(err) {
		return err
	}
	return fmt.Errorf("%w\nhint: this package is still in Play's draft bootstrap state. Run `gpc release init --package-name <package> --dir ./play` to generate the bootstrap workflow, commit the draft bootstrap release with `gpc release full --manifest ./play/release.yaml --confirm`, wait for Play processing, then rerun screenshots sync.", err)
}

func scanScreenshotsDir(root string) ([]localeScreenshots, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("screenshots directory is required")
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read screenshots directory: %w", err)
	}

	locales := make([]localeScreenshots, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		localePath := filepath.Join(root, entry.Name())
		images, err := scanLocaleImages(localePath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		if len(images) == 0 {
			return nil, fmt.Errorf("%s: no screenshot directories found", entry.Name())
		}
		locales = append(locales, localeScreenshots{
			Locale: entry.Name(),
			Images: images,
		})
	}

	sort.Slice(locales, func(i, j int) bool {
		return locales[i].Locale < locales[j].Locale
	})
	if len(locales) == 0 {
		return nil, fmt.Errorf("no locale directories found in %s", root)
	}
	return locales, nil
}

func scanLocaleImages(localePath string) (map[string][]string, error) {
	images := map[string][]string{}

	entries, err := os.ReadDir(localePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if entry.Name() == "images" {
			listingImages, err := scanListingImagesDir(filepath.Join(localePath, entry.Name()))
			if err != nil {
				return nil, err
			}
			for imageType, paths := range listingImages {
				if _, exists := images[imageType]; exists {
					return nil, fmt.Errorf("duplicate screenshot image type %q", imageType)
				}
				images[imageType] = paths
			}
			continue
		}

		imageType, err := normalizeScreenshotDirName(entry.Name())
		if err != nil {
			return nil, err
		}
		if _, exists := images[imageType]; exists {
			return nil, fmt.Errorf("duplicate screenshot image type %q", imageType)
		}

		files, err := collectImageFiles(filepath.Join(localePath, entry.Name()))
		if err != nil {
			return nil, err
		}
		images[imageType] = files
	}

	return images, nil
}

func scanListingImagesDir(dir string) (map[string][]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	images := make(map[string][]string, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if _, ok := screenshotImageTypes[entry.Name()]; !ok {
			return nil, fmt.Errorf("unsupported listing screenshot directory %q", entry.Name())
		}
		files, err := collectImageFiles(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		images[entry.Name()] = files
	}
	return images, nil
}

func collectImageFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			return nil, fmt.Errorf("nested directories are not supported in %s", dir)
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		switch ext {
		case ".png", ".jpg", ".jpeg":
			files = append(files, filepath.Join(dir, entry.Name()))
		default:
			return nil, fmt.Errorf("unsupported screenshot file %q", entry.Name())
		}
	}

	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("directory %s does not contain image files", dir)
	}
	return files, nil
}

func normalizeScreenshotDirName(name string) (string, error) {
	if _, ok := screenshotImageTypes[name]; ok {
		return name, nil
	}

	key := strings.ToLower(strings.TrimSpace(name))
	key = strings.ReplaceAll(key, " ", "-")
	if imageType, ok := screenshotDirAliases[key]; ok {
		return imageType, nil
	}
	return "", fmt.Errorf("unsupported screenshot directory %q", name)
}

func sortedImageTypes(images map[string][]string) []string {
	keys := make([]string, 0, len(images))
	for imageType := range images {
		keys = append(keys, imageType)
	}
	sort.Strings(keys)
	return keys
}

func buildPlannedActions(locales []localeScreenshots) []string {
	actions := make([]string, 0)
	for _, locale := range locales {
		for _, imageType := range sortedImageTypes(locale.Images) {
			actions = append(actions, fmt.Sprintf("replace %s images for %s (%d files)", imageType, locale.Locale, len(locale.Images[imageType])))
		}
	}
	return actions
}
