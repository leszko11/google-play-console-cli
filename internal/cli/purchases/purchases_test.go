package purchases

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	productPurchase        gpc.ProductPurchaseInfo
	productPurchaseErr     error
	productPurchaseV2      gpc.ProductPurchaseV2Info
	productPurchaseV2Err   error
	ackErr                 error
	consumeErr             error
	subscriptionPurchase   gpc.SubscriptionPurchaseInfo
	subscriptionErr        error
	legacySubscription     gpc.SubscriptionPurchaseInfo
	legacySubscriptionErr  error
	cancelErr              error
	legacyAckErr           error
	legacyCancelErr        error
	deferResult            gpc.SubscriptionDeferInfo
	deferErr               error
	legacyDeferResult      gpc.SubscriptionDeferInfo
	legacyDeferErr         error
	revokeErr              error
	legacyRefundErr        error
	legacyRevokeErr        error
	voided                 gpc.VoidedPurchasesListInfo
	voidedErr              error
	capturedProductID      string
	capturedSubscriptionID string
	capturedToken          string
	capturedPayload        string
	capturedCancelType     string
	capturedEtag           string
	capturedDeferDur       string
	capturedExpectedExpiry int64
	capturedDesiredExpiry  int64
	capturedValidateOnly   bool
	capturedRefundType     string
	capturedVoidedQuery    gpc.VoidedPurchasesQuery
}

func (f *fakeClient) GetProductPurchase(_ context.Context, _, productID, token string) (gpc.ProductPurchaseInfo, error) {
	f.capturedProductID = productID
	f.capturedToken = token
	return f.productPurchase, f.productPurchaseErr
}

func (f *fakeClient) GetProductPurchaseV2(_ context.Context, _, token string) (gpc.ProductPurchaseV2Info, error) {
	f.capturedToken = token
	return f.productPurchaseV2, f.productPurchaseV2Err
}

func (f *fakeClient) AcknowledgeProductPurchase(_ context.Context, _, productID, token, developerPayload string) error {
	f.capturedProductID = productID
	f.capturedToken = token
	f.capturedPayload = developerPayload
	return f.ackErr
}

func (f *fakeClient) ConsumeProductPurchase(_ context.Context, _, productID, token string) error {
	f.capturedProductID = productID
	f.capturedToken = token
	return f.consumeErr
}

func (f *fakeClient) GetSubscriptionPurchase(_ context.Context, _, token string) (gpc.SubscriptionPurchaseInfo, error) {
	f.capturedToken = token
	return f.subscriptionPurchase, f.subscriptionErr
}

func (f *fakeClient) GetLegacySubscriptionPurchase(_ context.Context, _, subscriptionID, token string) (gpc.SubscriptionPurchaseInfo, error) {
	f.capturedSubscriptionID = subscriptionID
	f.capturedToken = token
	return f.legacySubscription, f.legacySubscriptionErr
}

func (f *fakeClient) AcknowledgeLegacySubscriptionPurchase(_ context.Context, _, subscriptionID, token, developerPayload string) error {
	f.capturedSubscriptionID = subscriptionID
	f.capturedToken = token
	f.capturedPayload = developerPayload
	return f.legacyAckErr
}

func (f *fakeClient) CancelSubscriptionPurchase(_ context.Context, _, token, cancellationType string) error {
	f.capturedToken = token
	f.capturedCancelType = cancellationType
	return f.cancelErr
}

func (f *fakeClient) CancelLegacySubscriptionPurchase(_ context.Context, _, subscriptionID, token string) error {
	f.capturedSubscriptionID = subscriptionID
	f.capturedToken = token
	return f.legacyCancelErr
}

func (f *fakeClient) DeferSubscriptionPurchase(_ context.Context, _, token, etag, deferDuration string, validateOnly bool) (gpc.SubscriptionDeferInfo, error) {
	f.capturedToken = token
	f.capturedEtag = etag
	f.capturedDeferDur = deferDuration
	f.capturedValidateOnly = validateOnly
	return f.deferResult, f.deferErr
}

func (f *fakeClient) DeferLegacySubscriptionPurchase(_ context.Context, _, subscriptionID, token string, expectedExpiryTimeMillis, desiredExpiryTimeMillis int64) (gpc.SubscriptionDeferInfo, error) {
	f.capturedSubscriptionID = subscriptionID
	f.capturedToken = token
	f.capturedExpectedExpiry = expectedExpiryTimeMillis
	f.capturedDesiredExpiry = desiredExpiryTimeMillis
	return f.legacyDeferResult, f.legacyDeferErr
}

