package apps

import (
	"context"
	"flag"
	"fmt"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type listItem struct {
	PackageName string `json:"packageName"`
	Status      string `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
}

func NewListCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var (
		verify bool
		output string
	)
	fs.BoolVar(&verify, "verify", false, "Verify API access for each configured package")
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown, csv, tsv")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List configured packages",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			items := make([]listItem, 0, len(cfg.Packages))
			for _, pkg := range cfg.Packages {
				items = append(items, listItem{PackageName: pkg})
			}

			if verify && len(items) > 0 {
				resolved, err := shared.ResolveCredentials(cfg, deps.LookupEnv)
				if err != nil {
					return err
				}

				requestCtx, cancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
				defer cancel()

				client, err := deps.NewClient(requestCtx, resolved.Input)
				if err != nil {
					return err
				}

				for i := range items {
					if err := client.VerifyPackageAccess(requestCtx, items[i].PackageName); err != nil {
						items[i].Status = "error"
						items[i].Error = err.Error()
						continue
					}
					items[i].Status = "ok"
				}
			}

			return writeListOutput(deps, shared.ResolveOutput(output), items)
		},
	}
}

func writeListOutput(deps Deps, output string, items []listItem) error {
	switch output {
	case "json":
		return shared.WriteJSON(deps.Stdout, items)
	case "table":
		if _, err := fmt.Fprintln(deps.Stdout, "PACKAGE\tSTATUS"); err != nil {
			return err
		}
		for _, item := range items {
			status := item.Status
			if status == "" {
				status = "configured"
			}
			if _, err := fmt.Fprintf(deps.Stdout, "%s\t%s\n", item.PackageName, status); err != nil {
				return err
			}
		}
		return nil
	case "csv", "tsv":
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			status := item.Status
			if status == "" {
				status = "configured"
			}
			rows = append(rows, []string{item.PackageName, status, item.Error})
		}
		return shared.WriteDelimited(deps.Stdout, output, []string{"packageName", "status", "error"}, rows)
	case "markdown":
		if _, err := fmt.Fprintln(deps.Stdout, "| package | status |"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(deps.Stdout, "| --- | --- |"); err != nil {
			return err
		}
		for _, item := range items {
			status := item.Status
			if status == "" {
				status = "configured"
			}
			if _, err := fmt.Fprintf(deps.Stdout, "| %s | %s |\n", item.PackageName, status); err != nil {
				return err
			}
		}
		return nil
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}
