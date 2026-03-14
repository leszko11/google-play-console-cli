package release

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
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
	CommitEdit(ctx context.Context, packageName, editID string) (gpc.EditInfo, error)
	UpdateTrack(ctx context.Context, packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error)
	UploadBundle(ctx context.Context, packageName, editID, bundlePath string) (gpc.BundleInfo, error)
	UploadAPK(ctx context.Context, packageName, editID, apkPath string) (gpc.APKInfo, error)
	UploadDeobfuscationFile(ctx context.Context, packageName, editID string, versionCode int64, fileType, filePath string) (gpc.DeobfuscationFileInfo, error)
	GetTrack(ctx context.Context, packageName, editID, trackName string) (gpc.TrackInfo, error)
}

type ReportingClient interface {
	QueryVitalsMetricSet(ctx context.Context, packageName string, metricSet gpc.ReportingVitalsMetricSet, request *gpc.ReportingVitalsQueryRequest) (gpc.ReportingVitalsQueryResult, error)
}

type RunCommandFunc func(ctx context.Context, dir, name string, args ...string) (string, error)

type Deps struct {
	LoadConfig         func() (config.Config, error)
	NewClient          func(context.Context, gpc.CredentialInput) (Client, error)
	NewReportingClient func(context.Context, gpc.CredentialInput) (ReportingClient, error)
	LookupEnv          func(string) string
	RunCommand         RunCommandFunc
	Now                func() time.Time
	Sleep              func(context.Context, time.Duration) error
	Stdout             io.Writer
	Stderr             io.Writer
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	return &ffcli.Command{
		Name:      "release",
		ShortHelp: "Release workflows for staged Google Play deploys",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newVerifyCommand(deps),
			newAlphaCommand(deps),
			newFullCommand(deps),
			newPromoteCommand(deps),
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
