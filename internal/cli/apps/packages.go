package apps

import (
	"context"
	"flag"
	"fmt"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewAddPackageCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("add-package", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName string
	fs.StringVar(&packageName, "package-name", "", "Package name to store for list/verify flows")

	return &ffcli.Command{
		Name:      "add-package",
		ShortHelp: "Add package to local app list",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			pkg, err := shared.ResolvePackageName(packageName)
			if err != nil {
				return err
			}

			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			for _, configured := range cfg.Packages {
				if configured == pkg {
					return shared.WriteJSON(deps.Stdout, map[string]any{
						"action":      "already-present",
						"packageName": pkg,
						"packages":    cfg.Packages,
					})
				}
			}

			cfg.Packages = append(cfg.Packages, pkg)
			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"action":      "added",
				"packageName": pkg,
				"packages":    cfg.Packages,
			})
		},
	}
}

func NewRemovePackageCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("remove-package", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName string
	fs.StringVar(&packageName, "package-name", "", "Package name to remove from local app list")

	return &ffcli.Command{
		Name:      "remove-package",
		ShortHelp: "Remove package from local app list",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			pkg, err := shared.ResolvePackageName(packageName)
			if err != nil {
				return err
			}

			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			filtered := make([]string, 0, len(cfg.Packages))
			removed := false
			for _, configured := range cfg.Packages {
				if configured == pkg {
					removed = true
					continue
				}
				filtered = append(filtered, configured)
			}
			if !removed {
				return fmt.Errorf("package %q not configured", pkg)
			}

			cfg.Packages = filtered
			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"action":      "removed",
				"packageName": pkg,
				"packages":    cfg.Packages,
			})
		},
	}
}
