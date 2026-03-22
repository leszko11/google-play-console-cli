package shared

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeEditCommitClient struct {
	commitFn      func(context.Context, string, string, bool) (gpc.EditInfo, error)
	getTrackFn    func(context.Context, string, string, string) (gpc.TrackInfo, error)
	updateTrackFn func(context.Context, string, string, string, gpc.TrackUpdate) (gpc.TrackInfo, error)
}

func (f fakeEditCommitClient) CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) (gpc.EditInfo, error) {
	return f.commitFn(ctx, packageName, editID, changesNotSentForReview)
}

func (f fakeEditCommitClient) GetTrack(ctx context.Context, packageName, editID, trackName string) (gpc.TrackInfo, error) {
	return f.getTrackFn(ctx, packageName, editID, trackName)
}

func (f fakeEditCommitClient) UpdateTrack(ctx context.Context, packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
	return f.updateTrackFn(ctx, packageName, editID, trackName, update)
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
		`Automatic track fixes are not applied by gpc`,
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

func TestCommitEditWithOptions_AutoFixesDraftTrackConflict(t *testing.T) {
	var commitCalls []bool
	var updated gpc.TrackUpdate
	fixed := false
	client := fakeEditCommitClient{
		commitFn: func(_ context.Context, packageName, editID string, changesNotSentForReview bool) (gpc.EditInfo, error) {
			commitCalls = append(commitCalls, changesNotSentForReview)
			if packageName != "com.example.app" || editID != "edit-1" {
				t.Fatalf("unexpected target %q %q", packageName, editID)
			}
			if !fixed {
				return gpc.EditInfo{}, errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app.")
			}
			return gpc.EditInfo{ID: "edit-1"}, nil
		},
		getTrackFn: func(_ context.Context, _, _, trackName string) (gpc.TrackInfo, error) {
			if trackName != "internal" {
				t.Fatalf("unexpected track %q", trackName)
			}
			return gpc.TrackInfo{
				Name: "internal",
				Releases: []gpc.TrackReleaseInfo{{
					Name:           "1.0.0",
					Status:         "completed",
					VersionCodes:   []int64{123},
					UpdatePriority: 2,
					ReleaseNotes:   []gpc.LocalizedText{{Language: "en-US", Text: "note"}},
				}},
			}, nil
		},
		updateTrackFn: func(_ context.Context, _, _, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
			if trackName != "internal" {
				t.Fatalf("unexpected track %q", trackName)
			}
			updated = update
			fixed = true
			return gpc.TrackInfo{Name: "internal"}, nil
		},
	}

	result, err := CommitEditWithOptions(context.Background(), client, "com.example.app", "edit-1", EditCommitOptions{
		AutoFixDraftTrack: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commitCalls) != 2 || commitCalls[0] || commitCalls[1] {
		t.Fatalf("unexpected commit calls: %+v", commitCalls)
	}
	if updated.Status != "draft" || updated.ReleaseName != "1.0.0" || len(updated.VersionCodes) != 1 || updated.VersionCodes[0] != 123 {
		t.Fatalf("unexpected updated track: %+v", updated)
	}
	if !result.DraftTrackAutoFixed {
		t.Fatalf("expected draft track to be auto-fixed: %+v", result)
	}
}

func TestCommitEditWithOptions_AutoFixesAfterReviewFallbackRetry(t *testing.T) {
	var commitCalls []bool
	fixed := false
	client := fakeEditCommitClient{
		commitFn: func(_ context.Context, _, _ string, changesNotSentForReview bool) (gpc.EditInfo, error) {
			commitCalls = append(commitCalls, changesNotSentForReview)
			switch len(commitCalls) {
			case 1:
				return gpc.EditInfo{}, errors.New("androidpublisher api error (400): Changes cannot be sent for review automatically. Please set the query parameter changesNotSentForReview to true.")
			case 2:
				if fixed {
					t.Fatal("expected draft-app failure before auto-fix")
				}
				return gpc.EditInfo{}, errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app.")
			default:
				if !fixed {
					t.Fatal("expected auto-fix before final retry")
				}
				return gpc.EditInfo{ID: "edit-1"}, nil
			}
		},
		getTrackFn: func(_ context.Context, _, _, _ string) (gpc.TrackInfo, error) {
			return gpc.TrackInfo{
				Name: "internal",
				Releases: []gpc.TrackReleaseInfo{{
					Status:       "completed",
					VersionCodes: []int64{321},
				}},
			}, nil
		},
		updateTrackFn: func(_ context.Context, _, _, _ string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
			if update.Status != "draft" {
				t.Fatalf("unexpected update: %+v", update)
			}
			fixed = true
			return gpc.TrackInfo{Name: "internal"}, nil
		},
	}

	result, err := CommitEditWithOptions(context.Background(), client, "com.example.app", "edit-1", EditCommitOptions{
		AutoFixDraftTrack: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commitCalls) != 3 || commitCalls[0] || !commitCalls[1] || !commitCalls[2] {
		t.Fatalf("unexpected commit calls: %+v", commitCalls)
	}
	if !result.ChangesNotSentForReview || !result.RetriedWithChangesNotSentForReview || !result.DraftTrackAutoFixed {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCommitEditWithOptions_ReportsAutoFixFailure(t *testing.T) {
	client := fakeEditCommitClient{
		commitFn: func(_ context.Context, _, _ string, _ bool) (gpc.EditInfo, error) {
			return gpc.EditInfo{}, errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app.")
		},
		getTrackFn: func(_ context.Context, _, _, _ string) (gpc.TrackInfo, error) {
			return gpc.TrackInfo{
				Name: "internal",
				Releases: []gpc.TrackReleaseInfo{{
					Status:       "completed",
					VersionCodes: []int64{321},
				}},
			}, nil
		},
		updateTrackFn: func(_ context.Context, _, _, _ string, _ gpc.TrackUpdate) (gpc.TrackInfo, error) {
			return gpc.TrackInfo{}, errors.New("permission denied")
		},
	}

	_, err := CommitEditWithOptions(context.Background(), client, "com.example.app", "edit-1", EditCommitOptions{
		AutoFixDraftTrack: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{
		`Only releases with status draft may be created on draft app`,
		`gpc tried to auto-fix the internal track`,
		`permission denied`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in error: %v", want, err)
		}
	}
}
