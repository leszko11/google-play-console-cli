package apks

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
	"google.golang.org/api/androidpublisher/v3"
)

type fakeClient struct {
	list              []gpc.APKInfo
	listErr           error
	upload            gpc.APKInfo
	uploadErr         error
	externallyHosted  gpc.ExternallyHostedAPKInfo
	externallyHostErr error
}

func (f fakeClient) ListAPKs(_ context.Context, _, _ string) ([]gpc.APKInfo, error) {
	return f.list, f.listErr
}

func (f fakeClient) UploadAPK(_ context.Context, _, _, _ string) (gpc.APKInfo, error) {
	return f.upload, f.uploadErr
}

func (f fakeClient) AddExternallyHostedAPK(_ context.Context, _, _ string, _ *androidpublisher.ExternallyHostedApk) (gpc.ExternallyHostedAPKInfo, error) {
	return f.externallyHosted, f.externallyHostErr
}

func runAPKs(t *testing.T, deps Deps, args ...string) (string, error) {
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

func writeExternallyHostedAPKPayload(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "externally-hosted-apk.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}

func TestAPKsList_ReturnsAPKs(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				list: []gpc.APKInfo{
					{VersionCode: 1},
					{VersionCode: 2},
				},
			}, nil
		},
	}

	out, err := runAPKs(t, deps, "list", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"versionCode":1`) || !strings.Contains(out, `"versionCode":2`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAPKsUpload_RequiresFile(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runAPKs(t, deps, "upload", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--file is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPKsUpload_ReturnsUploaded(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				upload: gpc.APKInfo{VersionCode: 123},
			}, nil
		},
	}

	out, err := runAPKs(t, deps, "upload", "--package-name", "com.example.app", "--edit-id", "edit-1", "--file", "/tmp/app.apk")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"uploaded"`) || !strings.Contains(out, `"versionCode":123`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAPKsUpload_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				uploadErr: errors.New("conflict"),
			}, nil
		},
	}

	_, err := runAPKs(t, deps, "upload", "--package-name", "com.example.app", "--edit-id", "edit-1", "--file", "/tmp/app.apk")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to upload apk") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPKsAddExternallyHosted_ReturnsAdded(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				externallyHosted: gpc.ExternallyHostedAPKInfo{
					PackageName:         "com.example.app",
					ExternallyHostedURL: "https://example.com/app.apk",
					VersionCode:         42,
				},
			}, nil
		},
	}

	out, err := runAPKs(
		t,
		deps,
		"add-externally-hosted",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--input", writeExternallyHostedAPKPayload(t, `{"packageName":"com.example.app","externallyHostedUrl":"https://example.com/app.apk","versionCode":42}`),
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"added"`) || !strings.Contains(out, `"externallyHostedUrl":"https://example.com/app.apk"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAPKsAddExternallyHosted_RequiresInput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runAPKs(t, deps, "add-externally-hosted", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err == nil || !strings.Contains(err.Error(), "--input is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
