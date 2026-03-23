package shared

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
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
	prevProbe := authStatusProbeKeychainAccess
	defer func() {
		boundGlobalFlags = prevFlags
		resolveCredentialsShouldBypassKeychain = prevBypass
		authStatusProbeKeychainAccess = prevProbe
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return true }
	authStatusProbeKeychainAccess = func(func(string) string) authresolver.KeychainProbeResult {
		return authresolver.KeychainProbeResult{}
	}

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
	if status.Health != string(AuthHealthReady) {
		t.Fatalf("expected ready auth health, got %q", status.Health)
	}
}

func TestBuildAuthStatusSnapshot_LegacyMixedSources(t *testing.T) {
	prevFlags := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	prevProbe := authStatusProbeKeychainAccess
	prevKeychainAvailable := authStatusKeychainAvailable
	prevLoad := resolveCredentialsLoadProfileCredential
	prevNotFound := resolveCredentialsIsCredentialNotFound
	prevUnavailable := resolveCredentialsIsKeyringUnavailable
	defer func() {
		boundGlobalFlags = prevFlags
		resolveCredentialsShouldBypassKeychain = prevBypass
		authStatusProbeKeychainAccess = prevProbe
		authStatusKeychainAvailable = prevKeychainAvailable
		resolveCredentialsLoadProfileCredential = prevLoad
		resolveCredentialsIsCredentialNotFound = prevNotFound
		resolveCredentialsIsKeyringUnavailable = prevUnavailable
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return false }
	authStatusProbeKeychainAccess = func(func(string) string) authresolver.KeychainProbeResult {
		return authresolver.KeychainProbeResult{Available: true}
	}
	authStatusKeychainAvailable = func(func(string) string) (bool, error) { return true, nil }
	resolveCredentialsLoadProfileCredential = func(string) ([]byte, error) {
		return []byte(`{"type":"service_account","project_id":"example"}`), nil
	}
	resolveCredentialsIsCredentialNotFound = func(error) bool { return false }
	resolveCredentialsIsKeyringUnavailable = func(error) bool { return false }

	serviceAccountPath := writeStatusFixtureServiceAccount(t)
	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: serviceAccountPath},
		},
	}

	status := BuildAuthStatusSnapshot(cfg, func(string) string { return "" })
	if status.Health != string(AuthHealthMixedSources) {
		t.Fatalf("expected mixed_sources health, got %q", status.Health)
	}
	if status.FixCommand == "" {
		t.Fatal("expected fix command")
	}
}

func TestBuildAuthStatusSnapshot_StalePath(t *testing.T) {
	prevFlags := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	prevProbe := authStatusProbeKeychainAccess
	defer func() {
		boundGlobalFlags = prevFlags
		resolveCredentialsShouldBypassKeychain = prevBypass
		authStatusProbeKeychainAccess = prevProbe
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return true }
	authStatusProbeKeychainAccess = func(func(string) string) authresolver.KeychainProbeResult {
		return authresolver.KeychainProbeResult{}
	}

	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				Storage:            config.StoragePath,
				ServiceAccountPath: filepath.Join(t.TempDir(), "missing.json"),
			},
		},
	}

	status := BuildAuthStatusSnapshot(cfg, func(string) string { return "" })
	if status.Health != string(AuthHealthStalePath) {
		t.Fatalf("expected stale_path health, got %q", status.Health)
	}
}

func TestBuildAuthStatusSnapshot_KeychainBlocked(t *testing.T) {
	prevFlags := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	prevProbe := authStatusProbeKeychainAccess
	defer func() {
		boundGlobalFlags = prevFlags
		resolveCredentialsShouldBypassKeychain = prevBypass
		authStatusProbeKeychainAccess = prevProbe
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return false }
	authStatusProbeKeychainAccess = func(func(string) string) authresolver.KeychainProbeResult {
		return authresolver.KeychainProbeResult{Blocked: true}
	}

	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {Storage: config.StorageKeychain},
		},
	}

	status := BuildAuthStatusSnapshot(cfg, func(string) string { return "" })
	if status.Health != string(AuthHealthKeychainBlocked) {
		t.Fatalf("expected keychain_blocked health, got %q", status.Health)
	}
	if status.DiagnosticCommand == "" {
		t.Fatal("expected diagnostic command")
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
