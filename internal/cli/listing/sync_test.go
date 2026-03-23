package listing

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
	createEditErr error
	listListings  []gpc.ListingInfo

	updateCalls int
	deleteCalls int
	uploadCalls int
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	if f.createEditErr != nil {
		return gpc.EditInfo{}, f.createEditErr
	}
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
func (f *fakeClient) ListListings(_ context.Context, _, _ string) ([]gpc.ListingInfo, error) {
	return f.listListings, nil
}
func (f *fakeClient) UpdateListing(_ context.Context, _, _, _ string, _ gpc.ListingUpdate) (gpc.ListingInfo, error) {
	f.updateCalls++
	return gpc.ListingInfo{}, nil
}
func (f *fakeClient) DeleteListing(_ context.Context, _, _, _ string) error { return nil }
func (f *fakeClient) DeleteAllImages(_ context.Context, _, _, _, _ string) ([]gpc.ImageInfo, error) {
	return nil, nil
}
func (f *fakeClient) UploadImage(_ context.Context, _, _, _, _, _ string) (gpc.ImageInfo, error) {
	f.uploadCalls++
	return gpc.ImageInfo{}, nil
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
	mustWrite(filepath.Join(root, "en-US", "images", "phoneScreenshots", "1.png"), "one")
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

func TestListingSyncDryRun(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--dir", writeListingFixture(t), "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.updateCalls != 1 || client.uploadCalls != 1 {
		t.Fatalf("dry-run should stage listing changes in the temporary edit: updates=%d uploads=%d", client.updateCalls, client.uploadCalls)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected dry-run cleanup delete, got %d", client.deleteCalls)
	}
}

func TestListingSyncCommit(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--dir", writeListingFixture(t), "--confirm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"committed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.updateCalls != 1 {
		t.Fatalf("expected one listing update, got %d", client.updateCalls)
	}
	if client.uploadCalls != 1 {
		t.Fatalf("expected one image upload, got %d", client.uploadCalls)
	}
}

func TestListingSyncDeleteMissingPlansRemoteDeletes(t *testing.T) {
	client := &fakeClient{
		listListings: []gpc.ListingInfo{
			{Language: "en-US"},
			{Language: "de-DE"},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--dir", writeListingFixture(t), "--delete-missing", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"deletedLocales":["de-DE"]`) {
		t.Fatalf("unexpected output: %s", out)
	}
}
