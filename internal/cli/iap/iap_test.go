package iap

import (
	"bytes"
	"context"
	"errors"
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
	list      gpc.IAPsListInfo
	listErr   error
	get       gpc.IAPInfo
	getErr    error
	create    gpc.IAPInfo
	createErr error
	update    gpc.IAPInfo
	updateErr error
	deleteErr error

	createFn func(packageName string, product *androidpublisher.InAppProduct) (gpc.IAPInfo, error)
	updateFn func(packageName, sku string, product *androidpublisher.InAppProduct) (gpc.IAPInfo, error)
	capture  *iapListCapture
}

func (f fakeClient) ListIAPs(_ context.Context, _ string, maxResults int64, pageToken string, paginate bool) (gpc.IAPsListInfo, error) {
	if f.capture != nil {
		f.capture.maxResults = maxResults
		f.capture.pageToken = pageToken
		f.capture.paginate = paginate
	}
	return f.list, f.listErr
}

func (f fakeClient) GetIAP(_ context.Context, _, _ string) (gpc.IAPInfo, error) {
	return f.get, f.getErr
}

func (f fakeClient) CreateIAP(_ context.Context, packageName string, product *androidpublisher.InAppProduct) (gpc.IAPInfo, error) {
	if f.createFn != nil {
		return f.createFn(packageName, product)
	}
	return f.create, f.createErr
}

func (f fakeClient) UpdateIAP(_ context.Context, packageName, sku string, product *androidpublisher.InAppProduct) (gpc.IAPInfo, error) {
	if f.updateFn != nil {
		return f.updateFn(packageName, sku, product)
	}
	return f.update, f.updateErr
}

func (f fakeClient) DeleteIAP(_ context.Context, _, _ string) error { return f.deleteErr }

func runIAP(t *testing.T, deps Deps, args ...string) (string, error) {
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

type iapListCapture struct {
	maxResults int64
	pageToken  string
	paginate   bool
}

func bindGlobalPaginate(t *testing.T, paginate bool) {
	t.Helper()
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	cfg := &shared.GlobalFlags{}
	shared.BindGlobalFlags(fs, cfg)
	cfg.Paginate = paginate
}

func TestIAPList_ReturnsProducts(t *testing.T) {
	bindGlobalPaginate(t, false)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				list: gpc.IAPsListInfo{
					Products: []gpc.IAPInfo{{SKU: "coins_100"}},
				},
			}, nil
		},
	}

	out, err := runIAP(t, deps, "list", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"sku":"coins_100"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestIAPList_UsesGlobalPaginate(t *testing.T) {
	bindGlobalPaginate(t, true)
	capture := &iapListCapture{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{capture: capture}, nil
		},
	}

	_, err := runIAP(t, deps, "list", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !capture.paginate {
		t.Fatal("expected paginate=true from global flags")
	}
}

func TestIAPGet_RequiresSKU(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}
	_, err := runIAP(t, deps, "get", "--package-name", "com.example.app")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--sku is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIAPCreate_ReturnsCreated(t *testing.T) {
	inputPath := writeJSON(t, `{"sku":"coins_100","packageName":"com.example.app"}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				createFn: func(packageName string, product *androidpublisher.InAppProduct) (gpc.IAPInfo, error) {
					if packageName != "com.example.app" {
						t.Fatalf("unexpected package name: %q", packageName)
					}
					if product.Sku != "coins_100" {
						t.Fatalf("unexpected payload sku: %q", product.Sku)
					}
					return gpc.IAPInfo{PackageName: packageName, SKU: product.Sku}, nil
				},
			}, nil
		},
	}

	out, err := runIAP(t, deps, "create", "--package-name", "com.example.app", "--input", inputPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"created"`) || !strings.Contains(out, `"sku":"coins_100"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestIAPUpdate_ReturnsUpdated(t *testing.T) {
	inputPath := writeJSON(t, `{"status":"active"}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateFn: func(packageName, sku string, _ *androidpublisher.InAppProduct) (gpc.IAPInfo, error) {
					if packageName != "com.example.app" || sku != "coins_100" {
						t.Fatalf("unexpected args: package=%q sku=%q", packageName, sku)
					}
					return gpc.IAPInfo{PackageName: packageName, SKU: sku, Status: "active"}, nil
				},
			}, nil
		},
	}

	out, err := runIAP(
		t,
		deps,
		"update",
		"--package-name", "com.example.app",
		"--sku", "coins_100",
		"--input", inputPath,
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"sku":"coins_100"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestIAPDelete_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runIAP(t, deps, "delete", "--package-name", "com.example.app", "--sku", "coins_100")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIAPDelete_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{deleteErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runIAP(t, deps, "delete", "--package-name", "com.example.app", "--sku", "coins_100", "--confirm")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete in-app product") {
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
