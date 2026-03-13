package completion

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewCommand(root *ffcli.Command) *ffcli.Command {
	return &ffcli.Command{
		Name:      "completion",
		ShortHelp: "Generate shell completion script",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			shellCommand("bash", func() string { return bashScript(root) }, os.Stdout),
			shellCommand("zsh", func() string { return zshScript(root) }, os.Stdout),
			shellCommand("fish", func() string { return fishScript(root) }, os.Stdout),
		},
	}
}

func shellCommand(name string, render func() string, out io.Writer) *ffcli.Command {
	return &ffcli.Command{
		Name:      name,
		ShortHelp: fmt.Sprintf("Print %s completion script", name),
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			_, err := io.WriteString(out, render())
			return err
		},
	}
}
