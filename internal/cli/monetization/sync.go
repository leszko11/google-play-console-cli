package monetization

import (
	"context"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

const offerSyncUpdateMask = "phases,regionalConfigs"

type syncResult struct {
	PackageName         string       `json:"packageName"`
	Manifest            string       `json:"manifest"`
	ProductID           string       `json:"productId"`
	Status              string       `json:"status"`
	PlannedActions      []string     `json:"plannedActions,omitempty"`
	CreatedSubscription bool         `json:"createdSubscription"`
	UpdatedSubscription bool         `json:"updatedSubscription"`
	CreatedBasePlans    []string     `json:"createdBasePlans,omitempty"`
	UpdatedBasePlans    []string     `json:"updatedBasePlans,omitempty"`
	CreatedOffers       []string     `json:"createdOffers,omitempty"`
	UpdatedOffers       []string     `json:"updatedOffers,omitempty"`
	ActivatedBasePlans  []string     `json:"activatedBasePlans,omitempty"`
	ActivatedOffers     []string     `json:"activatedOffers,omitempty"`
	UnmanagedListings   []string     `json:"unmanagedListings,omitempty"`
	UnmanagedBasePlans  []string     `json:"unmanagedBasePlans,omitempty"`
	UnmanagedOffers     []string     `json:"unmanagedOffers,omitempty"`
	Steps               []stepResult `json:"steps"`
}

type subscriptionMergeResult struct {
	Merged               *androidpublisher.Subscription
	Changed              bool
	CreatedBasePlans     []string
	UpdatedBasePlans     []string
	UnmanagedListings    []string
	UnmanagedBasePlans   []string
	ExistingBasePlanInfo map[string]*androidpublisher.BasePlan
}

func newSyncCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts setupOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.ManifestPath, "manifest", "", "Path to monetization manifest (.json/.yaml/.yml)")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm syncing monetization resources (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Validate manifest and show planned sync actions without applying changes")
	fs.BoolVar(&opts.Activate, "activate", false, "Activate synced base plans and offers when needed")

	return &ffcli.Command{
		Name:      "sync",
		ShortHelp: "Create or update a subscription product from a YAML or JSON manifest",
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

			return runSync(requestCtx, client, deps.Stdout, opts, parsedManifest)
		},
	}
}

