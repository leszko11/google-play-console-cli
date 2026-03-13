package monetization

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"google.golang.org/api/androidpublisher/v3"
)

type fakeClient struct {
	listSubscriptionsResult gpc.SubscriptionsListInfo
	listSubscriptionsErr    error
	getSubscriptionRaw      *androidpublisher.Subscription
	getSubscriptionRawErr   error
	createSubscriptionErr   error
	updateSubscriptionErr   error
	createOfferErr          map[string]error
	updateOfferErr          map[string]error
	listOffersResult        map[string]gpc.SubscriptionOffersListInfo
	listOffersErr           map[string]error
	activateBasePlanErr     map[string]error
	activateOfferErr        map[string]error
	monetizationRegions     gpc.MonetizationRegionsInfo
	monetizationRegionsErr  error

	createdSubscription *androidpublisher.Subscription
	updatedSubscription *androidpublisher.Subscription
	createdOffers       []*androidpublisher.SubscriptionOffer
	updatedOffers       []*androidpublisher.SubscriptionOffer
	activatedBasePlans  []string
	activatedOffers     []string
}

func (f *fakeClient) ListSubscriptions(_ context.Context, _ string, _ int64, _ string, _ bool) (gpc.SubscriptionsListInfo, error) {
	if f.listSubscriptionsErr != nil {
		return gpc.SubscriptionsListInfo{}, f.listSubscriptionsErr
	}
	return f.listSubscriptionsResult, nil
}

func (f *fakeClient) CreateSubscription(_ context.Context, _ string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error) {
	if f.createSubscriptionErr != nil {
		return gpc.SubscriptionInfo{}, f.createSubscriptionErr
	}
	f.createdSubscription = subscription
	return gpc.SubscriptionInfo{ProductID: subscription.ProductId}, nil
}

func (f *fakeClient) GetSubscriptionRaw(_ context.Context, _, _ string) (*androidpublisher.Subscription, error) {
	if f.getSubscriptionRawErr != nil {
		return nil, f.getSubscriptionRawErr
	}
	return f.getSubscriptionRaw, nil
}

func (f *fakeClient) UpdateSubscription(_ context.Context, _ string, _ string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error) {
	if f.updateSubscriptionErr != nil {
		return gpc.SubscriptionInfo{}, f.updateSubscriptionErr
	}
	f.updatedSubscription = subscription
	return gpc.SubscriptionInfo{ProductID: subscription.ProductId}, nil
}

func (f *fakeClient) ActivateSubscriptionBasePlan(_ context.Context, _ string, productID, basePlanID string) ([]gpc.SubscriptionInfo, error) {
	if err := f.activateBasePlanErr[basePlanID]; err != nil {
		return nil, err
	}
	f.activatedBasePlans = append(f.activatedBasePlans, basePlanID)
	return []gpc.SubscriptionInfo{{ProductID: productID}}, nil
}

func (f *fakeClient) ListSubscriptionOffers(_ context.Context, _ string, _ string, basePlanID string, _ int64, _ string, _ bool) (gpc.SubscriptionOffersListInfo, error) {
	if err := f.listOffersErr[basePlanID]; err != nil {
		return gpc.SubscriptionOffersListInfo{}, err
	}
	return f.listOffersResult[basePlanID], nil
}

func (f *fakeClient) CreateSubscriptionOffer(_ context.Context, _ string, _, _ string, offer *androidpublisher.SubscriptionOffer) (gpc.SubscriptionOfferInfo, error) {
	if err := f.createOfferErr[offer.OfferId]; err != nil {
		return gpc.SubscriptionOfferInfo{}, err
	}
	f.createdOffers = append(f.createdOffers, offer)
	return gpc.SubscriptionOfferInfo{OfferID: offer.OfferId, BasePlanID: offer.BasePlanId}, nil
}

