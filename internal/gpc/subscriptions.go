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

func (c *Client) CreateSubscription(ctx context.Context, packageName string, subscription *androidpublisher.Subscription) (SubscriptionInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionInfo{}, fmt.Errorf("package name is required")
	}
	if subscription == nil {
		return SubscriptionInfo{}, fmt.Errorf("subscription payload is required")
	}
	if c == nil || c.service == nil {
		return SubscriptionInfo{}, ErrInvalidCredentials
	}

	created, err := c.service.Monetization.Subscriptions.Create(packageName, subscription).Context(ctx).Do()
	if err != nil {
		return SubscriptionInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionInfoFromSubscription(created), nil
}

func (c *Client) UpdateSubscription(ctx context.Context, packageName, productID string, subscription *androidpublisher.Subscription) (SubscriptionInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return SubscriptionInfo{}, fmt.Errorf("product id is required")
	}
	if subscription == nil {
		return SubscriptionInfo{}, fmt.Errorf("subscription payload is required")
	}
	if c == nil || c.service == nil {
		return SubscriptionInfo{}, ErrInvalidCredentials
	}

	updated, err := c.service.Monetization.Subscriptions.Patch(packageName, productID, subscription).Context(ctx).Do()
	if err != nil {
		return SubscriptionInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionInfoFromSubscription(updated), nil
}

func (c *Client) DeleteSubscription(ctx context.Context, packageName, productID string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Monetization.Subscriptions.Delete(packageName, productID).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) ArchiveSubscription(ctx context.Context, packageName, productID string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if _, err := c.service.Monetization.Subscriptions.Archive(packageName, productID, &androidpublisher.ArchiveSubscriptionRequest{}).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) ActivateSubscriptionBasePlan(ctx context.Context, packageName, productID, basePlanID string) ([]SubscriptionInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("product id is required")
	}
	basePlanID = strings.TrimSpace(basePlanID)
	if basePlanID == "" {
		return nil, fmt.Errorf("base plan id is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Subscriptions.BasePlans.BatchUpdateStates(
		packageName,
		productID,
		&androidpublisher.BatchUpdateBasePlanStatesRequest{
			Requests: []*androidpublisher.UpdateBasePlanStateRequest{
				{
					ActivateBasePlanRequest: &androidpublisher.ActivateBasePlanRequest{
						PackageName: packageName,
						ProductId:   productID,
						BasePlanId:  basePlanID,
					},
				},
			},
		},
	).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return subscriptionInfosFromSlice(resp.Subscriptions), nil
}

func (c *Client) DeactivateSubscriptionBasePlan(ctx context.Context, packageName, productID, basePlanID string) ([]SubscriptionInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("product id is required")
	}
	basePlanID = strings.TrimSpace(basePlanID)
	if basePlanID == "" {
		return nil, fmt.Errorf("base plan id is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Subscriptions.BasePlans.BatchUpdateStates(
		packageName,
		productID,
		&androidpublisher.BatchUpdateBasePlanStatesRequest{
			Requests: []*androidpublisher.UpdateBasePlanStateRequest{
				{
					DeactivateBasePlanRequest: &androidpublisher.DeactivateBasePlanRequest{
						PackageName: packageName,
						ProductId:   productID,
						BasePlanId:  basePlanID,
					},
				},
			},
		},
	).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return subscriptionInfosFromSlice(resp.Subscriptions), nil
}

func (c *Client) DeleteSubscriptionBasePlan(ctx context.Context, packageName, productID, basePlanID string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}
	basePlanID = strings.TrimSpace(basePlanID)
	if basePlanID == "" {
		return fmt.Errorf("base plan id is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Monetization.Subscriptions.BasePlans.Delete(packageName, productID, basePlanID).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) ListSubscriptionOffers(ctx context.Context, packageName, productID, basePlanID string, pageSize int64, pageToken string, paginate bool) (SubscriptionOffersListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionOffersListInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return SubscriptionOffersListInfo{}, fmt.Errorf("product id is required")
	}
	basePlanID = strings.TrimSpace(basePlanID)
	if basePlanID == "" {
		return SubscriptionOffersListInfo{}, fmt.Errorf("base plan id is required")
	}
	if pageSize < 0 {
		return SubscriptionOffersListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}
	pageToken = strings.TrimSpace(pageToken)
	if c == nil || c.service == nil {
		return SubscriptionOffersListInfo{}, ErrInvalidCredentials
	}

	if !paginate {
		resp, err := c.subscriptionOffersListCall(ctx, packageName, productID, basePlanID, pageSize, pageToken).Do()
		if err != nil {
			return SubscriptionOffersListInfo{}, mapGoogleAPIError(err)
		}
		return subscriptionOffersListInfoFromResponse(resp), nil
	}

	result := SubscriptionOffersListInfo{}
	nextToken := pageToken
	for {
		resp, err := c.subscriptionOffersListCall(ctx, packageName, productID, basePlanID, pageSize, nextToken).Do()
		if err != nil {
			return SubscriptionOffersListInfo{}, mapGoogleAPIError(err)
		}
		page := subscriptionOffersListInfoFromResponse(resp)
		result.Offers = append(result.Offers, page.Offers...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return SubscriptionOffersListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *Client) GetSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string) (SubscriptionOfferInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("product id is required")
	}
	basePlanID = strings.TrimSpace(basePlanID)
	if basePlanID == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("base plan id is required")
	}
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("offer id is required")
	}
	if c == nil || c.service == nil {
		return SubscriptionOfferInfo{}, ErrInvalidCredentials
	}

	offer, err := c.service.Monetization.Subscriptions.BasePlans.Offers.Get(packageName, productID, basePlanID, offerID).Context(ctx).Do()
	if err != nil {
		return SubscriptionOfferInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionOfferInfoFromOffer(offer), nil
}

