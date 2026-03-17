package diff

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	listings    []gpc.ListingInfo
	images      map[string][]gpc.ImageInfo
	tracks      []gpc.TrackInfo
	deleteCalls int
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return nil
}

func (f *fakeClient) ListListings(_ context.Context, _, _ string) ([]gpc.ListingInfo, error) {
	return f.listings, nil
}

func (f *fakeClient) ListImages(_ context.Context, _, _, language, imageType string) ([]gpc.ImageInfo, error) {
	return f.images[language+"/"+imageType], nil
}

func (f *fakeClient) ListTracks(_ context.Context, _, _ string) ([]gpc.TrackInfo, error) {
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

func writeListingFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite := func(path, contents string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	mustWrite(filepath.Join(root, "en-US", "title.txt"), "Title")
	mustWrite(filepath.Join(root, "en-US", "short-description.txt"), "Short")
	mustWrite(filepath.Join(root, "en-US", "full-description.txt"), "Full")
	mustWrite(filepath.Join(root, "en-US", "images", "phoneScreenshots", "1.png"), "local-image")
	return root
}

func localImageHash(t *testing.T, contents string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(contents))
	return hex.EncodeToString(sum[:])
}

func runCommand(t *testing.T, deps Deps, args ...string) (string, error) {
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

func TestListingDiffJSON(t *testing.T) {
	client := &fakeClient{
		listings: []gpc.ListingInfo{
			{Language: "en-US", Title: "Old title", ShortDescription: "Short", FullDescription: "Full"},
			{Language: "de-DE", Title: "Titel", ShortDescription: "Kurz", FullDescription: "Lang"},
		},
		images: map[string][]gpc.ImageInfo{
			"en-US/phoneScreenshots": {{SHA256: localImageHash(t, "remote-image")}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "listing", "--package-name", "com.example.app", "--dir", writeListingFixture(t), "--delete-missing", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"hasDiff":true`) {
		t.Fatalf("expected diff output, got %s", out)
	}
	if !strings.Contains(out, `"field":"title"`) {
		t.Fatalf("expected title diff, got %s", out)
	}
	if !strings.Contains(out, `"field":"phoneScreenshots"`) {
		t.Fatalf("expected image diff, got %s", out)
	}
	if !strings.Contains(out, `"action":"delete_locale"`) {
		t.Fatalf("expected delete locale action, got %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected cleanup delete call, got %d", client.deleteCalls)
	}
}

func TestListingDiffTableOutput(t *testing.T) {
	client := &fakeClient{
		listings: []gpc.ListingInfo{
			{Language: "en-US", Title: "Title", ShortDescription: "Short", FullDescription: "Full"},
		},
		images: map[string][]gpc.ImageInfo{
			"en-US/phoneScreenshots": {{SHA256: localImageHash(t, "local-image")}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "listing", "--package-name", "com.example.app", "--dir", writeListingFixture(t), "--output", "table")
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

func TestListingDiffMarkdownOutput(t *testing.T) {
	client := &fakeClient{
		listings: []gpc.ListingInfo{
			{Language: "en-US", Title: "Old title", ShortDescription: "Short", FullDescription: "Full"},
		},
		images: map[string][]gpc.ImageInfo{
			"en-US/phoneScreenshots": {{SHA256: localImageHash(t, "remote-image")}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "listing", "--package-name", "com.example.app", "--dir", writeListingFixture(t), "--output", "markdown")
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

func TestListingDiffYAMLOutput(t *testing.T) {
	client := &fakeClient{
		listings: []gpc.ListingInfo{
			{Language: "en-US", Title: "Old title", ShortDescription: "Short", FullDescription: "Full"},
		},
		images: map[string][]gpc.ImageInfo{
			"en-US/phoneScreenshots": {{SHA256: localImageHash(t, "remote-image")}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "listing", "--package-name", "com.example.app", "--dir", writeListingFixture(t), "--output", "yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"packageName: com.example.app", "hasDiff: true", "field: title"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestTrackDiffJSON(t *testing.T) {
	client := &fakeClient{
		tracks: []gpc.TrackInfo{
			{
				Name: "production",
				Releases: []gpc.TrackReleaseInfo{{
					Name:           "1.0.0",
					Status:         "draft",
					VersionCodes:   []int64{100},
					UpdatePriority: 2,
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

	out, err := runCommand(
		t,
		deps,
		"track",
		"--package-name", "com.example.app",
		"--track", "production",
		"--status", "completed",
		"--version-codes", "200",
		"--release-name", "2.0.0",
		"--release-notes-text", "New notes",
		"--output", "json",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"hasDiff":true`) {
		t.Fatalf("expected track diff, got %s", out)
	}
	for _, field := range []string{`"field":"status"`, `"field":"versionCodes"`, `"field":"releaseName"`, `"field":"releaseNotes"`} {
		if !strings.Contains(out, field) {
			t.Fatalf("expected %s in output, got %s", field, out)
		}
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected cleanup delete call, got %d", client.deleteCalls)
	}
}

func TestTrackDiffRequiresVersionCodes(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runCommand(t, deps, "track", "--package-name", "com.example.app", "--track", "production", "--status", "completed")
	if err == nil || !strings.Contains(err.Error(), "--version-codes is required") {
		t.Fatalf("expected version code usage error, got %v", err)
	}
}

func TestTrackDiffMarkdownOutput(t *testing.T) {
	client := &fakeClient{
		tracks: []gpc.TrackInfo{
			{
				Name: "production",
				Releases: []gpc.TrackReleaseInfo{{
					Name:         "1.0.0",
					Status:       "draft",
					VersionCodes: []int64{100},
				}},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "track", "--package-name", "com.example.app", "--track", "production", "--status", "completed", "--version-codes", "200", "--output", "markdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"| field | value |",
		"| track | production |",
		"| scope | target | field | action | live | desired |",
		"| track | production | status | update | draft | completed |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestTrackDiffYAMLOutput(t *testing.T) {
	client := &fakeClient{
		tracks: []gpc.TrackInfo{
			{
				Name: "production",
				Releases: []gpc.TrackReleaseInfo{{
					Name:         "1.0.0",
					Status:       "draft",
					VersionCodes: []int64{100},
				}},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "track", "--package-name", "com.example.app", "--track", "production", "--status", "completed", "--version-codes", "200", "--output", "yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"packageName: com.example.app", "track: production", "field: status"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}
