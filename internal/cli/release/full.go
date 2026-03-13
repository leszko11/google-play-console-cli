package release

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

type fullStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type fullResult struct {
	PackageName          string     `json:"packageName"`
	Manifest             string     `json:"manifest"`
	Track                string     `json:"track"`
	Status               string     `json:"status"`
	UploadedArtifactType string     `json:"uploadedArtifactType,omitempty"`
	VersionCode          int64      `json:"versionCode,omitempty"`
	Committed            bool       `json:"committed"`
	CleanupPerformed     bool       `json:"cleanupPerformed"`
	Steps                []fullStep `json:"steps"`
}

type fullOptions struct {
	PackageName     string
	ManifestPath    string
	Confirm         bool
	DryRun          bool
	AllowProduction bool
}

func newFullCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("full", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts fullOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Target package name")
	fs.StringVar(&opts.ManifestPath, "manifest", "", "Path to release manifest (.json/.yaml/.yml)")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm committing release (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Run all steps but delete edit instead of committing")
	fs.BoolVar(&opts.AllowProduction, "allow-production", false, "Allow track=production")

	return &ffcli.Command{
		Name:      "full",
		ShortHelp: "Deploy a release from a YAML or JSON manifest",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, manifest, err := validateFullOptions(opts)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			return runFull(ctx, requestCtx, client, deps.Stdout, opts, manifest)
		},
	}
}

func validateFullOptions(opts fullOptions) (fullOptions, fullManifest, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return fullOptions{}, fullManifest{}, err
	}
	opts.PackageName = pkg
	opts.ManifestPath, err = shared.ResolveProjectPath(opts.ManifestPath, func(cfg config.ProjectConfig) string { return cfg.ReleaseManifest })
	if err != nil {
		return fullOptions{}, fullManifest{}, err
	}
	opts.ManifestPath = strings.TrimSpace(opts.ManifestPath)
	if opts.ManifestPath == "" {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--manifest is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}

	manifest, err := loadFullManifest(opts.ManifestPath)
	if err != nil {
		return fullOptions{}, fullManifest{}, err
	}
	if manifest.Track == "production" && !opts.AllowProduction {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--allow-production is required when --track=production")
	}
	return opts, manifest, nil
}

