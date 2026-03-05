package auth

import (
	"context"
	"flag"
	"strings"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
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
				return shared.UsageErrorf("--profile is required")
			}

			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			exists := false
			if cfg.Profiles != nil {
				if _, ok := cfg.Profiles[profile]; ok {
					exists = true
				}
			}
			if !exists && !authresolver.ShouldBypassKeychain(deps.LookupEnv) {
				names, err := authresolver.ListKeychainProfiles()
				if err != nil && !authresolver.IsKeyringUnavailable(err) {
					return err
				}
				for _, name := range names {
					if name == profile {
						exists = true
						break
					}
				}
			}
			if !exists {
				return shared.UsageErrorf("profile %q not found", profile)
			}

			if cfg.Profiles == nil {
				cfg.Profiles = map[string]config.Profile{}
			}
			if _, ok := cfg.Profiles[profile]; !ok {
				cfg.Profiles[profile] = config.Profile{}
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
