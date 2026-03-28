package release

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/appinit"
	productscmd "github.com/leszko11/google-play-console-cli/internal/cli/products"
	"github.com/leszko11/google-play-console-cli/internal/cli/screenshots"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	subscriptionscmd "github.com/leszko11/google-play-console-cli/internal/cli/subscriptions"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const (
	defaultStagingPackage = "com.example.app.staging"
	defaultTrack          = "alpha"
	defaultReleaseStatus  = "completed"
	defaultBuildTask      = ":app:bundleStagingRelease"
)

type Client interface {
	VerifyPackageAccess(ctx context.Context, packageName string) error
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ValidateEdit(ctx context.Context, packageName, editID string) error
	CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) (gpc.EditInfo, error)
	UpdateTrack(ctx context.Context, packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error)
	UploadBundle(ctx context.Context, packageName, editID, bundlePath string) (gpc.BundleInfo, error)
	UploadAPK(ctx context.Context, packageName, editID, apkPath string) (gpc.APKInfo, error)
	UploadDeobfuscationFile(ctx context.Context, packageName, editID string, versionCode int64, fileType, filePath string) (gpc.DeobfuscationFileInfo, error)
	GetTrack(ctx context.Context, packageName, editID, trackName string) (gpc.TrackInfo, error)
	ListOneTimeProducts(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (gpc.OneTimeProductsListInfo, error)
	ListSubscriptions(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (gpc.SubscriptionsListInfo, error)
}

type ReportingClient interface {
	QueryVitalsMetricSet(ctx context.Context, packageName string, metricSet gpc.ReportingVitalsMetricSet, request *gpc.ReportingVitalsQueryRequest) (gpc.ReportingVitalsQueryResult, error)
	SearchApps(ctx context.Context, pageSize int64, pageToken string, paginate bool) (gpc.ReportingAppsListInfo, error)
}

type RunCommandFunc func(ctx context.Context, dir, name string, args ...string) (string, error)
type RunSubcommandFunc func(ctx context.Context, args []string) error

type Deps struct {
	LoadConfig           func() (config.Config, error)
	NewClient            func(context.Context, gpc.CredentialInput) (Client, error)
	NewReportingClient   func(context.Context, gpc.CredentialInput) (ReportingClient, error)
	RunBootstrap         RunSubcommandFunc
	RunAppInit           RunSubcommandFunc
	RunScreenshotsSync   RunSubcommandFunc
	RunProductsSync      RunSubcommandFunc
	RunSubscriptionsSync RunSubcommandFunc
	LookupEnv            func(string) string
	RunCommand           RunCommandFunc
	Now                  func() time.Time
	Sleep                func(context.Context, time.Duration) error
	Stdin                io.Reader
	Stdout               io.Writer
	Stderr               io.Writer
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	return &ffcli.Command{
		Name:      "release",
		ShortHelp: "Release workflows for staged Google Play deploys",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newInitCommand(deps),
			newVerifyCommand(deps),
			newAlphaCommand(deps),
			newFullCommand(deps),
			newPromoteCommand(deps),
			newRollbackCommand(deps),
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
	if deps.NewReportingClient == nil {
		deps.NewReportingClient = func(ctx context.Context, creds gpc.CredentialInput) (ReportingClient, error) {
			return gpc.NewReportingClient(ctx, creds)
		}
	}
	if deps.RunBootstrap == nil {
		deps.RunBootstrap = func(ctx context.Context, args []string) error {
			cmd := appinit.NewBootstrapCommand(appinit.Deps{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				Stdout:     io.Discard,
				Stderr:     deps.Stderr,
			})
			return cmd.ParseAndRun(ctx, args)
		}
	}
	if deps.RunAppInit == nil {
		deps.RunAppInit = func(ctx context.Context, args []string) error {
			cmd := appinit.NewCommand(appinit.Deps{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				Stdout:     io.Discard,
				Stderr:     deps.Stderr,
			})
			return cmd.ParseAndRun(ctx, args)
		}
	}
	if deps.RunScreenshotsSync == nil {
		deps.RunScreenshotsSync = func(ctx context.Context, args []string) error {
			cmd := screenshots.NewCommand(screenshots.Deps{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				Stdout:     io.Discard,
				Stderr:     deps.Stderr,
			})
			return cmd.ParseAndRun(ctx, append([]string{"sync"}, args...))
		}
	}
	if deps.RunProductsSync == nil {
		deps.RunProductsSync = func(ctx context.Context, args []string) error {
			cmd := productscmd.NewCommand(productscmd.Deps{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				Stdout:     io.Discard,
				Stderr:     deps.Stderr,
			})
			return cmd.ParseAndRun(ctx, append([]string{"sync"}, args...))
		}
	}
	if deps.RunSubscriptionsSync == nil {
		deps.RunSubscriptionsSync = func(ctx context.Context, args []string) error {
			cmd := subscriptionscmd.NewCommand(subscriptionscmd.Deps{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				Stdout:     io.Discard,
				Stderr:     deps.Stderr,
			})
			return cmd.ParseAndRun(ctx, append([]string{"sync"}, args...))
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
	}
	if deps.RunCommand == nil {
		deps.RunCommand = runCommand
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Sleep == nil {
		deps.Sleep = sleepContext
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

func buildClient(ctx context.Context, deps Deps) (Client, context.Context, context.CancelFunc, error) {
	return shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
	})
}

func runCommand(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errText := stderr.String()
		if errText == "" {
			errText = stdout.String()
		}
		return "", fmt.Errorf("%w: %s", err, errText)
	}
	if stdout.Len() > 0 {
		return stdout.String(), nil
	}
	return stderr.String(), nil
}

func buildReportingClient(ctx context.Context, deps Deps) (ReportingClient, context.Context, context.CancelFunc, error) {
	return shared.BuildClient[ReportingClient](ctx, shared.BuildClientDeps[ReportingClient]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewReportingClient,
	})
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
