package doctor

import (
	"bytes"
	"context"
	"flag"
	"fmt"
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
	getTrackInfo            gpc.TrackInfo
	getTrackErr             error
	usersList               gpc.UsersListInfo
	usersListErr            error
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

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeClient) ValidateEdit(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeClient) GetTrack(_ context.Context, _, _, _ string) (gpc.TrackInfo, error) {
	if f.getTrackErr != nil {
		return gpc.TrackInfo{}, f.getTrackErr
	}
	if f.getTrackInfo.Name == "" {
		return gpc.TrackInfo{Name: "internal"}, nil
	}
	return f.getTrackInfo, nil
}

func (f *fakeClient) ListUsers(_ context.Context, _ string, _ int64, _ string, _ bool) (gpc.UsersListInfo, error) {
	return f.usersList, f.usersListErr
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

type fakeReportingClient struct {
	apps []string
	err  error
}

func (f *fakeReportingClient) SearchApps(_ context.Context, _ int64, _ string, _ bool) (gpc.ReportingAppsListInfo, error) {
	if f.err != nil {
		return gpc.ReportingAppsListInfo{}, f.err
	}
	resp := gpc.ReportingAppsListInfo{Apps: make([]*gpc.ReportingApp, 0, len(f.apps))}
	for _, app := range f.apps {
		resp.Apps = append(resp.Apps, &gpc.ReportingApp{Name: app})
	}
	return resp, nil
}

func TestRunWithoutPackageNameAuthOnly(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			t.Fatalf("reporting client should not be created without package")
			return nil, nil
		},
		LookupEnv: lookupEnv,
	}, options{})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if res.Status != "warn" {
		t.Fatalf("expected warn status due to keychain bypass, got %+v", res)
	}
	if checkStatus(res, "auth") != "ok" {
		t.Fatalf("expected auth check ok, got %+v", res.Checks)
	}
	if checkStatus(res, "developer_id") != "ok" {
		t.Fatalf("expected developer_id check ok, got %+v", res.Checks)
	}
	if checkStatus(res, "package_access") != "skipped" {
		t.Fatalf("expected package_access skipped, got %+v", res.Checks)
	}
}

func TestRunWarnsWhenConfiguredDeveloperIDIsInvalid(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{usersListErr: fmt.Errorf("invalid developer id")}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, options{})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if checkStatus(res, "developer_id") != "warn" {
		t.Fatalf("expected developer_id warn, got %+v", res.Checks)
	}
	if !containsSubstring(res.NextSteps, "gpc auth init --developer-id <id>") {
		t.Fatalf("expected developer id next step, got %+v", res.NextSteps)
	}
}

func TestRunFailsWhenCredentialsMissing(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return config.Config{}, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			t.Fatalf("client should not be created without credentials")
			return nil, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			t.Fatalf("reporting client should not be created without credentials")
			return nil, nil
		},
		LookupEnv: envMap(map[string]string{shared.EnvStrictAuth: ""}),
	}, options{})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if res.Status != "failed" {
		t.Fatalf("expected failed status, got %+v", res)
	}
	if len(res.BlockingIssues) == 0 {
		t.Fatalf("expected blocking issues, got %+v", res)
	}
}

func TestRunPackageNotFoundAddsBootstrapHint(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{verifyPackageAccessErr: fmt.Errorf("%w: missing package", gpc.ErrPackageNotFound)}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app"})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if res.Status != "failed" {
		t.Fatalf("expected failed status, got %+v", res)
	}
	if !containsString(res.Warnings, "package is not initialized in Google Play yet") {
		t.Fatalf("expected bootstrap warning, got %+v", res.Warnings)
	}
	if !containsSubstring(res.NextSteps, "Upload the first APK or AAB once in Play Console") {
		t.Fatalf("expected bootstrap next step, got %+v", res.NextSteps)
	}
}

func TestRunIncludesBootstrapSummaryFromProjectState(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	root := t.TempDir()
	t.Chdir(root)
	cfg, lookupEnv := validConfig(t)
	if err := os.WriteFile(filepath.Join(root, ".gpc.yaml"), []byte("release-manifest: play/release.yaml\n"), 0o600); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "play"), 0o755); err != nil {
		t.Fatalf("mkdir play: %v", err)
	}
	if err := shared.WriteBootstrapState(filepath.Join(root, "play", "bootstrap-state.json"), shared.BootstrapState{
		PackageName:              "com.example.app",
		PackageReadiness:         "draft_bootstrap_required",
		BootstrapDraftExists:     true,
		BootstrapVersionCodes:    []int64{123},
		LastReadinessRecheck:     "draft_bootstrap_required",
		LastBootstrapCommittedAt: "2026-03-22T12:00:00Z",
	}); err != nil {
		t.Fatalf("write bootstrap state: %v", err)
	}

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				getTrackInfo: gpc.TrackInfo{
					Name: "internal",
					Releases: []gpc.TrackReleaseInfo{
						{Status: "draft", VersionCodes: []int64{123}},
					},
				},
			}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app"})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if !res.BootstrapDraftExists {
		t.Fatalf("expected bootstrap draft summary, got %+v", res)
	}
	if len(res.BootstrapVersionCodes) != 1 || res.BootstrapVersionCodes[0] != 123 {
		t.Fatalf("unexpected version codes: %+v", res.BootstrapVersionCodes)
	}
	if res.BootstrapStatePath == "" || !strings.Contains(filepath.ToSlash(res.BootstrapStatePath), "play/bootstrap-state.json") {
		t.Fatalf("unexpected bootstrap state path: %q", res.BootstrapStatePath)
	}
	if res.RecommendedNextCommand == "" || !strings.Contains(res.RecommendedNextCommand, "release full") {
		t.Fatalf("unexpected recommended command: %q", res.RecommendedNextCommand)
	}
}

