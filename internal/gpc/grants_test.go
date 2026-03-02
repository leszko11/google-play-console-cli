package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestGrantMethods_RejectMissingClient(t *testing.T) {
	var c *Client
	grant := &androidpublisher.Grant{Name: "developers/123/users/dev@example.com/grants/com.example.app"}

	if _, err := c.CreateGrant(context.Background(), "developers/123/users/dev@example.com", grant); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateGrant, got %v", err)
	}
	if _, err := c.UpdateGrant(context.Background(), "developers/123/users/dev@example.com/grants/com.example.app", grant, "appLevelPermissions"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UpdateGrant, got %v", err)
	}
	if err := c.DeleteGrant(context.Background(), "developers/123/users/dev@example.com/grants/com.example.app"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteGrant, got %v", err)
	}
}

func TestGrantMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.CreateGrant(context.Background(), "", &androidpublisher.Grant{}); err == nil || !strings.Contains(err.Error(), "parent is required") {
		t.Fatalf("unexpected CreateGrant parent error: %v", err)
	}
	if _, err := c.CreateGrant(context.Background(), "developers/123/users/dev@example.com", nil); err == nil || !strings.Contains(err.Error(), "grant payload is required") {
		t.Fatalf("unexpected CreateGrant payload error: %v", err)
	}
	if _, err := c.UpdateGrant(context.Background(), "", &androidpublisher.Grant{}, ""); err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("unexpected UpdateGrant name error: %v", err)
	}
	if _, err := c.UpdateGrant(context.Background(), "developers/123/users/dev@example.com/grants/com.example.app", nil, ""); err == nil || !strings.Contains(err.Error(), "grant payload is required") {
		t.Fatalf("unexpected UpdateGrant payload error: %v", err)
	}
	if err := c.DeleteGrant(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("unexpected DeleteGrant name error: %v", err)
	}
}

func TestGrantInfoFromGrant(t *testing.T) {
	got := grantInfoFromGrant(&androidpublisher.Grant{
		Name:                "developers/123/users/dev@example.com/grants/com.example.app",
		PackageName:         "com.example.app",
		AppLevelPermissions: []string{"CAN_VIEW_NON_FINANCIAL_DATA", "CAN_REPLY_TO_REVIEWS"},
	})
	if got.PackageName != "com.example.app" || got.PermissionCount != 2 {
		t.Fatalf("unexpected grant info map: %+v", got)
	}
	if len(got.AppLevelPermissions) != 2 {
		t.Fatalf("unexpected permissions map: %+v", got)
	}
}
