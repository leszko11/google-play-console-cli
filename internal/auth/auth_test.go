package auth

import "testing"

func TestResolveCredentialSource_StrictModeConflict(t *testing.T) {
	_, err := ResolveCredentialSource(Input{
		FlagPath: "/tmp/a.json",
		EnvPath:  "/tmp/b.json",
		Strict:   true,
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestResolveCredentialSource_PreferenceOrder(t *testing.T) {
	got, err := ResolveCredentialSource(Input{
		FlagPath:   "/tmp/flag.json",
		EnvPath:    "/tmp/env.json",
		ConfigPath: "/tmp/config.json",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Kind != SourceFlag || got.Path != "/tmp/flag.json" {
		t.Fatalf("unexpected source: %+v", got)
	}
}

func TestResolveCredentialSource_EnvOverConfig(t *testing.T) {
	got, err := ResolveCredentialSource(Input{
		EnvPath:    "/tmp/env.json",
		ConfigPath: "/tmp/config.json",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Kind != SourceEnv || got.Path != "/tmp/env.json" {
		t.Fatalf("unexpected source: %+v", got)
	}
}
