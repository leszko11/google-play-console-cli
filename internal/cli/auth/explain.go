package auth

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewExplainCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var output string
	fs.StringVar(&output, "output", "", "Output format: json, table")

	return &ffcli.Command{
		Name:      "explain",
		ShortHelp: "Explain how credentials are resolved for the current profile",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			result := shared.BuildAuthExplainSnapshot(cfg, deps.LookupEnv)
			switch resolved := shared.ResolveOutput(output); resolved {
			case "json":
				return shared.WriteJSON(deps.Stdout, result)
			case "table":
				return writeExplainTable(deps.Stdout, result)
			default:
				return shared.UsageErrorf("unsupported output format %q", strings.TrimSpace(resolved))
			}
		},
	}
}

func writeExplainTable(out io.Writer, result shared.AuthExplainSnapshot) error {
	if _, err := fmt.Fprintln(out, "FIELD\tVALUE"); err != nil {
		return err
	}
	rows := [][2]string{
		{"selectedProfile", result.SelectedProfile},
		{"activeProfile", result.Status.ActiveProfile},
		{"authenticated", fmt.Sprintf("%t", result.Status.Authenticated)},
		{"health", result.Status.Health},
		{"source", result.FinalSource},
		{"configPath", result.ConfigPath},
		{"persistedStorage", result.PersistedStorage},
		{"strictAuthEnabled", fmt.Sprintf("%t", result.StrictAuth.Enabled)},
		{"strictAuthWouldFail", fmt.Sprintf("%t", result.StrictAuth.WouldFail)},
		{"mixedSourceRisk", fmt.Sprintf("%t", result.MixedSourceRisk)},
		{"keychainBypassed", fmt.Sprintf("%t", result.Keychain.Bypassed)},
		{"keychainAvailable", fmt.Sprintf("%t", result.Keychain.Available)},
		{"ciRecommendation", result.CIRecommendation},
	}
	for _, row := range rows {
		if strings.TrimSpace(row[1]) == "" {
			continue
		}
		if _, err := fmt.Fprintf(out, "%s\t%s\n", row[0], row[1]); err != nil {
			return err
		}
	}
	for _, source := range result.Sources {
		value := fmt.Sprintf("available=%t selected=%t", source.Available, source.Selected)
		if source.Detail != "" {
			value += " detail=" + source.Detail
		}
		if _, err := fmt.Fprintf(out, "source.%s\t%s\n", source.Name, value); err != nil {
			return err
		}
	}
	for _, warning := range result.Status.Warnings {
		if _, err := fmt.Fprintf(out, "warning\t%s\n", warning); err != nil {
			return err
		}
	}
	return nil
}
