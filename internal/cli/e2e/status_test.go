package e2e

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
	verifyPackageAccessErr  error
	subscriptionDiagnostic  gpc.SubscriptionDiagnosticInfo
	subscriptionErr         error
	oneTimeProductDiag      gpc.OneTimeProductDiagnosticInfo
	oneTimeProductErr       error
	orderInfo               gpc.OrderInfo
	orderErr                error
	subscriptionPurchase    gpc.SubscriptionPurchaseInfo
	subscriptionPurchaseErr error
	productPurchase         gpc.ProductPurchaseInfo
	productPurchaseErr      error
	productPurchaseV2       gpc.ProductPurchaseV2Info
	productPurchaseV2Err    error
	externalTransaction     *androidpublisher.ExternalTransaction
	externalTransactionErr  error
	generatedResp           *androidpublisher.GeneratedApksListResponse
	generatedErr            error
	systemResp              *androidpublisher.SystemApksListResponse
	systemErr               error
	recoveriesResp          *androidpublisher.ListAppRecoveriesResponse
	recoveriesErr           error
}

func (f *fakeClient) VerifyPackageAccess(_ context.Context, _ string) error {
	return f.verifyPackageAccessErr
}

func (f *fakeClient) GetSubscriptionDiagnostic(_ context.Context, _, _ string) (gpc.SubscriptionDiagnosticInfo, error) {
	return f.subscriptionDiagnostic, f.subscriptionErr
}

func (f *fakeClient) GetOneTimeProductDiagnostic(_ context.Context, _, _ string) (gpc.OneTimeProductDiagnosticInfo, error) {
	return f.oneTimeProductDiag, f.oneTimeProductErr
}

func (f *fakeClient) GetOrder(_ context.Context, _, _ string) (gpc.OrderInfo, error) {
	return f.orderInfo, f.orderErr
}

func (f *fakeClient) GetSubscriptionPurchase(_ context.Context, _, _ string) (gpc.SubscriptionPurchaseInfo, error) {
	return f.subscriptionPurchase, f.subscriptionPurchaseErr
}

func (f *fakeClient) GetProductPurchase(_ context.Context, _, _, _ string) (gpc.ProductPurchaseInfo, error) {
	return f.productPurchase, f.productPurchaseErr
}

func (f *fakeClient) GetProductPurchaseV2(_ context.Context, _, _ string) (gpc.ProductPurchaseV2Info, error) {
	return f.productPurchaseV2, f.productPurchaseV2Err
}

func (f *fakeClient) GetExternalTransaction(_ context.Context, _, _ string) (*androidpublisher.ExternalTransaction, error) {
	return f.externalTransaction, f.externalTransactionErr
}

func (f *fakeClient) ListGeneratedAPKs(_ context.Context, _ string, _ int64) (*androidpublisher.GeneratedApksListResponse, error) {
	return f.generatedResp, f.generatedErr
}

func (f *fakeClient) ListSystemAPKVariants(_ context.Context, _ string, _ int64) (*androidpublisher.SystemApksListResponse, error) {
	return f.systemResp, f.systemErr
}

func (f *fakeClient) ListAppRecoveries(_ context.Context, _ string, _ int64) (*androidpublisher.ListAppRecoveriesResponse, error) {
	return f.recoveriesResp, f.recoveriesErr
}

func TestRunStatusMissingFixtures(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)

	res, err := runStatus(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, statusOptions{PackageName: "com.example.app"})
	if err != nil {
		t.Fatalf("runStatus returned error: %v", err)
	}

	if res.Status != "warn" {
		t.Fatalf("expected warn status, got %+v", res)
	}
	for _, name := range []string{"subscription_catalog", "product_catalog", "order_fixture", "subscription_token", "subscription_etag", "product_token", "google_group", "external_transaction", "version_artifacts"} {
		if checkStatus(res, name) == "" {
			t.Fatalf("expected fixture check %q, got %+v", name, res.Checks)
		}
	}
	if checkStatus(res, "subscription_catalog") != "missing" {
		t.Fatalf("expected missing subscription catalog, got %+v", res.Checks)
	}
}

func TestRunStatusMarksSubscriptionEtagStaleForExpiredPurchase(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)
	fixturesPath := writeFixtures(t, `{"subscriptionProductId":"premium","subscriptionToken":"token-123","subscriptionEtag":"etag-123"}`)

	res, err := runStatus(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				subscriptionDiagnostic: gpc.SubscriptionDiagnosticInfo{
					ProductID:            "premium",
					BasePlanCount:        1,
					ActiveBasePlanCount:  1,
					ListingCount:         1,
					RegionCount:          2,
					AvailableRegionCount: 1,
				},
				subscriptionPurchase: gpc.SubscriptionPurchaseInfo{
					SubscriptionState: "SUBSCRIPTION_STATE_EXPIRED",
					LatestOrderID:     "GPA.1",
				},
			}, nil
		},
		LookupEnv: lookupEnv,
	}, statusOptions{PackageName: "com.example.app", FixturesPath: fixturesPath})
	if err != nil {
		t.Fatalf("runStatus returned error: %v", err)
	}

	if checkStatus(res, "subscription_token") != "stale" {
		t.Fatalf("expected stale subscription token, got %+v", res.Checks)
	}
	if checkStatus(res, "subscription_etag") != "stale" {
		t.Fatalf("expected stale subscription etag, got %+v", res.Checks)
	}
	if !containsSubstring(res.NextSteps, "Mint a fresh sandbox subscription") {
		t.Fatalf("expected stale next step, got %+v", res.NextSteps)
	}
}

