package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestOneTimeProductMethods_RejectMissingClient(t *testing.T) {
	var c *Client
	payload := &androidpublisher.OneTimeProduct{ProductId: "coins_100"}

	if _, err := c.ListOneTimeProducts(context.Background(), "com.example.app", 0, "", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListOneTimeProducts, got %v", err)
	}
	if _, err := c.GetOneTimeProduct(context.Background(), "com.example.app", "coins_100"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetOneTimeProduct, got %v", err)
	}
	if _, err := c.BatchGetOneTimeProducts(context.Background(), "com.example.app", []string{"coins_100"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchGetOneTimeProducts, got %v", err)
	}
	if _, err := c.BatchUpdateOneTimeProducts(context.Background(), "com.example.app", []*androidpublisher.UpdateOneTimeProductRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchUpdateOneTimeProducts, got %v", err)
	}
	if err := c.BatchDeleteOneTimeProducts(context.Background(), "com.example.app", []*androidpublisher.DeleteOneTimeProductRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchDeleteOneTimeProducts, got %v", err)
	}
	if _, err := c.CreateOneTimeProduct(context.Background(), "com.example.app", payload); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateOneTimeProduct, got %v", err)
	}
	if _, err := c.UpdateOneTimeProduct(context.Background(), "com.example.app", "coins_100", payload, "listings"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UpdateOneTimeProduct, got %v", err)
	}
	if err := c.DeleteOneTimeProduct(context.Background(), "com.example.app", "coins_100"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteOneTimeProduct, got %v", err)
	}
	if _, err := c.ListOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", 0, "", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListOneTimeProductOffers, got %v", err)
	}
	if _, err := c.BatchGetOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", []string{"offer_intro"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchGetOneTimeProductOffers, got %v", err)
	}
	if _, err := c.BatchUpdateOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", []*androidpublisher.UpdateOneTimeProductOfferRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchUpdateOneTimeProductOffers, got %v", err)
	}
	if _, err := c.BatchUpdateOneTimeProductOfferStates(context.Background(), "com.example.app", "coins_100", "buy", []*androidpublisher.UpdateOneTimeProductOfferStateRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchUpdateOneTimeProductOfferStates, got %v", err)
	}
	if err := c.BatchDeleteOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", []*androidpublisher.DeleteOneTimeProductOfferRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchDeleteOneTimeProductOffers, got %v", err)
	}
	if _, err := c.ActivateOneTimeProductOffer(context.Background(), "com.example.app", "coins_100", "buy", "offer_intro"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ActivateOneTimeProductOffer, got %v", err)
	}
	if _, err := c.DeactivateOneTimeProductOffer(context.Background(), "com.example.app", "coins_100", "buy", "offer_intro"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeactivateOneTimeProductOffer, got %v", err)
	}
	if _, err := c.CancelOneTimeProductOffer(context.Background(), "com.example.app", "coins_100", "buy", "offer_intro"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CancelOneTimeProductOffer, got %v", err)
	}
	if _, err := c.ActivateOneTimeProductPurchaseOption(context.Background(), "com.example.app", "coins_100", "buy"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ActivateOneTimeProductPurchaseOption, got %v", err)
	}
	if _, err := c.DeactivateOneTimeProductPurchaseOption(context.Background(), "com.example.app", "coins_100", "buy"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeactivateOneTimeProductPurchaseOption, got %v", err)
	}
	if err := c.DeleteOneTimeProductPurchaseOption(context.Background(), "com.example.app", "coins_100", "buy", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteOneTimeProductPurchaseOption, got %v", err)
	}
}

func TestOneTimeProductMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListOneTimeProducts(context.Background(), "", 0, "", false); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListOneTimeProducts package error: %v", err)
	}
	if _, err := c.ListOneTimeProducts(context.Background(), "com.example.app", -1, "", false); err == nil || !strings.Contains(err.Error(), "page size must be greater than or equal to zero") {
		t.Fatalf("unexpected ListOneTimeProducts page size error: %v", err)
	}
	if _, err := c.GetOneTimeProduct(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected GetOneTimeProduct product id error: %v", err)
	}
	if _, err := c.BatchGetOneTimeProducts(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "at least one product id is required") {
		t.Fatalf("unexpected BatchGetOneTimeProducts empty IDs error: %v", err)
	}
	tooManyProductIDs := make([]string, 101)
	for i := range tooManyProductIDs {
		tooManyProductIDs[i] = "coins"
	}
	if _, err := c.BatchGetOneTimeProducts(context.Background(), "com.example.app", tooManyProductIDs); err == nil || !strings.Contains(err.Error(), "product id count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchGetOneTimeProducts count error: %v", err)
	}
	if _, err := c.BatchUpdateOneTimeProducts(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "at least one batch update request is required") {
		t.Fatalf("unexpected BatchUpdateOneTimeProducts empty request error: %v", err)
	}
	tooManyProductUpdateRequests := make([]*androidpublisher.UpdateOneTimeProductRequest, 101)
	for i := range tooManyProductUpdateRequests {
		tooManyProductUpdateRequests[i] = &androidpublisher.UpdateOneTimeProductRequest{}
	}
	if _, err := c.BatchUpdateOneTimeProducts(context.Background(), "com.example.app", tooManyProductUpdateRequests); err == nil || !strings.Contains(err.Error(), "batch update request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchUpdateOneTimeProducts count error: %v", err)
	}
	if err := c.BatchDeleteOneTimeProducts(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "at least one batch delete request is required") {
		t.Fatalf("unexpected BatchDeleteOneTimeProducts empty request error: %v", err)
	}
	tooManyProductDeleteRequests := make([]*androidpublisher.DeleteOneTimeProductRequest, 101)
	for i := range tooManyProductDeleteRequests {
		tooManyProductDeleteRequests[i] = &androidpublisher.DeleteOneTimeProductRequest{}
	}
	if err := c.BatchDeleteOneTimeProducts(context.Background(), "com.example.app", tooManyProductDeleteRequests); err == nil || !strings.Contains(err.Error(), "batch delete request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchDeleteOneTimeProducts count error: %v", err)
	}
	if _, err := c.CreateOneTimeProduct(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "one-time product payload is required") {
		t.Fatalf("unexpected CreateOneTimeProduct payload error: %v", err)
	}
	if _, err := c.CreateOneTimeProduct(context.Background(), "com.example.app", &androidpublisher.OneTimeProduct{}); err == nil || !strings.Contains(err.Error(), "must include productId") {
		t.Fatalf("unexpected CreateOneTimeProduct productId error: %v", err)
	}
	if _, err := c.UpdateOneTimeProduct(context.Background(), "com.example.app", "", &androidpublisher.OneTimeProduct{}, "listings"); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected UpdateOneTimeProduct product id error: %v", err)
	}
	if _, err := c.UpdateOneTimeProduct(context.Background(), "com.example.app", "coins_100", nil, "listings"); err == nil || !strings.Contains(err.Error(), "one-time product payload is required") {
		t.Fatalf("unexpected UpdateOneTimeProduct payload error: %v", err)
	}
	if _, err := c.UpdateOneTimeProduct(context.Background(), "com.example.app", "coins_100", &androidpublisher.OneTimeProduct{}, ""); err == nil || !strings.Contains(err.Error(), "update mask is required") {
		t.Fatalf("unexpected UpdateOneTimeProduct update mask error: %v", err)
	}
	if err := c.DeleteOneTimeProduct(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected DeleteOneTimeProduct product id error: %v", err)
	}
	if _, err := c.ListOneTimeProductOffers(context.Background(), "com.example.app", "", "buy", 0, "", false); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected ListOneTimeProductOffers product id error: %v", err)
	}
	if _, err := c.ListOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "", 0, "", false); err == nil || !strings.Contains(err.Error(), "purchase option id is required") {
		t.Fatalf("unexpected ListOneTimeProductOffers purchase option error: %v", err)
	}
	if _, err := c.ListOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", -1, "", false); err == nil || !strings.Contains(err.Error(), "page size must be greater than or equal to zero") {
		t.Fatalf("unexpected ListOneTimeProductOffers page size error: %v", err)
	}
	if _, err := c.BatchGetOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", nil); err == nil || !strings.Contains(err.Error(), "at least one offer id is required") {
		t.Fatalf("unexpected BatchGetOneTimeProductOffers empty IDs error: %v", err)
	}
	tooManyOfferIDs := make([]string, 101)
	for i := range tooManyOfferIDs {
		tooManyOfferIDs[i] = "offer"
	}
	if _, err := c.BatchGetOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", tooManyOfferIDs); err == nil || !strings.Contains(err.Error(), "offer id count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchGetOneTimeProductOffers count error: %v", err)
	}
	if _, err := c.BatchUpdateOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", nil); err == nil || !strings.Contains(err.Error(), "at least one batch update request is required") {
		t.Fatalf("unexpected BatchUpdateOneTimeProductOffers empty request error: %v", err)
	}
	tooManyUpdateRequests := make([]*androidpublisher.UpdateOneTimeProductOfferRequest, 101)
	for i := range tooManyUpdateRequests {
		tooManyUpdateRequests[i] = &androidpublisher.UpdateOneTimeProductOfferRequest{}
	}
	if _, err := c.BatchUpdateOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", tooManyUpdateRequests); err == nil || !strings.Contains(err.Error(), "batch update request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchUpdateOneTimeProductOffers count error: %v", err)
	}
	if _, err := c.BatchUpdateOneTimeProductOfferStates(context.Background(), "com.example.app", "coins_100", "buy", nil); err == nil || !strings.Contains(err.Error(), "at least one batch state update request is required") {
		t.Fatalf("unexpected BatchUpdateOneTimeProductOfferStates empty request error: %v", err)
	}
	tooManyStateRequests := make([]*androidpublisher.UpdateOneTimeProductOfferStateRequest, 101)
	for i := range tooManyStateRequests {
		tooManyStateRequests[i] = &androidpublisher.UpdateOneTimeProductOfferStateRequest{}
	}
	if _, err := c.BatchUpdateOneTimeProductOfferStates(context.Background(), "com.example.app", "coins_100", "buy", tooManyStateRequests); err == nil || !strings.Contains(err.Error(), "batch state update request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchUpdateOneTimeProductOfferStates count error: %v", err)
	}
	if err := c.BatchDeleteOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", nil); err == nil || !strings.Contains(err.Error(), "at least one batch delete request is required") {
		t.Fatalf("unexpected BatchDeleteOneTimeProductOffers empty request error: %v", err)
	}
	tooManyDeleteRequests := make([]*androidpublisher.DeleteOneTimeProductOfferRequest, 101)
	for i := range tooManyDeleteRequests {
		tooManyDeleteRequests[i] = &androidpublisher.DeleteOneTimeProductOfferRequest{}
	}
	if err := c.BatchDeleteOneTimeProductOffers(context.Background(), "com.example.app", "coins_100", "buy", tooManyDeleteRequests); err == nil || !strings.Contains(err.Error(), "batch delete request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchDeleteOneTimeProductOffers count error: %v", err)
	}
	if _, err := c.ActivateOneTimeProductOffer(context.Background(), "com.example.app", "coins_100", "buy", ""); err == nil || !strings.Contains(err.Error(), "offer id is required") {
		t.Fatalf("unexpected ActivateOneTimeProductOffer offer id error: %v", err)
	}
	if _, err := c.DeactivateOneTimeProductOffer(context.Background(), "com.example.app", "coins_100", "buy", ""); err == nil || !strings.Contains(err.Error(), "offer id is required") {
		t.Fatalf("unexpected DeactivateOneTimeProductOffer offer id error: %v", err)
	}
	if _, err := c.CancelOneTimeProductOffer(context.Background(), "com.example.app", "coins_100", "buy", ""); err == nil || !strings.Contains(err.Error(), "offer id is required") {
		t.Fatalf("unexpected CancelOneTimeProductOffer offer id error: %v", err)
	}
	if _, err := c.ActivateOneTimeProductPurchaseOption(context.Background(), "com.example.app", "coins_100", ""); err == nil || !strings.Contains(err.Error(), "purchase option id is required") {
		t.Fatalf("unexpected ActivateOneTimeProductPurchaseOption purchase option error: %v", err)
	}
	if _, err := c.DeactivateOneTimeProductPurchaseOption(context.Background(), "com.example.app", "", "buy"); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected DeactivateOneTimeProductPurchaseOption product id error: %v", err)
	}
	if err := c.DeleteOneTimeProductPurchaseOption(context.Background(), "com.example.app", "coins_100", "", false); err == nil || !strings.Contains(err.Error(), "purchase option id is required") {
		t.Fatalf("unexpected DeleteOneTimeProductPurchaseOption purchase option error: %v", err)
	}
}

