package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
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
	out, err := runAuthWithErr(t, deps, args...)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	return out
}

func runAuthWithErr(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(name string) string {
			if name == authresolver.EnvBypassKeychain {
				return "1"
			}
			return ""
		}
	}
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func runAuthIO(t *testing.T, deps Deps, args ...string) (string, string, error) {
	t.Helper()

	var out bytes.Buffer
	var errOut bytes.Buffer
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(name string) string {
			if name == authresolver.EnvBypassKeychain {
				return "1"
			}
			return ""
		}
	}
	deps.Stdout = &out
	deps.Stderr = &errOut
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), errOut.String(), err
}

func writeServiceAccountFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(path, []byte(`{"type":"service_account","project_id":"example"}`), 0o600); err != nil {
		t.Fatalf("write service account file: %v", err)
	}
	return path
}

func useTempConfigPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("GPC_CONFIG_PATH", path)
	return path
}

func TestAuthInitStoresProfileMetadata(t *testing.T) {
	useTempConfigPath(t)
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

	saPath := writeServiceAccountFile(t)
	runAuth(t, deps, "init", "--service-account", saPath)

	if !saved {
		t.Fatal("expected config to be saved")
	}
	if stored.ActiveProfile != "default" {
		t.Fatalf("unexpected active profile: %q", stored.ActiveProfile)
	}
	managedPath, err := config.ManagedServiceAccountPath("default")
	if err != nil {
		t.Fatalf("managed path: %v", err)
	}
	if stored.Profiles["default"].ServiceAccountPath != managedPath {
		t.Fatalf("unexpected path: %+v", stored.Profiles["default"])
	}
	if stored.Profiles["default"].Storage != config.StoragePath {
		t.Fatalf("unexpected storage: %+v", stored.Profiles["default"])
	}
	if stored.Profiles["default"].DeveloperID != "" {
		t.Fatalf("expected empty developer id, got %+v", stored.Profiles["default"])
	}
}

func TestAuthInitStoresDeveloperID(t *testing.T) {
	useTempConfigPath(t)
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

	saPath := writeServiceAccountFile(t)
	runAuth(t, deps, "init", "--service-account", saPath, "--developer-id", "developers/123")

	if stored.Profiles["default"].DeveloperID != "123" {
		t.Fatalf("unexpected developer id: %+v", stored.Profiles["default"])
	}
}

func TestAuthInitStoragePathKeepsExplicitPath(t *testing.T) {
	useTempConfigPath(t)
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

	saPath := writeServiceAccountFile(t)
	runAuth(t, deps, "init", "--service-account", saPath, "--storage", "path")

	if stored.Profiles["default"].ServiceAccountPath != saPath {
		t.Fatalf("unexpected explicit path storage: %+v", stored.Profiles["default"])
	}
	if stored.Profiles["default"].Storage != config.StoragePath {
		t.Fatalf("unexpected storage: %+v", stored.Profiles["default"])
	}
}

func TestAuthInitStorageKeychainStoresCredential(t *testing.T) {
	useTempConfigPath(t)
	var stored config.Config
	var storedProfile string
	var storedPayload []byte

	prevBypass := storageShouldBypassKeychain
	prevStore := storageStoreProfileCredential
	prevRemove := storageRemoveProfileCredential
	prevNotFound := storageIsCredentialNotFound
	prevUnavailable := storageIsKeyringUnavailable
	t.Cleanup(func() {
		storageShouldBypassKeychain = prevBypass
		storageStoreProfileCredential = prevStore
		storageRemoveProfileCredential = prevRemove
		storageIsCredentialNotFound = prevNotFound
		storageIsKeyringUnavailable = prevUnavailable
	})

	storageShouldBypassKeychain = func(func(string) string) bool { return false }
	storageStoreProfileCredential = func(profile string, payload []byte) error {
		storedProfile = profile
		storedPayload = append([]byte(nil), payload...)
		return nil
	}
	storageRemoveProfileCredential = func(string) error { return authresolver.ErrCredentialNotFound }
	storageIsCredentialNotFound = func(err error) bool { return err == authresolver.ErrCredentialNotFound }
	storageIsKeyringUnavailable = func(error) bool { return false }

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

	saPath := writeServiceAccountFile(t)
	runAuth(t, deps, "init", "--service-account", saPath, "--storage", "keychain")

	if storedProfile != "default" || !bytes.Contains(storedPayload, []byte(`"type":"service_account"`)) {
		t.Fatalf("expected keychain store call, got profile=%q payload=%q", storedProfile, string(storedPayload))
	}
	if stored.Profiles["default"].Storage != config.StorageKeychain {
		t.Fatalf("unexpected storage: %+v", stored.Profiles["default"])
	}
	if stored.Profiles["default"].ServiceAccountPath != saPath {
		t.Fatalf("unexpected persisted path: %+v", stored.Profiles["default"])
	}
	managedPath, err := config.ManagedServiceAccountPath("default")
	if err != nil {
		t.Fatalf("managed path: %v", err)
	}
	if _, err := os.Stat(managedPath); !os.IsNotExist(err) {
		t.Fatalf("expected no managed secret file, got err=%v", err)
	}
}

