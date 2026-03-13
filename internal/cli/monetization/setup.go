package monetization

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

const (
	setupStatusCompleted = "completed"
	setupStatusDryRun    = "dry-run"
	setupStatusFailed    = "failed"
)

type Client interface {
	ListSubscriptions(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (gpc.SubscriptionsListInfo, error)
	GetSubscriptionRaw(ctx context.Context, packageName, productID string) (*androidpublisher.Subscription, error)
	CreateSubscription(ctx context.Context, packageName string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error)
	UpdateSubscription(ctx context.Context, packageName, productID string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error)
	ActivateSubscriptionBasePlan(ctx context.Context, packageName, productID, basePlanID string) ([]gpc.SubscriptionInfo, error)
	ListSubscriptionOffers(ctx context.Context, packageName, productID, basePlanID string, pageSize int64, pageToken string, paginate bool) (gpc.SubscriptionOffersListInfo, error)
	CreateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID string, offer *androidpublisher.SubscriptionOffer) (gpc.SubscriptionOfferInfo, error)
	UpdateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string, offer *androidpublisher.SubscriptionOffer, updateMask string) (gpc.SubscriptionOfferInfo, error)
	ActivateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string) (gpc.SubscriptionOfferInfo, error)
	GetMonetizationRegions(ctx context.Context, packageName string) (gpc.MonetizationRegionsInfo, error)
}

type stepResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type setupOptions struct {
	PackageName  string
	ManifestPath string
	Confirm      bool
	DryRun       bool
	Activate     bool
}

type SetupOptions = setupOptions

type setupResult struct {
	PackageName         string       `json:"packageName"`
	Manifest            string       `json:"manifest"`
	ProductID           string       `json:"productId"`
	Status              string       `json:"status"`
	PlannedActions      []string     `json:"plannedActions,omitempty"`
	CreatedSubscription bool         `json:"createdSubscription"`
	CreatedBasePlans    []string     `json:"createdBasePlans,omitempty"`
	CreatedOffers       []string     `json:"createdOffers,omitempty"`
	ActivatedBasePlans  []string     `json:"activatedBasePlans,omitempty"`
	ActivatedOffers     []string     `json:"activatedOffers,omitempty"`
	Steps               []stepResult `json:"steps"`
}

func newSetupCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts setupOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.ManifestPath, "manifest", "", "Path to monetization manifest (.json/.yaml/.yml)")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm creating monetization resources (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Validate manifest and check for conflicts without creating resources")
	fs.BoolVar(&opts.Activate, "activate", false, "Activate created base plans and offers after creation")

	return &ffcli.Command{
		Name:      "setup",
		ShortHelp: "Create a subscription product from a YAML or JSON manifest",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, parsedManifest, err := validateSetupOptions(opts)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
			})
			if err != nil {
				return err
			}
			defer cancel()

			return runSetup(requestCtx, client, deps.Stdout, opts, parsedManifest)
		},
	}
}

