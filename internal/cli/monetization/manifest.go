package monetization

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"google.golang.org/api/androidpublisher/v3"
)

const (
	phaseTypeFreeTrial  = "FREE_TRIAL"
	phaseTypeDiscounted = "DISCOUNTED"
)

var regionCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

type manifest struct {
	Subscription *androidpublisher.Subscription
	Offers       []*androidpublisher.SubscriptionOffer
}

type Manifest = manifest

type rawManifest struct {
	Subscription rawSubscription `json:"subscription" yaml:"subscription"`
}

type rawSubscription struct {
	ProductID string                    `json:"productId" yaml:"productId"`
	Listings  map[string]rawListing     `json:"listings" yaml:"listings"`
	BasePlans []rawBasePlan             `json:"basePlans" yaml:"basePlans"`
	Offers    []rawSubscriptionOfferDef `json:"offers" yaml:"offers"`
}

type rawListing struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}

type rawBasePlan struct {
	BasePlanID      string              `json:"basePlanId" yaml:"basePlanId"`
	BillingPeriod   string              `json:"billingPeriod" yaml:"billingPeriod"`
	RegionalConfigs []rawRegionalConfig `json:"regionalConfigs" yaml:"regionalConfigs"`
}

type rawRegionalConfig struct {
	RegionCode                string   `json:"regionCode" yaml:"regionCode"`
	Price                     rawMoney `json:"price" yaml:"price"`
	NewSubscriberAvailability *bool    `json:"newSubscriberAvailability" yaml:"newSubscriberAvailability"`
}

type rawMoney struct {
	CurrencyCode string `json:"currencyCode" yaml:"currencyCode"`
	Units        int64  `json:"units" yaml:"units"`
	Nanos        int64  `json:"nanos" yaml:"nanos"`
}

type rawSubscriptionOfferDef struct {
	OfferID    string          `json:"offerId" yaml:"offerId"`
	BasePlanID string          `json:"basePlanId" yaml:"basePlanId"`
	Phases     []rawOfferPhase `json:"phases" yaml:"phases"`
}

type rawOfferPhase struct {
	Type             string   `json:"type" yaml:"type"`
	Duration         string   `json:"duration" yaml:"duration"`
	RecurrenceCount  int64    `json:"recurrenceCount" yaml:"recurrenceCount"`
	RelativeDiscount *float64 `json:"relativeDiscount" yaml:"relativeDiscount"`
}

func loadManifest(path string) (manifest, error) {
	var raw rawManifest
	if err := shared.LoadManifest(path, &raw); err != nil {
		return manifest{}, err
	}
	return normalizeManifest(raw)
}

func LoadManifest(path string) (Manifest, error) {
	return loadManifest(path)
}

