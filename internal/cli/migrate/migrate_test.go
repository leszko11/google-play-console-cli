package migrate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runMigrate(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func TestFastlaneImportCopiesListingImagesAndChangelogs(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "fastlane", "metadata")
	outDir := filepath.Join(root, "play")

	writeFile(t, filepath.Join(source, "android", "en-US", "title.txt"), "English title")
	writeFile(t, filepath.Join(source, "android", "en-US", "short_description.txt"), "English short")
	writeFile(t, filepath.Join(source, "android", "en-US", "full_description.txt"), "English full")
	writeFile(t, filepath.Join(source, "android", "en-US", "video.txt"), "ignored")
	writeFile(t, filepath.Join(source, "android", "en-US", "images", "icon.png"), "icon")
	writeFile(t, filepath.Join(source, "android", "en-US", "images", "phoneScreenshots", "01.png"), "shot1")
	writeFile(t, filepath.Join(source, "android", "en-US", "changelogs", "default.txt"), "English default")
	writeFile(t, filepath.Join(source, "android", "en-US", "changelogs", "123.txt"), "English 123")

	writeFile(t, filepath.Join(source, "android", "pl-PL", "title.txt"), "Polski tytul")
	writeFile(t, filepath.Join(source, "android", "pl-PL", "short_description.txt"), "Polski krotki")
	writeFile(t, filepath.Join(source, "android", "pl-PL", "full_description.txt"), "Polski pelny")
	writeFile(t, filepath.Join(source, "android", "pl-PL", "changelogs", "default.txt"), "Polski domyslny")

	out, err := runMigrate(t, Deps{},
		"fastlane", "import",
		"--from-dir", source,
		"--dir", outDir,
		"--track", "production",
		"--version-code", "123",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"imported"`) || !strings.Contains(out, `"listingLocales":2`) || !strings.Contains(out, `"changelogLocales":2`) {
		t.Fatalf("unexpected output: %s", out)
	}

	if got := readFile(t, filepath.Join(outDir, "listing", "en-US", "short-description.txt")); got != "English short" {
		t.Fatalf("unexpected short description: %q", got)
	}
	if got := readFile(t, filepath.Join(outDir, "listing", "en-US", "images", "icon.png")); got != "icon" {
		t.Fatalf("unexpected icon copy: %q", got)
	}
	if got := readFile(t, filepath.Join(outDir, "listing", "en-US", "images", "phoneScreenshots", "01.png")); got != "shot1" {
		t.Fatalf("unexpected screenshot copy: %q", got)
	}
	if got := readFile(t, filepath.Join(outDir, "changelog", "production", "en-US.txt")); got != "English 123" {
		t.Fatalf("expected version-specific changelog, got %q", got)
	}
	if got := readFile(t, filepath.Join(outDir, "changelog", "production", "pl-PL.txt")); got != "Polski domyslny" {
		t.Fatalf("expected default changelog fallback, got %q", got)
	}
}

func TestFastlaneImportWritesProjectConfig(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "fastlane", "metadata", "android", "en-US")
	outDir := filepath.Join(root, "play")

	writeFile(t, filepath.Join(source, "title.txt"), "Title")
	writeFile(t, filepath.Join(source, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(source, "full_description.txt"), "Full")

	_, err := runMigrate(t, Deps{},
		"fastlane", "import",
		"--from-dir", filepath.Join(root, "fastlane"),
		"--dir", outDir,
		"--package-name", "com.example.app",
		"--write-project-config",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	cfg := readFile(t, filepath.Join(outDir, ".gpc.yaml"))
	if !strings.Contains(cfg, "package-name: com.example.app") || !strings.Contains(cfg, "listing-dir: ./listing") || !strings.Contains(cfg, "changelog-dir: ./changelog") {
		t.Fatalf("unexpected project config: %s", cfg)
	}
}

func TestFastlaneImportRequiresPackageNameForProjectConfig(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "fastlane", "metadata", "android", "en-US")
	writeFile(t, filepath.Join(source, "title.txt"), "Title")
	writeFile(t, filepath.Join(source, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(source, "full_description.txt"), "Full")

	_, err := runMigrate(t, Deps{},
		"fastlane", "import",
		"--from-dir", filepath.Join(root, "fastlane"),
		"--dir", filepath.Join(root, "play"),
		"--write-project-config",
	)
	if err == nil || !strings.Contains(err.Error(), "--package-name is required when --write-project-config is set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFastlaneImportRequiresListingFiles(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "fastlane", "metadata", "android", "en-US")
	writeFile(t, filepath.Join(source, "title.txt"), "Title")
	writeFile(t, filepath.Join(source, "full_description.txt"), "Full")

	_, err := runMigrate(t, Deps{},
		"fastlane", "import",
		"--from-dir", filepath.Join(root, "fastlane"),
		"--dir", filepath.Join(root, "play"),
	)
	if err == nil || !strings.Contains(err.Error(), "missing required file short_description.txt") {
		t.Fatalf("unexpected error: %v", err)
	}
}
