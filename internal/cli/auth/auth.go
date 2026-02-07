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

const envServiceAccountPath = "GPC_SERVICE_ACCOUNT_PATH"

type PackageVerifier interface {
	VerifyPackageAccess(ctx context.Context, packageName string) error
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	SaveConfig func(config.Config) error
	NewClient  func(context.Context, gpc.CredentialInput) (PackageVerifier, error)
	LookupEnv  func(string) string
	Now        func() time.Time
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
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}

	return deps
}

func writeJSON(w io.Writer, v any) error {
	out, err := shared.RenderJSON(v, false)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}
