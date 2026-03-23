package release

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/listing"
	screenshotscmd "github.com/leszko11/google-play-console-cli/internal/cli/screenshots"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
	"gopkg.in/yaml.v3"
)

type initOptions struct {
	PackageName        string
	Dir                string
	Guided             bool
	Track              string
	DefaultLocale      string
	AndroidProjectDir  string
	BuildTask          string
	ArtifactPath       string
	NotesFile          string
	StrictExport       bool
	WriteProjectConfig bool
}

type initStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

type initResult struct {
	Status                 string      `json:"status"`
	PackageName            string      `json:"packageName"`
	PackageReadiness       string      `json:"packageReadiness,omitempty"`
	WorkspaceReadiness     string      `json:"workspaceReadiness,omitempty"`
	BootstrapDraftExists   bool        `json:"bootstrapDraftExists,omitempty"`
	BootstrapVersionCodes  []int64     `json:"bootstrapVersionCodes,omitempty"`
	LastKnownReadiness     string      `json:"lastKnownReadiness,omitempty"`
	RecommendedNextCommand string      `json:"recommendedNextCommand,omitempty"`
	WorkspaceDir           string      `json:"workspaceDir,omitempty"`
	ProjectConfigPath      string      `json:"projectConfigPath,omitempty"`
	ReleaseManifest        string      `json:"releaseManifest,omitempty"`
	WorkflowPath           string      `json:"workflowPath,omitempty"`
	ManualBridgePath       string      `json:"manualBridgePath,omitempty"`
	BootstrapNotePath      string      `json:"bootstrapNotePath,omitempty"`
	BootstrapStatePath     string      `json:"bootstrapStatePath,omitempty"`
	Issues                 []initIssue `json:"issues,omitempty"`
	Steps                  []initStep  `json:"steps"`
	Warnings               []string    `json:"warnings,omitempty"`
	NextSteps              []string    `json:"nextSteps,omitempty"`
}

type initIssue struct {
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	Detail   string `json:"detail"`
	NextStep string `json:"nextStep,omitempty"`
	Blocking bool   `json:"blocking,omitempty"`
}

func newInitCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts initOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Target package name")
	fs.StringVar(&opts.Dir, "dir", "./play", "Workspace directory to generate")
	fs.BoolVar(&opts.Guided, "guided", false, "Prompt for missing values when running in a TTY")
	fs.StringVar(&opts.Track, "track", "internal", "Default non-production release track")
	fs.StringVar(&opts.DefaultLocale, "default-locale", "", "Default store locale")
	fs.StringVar(&opts.AndroidProjectDir, "android-project-dir", "", "Android project directory")
	fs.StringVar(&opts.BuildTask, "build-task", "", "Gradle build task used for release verification")
	fs.StringVar(&opts.ArtifactPath, "artifact-path", "", "Release artifact path")
	fs.StringVar(&opts.NotesFile, "notes-file", "", "Release notes file path")
	fs.BoolVar(&opts.StrictExport, "strict-export", false, "Fail when exported release content is not ready to release")
	fs.BoolVar(&opts.WriteProjectConfig, "write-project-config", true, "Write .gpc.yaml in the repo root")

	return &ffcli.Command{
		Name:      "init",
		ShortHelp: "Initialize a local Android release workspace from package state",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			res, err := runInit(ctx, deps, opts)
			if writeErr := shared.WriteJSON(deps.Stdout, res); writeErr != nil {
				return writeErr
			}
			return err
		},
	}
}