func normalizeManifest(raw rawManifest) (manifest, error) {
	raw.Subscription.ProductID = strings.TrimSpace(raw.Subscription.ProductID)
	if raw.Subscription.ProductID == "" {
		return manifest{}, shared.UsageErrorf("subscription.productId is required")
	}
	if len(raw.Subscription.Listings) == 0 {
		return manifest{}, shared.UsageErrorf("subscription.listings is required")
	}
	if len(raw.Subscription.BasePlans) == 0 {
		return manifest{}, shared.UsageErrorf("subscription.basePlans is required")
	}

	listings := make([]*androidpublisher.SubscriptionListing, 0, len(raw.Subscription.Listings))
	for locale, listing := range raw.Subscription.Listings {
		locale = strings.TrimSpace(locale)
		listing.Title = strings.TrimSpace(listing.Title)
		listing.Description = strings.TrimSpace(listing.Description)
		if locale == "" || listing.Title == "" || listing.Description == "" {
			return manifest{}, shared.UsageErrorf("subscription.listings entries require non-empty locale, title, and description")
		}
		listings = append(listings, &androidpublisher.SubscriptionListing{
			LanguageCode: locale,
			Title:        listing.Title,
			Description:  listing.Description,
		})
	}
	sort.Slice(listings, func(i, j int) bool { return listings[i].LanguageCode < listings[j].LanguageCode })

	basePlans := make([]*androidpublisher.BasePlan, 0, len(raw.Subscription.BasePlans))
	basePlanRegions := make(map[string][]rawRegionalConfig, len(raw.Subscription.BasePlans))
	basePlanIDs := make(map[string]struct{}, len(raw.Subscription.BasePlans))
	for _, plan := range raw.Subscription.BasePlans {
		normalizedPlan, regions, err := normalizeBasePlan(plan)
		if err != nil {
			return manifest{}, err
		}
		if _, exists := basePlanIDs[normalizedPlan.BasePlanId]; exists {
			return manifest{}, shared.UsageErrorf("duplicate base plan id %q", normalizedPlan.BasePlanId)
		}
		basePlanIDs[normalizedPlan.BasePlanId] = struct{}{}
		basePlanRegions[normalizedPlan.BasePlanId] = regions
		basePlans = append(basePlans, normalizedPlan)
	}
	sort.Slice(basePlans, func(i, j int) bool { return basePlans[i].BasePlanId < basePlans[j].BasePlanId })

	offers := make([]*androidpublisher.SubscriptionOffer, 0, len(raw.Subscription.Offers))
	offerIDs := make(map[string]struct{}, len(raw.Subscription.Offers))
	for _, offer := range raw.Subscription.Offers {
		normalizedOffer, err := normalizeOffer(raw.Subscription.ProductID, offer, basePlanRegions)
		if err != nil {
			return manifest{}, err
		}
		if _, exists := offerIDs[normalizedOffer.OfferId]; exists {
			return manifest{}, shared.UsageErrorf("duplicate offer id %q", normalizedOffer.OfferId)
		}
		offerIDs[normalizedOffer.OfferId] = struct{}{}
		offers = append(offers, normalizedOffer)
	}
	sort.Slice(offers, func(i, j int) bool { return offers[i].OfferId < offers[j].OfferId })

	return manifest{
		Subscription: &androidpublisher.Subscription{
			ProductId: raw.Subscription.ProductID,
			Listings:  listings,
			BasePlans: basePlans,
		},
		Offers: offers,
	}, nil
}

func normalizeBasePlan(plan rawBasePlan) (*androidpublisher.BasePlan, []rawRegionalConfig, error) {
	plan.BasePlanID = strings.TrimSpace(plan.BasePlanID)
	plan.BillingPeriod = strings.TrimSpace(plan.BillingPeriod)
	if plan.BasePlanID == "" {
		return nil, nil, shared.UsageErrorf("subscription.basePlans[].basePlanId is required")
	}
	if plan.BillingPeriod == "" {
		return nil, nil, shared.UsageErrorf("subscription.basePlans[%q].billingPeriod is required", plan.BasePlanID)
	}
	if len(plan.RegionalConfigs) == 0 {
		return nil, nil, shared.UsageErrorf("subscription.basePlans[%q].regionalConfigs is required", plan.BasePlanID)
	}

	regions := make([]rawRegionalConfig, 0, len(plan.RegionalConfigs))
	regionalConfigs := make([]*androidpublisher.RegionalBasePlanConfig, 0, len(plan.RegionalConfigs))
	seenRegions := make(map[string]struct{}, len(plan.RegionalConfigs))
	for _, region := range plan.RegionalConfigs {
		normalizedRegion, cfg, err := normalizeBasePlanRegion(plan.BasePlanID, region)
		if err != nil {
			return nil, nil, err
		}
		if _, exists := seenRegions[normalizedRegion.RegionCode]; exists {
			return nil, nil, shared.UsageErrorf("subscription.basePlans[%q] has duplicate region %q", plan.BasePlanID, normalizedRegion.RegionCode)
		}
		seenRegions[normalizedRegion.RegionCode] = struct{}{}
		regions = append(regions, normalizedRegion)
		regionalConfigs = append(regionalConfigs, cfg)
	}
	sort.Slice(regions, func(i, j int) bool { return regions[i].RegionCode < regions[j].RegionCode })
	sort.Slice(regionalConfigs, func(i, j int) bool { return regionalConfigs[i].RegionCode < regionalConfigs[j].RegionCode })

	return &androidpublisher.BasePlan{
		BasePlanId: plan.BasePlanID,
		AutoRenewingBasePlanType: &androidpublisher.AutoRenewingBasePlanType{
			BillingPeriodDuration: plan.BillingPeriod,
		},
		RegionalConfigs: regionalConfigs,
	}, regions, nil
}

