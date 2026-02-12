package bundles

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	list      []gpc.BundleInfo
	listErr   error
	upload    gpc.BundleInfo
	uploadErr error
}

func (f fakeClient) ListBundles(_ context.Context, _, _ string) ([]gpc.BundleInfo, error) {
	return f.list, f.listErr
}

func (f fakeClient) UploadBundle(_ context.Context, _, _, _ string) (gpc.BundleInfo, error) {
	return f.upload, f.uploadErr
}

func runBundles(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}

	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func TestBundlesList_ReturnsBundles(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				list: []gpc.BundleInfo{
					{VersionCode: 1},
					{VersionCode: 2},
				},
			}, nil
		},
	}

	out, err := runBundles(t, deps, "list", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"versionCode":1`) || !strings.Contains(out, `"versionCode":2`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestBundlesUpload_RequiresFile(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runBundles(t, deps, "upload", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--file is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBundlesUpload_ReturnsUploaded(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				upload: gpc.BundleInfo{VersionCode: 123},
			}, nil
		},
	}

	out, err := runBundles(t, deps, "upload", "--package-name", "com.example.app", "--edit-id", "edit-1", "--file", "/tmp/app.aab")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"uploaded"`) || !strings.Contains(out, `"versionCode":123`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestBundlesUpload_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				uploadErr: errors.New("conflict"),
			}, nil
		},
	}

	_, err := runBundles(t, deps, "upload", "--package-name", "com.example.app", "--edit-id", "edit-1", "--file", "/tmp/app.aab")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to upload bundle") {
		t.Fatalf("unexpected error: %v", err)
	}
}
