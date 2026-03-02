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
	productPurchase      gpc.ProductPurchaseInfo
	productPurchaseErr   error
	ackErr               error
	consumeErr           error
	subscriptionPurchase gpc.SubscriptionPurchaseInfo
	subscriptionErr      error
	cancelErr            error
	revokeErr            error
	voided               gpc.VoidedPurchasesListInfo
	voidedErr            error
	capturedProductID    string
	capturedToken        string
	capturedPayload      string
	capturedCancelType   string
	capturedRefundType   string
	capturedVoidedQuery  gpc.VoidedPurchasesQuery
}

func (f *fakeClient) GetProductPurchase(_ context.Context, _, productID, token string) (gpc.ProductPurchaseInfo, error) {
	f.capturedProductID = productID
	f.capturedToken = token
	return f.productPurchase, f.productPurchaseErr
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

func (f *fakeClient) CancelSubscriptionPurchase(_ context.Context, _, token, cancellationType string) error {
	f.capturedToken = token
	f.capturedCancelType = cancellationType
	return f.cancelErr
}

func (f *fakeClient) RevokeSubscriptionPurchase(_ context.Context, _, token, refundType string) error {
	f.capturedToken = token
	f.capturedRefundType = refundType
	return f.revokeErr
}

func (f *fakeClient) ListVoidedPurchases(_ context.Context, _ string, query gpc.VoidedPurchasesQuery) (gpc.VoidedPurchasesListInfo, error) {
	f.capturedVoidedQuery = query
	return f.voided, f.voidedErr
}

func runPurchases(t *testing.T, deps Deps, args ...string) (string, error) {
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
