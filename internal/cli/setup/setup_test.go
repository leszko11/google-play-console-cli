package setup

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeVerifyClient struct {
	verifyErr       error
	capturedPackage string
}

func (f *fakeVerifyClient) VerifyPackageAccess(_ context.Context, packageName string) error {
	f.capturedPackage = packageName
	return f.verifyErr
}

func TestSetupAutoSuccessBootstrapsWorkspace(t *testing.T) {
	var gcloudCalls [][]string
	var authArgs []string
	var bootstrapArgs []string
	verifyClient := &fakeVerifyClient{}
	keyPath := filepath.Join(t.TempDir(), ".gpc", "default-service-account.json")

	deps := Deps{
		RunCommand: func(_ context.Context, _ string, name string, args ...string) (string, error) {
			if name != "gcloud" {
				t.Fatalf("unexpected command: %s %v", name, args)
			}
			gcloudCalls = append(gcloudCalls, append([]string{name}, args...))
			switch {
			case len(args) >= 1 && args[0] == "--version":
				return "Google Cloud SDK 999.0.0", nil
			case len(args) >= 3 && args[0] == "iam" && args[1] == "service-accounts" && args[2] == "describe":
				return "", errExit1
			case len(args) >= 3 && args[0] == "iam" && args[1] == "service-accounts" && args[2] == "create":
				return "", nil
			case len(args) >= 3 && args[0] == "services" && args[1] == "enable":
				return "", nil
			case len(args) >= 3 && args[0] == "iam" && args[1] == "service-accounts" && args[2] == "keys":
				if err := os.WriteFile(keyPath, []byte(`{"type":"service_account"}`), 0o600); err != nil {
					t.Fatalf("write key: %v", err)
				}
				return "", nil
			default:
				return "", nil
			}
		},
		RunAuthInit: func(_ context.Context, args []string) error {
			authArgs = append([]string(nil), args...)
			return nil
		},
		RunBootstrap: func(_ context.Context, args []string) error {
			bootstrapArgs = append([]string(nil), args...)
			return nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (VerifyClient, error) {
			return verifyClient, nil
		},
		Now:    func() time.Time { return time.Unix(1_750_000_000, 0) },
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	out, err := runSetup(t, deps, "--auto", "--project-id", "play-prod", "--package-name", "com.example.app", "--dir", "./play", "--service-account-key", keyPath, "--developer-id", "123456")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"ok"`) || !strings.Contains(out, `"serviceAccountEmail":"gpc-default@play-prod.iam.gserviceaccount.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if verifyClient.capturedPackage != "com.example.app" {
		t.Fatalf("expected package verification, got %q", verifyClient.capturedPackage)
	}
	if len(authArgs) == 0 || !contains(authArgs, "--service-account") || !contains(authArgs, keyPath) {
		t.Fatalf("unexpected auth args: %v", authArgs)
	}
	if len(bootstrapArgs) == 0 || !contains(bootstrapArgs, "--write-project-config") || !contains(bootstrapArgs, "./play") {
		t.Fatalf("unexpected bootstrap args: %v", bootstrapArgs)
	}
	if len(gcloudCalls) == 0 {
		t.Fatalf("expected gcloud calls")
	}
}

func TestSetupAutoWarnsWhenPackageAccessFails(t *testing.T) {
	var bootstrapCalled bool
	verifyClient := &fakeVerifyClient{verifyErr: gpc.ErrAccessDenied}
	keyDir := t.TempDir()
	keyPath := filepath.Join(keyDir, "setup-key.json")
	if err := os.WriteFile(keyPath, []byte(`{"type":"service_account"}`), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	deps := Deps{
		RunCommand: func(_ context.Context, _ string, name string, args ...string) (string, error) {
			if name != "gcloud" {
				t.Fatalf("unexpected command: %s %v", name, args)
			}
			if len(args) >= 1 && args[0] == "--version" {
				return "Google Cloud SDK 999.0.0", nil
			}
			return "", nil
		},
		RunAuthInit: func(_ context.Context, args []string) error { return nil },
		RunBootstrap: func(_ context.Context, args []string) error {
			bootstrapCalled = true
			return nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (VerifyClient, error) {
			return verifyClient, nil
		},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	out, err := runSetup(t, deps, "--auto", "--project-id", "play-prod", "--package-name", "com.example.app", "--service-account-key", keyPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"warn"`) || !strings.Contains(out, `"playConsoleAccess"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if bootstrapCalled {
		t.Fatalf("bootstrap should not run when package access fails")
	}
}

func TestSetupAutoRequiresProjectID(t *testing.T) {
	deps := Deps{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	_, err := runSetup(t, deps, "--auto")
	if err == nil || !strings.Contains(err.Error(), "--project-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func runSetup(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	if deps.Stderr == nil {
		deps.Stderr = &bytes.Buffer{}
	}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

var errExit1 = &exitError{}

type exitError struct{}

func (e *exitError) Error() string { return "exit status 1" }
