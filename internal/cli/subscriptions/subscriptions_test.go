package subscriptions

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
	list                       gpc.SubscriptionsListInfo
	listErr                    error
	get                        gpc.SubscriptionInfo
	getErr                     error
	getDiagnostic              gpc.SubscriptionDiagnosticInfo
	getDiagnosticErr           error
	batchGet                   gpc.SubscriptionsListInfo
	batchGetErr                error
	create                     gpc.SubscriptionInfo
	createErr                  error
	batchUpdate                gpc.SubscriptionsListInfo
	batchUpdateErr             error
	update                     gpc.SubscriptionInfo
	updateErr                  error
	deleteErr                  error
	archiveErr                 error
	offersList                 gpc.SubscriptionOffersListInfo
	offersListErr              error
	offersBatchGet             gpc.SubscriptionOffersListInfo
	offersBatchGetErr          error
	offersBatchUpdate          gpc.SubscriptionOffersListInfo
	offersBatchUpdateErr       error
	offersBatchUpdateStates    gpc.SubscriptionOffersListInfo
	offersBatchUpdateStatesErr error
	offerGet                   gpc.SubscriptionOfferInfo
	offerGetErr                error
	offerCreate                gpc.SubscriptionOfferInfo
	offerCreateErr             error
	offerActivate              gpc.SubscriptionOfferInfo
	offerActivateErr           error
	offerDeactivate            gpc.SubscriptionOfferInfo
	offerDeactivateErr         error
	offerUpdate                gpc.SubscriptionOfferInfo
	offerUpdateErr             error
	offerDeleteErr             error
	basePlanUpdates            []gpc.SubscriptionInfo
	basePlanActErr             error
	basePlanDeactErr           error
	basePlanBatchStateErr      error
	basePlanDeleteErr          error
	basePlanMigrateErr         error
	basePlanBatchMigrateErr    error
	basePlanMigrateInput       *androidpublisher.MigrateBasePlanPricesRequest
	basePlanBatchStateInput    []*androidpublisher.UpdateBasePlanStateRequest
	basePlanBatchMigrateInput  []*androidpublisher.MigrateBasePlanPricesRequest
	basePlanBatchMigrateCount  int
	capturedInput              *androidpublisher.Subscription
	capturedOfferInput         *androidpublisher.SubscriptionOffer
	captured                   struct {
		pageSize int64
		pageTok  string
		paginate bool
	}
	productID                string
	productIDs               []string
	basePlanID               string
	offerID                  string
	offerIDs                 []string
	updateMask               string
	updateRequests           []*androidpublisher.UpdateSubscriptionRequest
	offerUpdateRequests      []*androidpublisher.UpdateSubscriptionOfferRequest
	offerStateUpdateRequests []*androidpublisher.UpdateSubscriptionOfferStateRequest
}

func (f *fakeClient) ListSubscriptions(_ context.Context, _ string, pageSize int64, pageToken string, paginate bool) (gpc.SubscriptionsListInfo, error) {
	f.captured.pageSize = pageSize
	f.captured.pageTok = pageToken
	f.captured.paginate = paginate
	return f.list, f.listErr
}

func (f *fakeClient) GetSubscription(_ context.Context, _, productID string) (gpc.SubscriptionInfo, error) {
	f.productID = productID
	return f.get, f.getErr
}

func (f *fakeClient) GetSubscriptionDiagnostic(_ context.Context, _, productID string) (gpc.SubscriptionDiagnosticInfo, error) {
	f.productID = productID
	return f.getDiagnostic, f.getDiagnosticErr
}

func (f *fakeClient) BatchGetSubscriptions(_ context.Context, _ string, productIDs []string) (gpc.SubscriptionsListInfo, error) {
	f.productIDs = append([]string(nil), productIDs...)
	return f.batchGet, f.batchGetErr
}

func (f *fakeClient) CreateSubscription(_ context.Context, _ string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error) {
	f.capturedInput = subscription
	return f.create, f.createErr
}

