package monetization

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"google.golang.org/api/androidpublisher/v3"
)

func TestRegionsCommandReturnsSortedRegions(t *testing.T) {
	client := &fakeClient{
		monetizationRegions: gpc.MonetizationRegionsInfo{
			RegionsVersion: "2025/03",
			Regions: []gpc.MonetizationRegionInfo{
				{RegionCode: "US", CurrencyCode: "USD"},
				{RegionCode: "PL", CurrencyCode: "PLN"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "regions", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"regionsVersion":"2025/03"`, `"count":2`, `"regionCode":"PL"`, `"currencyCode":"PLN"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got %s", want, out)
		}
	}
}

func TestSyncDryRunForNewSubscriptionReportsCreateActions(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--manifest", writeSetupManifest(t), "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) || !strings.Contains(out, `create subscription premium`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSyncDryRunForExistingReportsUpdatesAndDrift(t *testing.T) {
	client := &fakeClient{
		listSubscriptionsResult: gpc.SubscriptionsListInfo{
			Subscriptions: []gpc.SubscriptionInfo{{ProductID: "premium"}},
		},
		getSubscriptionRaw: &androidpublisher.Subscription{
			PackageName: "com.example.app",
			ProductId:   "premium",
			Listings: []*androidpublisher.SubscriptionListing{
				{LanguageCode: "en-US", Title: "Old Premium", Description: "Old description"},
				{LanguageCode: "pl-PL", Title: "Premium PL", Description: "Opis"},
			},
			BasePlans: []*androidpublisher.BasePlan{
				{
					BasePlanId: "monthly",
					State:      "INACTIVE",
					AutoRenewingBasePlanType: &androidpublisher.AutoRenewingBasePlanType{
						BillingPeriodDuration: "P1M",
					},
					RegionalConfigs: []*androidpublisher.RegionalBasePlanConfig{
						{RegionCode: "US", NewSubscriberAvailability: false, Price: &androidpublisher.Money{CurrencyCode: "USD", Units: 8, Nanos: 990000000}},
					},
				},
				{
					BasePlanId: "yearly",
					State:      "ACTIVE",
					AutoRenewingBasePlanType: &androidpublisher.AutoRenewingBasePlanType{
						BillingPeriodDuration: "P1Y",
					},
				},
			},
		},
		listOffersResult: map[string]gpc.SubscriptionOffersListInfo{
			"monthly": {Offers: []gpc.SubscriptionOfferInfo{
				{BasePlanID: "monthly", OfferID: "intro_monthly", State: "INACTIVE"},
				{BasePlanID: "monthly", OfferID: "loyalty", State: "ACTIVE"},
			}},
			"yearly": {Offers: []gpc.SubscriptionOfferInfo{
				{BasePlanID: "yearly", OfferID: "legacy", State: "ACTIVE"},
			}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--manifest", writeSetupManifest(t), "--dry-run", "--activate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		`"status":"dry-run"`,
		`update subscription premium`,
		`update offer intro_monthly on monthly`,
		`activate base plan monthly`,
		`activate offer intro_monthly on monthly`,
		`"unmanagedListings":["pl-PL"]`,
		`"unmanagedBasePlans":["yearly"]`,
		`"unmanagedOffers":["monthly/loyalty","yearly/legacy"]`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got %s", want, out)
		}
	}
}

func TestSyncCommitExistingPreservesUnmanagedResources(t *testing.T) {
	client := &fakeClient{
		listSubscriptionsResult: gpc.SubscriptionsListInfo{
			Subscriptions: []gpc.SubscriptionInfo{{ProductID: "premium"}},
		},
		getSubscriptionRaw: &androidpublisher.Subscription{
			PackageName: "com.example.app",
			ProductId:   "premium",
			Listings: []*androidpublisher.SubscriptionListing{
				{LanguageCode: "en-US", Title: "Old Premium", Description: "Old description"},
				{LanguageCode: "pl-PL", Title: "Premium PL", Description: "Opis"},
			},
			BasePlans: []*androidpublisher.BasePlan{
				{
					BasePlanId: "monthly",
					State:      "INACTIVE",
					AutoRenewingBasePlanType: &androidpublisher.AutoRenewingBasePlanType{
						BillingPeriodDuration: "P1M",
					},
					RegionalConfigs: []*androidpublisher.RegionalBasePlanConfig{
						{RegionCode: "US", NewSubscriberAvailability: false, Price: &androidpublisher.Money{CurrencyCode: "USD", Units: 8, Nanos: 990000000}},
					},
				},
				{
					BasePlanId: "yearly",
					State:      "ACTIVE",
					AutoRenewingBasePlanType: &androidpublisher.AutoRenewingBasePlanType{
						BillingPeriodDuration: "P1Y",
					},
				},
			},
		},
		listOffersResult: map[string]gpc.SubscriptionOffersListInfo{
			"monthly": {Offers: []gpc.SubscriptionOfferInfo{
				{BasePlanID: "monthly", OfferID: "intro_monthly", State: "INACTIVE"},
			}},
			"yearly": {Offers: []gpc.SubscriptionOfferInfo{
				{BasePlanID: "yearly", OfferID: "legacy", State: "ACTIVE"},
			}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--manifest", writeSetupManifest(t), "--confirm", "--activate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"completed"`) || client.updatedSubscription == nil {
		t.Fatalf("unexpected output or missing subscription update: %s %+v", out, client.updatedSubscription)
	}
	if got := len(client.updatedSubscription.Listings); got != 2 {
		t.Fatalf("expected merged listings, got %d", got)
	}
	if got := len(client.updatedSubscription.BasePlans); got != 2 {
		t.Fatalf("expected merged base plans, got %d", got)
	}
	if len(client.updatedOffers) != 1 || client.updatedOffers[0].OfferId != "intro_monthly" {
		t.Fatalf("unexpected updated offers: %+v", client.updatedOffers)
	}
	if len(client.activatedBasePlans) != 1 || client.activatedBasePlans[0] != "monthly" {
		t.Fatalf("unexpected base plan activations: %+v", client.activatedBasePlans)
	}
	if len(client.activatedOffers) != 1 || client.activatedOffers[0] != "intro_monthly" {
		t.Fatalf("unexpected offer activations: %+v", client.activatedOffers)
	}
}

func TestSyncReportsPartialProgressOnOfferUpdateFailure(t *testing.T) {
	client := &fakeClient{
		listSubscriptionsResult: gpc.SubscriptionsListInfo{
			Subscriptions: []gpc.SubscriptionInfo{{ProductID: "premium"}},
		},
		getSubscriptionRaw: &androidpublisher.Subscription{
			PackageName: "com.example.app",
			ProductId:   "premium",
			Listings: []*androidpublisher.SubscriptionListing{
				{LanguageCode: "en-US", Title: "Old Premium", Description: "Old description"},
			},
			BasePlans: []*androidpublisher.BasePlan{
				{
					BasePlanId: "monthly",
					State:      "ACTIVE",
					AutoRenewingBasePlanType: &androidpublisher.AutoRenewingBasePlanType{
						BillingPeriodDuration: "P1M",
					},
					RegionalConfigs: []*androidpublisher.RegionalBasePlanConfig{
						{RegionCode: "US", NewSubscriberAvailability: true, Price: &androidpublisher.Money{CurrencyCode: "USD", Units: 9, Nanos: 990000000}},
					},
				},
			},
		},
		listOffersResult: map[string]gpc.SubscriptionOffersListInfo{
			"monthly": {Offers: []gpc.SubscriptionOfferInfo{
				{BasePlanID: "monthly", OfferID: "intro_monthly", State: "INACTIVE"},
			}},
		},
		updateOfferErr: map[string]error{"intro_monthly": fmt.Errorf("boom")},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--manifest", writeSetupManifest(t), "--confirm")
	if err == nil || !strings.Contains(err.Error(), "failed to update offer") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"updatedOffers"`) && client.updatedSubscription == nil {
		t.Fatalf("expected partial progress to be recorded, got %s", out)
	}
}
