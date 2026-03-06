package gpc

import (
	"context"
	"fmt"
	"slices"
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

	result := SubscriptionsListInfo{Subscriptions: make([]SubscriptionInfo, 0)}
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

func (c *Client) BatchGetSubscriptions(ctx context.Context, packageName string, productIDs []string) (SubscriptionsListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionsListInfo{}, fmt.Errorf("package name is required")
	}
	filteredProductIDs := make([]string, 0, len(productIDs))
	for _, rawID := range productIDs {
		productID := strings.TrimSpace(rawID)
		if productID == "" {
			continue
		}
		filteredProductIDs = append(filteredProductIDs, productID)
	}
	if len(filteredProductIDs) == 0 {
		return SubscriptionsListInfo{}, fmt.Errorf("at least one product id is required")
	}
	if len(filteredProductIDs) > 100 {
		return SubscriptionsListInfo{}, fmt.Errorf("product id count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return SubscriptionsListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Subscriptions.BatchGet(packageName).ProductIds(filteredProductIDs...).Context(ctx).Do()
	if err != nil {
		return SubscriptionsListInfo{}, mapGoogleAPIError(err)
	}
	return SubscriptionsListInfo{
		Subscriptions: subscriptionInfosFromSlice(resp.Subscriptions),
	}, nil
}

func (c *Client) BatchUpdateSubscriptions(ctx context.Context, packageName string, requests []*androidpublisher.UpdateSubscriptionRequest) (SubscriptionsListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionsListInfo{}, fmt.Errorf("package name is required")
	}
	filteredRequests := make([]*androidpublisher.UpdateSubscriptionRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return SubscriptionsListInfo{}, fmt.Errorf("at least one batch update request is required")
	}
	if len(filteredRequests) > 100 {
		return SubscriptionsListInfo{}, fmt.Errorf("batch update request count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return SubscriptionsListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Subscriptions.BatchUpdate(packageName, &androidpublisher.BatchUpdateSubscriptionsRequest{
		Requests: filteredRequests,
	}).Context(ctx).Do()
	if err != nil {
		return SubscriptionsListInfo{}, mapGoogleAPIError(err)
	}
	return SubscriptionsListInfo{
		Subscriptions: subscriptionInfosFromSlice(resp.Subscriptions),
	}, nil
}

func (c *Client) CreateSubscription(ctx context.Context, packageName string, subscription *androidpublisher.Subscription) (SubscriptionInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionInfo{}, fmt.Errorf("package name is required")
	}
	if subscription == nil {
		return SubscriptionInfo{}, fmt.Errorf("subscription payload is required")
	}
	productID := strings.TrimSpace(subscription.ProductId)
	if productID == "" {
		return SubscriptionInfo{}, fmt.Errorf("subscription payload must include productId")
	}
	if c == nil || c.service == nil {
		return SubscriptionInfo{}, ErrInvalidCredentials
	}
	regionsVersion, err := c.resolveRegionsVersion(ctx, packageName, "")
	if err != nil {
		return SubscriptionInfo{}, err
	}

	created, err := c.service.Monetization.Subscriptions.Create(packageName, subscription).
		ProductId(productID).
		RegionsVersionVersion(regionsVersion).
		Context(ctx).
		Do()
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
	updateMask := subscriptionUpdateMask(subscription)
	if updateMask == "" {
		return SubscriptionInfo{}, fmt.Errorf("subscription payload must include at least one mutable field")
	}
	regionsVersion, err := c.resolveRegionsVersion(ctx, packageName, "")
	if err != nil {
		return SubscriptionInfo{}, err
	}

	updated, err := c.service.Monetization.Subscriptions.Patch(packageName, productID, subscription).
		UpdateMask(updateMask).
		RegionsVersionVersion(regionsVersion).
		Context(ctx).
		Do()
	if err != nil {
		return SubscriptionInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionInfoFromSubscription(updated), nil
}

func subscriptionUpdateMask(subscription *androidpublisher.Subscription) string {
	if subscription == nil {
		return ""
	}
	paths := make([]string, 0, 4)
	if len(subscription.BasePlans) > 0 {
		paths = append(paths, "basePlans")
	}
	if len(subscription.Listings) > 0 {
		paths = append(paths, "listings")
	}
	if subscription.RestrictedPaymentCountries != nil {
		paths = append(paths, "restrictedPaymentCountries")
	}
	if subscription.TaxAndComplianceSettings != nil {
		paths = append(paths, "taxAndComplianceSettings")
	}
	if len(paths) == 0 {
		return ""
	}
	slices.Sort(paths)
	return strings.Join(paths, ",")
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

	resp, err := c.service.Monetization.Subscriptions.BasePlans.Activate(
		packageName,
		productID,
		basePlanID,
		&androidpublisher.ActivateBasePlanRequest{},
	).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return []SubscriptionInfo{subscriptionInfoFromSubscription(resp)}, nil
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

	resp, err := c.service.Monetization.Subscriptions.BasePlans.Deactivate(
		packageName,
		productID,
		basePlanID,
		&androidpublisher.DeactivateBasePlanRequest{},
	).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return []SubscriptionInfo{subscriptionInfoFromSubscription(resp)}, nil
}

func (c *Client) BatchUpdateSubscriptionBasePlanStates(ctx context.Context, packageName, productID string, requests []*androidpublisher.UpdateBasePlanStateRequest) ([]SubscriptionInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("product id is required")
	}
	filteredRequests := make([]*androidpublisher.UpdateBasePlanStateRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return nil, fmt.Errorf("at least one base plan state update request is required")
	}
	if len(filteredRequests) > 100 {
		return nil, fmt.Errorf("base plan state update request count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Subscriptions.BasePlans.BatchUpdateStates(
		packageName,
		productID,
		&androidpublisher.BatchUpdateBasePlanStatesRequest{Requests: filteredRequests},
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

func (c *Client) MigrateSubscriptionBasePlanPrices(ctx context.Context, packageName, productID, basePlanID string, request *androidpublisher.MigrateBasePlanPricesRequest) error {
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
	if request == nil {
		return fmt.Errorf("migrate prices payload is required")
	}
	if len(request.RegionalPriceMigrations) == 0 {
		return fmt.Errorf("migrate prices payload must include at least one regional price migration")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	regionsVersion, err := c.resolveRegionsVersion(ctx, packageName, regionsVersionFromMigrateRequest(request))
	if err != nil {
		return err
	}
	normalized := normalizeMigrateBasePlanPricesRequest(packageName, productID, basePlanID, request, regionsVersion)
	if _, err := c.service.Monetization.Subscriptions.BasePlans.MigratePrices(packageName, productID, basePlanID, normalized).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) BatchMigrateSubscriptionBasePlanPrices(ctx context.Context, packageName, productID string, requests []*androidpublisher.MigrateBasePlanPricesRequest) (int, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return 0, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return 0, fmt.Errorf("product id is required")
	}
	filteredRequests := make([]*androidpublisher.MigrateBasePlanPricesRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return 0, fmt.Errorf("at least one base plan migration request is required")
	}
	if len(filteredRequests) > 100 {
		return 0, fmt.Errorf("base plan migration request count must be less than or equal to 100")
	}

	seenBasePlans := make(map[string]struct{}, len(filteredRequests))
	validatedRequests := make([]*androidpublisher.MigrateBasePlanPricesRequest, 0, len(filteredRequests))
	for _, request := range filteredRequests {
		basePlanID := strings.TrimSpace(request.BasePlanId)
		if basePlanID == "" {
			return 0, fmt.Errorf("base plan id is required in every migration request")
		}
		if _, exists := seenBasePlans[basePlanID]; exists {
			return 0, fmt.Errorf("duplicate base plan id in migration requests: %s", basePlanID)
		}
		seenBasePlans[basePlanID] = struct{}{}
		if len(request.RegionalPriceMigrations) == 0 {
			return 0, fmt.Errorf("every migration request must include at least one regional price migration")
		}
		validatedRequests = append(validatedRequests, request)
	}
	if c == nil || c.service == nil {
		return 0, ErrInvalidCredentials
	}

	normalizedRequests := make([]*androidpublisher.MigrateBasePlanPricesRequest, 0, len(validatedRequests))
	cachedDefaultRegionsVersion := ""
	for _, request := range validatedRequests {
		basePlanID := strings.TrimSpace(request.BasePlanId)
		fallbackRegionsVersion := regionsVersionFromMigrateRequest(request)
		if fallbackRegionsVersion == "" {
			if cachedDefaultRegionsVersion == "" {
				resolvedDefault, err := c.resolveRegionsVersion(ctx, packageName, "")
				if err != nil {
					return 0, err
				}
				cachedDefaultRegionsVersion = resolvedDefault
			}
			fallbackRegionsVersion = cachedDefaultRegionsVersion
		}
		regionsVersion, err := c.resolveRegionsVersion(ctx, packageName, fallbackRegionsVersion)
		if err != nil {
			return 0, err
		}
		normalizedRequests = append(normalizedRequests, normalizeMigrateBasePlanPricesRequest(packageName, productID, basePlanID, request, regionsVersion))
	}

	resp, err := c.service.Monetization.Subscriptions.BasePlans.BatchMigratePrices(packageName, productID, &androidpublisher.BatchMigrateBasePlanPricesRequest{
		Requests: normalizedRequests,
	}).Context(ctx).Do()
	if err != nil {
		return 0, mapGoogleAPIError(err)
	}
	if resp == nil {
		return 0, nil
	}
	return len(resp.Responses), nil
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

	result := SubscriptionOffersListInfo{Offers: make([]SubscriptionOfferInfo, 0)}
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

func (c *Client) BatchGetSubscriptionOffers(ctx context.Context, packageName, productID, basePlanID string, offerIDs []string) (SubscriptionOffersListInfo, error) {
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

	requests := make([]*androidpublisher.GetSubscriptionOfferRequest, 0, len(offerIDs))
	for _, rawID := range offerIDs {
		offerID := strings.TrimSpace(rawID)
		if offerID == "" {
			continue
		}
		requests = append(requests, &androidpublisher.GetSubscriptionOfferRequest{
			PackageName: packageName,
			ProductId:   productID,
			BasePlanId:  basePlanID,
			OfferId:     offerID,
		})
	}
	if len(requests) == 0 {
		return SubscriptionOffersListInfo{}, fmt.Errorf("at least one offer id is required")
	}
	if len(requests) > 100 {
		return SubscriptionOffersListInfo{}, fmt.Errorf("offer id count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return SubscriptionOffersListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Subscriptions.BasePlans.Offers.BatchGet(packageName, productID, basePlanID, &androidpublisher.BatchGetSubscriptionOffersRequest{
		Requests: requests,
	}).Context(ctx).Do()
	if err != nil {
		return SubscriptionOffersListInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionOffersListInfoFromResponse(&androidpublisher.ListSubscriptionOffersResponse{
		SubscriptionOffers: resp.SubscriptionOffers,
	}), nil
}

func (c *Client) BatchUpdateSubscriptionOffers(ctx context.Context, packageName, productID, basePlanID string, requests []*androidpublisher.UpdateSubscriptionOfferRequest) (SubscriptionOffersListInfo, error) {
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
	filteredRequests := make([]*androidpublisher.UpdateSubscriptionOfferRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return SubscriptionOffersListInfo{}, fmt.Errorf("at least one batch update request is required")
	}
	if len(filteredRequests) > 100 {
		return SubscriptionOffersListInfo{}, fmt.Errorf("batch update request count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return SubscriptionOffersListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Subscriptions.BasePlans.Offers.BatchUpdate(
		packageName,
		productID,
		basePlanID,
		&androidpublisher.BatchUpdateSubscriptionOffersRequest{
			Requests: filteredRequests,
		},
	).Context(ctx).Do()
	if err != nil {
		return SubscriptionOffersListInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionOffersListInfoFromResponse(&androidpublisher.ListSubscriptionOffersResponse{
		SubscriptionOffers: resp.SubscriptionOffers,
	}), nil
}

func (c *Client) BatchUpdateSubscriptionOfferStates(ctx context.Context, packageName, productID, basePlanID string, requests []*androidpublisher.UpdateSubscriptionOfferStateRequest) (SubscriptionOffersListInfo, error) {
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
	filteredRequests := make([]*androidpublisher.UpdateSubscriptionOfferStateRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return SubscriptionOffersListInfo{}, fmt.Errorf("at least one batch state update request is required")
	}
	if len(filteredRequests) > 100 {
		return SubscriptionOffersListInfo{}, fmt.Errorf("batch state update request count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return SubscriptionOffersListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Subscriptions.BasePlans.Offers.BatchUpdateStates(
		packageName,
		productID,
		basePlanID,
		&androidpublisher.BatchUpdateSubscriptionOfferStatesRequest{
			Requests: filteredRequests,
		},
	).Context(ctx).Do()
	if err != nil {
		return SubscriptionOffersListInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionOffersListInfoFromResponse(&androidpublisher.ListSubscriptionOffersResponse{
		SubscriptionOffers: resp.SubscriptionOffers,
	}), nil
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
	offerID := strings.TrimSpace(offer.OfferId)
	if offerID == "" {
		return SubscriptionOfferInfo{}, fmt.Errorf("subscription offer payload must include offerId")
	}
	regionsVersion, err := c.resolveRegionsVersion(ctx, packageName, "")
	if err != nil {
		return SubscriptionOfferInfo{}, err
	}

	created, err := c.service.Monetization.Subscriptions.BasePlans.Offers.Create(packageName, productID, basePlanID, offer).
		OfferId(offerID).
		RegionsVersionVersion(regionsVersion).
		Context(ctx).
		Do()
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
	regionsVersion, err := c.resolveRegionsVersion(ctx, packageName, "")
	if err != nil {
		return SubscriptionOfferInfo{}, err
	}

	updated, err := c.service.Monetization.Subscriptions.BasePlans.Offers.Patch(packageName, productID, basePlanID, offerID, offer).
		UpdateMask(updateMask).
		RegionsVersionVersion(regionsVersion).
		Context(ctx).
		Do()
	if err != nil {
		return SubscriptionOfferInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionOfferInfoFromOffer(updated), nil
}

func (c *Client) ActivateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string) (SubscriptionOfferInfo, error) {
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

	offer, err := c.service.Monetization.Subscriptions.BasePlans.Offers.Activate(
		packageName,
		productID,
		basePlanID,
		offerID,
		&androidpublisher.ActivateSubscriptionOfferRequest{},
	).Context(ctx).Do()
	if err != nil {
		return SubscriptionOfferInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionOfferInfoFromOffer(offer), nil
}

func (c *Client) DeactivateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string) (SubscriptionOfferInfo, error) {
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

	offer, err := c.service.Monetization.Subscriptions.BasePlans.Offers.Deactivate(
		packageName,
		productID,
		basePlanID,
		offerID,
		&androidpublisher.DeactivateSubscriptionOfferRequest{},
	).Context(ctx).Do()
	if err != nil {
		return SubscriptionOfferInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionOfferInfoFromOffer(offer), nil
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

func regionsVersionFromMigrateRequest(request *androidpublisher.MigrateBasePlanPricesRequest) string {
	if request == nil || request.RegionsVersion == nil {
		return ""
	}
	return strings.TrimSpace(request.RegionsVersion.Version)
}

func normalizeMigrateBasePlanPricesRequest(packageName, productID, basePlanID string, request *androidpublisher.MigrateBasePlanPricesRequest, regionsVersion string) *androidpublisher.MigrateBasePlanPricesRequest {
	return &androidpublisher.MigrateBasePlanPricesRequest{
		PackageName:             packageName,
		ProductId:               productID,
		BasePlanId:              basePlanID,
		LatencyTolerance:        strings.TrimSpace(request.LatencyTolerance),
		RegionalPriceMigrations: request.RegionalPriceMigrations,
		RegionsVersion:          &androidpublisher.RegionsVersion{Version: regionsVersion},
	}
}