func runInit(ctx context.Context, deps Deps, opts initOptions) (initResult, error) {
	res := initResult{
		Status: "failed",
		Steps:  make([]initStep, 0, 8),
	}
	var err error
	opts, err = validateInitOptions(deps, opts)
	if err != nil {
		return res, err
	}
	res.PackageName = opts.PackageName
	res.WorkspaceDir = opts.Dir
	res.BootstrapStatePath = shared.BootstrapStatePathForWorkspace(opts.Dir)

	cfg, err := deps.LoadConfig()
	if err != nil {
		return res, err
	}
	authStatus := shared.BuildAuthStatusSnapshot(cfg, deps.LookupEnv)
	if !authStatus.Authenticated {
		res.Steps = append(res.Steps, initStep{Name: "auth", Status: "error", Error: shared.AuthStatusSummary(authStatus)})
		if authStatus.FixCommand != "" {
			res.NextSteps = append(res.NextSteps, "Run `"+authStatus.FixCommand+"`.")
		} else {
			res.NextSteps = append(res.NextSteps, "Run `gpc auth init --service-account <path> --storage path` or `gpc setup --auto --project-id <id> --package-name "+opts.PackageName+"`.")
		}
		if authStatus.DiagnosticCommand != "" {
			res.NextSteps = append(res.NextSteps, "Diagnostic: `"+authStatus.DiagnosticCommand+"`.")
		}
		return res, fmt.Errorf("authentication is required before release init")
	}
	res.Steps = append(res.Steps, initStep{Name: "auth", Status: "ok", Detail: shared.AuthStatusSummary(authStatus)})

	if err := ensureReleaseWorkspace(opts); err != nil {
		return res, err
	}
	res.Steps = append(res.Steps, initStep{Name: "workspace", Status: "ok", Detail: opts.Dir})

	client, requestCtx, cancel, err := buildClient(ctx, deps)
	if err != nil {
		res.Steps = append(res.Steps, initStep{Name: "client", Status: "error", Error: err.Error()})
		return res, err
	}
	defer cancel()

	readiness, err := shared.DetectPackageReadiness(requestCtx, client, opts.PackageName)
	if err != nil {
		res.Steps = append(res.Steps, initStep{Name: "package_readiness", Status: "error", Error: err.Error()})
		return res, err
	}
	res.PackageReadiness = string(readiness.Status)
	res.Steps = append(res.Steps, initStep{Name: "package_readiness", Status: "ok", Detail: readiness.Detail})

	if readiness.Status != shared.PackageReadinessUninitialized {
		if err := deps.RunBootstrap(ctx, []string{"--package-name", opts.PackageName, "--dir", opts.Dir}); err != nil {
			res.Steps = append(res.Steps, initStep{Name: "bootstrap_export", Status: "error", Error: err.Error()})
			return res, err
		}
		res.Steps = append(res.Steps, initStep{Name: "bootstrap_export", Status: "ok", Detail: opts.Dir})
		if err := mirrorListingScreenshots(opts.Dir); err != nil {
			return res, err
		}
	} else {
		res.Steps = append(res.Steps, initStep{Name: "bootstrap_export", Status: "skipped", Detail: "package is not initialized in Play yet"})
	}

	if err := ensureDefaultNotesFile(opts); err != nil {
		return res, err
	}
	if err := ensureReleaseManifest(opts); err != nil {
		return res, err
	}
	if err := ensureWorkflowFile(opts); err != nil {
		return res, err
	}
	if opts.WriteProjectConfig {
		if err := ensureProjectConfig(opts); err != nil {
			return res, err
		}
		res.ProjectConfigPath = filepath.Join(repoRootForWorkspace(opts.Dir), ".gpc.yaml")
	}
	res.ReleaseManifest = filepath.Join(opts.Dir, "release.yaml")
	res.WorkflowPath = filepath.Join(repoRootForWorkspace(opts.Dir), ".gpc", "workflow.yml")
	res.RecommendedNextCommand = shared.RecommendedReleaseCommand(opts.PackageName, res.PackageReadiness, opts.Dir, res.ReleaseManifest)

	stateInfo, err := shared.ReadBootstrapState(res.BootstrapStatePath)
	if err != nil {
		return res, err
	}
	if strings.TrimSpace(stateInfo.State.LastReadinessRecheck) != "" {
		res.LastKnownReadiness = stateInfo.State.LastReadinessRecheck
	}

	bootstrapDraft := shared.BootstrapDraftInfo{}
	if readiness.Status != shared.PackageReadinessUninitialized {
		bootstrapDraft, err = shared.DetectBootstrapDraftState(requestCtx, client, opts.PackageName)
		if err != nil {
			return res, err
		}
		res.BootstrapDraftExists = bootstrapDraft.Exists
		res.BootstrapVersionCodes = append([]int64(nil), bootstrapDraft.VersionCodes...)
		if !bootstrapDraft.Exists && len(stateInfo.State.BootstrapVersionCodes) > 0 {
			res.BootstrapVersionCodes = append([]int64(nil), stateInfo.State.BootstrapVersionCodes...)
		}
	}
	if res.LastKnownReadiness == "" {
		res.LastKnownReadiness = res.PackageReadiness
	}

	if readiness.Status != shared.PackageReadinessUninitialized {
		workspaceReadiness, issues, warnings := auditInitWorkspace(opts)
		res.WorkspaceReadiness = workspaceReadiness
		res.Issues = issues
		res.Warnings = append(res.Warnings, warnings...)
	}

	switch readiness.Status {
	case shared.PackageReadinessUninitialized:
		bridgePath, err := ensureManualBridge(opts)
		if err != nil {
			return res, err
		}
		res.Status = "manual_required"
		res.WorkspaceReadiness = "manual_required"
		res.ManualBridgePath = bridgePath
		res.NextSteps = append(res.NextSteps, "Follow "+bridgePath+" for the first Play Console upload.")
		res.NextSteps = append(res.NextSteps, "Then rerun `gpc release init --package-name "+opts.PackageName+" --dir "+opts.Dir+"`.")
	case shared.PackageReadinessDraftBootstrapRequired:
		res.Status = "draft_bootstrap_required"
		res.WorkspaceReadiness = "draft_bootstrap_required"
		res.Warnings = append(res.Warnings, readiness.Warning)
		notePath, err := ensureDraftBootstrapNote(opts)
		if err != nil {
			return res, err
		}
		res.BootstrapNotePath = notePath
		res.NextSteps = append(res.NextSteps, readiness.NextStep)
		res.NextSteps = append(res.NextSteps, "Review "+notePath+" for the draft bootstrap handoff and rerun guidance.")
		if res.RecommendedNextCommand != "" {
			res.NextSteps = append(res.NextSteps, "After the draft bootstrap deploy, use `"+res.RecommendedNextCommand+"`.")
		}
	default:
		res.Status = "ready"
		if res.WorkspaceReadiness == "" {
			res.WorkspaceReadiness = "ready"
		}
		if res.WorkspaceReadiness == "ready" {
			if warnings, err := summarizeWorkspaceWarnings(opts); err == nil {
				res.Warnings = append(res.Warnings, warnings...)
			}
		}
		if res.RecommendedNextCommand != "" {
			res.NextSteps = append(res.NextSteps, "Run `"+res.RecommendedNextCommand+"` for the next internal or alpha release.")
		}
	}

	nextState := stateInfo.State
	nextState.PackageName = opts.PackageName
	nextState.PackageReadiness = res.PackageReadiness
	nextState.BootstrapDraftExists = res.BootstrapDraftExists
	nextState.BootstrapVersionCodes = append([]int64(nil), res.BootstrapVersionCodes...)
	nextState.LastReadinessRecheck = res.PackageReadiness
	if err := shared.WriteBootstrapState(res.BootstrapStatePath, nextState); err != nil {
		return res, err
	}

	if opts.StrictExport && (res.WorkspaceReadiness == "needs_content" || res.WorkspaceReadiness == "needs_artifact") {
		return res, fmt.Errorf("workspace audit failed; fix the reported issues before releasing")
	}

	return res, nil
}

