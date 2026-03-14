package release

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	notesgen "github.com/leszko11/google-play-console-cli/internal/release/notes"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const maxAndroidVersionCode = 2_100_000_000

type alphaStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

type alphaResult struct {
	PackageName          string        `json:"packageName"`
	Track                string        `json:"track"`
	ReleaseStatus        string        `json:"releaseStatus"`
	Status               string        `json:"status"`
	RequestedVersionCode int64         `json:"requestedVersionCode,omitempty"`
	UploadedVersionCode  int64         `json:"uploadedVersionCode,omitempty"`
	VersionName          string        `json:"versionName,omitempty"`
	ArtifactPath         string        `json:"artifactPath,omitempty"`
	ReleaseNotesSource   string        `json:"releaseNotesSource,omitempty"`
	ReleaseNotesLocale   string        `json:"releaseNotesLocale,omitempty"`
	Committed            bool          `json:"committed"`
	CleanupPerformed     bool          `json:"cleanupPerformed"`
	Steps                []alphaStep   `json:"steps"`
	Verify               *verifyResult `json:"verify,omitempty"`
}

type alphaOptions struct {
	PackageName      string
	Track            string
	ReleaseStatus    string
	ProjectDir       string
	BuildTask        string
	AABPath          string
	SkipBuild        bool
	Confirm          bool
	DryRun           bool
	AllowProduction  bool
	CleanupOnFailure bool
	UserFraction     float64
	UpdatePriority   int64
	ReleaseName      string
	VersionCode      int64
	VersionName      string
	NotesMode        string
	NotesFile        string
	NotesLocale      string
	NotesText        string
	ProbeTrack       bool
}

func newAlphaCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("alpha", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts alphaOptions
	fs.StringVar(&opts.PackageName, "package-name", defaultStagingPackage, "Target package name")
	fs.StringVar(&opts.Track, "track", defaultTrack, "Target track name")
	fs.StringVar(&opts.ReleaseStatus, "status", defaultReleaseStatus, "Release status (draft, inProgress, halted, completed)")
	fs.StringVar(&opts.ProjectDir, "project-dir", ".", "Android project directory")
	fs.StringVar(&opts.BuildTask, "build-task", defaultBuildTask, "Gradle build task")
	fs.StringVar(&opts.AABPath, "aab", "", "Path to prebuilt .aab (optional)")
	fs.BoolVar(&opts.SkipBuild, "skip-build", false, "Skip Gradle build and use prebuilt artifact")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm committing release (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Run all steps but delete edit instead of committing")
	fs.BoolVar(&opts.AllowProduction, "allow-production", false, "Allow track=production")
	fs.BoolVar(&opts.CleanupOnFailure, "cleanup-on-failure", true, "Delete edit when deployment fails")
	fs.Float64Var(&opts.UserFraction, "user-fraction", -1, "Rollout user fraction (0-1)")
	fs.Int64Var(&opts.UpdatePriority, "update-priority", 0, "In-app update priority (0-5)")
	fs.StringVar(&opts.ReleaseName, "release-name", "", "Optional release name")
	fs.Int64Var(&opts.VersionCode, "version-code", 0, "Release versionCode override (default computed if empty)")
	fs.StringVar(&opts.VersionName, "version-name", "", "Release versionName override")
	fs.StringVar(&opts.NotesMode, "notes-mode", notesgen.ModeGit, "Release notes mode: git, file, none")
	fs.StringVar(&opts.NotesFile, "notes-file", "", "Release notes file path when notes-mode=file")
	fs.StringVar(&opts.NotesLocale, "notes-locale", notesgen.DefaultLocale, "Release notes locale")
	fs.StringVar(&opts.NotesText, "notes-text", "", "Inline release notes text override")
	fs.BoolVar(&opts.ProbeTrack, "probe-track", false, "Probe track existence during preflight verify")

	return &ffcli.Command{
		Name:      "alpha",
		ShortHelp: "Build staging AAB and deploy to alpha track in one flow",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := validateAlphaOptions(opts); err != nil {
				return err
			}

			result, err := runAlpha(ctx, deps, opts)
			_ = shared.WriteJSON(deps.Stdout, result)
			return err
		},
	}
}

