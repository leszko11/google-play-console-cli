package monetization

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifestFixture(t *testing.T, ext, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest"+ext)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func TestLoadManifestYAML(t *testing.T) {
	path := writeManifestFixture(t, ".yaml", `
subscription:
  productId: premium
  listings:
    en-US:
      title: Premium
      description: Unlock all features
  basePlans:
    - basePlanId: monthly
      billingPeriod: P1M
      regionalConfigs:
        - regionCode: US
          price:
            currencyCode: USD
            units: 9
            nanos: 990000000
  offers:
    - offerId: intro_monthly
      basePlanId: monthly
      phases:
        - type: FREE_TRIAL
          duration: P7D
`)

	got, err := loadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Subscription == nil || got.Subscription.ProductId != "premium" {
		t.Fatalf("unexpected subscription: %+v", got.Subscription)
	}
	if len(got.Subscription.Listings) != 1 || got.Subscription.Listings[0].LanguageCode != "en-US" {
		t.Fatalf("unexpected listings: %+v", got.Subscription.Listings)
	}
	if len(got.Subscription.BasePlans) != 1 || got.Subscription.BasePlans[0].BasePlanId != "monthly" {
		t.Fatalf("unexpected base plans: %+v", got.Subscription.BasePlans)
	}
	if got.Subscription.BasePlans[0].AutoRenewingBasePlanType == nil || got.Subscription.BasePlans[0].AutoRenewingBasePlanType.BillingPeriodDuration != "P1M" {
		t.Fatalf("unexpected base plan type: %+v", got.Subscription.BasePlans[0].AutoRenewingBasePlanType)
	}
	if len(got.Offers) != 1 || got.Offers[0].OfferId != "intro_monthly" {
		t.Fatalf("unexpected offers: %+v", got.Offers)
	}
	if len(got.Offers[0].Phases) != 1 || got.Offers[0].Phases[0].RegionalConfigs[0].Free == nil {
		t.Fatalf("unexpected offer phases: %+v", got.Offers[0].Phases)
	}
}

func TestLoadManifestJSONDiscountedOffer(t *testing.T) {
	path := writeManifestFixture(t, ".json", `{
  "subscription": {
    "productId": "premium",
    "listings": {
      "en-US": { "title": "Premium", "description": "Unlock all features" }
    },
    "basePlans": [{
      "basePlanId": "monthly",
      "billingPeriod": "P1M",
      "regionalConfigs": [{
        "regionCode": "US",
        "price": { "currencyCode": "USD", "units": 9, "nanos": 990000000 }
      }]
    }],
    "offers": [{
      "offerId": "intro_discount",
      "basePlanId": "monthly",
      "phases": [{
        "type": "DISCOUNTED",
        "duration": "P1M",
        "relativeDiscount": 0.5
      }]
    }]
  }
}`)

	got, err := loadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Offers) != 1 || len(got.Offers[0].Phases) != 1 {
		t.Fatalf("unexpected offers: %+v", got.Offers)
	}
	if got.Offers[0].Phases[0].RegionalConfigs[0].RelativeDiscount != 0.5 {
		t.Fatalf("unexpected discounted phase: %+v", got.Offers[0].Phases[0].RegionalConfigs)
	}
}

func TestLoadManifestRejectsUnknownBasePlan(t *testing.T) {
	path := writeManifestFixture(t, ".yaml", `
subscription:
  productId: premium
  listings:
    en-US:
      title: Premium
      description: Unlock all features
  basePlans:
    - basePlanId: monthly
      billingPeriod: P1M
      regionalConfigs:
        - regionCode: US
          price:
            currencyCode: USD
            units: 9
            nanos: 990000000
  offers:
    - offerId: intro_monthly
      basePlanId: yearly
      phases:
        - type: FREE_TRIAL
          duration: P7D
`)

	if _, err := loadManifest(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadManifestRejectsInvalidPhase(t *testing.T) {
	path := writeManifestFixture(t, ".yaml", `
subscription:
  productId: premium
  listings:
    en-US:
      title: Premium
      description: Unlock all features
  basePlans:
    - basePlanId: monthly
      billingPeriod: P1M
      regionalConfigs:
        - regionCode: US
          price:
            currencyCode: USD
            units: 9
            nanos: 990000000
  offers:
    - offerId: intro_monthly
      basePlanId: monthly
      phases:
        - type: DISCOUNTED
          duration: P7D
`)

	if _, err := loadManifest(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadManifestRejectsYAMLBooleanLikeRegionCode(t *testing.T) {
	path := writeManifestFixture(t, ".yaml", `
subscription:
  productId: premium
  listings:
    en-US:
      title: Premium
      description: Unlock all features
  basePlans:
    - basePlanId: monthly
      billingPeriod: P1M
      regionalConfigs:
        - regionCode: no
          price:
            currencyCode: USD
            units: 9
            nanos: 990000000
  offers: []
`)

	_, err := loadManifest(path)
	if err == nil || !strings.Contains(err.Error(), `quote region codes like "NO"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadManifestRejectsNonUppercaseRegionCode(t *testing.T) {
	path := writeManifestFixture(t, ".yaml", `
subscription:
  productId: premium
  listings:
    en-US:
      title: Premium
      description: Unlock all features
  basePlans:
    - basePlanId: monthly
      billingPeriod: P1M
      regionalConfigs:
        - regionCode: Usa
          price:
            currencyCode: USD
            units: 9
            nanos: 990000000
  offers: []
`)

	_, err := loadManifest(path)
	if err == nil || !strings.Contains(err.Error(), "2-letter uppercase ISO 3166-1 alpha-2 code") {
		t.Fatalf("unexpected error: %v", err)
	}
}
