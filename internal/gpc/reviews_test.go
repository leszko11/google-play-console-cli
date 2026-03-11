package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestReviewMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.ListReviews(context.Background(), "com.example.app", 10, 0, "", "", false); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListReviews, got %v", err)
	}
	if _, err := c.GetReview(context.Background(), "com.example.app", "review-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetReview, got %v", err)
	}
	if _, err := c.ReplyReview(context.Background(), "com.example.app", "review-1", "Thanks!"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ReplyReview, got %v", err)
	}
}

func TestReviewMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListReviews(context.Background(), "", 10, 0, "", "", false); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListReviews error: %v", err)
	}
	if _, err := c.GetReview(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "review id is required") {
		t.Fatalf("unexpected GetReview error: %v", err)
	}
	if _, err := c.ReplyReview(context.Background(), "com.example.app", "review-1", ""); err == nil || !strings.Contains(err.Error(), "reply text is required") {
		t.Fatalf("unexpected ReplyReview error: %v", err)
	}
}

func TestReviewsListInfoFromResponse(t *testing.T) {
	got := reviewsListInfoFromResponse(&androidpublisher.ReviewsListResponse{
		Reviews: []*androidpublisher.Review{
			{
				ReviewId:   "review-1",
				AuthorName: "Alice",
				Comments:   []*androidpublisher.Comment{{UserComment: &androidpublisher.UserComment{StarRating: 1, Text: "bad"}}},
			},
			{
				ReviewId:   "review-2",
				AuthorName: "Bob",
				Comments: []*androidpublisher.Comment{{
					UserComment:      &androidpublisher.UserComment{StarRating: 5, Text: "great"},
					DeveloperComment: &androidpublisher.DeveloperComment{Text: "thanks"},
				}},
			},
		},
		TokenPagination: &androidpublisher.TokenPagination{NextPageToken: "next-token"},
	})
	if len(got.Reviews) != 2 || got.NextToken != "next-token" {
		t.Fatalf("unexpected reviews list map: %+v", got)
	}
}

func TestReviewInfoFromReview(t *testing.T) {
	got := reviewInfoFromReview(&androidpublisher.Review{
		ReviewId:   "review-1",
		AuthorName: "Alice",
		Comments: []*androidpublisher.Comment{{
			UserComment:      &androidpublisher.UserComment{StarRating: 2, Text: "needs work"},
			DeveloperComment: &androidpublisher.DeveloperComment{Text: "we are fixing it"},
		}},
	})
	if got.ReviewID != "review-1" || got.AuthorName != "Alice" || got.StarRating != 2 || got.Comment != "needs work" || !got.HasReply || got.ReplyText != "we are fixing it" {
		t.Fatalf("unexpected review map: %+v", got)
	}
}

func TestReviewReplyInfoFromResponse(t *testing.T) {
	got := reviewReplyInfoFromResponse(&androidpublisher.ReviewsReplyResponse{
		Result: &androidpublisher.ReviewReplyResult{
			ReplyText: "Thanks for feedback",
			LastEdited: &androidpublisher.Timestamp{
				Seconds: 123,
				Nanos:   456,
			},
		},
	})
	if got.ReplyText != "Thanks for feedback" || got.LastEditedSeconds != 123 || got.LastEditedNanos != 456 {
		t.Fatalf("unexpected review reply map: %+v", got)
	}
}
