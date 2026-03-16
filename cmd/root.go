package cmd

import (
	"context"
	"flag"
	"os"

	"github.com/leszko11/google-play-console-cli/internal/cli/registry"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func newRootCommand() *ffcli.Command {
	var showVersion bool
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	globalFlags := &shared.GlobalFlags{}
	shared.BindGlobalFlags(fs, globalFlags)
	fs.BoolVar(&showVersion, "version", false, "Show build version information")

	root := &ffcli.Command{
		Name:       "gpc",
		ShortUsage: "gpc [flags] <command>",
		ShortHelp:  "Google Play Console CLI",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			if showVersion {
				return shared.WriteJSON(os.Stdout, versionPayload())
			}
			return flag.ErrHelp
		},
	}

	registry.Register(root, registry.Deps{GlobalFlags: globalFlags, CurrentVersion: Version})
	return root
}
