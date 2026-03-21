package health

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

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

	reviews    gpc.ReviewsListInfo
	reviewsErr error
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
					{Status: "completed", VersionCodes: []int64{100}},
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
				{ReviewID: "r2", StarRating: 4, HasReply: true},
				{ReviewID: "r3", StarRating: 4, HasReply: true},
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

func TestHealthRequiresPackageName(t *testing.T) {
	_, err := runCommand(t, Deps{})
	if err == nil || !strings.Contains(err.Error(), "--package-name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHealthJSONHealthyApp(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"grade":"healthy"`) {
		t.Fatalf("expected healthy grade: %s", out)
	}
	if !strings.Contains(out, `"score":100`) {
		t.Fatalf("expected score 100: %s", out)
	}
}

func TestHealthLowRating(t *testing.T) {
	client := &fakeClient{
		reviews: gpc.ReviewsListInfo{
			Reviews: []gpc.ReviewInfo{
				{ReviewID: "r1", StarRating: 1, HasReply: true},
				{ReviewID: "r2", StarRating: 2, HasReply: true},
				{ReviewID: "r3", StarRating: 1, HasReply: true},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"grade":"fair"`) {
		t.Fatalf("expected fair grade for low rating: %s", out)
	}
}

func TestHealthHaltedTrack(t *testing.T) {
	client := &fakeClient{
		listTracks: []gpc.TrackInfo{
			{
				Name: "production",
				Releases: []gpc.TrackReleaseInfo{
					{Status: "halted", VersionCodes: []int64{100}},
				},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"haltedTracks":1`) {
		t.Fatalf("expected halted track: %s", out)
	}
	if !strings.Contains(out, `"score":90`) {
		t.Fatalf("expected score 90: %s", out)
	}
}

func TestHealthTableOutput(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "table")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{"SCORE\t100", "GRADE\thealthy", "PACKAGE\tcom.example.app"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestHealthMarkdownOutput(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "markdown")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{
		"| field | value |",
		"| score | 100 |",
		"| grade | healthy |",
		"| check | status | impact | detail |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestHealthPendingRepliesDeduction(t *testing.T) {
	reviews := make([]gpc.ReviewInfo, 15)
	for i := range reviews {
		reviews[i] = gpc.ReviewInfo{
			ReviewID:   "r" + strings.Repeat("x", i),
			StarRating: 5,
			HasReply:   false,
		}
	}
	client := &fakeClient{
		reviews: gpc.ReviewsListInfo{Reviews: reviews},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	// 15 unreplied reviews => -15, score = 85
	if !strings.Contains(out, `"score":85`) {
		t.Fatalf("expected score 85 for pending replies: %s", out)
	}
	if !strings.Contains(out, `"grade":"fair"`) {
		t.Fatalf("expected fair grade: %s", out)
	}
}

func TestHealthCriticalGrade(t *testing.T) {
	// Low rating + many unreplied + halted = critical
	reviews := make([]gpc.ReviewInfo, 15)
	for i := range reviews {
		reviews[i] = gpc.ReviewInfo{
			ReviewID:   "r" + strings.Repeat("x", i),
			StarRating: 1,
			HasReply:   false,
		}
	}
	client := &fakeClient{
		listTracks: []gpc.TrackInfo{
			{
				Name: "production",
				Releases: []gpc.TrackReleaseInfo{
					{Status: "halted", VersionCodes: []int64{100}},
				},
			},
		},
		reviews: gpc.ReviewsListInfo{Reviews: reviews},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	// 100 - 10 (halted) - 20 (rating <3.0) - 15 (pending) = 55
	if !strings.Contains(out, `"score":55`) {
		t.Fatalf("expected score 55: %s", out)
	}
	if !strings.Contains(out, `"grade":"degraded"`) {
		t.Fatalf("expected degraded grade: %s", out)
	}
}

func TestHealthDeleteEditCalled(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected 1 delete call, got %d", client.deleteCalls)
	}
}

func TestHealthFailsWhenCreateEditFails(t *testing.T) {
	client := &fakeClient{createEditErr: errors.New("boom")}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "failed to collect health data: create edit: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHealthFailsWhenListTracksFails(t *testing.T) {
	client := &fakeClient{listTracksErr: errors.New("boom")}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "failed to collect health data: list tracks: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected delete after list-track failure, got %d calls", client.deleteCalls)
	}
}

func TestHealthFailsWhenListReviewsFails(t *testing.T) {
	client := &fakeClient{reviewsErr: errors.New("boom")}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "--package-name", "com.example.app", "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "failed to collect health data: list reviews: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}