func TestRunReportingDisabledIsWarning(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{err: fmt.Errorf("playdeveloperreporting api error (403): API has not been used in project")}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app"})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if res.Status != "warn" {
		t.Fatalf("expected warn status, got %+v", res)
	}
	if checkStatus(res, "reporting") != "warn" {
		t.Fatalf("expected reporting warn, got %+v", res.Checks)
	}
	if !containsSubstring(res.NextSteps, "Enable Google Play Developer Reporting API") {
		t.Fatalf("expected reporting next step, got %+v", res.NextSteps)
	}
}

func TestRunReportingPermissionDeniedIsWarning(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{err: fmt.Errorf("playdeveloperreporting api error (403): The caller does not have permission")}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app"})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if checkStatus(res, "reporting") != "warn" {
		t.Fatalf("expected reporting warn, got %+v", res.Checks)
	}
	if !containsSubstring(res.NextSteps, "Grant the service account access to Google Play Developer Reporting") {
		t.Fatalf("expected permission next step, got %+v", res.NextSteps)
	}
}

func TestRunReportingPackageNotAccessibleWarns(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{apps: []string{"apps/com.other.app"}}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app"})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if checkStatus(res, "reporting") != "warn" {
		t.Fatalf("expected reporting warn, got %+v", res.Checks)
	}
	if !containsSubstring(res.Warnings, "package is not visible in reporting app discovery") {
		t.Fatalf("expected reporting package warning, got %+v", res.Warnings)
	}
}

func TestRunInvalidFixturesFileReturnsError(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	path := filepath.Join(t.TempDir(), "fixtures.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("write fixtures: %v", err)
	}
	cfg, lookupEnv := validConfig(t)

	_, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, options{FixturesPath: path})
	if err == nil || !strings.Contains(err.Error(), "failed to parse fixtures file JSON") {
		t.Fatalf("unexpected fixtures error: %v", err)
	}
}

func TestRunVersionChecksExecuteWhenVersionCodeProvided(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				generatedResp:  &androidpublisher.GeneratedApksListResponse{GeneratedApks: []*androidpublisher.GeneratedApksPerSigningKey{{}}},
				systemResp:     &androidpublisher.SystemApksListResponse{Variants: []*androidpublisher.Variant{{}}},
				recoveriesResp: &androidpublisher.ListAppRecoveriesResponse{RecoveryActions: []*androidpublisher.AppRecoveryAction{{}}},
			}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app", VersionCode: 123})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	for _, name := range []string{"generated_apks", "system_apks", "app_recoveries"} {
		if checkStatus(res, name) != "ok" {
			t.Fatalf("expected %s check ok, got %+v", name, res.Checks)
		}
	}
}

func TestRunSubscriptionDiagnosticWarnsOnInactiveBasePlansAndRegions(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)
	fixturesPath := writeFixtures(t, `{"subscriptionProductId":"premium"}`)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				subscriptionDiagnostic: gpc.SubscriptionDiagnosticInfo{
					ProductID:            "premium",
					BasePlanCount:        2,
					ActiveBasePlanCount:  1,
					ListingCount:         1,
					RegionCount:          2,
					AvailableRegionCount: 1,
					BasePlans: []gpc.SubscriptionBasePlanDiagnosticInfo{
						{BasePlanID: "monthly", State: "ACTIVE", RegionalConfigCount: 2, AvailableRegionCount: 1},
						{BasePlanID: "legacy", State: "INACTIVE", RegionalConfigCount: 0, AvailableRegionCount: 0},
					},
				},
			}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app", FixturesPath: fixturesPath})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if checkStatus(res, "subscription_fixture") != "warn" {
		t.Fatalf("expected subscription warn, got %+v", res.Checks)
	}
	if !containsSubstring(res.NextSteps, "Review subscription regional availability") {
		t.Fatalf("expected subscription regional next step, got %+v", res.NextSteps)
	}
}