func TestRunStatusValidFixtures(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)
	fixturesPath := writeFixtures(t, `{
		"subscriptionProductId":"premium",
		"productId":"coins_100",
		"orderId":"GPA.1",
		"subscriptionToken":"sub-token",
		"subscriptionEtag":"etag-123",
		"productToken":"prod-token",
		"googleGroup":"qa@example.com",
		"externalTransactionId":"ext-123"
	}`)

	res, err := runStatus(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				subscriptionDiagnostic: gpc.SubscriptionDiagnosticInfo{
					ProductID:            "premium",
					BasePlanCount:        1,
					ActiveBasePlanCount:  1,
					ListingCount:         1,
					RegionCount:          2,
					AvailableRegionCount: 2,
				},
				oneTimeProductDiag: gpc.OneTimeProductDiagnosticInfo{
					ProductID:                 "coins_100",
					PurchaseOptionCount:       1,
					ActivePurchaseOptionCount: 1,
					ListingCount:              1,
					RegionCount:               2,
					AvailableRegionCount:      2,
				},
				orderInfo: gpc.OrderInfo{State: "COMPLETE", LineItemCount: 1},
				subscriptionPurchase: gpc.SubscriptionPurchaseInfo{
					SubscriptionState: "SUBSCRIPTION_STATE_ACTIVE",
					LatestOrderID:     "GPA.1",
				},
				productPurchase: gpc.ProductPurchaseInfo{
					ProductID: "coins_100",
					OrderID:   "GPA.1",
				},
				productPurchaseV2: gpc.ProductPurchaseV2Info{
					PurchaseState: "PURCHASED",
					OrderID:       "GPA.1",
				},
				externalTransaction: &androidpublisher.ExternalTransaction{ExternalTransactionId: "ext-123"},
				generatedResp:       &androidpublisher.GeneratedApksListResponse{GeneratedApks: []*androidpublisher.GeneratedApksPerSigningKey{{}}},
				systemResp:          &androidpublisher.SystemApksListResponse{Variants: []*androidpublisher.Variant{{}}},
				recoveriesResp:      &androidpublisher.ListAppRecoveriesResponse{RecoveryActions: []*androidpublisher.AppRecoveryAction{{}}},
			}, nil
		},
		LookupEnv: lookupEnv,
	}, statusOptions{PackageName: "com.example.app", FixturesPath: fixturesPath, VersionCode: 123})
	if err != nil {
		t.Fatalf("runStatus returned error: %v", err)
	}

	if res.Status != "ok" {
		t.Fatalf("expected ok status, got %+v", res)
	}
	for _, name := range []string{"subscription_catalog", "product_catalog", "order_fixture", "subscription_token", "subscription_etag", "product_token", "google_group", "external_transaction", "version_artifacts"} {
		if checkStatus(res, name) != "valid" {
			t.Fatalf("expected %s to be valid, got %+v", name, res.Checks)
		}
	}
}

func TestRunStatusInvalidFixturesFileReturnsError(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)
	path := filepath.Join(t.TempDir(), "fixtures.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("write fixtures: %v", err)
	}

	_, err := runStatus(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, statusOptions{PackageName: "com.example.app", FixturesPath: path})
	if err == nil || !strings.Contains(err.Error(), "failed to parse fixtures file JSON") {
		t.Fatalf("unexpected fixtures error: %v", err)
	}
}

func TestStatusCommandStrictReturnsErrorOnWarnings(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{Output: "json"})
	cfg, lookupEnv := validConfig(t)

	var stdout bytes.Buffer
	cmd := NewCommand(Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
		LookupEnv: lookupEnv,
		Stdout:    &stdout,
		Stderr:    &stdout,
	})
	if err := cmd.ParseAndRun(context.Background(), []string{"fixtures", "status", "--package-name", "com.example.app", "--strict"}); err == nil || !strings.Contains(err.Error(), "warnings") {
		t.Fatalf("expected strict warning error, got %v", err)
	}
	if !strings.Contains(stdout.String(), `"status":"warn"`) {
		t.Fatalf("expected json output with warn status, got %s", stdout.String())
	}
}

func TestWriteTable(t *testing.T) {
	var out bytes.Buffer
	err := writeStatusTable(&out, statusResult{
		Status:      "warn",
		PackageName: "com.example.app",
		VersionCode: 123,
		Checks: []fixtureCheck{
			{Name: "subscription_catalog", Status: "missing", Detail: "missing fixture"},
		},
		NextSteps: []string{"next step"},
	})
	if err != nil {
		t.Fatalf("writeStatusTable returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"STATUS\twarn", "PACKAGE\tcom.example.app", "CHECK\tSTATUS\tBLOCKING\tDETAIL", "subscription_catalog\tmissing\tfalse\tmissing fixture", "nextStep\tnext step"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected table output to contain %q, got %s", want, got)
		}
	}
}

func validConfig(t *testing.T) (config.Config, func(string) string) {
	t.Helper()
	serviceAccountPath := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(serviceAccountPath, []byte(`{"type":"service_account"}`), 0o600); err != nil {
		t.Fatalf("write service account: %v", err)
	}
	cfg := config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				ServiceAccountPath: serviceAccountPath,
				DeveloperID:        "1234567890",
			},
		},
	}
	return cfg, envMap(map[string]string{"GPC_BYPASS_KEYCHAIN": "1"})
}

func envMap(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func writeFixtures(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixtures.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixtures: %v", err)
	}
	return path
}

func checkStatus(res statusResult, name string) string {
	for _, check := range res.Checks {
		if check.Name == name {
			return check.Status
		}
	}
	return ""
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func resetGlobalFlags(t *testing.T, flags shared.GlobalFlags) {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	shared.BindGlobalFlags(fs, &flags)
}
