package initcmd

import (
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

const (
	defaultWorkspaceDir = "./play"
	defaultTrack        = "internal"
	defaultLocale       = "en-US"
	defaultAndroidDir   = "./android"
	defaultBuildTask    = ":app:bundleRelease"
)

type Deps struct {
	Stdout io.Writer
	Stderr io.Writer
}

type options struct {
	PackageName        string
	Dir                string
	Track              string
	DefaultLocale      string
	AndroidProjectDir  string
	BuildTask          string
	ArtifactPath       string
	NotesFile          string
	PrepareWorkflow    bool
	WriteProjectConfig bool
	Output             string
}

type stepResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

type result struct {
	PackageName       string       `json:"packageName"`
	Status            string       `json:"status"`
	WorkspaceDir      string       `json:"workspaceDir"`
	DefaultTrack      string       `json:"defaultTrack"`
	DefaultLocale     string       `json:"defaultLocale"`
	AndroidProjectDir string       `json:"androidProjectDir"`
	BuildTask         string       `json:"buildTask"`
	ArtifactPath      string       `json:"artifactPath"`
	NotesFile         string       `json:"notesFile"`
	AppInitManifest   string       `json:"appInitManifest"`
	ReleaseManifest   string       `json:"releaseManifest"`
	WorkflowPath      string       `json:"workflowPath,omitempty"`
	ProjectConfigPath string       `json:"projectConfigPath,omitempty"`
	CreatedPaths      []string     `json:"createdPaths,omitempty"`
	ExistingPaths     []string     `json:"existingPaths,omitempty"`
	Steps             []stepResult `json:"steps"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts options
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name to persist into the local scaffold")
	fs.StringVar(&opts.Dir, "dir", defaultWorkspaceDir, "Workspace directory to create")
	fs.StringVar(&opts.Track, "track", defaultTrack, "Default non-production release track")
	fs.StringVar(&opts.DefaultLocale, "default-locale", defaultLocale, "Default store locale")
	fs.StringVar(&opts.AndroidProjectDir, "android-project-dir", defaultAndroidDir, "Android project directory")
	fs.StringVar(&opts.BuildTask, "build-task", defaultBuildTask, "Gradle build task used for release verification")
	fs.StringVar(&opts.ArtifactPath, "artifact-path", "", "Release artifact path")
	fs.StringVar(&opts.NotesFile, "notes-file", "", "Release notes file path")
	fs.BoolVar(&opts.PrepareWorkflow, "write-workflow", true, "Write .gpc/workflow.yml scaffold")
	fs.BoolVar(&opts.WriteProjectConfig, "write-project-config", true, "Write .gpc.yaml with project-local defaults")
	fs.StringVar(&opts.Output, "output", "", "Output format: json, table")

	return &ffcli.Command{
		Name:      "init",
		ShortHelp: "Create a local gpc workspace scaffold without calling Play APIs",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			_ = ctx
			res, err := run(opts)
			if err != nil {
				return err
			}
			switch shared.ResolveOutput(opts.Output) {
			case "json":
				return shared.WriteJSON(deps.Stdout, res)
			case "table":
				return writeTable(deps.Stdout, res)
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(opts.Output))
			}
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return deps
}

func run(opts options) (result, error) {
	opts, err := validateOptions(opts)
	if err != nil {
		return result{}, err
	}

	res := result{
		PackageName:       opts.PackageName,
		Status:            "scaffolded",
		WorkspaceDir:      opts.Dir,
		DefaultTrack:      opts.Track,
		DefaultLocale:     opts.DefaultLocale,
		AndroidProjectDir: opts.AndroidProjectDir,
		BuildTask:         opts.BuildTask,
		ArtifactPath:      opts.ArtifactPath,
		NotesFile:         opts.NotesFile,
		AppInitManifest:   filepath.Join(opts.Dir, "appinit.yaml"),
		ReleaseManifest:   filepath.Join(opts.Dir, "release.yaml"),
		Steps:             make([]stepResult, 0, 10),
	}
	if opts.PrepareWorkflow {
		res.WorkflowPath = filepath.Join(repoRootForWorkspace(opts.Dir), ".gpc", "workflow.yml")
	}
	if opts.WriteProjectConfig {
		res.ProjectConfigPath = filepath.Join(repoRootForWorkspace(opts.Dir), ".gpc.yaml")
	}

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
			return result{}, err
		}
	}
	res.Steps = append(res.Steps, stepResult{Name: "workspace_dirs", Status: "ok", Detail: opts.Dir})

	if err := ensureFile(&res, res.AppInitManifest, buildAppInitManifest(opts)); err != nil {
		return result{}, err
	}
	if err := ensureFile(&res, res.ReleaseManifest, buildReleaseManifest(opts)); err != nil {
		return result{}, err
	}
	if err := ensureFile(&res, opts.NotesFile, []byte("Internal validation build for the gpc Android release flow.\n")); err != nil {
		return result{}, err
	}
	if opts.PrepareWorkflow {
		if err := ensureFile(&res, res.WorkflowPath, buildWorkflow(opts, res.WorkflowPath)); err != nil {
			return result{}, err
		}
	} else {
		res.Steps = append(res.Steps, stepResult{Name: "workflow", Status: "skipped", Detail: "disabled by --write-workflow=false"})
	}
	if opts.WriteProjectConfig {
		if err := ensureFile(&res, res.ProjectConfigPath, buildProjectConfig(opts)); err != nil {
			return result{}, err
		}
	} else {
		res.Steps = append(res.Steps, stepResult{Name: "project_config", Status: "skipped", Detail: "disabled by --write-project-config=false"})
	}

	return res, nil
}

func validateOptions(opts options) (options, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return options{}, err
	}
	opts.PackageName = pkg

	opts.Dir = strings.TrimSpace(opts.Dir)
	if opts.Dir == "" {
		opts.Dir = defaultWorkspaceDir
	}
	opts.Track = strings.TrimSpace(opts.Track)
	if opts.Track == "" {
		opts.Track = defaultTrack
	}
	opts.DefaultLocale = strings.TrimSpace(opts.DefaultLocale)
	if opts.DefaultLocale == "" {
		opts.DefaultLocale = defaultLocale
	}
	opts.AndroidProjectDir = strings.TrimSpace(opts.AndroidProjectDir)
	if opts.AndroidProjectDir == "" {
		if value, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.AndroidProjectDir }); err == nil && strings.TrimSpace(value) != "" {
			opts.AndroidProjectDir = value
		} else {
			opts.AndroidProjectDir = defaultAndroidDir
		}
	}
	opts.BuildTask = strings.TrimSpace(opts.BuildTask)
	if opts.BuildTask == "" {
		if value, err := shared.ResolveProjectValue("", func(cfg config.ProjectConfig) string { return cfg.BuildTask }); err == nil && strings.TrimSpace(value) != "" {
			opts.BuildTask = value
		} else {
			opts.BuildTask = defaultBuildTask
		}
	}
	opts.ArtifactPath = strings.TrimSpace(opts.ArtifactPath)
	if opts.ArtifactPath == "" {
		if value, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.ArtifactPath }); err == nil && strings.TrimSpace(value) != "" {
			opts.ArtifactPath = value
		} else {
			opts.ArtifactPath = filepath.Join(opts.AndroidProjectDir, "app", "build", "outputs", "bundle", "release", "app-release.aab")
		}
	}
	opts.NotesFile = strings.TrimSpace(opts.NotesFile)
	if opts.NotesFile == "" {
		if value, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.NotesFile }); err == nil && strings.TrimSpace(value) != "" {
			opts.NotesFile = value
		} else {
			opts.NotesFile = filepath.Join(opts.Dir, "changelog", opts.Track, opts.DefaultLocale+".txt")
		}
	}
	if format := shared.ResolveOutput(opts.Output); format != "json" && format != "table" {
		return options{}, shared.UsageErrorf("unsupported output format %q", format)
	}
	return opts, nil
}

func ensureFile(res *result, path string, content []byte) error {
	name := filepath.Base(path)
	if _, err := os.Stat(path); err == nil {
		res.ExistingPaths = append(res.ExistingPaths, path)
		res.Steps = append(res.Steps, stepResult{Name: name, Status: "skipped_existing", Detail: path})
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return err
	}
	res.CreatedPaths = append(res.CreatedPaths, path)
	res.Steps = append(res.Steps, stepResult{Name: name, Status: "created", Detail: path})
	return nil
}

func buildAppInitManifest(opts options) []byte {
	raw, _ := yaml.Marshal(map[string]any{
		"appDetails": map[string]any{
			"defaultLanguage": opts.DefaultLocale,
		},
		"listing": map[string]any{
			"dir": "./listing",
		},
		"products": map[string]any{
			"dir": "./products",
		},
		"subscriptions": map[string]any{
			"dir": "./subscriptions",
		},
	})
	return raw
}

func buildReleaseManifest(opts options) []byte {
	raw, _ := yaml.Marshal(map[string]any{
		"artifact":    relativeFrom(filepath.Dir(filepath.Join(opts.Dir, "release.yaml")), opts.ArtifactPath),
		"track":       opts.Track,
		"status":      "completed",
		"notesFile":   relativeFrom(filepath.Dir(filepath.Join(opts.Dir, "release.yaml")), opts.NotesFile),
		"notesLocale": opts.DefaultLocale,
	})
	return raw
}

func buildWorkflow(opts options, workflowPath string) []byte {
	raw, _ := yaml.Marshal(map[string]any{
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
	return raw
}

func buildProjectConfig(opts options) []byte {
	root := repoRootForWorkspace(opts.Dir)
	raw, _ := yaml.Marshal(map[string]any{
		"package-name":        opts.PackageName,
		"default-track":       opts.Track,
		"default-locale":      opts.DefaultLocale,
		"listing-dir":         relativeFrom(root, filepath.Join(opts.Dir, "listing")),
		"screenshots-dir":     relativeFrom(root, filepath.Join(opts.Dir, "screenshots")),
		"products-dir":        relativeFrom(root, filepath.Join(opts.Dir, "products")),
		"subscriptions-dir":   relativeFrom(root, filepath.Join(opts.Dir, "subscriptions")),
		"changelog-dir":       relativeFrom(root, filepath.Join(opts.Dir, "changelog")),
		"android-project-dir": relativeFrom(root, opts.AndroidProjectDir),
		"build-task":          opts.BuildTask,
		"artifact-path":       relativeFrom(root, opts.ArtifactPath),
		"notes-file":          relativeFrom(root, opts.NotesFile),
		"appinit-manifest":    relativeFrom(root, filepath.Join(opts.Dir, "appinit.yaml")),
		"release-manifest":    relativeFrom(root, filepath.Join(opts.Dir, "release.yaml")),
	})
	return raw
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

func writeTable(out io.Writer, res result) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", res.Status); err != nil {
		return err
	}
	for _, pair := range [][2]string{
		{"PACKAGE", res.PackageName},
		{"WORKSPACE", res.WorkspaceDir},
		{"TRACK", res.DefaultTrack},
		{"LOCALE", res.DefaultLocale},
		{"ARTIFACT", res.ArtifactPath},
		{"RELEASE_MANIFEST", res.ReleaseManifest},
		{"APPINIT_MANIFEST", res.AppInitManifest},
		{"NOTES_FILE", res.NotesFile},
	} {
		if _, err := fmt.Fprintf(out, "%s\t%s\n", pair[0], pair[1]); err != nil {
			return err
		}
	}
	if res.ProjectConfigPath != "" {
		if _, err := fmt.Fprintf(out, "PROJECT_CONFIG\t%s\n", res.ProjectConfigPath); err != nil {
			return err
		}
	}
	if res.WorkflowPath != "" {
		if _, err := fmt.Fprintf(out, "WORKFLOW\t%s\n", res.WorkflowPath); err != nil {
			return err
		}
	}
	if len(res.CreatedPaths) > 0 {
		if _, err := fmt.Fprintf(out, "CREATED\t%s\n", strings.Join(res.CreatedPaths, ",")); err != nil {
			return err
		}
	}
	if len(res.ExistingPaths) > 0 {
		if _, err := fmt.Fprintf(out, "EXISTING\t%s\n", strings.Join(res.ExistingPaths, ",")); err != nil {
			return err
		}
	}
	return nil
}
