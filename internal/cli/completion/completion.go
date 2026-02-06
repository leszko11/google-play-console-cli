package completion

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewCommand() *ffcli.Command {
	return &ffcli.Command{
		Name:      "completion",
		ShortHelp: "Generate shell completion script",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			shellCommand("bash", bashScript(), os.Stdout),
			shellCommand("zsh", zshScript(), os.Stdout),
			shellCommand("fish", fishScript(), os.Stdout),
		},
	}
}

func shellCommand(name, script string, out io.Writer) *ffcli.Command {
	return &ffcli.Command{
		Name:      name,
		ShortHelp: fmt.Sprintf("Print %s completion script", name),
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			_, err := io.WriteString(out, script)
			return err
		},
	}
}

func bashScript() string {
	return `# gpc bash completion
_gpc_completion() {
  COMPREPLY=()
}
complete -F _gpc_completion gpc
`
}

func zshScript() string {
	return `#compdef gpc
_gpc_completion() {
  reply=()
}
compctl -K _gpc_completion gpc
`
}

func fishScript() string {
	return `# gpc fish completion
complete -c gpc -f
`
}
