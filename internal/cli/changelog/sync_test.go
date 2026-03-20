package changelog

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

type fakeClient struct {
	getTrack gpc.TrackInfo

	updateTrack gpc.TrackUpdate
	deleteCalls int
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return nil
}

func (f *fakeClient) ValidateEdit(_ context.Context, _, _ string) error { return nil }
func (f *fakeClient) CommitEdit(_ context.Context, _, _ string, _ bool) (gpc.EditInfo, error) {
	return gpc.EditInfo{ID: "edit-1"}, nil
}
func (f *fakeClient) GetTrack(_ context.Context, _, _, _ string) (gpc.TrackInfo, error) {
	if f.getTrack.Name == "" {
		return gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Name: "1.2.3", Status: "inProgress", UserFraction: 0.1, VersionCodes: []int64{123}},
			},
		}, nil
	}
	return f.getTrack, nil
}
func (f *fakeClient) UpdateTrack(_ context.Context, _, _, _ string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
	f.updateTrack = update
	return gpc.TrackInfo{Name: "production"}, nil
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func writeNotesDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "en-US.txt"), []byte("Hello"), 0o600); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ja-JP.txt"), []byte("こんにちは"), 0o600); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	return root
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

func TestLoadReleaseNotesDir(t *testing.T) {
	notes, _, err := loadReleaseNotesDir(writeNotesDir(t), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 || notes[0].Language != "en-US" || notes[1].Language != "ja-JP" {
		t.Fatalf("unexpected notes: %+v", notes)
	}
}

func TestChangelogSyncDryRun(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--track", "production", "--dir", writeNotesDir(t), "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected delete call, got %d", client.deleteCalls)
	}
	if len(client.updateTrack.ReleaseNotes) != 0 {
		t.Fatalf("dry-run should not update track: %+v", client.updateTrack)
	}
}

func TestChangelogSyncCommit(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--track", "production", "--dir", writeNotesDir(t), "--confirm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"committed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.updateTrack.Status != "inProgress" || len(client.updateTrack.ReleaseNotes) != 2 || client.updateTrack.ReleaseName != "1.2.3" {
		t.Fatalf("unexpected update: %+v", client.updateTrack)
	}
}

func TestChangelogSyncFailsOnMultipleReleases(t *testing.T) {
	client := &fakeClient{
		getTrack: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Name: "a", Status: "completed", VersionCodes: []int64{1}},
				{Name: "b", Status: "inProgress", VersionCodes: []int64{2}},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--track", "production", "--dir", writeNotesDir(t), "--confirm")
	if err == nil || !strings.Contains(err.Error(), "multiple releases") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangelogSyncUsesDefaultFallbackForMissingLocales(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "default.txt"), []byte("Fallback"), 0o600); err != nil {
		t.Fatalf("write fallback notes: %v", err)
	}
	client := &fakeClient{
		getTrack: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{
					Name:         "1.2.3",
					Status:       "completed",
					VersionCodes: []int64{123},
					ReleaseNotes: []gpc.LocalizedText{
						{Language: "en-US", Text: "Old EN"},
						{Language: "fr-FR", Text: "Old FR"},
					},
				},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--track", "production", "--dir", root, "--confirm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.updateTrack.ReleaseNotes) != 2 {
		t.Fatalf("unexpected release notes: %+v", client.updateTrack.ReleaseNotes)
	}
	for _, note := range client.updateTrack.ReleaseNotes {
		if note.Text != "Fallback" {
			t.Fatalf("expected fallback text, got %+v", client.updateTrack.ReleaseNotes)
		}
	}
}
