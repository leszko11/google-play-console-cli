package generatedapks

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"google.golang.org/api/androidpublisher/v3"
)

type fakeClient struct {
	listResp            *androidpublisher.GeneratedApksListResponse
	downloadResp        []byte
	listErr             error
	downloadErr         error
	capturedVersionCode int64
	capturedDownloadID  string
}

func (f *fakeClient) ListGeneratedAPKs(_ context.Context, _ string, versionCode int64) (*androidpublisher.GeneratedApksListResponse, error) {
	f.capturedVersionCode = versionCode
	return f.listResp, f.listErr
}

func (f *fakeClient) DownloadGeneratedAPK(_ context.Context, _ string, versionCode int64, downloadID string) ([]byte, error) {
	f.capturedVersionCode = versionCode
	f.capturedDownloadID = downloadID
	return append([]byte(nil), f.downloadResp...), f.downloadErr
}

func runGeneratedAPKs(t *testing.T, deps Deps, args ...string) (string, error) {
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

func bindGlobalPackageName(t *testing.T, packageName string) {
	t.Helper()
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	cfg := &shared.GlobalFlags{}
	shared.BindGlobalFlags(fs, cfg)
	cfg.PackageName = packageName
}

func TestGeneratedAPKsList_ReturnsMetadata(t *testing.T) {
	fc := &fakeClient{
		listResp: &androidpublisher.GeneratedApksListResponse{
			GeneratedApks: []*androidpublisher.GeneratedApksPerSigningKey{
				{
					CertificateSha256Hash: "cert-1",
					GeneratedStandaloneApks: []*androidpublisher.GeneratedStandaloneApk{
						{DownloadId: "download-1", VariantId: 7},
					},
				},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runGeneratedAPKs(t, deps, "list", "--package-name", "com.example.app", "--version-code", "123")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"downloadId":"download-1"`) || !strings.Contains(out, `"certificateSha256Hash":"cert-1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedVersionCode != 123 {
		t.Fatalf("unexpected version code: %d", fc.capturedVersionCode)
	}
}

func TestGeneratedAPKsDownload_RequiresDownloadID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runGeneratedAPKs(t, deps, "download", "--package-name", "com.example.app", "--version-code", "123", "--output", filepath.Join(t.TempDir(), "generated.apk"))
	if err == nil || !strings.Contains(err.Error(), "--download-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGeneratedAPKsDownload_WritesFile(t *testing.T) {
	fc := &fakeClient{
		downloadResp: []byte("generated-apk"),
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	outputPath := filepath.Join(t.TempDir(), "generated.apk")
	out, err := runGeneratedAPKs(t, deps, "download", "--package-name", "com.example.app", "--version-code", "123", "--download-id", "download-1", "--output", outputPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"downloadId":"download-1"`) || !strings.Contains(out, `"sizeBytes":13`) {
		t.Fatalf("unexpected output: %s", out)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read download: %v", err)
	}
	if string(raw) != "generated-apk" {
		t.Fatalf("unexpected file contents: %q", string(raw))
	}
	if fc.capturedVersionCode != 123 || fc.capturedDownloadID != "download-1" {
		t.Fatalf("unexpected download args: version=%d download=%q", fc.capturedVersionCode, fc.capturedDownloadID)
	}
}

func TestGeneratedAPKsList_UsesGlobalPackageName(t *testing.T) {
	bindGlobalPackageName(t, "com.example.global")
	fc := &fakeClient{
		listResp: &androidpublisher.GeneratedApksListResponse{},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	if _, err := runGeneratedAPKs(t, deps, "list", "--version-code", "123"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
}
