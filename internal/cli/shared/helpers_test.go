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
	prevEnv := resolveOutputLookupEnv
	prevTTY := resolveOutputDetectTTY
	defer func() {
		boundGlobalFlags = prev
		resolveOutputLookupEnv = prevEnv
		resolveOutputDetectTTY = prevTTY
	}()

	boundGlobalFlags = &GlobalFlags{Output: "markdown"}
	resolveOutputLookupEnv = func(string) string { return "" }
	resolveOutputDetectTTY = func() bool { return false }
	if got := ResolveOutput(""); got != "markdown" {
		t.Fatalf("expected markdown, got %q", got)
	}
}

func TestResolveOutput_EnvFallback(t *testing.T) {
	prev := boundGlobalFlags
	prevEnv := resolveOutputLookupEnv
	prevTTY := resolveOutputDetectTTY
	defer func() {
		boundGlobalFlags = prev
		resolveOutputLookupEnv = prevEnv
		resolveOutputDetectTTY = prevTTY
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveOutputLookupEnv = func(name string) string {
		if name == EnvDefaultOutput {
			return "table"
		}
		return ""
	}
	resolveOutputDetectTTY = func() bool { return false }

	if got := ResolveOutput(""); got != "table" {
		t.Fatalf("expected table from env, got %q", got)
	}
}

func TestResolveOutput_TTYFallback(t *testing.T) {
	prev := boundGlobalFlags
	prevEnv := resolveOutputLookupEnv
	prevTTY := resolveOutputDetectTTY
	defer func() {
		boundGlobalFlags = prev
		resolveOutputLookupEnv = prevEnv
		resolveOutputDetectTTY = prevTTY
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveOutputLookupEnv = func(string) string { return "" }
	resolveOutputDetectTTY = func() bool { return true }

	if got := ResolveOutput(""); got != "table" {
		t.Fatalf("expected table from tty fallback, got %q", got)
	}
}

func TestResolveOutput_NonTTYFallback(t *testing.T) {
	prev := boundGlobalFlags
	prevEnv := resolveOutputLookupEnv
	prevTTY := resolveOutputDetectTTY
	defer func() {
		boundGlobalFlags = prev
		resolveOutputLookupEnv = prevEnv
		resolveOutputDetectTTY = prevTTY
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveOutputLookupEnv = func(string) string { return "" }
	resolveOutputDetectTTY = func() bool { return false }

	if got := ResolveOutput(""); got != "json" {
		t.Fatalf("expected json fallback, got %q", got)
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

func TestNormalizeDeveloperID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "numeric", input: "9023817352750250026", want: "9023817352750250026"},
		{name: "prefixed", input: "developers/9023817352750250026", want: "9023817352750250026"},
		{name: "empty", input: "", want: ""},
		{name: "invalid", input: "developers/not-a-number", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeDeveloperID(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestResolveDeveloperID(t *testing.T) {
	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {DeveloperID: "9023817352750250026"},
		},
	}

	got, err := ResolveDeveloperID("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "9023817352750250026" {
		t.Fatalf("expected configured developer id, got %q", got)
	}
}

func TestResolveDeveloperID_RequiresValue(t *testing.T) {
	_, err := ResolveDeveloperID("", config.Config{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--developer-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