func (f *fakeClient) RevokeSubscriptionPurchase(_ context.Context, _, token, refundType string) error {
	f.capturedToken = token
	f.capturedRefundType = refundType
	return f.revokeErr
}

func (f *fakeClient) RefundLegacySubscriptionPurchase(_ context.Context, _, subscriptionID, token string) error {
	f.capturedSubscriptionID = subscriptionID
	f.capturedToken = token
	return f.legacyRefundErr
}

func (f *fakeClient) RevokeLegacySubscriptionPurchase(_ context.Context, _, subscriptionID, token string) error {
	f.capturedSubscriptionID = subscriptionID
	f.capturedToken = token
	return f.legacyRevokeErr
}

func (f *fakeClient) ListVoidedPurchases(_ context.Context, _ string, query gpc.VoidedPurchasesQuery) (gpc.VoidedPurchasesListInfo, error) {
	f.capturedVoidedQuery = query
	return f.voided, f.voidedErr
}

func runPurchases(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		}
	}
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

func bindGlobalPaginate(t *testing.T, paginate bool) {
	t.Helper()
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	cfg := &shared.GlobalFlags{}
	shared.BindGlobalFlags(fs, cfg)
	cfg.Paginate = paginate
}

func TestPurchasesProductsGet_ReturnsPurchase(t *testing.T) {
	fc := &fakeClient{
		productPurchase: gpc.ProductPurchaseInfo{OrderID: "GPA.1", ProductID: "premium", PurchaseToken: "tok-1"},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runPurchases(t, deps, "products", "get", "--package-name", "com.example.app", "--product-id", "premium", "--token", "tok-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"orderId":"GPA.1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestPurchasesProductsV2Get_ReturnsPurchase(t *testing.T) {
	fc := &fakeClient{
		productPurchaseV2: gpc.ProductPurchaseV2Info{OrderID: "GPA.2", PurchaseState: "PENDING"},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runPurchases(t, deps, "products-v2", "get", "--package-name", "com.example.app", "--token", "tok-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"orderId":"GPA.2"`) || !strings.Contains(out, `"purchaseState":"PENDING"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestPurchasesSubscriptionsGet_ReturnsPurchaseWithEtag(t *testing.T) {
	fc := &fakeClient{
		subscriptionPurchase: gpc.SubscriptionPurchaseInfo{
			Etag:              "etag-1",
			LatestOrderID:     "GPA.3",
			SubscriptionState: "SUBSCRIPTION_STATE_ACTIVE",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runPurchases(t, deps, "subscriptions", "get", "--package-name", "com.example.app", "--token", "tok-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"etag":"etag-1"`) || !strings.Contains(out, `"latestOrderId":"GPA.3"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestPurchasesProductsGet_PackageNotFoundHint(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{productPurchaseErr: gpc.ErrPackageNotFound}, nil
		},
	}

	_, err := runPurchases(t, deps, "products", "get", "--package-name", "com.example.app", "--product-id", "premium", "--token", "tok-1")
	if err == nil || !strings.Contains(err.Error(), "purchase history is unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPurchasesProductsConsume_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runPurchases(t, deps, "products", "consume", "--package-name", "com.example.app", "--product-id", "premium", "--token", "tok-1")
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPurchasesSubscriptionsCancel_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runPurchases(t, deps, "subscriptions", "cancel", "--package-name", "com.example.app", "--token", "tok-1")
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPurchasesSubscriptionsRevoke_ReturnsStatus(t *testing.T) {
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runPurchases(t, deps, "subscriptions", "revoke", "--package-name", "com.example.app", "--token", "tok-1", "--refund-type", "full", "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"revoked"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedRefundType != "full" {
		t.Fatalf("unexpected refund type: %s", fc.capturedRefundType)
	}
}

func TestPurchasesSubscriptionsDefer_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runPurchases(
		t,
		deps,
		"subscriptions", "defer",
		"--package-name", "com.example.app",
		"--token", "tok-1",
		"--etag", "etag-1",
		"--defer-duration", "604800s",
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPurchasesSubscriptionsDefer_ValidateOnly(t *testing.T) {
	fc := &fakeClient{
		deferResult: gpc.SubscriptionDeferInfo{
			ItemExpiryTimeDetails: []gpc.SubscriptionItemExpiryInfo{
				{ProductID: "premium_monthly", ExpiryTime: "2026-04-01T00:00:00Z"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runPurchases(
		t,
		deps,
		"subscriptions", "defer",
		"--package-name", "com.example.app",
		"--token", "tok-1",
		"--etag", "etag-1",
		"--defer-duration", "604800s",
		"--validate-only",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"validated"`) || !strings.Contains(out, `"productId":"premium_monthly"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !fc.capturedValidateOnly || fc.capturedEtag != "etag-1" || fc.capturedDeferDur != "604800s" {
		t.Fatalf("unexpected defer args: validateOnly=%v etag=%s deferDuration=%s", fc.capturedValidateOnly, fc.capturedEtag, fc.capturedDeferDur)
	}
}

func TestPurchasesSubscriptionsLegacyGet_ReturnsPurchase(t *testing.T) {
	fc := &fakeClient{
		legacySubscription: gpc.SubscriptionPurchaseInfo{LatestOrderID: "GPA.legacy.1", AutoRenewing: true},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runPurchases(
		t,
		deps,
		"subscriptions-legacy", "get",
		"--package-name", "com.example.app",
		"--subscription-id", "premium_monthly",
		"--token", "tok-legacy",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"latestOrderId":"GPA.legacy.1"`) || !strings.Contains(out, `"api":"legacy"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedSubscriptionID != "premium_monthly" || fc.capturedToken != "tok-legacy" {
		t.Fatalf("unexpected legacy get args: subscriptionId=%s token=%s", fc.capturedSubscriptionID, fc.capturedToken)
	}
}

func TestPurchasesSubscriptionsLegacyAcknowledge_ReturnsStatus(t *testing.T) {
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runPurchases(
		t,
		deps,
		"subscriptions-legacy", "acknowledge",
		"--package-name", "com.example.app",
		"--subscription-id", "premium_monthly",
		"--token", "tok-legacy",
		"--developer-payload", "payload-1",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"acknowledged"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedPayload != "payload-1" {
		t.Fatalf("unexpected payload: %s", fc.capturedPayload)
	}
}

func TestPurchasesSubscriptionsLegacyRefund_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runPurchases(
		t,
		deps,
		"subscriptions-legacy", "refund",
		"--package-name", "com.example.app",
		"--subscription-id", "premium_monthly",
		"--token", "tok-legacy",
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPurchasesSubscriptionsLegacyDefer_ReturnsStatus(t *testing.T) {
	fc := &fakeClient{
		legacyDeferResult: gpc.SubscriptionDeferInfo{NewExpiryTimeMillis: 987654321},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runPurchases(
		t,
		deps,
		"subscriptions-legacy", "defer",
		"--package-name", "com.example.app",
		"--subscription-id", "premium_monthly",
		"--token", "tok-legacy",
		"--expected-expiry-time-millis", "1700000000000",
		"--desired-expiry-time-millis", "1700600000000",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deferred"`) || !strings.Contains(out, `"newExpiryTimeMillis":987654321`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedExpectedExpiry != 1700000000000 || fc.capturedDesiredExpiry != 1700600000000 {
		t.Fatalf("unexpected legacy defer args: expected=%d desired=%d", fc.capturedExpectedExpiry, fc.capturedDesiredExpiry)
	}
}

func TestPurchasesVoidedList_UsesGlobalPaginate(t *testing.T) {
	bindGlobalPaginate(t, true)
	fc := &fakeClient{
		voided: gpc.VoidedPurchasesListInfo{
			VoidedPurchases: []gpc.VoidedPurchaseInfo{{OrderID: "GPA.2"}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runPurchases(t, deps, "voided", "list", "--package-name", "com.example.app", "--max-results", "100")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"orderId":"GPA.2"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !fc.capturedVoidedQuery.Paginate {
		t.Fatal("expected paginate=true from global flags")
	}
}

func TestPurchasesVoidedList_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{voidedErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runPurchases(t, deps, "voided", "list", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "failed to list voided purchases") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPurchasesVoidedList_PackageNotFoundHint(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{voidedErr: gpc.ErrPackageNotFound}, nil
		},
	}

	_, err := runPurchases(t, deps, "voided", "list", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "purchase history is unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}
