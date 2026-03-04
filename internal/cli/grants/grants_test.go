package grants

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
	create    gpc.GrantInfo
	createErr error
	update    gpc.GrantInfo
	updateErr error
	deleteErr error

	createFn func(parent string, grant *androidpublisher.Grant) (gpc.GrantInfo, error)
	updateFn func(name string, grant *androidpublisher.Grant, updateMask string) (gpc.GrantInfo, error)
}

func (f fakeClient) CreateGrant(_ context.Context, parent string, grant *androidpublisher.Grant) (gpc.GrantInfo, error) {
	if f.createFn != nil {
		return f.createFn(parent, grant)
	}
	return f.create, f.createErr
}

func (f fakeClient) UpdateGrant(_ context.Context, name string, grant *androidpublisher.Grant, updateMask string) (gpc.GrantInfo, error) {
	if f.updateFn != nil {
		return f.updateFn(name, grant, updateMask)
	}
	return f.update, f.updateErr
}

func (f fakeClient) DeleteGrant(_ context.Context, _ string) error { return f.deleteErr }

func runGrants(t *testing.T, deps Deps, args ...string) (string, error) {
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

func configWithDeveloperID(developerID string) config.Config {
	cfg := defaultConfig()
	profile := cfg.Profiles["default"]
	profile.DeveloperID = developerID
	cfg.Profiles["default"] = profile
	return cfg
}

func TestGrantsCreate_ReturnsCreated(t *testing.T) {
	inputPath := writeJSON(t, `{"name":"developers/123/users/dev@example.com/grants/com.example.app","packageName":"com.example.app","appLevelPermissions":["CAN_VIEW_NON_FINANCIAL_DATA"]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				createFn: func(parent string, grant *androidpublisher.Grant) (gpc.GrantInfo, error) {
					if parent != "developers/123/users/dev@example.com" {
						t.Fatalf("unexpected parent: %q", parent)
					}
					if grant.PackageName != "com.example.app" {
						t.Fatalf("unexpected package name: %q", grant.PackageName)
					}
					return gpc.GrantInfo{Name: grant.Name, PackageName: grant.PackageName}, nil
				},
			}, nil
		},
	}

	out, err := runGrants(
		t,
		deps,
		"create",
		"--parent", "developers/123/users/dev@example.com",
		"--input", inputPath,
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"created"`) || !strings.Contains(out, `"packageName":"com.example.app"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestGrantsCreate_InvalidJSON(t *testing.T) {
	inputPath := writeJSON(t, `{not-json}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runGrants(t, deps, "create", "--parent", "developers/123/users/dev@example.com", "--input", inputPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid grant JSON payload") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGrantsCreate_UsesStoredDeveloperID(t *testing.T) {
	inputPath := writeJSON(t, `{"packageName":"com.example.app","appLevelPermissions":["CAN_VIEW_NON_FINANCIAL_DATA"]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return configWithDeveloperID("1234567890123456789"), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				createFn: func(parent string, _ *androidpublisher.Grant) (gpc.GrantInfo, error) {
					if parent != "developers/1234567890123456789/users/dev@example.com" {
						t.Fatalf("unexpected parent: %q", parent)
					}
					return gpc.GrantInfo{Name: parent + "/grants/com.example.app", PackageName: "com.example.app"}, nil
				},
			}, nil
		},
	}

	out, err := runGrants(t, deps, "create", "--user-email", "dev@example.com", "--input", inputPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"parent":"developers/1234567890123456789/users/dev@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestGrantsUpdate_ReturnsUpdated(t *testing.T) {
	inputPath := writeJSON(t, `{"appLevelPermissions":["CAN_REPLY_TO_REVIEWS"]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateFn: func(name string, grant *androidpublisher.Grant, updateMask string) (gpc.GrantInfo, error) {
					if name != "developers/123/users/dev@example.com/grants/com.example.app" {
						t.Fatalf("unexpected name: %q", name)
					}
					if updateMask != "appLevelPermissions" {
						t.Fatalf("unexpected update mask: %q", updateMask)
					}
					return gpc.GrantInfo{
						Name:                name,
						AppLevelPermissions: grant.AppLevelPermissions,
						PermissionCount:     len(grant.AppLevelPermissions),
					}, nil
				},
			}, nil
		},
	}

	out, err := runGrants(
		t,
		deps,
		"update",
		"--name", "developers/123/users/dev@example.com/grants/com.example.app",
		"--input", inputPath,
		"--update-mask", "appLevelPermissions",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"permissionCount":1`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestGrantsUpdate_RequiresName(t *testing.T) {
	inputPath := writeJSON(t, `{"packageName":"com.example.app"}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runGrants(t, deps, "update", "--input", inputPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGrantsUpdate_UsesStoredDeveloperIDAndPackage(t *testing.T) {
	inputPath := writeJSON(t, `{"appLevelPermissions":["CAN_REPLY_TO_REVIEWS"]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return configWithDeveloperID("1234567890123456789"), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateFn: func(name string, _ *androidpublisher.Grant, _ string) (gpc.GrantInfo, error) {
					if name != "developers/1234567890123456789/users/dev@example.com/grants/com.example.app" {
						t.Fatalf("unexpected name: %q", name)
					}
					return gpc.GrantInfo{Name: name}, nil
				},
			}, nil
		},
	}

	out, err := runGrants(
		t,
		deps,
		"update",
		"--user-email", "dev@example.com",
		"--package-name", "com.example.app",
		"--input", inputPath,
		"--update-mask", "appLevelPermissions",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"name":"developers/1234567890123456789/users/dev@example.com/grants/com.example.app"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestGrantsDelete_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runGrants(t, deps, "delete", "--name", "developers/123/users/dev@example.com/grants/com.example.app")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGrantsDelete_ReturnsDeleted(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	out, err := runGrants(t, deps, "delete", "--name", "developers/123/users/dev@example.com/grants/com.example.app", "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deleted"`) || !strings.Contains(out, `"name":"developers/123/users/dev@example.com/grants/com.example.app"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestGrantsDelete_UsesStoredDeveloperIDAndPackage(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return configWithDeveloperID("1234567890123456789"), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	out, err := runGrants(
		t,
		deps,
		"delete",
		"--user-email", "dev@example.com",
		"--package-name", "com.example.app",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"name":"developers/1234567890123456789/users/dev@example.com/grants/com.example.app"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestGrantsDelete_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{deleteErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runGrants(t, deps, "delete", "--name", "developers/123/users/dev@example.com/grants/com.example.app", "--confirm")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete grant") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeJSON(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}
