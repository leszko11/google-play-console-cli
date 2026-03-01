package auth

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewSwitchCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("switch", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var profile string
	fs.StringVar(&profile, "profile", "", "Profile name to activate")

	return &ffcli.Command{
		Name:      "switch",
		ShortHelp: "Switch active authentication profile",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			profile = strings.TrimSpace(profile)
			if profile == "" {
				return fmt.Errorf("--profile is required")
			}

			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			if cfg.Profiles == nil {
				return fmt.Errorf("profile %q not found", profile)
			}
			if _, ok := cfg.Profiles[profile]; !ok {
				return fmt.Errorf("profile %q not found", profile)
			}

			cfg.ActiveProfile = profile
			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"activeProfile": profile,
			})
		},
	}
}
