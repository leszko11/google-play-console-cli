package migrate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
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

type fakeDiffClient struct {
	listings    []gpc.ListingInfo
	images      map[string][]gpc.ImageInfo
	tracks      []gpc.TrackInfo
	deleteCalls int
}

func (f *fakeDiffClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeDiffClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return nil
}

func (f *fakeDiffClient) ListListings(_ context.Context, _, _ string) ([]gpc.ListingInfo, error) {
	return f.listings, nil
}

func (f *fakeDiffClient) ListImages(_ context.Context, _, _, language, imageType string) ([]gpc.ImageInfo, error) {
	return f.images[language+"/"+imageType], nil
}

func (f *fakeDiffClient) ListTracks(_ context.Context, _, _ string) ([]gpc.TrackInfo, error) {
	return f.tracks, nil
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func writeFastlaneFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "fastlane", "metadata", "android")
	writeFile(t, filepath.Join(source, "en-US", "title.txt"), "Title")
	writeFile(t, filepath.Join(source, "en-US", "short_description.txt"), "Short")
	writeFile(t, filepath.Join(source, "en-US", "full_description.txt"), "Full")
	writeFile(t, filepath.Join(source, "en-US", "images", "phoneScreenshots", "1.png"), "local-image")
	writeFile(t, filepath.Join(source, "en-US", "changelogs", "123.txt"), "Fresh notes")
	return filepath.Join(root, "fastlane")
}

func runMigrateWithOutput(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		}
	}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestFastlaneDiffJSON(t *testing.T) {
	client := &fakeDiffClient{
		listings: []gpc.ListingInfo{
			{Language: "en-US", Title: "Old title", ShortDescription: "Short", FullDescription: "Full"},
			{Language: "de-DE", Title: "Titel", ShortDescription: "Kurz", FullDescription: "Lang"},
		},
		images: map[string][]gpc.ImageInfo{
			"en-US/phoneScreenshots": {{SHA256: "remote-image"}},
		},
		tracks: []gpc.TrackInfo{
			{
				Name: "production",
				Releases: []gpc.TrackReleaseInfo{{
					Name: "1.0.0",
					ReleaseNotes: []gpc.LocalizedText{{
						Language: "en-US",
						Text:     "Old notes",
					}},
				}},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runMigrateWithOutput(t, deps,
		"fastlane", "diff",
		"--from-dir", writeFastlaneFixture(t),
		"--package-name", "com.example.app",
		"--track", "production",
		"--version-code", "123",
		"--output", "json",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"hasDiff":true`, `"field":"title"`, `"field":"phoneScreenshots"`, `"field":"releaseNotes"`, `"action":"remote_only_locale"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected cleanup delete call, got %d", client.deleteCalls)
	}
}

func TestFastlaneDiffTableOutput(t *testing.T) {
	client := &fakeDiffClient{
		listings: []gpc.ListingInfo{
			{Language: "en-US", Title: "Title", ShortDescription: "Short", FullDescription: "Full"},
		},
		images: map[string][]gpc.ImageInfo{
			"en-US/phoneScreenshots": {{SHA256: "1f1d3f1d658dca9e96689387cc2d6eefa5b73b08b6adf74ddf2123321892d304"}},
		},
		tracks: []gpc.TrackInfo{
			{
				Name: "production",
				Releases: []gpc.TrackReleaseInfo{{
					Name: "1.0.0",
					ReleaseNotes: []gpc.LocalizedText{{
						Language: "en-US",
						Text:     "Fresh notes",
					}},
				}},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runMigrateWithOutput(t, deps,
		"fastlane", "diff",
		"--from-dir", writeFastlaneFixture(t),
		"--package-name", "com.example.app",
		"--track", "production",
		"--version-code", "123",
		"--output", "table",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "STATUS\tno-diff") {
		t.Fatalf("expected no-diff table, got %s", out)
	}
	if !strings.Contains(out, "SCOPE\tTARGET\tFIELD\tACTION\tLIVE\tDESIRED") {
		t.Fatalf("expected table header, got %s", out)
	}
}

func TestFastlaneDiffMarkdownOutput(t *testing.T) {
	client := &fakeDiffClient{
		listings: []gpc.ListingInfo{
			{Language: "en-US", Title: "Old title", ShortDescription: "Short", FullDescription: "Full"},
		},
		images: map[string][]gpc.ImageInfo{
			"en-US/phoneScreenshots": {{SHA256: "remote-image"}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runMigrateWithOutput(t, deps,
		"fastlane", "diff",
		"--from-dir", writeFastlaneFixture(t),
		"--package-name", "com.example.app",
		"--output", "markdown",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"| field | value |",
		"| status | diff |",
		"| scope | target | field | action | live | desired |",
		"| listing | en-US | title | update | Old title | Title |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestFastlaneDiffYAMLOutput(t *testing.T) {
	client := &fakeDiffClient{
		listings: []gpc.ListingInfo{
			{Language: "en-US", Title: "Old title", ShortDescription: "Short", FullDescription: "Full"},
		},
		images: map[string][]gpc.ImageInfo{
			"en-US/phoneScreenshots": {{SHA256: "remote-image"}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runMigrateWithOutput(t, deps,
		"fastlane", "diff",
		"--from-dir", writeFastlaneFixture(t),
		"--package-name", "com.example.app",
		"--output", "yaml",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"packageName: com.example.app", "hasDiff: true", "field: title"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}
