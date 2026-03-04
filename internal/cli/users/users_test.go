package users

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
	list      gpc.UsersListInfo
	listErr   error
	create    gpc.UserInfo
	createErr error
	update    gpc.UserInfo
	updateErr error
	deleteErr error

	listFn   func(developerID string, pageSize int64, pageToken string, paginate bool) (gpc.UsersListInfo, error)
	createFn func(developerID string, user *androidpublisher.User) (gpc.UserInfo, error)
	updateFn func(name string, user *androidpublisher.User, updateMask string) (gpc.UserInfo, error)
}

func (f fakeClient) ListUsers(_ context.Context, developerID string, pageSize int64, pageToken string, paginate bool) (gpc.UsersListInfo, error) {
	if f.listFn != nil {
		return f.listFn(developerID, pageSize, pageToken, paginate)
	}
	return f.list, f.listErr
}

func (f fakeClient) CreateUser(_ context.Context, developerID string, user *androidpublisher.User) (gpc.UserInfo, error) {
	if f.createFn != nil {
		return f.createFn(developerID, user)
	}
	return f.create, f.createErr
}

func (f fakeClient) UpdateUser(_ context.Context, name string, user *androidpublisher.User, updateMask string) (gpc.UserInfo, error) {
	if f.updateFn != nil {
		return f.updateFn(name, user, updateMask)
	}
	return f.update, f.updateErr
}

func (f fakeClient) DeleteUser(_ context.Context, _ string) error { return f.deleteErr }

