package shared

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
)

func TestBuildAuthStatusSnapshot_PathProfileUsesPathBackend(t *testing.T) {
	prevFlags := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	prevKeychainAvailable := authStatusKeychainAvailable
	defer func() {
		boundGlobalFlags = prevFlags
		resolveCredentialsShouldBypassKeychain = prevBypass
		authStatusKeychainAvailable = prevKeychainAvailable
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return false }
	authStatusKeychainAvailable = func(func(string) string) (bool, error) { return true, nil }

	serviceAccountPath := writeStatusFixtureServiceAccount(t)
	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				Storage:            config.StoragePath,
				ServiceAccountPath: serviceAccountPath,
			},
		},
	}

	status := BuildAuthStatusSnapshot(cfg, func(string) string { return "" })
	if status.StorageBackend != config.StoragePath {
		t.Fatalf("expected path backend, got %q", status.StorageBackend)
	}
	if status.ProfileStorage != config.StoragePath {
		t.Fatalf("expected path profile storage, got %q", status.ProfileStorage)
	}
	if status.Source != "config" {
		t.Fatalf("expected config source, got %q", status.Source)
	}
	for _, warning := range status.Warnings {
		if strings.Contains(warning, "keychain") {
			t.Fatalf("did not expect keychain warning for path profile, got %q", warning)
		}
	}
}

func TestBuildAuthStatusSnapshot_KeychainProfileBypassUsesPathBackend(t *testing.T) {
	prevFlags := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	defer func() {
		boundGlobalFlags = prevFlags
		resolveCredentialsShouldBypassKeychain = prevBypass
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return true }

	serviceAccountPath := writeStatusFixtureServiceAccount(t)
	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				Storage:            config.StorageKeychain,
				ServiceAccountPath: serviceAccountPath,
			},
		},
	}

	status := BuildAuthStatusSnapshot(cfg, func(string) string { return "" })
	if status.StorageBackend != config.StoragePath {
		t.Fatalf("expected path backend during bypass, got %q", status.StorageBackend)
	}
	if status.ProfileStorage != config.StorageKeychain {
		t.Fatalf("expected keychain profile storage, got %q", status.ProfileStorage)
	}
	if status.Source != "config" {
		t.Fatalf("expected config source during bypass, got %q", status.Source)
	}
	if !slices.Contains(status.Warnings, "keychain bypassed via GPC_BYPASS_KEYCHAIN") {
		t.Fatalf("expected bypass warning, got %v", status.Warnings)
	}
}

func writeStatusFixtureServiceAccount(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "service-account.json")
	if err := os.WriteFile(path, []byte(`{"type":"service_account","project_id":"demo"}`), 0o600); err != nil {
		t.Fatalf("write service account fixture: %v", err)
	}
	return path
}
