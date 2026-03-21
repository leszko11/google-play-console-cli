package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/registry"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/cli/suggest"
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
	}

	registry.Register(root, registry.Deps{GlobalFlags: globalFlags, CurrentVersion: Version})

	root.Exec = func(_ context.Context, args []string) error {
		if showVersion {
			return shared.WriteJSON(os.Stdout, versionPayload())
		}
		if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
			names := subcommandNames(root)
			if suggestions := suggest.Suggest(args[0], names, 3); len(suggestions) > 0 {
				fmt.Fprintf(os.Stderr, "Unknown command %q. Did you mean:\n", args[0])
				for _, s := range suggestions {
					fmt.Fprintf(os.Stderr, "  gpc %s\n", s)
				}
				fmt.Fprintln(os.Stderr)
			}
		}
		return flag.ErrHelp
	}

	return root
}

func subcommandNames(cmd *ffcli.Command) []string {
	names := make([]string, 0, len(cmd.Subcommands))
	for _, sub := range cmd.Subcommands {
		names = append(names, sub.Name)
	}
	return names
}