func validateAlphaOptions(opts alphaOptions) error {
	if strings.TrimSpace(opts.PackageName) == "" {
		return shared.UsageErrorf("--package-name is required")
	}
	if strings.TrimSpace(opts.Track) == "" {
		return shared.UsageErrorf("--track is required")
	}
	if opts.Track == "production" && !opts.AllowProduction {
		return shared.UsageErrorf("--allow-production is required when --track=production")
	}
	if strings.TrimSpace(opts.ReleaseStatus) == "" {
		return shared.UsageErrorf("--status is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}
	if opts.UserFraction > 1 || (opts.UserFraction >= 0 && opts.UserFraction <= 0) {
		return shared.UsageErrorf("--user-fraction must be within (0,1] when set")
	}
	if opts.UpdatePriority < 0 || opts.UpdatePriority > 5 {
		return shared.UsageErrorf("--update-priority must be between 0 and 5")
	}
	return nil
}

func runAlpha(ctx context.Context, deps Deps, opts alphaOptions) (alphaResult, error) {
	result := alphaResult{
		PackageName:   strings.TrimSpace(opts.PackageName),
		Track:         strings.TrimSpace(opts.Track),
		ReleaseStatus: strings.TrimSpace(opts.ReleaseStatus),
		Status:        "failed",
		Steps:         make([]alphaStep, 0, 16),
	}
	projectDir := strings.TrimSpace(opts.ProjectDir)
	if projectDir == "" {
		projectDir = "."
	}
	buildTask := strings.TrimSpace(opts.BuildTask)
	if buildTask == "" {
		buildTask = defaultBuildTask
	}

	var currentEditID string
	fail := func(stepName string, err error) (alphaResult, error) {
		result.Steps = append(result.Steps, alphaStep{
			Name:   stepName,
			Status: "error",
			Error:  err.Error(),
		})
		if currentEditID != "" && opts.CleanupOnFailure {
			client, requestCtx, cancel, clientErr := buildClient(ctx, deps)
			if clientErr == nil {
				if cleanupErr := client.DeleteEdit(requestCtx, result.PackageName, currentEditID); cleanupErr == nil {
					result.CleanupPerformed = true
					result.Steps = append(result.Steps, alphaStep{Name: "cleanup_delete_edit", Status: "ok"})
				} else {
					result.Steps = append(result.Steps, alphaStep{Name: "cleanup_delete_edit", Status: "error", Error: cleanupErr.Error()})
				}
				cancel()
			}
		}
		return result, err
	}

	verify, _ := runVerify(ctx, deps, verifyOptions{
		PackageName: result.PackageName,
		Track:       result.Track,
		ProjectDir:  projectDir,
		BuildTask:   buildTask,
		AABPath:     strings.TrimSpace(opts.AABPath),
		ProbeTrack:  opts.ProbeTrack,
		NotesMode:   opts.NotesMode,
		NotesFile:   opts.NotesFile,
		NotesLocale: opts.NotesLocale,
		NotesText:   opts.NotesText,
	})
	result.Verify = &verify
	if verify.Status != "ok" {
		return fail("preflight_verify", fmt.Errorf("release verification failed"))
	}
	result.Steps = append(result.Steps, alphaStep{Name: "preflight_verify", Status: "ok"})

	versionCode, versionName, err := resolveVersionInfo(opts, deps.LookupEnv, deps.Now)
	if err != nil {
		return fail("resolve_version", err)
	}
	result.RequestedVersionCode = versionCode
	result.VersionName = versionName
	result.Steps = append(result.Steps, alphaStep{
		Name:   "resolve_version",
		Status: "ok",
		Detail: fmt.Sprintf("versionCode=%d", versionCode),
	})

	if !opts.SkipBuild {
		buildArgs := []string{}
		buildArgs = append(buildArgs, fmt.Sprintf("APP_VERSION_CODE=%d", versionCode))
		if versionName != "" {
			buildArgs = append(buildArgs, fmt.Sprintf("APP_VERSION_NAME=%s", versionName))
		}
		buildArgs = append(buildArgs, "./gradlew", buildTask, "--stacktrace", "--no-daemon")
		if _, err := deps.RunCommand(ctx, projectDir, "env", buildArgs...); err != nil {
			return fail("build_artifact", fmt.Errorf("Gradle build failed: %w", err))
		}
		result.Steps = append(result.Steps, alphaStep{Name: "build_artifact", Status: "ok"})
	} else {
		result.Steps = append(result.Steps, alphaStep{Name: "build_artifact", Status: "skipped", Detail: "--skip-build enabled"})
	}

	artifactPath, err := resolveAABPath(projectDir, strings.TrimSpace(opts.AABPath))
	if err != nil {
		return fail("resolve_artifact", err)
	}
	result.ArtifactPath = artifactPath
	result.Steps = append(result.Steps, alphaStep{Name: "resolve_artifact", Status: "ok", Detail: artifactPath})

	notesResult, err := notesgen.Generate(notesgen.Input{
		Mode:     opts.NotesMode,
		RepoDir:  projectDir,
		FilePath: opts.NotesFile,
		Locale:   opts.NotesLocale,
		Text:     opts.NotesText,
	}, notesgen.Deps{})
	if err != nil {
		return fail("generate_notes", err)
	}
	result.ReleaseNotesSource = notesResult.Source
	result.ReleaseNotesLocale = notesResult.Locale
	result.Steps = append(result.Steps, alphaStep{Name: "generate_notes", Status: "ok", Detail: notesResult.Source})

	parsedNotes, err := notesgen.ParseLocalizedText(notesResult.Text, notesResult.Locale)
	if err != nil {
		return fail("generate_notes", err)
	}
	releaseNotes := make([]gpc.LocalizedReleaseNote, 0, len(parsedNotes))
	for _, note := range parsedNotes {
		releaseNotes = append(releaseNotes, gpc.LocalizedReleaseNote{
			Language: note.Locale,
			Text:     note.Text,
		})
	}

	client, requestCtx, cancel, err := buildClient(ctx, deps)
	if err != nil {
		return fail("create_client", err)
	}
	defer cancel()

	edit, err := client.CreateEdit(requestCtx, result.PackageName)
	if err != nil {
		return fail("create_edit", fmt.Errorf("failed to create edit: %w", err))
	}
	currentEditID = edit.ID
	result.Steps = append(result.Steps, alphaStep{Name: "create_edit", Status: "ok", Detail: currentEditID})

	uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(ctx, shared.ActiveGlobalFlags().UploadTimeout)
	bundleInfo, err := client.UploadBundle(uploadCtx, result.PackageName, currentEditID, artifactPath)
	uploadCancel()
	if err != nil {
		return fail("upload_bundle", fmt.Errorf("failed to upload bundle: %w", err))
	}
	if bundleInfo.VersionCode <= 0 {
		return fail("upload_bundle", fmt.Errorf("uploaded bundle did not return a valid versionCode"))
	}
	result.UploadedVersionCode = bundleInfo.VersionCode
	result.Steps = append(result.Steps, alphaStep{Name: "upload_bundle", Status: "ok", Detail: fmt.Sprintf("versionCode=%d", bundleInfo.VersionCode)})

	if _, err := client.UpdateTrack(requestCtx, result.PackageName, currentEditID, result.Track, gpc.TrackUpdate{
		Status:         result.ReleaseStatus,
		ReleaseName:    strings.TrimSpace(opts.ReleaseName),
		UserFraction:   opts.UserFraction,
		VersionCodes:   []int64{bundleInfo.VersionCode},
		UpdatePriority: opts.UpdatePriority,
		ReleaseNotes:   releaseNotes,
	}); err != nil {
		return fail("update_track", fmt.Errorf("failed to update track: %w", err))
	}
	result.Steps = append(result.Steps, alphaStep{Name: "update_track", Status: "ok"})

	if err := client.ValidateEdit(requestCtx, result.PackageName, currentEditID); err != nil {
		return fail("validate_edit", fmt.Errorf("failed to validate edit: %w", err))
	}
	result.Steps = append(result.Steps, alphaStep{Name: "validate_edit", Status: "ok"})

	if opts.DryRun {
		if err := client.DeleteEdit(requestCtx, result.PackageName, currentEditID); err != nil {
			return fail("delete_edit_dry_run", fmt.Errorf("failed to delete dry-run edit: %w", err))
		}
		currentEditID = ""
		result.Status = "dry-run"
		result.Steps = append(result.Steps, alphaStep{Name: "delete_edit_dry_run", Status: "ok"})
		return result, nil
	}

	if _, err := client.CommitEdit(requestCtx, result.PackageName, currentEditID); err != nil {
		return fail("commit_edit", fmt.Errorf("failed to commit edit: %w", err))
	}
	currentEditID = ""
	result.Committed = true
	result.Status = "committed"
	result.Steps = append(result.Steps, alphaStep{Name: "commit_edit", Status: "ok"})

	if err := verifyTrackContainsVersion(ctx, client, result.PackageName, result.Track, bundleInfo.VersionCode); err != nil {
		return fail("post_deploy_verify", err)
	}
	result.Steps = append(result.Steps, alphaStep{Name: "post_deploy_verify", Status: "ok"})

	return result, nil
}

func resolveVersionInfo(opts alphaOptions, lookupEnv func(string) string, now func() time.Time) (int64, string, error) {
	versionName := strings.TrimSpace(opts.VersionName)
	if versionName == "" && lookupEnv != nil {
		versionName = strings.TrimSpace(lookupEnv("APP_VERSION_NAME"))
	}

	if opts.VersionCode > 0 {
		if opts.VersionCode > maxAndroidVersionCode {
			return 0, "", shared.UsageErrorf("--version-code exceeds Play maximum (%d)", maxAndroidVersionCode)
		}
		return opts.VersionCode, versionName, nil
	}

	if lookupEnv != nil {
		if raw := strings.TrimSpace(lookupEnv("APP_VERSION_CODE")); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return 0, "", fmt.Errorf("invalid APP_VERSION_CODE: %w", err)
			}
			if parsed <= 0 || parsed > maxAndroidVersionCode {
				return 0, "", fmt.Errorf("APP_VERSION_CODE must be within 1..%d", maxAndroidVersionCode)
			}
			return parsed, versionName, nil
		}
	}

	ts := time.Now
	if now != nil {
		ts = now
	}
	computed := ts().UTC().Unix()
	if computed <= 0 || computed > maxAndroidVersionCode {
		return 0, "", fmt.Errorf("computed versionCode %d is outside 1..%d", computed, maxAndroidVersionCode)
	}
	return computed, versionName, nil
}

