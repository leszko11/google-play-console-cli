package appinit

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"google.golang.org/api/androidpublisher/v3"
)

type fakeClient struct {
	listListingsResult      []gpc.ListingInfo
	listTracksResult        []gpc.TrackInfo
	listProductsResult      gpc.OneTimeProductsListInfo
	listSubscriptionsResult gpc.SubscriptionsListInfo
	listSubscriptionsErr    error
	createSubscriptionErr   error
	createOfferErr          map[string]error
	createEditErr           error

	createEditCalls     int
	deleteEditCalls     int
	validateEditCalls   int
	commitEditCalls     int
	updateListingCalls  int
	deleteImagesCalls   int
	uploadImageCalls    int
	updateDetailsCalls  int
	createSubCalls      int
	createOfferCalls    int
	createdSubscription *androidpublisher.Subscription
	createdOffers       []*androidpublisher.SubscriptionOffer
	appDetailsUpdate    gpc.AppDetailsUpdate
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	if f.createEditErr != nil {
		return gpc.EditInfo{}, f.createEditErr
	}
	f.createEditCalls++
	return gpc.EditInfo{ID: fmt.Sprintf("edit-%d", f.createEditCalls)}, nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteEditCalls++
	return nil
}

func (f *fakeClient) ValidateEdit(_ context.Context, _, _ string) error {
	f.validateEditCalls++
	return nil
}

func (f *fakeClient) CommitEdit(_ context.Context, _, _ string) (gpc.EditInfo, error) {
	f.commitEditCalls++
	return gpc.EditInfo{ID: "committed"}, nil
}

func (f *fakeClient) UpdateAppDetails(_ context.Context, _, _ string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error) {
	f.updateDetailsCalls++
	f.appDetailsUpdate = update
	return gpc.AppDetailsInfo{DefaultLanguage: update.DefaultLanguage, ContactEmail: update.ContactEmail}, nil
}

func (f *fakeClient) GetAppDetails(_ context.Context, _, _ string) (gpc.AppDetailsInfo, error) {
	return gpc.AppDetailsInfo{
		DefaultLanguage: "en-US",
		ContactEmail:    "support@example.com",
	}, nil
}

func (f *fakeClient) ListListings(_ context.Context, _, _ string) ([]gpc.ListingInfo, error) {
	return f.listListingsResult, nil
}

func (f *fakeClient) UpdateListing(_ context.Context, _, _, _ string, _ gpc.ListingUpdate) (gpc.ListingInfo, error) {
	f.updateListingCalls++
	return gpc.ListingInfo{}, nil
}

func (f *fakeClient) DeleteListing(_ context.Context, _, _, _ string) error { return nil }

func (f *fakeClient) DeleteAllImages(_ context.Context, _, _, _, _ string) ([]gpc.ImageInfo, error) {
	f.deleteImagesCalls++
	return nil, nil
}

func (f *fakeClient) UploadImage(_ context.Context, _, _, _, _, _ string) (gpc.ImageInfo, error) {
	f.uploadImageCalls++
	return gpc.ImageInfo{}, nil
}

func (f *fakeClient) ListImages(_ context.Context, _, _, _, _ string) ([]gpc.ImageInfo, error) {
	return nil, nil
}

func (f *fakeClient) ListTracks(_ context.Context, _, _ string) ([]gpc.TrackInfo, error) {
	return f.listTracksResult, nil
}

func (f *fakeClient) ListOneTimeProducts(_ context.Context, _ string, _ int64, _ string, _ bool) (gpc.OneTimeProductsListInfo, error) {
	return f.listProductsResult, nil
}

func (f *fakeClient) GetOneTimeProductResource(_ context.Context, _, productID string) (*androidpublisher.OneTimeProduct, error) {
	return &androidpublisher.OneTimeProduct{ProductId: productID}, nil
}

func (f *fakeClient) ListSubscriptions(_ context.Context, _ string, _ int64, _ string, _ bool) (gpc.SubscriptionsListInfo, error) {
	if f.listSubscriptionsErr != nil {
		return gpc.SubscriptionsListInfo{}, f.listSubscriptionsErr
	}
	return f.listSubscriptionsResult, nil
}

func (f *fakeClient) GetSubscriptionResource(_ context.Context, _, productID string) (*androidpublisher.Subscription, error) {
	return &androidpublisher.Subscription{ProductId: productID}, nil
}

func (f *fakeClient) GetLatestRegionsVersion(_ context.Context, _ string) (string, error) {
	return "2026/01", nil
}

