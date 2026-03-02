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
