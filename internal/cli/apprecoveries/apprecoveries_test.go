package apprecoveries

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
	listResp             *androidpublisher.ListAppRecoveriesResponse
	createResp           *androidpublisher.AppRecoveryAction
	listErr              error
	createErr            error
	addTargetingErr      error
	cancelErr            error
	deployErr            error
	capturedVersionCode  int64
	capturedRecoveryID   int64
	capturedCreate       *androidpublisher.CreateDraftAppRecoveryRequest
	capturedTargetingReq *androidpublisher.AddTargetingRequest
}

func (f *fakeClient) ListAppRecoveries(_ context.Context, _ string, versionCode int64) (*androidpublisher.ListAppRecoveriesResponse, error) {
	f.capturedVersionCode = versionCode
	return f.listResp, f.listErr
}

func (f *fakeClient) CreateAppRecovery(_ context.Context, _ string, request *androidpublisher.CreateDraftAppRecoveryRequest) (*androidpublisher.AppRecoveryAction, error) {
	f.capturedCreate = request
	return f.createResp, f.createErr
}

func (f *fakeClient) AddAppRecoveryTargeting(_ context.Context, _ string, appRecoveryID int64, request *androidpublisher.AddTargetingRequest) error {
	f.capturedRecoveryID = appRecoveryID
	f.capturedTargetingReq = request
	return f.addTargetingErr
}

func (f *fakeClient) CancelAppRecovery(_ context.Context, _ string, appRecoveryID int64) error {
	f.capturedRecoveryID = appRecoveryID
	return f.cancelErr
}

func (f *fakeClient) DeployAppRecovery(_ context.Context, _ string, appRecoveryID int64) error {
	f.capturedRecoveryID = appRecoveryID
	return f.deployErr
}

func runAppRecoveries(t *testing.T, deps Deps, args ...string) (string, error) {
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

func TestAppRecoveriesList_ReturnsActions(t *testing.T) {
	fc := &fakeClient{
		listResp: &androidpublisher.ListAppRecoveriesResponse{
			RecoveryActions: []*androidpublisher.AppRecoveryAction{
				{AppRecoveryId: 7, Status: "RECOVERY_STATUS_DRAFT"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runAppRecoveries(t, deps, "list", "--package-name", "com.example.app", "--version-code", "123")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"appRecoveryId":"7"`) || !strings.Contains(out, `"RECOVERY_STATUS_DRAFT"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedVersionCode != 123 {
		t.Fatalf("unexpected version code: %d", fc.capturedVersionCode)
	}
}

func TestAppRecoveriesCreate_ReadsPayload(t *testing.T) {
	fc := &fakeClient{
		createResp: &androidpublisher.AppRecoveryAction{AppRecoveryId: 7, Status: "RECOVERY_STATUS_DRAFT"},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	payload := writePayloadFile(t, "create.json", `{
		"remoteInAppUpdate":{"isRemoteInAppUpdateRequested":true},
		"targeting":{"allUsers":{"isAllUsersRequested":true}}
	}`)
	out, err := runAppRecoveries(t, deps, "create", "--package-name", "com.example.app", "--input", payload)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"appRecoveryId":"7"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedCreate == nil || fc.capturedCreate.RemoteInAppUpdate == nil || !fc.capturedCreate.RemoteInAppUpdate.IsRemoteInAppUpdateRequested {
		t.Fatalf("unexpected create payload: %#v", fc.capturedCreate)
	}
}

func TestAppRecoveriesAddTargeting_ReadsPayload(t *testing.T) {
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	payload := writePayloadFile(t, "targeting.json", `{
		"targetingUpdate":{"regions":{"regionCodes":["US","PL"]}}
	}`)
	out, err := runAppRecoveries(t, deps, "add-targeting", "--package-name", "com.example.app", "--app-recovery-id", "7", "--input", payload)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"targeting-updated"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedRecoveryID != 7 || fc.capturedTargetingReq == nil || fc.capturedTargetingReq.TargetingUpdate == nil || fc.capturedTargetingReq.TargetingUpdate.Regions == nil {
		t.Fatalf("unexpected targeting capture: %#v", fc.capturedTargetingReq)
	}
}

func TestAppRecoveriesCancel_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runAppRecoveries(t, deps, "cancel", "--package-name", "com.example.app", "--app-recovery-id", "7")
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppRecoveriesCancel_ReturnsStatus(t *testing.T) {
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runAppRecoveries(t, deps, "cancel", "--package-name", "com.example.app", "--app-recovery-id", "7", "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"canceled"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedRecoveryID != 7 {
		t.Fatalf("unexpected app recovery id: %d", fc.capturedRecoveryID)
	}
}

func TestAppRecoveriesDeploy_ReturnsStatus(t *testing.T) {
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runAppRecoveries(t, deps, "deploy", "--package-name", "com.example.app", "--app-recovery-id", "7", "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deployed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedRecoveryID != 7 {
		t.Fatalf("unexpected app recovery id: %d", fc.capturedRecoveryID)
	}
}

func TestAppRecoveriesList_UsesGlobalPackageName(t *testing.T) {
	bindGlobalPackageName(t, "com.example.global")
	fc := &fakeClient{
		listResp: &androidpublisher.ListAppRecoveriesResponse{},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	if _, err := runAppRecoveries(t, deps, "list", "--version-code", "123"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
}
