package e2e

import (
	"context"
	"encoding/json"
	"errors"
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

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
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

type fixtureCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Blocking bool   `json:"blocking,omitempty"`
}

type statusResult struct {
	Status         string         `json:"status"`
	PackageName    string         `json:"packageName,omitempty"`
	VersionCode    int64          `json:"versionCode,omitempty"`
	Checks         []fixtureCheck `json:"checks"`
	Warnings       []string       `json:"warnings,omitempty"`
	BlockingIssues []string       `json:"blockingIssues,omitempty"`
	NextSteps      []string       `json:"nextSteps,omitempty"`
}

type statusOptions struct {
	PackageName  string
	FixturesPath string
	VersionCode  int64
	Strict       bool
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "e2e",
		ShortHelp: "E2E fixture and smoke-testing helpers",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newFixturesCommand(deps),
		},
	}
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

func newFixturesCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "fixtures",
		ShortHelp: "Inspect live e2e fixture readiness",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newFixturesStatusCommand(deps),
		},
	}
}

func newFixturesStatusCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName string
	var fixturesPath string
	var versionCode int64
	var strict bool

	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&fixturesPath, "fixtures-file", "", "Path to JSON fixture file")
	fs.Int64Var(&versionCode, "version-code", 0, "Version code for delivery fixtures")
	fs.BoolVar(&strict, "strict", false, "Fail on warnings")

	return &ffcli.Command{
		Name:      "status",
		ShortHelp: "Check which live e2e fixtures are valid, missing, or stale",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			res, err := runStatus(ctx, deps, statusOptions{
				PackageName:  strings.TrimSpace(packageName),
				FixturesPath: strings.TrimSpace(fixturesPath),
				VersionCode:  versionCode,
				Strict:       strict,
			})
			if err != nil {
				return err
			}

			switch shared.ResolveOutput("") {
			case "json":
				if err := shared.WriteJSON(deps.Stdout, res); err != nil {
					return err
				}
			case "table":
				if err := writeStatusTable(deps.Stdout, res); err != nil {
					return err
				}
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(""))
			}

			if res.Status == "failed" {
				return fmt.Errorf("e2e fixture checks failed")
			}
			if strict && res.Status == "warn" {
				return fmt.Errorf("e2e fixture checks produced warnings")
			}
			return nil
		},
	}
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

func resolvePackageName(localValue string) string {
	if pkg := strings.TrimSpace(localValue); pkg != "" {
		return pkg
	}
	return strings.TrimSpace(shared.ActiveGlobalFlags().PackageName)
}

