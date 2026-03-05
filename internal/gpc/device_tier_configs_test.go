package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestDeviceTierConfigMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.ListDeviceTierConfigs(context.Background(), "com.example.app", 0, "", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListDeviceTierConfigs, got %v", err)
	}
	if _, err := c.GetDeviceTierConfig(context.Background(), "com.example.app", 1); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetDeviceTierConfig, got %v", err)
	}
	if _, err := c.CreateDeviceTierConfig(context.Background(), "com.example.app", &androidpublisher.DeviceTierConfig{}, false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateDeviceTierConfig, got %v", err)
	}
}

func TestDeviceTierConfigMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListDeviceTierConfigs(context.Background(), "", 0, "", false); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListDeviceTierConfigs package error: %v", err)
	}
	if _, err := c.ListDeviceTierConfigs(context.Background(), "com.example.app", -1, "", false); err == nil || !strings.Contains(err.Error(), "page size must be greater than or equal to zero") {
		t.Fatalf("unexpected ListDeviceTierConfigs page size error: %v", err)
	}
	if _, err := c.GetDeviceTierConfig(context.Background(), "com.example.app", 0); err == nil || !strings.Contains(err.Error(), "device tier config id must be greater than zero") {
		t.Fatalf("unexpected GetDeviceTierConfig id error: %v", err)
	}
	if _, err := c.CreateDeviceTierConfig(context.Background(), "", &androidpublisher.DeviceTierConfig{}, false); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected CreateDeviceTierConfig package error: %v", err)
	}
	if _, err := c.CreateDeviceTierConfig(context.Background(), "com.example.app", nil, false); err == nil || !strings.Contains(err.Error(), "payload is required") {
		t.Fatalf("unexpected CreateDeviceTierConfig payload error: %v", err)
	}
}
