package doctor

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

type Client interface {
	VerifyPackageAccess(ctx context.Context, packageName string) error
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ValidateEdit(ctx context.Context, packageName, editID string) error
	ListUsers(ctx context.Context, developerID string, pageSize int64, pageToken string, paginate bool) (gpc.UsersListInfo, error)
	GetSubscriptionDiagnostic(ctx context.Context, packageName, productID string) (gpc.SubscriptionDiagnosticInfo, error)
	GetOneTimeProductDiagnostic(ctx context.Context, packageName, productID string) (gpc.OneTimeProductDiagnosticInfo, error)
	GetOrder(ctx context.Context, packageName, orderID string) (gpc.OrderInfo, error)
	GetSubscriptionPurchase(ctx context.Context, packageName, token string) (gpc.SubscriptionPurchaseInfo, error)
	GetProductPurchase(ctx context.Context, packageName, productID, token string) (gpc.ProductPurchaseInfo, error)
	GetProductPurchaseV2(ctx context.Context, packageName, token string) (gpc.ProductPurchaseV2Info, error)
	GetExternalTransaction(ctx context.Context, packageName, externalTransactionID string) (*androidpublisher.ExternalTransaction, error)
	ListGeneratedAPKs(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.GeneratedApksListResponse, error)
	ListSystemAPKVariants(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.SystemApksListResponse, error)
	ListAppRecoveries(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.ListAppRecoveriesResponse, error)
}

type ReportingClient interface {
	SearchApps(ctx context.Context, pageSize int64, pageToken string, paginate bool) (gpc.ReportingAppsListInfo, error)
}

type Deps struct {
	LoadConfig         func() (config.Config, error)
	NewClient          func(context.Context, gpc.CredentialInput) (Client, error)
	NewReportingClient func(context.Context, gpc.CredentialInput) (ReportingClient, error)
	LookupEnv          func(string) string
	Stdout             io.Writer
	Stderr             io.Writer
}

type fixturesFile struct {
	SubscriptionProductID string `json:"subscriptionProductId"`
	ProductID             string `json:"productId"`
	OrderID               string `json:"orderId"`
	SubscriptionToken     string `json:"subscriptionToken"`
	SubscriptionEtag      string `json:"subscriptionEtag"`
	ProductToken          string `json:"productToken"`
	GoogleGroup           string `json:"googleGroup"`
	ExternalTransactionID string `json:"externalTransactionId"`
}

type doctorCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Blocking bool   `json:"blocking,omitempty"`
}

type result struct {
	Status           string        `json:"status"`
	PackageName      string        `json:"packageName,omitempty"`
	PackageReadiness string        `json:"packageReadiness,omitempty"`
	VersionCode      int64         `json:"versionCode,omitempty"`
	ProjectConfig    *projectInfo  `json:"projectConfig,omitempty"`
	Checks           []doctorCheck `json:"checks"`
	Warnings         []string      `json:"warnings,omitempty"`
	BlockingIssues   []string      `json:"blockingIssues,omitempty"`
	NextSteps        []string      `json:"nextSteps,omitempty"`
}

