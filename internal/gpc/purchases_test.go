package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestPurchaseMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.GetProductPurchase(context.Background(), "com.example.app", "premium", "token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetProductPurchase, got %v", err)
	}
	if _, err := c.GetProductPurchaseV2(context.Background(), "com.example.app", "token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetProductPurchaseV2, got %v", err)
	}
	if err := c.AcknowledgeProductPurchase(context.Background(), "com.example.app", "premium", "token", "payload"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from AcknowledgeProductPurchase, got %v", err)
	}
	if err := c.ConsumeProductPurchase(context.Background(), "com.example.app", "premium", "token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ConsumeProductPurchase, got %v", err)
	}
	if _, err := c.GetSubscriptionPurchase(context.Background(), "com.example.app", "token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetSubscriptionPurchase, got %v", err)
	}
	if _, err := c.GetLegacySubscriptionPurchase(context.Background(), "com.example.app", "premium_monthly", "token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetLegacySubscriptionPurchase, got %v", err)
	}
	if err := c.CancelSubscriptionPurchase(context.Background(), "com.example.app", "token", CancellationTypeUserRequestedStopRenewals); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CancelSubscriptionPurchase, got %v", err)
	}
	if err := c.AcknowledgeLegacySubscriptionPurchase(context.Background(), "com.example.app", "premium_monthly", "token", "payload"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from AcknowledgeLegacySubscriptionPurchase, got %v", err)
	}
	if err := c.CancelLegacySubscriptionPurchase(context.Background(), "com.example.app", "premium_monthly", "token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CancelLegacySubscriptionPurchase, got %v", err)
	}
	if err := c.RevokeSubscriptionPurchase(context.Background(), "com.example.app", "token", RevocationRefundTypeFull); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from RevokeSubscriptionPurchase, got %v", err)
	}
	if err := c.RefundLegacySubscriptionPurchase(context.Background(), "com.example.app", "premium_monthly", "token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from RefundLegacySubscriptionPurchase, got %v", err)
	}
	if err := c.RevokeLegacySubscriptionPurchase(context.Background(), "com.example.app", "premium_monthly", "token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from RevokeLegacySubscriptionPurchase, got %v", err)
	}
	if _, err := c.DeferSubscriptionPurchase(context.Background(), "com.example.app", "token", "etag-1", "604800s", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeferSubscriptionPurchase, got %v", err)
	}
	if _, err := c.DeferLegacySubscriptionPurchase(context.Background(), "com.example.app", "premium_monthly", "token", 100, 200); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeferLegacySubscriptionPurchase, got %v", err)
	}
	if _, err := c.ListVoidedPurchases(context.Background(), "com.example.app", VoidedPurchasesQuery{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListVoidedPurchases, got %v", err)
	}
}

func TestPurchaseMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.GetProductPurchase(context.Background(), "", "premium", "token"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected GetProductPurchase package error: %v", err)
	}
	if _, err := c.GetProductPurchaseV2(context.Background(), "", "token"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected GetProductPurchaseV2 package error: %v", err)
	}
	if _, err := c.GetProductPurchaseV2(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "purchase token is required") {
		t.Fatalf("unexpected GetProductPurchaseV2 token error: %v", err)
	}
	if err := c.AcknowledgeProductPurchase(context.Background(), "com.example.app", "", "token", "payload"); err == nil || !strings.Contains(err.Error(), "product id is required") {
		t.Fatalf("unexpected AcknowledgeProductPurchase product error: %v", err)
	}
	if err := c.ConsumeProductPurchase(context.Background(), "com.example.app", "premium", ""); err == nil || !strings.Contains(err.Error(), "purchase token is required") {
		t.Fatalf("unexpected ConsumeProductPurchase token error: %v", err)
	}
	if _, err := c.GetSubscriptionPurchase(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "purchase token is required") {
		t.Fatalf("unexpected GetSubscriptionPurchase token error: %v", err)
	}
	if _, err := c.GetLegacySubscriptionPurchase(context.Background(), "com.example.app", "", "token"); err == nil || !strings.Contains(err.Error(), "subscription id is required") {
		t.Fatalf("unexpected GetLegacySubscriptionPurchase subscription id error: %v", err)
	}
	if err := c.CancelSubscriptionPurchase(context.Background(), "com.example.app", "token", "UNKNOWN"); err == nil || !strings.Contains(err.Error(), "unsupported cancellation type") {
		t.Fatalf("unexpected CancelSubscriptionPurchase type error: %v", err)
	}
	if err := c.AcknowledgeLegacySubscriptionPurchase(context.Background(), "com.example.app", "", "token", "payload"); err == nil || !strings.Contains(err.Error(), "subscription id is required") {
		t.Fatalf("unexpected AcknowledgeLegacySubscriptionPurchase subscription id error: %v", err)
	}
	if err := c.CancelLegacySubscriptionPurchase(context.Background(), "com.example.app", "", "token"); err == nil || !strings.Contains(err.Error(), "subscription id is required") {
		t.Fatalf("unexpected CancelLegacySubscriptionPurchase subscription id error: %v", err)
	}
	if err := c.RevokeSubscriptionPurchase(context.Background(), "com.example.app", "token", "UNKNOWN"); err == nil || !strings.Contains(err.Error(), "unsupported refund type") {
		t.Fatalf("unexpected RevokeSubscriptionPurchase refund type error: %v", err)
	}
	if err := c.RefundLegacySubscriptionPurchase(context.Background(), "com.example.app", "", "token"); err == nil || !strings.Contains(err.Error(), "subscription id is required") {
		t.Fatalf("unexpected RefundLegacySubscriptionPurchase subscription id error: %v", err)
	}
	if err := c.RevokeLegacySubscriptionPurchase(context.Background(), "com.example.app", "", "token"); err == nil || !strings.Contains(err.Error(), "subscription id is required") {
		t.Fatalf("unexpected RevokeLegacySubscriptionPurchase subscription id error: %v", err)
	}
	if _, err := c.DeferSubscriptionPurchase(context.Background(), "com.example.app", "token", "", "P7D", false); err == nil || !strings.Contains(err.Error(), "etag is required") {
		t.Fatalf("unexpected DeferSubscriptionPurchase etag error: %v", err)
	}
	if _, err := c.DeferSubscriptionPurchase(context.Background(), "com.example.app", "token", "etag-1", "", false); err == nil || !strings.Contains(err.Error(), "defer duration is required") {
		t.Fatalf("unexpected DeferSubscriptionPurchase duration error: %v", err)
	}
	if _, err := c.DeferSubscriptionPurchase(context.Background(), "com.example.app", "token", "etag-1", "P7D", false); err == nil || !strings.Contains(err.Error(), "invalid defer duration format") {
		t.Fatalf("unexpected DeferSubscriptionPurchase format error: %v", err)
	}
	if _, err := c.DeferLegacySubscriptionPurchase(context.Background(), "com.example.app", "premium_monthly", "token", 0, 200); err == nil || !strings.Contains(err.Error(), "expected expiry time millis must be greater than zero") {
		t.Fatalf("unexpected DeferLegacySubscriptionPurchase expected expiry error: %v", err)
	}
	if _, err := c.DeferLegacySubscriptionPurchase(context.Background(), "com.example.app", "premium_monthly", "token", 200, 100); err == nil || !strings.Contains(err.Error(), "desired expiry time millis must be greater than expected expiry time millis") {
		t.Fatalf("unexpected DeferLegacySubscriptionPurchase desired expiry error: %v", err)
	}
	if _, err := c.ListVoidedPurchases(context.Background(), "com.example.app", VoidedPurchasesQuery{MaxResults: -1}); err == nil || !strings.Contains(err.Error(), "max results must be greater than or equal to zero") {
		t.Fatalf("unexpected ListVoidedPurchases max results error: %v", err)
	}
	if _, err := c.ListVoidedPurchases(context.Background(), "com.example.app", VoidedPurchasesQuery{Type: 2}); err == nil || !strings.Contains(err.Error(), "type must be 0 or 1") {
		t.Fatalf("unexpected ListVoidedPurchases type error: %v", err)
	}
}

func TestProductPurchaseInfoFromProductPurchase(t *testing.T) {
	got := productPurchaseInfoFromProductPurchase(&androidpublisher.ProductPurchase{
		OrderId:              "GPA.1",
		ProductId:            "premium",
		PurchaseToken:        "token-1",
		PurchaseState:        0,
		AcknowledgementState: 1,
		ConsumptionState:     0,
		PurchaseTimeMillis:   123,
		RegionCode:           "PL",
	})
	if got.OrderID != "GPA.1" || got.ProductID != "premium" || got.PurchaseToken != "token-1" {
		t.Fatalf("unexpected product purchase map: %+v", got)
	}
}

func TestProductPurchaseV2InfoFromProductPurchaseV2(t *testing.T) {
	got := productPurchaseV2InfoFromProductPurchaseV2(&androidpublisher.ProductPurchaseV2{
		Kind:                   "androidpublisher#productPurchaseV2",
		OrderId:                "GPA.10",
		AcknowledgementState:   "ACKNOWLEDGEMENT_STATE_PENDING",
		PurchaseStateContext:   &androidpublisher.PurchaseStateContext{PurchaseState: "PENDING"},
		RegionCode:             "PL",
		PurchaseCompletionTime: "2026-03-01T10:00:00Z",
		ProductLineItem:        []*androidpublisher.ProductLineItem{{}, {}},
	})
	if got.OrderID != "GPA.10" || got.PurchaseState != "PENDING" || got.LineItemCount != 2 {
		t.Fatalf("unexpected product purchase v2 map: %+v", got)
	}
}