func normalizeBasePlanRegion(basePlanID string, region rawRegionalConfig) (rawRegionalConfig, *androidpublisher.RegionalBasePlanConfig, error) {
	region.RegionCode = strings.TrimSpace(region.RegionCode)
	region.Price.CurrencyCode = strings.TrimSpace(region.Price.CurrencyCode)
	if region.RegionCode == "" {
		return rawRegionalConfig{}, nil, shared.UsageErrorf("subscription.basePlans[%q].regionalConfigs[].regionCode is required", basePlanID)
	}
	if regionCodePattern.MatchString(region.RegionCode) {
		// Valid ISO alpha-2 codes like NO must remain accepted even if they overlap
		// with YAML 1.1 boolean spellings when unquoted in source manifests.
	} else if isYAMLBooleanLike(region.RegionCode) {
		return rawRegionalConfig{}, nil, shared.UsageErrorf(
			"subscription.basePlans[%q].regionalConfigs[%q].regionCode looks like a YAML boolean; quote region codes like %q to prevent coercion",
			basePlanID,
			region.RegionCode,
			strings.ToUpper(region.RegionCode),
		)
	} else {
		return rawRegionalConfig{}, nil, shared.UsageErrorf(
			"subscription.basePlans[%q].regionalConfigs[%q].regionCode must be a 2-letter uppercase ISO 3166-1 alpha-2 code",
			basePlanID,
			region.RegionCode,
		)
	}
	if region.Price.CurrencyCode == "" {
		return rawRegionalConfig{}, nil, shared.UsageErrorf("subscription.basePlans[%q].regionalConfigs[%q].price.currencyCode is required", basePlanID, region.RegionCode)
	}
	available := true
	if region.NewSubscriberAvailability != nil {
		available = *region.NewSubscriberAvailability
	}
	return region, &androidpublisher.RegionalBasePlanConfig{
		RegionCode:                region.RegionCode,
		NewSubscriberAvailability: available,
		Price: &androidpublisher.Money{
			CurrencyCode: region.Price.CurrencyCode,
			Units:        region.Price.Units,
			Nanos:        region.Price.Nanos,
		},
	}, nil
}

func isYAMLBooleanLike(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes", "n", "no", "on", "off", "true", "false":
		return true
	default:
		return false
	}
}

func normalizeOffer(productID string, offer rawSubscriptionOfferDef, basePlanRegions map[string][]rawRegionalConfig) (*androidpublisher.SubscriptionOffer, error) {
	offer.OfferID = strings.TrimSpace(offer.OfferID)
	offer.BasePlanID = strings.TrimSpace(offer.BasePlanID)
	if offer.OfferID == "" {
		return nil, shared.UsageErrorf("subscription.offers[].offerId is required")
	}
	if offer.BasePlanID == "" {
		return nil, shared.UsageErrorf("subscription.offers[%q].basePlanId is required", offer.OfferID)
	}
	regions, ok := basePlanRegions[offer.BasePlanID]
	if !ok {
		return nil, shared.UsageErrorf("subscription.offers[%q] references unknown base plan %q", offer.OfferID, offer.BasePlanID)
	}
	if len(offer.Phases) == 0 || len(offer.Phases) > 2 {
		return nil, shared.UsageErrorf("subscription.offers[%q].phases must contain 1 or 2 entries", offer.OfferID)
	}

	phases := make([]*androidpublisher.SubscriptionOfferPhase, 0, len(offer.Phases))
	for _, phase := range offer.Phases {
		normalizedPhase, err := normalizeOfferPhase(offer.OfferID, phase, regions)
		if err != nil {
			return nil, err
		}
		phases = append(phases, normalizedPhase)
	}

	regionalConfigs := make([]*androidpublisher.RegionalSubscriptionOfferConfig, 0, len(regions))
	for _, region := range regions {
		available := true
		if region.NewSubscriberAvailability != nil {
			available = *region.NewSubscriberAvailability
		}
		regionalConfigs = append(regionalConfigs, &androidpublisher.RegionalSubscriptionOfferConfig{
			RegionCode:                region.RegionCode,
			NewSubscriberAvailability: available,
		})
	}
	sort.Slice(regionalConfigs, func(i, j int) bool { return regionalConfigs[i].RegionCode < regionalConfigs[j].RegionCode })

	return &androidpublisher.SubscriptionOffer{
		PackageName:     "",
		ProductId:       productID,
		BasePlanId:      offer.BasePlanID,
		OfferId:         offer.OfferID,
		Phases:          phases,
		RegionalConfigs: regionalConfigs,
	}, nil
}