func TestOneTimeProductsListInfoFromResponse(t *testing.T) {
	resp := &androidpublisher.ListOneTimeProductsResponse{
		NextPageToken: "next",
		OneTimeProducts: []*androidpublisher.OneTimeProduct{
			{
				PackageName:     "com.example.app",
				ProductId:       "coins_100",
				Listings:        []*androidpublisher.OneTimeProductListing{{LanguageCode: "en-US"}},
				PurchaseOptions: []*androidpublisher.OneTimeProductPurchaseOption{{}},
				OfferTags:       []*androidpublisher.OfferTag{{Tag: "coins"}},
			},
		},
	}

	got := oneTimeProductsListInfoFromResponse(resp)
	if got.NextPageToken != "next" || len(got.Products) != 1 {
		t.Fatalf("unexpected one-time product list map: %+v", got)
	}
	if got.Products[0].ProductID != "coins_100" || got.Products[0].PurchaseOptionCount != 1 {
		t.Fatalf("unexpected one-time product map: %+v", got.Products[0])
	}
}

func TestOneTimeProductInfoFromProduct(t *testing.T) {
	got := oneTimeProductInfoFromProduct(&androidpublisher.OneTimeProduct{
		PackageName:     "com.example.app",
		ProductId:       "coins_100",
		Listings:        []*androidpublisher.OneTimeProductListing{{LanguageCode: "en-US"}, {LanguageCode: "pl-PL"}},
		PurchaseOptions: []*androidpublisher.OneTimeProductPurchaseOption{{}},
		OfferTags:       []*androidpublisher.OfferTag{{Tag: "coins"}, {Tag: "sale"}},
	})
	if got.PackageName != "com.example.app" || got.ProductID != "coins_100" {
		t.Fatalf("unexpected one-time product info map: %+v", got)
	}
	if got.ListingCount != 2 || got.PurchaseOptionCount != 1 || got.OfferTagCount != 2 {
		t.Fatalf("unexpected one-time product info counts: %+v", got)
	}
}