type projectInfo struct {
	Path          string   `json:"path"`
	PackageName   string   `json:"packageName,omitempty"`
	Profile       string   `json:"profile,omitempty"`
	Output        string   `json:"output,omitempty"`
	DefaultTrack  string   `json:"defaultTrack,omitempty"`
	DefaultLocale string   `json:"defaultLocale,omitempty"`
	DefaultPaths  []string `json:"defaultPaths,omitempty"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName string
	var fixturesPath string
	var versionCode int64
	var strict bool

	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&fixturesPath, "fixtures-file", "", "Path to JSON fixture file")
	fs.Int64Var(&versionCode, "version-code", 0, "Version code for delivery diagnostics")
	fs.BoolVar(&strict, "strict", false, "Fail on warnings")

	return &ffcli.Command{
		Name:      "doctor",
		ShortHelp: "Run read-only diagnostics for auth, package access, and e2e fixtures",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts := options{
				PackageName:  strings.TrimSpace(packageName),
				FixturesPath: strings.TrimSpace(fixturesPath),
				VersionCode:  versionCode,
				Strict:       strict,
			}

			result, err := run(ctx, deps, opts)
			if err != nil {
				return err
			}

			switch shared.ResolveOutput("") {
			case "json":
				if err := shared.WriteJSON(deps.Stdout, result); err != nil {
					return err
				}
			case "table":
				if err := writeTable(deps.Stdout, result); err != nil {
					return err
				}
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(""))
			}

			if result.Status == "failed" {
				return fmt.Errorf("doctor checks failed")
			}
			if opts.Strict && result.Status == "warn" {
				return fmt.Errorf("doctor checks produced warnings")
			}
			return nil
		},
	}
}

type options struct {
	PackageName  string
	FixturesPath string
	VersionCode  int64
	Strict       bool
}

func withDefaults(deps Deps) Deps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewClient == nil {
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (Client, error) {
			return gpc.NewClient(ctx, creds)
		}
	}
	if deps.NewReportingClient == nil {
		deps.NewReportingClient = func(ctx context.Context, creds gpc.CredentialInput) (ReportingClient, error) {
			return gpc.NewReportingClient(ctx, creds)
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return deps
}

func run(ctx context.Context, deps Deps, opts options) (result, error) {
	cfg, err := deps.LoadConfig()
	if err != nil {
		return result{}, err
	}

	fixtures, err := loadFixtures(opts.FixturesPath)
	if err != nil {
		return result{}, err
	}

	res := result{
		Status:      "ok",
		PackageName: resolvePackageName(opts.PackageName),
		VersionCode: opts.VersionCode,
		Checks:      make([]doctorCheck, 0, 16),
	}
	projectCfg, projectErr := config.LoadProject()
	if projectErr == nil && projectCfg.Path != "" {
		defaultPaths := make([]string, 0, 8)
		if projectCfg.Config.ListingDir != "" {
			defaultPaths = append(defaultPaths, "listing-dir="+projectCfg.Config.ListingDir)
		}
		if projectCfg.Config.ScreenshotsDir != "" {
			defaultPaths = append(defaultPaths, "screenshots-dir="+projectCfg.Config.ScreenshotsDir)
		}
		if projectCfg.Config.ProductsDir != "" {
			defaultPaths = append(defaultPaths, "products-dir="+projectCfg.Config.ProductsDir)
		}
		if projectCfg.Config.SubscriptionsDir != "" {
			defaultPaths = append(defaultPaths, "subscriptions-dir="+projectCfg.Config.SubscriptionsDir)
		}
		if projectCfg.Config.ChangelogDir != "" {
			defaultPaths = append(defaultPaths, "changelog-dir="+projectCfg.Config.ChangelogDir)
		}
		if projectCfg.Config.AndroidProjectDir != "" {
			defaultPaths = append(defaultPaths, "android-project-dir="+projectCfg.Config.AndroidProjectDir)
		}
		if projectCfg.Config.ArtifactPath != "" {
			defaultPaths = append(defaultPaths, "artifact-path="+projectCfg.Config.ArtifactPath)
		}
		if projectCfg.Config.NotesFile != "" {
			defaultPaths = append(defaultPaths, "notes-file="+projectCfg.Config.NotesFile)
		}
		if projectCfg.Config.AppInitManifest != "" {
			defaultPaths = append(defaultPaths, "appinit-manifest="+projectCfg.Config.AppInitManifest)
		}
		if projectCfg.Config.ReleaseManifest != "" {
			defaultPaths = append(defaultPaths, "release-manifest="+projectCfg.Config.ReleaseManifest)
		}
		res.ProjectConfig = &projectInfo{
			Path:          projectCfg.Path,
			PackageName:   projectCfg.Config.PackageName,
			Profile:       projectCfg.Config.Profile,
			Output:        projectCfg.Config.Output,
			DefaultTrack:  projectCfg.Config.DefaultTrack,
			DefaultLocale: projectCfg.Config.DefaultLocale,
			DefaultPaths:  defaultPaths,
		}
	}

	authStatus := shared.BuildAuthStatusSnapshot(cfg, deps.LookupEnv)
	authDetail := shared.AuthStatusSummary(authStatus)
	if authDetail == "" {
		authDetail = "authentication status unavailable"
	}
	if authStatus.Authenticated {
		res.addOK("auth", authDetail)
	} else {
		res.addBlocking("auth", authDetail)
	}
	for _, warning := range authStatus.Warnings {
		res.addWarning(warning)
	}

	if !authStatus.Authenticated {
		res.finalize()
		return res, nil
	}

	resolved, err := shared.ResolveCredentials(cfg, deps.LookupEnv)
	if err != nil {
		res.addBlocking("credentials", err.Error())
		res.finalize()
		return res, nil
	}

	requestCtx, cancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
	defer cancel()

	client, err := deps.NewClient(requestCtx, resolved.Input)
	if err != nil {
		res.addBlocking("client", err.Error())
		res.finalize()
		return res, nil
	}

	runDeveloperIDChecks(requestCtx, client, cfg, &res)

	if res.PackageName == "" {
		res.addSkipped("package_access", "skipped (--package-name not provided)")
		res.addSkipped("reporting", "skipped (--package-name not provided)")
		res.addSkipped("subscription_fixture", "skipped (--package-name not provided)")
		res.addSkipped("product_fixture", "skipped (--package-name not provided)")
		res.addSkipped("order_fixture", "skipped (--package-name not provided)")
		res.addSkipped("subscription_purchase_fixture", "skipped (--package-name not provided)")
		res.addSkipped("product_purchase_fixture", "skipped (--package-name not provided)")
		res.addSkipped("external_transaction_fixture", "skipped (--package-name not provided)")
		res.addSkipped("tester_fixture", "skipped (--package-name not provided)")
		res.addSkipped("delivery_version", "skipped (--package-name not provided)")
		res.finalize()
		return res, nil
	}

	readiness, readinessErr := shared.DetectPackageReadiness(requestCtx, client, res.PackageName)
	switch {
	case readinessErr != nil:
		res.addBlocking("package_access", readinessErr.Error())
	case readiness.Status == shared.PackageReadinessUninitialized:
		res.PackageReadiness = string(readiness.Status)
		res.addBlocking("package_access", readiness.Detail)
		res.addWarning("package is not initialized in Google Play yet")
		res.addNextStep(readiness.NextStep)
	case readiness.Status == shared.PackageReadinessDraftBootstrapRequired:
		res.PackageReadiness = string(readiness.Status)
		res.addOK("package_access", "package access verified")
		res.addWarn("package_readiness", readiness.Detail)
		if readiness.Warning != "" {
			res.addWarning(readiness.Warning)
		}
		res.addNextStep(readiness.NextStep)
	default:
		res.PackageReadiness = string(readiness.Status)
		res.addOK("package_access", "package access verified")
		res.addOK("package_readiness", readiness.Detail)
	}

	reportingClient, err := deps.NewReportingClient(requestCtx, resolved.Input)
	if err != nil {
		res.addWarning(fmt.Sprintf("reporting client init failed: %v", err))
		res.addWarn("reporting", "reporting client initialization failed")
		res.addNextStep("Enable Google Play Developer Reporting API for the service-account project.")
	} else {
		apps, reportingErr := reportingClient.SearchApps(requestCtx, 100, "", true)
		switch {
		case reportingErr == nil:
			reportingCheckStatus, detail, nextStep := evaluateReportingProbe(res.PackageName, apps)
			switch reportingCheckStatus {
			case "ok":
				res.addOK("reporting", detail)
			case "warn":
				res.addWarn("reporting", detail)
			default:
				res.addBlocking("reporting", detail)
			}
			res.addNextStep(nextStep)
		case isReportingPermissionError(reportingErr):
			res.addWarn("reporting", reportingErr.Error())
			res.addNextStep("Grant the service account access to Google Play Developer Reporting and verify the Cloud project binding.")
		case isReportingAPIEnablementError(reportingErr):
			res.addWarn("reporting", reportingErr.Error())
			res.addNextStep("Enable Google Play Developer Reporting API for the service-account project.")
		default:
			res.addBlocking("reporting", reportingErr.Error())
		}
	}

	runSubscriptionChecks(requestCtx, client, res.PackageName, fixtures, &res)
	runProductChecks(requestCtx, client, res.PackageName, fixtures, &res)
	runOrderChecks(requestCtx, client, res.PackageName, fixtures, &res)
	runExternalTransactionChecks(requestCtx, client, res.PackageName, fixtures, &res)
	runTesterFixtureChecks(fixtures, &res)
	runVersionChecks(requestCtx, client, res.PackageName, opts.VersionCode, &res)

	res.finalize()
	return res, nil
}

func resolvePackageName(localValue string) string {
	if pkg := strings.TrimSpace(localValue); pkg != "" {
		return pkg
	}
	return strings.TrimSpace(shared.ActiveGlobalFlags().PackageName)
}

func loadFixtures(path string) (fixturesFile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return fixturesFile{}, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return fixturesFile{}, fmt.Errorf("failed to read fixtures file: %w", err)
	}

	var fixtures fixturesFile
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		return fixturesFile{}, fmt.Errorf("failed to parse fixtures file JSON: %w", err)
	}

	fixtures.SubscriptionProductID = strings.TrimSpace(fixtures.SubscriptionProductID)
	fixtures.ProductID = strings.TrimSpace(fixtures.ProductID)
	fixtures.OrderID = strings.TrimSpace(fixtures.OrderID)
	fixtures.SubscriptionToken = strings.TrimSpace(fixtures.SubscriptionToken)
	fixtures.SubscriptionEtag = strings.TrimSpace(fixtures.SubscriptionEtag)
	fixtures.ProductToken = strings.TrimSpace(fixtures.ProductToken)
	fixtures.GoogleGroup = strings.TrimSpace(fixtures.GoogleGroup)
	fixtures.ExternalTransactionID = strings.TrimSpace(fixtures.ExternalTransactionID)
	return fixtures, nil
}

func runDeveloperIDChecks(ctx context.Context, client Client, cfg config.Config, res *result) {
	developerID, err := shared.ResolveDeveloperID("", cfg)
	if err != nil {
		res.addSkipped("developer_id", "skipped (developerId not configured for selected profile)")
		return
	}

	users, err := client.ListUsers(ctx, developerID, 1, "", false)
	if err != nil {
		res.addWarn("developer_id", fmt.Sprintf("configured developer id %s could not be verified: %v", developerID, err))
		res.addNextStep("Update the selected profile with the correct developer ID via `gpc auth init --developer-id <id>`.")
		return
	}

	res.addOK("developer_id", fmt.Sprintf("developer id verified (%s, visibleUsers=%d)", developerID, len(users.Users)))
}

func runSubscriptionChecks(ctx context.Context, client Client, packageName string, fixtures fixturesFile, res *result) {
	if fixtures.SubscriptionProductID == "" {
		res.addSkipped("subscription_fixture", "skipped (subscriptionProductId not provided)")
	} else {
		subscription, err := client.GetSubscriptionDiagnostic(ctx, packageName, fixtures.SubscriptionProductID)
		if err != nil {
			res.addBlocking("subscription_fixture", err.Error())
		} else {
			detail, needsReview := summarizeSubscriptionDiagnostic(subscription)
			if needsReview {
				res.addWarn("subscription_fixture", detail)
				res.addNextStep("Review subscription regional availability and add required regions such as PL/US for billing smoke tests.")
				res.addNextStep("Activate the subscription base plan intended for billing smoke tests.")
			} else {
				res.addOK("subscription_fixture", detail)
			}
		}
	}

	if fixtures.SubscriptionToken == "" {
		res.addSkipped("subscription_purchase_fixture", "skipped (subscriptionToken not provided)")
		return
	}

	purchase, err := client.GetSubscriptionPurchase(ctx, packageName, fixtures.SubscriptionToken)
	if err != nil {
		res.addBlocking("subscription_purchase_fixture", err.Error())
		return
	}

	if strings.Contains(strings.ToUpper(purchase.SubscriptionState), "EXPIRED") || strings.Contains(strings.ToUpper(purchase.SubscriptionState), "CANCELED") {
		res.addWarn("subscription_purchase_fixture", fmt.Sprintf("subscription purchase found but state=%s", purchase.SubscriptionState))
		res.addNextStep("Mint a fresh sandbox subscription and refresh subscriptionEtag.")
	} else {
		res.addOK("subscription_purchase_fixture", fmt.Sprintf("subscription purchase verified (state=%s latestOrderId=%s)", purchase.SubscriptionState, purchase.LatestOrderID))
	}

	if fixtures.SubscriptionEtag == "" {
		res.addWarning("subscriptionEtag missing; defer --validate-only readiness is unconfirmed")
		res.addNextStep("Mint a fresh sandbox subscription and refresh subscriptionEtag.")
	}
}

func runProductChecks(ctx context.Context, client Client, packageName string, fixtures fixturesFile, res *result) {
	if fixtures.ProductID == "" {
		res.addSkipped("product_fixture", "skipped (productId not provided)")
	} else {
		product, err := client.GetOneTimeProductDiagnostic(ctx, packageName, fixtures.ProductID)
		if err != nil {
			res.addBlocking("product_fixture", err.Error())
		} else {
			detail, needsReview := summarizeProductDiagnostic(product)
			if needsReview {
				res.addWarn("product_fixture", detail)
				res.addNextStep("Review one-time product regional availability and add required regions such as PL/US for billing smoke tests.")
				res.addNextStep("Activate the one-time product purchase option intended for billing smoke tests.")
			} else {
				res.addOK("product_fixture", detail)
			}
		}
	}

	if fixtures.ProductToken == "" {
		res.addSkipped("product_purchase_fixture", "skipped (productToken not provided)")
		return
	}
	if fixtures.ProductID == "" {
		res.addWarn("product_purchase_fixture", "productToken provided but productId is missing")
		res.addNextStep("Set productId in the fixtures file to validate product purchase fixtures.")
		return
	}

	purchase, err := client.GetProductPurchase(ctx, packageName, fixtures.ProductID, fixtures.ProductToken)
	if err != nil {
		res.addBlocking("product_purchase_fixture", err.Error())
		return
	}
	purchaseV2, err := client.GetProductPurchaseV2(ctx, packageName, fixtures.ProductToken)
	if err != nil {
		res.addBlocking("product_purchase_v2_fixture", err.Error())
		return
	}

	res.addOK("product_purchase_fixture", fmt.Sprintf("product purchase verified (productId=%s orderId=%s)", purchase.ProductID, purchase.OrderID))
	res.addOK("product_purchase_v2_fixture", fmt.Sprintf("product purchase v2 verified (state=%s orderId=%s)", purchaseV2.PurchaseState, purchaseV2.OrderID))
}

func runOrderChecks(ctx context.Context, client Client, packageName string, fixtures fixturesFile, res *result) {
	if fixtures.OrderID == "" {
		res.addSkipped("order_fixture", "skipped (orderId not provided)")
		return
	}

	order, err := client.GetOrder(ctx, packageName, fixtures.OrderID)
	if err != nil {
		res.addBlocking("order_fixture", err.Error())
		return
	}
	res.addOK("order_fixture", fmt.Sprintf("order verified (lineItems=%d state=%s)", order.LineItemCount, order.State))
}

func runExternalTransactionChecks(ctx context.Context, client Client, packageName string, fixtures fixturesFile, res *result) {
	if fixtures.ExternalTransactionID == "" {
		res.addSkipped("external_transaction_fixture", "skipped (externalTransactionId not provided)")
		res.addNextStep("Set externalTransactionId in the fixtures file to validate external transaction reads.")
		return
	}

	transaction, err := client.GetExternalTransaction(ctx, packageName, fixtures.ExternalTransactionID)
	if err != nil {
		res.addBlocking("external_transaction_fixture", err.Error())
		return
	}
	res.addOK("external_transaction_fixture", fmt.Sprintf("external transaction verified (%s)", strings.TrimSpace(transaction.ExternalTransactionId)))
}

func runTesterFixtureChecks(fixtures fixturesFile, res *result) {
	if fixtures.GoogleGroup == "" {
		res.addSkipped("tester_fixture", "skipped (googleGroup not provided)")
		res.addNextStep("Set GPC_TEST_GOOGLE_GROUP to a real Google Group email.")
		return
	}

	res.addOK("tester_fixture", fmt.Sprintf("google group fixture present (%s); write-path smoke remains separate", fixtures.GoogleGroup))
}

func runVersionChecks(ctx context.Context, client Client, packageName string, versionCode int64, res *result) {
	if versionCode <= 0 {
		res.addSkipped("delivery_version", "skipped (--version-code not provided)")
		return
	}

	generated, err := client.ListGeneratedAPKs(ctx, packageName, versionCode)
	if err != nil {
		res.addBlocking("generated_apks", err.Error())
	} else {
		count := 0
		if generated != nil {
			count = len(generated.GeneratedApks)
		}
		res.addOK("generated_apks", fmt.Sprintf("generated APKs listed (count=%d)", count))
	}

	systemAPKs, err := client.ListSystemAPKVariants(ctx, packageName, versionCode)
	if err != nil {
		res.addBlocking("system_apks", err.Error())
	} else {
		count := 0
		if systemAPKs != nil {
			count = len(systemAPKs.Variants)
		}
		res.addOK("system_apks", fmt.Sprintf("system APK variants listed (count=%d)", count))
	}

	recoveries, err := client.ListAppRecoveries(ctx, packageName, versionCode)
	if err != nil {
		res.addBlocking("app_recoveries", err.Error())
	} else {
		count := 0
		if recoveries != nil {
			count = len(recoveries.RecoveryActions)
		}
		res.addOK("app_recoveries", fmt.Sprintf("app recoveries listed (count=%d)", count))
	}
}

func isReportingAPIEnablementError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "playdeveloperreporting api error") ||
		strings.Contains(lower, "serviceusage.services.enable") ||
		strings.Contains(lower, "has not been used in project") ||
		strings.Contains(lower, "api has not been used in project") ||
		strings.Contains(lower, "play developer reporting api")
}

func isReportingPermissionError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "permission") ||
		strings.Contains(lower, "caller does not have permission") ||
		strings.Contains(lower, "permission denied")
}

func evaluateReportingProbe(packageName string, apps gpc.ReportingAppsListInfo) (status string, detail string, nextStep string) {
	if len(apps.Apps) == 0 {
		return "warn", "reporting probe succeeded but returned zero accessible apps", "Verify that the app has reporting data and that the service account can see the reporting app inventory."
	}

	target := "apps/" + strings.TrimSpace(packageName)
	for _, app := range apps.Apps {
		if app != nil && strings.TrimSpace(app.Name) == target {
			return "ok", fmt.Sprintf("reporting package visible in accessible app discovery (count=%d)", len(apps.Apps)), ""
		}
	}

	return "warn", fmt.Sprintf("package is not visible in reporting app discovery (count=%d)", len(apps.Apps)), "Verify that the package has reporting data and that the service account can access reporting for this app."
}

func summarizeSubscriptionDiagnostic(subscription gpc.SubscriptionDiagnosticInfo) (string, bool) {
	basePlanStates := make([]string, 0, len(subscription.BasePlans))
	for _, plan := range subscription.BasePlans {
		basePlanStates = append(basePlanStates, fmt.Sprintf("%s:%s", plan.BasePlanID, plan.State))
	}
	detail := fmt.Sprintf(
		"subscription %s verified (basePlans=%d activeBasePlans=%d listings=%d regions=%d availableRegions=%d states=%s)",
		subscription.ProductID,
		subscription.BasePlanCount,
		subscription.ActiveBasePlanCount,
		subscription.ListingCount,
		subscription.RegionCount,
		subscription.AvailableRegionCount,
		joinDiagnosticStates(basePlanStates),
	)
	needsReview := subscription.ActiveBasePlanCount < subscription.BasePlanCount || subscription.AvailableRegionCount < subscription.RegionCount
	return detail, needsReview
}

func summarizeProductDiagnostic(product gpc.OneTimeProductDiagnosticInfo) (string, bool) {
	purchaseOptionStates := make([]string, 0, len(product.PurchaseOptions))
	for _, option := range product.PurchaseOptions {
		purchaseOptionStates = append(purchaseOptionStates, fmt.Sprintf("%s:%s", option.PurchaseOptionID, option.State))
	}
	detail := fmt.Sprintf(
		"one-time product %s verified (purchaseOptions=%d activePurchaseOptions=%d listings=%d regions=%d availableRegions=%d states=%s)",
		product.ProductID,
		product.PurchaseOptionCount,
		product.ActivePurchaseOptionCount,
		product.ListingCount,
		product.RegionCount,
		product.AvailableRegionCount,
		joinDiagnosticStates(purchaseOptionStates),
	)
	needsReview := product.ActivePurchaseOptionCount < product.PurchaseOptionCount || product.AvailableRegionCount < product.RegionCount
	return detail, needsReview
}

func joinDiagnosticStates(values []string) string {
	if len(values) == 0 {
		return "n/a"
	}
	return strings.Join(values, ",")
}

func (r *result) addOK(name, detail string) {
	r.Checks = append(r.Checks, doctorCheck{Name: name, Status: "ok", Detail: detail})
}

func (r *result) addWarn(name, detail string) {
	r.Checks = append(r.Checks, doctorCheck{Name: name, Status: "warn", Detail: detail})
	r.addWarning(detail)
}

func (r *result) addBlocking(name, detail string) {
	r.Checks = append(r.Checks, doctorCheck{Name: name, Status: "failed", Detail: detail, Blocking: true})
	r.BlockingIssues = append(r.BlockingIssues, detail)
}

func (r *result) addSkipped(name, detail string) {
	r.Checks = append(r.Checks, doctorCheck{Name: name, Status: "skipped", Detail: detail})
}

func (r *result) addWarning(detail string) {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return
	}
	r.Warnings = append(r.Warnings, detail)
}

func (r *result) addNextStep(step string) {
	step = strings.TrimSpace(step)
	if step == "" {
		return
	}
	r.NextSteps = append(r.NextSteps, step)
}

func (r *result) finalize() {
	r.Warnings = uniqueStrings(r.Warnings)
	r.BlockingIssues = uniqueStrings(r.BlockingIssues)
	r.NextSteps = uniqueStrings(r.NextSteps)

	status := "ok"
	for _, check := range r.Checks {
		if check.Status == "failed" && check.Blocking {
			r.Status = "failed"
			return
		}
		if check.Status == "warn" {
			status = "warn"
		}
	}
	if len(r.Warnings) > 0 && status == "ok" {
		status = "warn"
	}
	r.Status = status
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func writeTable(out io.Writer, res result) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", res.Status); err != nil {
		return err
	}
	if res.PackageName != "" {
		if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", res.PackageName); err != nil {
			return err
		}
	}
	if res.VersionCode > 0 {
		if _, err := fmt.Fprintf(out, "VERSION_CODE\t%d\n", res.VersionCode); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(out, "CHECK\tSTATUS\tBLOCKING\tDETAIL"); err != nil {
		return err
	}
	for _, check := range res.Checks {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%t\t%s\n", check.Name, check.Status, check.Blocking, check.Detail); err != nil {
			return err
		}
	}
	for _, warning := range res.Warnings {
		if _, err := fmt.Fprintf(out, "warning\t%s\n", warning); err != nil {
			return err
		}
	}
	for _, issue := range res.BlockingIssues {
		if _, err := fmt.Fprintf(out, "blockingIssue\t%s\n", issue); err != nil {
			return err
		}
	}
	for _, step := range res.NextSteps {
		if _, err := fmt.Fprintf(out, "nextStep\t%s\n", step); err != nil {
			return err
		}
	}
	return nil
}
