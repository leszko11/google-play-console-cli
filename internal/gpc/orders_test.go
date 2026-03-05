package gpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestOrderMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.GetOrder(context.Background(), "com.example.app", "GPA.1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetOrder, got %v", err)
	}
	if _, err := c.BatchGetOrders(context.Background(), "com.example.app", []string{"GPA.1"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from BatchGetOrders, got %v", err)
	}
	if err := c.RefundOrder(context.Background(), "com.example.app", "GPA.1", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from RefundOrder, got %v", err)
	}
}

func TestOrderMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.GetOrder(context.Background(), "", "GPA.1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected GetOrder package error: %v", err)
	}
	if _, err := c.GetOrder(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "order id is required") {
		t.Fatalf("unexpected GetOrder order id error: %v", err)
	}
	if _, err := c.BatchGetOrders(context.Background(), "", []string{"GPA.1"}); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected BatchGetOrders package error: %v", err)
	}
	if _, err := c.BatchGetOrders(context.Background(), "com.example.app", []string{"", "   "}); err == nil || !strings.Contains(err.Error(), "at least one order id is required") {
		t.Fatalf("unexpected BatchGetOrders empty order ids error: %v", err)
	}
	tooMany := make([]string, maxBatchGetOrders+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("GPA.%d", i+1)
	}
	if _, err := c.BatchGetOrders(context.Background(), "com.example.app", tooMany); err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("unexpected BatchGetOrders max order ids error: %v", err)
	}
	if err := c.RefundOrder(context.Background(), "", "GPA.1", false); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected RefundOrder package error: %v", err)
	}
	if err := c.RefundOrder(context.Background(), "com.example.app", "", false); err == nil || !strings.Contains(err.Error(), "order id is required") {
		t.Fatalf("unexpected RefundOrder order id error: %v", err)
	}
}

func TestNormalizeOrderIDs(t *testing.T) {
	got := normalizeOrderIDs([]string{" GPA.1 ", "", "GPA.2", "GPA.1", "   "})
	if len(got) != 2 || got[0] != "GPA.1" || got[1] != "GPA.2" {
		t.Fatalf("unexpected normalized order ids: %#v", got)
	}
}

func TestOrderInfoFromOrder(t *testing.T) {
	got := orderInfoFromOrder(&androidpublisher.Order{
		OrderId:       "GPA.1",
		PurchaseToken: "tok-1",
		State:         "PROCESSED",
		SalesChannel:  "PLAY_STORE",
		CreateTime:    "2026-03-05T10:00:00Z",
		LastEventTime: "2026-03-05T10:05:00Z",
		BuyerAddress:  &androidpublisher.BuyerAddress{BuyerCountry: "PL"},
		LineItems: []*androidpublisher.LineItem{
			{ProductId: "premium_monthly"},
			nil,
			{ProductId: "coins_100"},
		},
		Total: &androidpublisher.Money{CurrencyCode: "USD", Units: 12, Nanos: 340000000},
		Tax:   &androidpublisher.Money{CurrencyCode: "USD", Units: 2},
		DeveloperRevenueInBuyerCurrency: &androidpublisher.Money{
			CurrencyCode: "USD",
			Units:        9,
			Nanos:        120000000,
		},
	})
	if got.OrderID != "GPA.1" || got.BuyerCountry != "PL" || got.LineItemCount != 3 {
		t.Fatalf("unexpected order map: %+v", got)
	}
	if len(got.LineItemProductIDs) != 2 || got.LineItemProductIDs[0] != "premium_monthly" || got.LineItemProductIDs[1] != "coins_100" {
		t.Fatalf("unexpected product ids: %#v", got.LineItemProductIDs)
	}
	if got.Total.CurrencyCode != "USD" || got.Total.Units != 12 || got.DeveloperRevenueInBuyer.Units != 9 {
		t.Fatalf("unexpected money mapping: %+v", got)
	}
}
