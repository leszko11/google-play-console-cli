package release

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type promoteStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

type promoteResult struct {
	PackageName      string         `json:"packageName"`
	FromTrack        string         `json:"fromTrack"`
	ToTrack          string         `json:"toTrack"`
	ReleaseStatus    string         `json:"releaseStatus,omitempty"`
	ReleaseName      string         `json:"releaseName,omitempty"`
	Status           string         `json:"status"`
	Committed        bool           `json:"committed"`
	CleanupPerformed bool           `json:"cleanupPerformed"`
	SourceTrack      *gpc.TrackInfo `json:"sourceTrack,omitempty"`
	TargetTrack      *gpc.TrackInfo `json:"targetTrack,omitempty"`
	Steps            []promoteStep  `json:"steps"`
}

type promoteOptions struct {
	PackageName string
	FromTrack   string
	ToTrack     string
	Status      string
	ReleaseName string
	DryRun      bool
	Confirm     bool
}

func newPromoteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts promoteOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Target package name")
	fs.StringVar(&opts.FromTrack, "from-track", "", "Source track name")
	fs.StringVar(&opts.ToTrack, "to-track", "", "Target track name")
	fs.StringVar(&opts.Status, "status", "", "Optional target release status override")
	fs.StringVar(&opts.ReleaseName, "release-name", "", "Optional target release name override")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Create and validate the promotion but delete the edit instead of committing")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm committing promotion (required unless --dry-run)")

	return &ffcli.Command{
		Name:      "promote",
		ShortHelp: "Promote the latest releasable release from one track to another",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := validatePromoteOptions(opts); err != nil {
				return err
			}

			result, err := runPromote(ctx, deps, opts)
			_ = shared.WriteJSON(deps.Stdout, result)
			return err
		},
	}
}

func validatePromoteOptions(opts promoteOptions) error {
	if strings.TrimSpace(opts.PackageName) == "" {
		return shared.UsageErrorf("--package-name is required")
	}
	if strings.TrimSpace(opts.FromTrack) == "" {
		return shared.UsageErrorf("--from-track is required")
	}
	if strings.TrimSpace(opts.ToTrack) == "" {
		return shared.UsageErrorf("--to-track is required")
	}
	if strings.EqualFold(strings.TrimSpace(opts.FromTrack), strings.TrimSpace(opts.ToTrack)) {
		return shared.UsageErrorf("--from-track and --to-track must be different")
	}
	if !opts.DryRun && !opts.Confirm {
		return shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}
	return nil
}

