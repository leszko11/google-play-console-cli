package products

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func TestProductsPurchaseOptionsActivate_ReturnsActivated(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				purchaseOptions: []gpc.OneTimeProductInfo{{ProductID: "coins_100", PurchaseOptionCount: 1}},
			}, nil
		},
	}

	out, err := runProducts(
		t,
		deps,
		"purchase-options",
		"activate",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"activated"`) || !strings.Contains(out, `"productId":"coins_100"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsPurchaseOptionsDeactivate_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(
		t,
		deps,
		"purchase-options",
		"deactivate",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsPurchaseOptionsDeactivate_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{deactivatePurchaseOptErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runProducts(
		t,
		deps,
		"purchase-options",
		"deactivate",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--confirm",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to deactivate one-time product purchase option") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsPurchaseOptionsDelete_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(
		t,
		deps,
		"purchase-options",
		"delete",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsPurchaseOptionsDelete_PassesForceAndReturnsDeleted(t *testing.T) {
	var gotForce bool
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				deletePurchaseOptFn: func(_, _, _ string, force bool) error {
					gotForce = force
					return nil
				},
			}, nil
		},
	}

	out, err := runProducts(
		t,
		deps,
		"purchase-options",
		"delete",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--force",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !gotForce {
		t.Fatal("expected force=true to be passed to client")
	}
	if !strings.Contains(out, `"status":"deleted"`) || !strings.Contains(out, `"force":true`) {
		t.Fatalf("unexpected output: %s", out)
	}
}
