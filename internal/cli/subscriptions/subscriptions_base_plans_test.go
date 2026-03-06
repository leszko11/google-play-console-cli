package subscriptions

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func TestSubscriptionsBasePlansActivate_ReturnsActivated(t *testing.T) {
	fc := &fakeClient{
		basePlanUpdates: []gpc.SubscriptionInfo{
			{PackageName: "com.example.app", ProductID: "premium_monthly", BasePlanCount: 1},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runSubscriptions(
		t,
		deps,
		"base-plans",
		"activate",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"activated"`) || !strings.Contains(out, `"productId":"premium_monthly"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.productID != "premium_monthly" || fc.basePlanID != "monthly" {
		t.Fatalf("unexpected captured ids: product=%q basePlan=%q", fc.productID, fc.basePlanID)
	}
}

func TestSubscriptionsBasePlansDeactivate_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(
		t,
		deps,
		"base-plans",
		"deactivate",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsBasePlansDeactivate_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{basePlanDeactErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runSubscriptions(
		t,
		deps,
		"base-plans",
		"deactivate",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "failed to deactivate subscription base plan") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsBasePlansBatchUpdateStates_ReturnsUpdated(t *testing.T) {
	inputPath := writeJSON(t, `{"requests":[{"activateBasePlanRequest":{"packageName":"com.example.app","productId":"premium_monthly","basePlanId":"monthly"}}]}`)
	fc := &fakeClient{
		basePlanUpdates: []gpc.SubscriptionInfo{
			{PackageName: "com.example.app", ProductID: "premium_monthly", BasePlanCount: 1},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runSubscriptions(
		t,
		deps,
		"base-plans",
		"batch-update-states",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--input", inputPath,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"updatedCount":1`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if len(fc.basePlanBatchStateInput) != 1 || fc.productID != "premium_monthly" {
		t.Fatalf("unexpected captured batch update state input: product=%q requests=%d", fc.productID, len(fc.basePlanBatchStateInput))
	}
}

func TestSubscriptionsBasePlansDelete_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(
		t,
		deps,
		"base-plans",
		"delete",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsBasePlansDelete_ReturnsDeleted(t *testing.T) {
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runSubscriptions(
		t,
		deps,
		"base-plans",
		"delete",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deleted"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.productID != "premium_monthly" || fc.basePlanID != "monthly" {
		t.Fatalf("unexpected captured ids: product=%q basePlan=%q", fc.productID, fc.basePlanID)
	}
}
