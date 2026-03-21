package reviews

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
	list         gpc.ReviewsListInfo
	listErr      error
	get          gpc.ReviewInfo
	getErr       error
	reply        gpc.ReviewReplyInfo
	replyErr     error
	capturedList struct {
		maxResults int64
		startIndex int64
		token      string
		lang       string
		paginate   bool
	}
}

func (f *fakeClient) ListReviews(_ context.Context, _ string, maxResults, startIndex int64, token, translationLanguage string, paginate bool) (gpc.ReviewsListInfo, error) {
	f.capturedList.maxResults = maxResults
	f.capturedList.startIndex = startIndex
	f.capturedList.token = token
	f.capturedList.lang = translationLanguage
	f.capturedList.paginate = paginate
	return f.list, f.listErr
}

func (f *fakeClient) GetReview(_ context.Context, _, _ string) (gpc.ReviewInfo, error) {
	return f.get, f.getErr
}

func (f *fakeClient) ReplyReview(_ context.Context, _, _, _ string) (gpc.ReviewReplyInfo, error) {
	return f.reply, f.replyErr
}

func runReviews(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		}
	}
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

func TestReviewsList_ReturnsReviews(t *testing.T) {
	bindGlobalPaginate(t, false)
	fc := &fakeClient{
		list: gpc.ReviewsListInfo{
			Reviews: []gpc.ReviewInfo{{ReviewID: "review-1", AuthorName: "Alice"}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runReviews(
		t,
		deps,
		"list",
		"--package-name", "com.example.app",
		"--max-results", "50",
		"--start-index", "10",
		"--token", "tok-1",
		"--translation-language", "pl-PL",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"reviewId":"review-1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedList.maxResults != 50 || fc.capturedList.startIndex != 10 || fc.capturedList.token != "tok-1" || fc.capturedList.lang != "pl-PL" || fc.capturedList.paginate {
		t.Fatalf("unexpected list args: %+v", fc.capturedList)
	}
}

func TestReviewsList_MinimalOutput(t *testing.T) {
	bindGlobalPaginate(t, false)
	fc := &fakeClient{
		list: gpc.ReviewsListInfo{
			Reviews: []gpc.ReviewInfo{
				{ReviewID: "review-1"},
				{ReviewID: "review-2"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runReviews(t, deps, "list", "--package-name", "com.example.app", "--output", "minimal")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if out != "review-1\nreview-2\n" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestReviewsList_RejectsUnsupportedOutput(t *testing.T) {
	bindGlobalPaginate(t, false)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runReviews(t, deps, "list", "--package-name", "com.example.app", "--output", "table")
	if err == nil || !strings.Contains(err.Error(), `unsupported output format "table"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewsList_UsesGlobalPaginate(t *testing.T) {
	bindGlobalPaginate(t, true)
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	_, err := runReviews(t, deps, "list", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !fc.capturedList.paginate {
		t.Fatal("expected paginate=true from global flags")
	}
}

func TestReviewsGet_RequiresReviewID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runReviews(t, deps, "get", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "--review-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewsReply_ReturnsStatusReplied(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				reply: gpc.ReviewReplyInfo{
					ReplyText: "Thanks",
				},
			}, nil
		},
	}

	out, err := runReviews(
		t,
		deps,
		"reply",
		"--package-name", "com.example.app",
		"--review-id", "review-1",
		"--reply-text", "Thanks",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"replied"`) || !strings.Contains(out, `"replyText":"Thanks"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReviewsReply_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{replyErr: errors.New("conflict")}, nil
		},
	}

	_, err := runReviews(
		t,
		deps,
		"reply",
		"--package-name", "com.example.app",
		"--review-id", "review-1",
		"--reply-text", "Thanks",
	)
	if err == nil || !strings.Contains(err.Error(), "failed to reply to review") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewsTriage_ReturnsPendingAndRepliedBuckets(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				list: gpc.ReviewsListInfo{
					Reviews: []gpc.ReviewInfo{
						{ReviewID: "review-1", AuthorName: "Alice", StarRating: 1, HasReply: false},
						{ReviewID: "review-2", AuthorName: "Bob", StarRating: 5, HasReply: true},
					},
				},
			}, nil
		},
	}

	out, err := runReviews(t, deps, "triage", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{`"status":"warn"`, `"pendingReplyCount":1`, `"repliedCount":1`, `"reviewId":"review-1"`, `"reviewId":"review-2"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %s", want, out)
		}
	}
}

func TestReviewsTriage_ReturnsOkWhenNoPendingReplies(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				list: gpc.ReviewsListInfo{
					Reviews: []gpc.ReviewInfo{{ReviewID: "review-1", AuthorName: "Alice", HasReply: true}},
				},
			}, nil
		},
	}

	out, err := runReviews(t, deps, "triage", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"ok"`) || !strings.Contains(out, `"pendingReplyCount":0`) {
		t.Fatalf("unexpected output: %s", out)
	}
}
