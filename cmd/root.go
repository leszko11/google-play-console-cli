package cmd

import (
	"context"
	"flag"

	"github.com/leszko11/google-play-console-cli/internal/cli/registry"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func newRootCommand() *ffcli.Command {
	root := &ffcli.Command{
		Name:       "gpc",
		ShortUsage: "gpc [flags] <command>",
		ShortHelp:  "Google Play Console CLI",
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}

	registry.Register(root)
	return root
}
