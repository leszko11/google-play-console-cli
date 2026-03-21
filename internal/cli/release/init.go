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
	WriteProjectConfig bool
}

type initStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

type initResult struct {
	Status            string     `json:"status"`
	PackageName       string     `json:"packageName"`
	PackageReadiness  string     `json:"packageReadiness,omitempty"`
	WorkspaceDir      string     `json:"workspaceDir,omitempty"`
	ProjectConfigPath string     `json:"projectConfigPath,omitempty"`
	ReleaseManifest   string     `json:"releaseManifest,omitempty"`
	WorkflowPath      string     `json:"workflowPath,omitempty"`
	ManualBridgePath  string     `json:"manualBridgePath,omitempty"`
	Steps             []initStep `json:"steps"`
	Warnings          []string   `json:"warnings,omitempty"`
	NextSteps         []string   `json:"nextSteps,omitempty"`
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

	cfg, err := deps.LoadConfig()
	if err != nil {
		return res, err
	}
	authStatus := shared.BuildAuthStatusSnapshot(cfg, deps.LookupEnv)
	if !authStatus.Authenticated {
		res.Steps = append(res.Steps, initStep{Name: "auth", Status: "error", Error: shared.AuthStatusSummary(authStatus)})
		res.NextSteps = append(res.NextSteps, "Run `gpc auth init --service-account <path>` or `gpc setup --auto --project-id <id> --package-name "+opts.PackageName+"`.")
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

	switch readiness.Status {
	case shared.PackageReadinessUninitialized:
		bridgePath, err := ensureManualBridge(opts)
		if err != nil {
			return res, err
		}
		res.Status = "manual_required"
		res.ManualBridgePath = bridgePath
		res.NextSteps = append(res.NextSteps, "Follow "+bridgePath+" for the first Play Console upload.")
		res.NextSteps = append(res.NextSteps, "Then rerun `gpc release init --package-name "+opts.PackageName+" --dir "+opts.Dir+"`.")
	case shared.PackageReadinessDraftBootstrapRequired:
		res.Status = "draft_bootstrap_required"
		res.Warnings = append(res.Warnings, readiness.Warning)
		res.NextSteps = append(res.NextSteps, readiness.NextStep)
		res.NextSteps = append(res.NextSteps, "After the draft bootstrap deploy, use `gpc release full --manifest "+filepath.Join(opts.Dir, "release.yaml")+" --confirm`.")
	default:
		res.Status = "ready"
		if warnings, err := summarizeWorkspaceWarnings(opts); err == nil {
			res.Warnings = append(res.Warnings, warnings...)
		}
		res.NextSteps = append(res.NextSteps, "Run `gpc release full --manifest "+filepath.Join(opts.Dir, "release.yaml")+" --confirm` for the next internal or alpha release.")
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

## Build the first artifact

1. Change into the Android project: %s
2. Run the release build task: ./gradlew %s
3. Confirm the artifact exists at: %s

## Manual Play Console bridge

1. Open Google Play Console for package %s.
2. Create or open the app entry.
3. Go to a non-production track such as Internal testing.
4. Upload the first AAB or APK manually.
5. Save the release so the package is initialized in Play.

## Rerun gpc

- Refresh the workspace: gpc release init --package-name %s --dir %s
- Run the full non-production flow: gpc release full --manifest %s --confirm
`, opts.PackageName, opts.AndroidProjectDir, opts.BuildTask, opts.ArtifactPath, opts.PackageName, opts.PackageName, opts.Dir, filepath.Join(opts.Dir, "release.yaml"))
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
