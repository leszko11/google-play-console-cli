package registry

import (
	"github.com/leszko11/google-play-console-cli/internal/cli/apps"
	"github.com/leszko11/google-play-console-cli/internal/cli/auth"
	"github.com/leszko11/google-play-console-cli/internal/cli/completion"
	"github.com/leszko11/google-play-console-cli/internal/cli/edits"
	"github.com/leszko11/google-play-console-cli/internal/cli/tracks"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func Register(root *ffcli.Command) {
	root.Subcommands = []*ffcli.Command{
		auth.NewCommand(auth.Deps{}),
		apps.NewCommand(apps.Deps{}),
		edits.NewCommand(edits.Deps{}),
		tracks.NewCommand(tracks.Deps{}),
		completion.NewCommand(),
	}
}
