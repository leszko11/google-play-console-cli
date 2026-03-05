package devicetierconfigs

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
	listResp             *androidpublisher.ListDeviceTierConfigsResponse
	getResp              *androidpublisher.DeviceTierConfig
	createResp           *androidpublisher.DeviceTierConfig
	listErr              error
	getErr               error
	createErr            error
	capturedPageSize     int64
	capturedPageToken    string
	capturedPaginate     bool
	capturedConfigID     int64
	capturedCreateConfig *androidpublisher.DeviceTierConfig
	capturedAllowUnknown bool
}

func (f *fakeClient) ListDeviceTierConfigs(_ context.Context, _ string, pageSize int64, pageToken string, paginate bool) (*androidpublisher.ListDeviceTierConfigsResponse, error) {
	f.capturedPageSize = pageSize
	f.capturedPageToken = pageToken
	f.capturedPaginate = paginate
	return f.listResp, f.listErr
}

func (f *fakeClient) GetDeviceTierConfig(_ context.Context, _ string, deviceTierConfigID int64) (*androidpublisher.DeviceTierConfig, error) {
	f.capturedConfigID = deviceTierConfigID
	return f.getResp, f.getErr
}

func (f *fakeClient) CreateDeviceTierConfig(_ context.Context, _ string, config *androidpublisher.DeviceTierConfig, allowUnknownDevices bool) (*androidpublisher.DeviceTierConfig, error) {
	f.capturedCreateConfig = config
	f.capturedAllowUnknown = allowUnknownDevices
	return f.createResp, f.createErr
}

func runDeviceTierConfigs(t *testing.T, deps Deps, args ...string) (string, error) {
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

func TestDeviceTierConfigsList_ReturnsConfigs(t *testing.T) {
	fc := &fakeClient{
		listResp: &androidpublisher.ListDeviceTierConfigsResponse{
			DeviceTierConfigs: []*androidpublisher.DeviceTierConfig{{DeviceTierConfigId: 1}, {DeviceTierConfigId: 2}},
			NextPageToken:     "next-1",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runDeviceTierConfigs(t, deps, "list", "--package-name", "com.example.app", "--page-size", "50", "--page-token", "next-0")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"deviceTierConfigId":"1"`) || !strings.Contains(out, `"nextPageToken":"next-1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedPageSize != 50 || fc.capturedPageToken != "next-0" || fc.capturedPaginate {
		t.Fatalf("unexpected list args: size=%d token=%q paginate=%t", fc.capturedPageSize, fc.capturedPageToken, fc.capturedPaginate)
	}
}

func TestDeviceTierConfigsGet_RequiresPositiveID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runDeviceTierConfigs(t, deps, "get", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "--device-tier-config-id must be greater than zero") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeviceTierConfigsGet_ReturnsConfig(t *testing.T) {
	fc := &fakeClient{
		getResp: &androidpublisher.DeviceTierConfig{DeviceTierConfigId: 7},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runDeviceTierConfigs(t, deps, "get", "--package-name", "com.example.app", "--device-tier-config-id", "7")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"deviceTierConfigId":"7"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedConfigID != 7 {
		t.Fatalf("unexpected config id: %d", fc.capturedConfigID)
	}
}

func TestDeviceTierConfigsCreate_ReadsPayload(t *testing.T) {
	fc := &fakeClient{
		createResp: &androidpublisher.DeviceTierConfig{DeviceTierConfigId: 9},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	payload := writePayloadFile(t, "device-tier-config.json", `{
		"deviceGroups":[{"name":"phone-group","deviceSelectors":[{"deviceRam":{"minBytes":"2147483648"}}]}]
	}`)
	out, err := runDeviceTierConfigs(t, deps, "create", "--package-name", "com.example.app", "--input", payload, "--allow-unknown-devices")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"deviceTierConfigId":"9"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedCreateConfig == nil || len(fc.capturedCreateConfig.DeviceGroups) != 1 || !fc.capturedAllowUnknown {
		t.Fatalf("unexpected create capture: %#v allowUnknown=%t", fc.capturedCreateConfig, fc.capturedAllowUnknown)
	}
}

func TestDeviceTierConfigsGet_UsesGlobalPackageName(t *testing.T) {
	bindGlobalPackageName(t, "com.example.global")
	fc := &fakeClient{
		getResp: &androidpublisher.DeviceTierConfig{DeviceTierConfigId: 7},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	if _, err := runDeviceTierConfigs(t, deps, "get", "--device-tier-config-id", "7"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
}
