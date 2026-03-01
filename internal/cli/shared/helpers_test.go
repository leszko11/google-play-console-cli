package shared

import (
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
)

func TestResolvePackageName_GlobalFallback(t *testing.T) {
	prev := boundGlobalFlags
	defer func() { boundGlobalFlags = prev }()

	boundGlobalFlags = &GlobalFlags{PackageName: "com.example.global"}

	got, err := ResolvePackageName("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "com.example.global" {
		t.Fatalf("expected global package, got %q", got)
	}
}

func TestResolvePackageName_Required(t *testing.T) {
	prev := boundGlobalFlags
	defer func() { boundGlobalFlags = prev }()

	boundGlobalFlags = &GlobalFlags{}

	_, err := ResolvePackageName("")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveOutput_GlobalFallback(t *testing.T) {
	prev := boundGlobalFlags
	defer func() { boundGlobalFlags = prev }()

	boundGlobalFlags = &GlobalFlags{Output: "markdown"}
	if got := ResolveOutput(""); got != "markdown" {
		t.Fatalf("expected markdown, got %q", got)
	}
}

func TestResolveServiceAccountPath_Precedence(t *testing.T) {
	prev := boundGlobalFlags
	defer func() { boundGlobalFlags = prev }()

	boundGlobalFlags = &GlobalFlags{ServiceAccount: "/tmp/flag.json"}

	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/config.json"},
		},
	}

	got, err := ResolveServiceAccountPath(cfg, func(string) string { return "/tmp/env.json" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/tmp/flag.json" {
		t.Fatalf("expected flag precedence, got %q", got)
	}
}

func TestResolveServiceAccountPath_RequiresSource(t *testing.T) {
	prev := boundGlobalFlags
	defer func() { boundGlobalFlags = prev }()

	boundGlobalFlags = &GlobalFlags{}
	_, err := ResolveServiceAccountPath(config.Config{}, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no service account configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}
