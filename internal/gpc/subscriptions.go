package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListSubscriptions(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (SubscriptionsListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionsListInfo{}, fmt.Errorf("package name is required")
	}
	if pageSize < 0 {
		return SubscriptionsListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}
	pageToken = strings.TrimSpace(pageToken)
	if c == nil || c.service == nil {
		return SubscriptionsListInfo{}, ErrInvalidCredentials
	}

	if !paginate {
		resp, err := c.subscriptionsListCall(ctx, packageName, pageSize, pageToken).Do()
		if err != nil {
			return SubscriptionsListInfo{}, mapGoogleAPIError(err)
		}
		return subscriptionsListInfoFromResponse(resp), nil
	}

	result := SubscriptionsListInfo{}
	nextToken := pageToken
	for {
		resp, err := c.subscriptionsListCall(ctx, packageName, pageSize, nextToken).Do()
		if err != nil {
			return SubscriptionsListInfo{}, mapGoogleAPIError(err)
		}
		page := subscriptionsListInfoFromResponse(resp)
		result.Subscriptions = append(result.Subscriptions, page.Subscriptions...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return SubscriptionsListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *Client) GetSubscription(ctx context.Context, packageName, productID string) (SubscriptionInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return SubscriptionInfo{}, fmt.Errorf("product id is required")
	}
	if c == nil || c.service == nil {
		return SubscriptionInfo{}, ErrInvalidCredentials
	}

	subscription, err := c.service.Monetization.Subscriptions.Get(packageName, productID).Context(ctx).Do()
	if err != nil {
		return SubscriptionInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionInfoFromSubscription(subscription), nil
}

func (c *Client) subscriptionsListCall(ctx context.Context, packageName string, pageSize int64, pageToken string) *androidpublisher.MonetizationSubscriptionsListCall {
	call := c.service.Monetization.Subscriptions.List(packageName).Context(ctx)
	if pageSize > 0 {
		call.PageSize(pageSize)
	}
	if pageToken != "" {
		call.PageToken(pageToken)
	}
	return call
}

func subscriptionsListInfoFromResponse(resp *androidpublisher.ListSubscriptionsResponse) SubscriptionsListInfo {
	if resp == nil {
		return SubscriptionsListInfo{}
	}
	result := SubscriptionsListInfo{
		Subscriptions: make([]SubscriptionInfo, 0, len(resp.Subscriptions)),
		NextPageToken: resp.NextPageToken,
	}
	for _, subscription := range resp.Subscriptions {
		result.Subscriptions = append(result.Subscriptions, subscriptionInfoFromSubscription(subscription))
	}
	return result
}

func subscriptionInfoFromSubscription(subscription *androidpublisher.Subscription) SubscriptionInfo {
	if subscription == nil {
		return SubscriptionInfo{}
	}
	return SubscriptionInfo{
		PackageName:   subscription.PackageName,
		ProductID:     subscription.ProductId,
		Archived:      subscription.Archived,
		BasePlanCount: len(subscription.BasePlans),
		ListingCount:  len(subscription.Listings),
	}
}
