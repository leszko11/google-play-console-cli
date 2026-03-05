package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestExternalTransactionMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.GetExternalTransaction(context.Background(), "com.example.app", "ext-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetExternalTransaction, got %v", err)
	}
	if _, err := c.CreateExternalTransaction(context.Background(), "com.example.app", "ext-1", &androidpublisher.ExternalTransaction{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateExternalTransaction, got %v", err)
	}
	if _, err := c.RefundExternalTransaction(context.Background(), "com.example.app", "ext-1", &androidpublisher.RefundExternalTransactionRequest{
		RefundTime: "2026-03-05T10:00:00Z",
		FullRefund: &androidpublisher.FullRefund{},
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from RefundExternalTransaction, got %v", err)
	}
}

func TestExternalTransactionMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.GetExternalTransaction(context.Background(), "", "ext-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected GetExternalTransaction package error: %v", err)
	}
	if _, err := c.GetExternalTransaction(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "external transaction id is required") {
		t.Fatalf("unexpected GetExternalTransaction id error: %v", err)
	}
	if _, err := c.CreateExternalTransaction(context.Background(), "com.example.app", "ext-1", nil); err == nil || !strings.Contains(err.Error(), "payload is required") {
		t.Fatalf("unexpected CreateExternalTransaction payload error: %v", err)
	}
	if _, err := c.RefundExternalTransaction(context.Background(), "com.example.app", "ext-1", nil); err == nil || !strings.Contains(err.Error(), "refund payload is required") {
		t.Fatalf("unexpected RefundExternalTransaction nil request error: %v", err)
	}
	if _, err := c.RefundExternalTransaction(context.Background(), "com.example.app", "ext-1", &androidpublisher.RefundExternalTransactionRequest{}); err == nil || !strings.Contains(err.Error(), "refund time is required") {
		t.Fatalf("unexpected RefundExternalTransaction refund time error: %v", err)
	}
	if _, err := c.RefundExternalTransaction(context.Background(), "com.example.app", "ext-1", &androidpublisher.RefundExternalTransactionRequest{
		RefundTime: "2026-03-05T10:00:00Z",
	}); err == nil || !strings.Contains(err.Error(), "exactly one of fullRefund or partialRefund is required") {
		t.Fatalf("unexpected RefundExternalTransaction refund kind error: %v", err)
	}
	if _, err := c.RefundExternalTransaction(context.Background(), "com.example.app", "ext-1", &androidpublisher.RefundExternalTransactionRequest{
		RefundTime:    "2026-03-05T10:00:00Z",
		FullRefund:    &androidpublisher.FullRefund{},
		PartialRefund: &androidpublisher.PartialRefund{},
	}); err == nil || !strings.Contains(err.Error(), "exactly one of fullRefund or partialRefund is required") {
		t.Fatalf("unexpected RefundExternalTransaction mixed refund kind error: %v", err)
	}
	if _, err := c.RefundExternalTransaction(context.Background(), "com.example.app", "ext-1", &androidpublisher.RefundExternalTransactionRequest{
		RefundTime:    "2026-03-05T10:00:00Z",
		PartialRefund: &androidpublisher.PartialRefund{},
	}); err == nil || !strings.Contains(err.Error(), "partial refund id is required") {
		t.Fatalf("unexpected RefundExternalTransaction partial refund id error: %v", err)
	}
	if _, err := c.RefundExternalTransaction(context.Background(), "com.example.app", "ext-1", &androidpublisher.RefundExternalTransactionRequest{
		RefundTime: "2026-03-05T10:00:00Z",
		PartialRefund: &androidpublisher.PartialRefund{
			RefundId: "partial-1",
		},
	}); err == nil || !strings.Contains(err.Error(), "partial refund pre-tax amount is required") {
		t.Fatalf("unexpected RefundExternalTransaction partial refund amount error: %v", err)
	}
}

func TestExternalTransactionResourceNames(t *testing.T) {
	if got := externalTransactionParent("com.example.app"); got != "applications/com.example.app" {
		t.Fatalf("unexpected parent: %q", got)
	}
	if got := externalTransactionName("com.example.app", "ext-1"); got != "applications/com.example.app/externalTransactions/ext-1" {
		t.Fatalf("unexpected name: %q", got)
	}
}
