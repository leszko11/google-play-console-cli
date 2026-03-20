package rollback

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const (
	rollbackStatusCommitted = "committed"
	rollbackStatusDryRun    = "dry-run"
	rollbackStatusFailed    = "failed"
)

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ValidateEdit(ctx context.Context, packageName, editID string) error
	CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) (gpc.EditInfo, error)
	GetTrack(ctx context.Context, packageName, editID, trackName string) (gpc.TrackInfo, error)
	UpdateTrack(ctx context.Context, packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
}

type stepResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type haltedReleaseResult struct {
	VersionCodes     []int64 `json:"versionCodes,omitempty"`
	PreviousFraction float64 `json:"previousFraction,omitempty"`
}

type rollbackResult struct {
	PackageName      string               `json:"packageName"`
	Track            string               `json:"track"`
	Status           string               `json:"status"`
	HaltedRelease    *haltedReleaseResult `json:"haltedRelease,omitempty"`
	Steps            []stepResult         `json:"steps"`
	Committed        bool                 `json:"committed"`
	CleanupPerformed bool                 `json:"cleanupPerformed"`
}

type options struct {
	PackageName string
	Track       string
	Confirm     bool
	DryRun      bool
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("rollback", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts options
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Track, "track", "", "Track name (e.g. production)")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm halting the active rollout (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Create and validate the edit, then delete it instead of updating the track")

	return &ffcli.Command{
		Name:      "rollback",
		ShortHelp: "Halt the active staged rollout on a track",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			resolved, err := validateOptions(opts)
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

			return runRollback(ctx, requestCtx, client, deps.Stdout, resolved)
		},
	}
}

func validateOptions(opts options) (options, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return options{}, err
	}

	opts.PackageName = pkg
	opts.Track = strings.TrimSpace(opts.Track)
	if opts.Track == "" {
		return options{}, shared.UsageErrorf("--track is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return options{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}
	return opts, nil
}

func runRollback(parentCtx, requestCtx context.Context, client Client, out io.Writer, opts options) error {
	result := rollbackResult{
		PackageName: opts.PackageName,
		Track:       opts.Track,
		Status:      rollbackStatusFailed,
		Steps:       make([]stepResult, 0, 7),
	}

	var editID string
	fail := func(stepName string, err error) error {
		result.Steps = append(result.Steps, stepResult{Name: stepName, Status: "error", Error: err.Error()})
		if editID != "" {
			cleanupCtx, cleanupCancel := shared.ContextWithTimeout(parentCtx, shared.ActiveGlobalFlags().Timeout)
			cleanupErr := client.DeleteEdit(cleanupCtx, opts.PackageName, editID)
			cleanupCancel()
			if cleanupErr != nil {
				result.Steps = append(result.Steps, stepResult{Name: "cleanup_delete_edit", Status: "error", Error: cleanupErr.Error()})
			} else {
				result.CleanupPerformed = true
				result.Steps = append(result.Steps, stepResult{Name: "cleanup_delete_edit", Status: "ok"})
			}
		}
		_ = shared.WriteJSON(out, result)
		return err
	}

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return fail("create_edit", fmt.Errorf("failed to create edit: %w", err))
	}
	editID = edit.ID
	result.Steps = append(result.Steps, stepResult{Name: "create_edit", Status: "ok"})

	track, err := client.GetTrack(requestCtx, opts.PackageName, editID, opts.Track)
	if err != nil {
		return fail("get_track", fmt.Errorf("failed to read track: %w", err))
	}
	result.Steps = append(result.Steps, stepResult{Name: "get_track", Status: "ok"})

	release, err := selectRolloutToHalt(track)
	if err != nil {
		return fail("select_release", err)
	}
	result.HaltedRelease = &haltedReleaseResult{
		VersionCodes:     append([]int64(nil), release.VersionCodes...),
		PreviousFraction: release.UserFraction,
	}
	result.Steps = append(result.Steps, stepResult{Name: "select_release", Status: "ok"})

	if !opts.DryRun {
		_, err = client.UpdateTrack(requestCtx, opts.PackageName, editID, opts.Track, gpc.TrackUpdate{
			Status:         "halted",
			ReleaseName:    release.Name,
			UserFraction:   release.UserFraction,
			VersionCodes:   append([]int64(nil), release.VersionCodes...),
			UpdatePriority: release.UpdatePriority,
			ReleaseNotes:   append([]gpc.LocalizedText(nil), release.ReleaseNotes...),
		})
		if err != nil {
			return fail("update_track", fmt.Errorf("failed to update track: %w", err))
		}
		result.Steps = append(result.Steps, stepResult{Name: "update_track", Status: "ok"})
	} else {
		result.Steps = append(result.Steps, stepResult{Name: "update_track", Status: "skipped"})
	}

	if err := client.ValidateEdit(requestCtx, opts.PackageName, editID); err != nil {
		return fail("validate_edit", fmt.Errorf("failed to validate edit: %w", err))
	}
	result.Steps = append(result.Steps, stepResult{Name: "validate_edit", Status: "ok"})

	if opts.DryRun {
		if err := client.DeleteEdit(requestCtx, opts.PackageName, editID); err != nil {
			return fail("delete_edit_dry_run", fmt.Errorf("failed to delete dry-run edit: %w", err))
		}
		result.CleanupPerformed = true
		result.Status = rollbackStatusDryRun
		result.Steps = append(result.Steps, stepResult{Name: "delete_edit_dry_run", Status: "ok"})
		return shared.WriteJSON(out, result)
	}

	if _, err := client.CommitEdit(requestCtx, opts.PackageName, editID, false); err != nil {
		return fail("commit_edit", fmt.Errorf("failed to commit edit: %w", err))
	}
	result.Committed = true
	result.Status = rollbackStatusCommitted
	result.Steps = append(result.Steps, stepResult{Name: "commit_edit", Status: "ok"})
	return shared.WriteJSON(out, result)
}

func selectRolloutToHalt(track gpc.TrackInfo) (gpc.TrackReleaseInfo, error) {
	inProgress := make([]gpc.TrackReleaseInfo, 0, len(track.Releases))
	hasCompleted := false
	for _, release := range track.Releases {
		switch strings.TrimSpace(release.Status) {
		case "inProgress":
			inProgress = append(inProgress, release)
		case "completed":
			hasCompleted = true
		}
		if release.UserFraction >= 1 {
			hasCompleted = true
		}
	}

	switch len(inProgress) {
	case 1:
		if len(inProgress[0].VersionCodes) == 0 {
			return gpc.TrackReleaseInfo{}, fmt.Errorf("in-progress release on track %q has no version codes", track.Name)
		}
		return inProgress[0], nil
	case 0:
		if hasCompleted {
			return gpc.TrackReleaseInfo{}, fmt.Errorf("cannot halt a completed rollout on track %q", track.Name)
		}
		return gpc.TrackReleaseInfo{}, fmt.Errorf("track %q has no in-progress release to halt", track.Name)
	default:
		return gpc.TrackReleaseInfo{}, fmt.Errorf("track %q has multiple in-progress releases; refusing to halt implicitly", track.Name)
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