func runFull(parentCtx, requestCtx context.Context, client Client, out io.Writer, opts fullOptions, manifest fullManifest) error {
	result := fullResult{
		PackageName: opts.PackageName,
		Manifest:    opts.ManifestPath,
		Track:       manifest.Track,
		Status:      "failed",
		Steps:       make([]fullStep, 0, 8),
	}

	var currentEditID string
	fail := func(step string, err error) error {
		result.Steps = append(result.Steps, fullStep{Name: step, Status: "error", Error: err.Error()})
		if currentEditID != "" {
			cleanupCtx, cleanupCancel := shared.ContextWithTimeout(parentCtx, shared.ActiveGlobalFlags().Timeout)
			cleanupErr := client.DeleteEdit(cleanupCtx, opts.PackageName, currentEditID)
			cleanupCancel()
			if cleanupErr == nil {
				result.CleanupPerformed = true
				result.Steps = append(result.Steps, fullStep{Name: "cleanup_delete_edit", Status: "ok"})
			} else {
				result.Steps = append(result.Steps, fullStep{Name: "cleanup_delete_edit", Status: "error", Error: cleanupErr.Error()})
			}
		}
		_ = shared.WriteJSON(out, result)
		return err
	}

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return fail("create_edit", fmt.Errorf("failed to create edit: %w", err))
	}
	currentEditID = edit.ID
	result.Steps = append(result.Steps, fullStep{Name: "create_edit", Status: "ok"})

	uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(parentCtx, shared.ActiveGlobalFlags().UploadTimeout)
	versionCode, err := uploadManifestArtifact(uploadCtx, client, opts.PackageName, currentEditID, manifest)
	uploadCancel()
	if err != nil {
		return fail("upload_artifact", err)
	}
	result.VersionCode = versionCode
	result.UploadedArtifactType = manifest.ArtifactType
	result.Steps = append(result.Steps, fullStep{Name: "upload_artifact", Status: "ok"})

	if manifest.MappingFile != "" {
		mapCtx, mapCancel := shared.ContextWithUploadTimeout(parentCtx, shared.ActiveGlobalFlags().UploadTimeout)
		_, err := client.UploadDeobfuscationFile(mapCtx, opts.PackageName, currentEditID, versionCode, manifest.MappingType, manifest.MappingFile)
		mapCancel()
		if err != nil {
			return fail("upload_mapping", fmt.Errorf("failed to upload mapping file: %w", err))
		}
		result.Steps = append(result.Steps, fullStep{Name: "upload_mapping", Status: "ok"})
	} else {
		result.Steps = append(result.Steps, fullStep{Name: "upload_mapping", Status: "skipped"})
	}

	_, err = client.UpdateTrack(requestCtx, opts.PackageName, currentEditID, manifest.Track, gpc.TrackUpdate{
		Status:         manifest.Status,
		ReleaseName:    manifest.ReleaseName,
		UserFraction:   manifest.UserFraction,
		VersionCodes:   []int64{versionCode},
		UpdatePriority: manifest.UpdatePriority,
		ReleaseNotes:   manifest.ReleaseNotes,
	})
	if err != nil {
		return fail("update_track", fmt.Errorf("failed to update track: %w", err))
	}
	result.Steps = append(result.Steps, fullStep{Name: "update_track", Status: "ok"})

	if err := client.ValidateEdit(requestCtx, opts.PackageName, currentEditID); err != nil {
		return fail("validate_edit", fmt.Errorf("failed to validate edit: %w", err))
	}
	result.Steps = append(result.Steps, fullStep{Name: "validate_edit", Status: "ok"})

	if opts.DryRun {
		if err := client.DeleteEdit(requestCtx, opts.PackageName, currentEditID); err != nil {
			return fail("delete_edit_dry_run", fmt.Errorf("failed to delete dry-run edit: %w", err))
		}
		result.CleanupPerformed = true
		result.Status = "dry-run"
		result.Steps = append(result.Steps, fullStep{Name: "delete_edit_dry_run", Status: "ok"})
		return shared.WriteJSON(out, result)
	}

	if _, err := client.CommitEdit(requestCtx, opts.PackageName, currentEditID); err != nil {
		return fail("commit_edit", fmt.Errorf("failed to commit edit: %w", err))
	}
	result.Committed = true
	result.Status = "committed"
	result.Steps = append(result.Steps, fullStep{Name: "commit_edit", Status: "ok"})
	return shared.WriteJSON(out, result)
}

func uploadManifestArtifact(ctx context.Context, client Client, packageName, editID string, manifest fullManifest) (int64, error) {
	switch manifest.ArtifactType {
	case artifactTypeAAB:
		bundle, err := client.UploadBundle(ctx, packageName, editID, manifest.ArtifactPath)
		if err != nil {
			return 0, fmt.Errorf("failed to upload aab: %w", err)
		}
		if bundle.VersionCode <= 0 {
			return 0, fmt.Errorf("uploaded aab did not return a valid version code")
		}
		return bundle.VersionCode, nil
	case artifactTypeAPK:
		apk, err := client.UploadAPK(ctx, packageName, editID, manifest.ArtifactPath)
		if err != nil {
			return 0, fmt.Errorf("failed to upload apk: %w", err)
		}
		if apk.VersionCode <= 0 {
			return 0, fmt.Errorf("uploaded apk did not return a valid version code")
		}
		return apk.VersionCode, nil
	default:
		return 0, fmt.Errorf("unsupported artifact type %q", manifest.ArtifactType)
	}
}
