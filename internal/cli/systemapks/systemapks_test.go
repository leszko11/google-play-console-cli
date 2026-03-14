package systemapks

import (
	"bytes"
	"context"
	"encoding/json"
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
	listResp            *androidpublisher.SystemApksListResponse
	getResp             *androidpublisher.Variant
	createResp          *androidpublisher.Variant
	downloadResp        []byte
	listErr             error
	getErr              error
	createErr           error
	downloadErr         error
	capturedVersionCode int64
	capturedVariantID   int64
	capturedCreate      *androidpublisher.Variant
}

func (f *fakeClient) ListSystemAPKVariants(_ context.Context, _ string, versionCode int64) (*androidpublisher.SystemApksListResponse, error) {
	f.capturedVersionCode = versionCode
	return f.listResp, f.listErr
}

func (f *fakeClient) GetSystemAPKVariant(_ context.Context, _ string, versionCode, variantID int64) (*androidpublisher.Variant, error) {
	f.capturedVersionCode = versionCode
	f.capturedVariantID = variantID
	return f.getResp, f.getErr
}

func (f *fakeClient) CreateSystemAPKVariant(_ context.Context, _ string, versionCode int64, variant *androidpublisher.Variant) (*androidpublisher.Variant, error) {
	f.capturedVersionCode = versionCode
	f.capturedCreate = variant
	return f.createResp, f.createErr
}

func (f *fakeClient) DownloadSystemAPKVariant(_ context.Context, _ string, versionCode, variantID int64) ([]byte, error) {
	f.capturedVersionCode = versionCode
	f.capturedVariantID = variantID
	return append([]byte(nil), f.downloadResp...), f.downloadErr
}

func runSystemAPKs(t *testing.T, deps Deps, args ...string) (string, error) {
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

func writePayloadFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write payload: %v", err)
	}
	return path
}

func TestSystemAPKsList_ReturnsVariants(t *testing.T) {
	fc := &fakeClient{
		listResp: &androidpublisher.SystemApksListResponse{
			Variants: []*androidpublisher.Variant{{VariantId: 7}, {VariantId: 8}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runSystemAPKs(t, deps, "list", "--package-name", "com.example.app", "--version-code", "123")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"variantId":7`) || !strings.Contains(out, `"variantId":8`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedVersionCode != 123 {
		t.Fatalf("unexpected version code: %d", fc.capturedVersionCode)
	}
}

func TestSystemAPKsGet_RequiresVariantID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runSystemAPKs(t, deps, "get", "--package-name", "com.example.app", "--version-code", "123")
	if err == nil || !strings.Contains(err.Error(), "--variant-id must be greater than zero") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSystemAPKsGet_ReturnsVariant(t *testing.T) {
	fc := &fakeClient{
		getResp: &androidpublisher.Variant{
			VariantId: 7,
			DeviceSpec: &androidpublisher.DeviceSpec{
				SupportedAbis: []string{"arm64-v8a"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runSystemAPKs(t, deps, "get", "--package-name", "com.example.app", "--version-code", "123", "--variant-id", "7")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"variantId":7`) || !strings.Contains(out, `"supportedAbis":["arm64-v8a"]`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSystemAPKsCreate_ReadsPayload(t *testing.T) {
	fc := &fakeClient{
		createResp: &androidpublisher.Variant{VariantId: 9},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	payload := writePayloadFile(t, "variant.json", `{
		"deviceSpec":{
			"screenDensity":480,
			"supportedAbis":["arm64-v8a","armeabi-v7a"],
			"supportedLocales":["en-US"]
		},
		"options":{"rotated":true}
	}`)
	out, err := runSystemAPKs(t, deps, "create", "--package-name", "com.example.app", "--version-code", "123", "--input", payload)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"variantId":9`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedCreate == nil || fc.capturedCreate.DeviceSpec == nil || fc.capturedCreate.DeviceSpec.ScreenDensity != 480 || fc.capturedCreate.Options == nil || !fc.capturedCreate.Options.Rotated {
		t.Fatalf("unexpected create payload: %#v", fc.capturedCreate)
	}
}

func TestSystemAPKsDownload_WritesFile(t *testing.T) {
	fc := &fakeClient{
		downloadResp: []byte("apk-bytes"),
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	outputPath := filepath.Join(t.TempDir(), "system.apk")
	out, err := runSystemAPKs(t, deps, "download", "--package-name", "com.example.app", "--version-code", "123", "--variant-id", "7", "--output", outputPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	jsonPath, _ := json.Marshal(outputPath)
	if !strings.Contains(out, `"outputPath":`+string(jsonPath)) || !strings.Contains(out, `"sizeBytes":9`) {
		t.Fatalf("unexpected output: %s", out)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read download: %v", err)
	}
	if string(raw) != "apk-bytes" {
		t.Fatalf("unexpected file contents: %q", string(raw))
	}
	if fc.capturedVersionCode != 123 || fc.capturedVariantID != 7 {
		t.Fatalf("unexpected download args: version=%d variant=%d", fc.capturedVersionCode, fc.capturedVariantID)
	}
}

func TestSystemAPKsDownload_RequiresOutput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runSystemAPKs(t, deps, "download", "--package-name", "com.example.app", "--version-code", "123", "--variant-id", "7")
	if err == nil || !strings.Contains(err.Error(), "--output is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSystemAPKsGet_UsesGlobalPackageName(t *testing.T) {
	bindGlobalPackageName(t, "com.example.global")
	fc := &fakeClient{
		getResp: &androidpublisher.Variant{VariantId: 7},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	if _, err := runSystemAPKs(t, deps, "get", "--version-code", "123", "--variant-id", "7"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
}