func runSync(ctx context.Context, client Client, out io.Writer, opts setupOptions, manifest manifest) error {
	manifest.Subscription.PackageName = opts.PackageName
	for _, offer := range manifest.Offers {
		offer.PackageName = opts.PackageName
	}

	result := syncResult{
		PackageName: opts.PackageName,
		Manifest:    opts.ManifestPath,
		ProductID:   manifest.Subscription.ProductId,
		Status:      setupStatusFailed,
		Steps:       make([]stepResult, 0, 4+len(manifest.Subscription.BasePlans)+(len(manifest.Offers)*2)),
	}

	fail := func(step string, err error) error {
		result.Steps = append(result.Steps, stepResult{Name: step, Status: "error", Error: err.Error()})
		_ = shared.WriteJSON(out, result)
		return err
	}

	subscriptions, err := client.ListSubscriptions(ctx, opts.PackageName, 0, "", true)
	if err != nil {
		return fail("check_existing", fmt.Errorf("failed to list subscriptions: %w", err))
	}
	exists := false
	for _, subscription := range subscriptions.Subscriptions {
		if subscription.ProductID == manifest.Subscription.ProductId {
			exists = true
			break
		}
	}
	result.Steps = append(result.Steps, stepResult{Name: "check_existing", Status: "ok"})

	if !exists {
		result.PlannedActions = manifest.plannedActions(opts.Activate)
		if opts.DryRun {
			result.Status = setupStatusDryRun
			return shared.WriteJSON(out, result)
		}
		if err := applyCreateFlow(ctx, client, opts, manifest, &result, fail); err != nil {
			return err
		}
		result.Status = setupStatusCompleted
		return shared.WriteJSON(out, result)
	}

	existingSubscription, err := client.GetSubscriptionRaw(ctx, opts.PackageName, manifest.Subscription.ProductId)
	if err != nil {
		return fail("load_subscription", fmt.Errorf("failed to load subscription: %w", err))
	}
	result.Steps = append(result.Steps, stepResult{Name: "load_subscription", Status: "ok"})

	offerMap, err := listExistingOffers(ctx, client, opts.PackageName, manifest.Subscription.ProductId, existingSubscription.BasePlans)
	if err != nil {
		return fail("list_offers", fmt.Errorf("failed to list subscription offers: %w", err))
	}
	result.Steps = append(result.Steps, stepResult{Name: "list_offers", Status: "ok"})

	mergeResult := mergeSubscription(existingSubscription, manifest.Subscription)
	result.CreatedBasePlans = append(result.CreatedBasePlans, mergeResult.CreatedBasePlans...)
	result.UpdatedBasePlans = append(result.UpdatedBasePlans, mergeResult.UpdatedBasePlans...)
	result.UnmanagedListings = append(result.UnmanagedListings, mergeResult.UnmanagedListings...)
	result.UnmanagedBasePlans = append(result.UnmanagedBasePlans, mergeResult.UnmanagedBasePlans...)
	result.UnmanagedOffers = unmanagedOfferRefs(manifest.Offers, offerMap)
	result.PlannedActions = buildSyncPlannedActions(manifest, mergeResult, offerMap, opts.Activate)

	if opts.DryRun {
		result.Status = setupStatusDryRun
		return shared.WriteJSON(out, result)
	}

	if mergeResult.Changed {
		if _, err := client.UpdateSubscription(ctx, opts.PackageName, manifest.Subscription.ProductId, mergeResult.Merged); err != nil {
			return fail("update_subscription", fmt.Errorf("failed to update subscription: %w", err))
		}
		result.UpdatedSubscription = true
		result.Steps = append(result.Steps, stepResult{Name: "update_subscription", Status: "ok"})
	} else {
		result.Steps = append(result.Steps, stepResult{Name: "update_subscription", Status: "skipped"})
	}

	if opts.Activate {
		for _, basePlan := range manifest.Subscription.BasePlans {
			if !shouldActivateBasePlan(mergeResult.ExistingBasePlanInfo[basePlan.BasePlanId]) {
				continue
			}
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
		existingOffer, existsOnBasePlan := offerMap[offer.BasePlanId][offer.OfferId]
		var syncedOffer gpc.SubscriptionOfferInfo
		if existsOnBasePlan {
			updated, err := client.UpdateSubscriptionOffer(ctx, opts.PackageName, manifest.Subscription.ProductId, offer.BasePlanId, offer.OfferId, offer, offerSyncUpdateMask)
			if err != nil {
				return fail("update_offer_"+offer.OfferId, fmt.Errorf("failed to update offer %q: %w", offer.OfferId, err))
			}
			syncedOffer = updated
			result.UpdatedOffers = append(result.UpdatedOffers, offerRef(offer.BasePlanId, offer.OfferId))
			result.Steps = append(result.Steps, stepResult{Name: "update_offer_" + offer.OfferId, Status: "ok"})
		} else {
			created, err := client.CreateSubscriptionOffer(ctx, opts.PackageName, manifest.Subscription.ProductId, offer.BasePlanId, offer)
			if err != nil {
				return fail("create_offer_"+offer.OfferId, fmt.Errorf("failed to create offer %q: %w", offer.OfferId, err))
			}
			syncedOffer = created
			result.CreatedOffers = append(result.CreatedOffers, offerRef(offer.BasePlanId, offer.OfferId))
			result.Steps = append(result.Steps, stepResult{Name: "create_offer_" + offer.OfferId, Status: "ok"})
		}

		if !opts.Activate {
			continue
		}
		if syncedOffer.State == "ACTIVE" {
			continue
		}
		if existsOnBasePlan && existingOffer.State == "ACTIVE" {
			continue
		}
		if _, err := client.ActivateSubscriptionOffer(ctx, opts.PackageName, manifest.Subscription.ProductId, offer.BasePlanId, offer.OfferId); err != nil {
			return fail("activate_offer_"+offer.OfferId, fmt.Errorf("failed to activate offer %q: %w", offer.OfferId, err))
		}
		result.ActivatedOffers = append(result.ActivatedOffers, offerRef(offer.BasePlanId, offer.OfferId))
		result.Steps = append(result.Steps, stepResult{Name: "activate_offer_" + offer.OfferId, Status: "ok"})
	}

	if !opts.Activate {
		result.Steps = append(result.Steps, stepResult{Name: "activate_offers", Status: "skipped"})
	}
	result.Status = setupStatusCompleted
	return shared.WriteJSON(out, result)
}

func applyCreateFlow(ctx context.Context, client Client, opts setupOptions, manifest manifest, result *syncResult, fail func(string, error) error) error {
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
		result.CreatedOffers = append(result.CreatedOffers, offerRef(offer.BasePlanId, offer.OfferId))
		result.Steps = append(result.Steps, stepResult{Name: "create_offer_" + offer.OfferId, Status: "ok"})

		if !opts.Activate {
			continue
		}
		if _, err := client.ActivateSubscriptionOffer(ctx, opts.PackageName, manifest.Subscription.ProductId, offer.BasePlanId, offer.OfferId); err != nil {
			return fail("activate_offer_"+offer.OfferId, fmt.Errorf("failed to activate offer %q: %w", offer.OfferId, err))
		}
		result.ActivatedOffers = append(result.ActivatedOffers, offerRef(offer.BasePlanId, offer.OfferId))
		result.Steps = append(result.Steps, stepResult{Name: "activate_offer_" + offer.OfferId, Status: "ok"})
	}

	if !opts.Activate {
		result.Steps = append(result.Steps, stepResult{Name: "activate_offers", Status: "skipped"})
	}
	return nil
}