func TestAuthInitPromptsForDeveloperID(t *testing.T) {
	useTempConfigPath(t)
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

	saPath := writeServiceAccountFile(t)
	runAuth(t, deps, "init", "--service-account", saPath, "--prompt-developer-id")

	if stored.Profiles["default"].DeveloperID != "1234567890123456789" {
		t.Fatalf("unexpected developer id: %+v", stored.Profiles["default"])
	}
}

func TestAuthInitDoesNotPromptByDefault(t *testing.T) {
	useTempConfigPath(t)
	var stored config.Config
	promptCalled := false
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
			promptCalled = true
			return "9023817352750250026", nil
		},
		Now: func() time.Time { return time.Date(2026, 2, 6, 1, 2, 3, 0, time.UTC) },
	}

	saPath := writeServiceAccountFile(t)
	runAuth(t, deps, "init", "--service-account", saPath)
	if promptCalled {
		t.Fatal("did not expect developer-id prompt by default")
	}
	if stored.Profiles["default"].DeveloperID != "" {
		t.Fatalf("expected empty developer id, got %+v", stored.Profiles["default"])
	}
}

func TestAuthStatusPrintsActiveProfileJSON(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			saPath := writeServiceAccountFile(t)
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: saPath, DeveloperID: "1234567890123456789"},
				},
			}, nil
		},
		LookupEnv: func(name string) string {
			if name == authresolver.EnvBypassKeychain {
				return "1"
			}
			return ""
		},
	}

	out := runAuth(t, deps, "status", "--output", "json")
	if !bytes.Contains([]byte(out), []byte(`"activeProfile":"default"`)) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte(`"developerId":"1234567890123456789"`)) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAuthExplainPrintsJSON(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			saPath := writeServiceAccountFile(t)
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {
						ServiceAccountPath: saPath,
						Storage:            config.StoragePath,
						DeveloperID:        "1234567890123456789",
					},
				},
			}, nil
		},
		LookupEnv: func(name string) string {
			switch name {
			case authresolver.EnvBypassKeychain:
				return "1"
			case shared.EnvStrictAuth:
				return "1"
			default:
				return ""
			}
		},
	}

	out := runAuth(t, deps, "explain", "--output", "json")
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("unmarshal explain output: %v", err)
	}
	if payload["finalSource"] != string(authresolver.SourceConfig) {
		t.Fatalf("unexpected finalSource: %#v", payload["finalSource"])
	}
	if payload["ciRecommendation"] == "" {
		t.Fatalf("expected ciRecommendation, got %#v", payload["ciRecommendation"])
	}
	status, ok := payload["status"].(map[string]any)
	if !ok {
		t.Fatalf("expected status object, got %#v", payload["status"])
	}
	if status["health"] == "" {
		t.Fatalf("expected status health, got %#v", status["health"])
	}
}