func validateInitOptions(deps Deps, opts initOptions) (initOptions, error) {
	opts.PackageName = strings.TrimSpace(opts.PackageName)
	opts.Dir = strings.TrimSpace(opts.Dir)
	opts.Track = strings.TrimSpace(opts.Track)
	opts.DefaultLocale = strings.TrimSpace(opts.DefaultLocale)
	opts.AndroidProjectDir = strings.TrimSpace(opts.AndroidProjectDir)
	opts.BuildTask = strings.TrimSpace(opts.BuildTask)
	opts.ArtifactPath = strings.TrimSpace(opts.ArtifactPath)
	opts.NotesFile = strings.TrimSpace(opts.NotesFile)

	if opts.Guided && !isInteractiveInput(deps.Stdin) {
		return initOptions{}, shared.UsageErrorf("--guided requires a TTY")
	}
	if opts.Guided && opts.PackageName == "" {
		value, err := promptInput(deps.Stdin, deps.Stderr, "Package name")
		if err != nil {
			return initOptions{}, err
		}
		opts.PackageName = value
	}
	if opts.PackageName == "" {
		pkg, err := shared.ResolvePackageName(opts.PackageName)
		if err == nil {
			opts.PackageName = pkg
		}
	}
	if opts.PackageName == "" {
		return initOptions{}, shared.UsageErrorf("--package-name is required")
	}
	if opts.Dir == "" {
		opts.Dir = "./play"
	}
	if opts.Track == "" {
		opts.Track = "internal"
	}
	if opts.DefaultLocale == "" {
		locale, err := shared.ResolveDefaultLocale("")
		if err == nil && strings.TrimSpace(locale) != "" {
			opts.DefaultLocale = locale
		}
	}
	if opts.DefaultLocale == "" {
		opts.DefaultLocale = "en-US"
	}
	if opts.AndroidProjectDir == "" {
		if value, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.AndroidProjectDir }); err == nil && strings.TrimSpace(value) != "" {
			opts.AndroidProjectDir = value
		} else {
			opts.AndroidProjectDir = "./android"
		}
	}
	if opts.BuildTask == "" {
		if value, err := shared.ResolveProjectValue("", func(cfg config.ProjectConfig) string { return cfg.BuildTask }); err == nil && strings.TrimSpace(value) != "" {
			opts.BuildTask = value
		} else {
			opts.BuildTask = ":app:bundleRelease"
		}
	}
	if opts.ArtifactPath == "" {
		if value, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.ArtifactPath }); err == nil && strings.TrimSpace(value) != "" {
			opts.ArtifactPath = value
		} else {
			opts.ArtifactPath = filepath.Join(opts.AndroidProjectDir, "app", "build", "outputs", "bundle", "release", "app-release.aab")
		}
	}
	if opts.NotesFile == "" {
		if value, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.NotesFile }); err == nil && strings.TrimSpace(value) != "" {
			opts.NotesFile = value
		} else {
			opts.NotesFile = filepath.Join(opts.Dir, "changelog", opts.Track, opts.DefaultLocale+".txt")
		}
	}
	return opts, nil
}

