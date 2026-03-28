package initcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := NewCommand(Deps{
		Stdout: &out,
		Stderr: &bytes.Buffer{},
	})
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestInitCreatesScaffold(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	out, err := runCommand(t, "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, path := range []string{
		filepath.Join(root, "play", "appinit.yaml"),
		filepath.Join(root, "play", "release.yaml"),
		filepath.Join(root, "play", "changelog", "internal", "en-US.txt"),
		filepath.Join(root, ".gpc", "workflow.yml"),
		filepath.Join(root, ".gpc.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	for _, want := range []string{
		`"status":"scaffolded"`,
		`"packageName":"com.example.app"`,
		`"defaultTrack":"internal"`,
		`"defaultLocale":"en-US"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestInitPreservesExistingFiles(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	releaseManifest := filepath.Join(root, "play", "release.yaml")
	if err := os.MkdirAll(filepath.Dir(releaseManifest), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(releaseManifest, []byte("track: production\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	out, err := runCommand(t, "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := os.ReadFile(releaseManifest)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if string(raw) != "track: production\n" {
		t.Fatalf("release manifest was overwritten: %s", string(raw))
	}
	if !strings.Contains(out, `"status":"skipped_existing"`) {
		t.Fatalf("expected skipped_existing in output: %s", out)
	}
}

func TestInitWritesRelativeProjectConfigPaths(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	_, err := runCommand(t,
		"--package-name", "com.example.app",
		"--dir", "./play",
		"--android-project-dir", "./android",
		"--build-task", ":app:bundleRelease",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(root, ".gpc.yaml"))
	if err != nil {
		t.Fatalf("read .gpc.yaml: %v", err)
	}
	got := filepath.ToSlash(string(raw))
	for _, want := range []string{
		"listing-dir: play/listing",
		"screenshots-dir: play/screenshots",
		"products-dir: play/products",
		"subscriptions-dir: play/subscriptions",
		"android-project-dir: android",
		"artifact-path: android/app/build/outputs/bundle/release/app-release.aab",
		"appinit-manifest: play/appinit.yaml",
		"release-manifest: play/release.yaml",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestInitCanSkipWorkflowAndProjectConfig(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	out, err := runCommand(t,
		"--package-name", "com.example.app",
		"--write-workflow=false",
		"--write-project-config=false",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".gpc", "workflow.yml")); !os.IsNotExist(err) {
		t.Fatalf("expected workflow to be skipped, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".gpc.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected project config to be skipped, got %v", err)
	}
	for _, want := range []string{`"workflow","status":"skipped"`, `"project_config","status":"skipped"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestInitRejectsUnsupportedOutput(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	_, err := runCommand(t, "--package-name", "com.example.app", "--output", "yaml")
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Fatalf("unexpected error: %v", err)
	}
}
