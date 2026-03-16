package publish

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

const (
	artifactTypeAAB     = "aab"
	artifactTypeAPK     = "apk"
	mappingTypeProguard = "proguard"
	mappingTypeNative   = "nativeCode"
	defaultNotesLocale  = "en-US"
	defaultWaitInterval = 5 * time.Second
	statusCommitted     = "committed"
	statusDryRun        = "dry-run"
	statusFailed        = "failed"
)

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ValidateEdit(ctx context.Context, packageName, editID string) error
	CommitEdit(ctx context.Context, packageName, editID string) (gpc.EditInfo, error)
	UpdateTrack(ctx context.Context, packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error)
	UploadBundle(ctx context.Context, packageName, editID, bundlePath string) (gpc.BundleInfo, error)
	UploadAPK(ctx context.Context, packageName, editID, apkPath string) (gpc.APKInfo, error)
	UploadDeobfuscationFile(ctx context.Context, packageName, editID string, versionCode int64, fileType, filePath string) (gpc.DeobfuscationFileInfo, error)
	ListGeneratedAPKs(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.GeneratedApksListResponse, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Sleep      func(context.Context, time.Duration) error
	Now        func() time.Time
	Stdout     io.Writer
	Stderr     io.Writer
}

type publishOptions struct {
	PackageName      string
	Track            string
	TrackLabel       string
	AABPath          string
	APKPath          string
	ReleaseStatus    string
	ReleaseName      string
	UserFraction     float64
	UpdatePriority   int64
	MappingFile      string
	MappingType      string
	ReleaseNotesFile string
	ReleaseNotesText string
	ReleaseNotesLoc  string
	Confirm          bool
	CleanupOnFailure bool
	DryRun           bool
	WaitTimeout      time.Duration
	WaitInterval     time.Duration
}

type stepResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type waitResult struct {
	Status            string `json:"status"`
	Attempts          int    `json:"attempts,omitempty"`
	Elapsed           string `json:"elapsed,omitempty"`
	Interval          string `json:"interval,omitempty"`
	GeneratedApkCount int    `json:"generatedApkCount,omitempty"`
	Detail            string `json:"detail,omitempty"`
}

