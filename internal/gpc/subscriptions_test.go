package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestSubscriptionMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.ListSubscriptions(context.Background(), "com.example.app", 50, "", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListSubscriptions, got %v", err)
	}
	if _, err := c.GetSubscription(context.Background(), "com.example.app", "premium_monthly"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetSubscription, got %v", err)
	}
	if _, err := c.BatchGetSubscriptions(context.Background(), "com.example.app", []string{"premium_monthly"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchGetSubscriptions, got %v", err)
	}
	if _, err := c.BatchUpdateSubscriptions(context.Background(), "com.example.app", []*androidpublisher.UpdateSubscriptionRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchUpdateSubscriptions, got %v", err)
	}
	if _, err := c.CreateSubscription(context.Background(), "com.example.app", &androidpublisher.Subscription{ProductId: "premium_monthly"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateSubscription, got %v", err)
	}
	if _, err := c.UpdateSubscription(context.Background(), "com.example.app", "premium_monthly", &androidpublisher.Subscription{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UpdateSubscription, got %v", err)
	}
	if err := c.DeleteSubscription(context.Background(), "com.example.app", "premium_monthly"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteSubscription, got %v", err)
	}
	if err := c.ArchiveSubscription(context.Background(), "com.example.app", "premium_monthly"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ArchiveSubscription, got %v", err)
	}
	if _, err := c.ActivateSubscriptionBasePlan(context.Background(), "com.example.app", "premium_monthly", "monthly"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ActivateSubscriptionBasePlan, got %v", err)
	}
	if _, err := c.DeactivateSubscriptionBasePlan(context.Background(), "com.example.app", "premium_monthly", "monthly"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeactivateSubscriptionBasePlan, got %v", err)
	}
	if _, err := c.BatchUpdateSubscriptionBasePlanStates(context.Background(), "com.example.app", "premium_monthly", []*androidpublisher.UpdateBasePlanStateRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchUpdateSubscriptionBasePlanStates, got %v", err)
	}
	if err := c.DeleteSubscriptionBasePlan(context.Background(), "com.example.app", "premium_monthly", "monthly"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteSubscriptionBasePlan, got %v", err)
	}
	if err := c.MigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", "monthly", &androidpublisher.MigrateBasePlanPricesRequest{
		RegionalPriceMigrations: []*androidpublisher.RegionalPriceMigrationConfig{{RegionCode: "US", OldestAllowedPriceVersionTime: "2025-01-01T00:00:00Z"}},
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from MigrateSubscriptionBasePlanPrices, got %v", err)
	}
	if _, err := c.BatchMigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", []*androidpublisher.MigrateBasePlanPricesRequest{
		{
			BasePlanId: "monthly",
			RegionalPriceMigrations: []*androidpublisher.RegionalPriceMigrationConfig{
				{RegionCode: "US", OldestAllowedPriceVersionTime: "2025-01-01T00:00:00Z"},
			},
		},
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchMigrateSubscriptionBasePlanPrices, got %v", err)
	}
	if _, err := c.ListSubscriptionOffers(context.Background(), "com.example.app", "premium_monthly", "monthly", 50, "", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListSubscriptionOffers, got %v", err)
	}
	if _, err := c.GetSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", "offer1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetSubscriptionOffer, got %v", err)
	}
	if _, err := c.BatchGetSubscriptionOffers(context.Background(), "com.example.app", "premium_monthly", "monthly", []string{"offer1"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchGetSubscriptionOffers, got %v", err)
	}
	if _, err := c.BatchUpdateSubscriptionOffers(context.Background(), "com.example.app", "premium_monthly", "monthly", []*androidpublisher.UpdateSubscriptionOfferRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchUpdateSubscriptionOffers, got %v", err)
	}
	if _, err := c.BatchUpdateSubscriptionOfferStates(context.Background(), "com.example.app", "premium_monthly", "monthly", []*androidpublisher.UpdateSubscriptionOfferStateRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchUpdateSubscriptionOfferStates, got %v", err)
	}
	if _, err := c.CreateSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", &androidpublisher.SubscriptionOffer{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateSubscriptionOffer, got %v", err)
	}
	if _, err := c.UpdateSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", "offer1", &androidpublisher.SubscriptionOffer{}, "phases"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UpdateSubscriptionOffer, got %v", err)
	}
	if _, err := c.ActivateSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", "offer1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ActivateSubscriptionOffer, got %v", err)
	}
	if _, err := c.DeactivateSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", "offer1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeactivateSubscriptionOffer, got %v", err)
	}
	if err := c.DeleteSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", "offer1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteSubscriptionOffer, got %v", err)
	}
}

func TestSubscriptionMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListSubscriptions(context.Background(), "", 50, "", false); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListSubscriptions package error: %v", err)
	}
	if _, err := c.ListSubscriptions(context.Background(), "com.example.app", -1, "", false); err == nil || !strings.Contains(err.Error(), "page size must be greater than or equal to zero") {
		t.Fatalf("unexpected ListSubscriptions page size error: %v", err)
	}
	if _, err := c.GetSubscription(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected GetSubscription product id error: %v", err)
	}
	if _, err := c.BatchGetSubscriptions(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "at least one product id is required") {
		t.Fatalf("unexpected BatchGetSubscriptions empty IDs error: %v", err)
	}
	tooManyProductIDs := make([]string, 101)
	for i := range tooManyProductIDs {
		tooManyProductIDs[i] = "premium"
	}
	if _, err := c.BatchGetSubscriptions(context.Background(), "com.example.app", tooManyProductIDs); err == nil || !strings.Contains(err.Error(), "product id count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchGetSubscriptions count error: %v", err)
	}
	if _, err := c.BatchUpdateSubscriptions(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "at least one batch update request is required") {
		t.Fatalf("unexpected BatchUpdateSubscriptions empty request error: %v", err)
	}
	tooManyBatchUpdateRequests := make([]*androidpublisher.UpdateSubscriptionRequest, 101)
	for i := range tooManyBatchUpdateRequests {
		tooManyBatchUpdateRequests[i] = &androidpublisher.UpdateSubscriptionRequest{}
	}
	if _, err := c.BatchUpdateSubscriptions(context.Background(), "com.example.app", tooManyBatchUpdateRequests); err == nil || !strings.Contains(err.Error(), "batch update request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchUpdateSubscriptions count error: %v", err)
	}
	if _, err := c.CreateSubscription(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "subscription payload is required") {
		t.Fatalf("unexpected CreateSubscription payload error: %v", err)
	}
	if _, err := c.UpdateSubscription(context.Background(), "com.example.app", "", &androidpublisher.Subscription{}); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected UpdateSubscription product id error: %v", err)
	}
	if _, err := c.UpdateSubscription(context.Background(), "com.example.app", "premium_monthly", nil); err == nil || !strings.Contains(err.Error(), "subscription payload is required") {
		t.Fatalf("unexpected UpdateSubscription payload error: %v", err)
	}
	if err := c.DeleteSubscription(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected DeleteSubscription product id error: %v", err)
	}
	if err := c.ArchiveSubscription(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected ArchiveSubscription product id error: %v", err)
	}
	if _, err := c.ActivateSubscriptionBasePlan(context.Background(), "com.example.app", "premium_monthly", ""); err == nil || !strings.Contains(err.Error(), "base plan id is required") {
		t.Fatalf("unexpected ActivateSubscriptionBasePlan base plan id error: %v", err)
	}
	if _, err := c.DeactivateSubscriptionBasePlan(context.Background(), "com.example.app", "", "monthly"); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected DeactivateSubscriptionBasePlan product id error: %v", err)
	}
	if _, err := c.BatchUpdateSubscriptionBasePlanStates(context.Background(), "com.example.app", "premium_monthly", nil); err == nil || !strings.Contains(err.Error(), "at least one base plan state update request is required") {
		t.Fatalf("unexpected BatchUpdateSubscriptionBasePlanStates empty request error: %v", err)
	}
	tooManyBasePlanStateRequests := make([]*androidpublisher.UpdateBasePlanStateRequest, 101)
	for i := range tooManyBasePlanStateRequests {
		tooManyBasePlanStateRequests[i] = &androidpublisher.UpdateBasePlanStateRequest{}
	}
	if _, err := c.BatchUpdateSubscriptionBasePlanStates(context.Background(), "com.example.app", "premium_monthly", tooManyBasePlanStateRequests); err == nil || !strings.Contains(err.Error(), "base plan state update request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchUpdateSubscriptionBasePlanStates count error: %v", err)
	}
	if err := c.DeleteSubscriptionBasePlan(context.Background(), "com.example.app", "premium_monthly", ""); err == nil || !strings.Contains(err.Error(), "base plan id is required") {
		t.Fatalf("unexpected DeleteSubscriptionBasePlan base plan id error: %v", err)
	}
	if err := c.MigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", "", &androidpublisher.MigrateBasePlanPricesRequest{}); err == nil || !strings.Contains(err.Error(), "base plan id is required") {
		t.Fatalf("unexpected MigrateSubscriptionBasePlanPrices base plan id error: %v", err)
	}
	if err := c.MigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", "monthly", nil); err == nil || !strings.Contains(err.Error(), "migrate prices payload is required") {
		t.Fatalf("unexpected MigrateSubscriptionBasePlanPrices payload error: %v", err)
	}
	if err := c.MigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", "monthly", &androidpublisher.MigrateBasePlanPricesRequest{}); err == nil || !strings.Contains(err.Error(), "at least one regional price migration") {
		t.Fatalf("unexpected MigrateSubscriptionBasePlanPrices regional migration error: %v", err)
	}
	if _, err := c.BatchMigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", nil); err == nil || !strings.Contains(err.Error(), "at least one base plan migration request is required") {
		t.Fatalf("unexpected BatchMigrateSubscriptionBasePlanPrices empty request error: %v", err)
	}
	tooManyMigrateRequests := make([]*androidpublisher.MigrateBasePlanPricesRequest, 101)
	for i := range tooManyMigrateRequests {
		tooManyMigrateRequests[i] = &androidpublisher.MigrateBasePlanPricesRequest{
			BasePlanId: "monthly",
			RegionalPriceMigrations: []*androidpublisher.RegionalPriceMigrationConfig{
				{RegionCode: "US", OldestAllowedPriceVersionTime: "2025-01-01T00:00:00Z"},
			},
		}
	}
	if _, err := c.BatchMigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", tooManyMigrateRequests); err == nil || !strings.Contains(err.Error(), "base plan migration request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchMigrateSubscriptionBasePlanPrices count error: %v", err)
	}
	if _, err := c.BatchMigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", []*androidpublisher.MigrateBasePlanPricesRequest{
		{RegionalPriceMigrations: []*androidpublisher.RegionalPriceMigrationConfig{{RegionCode: "US", OldestAllowedPriceVersionTime: "2025-01-01T00:00:00Z"}}},
	}); err == nil || !strings.Contains(err.Error(), "base plan id is required in every migration request") {
		t.Fatalf("unexpected BatchMigrateSubscriptionBasePlanPrices base plan id error: %v", err)
	}
	if _, err := c.BatchMigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", []*androidpublisher.MigrateBasePlanPricesRequest{
		{BasePlanId: "monthly"},
	}); err == nil || !strings.Contains(err.Error(), "at least one regional price migration") {
		t.Fatalf("unexpected BatchMigrateSubscriptionBasePlanPrices regional migration error: %v", err)
	}
	if _, err := c.BatchMigrateSubscriptionBasePlanPrices(context.Background(), "com.example.app", "premium_monthly", []*androidpublisher.MigrateBasePlanPricesRequest{
		{BasePlanId: "monthly", RegionalPriceMigrations: []*androidpublisher.RegionalPriceMigrationConfig{{RegionCode: "US", OldestAllowedPriceVersionTime: "2025-01-01T00:00:00Z"}}},
		{BasePlanId: "monthly", RegionalPriceMigrations: []*androidpublisher.RegionalPriceMigrationConfig{{RegionCode: "PL", OldestAllowedPriceVersionTime: "2025-01-01T00:00:00Z"}}},
	}); err == nil || !strings.Contains(err.Error(), "duplicate base plan id") {
		t.Fatalf("unexpected BatchMigrateSubscriptionBasePlanPrices duplicate base plan id error: %v", err)
	}
	if _, err := c.ListSubscriptionOffers(context.Background(), "com.example.app", "", "monthly", 50, "", false); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected ListSubscriptionOffers product id error: %v", err)
	}
	if _, err := c.GetSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", ""); err == nil || !strings.Contains(err.Error(), "offer id is required") {
		t.Fatalf("unexpected GetSubscriptionOffer offer id error: %v", err)
	}
	if _, err := c.BatchGetSubscriptionOffers(context.Background(), "com.example.app", "premium_monthly", "monthly", nil); err == nil || !strings.Contains(err.Error(), "at least one offer id is required") {
		t.Fatalf("unexpected BatchGetSubscriptionOffers empty IDs error: %v", err)
	}
	tooManyIDs := make([]string, 101)
	for i := range tooManyIDs {
		tooManyIDs[i] = "offer"
	}
	if _, err := c.BatchGetSubscriptionOffers(context.Background(), "com.example.app", "premium_monthly", "monthly", tooManyIDs); err == nil || !strings.Contains(err.Error(), "offer id count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchGetSubscriptionOffers count error: %v", err)
	}
	if _, err := c.BatchUpdateSubscriptionOffers(context.Background(), "com.example.app", "premium_monthly", "monthly", nil); err == nil || !strings.Contains(err.Error(), "at least one batch update request is required") {
		t.Fatalf("unexpected BatchUpdateSubscriptionOffers empty request error: %v", err)
	}
	tooManyOfferUpdateRequests := make([]*androidpublisher.UpdateSubscriptionOfferRequest, 101)
	for i := range tooManyOfferUpdateRequests {
		tooManyOfferUpdateRequests[i] = &androidpublisher.UpdateSubscriptionOfferRequest{}
	}
	if _, err := c.BatchUpdateSubscriptionOffers(context.Background(), "com.example.app", "premium_monthly", "monthly", tooManyOfferUpdateRequests); err == nil || !strings.Contains(err.Error(), "batch update request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchUpdateSubscriptionOffers count error: %v", err)
	}
	if _, err := c.BatchUpdateSubscriptionOfferStates(context.Background(), "com.example.app", "premium_monthly", "monthly", nil); err == nil || !strings.Contains(err.Error(), "at least one batch state update request is required") {
		t.Fatalf("unexpected BatchUpdateSubscriptionOfferStates empty request error: %v", err)
	}
	tooManyOfferStateRequests := make([]*androidpublisher.UpdateSubscriptionOfferStateRequest, 101)
	for i := range tooManyOfferStateRequests {
		tooManyOfferStateRequests[i] = &androidpublisher.UpdateSubscriptionOfferStateRequest{}
	}
	if _, err := c.BatchUpdateSubscriptionOfferStates(context.Background(), "com.example.app", "premium_monthly", "monthly", tooManyOfferStateRequests); err == nil || !strings.Contains(err.Error(), "batch state update request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchUpdateSubscriptionOfferStates count error: %v", err)
	}
	if _, err := c.CreateSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", nil); err == nil || !strings.Contains(err.Error(), "subscription offer payload is required") {
		t.Fatalf("unexpected CreateSubscriptionOffer payload error: %v", err)
	}
	if _, err := c.UpdateSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", "offer1", &androidpublisher.SubscriptionOffer{}, ""); err == nil || !strings.Contains(err.Error(), "update mask is required") {
		t.Fatalf("unexpected UpdateSubscriptionOffer update mask error: %v", err)
	}
	if _, err := c.ActivateSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "", "offer1"); err == nil || !strings.Contains(err.Error(), "base plan id is required") {
		t.Fatalf("unexpected ActivateSubscriptionOffer base plan id error: %v", err)
	}
	if _, err := c.DeactivateSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", ""); err == nil || !strings.Contains(err.Error(), "offer id is required") {
		t.Fatalf("unexpected DeactivateSubscriptionOffer offer id error: %v", err)
	}
	if err := c.DeleteSubscriptionOffer(context.Background(), "com.example.app", "premium_monthly", "monthly", ""); err == nil || !strings.Contains(err.Error(), "offer id is required") {
		t.Fatalf("unexpected DeleteSubscriptionOffer offer id error: %v", err)
	}
}

