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
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown")

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

			status := buildStatus(cfg, deps.LookupEnv)
			resolvedOutput := shared.ResolveOutput(output)
			switch resolvedOutput {
			case "json":
				return shared.WriteJSON(deps.Stdout, status)
			case "table":
				return writeStatusTable(deps.Stdout, status)
			case "markdown":
				return writeStatusMarkdown(deps.Stdout, status)
			default:
				return shared.UsageErrorf("unsupported output format %q", strings.TrimSpace(resolvedOutput))
			}
		},
	}
}

type statusPayload struct {
	shared.AuthStatusSnapshot
}

func buildStatus(cfg config.Config, lookupEnv func(string) string) statusPayload {
	return statusPayload{AuthStatusSnapshot: shared.BuildAuthStatusSnapshot(cfg, lookupEnv)}
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
	if status.SelectedProfile != "" {
		if _, err := fmt.Fprintf(out, "selectedProfile\t%s\n", status.SelectedProfile); err != nil {
			return err
		}
	}
	if status.Source != "" {
		if _, err := fmt.Fprintf(out, "source\t%s\n", status.Source); err != nil {
			return err
		}
	}
	if status.StorageBackend != "" {
		if _, err := fmt.Fprintf(out, "storageBackend\t%s\n", status.StorageBackend); err != nil {
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
	for _, warning := range status.Warnings {
		if _, err := fmt.Fprintf(out, "warning\t%s\n", warning); err != nil {
			return err
		}
	}
	return nil
}

func writeStatusMarkdown(out io.Writer, status statusPayload) error {
	rows := [][]string{{"authenticated", fmt.Sprintf("%t", status.Authenticated)}}
	if status.ActiveProfile != "" {
		rows = append(rows, []string{"activeProfile", status.ActiveProfile})
	}
	if status.SelectedProfile != "" {
		rows = append(rows, []string{"selectedProfile", status.SelectedProfile})
	}
	if status.Source != "" {
		rows = append(rows, []string{"source", status.Source})
	}
	if status.StorageBackend != "" {
		rows = append(rows, []string{"storageBackend", status.StorageBackend})
	}
	if status.ServiceAccountPath != "" {
		rows = append(rows, []string{"serviceAccountPath", status.ServiceAccountPath})
	}
	if status.LastValidatedAt != "" {
		rows = append(rows, []string{"lastValidatedAt", status.LastValidatedAt})
	}
	if status.DeveloperID != "" {
		rows = append(rows, []string{"developerId", status.DeveloperID})
	}
	for _, warning := range status.Warnings {
		rows = append(rows, []string{"warning", warning})
	}
	return shared.WriteMarkdownTable(out, []string{"field", "value"}, rows)
}
