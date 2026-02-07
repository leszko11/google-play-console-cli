package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestEditMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.CreateEdit(context.Background(), "com.example.app"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateEdit, got %v", err)
	}
	if _, err := c.GetEdit(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetEdit, got %v", err)
	}
	if err := c.ValidateEdit(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ValidateEdit, got %v", err)
	}
	if _, err := c.CommitEdit(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CommitEdit, got %v", err)
	}
	if err := c.DeleteEdit(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteEdit, got %v", err)
	}
}

func TestEditMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.CreateEdit(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected CreateEdit error: %v", err)
	}
	if _, err := c.GetEdit(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected GetEdit error: %v", err)
	}
	if err := c.ValidateEdit(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected ValidateEdit error: %v", err)
	}
	if _, err := c.CommitEdit(context.Background(), "", "edit-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected CommitEdit error: %v", err)
	}
	if err := c.DeleteEdit(context.Background(), "", "edit-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected DeleteEdit error: %v", err)
	}
}

func TestEditInfoFromAppEdit(t *testing.T) {
	got := editInfoFromAppEdit(&androidpublisher.AppEdit{
		Id:                "edit-123",
		ExpiryTimeSeconds: "1234567890",
	})
	if got.ID != "edit-123" || got.ExpiryTimeSeconds != "1234567890" {
		t.Fatalf("unexpected mapped edit: %+v", got)
	}
}
