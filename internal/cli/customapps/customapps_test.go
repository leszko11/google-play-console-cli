package customapps

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	playcustomapp "google.golang.org/api/playcustomapp/v1"
)

type fakeClient struct {
	created             gpc.CustomAppInfo
	err                 error
	capturedDeveloperID string
	capturedPayload     *playcustomapp.CustomApp
}

func (f *fakeClient) CreateCustomApp(_ context.Context, developerID string, app *playcustomapp.CustomApp) (gpc.CustomAppInfo, error) {
	f.capturedDeveloperID = developerID
	f.capturedPayload = app
	return f.created, f.err
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				ServiceAccountPath: "/tmp/sa.json",
				DeveloperID:        "developers/123456",
			},
		},
	}
}

func runCustomApps(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		}
	}
	if deps.Stdin == nil {
		deps.Stdin = bytes.NewBuffer(nil)
	}
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestCustomAppsCreateJSON(t *testing.T) {
	client := &fakeClient{
		created: gpc.CustomAppInfo{
			Title:        "Private App",
			LanguageCode: "en-US",
			PackageName:  "com.example.private",
			Organizations: []gpc.CustomAppOrganizationInfo{
				{ID: "org-1", Name: "Acme"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
		Stdin: strings.NewReader(`{
  "title": "Private App",
  "languageCode": "en-US",
  "organizations": [{"organizationId":"org-1","organizationName":"Acme"}]
}`),
	}

	out, err := runCustomApps(t, deps, "create", "--input", "-")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if client.capturedDeveloperID != "123456" {
		t.Fatalf("expected normalized developer id, got %q", client.capturedDeveloperID)
	}
	if client.capturedPayload == nil || client.capturedPayload.Title != "Private App" || client.capturedPayload.LanguageCode != "en-US" {
		t.Fatalf("unexpected payload: %+v", client.capturedPayload)
	}
	for _, want := range []string{`"developerId":"123456"`, `"status":"created"`, `"packageName":"com.example.private"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %s", want, out)
		}
	}
}

func TestCustomAppsCreateRequiresInput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runCustomApps(t, deps, "create")
	if err == nil || !strings.Contains(err.Error(), "--input is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCustomAppsCreateRejectsInvalidJSON(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		Stdin:      strings.NewReader("{"),
	}

	_, err := runCustomApps(t, deps, "create", "--input", "-")
	if err == nil || !strings.Contains(err.Error(), "invalid custom app JSON payload") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCustomAppsCreatePropagatesAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{err: context.DeadlineExceeded}, nil
		},
		Stdin: strings.NewReader(`{"title":"Private App","languageCode":"en-US"}`),
	}

	_, err := runCustomApps(t, deps, "create", "--input", "-")
	if err == nil || !strings.Contains(err.Error(), "failed to create custom app") {
		t.Fatalf("unexpected error: %v", err)
	}
}