func TestAuthExplainReportsStrictConflictRisk(t *testing.T) {
	saPath := writeServiceAccountFile(t)
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {
						ServiceAccountPath: saPath,
						Storage:            config.StoragePath,
					},
				},
			}, nil
		},
		LookupEnv: func(name string) string {
			switch name {
			case shared.EnvServiceAccountPath:
				return "/tmp/from-env.json"
			case authresolver.EnvBypassKeychain:
				return "1"
			case shared.EnvStrictAuth:
				return "1"
			default:
				return ""
			}
		},
	}

	out := runAuth(t, deps, "explain", "--output", "json")
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("unmarshal explain output: %v", err)
	}
	strictAuth, ok := payload["strictAuth"].(map[string]any)
	if !ok {
		t.Fatalf("expected strictAuth object, got %#v", payload["strictAuth"])
	}
	if strictAuth["wouldFail"] != true {
		t.Fatalf("expected strictAuth.wouldFail=true, got %#v", strictAuth["wouldFail"])
	}
}

func TestAuthExplainPrintsTable(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			saPath := writeServiceAccountFile(t)
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: saPath, Storage: config.StoragePath},
				},
			}, nil
		},
	}

	out := runAuth(t, deps, "explain", "--output", "table")
	for _, want := range []string{"FIELD\tVALUE", "source\tconfig", "ciRecommendation\t"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in explain table output, got %q", want, out)
		}
	}
}

func TestAuthStatus_TableOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			saPath := writeServiceAccountFile(t)
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: saPath},
				},
			}, nil
		},
		LookupEnv: func(name string) string {
			if name == authresolver.EnvBypassKeychain {
				return "1"
			}
			return ""
		},
	}

	out := runAuth(t, deps, "status", "--output", "table")
	if !bytes.Contains([]byte(out), []byte("FIELD\tVALUE")) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("authenticated\ttrue")) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAuthStatus_RejectsCSVOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			saPath := writeServiceAccountFile(t)
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: saPath},
				},
			}, nil
		},
	}

	_, _, err := runAuthIO(t, deps, "status", "--output", "csv")
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthStatus_UnauthenticatedWhenPathMissing(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/definitely/missing/path.json"},
				},
			}, nil
		},
		LookupEnv: func(name string) string {
			if name == authresolver.EnvBypassKeychain {
				return "1"
			}
			return ""
		},
	}

	out := runAuth(t, deps, "status", "--output", "json")
	if !bytes.Contains([]byte(out), []byte(`"authenticated":false`)) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAuthProfilesList_CSVOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "work",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/default.json", Storage: config.StoragePath},
					"work":    {ServiceAccountPath: "/tmp/work.json", Storage: config.StoragePath},
				},
			}, nil
		},
	}

	out, errOut, err := runAuthIO(t, deps, "profiles", "list", "--output", "csv")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	want := "profile,active,storage,serviceAccountPath\ndefault,false,path,/tmp/default.json\nwork,true,path,/tmp/work.json\n"
	if out != want {
		t.Fatalf("unexpected csv output %q, want %q", out, want)
	}
	if !strings.Contains(errOut, "warning: keychain bypassed via GPC_BYPASS_KEYCHAIN") {
		t.Fatalf("expected keychain warning on stderr, got %q", errOut)
	}
}

func TestAuthProfilesList_MarkdownOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "work",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/default.json", Storage: config.StoragePath},
					"work":    {ServiceAccountPath: "/tmp/work.json", Storage: config.StoragePath},
				},
			}, nil
		},
	}

	out, errOut, err := runAuthIO(t, deps, "profiles", "list", "--output", "markdown")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{
		"| profile | active | storage | serviceAccountPath |",
		"| default | false | path | /tmp/default.json |",
		"| work | true | path | /tmp/work.json |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
	if !strings.Contains(errOut, "warning: keychain bypassed via GPC_BYPASS_KEYCHAIN") {
		t.Fatalf("expected keychain warning on stderr, got %q", errOut)
	}
}

func TestAuthProfilesList_YAMLOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "work",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/default.json", Storage: config.StoragePath},
					"work":    {ServiceAccountPath: "/tmp/work.json", Storage: config.StoragePath},
				},
			}, nil
		},
	}

	out, errOut, err := runAuthIO(t, deps, "profiles", "list", "--output", "yaml")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{"selectedProfile: work", "profile: default", "profile: work"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
	if !strings.Contains(errOut, "warning: keychain bypassed via GPC_BYPASS_KEYCHAIN") {
		t.Fatalf("expected keychain warning on stderr, got %q", errOut)
	}
}

