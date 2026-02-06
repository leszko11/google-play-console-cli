package auth

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewInitCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var (
		serviceAccountPath string
		profile            string
		packageName        string
	)

	fs.StringVar(&serviceAccountPath, "service-account", "", "Path to service account JSON")
	fs.StringVar(&profile, "profile", "default", "Auth profile name")
	fs.StringVar(&packageName, "package-name", "", "Verify package access for this package")

	return &ffcli.Command{
		Name:      "init",
		ShortHelp: "Initialize authentication profile",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if strings.TrimSpace(serviceAccountPath) == "" {
				serviceAccountPath = strings.TrimSpace(deps.LookupEnv(envServiceAccountPath))
			}
			if strings.TrimSpace(serviceAccountPath) == "" {
				return fmt.Errorf("--service-account is required or set %s", envServiceAccountPath)
			}

			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			if cfg.Profiles == nil {
				cfg.Profiles = make(map[string]config.Profile)
			}

			if strings.TrimSpace(packageName) != "" {
				client, err := deps.NewClient(ctx, gpc.CredentialInput{ServiceAccountPath: serviceAccountPath})
				if err != nil {
					return err
				}
				if err := client.VerifyPackageAccess(ctx, packageName); err != nil {
					return fmt.Errorf("failed to verify package access: %w", err)
				}
			}

			cfg.Profiles[profile] = config.Profile{
				ServiceAccountPath: serviceAccountPath,
				LastValidatedAt:    deps.Now().UTC().Format(time.RFC3339),
			}
			cfg.ActiveProfile = profile

			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}

			return writeJSON(deps.Stdout, map[string]any{
				"activeProfile":      cfg.ActiveProfile,
				"serviceAccountPath": serviceAccountPath,
			})
		},
	}
}
