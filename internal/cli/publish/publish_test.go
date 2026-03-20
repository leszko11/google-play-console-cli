package publish

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"google.golang.org/api/androidpublisher/v3"
)

type fakeClient struct {
	createEditErr error
	validateErr   error
	commitErr     error
	deleteErr     error

	uploadBundle    gpc.BundleInfo
	uploadBundleErr error
	uploadAPK       gpc.APKInfo
	uploadAPKErr    error

	uploadMappingErr  error
	lastMappingType   string
	mappingUploadCall int

	updateTrackErr error
	lastTrackName  string
	lastTrack      gpc.TrackUpdate

	deleteCalls     int
	commitCalls     int
	generatedAPKsFn func(context.Context, string, int64) (*androidpublisher.GeneratedApksListResponse, error)
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	if f.createEditErr != nil {
		return gpc.EditInfo{}, f.createEditErr
	}
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return f.deleteErr
}

func (f *fakeClient) ValidateEdit(_ context.Context, _, _ string) error {
	return f.validateErr
}

func (f *fakeClient) CommitEdit(_ context.Context, _, _ string, _ bool) (gpc.EditInfo, error) {
	if f.commitErr != nil {
		return gpc.EditInfo{}, f.commitErr
	}
	f.commitCalls++
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) UpdateTrack(_ context.Context, _, _, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
	f.lastTrackName = trackName
	f.lastTrack = update
	if f.updateTrackErr != nil {
		return gpc.TrackInfo{}, f.updateTrackErr
	}
	return gpc.TrackInfo{Name: trackName}, nil
}

func (f *fakeClient) UploadBundle(_ context.Context, _, _, _ string) (gpc.BundleInfo, error) {
	if f.uploadBundleErr != nil {
		return gpc.BundleInfo{}, f.uploadBundleErr
	}
	if f.uploadBundle.VersionCode == 0 {
		return gpc.BundleInfo{VersionCode: 123}, nil
	}
	return f.uploadBundle, nil
}

func (f *fakeClient) UploadAPK(_ context.Context, _, _, _ string) (gpc.APKInfo, error) {
	if f.uploadAPKErr != nil {
		return gpc.APKInfo{}, f.uploadAPKErr
	}
	if f.uploadAPK.VersionCode == 0 {
		return gpc.APKInfo{VersionCode: 123}, nil
	}
	return f.uploadAPK, nil
}

func (f *fakeClient) UploadDeobfuscationFile(_ context.Context, _, _ string, _ int64, fileType, _ string) (gpc.DeobfuscationFileInfo, error) {
	f.mappingUploadCall++
	f.lastMappingType = fileType
	if f.uploadMappingErr != nil {
		return gpc.DeobfuscationFileInfo{}, f.uploadMappingErr
	}
	return gpc.DeobfuscationFileInfo{SymbolType: fileType}, nil
}

func (f *fakeClient) ListGeneratedAPKs(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.GeneratedApksListResponse, error) {
	if f.generatedAPKsFn == nil {
		return &androidpublisher.GeneratedApksListResponse{}, nil
	}
	return f.generatedAPKsFn(ctx, packageName, versionCode)
}

func runPublish(t *testing.T, deps Deps, args ...string) (string, error) {
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

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func writeTempFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestPublishAlpha_CommitsAndWaitsForAABProcessing(t *testing.T) {
	attempts := 0
	client := &fakeClient{
		generatedAPKsFn: func(_ context.Context, packageName string, versionCode int64) (*androidpublisher.GeneratedApksListResponse, error) {
			attempts++
			if packageName != "com.example.app" {
				t.Fatalf("unexpected package %q", packageName)
			}
			if versionCode != 123 {
				t.Fatalf("unexpected versionCode %d", versionCode)
			}
			if attempts < 2 {
				return &androidpublisher.GeneratedApksListResponse{}, nil
			}
			return &androidpublisher.GeneratedApksListResponse{
				GeneratedApks: []*androidpublisher.GeneratedApksPerSigningKey{{CertificateSha256Hash: "hash"}},
			}, nil
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
		Sleep:      func(context.Context, time.Duration) error { return nil },
	}

	out, err := runPublish(
		t,
		deps,
		"alpha",
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--confirm",
		"--wait-timeout", "50ms",
		"--wait-interval", "1ms",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{`"track":"alpha"`, `"status":"committed"`, `"generatedApkCount":1`, `"attempts":2`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %s", want, out)
		}
	}
	if client.lastTrackName != "alpha" {
		t.Fatalf("expected alpha track, got %q", client.lastTrackName)
	}
	if client.commitCalls != 1 {
		t.Fatalf("expected commit call, got %d", client.commitCalls)
	}
}

func TestPublishProduction_UsesProductionTrack(t *testing.T) {
	client := &fakeClient{
		generatedAPKsFn: func(_ context.Context, _ string, _ int64) (*androidpublisher.GeneratedApksListResponse, error) {
			return &androidpublisher.GeneratedApksListResponse{
				GeneratedApks: []*androidpublisher.GeneratedApksPerSigningKey{{CertificateSha256Hash: "hash"}},
			}, nil
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
		Sleep:      func(context.Context, time.Duration) error { return nil },
	}

	out, err := runPublish(
		t,
		deps,
		"production",
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--confirm",
		"--wait-timeout", "50ms",
		"--wait-interval", "1ms",
		"--status", "inProgress",
		"--user-fraction", "0.2",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if client.lastTrackName != "production" {
		t.Fatalf("expected production track, got %q", client.lastTrackName)
	}
	if client.lastTrack.UserFraction != 0.2 {
		t.Fatalf("unexpected user fraction: %+v", client.lastTrack)
	}
	if !strings.Contains(out, `"track":"production"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestPublishSkipsWaitForAPK(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runPublish(
		t,
		deps,
		"alpha",
		"--package-name", "com.example.app",
		"--apk", writeTempFile(t, "app.apk"),
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if strings.Contains(out, `"status":"ready"`) {
		t.Fatalf("expected wait skip, got %s", out)
	}
	if !strings.Contains(out, `"status":"skipped"`) {
		t.Fatalf("expected skipped wait output, got %s", out)
	}
}

func TestPublishDryRunDeletesEditAndSkipsWait(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runPublish(
		t,
		deps,
		"alpha",
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected delete call, got %d", client.deleteCalls)
	}
	if client.commitCalls != 0 {
		t.Fatalf("expected no commit call, got %d", client.commitCalls)
	}
}

func TestPublishRequiresConfirmUnlessDryRun(t *testing.T) {
	deps := Deps{}

	_, err := runPublish(
		t,
		deps,
		"alpha",
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required unless --dry-run is set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishWaitFailureReturnsErrorAndCleansUp(t *testing.T) {
	client := &fakeClient{
		generatedAPKsFn: func(_ context.Context, _ string, _ int64) (*androidpublisher.GeneratedApksListResponse, error) {
			return nil, errors.New("boom")
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runPublish(
		t,
		deps,
		"alpha",
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--confirm",
		"--wait-timeout", "50ms",
		"--wait-interval", "1ms",
	)
	if err == nil || !strings.Contains(err.Error(), "failed to wait for bundle processing") {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected cleanup delete, got %d", client.deleteCalls)
	}
	if !strings.Contains(out, `"status":"failed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}