func runUsers(t *testing.T, deps Deps, args ...string) (string, error) {
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

func TestUsersList_ReturnsUsers(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				list: gpc.UsersListInfo{
					Users: []gpc.UserInfo{{Name: "developers/123/users/dev@example.com", Email: "dev@example.com"}},
				},
			}, nil
		},
	}

	out, err := runUsers(t, deps, "list", "--developer-id", "123")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"developerId":"123"`) || !strings.Contains(out, `"email":"dev@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUsersList_RequiresDeveloperID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runUsers(t, deps, "list")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--developer-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsersList_UsesConfiguredDeveloperID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return configWithDeveloperID("1234567890123456789"), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				listFn: func(developerID string, _ int64, _ string, _ bool) (gpc.UsersListInfo, error) {
					if developerID != "1234567890123456789" {
						t.Fatalf("unexpected developer id: %q", developerID)
					}
					return gpc.UsersListInfo{
						Users: []gpc.UserInfo{{Name: "developers/1234567890123456789/users/dev@example.com", Email: "dev@example.com"}},
					}, nil
				},
			}, nil
		},
	}

	out, err := runUsers(t, deps, "list")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"developerId":"1234567890123456789"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUsersList_DefaultPageSizeIsMinusOne(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return configWithDeveloperID("1234567890123456789"), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				listFn: func(_ string, pageSize int64, _ string, _ bool) (gpc.UsersListInfo, error) {
					if pageSize != -1 {
						t.Fatalf("expected page size -1, got %d", pageSize)
					}
					return gpc.UsersListInfo{}, nil
				},
			}, nil
		},
	}

	if _, err := runUsers(t, deps, "list"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestUsersList_InvalidPageSize(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return configWithDeveloperID("1234567890123456789"), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runUsers(t, deps, "list", "--page-size", "-2")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--page-size must be -1 or greater") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsersCreate_ReturnsCreated(t *testing.T) {
	inputPath := writeJSON(t, `{"name":"developers/123/users/dev@example.com","email":"dev@example.com"}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				createFn: func(developerID string, user *androidpublisher.User) (gpc.UserInfo, error) {
					if developerID != "123" {
						t.Fatalf("unexpected developer id: %q", developerID)
					}
					if user.Email != "dev@example.com" {
						t.Fatalf("unexpected payload email: %q", user.Email)
					}
					return gpc.UserInfo{Name: user.Name, Email: user.Email}, nil
				},
			}, nil
		},
	}

	out, err := runUsers(t, deps, "create", "--developer-id", "123", "--input", inputPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"created"`) || !strings.Contains(out, `"email":"dev@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUsersCreate_InvalidJSON(t *testing.T) {
	inputPath := writeJSON(t, `{not-json}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runUsers(t, deps, "create", "--developer-id", "123", "--input", inputPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid user JSON payload") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsersCreate_UsesConfiguredDeveloperID(t *testing.T) {
	inputPath := writeJSON(t, `{"name":"developers/123/users/dev@example.com","email":"dev@example.com"}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return configWithDeveloperID("1234567890123456789"), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				createFn: func(developerID string, user *androidpublisher.User) (gpc.UserInfo, error) {
					if developerID != "1234567890123456789" {
						t.Fatalf("unexpected developer id: %q", developerID)
					}
					return gpc.UserInfo{Name: user.Name, Email: user.Email}, nil
				},
			}, nil
		},
	}

	out, err := runUsers(t, deps, "create", "--input", inputPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"developerId":"1234567890123456789"`) || !strings.Contains(out, `"status":"created"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUsersUpdate_ReturnsUpdated(t *testing.T) {
	inputPath := writeJSON(t, `{"expirationTime":"2026-05-01T00:00:00Z"}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateFn: func(name string, user *androidpublisher.User, updateMask string) (gpc.UserInfo, error) {
					if name != "developers/123/users/dev@example.com" {
						t.Fatalf("unexpected name: %q", name)
					}
					if updateMask != "expirationTime" {
						t.Fatalf("unexpected update mask: %q", updateMask)
					}
					return gpc.UserInfo{Name: name, ExpirationTime: user.ExpirationTime}, nil
				},
			}, nil
		},
	}

	out, err := runUsers(
		t,
		deps,
		"update",
		"--name", "developers/123/users/dev@example.com",
		"--input", inputPath,
		"--update-mask", "expirationTime",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"expirationTime":"2026-05-01T00:00:00Z"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUsersUpdate_RequiresName(t *testing.T) {
	inputPath := writeJSON(t, `{"email":"dev@example.com"}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runUsers(t, deps, "update", "--input", inputPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsersUpdate_UsesStoredDeveloperID(t *testing.T) {
	inputPath := writeJSON(t, `{"expirationTime":"2026-05-01T00:00:00Z"}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return configWithDeveloperID("1234567890123456789"), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateFn: func(name string, user *androidpublisher.User, updateMask string) (gpc.UserInfo, error) {
					if name != "developers/1234567890123456789/users/dev@example.com" {
						t.Fatalf("unexpected name: %q", name)
					}
					if updateMask != "expirationTime" {
						t.Fatalf("unexpected update mask: %q", updateMask)
					}
					return gpc.UserInfo{Name: name, ExpirationTime: user.ExpirationTime}, nil
				},
			}, nil
		},
	}

	out, err := runUsers(
		t,
		deps,
		"update",
		"--user-email", "dev@example.com",
		"--input", inputPath,
		"--update-mask", "expirationTime",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"name":"developers/1234567890123456789/users/dev@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUsersDelete_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runUsers(t, deps, "delete", "--name", "developers/123/users/dev@example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsersDelete_ReturnsDeleted(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	out, err := runUsers(t, deps, "delete", "--name", "developers/123/users/dev@example.com", "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deleted"`) || !strings.Contains(out, `"name":"developers/123/users/dev@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUsersDelete_UsesStoredDeveloperID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return configWithDeveloperID("1234567890123456789"), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	out, err := runUsers(t, deps, "delete", "--user-email", "dev@example.com", "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"name":"developers/1234567890123456789/users/dev@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUsersDelete_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{deleteErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runUsers(t, deps, "delete", "--name", "developers/123/users/dev@example.com", "--confirm")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete user") {
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
