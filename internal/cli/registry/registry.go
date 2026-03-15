package registry

import (
	"github.com/leszko11/google-play-console-cli/internal/cli/apks"
	"github.com/leszko11/google-play-console-cli/internal/cli/appinit"
	"github.com/leszko11/google-play-console-cli/internal/cli/apprecoveries"
	"github.com/leszko11/google-play-console-cli/internal/cli/apps"
	"github.com/leszko11/google-play-console-cli/internal/cli/auth"
	"github.com/leszko11/google-play-console-cli/internal/cli/bundles"
	"github.com/leszko11/google-play-console-cli/internal/cli/changelog"
	"github.com/leszko11/google-play-console-cli/internal/cli/completion"
	"github.com/leszko11/google-play-console-cli/internal/cli/deobfuscation"
	"github.com/leszko11/google-play-console-cli/internal/cli/deploy"
	"github.com/leszko11/google-play-console-cli/internal/cli/devicetierconfigs"
	"github.com/leszko11/google-play-console-cli/internal/cli/diff"
	"github.com/leszko11/google-play-console-cli/internal/cli/doctor"
	"github.com/leszko11/google-play-console-cli/internal/cli/e2e"
	"github.com/leszko11/google-play-console-cli/internal/cli/edits"
	"github.com/leszko11/google-play-console-cli/internal/cli/externaltransactions"
	"github.com/leszko11/google-play-console-cli/internal/cli/generatedapks"
	"github.com/leszko11/google-play-console-cli/internal/cli/grants"
	"github.com/leszko11/google-play-console-cli/internal/cli/iap"
	"github.com/leszko11/google-play-console-cli/internal/cli/internalsharing"
	"github.com/leszko11/google-play-console-cli/internal/cli/listing"
	"github.com/leszko11/google-play-console-cli/internal/cli/migrate"
	"github.com/leszko11/google-play-console-cli/internal/cli/monetization"
	"github.com/leszko11/google-play-console-cli/internal/cli/notify"
	"github.com/leszko11/google-play-console-cli/internal/cli/orders"
	"github.com/leszko11/google-play-console-cli/internal/cli/products"
	"github.com/leszko11/google-play-console-cli/internal/cli/purchases"
	"github.com/leszko11/google-play-console-cli/internal/cli/release"
	"github.com/leszko11/google-play-console-cli/internal/cli/reports"
	"github.com/leszko11/google-play-console-cli/internal/cli/reviews"
	"github.com/leszko11/google-play-console-cli/internal/cli/rollback"
	"github.com/leszko11/google-play-console-cli/internal/cli/setup"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/cli/status"
	"github.com/leszko11/google-play-console-cli/internal/cli/subscriptions"
	"github.com/leszko11/google-play-console-cli/internal/cli/systemapks"
	"github.com/leszko11/google-play-console-cli/internal/cli/tracks"
	"github.com/leszko11/google-play-console-cli/internal/cli/users"
	"github.com/leszko11/google-play-console-cli/internal/cli/workflow"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Deps struct {
	GlobalFlags *shared.GlobalFlags
}

func Register(root *ffcli.Command, deps Deps) {
	_ = deps

	root.Subcommands = []*ffcli.Command{
		auth.NewCommand(auth.Deps{}),
		appinit.NewBootstrapCommand(appinit.Deps{}),
		appinit.NewCommand(appinit.Deps{}),
		apps.NewCommand(apps.Deps{}),
		apprecoveries.NewCommand(apprecoveries.Deps{}),
		changelog.NewCommand(changelog.Deps{}),
		edits.NewCommand(edits.Deps{}),
		tracks.NewCommand(tracks.Deps{}),
		apks.NewCommand(apks.Deps{}),
		bundles.NewCommand(bundles.Deps{}),
		deobfuscation.NewCommand(deobfuscation.Deps{}),
		deploy.NewCommand(deploy.Deps{}),
		diff.NewCommand(diff.Deps{}),
		doctor.NewCommand(doctor.Deps{}),
		e2e.NewCommand(e2e.Deps{}),
		release.NewCommand(release.Deps{}),
		rollback.NewCommand(rollback.Deps{}),
		setup.NewCommand(setup.Deps{}),
		status.NewCommand(status.Deps{}),
		reviews.NewCommand(reviews.Deps{}),
		reports.NewCommand(reports.Deps{}),
		orders.NewCommand(orders.Deps{}),
		externaltransactions.NewCommand(externaltransactions.Deps{}),
		devicetierconfigs.NewCommand(devicetierconfigs.Deps{}),
		systemapks.NewCommand(systemapks.Deps{}),
		generatedapks.NewCommand(generatedapks.Deps{}),
		subscriptions.NewCommand(subscriptions.Deps{}),
		monetization.NewCommand(monetization.Deps{}),
		migrate.NewCommand(migrate.Deps{}),
		notify.NewCommand(notify.Deps{}),
		products.NewCommand(products.Deps{}),
		iap.NewCommand(iap.Deps{}),
		listing.NewCommand(listing.Deps{}),
		purchases.NewCommand(purchases.Deps{}),
		users.NewCommand(users.Deps{}),
		workflow.NewCommand(workflow.Deps{}),
		grants.NewCommand(grants.Deps{}),
		internalsharing.NewCommand(internalsharing.Deps{}),
		completion.NewCommand(root),
	}
}
