package status

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	verifyPackageErr error

	createEdit    gpc.EditInfo
	createEditErr error

	listTracks    []gpc.TrackInfo
	listTracksErr error

	deleteEditErr error
	deleteCalls   int

	reviews    gpc.ReviewsListInfo
	reviewsErr error

	subscriptions    gpc.SubscriptionsListInfo
	subscriptionsErr error
}

func (f *fakeClient) VerifyPackageAccess(_ context.Context, _ string) error {
	return f.verifyPackageErr
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	if f.createEditErr != nil {
		return gpc.EditInfo{}, f.createEditErr
	}
	if f.createEdit.ID == "" {
		return gpc.EditInfo{ID: "edit-1"}, nil
	}
	return f.createEdit, nil
}

func (f *fakeClient) ValidateEdit(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return f.deleteEditErr
}

func (f *fakeClient) ListTracks(_ context.Context, _, _ string) ([]gpc.TrackInfo, error) {
	if f.listTracksErr != nil {
		return nil, f.listTracksErr
	}
	if len(f.listTracks) == 0 {
		return []gpc.TrackInfo{
			{
				Name: "production",
				Releases: []gpc.TrackReleaseInfo{
					{Status: "inProgress", UserFraction: 0.1, VersionCodes: []int64{123}},
				},
			},
			{
				Name: "beta",
				Releases: []gpc.TrackReleaseInfo{
					{Status: "completed", VersionCodes: []int64{122}},
				},
			},
		}, nil
	}
	return f.listTracks, nil
}

func (f *fakeClient) ListReviews(_ context.Context, _ string, _, _ int64, _, _ string, _ bool) (gpc.ReviewsListInfo, error) {
	if f.reviewsErr != nil {
		return gpc.ReviewsListInfo{}, f.reviewsErr
	}
	if len(f.reviews.Reviews) == 0 {
		return gpc.ReviewsListInfo{
			Reviews: []gpc.ReviewInfo{
				{ReviewID: "r1", StarRating: 5, HasReply: true},
				{ReviewID: "r2", StarRating: 4},
				{ReviewID: "r3", StarRating: 1},
			},
		}, nil
	}
	return f.reviews, nil
}

func (f *fakeClient) ListSubscriptions(_ context.Context, _ string, _ int64, _ string, _ bool) (gpc.SubscriptionsListInfo, error) {
	if f.subscriptionsErr != nil {
		return gpc.SubscriptionsListInfo{}, f.subscriptionsErr
	}
	if len(f.subscriptions.Subscriptions) == 0 {
		return gpc.SubscriptionsListInfo{
			Subscriptions: []gpc.SubscriptionInfo{{ProductID: "monthly"}, {ProductID: "yearly"}},
		}, nil
	}
	return f.subscriptions, nil
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
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

func TestStatusRequiresPackageName(t *testing.T) {
	_, err := runCommand(t, Deps{})
	if err == nil || !strings.Contains(err.Error(), "--package-name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusJSONSuccess(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"packageName":"com.example.app"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, `"alerts":["2 unreplied reviews"]`) {
		t.Fatalf("unexpected alerts: %s", out)
	}
	if !strings.Contains(out, `"averageRating":3.3333333333333335`) {
		t.Fatalf("unexpected reviews aggregate: %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected delete edit call, got %d", client.deleteCalls)
	}
}

func TestStatusReviewsFailureIsNonFatal(t *testing.T) {
	client := &fakeClient{reviewsErr: context.DeadlineExceeded}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"reviewsError":"failed to list reviews: context deadline exceeded"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, `"tracks":[`) {
		t.Fatalf("expected tracks section: %s", out)
	}
}

func TestStatusTrackReadFailureIsNonFatal(t *testing.T) {
	client := &fakeClient{listTracksErr: context.DeadlineExceeded}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"trackError":"failed to list tracks: context deadline exceeded"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, `"reviews":{"total":3`) {
		t.Fatalf("expected reviews section: %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected delete edit call, got %d", client.deleteCalls)
	}
}

func TestStatusTableOutput(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"STATUS\twarn",
		"PACKAGE\tcom.example.app",
		"TRACK\tRELEASE_STATUS\tUSER_FRACTION\tVERSION_CODES",
		"production\tinProgress\t0.100\t123",
		"REVIEW_TOTAL\t3",
		"ALERT\t2 unreplied reviews",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestStatusMarkdownOutput(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "markdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"| field | value |",
		"| status | warn |",
		"| package | com.example.app |",
		"| track | releaseStatus | userFraction | versionCodes |",
		"| production | inProgress | 0.100 | 123 |",
		"| alert |",
		"| 2 unreplied reviews |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestStatusMinimalOutput(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "minimal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The fake client returns reviews with pending replies, so status should be "warn".
	want := "warn\n"
	if out != want {
		t.Fatalf("unexpected output %q, want %q", out, want)
	}
}

func TestStatusYAMLOutput(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"packageName: com.example.app", "status: warn", "pendingReply: 2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestStatusIncludeSectionsAddsDashboardData(t *testing.T) {
	root := t.TempDir()
	listingDir := filepath.Join(root, "play", "listing", "en-US")
	screenshotsDir := filepath.Join(root, "play", "screenshots", "en-US")
	subscriptionsDir := filepath.Join(root, "play", "subscriptions")
	changelogDir := filepath.Join(root, "play", "changelog")
	for _, dir := range []string{listingDir, screenshotsDir, subscriptionsDir, changelogDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for name, contents := range map[string]string{
		filepath.Join(listingDir, "title.txt"):             "Example title",
		filepath.Join(listingDir, "short-description.txt"): "Example short description",
		filepath.Join(listingDir, "full-description.txt"):  "Example full description for Android listing.",
		filepath.Join(subscriptionsDir, "monthly.json"):    `{"subscription":{"productId":"monthly","basePlans":[{"basePlanId":"monthly"}],"listings":{"en-US":{"title":"Monthly","benefits":["One"]}}}}`,
	} {
		if err := os.WriteFile(name, []byte(contents), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		LoadProject: func() (config.ProjectConfigInfo, error) {
			return config.ProjectConfigInfo{Config: config.ProjectConfig{
				DefaultLocale:    "en-US",
				ListingDir:       filepath.Join(root, "play", "listing"),
				ScreenshotsDir:   filepath.Join(root, "play", "screenshots"),
				SubscriptionsDir: subscriptionsDir,
				ChangelogDir:     changelogDir,
			}}, nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--include", "tracks,reviews,auth,readiness,listing,subscriptions", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"auth":{`, `"readiness":{`, `"listing":{`, `"subscriptions":{`, `"remoteCount":2`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestStatusIncludeAbsentPreservesDefaultShape(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig:  func() (config.Config, error) { return defaultConfig(), nil },
		LoadProject: func() (config.ProjectConfigInfo, error) { return config.ProjectConfigInfo{}, nil },
		NewClient:   func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, `"auth":`) || strings.Contains(out, `"listing":`) || strings.Contains(out, `"subscriptions":`) {
		t.Fatalf("expected default shape without new sections, got %s", out)
	}
}
