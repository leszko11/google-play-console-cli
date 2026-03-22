package shared

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
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

func TestWriteMarkdownTable(t *testing.T) {
	var out bytes.Buffer
	err := WriteMarkdownTable(&out, []string{"name", "notes"}, [][]string{{"alpha|beta", "line1\nline2"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "| name | notes |\n| --- | --- |\n| alpha\\|beta | line1<br>line2 |\n"
	if out.String() != want {
		t.Fatalf("unexpected markdown output %q, want %q", out.String(), want)
	}
}

func TestWriteMinimal(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{
			name:   "multiple values",
			values: []string{"com.example.app", "com.example.other"},
			want:   "com.example.app\ncom.example.other\n",
		},
		{
			name:   "single value",
			values: []string{"com.example.app"},
			want:   "com.example.app\n",
		},
		{
			name:   "empty",
			values: []string{},
			want:   "",
		},
		{
			name:   "nil",
			values: nil,
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := WriteMinimal(&out, tc.values)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.String() != tc.want {
				t.Fatalf("unexpected output %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestWriteMinimal_NilWriter(t *testing.T) {
	err := WriteMinimal(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteYAML_UsesJSONFieldNames(t *testing.T) {
	type sample struct {
		PackageName string `json:"packageName"`
		Status      string `json:"status"`
	}

	var out bytes.Buffer
	err := WriteYAML(&out, sample{PackageName: "com.example.app", Status: "ok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"packageName: com.example.app", "status: ok"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in yaml output: %s", want, got)
		}
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

func TestResolveCredentials_PathProfileSkipsKeychain(t *testing.T) {
	prevGlobals := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	prevLoad := resolveCredentialsLoadProfileCredential
	defer func() {
		boundGlobalFlags = prevGlobals
		resolveCredentialsShouldBypassKeychain = prevBypass
		resolveCredentialsLoadProfileCredential = prevLoad
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return false }
	resolveCredentialsLoadProfileCredential = func(string) ([]byte, error) {
		t.Fatal("did not expect keychain lookup for path-backed profile")
		return nil, nil
	}

	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/path.json", Storage: config.StoragePath},
		},
	}

	resolved, err := ResolveCredentials(cfg, func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Source != authresolver.SourceConfig {
		t.Fatalf("expected config source, got %q", resolved.Source)
	}
	if resolved.Input.ServiceAccountPath != "/tmp/path.json" {
		t.Fatalf("unexpected path: %q", resolved.Input.ServiceAccountPath)
	}
	if resolved.ProfileStorage != config.StoragePath {
		t.Fatalf("unexpected profile storage: %q", resolved.ProfileStorage)
	}
}

func TestResolveCredentials_KeychainProfileBypassFallsBackToPath(t *testing.T) {
	prevGlobals := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	prevLoad := resolveCredentialsLoadProfileCredential
	prevProbe := resolveCredentialsProbeKeychainAccess
	defer func() {
		boundGlobalFlags = prevGlobals
		resolveCredentialsShouldBypassKeychain = prevBypass
		resolveCredentialsLoadProfileCredential = prevLoad
		resolveCredentialsProbeKeychainAccess = prevProbe
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return true }
	resolveCredentialsProbeKeychainAccess = func(func(string) string) authresolver.KeychainProbeResult {
		t.Fatal("did not expect keychain probe while bypass is enabled")
		return authresolver.KeychainProbeResult{}
	}
	resolveCredentialsLoadProfileCredential = func(string) ([]byte, error) {
		t.Fatal("did not expect keychain lookup while bypass is enabled")
		return nil, nil
	}

	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/fallback.json", Storage: config.StorageKeychain},
		},
	}

	resolved, err := ResolveCredentials(cfg, func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Source != authresolver.SourceConfig {
		t.Fatalf("expected config fallback, got %q", resolved.Source)
	}
	if !slices.Contains(resolved.Warnings, "keychain bypassed via GPC_BYPASS_KEYCHAIN") {
		t.Fatalf("expected bypass warning, got %v", resolved.Warnings)
	}
}

func TestResolveCredentials_LegacyProfileBypassPrefersConfigPath(t *testing.T) {
	prevGlobals := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	prevLoad := resolveCredentialsLoadProfileCredential
	prevProbe := resolveCredentialsProbeKeychainAccess
	defer func() {
		boundGlobalFlags = prevGlobals
		resolveCredentialsShouldBypassKeychain = prevBypass
		resolveCredentialsLoadProfileCredential = prevLoad
		resolveCredentialsProbeKeychainAccess = prevProbe
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return true }
	resolveCredentialsProbeKeychainAccess = func(func(string) string) authresolver.KeychainProbeResult {
		t.Fatal("did not expect keychain probe while bypass is enabled")
		return authresolver.KeychainProbeResult{}
	}
	resolveCredentialsLoadProfileCredential = func(string) ([]byte, error) {
		t.Fatal("did not expect keychain lookup for legacy bypass path")
		return nil, nil
	}

	path := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(path, []byte(`{"type":"service_account"}`), 0o600); err != nil {
		t.Fatalf("write service account: %v", err)
	}

	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: path},
		},
	}

	resolved, err := ResolveCredentials(cfg, func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Source != authresolver.SourceConfig {
		t.Fatalf("expected config source, got %q", resolved.Source)
	}
	if resolved.ServiceAccountPath != path {
		t.Fatalf("unexpected path %q", resolved.ServiceAccountPath)
	}
}

func TestResolveCredentials_LegacyProfileBlockedKeychainFallsBackToPath(t *testing.T) {
	prevGlobals := boundGlobalFlags
	prevBypass := resolveCredentialsShouldBypassKeychain
	prevLoad := resolveCredentialsLoadProfileCredential
	prevProbe := resolveCredentialsProbeKeychainAccess
	defer func() {
		boundGlobalFlags = prevGlobals
		resolveCredentialsShouldBypassKeychain = prevBypass
		resolveCredentialsLoadProfileCredential = prevLoad
		resolveCredentialsProbeKeychainAccess = prevProbe
	}()

	boundGlobalFlags = &GlobalFlags{}
	resolveCredentialsShouldBypassKeychain = func(func(string) string) bool { return false }
	resolveCredentialsProbeKeychainAccess = func(func(string) string) authresolver.KeychainProbeResult {
		return authresolver.KeychainProbeResult{Blocked: true}
	}
	resolveCredentialsLoadProfileCredential = func(string) ([]byte, error) {
		t.Fatal("did not expect keychain lookup when probe says blocked")
		return nil, nil
	}

	path := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(path, []byte(`{"type":"service_account"}`), 0o600); err != nil {
		t.Fatalf("write service account: %v", err)
	}

	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: path},
		},
	}

	resolved, err := ResolveCredentials(cfg, func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Source != authresolver.SourceConfig {
		t.Fatalf("expected config source, got %q", resolved.Source)
	}
	if !slices.Contains(resolved.Warnings, "system keychain access appears blocked; using config/environment/flags") {
		t.Fatalf("expected blocked keychain warning, got %v", resolved.Warnings)
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
