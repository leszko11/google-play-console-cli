package apps

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewGetCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var (
		packageName string
		output      string
	)
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&output, "output", "json", "Output format: json, table, markdown")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get app details for a package",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			packageName = strings.TrimSpace(packageName)
			if packageName == "" {
				return fmt.Errorf("--package-name is required")
			}

			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			serviceAccountPath, err := resolveServiceAccountPath(cfg, deps.LookupEnv)
			if err != nil {
				return err
			}

			client, err := deps.NewClient(ctx, credential(serviceAccountPath))
			if err != nil {
				return err
			}

			app, err := client.GetApp(ctx, packageName)
			if err != nil {
				return fmt.Errorf("failed to fetch app: %w", err)
			}

			return writeGetOutput(deps, strings.ToLower(output), app)
		},
	}
}

func credential(path string) gpc.CredentialInput {
	return gpc.CredentialInput{ServiceAccountPath: path}
}

func writeGetOutput(deps Deps, output string, app gpc.AppInfo) error {
	switch output {
	case "json":
		out, err := shared.RenderJSON(app, false)
		if err != nil {
			return err
		}
		_, err = deps.Stdout.Write(out)
		return err
	case "table":
		if _, err := fmt.Fprintln(deps.Stdout, "PACKAGE"); err != nil {
			return err
		}
		_, err := fmt.Fprintln(deps.Stdout, app.PackageName)
		return err
	case "markdown":
		if _, err := fmt.Fprintln(deps.Stdout, "| package |"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(deps.Stdout, "| --- |"); err != nil {
			return err
		}
		_, err := fmt.Fprintf(deps.Stdout, "| %s |\n", app.PackageName)
		return err
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}
