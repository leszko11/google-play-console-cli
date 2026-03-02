package internalsharing

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
)

type fakeClient struct {
	apkArtifact    gpc.InternalSharingArtifactInfo
	bundleArtifact gpc.InternalSharingArtifactInfo
	apkErr         error
	bundleErr      error

	uploadAPKFn    func(packageName, apkPath string) (gpc.InternalSharingArtifactInfo, error)
	uploadBundleFn func(packageName, bundlePath string) (gpc.InternalSharingArtifactInfo, error)
}

func (f fakeClient) UploadInternalSharingAPK(_ context.Context, packageName, apkPath string) (gpc.InternalSharingArtifactInfo, error) {
	if f.uploadAPKFn != nil {
		return f.uploadAPKFn(packageName, apkPath)
	}
	return f.apkArtifact, f.apkErr
}

func (f fakeClient) UploadInternalSharingBundle(_ context.Context, packageName, bundlePath string) (gpc.InternalSharingArtifactInfo, error) {
	if f.uploadBundleFn != nil {
		return f.uploadBundleFn(packageName, bundlePath)
	}
	return f.bundleArtifact, f.bundleErr
}

func runInternalSharing(t *testing.T, deps Deps, args ...string) (string, error) {
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

func TestInternalSharingUploadAPK_ReturnsUploaded(t *testing.T) {
	apkPath := writeBinary(t, "test.apk", []byte("apk-bytes"))
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				uploadAPKFn: func(packageName, gotPath string) (gpc.InternalSharingArtifactInfo, error) {
					if packageName != "com.example.app" {
						t.Fatalf("unexpected package name: %q", packageName)
					}
					if gotPath != apkPath {
						t.Fatalf("unexpected apk path: %q", gotPath)
					}
					return gpc.InternalSharingArtifactInfo{DownloadURL: "https://download.example/apk"}, nil
				},
			}, nil
		},
	}

	out, err := runInternalSharing(t, deps, "upload", "--package-name", "com.example.app", "--apk", apkPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"artifactType":"apk"`) || !strings.Contains(out, `"status":"uploaded"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestInternalSharingUploadAAB_ReturnsUploaded(t *testing.T) {
	aabPath := writeBinary(t, "test.aab", []byte("aab-bytes"))
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				uploadBundleFn: func(packageName, gotPath string) (gpc.InternalSharingArtifactInfo, error) {
					if packageName != "com.example.app" {
						t.Fatalf("unexpected package name: %q", packageName)
					}
					if gotPath != aabPath {
						t.Fatalf("unexpected aab path: %q", gotPath)
					}
					return gpc.InternalSharingArtifactInfo{DownloadURL: "https://download.example/aab"}, nil
				},
			}, nil
		},
	}

	out, err := runInternalSharing(t, deps, "upload", "--package-name", "com.example.app", "--aab", aabPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"artifactType":"aab"`) || !strings.Contains(out, `"status":"uploaded"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestInternalSharingUpload_RequiresExactlyOneArtifactFlag(t *testing.T) {
	apkPath := writeBinary(t, "test.apk", []byte("apk"))
	aabPath := writeBinary(t, "test.aab", []byte("aab"))
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runInternalSharing(t, deps, "upload", "--package-name", "com.example.app")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "exactly one of --apk or --aab is required") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = runInternalSharing(t, deps, "upload", "--package-name", "com.example.app", "--apk", apkPath, "--aab", aabPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "exactly one of --apk or --aab is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInternalSharingUpload_ValidatesReadableFile(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runInternalSharing(t, deps, "upload", "--package-name", "com.example.app", "--apk", "/tmp/does-not-exist.apk")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--apk does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInternalSharingUpload_ReturnsAPIError(t *testing.T) {
	apkPath := writeBinary(t, "test.apk", []byte("apk-bytes"))
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{apkErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runInternalSharing(t, deps, "upload", "--package-name", "com.example.app", "--apk", apkPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to upload internal sharing artifact") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeBinary(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	return path
}
