package apps

import (
	"context"
	"io"
	"os"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	VerifyPackageAccess(ctx context.Context, packageName string) error
	GetApp(ctx context.Context, packageName string) (gpc.AppInfo, error)
	SetDataSafety(ctx context.Context, packageName, safetyLabelsCSV string) error
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	SaveConfig func(config.Config) error
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	return &ffcli.Command{
		Name:      "apps",
		ShortHelp: "App visibility and metadata commands",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			NewListCommand(deps),
			NewGetCommand(deps),
			NewDataSafetyCommand(deps),
			NewAddPackageCommand(deps),
			NewRemovePackageCommand(deps),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.SaveConfig == nil {
		deps.SaveConfig = config.Save
	}
	if deps.NewClient == nil {
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (Client, error) {
			return gpc.NewClient(ctx, creds)
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
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
