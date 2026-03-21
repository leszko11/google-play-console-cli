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

type EditCommitResult struct {
	Edit                               gpc.EditInfo
	ChangesNotSentForReview            bool
	RetriedWithChangesNotSentForReview bool
}

func CommitEditWithReviewFallback(ctx context.Context, client EditCommitClient, packageName, editID string, changesNotSentForReview bool) (EditCommitResult, error) {
	result := EditCommitResult{
		ChangesNotSentForReview: changesNotSentForReview,
	}

	edit, err := client.CommitEdit(ctx, packageName, editID, changesNotSentForReview)
	if err == nil {
		result.Edit = edit
		return result, nil
	}

	if changesNotSentForReview || !shouldRetryCommitWithChangesNotSentForReview(err) {
		return result, explainDraftAppCommitError(err)
	}

	edit, err = client.CommitEdit(ctx, packageName, editID, true)
	if err != nil {
		return EditCommitResult{
			ChangesNotSentForReview:            true,
			RetriedWithChangesNotSentForReview: true,
		}, explainDraftAppCommitError(err)
	}

	result.Edit = edit
	result.ChangesNotSentForReview = true
	result.RetriedWithChangesNotSentForReview = true
	return result, nil
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

func explainDraftAppCommitError(err error) error {
	if err == nil {
		return nil
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "only releases with status draft may be created on draft app") {
		return fmt.Errorf("%w\nhint: this draft app still has a track release with status \"completed\". Update the internal track release to \"draft\" in Play Console, or run `gpc release init --package-name <package> --dir ./play` to generate the bootstrap release workflow.", err)
	}
	return err
}
