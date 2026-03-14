package setup

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/appinit"
	"github.com/leszko11/google-play-console-cli/internal/cli/auth"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type VerifyClient interface {
	VerifyPackageAccess(ctx context.Context, packageName string) error
}

type RunCommandFunc func(ctx context.Context, dir, name string, args ...string) (string, error)
type RunSubcommandFunc func(ctx context.Context, args []string) error

type Deps struct {
	RunCommand   RunCommandFunc
	RunAuthInit  RunSubcommandFunc
	RunBootstrap RunSubcommandFunc
	NewClient    func(context.Context, gpc.CredentialInput) (VerifyClient, error)
	LookupEnv    func(string) string
	Now          func() time.Time
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
}

type options struct {
	Auto                  bool
	ProjectID             string
	PackageName           string
	Profile               string
	DeveloperID           string
	ServiceAccountName    string
	ServiceAccountDisplay string
	ServiceAccountKey     string
	Dir                   string
	SkipBootstrap         bool
	WriteProjectConfig    bool
}

type setupStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

type result struct {
	Status              string      `json:"status"`
	ProjectID           string      `json:"projectId,omitempty"`
	Profile             string      `json:"profile"`
	PackageName         string      `json:"packageName,omitempty"`
	ServiceAccountEmail string      `json:"serviceAccountEmail,omitempty"`
	ServiceAccountPath  string      `json:"serviceAccountPath,omitempty"`
	BootstrapDir        string      `json:"bootstrapDir,omitempty"`
	Steps               []setupStep `json:"steps"`
	Warnings            []string    `json:"warnings,omitempty"`
	NextSteps           []string    `json:"nextSteps,omitempty"`
}

