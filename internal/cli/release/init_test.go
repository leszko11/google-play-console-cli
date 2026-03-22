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
		filepath.Join(root, "play", "bootstrap-state.json"),
		filepath.Join(root, ".gpc.yaml"),
		filepath.Join(root, ".gpc", "workflow.yml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", path, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "play", "MANUAL_FIRST_UPLOAD.md"))
	if err != nil {
		t.Fatalf("read manual bridge: %v", err)
	}
	for _, want := range []string{"## What gpc can do now", "## What must be done in Play Console", "## What gpc cannot do for public apps", "Internal testing", "save the release as draft"} {
		if !strings.Contains(strings.ToLower(string(raw)), strings.ToLower(want)) {
			t.Fatalf("missing %q in manual bridge: %s", want, string(raw))
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
			getTrackInfo: gpc.TrackInfo{
				Name: "internal",
				Releases: []gpc.TrackReleaseInfo{
					{Status: "draft", VersionCodes: []int64{321}},
				},
			},
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
	if !strings.Contains(out, `"workspaceReadiness":"draft_bootstrap_required"`) {
		t.Fatalf("unexpected workspace readiness: %s", out)
	}
	if !strings.Contains(out, `DRAFT_BOOTSTRAP.md`) {
		t.Fatalf("expected bootstrap note in output: %s", out)
	}
	if !strings.Contains(out, `"bootstrapDraftExists":true`) || !strings.Contains(out, `"bootstrapVersionCodes":[321]`) {
		t.Fatalf("expected bootstrap summary in output: %s", out)
	}
	stateRaw, err := os.ReadFile(filepath.Join(root, "play", "bootstrap-state.json"))
	if err != nil {
		t.Fatalf("read bootstrap state: %v", err)
	}
	if !strings.Contains(string(stateRaw), `"bootstrapDraftExists": true`) {
		t.Fatalf("unexpected bootstrap state: %s", string(stateRaw))
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
	if !strings.Contains(out, `"workspaceReadiness":"needs_content"`) {
		t.Fatalf("expected content audit issues in output: %s", out)
	}
}

func TestReleaseInitReadyWorkspaceAuditPasses(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)
	deps.RunBootstrap = func(_ context.Context, _ []string) error {
		for _, path := range []string{
			filepath.Join(root, "play", "listing", "en-US"),
			filepath.Join(root, "play", "screenshots", "en-US", "phone"),
		} {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		}
		if err := os.WriteFile(filepath.Join(root, "play", "appinit.yaml"), []byte("appDetails:\n  defaultLanguage: en-US\n"), 0o600); err != nil {
			return err
		}
		for name, value := range map[string]string{
			filepath.Join(root, "play", "listing", "en-US", "title.txt"):             "Title",
			filepath.Join(root, "play", "listing", "en-US", "short-description.txt"): "Short description",
			filepath.Join(root, "play", "listing", "en-US", "full-description.txt"):  "Full description with enough text.",
		} {
			if err := os.WriteFile(name, []byte(value), 0o600); err != nil {
				return err
			}
		}
		return writeTinyPNG(filepath.Join(root, "play", "screenshots", "en-US", "phone", "01.png"))
	}

	artifactPath := filepath.Join(root, "android", "app", "build", "outputs", "bundle", "release", "app-release.aab")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatalf("mkdir artifact: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("bundle"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	out, err := runInitCommand(t, deps, "--package-name", "com.example.app", "--dir", "./play")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"workspaceReadiness":"ready"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseInitStrictExportFailsOnMissingContent(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)
	deps.RunBootstrap = func(_ context.Context, _ []string) error {
		if err := os.MkdirAll(filepath.Join(root, "play", "listing", "en-US"), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, "play", "appinit.yaml"), []byte("appDetails:\n  defaultLanguage: en-US\n"), 0o600)
	}

	_, err := runInitCommand(t, deps, "--package-name", "com.example.app", "--dir", "./play", "--strict-export")
	if err == nil || !strings.Contains(err.Error(), "workspace audit failed") {
		t.Fatalf("unexpected error: %v", err)
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

func writeTinyPNG(path string) error {
	if err := os.WriteFile(path, []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x04, 0x00, 0x01, 0xe2, 0x26, 0x05, 0x9b,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44,
		0xae, 0x42, 0x60, 0x82,
	}, 0o600); err != nil {
		return err
	}
	return nil
}
