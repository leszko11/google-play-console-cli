package release

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func seedInitAuth(t *testing.T, deps *Deps, root string) {
	t.Helper()
	saPath := filepath.Join(root, "service-account.json")
	if err := os.WriteFile(saPath, []byte(`{"type":"service_account"}`), 0o600); err != nil {
		t.Fatalf("write service account: %v", err)
	}
	deps.LoadConfig = func() (config.Config, error) {
		return config.Config{
			ActiveProfile: "default",
			Profiles: map[string]config.Profile{
				"default": {ServiceAccountPath: saPath, Storage: config.StoragePath},
			},
		}, nil
	}
	deps.LookupEnv = func(key string) string {
		if key == "GPC_BYPASS_KEYCHAIN" {
			return "1"
		}
		return ""
	}
}

func runInitCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), append([]string{"init"}, args...))
	return out.String(), err
}

func TestReleaseInitManualRequired(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	client := &fakeReleaseClient{verifyErr: mustPackageNotFoundErr()}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)
	deps.RunBootstrap = func(context.Context, []string) error {
		t.Fatal("bootstrap should not run for uninitialized packages")
		return nil
	}

	out, err := runInitCommand(t, deps, "--package-name", "com.example.app", "--dir", "./play")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"manual_required"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	for _, path := range []string{
		filepath.Join(root, "play", "release.yaml"),
		filepath.Join(root, "play", "MANUAL_FIRST_UPLOAD.md"),
		filepath.Join(root, ".gpc.yaml"),
		filepath.Join(root, ".gpc", "workflow.yml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", path, err)
		}
	}
}

func TestReleaseInitDraftBootstrapRequired(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	deps := baseReleaseDeps(t, &fakeReleaseClient{})
	seedInitAuth(t, &deps, root)
	deps.NewClient = func(context.Context, gpc.CredentialInput) (Client, error) {
		return &fakeReleaseClient{
			validateEditErr: errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
		}, nil
	}
	deps.RunBootstrap = func(_ context.Context, _ []string) error {
		return os.WriteFile(filepath.Join(root, "play", "appinit.yaml"), []byte("appDetails:\n  defaultLanguage: en-US\n"), 0o600)
	}

	out, err := runInitCommand(t, deps, "--package-name", "com.example.app", "--dir", "./play")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"draft_bootstrap_required"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseInitReady(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)
	deps.RunBootstrap = func(_ context.Context, _ []string) error {
		if err := os.MkdirAll(filepath.Join(root, "play", "listing", "en-US"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, "play", "appinit.yaml"), []byte("appDetails:\n  defaultLanguage: en-US\n"), 0o600); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, "play", "listing", "en-US", "title.txt"), []byte("Title"), 0o600)
	}

	out, err := runInitCommand(t, deps, "--package-name", "com.example.app", "--dir", "./play")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"ready"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseInitWritesProjectDefaults(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)
	deps.RunBootstrap = func(_ context.Context, _ []string) error {
		return os.WriteFile(filepath.Join(root, "play", "appinit.yaml"), []byte("appDetails:\n  defaultLanguage: en-US\n"), 0o600)
	}

	if _, err := runInitCommand(t, deps, "--package-name", "com.example.app", "--dir", "./play", "--android-project-dir", "./android", "--build-task", ":app:bundleRelease"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".gpc.yaml"))
	if err != nil {
		t.Fatalf("read .gpc.yaml: %v", err)
	}
	got := string(raw)
	for _, want := range []string{"screenshots-dir: play/screenshots", "products-dir: play/products", "subscriptions-dir: play/subscriptions", "android-project-dir: android", "build-task: :app:bundleRelease"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in project config: %s", want, got)
		}
	}
}

func TestReleaseInitAuthRequired(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	deps := Deps{
		LoadConfig: func() (config.Config, error) { return config.Config{}, nil },
		LookupEnv:  func(string) string { return "" },
		Stdin:      &bytes.Buffer{},
	}
	_, err := runInitCommand(t, deps, "--package-name", "com.example.app", "--dir", "./play")
	if err == nil || !strings.Contains(err.Error(), "authentication is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
