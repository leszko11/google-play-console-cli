package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestUserMethods_RejectMissingClient(t *testing.T) {
	var c *Client
	user := &androidpublisher.User{Name: "developers/123/users/dev@example.com", Email: "dev@example.com"}

	if _, err := c.ListUsers(context.Background(), "123", 0, "", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListUsers, got %v", err)
	}
	if _, err := c.CreateUser(context.Background(), "123", user); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateUser, got %v", err)
	}
	if _, err := c.UpdateUser(context.Background(), "developers/123/users/dev@example.com", user, "expirationTime"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UpdateUser, got %v", err)
	}
	if err := c.DeleteUser(context.Background(), "developers/123/users/dev@example.com"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteUser, got %v", err)
	}
}

func TestUserMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListUsers(context.Background(), "", 0, "", false); err == nil || !strings.Contains(err.Error(), "developer id is required") {
		t.Fatalf("unexpected ListUsers developer id error: %v", err)
	}
	if _, err := c.ListUsers(context.Background(), "123", -1, "", false); err == nil || !strings.Contains(err.Error(), "page size must be greater than or equal to zero") {
		t.Fatalf("unexpected ListUsers page size error: %v", err)
	}
	if _, err := c.CreateUser(context.Background(), "123", nil); err == nil || !strings.Contains(err.Error(), "user payload is required") {
		t.Fatalf("unexpected CreateUser payload error: %v", err)
	}
	if _, err := c.UpdateUser(context.Background(), "", &androidpublisher.User{}, ""); err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("unexpected UpdateUser name error: %v", err)
	}
	if _, err := c.UpdateUser(context.Background(), "developers/123/users/dev@example.com", nil, ""); err == nil || !strings.Contains(err.Error(), "user payload is required") {
		t.Fatalf("unexpected UpdateUser payload error: %v", err)
	}
	if err := c.DeleteUser(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("unexpected DeleteUser name error: %v", err)
	}
}

func TestNormalizeDeveloperParent(t *testing.T) {
	got, err := normalizeDeveloperParent("123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "developers/123456" {
		t.Fatalf("unexpected parent: %q", got)
	}

	got, err = normalizeDeveloperParent("developers/654321")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "developers/654321" {
		t.Fatalf("unexpected parent pass-through: %q", got)
	}
}

func TestUsersListInfoFromResponse(t *testing.T) {
	resp := &androidpublisher.ListUsersResponse{
		NextPageToken: "next",
		Users: []*androidpublisher.User{
			{
				Name:                        "developers/123/users/dev@example.com",
				Email:                       "dev@example.com",
				AccessState:                 "ACCESS_GRANTED",
				DeveloperAccountPermissions: []string{"CAN_VIEW_NON_FINANCIAL_DATA_GLOBAL"},
				Grants:                      []*androidpublisher.Grant{{Name: "grant-1"}},
			},
		},
	}

	got := usersListInfoFromResponse(resp)
	if got.NextPageToken != "next" || len(got.Users) != 1 {
		t.Fatalf("unexpected users list map: %+v", got)
	}
	if got.Users[0].Email != "dev@example.com" || got.Users[0].GrantCount != 1 {
		t.Fatalf("unexpected user map: %+v", got.Users[0])
	}
}

func TestUserInfoFromUser(t *testing.T) {
	got := userInfoFromUser(&androidpublisher.User{
		Name:                        "developers/123/users/dev@example.com",
		Email:                       "dev@example.com",
		AccessState:                 "ACCESS_GRANTED",
		ExpirationTime:              "2026-05-01T00:00:00Z",
		Partial:                     true,
		DeveloperAccountPermissions: []string{"CAN_VIEW_NON_FINANCIAL_DATA_GLOBAL"},
		Grants:                      []*androidpublisher.Grant{{Name: "grant-1"}, {Name: "grant-2"}},
	})
	if got.Name != "developers/123/users/dev@example.com" || got.Email != "dev@example.com" {
		t.Fatalf("unexpected user info map: %+v", got)
	}
	if !got.Partial || got.GrantCount != 2 || len(got.DeveloperAccountPermissions) != 1 {
		t.Fatalf("unexpected user info map details: %+v", got)
	}
}
