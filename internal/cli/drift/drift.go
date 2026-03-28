package drift

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/listing"
	"github.com/leszko11/google-play-console-cli/internal/cli/screenshots"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var validIncludes = map[string]struct{}{
	"listing":       {},
	"screenshots":   {},
	"changelog":     {},
	"products":      {},
	"subscriptions": {},
	"track":         {},
}

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ListListings(ctx context.Context, packageName, editID string) ([]gpc.ListingInfo, error)
	ListImages(ctx context.Context, packageName, editID, language, imageType string) ([]gpc.ImageInfo, error)
	GetTrack(ctx context.Context, packageName, editID, trackName string) (gpc.TrackInfo, error)
	ListOneTimeProducts(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (gpc.OneTimeProductsListInfo, error)
	ListSubscriptions(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (gpc.SubscriptionsListInfo, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
}

type options struct {
	PackageName        string
	Dir                string
	Track              string
	Includes           []string
	ExplicitIncludes   map[string]bool
	Output             string
	ReleaseStatus      string
	ReleaseName        string
	VersionCodesCSV    string
	UserFraction       float64
	UpdatePriority     int64
	ReleaseNotesFile   string
	ReleaseNotesLocale string
	ReleaseNotesText   string
}

type desiredTrackDraft struct {
	Status         string
	ReleaseName    string
	VersionCodes   []int64
	UserFraction   *float64
	UpdatePriority *int64
	ReleaseNotes   []gpc.LocalizedText
}

type surfaceResult struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Dir         string   `json:"dir,omitempty"`
	HasDiff     bool     `json:"hasDiff"`
	ChangeCount int      `json:"changeCount"`
	Errors      []string `json:"errors,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

type reportResult struct {
	PackageName string          `json:"packageName"`
	Workspace   string          `json:"workspace,omitempty"`
	Track       string          `json:"track"`
	Status      string          `json:"status"`
	HasDiff     bool            `json:"hasDiff"`
	Surfaces    []surfaceResult `json:"surfaces"`
}

type surfacePaths struct {
	Workspace        string
	ListingDir       string
	ScreenshotsDir   string
	ProductsDir      string
	SubscriptionsDir string
	ChangelogDir     string
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "drift",
		ShortHelp: "Aggregate read-only workspace drift against live Play state",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newReportCommand(deps),
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

func newReportCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts options
	var includes csvStrings
	opts.UserFraction = -1
	opts.ReleaseNotesLocale = "en-US"

	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Dir, "dir", "", "Workspace root containing listing/, screenshots/, changelog/, products/, and subscriptions/")
	fs.StringVar(&opts.Track, "track", "", "Track name for changelog and optional track drift")
	fs.Var(&includes, "include", "Surface(s) to include: listing, screenshots, changelog, products, subscriptions, track")
	fs.StringVar(&opts.Output, "output", "", "Output format: json, table, markdown, yaml")
	fs.StringVar(&opts.ReleaseStatus, "status", "", "Desired track release status (draft, inProgress, halted, completed)")
	fs.StringVar(&opts.ReleaseName, "release-name", "", "Desired track release name")
	fs.StringVar(&opts.VersionCodesCSV, "version-codes", "", "Comma-separated version codes for desired track state")
	fs.Float64Var(&opts.UserFraction, "user-fraction", -1, "Desired rollout user fraction (0-1)")
	fs.Int64Var(&opts.UpdatePriority, "update-priority", 0, "Desired in-app update priority (0-5)")
	fs.StringVar(&opts.ReleaseNotesFile, "release-notes-file", "", "Desired release notes file")
	fs.StringVar(&opts.ReleaseNotesLocale, "release-notes-locale", "en-US", "Desired release notes locale")
	fs.StringVar(&opts.ReleaseNotesText, "release-notes-text", "", "Desired inline release notes text")

	return &ffcli.Command{
		Name:      "report",
		ShortHelp: "Roll up read-only drift across store content surfaces",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts.Includes = includes.Values()
			resolved, draft, paths, err := validateOptions(opts)
			if err != nil {
				return err
			}

			result, runErr := runReport(ctx, deps, resolved, draft, paths)
			if writeErr := writeReport(deps.Stdout, shared.ResolveOutput(resolved.Output), result); writeErr != nil {
				return writeErr
			}
			return runErr
		},
	}
}

func validateOptions(opts options) (options, desiredTrackDraft, surfacePaths, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return options{}, desiredTrackDraft{}, surfacePaths{}, err
	}
	opts.PackageName = pkg

	track, err := shared.ResolveDefaultTrack(opts.Track)
	if err != nil {
		return options{}, desiredTrackDraft{}, surfacePaths{}, err
	}
	opts.Track = strings.TrimSpace(track)
	if opts.Track == "" {
		return options{}, desiredTrackDraft{}, surfacePaths{}, shared.UsageErrorf("--track is required")
	}

	switch shared.ResolveOutput(opts.Output) {
	case "json", "table", "markdown", "yaml":
	default:
		return options{}, desiredTrackDraft{}, surfacePaths{}, shared.UsageErrorf("output must be json, table, markdown, or yaml")
	}

	selected, explicit, err := resolveIncludes(opts.Includes, hasTrackDraftInput(opts))
	if err != nil {
		return options{}, desiredTrackDraft{}, surfacePaths{}, err
	}
	opts.Includes = selected
	opts.ExplicitIncludes = explicit

	if explicit["track"] && !hasTrackDraftInput(opts) {
		return options{}, desiredTrackDraft{}, surfacePaths{}, shared.UsageErrorf("track drift requires desired release flags such as --status and --version-codes")
	}
	draft := desiredTrackDraft{}
	if shouldRunTrack(selected) {
		draft, err = buildDesiredTrackDraft(opts)
		if err != nil {
			return options{}, desiredTrackDraft{}, surfacePaths{}, err
		}
	}

	paths, err := resolveSurfacePaths(opts.Dir, opts.Track)
	if err != nil {
		return options{}, desiredTrackDraft{}, surfacePaths{}, err
	}

	return opts, draft, paths, nil
}

func runReport(ctx context.Context, deps Deps, opts options, draft desiredTrackDraft, paths surfacePaths) (reportResult, error) {
	result := reportResult{
		PackageName: opts.PackageName,
		Workspace:   paths.Workspace,
		Track:       opts.Track,
		Status:      "ok",
		Surfaces:    make([]surfaceResult, 0, len(opts.Includes)),
	}

	client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
	})
	if err != nil {
		for _, name := range opts.Includes {
			result.Surfaces = append(result.Surfaces, surfaceResult{
				Name:   name,
				Status: "error",
				Errors: []string{err.Error()},
			})
		}
		result.Status = "error"
		return result, err
	}
	defer cancel()

	needsEdit := includesAny(opts.Includes, "listing", "screenshots", "changelog", "track")
	editID := ""
	if needsEdit {
		edit, editErr := client.CreateEdit(requestCtx, opts.PackageName)
		if editErr != nil {
			for _, name := range opts.Includes {
				if name == "products" || name == "subscriptions" {
					continue
				}
				result.Surfaces = append(result.Surfaces, surfaceResult{
					Name:   name,
					Status: "error",
					Dir:    dirForSurface(name, paths),
					Errors: []string{fmt.Sprintf("failed to create inspection edit: %v", editErr)},
				})
			}
			editID = ""
		} else {
			editID = edit.ID
			defer client.DeleteEdit(ctx, opts.PackageName, editID)
		}
	}

	for _, name := range opts.Includes {
		var surface surfaceResult
		switch name {
		case "listing":
			surface = runListingSurface(ctx, requestCtx, client, opts, paths, editID)
		case "screenshots":
			surface = runScreenshotsSurface(ctx, requestCtx, client, opts, paths, editID)
		case "changelog":
			surface = runChangelogSurface(ctx, requestCtx, client, opts, paths, editID)
		case "products":
			surface = runProductsSurface(requestCtx, client, opts, paths)
		case "subscriptions":
			surface = runSubscriptionsSurface(requestCtx, client, opts, paths)
		case "track":
			surface = runTrackSurface(ctx, requestCtx, client, opts, editID, draft)
		default:
			surface = surfaceResult{Name: name, Status: "error", Errors: []string{"unsupported surface"}}
		}
		result.Surfaces = append(result.Surfaces, surface)
		if surface.HasDiff {
			result.HasDiff = true
		}
		if surface.Status == "error" {
			result.Status = "error"
		}
	}

	if result.Status == "ok" && result.HasDiff {
		result.Status = "diff"
	}

	if result.Status == "error" {
		return result, fmt.Errorf("drift report found surface errors")
	}
	return result, nil
}

func runListingSurface(ctx, requestCtx context.Context, client Client, opts options, paths surfacePaths, editID string) surfaceResult {
	surface := surfaceResult{Name: "listing", Dir: paths.ListingDir}
	if editID == "" {
		surface.Status = "error"
		surface.Errors = []string{"listing inspection unavailable because the transient edit could not be created"}
		return surface
	}
	explicit := opts.ExplicitIncludes["listing"]
	exists, err := dirExists(paths.ListingDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	if !exists {
		if explicit {
			surface.Status = "error"
			surface.Errors = []string{fmt.Sprintf("listing directory not found: %s", paths.ListingDir)}
		} else {
			surface.Status = "skipped"
		}
		return surface
	}

	locales, err := listing.ScanListingsDir(paths.ListingDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	remote, err := client.ListListings(requestCtx, opts.PackageName, editID)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{fmt.Sprintf("failed to list live listings: %v", err)}
		return surface
	}

	remoteByLocale := make(map[string]gpc.ListingInfo, len(remote))
	for _, item := range remote {
		remoteByLocale[item.Language] = item
	}
	changes := 0
	for _, locale := range locales {
		live, ok := remoteByLocale[locale.Locale]
		if !ok {
			changes++
			continue
		}
		if live.Title != locale.Listing.Title {
			changes++
		}
		if live.ShortDescription != locale.Listing.ShortDescription {
			changes++
		}
		if live.FullDescription != locale.Listing.FullDescription {
			changes++
		}
		delete(remoteByLocale, locale.Locale)
	}
	changes += len(remoteByLocale)
	surface.ChangeCount = changes
	surface.HasDiff = changes > 0
	if surface.HasDiff {
		surface.Status = "diff"
	} else {
		surface.Status = "ok"
	}
	_ = ctx
	return surface
}

func runScreenshotsSurface(ctx, requestCtx context.Context, client Client, opts options, paths surfacePaths, editID string) surfaceResult {
	surface := surfaceResult{Name: "screenshots", Dir: paths.ScreenshotsDir}
	if editID == "" {
		surface.Status = "error"
		surface.Errors = []string{"screenshots inspection unavailable because the transient edit could not be created"}
		return surface
	}
	explicit := opts.ExplicitIncludes["screenshots"]
	exists, err := dirExists(paths.ScreenshotsDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	if !exists {
		if explicit {
			surface.Status = "error"
			surface.Errors = []string{fmt.Sprintf("screenshots directory not found: %s", paths.ScreenshotsDir)}
		} else {
			surface.Status = "skipped"
		}
		return surface
	}

	locales, err := screenshots.ScanScreenshotsDir(paths.ScreenshotsDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	changes := 0
	for _, locale := range locales {
		imageTypes := sortedKeys(locale.Images)
		for _, imageType := range imageTypes {
			remote, err := client.ListImages(requestCtx, opts.PackageName, editID, locale.Locale, imageType)
			if err != nil {
				surface.Status = "error"
				surface.Errors = []string{fmt.Sprintf("failed to list live images for %s/%s: %v", locale.Locale, imageType, err)}
				return surface
			}
			liveHashes := normalizeRemoteImages(remote)
			desiredHashes, err := hashFiles(locale.Images[imageType])
			if err != nil {
				surface.Status = "error"
				surface.Errors = []string{fmt.Sprintf("failed to hash local screenshots for %s/%s: %v", locale.Locale, imageType, err)}
				return surface
			}
			if !reflect.DeepEqual(liveHashes, desiredHashes) {
				changes++
			}
		}
	}
	surface.ChangeCount = changes
	surface.HasDiff = changes > 0
	if surface.HasDiff {
		surface.Status = "diff"
	} else {
		surface.Status = "ok"
	}
	_ = ctx
	return surface
}

func runChangelogSurface(ctx, requestCtx context.Context, client Client, opts options, paths surfacePaths, editID string) surfaceResult {
	surface := surfaceResult{Name: "changelog", Dir: paths.ChangelogDir}
	if editID == "" {
		surface.Status = "error"
		surface.Errors = []string{"changelog inspection unavailable because the transient edit could not be created"}
		return surface
	}
	explicit := opts.ExplicitIncludes["changelog"]
	exists, err := dirExists(paths.ChangelogDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	if !exists {
		if explicit {
			surface.Status = "error"
			surface.Errors = []string{fmt.Sprintf("changelog directory not found: %s", paths.ChangelogDir)}
		} else {
			surface.Status = "skipped"
		}
		return surface
	}

	localNotes, err := loadChangelogDir(paths.ChangelogDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	track, err := client.GetTrack(requestCtx, opts.PackageName, editID, opts.Track)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{fmt.Sprintf("failed to read track %q: %v", opts.Track, err)}
		return surface
	}
	release, err := selectRelease(track)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}

	liveNotes := make(map[string]string, len(release.ReleaseNotes))
	for _, note := range release.ReleaseNotes {
		liveNotes[note.Language] = note.Text
	}
	changes := 0
	for _, note := range localNotes {
		if liveNotes[note.Language] != note.Text {
			changes++
		}
		delete(liveNotes, note.Language)
	}
	changes += len(liveNotes)
	surface.ChangeCount = changes
	surface.HasDiff = changes > 0
	if surface.HasDiff {
		surface.Status = "diff"
	} else {
		surface.Status = "ok"
	}
	_ = ctx
	return surface
}

func runProductsSurface(ctx context.Context, client Client, opts options, paths surfacePaths) surfaceResult {
	surface := surfaceResult{Name: "products", Dir: paths.ProductsDir}
	explicit := opts.ExplicitIncludes["products"]
	exists, err := dirExists(paths.ProductsDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	if !exists {
		if explicit {
			surface.Status = "error"
			surface.Errors = []string{fmt.Sprintf("products directory not found: %s", paths.ProductsDir)}
		} else {
			surface.Status = "skipped"
		}
		return surface
	}
	localIDs, err := jsonIDsInDir(paths.ProductsDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	remote, err := client.ListOneTimeProducts(ctx, opts.PackageName, 0, "", true)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{fmt.Sprintf("failed to list remote products: %v", err)}
		return surface
	}
	remoteIDs := make(map[string]struct{}, len(remote.Products))
	for _, item := range remote.Products {
		remoteIDs[item.ProductID] = struct{}{}
	}
	changes := setDiffCount(localIDs, remoteIDs)
	surface.ChangeCount = changes
	surface.HasDiff = changes > 0
	if surface.HasDiff {
		surface.Status = "diff"
	} else {
		surface.Status = "ok"
	}
	return surface
}

func runSubscriptionsSurface(ctx context.Context, client Client, opts options, paths surfacePaths) surfaceResult {
	surface := surfaceResult{Name: "subscriptions", Dir: paths.SubscriptionsDir}
	explicit := opts.ExplicitIncludes["subscriptions"]
	exists, err := dirExists(paths.SubscriptionsDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	if !exists {
		if explicit {
			surface.Status = "error"
			surface.Errors = []string{fmt.Sprintf("subscriptions directory not found: %s", paths.SubscriptionsDir)}
		} else {
			surface.Status = "skipped"
		}
		return surface
	}
	localIDs, err := jsonIDsInDir(paths.SubscriptionsDir)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{err.Error()}
		return surface
	}
	remote, err := client.ListSubscriptions(ctx, opts.PackageName, 0, "", true)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{fmt.Sprintf("failed to list remote subscriptions: %v", err)}
		return surface
	}
	remoteIDs := make(map[string]struct{}, len(remote.Subscriptions))
	for _, item := range remote.Subscriptions {
		remoteIDs[item.ProductID] = struct{}{}
	}
	changes := setDiffCount(localIDs, remoteIDs)
	surface.ChangeCount = changes
	surface.HasDiff = changes > 0
	if surface.HasDiff {
		surface.Status = "diff"
	} else {
		surface.Status = "ok"
	}
	return surface
}

func runTrackSurface(ctx, requestCtx context.Context, client Client, opts options, editID string, draft desiredTrackDraft) surfaceResult {
	surface := surfaceResult{Name: "track"}
	if editID == "" {
		surface.Status = "error"
		surface.Errors = []string{"track inspection unavailable because the transient edit could not be created"}
		return surface
	}
	track, err := client.GetTrack(requestCtx, opts.PackageName, editID, opts.Track)
	if err != nil {
		surface.Status = "error"
		surface.Errors = []string{fmt.Sprintf("failed to read track %q: %v", opts.Track, err)}
		return surface
	}
	if len(track.Releases) == 0 {
		surface.Status = "diff"
		surface.HasDiff = true
		surface.ChangeCount = 1
		return surface
	}
	live := track.Releases[0]
	changes := 0
	if live.Status != draft.Status {
		changes++
	}
	if live.Name != draft.ReleaseName {
		changes++
	}
	if !reflect.DeepEqual(normalizeVersionCodes(live.VersionCodes), normalizeVersionCodes(draft.VersionCodes)) {
		changes++
	}
	if !equalOptionalFloat(live.UserFraction, draft.UserFraction) {
		changes++
	}
	if !equalOptionalInt(live.UpdatePriority, draft.UpdatePriority) {
		changes++
	}
	if !reflect.DeepEqual(normalizeNotes(live.ReleaseNotes), normalizeNotes(draft.ReleaseNotes)) {
		changes++
	}
	surface.ChangeCount = changes
	surface.HasDiff = changes > 0
	if surface.HasDiff {
		surface.Status = "diff"
	} else {
		surface.Status = "ok"
	}
	_ = ctx
	return surface
}

func resolveIncludes(raw []string, includeTrackByDefault bool) ([]string, map[string]bool, error) {
	explicit := map[string]bool{}
	if len(raw) == 0 {
		out := []string{"listing", "screenshots", "changelog", "products", "subscriptions"}
		if includeTrackByDefault {
			out = append(out, "track")
		}
		return out, explicit, nil
	}
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if value == "all" {
			out = []string{"listing", "screenshots", "changelog", "products", "subscriptions", "track"}
			for _, name := range out {
				explicit[name] = true
			}
			return out, explicit, nil
		}
		if _, ok := validIncludes[value]; !ok {
			return nil, nil, shared.UsageErrorf("unsupported include %q", value)
		}
		if !explicit[value] {
			out = append(out, value)
			explicit[value] = true
		}
	}
	return out, explicit, nil
}

func resolveSurfacePaths(root, track string) (surfacePaths, error) {
	track = strings.TrimSpace(track)
	if strings.TrimSpace(root) != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return surfacePaths{}, err
		}
		return surfacePaths{
			Workspace:        abs,
			ListingDir:       filepath.Join(abs, "listing"),
			ScreenshotsDir:   filepath.Join(abs, "screenshots"),
			ProductsDir:      filepath.Join(abs, "products"),
			SubscriptionsDir: filepath.Join(abs, "subscriptions"),
			ChangelogDir:     filepath.Join(abs, "changelog", track),
		}, nil
	}

	listingDir, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.ListingDir })
	if err != nil {
		return surfacePaths{}, err
	}
	screenshotsDir, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.ScreenshotsDir })
	if err != nil {
		return surfacePaths{}, err
	}
	productsDir, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.ProductsDir })
	if err != nil {
		return surfacePaths{}, err
	}
	subscriptionsDir, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.SubscriptionsDir })
	if err != nil {
		return surfacePaths{}, err
	}
	changelogRoot, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.ChangelogDir })
	if err != nil {
		return surfacePaths{}, err
	}
	changelogDir := ""
	if strings.TrimSpace(changelogRoot) != "" {
		changelogDir = filepath.Join(changelogRoot, track)
	}
	return surfacePaths{
		ListingDir:       listingDir,
		ScreenshotsDir:   screenshotsDir,
		ProductsDir:      productsDir,
		SubscriptionsDir: subscriptionsDir,
		ChangelogDir:     changelogDir,
	}, nil
}

func buildDesiredTrackDraft(opts options) (desiredTrackDraft, error) {
	if strings.TrimSpace(opts.ReleaseStatus) == "" {
		return desiredTrackDraft{}, shared.UsageErrorf("--status is required when track drift is requested")
	}
	versionCodes, err := parseVersionCodes(opts.VersionCodesCSV)
	if err != nil {
		return desiredTrackDraft{}, err
	}
	if len(versionCodes) == 0 {
		return desiredTrackDraft{}, shared.UsageErrorf("--version-codes is required when track drift is requested")
	}
	if opts.UserFraction > 1 || (opts.UserFraction <= 0 && opts.UserFraction != -1) {
		return desiredTrackDraft{}, shared.UsageErrorf("--user-fraction must be within (0,1] when set")
	}
	if opts.UpdatePriority < 0 || opts.UpdatePriority > 5 {
		return desiredTrackDraft{}, shared.UsageErrorf("--update-priority must be between 0 and 5")
	}
	notes, err := shared.ParseReleaseNotesInput(opts.ReleaseNotesFile, opts.ReleaseNotesText, opts.ReleaseNotesLocale, nil)
	if err != nil {
		return desiredTrackDraft{}, err
	}
	draft := desiredTrackDraft{
		Status:       strings.TrimSpace(opts.ReleaseStatus),
		ReleaseName:  strings.TrimSpace(opts.ReleaseName),
		VersionCodes: versionCodes,
		ReleaseNotes: normalizeNotes(notes),
	}
	if opts.UserFraction > 0 {
		value := opts.UserFraction
		draft.UserFraction = &value
	}
	if opts.UpdatePriority > 0 {
		value := opts.UpdatePriority
		draft.UpdatePriority = &value
	}
	return draft, nil
}

func hasTrackDraftInput(opts options) bool {
	return strings.TrimSpace(opts.ReleaseStatus) != "" ||
		strings.TrimSpace(opts.ReleaseName) != "" ||
		strings.TrimSpace(opts.VersionCodesCSV) != "" ||
		opts.UserFraction >= 0 ||
		opts.UpdatePriority > 0 ||
		strings.TrimSpace(opts.ReleaseNotesFile) != "" ||
		strings.TrimSpace(opts.ReleaseNotesText) != ""
}

func shouldRunTrack(includes []string) bool {
	for _, include := range includes {
		if include == "track" {
			return true
		}
	}
	return false
}

func includesContains(includes []string, name string) bool {
	for _, include := range includes {
		if include == name {
			return true
		}
	}
	return false
}

func includesAny(includes []string, names ...string) bool {
	for _, name := range names {
		if includesContains(includes, name) {
			return true
		}
	}
	return false
}

func dirForSurface(name string, paths surfacePaths) string {
	switch name {
	case "listing":
		return paths.ListingDir
	case "screenshots":
		return paths.ScreenshotsDir
	case "changelog":
		return paths.ChangelogDir
	case "products":
		return paths.ProductsDir
	case "subscriptions":
		return paths.SubscriptionsDir
	default:
		return ""
	}
}

func loadChangelogDir(dir string) ([]gpc.LocalizedText, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read changelog directory: %w", err)
	}
	notes := make([]gpc.LocalizedText, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || strings.ToLower(filepath.Ext(entry.Name())) != ".txt" {
			continue
		}
		locale := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(string(raw))
		if text == "" {
			return nil, fmt.Errorf("release notes file %s is empty", entry.Name())
		}
		notes = append(notes, gpc.LocalizedText{Language: locale, Text: text})
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].Language < notes[j].Language })
	if len(notes) == 0 {
		return nil, fmt.Errorf("no release notes files found in %s", dir)
	}
	return notes, nil
}

func selectRelease(track gpc.TrackInfo) (gpc.TrackReleaseInfo, error) {
	candidates := make([]gpc.TrackReleaseInfo, 0, len(track.Releases))
	for _, release := range track.Releases {
		if len(release.VersionCodes) > 0 {
			candidates = append(candidates, release)
		}
	}
	switch len(candidates) {
	case 0:
		return gpc.TrackReleaseInfo{}, fmt.Errorf("track %q has no releasable release", track.Name)
	case 1:
		return candidates[0], nil
	default:
		return gpc.TrackReleaseInfo{}, fmt.Errorf("track %q has multiple releases; refusing to diff changelog implicitly", track.Name)
	}
}

func jsonIDsInDir(dir string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}
	ids := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if strings.TrimSpace(id) != "" {
			ids[id] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no JSON files found in %s", dir)
	}
	return ids, nil
}

func setDiffCount(local map[string]struct{}, remote map[string]struct{}) int {
	changes := 0
	for id := range local {
		if _, ok := remote[id]; !ok {
			changes++
		}
	}
	for id := range remote {
		if _, ok := local[id]; !ok {
			changes++
		}
	}
	return changes
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
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

func normalizeVersionCodes(values []int64) []int64 {
	out := append([]int64(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
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

func equalOptionalFloat(live float64, desired *float64) bool {
	if desired == nil {
		return live == 0
	}
	return live == *desired
}

func equalOptionalInt(live int64, desired *int64) bool {
	if desired == nil {
		return live == 0
	}
	return live == *desired
}

func parseVersionCodes(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	values := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, shared.UsageErrorf("invalid version code %q", part)
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil, shared.UsageErrorf("--version-codes must include at least one valid integer")
	}
	return values, nil
}

func dirExists(path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func writeReport(out io.Writer, output string, result reportResult) error {
	switch output {
	case "json":
		return shared.WriteJSON(out, result)
	case "yaml":
		return shared.WriteYAML(out, result)
	case "table":
		return writeTable(out, result)
	case "markdown":
		return writeMarkdown(out, result)
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}

func writeTable(out io.Writer, result reportResult) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", result.PackageName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "TRACK\t%s\n", result.Track); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "SURFACE\tSTATUS\tHAS_DIFF\tCHANGE_COUNT\tDIR\tDETAIL"); err != nil {
		return err
	}
	for _, surface := range result.Surfaces {
		detail := ""
		if len(surface.Errors) > 0 {
			detail = strings.Join(surface.Errors, "; ")
		} else if len(surface.Warnings) > 0 {
			detail = strings.Join(surface.Warnings, "; ")
		}
		if _, err := fmt.Fprintf(out, "%s\t%s\t%t\t%d\t%s\t%s\n", surface.Name, surface.Status, surface.HasDiff, surface.ChangeCount, surface.Dir, detail); err != nil {
			return err
		}
	}
	return nil
}

func writeMarkdown(out io.Writer, result reportResult) error {
	rows := [][]string{
		{"status", result.Status},
		{"package", result.PackageName},
		{"track", result.Track},
	}
	if result.Workspace != "" {
		rows = append(rows, []string{"workspace", result.Workspace})
	}
	if err := shared.WriteMarkdownTable(out, []string{"field", "value"}, rows); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	surfaceRows := make([][]string, 0, len(result.Surfaces))
	for _, surface := range result.Surfaces {
		detail := ""
		if len(surface.Errors) > 0 {
			detail = strings.Join(surface.Errors, "; ")
		} else if len(surface.Warnings) > 0 {
			detail = strings.Join(surface.Warnings, "; ")
		}
		surfaceRows = append(surfaceRows, []string{
			surface.Name,
			surface.Status,
			strconv.FormatBool(surface.HasDiff),
			strconv.Itoa(surface.ChangeCount),
			surface.Dir,
			detail,
		})
	}
	return shared.WriteMarkdownTable(out, []string{"surface", "status", "hasDiff", "changeCount", "dir", "detail"}, surfaceRows)
}

type csvStrings struct {
	values []string
}

func (c *csvStrings) String() string {
	return strings.Join(c.values, ",")
}

func (c *csvStrings) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			c.values = append(c.values, part)
		}
	}
	return nil
}

func (c *csvStrings) Values() []string {
	return append([]string(nil), c.values...)
}