type publishResult struct {
	PackageName          string       `json:"packageName"`
	Track                string       `json:"track"`
	Status               string       `json:"status"`
	EditID               string       `json:"editId,omitempty"`
	UploadedArtifactType string       `json:"uploadedArtifactType,omitempty"`
	VersionCode          int64        `json:"versionCode,omitempty"`
	Committed            bool         `json:"committed"`
	CleanupPerformed     bool         `json:"cleanupPerformed"`
	Wait                 waitResult   `json:"wait"`
	Steps                []stepResult `json:"steps"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "publish",
		ShortHelp: "Common publish flows with track presets",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newTrackCommand(deps, "alpha"),
			newTrackCommand(deps, "production"),
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
	if deps.Sleep == nil {
		deps.Sleep = sleepContext
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return deps
}

func newTrackCommand(deps Deps, track string) *ffcli.Command {
	fs := flag.NewFlagSet(track, flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	opts := publishOptions{
		Track:            track,
		TrackLabel:       track,
		ReleaseStatus:    "completed",
		CleanupOnFailure: true,
		ReleaseNotesLoc:  defaultNotesLocale,
		WaitTimeout:      shared.DefaultTimeout,
		WaitInterval:     defaultWaitInterval,
	}

	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.AABPath, "aab", "", "Path to .aab file")
	fs.StringVar(&opts.APKPath, "apk", "", "Path to .apk file")
	fs.StringVar(&opts.ReleaseStatus, "status", opts.ReleaseStatus, "Release status (draft, inProgress, halted, completed)")
	fs.StringVar(&opts.ReleaseName, "release-name", "", "Release name")
	fs.Float64Var(&opts.UserFraction, "user-fraction", -1, "Rollout user fraction (0-1)")
	fs.Int64Var(&opts.UpdatePriority, "update-priority", 0, "In-app update priority (0-5)")
	fs.StringVar(&opts.MappingFile, "mapping-file", "", "Path to deobfuscation mapping file")
	fs.StringVar(&opts.MappingType, "mapping-type", "", "Mapping type: proguard or nativeCode (defaults to proguard)")
	fs.StringVar(&opts.ReleaseNotesFile, "release-notes-file", "", "Path to release notes file (JSON object/array, tagged blocks, or plain text)")
	fs.StringVar(&opts.ReleaseNotesLoc, "release-notes-locale", opts.ReleaseNotesLoc, "Release notes locale (BCP-47)")
	fs.StringVar(&opts.ReleaseNotesText, "release-notes-text", "", "Release notes text")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm committing the edit (required unless --dry-run)")
	fs.BoolVar(&opts.CleanupOnFailure, "cleanup-on-failure", true, "Delete edit if publish fails before commit")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Run publish steps, then delete edit instead of committing")
	fs.DurationVar(&opts.WaitTimeout, "wait-timeout", opts.WaitTimeout, "Maximum time to wait for generated APK availability after bundle upload")
	fs.DurationVar(&opts.WaitInterval, "wait-interval", opts.WaitInterval, "Polling interval between generated APK checks")

	return &ffcli.Command{
		Name:      track,
		ShortHelp: fmt.Sprintf("Upload and publish to the %s track in one flow", track),
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			params, err := validateOptions(opts)
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

			spinner := shared.NewSpinner(deps.Stderr, fmt.Sprintf("Running %s publish flow", track))
			result, err := executePublish(ctx, requestCtx, deps, client, params)
			if writeErr := shared.WriteJSON(deps.Stdout, result); writeErr != nil {
				spinner.Fail("Publish flow failed")
				return writeErr
			}
			if err != nil {
				spinner.Fail("Publish flow failed")
				return err
			}
			spinner.Success("Publish flow finished")
			return nil
		},
	}
}

func validateOptions(opts publishOptions) (publishOptions, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return publishOptions{}, err
	}
	opts.PackageName = pkg

	opts.AABPath = strings.TrimSpace(opts.AABPath)
	opts.APKPath = strings.TrimSpace(opts.APKPath)
	if (opts.AABPath == "" && opts.APKPath == "") || (opts.AABPath != "" && opts.APKPath != "") {
		return publishOptions{}, shared.UsageErrorf("exactly one of --aab or --apk is required")
	}

	if opts.AABPath != "" {
		if err := validateReadableFile(opts.AABPath, "artifact"); err != nil {
			return publishOptions{}, err
		}
	}
	if opts.APKPath != "" {
		if err := validateReadableFile(opts.APKPath, "artifact"); err != nil {
			return publishOptions{}, err
		}
	}

	opts.ReleaseStatus = strings.TrimSpace(opts.ReleaseStatus)
	if opts.ReleaseStatus == "" {
		return publishOptions{}, shared.UsageErrorf("--status is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return publishOptions{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}
	if opts.UserFraction > 1 || (opts.UserFraction >= 0 && opts.UserFraction <= 0) {
		return publishOptions{}, shared.UsageErrorf("--user-fraction must be within (0,1] when set")
	}
	if opts.UpdatePriority < 0 || opts.UpdatePriority > 5 {
		return publishOptions{}, shared.UsageErrorf("--update-priority must be between 0 and 5")
	}

	opts.MappingFile = strings.TrimSpace(opts.MappingFile)
	opts.MappingType = strings.TrimSpace(opts.MappingType)
	if opts.MappingFile == "" && opts.MappingType != "" {
		return publishOptions{}, shared.UsageErrorf("--mapping-type requires --mapping-file")
	}
	if opts.MappingFile != "" {
		if err := validateReadableFile(opts.MappingFile, "mapping file"); err != nil {
			return publishOptions{}, err
		}
		if opts.MappingType == "" {
			opts.MappingType = mappingTypeProguard
		}
		if opts.MappingType != mappingTypeProguard && opts.MappingType != mappingTypeNative {
			return publishOptions{}, shared.UsageErrorf("--mapping-type must be one of: %s, %s", mappingTypeProguard, mappingTypeNative)
		}
	}

	if opts.WaitTimeout <= 0 {
		return publishOptions{}, shared.UsageErrorf("--wait-timeout must be greater than zero")
	}
	if opts.WaitInterval <= 0 {
		return publishOptions{}, shared.UsageErrorf("--wait-interval must be greater than zero")
	}

	_, err = shared.ParseReleaseNotesInput(opts.ReleaseNotesFile, opts.ReleaseNotesText, opts.ReleaseNotesLoc, os.ReadFile)
	if err != nil {
		return publishOptions{}, err
	}

	return opts, nil
}

func executePublish(parentCtx, requestCtx context.Context, deps Deps, client Client, opts publishOptions) (publishResult, error) {
	result := publishResult{
		PackageName: opts.PackageName,
		Track:       opts.Track,
		Status:      statusFailed,
		Steps:       make([]stepResult, 0, 8),
		Wait:        waitResult{Status: "skipped"},
	}

	fail := func(stepName string, err error) (publishResult, error) {
		result.Steps = append(result.Steps, stepResult{Name: stepName, Status: "error", Error: err.Error()})
		if result.EditID != "" && opts.CleanupOnFailure && !result.Committed {
			cleanupCtx, cleanupCancel := shared.ContextWithTimeout(parentCtx, shared.ActiveGlobalFlags().Timeout)
			cleanupErr := client.DeleteEdit(cleanupCtx, opts.PackageName, result.EditID)
			cleanupCancel()
			if cleanupErr != nil {
				result.Steps = append(result.Steps, stepResult{Name: "cleanup_delete_edit", Status: "error", Error: cleanupErr.Error()})
			} else {
				result.CleanupPerformed = true
				result.Steps = append(result.Steps, stepResult{Name: "cleanup_delete_edit", Status: "ok"})
			}
		}
		return result, err
	}

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return fail("create_edit", fmt.Errorf("failed to create edit: %w", err))
	}
	result.EditID = edit.ID
	result.Steps = append(result.Steps, stepResult{Name: "create_edit", Status: "ok"})

	artifactType := artifactTypeAPK
	artifactPath := opts.APKPath
	if opts.AABPath != "" {
		artifactType = artifactTypeAAB
		artifactPath = opts.AABPath
	}

	uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(parentCtx, shared.ActiveGlobalFlags().UploadTimeout)
	versionCode, err := uploadArtifact(uploadCtx, client, opts.PackageName, result.EditID, artifactType, artifactPath)
	uploadCancel()
	if err != nil {
		return fail("upload_artifact", err)
	}
	result.UploadedArtifactType = artifactType
	result.VersionCode = versionCode
	result.Steps = append(result.Steps, stepResult{Name: "upload_artifact", Status: "ok"})

	if opts.MappingFile != "" {
		mapCtx, mapCancel := shared.ContextWithUploadTimeout(parentCtx, shared.ActiveGlobalFlags().UploadTimeout)
		_, err := client.UploadDeobfuscationFile(mapCtx, opts.PackageName, result.EditID, versionCode, opts.MappingType, opts.MappingFile)
		mapCancel()
		if err != nil {
			return fail("upload_mapping", fmt.Errorf("failed to upload mapping file: %w", err))
		}
		result.Steps = append(result.Steps, stepResult{Name: "upload_mapping", Status: "ok"})
	} else {
		result.Steps = append(result.Steps, stepResult{Name: "upload_mapping", Status: "skipped"})
	}

	if artifactType == artifactTypeAAB && !opts.DryRun {
		wait, err := waitForGeneratedAPKs(parentCtx, deps, client, opts.PackageName, versionCode, opts.WaitTimeout, opts.WaitInterval)
		result.Wait = wait
		if err != nil {
			return fail("wait_for_processing", fmt.Errorf("failed to wait for bundle processing: %w", err))
		}
		result.Steps = append(result.Steps, stepResult{Name: "wait_for_processing", Status: "ok", Detail: fmt.Sprintf("attempts=%d", wait.Attempts)})
	} else {
		if opts.DryRun {
			result.Wait = waitResult{Status: "skipped", Detail: "dry-run skips generated APK wait"}
		} else {
			result.Wait = waitResult{Status: "skipped", Detail: "apk does not require generated APK wait"}
		}
		result.Steps = append(result.Steps, stepResult{Name: "wait_for_processing", Status: "skipped", Detail: result.Wait.Detail})
	}

	releaseNotes, err := shared.ParseReleaseNotesInput(opts.ReleaseNotesFile, opts.ReleaseNotesText, opts.ReleaseNotesLoc, os.ReadFile)
	if err != nil {
		return fail("parse_release_notes", err)
	}

	_, err = client.UpdateTrack(requestCtx, opts.PackageName, result.EditID, opts.Track, gpc.TrackUpdate{
		Status:         opts.ReleaseStatus,
		ReleaseName:    strings.TrimSpace(opts.ReleaseName),
		UserFraction:   opts.UserFraction,
		VersionCodes:   []int64{versionCode},
		UpdatePriority: opts.UpdatePriority,
		ReleaseNotes:   releaseNotes,
	})
	if err != nil {
		return fail("update_track", fmt.Errorf("failed to update track: %w", err))
	}
	result.Steps = append(result.Steps, stepResult{Name: "update_track", Status: "ok"})

	if err := client.ValidateEdit(requestCtx, opts.PackageName, result.EditID); err != nil {
		return fail("validate_edit", fmt.Errorf("failed to validate edit: %w", err))
	}
	result.Steps = append(result.Steps, stepResult{Name: "validate_edit", Status: "ok"})

	if opts.DryRun {
		if err := client.DeleteEdit(requestCtx, opts.PackageName, result.EditID); err != nil {
			return fail("delete_edit_dry_run", fmt.Errorf("failed to delete dry-run edit: %w", err))
		}
		result.CleanupPerformed = true
		result.Status = statusDryRun
		result.Steps = append(result.Steps, stepResult{Name: "delete_edit_dry_run", Status: "ok"})
		return result, nil
	}

	if _, err := client.CommitEdit(requestCtx, opts.PackageName, result.EditID); err != nil {
		return fail("commit_edit", fmt.Errorf("failed to commit edit: %w", err))
	}
	result.Committed = true
	result.Status = statusCommitted
	result.Steps = append(result.Steps, stepResult{Name: "commit_edit", Status: "ok"})
	return result, nil
}

func uploadArtifact(ctx context.Context, client Client, packageName, editID, artifactType, artifactPath string) (int64, error) {
	switch artifactType {
	case artifactTypeAAB:
		bundle, err := client.UploadBundle(ctx, packageName, editID, artifactPath)
		if err != nil {
			return 0, fmt.Errorf("failed to upload aab: %w", err)
		}
		if bundle.VersionCode <= 0 {
			return 0, fmt.Errorf("uploaded aab did not return a valid version code")
		}
		return bundle.VersionCode, nil
	case artifactTypeAPK:
		apk, err := client.UploadAPK(ctx, packageName, editID, artifactPath)
		if err != nil {
			return 0, fmt.Errorf("failed to upload apk: %w", err)
		}
		if apk.VersionCode <= 0 {
			return 0, fmt.Errorf("uploaded apk did not return a valid version code")
		}
		return apk.VersionCode, nil
	default:
		return 0, fmt.Errorf("unsupported artifact type %q", artifactType)
	}
}

func waitForGeneratedAPKs(ctx context.Context, deps Deps, client Client, packageName string, versionCode int64, timeout, interval time.Duration) (waitResult, error) {
	start := deps.Now()
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := waitResult{
		Status:   "waiting",
		Interval: interval.String(),
	}

	for {
		result.Attempts++
		resp, err := client.ListGeneratedAPKs(waitCtx, packageName, versionCode)
		if err == nil && resp != nil && len(resp.GeneratedApks) > 0 {
			result.Status = "ready"
			result.Elapsed = deps.Now().Sub(start).String()
			result.GeneratedApkCount = len(resp.GeneratedApks)
			return result, nil
		}
		if waitCtx.Err() != nil {
			result.Status = "timeout"
			result.Elapsed = deps.Now().Sub(start).String()
			return result, fmt.Errorf("timed out waiting for generated apks for versionCode %d", versionCode)
		}
		if err != nil {
			return result, fmt.Errorf("failed to list generated apks: %w", err)
		}
		if err := deps.Sleep(waitCtx, interval); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				result.Status = "timeout"
				result.Elapsed = deps.Now().Sub(start).String()
				return result, fmt.Errorf("timed out waiting for generated apks for versionCode %d", versionCode)
			}
			return result, err
		}
	}
}

func validateReadableFile(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return shared.UsageErrorf("%s does not exist: %s", label, path)
		}
		return fmt.Errorf("failed to stat %s: %w", label, err)
	}
	if info.IsDir() {
		return shared.UsageErrorf("%s must be a file, got directory: %s", label, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return shared.UsageErrorf("%s is not readable: %v", label, err)
	}
	return f.Close()
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
