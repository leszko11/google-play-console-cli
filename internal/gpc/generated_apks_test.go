package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGeneratedAPKMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.ListGeneratedAPKs(context.Background(), "com.example.app", 123); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListGeneratedAPKs, got %v", err)
	}
	if _, err := c.DownloadGeneratedAPK(context.Background(), "com.example.app", 123, "download-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DownloadGeneratedAPK, got %v", err)
	}
}

func TestGeneratedAPKMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListGeneratedAPKs(context.Background(), "", 123); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListGeneratedAPKs package error: %v", err)
	}
	if _, err := c.ListGeneratedAPKs(context.Background(), "com.example.app", 0); err == nil || !strings.Contains(err.Error(), "version code must be greater than zero") {
		t.Fatalf("unexpected ListGeneratedAPKs version code error: %v", err)
	}
	if _, err := c.DownloadGeneratedAPK(context.Background(), "com.example.app", 123, ""); err == nil || !strings.Contains(err.Error(), "download id is required") {
		t.Fatalf("unexpected DownloadGeneratedAPK download id error: %v", err)
	}
}
