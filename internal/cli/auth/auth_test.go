package auth

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	verifyErr error
}

func (f fakeClient) VerifyPackageAccess(_ context.Context, _ string) error {
	return f.verifyErr
}

func runAuth(t *testing.T, deps Deps, args ...string) string {
	t.Helper()

	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	if err := cmd.ParseAndRun(context.Background(), args); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	return out.String()
}

func TestAuthInitStoresProfileMetadata(t *testing.T) {
	var stored config.Config
	var saved bool

	deps := Deps{
		LoadConfig: func() (config.Config, error) { return stored, nil },
		SaveConfig: func(cfg config.Config) error {
			stored = cfg
			saved = true
			return nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (PackageVerifier, error) {
			return fakeClient{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 2, 6, 1, 2, 3, 0, time.UTC) },
	}

	runAuth(t, deps, "init", "--service-account", "/tmp/sa.json")

	if !saved {
		t.Fatal("expected config to be saved")
	}
	if stored.ActiveProfile != "default" {
		t.Fatalf("unexpected active profile: %q", stored.ActiveProfile)
	}
	if stored.Profiles["default"].ServiceAccountPath != "/tmp/sa.json" {
		t.Fatalf("unexpected path: %+v", stored.Profiles["default"])
	}
	if stored.Profiles["default"].DeveloperID != "" {
		t.Fatalf("expected empty developer id, got %+v", stored.Profiles["default"])
	}
}

func TestAuthInitStoresDeveloperID(t *testing.T) {
	var stored config.Config
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return stored, nil },
		SaveConfig: func(cfg config.Config) error {
			stored = cfg
			return nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (PackageVerifier, error) {
			return fakeClient{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 2, 6, 1, 2, 3, 0, time.UTC) },
	}

	runAuth(t, deps, "init", "--service-account", "/tmp/sa.json", "--developer-id", "developers/123")

	if stored.Profiles["default"].DeveloperID != "123" {
		t.Fatalf("unexpected developer id: %+v", stored.Profiles["default"])
	}
}

func TestAuthInitPromptsForDeveloperID(t *testing.T) {
	var stored config.Config
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return stored, nil },
		SaveConfig: func(cfg config.Config) error {
			stored = cfg
			return nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (PackageVerifier, error) {
			return fakeClient{}, nil
		},
		PromptID: func(_ io.Reader, _ io.Writer) (string, error) {
			return "1234567890123456789", nil
		},
		Now: func() time.Time { return time.Date(2026, 2, 6, 1, 2, 3, 0, time.UTC) },
	}

	runAuth(t, deps, "init", "--service-account", "/tmp/sa.json")

	if stored.Profiles["default"].DeveloperID != "1234567890123456789" {
		t.Fatalf("unexpected developer id: %+v", stored.Profiles["default"])
	}
}

func TestAuthStatusPrintsActiveProfileJSON(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/sa.json", DeveloperID: "1234567890123456789"},
				},
			}, nil
		},
	}

	out := runAuth(t, deps, "status")
	if !bytes.Contains([]byte(out), []byte(`"activeProfile":"default"`)) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte(`"developerId":"1234567890123456789"`)) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAuthLogoutRemovesActiveProfile(t *testing.T) {
	stored := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}

	deps := Deps{
		LoadConfig: func() (config.Config, error) { return stored, nil },
		SaveConfig: func(cfg config.Config) error {
			stored = cfg
			return nil
		},
	}

	runAuth(t, deps, "logout")

	if stored.ActiveProfile != "" {
		t.Fatalf("expected active profile cleared, got %q", stored.ActiveProfile)
	}
	if _, ok := stored.Profiles["default"]; ok {
		t.Fatalf("expected default profile removed, got %+v", stored.Profiles)
	}
}
