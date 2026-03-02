package subscriptions

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	list     gpc.SubscriptionsListInfo
	listErr  error
	get      gpc.SubscriptionInfo
	getErr   error
	captured struct {
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
