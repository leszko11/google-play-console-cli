package shared

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeEditCommitClient struct {
	commitFn func(context.Context, string, string, bool) (gpc.EditInfo, error)
}

func (f fakeEditCommitClient) CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) (gpc.EditInfo, error) {
	return f.commitFn(ctx, packageName, editID, changesNotSentForReview)
}

func TestCommitEditWithReviewFallback_RetriesWithChangesNotSentForReview(t *testing.T) {
	var flags []bool
	client := fakeEditCommitClient{
		commitFn: func(_ context.Context, packageName, editID string, changesNotSentForReview bool) (gpc.EditInfo, error) {
			flags = append(flags, changesNotSentForReview)
			if packageName != "com.example.app" || editID != "edit-1" {
				t.Fatalf("unexpected target %q %q", packageName, editID)
			}
			if !changesNotSentForReview {
				return gpc.EditInfo{}, errors.New("androidpublisher api error (400): Changes cannot be sent for review automatically. Please set the query parameter changesNotSentForReview to true.")
			}
			return gpc.EditInfo{ID: "edit-1"}, nil
		},
	}

	result, err := CommitEditWithReviewFallback(context.Background(), client, "com.example.app", "edit-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags) != 2 || flags[0] || !flags[1] {
		t.Fatalf("unexpected retry flags: %+v", flags)
	}
	if !result.ChangesNotSentForReview || !result.RetriedWithChangesNotSentForReview {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Edit.ID != "edit-1" {
		t.Fatalf("unexpected edit: %+v", result.Edit)
	}
}

func TestCommitEditWithReviewFallback_DoesNotRetryWhenExplicitlyEnabled(t *testing.T) {
	var calls int
	client := fakeEditCommitClient{
		commitFn: func(_ context.Context, _, _ string, changesNotSentForReview bool) (gpc.EditInfo, error) {
			calls++
			if !changesNotSentForReview {
				t.Fatal("expected explicit changesNotSentForReview=true")
			}
			return gpc.EditInfo{ID: "edit-1"}, nil
		},
	}

	result, err := CommitEditWithReviewFallback(context.Background(), client, "com.example.app", "edit-1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one commit call, got %d", calls)
	}
	if !result.ChangesNotSentForReview || result.RetriedWithChangesNotSentForReview {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCommitEditWithReviewFallback_ExplainsDraftTrackConflict(t *testing.T) {
	client := fakeEditCommitClient{
		commitFn: func(_ context.Context, _, _ string, _ bool) (gpc.EditInfo, error) {
			return gpc.EditInfo{}, errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app.")
		},
	}

	_, err := CommitEditWithReviewFallback(context.Background(), client, "com.example.app", "edit-1", true)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{
		`Only releases with status draft may be created on draft app`,
		`track release with status "completed"`,
		"`gpc release init --package-name <package> --dir ./play`",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in error: %v", want, err)
		}
	}
}

func TestCommitEditWithReviewFallback_DoesNotRetryUnrelatedError(t *testing.T) {
	var calls int
	client := fakeEditCommitClient{
		commitFn: func(_ context.Context, _, _ string, _ bool) (gpc.EditInfo, error) {
			calls++
			return gpc.EditInfo{}, errors.New("conflict")
		},
	}

	_, err := CommitEditWithReviewFallback(context.Background(), client, "com.example.app", "edit-1", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected one commit call, got %d", calls)
	}
	if err.Error() != "conflict" {
		t.Fatalf("unexpected error: %v", err)
	}
}
