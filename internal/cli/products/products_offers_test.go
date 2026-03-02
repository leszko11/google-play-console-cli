package products

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func TestProductsOffersList_ReturnsOffers(t *testing.T) {
	bindGlobalPaginate(t, false)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				offers: gpc.OneTimeProductOffersListInfo{
					Offers: []gpc.OneTimeProductOfferInfo{{OfferID: "offer_intro"}},
				},
			}, nil
		},
	}

	out, err := runProducts(
		t,
		deps,
		"offers",
		"list",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"offerId":"offer_intro"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsOffersList_UsesGlobalPaginate(t *testing.T) {
	bindGlobalPaginate(t, true)
	capture := &paginateCapture{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{capture: capture}, nil
		},
	}

	_, err := runProducts(
		t,
		deps,
		"offers",
		"list",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !capture.offersPaginate {
		t.Fatal("expected paginate=true from global flags")
	}
}

func TestProductsOffersActivate_ReturnsActivated(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{offer: gpc.OneTimeProductOfferInfo{OfferID: "offer_intro", State: "ACTIVE"}}, nil
		},
	}

	out, err := runProducts(
		t,
		deps,
		"offers",
		"activate",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--offer-id", "offer_intro",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"activated"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsOffersDeactivate_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(
		t,
		deps,
		"offers",
		"deactivate",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--offer-id", "offer_intro",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsOffersCancel_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(
		t,
		deps,
		"offers",
		"cancel",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--offer-id", "offer_intro",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsOffersDeactivate_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{deactivateErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runProducts(
		t,
		deps,
		"offers",
		"deactivate",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--offer-id", "offer_intro",
		"--confirm",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to deactivate one-time product offer") {
		t.Fatalf("unexpected error: %v", err)
	}
}