func mergeSubscription(existing, desired *androidpublisher.Subscription) subscriptionMergeResult {
	result := subscriptionMergeResult{
		Merged:               &androidpublisher.Subscription{PackageName: desired.PackageName, ProductId: desired.ProductId},
		ExistingBasePlanInfo: make(map[string]*androidpublisher.BasePlan, len(existing.BasePlans)),
	}

	existingListings := make(map[string]*androidpublisher.SubscriptionListing, len(existing.Listings))
	for _, listing := range existing.Listings {
		if listing == nil {
			continue
		}
		existingListings[strings.TrimSpace(listing.LanguageCode)] = listing
	}
	mergedListings := make([]*androidpublisher.SubscriptionListing, 0, len(existing.Listings)+len(desired.Listings))
	for _, listing := range desired.Listings {
		if listing == nil {
			continue
		}
		locale := strings.TrimSpace(listing.LanguageCode)
		current := existingListings[locale]
		if !listingEqual(current, listing) {
			result.Changed = true
		}
		delete(existingListings, locale)
		mergedListings = append(mergedListings, listing)
	}
	for locale, listing := range existingListings {
		if listing == nil {
			continue
		}
		result.UnmanagedListings = append(result.UnmanagedListings, locale)
		mergedListings = append(mergedListings, listing)
	}
	slices.SortFunc(mergedListings, func(a, b *androidpublisher.SubscriptionListing) int {
		return strings.Compare(strings.TrimSpace(a.LanguageCode), strings.TrimSpace(b.LanguageCode))
	})
	slices.Sort(result.UnmanagedListings)
	result.Merged.Listings = mergedListings

	existingBasePlans := make(map[string]*androidpublisher.BasePlan, len(existing.BasePlans))
	for _, basePlan := range existing.BasePlans {
		if basePlan == nil {
			continue
		}
		basePlanID := strings.TrimSpace(basePlan.BasePlanId)
		existingBasePlans[basePlanID] = basePlan
		result.ExistingBasePlanInfo[basePlanID] = basePlan
	}
	mergedBasePlans := make([]*androidpublisher.BasePlan, 0, len(existing.BasePlans)+len(desired.BasePlans))
	for _, basePlan := range desired.BasePlans {
		if basePlan == nil {
			continue
		}
		basePlanID := strings.TrimSpace(basePlan.BasePlanId)
		current := existingBasePlans[basePlanID]
		switch {
		case current == nil:
			result.Changed = true
			result.CreatedBasePlans = append(result.CreatedBasePlans, basePlanID)
		case !basePlanManagedEqual(current, basePlan):
			result.Changed = true
			result.UpdatedBasePlans = append(result.UpdatedBasePlans, basePlanID)
		}
		delete(existingBasePlans, basePlanID)
		mergedBasePlans = append(mergedBasePlans, basePlan)
	}
	for basePlanID, basePlan := range existingBasePlans {
		if basePlan == nil {
			continue
		}
		result.UnmanagedBasePlans = append(result.UnmanagedBasePlans, basePlanID)
		mergedBasePlans = append(mergedBasePlans, basePlan)
	}
	slices.SortFunc(mergedBasePlans, func(a, b *androidpublisher.BasePlan) int {
		return strings.Compare(strings.TrimSpace(a.BasePlanId), strings.TrimSpace(b.BasePlanId))
	})
	slices.Sort(result.CreatedBasePlans)
	slices.Sort(result.UpdatedBasePlans)
	slices.Sort(result.UnmanagedBasePlans)
	result.Merged.BasePlans = mergedBasePlans

	return result
}

