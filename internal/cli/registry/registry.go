package registry

import (
	"github.com/leszko11/google-play-console-cli/internal/cli/apks"
	"github.com/leszko11/google-play-console-cli/internal/cli/apps"
	"github.com/leszko11/google-play-console-cli/internal/cli/auth"
	"github.com/leszko11/google-play-console-cli/internal/cli/bundles"
	"github.com/leszko11/google-play-console-cli/internal/cli/completion"
	"github.com/leszko11/google-play-console-cli/internal/cli/deobfuscation"
	"github.com/leszko11/google-play-console-cli/internal/cli/deploy"
	"github.com/leszko11/google-play-console-cli/internal/cli/edits"
	"github.com/leszko11/google-play-console-cli/internal/cli/grants"
	"github.com/leszko11/google-play-console-cli/internal/cli/internalsharing"
	"github.com/leszko11/google-play-console-cli/internal/cli/products"
	"github.com/leszko11/google-play-console-cli/internal/cli/purchases"
	"github.com/leszko11/google-play-console-cli/internal/cli/reviews"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/cli/subscriptions"
	"github.com/leszko11/google-play-console-cli/internal/cli/tracks"
	"github.com/leszko11/google-play-console-cli/internal/cli/users"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Deps struct {
	GlobalFlags *shared.GlobalFlags
}

func Register(root *ffcli.Command, deps Deps) {
	_ = deps

	root.Subcommands = []*ffcli.Command{
		auth.NewCommand(auth.Deps{}),
		apps.NewCommand(apps.Deps{}),
		edits.NewCommand(edits.Deps{}),
		tracks.NewCommand(tracks.Deps{}),
		apks.NewCommand(apks.Deps{}),
		bundles.NewCommand(bundles.Deps{}),
		deobfuscation.NewCommand(deobfuscation.Deps{}),
		deploy.NewCommand(deploy.Deps{}),
		reviews.NewCommand(reviews.Deps{}),
		subscriptions.NewCommand(subscriptions.Deps{}),
		products.NewCommand(products.Deps{}),
		purchases.NewCommand(purchases.Deps{}),
		users.NewCommand(users.Deps{}),
		grants.NewCommand(grants.Deps{}),
		internalsharing.NewCommand(internalsharing.Deps{}),
		completion.NewCommand(),
	}
}