func normalizeOfferPhase(offerID string, phase rawOfferPhase, regions []rawRegionalConfig) (*androidpublisher.SubscriptionOfferPhase, error) {
	phase.Type = strings.ToUpper(strings.TrimSpace(phase.Type))
	phase.Duration = strings.TrimSpace(phase.Duration)
	if phase.Type == "" {
		return nil, shared.UsageErrorf("subscription.offers[%q].phases[].type is required", offerID)
	}
	if phase.Duration == "" {
		return nil, shared.UsageErrorf("subscription.offers[%q].phases[%q].duration is required", offerID, phase.Type)
	}
	if phase.RecurrenceCount < 0 {
		return nil, shared.UsageErrorf("subscription.offers[%q].phases[%q].recurrenceCount must be greater than or equal to zero", offerID, phase.Type)
	}
	recurrenceCount := phase.RecurrenceCount
	if recurrenceCount == 0 {
		recurrenceCount = 1
	}

	regionalConfigs := make([]*androidpublisher.RegionalSubscriptionOfferPhaseConfig, 0, len(regions))
	switch phase.Type {
	case phaseTypeFreeTrial:
		if phase.RelativeDiscount != nil {
			return nil, shared.UsageErrorf("subscription.offers[%q].phases[%q] does not support relativeDiscount", offerID, phase.Type)
		}
		for _, region := range regions {
			regionalConfigs = append(regionalConfigs, &androidpublisher.RegionalSubscriptionOfferPhaseConfig{
				RegionCode: region.RegionCode,
				Free:       &androidpublisher.RegionalSubscriptionOfferPhaseFreePriceOverride{},
			})
		}
	case phaseTypeDiscounted:
		if phase.RelativeDiscount == nil {
			return nil, shared.UsageErrorf("subscription.offers[%q].phases[%q].relativeDiscount is required", offerID, phase.Type)
		}
		if *phase.RelativeDiscount <= 0 || *phase.RelativeDiscount >= 1 {
			return nil, shared.UsageErrorf("subscription.offers[%q].phases[%q].relativeDiscount must be within (0,1)", offerID, phase.Type)
		}
		for _, region := range regions {
			regionalConfigs = append(regionalConfigs, &androidpublisher.RegionalSubscriptionOfferPhaseConfig{
				RegionCode:       region.RegionCode,
				RelativeDiscount: *phase.RelativeDiscount,
			})
		}
	default:
		return nil, shared.UsageErrorf("subscription.offers[%q].phases[].type must be one of: %s, %s", offerID, phaseTypeFreeTrial, phaseTypeDiscounted)
	}

	slices.SortFunc(regionalConfigs, func(a, b *androidpublisher.RegionalSubscriptionOfferPhaseConfig) int {
		return strings.Compare(a.RegionCode, b.RegionCode)
	})

	return &androidpublisher.SubscriptionOfferPhase{
		Duration:        phase.Duration,
		RecurrenceCount: recurrenceCount,
		RegionalConfigs: regionalConfigs,
	}, nil
}

func (m manifest) plannedActions(activate bool) []string {
	if m.Subscription == nil {
		return nil
	}
	actions := []string{fmt.Sprintf("create subscription %s", m.Subscription.ProductId)}
	for _, basePlan := range m.Subscription.BasePlans {
		if activate {
			actions = append(actions, fmt.Sprintf("activate base plan %s", basePlan.BasePlanId))
		}
	}
	for _, offer := range m.Offers {
		actions = append(actions, fmt.Sprintf("create offer %s on %s", offer.OfferId, offer.BasePlanId))
		if activate {
			actions = append(actions, fmt.Sprintf("activate offer %s on %s", offer.OfferId, offer.BasePlanId))
		}
	}
	return actions
}
