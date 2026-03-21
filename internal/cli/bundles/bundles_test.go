package bundles

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"google.golang.org/api/androidpublisher/v3"
)

type fakeClient struct {
	list            []gpc.BundleInfo
	listErr         error
	upload          gpc.BundleInfo
	uploadErr       error
	generatedAPKsFn func(context.Context, string, int64) (*androidpublisher.GeneratedApksListResponse, error)
}

func (f fakeClient) ListBundles(_ context.Context, _, _ string) ([]gpc.BundleInfo, error) {
	return f.list, f.listErr
}

func (f fakeClient) UploadBundle(_ context.Context, _, _, _ string) (gpc.BundleInfo, error) {
	return f.upload, f.uploadErr
}

func (f fakeClient) ListGeneratedAPKs(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.GeneratedApksListResponse, error) {
	if f.generatedAPKsFn == nil {
		return &androidpublisher.GeneratedApksListResponse{}, nil
	}
	return f.generatedAPKsFn(ctx, packageName, versionCode)
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

func TestBundlesList_MinimalOutput(t *testing.T) {
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

	out, err := runBundles(t, deps, "list", "--package-name", "com.example.app", "--edit-id", "edit-1", "--output", "minimal")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if out != "1\n2\n" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestBundlesList_RejectsUnsupportedOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runBundles(t, deps, "list", "--package-name", "com.example.app", "--edit-id", "edit-1", "--output", "table")
	if err == nil || !strings.Contains(err.Error(), `unsupported output format "table"`) {
		t.Fatalf("unexpected error: %v", err)
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

func TestBundlesWait_ReturnsReadyWhenGeneratedAPKsAppear(t *testing.T) {
	attempts := 0
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		Sleep:      func(context.Context, time.Duration) error { return nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				generatedAPKsFn: func(_ context.Context, packageName string, versionCode int64) (*androidpublisher.GeneratedApksListResponse, error) {
					attempts++
					if packageName != "com.example.app" {
						t.Fatalf("unexpected package name: %q", packageName)
					}
					if versionCode != 123 {
						t.Fatalf("unexpected version code: %d", versionCode)
					}
					if attempts < 3 {
						return &androidpublisher.GeneratedApksListResponse{}, nil
					}
					return &androidpublisher.GeneratedApksListResponse{
						GeneratedApks: []*androidpublisher.GeneratedApksPerSigningKey{
							{CertificateSha256Hash: "hash"},
						},
					}, nil
				},
			}, nil
		},
	}

	out, err := runBundles(t, deps, "wait", "--package-name", "com.example.app", "--version-code", "123", "--timeout", "50ms", "--interval", "1ms")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"ready"`) || !strings.Contains(out, `"attempts":3`) || !strings.Contains(out, `"generatedApkCount":1`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestBundlesWait_TimesOutWhenGeneratedAPKsNeverAppear(t *testing.T) {
	sleeps := 0
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		Sleep: func(_ context.Context, _ time.Duration) error {
			sleeps++
			if sleeps >= 2 {
				return context.DeadlineExceeded
			}
			return nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				generatedAPKsFn: func(_ context.Context, _ string, _ int64) (*androidpublisher.GeneratedApksListResponse, error) {
					return &androidpublisher.GeneratedApksListResponse{}, nil
				},
			}, nil
		},
	}

	out, err := runBundles(t, deps, "wait", "--package-name", "com.example.app", "--version-code", "123", "--timeout", "3ms", "--interval", "1ms")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out waiting for generated apks") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"timeout"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestBundlesWait_ReturnsPollingError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				generatedAPKsFn: func(_ context.Context, _ string, _ int64) (*androidpublisher.GeneratedApksListResponse, error) {
					return nil, errors.New("boom")
				},
			}, nil
		},
	}

	_, err := runBundles(t, deps, "wait", "--package-name", "com.example.app", "--version-code", "123", "--timeout", "50ms", "--interval", "1ms")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to list generated apks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBundlesWait_ValidatesFlags(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runBundles(t, deps, "wait", "--package-name", "com.example.app", "--timeout", "50ms", "--interval", "1ms")
	if err == nil || !strings.Contains(err.Error(), "--version-code must be greater than zero") {
		t.Fatalf("unexpected version-code error: %v", err)
	}

	_, err = runBundles(t, deps, "wait", "--package-name", "com.example.app", "--version-code", "123", "--timeout", "50ms", "--interval", "0s")
	if err == nil || !strings.Contains(err.Error(), "--interval must be greater than zero") {
		t.Fatalf("unexpected interval error: %v", err)
	}

	_, err = runBundles(t, deps, "wait", "--package-name", "com.example.app", "--version-code", "123", "--timeout", "0s", "--interval", "1ms")
	if err == nil || !strings.Contains(err.Error(), "--timeout must be greater than zero") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func TestBundlesWait_UsesDefaultInterval(t *testing.T) {
	attempts := 0
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		Sleep: func(_ context.Context, d time.Duration) error {
			if d != defaultWaitInterval {
				t.Fatalf("expected default interval %s, got %s", defaultWaitInterval, d)
			}
			return nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				generatedAPKsFn: func(_ context.Context, _ string, _ int64) (*androidpublisher.GeneratedApksListResponse, error) {
					attempts++
					if attempts == 1 {
						return &androidpublisher.GeneratedApksListResponse{}, nil
					}
					return &androidpublisher.GeneratedApksListResponse{
						GeneratedApks: []*androidpublisher.GeneratedApksPerSigningKey{{CertificateSha256Hash: "hash"}},
					}, nil
				},
			}, nil
		},
	}

	out, err := runBundles(t, deps, "wait", "--package-name", "com.example.app", "--version-code", "123", "--timeout", "20s")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"interval":"5s"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}
