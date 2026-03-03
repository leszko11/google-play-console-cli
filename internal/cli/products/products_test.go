package products

import (
	"bytes"
	"context"
	"errors"
	"flag"
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
	list           gpc.OneTimeProductsListInfo
	listErr        error
	get            gpc.OneTimeProductInfo
	getErr         error
	batchGet       gpc.OneTimeProductsListInfo
	batchGetErr    error
	batchUpdate    gpc.OneTimeProductsListInfo
	batchUpdateErr error
	batchDeleteErr error
	create         gpc.OneTimeProductInfo
	createErr      error
	update         gpc.OneTimeProductInfo
	updateErr      error
	deleteErr      error

	createFn      func(packageName string, product *androidpublisher.OneTimeProduct) (gpc.OneTimeProductInfo, error)
	updateFn      func(packageName, productID string, product *androidpublisher.OneTimeProduct, updateMask string) (gpc.OneTimeProductInfo, error)
	batchGetFn    func(packageName string, productIDs []string) (gpc.OneTimeProductsListInfo, error)
	batchUpdateFn func(packageName string, requests []*androidpublisher.UpdateOneTimeProductRequest) (gpc.OneTimeProductsListInfo, error)
	batchDeleteFn func(packageName string, requests []*androidpublisher.DeleteOneTimeProductRequest) error

	offers               gpc.OneTimeProductOffersListInfo
	offer                gpc.OneTimeProductOfferInfo
	offersErr            error
	offersBatchGet       gpc.OneTimeProductOffersListInfo
	offersBatchGetErr    error
	offersBatchUpdate    gpc.OneTimeProductOffersListInfo
	offersBatchUpdateErr error
	offersBatchDeleteErr error
	activateOfferErr     error
	deactivateErr        error
	cancelErr            error

	purchaseOptions          []gpc.OneTimeProductInfo
	activatePurchaseOptErr   error
	deactivatePurchaseOptErr error
	deletePurchaseOptErr     error

	activatePurchaseOptFn   func(packageName, productID, purchaseOptionID string) ([]gpc.OneTimeProductInfo, error)
	deactivatePurchaseOptFn func(packageName, productID, purchaseOptionID string) ([]gpc.OneTimeProductInfo, error)
	deletePurchaseOptFn     func(packageName, productID, purchaseOptionID string, force bool) error
	batchGetOffersFn        func(packageName, productID, purchaseOptionID string, offerIDs []string) (gpc.OneTimeProductOffersListInfo, error)
	batchUpdateOffersFn     func(packageName, productID, purchaseOptionID string, requests []*androidpublisher.UpdateOneTimeProductOfferRequest) (gpc.OneTimeProductOffersListInfo, error)
	batchDeleteOffersFn     func(packageName, productID, purchaseOptionID string, requests []*androidpublisher.DeleteOneTimeProductOfferRequest) error

	capture *paginateCapture
}

func (f fakeClient) ListOneTimeProducts(_ context.Context, _ string, pageSize int64, pageToken string, paginate bool) (gpc.OneTimeProductsListInfo, error) {
	if f.capture != nil {
		f.capture.productsPageSize = pageSize
		f.capture.productsPageTok = pageToken
		f.capture.productsPaginate = paginate
	}
	return f.list, f.listErr
}

func (f fakeClient) GetOneTimeProduct(_ context.Context, _, _ string) (gpc.OneTimeProductInfo, error) {
	return f.get, f.getErr
}

func (f fakeClient) BatchGetOneTimeProducts(_ context.Context, packageName string, productIDs []string) (gpc.OneTimeProductsListInfo, error) {
	if f.batchGetFn != nil {
		return f.batchGetFn(packageName, productIDs)
	}
	return f.batchGet, f.batchGetErr
}

func (f fakeClient) BatchUpdateOneTimeProducts(_ context.Context, packageName string, requests []*androidpublisher.UpdateOneTimeProductRequest) (gpc.OneTimeProductsListInfo, error) {
	if f.batchUpdateFn != nil {
		return f.batchUpdateFn(packageName, requests)
	}
	return f.batchUpdate, f.batchUpdateErr
}

