package apps

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	verify          map[string]error
	get             map[string]gpc.AppInfo
	getErr          map[string]error
	setDataSafetyFn func(context.Context, string, string) error
}

func (f fakeClient) VerifyPackageAccess(_ context.Context, packageName string) error {
	if f.verify == nil {
		return nil
	}
	return f.verify[packageName]
}

func (f fakeClient) GetApp(_ context.Context, packageName string) (gpc.AppInfo, error) {
	if err := f.getErr[packageName]; err != nil {
		return gpc.AppInfo{}, err
	}
	if app, ok := f.get[packageName]; ok {
		return app, nil
	}
	return gpc.AppInfo{PackageName: packageName}, nil
}

func (f fakeClient) SetDataSafety(ctx context.Context, packageName, safetyLabelsCSV string) error {
	if f.setDataSafetyFn != nil {
		return f.setDataSafetyFn(ctx, packageName, safetyLabelsCSV)
	}
	return nil
}

func runApps(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}

	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestAppsList_DefaultJSON(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{Packages: []string{"com.example.one", "com.example.two"}}, nil
		},
	}

	out, err := runApps(t, deps, "list", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"packageName":"com.example.one"`) {
		t.Fatalf("missing package in output: %s", out)
	}
	if !strings.Contains(out, `"packageName":"com.example.two"`) {
		t.Fatalf("missing package in output: %s", out)
	}
}

func TestAppsListVerify_IncludesStatus(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Packages:      []string{"com.example.ok", "com.example.bad"},
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/sa.json"},
				},
			}, nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				verify: map[string]error{
					"com.example.bad": errors.New("403 forbidden"),
				},
			}, nil
		},
	}

	out, err := runApps(t, deps, "list", "--verify", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("expected ok status in output: %s", out)
	}
	if !strings.Contains(out, `"status":"error"`) {
		t.Fatalf("expected error status in output: %s", out)
	}
}

func TestAppsGet_ReturnsClearAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/sa.json"},
				},
			}, nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				getErr: map[string]error{"com.example.missing": gpc.ErrPackageNotFound},
			}, nil
		},
	}

	_, err := runApps(t, deps, "get", "--package-name", "com.example.missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "package not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppsAddPackage_PersistsToConfig(t *testing.T) {
	stored := config.Config{
		Packages: []string{"com.example.one"},
	}

	deps := Deps{
		LoadConfig: func() (config.Config, error) { return stored, nil },
		SaveConfig: func(cfg config.Config) error {
			stored = cfg
			return nil
		},
	}

	out, err := runApps(t, deps, "add-package", "--package-name", "com.example.two")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"action":"added"`) {
		t.Fatalf("expected added action, got: %s", out)
	}
	if len(stored.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %v", stored.Packages)
	}
}

func TestAppsRemovePackage_RemovesFromConfig(t *testing.T) {
	stored := config.Config{
		Packages: []string{"com.example.one", "com.example.two"},
	}

	deps := Deps{
		LoadConfig: func() (config.Config, error) { return stored, nil },
		SaveConfig: func(cfg config.Config) error {
			stored = cfg
			return nil
		},
	}

	out, err := runApps(t, deps, "remove-package", "--package-name", "com.example.one")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"action":"removed"`) {
		t.Fatalf("expected removed action, got: %s", out)
	}
	if len(stored.Packages) != 1 || stored.Packages[0] != "com.example.two" {
		t.Fatalf("unexpected packages after remove: %v", stored.Packages)
	}
}
