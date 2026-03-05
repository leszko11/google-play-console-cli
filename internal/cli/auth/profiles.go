package auth

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type profileRow struct {
	Profile string `json:"profile"`
	Active  bool   `json:"active"`
	Storage string `json:"storage"`
}

type profilesPayload struct {
	SelectedProfile string       `json:"selectedProfile,omitempty"`
	Profiles        []profileRow `json:"profiles"`
	Warnings        []string     `json:"warnings,omitempty"`
}

func NewProfilesCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	return &ffcli.Command{
		Name:      "profiles",
		ShortHelp: "Manage authentication profiles",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newProfilesListCommand(deps),
		},
	}
}

func newProfilesListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var output string
	fs.StringVar(&output, "output", "", "Output format: json, table")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List authentication profiles",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			selected := shared.ResolveProfileName(cfg)
			warnings := []string{}
			keychainProfiles := map[string]struct{}{}
			if authresolver.ShouldBypassKeychain(deps.LookupEnv) {
				warnings = append(warnings, "keychain bypassed via GPC_BYPASS_KEYCHAIN")
			} else {
				profiles, err := authresolver.ListKeychainProfiles()
				if err == nil {
					for _, profile := range profiles {
						keychainProfiles[profile] = struct{}{}
					}
				} else if authresolver.IsKeyringUnavailable(err) {
					warnings = append(warnings, "system keychain unavailable; showing config profiles only")
				} else {
					return err
				}
			}

			names := map[string]struct{}{}
			for name := range cfg.Profiles {
				trimmed := strings.TrimSpace(name)
				if trimmed != "" {
					names[trimmed] = struct{}{}
				}
			}
			for name := range keychainProfiles {
				names[name] = struct{}{}
			}

			rows := make([]profileRow, 0, len(names))
			for name := range names {
				storage := "unknown"
				if _, ok := keychainProfiles[name]; ok {
					storage = "keychain"
				} else if profile, ok := cfg.Profiles[name]; ok && strings.TrimSpace(profile.ServiceAccountPath) != "" {
					storage = "config"
				}
				rows = append(rows, profileRow{
					Profile: name,
					Active:  name == selected,
					Storage: storage,
				})
			}
			sort.Slice(rows, func(i, j int) bool {
				return rows[i].Profile < rows[j].Profile
			})

			payload := profilesPayload{
				SelectedProfile: selected,
				Profiles:        rows,
			}
			if len(warnings) > 0 {
				payload.Warnings = warnings
			}

			switch shared.ResolveOutput(output) {
			case "json":
				return shared.WriteJSON(deps.Stdout, payload)
			case "table":
				if _, err := fmt.Fprintln(deps.Stdout, "PROFILE\tACTIVE\tSTORAGE"); err != nil {
					return err
				}
				for _, row := range rows {
					active := "no"
					if row.Active {
						active = "yes"
					}
					if _, err := fmt.Fprintf(deps.Stdout, "%s\t%s\t%s\n", row.Profile, active, row.Storage); err != nil {
						return err
					}
				}
				for _, warning := range warnings {
					if _, err := fmt.Fprintf(deps.Stderr, "warning: %s\n", warning); err != nil {
						return err
					}
				}
				return nil
			default:
				return shared.UsageErrorf("unsupported output format %q", strings.TrimSpace(shared.ResolveOutput(output)))
			}
		},
	}
}