func runStatus(ctx context.Context, deps Deps, opts statusOptions) (statusResult, error) {
	cfg, err := deps.LoadConfig()
	if err != nil {
		return statusResult{}, err
	}
	fixtures, err := loadFixtures(opts.FixturesPath)
	if err != nil {
		return statusResult{}, err
	}

	res := statusResult{
		Status:      "ok",
		PackageName: resolvePackageName(opts.PackageName),
		VersionCode: opts.VersionCode,
		Checks:      make([]fixtureCheck, 0, 12),
	}

	authStatus := shared.BuildAuthStatusSnapshot(cfg, deps.LookupEnv)
	authDetail := shared.AuthStatusSummary(authStatus)
	if authDetail == "" {
		authDetail = "authentication status unavailable"
	}
	if authStatus.Authenticated {
		res.addCheck("auth", "ok", authDetail, false)
	} else {
		res.addBlocking("auth", authDetail)
	}
	if !authStatus.Authenticated {
		res.finalize()
		return res, nil
	}

	if res.PackageName == "" {
		res.addBlocking("package_access", "package name is required (--package-name or global --package-name)")
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

	switch err := client.VerifyPackageAccess(requestCtx, res.PackageName); {
	case err == nil:
		res.addCheck("package_access", "ok", "package access verified", false)
	case errors.Is(err, gpc.ErrPackageNotFound):
		res.addBlocking("package_access", err.Error())
		res.addNextStep("Upload the first APK or AAB once in Play Console, then rerun gpc e2e fixtures status.")
	default:
		res.addBlocking("package_access", err.Error())
	}

	if res.Status != "failed" {
		runSubscriptionChecks(requestCtx, client, res.PackageName, fixtures, &res)
		runProductChecks(requestCtx, client, res.PackageName, fixtures, &res)
		runOrderChecks(requestCtx, client, res.PackageName, fixtures, &res)
		runGoogleGroupCheck(fixtures, &res)
		runExternalTransactionChecks(requestCtx, client, res.PackageName, fixtures, &res)
		runVersionArtifactChecks(requestCtx, client, res.PackageName, opts.VersionCode, &res)
	}

	res.finalize()
	return res, nil
}

func runSubscriptionChecks(ctx context.Context, client Client, packageName string, fixtures fixturesFile, res *statusResult) {
	if fixtures.SubscriptionProductID == "" {
		res.addCheck("subscription_catalog", "missing", "subscriptionProductId missing from fixtures file", false)
		res.addNextStep("Set subscriptionProductId in the fixtures file to validate subscription catalog coverage.")
	} else {
		subscription, err := client.GetSubscriptionDiagnostic(ctx, packageName, fixtures.SubscriptionProductID)
		if err != nil {
			res.addBlocking("subscription_catalog", err.Error())
		} else {
			res.addCheck("subscription_catalog", "valid", fmt.Sprintf(
				"subscription %s verified (basePlans=%d activeBasePlans=%d listings=%d regions=%d availableRegions=%d)",
				subscription.ProductID,
				subscription.BasePlanCount,
				subscription.ActiveBasePlanCount,
				subscription.ListingCount,
				subscription.RegionCount,
				subscription.AvailableRegionCount,
			), false)
		}
	}

	if fixtures.SubscriptionToken == "" {
		res.addCheck("subscription_token", "missing", "subscriptionToken missing from fixtures file", false)
		res.addCheck("subscription_etag", "missing", "subscriptionEtag missing from fixtures file", false)
		res.addNextStep("Mint a fresh sandbox subscription and capture subscriptionToken plus subscriptionEtag.")
		return
	}

	purchase, err := client.GetSubscriptionPurchase(ctx, packageName, fixtures.SubscriptionToken)
	if err != nil {
		res.addBlocking("subscription_token", err.Error())
		if fixtures.SubscriptionEtag != "" {
			res.addCheck("subscription_etag", "stale", "subscriptionEtag is present but subscriptionToken lookup failed", false)
		} else {
			res.addCheck("subscription_etag", "missing", "subscriptionEtag missing from fixtures file", false)
		}
		res.addNextStep("Mint a fresh sandbox subscription and refresh subscriptionToken plus subscriptionEtag.")
		return
	}

	state := strings.ToUpper(strings.TrimSpace(purchase.SubscriptionState))
	if strings.Contains(state, "EXPIRED") || strings.Contains(state, "CANCELED") {
		res.addCheck("subscription_token", "stale", fmt.Sprintf("subscription purchase found but state=%s", purchase.SubscriptionState), false)
		res.addCheck("subscription_etag", "stale", "subscriptionEtag should be refreshed alongside an active sandbox subscription", false)
		res.addNextStep("Mint a fresh sandbox subscription and refresh subscriptionToken plus subscriptionEtag.")
		return
	}

	res.addCheck("subscription_token", "valid", fmt.Sprintf("subscription purchase verified (state=%s latestOrderId=%s)", purchase.SubscriptionState, purchase.LatestOrderID), false)
	if fixtures.SubscriptionEtag == "" {
		res.addCheck("subscription_etag", "missing", "subscriptionEtag missing from fixtures file", false)
		res.addNextStep("Capture subscriptionEtag from gpc purchases subscriptions get for defer --validate-only coverage.")
		return
	}
	res.addCheck("subscription_etag", "valid", "subscriptionEtag present and paired with an active subscription fixture", false)
}

func runProductChecks(ctx context.Context, client Client, packageName string, fixtures fixturesFile, res *statusResult) {
	if fixtures.ProductID == "" {
		res.addCheck("product_catalog", "missing", "productId missing from fixtures file", false)
		res.addNextStep("Set productId in the fixtures file to validate one-time product catalog coverage.")
	} else {
		product, err := client.GetOneTimeProductDiagnostic(ctx, packageName, fixtures.ProductID)
		if err != nil {
			res.addBlocking("product_catalog", err.Error())
		} else {
			res.addCheck("product_catalog", "valid", fmt.Sprintf(
				"one-time product %s verified (purchaseOptions=%d activePurchaseOptions=%d listings=%d regions=%d availableRegions=%d)",
				product.ProductID,
				product.PurchaseOptionCount,
				product.ActivePurchaseOptionCount,
				product.ListingCount,
				product.RegionCount,
				product.AvailableRegionCount,
			), false)
		}
	}

	if fixtures.ProductToken == "" {
		res.addCheck("product_token", "missing", "productToken missing from fixtures file", false)
		res.addNextStep("Complete a sandbox one-time purchase and capture productToken.")
		return
	}
	if fixtures.ProductID == "" {
		res.addCheck("product_token", "failed", "productToken provided but productId is missing", true)
		return
	}

	purchase, err := client.GetProductPurchase(ctx, packageName, fixtures.ProductID, fixtures.ProductToken)
	if err != nil {
		res.addBlocking("product_token", err.Error())
		return
	}
	purchaseV2, err := client.GetProductPurchaseV2(ctx, packageName, fixtures.ProductToken)
	if err != nil {
		res.addBlocking("product_token", err.Error())
		return
	}

	res.addCheck("product_token", "valid", fmt.Sprintf(
		"product purchase verified (productId=%s orderId=%s state=%s)",
		purchase.ProductID,
		purchase.OrderID,
		purchaseV2.PurchaseState,
	), false)
}

func runOrderChecks(ctx context.Context, client Client, packageName string, fixtures fixturesFile, res *statusResult) {
	if fixtures.OrderID == "" {
		res.addCheck("order_fixture", "missing", "orderId missing from fixtures file", false)
		res.addNextStep("Capture a real GPA order ID from a sandbox purchase to validate Orders API reads.")
		return
	}

	order, err := client.GetOrder(ctx, packageName, fixtures.OrderID)
	if err != nil {
		res.addBlocking("order_fixture", err.Error())
		return
	}
	res.addCheck("order_fixture", "valid", fmt.Sprintf("order verified (lineItems=%d state=%s)", order.LineItemCount, order.State), false)
}

func runGoogleGroupCheck(fixtures fixturesFile, res *statusResult) {
	if fixtures.GoogleGroup == "" {
		res.addCheck("google_group", "missing", "googleGroup missing from fixtures file", false)
		res.addNextStep("Set GPC_TEST_GOOGLE_GROUP to a real Google Group email.")
		return
	}
	res.addCheck("google_group", "valid", fmt.Sprintf("google group fixture present (%s)", fixtures.GoogleGroup), false)
}

func runExternalTransactionChecks(ctx context.Context, client Client, packageName string, fixtures fixturesFile, res *statusResult) {
	if fixtures.ExternalTransactionID == "" {
		res.addCheck("external_transaction", "missing", "externalTransactionId missing from fixtures file", false)
		res.addNextStep("Set externalTransactionId in the fixtures file to validate external transaction reads.")
		return
	}

	transaction, err := client.GetExternalTransaction(ctx, packageName, fixtures.ExternalTransactionID)
	if err != nil {
		res.addBlocking("external_transaction", err.Error())
		return
	}
	res.addCheck("external_transaction", "valid", fmt.Sprintf("external transaction verified (%s)", strings.TrimSpace(transaction.ExternalTransactionId)), false)
}

func runVersionArtifactChecks(ctx context.Context, client Client, packageName string, versionCode int64, res *statusResult) {
	if versionCode <= 0 {
		res.addCheck("version_artifacts", "skipped", "versionCode not provided", false)
		return
	}

	generated, err := client.ListGeneratedAPKs(ctx, packageName, versionCode)
	if err != nil {
		res.addBlocking("version_artifacts", err.Error())
		return
	}
	systemAPKs, err := client.ListSystemAPKVariants(ctx, packageName, versionCode)
	if err != nil {
		res.addBlocking("version_artifacts", err.Error())
		return
	}
	recoveries, err := client.ListAppRecoveries(ctx, packageName, versionCode)
	if err != nil {
		res.addBlocking("version_artifacts", err.Error())
		return
	}

	generatedCount := 0
	if generated != nil {
		generatedCount = len(generated.GeneratedApks)
	}
	systemCount := 0
	if systemAPKs != nil {
		systemCount = len(systemAPKs.Variants)
	}
	recoveryCount := 0
	if recoveries != nil {
		recoveryCount = len(recoveries.RecoveryActions)
	}
	res.addCheck("version_artifacts", "valid", fmt.Sprintf(
		"delivery fixtures verified for versionCode=%d (generatedApks=%d systemApkVariants=%d appRecoveries=%d)",
		versionCode,
		generatedCount,
		systemCount,
		recoveryCount,
	), false)
}

func (r *statusResult) addCheck(name, status, detail string, blocking bool) {
	r.Checks = append(r.Checks, fixtureCheck{Name: name, Status: status, Detail: detail, Blocking: blocking})
	if blocking {
		r.BlockingIssues = append(r.BlockingIssues, strings.TrimSpace(detail))
	}
}

func (r *statusResult) addBlocking(name, detail string) {
	r.addCheck(name, "failed", detail, true)
}

func (r *statusResult) addWarning(detail string) {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return
	}
	r.Warnings = append(r.Warnings, detail)
}

func (r *statusResult) addNextStep(step string) {
	step = strings.TrimSpace(step)
	if step == "" {
		return
	}
	r.NextSteps = append(r.NextSteps, step)
}

func (r *statusResult) finalize() {
	r.Warnings = uniqueStrings(r.Warnings)
	r.BlockingIssues = uniqueStrings(r.BlockingIssues)
	r.NextSteps = uniqueStrings(r.NextSteps)

	status := "ok"
	for _, check := range r.Checks {
		if check.Status == "failed" && check.Blocking {
			r.Status = "failed"
			return
		}
		if check.Status == "missing" || check.Status == "stale" {
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

func writeStatusTable(out io.Writer, res statusResult) error {
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
