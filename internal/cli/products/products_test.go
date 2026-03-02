package products

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"google.golang.org/api/androidpublisher/v3"
)

type fakeClient struct {
	list      gpc.OneTimeProductsListInfo
	listErr   error
	get       gpc.OneTimeProductInfo
	getErr    error
	create    gpc.OneTimeProductInfo
	createErr error
	update    gpc.OneTimeProductInfo
	updateErr error
	deleteErr error

	createFn func(packageName string, product *androidpublisher.OneTimeProduct) (gpc.OneTimeProductInfo, error)
	updateFn func(packageName, productID string, product *androidpublisher.OneTimeProduct, updateMask string) (gpc.OneTimeProductInfo, error)

	offers           gpc.OneTimeProductOffersListInfo
	offer            gpc.OneTimeProductOfferInfo
	offersErr        error
	activateOfferErr error
	deactivateErr    error
	cancelErr        error

	purchaseOptions          []gpc.OneTimeProductInfo
	activatePurchaseOptErr   error
	deactivatePurchaseOptErr error
	deletePurchaseOptErr     error

	activatePurchaseOptFn   func(packageName, productID, purchaseOptionID string) ([]gpc.OneTimeProductInfo, error)
	deactivatePurchaseOptFn func(packageName, productID, purchaseOptionID string) ([]gpc.OneTimeProductInfo, error)
	deletePurchaseOptFn     func(packageName, productID, purchaseOptionID string, force bool) error
}

func (f fakeClient) ListOneTimeProducts(_ context.Context, _ string, _ int64, _ string, _ bool) (gpc.OneTimeProductsListInfo, error) {
	return f.list, f.listErr
}

func (f fakeClient) GetOneTimeProduct(_ context.Context, _, _ string) (gpc.OneTimeProductInfo, error) {
	return f.get, f.getErr
}

func (f fakeClient) CreateOneTimeProduct(_ context.Context, packageName string, product *androidpublisher.OneTimeProduct) (gpc.OneTimeProductInfo, error) {
	if f.createFn != nil {
		return f.createFn(packageName, product)
	}
	return f.create, f.createErr
}

func (f fakeClient) UpdateOneTimeProduct(_ context.Context, packageName, productID string, product *androidpublisher.OneTimeProduct, updateMask string) (gpc.OneTimeProductInfo, error) {
	if f.updateFn != nil {
		return f.updateFn(packageName, productID, product, updateMask)
	}
	return f.update, f.updateErr
}

func (f fakeClient) DeleteOneTimeProduct(_ context.Context, _, _ string) error { return f.deleteErr }
func (f fakeClient) ListOneTimeProductOffers(_ context.Context, _, _, _ string, _ int64, _ string, _ bool) (gpc.OneTimeProductOffersListInfo, error) {
	return f.offers, f.offersErr
}
func (f fakeClient) ActivateOneTimeProductOffer(_ context.Context, _, _, _, _ string) (gpc.OneTimeProductOfferInfo, error) {
	return f.offer, f.activateOfferErr
}
func (f fakeClient) DeactivateOneTimeProductOffer(_ context.Context, _, _, _, _ string) (gpc.OneTimeProductOfferInfo, error) {
	return f.offer, f.deactivateErr
}
func (f fakeClient) CancelOneTimeProductOffer(_ context.Context, _, _, _, _ string) (gpc.OneTimeProductOfferInfo, error) {
	return f.offer, f.cancelErr
}
func (f fakeClient) ActivateOneTimeProductPurchaseOption(_ context.Context, packageName, productID, purchaseOptionID string) ([]gpc.OneTimeProductInfo, error) {
	if f.activatePurchaseOptFn != nil {
		return f.activatePurchaseOptFn(packageName, productID, purchaseOptionID)
	}
	return f.purchaseOptions, f.activatePurchaseOptErr
}
func (f fakeClient) DeactivateOneTimeProductPurchaseOption(_ context.Context, packageName, productID, purchaseOptionID string) ([]gpc.OneTimeProductInfo, error) {
	if f.deactivatePurchaseOptFn != nil {
		return f.deactivatePurchaseOptFn(packageName, productID, purchaseOptionID)
	}
	return f.purchaseOptions, f.deactivatePurchaseOptErr
}
func (f fakeClient) DeleteOneTimeProductPurchaseOption(_ context.Context, packageName, productID, purchaseOptionID string, force bool) error {
	if f.deletePurchaseOptFn != nil {
		return f.deletePurchaseOptFn(packageName, productID, purchaseOptionID, force)
	}
	return f.deletePurchaseOptErr
}

func runProducts(t *testing.T, deps Deps, args ...string) (string, error) {
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

func TestProductsList_ReturnsProducts(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				list: gpc.OneTimeProductsListInfo{
					Products: []gpc.OneTimeProductInfo{{ProductID: "coins_100"}},
				},
			}, nil
		},
	}

	out, err := runProducts(t, deps, "list", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"productId":"coins_100"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsGet_RequiresProductID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}
	_, err := runProducts(t, deps, "get", "--package-name", "com.example.app")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--product-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsCreate_ReturnsCreated(t *testing.T) {
	inputPath := writeJSON(t, `{"productId":"coins_100","packageName":"com.example.app"}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				createFn: func(packageName string, product *androidpublisher.OneTimeProduct) (gpc.OneTimeProductInfo, error) {
					if packageName != "com.example.app" {
						t.Fatalf("unexpected package name: %q", packageName)
					}
					if product.ProductId != "coins_100" {
						t.Fatalf("unexpected payload product id: %q", product.ProductId)
					}
					return gpc.OneTimeProductInfo{PackageName: packageName, ProductID: product.ProductId}, nil
				},
			}, nil
		},
	}

	out, err := runProducts(t, deps, "create", "--package-name", "com.example.app", "--input", inputPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"created"`) || !strings.Contains(out, `"productId":"coins_100"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsCreate_InvalidJSON(t *testing.T) {
	inputPath := writeJSON(t, `{not-json}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(t, deps, "create", "--package-name", "com.example.app", "--input", inputPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid one-time product JSON payload") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsUpdate_ReturnsUpdated(t *testing.T) {
	inputPath := writeJSON(t, `{"listings":[{"languageCode":"en-US","title":"Coins","description":"Coins pack"}]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateFn: func(packageName, productID string, _ *androidpublisher.OneTimeProduct, updateMask string) (gpc.OneTimeProductInfo, error) {
					if packageName != "com.example.app" || productID != "coins_100" {
						t.Fatalf("unexpected args: package=%q product=%q", packageName, productID)
					}
					if updateMask != "listings" {
						t.Fatalf("unexpected update mask: %q", updateMask)
					}
					return gpc.OneTimeProductInfo{PackageName: packageName, ProductID: productID}, nil
				},
			}, nil
		},
	}

	out, err := runProducts(
		t,
		deps,
		"update",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--input", inputPath,
		"--update-mask", "listings",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"productId":"coins_100"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsDelete_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(t, deps, "delete", "--package-name", "com.example.app", "--product-id", "coins_100")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsDelete_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{deleteErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runProducts(t, deps, "delete", "--package-name", "com.example.app", "--product-id", "coins_100", "--confirm")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete one-time product") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeJSON(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}
