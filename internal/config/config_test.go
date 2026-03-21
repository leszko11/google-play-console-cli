package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveAndLoad_WithConfigPathOverride(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("GPC_CONFIG_PATH", cfgPath)

	want := Config{
		ActiveProfile: "default",
		Packages:      []string{"com.example.app"},
		Profiles: map[string]Profile{
			"default": {
				ServiceAccountPath: "/tmp/sa.json",
				LastValidatedAt:    "2026-02-06T00:00:00Z",
			},
		},
	}

	if err := Save(want); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if got.ActiveProfile != want.ActiveProfile {
		t.Fatalf("active profile mismatch: got %q want %q", got.ActiveProfile, want.ActiveProfile)
	}

	if len(got.Packages) != 1 || got.Packages[0] != "com.example.app" {
		t.Fatalf("packages mismatch: %+v", got.Packages)
	}

	if got.Profiles["default"].ServiceAccountPath != "/tmp/sa.json" {
		t.Fatalf("profile mismatch: %+v", got.Profiles["default"])
	}
}

func TestLoad_EmptyConfigFileReturnsZeroConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("GPC_CONFIG_PATH", cfgPath)

	if err := os.WriteFile(cfgPath, nil, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if got.ActiveProfile != "" {
		t.Fatalf("expected empty active profile, got %q", got.ActiveProfile)
	}
	if len(got.Packages) != 0 {
		t.Fatalf("expected no packages, got %v", got.Packages)
	}
}

func TestWriteManagedServiceAccount_UsesConfigBaseDir(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "nested", "config.json")
	t.Setenv("GPC_CONFIG_PATH", cfgPath)

	path, err := WriteManagedServiceAccount("work", []byte(`{"type":"service_account"}`))
	if err != nil {
		t.Fatalf("write managed service account: %v", err)
	}

	wantPath := filepath.Join(filepath.Dir(cfgPath), "credentials", "work.json")
	if path != wantPath {
		t.Fatalf("unexpected managed path: got %q want %q", path, wantPath)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat managed service account: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected file mode: %o", info.Mode().Perm())
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read managed service account: %v", err)
	}
	if string(payload) != "{\"type\":\"service_account\"}\n" {
		t.Fatalf("unexpected managed payload: %q", string(payload))
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat managed credentials dir: %v", err)
	}
	if runtime.GOOS != "windows" && dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("unexpected dir mode: %o", dirInfo.Mode().Perm())
	}
}
