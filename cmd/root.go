package cmd

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newRootCommand() *ffcli.Command {
	return &ffcli.Command{
		Name:      "gpc",
		ShortHelp: "Google Play Console CLI",
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}
}
