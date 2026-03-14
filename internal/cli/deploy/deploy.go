package deploy

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
	artifactTypeAAB       = "aab"
	artifactTypeAPK       = "apk"
	deployStatusCommitted = "committed"
	deployStatusDryRun    = "dry-run"
	deployStatusFailed    = "failed"
	mappingTypeProguard   = "proguard"
	mappingTypeNativeCode = "nativeCode"
	defaultNotesLocale    = "en-US"
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

type deployResult struct {
	PackageName          string       `json:"packageName"`
	EditID               string       `json:"editId,omitempty"`
	UploadedArtifactType string       `json:"uploadedArtifactType,omitempty"`
	VersionCode          int64        `json:"versionCode,omitempty"`
	Track                string       `json:"track"`
	Status               string       `json:"status"`
	Steps                []stepResult `json:"steps"`
	Committed            bool         `json:"committed"`
	CleanupPerformed     bool         `json:"cleanupPerformed"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var (
		packageName        string
		aabPath            string
		apkPath            string
		track              string
		releaseStatus      string
		releaseName        string
		userFraction       float64
		updatePriority     int64
		mappingFile        string
		mappingType        string
		releaseNotesFile   string
		releaseNotesLocale string
		releaseNotesText   string
		confirm            bool
		allowProduction    bool
		cleanupOnFailure   bool
		dryRun             bool
	)

	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&aabPath, "aab", "", "Path to .aab file")
	fs.StringVar(&apkPath, "apk", "", "Path to .apk file")
	fs.StringVar(&track, "track", "", "Track name (e.g. internal, production)")
	fs.StringVar(&releaseStatus, "status", "", "Release status (draft, inProgress, halted, completed)")
	fs.StringVar(&releaseName, "release-name", "", "Release name")
	fs.Float64Var(&userFraction, "user-fraction", -1, "Rollout user fraction (0-1)")
	fs.Int64Var(&updatePriority, "update-priority", 0, "In-app update priority (0-5)")
	fs.StringVar(&mappingFile, "mapping-file", "", "Path to deobfuscation mapping file")
	fs.StringVar(&mappingType, "mapping-type", "", "Mapping type: proguard or nativeCode (defaults to proguard)")
	fs.StringVar(&releaseNotesFile, "release-notes-file", "", "Path to release notes file (JSON object/array, tagged blocks, or plain text)")
	fs.StringVar(&releaseNotesLocale, "release-notes-locale", defaultNotesLocale, "Release notes locale (BCP-47)")
	fs.StringVar(&releaseNotesText, "release-notes-text", "", "Release notes text")
	fs.BoolVar(&confirm, "confirm", false, "Confirm committing the edit (required unless --dry-run)")
	fs.BoolVar(&allowProduction, "allow-production", false, "Allow deploys to production track")
	fs.BoolVar(&cleanupOnFailure, "cleanup-on-failure", true, "Delete edit if deploy fails")
	fs.BoolVar(&dryRun, "dry-run", false, "Run deploy steps, then delete edit instead of committing")

	return &ffcli.Command{
		Name:      "deploy",
		ShortHelp: "Upload artifact and publish to a track in one flow",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			params, err := validateFlags(
				packageName,
				aabPath,
				apkPath,
				track,
				releaseStatus,
				releaseName,
				userFraction,
				updatePriority,
				mappingFile,
				mappingType,
				releaseNotesFile,
				releaseNotesLocale,
				releaseNotesText,
				confirm,
				allowProduction,
				cleanupOnFailure,
				dryRun,
			)
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

			spinner := shared.NewSpinner(deps.Stderr, "Running deploy flow")
			err = executeDeploy(ctx, requestCtx, client, deps.Stdout, params)
			if err != nil {
				spinner.Fail("Deploy flow failed")
				return err
			}
			spinner.Success("Deploy flow finished")
			return nil
		},
	}
}

type deployParams struct {
	PackageName      string
	ArtifactType     string
	ArtifactPath     string
	Track            string
	ReleaseStatus    string
	ReleaseName      string
	UserFraction     float64
	UpdatePriority   int64
	MappingFile      string
	MappingType      string
	ReleaseNotes     []gpc.LocalizedText
	Confirm          bool
	CleanupOnFailure bool
	DryRun           bool
}

func validateFlags(
	packageName,
	aabPath,
	apkPath,
	track,
	releaseStatus,
	releaseName string,
	userFraction float64,
	updatePriority int64,
	mappingFile,
	mappingType,
	releaseNotesFile,
	releaseNotesLocale,
	releaseNotesText string,
	confirm,
	allowProduction,
	cleanupOnFailure,
	dryRun bool,
) (deployParams, error) {
	pkg, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return deployParams{}, err
	}

	aabPath = strings.TrimSpace(aabPath)
	apkPath = strings.TrimSpace(apkPath)
	if (aabPath == "" && apkPath == "") || (aabPath != "" && apkPath != "") {
		return deployParams{}, shared.UsageErrorf("exactly one of --aab or --apk is required")
	}

	artifactType := artifactTypeAAB
	artifactPath := aabPath
	if apkPath != "" {
		artifactType = artifactTypeAPK
		artifactPath = apkPath
	}
	if err := validateReadableFile(artifactPath, "artifact"); err != nil {
		return deployParams{}, err
	}

	track = strings.TrimSpace(track)
	if track == "" {
		return deployParams{}, shared.UsageErrorf("--track is required")
	}
	if track == "production" && !allowProduction {
		return deployParams{}, shared.UsageErrorf("--allow-production is required when --track=production")
	}

	releaseStatus = strings.TrimSpace(releaseStatus)
	if releaseStatus == "" {
		return deployParams{}, shared.UsageErrorf("--status is required")
	}

	if !dryRun && !confirm {
		return deployParams{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}

	if userFraction > 1 || (userFraction >= 0 && userFraction <= 0) {
		return deployParams{}, shared.UsageErrorf("--user-fraction must be within (0,1] when set")
	}
	if updatePriority < 0 || updatePriority > 5 {
		return deployParams{}, shared.UsageErrorf("--update-priority must be between 0 and 5")
	}

	mappingFile = strings.TrimSpace(mappingFile)
	mappingType = strings.TrimSpace(mappingType)
	if mappingFile == "" && mappingType != "" {
		return deployParams{}, shared.UsageErrorf("--mapping-type requires --mapping-file")
	}
	if mappingFile != "" {
		if err := validateReadableFile(mappingFile, "mapping file"); err != nil {
			return deployParams{}, err
		}
		if mappingType == "" {
			mappingType = mappingTypeProguard
		}
		if mappingType != mappingTypeProguard && mappingType != mappingTypeNativeCode {
			return deployParams{}, shared.UsageErrorf("--mapping-type must be one of: %s, %s", mappingTypeProguard, mappingTypeNativeCode)
		}
	}

	releaseNotes, err := shared.ParseReleaseNotesInput(
		releaseNotesFile,
		releaseNotesText,
		releaseNotesLocale,
		os.ReadFile,
	)
	if err != nil {
		return deployParams{}, err
	}

	return deployParams{
		PackageName:      pkg,
		ArtifactType:     artifactType,
		ArtifactPath:     artifactPath,
		Track:            track,
		ReleaseStatus:    releaseStatus,
		ReleaseName:      strings.TrimSpace(releaseName),
		UserFraction:     userFraction,
		UpdatePriority:   updatePriority,
		MappingFile:      mappingFile,
		MappingType:      mappingType,
		ReleaseNotes:     releaseNotes,
		Confirm:          confirm,
		CleanupOnFailure: cleanupOnFailure,
		DryRun:           dryRun,
	}, nil
}

func executeDeploy(parentCtx, requestCtx context.Context, client Client, out io.Writer, p deployParams) error {
	result := deployResult{
		PackageName: p.PackageName,
		Track:       p.Track,
		Status:      deployStatusFailed,
		Steps:       make([]stepResult, 0, 8),
	}

	fail := func(stepName string, err error) error {
		result.Steps = append(result.Steps, stepResult{Name: stepName, Status: "error", Error: err.Error()})
		if result.EditID != "" && p.CleanupOnFailure {
			cleanupCtx, cleanupCancel := shared.ContextWithTimeout(parentCtx, shared.ActiveGlobalFlags().Timeout)
			cleanupErr := client.DeleteEdit(cleanupCtx, p.PackageName, result.EditID)
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

	edit, err := client.CreateEdit(requestCtx, p.PackageName)
	if err != nil {
		return fail("create_edit", fmt.Errorf("failed to create edit: %w", err))
	}
	result.EditID = edit.ID
	result.Steps = append(result.Steps, stepResult{Name: "create_edit", Status: "ok"})

	uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(parentCtx, shared.ActiveGlobalFlags().UploadTimeout)
	versionCode, err := uploadArtifact(uploadCtx, client, p, result.EditID)
	uploadCancel()
	if err != nil {
		return fail("upload_artifact", err)
	}
	result.UploadedArtifactType = p.ArtifactType
	result.VersionCode = versionCode
	result.Steps = append(result.Steps, stepResult{Name: "upload_artifact", Status: "ok"})

	if p.MappingFile != "" {
		uploadMapCtx, mapCancel := shared.ContextWithUploadTimeout(parentCtx, shared.ActiveGlobalFlags().UploadTimeout)
		_, err := client.UploadDeobfuscationFile(uploadMapCtx, p.PackageName, result.EditID, versionCode, p.MappingType, p.MappingFile)
		mapCancel()
		if err != nil {
			return fail("upload_mapping", fmt.Errorf("failed to upload mapping file: %w", err))
		}
		result.Steps = append(result.Steps, stepResult{Name: "upload_mapping", Status: "ok"})
	} else {
		result.Steps = append(result.Steps, stepResult{Name: "upload_mapping", Status: "skipped"})
	}

	_, err = client.UpdateTrack(requestCtx, p.PackageName, result.EditID, p.Track, gpc.TrackUpdate{
		Status:         p.ReleaseStatus,
		ReleaseName:    p.ReleaseName,
		UserFraction:   p.UserFraction,
		VersionCodes:   []int64{versionCode},
		UpdatePriority: p.UpdatePriority,
		ReleaseNotes:   p.ReleaseNotes,
	})
	if err != nil {
		return fail("update_track", fmt.Errorf("failed to update track: %w", err))
	}
	result.Steps = append(result.Steps, stepResult{Name: "update_track", Status: "ok"})

	if err := client.ValidateEdit(requestCtx, p.PackageName, result.EditID); err != nil {
		return fail("validate_edit", fmt.Errorf("failed to validate edit: %w", err))
	}
	result.Steps = append(result.Steps, stepResult{Name: "validate_edit", Status: "ok"})

	if p.DryRun {
		if err := client.DeleteEdit(requestCtx, p.PackageName, result.EditID); err != nil {
			return fail("delete_edit_dry_run", fmt.Errorf("failed to delete dry-run edit: %w", err))
		}
		result.Status = deployStatusDryRun
		result.Steps = append(result.Steps, stepResult{Name: "delete_edit_dry_run", Status: "ok"})
		return shared.WriteJSON(out, result)
	}

	if _, err := client.CommitEdit(requestCtx, p.PackageName, result.EditID); err != nil {
		return fail("commit_edit", fmt.Errorf("failed to commit edit: %w", err))
	}
	result.Committed = true
	result.Status = deployStatusCommitted
	result.Steps = append(result.Steps, stepResult{Name: "commit_edit", Status: "ok"})

	return shared.WriteJSON(out, result)
}

func uploadArtifact(ctx context.Context, client Client, p deployParams, editID string) (int64, error) {
	switch p.ArtifactType {
	case artifactTypeAAB:
		bundle, err := client.UploadBundle(ctx, p.PackageName, editID, p.ArtifactPath)
		if err != nil {
			return 0, fmt.Errorf("failed to upload aab: %w", err)
		}
		if bundle.VersionCode <= 0 {
			return 0, fmt.Errorf("uploaded aab did not return a valid version code")
		}
		return bundle.VersionCode, nil
	case artifactTypeAPK:
		apk, err := client.UploadAPK(ctx, p.PackageName, editID, p.ArtifactPath)
		if err != nil {
			return 0, fmt.Errorf("failed to upload apk: %w", err)
		}
		if apk.VersionCode <= 0 {
			return 0, fmt.Errorf("uploaded apk did not return a valid version code")
		}
		return apk.VersionCode, nil
	default:
		return 0, fmt.Errorf("unsupported artifact type %q", p.ArtifactType)
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