func TestRunProductDiagnosticWarnsOnInactivePurchaseOptionsAndRegions(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)
	fixturesPath := writeFixtures(t, `{"productId":"coins_100"}`)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				oneTimeProductDiag: gpc.OneTimeProductDiagnosticInfo{
					ProductID:                 "coins_100",
					PurchaseOptionCount:       2,
					ActivePurchaseOptionCount: 1,
					ListingCount:              1,
					RegionCount:               2,
					AvailableRegionCount:      1,
					PurchaseOptions: []gpc.OneTimeProductPurchaseOptionDiagnosticInfo{
						{PurchaseOptionID: "buy", State: "ACTIVE", RegionalConfigCount: 2, AvailableRegionCount: 1},
						{PurchaseOptionID: "legacy", State: "DRAFT", RegionalConfigCount: 0, AvailableRegionCount: 0},
					},
				},
			}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app", FixturesPath: fixturesPath})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if checkStatus(res, "product_fixture") != "warn" {
		t.Fatalf("expected product warn, got %+v", res.Checks)
	}
	if !containsSubstring(res.NextSteps, "Review one-time product regional availability") {
		t.Fatalf("expected product regional next step, got %+v", res.NextSteps)
	}
}

func TestRunProductAndPurchaseFixtures(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)
	fixturesPath := writeFixtures(t, `{"productId":"coins_100","productToken":"token-123"}`)

	res, err := run(context.Background(), Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				oneTimeProductDiag: gpc.OneTimeProductDiagnosticInfo{
					ProductID:                 "coins_100",
					PurchaseOptionCount:       1,
					ActivePurchaseOptionCount: 1,
					ListingCount:              1,
					RegionCount:               2,
					AvailableRegionCount:      2,
				},
				productPurchase: gpc.ProductPurchaseInfo{
					ProductID: "coins_100",
					OrderID:   "GPA.1",
				},
				productPurchaseV2: gpc.ProductPurchaseV2Info{
					OrderID:       "GPA.1",
					PurchaseState: "PURCHASED",
				},
			}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app", FixturesPath: fixturesPath})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if checkStatus(res, "product_fixture") != "ok" || checkStatus(res, "product_purchase_fixture") != "ok" || checkStatus(res, "product_purchase_v2_fixture") != "ok" {
		t.Fatalf("expected product checks ok, got %+v", res.Checks)
	}
}

func TestRunSubscriptionWarningWhenEtagMissing(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{})
	cfg, lookupEnv := validConfig(t)
	fixturesPath := writeFixtures(t, `{"subscriptionProductId":"premium","subscriptionToken":"token-123"}`)

	res, err := run(context.Background(), Deps{
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
					SubscriptionState: "SUBSCRIPTION_STATE_ACTIVE",
					LatestOrderID:     "GPA.1",
				},
			}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{}, nil
		},
		LookupEnv: lookupEnv,
	}, options{PackageName: "com.example.app", FixturesPath: fixturesPath})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if res.Status != "warn" {
		t.Fatalf("expected warn status, got %+v", res)
	}
	if !containsSubstring(res.Warnings, "subscriptionEtag missing") {
		t.Fatalf("expected subscriptionEtag warning, got %+v", res.Warnings)
	}
}

func TestDoctorCommandStrictReturnsErrorOnWarnings(t *testing.T) {
	resetGlobalFlags(t, shared.GlobalFlags{Output: "json"})
	cfg, lookupEnv := validConfig(t)

	var stdout bytes.Buffer
	cmd := NewCommand(Deps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReportingClient{err: fmt.Errorf("playdeveloperreporting api error (403): API has not been used in project")}, nil
		},
		LookupEnv: lookupEnv,
		Stdout:    &stdout,
		Stderr:    &stdout,
	})
	if err := cmd.FlagSet.Parse([]string{"--package-name", "com.example.app", "--strict"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	err := cmd.Exec(context.Background(), cmd.FlagSet.Args())
	if err == nil || !strings.Contains(err.Error(), "warnings") {
		t.Fatalf("expected strict warning error, got %v", err)
	}
	if !strings.Contains(stdout.String(), `"status":"warn"`) {
		t.Fatalf("expected json output with warn status, got %s", stdout.String())
	}
}

func TestWriteTable(t *testing.T) {
	var out bytes.Buffer
	err := writeTable(&out, result{
		Status:      "warn",
		PackageName: "com.example.app",
		VersionCode: 123,
		Checks: []doctorCheck{
			{Name: "auth", Status: "ok", Detail: "ready"},
		},
		Warnings:  []string{"warning one"},
		NextSteps: []string{"next step"},
	})
	if err != nil {
		t.Fatalf("writeTable returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"STATUS\twarn", "PACKAGE\tcom.example.app", "CHECK\tSTATUS\tBLOCKING\tDETAIL", "auth\tok\tfalse\tready", "warning\twarning one", "nextStep\tnext step"} {
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
	return cfg, envMap(map[string]string{
		"GPC_BYPASS_KEYCHAIN": "1",
	})
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

func checkStatus(res result, name string) string {
	for _, check := range res.Checks {
		if check.Name == name {
			return check.Status
		}
	}
	return ""
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
