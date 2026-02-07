package auth

import (
	"context"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewLogoutCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	return &ffcli.Command{
		Name:      "logout",
		ShortHelp: "Log out current profile",
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			if cfg.ActiveProfile != "" && cfg.Profiles != nil {
				delete(cfg.Profiles, cfg.ActiveProfile)
			}
			cfg.ActiveProfile = ""

			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}

			return writeJSON(deps.Stdout, map[string]any{
				"activeProfile": "",
				"authenticated": false,
			})
		},
	}
}
