package auth

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type PackageVerifier interface {
	VerifyPackageAccess(ctx context.Context, packageName string) error
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	SaveConfig func(config.Config) error
	NewClient  func(context.Context, gpc.CredentialInput) (PackageVerifier, error)
	LookupEnv  func(string) string
	PromptID   func(stdin io.Reader, stderr io.Writer) (string, error)
	Now        func() time.Time
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	return &ffcli.Command{
		Name:      "auth",
		ShortHelp: "Manage authentication profiles",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			NewInitCommand(deps),
			NewStatusCommand(deps),
			NewProfilesCommand(deps),
			NewSwitchCommand(deps),
			NewLogoutCommand(deps),
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
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (PackageVerifier, error) {
			return gpc.NewClient(ctx, creds)
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.PromptID == nil {
		deps.PromptID = promptDeveloperID
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
