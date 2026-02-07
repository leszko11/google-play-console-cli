package apps

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const envServiceAccountPath = "GPC_SERVICE_ACCOUNT_PATH"

type Client interface {
	VerifyPackageAccess(ctx context.Context, packageName string) error
	GetApp(ctx context.Context, packageName string) (gpc.AppInfo, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	SaveConfig func(config.Config) error
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
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
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return deps
}

func resolveServiceAccountPath(cfg config.Config, lookupEnv func(string) string) (string, error) {
	if cfg.ActiveProfile != "" && cfg.Profiles != nil {
		if profile, ok := cfg.Profiles[cfg.ActiveProfile]; ok && profile.ServiceAccountPath != "" {
			return profile.ServiceAccountPath, nil
		}
	}

	if envPath := lookupEnv(envServiceAccountPath); envPath != "" {
		return envPath, nil
	}

	return "", fmt.Errorf("no service account configured")
}

func writeJSON(out io.Writer, v any) error {
	b, err := shared.RenderJSON(v, false)
	if err != nil {
		return err
	}
	_, err = out.Write(b)
	return err
}
