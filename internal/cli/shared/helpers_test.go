package shared

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
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
	prevBypass := resolveCredentialsShouldBypassKeychain
	defer func() {
		boundGlobalFlags = prev
		resolveCredentialsShouldBypassKeychain = prevBypass
	}()

	boundGlobalFlags = &GlobalFlags{ServiceAccount: "/tmp/flag.json"}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return true }

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
	if !errors.Is(err, gpc.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
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
		{name: "numeric", input: "1234567890123456789", want: "1234567890123456789"},
		{name: "prefixed", input: "developers/1234567890123456789", want: "1234567890123456789"},
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
			"default": {DeveloperID: "1234567890123456789"},
		},
	}

	got, err := ResolveDeveloperID("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1234567890123456789" {
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

func TestWriteDelimited(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   string
	}{
		{
			name:   "csv",
			format: "csv",
			want:   "packageName,status\ncom.example.app,ok\n",
		},
		{
			name:   "tsv",
			format: "tsv",
			want:   "packageName\tstatus\ncom.example.app\tok\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := WriteDelimited(&out, tc.format, []string{"packageName", "status"}, [][]string{{"com.example.app", "ok"}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.String() != tc.want {
				t.Fatalf("unexpected output %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestWriteDelimitedRejectsUnsupportedFormat(t *testing.T) {
	err := WriteDelimited(&bytes.Buffer{}, "json", []string{"packageName"}, [][]string{{"com.example.app"}})
	if err == nil || !strings.Contains(err.Error(), "unsupported delimited output format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveProfileName_GlobalOverride(t *testing.T) {
	prev := boundGlobalFlags
	defer func() { boundGlobalFlags = prev }()

	boundGlobalFlags = &GlobalFlags{Profile: "override"}
	cfg := config.Config{ActiveProfile: "default"}
	if got := ResolveProfileName(cfg); got != "override" {
		t.Fatalf("expected override profile, got %q", got)
	}
}

func TestResolveStrictAuth_FromEnv(t *testing.T) {
	prev := boundGlobalFlags
	defer func() { boundGlobalFlags = prev }()

	boundGlobalFlags = &GlobalFlags{}
	if !ResolveStrictAuth(func(name string) string {
		if name == EnvStrictAuth {
			return "true"
		}
		return ""
	}) {
		t.Fatal("expected strict auth from env")
	}
}

func TestResolveCredentials_StrictAuthConflict(t *testing.T) {
	prev := boundGlobalFlags
	defer func() { boundGlobalFlags = prev }()

	boundGlobalFlags = &GlobalFlags{
		ServiceAccount: "/tmp/flag.json",
		StrictAuth:     true,
	}
	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/config.json"},
		},
	}
	_, err := ResolveCredentials(cfg, func(name string) string {
		switch name {
		case EnvServiceAccountPath:
			return "/tmp/env.json"
		case authresolver.EnvBypassKeychain:
			return "1"
		default:
			return ""
		}
	})
	if err == nil {
		t.Fatal("expected strict conflict error")
	}
	if !strings.Contains(err.Error(), "multiple credential sources found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveCredentials_UsesProfileOverride(t *testing.T) {
	prev := boundGlobalFlags
	defer func() { boundGlobalFlags = prev }()

	boundGlobalFlags = &GlobalFlags{Profile: "ci"}
	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/default.json"},
			"ci":      {ServiceAccountPath: "/tmp/ci.json"},
		},
	}
	resolved, err := ResolveCredentials(cfg, func(name string) string {
		if name == authresolver.EnvBypassKeychain {
			return "1"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Profile != "ci" {
		t.Fatalf("expected profile ci, got %q", resolved.Profile)
	}
	if resolved.Input.ServiceAccountPath != "/tmp/ci.json" {
		t.Fatalf("expected ci path, got %q", resolved.Input.ServiceAccountPath)
	}
}

func TestResolveCredentials_KeychainSource(t *testing.T) {
	prevGlobals := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	prevLoad := resolveCredentialsLoadProfileCredential
	prevNotFound := resolveCredentialsIsCredentialNotFound
	prevUnavailable := resolveCredentialsIsKeyringUnavailable
	defer func() {
		boundGlobalFlags = prevGlobals
		resolveCredentialsShouldBypassKeychain = prevBypass
		resolveCredentialsLoadProfileCredential = prevLoad
		resolveCredentialsIsCredentialNotFound = prevNotFound
		resolveCredentialsIsKeyringUnavailable = prevUnavailable
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return false }
	resolveCredentialsLoadProfileCredential = func(profile string) ([]byte, error) {
		if profile != "default" {
			t.Fatalf("unexpected profile: %q", profile)
		}
		return []byte(`{"type":"service_account","project_id":"example"}`), nil
	}
	resolveCredentialsIsCredentialNotFound = func(error) bool { return false }
	resolveCredentialsIsKeyringUnavailable = func(error) bool { return false }

	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/default.json"},
		},
	}
	resolved, err := ResolveCredentials(cfg, func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Source != authresolver.SourceKeychain {
		t.Fatalf("expected keychain source, got %q", resolved.Source)
	}
	if len(resolved.Input.ServiceAccountJSON) == 0 {
		t.Fatal("expected keychain json payload")
	}
	if !CredentialLocallyValid(resolved.Input) {
		t.Fatal("expected locally valid keychain credential")
	}
}

func TestCredentialLocallyValid_PathFailures(t *testing.T) {
	if CredentialLocallyValid(gpc.CredentialInput{ServiceAccountPath: "/definitely/missing/path.json"}) {
		t.Fatal("expected missing path to be invalid")
	}

	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("write invalid file: %v", err)
	}
	if CredentialLocallyValid(gpc.CredentialInput{ServiceAccountPath: invalidPath}) {
		t.Fatal("expected invalid json path to be invalid")
	}
}