func runPromote(ctx context.Context, deps Deps, opts promoteOptions) (promoteResult, error) {
	result := promoteResult{
		PackageName: strings.TrimSpace(opts.PackageName),
		FromTrack:   strings.TrimSpace(opts.FromTrack),
		ToTrack:     strings.TrimSpace(opts.ToTrack),
		Status:      "failed",
		Steps:       make([]promoteStep, 0, 8),
	}

	var currentEditID string
	fail := func(stepName string, err error) (promoteResult, error) {
		result.Steps = append(result.Steps, promoteStep{
			Name:   stepName,
			Status: "error",
			Error:  err.Error(),
		})
		if currentEditID != "" {
			client, requestCtx, cancel, clientErr := buildClient(ctx, deps)
			if clientErr == nil {
				if cleanupErr := client.DeleteEdit(requestCtx, result.PackageName, currentEditID); cleanupErr == nil {
					result.CleanupPerformed = true
					result.Steps = append(result.Steps, promoteStep{Name: "cleanup_delete_edit", Status: "ok"})
				} else {
					result.Steps = append(result.Steps, promoteStep{Name: "cleanup_delete_edit", Status: "error", Error: cleanupErr.Error()})
				}
				cancel()
			}
		}
		return result, err
	}

	client, requestCtx, cancel, err := buildClient(ctx, deps)
	if err != nil {
		return fail("create_client", err)
	}
	defer cancel()

	if err := client.VerifyPackageAccess(requestCtx, result.PackageName); err != nil {
		return fail("package_access", err)
	}
	result.Steps = append(result.Steps, promoteStep{Name: "package_access", Status: "ok"})

	edit, err := client.CreateEdit(requestCtx, result.PackageName)
	if err != nil {
		return fail("create_edit", fmt.Errorf("failed to create edit: %w", err))
	}
	currentEditID = edit.ID
	result.Steps = append(result.Steps, promoteStep{Name: "create_edit", Status: "ok", Detail: currentEditID})

	sourceTrack, err := client.GetTrack(requestCtx, result.PackageName, currentEditID, result.FromTrack)
	if err != nil {
		return fail("read_source_track", fmt.Errorf("failed to read source track: %w", err))
	}
	result.SourceTrack = &sourceTrack
	result.Steps = append(result.Steps, promoteStep{Name: "read_source_track", Status: "ok"})

	sourceRelease, err := latestPromotableRelease(sourceTrack)
	if err != nil {
		return fail("select_source_release", err)
	}
	result.Steps = append(result.Steps, promoteStep{
		Name:   "select_source_release",
		Status: "ok",
		Detail: fmt.Sprintf("versionCodes=%v", sourceRelease.VersionCodes),
	})

	targetStatus := strings.TrimSpace(opts.Status)
	if targetStatus == "" {
		targetStatus = strings.TrimSpace(sourceRelease.Status)
	}
	targetReleaseName := strings.TrimSpace(opts.ReleaseName)
	if targetReleaseName == "" {
		targetReleaseName = strings.TrimSpace(sourceRelease.Name)
	}
	result.ReleaseStatus = targetStatus
	result.ReleaseName = targetReleaseName

	targetTrack, err := client.UpdateTrack(requestCtx, result.PackageName, currentEditID, result.ToTrack, gpc.TrackUpdate{
		Status:         targetStatus,
		ReleaseName:    targetReleaseName,
		UserFraction:   sourceRelease.UserFraction,
		VersionCodes:   append([]int64(nil), sourceRelease.VersionCodes...),
		UpdatePriority: sourceRelease.UpdatePriority,
		ReleaseNotes:   append([]gpc.LocalizedText(nil), sourceRelease.ReleaseNotes...),
	})
	if err != nil {
		return fail("update_target_track", fmt.Errorf("failed to update target track: %w", err))
	}
	result.TargetTrack = &targetTrack
	result.Steps = append(result.Steps, promoteStep{Name: "update_target_track", Status: "ok"})

	if err := client.ValidateEdit(requestCtx, result.PackageName, currentEditID); err != nil {
		return fail("validate_edit", fmt.Errorf("failed to validate edit: %w", err))
	}
	result.Steps = append(result.Steps, promoteStep{Name: "validate_edit", Status: "ok"})

	if opts.DryRun {
		if err := client.DeleteEdit(requestCtx, result.PackageName, currentEditID); err != nil {
			return fail("delete_edit_dry_run", fmt.Errorf("failed to delete dry-run edit: %w", err))
		}
		result.CleanupPerformed = true
		result.Status = "dry-run"
		result.Steps = append(result.Steps, promoteStep{Name: "delete_edit_dry_run", Status: "ok"})
		return result, nil
	}

	if _, err := client.CommitEdit(requestCtx, result.PackageName, currentEditID, false); err != nil {
		return fail("commit_edit", fmt.Errorf("failed to commit edit: %w", err))
	}
	result.Committed = true
	result.Status = "committed"
	result.Steps = append(result.Steps, promoteStep{Name: "commit_edit", Status: "ok"})
	return result, nil
}

func latestPromotableRelease(track gpc.TrackInfo) (gpc.TrackReleaseInfo, error) {
	if len(track.Releases) == 0 {
		return gpc.TrackReleaseInfo{}, fmt.Errorf("source track %q has no releases to promote", track.Name)
	}
	for _, release := range track.Releases {
		if len(release.VersionCodes) == 0 {
			continue
		}
		return release, nil
	}
	return gpc.TrackReleaseInfo{}, fmt.Errorf("source track %q has no releasable versions to promote", track.Name)
}