func listExistingOffers(ctx context.Context, client Client, packageName, productID string, basePlans []*androidpublisher.BasePlan) (map[string]map[string]gpc.SubscriptionOfferInfo, error) {
	result := make(map[string]map[string]gpc.SubscriptionOfferInfo, len(basePlans))
	for _, basePlan := range basePlans {
		if basePlan == nil {
			continue
		}
		basePlanID := strings.TrimSpace(basePlan.BasePlanId)
		if basePlanID == "" {
			continue
		}
		offers, err := client.ListSubscriptionOffers(ctx, packageName, productID, basePlanID, 0, "", true)
		if err != nil {
			return nil, err
		}
		basePlanOffers := make(map[string]gpc.SubscriptionOfferInfo, len(offers.Offers))
		for _, offer := range offers.Offers {
			basePlanOffers[offer.OfferID] = offer
		}
		result[basePlanID] = basePlanOffers
	}
	return result, nil
}

func buildSyncPlannedActions(manifest manifest, mergeResult subscriptionMergeResult, offerMap map[string]map[string]gpc.SubscriptionOfferInfo, activate bool) []string {
	actions := make([]string, 0, 1+len(manifest.Subscription.BasePlans)+(len(manifest.Offers)*2))
	if mergeResult.Changed {
		actions = append(actions, fmt.Sprintf("update subscription %s", manifest.Subscription.ProductId))
	}
	for _, basePlan := range manifest.Subscription.BasePlans {
		if activate && basePlan != nil && shouldActivateBasePlan(mergeResult.ExistingBasePlanInfo[basePlan.BasePlanId]) {
			actions = append(actions, fmt.Sprintf("activate base plan %s", basePlan.BasePlanId))
		}
	}
	for _, offer := range manifest.Offers {
		if offer == nil {
			continue
		}
		if _, exists := offerMap[offer.BasePlanId][offer.OfferId]; exists {
			actions = append(actions, fmt.Sprintf("update offer %s on %s", offer.OfferId, offer.BasePlanId))
		} else {
			actions = append(actions, fmt.Sprintf("create offer %s on %s", offer.OfferId, offer.BasePlanId))
		}
		if !activate {
			continue
		}
		existingOffer, exists := offerMap[offer.BasePlanId][offer.OfferId]
		if !exists || existingOffer.State != "ACTIVE" {
			actions = append(actions, fmt.Sprintf("activate offer %s on %s", offer.OfferId, offer.BasePlanId))
		}
	}
	return dedupeStrings(actions)
}

