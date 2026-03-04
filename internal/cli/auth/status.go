package auth

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewStatusCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var output string
	fs.StringVar(&output, "output", "", "Output format: json, table")

	return &ffcli.Command{
		Name:      "status",
		ShortHelp: "Show authentication status",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			status := buildStatus(cfg)
			resolvedOutput := shared.ResolveOutput(output)
			switch resolvedOutput {
			case "json":
				return shared.WriteJSON(deps.Stdout, status)
			case "table":
				return writeStatusTable(deps.Stdout, status)
			default:
				return shared.UsageErrorf("unsupported output format %q", strings.TrimSpace(resolvedOutput))
			}
		},
	}
}

type statusPayload struct {
	ActiveProfile      string `json:"activeProfile,omitempty"`
	Authenticated      bool   `json:"authenticated"`
	ServiceAccountPath string `json:"serviceAccountPath,omitempty"`
	LastValidatedAt    string `json:"lastValidatedAt,omitempty"`
	DeveloperID        string `json:"developerId,omitempty"`
}

func buildStatus(cfg config.Config) statusPayload {
	out := statusPayload{
		ActiveProfile: cfg.ActiveProfile,
		Authenticated: false,
	}

	if cfg.ActiveProfile == "" {
		return out
	}

	profile, ok := cfg.Profiles[cfg.ActiveProfile]
	if !ok {
		return out
	}

	out.Authenticated = profile.ServiceAccountPath != ""
	out.ServiceAccountPath = profile.ServiceAccountPath
	out.LastValidatedAt = profile.LastValidatedAt
	if profile.DeveloperID != "" {
		out.DeveloperID = profile.DeveloperID
	}
	return out
}

func writeStatusTable(out io.Writer, status statusPayload) error {
	if _, err := fmt.Fprintln(out, "FIELD\tVALUE"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "authenticated\t%t\n", status.Authenticated); err != nil {
		return err
	}
	if status.ActiveProfile != "" {
		if _, err := fmt.Fprintf(out, "activeProfile\t%s\n", status.ActiveProfile); err != nil {
			return err
		}
	}
	if status.ServiceAccountPath != "" {
		if _, err := fmt.Fprintf(out, "serviceAccountPath\t%s\n", status.ServiceAccountPath); err != nil {
			return err
		}
	}
	if status.LastValidatedAt != "" {
		if _, err := fmt.Fprintf(out, "lastValidatedAt\t%s\n", status.LastValidatedAt); err != nil {
			return err
		}
	}
	if status.DeveloperID != "" {
		if _, err := fmt.Fprintf(out, "developerId\t%s\n", status.DeveloperID); err != nil {
			return err
		}
	}
	return nil
}