func TestSubscriptionPurchaseInfoFromSubscriptionPurchaseV2(t *testing.T) {
	got := subscriptionPurchaseInfoFromSubscriptionPurchaseV2(&androidpublisher.SubscriptionPurchaseV2{
		Kind:                 "androidpublisher#subscriptionPurchaseV2",
		LatestOrderId:        "GPA.2",
		SubscriptionState:    "SUBSCRIPTION_STATE_ACTIVE",
		AcknowledgementState: "ACKNOWLEDGEMENT_STATE_ACKNOWLEDGED",
		RegionCode:           "US",
		StartTime:            "2026-01-01T00:00:00Z",
		LineItems:            []*androidpublisher.SubscriptionPurchaseLineItem{{}, {}},
	})
	if got.LatestOrderID != "GPA.2" || got.LineItemCount != 2 || got.SubscriptionState != "SUBSCRIPTION_STATE_ACTIVE" {
		t.Fatalf("unexpected subscription purchase map: %+v", got)
	}
}

func TestSubscriptionPurchaseInfoFromLegacySubscriptionPurchase(t *testing.T) {
	paymentState := int64(1)
	got := subscriptionPurchaseInfoFromLegacySubscriptionPurchase(&androidpublisher.SubscriptionPurchase{
		Kind:                 "androidpublisher#subscriptionPurchase",
		OrderId:              "GPA.3",
		AcknowledgementState: 1,
		AutoRenewing:         true,
		CountryCode:          "PL",
		ExpiryTimeMillis:     123456789,
		PaymentState:         &paymentState,
	})
	if got.LatestOrderID != "GPA.3" || !got.AutoRenewing || got.ExpiryTimeMillis != 123456789 {
		t.Fatalf("unexpected legacy subscription purchase map: %+v", got)
	}
	if got.AcknowledgementState != "ACKNOWLEDGED" || got.RegionCode != "PL" {
		t.Fatalf("unexpected legacy subscription purchase metadata: %+v", got)
	}
}

func TestSubscriptionDeferInfoFromResponse(t *testing.T) {
	got := subscriptionDeferInfoFromResponse(&androidpublisher.DeferSubscriptionPurchaseResponse{
		ItemExpiryTimeDetails: []*androidpublisher.ItemExpiryTimeDetails{
			{ProductId: "premium_monthly", ExpiryTime: "2026-04-01T00:00:00Z"},
			nil,
		},
	})
	if len(got.ItemExpiryTimeDetails) != 2 {
		t.Fatalf("unexpected item count: %+v", got)
	}
	if got.ItemExpiryTimeDetails[0].ProductID != "premium_monthly" || got.ItemExpiryTimeDetails[0].ExpiryTime == "" {
		t.Fatalf("unexpected first item mapping: %+v", got.ItemExpiryTimeDetails[0])
	}
	if got.ItemExpiryTimeDetails[1] != (SubscriptionItemExpiryInfo{}) {
		t.Fatalf("unexpected nil item mapping: %+v", got.ItemExpiryTimeDetails[1])
	}
}

func TestLegacySubscriptionDeferInfoFromResponse(t *testing.T) {
	got := subscriptionDeferInfoFromLegacyResponse(&androidpublisher.SubscriptionPurchasesDeferResponse{
		NewExpiryTimeMillis: 987654321,
	})
	if got.NewExpiryTimeMillis != 987654321 {
		t.Fatalf("unexpected legacy defer mapping: %+v", got)
	}
}

func TestVoidedPurchasesListInfoFromResponse(t *testing.T) {
	got := voidedPurchasesListInfoFromResponse(&androidpublisher.VoidedPurchasesListResponse{
		TokenPagination: &androidpublisher.TokenPagination{NextPageToken: "next-token"},
		VoidedPurchases: []*androidpublisher.VoidedPurchase{
			{
				OrderId:            "GPA.3",
				PurchaseToken:      "token-3",
				PurchaseTimeMillis: 123,
				VoidedTimeMillis:   456,
				VoidedReason:       7,
				VoidedSource:       2,
				VoidedQuantity:     1,
			},
		},
	})
	if got.NextToken != "next-token" || len(got.VoidedPurchases) != 1 || got.VoidedPurchases[0].OrderID != "GPA.3" {
		t.Fatalf("unexpected voided purchases map: %+v", got)
	}
}
