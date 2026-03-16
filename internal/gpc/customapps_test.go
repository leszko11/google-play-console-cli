package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	playcustomapp "google.golang.org/api/playcustomapp/v1"
)

func TestCustomAppsClient_RejectsMissingCredentials(t *testing.T) {
	if _, err := NewCustomAppsClient(context.Background(), CredentialInput{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestCustomAppsCreate_ValidateArgs(t *testing.T) {
	c := &CustomAppsClient{}

	if _, err := c.CreateCustomApp(context.Background(), "", &playcustomapp.CustomApp{Title: "My App"}); err == nil || !strings.Contains(err.Error(), "developer id is required") {
		t.Fatalf("unexpected developer id error: %v", err)
	}
	if _, err := c.CreateCustomApp(context.Background(), "abc", &playcustomapp.CustomApp{Title: "My App"}); err == nil || !strings.Contains(err.Error(), "developer id must be numeric") {
		t.Fatalf("unexpected developer id parse error: %v", err)
	}
	if _, err := c.CreateCustomApp(context.Background(), "123", nil); err == nil || !strings.Contains(err.Error(), "custom app payload is required") {
		t.Fatalf("unexpected payload error: %v", err)
	}
	if _, err := c.CreateCustomApp(context.Background(), "123", &playcustomapp.CustomApp{}); err == nil || !strings.Contains(err.Error(), "custom app title is required") {
		t.Fatalf("unexpected title validation error: %v", err)
	}
	if _, err := c.CreateCustomApp(context.Background(), "123", &playcustomapp.CustomApp{Title: "My App"}); err == nil || !strings.Contains(err.Error(), "custom app languageCode is required") {
		t.Fatalf("unexpected language validation error: %v", err)
	}
	if _, err := c.CreateCustomApp(context.Background(), "123", &playcustomapp.CustomApp{
		Title:        "My App",
		LanguageCode: "en-US",
		Organizations: []*playcustomapp.Organization{
			{},
		},
	}); err == nil || !strings.Contains(err.Error(), "organizationId is required") {
		t.Fatalf("unexpected organization validation error: %v", err)
	}
}

func TestCustomAppInfoFromResource(t *testing.T) {
	info := customAppInfoFromResource(&playcustomapp.CustomApp{
		Title:        "Private App",
		LanguageCode: "en-US",
		PackageName:  "com.example.private",
		Organizations: []*playcustomapp.Organization{
			{OrganizationId: "org-1", OrganizationName: "Acme"},
		},
	})

	if info.Title != "Private App" || info.LanguageCode != "en-US" || info.PackageName != "com.example.private" {
		t.Fatalf("unexpected custom app info: %+v", info)
	}
	if len(info.Organizations) != 1 || info.Organizations[0].ID != "org-1" || info.Organizations[0].Name != "Acme" {
		t.Fatalf("unexpected organizations: %+v", info.Organizations)
	}
}