func unmanagedOfferRefs(manifestOffers []*androidpublisher.SubscriptionOffer, offerMap map[string]map[string]gpc.SubscriptionOfferInfo) []string {
	managed := make(map[string]struct{}, len(manifestOffers))
	for _, offer := range manifestOffers {
		if offer == nil {
			continue
		}
		managed[offerRef(offer.BasePlanId, offer.OfferId)] = struct{}{}
	}

	out := make([]string, 0)
	for basePlanID, offers := range offerMap {
		for offerID := range offers {
			ref := offerRef(basePlanID, offerID)
			if _, exists := managed[ref]; exists {
				continue
			}
			out = append(out, ref)
		}
	}
	slices.Sort(out)
	return out
}

func offerRef(basePlanID, offerID string) string {
	return strings.TrimSpace(basePlanID) + "/" + strings.TrimSpace(offerID)
}

func shouldActivateBasePlan(basePlan *androidpublisher.BasePlan) bool {
	if basePlan == nil {
		return true
	}
	return strings.TrimSpace(basePlan.State) != "ACTIVE"
}

func listingEqual(current, desired *androidpublisher.SubscriptionListing) bool {
	if current == nil || desired == nil {
		return current == desired
	}
	return strings.TrimSpace(current.LanguageCode) == strings.TrimSpace(desired.LanguageCode) &&
		strings.TrimSpace(current.Title) == strings.TrimSpace(desired.Title) &&
		strings.TrimSpace(current.Description) == strings.TrimSpace(desired.Description)
}

func basePlanManagedEqual(current, desired *androidpublisher.BasePlan) bool {
	if current == nil || desired == nil {
		return current == desired
	}
	if strings.TrimSpace(current.BasePlanId) != strings.TrimSpace(desired.BasePlanId) {
		return false
	}
	currentBilling := ""
	if current.AutoRenewingBasePlanType != nil {
		currentBilling = strings.TrimSpace(current.AutoRenewingBasePlanType.BillingPeriodDuration)
	}
	desiredBilling := ""
	if desired.AutoRenewingBasePlanType != nil {
		desiredBilling = strings.TrimSpace(desired.AutoRenewingBasePlanType.BillingPeriodDuration)
	}
	if currentBilling != desiredBilling {
		return false
	}
	if len(current.RegionalConfigs) != len(desired.RegionalConfigs) {
		return false
	}

	currentRegions := make(map[string]*androidpublisher.RegionalBasePlanConfig, len(current.RegionalConfigs))
	for _, cfg := range current.RegionalConfigs {
		if cfg == nil {
			continue
		}
		currentRegions[strings.TrimSpace(cfg.RegionCode)] = cfg
	}
	for _, cfg := range desired.RegionalConfigs {
		if cfg == nil {
			continue
		}
		currentCfg := currentRegions[strings.TrimSpace(cfg.RegionCode)]
		if !basePlanRegionEqual(currentCfg, cfg) {
			return false
		}
	}
	return true
}

func basePlanRegionEqual(current, desired *androidpublisher.RegionalBasePlanConfig) bool {
	if current == nil || desired == nil {
		return current == desired
	}
	if strings.TrimSpace(current.RegionCode) != strings.TrimSpace(desired.RegionCode) {
		return false
	}
	if current.NewSubscriberAvailability != desired.NewSubscriberAvailability {
		return false
	}
	return moneyEqual(current.Price, desired.Price)
}

func moneyEqual(current, desired *androidpublisher.Money) bool {
	if current == nil || desired == nil {
		return current == desired
	}
	return strings.TrimSpace(current.CurrencyCode) == strings.TrimSpace(desired.CurrencyCode) &&
		current.Units == desired.Units &&
		current.Nanos == desired.Nanos
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