var osExecCommandContext = exec.CommandContext

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts options
	fs.BoolVar(&opts.Auto, "auto", false, "Run scripted setup without prompts")
	fs.StringVar(&opts.ProjectID, "project-id", "", "Google Cloud project ID")
	fs.StringVar(&opts.PackageName, "package-name", "", "Optional package name to verify and bootstrap")
	fs.StringVar(&opts.Profile, "profile", "default", "Auth profile name")
	fs.StringVar(&opts.DeveloperID, "developer-id", "", "Optional developer account ID")
	fs.StringVar(&opts.ServiceAccountName, "service-account-name", "", "Service account name (defaults to gpc-<profile>)")
	fs.StringVar(&opts.ServiceAccountDisplay, "service-account-display-name", "", "Service account display name")
	fs.StringVar(&opts.ServiceAccountKey, "service-account-key", "", "Path to write or reuse the service account key JSON")
	fs.StringVar(&opts.Dir, "dir", "", "Bootstrap directory (defaults to ./play)")
	fs.BoolVar(&opts.SkipBootstrap, "skip-bootstrap", false, "Skip local bootstrap export even when package access is available")
	fs.BoolVar(&opts.WriteProjectConfig, "write-project-config", true, "Write .gpc.yaml when bootstrap runs")

	return &ffcli.Command{
		Name:      "setup",
		ShortHelp: "Provision auth and optional bootstrap workspace for gpc",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			res, err := run(ctx, deps, opts)
			writeErr := writeResult(deps.Stdout, res)
			if err != nil {
				return err
			}
			return writeErr
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.RunCommand == nil {
		deps.RunCommand = runCommand
	}
	if deps.RunAuthInit == nil {
		deps.RunAuthInit = func(ctx context.Context, args []string) error {
			cmd := auth.NewInitCommand(auth.Deps{})
			return cmd.ParseAndRun(ctx, args)
		}
	}
	if deps.RunBootstrap == nil {
		deps.RunBootstrap = func(ctx context.Context, args []string) error {
			cmd := appinit.NewBootstrapCommand(appinit.Deps{})
			return cmd.ParseAndRun(ctx, args)
		}
	}
	if deps.NewClient == nil {
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (VerifyClient, error) {
			return gpc.NewClient(ctx, creds)
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Stdin == nil {
		deps.Stdin = os.Stdin
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return deps
}

func run(ctx context.Context, deps Deps, opts options) (result, error) {
	res := result{
		Status:   "failed",
		Profile:  strings.TrimSpace(opts.Profile),
		Steps:    make([]setupStep, 0, 8),
		Warnings: []string{},
	}

	resolved, err := resolveOptions(deps, opts)
	if err != nil {
		return res, err
	}
	res.ProjectID = resolved.ProjectID
	res.PackageName = resolved.PackageName
	res.ServiceAccountPath = resolved.ServiceAccountKey
	res.BootstrapDir = resolved.Dir
	res.ServiceAccountEmail = serviceAccountEmail(resolved.ServiceAccountName, resolved.ProjectID)

	fail := func(step string, err error) (result, error) {
		res.Steps = append(res.Steps, setupStep{Name: step, Status: "error", Error: err.Error()})
		return res, err
	}

	if _, err := deps.RunCommand(ctx, "", "gcloud", "--version"); err != nil {
		return fail("gcloud_detect", fmt.Errorf("failed to detect gcloud: %w", err))
	}
	res.Steps = append(res.Steps, setupStep{Name: "gcloud_detect", Status: "ok"})

	if _, err := deps.RunCommand(ctx, "", "gcloud", "services", "enable", "androidpublisher.googleapis.com", "--project", resolved.ProjectID); err != nil {
		return fail("enable_androidpublisher_api", fmt.Errorf("failed to enable androidpublisher api: %w", err))
	}
	res.Steps = append(res.Steps, setupStep{Name: "enable_androidpublisher_api", Status: "ok"})

	email := serviceAccountEmail(resolved.ServiceAccountName, resolved.ProjectID)
	if _, err := deps.RunCommand(ctx, "", "gcloud", "iam", "service-accounts", "describe", email, "--project", resolved.ProjectID); err != nil {
		if _, createErr := deps.RunCommand(ctx, "", "gcloud", "iam", "service-accounts", "create", resolved.ServiceAccountName, "--project", resolved.ProjectID, "--display-name", resolved.ServiceAccountDisplay); createErr != nil {
			return fail("create_service_account", fmt.Errorf("failed to create service account: %w", createErr))
		}
		res.Steps = append(res.Steps, setupStep{Name: "create_service_account", Status: "ok", Detail: email})
	} else {
		res.Steps = append(res.Steps, setupStep{Name: "create_service_account", Status: "skipped", Detail: email})
	}

	if fileLooksLikeJSON(resolved.ServiceAccountKey) {
		res.Steps = append(res.Steps, setupStep{Name: "create_service_account_key", Status: "skipped", Detail: resolved.ServiceAccountKey})
	} else {
		if err := os.MkdirAll(filepath.Dir(resolved.ServiceAccountKey), 0o755); err != nil {
			return fail("create_service_account_key", fmt.Errorf("failed to create key directory: %w", err))
		}
		if _, err := deps.RunCommand(ctx, "", "gcloud", "iam", "service-accounts", "keys", "create", resolved.ServiceAccountKey, "--iam-account", email, "--project", resolved.ProjectID); err != nil {
			return fail("create_service_account_key", fmt.Errorf("failed to create service account key: %w", err))
		}
		res.Steps = append(res.Steps, setupStep{Name: "create_service_account_key", Status: "ok", Detail: resolved.ServiceAccountKey})
	}

	authArgs := []string{"--service-account", resolved.ServiceAccountKey, "--profile", resolved.Profile}
	if strings.TrimSpace(resolved.DeveloperID) != "" {
		authArgs = append(authArgs, "--developer-id", strings.TrimSpace(resolved.DeveloperID))
	}
	if err := deps.RunAuthInit(ctx, authArgs); err != nil {
		return fail("auth_init", fmt.Errorf("failed to initialize auth profile: %w", err))
	}
	res.Steps = append(res.Steps, setupStep{Name: "auth_init", Status: "ok", Detail: resolved.Profile})

	if strings.TrimSpace(resolved.PackageName) == "" {
		res.Status = "ok"
		res.Steps = append(res.Steps, setupStep{Name: "package_access", Status: "skipped", Detail: "skipped (--package-name not provided)"})
		if resolved.SkipBootstrap {
			res.Steps = append(res.Steps, setupStep{Name: "bootstrap", Status: "skipped", Detail: "skipped (--skip-bootstrap enabled)"})
		}
		return res, nil
	}

	requestCtx, cancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
	defer cancel()
	client, err := deps.NewClient(requestCtx, gpc.CredentialInput{ServiceAccountPath: resolved.ServiceAccountKey})
	if err != nil {
		return fail("package_access", fmt.Errorf("failed to create Play client: %w", err))
	}
	if err := client.VerifyPackageAccess(requestCtx, resolved.PackageName); err != nil {
		res.Status = "warn"
		res.Steps = append(res.Steps, setupStep{Name: "package_access", Status: "warn", Error: err.Error()})
		res.Warnings = append(res.Warnings, "playConsoleAccess")
		res.NextSteps = append(res.NextSteps, "Grant the service account access in Play Console, then rerun `gpc setup --auto --package-name <package>` or `gpc bootstrap`.")
		res.Steps = append(res.Steps, setupStep{Name: "bootstrap", Status: "skipped", Detail: "skipped until package access succeeds"})
		return res, nil
	}
	res.Steps = append(res.Steps, setupStep{Name: "package_access", Status: "ok", Detail: resolved.PackageName})

	if resolved.SkipBootstrap {
		res.Status = "ok"
		res.Steps = append(res.Steps, setupStep{Name: "bootstrap", Status: "skipped", Detail: "skipped (--skip-bootstrap enabled)"})
		return res, nil
	}

	bootstrapArgs := []string{"--package-name", resolved.PackageName, "--dir", resolved.Dir}
	if resolved.WriteProjectConfig {
		bootstrapArgs = append(bootstrapArgs, "--write-project-config")
	}
	if err := deps.RunBootstrap(ctx, bootstrapArgs); err != nil {
		return fail("bootstrap", fmt.Errorf("failed to bootstrap local workspace: %w", err))
	}
	res.Status = "ok"
	res.Steps = append(res.Steps, setupStep{Name: "bootstrap", Status: "ok", Detail: resolved.Dir})
	return res, nil
}

func resolveOptions(deps Deps, opts options) (options, error) {
	opts.ProjectID = strings.TrimSpace(opts.ProjectID)
	opts.PackageName = strings.TrimSpace(opts.PackageName)
	opts.Profile = strings.TrimSpace(opts.Profile)
	opts.DeveloperID = strings.TrimSpace(opts.DeveloperID)
	opts.ServiceAccountName = strings.TrimSpace(opts.ServiceAccountName)
	opts.ServiceAccountDisplay = strings.TrimSpace(opts.ServiceAccountDisplay)
	opts.ServiceAccountKey = strings.TrimSpace(opts.ServiceAccountKey)
	opts.Dir = strings.TrimSpace(opts.Dir)

	if opts.Profile == "" {
		opts.Profile = "default"
	}

	if !opts.Auto {
		if !isInteractive(deps.Stdin) {
			return options{}, shared.UsageErrorf("--auto is required in non-interactive mode")
		}
		var err error
		opts.ProjectID, err = promptRequired(deps.Stdin, deps.Stderr, "Google Cloud project ID", opts.ProjectID)
		if err != nil {
			return options{}, err
		}
		opts.PackageName, err = promptOptional(deps.Stdin, deps.Stderr, "Package name (optional)", opts.PackageName)
		if err != nil {
			return options{}, err
		}
	}

	if opts.ProjectID == "" {
		return options{}, shared.UsageErrorf("--project-id is required")
	}
	if opts.ServiceAccountName == "" {
		opts.ServiceAccountName = "gpc-" + sanitizeName(opts.Profile)
	}
	if opts.ServiceAccountDisplay == "" {
		opts.ServiceAccountDisplay = "Google Play Console CLI (" + opts.Profile + ")"
	}
	if opts.ServiceAccountKey == "" {
		opts.ServiceAccountKey = filepath.Join(".gpc", opts.Profile+"-service-account.json")
	}
	if opts.Dir == "" {
		opts.Dir = "./play"
	}
	return opts, nil
}

func writeResult(out io.Writer, res result) error {
	switch shared.ResolveOutput("") {
	case "json":
		return shared.WriteJSON(out, res)
	case "table":
		return writeTable(out, res)
	default:
		return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(""))
	}
}

func writeTable(out io.Writer, res result) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", res.Status); err != nil {
		return err
	}
	if res.ProjectID != "" {
		if _, err := fmt.Fprintf(out, "PROJECT\t%s\n", res.ProjectID); err != nil {
			return err
		}
	}
	if res.ServiceAccountEmail != "" {
		if _, err := fmt.Fprintf(out, "SERVICE_ACCOUNT\t%s\n", res.ServiceAccountEmail); err != nil {
			return err
		}
	}
	if res.ServiceAccountPath != "" {
		if _, err := fmt.Fprintf(out, "KEY_PATH\t%s\n", res.ServiceAccountPath); err != nil {
			return err
		}
	}
	for _, step := range res.Steps {
		detail := step.Detail
		if detail == "" {
			detail = step.Error
		}
		if _, err := fmt.Fprintf(out, "STEP\t%s\t%s\t%s\n", step.Name, step.Status, detail); err != nil {
			return err
		}
	}
	for _, warning := range res.Warnings {
		if _, err := fmt.Fprintf(out, "warning\t%s\n", warning); err != nil {
			return err
		}
	}
	for _, next := range res.NextSteps {
		if _, err := fmt.Fprintf(out, "next\t%s\n", next); err != nil {
			return err
		}
	}
	return nil
}

func serviceAccountEmail(name, projectID string) string {
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", strings.TrimSpace(name), strings.TrimSpace(projectID))
}

func sanitizeName(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	replacer := strings.NewReplacer(" ", "-", "_", "-", ".", "-", "/", "-")
	raw = replacer.Replace(raw)
	if raw == "" {
		return "default"
	}
	return raw
}

func fileLooksLikeJSON(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(raw))) > 0 && strings.HasPrefix(strings.TrimSpace(string(raw)), "{")
}

func promptRequired(stdin io.Reader, stderr io.Writer, label, current string) (string, error) {
	if strings.TrimSpace(current) != "" {
		return strings.TrimSpace(current), nil
	}
	value, err := promptOptional(stdin, stderr, label, current)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", shared.UsageErrorf("%s is required", label)
	}
	return value, nil
}

func promptOptional(stdin io.Reader, stderr io.Writer, label, current string) (string, error) {
	if strings.TrimSpace(current) != "" {
		return strings.TrimSpace(current), nil
	}
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

func isInteractive(stdin io.Reader) bool {
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

func runCommand(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := osExecCommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}
