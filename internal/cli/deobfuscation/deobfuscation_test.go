package deobfuscation

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
	result gpc.DeobfuscationFileInfo
	err    error
}

func (f fakeClient) UploadDeobfuscationFile(_ context.Context, _, _ string, _ int64, _, _ string) (gpc.DeobfuscationFileInfo, error) {
	if f.err != nil {
		return gpc.DeobfuscationFileInfo{}, f.err
	}
	return f.result, nil
}

func runDeobfuscation(t *testing.T, deps Deps, args ...string) (string, error) {
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

func writeTempFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mapping.txt")
	if err := os.WriteFile(path, []byte("mapping"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestDeobfuscationUpload_Success(t *testing.T) {
	filePath := writeTempFile(t)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{result: gpc.DeobfuscationFileInfo{SymbolType: "proguard"}}, nil
		},
	}

	out, err := runDeobfuscation(
		t,
		deps,
		"upload",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--version-code", "123",
		"--type", "proguard",
		"--file", filePath,
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"uploaded"`) || !strings.Contains(out, `"symbolType":"proguard"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestDeobfuscationUpload_RequiresSupportedType(t *testing.T) {
	filePath := writeTempFile(t)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runDeobfuscation(
		t,
		deps,
		"upload",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--version-code", "123",
		"--type", "unsupported",
		"--file", filePath,
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--type must be one of") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeobfuscationUpload_ValidatesFileBeforeAPI(t *testing.T) {
	clientCreated := false
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			clientCreated = true
			return fakeClient{}, nil
		},
	}

	_, err := runDeobfuscation(
		t,
		deps,
		"upload",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--version-code", "123",
		"--type", "proguard",
		"--file", filepath.Join(t.TempDir(), "missing.txt"),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--file does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientCreated {
		t.Fatal("expected client not to be created when local file validation fails")
	}
}

func TestDeobfuscationUpload_ReturnsAPIError(t *testing.T) {
	filePath := writeTempFile(t)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{err: errors.New("conflict")}, nil
		},
	}

	_, err := runDeobfuscation(
		t,
		deps,
		"upload",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--version-code", "123",
		"--type", "proguard",
		"--file", filePath,
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to upload deobfuscation file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
