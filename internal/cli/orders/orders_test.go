package orders

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	order            gpc.OrderInfo
	orderErr         error
	orders           []gpc.OrderInfo
	ordersErr        error
	refundErr        error
	capturedOrderID  string
	capturedOrderIDs []string
	capturedRevoke   bool
}

func (f *fakeClient) GetOrder(_ context.Context, _ string, orderID string) (gpc.OrderInfo, error) {
	f.capturedOrderID = orderID
	return f.order, f.orderErr
}

func (f *fakeClient) BatchGetOrders(_ context.Context, _ string, orderIDs []string) ([]gpc.OrderInfo, error) {
	f.capturedOrderIDs = append([]string(nil), orderIDs...)
	return f.orders, f.ordersErr
}

func (f *fakeClient) RefundOrder(_ context.Context, _ string, orderID string, revoke bool) error {
	f.capturedOrderID = orderID
	f.capturedRevoke = revoke
	return f.refundErr
}

func runOrders(t *testing.T, deps Deps, args ...string) (string, error) {
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

func bindGlobalPackageName(t *testing.T, packageName string) {
	t.Helper()
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	cfg := &shared.GlobalFlags{}
	shared.BindGlobalFlags(fs, cfg)
	cfg.PackageName = packageName
}

func TestOrdersGet_ReturnsOrder(t *testing.T) {
	fc := &fakeClient{
		order: gpc.OrderInfo{OrderID: "GPA.1", State: "PROCESSED", BuyerCountry: "PL"},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runOrders(t, deps, "get", "--package-name", "com.example.app", "--order-id", "GPA.1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"orderId":"GPA.1"`) || !strings.Contains(out, `"buyerCountry":"PL"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestOrdersBatchGet_ParsesCSV(t *testing.T) {
	fc := &fakeClient{
		orders: []gpc.OrderInfo{{OrderID: "GPA.1"}, {OrderID: "GPA.2"}},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runOrders(t, deps, "batch-get", "--package-name", "com.example.app", "--order-ids", " GPA.1 , GPA.2 ")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"count":2`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if len(fc.capturedOrderIDs) != 2 || fc.capturedOrderIDs[0] != "GPA.1" || fc.capturedOrderIDs[1] != "GPA.2" {
		t.Fatalf("unexpected parsed order ids: %#v", fc.capturedOrderIDs)
	}
}

func TestOrdersBatchGet_RequiresOrderIDs(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runOrders(t, deps, "batch-get", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "--order-ids is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrdersRefund_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runOrders(t, deps, "refund", "--package-name", "com.example.app", "--order-id", "GPA.1")
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrdersRefund_ReturnsStatus(t *testing.T) {
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runOrders(t, deps, "refund", "--package-name", "com.example.app", "--order-id", "GPA.1", "--revoke", "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"refunded"`) || !strings.Contains(out, `"revoke":true`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedOrderID != "GPA.1" || !fc.capturedRevoke {
		t.Fatalf("unexpected captured refund call: order=%q revoke=%t", fc.capturedOrderID, fc.capturedRevoke)
	}
}

func TestOrdersGet_UsesGlobalPackageName(t *testing.T) {
	bindGlobalPackageName(t, "com.example.global")
	fc := &fakeClient{order: gpc.OrderInfo{OrderID: "GPA.1"}}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	if _, err := runOrders(t, deps, "get", "--order-id", "GPA.1"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
}
