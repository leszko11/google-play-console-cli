package bundles

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

type Client interface {
	ListBundles(ctx context.Context, packageName, editID string) ([]gpc.BundleInfo, error)
	UploadBundle(ctx context.Context, packageName, editID, bundlePath string) (gpc.BundleInfo, error)
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

const defaultWaitInterval = 5 * time.Second

type waitResult struct {
	PackageName       string `json:"packageName"`
	VersionCode       int64  `json:"versionCode"`
	Status            string `json:"status"`
	Attempts          int    `json:"attempts"`
	Elapsed           string `json:"elapsed"`
	Interval          string `json:"interval"`
	GeneratedAPKCount int    `json:"generatedApkCount,omitempty"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "bundles",
		ShortHelp: "Manage Android App Bundles in an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newUploadCommand(deps),
			newWaitCommand(deps),
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

func newListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, output string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&output, "output", "", "Output format: json, minimal")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List bundles in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, false)
			if err != nil {
				return err
			}
			defer cancel()
			bundles, err := client.ListBundles(requestCtx, pkg, eid)
			if err != nil {
				return fmt.Errorf("failed to list bundles: %w", err)
			}
			switch shared.ResolveOutput(output) {
			case "json":
				return shared.WriteJSON(deps.Stdout, map[string]any{
					"packageName": pkg,
					"editId":      eid,
					"bundles":     bundles,
				})
			case "minimal":
				values := make([]string, 0, len(bundles))
				for _, b := range bundles {
					values = append(values, strconv.FormatInt(b.VersionCode, 10))
				}
				return shared.WriteMinimal(deps.Stdout, values)
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(output))
			}
		},
	}
}

func newUploadCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, bundlePath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&bundlePath, "file", "", "Path to .aab file")

	return &ffcli.Command{
		Name:      "upload",
		ShortHelp: "Upload an Android App Bundle to an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, true)
			if err != nil {
				return err
			}
			defer cancel()
			bundlePath = strings.TrimSpace(bundlePath)
			if bundlePath == "" {
				return fmt.Errorf("--file is required")
			}
			spinner := shared.NewSpinner(deps.Stderr, "Uploading App Bundle")
			bundle, err := client.UploadBundle(requestCtx, pkg, eid, bundlePath)
			if err != nil {
				spinner.Fail("App Bundle upload failed")
				return fmt.Errorf("failed to upload bundle: %w", err)
			}
			spinner.Success("App Bundle uploaded")
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"bundle":      bundle,
				"status":      "uploaded",
			})
		},
	}
}

func newWaitCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("wait", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName string
	var versionCode int64
	var timeout time.Duration
	var interval time.Duration
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&versionCode, "version-code", 0, "Version code returned from bundle upload")
	fs.DurationVar(&timeout, "timeout", shared.DefaultTimeout, "Maximum time to wait for generated APK availability")
	fs.DurationVar(&interval, "interval", defaultWaitInterval, "Polling interval between generated APK checks")

	return &ffcli.Command{
		Name:      "wait",
		ShortHelp: "Wait until a bundle version code finishes processing",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, err := buildWaitClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			if err := validateWaitOptions(versionCode, timeout, interval); err != nil {
				return err
			}
			spinner := shared.NewSpinner(deps.Stderr, fmt.Sprintf("Waiting for bundle %d to finish processing", versionCode))
			err = runWait(ctx, deps, client, pkg, versionCode, timeout, interval)
			if err != nil {
				spinner.Fail(fmt.Sprintf("Bundle %d processing wait failed", versionCode))
				return err
			}
			spinner.Success(fmt.Sprintf("Bundle %d is ready", versionCode))
			return nil
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName, editID string, upload bool) (Client, string, string, context.Context, context.CancelFunc, error) {
	pkg, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return nil, "", "", nil, nil, err
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, "", "", nil, nil, fmt.Errorf("--edit-id is required")
	}

	client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
		Upload:     upload,
	})
	if err != nil {
		return nil, "", "", nil, nil, err
	}

	return client, pkg, editID, requestCtx, cancel, nil
}

func buildWaitClient(ctx context.Context, deps Deps, packageName string) (Client, string, error) {
	pkg, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return nil, "", err
	}

	cfg, err := deps.LoadConfig()
	if err != nil {
		return nil, "", err
	}
	resolved, err := shared.ResolveCredentials(cfg, deps.LookupEnv)
	if err != nil {
		return nil, "", err
	}
	client, err := deps.NewClient(ctx, resolved.Input)
	if err != nil {
		return nil, "", err
	}
	return client, pkg, nil
}

func validateWaitOptions(versionCode int64, timeout, interval time.Duration) error {
	if versionCode <= 0 {
		return fmt.Errorf("--version-code must be greater than zero")
	}
	if timeout <= 0 {
		return fmt.Errorf("--timeout must be greater than zero")
	}
	if interval <= 0 {
		return fmt.Errorf("--interval must be greater than zero")
	}
	return nil
}

func runWait(ctx context.Context, deps Deps, client Client, packageName string, versionCode int64, timeout, interval time.Duration) error {
	start := deps.Now()
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := waitResult{
		PackageName: packageName,
		VersionCode: versionCode,
		Status:      "waiting",
		Interval:    interval.String(),
	}

	for {
		result.Attempts++
		resp, err := client.ListGeneratedAPKs(waitCtx, packageName, versionCode)
		if err == nil && resp != nil && len(resp.GeneratedApks) > 0 {
			result.Status = "ready"
			result.Elapsed = deps.Now().Sub(start).String()
			result.GeneratedAPKCount = len(resp.GeneratedApks)
			return shared.WriteJSON(deps.Stdout, result)
		}
		if waitCtx.Err() != nil {
			result.Status = "timeout"
			result.Elapsed = deps.Now().Sub(start).String()
			_ = shared.WriteJSON(deps.Stdout, result)
			return fmt.Errorf("timed out waiting for generated apks for versionCode %d", versionCode)
		}
		if err != nil {
			return fmt.Errorf("failed to list generated apks: %w", err)
		}
		if err := deps.Sleep(waitCtx, interval); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				result.Status = "timeout"
				result.Elapsed = deps.Now().Sub(start).String()
				_ = shared.WriteJSON(deps.Stdout, result)
				return fmt.Errorf("timed out waiting for generated apks for versionCode %d", versionCode)
			}
			return err
		}
	}
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
