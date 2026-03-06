package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestAppRecoveryMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.ListAppRecoveries(context.Background(), "com.example.app", 123); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListAppRecoveries, got %v", err)
	}
	if _, err := c.CreateAppRecovery(context.Background(), "com.example.app", &androidpublisher.CreateDraftAppRecoveryRequest{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateAppRecovery, got %v", err)
	}
	if err := c.AddAppRecoveryTargeting(context.Background(), "com.example.app", 7, &androidpublisher.AddTargetingRequest{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from AddAppRecoveryTargeting, got %v", err)
	}
	if err := c.CancelAppRecovery(context.Background(), "com.example.app", 7); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CancelAppRecovery, got %v", err)
	}
	if err := c.DeployAppRecovery(context.Background(), "com.example.app", 7); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeployAppRecovery, got %v", err)
	}
}

func TestAppRecoveryMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListAppRecoveries(context.Background(), "", 123); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListAppRecoveries package error: %v", err)
	}
	if _, err := c.ListAppRecoveries(context.Background(), "com.example.app", 0); err == nil || !strings.Contains(err.Error(), "version code must be greater than zero") {
		t.Fatalf("unexpected ListAppRecoveries version code error: %v", err)
	}
	if _, err := c.CreateAppRecovery(context.Background(), "com.example.app", nil); err == nil || !strings.Contains(err.Error(), "payload is required") {
		t.Fatalf("unexpected CreateAppRecovery payload error: %v", err)
	}
	if err := c.AddAppRecoveryTargeting(context.Background(), "com.example.app", 0, &androidpublisher.AddTargetingRequest{}); err == nil || !strings.Contains(err.Error(), "app recovery id must be greater than zero") {
		t.Fatalf("unexpected AddAppRecoveryTargeting id error: %v", err)
	}
	if err := c.AddAppRecoveryTargeting(context.Background(), "com.example.app", 7, nil); err == nil || !strings.Contains(err.Error(), "payload is required") {
		t.Fatalf("unexpected AddAppRecoveryTargeting payload error: %v", err)
	}
	if err := c.CancelAppRecovery(context.Background(), "com.example.app", 0); err == nil || !strings.Contains(err.Error(), "app recovery id must be greater than zero") {
		t.Fatalf("unexpected CancelAppRecovery id error: %v", err)
	}
	if err := c.DeployAppRecovery(context.Background(), "com.example.app", 0); err == nil || !strings.Contains(err.Error(), "app recovery id must be greater than zero") {
		t.Fatalf("unexpected DeployAppRecovery id error: %v", err)
	}
}
