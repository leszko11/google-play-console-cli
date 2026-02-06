package config

import (
	"path/filepath"
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
