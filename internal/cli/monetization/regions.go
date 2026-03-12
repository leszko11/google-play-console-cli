package monetization

import (
	"context"
	"flag"
	"fmt"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func newRegionsCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("regions", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName string
	fs.StringVar(&packageName, "package-name", "", "Package name")

	return &ffcli.Command{
		Name:      "regions",
		ShortHelp: "List billable monetization regions",
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

			regions, err := client.GetMonetizationRegions(requestCtx, pkg)
			if err != nil {
				return fmt.Errorf("failed to list monetization regions: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":    pkg,
				"regionsVersion": regions.RegionsVersion,
				"regions":        regions.Regions,
				"count":          len(regions.Regions),
			})
		},
	}
}
