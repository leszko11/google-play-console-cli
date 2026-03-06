package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestSystemAPKMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.ListSystemAPKVariants(context.Background(), "com.example.app", 123); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListSystemAPKVariants, got %v", err)
	}
	if _, err := c.GetSystemAPKVariant(context.Background(), "com.example.app", 123, 7); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetSystemAPKVariant, got %v", err)
	}
	if _, err := c.CreateSystemAPKVariant(context.Background(), "com.example.app", 123, &androidpublisher.Variant{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateSystemAPKVariant, got %v", err)
	}
	if _, err := c.DownloadSystemAPKVariant(context.Background(), "com.example.app", 123, 7); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DownloadSystemAPKVariant, got %v", err)
	}
}

func TestSystemAPKMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListSystemAPKVariants(context.Background(), "", 123); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListSystemAPKVariants package error: %v", err)
	}
	if _, err := c.ListSystemAPKVariants(context.Background(), "com.example.app", 0); err == nil || !strings.Contains(err.Error(), "version code must be greater than zero") {
		t.Fatalf("unexpected ListSystemAPKVariants version code error: %v", err)
	}
	if _, err := c.GetSystemAPKVariant(context.Background(), "com.example.app", 123, 0); err == nil || !strings.Contains(err.Error(), "variant id must be greater than zero") {
		t.Fatalf("unexpected GetSystemAPKVariant variant id error: %v", err)
	}
	if _, err := c.CreateSystemAPKVariant(context.Background(), "", 123, &androidpublisher.Variant{}); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected CreateSystemAPKVariant package error: %v", err)
	}
	if _, err := c.CreateSystemAPKVariant(context.Background(), "com.example.app", 0, &androidpublisher.Variant{}); err == nil || !strings.Contains(err.Error(), "version code must be greater than zero") {
		t.Fatalf("unexpected CreateSystemAPKVariant version code error: %v", err)
	}
	if _, err := c.CreateSystemAPKVariant(context.Background(), "com.example.app", 123, nil); err == nil || !strings.Contains(err.Error(), "payload is required") {
		t.Fatalf("unexpected CreateSystemAPKVariant payload error: %v", err)
	}
	if _, err := c.DownloadSystemAPKVariant(context.Background(), "com.example.app", 123, 0); err == nil || !strings.Contains(err.Error(), "variant id must be greater than zero") {
		t.Fatalf("unexpected DownloadSystemAPKVariant variant id error: %v", err)
	}
}
