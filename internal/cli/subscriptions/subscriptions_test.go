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
	list          gpc.SubscriptionsListInfo
	listErr       error
	get           gpc.SubscriptionInfo
	getErr        error
	create        gpc.SubscriptionInfo
	createErr     error
	update        gpc.SubscriptionInfo
	updateErr     error
	deleteErr     error
	archiveErr    error
	capturedInput *androidpublisher.Subscription
	captured      struct {
		pageSize int64
		pageTok  string
		paginate bool
	}
	productID string
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

func (f *fakeClient) CreateSubscription(_ context.Context, _ string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error) {
	f.capturedInput = subscription
	return f.create, f.createErr
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

func writeSubscriptionPayload(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "subscription.json")
	payload := `{"packageName":"com.example.app","productId":"premium_monthly","listings":[{"languageCode":"en-US","title":"Premium"}]}`
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