func (f *fakeClient) UpdateSubscriptionOffer(_ context.Context, _ string, _, _, _ string, offer *androidpublisher.SubscriptionOffer, _ string) (gpc.SubscriptionOfferInfo, error) {
	if err := f.updateOfferErr[offer.OfferId]; err != nil {
		return gpc.SubscriptionOfferInfo{}, err
	}
	f.updatedOffers = append(f.updatedOffers, offer)
	return gpc.SubscriptionOfferInfo{OfferID: offer.OfferId, BasePlanID: offer.BasePlanId}, nil
}

func (f *fakeClient) ActivateSubscriptionOffer(_ context.Context, _ string, _, basePlanID, offerID string) (gpc.SubscriptionOfferInfo, error) {
	if err := f.activateOfferErr[offerID]; err != nil {
		return gpc.SubscriptionOfferInfo{}, err
	}
	f.activatedOffers = append(f.activatedOffers, offerID)
	return gpc.SubscriptionOfferInfo{OfferID: offerID, BasePlanID: basePlanID}, nil
}

func (f *fakeClient) GetMonetizationRegions(_ context.Context, _ string) (gpc.MonetizationRegionsInfo, error) {
	if f.monetizationRegionsErr != nil {
		return gpc.MonetizationRegionsInfo{}, f.monetizationRegionsErr
	}
	return f.monetizationRegions, nil
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func writeSetupManifest(t *testing.T) string {
	t.Helper()
	return writeManifestFixture(t, ".yaml", `
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
`)
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

func TestSetupDryRun(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "setup", "--package-name", "com.example.app", "--manifest", writeSetupManifest(t), "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.createdSubscription != nil || len(client.createdOffers) > 0 {
		t.Fatalf("dry-run should not create resources: %+v %+v", client.createdSubscription, client.createdOffers)
	}
	if !strings.Contains(out, `create subscription premium`) {
		t.Fatalf("expected planned action in output: %s", out)
	}
}

func TestSetupFailsOnExistingProduct(t *testing.T) {
	client := &fakeClient{
		listSubscriptionsResult: gpc.SubscriptionsListInfo{
			Subscriptions: []gpc.SubscriptionInfo{{ProductID: "premium"}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "setup", "--package-name", "com.example.app", "--manifest", writeSetupManifest(t), "--dry-run")
	if err == nil || !strings.Contains(err.Error(), `already exists`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetupCommitWithActivation(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "setup", "--package-name", "com.example.app", "--manifest", writeSetupManifest(t), "--confirm", "--activate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"completed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.createdSubscription == nil || client.createdSubscription.ProductId != "premium" {
		t.Fatalf("unexpected subscription payload: %+v", client.createdSubscription)
	}
	if client.createdSubscription.PackageName != "com.example.app" {
		t.Fatalf("expected package name on subscription payload, got %+v", client.createdSubscription)
	}
	if len(client.createdOffers) != 1 || client.createdOffers[0].OfferId != "intro_monthly" {
		t.Fatalf("unexpected created offers: %+v", client.createdOffers)
	}
	if client.createdOffers[0].PackageName != "com.example.app" {
		t.Fatalf("expected package name on offer payload, got %+v", client.createdOffers[0])
	}
	if len(client.activatedBasePlans) != 1 || client.activatedBasePlans[0] != "monthly" {
		t.Fatalf("unexpected base plan activations: %+v", client.activatedBasePlans)
	}
	if len(client.activatedOffers) != 1 || client.activatedOffers[0] != "intro_monthly" {
		t.Fatalf("unexpected offer activations: %+v", client.activatedOffers)
	}
}

func TestSetupReportsPartialProgressOnOfferFailure(t *testing.T) {
	client := &fakeClient{
		createOfferErr: map[string]error{"intro_monthly": fmt.Errorf("boom")},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "setup", "--package-name", "com.example.app", "--manifest", writeSetupManifest(t), "--confirm")
	if err == nil || !strings.Contains(err.Error(), "failed to create offer") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"createdSubscription":true`) || !strings.Contains(out, `"createdBasePlans":["monthly"]`) {
		t.Fatalf("expected partial progress in output: %s", out)
	}
	if len(client.createdOffers) != 0 {
		t.Fatalf("offer should not be recorded as created on failure: %+v", client.createdOffers)
	}
}