func (f fakeClient) BatchDeleteOneTimeProducts(_ context.Context, packageName string, requests []*androidpublisher.DeleteOneTimeProductRequest) error {
	if f.batchDeleteFn != nil {
		return f.batchDeleteFn(packageName, requests)
	}
	return f.batchDeleteErr
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
func (f fakeClient) ListOneTimeProductOffers(_ context.Context, _, _, _ string, pageSize int64, pageToken string, paginate bool) (gpc.OneTimeProductOffersListInfo, error) {
	if f.capture != nil {
		f.capture.offersPageSize = pageSize
		f.capture.offersPageTok = pageToken
		f.capture.offersPaginate = paginate
	}
	return f.offers, f.offersErr
}
func (f fakeClient) BatchGetOneTimeProductOffers(_ context.Context, packageName, productID, purchaseOptionID string, offerIDs []string) (gpc.OneTimeProductOffersListInfo, error) {
	if f.batchGetOffersFn != nil {
		return f.batchGetOffersFn(packageName, productID, purchaseOptionID, offerIDs)
	}
	return f.offersBatchGet, f.offersBatchGetErr
}
func (f fakeClient) BatchUpdateOneTimeProductOffers(_ context.Context, packageName, productID, purchaseOptionID string, requests []*androidpublisher.UpdateOneTimeProductOfferRequest) (gpc.OneTimeProductOffersListInfo, error) {
	if f.batchUpdateOffersFn != nil {
		return f.batchUpdateOffersFn(packageName, productID, purchaseOptionID, requests)
	}
	return f.offersBatchUpdate, f.offersBatchUpdateErr
}
func (f fakeClient) BatchDeleteOneTimeProductOffers(_ context.Context, packageName, productID, purchaseOptionID string, requests []*androidpublisher.DeleteOneTimeProductOfferRequest) error {
	if f.batchDeleteOffersFn != nil {
		return f.batchDeleteOffersFn(packageName, productID, purchaseOptionID, requests)
	}
	return f.offersBatchDeleteErr
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

type paginateCapture struct {
	productsPageSize int64
	productsPageTok  string
	productsPaginate bool
	offersPageSize   int64
	offersPageTok    string
	offersPaginate   bool
}

func bindGlobalPaginate(t *testing.T, paginate bool) {
	t.Helper()
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	cfg := &shared.GlobalFlags{}
	shared.BindGlobalFlags(fs, cfg)
	cfg.Paginate = paginate
}

func TestProductsList_ReturnsProducts(t *testing.T) {
	bindGlobalPaginate(t, false)
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

func TestProductsList_UsesGlobalPaginate(t *testing.T) {
	bindGlobalPaginate(t, true)
	capture := &paginateCapture{}
	fc := fakeClient{capture: capture}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	_, err := runProducts(t, deps, "list", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !capture.productsPaginate {
		t.Fatal("expected paginate=true from global flags")
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

func TestProductsBatchGet_RequiresProductIDs(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(t, deps, "batch-get", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "--product-ids is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsBatchGet_ReturnsProducts(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				batchGetFn: func(packageName string, productIDs []string) (gpc.OneTimeProductsListInfo, error) {
					if packageName != "com.example.app" {
						t.Fatalf("unexpected package name: %q", packageName)
					}
					if len(productIDs) != 2 || productIDs[0] != "coins_100" || productIDs[1] != "coins_500" {
						t.Fatalf("unexpected product IDs: %+v", productIDs)
					}
					return gpc.OneTimeProductsListInfo{
						Products: []gpc.OneTimeProductInfo{{ProductID: "coins_100"}, {ProductID: "coins_500"}},
					}, nil
				},
			}, nil
		},
	}

	out, err := runProducts(t, deps, "batch-get", "--package-name", "com.example.app", "--product-ids", "coins_100,coins_500")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"productId":"coins_100"`) || !strings.Contains(out, `"productId":"coins_500"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsBatchUpdate_ReturnsUpdated(t *testing.T) {
	inputPath := writeJSON(t, `{"requests":[{"allowMissing":true,"oneTimeProduct":{"packageName":"com.example.app","productId":"coins_100"}}]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				batchUpdateFn: func(packageName string, requests []*androidpublisher.UpdateOneTimeProductRequest) (gpc.OneTimeProductsListInfo, error) {
					if packageName != "com.example.app" {
						t.Fatalf("unexpected package name: %q", packageName)
					}
					if len(requests) != 1 || requests[0].OneTimeProduct == nil || requests[0].OneTimeProduct.ProductId != "coins_100" {
						t.Fatalf("unexpected requests: %+v", requests)
					}
					return gpc.OneTimeProductsListInfo{
						Products: []gpc.OneTimeProductInfo{{ProductID: "coins_100"}},
					}, nil
				},
			}, nil
		},
	}

	out, err := runProducts(t, deps, "batch-update", "--package-name", "com.example.app", "--input", inputPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"productId":"coins_100"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsBatchDelete_RequiresConfirm(t *testing.T) {
	inputPath := writeJSON(t, `{"requests":[{"packageName":"com.example.app","productId":"coins_100"}]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(t, deps, "batch-delete", "--package-name", "com.example.app", "--input", inputPath)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsBatchDelete_ReturnsDeleted(t *testing.T) {
	inputPath := writeJSON(t, `{"requests":[{"packageName":"com.example.app","productId":"coins_100"},{"packageName":"com.example.app","productId":"coins_500"}]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				batchDeleteFn: func(packageName string, requests []*androidpublisher.DeleteOneTimeProductRequest) error {
					if packageName != "com.example.app" {
						t.Fatalf("unexpected package name: %q", packageName)
					}
					if len(requests) != 2 || requests[0].ProductId != "coins_100" || requests[1].ProductId != "coins_500" {
						t.Fatalf("unexpected requests: %+v", requests)
					}
					return nil
				},
			}, nil
		},
	}

	out, err := runProducts(t, deps, "batch-delete", "--package-name", "com.example.app", "--input", inputPath, "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deleted"`) || !strings.Contains(out, `"deletedCount":2`) {
		t.Fatalf("unexpected output: %s", out)
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

func TestProductsOffersBatchGet_RequiresOfferIDs(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(
		t,
		deps,
		"offers", "batch-get",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
	)
	if err == nil || !strings.Contains(err.Error(), "--offer-ids is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsOffersBatchGet_ReturnsOffers(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				batchGetOffersFn: func(packageName, productID, purchaseOptionID string, offerIDs []string) (gpc.OneTimeProductOffersListInfo, error) {
					if packageName != "com.example.app" || productID != "coins_100" || purchaseOptionID != "buy" {
						t.Fatalf("unexpected args: package=%q product=%q purchaseOption=%q", packageName, productID, purchaseOptionID)
					}
					if len(offerIDs) != 2 || offerIDs[0] != "offer_intro" || offerIDs[1] != "offer_sale" {
						t.Fatalf("unexpected offer IDs: %+v", offerIDs)
					}
					return gpc.OneTimeProductOffersListInfo{
						Offers: []gpc.OneTimeProductOfferInfo{{OfferID: "offer_intro"}, {OfferID: "offer_sale"}},
					}, nil
				},
			}, nil
		},
	}

	out, err := runProducts(
		t,
		deps,
		"offers", "batch-get",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--offer-ids", "offer_intro,offer_sale",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"offerId":"offer_intro"`) || !strings.Contains(out, `"offerId":"offer_sale"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsOffersBatchUpdate_ReturnsUpdated(t *testing.T) {
	inputPath := writeJSON(t, `{"requests":[{"allowMissing":true,"updateMask":"offerTags","regionsVersion":{"version":"2022/02"},"oneTimeProductOffer":{"packageName":"com.example.app","productId":"coins_100","purchaseOptionId":"buy","offerId":"offer_intro"}}]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				batchUpdateOffersFn: func(packageName, productID, purchaseOptionID string, requests []*androidpublisher.UpdateOneTimeProductOfferRequest) (gpc.OneTimeProductOffersListInfo, error) {
					if packageName != "com.example.app" || productID != "coins_100" || purchaseOptionID != "buy" {
						t.Fatalf("unexpected args: package=%q product=%q purchaseOption=%q", packageName, productID, purchaseOptionID)
					}
					if len(requests) != 1 || requests[0].OneTimeProductOffer == nil || requests[0].OneTimeProductOffer.OfferId != "offer_intro" {
						t.Fatalf("unexpected requests: %+v", requests)
					}
					return gpc.OneTimeProductOffersListInfo{
						Offers: []gpc.OneTimeProductOfferInfo{{OfferID: "offer_intro"}},
					}, nil
				},
			}, nil
		},
	}

	out, err := runProducts(
		t,
		deps,
		"offers", "batch-update",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--input", inputPath,
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"offerId":"offer_intro"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestProductsOffersBatchDelete_RequiresConfirm(t *testing.T) {
	inputPath := writeJSON(t, `{"requests":[{"packageName":"com.example.app","productId":"coins_100","purchaseOptionId":"buy","offerId":"offer_intro"}]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runProducts(
		t,
		deps,
		"offers", "batch-delete",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--input", inputPath,
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsOffersBatchDelete_ReturnsDeleted(t *testing.T) {
	inputPath := writeJSON(t, `{"requests":[{"packageName":"com.example.app","productId":"coins_100","purchaseOptionId":"buy","offerId":"offer_intro"},{"packageName":"com.example.app","productId":"coins_100","purchaseOptionId":"buy","offerId":"offer_sale"}]}`)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				batchDeleteOffersFn: func(packageName, productID, purchaseOptionID string, requests []*androidpublisher.DeleteOneTimeProductOfferRequest) error {
					if packageName != "com.example.app" || productID != "coins_100" || purchaseOptionID != "buy" {
						t.Fatalf("unexpected args: package=%q product=%q purchaseOption=%q", packageName, productID, purchaseOptionID)
					}
					if len(requests) != 2 || requests[0].OfferId != "offer_intro" || requests[1].OfferId != "offer_sale" {
						t.Fatalf("unexpected requests: %+v", requests)
					}
					return nil
				},
			}, nil
		},
	}

	out, err := runProducts(
		t,
		deps,
		"offers", "batch-delete",
		"--package-name", "com.example.app",
		"--product-id", "coins_100",
		"--purchase-option-id", "buy",
		"--input", inputPath,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deleted"`) || !strings.Contains(out, `"deletedCount":2`) {
		t.Fatalf("unexpected output: %s", out)
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