func TestSubscriptionsListInfoFromResponse(t *testing.T) {
	got := subscriptionsListInfoFromResponse(&androidpublisher.ListSubscriptionsResponse{
		NextPageToken: "next-token",
		Subscriptions: []*androidpublisher.Subscription{
			{
				PackageName: "com.example.app",
				ProductId:   "premium_monthly",
				BasePlans:   []*androidpublisher.BasePlan{{BasePlanId: "monthly"}},
				Listings:    []*androidpublisher.SubscriptionListing{{LanguageCode: "en-US"}},
			},
		},
	})
	if got.NextPageToken != "next-token" || len(got.Subscriptions) != 1 || got.Subscriptions[0].ProductID != "premium_monthly" {
		t.Fatalf("unexpected list map: %+v", got)
	}
}

func TestSubscriptionInfoFromSubscription(t *testing.T) {
	got := subscriptionInfoFromSubscription(&androidpublisher.Subscription{
		PackageName: "com.example.app",
		ProductId:   "premium_yearly",
		Archived:    true,
		BasePlans: []*androidpublisher.BasePlan{
			{BasePlanId: "yearly"},
			{BasePlanId: "yearly_intro"},
		},
		Listings: []*androidpublisher.SubscriptionListing{
			{LanguageCode: "en-US"},
			{LanguageCode: "pl-PL"},
		},
	})
	if got.PackageName != "com.example.app" || got.ProductID != "premium_yearly" || !got.Archived || got.BasePlanCount != 2 || got.ListingCount != 2 {
		t.Fatalf("unexpected subscription map: %+v", got)
	}
}

