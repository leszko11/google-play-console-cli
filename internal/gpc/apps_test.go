package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSetDataSafety_RejectsMissingClient(t *testing.T) {
	var c *Client

	if err := c.SetDataSafety(context.Background(), "com.example.app", "header\nvalue\n"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from SetDataSafety, got %v", err)
	}
}

func TestSetDataSafety_ValidatesArgs(t *testing.T) {
	c := &Client{}

	if err := c.SetDataSafety(context.Background(), "", "header\nvalue\n"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected package error: %v", err)
	}
	if err := c.SetDataSafety(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "safety labels CSV is required") {
		t.Fatalf("unexpected CSV error: %v", err)
	}
}
