package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListReviews(ctx context.Context, packageName string, maxResults, startIndex int64, token, translationLanguage string, paginate bool) (ReviewsListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return ReviewsListInfo{}, fmt.Errorf("package name is required")
	}
	token = strings.TrimSpace(token)
	translationLanguage = strings.TrimSpace(translationLanguage)
	if c == nil || c.service == nil {
		return ReviewsListInfo{}, ErrInvalidCredentials
	}

	if !paginate {
		resp, err := c.reviewsListCall(ctx, packageName, maxResults, startIndex, token, translationLanguage).Do()
		if err != nil {
			return ReviewsListInfo{}, mapGoogleAPIError(err)
		}
		return reviewsListInfoFromResponse(resp), nil
	}

	result := ReviewsListInfo{}
	nextToken := token
	firstPage := true
	for {
		call := c.reviewsListCall(ctx, packageName, maxResults, startIndex, nextToken, translationLanguage)
		if !firstPage {
			call.StartIndex(0)
		}
		resp, err := call.Do()
		if err != nil {
			return ReviewsListInfo{}, mapGoogleAPIError(err)
		}
		page := reviewsListInfoFromResponse(resp)
		result.Reviews = append(result.Reviews, page.Reviews...)
		if page.NextToken == "" {
			result.NextToken = ""
			return result, nil
		}
		if page.NextToken == nextToken {
			return ReviewsListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextToken
		firstPage = false
	}
}

func (c *Client) GetReview(ctx context.Context, packageName, reviewID string) (ReviewInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return ReviewInfo{}, fmt.Errorf("package name is required")
	}
	reviewID = strings.TrimSpace(reviewID)
	if reviewID == "" {
		return ReviewInfo{}, fmt.Errorf("review id is required")
	}
	if c == nil || c.service == nil {
		return ReviewInfo{}, ErrInvalidCredentials
	}

	review, err := c.service.Reviews.Get(packageName, reviewID).Context(ctx).Do()
	if err != nil {
		return ReviewInfo{}, mapGoogleAPIError(err)
	}
	return reviewInfoFromReview(review), nil
}

func (c *Client) ReplyReview(ctx context.Context, packageName, reviewID, replyText string) (ReviewReplyInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return ReviewReplyInfo{}, fmt.Errorf("package name is required")
	}
	reviewID = strings.TrimSpace(reviewID)
	if reviewID == "" {
		return ReviewReplyInfo{}, fmt.Errorf("review id is required")
	}
	replyText = strings.TrimSpace(replyText)
	if replyText == "" {
		return ReviewReplyInfo{}, fmt.Errorf("reply text is required")
	}
	if c == nil || c.service == nil {
		return ReviewReplyInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Reviews.Reply(packageName, reviewID, &androidpublisher.ReviewsReplyRequest{
		ReplyText: replyText,
	}).Context(ctx).Do()
	if err != nil {
		return ReviewReplyInfo{}, mapGoogleAPIError(err)
	}
	return reviewReplyInfoFromResponse(resp), nil
}

func (c *Client) reviewsListCall(ctx context.Context, packageName string, maxResults, startIndex int64, token, translationLanguage string) *androidpublisher.ReviewsListCall {
	call := c.service.Reviews.List(packageName).Context(ctx)
	if maxResults > 0 {
		call.MaxResults(maxResults)
	}
	if startIndex > 0 {
		call.StartIndex(startIndex)
	}
	if token != "" {
		call.Token(token)
	}
	if translationLanguage != "" {
		call.TranslationLanguage(translationLanguage)
	}
	return call
}

func reviewsListInfoFromResponse(resp *androidpublisher.ReviewsListResponse) ReviewsListInfo {
	if resp == nil {
		return ReviewsListInfo{}
	}
	result := ReviewsListInfo{
		Reviews: make([]ReviewInfo, 0, len(resp.Reviews)),
	}
	for _, review := range resp.Reviews {
		result.Reviews = append(result.Reviews, reviewInfoFromReview(review))
	}
	if resp.TokenPagination != nil {
		result.NextToken = resp.TokenPagination.NextPageToken
	}
	return result
}

func reviewInfoFromReview(review *androidpublisher.Review) ReviewInfo {
	if review == nil {
		return ReviewInfo{}
	}
	return ReviewInfo{
		ReviewID:   review.ReviewId,
		AuthorName: review.AuthorName,
	}
}

func reviewReplyInfoFromResponse(resp *androidpublisher.ReviewsReplyResponse) ReviewReplyInfo {
	if resp == nil || resp.Result == nil {
		return ReviewReplyInfo{}
	}
	info := ReviewReplyInfo{
		ReplyText: resp.Result.ReplyText,
	}
	if resp.Result.LastEdited != nil {
		info.LastEditedSeconds = resp.Result.LastEdited.Seconds
		info.LastEditedNanos = resp.Result.LastEdited.Nanos
	}
	return info
}