func TestAuthStatus_UnauthenticatedWhenPathInvalidJSON(t *testing.T) {
	invalidPath := filepath.Join(t.TempDir(), "invalid-service-account.json")
	if err := os.WriteFile(invalidPath, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("write invalid file: %v", err)
	}

	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: invalidPath},
				},
			}, nil
		},
		LookupEnv: func(name string) string {
			if name == authresolver.EnvBypassKeychain {
				return "1"
			}
			return ""
		},
	}

	out := runAuth(t, deps, "status", "--output", "json")
	if !bytes.Contains([]byte(out), []byte(`"authenticated":false`)) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAuthStatus_InvalidOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
	}

	_, err := runAuthWithErr(t, deps, "status", "--output", "xml")
	if err == nil {
		t.Fatal("expected error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("unsupported output format")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthStatus_MarkdownOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			saPath := writeServiceAccountFile(t)
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: saPath, DeveloperID: "1234567890123456789"},
				},
			}, nil
		},
		LookupEnv: func(name string) string {
			if name == authresolver.EnvBypassKeychain {
				return "1"
			}
			return ""
		},
	}

	out := runAuth(t, deps, "status", "--output", "markdown")
	for _, want := range []string{
		"| field | value |",
		"| authenticated | true |",
		"| activeProfile | default |",
		"| developerId | 1234567890123456789 |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestAuthStatus_YAMLOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			saPath := writeServiceAccountFile(t)
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: saPath, DeveloperID: "1234567890123456789"},
				},
			}, nil
		},
		LookupEnv: func(name string) string {
			if name == authresolver.EnvBypassKeychain {
				return "1"
			}
			return ""
		},
	}

	out := runAuth(t, deps, "status", "--output", "yaml")
	for _, want := range []string{"activeProfile: default", "authenticated: true", "developerId: \"1234567890123456789\""} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestAuthLogoutRemovesActiveProfile(t *testing.T) {
	stored := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: writeServiceAccountFile(t)},
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

func TestAuthInitRejectsBlankProfile(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return config.Config{}, nil },
		SaveConfig: func(config.Config) error { return nil },
		NewClient: func(context.Context, gpc.CredentialInput) (PackageVerifier, error) {
			return fakeClient{}, nil
		},
	}
	saPath := writeServiceAccountFile(t)
	_, err := runAuthWithErr(t, deps, "init", "--service-account", saPath, "--profile", "   ")
	if err == nil {
		t.Fatal("expected error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("--profile is required")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthSwitchRejectsBlankProfile(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return config.Config{}, nil },
		SaveConfig: func(config.Config) error { return nil },
	}
	_, err := runAuthWithErr(t, deps, "switch", "--profile", "   ")
	if err == nil {
		t.Fatal("expected error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("--profile is required")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthProfilesListJSON(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: writeServiceAccountFile(t)},
					"ci":      {ServiceAccountPath: writeServiceAccountFile(t)},
				},
			}, nil
		},
	}

	out := runAuth(t, deps, "profiles", "list", "--output", "json")
	var payload struct {
		Profiles []struct {
			Profile string `json:"profile"`
			Active  bool   `json:"active"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(payload.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(payload.Profiles))
	}
}

func TestAuthLogoutAll(t *testing.T) {
	stored := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: writeServiceAccountFile(t)},
			"ci":      {ServiceAccountPath: writeServiceAccountFile(t)},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return stored, nil },
		SaveConfig: func(cfg config.Config) error {
			stored = cfg
			return nil
		},
	}

	runAuth(t, deps, "logout", "--all")
	if stored.ActiveProfile != "" {
		t.Fatalf("expected empty active profile, got %q", stored.ActiveProfile)
	}
	if len(stored.Profiles) != 0 {
		t.Fatalf("expected empty profiles, got %#v", stored.Profiles)
	}
}