func TestSubscriptionDiagnosticInfoFromSubscription(t *testing.T) {
	got := subscriptionDiagnosticInfoFromSubscription(&androidpublisher.Subscription{
		PackageName: "com.example.app",
		ProductId:   "premium_yearly",
		Archived:    true,
		BasePlans: []*androidpublisher.BasePlan{
			{
				BasePlanId: "yearly",
				State:      "ACTIVE",
				OfferTags:  []*androidpublisher.OfferTag{{Tag: "tag1"}},
				RegionalConfigs: []*androidpublisher.RegionalBasePlanConfig{
					{RegionCode: "US", NewSubscriberAvailability: true},
					{RegionCode: "PL", NewSubscriberAvailability: false},
				},
			},
			{
				BasePlanId: "legacy",
				State:      "INACTIVE",
				RegionalConfigs: []*androidpublisher.RegionalBasePlanConfig{
					{RegionCode: "DE", NewSubscriberAvailability: true},
				},
			},
		},
		Listings: []*androidpublisher.SubscriptionListing{
			{LanguageCode: "en-US"},
			{LanguageCode: "pl-PL"},
		},
	})

	if got.ProductID != "premium_yearly" || got.BasePlanCount != 2 || got.ActiveBasePlanCount != 1 || got.ListingCount != 2 || got.RegionCount != 3 || got.AvailableRegionCount != 2 {
		t.Fatalf("unexpected subscription diagnostic map: %+v", got)
	}
	if len(got.BasePlans) != 2 || got.BasePlans[0].BasePlanID != "yearly" {
		t.Fatalf("unexpected base plan diagnostic map: %+v", got.BasePlans)
	}
}