func validateSetupOptions(opts setupOptions) (setupOptions, manifest, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return setupOptions{}, manifest{}, err
	}
	opts.PackageName = pkg
	opts.ManifestPath = strings.TrimSpace(opts.ManifestPath)
	if opts.ManifestPath == "" {
		return setupOptions{}, manifest{}, shared.UsageErrorf("--manifest is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return setupOptions{}, manifest{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}

	parsedManifest, err := loadManifest(opts.ManifestPath)
	if err != nil {
		return setupOptions{}, manifest{}, err
	}
	return opts, parsedManifest, nil
}

func runSetup(ctx context.Context, client Client, out io.Writer, opts setupOptions, manifest manifest) error {
	manifest.Subscription.PackageName = opts.PackageName
	for _, offer := range manifest.Offers {
		offer.PackageName = opts.PackageName
	}

	result := setupResult{
		PackageName:    opts.PackageName,
		Manifest:       opts.ManifestPath,
		ProductID:      manifest.Subscription.ProductId,
		Status:         setupStatusFailed,
		PlannedActions: manifest.plannedActions(opts.Activate),
		Steps:          make([]stepResult, 0, 2+len(manifest.Subscription.BasePlans)+(len(manifest.Offers)*2)),
	}

	fail := func(step string, err error) error {
		result.Steps = append(result.Steps, stepResult{Name: step, Status: "error", Error: err.Error()})
		_ = shared.WriteJSON(out, result)
		return err
	}

	subscriptions, err := client.ListSubscriptions(ctx, opts.PackageName, 0, "", true)
	if err != nil {
		return fail("check_conflicts", fmt.Errorf("failed to list subscriptions: %w", err))
	}
	for _, subscription := range subscriptions.Subscriptions {
		if subscription.ProductID == manifest.Subscription.ProductId {
			return fail("check_conflicts", fmt.Errorf("subscription %q already exists", manifest.Subscription.ProductId))
		}
	}
	result.Steps = append(result.Steps, stepResult{Name: "check_conflicts", Status: "ok"})

	if opts.DryRun {
		result.Status = setupStatusDryRun
		return shared.WriteJSON(out, result)
	}

	if _, err := client.CreateSubscription(ctx, opts.PackageName, manifest.Subscription); err != nil {
		return fail("create_subscription", fmt.Errorf("failed to create subscription: %w", err))
	}
	result.CreatedSubscription = true
	for _, basePlan := range manifest.Subscription.BasePlans {
		result.CreatedBasePlans = append(result.CreatedBasePlans, basePlan.BasePlanId)
	}
	result.Steps = append(result.Steps, stepResult{Name: "create_subscription", Status: "ok"})

	if opts.Activate {
		for _, basePlan := range manifest.Subscription.BasePlans {
			if _, err := client.ActivateSubscriptionBasePlan(ctx, opts.PackageName, manifest.Subscription.ProductId, basePlan.BasePlanId); err != nil {
				return fail("activate_base_plan_"+basePlan.BasePlanId, fmt.Errorf("failed to activate base plan %q: %w", basePlan.BasePlanId, err))
			}
			result.ActivatedBasePlans = append(result.ActivatedBasePlans, basePlan.BasePlanId)
			result.Steps = append(result.Steps, stepResult{Name: "activate_base_plan_" + basePlan.BasePlanId, Status: "ok"})
		}
	} else {
		result.Steps = append(result.Steps, stepResult{Name: "activate_base_plans", Status: "skipped"})
	}

	for _, offer := range manifest.Offers {
		if _, err := client.CreateSubscriptionOffer(ctx, opts.PackageName, manifest.Subscription.ProductId, offer.BasePlanId, offer); err != nil {
			return fail("create_offer_"+offer.OfferId, fmt.Errorf("failed to create offer %q: %w", offer.OfferId, err))
		}
		result.CreatedOffers = append(result.CreatedOffers, offer.OfferId)
		result.Steps = append(result.Steps, stepResult{Name: "create_offer_" + offer.OfferId, Status: "ok"})

		if opts.Activate {
			if _, err := client.ActivateSubscriptionOffer(ctx, opts.PackageName, manifest.Subscription.ProductId, offer.BasePlanId, offer.OfferId); err != nil {
				return fail("activate_offer_"+offer.OfferId, fmt.Errorf("failed to activate offer %q: %w", offer.OfferId, err))
			}
			result.ActivatedOffers = append(result.ActivatedOffers, offer.OfferId)
			result.Steps = append(result.Steps, stepResult{Name: "activate_offer_" + offer.OfferId, Status: "ok"})
		}
	}

	if !opts.Activate {
		result.Steps = append(result.Steps, stepResult{Name: "activate_offers", Status: "skipped"})
	}
	result.Status = setupStatusCompleted
	return shared.WriteJSON(out, result)
}

func RunSetup(ctx context.Context, client Client, out io.Writer, opts SetupOptions, manifest Manifest) error {
	return runSetup(ctx, client, out, opts, manifest)
}