func ensureReleaseWorkspace(opts initOptions) error {
	for _, dir := range []string{
		opts.Dir,
		filepath.Join(opts.Dir, "listing"),
		filepath.Join(opts.Dir, "screenshots"),
		filepath.Join(opts.Dir, "products"),
		filepath.Join(opts.Dir, "subscriptions"),
		filepath.Join(opts.Dir, "changelog", opts.Track),
		filepath.Join(repoRootForWorkspace(opts.Dir), ".gpc"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func ensureDefaultNotesFile(opts initOptions) error {
	if err := os.MkdirAll(filepath.Dir(opts.NotesFile), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(opts.NotesFile); err == nil {
		return nil
	}
	return os.WriteFile(opts.NotesFile, []byte("Internal validation build for the gpc Android release flow.\n"), 0o600)
}

func ensureReleaseManifest(opts initOptions) error {
	manifestPath := filepath.Join(opts.Dir, "release.yaml")
	if _, err := os.Stat(manifestPath); err == nil {
		return nil
	}
	raw, err := yaml.Marshal(map[string]any{
		"artifact":    relativeFrom(filepath.Dir(manifestPath), opts.ArtifactPath),
		"track":       opts.Track,
		"status":      "completed",
		"notesFile":   relativeFrom(filepath.Dir(manifestPath), opts.NotesFile),
		"notesLocale": opts.DefaultLocale,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, raw, 0o600)
}

func ensureWorkflowFile(opts initOptions) error {
	workflowPath := filepath.Join(repoRootForWorkspace(opts.Dir), ".gpc", "workflow.yml")
	raw, err := yaml.Marshal(map[string]any{
		"version": 1,
		"vars": map[string]string{
			"packageName": opts.PackageName,
		},
		"steps": []map[string]any{
			{
				"id":  "release-full",
				"run": "release full --package-name ${packageName} --manifest " + relativeFrom(filepath.Dir(workflowPath), filepath.Join(opts.Dir, "release.yaml")) + " --confirm",
			},
		},
	})
	if err != nil {
		return err
	}
	return os.WriteFile(workflowPath, raw, 0o600)
}

func ensureProjectConfig(opts initOptions) error {
	cfgPath := filepath.Join(repoRootForWorkspace(opts.Dir), ".gpc.yaml")
	raw, err := yaml.Marshal(map[string]any{
		"package-name":        opts.PackageName,
		"default-track":       opts.Track,
		"default-locale":      opts.DefaultLocale,
		"listing-dir":         relativeFrom(repoRootForWorkspace(opts.Dir), filepath.Join(opts.Dir, "listing")),
		"screenshots-dir":     relativeFrom(repoRootForWorkspace(opts.Dir), filepath.Join(opts.Dir, "screenshots")),
		"products-dir":        relativeFrom(repoRootForWorkspace(opts.Dir), filepath.Join(opts.Dir, "products")),
		"subscriptions-dir":   relativeFrom(repoRootForWorkspace(opts.Dir), filepath.Join(opts.Dir, "subscriptions")),
		"changelog-dir":       relativeFrom(repoRootForWorkspace(opts.Dir), filepath.Join(opts.Dir, "changelog")),
		"android-project-dir": relativeFrom(repoRootForWorkspace(opts.Dir), opts.AndroidProjectDir),
		"build-task":          opts.BuildTask,
		"artifact-path":       relativeFrom(repoRootForWorkspace(opts.Dir), opts.ArtifactPath),
		"notes-file":          relativeFrom(repoRootForWorkspace(opts.Dir), opts.NotesFile),
		"appinit-manifest":    relativeFrom(repoRootForWorkspace(opts.Dir), filepath.Join(opts.Dir, "appinit.yaml")),
		"release-manifest":    relativeFrom(repoRootForWorkspace(opts.Dir), filepath.Join(opts.Dir, "release.yaml")),
	})
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, raw, 0o600)
}

func ensureManualBridge(opts initOptions) (string, error) {
	path := filepath.Join(opts.Dir, "MANUAL_FIRST_UPLOAD.md")
	content := fmt.Sprintf(`# First Public Play Upload

Package: %s

## What gpc can do now

1. Change into the Android project: %s
2. Run the release build task: ./gradlew %s
3. Confirm the artifact exists at: %s
4. Generate and keep this local release workspace.
5. After the first web upload, rerun:
   - gpc release init --package-name %s --dir %s
   - %s

## What must be done in Play Console

1. Open the public app entry in Google Play Console.
2. Go to Internal testing.
3. Create the first release.
4. Upload the first AAB manually.
5. Save the release as draft.
6. Wait for Play to finish processing the first upload.

## What gpc cannot do for public apps

- Create or initialize a public app entry end-to-end.
- Bypass the first Play Console upload.
- Force Google Play to leave the draft bootstrap state.

## Rerun gpc

- Refresh the workspace: gpc release init --package-name %s --dir %s
- Continue the non-production flow: %s
`, opts.PackageName, opts.AndroidProjectDir, opts.BuildTask, opts.ArtifactPath, opts.PackageName, opts.Dir, shared.RecommendedReleaseCommand(opts.PackageName, string(shared.PackageReadinessReady), opts.Dir, filepath.Join(opts.Dir, "release.yaml")), opts.PackageName, opts.Dir, shared.RecommendedReleaseCommand(opts.PackageName, string(shared.PackageReadinessReady), opts.Dir, filepath.Join(opts.Dir, "release.yaml")))
	return path, os.WriteFile(path, []byte(content), 0o600)
}

func ensureDraftBootstrapNote(opts initOptions) (string, error) {
	path := filepath.Join(opts.Dir, "DRAFT_BOOTSTRAP.md")
	content := fmt.Sprintf(`# Draft Bootstrap Handoff

Package: %s

## First bootstrap release

1. Build the release artifact so it exists at %s.
2. Commit the bootstrap release with:
   gpc release full --manifest %s --confirm
3. The bootstrap release will be forced to the Internal track with status "draft".

## What to expect next

- Google Play may keep the package in draft bootstrap state for several minutes after the first draft upload.
- While that state persists, metadata-only edits like listing and screenshot sync can still fail validation.
- gpc cannot force Play to leave this state; it can only poll and tell you when to rerun.

## Rerun after Play finishes processing

- Rerun:
  gpc release full --manifest %s --confirm
`, opts.PackageName, opts.ArtifactPath, filepath.Join(opts.Dir, "release.yaml"), filepath.Join(opts.Dir, "release.yaml"))
	return path, os.WriteFile(path, []byte(content), 0o600)
}

func mirrorListingScreenshots(workspaceDir string) error {
	listingDir := filepath.Join(workspaceDir, "listing")
	screenshotsDir := filepath.Join(workspaceDir, "screenshots")
	return filepath.WalkDir(listingDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(listingDir, path)
		if err != nil {
			return err
		}
		if !strings.Contains(rel, string(filepath.Separator)+"images"+string(filepath.Separator)) {
			return nil
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) < 4 || parts[1] != "images" {
			return nil
		}
		imageType := parts[2]
		switch imageType {
		case "phoneScreenshots", "sevenInchScreenshots", "tenInchScreenshots", "tvScreenshots", "wearScreenshots":
		default:
			return nil
		}
		dirName := screenshotDirName(imageType)
		target := filepath.Join(screenshotsDir, parts[0], dirName, parts[len(parts)-1])
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o600)
	})
}

func screenshotDirName(imageType string) string {
	switch imageType {
	case "phoneScreenshots":
		return "phone"
	case "sevenInchScreenshots":
		return "seven-inch"
	case "tenInchScreenshots":
		return "ten-inch"
	case "tvScreenshots":
		return "tv"
	case "wearScreenshots":
		return "wear"
	default:
		return imageType
	}
}

func summarizeWorkspaceWarnings(opts initOptions) ([]string, error) {
	warnings := []string{}
	if _, err := os.Stat(filepath.Join(opts.Dir, "screenshots")); err == nil {
		entries, readErr := os.ReadDir(filepath.Join(opts.Dir, "screenshots"))
		if readErr == nil && len(entries) == 0 {
			warnings = append(warnings, "no screenshots were exported; add files under "+filepath.Join(opts.Dir, "screenshots")+" before screenshot sync")
		}
	}
	return warnings, nil
}

func auditInitWorkspace(opts initOptions) (string, []initIssue, []string) {
	issues := make([]initIssue, 0, 8)
	warnings := make([]string, 0, 4)

	listingIssues := auditListingWorkspace(filepath.Join(opts.Dir, "listing"))
	issues = append(issues, listingIssues...)

	if _, err := screenshotscmd.ScanScreenshotsDir(filepath.Join(opts.Dir, "screenshots")); err != nil {
		issues = append(issues, initIssue{
			Code:     "screenshots_missing",
			Path:     filepath.Join(opts.Dir, "screenshots"),
			Detail:   err.Error(),
			NextStep: "Add at least one locale/device screenshot set before syncing screenshots.",
		})
	}

	if _, err := os.Stat(opts.NotesFile); err != nil {
		issues = append(issues, initIssue{
			Code:     "notes_missing",
			Path:     opts.NotesFile,
			Detail:   "release notes file is missing",
			NextStep: "Create the notes file or rerun `gpc release init` to regenerate defaults.",
		})
	}

	if strings.TrimSpace(opts.BuildTask) == "" {
		issues = append(issues, initIssue{
			Code:     "build_task_missing",
			Detail:   "build task did not resolve",
			NextStep: "Set `build-task` in .gpc.yaml or pass `--build-task`.",
			Blocking: true,
		})
	}
	if _, err := os.Stat(opts.ArtifactPath); err != nil {
		issues = append(issues, initIssue{
			Code:     "artifact_missing",
			Path:     opts.ArtifactPath,
			Detail:   "release artifact does not exist yet",
			NextStep: "Build the Android app with `./gradlew " + opts.BuildTask + "` before running `gpc release full`.",
			Blocking: true,
		})
	}

	for _, dirInfo := range []struct {
		code string
		path string
		name string
	}{
		{code: "products_dir_missing", path: filepath.Join(opts.Dir, "products"), name: "products"},
		{code: "subscriptions_dir_missing", path: filepath.Join(opts.Dir, "subscriptions"), name: "subscriptions"},
	} {
		if _, err := os.Stat(dirInfo.path); err != nil {
			issues = append(issues, initIssue{
				Code:     dirInfo.code,
				Path:     dirInfo.path,
				Detail:   dirInfo.name + " export directory is missing",
				NextStep: "Rerun `gpc release init --package-name " + opts.PackageName + " --dir " + opts.Dir + "` to regenerate the workspace.",
			})
		}
	}

	readiness := "ready"
	hasContentIssues := false
	hasArtifactIssues := false
	for _, issue := range issues {
		switch issue.Code {
		case "artifact_missing", "build_task_missing":
			hasArtifactIssues = true
		default:
			hasContentIssues = true
		}
	}
	switch {
	case hasContentIssues:
		readiness = "needs_content"
	case hasArtifactIssues:
		readiness = "needs_artifact"
	}
	if len(issues) == 0 {
		warnings = append(warnings, "workspace export is release-ready")
	}
	return readiness, issues, warnings
}

func auditListingWorkspace(root string) []initIssue {
	issues := make([]initIssue, 0, 4)
	entries, err := os.ReadDir(root)
	if err != nil {
		return []initIssue{{
			Code:     "listing_missing",
			Path:     root,
			Detail:   "listing directory is missing",
			NextStep: "Rerun `gpc release init` or add listing metadata under this directory.",
		}}
	}

	localeFound := false
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		localeFound = true
		for _, name := range []string{"title.txt", "short-description.txt", "full-description.txt"} {
			path := filepath.Join(root, entry.Name(), name)
			raw, err := os.ReadFile(path)
			switch {
			case os.IsNotExist(err):
				issues = append(issues, initIssue{
					Code:     "listing_file_missing",
					Path:     path,
					Detail:   "required listing file is missing",
					NextStep: "Populate this file before release validation.",
				})
			case err != nil:
				issues = append(issues, initIssue{
					Code:     "listing_file_unreadable",
					Path:     path,
					Detail:   err.Error(),
					NextStep: "Fix the file and rerun `gpc release init`.",
				})
			case strings.TrimSpace(string(raw)) == "":
				issues = append(issues, initIssue{
					Code:     "listing_file_empty",
					Path:     path,
					Detail:   "required listing file is empty",
					NextStep: "Fill in this metadata before release validation.",
				})
			}
		}
	}

	if !localeFound {
		issues = append(issues, initIssue{
			Code:     "listing_locale_missing",
			Path:     root,
			Detail:   "no locale directories found in exported listing workspace",
			NextStep: "Add at least one locale with title, short description, and full description.",
		})
	}

	// Keep the stricter parser signal as a final sanity check.
	if _, err := listing.ScanListingsDir(root); err != nil {
		issues = append(issues, initIssue{
			Code:     "listing_validation_failed",
			Path:     root,
			Detail:   err.Error(),
			NextStep: "Fix listing validation errors before release validation.",
		})
	}

	return dedupeInitIssues(issues)
}

func dedupeInitIssues(issues []initIssue) []initIssue {
	seen := map[string]struct{}{}
	out := make([]initIssue, 0, len(issues))
	for _, issue := range issues {
		key := issue.Code + "|" + issue.Path + "|" + issue.Detail
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, issue)
	}
	return out
}

func repoRootForWorkspace(workspaceDir string) string {
	return filepath.Dir(workspaceDir)
}

func relativeFrom(baseDir, target string) string {
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return target
	}
	if strings.TrimSpace(rel) == "" {
		return "."
	}
	return rel
}

func isInteractiveInput(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func promptInput(stdin io.Reader, stderr io.Writer, label string) (string, error) {
	if stderr == nil {
		stderr = os.Stderr
	}
	if _, err := fmt.Fprintf(stderr, "%s: ", label); err != nil {
		return "", err
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