func TestSubscriptionOffersListInfoFromResponse(t *testing.T) {
	got := subscriptionOffersListInfoFromResponse(&androidpublisher.ListSubscriptionOffersResponse{
		NextPageToken: "next-token",
		SubscriptionOffers: []*androidpublisher.SubscriptionOffer{
			{
				PackageName: "com.example.app",
				ProductId:   "premium_monthly",
				BasePlanId:  "monthly",
				OfferId:     "offer1",
				State:       "ACTIVE",
				Phases:      []*androidpublisher.SubscriptionOfferPhase{{Duration: "P1M"}},
				OfferTags:   []*androidpublisher.OfferTag{{Tag: "new-user"}},
			},
		},
	})
	if got.NextPageToken != "next-token" || len(got.Offers) != 1 || got.Offers[0].OfferID != "offer1" {
		t.Fatalf("unexpected offers list map: %+v", got)
	}
}

func TestSubscriptionOfferInfoFromOffer(t *testing.T) {
	got := subscriptionOfferInfoFromOffer(&androidpublisher.SubscriptionOffer{
		PackageName: "com.example.app",
		ProductId:   "premium_monthly",
		BasePlanId:  "monthly",
		OfferId:     "offer1",
		State:       "ACTIVE",
		Phases: []*androidpublisher.SubscriptionOfferPhase{
			{Duration: "P1M"},
			{Duration: "P1M"},
		},
		OfferTags: []*androidpublisher.OfferTag{
			{Tag: "new-user"},
		},
	})
	if got.PackageName != "com.example.app" || got.OfferID != "offer1" || got.PhaseCount != 2 || got.TagCount != 1 {
		t.Fatalf("unexpected offer map: %+v", got)
	}
}

func TestSubscriptionInfosFromSlice(t *testing.T) {
	got := subscriptionInfosFromSlice([]*androidpublisher.Subscription{
		{
			PackageName: "com.example.app",
			ProductId:   "premium_monthly",
			BasePlans:   []*androidpublisher.BasePlan{{BasePlanId: "monthly"}},
		},
		nil,
	})
	if len(got) != 2 {
		t.Fatalf("unexpected mapped count: %+v", got)
	}
	if got[0].ProductID != "premium_monthly" || got[0].BasePlanCount != 1 {
		t.Fatalf("unexpected first mapped subscription: %+v", got[0])
	}
	if got[1] != (SubscriptionInfo{}) {
		t.Fatalf("unexpected nil mapping result: %+v", got[1])
	}
}
