package shared

import (
	"context"
	"fmt"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type EditCommitClient interface {
	CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) (gpc.EditInfo, error)
}

type EditCommitOptions struct {
	ChangesNotSentForReview bool
	AutoFixDraftTrack       bool
}

type EditCommitResult struct {
	Edit                               gpc.EditInfo
	ChangesNotSentForReview            bool
	RetriedWithChangesNotSentForReview bool
	DraftTrackAutoFixed                bool
}

func CommitEditWithReviewFallback(ctx context.Context, client EditCommitClient, packageName, editID string, changesNotSentForReview bool) (EditCommitResult, error) {
	return CommitEditWithOptions(ctx, client, packageName, editID, EditCommitOptions{
		ChangesNotSentForReview: changesNotSentForReview,
	})
}

func CommitEditWithOptions(ctx context.Context, client EditCommitClient, packageName, editID string, opts EditCommitOptions) (EditCommitResult, error) {
	result := EditCommitResult{
		ChangesNotSentForReview: opts.ChangesNotSentForReview,
	}

	edit, err := client.CommitEdit(ctx, packageName, editID, opts.ChangesNotSentForReview)
	if err == nil {
		result.Edit = edit
		return result, nil
	}

	if !opts.ChangesNotSentForReview && shouldRetryCommitWithChangesNotSentForReview(err) {
		edit, err = client.CommitEdit(ctx, packageName, editID, true)
		if err == nil {
			result.Edit = edit
			result.ChangesNotSentForReview = true
			result.RetriedWithChangesNotSentForReview = true
			return result, nil
		}
		result.ChangesNotSentForReview = true
		result.RetriedWithChangesNotSentForReview = true
	}

	if opts.AutoFixDraftTrack && isDraftAppTrackConflict(err) {
		if fixErr := autoFixDraftAppInternalTrack(ctx, client, packageName, editID); fixErr != nil {
			return result, explainDraftAppTrackAutoFixError(err, fixErr)
		}
		edit, retryErr := client.CommitEdit(ctx, packageName, editID, result.ChangesNotSentForReview)
		if retryErr != nil {
			return EditCommitResult{
				ChangesNotSentForReview:            result.ChangesNotSentForReview,
				RetriedWithChangesNotSentForReview: result.RetriedWithChangesNotSentForReview,
				DraftTrackAutoFixed:                true,
			}, explainDraftAppTrackAutoFixRetryError(retryErr)
		}
		result.Edit = edit
		result.DraftTrackAutoFixed = true
		return result, nil
	}

	return result, explainDraftAppCommitError(err)
}

func shouldRetryCommitWithChangesNotSentForReview(err error) bool {
	if err == nil {
		return false
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "changesnotsentforreview") ||
		strings.Contains(lower, "changes not sent for review") ||
		strings.Contains(lower, "cannot be sent for review automatically") ||
		strings.Contains(lower, "not be reviewed until they are explicitly sent for review") ||
		strings.Contains(lower, "set the query parameter changesnotsentforreview to true")
}

type draftAppTrackFixClient interface {
	GetTrack(ctx context.Context, packageName, editID, trackName string) (gpc.TrackInfo, error)
	UpdateTrack(ctx context.Context, packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error)
}

func isDraftAppTrackConflict(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "only releases with status draft may be created on draft app")
}

func autoFixDraftAppInternalTrack(ctx context.Context, client EditCommitClient, packageName, editID string) error {
	fixer, ok := client.(draftAppTrackFixClient)
	if !ok {
		return fmt.Errorf("this command client cannot inspect and rewrite tracks")
	}

	track, err := fixer.GetTrack(ctx, packageName, editID, "internal")
	if err != nil {
		return fmt.Errorf("failed to read the internal track: %w", err)
	}
	if len(track.Releases) != 1 {
		return fmt.Errorf("internal track has %d releases; refusing automatic rewrite", len(track.Releases))
	}

	release := track.Releases[0]
	status := strings.ToLower(strings.TrimSpace(release.Status))
	if status == "draft" {
		return fmt.Errorf("internal track release is already draft")
	}
	if status != "completed" {
		return fmt.Errorf("internal track release status is %q, expected completed", release.Status)
	}
	if len(release.VersionCodes) == 0 {
		return fmt.Errorf("internal track release has no version codes")
	}

	_, err = fixer.UpdateTrack(ctx, packageName, editID, "internal", gpc.TrackUpdate{
		Status:         "draft",
		ReleaseName:    release.Name,
		UserFraction:   -1,
		VersionCodes:   append([]int64(nil), release.VersionCodes...),
		UpdatePriority: release.UpdatePriority,
		ReleaseNotes:   append([]gpc.LocalizedText(nil), release.ReleaseNotes...),
	})
	if err != nil {
		return fmt.Errorf("failed to rewrite the internal track release as draft: %w", err)
	}
	return nil
}

func explainDraftAppCommitError(err error) error {
	if err == nil {
		return nil
	}

	if isDraftAppTrackConflict(err) {
		return fmt.Errorf("%w\nhint: this draft app still has a track release with status \"completed\". Update the internal track release to \"draft\" in Play Console, then retry. Automatic track fixes are not applied by gpc.", err)
	}
	return err
}

func explainDraftAppTrackAutoFixError(commitErr, fixErr error) error {
	if commitErr == nil {
		return fixErr
	}
	return fmt.Errorf("%w\nhint: gpc tried to auto-fix the internal track by rewriting its completed release to draft, but that fix failed: %v", commitErr, fixErr)
}

func explainDraftAppTrackAutoFixRetryError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w\nhint: gpc auto-fixed the internal track to draft and retried the commit, but Play still rejected the edit.", err)
}