func (f *fakeClient) CreateSubscription(_ context.Context, _ string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error) {
	if f.createSubscriptionErr != nil {
		return gpc.SubscriptionInfo{}, f.createSubscriptionErr
	}
	f.createSubCalls++
	f.createdSubscription = subscription
	return gpc.SubscriptionInfo{ProductID: subscription.ProductId}, nil
}

func (f *fakeClient) GetSubscriptionRaw(_ context.Context, _, _ string) (*androidpublisher.Subscription, error) {
	return nil, nil
}

func (f *fakeClient) UpdateSubscription(_ context.Context, _, _ string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error) {
	return gpc.SubscriptionInfo{ProductID: subscription.ProductId}, nil
}

func (f *fakeClient) ActivateSubscriptionBasePlan(_ context.Context, _, _, _ string) ([]gpc.SubscriptionInfo, error) {
	return []gpc.SubscriptionInfo{{ProductID: "premium"}}, nil
}

func (f *fakeClient) ListSubscriptionOffers(_ context.Context, _, _, _ string, _ int64, _ string, _ bool) (gpc.SubscriptionOffersListInfo, error) {
	return gpc.SubscriptionOffersListInfo{}, nil
}

func (f *fakeClient) CreateSubscriptionOffer(_ context.Context, _ string, _, _ string, offer *androidpublisher.SubscriptionOffer) (gpc.SubscriptionOfferInfo, error) {
	if err := f.createOfferErr[offer.OfferId]; err != nil {
		return gpc.SubscriptionOfferInfo{}, err
	}
	f.createOfferCalls++
	f.createdOffers = append(f.createdOffers, offer)
	return gpc.SubscriptionOfferInfo{OfferID: offer.OfferId}, nil
}

func (f *fakeClient) UpdateSubscriptionOffer(_ context.Context, _, _, _, _ string, offer *androidpublisher.SubscriptionOffer, _ string) (gpc.SubscriptionOfferInfo, error) {
	return gpc.SubscriptionOfferInfo{OfferID: offer.OfferId}, nil
}

func (f *fakeClient) ActivateSubscriptionOffer(_ context.Context, _, _, _, offerID string) (gpc.SubscriptionOfferInfo, error) {
	return gpc.SubscriptionOfferInfo{OfferID: offerID}, nil
}

func (f *fakeClient) GetMonetizationRegions(_ context.Context, _ string) (gpc.MonetizationRegionsInfo, error) {
	return gpc.MonetizationRegionsInfo{}, nil
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func writeListingFixture(t *testing.T, root string) string {
	t.Helper()
	listingDir := filepath.Join(root, "listings")
	mustWrite := func(path, contents string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	mustWrite(filepath.Join(listingDir, "en-US", "title.txt"), "Title")
	mustWrite(filepath.Join(listingDir, "en-US", "short-description.txt"), "Short")
	mustWrite(filepath.Join(listingDir, "en-US", "full-description.txt"), "Full")
	mustWrite(filepath.Join(listingDir, "en-US", "images", "phoneScreenshots", "1.png"), "one")
	return listingDir
}

func writeMonetizationManifest(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "monetization.yaml")
	contents := `
subscription:
  productId: premium
  listings:
    en-US:
      title: Premium
      description: Unlock all features
  basePlans:
    - basePlanId: monthly
      billingPeriod: P1M
      regionalConfigs:
        - regionCode: US
          price:
            currencyCode: USD
            units: 9
            nanos: 990000000
  offers:
    - offerId: intro_monthly
      basePlanId: monthly
      phases:
        - type: FREE_TRIAL
          duration: P7D
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write monetization manifest: %v", err)
	}
	return path
}

func writeAppInitManifest(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "app-init.yaml")
	contents := `
appDetails:
  defaultLanguage: en-US
  contactEmail: support@example.com
listing:
  dir: ./listings
monetization:
  manifest: ./monetization.yaml
  activate: true
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write app init manifest: %v", err)
	}
	return path
}

func runCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		}
	}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestLoadManifestResolvesRelativePaths(t *testing.T) {
	root := t.TempDir()
	listingDir := writeListingFixture(t, root)
	monetizationPath := writeMonetizationManifest(t, root)
	manifestPath := writeAppInitManifest(t, root)

	got, err := loadManifest(manifestPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Listing == nil || got.Listing.Dir != listingDir {
		t.Fatalf("unexpected listing dir: %+v", got.Listing)
	}
	if got.Monetization == nil || got.Monetization.Manifest != monetizationPath {
		t.Fatalf("unexpected monetization manifest: %+v", got.Monetization)
	}
}

func TestAppInitDryRun(t *testing.T) {
	root := t.TempDir()
	writeListingFixture(t, root)
	writeMonetizationManifest(t, root)
	manifestPath := writeAppInitManifest(t, root)
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifestPath, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.updateDetailsCalls != 1 || client.createEditCalls != 2 || client.deleteEditCalls != 2 {
		t.Fatalf("unexpected dry-run edit flow: %+v", client)
	}
	if client.updateListingCalls != 0 || client.createSubCalls != 0 || client.createOfferCalls != 0 {
		t.Fatalf("dry-run should not apply listing/monetization writes: %+v", client)
	}
}

func TestAppInitCommit(t *testing.T) {
	root := t.TempDir()
	writeListingFixture(t, root)
	writeMonetizationManifest(t, root)
	manifestPath := writeAppInitManifest(t, root)
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifestPath, "--confirm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"completed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.updateDetailsCalls != 1 || client.commitEditCalls != 2 {
		t.Fatalf("unexpected edit flow: %+v", client)
	}
	if client.updateListingCalls != 1 || client.uploadImageCalls != 1 {
		t.Fatalf("listing sync did not run: %+v", client)
	}
	if client.createSubCalls != 1 || client.createOfferCalls != 1 {
		t.Fatalf("monetization setup did not run: %+v", client)
	}
	if client.appDetailsUpdate.ContactEmail != "support@example.com" {
		t.Fatalf("unexpected app details update: %+v", client.appDetailsUpdate)
	}
}

func TestAppInitStopsOnListingFailure(t *testing.T) {
	root := t.TempDir()
	writeMonetizationManifest(t, root)
	manifestPath := filepath.Join(root, "app-init.yaml")
	if err := os.WriteFile(manifestPath, []byte(`
listing:
  dir: ./missing
monetization:
  manifest: ./monetization.yaml
`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifestPath, "--confirm")
	if err == nil || !strings.Contains(err.Error(), "read listings directory") {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.createSubCalls != 0 {
		t.Fatalf("monetization should not run after listing failure: %+v", client)
	}
}

func TestAppInitExportWritesRoundTripFiles(t *testing.T) {
	root := t.TempDir()
	client := &fakeClient{
		listListingsResult: []gpc.ListingInfo{
			{
				Language:         "en-US",
				Title:            "Title",
				ShortDescription: "Short",
				FullDescription:  "Full",
			},
		},
		listTracksResult: []gpc.TrackInfo{
			{
				Name: "production",
				Releases: []gpc.TrackReleaseInfo{
					{
						Name: "1.0.0",
						ReleaseNotes: []gpc.LocalizedText{
							{Language: "en-US", Text: "Release note"},
						},
					},
				},
			},
		},
		listProductsResult: gpc.OneTimeProductsListInfo{
			Products: []gpc.OneTimeProductInfo{{ProductID: "coins_100"}},
		},
		listSubscriptionsResult: gpc.SubscriptionsListInfo{
			Subscriptions: []gpc.SubscriptionInfo{{ProductID: "premium_monthly"}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "export", "--package-name", "com.example.app", "--dir", root, "--skip-images", "--write-project-config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, path := range []string{
		filepath.Join(root, "appinit.yaml"),
		filepath.Join(root, ".gpc.yaml"),
		filepath.Join(root, "listing", "en-US", "title.txt"),
		filepath.Join(root, "changelog", "production", "en-US.txt"),
		filepath.Join(root, "products", "coins_100.json"),
		filepath.Join(root, "subscriptions", "premium_monthly.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected exported file %s: %v", path, err)
		}
	}
	if !strings.Contains(out, `"sections":["app-details","changelog","listing","products","subscriptions"]`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.deleteEditCalls != 1 {
		t.Fatalf("expected export cleanup edit delete, got %d", client.deleteEditCalls)
	}
}

func TestAppInitUsesProjectConfigManifestDefault(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	writeListingFixture(t, root)
	writeMonetizationManifest(t, root)
	manifestPath := writeAppInitManifest(t, root)
	if err := os.WriteFile(filepath.Join(root, ".gpc.yaml"), []byte("package-name: com.example.app\nappinit-manifest: ./app-init.yaml\n"), 0o600); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "--confirm")
	if err != nil {
		t.Fatalf("unexpected error using project manifest default %s: %v", manifestPath, err)
	}
	if client.commitEditCalls != 2 {
		t.Fatalf("expected appinit to commit both sections, got %+v", client)
	}
}