func (f *fakeClient) BatchUpdateSubscriptions(_ context.Context, _ string, requests []*androidpublisher.UpdateSubscriptionRequest) (gpc.SubscriptionsListInfo, error) {
	f.updateRequests = append([]*androidpublisher.UpdateSubscriptionRequest(nil), requests...)
	return f.batchUpdate, f.batchUpdateErr
}

func (f *fakeClient) UpdateSubscription(_ context.Context, _, productID string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error) {
	f.productID = productID
	f.capturedInput = subscription
	return f.update, f.updateErr
}

func (f *fakeClient) DeleteSubscription(_ context.Context, _, productID string) error {
	f.productID = productID
	return f.deleteErr
}

func (f *fakeClient) ArchiveSubscription(_ context.Context, _, productID string) error {
	f.productID = productID
	return f.archiveErr
}

func (f *fakeClient) ActivateSubscriptionBasePlan(_ context.Context, _ string, productID, basePlanID string) ([]gpc.SubscriptionInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	return f.basePlanUpdates, f.basePlanActErr
}

func (f *fakeClient) DeactivateSubscriptionBasePlan(_ context.Context, _ string, productID, basePlanID string) ([]gpc.SubscriptionInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	return f.basePlanUpdates, f.basePlanDeactErr
}

func (f *fakeClient) BatchUpdateSubscriptionBasePlanStates(_ context.Context, _ string, productID string, requests []*androidpublisher.UpdateBasePlanStateRequest) ([]gpc.SubscriptionInfo, error) {
	f.productID = productID
	f.basePlanBatchStateInput = append([]*androidpublisher.UpdateBasePlanStateRequest(nil), requests...)
	return f.basePlanUpdates, f.basePlanBatchStateErr
}

func (f *fakeClient) DeleteSubscriptionBasePlan(_ context.Context, _ string, productID, basePlanID string) error {
	f.productID = productID
	f.basePlanID = basePlanID
	return f.basePlanDeleteErr
}

func (f *fakeClient) MigrateSubscriptionBasePlanPrices(_ context.Context, _ string, productID, basePlanID string, request *androidpublisher.MigrateBasePlanPricesRequest) error {
	f.productID = productID
	f.basePlanID = basePlanID
	f.basePlanMigrateInput = request
	return f.basePlanMigrateErr
}

func (f *fakeClient) BatchMigrateSubscriptionBasePlanPrices(_ context.Context, _ string, productID string, requests []*androidpublisher.MigrateBasePlanPricesRequest) (int, error) {
	f.productID = productID
	f.basePlanBatchMigrateInput = append([]*androidpublisher.MigrateBasePlanPricesRequest(nil), requests...)
	return f.basePlanBatchMigrateCount, f.basePlanBatchMigrateErr
}

func (f *fakeClient) ListSubscriptionOffers(_ context.Context, _ string, productID, basePlanID string, pageSize int64, pageToken string, paginate bool) (gpc.SubscriptionOffersListInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	f.captured.pageSize = pageSize
	f.captured.pageTok = pageToken
	f.captured.paginate = paginate
	return f.offersList, f.offersListErr
}

func (f *fakeClient) GetSubscriptionOffer(_ context.Context, _ string, productID, basePlanID, offerID string) (gpc.SubscriptionOfferInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	f.offerID = offerID
	return f.offerGet, f.offerGetErr
}

func (f *fakeClient) BatchGetSubscriptionOffers(_ context.Context, _ string, productID, basePlanID string, offerIDs []string) (gpc.SubscriptionOffersListInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	f.offerIDs = append([]string(nil), offerIDs...)
	return f.offersBatchGet, f.offersBatchGetErr
}

func (f *fakeClient) BatchUpdateSubscriptionOffers(_ context.Context, _ string, productID, basePlanID string, requests []*androidpublisher.UpdateSubscriptionOfferRequest) (gpc.SubscriptionOffersListInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	f.offerUpdateRequests = append([]*androidpublisher.UpdateSubscriptionOfferRequest(nil), requests...)
	return f.offersBatchUpdate, f.offersBatchUpdateErr
}

