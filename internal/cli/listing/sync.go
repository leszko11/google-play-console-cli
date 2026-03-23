package listing

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

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
	ListListings(ctx context.Context, packageName, editID string) ([]gpc.ListingInfo, error)
	UpdateListing(ctx context.Context, packageName, editID, language string, update gpc.ListingUpdate) (gpc.ListingInfo, error)
	DeleteListing(ctx context.Context, packageName, editID, language string) error
	DeleteAllImages(ctx context.Context, packageName, editID, language, imageType string) ([]gpc.ImageInfo, error)
	UploadImage(ctx context.Context, packageName, editID, language, imageType, imagePath string) (gpc.ImageInfo, error)
}

type syncResult struct {
	PackageName             string   `json:"packageName"`
	Dir                     string   `json:"dir"`
	Status                  string   `json:"status"`
	LocaleCount             int      `json:"localeCount"`
	ImageUploadCount        int      `json:"imageUploadCount"`
	DeletedLocales          []string `json:"deletedLocales,omitempty"`
	PlannedActions          []string `json:"plannedActions,omitempty"`
	Committed               bool     `json:"committed"`
	CleanupPerformed        bool     `json:"cleanupPerformed"`
	ChangesNotSentForReview bool     `json:"changesNotSentForReview,omitempty"`
	CommitRetried           bool     `json:"commitRetried,omitempty"`
	DraftTrackAutoFixed     bool     `json:"draftTrackAutoFixed,omitempty"`
}

type syncOptions struct {
	PackageName       string
	Dir               string
	Confirm           bool
	DryRun            bool
	DeleteMissing     bool
	AutoFixDraftTrack bool
}

type SyncOptions = syncOptions

func newSyncCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts syncOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Dir, "dir", "", "Listings directory root")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm committing the edit (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Create and validate the edit, then delete it instead of mutating Play")
	fs.BoolVar(&opts.DeleteMissing, "delete-missing", false, "Delete remote locales that do not exist locally")
	fs.BoolVar(&opts.AutoFixDraftTrack, "auto-fix-draft-track", false, "If a draft-app commit fails because the internal track has a completed release, rewrite that internal release to draft and retry")

	return &ffcli.Command{
		Name:      "sync",
		ShortHelp: "Sync store listing metadata and images from a directory",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateSyncOptions(opts)
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

			locales, err := scanListingsDir(opts.Dir)
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
	opts.Dir, err = shared.ResolveProjectPath(opts.Dir, func(cfg config.ProjectConfig) string { return cfg.ListingDir })
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

func runSync(parentCtx, requestCtx context.Context, client Client, out io.Writer, opts syncOptions, locales []localeData) error {
	result := syncResult{
		PackageName: opts.PackageName,
		Dir:         opts.Dir,
		Status:      "failed",
		LocaleCount: len(locales),
	}

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

	if opts.DeleteMissing {
		remote, err := client.ListListings(requestCtx, opts.PackageName, edit.ID)
		if err != nil {
			return fail(fmt.Errorf("failed to list remote locales: %w", err))
		}
		result.DeletedLocales = localesToDelete(remote, locales)
	}

	result.PlannedActions = buildPlannedActions(locales, result.DeletedLocales)

	for _, locale := range locales {
		if _, err := client.UpdateListing(requestCtx, opts.PackageName, edit.ID, locale.Locale, locale.Listing); err != nil {
			return fail(fmt.Errorf("failed to update listing %q: %w", locale.Locale, err))
		}
		for imageType, paths := range locale.Images {
			if _, err := client.DeleteAllImages(requestCtx, opts.PackageName, edit.ID, locale.Locale, imageType); err != nil {
				return fail(fmt.Errorf("failed to replace images for %q/%q: %w", locale.Locale, imageType, err))
			}
			for _, path := range paths {
				if _, err := client.UploadImage(requestCtx, opts.PackageName, edit.ID, locale.Locale, imageType, path); err != nil {
					return fail(fmt.Errorf("failed to upload image %q for %q/%q: %w", path, locale.Locale, imageType, err))
				}
				result.ImageUploadCount++
			}
		}
	}

	for _, locale := range result.DeletedLocales {
		if err := client.DeleteListing(requestCtx, opts.PackageName, edit.ID, locale); err != nil {
			return fail(fmt.Errorf("failed to delete locale %q: %w", locale, err))
		}
	}

	if opts.DryRun {
		if err := client.ValidateEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
			return fail(fmt.Errorf("failed to validate edit: %w", err))
		}
		if err := client.DeleteEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
			return fail(fmt.Errorf("failed to delete dry-run edit: %w", err))
		}
		result.CleanupPerformed = true
		result.Status = "dry-run"
		return shared.WriteJSON(out, result)
	}

	if err := client.ValidateEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
		return fail(fmt.Errorf("failed to validate edit: %w", err))
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

func RunSync(parentCtx, requestCtx context.Context, client Client, out io.Writer, opts SyncOptions, locales []LocaleData) error {
	return runSync(parentCtx, requestCtx, client, out, opts, locales)
}

func localesToDelete(remote []gpc.ListingInfo, locales []localeData) []string {
	localSet := make(map[string]struct{}, len(locales))
	for _, locale := range locales {
		localSet[locale.Locale] = struct{}{}
	}

	deleted := make([]string, 0)
	for _, listing := range remote {
		if _, ok := localSet[listing.Language]; ok {
			continue
		}
		deleted = append(deleted, listing.Language)
	}
	return deleted
}

func buildPlannedActions(locales []localeData, deletedLocales []string) []string {
	actions := make([]string, 0, len(locales)+len(deletedLocales))
	for _, locale := range locales {
		actions = append(actions, fmt.Sprintf("update listing %s", locale.Locale))
		for imageType, paths := range locale.Images {
			actions = append(actions, fmt.Sprintf("replace %s images for %s (%d files)", imageType, locale.Locale, len(paths)))
		}
	}
	for _, locale := range deletedLocales {
		actions = append(actions, fmt.Sprintf("delete remote locale %s", locale))
	}
	return actions
}