func TestOneTimeProductDiagnosticInfoFromProduct(t *testing.T) {
	got := oneTimeProductDiagnosticInfoFromProduct(&androidpublisher.OneTimeProduct{
		PackageName: "com.example.app",
		ProductId:   "coins_100",
		Listings: []*androidpublisher.OneTimeProductListing{
			{LanguageCode: "en-US"},
		},
		OfferTags: []*androidpublisher.OfferTag{
			{Tag: "coins"},
			{Tag: "sale"},
		},
		PurchaseOptions: []*androidpublisher.OneTimeProductPurchaseOption{
			{
				PurchaseOptionId: "buy",
				State:            "ACTIVE",
				OfferTags:        []*androidpublisher.OfferTag{{Tag: "tag1"}},
				RegionalPricingAndAvailabilityConfigs: []*androidpublisher.OneTimeProductPurchaseOptionRegionalPricingAndAvailabilityConfig{
					{RegionCode: "US", Availability: "AVAILABLE"},
					{RegionCode: "PL", Availability: "NO_LONGER_AVAILABLE"},
				},
			},
			{
				PurchaseOptionId: "rent",
				State:            "INACTIVE",
				RegionalPricingAndAvailabilityConfigs: []*androidpublisher.OneTimeProductPurchaseOptionRegionalPricingAndAvailabilityConfig{
					{RegionCode: "DE", Availability: "AVAILABLE_FOR_OFFERS_ONLY"},
				},
			},
		},
	})

	if got.ProductID != "coins_100" || got.PurchaseOptionCount != 2 || got.ActivePurchaseOptionCount != 1 || got.ListingCount != 1 || got.RegionCount != 3 || got.AvailableRegionCount != 2 {
		t.Fatalf("unexpected one-time product diagnostic map: %+v", got)
	}
	if len(got.PurchaseOptions) != 2 || got.PurchaseOptions[0].PurchaseOptionID != "buy" {
		t.Fatalf("unexpected purchase option diagnostic map: %+v", got.PurchaseOptions)
	}
}