func (f *fakeClient) BatchUpdateSubscriptionOfferStates(_ context.Context, _ string, productID, basePlanID string, requests []*androidpublisher.UpdateSubscriptionOfferStateRequest) (gpc.SubscriptionOffersListInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	f.offerStateUpdateRequests = append([]*androidpublisher.UpdateSubscriptionOfferStateRequest(nil), requests...)
	return f.offersBatchUpdateStates, f.offersBatchUpdateStatesErr
}

func (f *fakeClient) CreateSubscriptionOffer(_ context.Context, _ string, productID, basePlanID string, offer *androidpublisher.SubscriptionOffer) (gpc.SubscriptionOfferInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	f.capturedOfferInput = offer
	return f.offerCreate, f.offerCreateErr
}

func (f *fakeClient) ActivateSubscriptionOffer(_ context.Context, _ string, productID, basePlanID, offerID string) (gpc.SubscriptionOfferInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	f.offerID = offerID
	return f.offerActivate, f.offerActivateErr
}

func (f *fakeClient) DeactivateSubscriptionOffer(_ context.Context, _ string, productID, basePlanID, offerID string) (gpc.SubscriptionOfferInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	f.offerID = offerID
	return f.offerDeactivate, f.offerDeactivateErr
}

func (f *fakeClient) UpdateSubscriptionOffer(_ context.Context, _ string, productID, basePlanID, offerID string, offer *androidpublisher.SubscriptionOffer, updateMask string) (gpc.SubscriptionOfferInfo, error) {
	f.productID = productID
	f.basePlanID = basePlanID
	f.offerID = offerID
	f.capturedOfferInput = offer
	f.updateMask = updateMask
	return f.offerUpdate, f.offerUpdateErr
}

func (f *fakeClient) DeleteSubscriptionOffer(_ context.Context, _ string, productID, basePlanID, offerID string) error {
	f.productID = productID
	f.basePlanID = basePlanID
	f.offerID = offerID
	return f.offerDeleteErr
}

