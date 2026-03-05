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

func NewLogoutCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("logout", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var (
		profile string
		all     bool
	)
	fs.StringVar(&profile, "profile", "", "Profile name to remove (defaults to selected active profile)")
	fs.BoolVar(&all, "all", false, "Remove all stored profiles and credentials")

	return &ffcli.Command{
		Name:      "logout",
		ShortHelp: "Log out current profile",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			profile = strings.TrimSpace(profile)
			if all && profile != "" {
				return shared.UsageErrorf("--all and --profile are mutually exclusive")
			}

			removed := []string{}
			if all {
				for name := range cfg.Profiles {
					removed = append(removed, name)
				}
				cfg.Profiles = map[string]config.Profile{}
				cfg.ActiveProfile = ""
				if !authresolver.ShouldBypassKeychain(deps.LookupEnv) {
					if err := authresolver.RemoveAllProfileCredentials(); err != nil && !authresolver.IsKeyringUnavailable(err) {
						return err
					}
				}
			} else {
				target := profile
				if target == "" {
					target = shared.ResolveProfileName(cfg)
				}
				if target == "" {
					return shared.UsageErrorf("no profile selected")
				}

				if cfg.Profiles != nil {
					if _, ok := cfg.Profiles[target]; ok {
						delete(cfg.Profiles, target)
						removed = append(removed, target)
					}
				}
				if cfg.ActiveProfile == target {
					cfg.ActiveProfile = ""
				}

				if !authresolver.ShouldBypassKeychain(deps.LookupEnv) {
					if err := authresolver.RemoveProfileCredential(target); err != nil &&
						!authresolver.IsCredentialNotFound(err) &&
						!authresolver.IsKeyringUnavailable(err) {
						return err
					}
				}
				if len(removed) == 0 {
					removed = append(removed, target)
				}
			}

			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"activeProfile": "",
				"authenticated": false,
				"removed":       removed,
			})
		},
	}
}
