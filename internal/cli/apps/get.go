package apps

import (
	"context"
	"flag"
	"fmt"

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
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown, csv, tsv")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get app details for a package",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			pkg, err := shared.ResolvePackageName(packageName)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
			})
			if err != nil {
				return err
			}
			defer cancel()

			app, err := client.GetApp(requestCtx, pkg)
			if err != nil {
				return fmt.Errorf("failed to fetch app: %w", err)
			}

			return writeGetOutput(deps, shared.ResolveOutput(output), app)
		},
	}
}

func writeGetOutput(deps Deps, output string, app gpc.AppInfo) error {
	switch output {
	case "json":
		return shared.WriteJSON(deps.Stdout, app)
	case "table":
		if _, err := fmt.Fprintln(deps.Stdout, "PACKAGE"); err != nil {
			return err
		}
		_, err := fmt.Fprintln(deps.Stdout, app.PackageName)
		return err
	case "csv", "tsv":
		return shared.WriteDelimited(deps.Stdout, output, []string{"packageName"}, [][]string{{app.PackageName}})
	case "markdown":
		return shared.WriteMarkdownTable(deps.Stdout, []string{"package"}, [][]string{{app.PackageName}})
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}