func runSubscriptions(t *testing.T, deps Deps, args ...string) (string, error) {
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

func writeJSON(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}

func writeSubscriptionPayload(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "subscription.json")
	payload := `{"packageName":"com.example.app","productId":"premium_monthly","listings":[{"languageCode":"en-US","title":"Premium"}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}

func writeSubscriptionBatchUpdatePayload(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "subscription-batch-update.json")
	payload := `{"requests":[{"subscription":{"packageName":"com.example.app","productId":"premium_monthly","listings":[{"languageCode":"en-US","title":"Premium"}]},"regionsVersion":{"version":"2022/02"},"updateMask":"listings"}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}

func writeOfferPayload(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "offer.json")
	payload := `{"packageName":"com.example.app","productId":"premium_monthly","basePlanId":"monthly","offerId":"intro","phases":[{"duration":"P1M","recurrenceCount":1}],"regionalConfigs":[{"regionCode":"US","newSubscriberAvailability":true}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}

func writeOfferBatchUpdatePayload(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "offer-batch-update.json")
	payload := `{"requests":[{"allowMissing":true,"updateMask":"offerTags","subscriptionOffer":{"packageName":"com.example.app","productId":"premium_monthly","basePlanId":"monthly","offerId":"intro","offerTags":[{"tag":"cli-test"}]}}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}

func writeOfferBatchUpdateStatesPayload(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "offer-batch-update-states.json")
	payload := `{"requests":[{"activateSubscriptionOfferRequest":{"packageName":"com.example.app","productId":"premium_monthly","basePlanId":"monthly","offerId":"intro"}}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}

func writeBasePlanMigratePayload(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "base-plan-migrate.json")
	payload := `{"regionalPriceMigrations":[{"regionCode":"US","oldestAllowedPriceVersionTime":"2025-01-01T00:00:00Z"}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}

func writeBasePlanBatchMigratePayload(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "base-plan-batch-migrate.json")
	payload := `{"requests":[{"basePlanId":"monthly","regionalPriceMigrations":[{"regionCode":"US","oldestAllowedPriceVersionTime":"2025-01-01T00:00:00Z"}]},{"basePlanId":"yearly","regionalPriceMigrations":[{"regionCode":"PL","oldestAllowedPriceVersionTime":"2025-01-01T00:00:00Z"}]}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return path
}

func TestSubscriptionsList_ReturnsSubscriptions(t *testing.T) {
	bindGlobalPaginate(t, false)
	fc := &fakeClient{
		list: gpc.SubscriptionsListInfo{
			Subscriptions: []gpc.SubscriptionInfo{
				{PackageName: "com.example.app", ProductID: "premium_monthly"},
			},
			NextPageToken: "next-token",
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
		"list",
		"--package-name", "com.example.app",
		"--page-size", "100",
		"--page-token", "tok-1",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"productId":"premium_monthly"`) || !strings.Contains(out, `"nextPageToken":"next-token"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.captured.pageSize != 100 || fc.captured.pageTok != "tok-1" || fc.captured.paginate {
		t.Fatalf("unexpected list args: %+v", fc.captured)
	}
}

func TestSubscriptionsList_UsesGlobalPaginate(t *testing.T) {
	bindGlobalPaginate(t, true)
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	_, err := runSubscriptions(t, deps, "list", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !fc.captured.paginate {
		t.Fatal("expected paginate=true from global flags")
	}
}

func TestSubscriptionsGet_RequiresProductID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(t, deps, "get", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "--product-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsGet_ReturnsSubscription(t *testing.T) {
	fc := &fakeClient{
		get: gpc.SubscriptionInfo{
			PackageName: "com.example.app",
			ProductID:   "premium_yearly",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runSubscriptions(t, deps, "get", "--package-name", "com.example.app", "--product-id", "premium_yearly")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"productId":"premium_yearly"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.productID != "premium_yearly" {
		t.Fatalf("unexpected product id passed to client: %s", fc.productID)
	}
}

func TestSubscriptionsGetVerbose_ReturnsDiagnostics(t *testing.T) {
	fc := &fakeClient{
		getDiagnostic: gpc.SubscriptionDiagnosticInfo{
			PackageName:          "com.example.app",
			ProductID:            "premium_yearly",
			Archived:             true,
			BasePlanCount:        2,
			ListingCount:         1,
			RegionCount:          3,
			AvailableRegionCount: 2,
			ActiveBasePlanCount:  1,
			BasePlans: []gpc.SubscriptionBasePlanDiagnosticInfo{
				{BasePlanID: "yearly", State: "ACTIVE"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runSubscriptions(t, deps, "get", "--package-name", "com.example.app", "--product-id", "premium_yearly", "--verbose")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{
		`"productId":"premium_yearly"`,
		`"diagnostic":`,
		`"regionCount":3`,
		`"activeBasePlanCount":1`,
		`"basePlanId":"yearly"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %s", want, out)
		}
	}
	if fc.productID != "premium_yearly" {
		t.Fatalf("unexpected product id passed to client: %s", fc.productID)
	}
}

func TestSubscriptionsBatchGet_RequiresProductIDs(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(t, deps, "batch-get", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "--product-ids is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsBatchGet_ReturnsSubscriptions(t *testing.T) {
	fc := &fakeClient{
		batchGet: gpc.SubscriptionsListInfo{
			Subscriptions: []gpc.SubscriptionInfo{
				{PackageName: "com.example.app", ProductID: "premium_monthly"},
				{PackageName: "com.example.app", ProductID: "premium_yearly"},
			},
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
		"batch-get",
		"--package-name", "com.example.app",
		"--product-ids", "premium_monthly,premium_yearly",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"productId":"premium_monthly"`) || !strings.Contains(out, `"productId":"premium_yearly"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if len(fc.productIDs) != 2 || fc.productIDs[0] != "premium_monthly" || fc.productIDs[1] != "premium_yearly" {
		t.Fatalf("unexpected product IDs passed to client: %+v", fc.productIDs)
	}
}

func TestSubscriptionsList_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{listErr: errors.New("forbidden")}, nil
		},
	}

	_, err := runSubscriptions(t, deps, "list", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "failed to list subscriptions") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsCreate_ReturnsStatusCreated(t *testing.T) {
	payloadPath := writeSubscriptionPayload(t)
	fc := &fakeClient{
		create: gpc.SubscriptionInfo{
			PackageName: "com.example.app",
			ProductID:   "premium_monthly",
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
		"create",
		"--package-name", "com.example.app",
		"--input", payloadPath,
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"created"`) || !strings.Contains(out, `"productId":"premium_monthly"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedInput == nil || fc.capturedInput.ProductId != "premium_monthly" {
		t.Fatalf("unexpected payload parsed: %+v", fc.capturedInput)
	}
}

func TestSubscriptionsBatchUpdate_ReturnsStatusUpdated(t *testing.T) {
	payloadPath := writeSubscriptionBatchUpdatePayload(t)
	fc := &fakeClient{
		batchUpdate: gpc.SubscriptionsListInfo{
			Subscriptions: []gpc.SubscriptionInfo{
				{PackageName: "com.example.app", ProductID: "premium_monthly"},
			},
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
		"batch-update",
		"--package-name", "com.example.app",
		"--input", payloadPath,
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"productId":"premium_monthly"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if len(fc.updateRequests) != 1 {
		t.Fatalf("unexpected request count passed to client: %d", len(fc.updateRequests))
	}
	if fc.updateRequests[0] == nil || fc.updateRequests[0].Subscription == nil || fc.updateRequests[0].Subscription.ProductId != "premium_monthly" {
		t.Fatalf("unexpected parsed update request: %+v", fc.updateRequests[0])
	}
}

func TestSubscriptionsUpdate_RequiresProductID(t *testing.T) {
	payloadPath := writeSubscriptionPayload(t)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(
		t,
		deps,
		"update",
		"--package-name", "com.example.app",
		"--input", payloadPath,
	)
	if err == nil || !strings.Contains(err.Error(), "--product-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsDelete_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(t, deps, "delete", "--package-name", "com.example.app", "--product-id", "premium_monthly")
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsArchive_ReturnsStatusArchived(t *testing.T) {
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runSubscriptions(t, deps, "archive", "--package-name", "com.example.app", "--product-id", "premium_monthly", "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"archived"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.productID != "premium_monthly" {
		t.Fatalf("unexpected product id passed to client: %s", fc.productID)
	}
}

func TestSubscriptionsBasePlansMigratePrices_RequiresConfirm(t *testing.T) {
	payloadPath := writeBasePlanMigratePayload(t)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(
		t,
		deps,
		"base-plans", "migrate-prices",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--input", payloadPath,
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsBasePlansMigratePrices_ReturnsStatusMigrated(t *testing.T) {
	payloadPath := writeBasePlanMigratePayload(t)
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
		"base-plans", "migrate-prices",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--input", payloadPath,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"migrated"`) || !strings.Contains(out, `"basePlanId":"monthly"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.productID != "premium_monthly" || fc.basePlanID != "monthly" {
		t.Fatalf("unexpected captured IDs: product=%s basePlan=%s", fc.productID, fc.basePlanID)
	}
	if fc.basePlanMigrateInput == nil || len(fc.basePlanMigrateInput.RegionalPriceMigrations) != 1 {
		t.Fatalf("unexpected parsed migration payload: %+v", fc.basePlanMigrateInput)
	}
}

func TestSubscriptionsBasePlansBatchMigratePrices_RequiresConfirm(t *testing.T) {
	payloadPath := writeBasePlanBatchMigratePayload(t)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(
		t,
		deps,
		"base-plans", "batch-migrate-prices",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--input", payloadPath,
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsBasePlansBatchMigratePrices_ReturnsStatusMigrated(t *testing.T) {
	payloadPath := writeBasePlanBatchMigratePayload(t)
	fc := &fakeClient{basePlanBatchMigrateCount: 2}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runSubscriptions(
		t,
		deps,
		"base-plans", "batch-migrate-prices",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--input", payloadPath,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"migrated"`) || !strings.Contains(out, `"migratedCount":2`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.productID != "premium_monthly" {
		t.Fatalf("unexpected product id: %s", fc.productID)
	}
	if len(fc.basePlanBatchMigrateInput) != 2 {
		t.Fatalf("unexpected request count captured: %d", len(fc.basePlanBatchMigrateInput))
	}
}

func TestSubscriptionsOffersList_ReturnsOffers(t *testing.T) {
	bindGlobalPaginate(t, true)
	fc := &fakeClient{
		offersList: gpc.SubscriptionOffersListInfo{
			Offers: []gpc.SubscriptionOfferInfo{
				{PackageName: "com.example.app", ProductID: "premium_monthly", BasePlanID: "monthly", OfferID: "intro"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runSubscriptions(t, deps, "offers", "list", "--package-name", "com.example.app", "--product-id", "premium_monthly", "--base-plan-id", "monthly", "--page-size", "100")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"offerId":"intro"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !fc.captured.paginate || fc.basePlanID != "monthly" {
		t.Fatalf("unexpected captured values: paginate=%v basePlanID=%s", fc.captured.paginate, fc.basePlanID)
	}
}

func TestSubscriptionsOffersDelete_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(t, deps, "offers", "delete", "--package-name", "com.example.app", "--product-id", "premium_monthly", "--base-plan-id", "monthly", "--offer-id", "intro")
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsOffersCreate_ReturnsStatusCreated(t *testing.T) {
	payloadPath := writeOfferPayload(t)
	fc := &fakeClient{
		offerCreate: gpc.SubscriptionOfferInfo{PackageName: "com.example.app", ProductID: "premium_monthly", BasePlanID: "monthly", OfferID: "intro"},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runSubscriptions(t, deps, "offers", "create", "--package-name", "com.example.app", "--product-id", "premium_monthly", "--base-plan-id", "monthly", "--input", payloadPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"created"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedOfferInput == nil || fc.capturedOfferInput.OfferId != "intro" {
		t.Fatalf("unexpected offer payload parsed: %+v", fc.capturedOfferInput)
	}
}

func TestSubscriptionsOffersBatchGet_RequiresOfferIDs(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(
		t,
		deps,
		"offers", "batch-get",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
	)
	if err == nil || !strings.Contains(err.Error(), "--offer-ids is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsOffersBatchGet_ReturnsOffers(t *testing.T) {
	fc := &fakeClient{
		offersBatchGet: gpc.SubscriptionOffersListInfo{
			Offers: []gpc.SubscriptionOfferInfo{
				{PackageName: "com.example.app", ProductID: "premium_monthly", BasePlanID: "monthly", OfferID: "intro"},
				{PackageName: "com.example.app", ProductID: "premium_monthly", BasePlanID: "monthly", OfferID: "loyalty"},
			},
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
		"offers", "batch-get",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--offer-ids", "intro,loyalty",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"offerId":"intro"`) || !strings.Contains(out, `"offerId":"loyalty"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if len(fc.offerIDs) != 2 || fc.offerIDs[0] != "intro" || fc.offerIDs[1] != "loyalty" {
		t.Fatalf("unexpected offer IDs passed to client: %+v", fc.offerIDs)
	}
}

func TestSubscriptionsOffersBatchUpdate_ReturnsUpdated(t *testing.T) {
	payloadPath := writeOfferBatchUpdatePayload(t)
	fc := &fakeClient{
		offersBatchUpdate: gpc.SubscriptionOffersListInfo{
			Offers: []gpc.SubscriptionOfferInfo{
				{PackageName: "com.example.app", ProductID: "premium_monthly", BasePlanID: "monthly", OfferID: "intro"},
			},
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
		"offers", "batch-update",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--input", payloadPath,
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"offerId":"intro"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if len(fc.offerUpdateRequests) != 1 {
		t.Fatalf("unexpected request count passed to client: %d", len(fc.offerUpdateRequests))
	}
	if fc.offerUpdateRequests[0] == nil || fc.offerUpdateRequests[0].SubscriptionOffer == nil || fc.offerUpdateRequests[0].SubscriptionOffer.OfferId != "intro" {
		t.Fatalf("unexpected parsed update request: %+v", fc.offerUpdateRequests[0])
	}
}

func TestSubscriptionsOffersBatchUpdateStates_RequiresConfirm(t *testing.T) {
	payloadPath := writeOfferBatchUpdateStatesPayload(t)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(
		t,
		deps,
		"offers", "batch-update-states",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--input", payloadPath,
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsOffersBatchUpdateStates_ReturnsUpdated(t *testing.T) {
	payloadPath := writeOfferBatchUpdateStatesPayload(t)
	fc := &fakeClient{
		offersBatchUpdateStates: gpc.SubscriptionOffersListInfo{
			Offers: []gpc.SubscriptionOfferInfo{
				{PackageName: "com.example.app", ProductID: "premium_monthly", BasePlanID: "monthly", OfferID: "intro"},
			},
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
		"offers", "batch-update-states",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--input", payloadPath,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"offerId":"intro"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if len(fc.offerStateUpdateRequests) != 1 {
		t.Fatalf("unexpected request count passed to client: %d", len(fc.offerStateUpdateRequests))
	}
	if fc.offerStateUpdateRequests[0] == nil || fc.offerStateUpdateRequests[0].ActivateSubscriptionOfferRequest == nil || fc.offerStateUpdateRequests[0].ActivateSubscriptionOfferRequest.OfferId != "intro" {
		t.Fatalf("unexpected parsed state update request: %+v", fc.offerStateUpdateRequests[0])
	}
}

func TestSubscriptionsOffersActivate_ReturnsStatusActivated(t *testing.T) {
	fc := &fakeClient{
		offerActivate: gpc.SubscriptionOfferInfo{
			PackageName: "com.example.app",
			ProductID:   "premium_monthly",
			BasePlanID:  "monthly",
			OfferID:     "intro",
			State:       "ACTIVE",
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
		"offers", "activate",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--offer-id", "intro",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"activated"`) || !strings.Contains(out, `"offerId":"intro"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.productID != "premium_monthly" || fc.basePlanID != "monthly" || fc.offerID != "intro" {
		t.Fatalf("unexpected captured IDs: product=%s basePlan=%s offer=%s", fc.productID, fc.basePlanID, fc.offerID)
	}
}

func TestSubscriptionsOffersDeactivate_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runSubscriptions(
		t,
		deps,
		"offers", "deactivate",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--offer-id", "intro",
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionsOffersDeactivate_ReturnsStatusDeactivated(t *testing.T) {
	fc := &fakeClient{
		offerDeactivate: gpc.SubscriptionOfferInfo{
			PackageName: "com.example.app",
			ProductID:   "premium_monthly",
			BasePlanID:  "monthly",
			OfferID:     "intro",
			State:       "INACTIVE",
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
		"offers", "deactivate",
		"--package-name", "com.example.app",
		"--product-id", "premium_monthly",
		"--base-plan-id", "monthly",
		"--offer-id", "intro",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deactivated"`) || !strings.Contains(out, `"offerId":"intro"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.productID != "premium_monthly" || fc.basePlanID != "monthly" || fc.offerID != "intro" {
		t.Fatalf("unexpected captured IDs: product=%s basePlan=%s offer=%s", fc.productID, fc.basePlanID, fc.offerID)
	}
}