func resolveAABPath(projectDir, explicitPath string) (string, error) {
	if explicitPath != "" {
		if err := validateExistingFile(explicitPath, "aab"); err != nil {
			return "", err
		}
		return explicitPath, nil
	}

	base := filepath.Join(projectDir, "app", "build", "outputs", "bundle", "stagingRelease")
	if latest, err := latestArtifact(base, ".aab"); err == nil {
		return latest, nil
	}

	fallbackBase := filepath.Join(projectDir, "app", "build", "outputs", "bundle")
	latest, err := latestArtifact(fallbackBase, ".aab")
	if err != nil {
		return "", fmt.Errorf("no AAB artifact found under %s", fallbackBase)
	}
	return latest, nil
}

func latestArtifact(root, suffix string) (string, error) {
	var newestPath string
	var newestTime time.Time
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), strings.ToLower(suffix)) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if newestPath == "" || info.ModTime().After(newestTime) {
			newestPath = path
			newestTime = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if newestPath == "" {
		return "", fmt.Errorf("no artifact with suffix %s found", suffix)
	}
	return newestPath, nil
}

func validateExistingFile(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s does not exist: %s", label, path)
		}
		return fmt.Errorf("failed to stat %s: %w", label, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s must be a file, got directory: %s", label, path)
	}
	return nil
}

func verifyTrackContainsVersion(ctx context.Context, client Client, packageName, track string, versionCode int64) error {
	requestCtx, cancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
	defer cancel()

	edit, err := client.CreateEdit(requestCtx, packageName)
	if err != nil {
		return fmt.Errorf("post-deploy verification failed to create edit: %w", err)
	}
	defer func() {
		_ = client.DeleteEdit(requestCtx, packageName, edit.ID)
	}()

	trackInfo, err := client.GetTrack(requestCtx, packageName, edit.ID, track)
	if err != nil {
		return fmt.Errorf("post-deploy verification failed to read track %q: %w", track, err)
	}
	for _, release := range trackInfo.Releases {
		for _, code := range release.VersionCodes {
			if code == versionCode {
				return nil
			}
		}
	}
	return fmt.Errorf("post-deploy verification failed: versionCode %d not found on track %q", versionCode, track)
}