func TestOneTimeProductOffersListInfoFromResponse(t *testing.T) {
	resp := &androidpublisher.ListOneTimeProductOffersResponse{
		NextPageToken: "next",
		OneTimeProductOffers: []*androidpublisher.OneTimeProductOffer{
			{
				PackageName:      "com.example.app",
				ProductId:        "coins_100",
				PurchaseOptionId: "buy",
				OfferId:          "offer_intro",
				State:            "ACTIVE",
				OfferTags:        []*androidpublisher.OfferTag{{Tag: "sale"}},
				RegionalPricingAndAvailabilityConfigs: []*androidpublisher.OneTimeProductOfferRegionalPricingAndAvailabilityConfig{
					{},
				},
			},
		},
	}

	got := oneTimeProductOffersListInfoFromResponse(resp)
	if got.NextPageToken != "next" || len(got.Offers) != 1 {
		t.Fatalf("unexpected one-time product offers list map: %+v", got)
	}
	if got.Offers[0].OfferID != "offer_intro" || got.Offers[0].OfferTagCount != 1 {
		t.Fatalf("unexpected one-time product offer map: %+v", got.Offers[0])
	}
}

func TestOneTimeProductOfferInfoFromOffer(t *testing.T) {
	got := oneTimeProductOfferInfoFromOffer(&androidpublisher.OneTimeProductOffer{
		PackageName:      "com.example.app",
		ProductId:        "coins_100",
		PurchaseOptionId: "buy",
		OfferId:          "offer_intro",
		State:            "DRAFT",
		OfferTags:        []*androidpublisher.OfferTag{{Tag: "sale"}, {Tag: "new"}},
		RegionalPricingAndAvailabilityConfigs: []*androidpublisher.OneTimeProductOfferRegionalPricingAndAvailabilityConfig{
			{},
			{},
		},
	})
	if got.PackageName != "com.example.app" || got.OfferID != "offer_intro" {
		t.Fatalf("unexpected one-time product offer info map: %+v", got)
	}
	if got.OfferTagCount != 2 || got.RegionalConfigCount != 2 {
		t.Fatalf("unexpected one-time product offer info counts: %+v", got)
	}
}

func TestOneTimeProductInfosFromSlice(t *testing.T) {
	got := oneTimeProductInfosFromSlice([]*androidpublisher.OneTimeProduct{
		{
			PackageName:     "com.example.app",
			ProductId:       "coins_100",
			Listings:        []*androidpublisher.OneTimeProductListing{{LanguageCode: "en-US"}},
			PurchaseOptions: []*androidpublisher.OneTimeProductPurchaseOption{{}},
		},
		nil,
	})
	if len(got) != 2 {
		t.Fatalf("unexpected mapped count: %+v", got)
	}
	if got[0].ProductID != "coins_100" || got[0].PurchaseOptionCount != 1 {
		t.Fatalf("unexpected first mapped product: %+v", got[0])
	}
	if got[1] != (OneTimeProductInfo{}) {
		t.Fatalf("unexpected nil mapping result: %+v", got[1])
	}
}
