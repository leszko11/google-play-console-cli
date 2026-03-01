package auth

import (
	"context"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewStatusCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	return &ffcli.Command{
		Name:      "status",
		ShortHelp: "Show authentication status",
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			return shared.WriteJSON(deps.Stdout, buildStatus(cfg))
		},
	}
}

func buildStatus(cfg config.Config) map[string]any {
	out := map[string]any{
		"activeProfile": cfg.ActiveProfile,
		"authenticated": false,
	}

	if cfg.ActiveProfile == "" {
		return out
	}

	profile, ok := cfg.Profiles[cfg.ActiveProfile]
	if !ok {
		return out
	}

	out["authenticated"] = profile.ServiceAccountPath != ""
	out["serviceAccountPath"] = profile.ServiceAccountPath
	out["lastValidatedAt"] = profile.LastValidatedAt
	return out
}
