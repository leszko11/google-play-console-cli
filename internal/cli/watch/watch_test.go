package watch

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	createEdit    gpc.EditInfo
	createEditErr error
	listTracks    []gpc.TrackInfo
	listTracksErr error
	deleteEditErr error
	deleteCalls   int
	reviews       gpc.ReviewsListInfo
	reviewsErr    error
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
					{Status: "inProgress", UserFraction: 0.5, VersionCodes: []int64{200}},
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
				{ReviewID: "r2", StarRating: 3},
			},
		}, nil
	}
	return f.reviews, nil
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func fixedClock() time.Time {
	return time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
}

func runCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	deps.PollOnce = true
	deps.Clock = fixedClock
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

func TestWatchRequiresPackageName(t *testing.T) {
	_, err := runCommand(t, Deps{})
	if err == nil || !strings.Contains(err.Error(), "--package-name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWatchRejectsLowInterval(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}
	_, err := runCommand(t, deps, "--package-name", "com.example.app", "--interval", "5")
	if err == nil || !strings.Contains(err.Error(), "at least 10") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWatchJSONOutput(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}
	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{`"packageName":"com.example.app"`, `"timestamp":"2025-06-15T12:00:00Z"`, `"reviewTotal":2`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestWatchTableOutput(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}
	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "table")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{"com.example.app", "production", "inProgress", "50.0%", "200"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestWatchRejectsUnsupportedFormat(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}
	_, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "markdown")
	if err == nil || !strings.Contains(err.Error(), "watch supports only json and table") {
		t.Fatalf("unexpected error: %v", err)
	}
}