func (c *Client) CreateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID string, offer *androidpublisher.SubscriptionOffer) (SubscriptionOfferInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("product id is required")
	}
	basePlanID = strings.TrimSpace(basePlanID)
	if basePlanID == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("base plan id is required")
	}
	if offer == nil {
		return SubscriptionOfferInfo{}, fmt.Errorf("subscription offer payload is required")
	}
	if c == nil || c.service == nil {
		return SubscriptionOfferInfo{}, ErrInvalidCredentials
	}

	created, err := c.service.Monetization.Subscriptions.BasePlans.Offers.Create(packageName, productID, basePlanID, offer).Context(ctx).Do()
	if err != nil {
		return SubscriptionOfferInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionOfferInfoFromOffer(created), nil
}

func (c *Client) UpdateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string, offer *androidpublisher.SubscriptionOffer, updateMask string) (SubscriptionOfferInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("product id is required")
	}
	basePlanID = strings.TrimSpace(basePlanID)
	if basePlanID == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("base plan id is required")
	}
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("offer id is required")
	}
	if offer == nil {
		return SubscriptionOfferInfo{}, fmt.Errorf("subscription offer payload is required")
	}
	updateMask = strings.TrimSpace(updateMask)
	if updateMask == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("update mask is required")
	}
	if c == nil || c.service == nil {
		return SubscriptionOfferInfo{}, ErrInvalidCredentials
	}

	updated, err := c.service.Monetization.Subscriptions.BasePlans.Offers.Patch(packageName, productID, basePlanID, offerID, offer).
		UpdateMask(updateMask).
		Context(ctx).
		Do()
	if err != nil {
		return SubscriptionOfferInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionOfferInfoFromOffer(updated), nil
}

func (c *Client) DeleteSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}
	basePlanID = strings.TrimSpace(basePlanID)
	if basePlanID == "" {
		return fmt.Errorf("base plan id is required")
	}
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return fmt.Errorf("offer id is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Monetization.Subscriptions.BasePlans.Offers.Delete(packageName, productID, basePlanID, offerID).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
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

func (c *Client) subscriptionOffersListCall(ctx context.Context, packageName, productID, basePlanID string, pageSize int64, pageToken string) *androidpublisher.MonetizationSubscriptionsBasePlansOffersListCall {
	call := c.service.Monetization.Subscriptions.BasePlans.Offers.List(packageName, productID, basePlanID).Context(ctx)
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

func subscriptionOffersListInfoFromResponse(resp *androidpublisher.ListSubscriptionOffersResponse) SubscriptionOffersListInfo {
	if resp == nil {
		return SubscriptionOffersListInfo{}
	}
	result := SubscriptionOffersListInfo{
		Offers:        make([]SubscriptionOfferInfo, 0, len(resp.SubscriptionOffers)),
		NextPageToken: resp.NextPageToken,
	}
	for _, offer := range resp.SubscriptionOffers {
		result.Offers = append(result.Offers, subscriptionOfferInfoFromOffer(offer))
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

func subscriptionOfferInfoFromOffer(offer *androidpublisher.SubscriptionOffer) SubscriptionOfferInfo {
	if offer == nil {
		return SubscriptionOfferInfo{}
	}
	return SubscriptionOfferInfo{
		PackageName: offer.PackageName,
		ProductID:   offer.ProductId,
		BasePlanID:  offer.BasePlanId,
		OfferID:     offer.OfferId,
		State:       offer.State,
		PhaseCount:  len(offer.Phases),
		TagCount:    len(offer.OfferTags),
	}
}

func subscriptionInfosFromSlice(subscriptions []*androidpublisher.Subscription) []SubscriptionInfo {
	if len(subscriptions) == 0 {
		return nil
	}
	out := make([]SubscriptionInfo, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		out = append(out, subscriptionInfoFromSubscription(subscription))
	}
	return out
}
