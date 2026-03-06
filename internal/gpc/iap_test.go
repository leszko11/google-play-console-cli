package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestIAPMethods_RejectMissingClient(t *testing.T) {
	var c *Client
	payload := &androidpublisher.InAppProduct{Sku: "coins_100"}

	if _, err := c.ListIAPs(context.Background(), "com.example.app", 0, "", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListIAPs, got %v", err)
	}
	if _, err := c.GetIAP(context.Background(), "com.example.app", "coins_100"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetIAP, got %v", err)
	}
	if _, err := c.BatchGetIAPs(context.Background(), "com.example.app", []string{"coins_100"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchGetIAPs, got %v", err)
	}
	if _, err := c.CreateIAP(context.Background(), "com.example.app", payload); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateIAP, got %v", err)
	}
	if _, err := c.BatchUpdateIAPs(context.Background(), "com.example.app", []*androidpublisher.InappproductsUpdateRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchUpdateIAPs, got %v", err)
	}
	if _, err := c.UpdateIAP(context.Background(), "com.example.app", "coins_100", payload); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UpdateIAP, got %v", err)
	}
	if _, err := c.PatchIAP(context.Background(), "com.example.app", "coins_100", payload); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from PatchIAP, got %v", err)
	}
	if err := c.BatchDeleteIAPs(context.Background(), "com.example.app", []*androidpublisher.InappproductsDeleteRequest{{}}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchDeleteIAPs, got %v", err)
	}
	if err := c.DeleteIAP(context.Background(), "com.example.app", "coins_100"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteIAP, got %v", err)
	}
}

func TestIAPMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListIAPs(context.Background(), "", 0, "", false); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListIAPs package error: %v", err)
	}
	if _, err := c.ListIAPs(context.Background(), "com.example.app", -1, "", false); err == nil || !strings.Contains(err.Error(), "max results must be greater than or equal to zero") {
		t.Fatalf("unexpected ListIAPs max results error: %v", err)
	}
	if _, err := c.GetIAP(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "sku is required") {
		t.Fatalf("unexpected GetIAP sku error: %v", err)
	}
	if _, err := c.BatchGetIAPs(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "at least one sku is required") {
		t.Fatalf("unexpected BatchGetIAPs sku error: %v", err)
	}
	if _, err := c.CreateIAP(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "in-app product payload is required") {
		t.Fatalf("unexpected CreateIAP payload error: %v", err)
	}
	if _, err := c.BatchUpdateIAPs(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "at least one batch update request is required") {
		t.Fatalf("unexpected BatchUpdateIAPs empty request error: %v", err)
	}
	batchUpdateRequests := make([]*androidpublisher.InappproductsUpdateRequest, 101)
	for i := range batchUpdateRequests {
		batchUpdateRequests[i] = &androidpublisher.InappproductsUpdateRequest{}
	}
	if _, err := c.BatchUpdateIAPs(context.Background(), "com.example.app", batchUpdateRequests); err == nil || !strings.Contains(err.Error(), "batch update request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchUpdateIAPs request count error: %v", err)
	}
	if _, err := c.UpdateIAP(context.Background(), "com.example.app", "", &androidpublisher.InAppProduct{}); err == nil || !strings.Contains(err.Error(), "sku is required") {
		t.Fatalf("unexpected UpdateIAP sku error: %v", err)
	}
	if _, err := c.UpdateIAP(context.Background(), "com.example.app", "coins_100", nil); err == nil || !strings.Contains(err.Error(), "in-app product payload is required") {
		t.Fatalf("unexpected UpdateIAP payload error: %v", err)
	}
	if _, err := c.PatchIAP(context.Background(), "com.example.app", "", &androidpublisher.InAppProduct{}); err == nil || !strings.Contains(err.Error(), "sku is required") {
		t.Fatalf("unexpected PatchIAP sku error: %v", err)
	}
	if _, err := c.PatchIAP(context.Background(), "com.example.app", "coins_100", nil); err == nil || !strings.Contains(err.Error(), "in-app product payload is required") {
		t.Fatalf("unexpected PatchIAP payload error: %v", err)
	}
	if err := c.BatchDeleteIAPs(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "at least one batch delete request is required") {
		t.Fatalf("unexpected BatchDeleteIAPs empty request error: %v", err)
	}
	batchDeleteRequests := make([]*androidpublisher.InappproductsDeleteRequest, 101)
	for i := range batchDeleteRequests {
		batchDeleteRequests[i] = &androidpublisher.InappproductsDeleteRequest{}
	}
	if err := c.BatchDeleteIAPs(context.Background(), "com.example.app", batchDeleteRequests); err == nil || !strings.Contains(err.Error(), "batch delete request count must be less than or equal to 100") {
		t.Fatalf("unexpected BatchDeleteIAPs request count error: %v", err)
	}
	if err := c.DeleteIAP(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "sku is required") {
		t.Fatalf("unexpected DeleteIAP sku error: %v", err)
	}
}

func TestIAPsListInfoFromResponse(t *testing.T) {
	resp := &androidpublisher.InappproductsListResponse{
		TokenPagination: &androidpublisher.TokenPagination{NextPageToken: "next"},
		Inappproduct: []*androidpublisher.InAppProduct{
			{
				PackageName:  "com.example.app",
				Sku:          "coins_100",
				Status:       "active",
				PurchaseType: "managedUser",
				Listings: map[string]androidpublisher.InAppProductListing{
					"en-US": {Title: "Coins"},
				},
				Prices: map[string]androidpublisher.Price{
					"US": {Currency: "USD"},
				},
			},
		},
	}

	got := iapsListInfoFromResponse(resp)
	if got.NextPageToken != "next" || len(got.Products) != 1 {
		t.Fatalf("unexpected iap list map: %+v", got)
	}
	if got.Products[0].SKU != "coins_100" || got.Products[0].PriceCount != 1 {
		t.Fatalf("unexpected iap map: %+v", got.Products[0])
	}
}

func TestIAPsListInfoFromProducts(t *testing.T) {
	got := iapsListInfoFromProducts([]*androidpublisher.InAppProduct{
		{
			PackageName: "com.example.app",
			Sku:         "coins_100",
		},
		nil,
	})
	if len(got.Products) != 2 {
		t.Fatalf("unexpected iap list size: %+v", got)
	}
	if got.Products[0].SKU != "coins_100" {
		t.Fatalf("unexpected first mapped product: %+v", got.Products[0])
	}
	if got.Products[1] != (IAPInfo{}) {
		t.Fatalf("unexpected nil product mapping: %+v", got.Products[1])
	}
}

func TestIAPInfoFromProduct(t *testing.T) {
	got := iapInfoFromProduct(&androidpublisher.InAppProduct{
		PackageName:  "com.example.app",
		Sku:          "coins_100",
		Status:       "inactive",
		PurchaseType: "managedUser",
		Listings: map[string]androidpublisher.InAppProductListing{
			"en-US": {Title: "Coins"},
			"pl-PL": {Title: "Monety"},
		},
		Prices: map[string]androidpublisher.Price{
			"US": {Currency: "USD"},
			"PL": {Currency: "PLN"},
		},
	})
	if got.PackageName != "com.example.app" || got.SKU != "coins_100" {
		t.Fatalf("unexpected iap info map: %+v", got)
	}
	if got.ListingCount != 2 || got.PriceCount != 2 {
		t.Fatalf("unexpected iap info counts: %+v", got)
	}
}
