package gpc

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

const (
	CancellationTypeUserRequestedStopRenewals     = "USER_REQUESTED_STOP_RENEWALS"
	CancellationTypeDeveloperRequestedStopPayment = "DEVELOPER_REQUESTED_STOP_PAYMENTS"

	RevocationRefundTypeFull     = "full"
	RevocationRefundTypeProrated = "prorated"
)

var protoDurationPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?s$`)

func (c *Client) GetProductPurchase(ctx context.Context, packageName, productID, token string) (ProductPurchaseInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return ProductPurchaseInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return ProductPurchaseInfo{}, fmt.Errorf("product id is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ProductPurchaseInfo{}, fmt.Errorf("purchase token is required")
	}
	if c == nil || c.service == nil {
		return ProductPurchaseInfo{}, ErrInvalidCredentials
	}

	purchase, err := c.service.Purchases.Products.Get(packageName, productID, token).Context(ctx).Do()
	if err != nil {
		return ProductPurchaseInfo{}, mapGoogleAPIError(err)
	}
	return productPurchaseInfoFromProductPurchase(purchase), nil
}

func (c *Client) GetProductPurchaseV2(ctx context.Context, packageName, token string) (ProductPurchaseV2Info, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return ProductPurchaseV2Info{}, fmt.Errorf("package name is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ProductPurchaseV2Info{}, fmt.Errorf("purchase token is required")
	}
	if c == nil || c.service == nil {
		return ProductPurchaseV2Info{}, ErrInvalidCredentials
	}

	purchase, err := c.service.Purchases.Productsv2.Getproductpurchasev2(packageName, token).Context(ctx).Do()
	if err != nil {
		return ProductPurchaseV2Info{}, mapGoogleAPIError(err)
	}
	return productPurchaseV2InfoFromProductPurchaseV2(purchase), nil
}

func (c *Client) AcknowledgeProductPurchase(ctx context.Context, packageName, productID, token, developerPayload string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("purchase token is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	req := &androidpublisher.ProductPurchasesAcknowledgeRequest{
		DeveloperPayload: strings.TrimSpace(developerPayload),
	}
	if err := c.service.Purchases.Products.Acknowledge(packageName, productID, token, req).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) ConsumeProductPurchase(ctx context.Context, packageName, productID, token string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("purchase token is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Purchases.Products.Consume(packageName, productID, token).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) GetSubscriptionPurchase(ctx context.Context, packageName, token string) (SubscriptionPurchaseInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionPurchaseInfo{}, fmt.Errorf("package name is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return SubscriptionPurchaseInfo{}, fmt.Errorf("purchase token is required")
	}
	if c == nil || c.service == nil {
		return SubscriptionPurchaseInfo{}, ErrInvalidCredentials
	}

	purchase, err := c.service.Purchases.Subscriptionsv2.Get(packageName, token).Context(ctx).Do()
	if err != nil {
		return SubscriptionPurchaseInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionPurchaseInfoFromSubscriptionPurchaseV2(purchase), nil
}

func (c *Client) GetLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string) (SubscriptionPurchaseInfo, error) {
	packageName, subscriptionID, token, err := normalizeLegacySubscriptionTarget(packageName, subscriptionID, token)
	if err != nil {
		return SubscriptionPurchaseInfo{}, err
	}
	if c == nil || c.service == nil {
		return SubscriptionPurchaseInfo{}, ErrInvalidCredentials
	}

	purchase, err := c.service.Purchases.Subscriptions.Get(packageName, subscriptionID, token).Context(ctx).Do()
	if err != nil {
		return SubscriptionPurchaseInfo{}, mapGoogleAPIError(err)
	}
	return subscriptionPurchaseInfoFromLegacySubscriptionPurchase(purchase), nil
}

func (c *Client) AcknowledgeLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token, developerPayload string) error {
	packageName, subscriptionID, token, err := normalizeLegacySubscriptionTarget(packageName, subscriptionID, token)
	if err != nil {
		return err
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	req := &androidpublisher.SubscriptionPurchasesAcknowledgeRequest{
		DeveloperPayload: strings.TrimSpace(developerPayload),
	}
	if err := c.service.Purchases.Subscriptions.Acknowledge(packageName, subscriptionID, token, req).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) CancelSubscriptionPurchase(ctx context.Context, packageName, token, cancellationType string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("purchase token is required")
	}
	cancellationType = strings.TrimSpace(cancellationType)
	if cancellationType == "" {
		return fmt.Errorf("cancellation type is required")
	}
	if cancellationType != CancellationTypeUserRequestedStopRenewals && cancellationType != CancellationTypeDeveloperRequestedStopPayment {
		return fmt.Errorf("unsupported cancellation type: %s", cancellationType)
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	req := &androidpublisher.CancelSubscriptionPurchaseRequest{
		CancellationContext: &androidpublisher.CancellationContext{
			CancellationType: cancellationType,
		},
	}
	if _, err := c.service.Purchases.Subscriptionsv2.Cancel(packageName, token, req).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) CancelLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string) error {
	packageName, subscriptionID, token, err := normalizeLegacySubscriptionTarget(packageName, subscriptionID, token)
	if err != nil {
		return err
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Purchases.Subscriptions.Cancel(packageName, subscriptionID, token).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) RevokeSubscriptionPurchase(ctx context.Context, packageName, token, refundType string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("purchase token is required")
	}
	refundType = strings.TrimSpace(strings.ToLower(refundType))
	if refundType == "" {
		return fmt.Errorf("refund type is required")
	}
	if refundType != RevocationRefundTypeFull && refundType != RevocationRefundTypeProrated {
		return fmt.Errorf("unsupported refund type: %s", refundType)
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	revocationContext := &androidpublisher.RevocationContext{}
	if refundType == RevocationRefundTypeFull {
		revocationContext.FullRefund = &androidpublisher.RevocationContextFullRefund{}
	} else {
		revocationContext.ProratedRefund = &androidpublisher.RevocationContextProratedRefund{}
	}

	req := &androidpublisher.RevokeSubscriptionPurchaseRequest{
		RevocationContext: revocationContext,
	}
	if _, err := c.service.Purchases.Subscriptionsv2.Revoke(packageName, token, req).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) RefundLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string) error {
	packageName, subscriptionID, token, err := normalizeLegacySubscriptionTarget(packageName, subscriptionID, token)
	if err != nil {
		return err
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Purchases.Subscriptions.Refund(packageName, subscriptionID, token).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) RevokeLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string) error {
	packageName, subscriptionID, token, err := normalizeLegacySubscriptionTarget(packageName, subscriptionID, token)
	if err != nil {
		return err
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Purchases.Subscriptions.Revoke(packageName, subscriptionID, token).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) DeferSubscriptionPurchase(ctx context.Context, packageName, token, etag, deferDuration string, validateOnly bool) (SubscriptionDeferInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return SubscriptionDeferInfo{}, fmt.Errorf("package name is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return SubscriptionDeferInfo{}, fmt.Errorf("purchase token is required")
	}
	etag = strings.TrimSpace(etag)
	if etag == "" {
		return SubscriptionDeferInfo{}, fmt.Errorf("etag is required")
	}
	deferDuration = strings.TrimSpace(deferDuration)
	if deferDuration == "" {
		return SubscriptionDeferInfo{}, fmt.Errorf("defer duration is required")
	}
	if !protoDurationPattern.MatchString(deferDuration) {
		return SubscriptionDeferInfo{}, fmt.Errorf("invalid defer duration format: expected protobuf duration ending with 's' (for example 604800s)")
	}
	if c == nil || c.service == nil {
		return SubscriptionDeferInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Purchases.Subscriptionsv2.Defer(packageName, token, &androidpublisher.DeferSubscriptionPurchaseRequest{
		DeferralContext: &androidpublisher.DeferralContext{
			Etag:          etag,
			DeferDuration: deferDuration,
			ValidateOnly:  validateOnly,
		},
	}).Context(ctx).Do()
	if err != nil {
		return SubscriptionDeferInfo{}, mapGoogleAPIError(err)
	}

	return subscriptionDeferInfoFromResponse(resp), nil
}

func (c *Client) DeferLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string, expectedExpiryTimeMillis, desiredExpiryTimeMillis int64) (SubscriptionDeferInfo, error) {
	packageName, subscriptionID, token, err := normalizeLegacySubscriptionTarget(packageName, subscriptionID, token)
	if err != nil {
		return SubscriptionDeferInfo{}, err
	}
	if expectedExpiryTimeMillis <= 0 {
		return SubscriptionDeferInfo{}, fmt.Errorf("expected expiry time millis must be greater than zero")
	}
	if desiredExpiryTimeMillis <= 0 {
		return SubscriptionDeferInfo{}, fmt.Errorf("desired expiry time millis must be greater than zero")
	}
	if desiredExpiryTimeMillis <= expectedExpiryTimeMillis {
		return SubscriptionDeferInfo{}, fmt.Errorf("desired expiry time millis must be greater than expected expiry time millis")
	}
	if c == nil || c.service == nil {
		return SubscriptionDeferInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Purchases.Subscriptions.Defer(packageName, subscriptionID, token, &androidpublisher.SubscriptionPurchasesDeferRequest{
		DeferralInfo: &androidpublisher.SubscriptionDeferralInfo{
			ExpectedExpiryTimeMillis: expectedExpiryTimeMillis,
			DesiredExpiryTimeMillis:  desiredExpiryTimeMillis,
		},
	}).Context(ctx).Do()
	if err != nil {
		return SubscriptionDeferInfo{}, mapGoogleAPIError(err)
	}

	return subscriptionDeferInfoFromLegacyResponse(resp), nil
}

func (c *Client) ListVoidedPurchases(ctx context.Context, packageName string, query VoidedPurchasesQuery) (VoidedPurchasesListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return VoidedPurchasesListInfo{}, fmt.Errorf("package name is required")
	}
	if query.MaxResults < 0 {
		return VoidedPurchasesListInfo{}, fmt.Errorf("max results must be greater than or equal to zero")
	}
	if query.StartIndex < 0 {
		return VoidedPurchasesListInfo{}, fmt.Errorf("start index must be greater than or equal to zero")
	}
	if query.StartTime < 0 || query.EndTime < 0 {
		return VoidedPurchasesListInfo{}, fmt.Errorf("start/end time must be greater than or equal to zero")
	}
	if query.StartTime > 0 && query.EndTime > 0 && query.StartTime > query.EndTime {
		return VoidedPurchasesListInfo{}, fmt.Errorf("start time must be less than or equal to end time")
	}
	if query.Type < 0 || query.Type > 1 {
		return VoidedPurchasesListInfo{}, fmt.Errorf("type must be 0 or 1")
	}
	query.Token = strings.TrimSpace(query.Token)
	if c == nil || c.service == nil {
		return VoidedPurchasesListInfo{}, ErrInvalidCredentials
	}

	if !query.Paginate {
		resp, err := c.voidedPurchasesListCall(ctx, packageName, query).Do()
		if err != nil {
			return VoidedPurchasesListInfo{}, mapGoogleAPIError(err)
		}
		return voidedPurchasesListInfoFromResponse(resp), nil
	}

	result := VoidedPurchasesListInfo{VoidedPurchases: make([]VoidedPurchaseInfo, 0)}
	nextToken := query.Token
	for {
		query.Token = nextToken
		resp, err := c.voidedPurchasesListCall(ctx, packageName, query).Do()
		if err != nil {
			return VoidedPurchasesListInfo{}, mapGoogleAPIError(err)
		}
		page := voidedPurchasesListInfoFromResponse(resp)
		result.VoidedPurchases = append(result.VoidedPurchases, page.VoidedPurchases...)
		if page.NextToken == "" {
			result.NextToken = ""
			return result, nil
		}
		if page.NextToken == nextToken {
			return VoidedPurchasesListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextToken
	}
}

func (c *Client) voidedPurchasesListCall(ctx context.Context, packageName string, query VoidedPurchasesQuery) *androidpublisher.PurchasesVoidedpurchasesListCall {
	call := c.service.Purchases.Voidedpurchases.List(packageName).Context(ctx)
	if query.MaxResults > 0 {
		call.MaxResults(query.MaxResults)
	}
	if query.StartIndex > 0 {
		call.StartIndex(query.StartIndex)
	}
	if query.StartTime > 0 {
		call.StartTime(query.StartTime)
	}
	if query.EndTime > 0 {
		call.EndTime(query.EndTime)
	}
	if query.Token != "" {
		call.Token(query.Token)
	}
	if query.Type == 1 {
		call.Type(query.Type)
	}
	if query.IncludeQuantityBasedPartialRefund {
		call.IncludeQuantityBasedPartialRefund(true)
	}
	return call
}

func productPurchaseInfoFromProductPurchase(purchase *androidpublisher.ProductPurchase) ProductPurchaseInfo {
	if purchase == nil {
		return ProductPurchaseInfo{}
	}
	return ProductPurchaseInfo{
		OrderID:              purchase.OrderId,
		ProductID:            purchase.ProductId,
		PurchaseToken:        purchase.PurchaseToken,
		PurchaseState:        purchase.PurchaseState,
		AcknowledgementState: purchase.AcknowledgementState,
		ConsumptionState:     purchase.ConsumptionState,
		PurchaseTimeMillis:   purchase.PurchaseTimeMillis,
		RegionCode:           purchase.RegionCode,
	}
}

func productPurchaseV2InfoFromProductPurchaseV2(purchase *androidpublisher.ProductPurchaseV2) ProductPurchaseV2Info {
	if purchase == nil {
		return ProductPurchaseV2Info{}
	}
	state := ""
	if purchase.PurchaseStateContext != nil {
		state = purchase.PurchaseStateContext.PurchaseState
	}
	return ProductPurchaseV2Info{
		Kind:                   purchase.Kind,
		OrderID:                purchase.OrderId,
		AcknowledgementState:   purchase.AcknowledgementState,
		PurchaseState:          state,
		RegionCode:             purchase.RegionCode,
		PurchaseCompletionTime: purchase.PurchaseCompletionTime,
		LineItemCount:          len(purchase.ProductLineItem),
	}
}

func subscriptionPurchaseInfoFromSubscriptionPurchaseV2(purchase *androidpublisher.SubscriptionPurchaseV2) SubscriptionPurchaseInfo {
	if purchase == nil {
		return SubscriptionPurchaseInfo{}
	}
	return SubscriptionPurchaseInfo{
		Kind:                 purchase.Kind,
		LatestOrderID:        purchase.LatestOrderId,
		SubscriptionState:    purchase.SubscriptionState,
		AcknowledgementState: purchase.AcknowledgementState,
		RegionCode:           purchase.RegionCode,
		StartTime:            purchase.StartTime,
		LineItemCount:        len(purchase.LineItems),
	}
}

func subscriptionPurchaseInfoFromLegacySubscriptionPurchase(purchase *androidpublisher.SubscriptionPurchase) SubscriptionPurchaseInfo {
	if purchase == nil {
		return SubscriptionPurchaseInfo{}
	}

	var paymentState int64
	if purchase.PaymentState != nil {
		paymentState = *purchase.PaymentState
	}

	return SubscriptionPurchaseInfo{
		Kind:                 purchase.Kind,
		LatestOrderID:        purchase.OrderId,
		AcknowledgementState: legacySubscriptionAcknowledgementState(purchase.AcknowledgementState),
		RegionCode:           purchase.CountryCode,
		AutoRenewing:         purchase.AutoRenewing,
		ExpiryTimeMillis:     purchase.ExpiryTimeMillis,
		CancelReason:         purchase.CancelReason,
		PaymentState:         paymentState,
	}
}

func subscriptionDeferInfoFromResponse(resp *androidpublisher.DeferSubscriptionPurchaseResponse) SubscriptionDeferInfo {
	if resp == nil {
		return SubscriptionDeferInfo{}
	}
	result := SubscriptionDeferInfo{
		ItemExpiryTimeDetails: make([]SubscriptionItemExpiryInfo, 0, len(resp.ItemExpiryTimeDetails)),
	}
	for _, item := range resp.ItemExpiryTimeDetails {
		if item == nil {
			result.ItemExpiryTimeDetails = append(result.ItemExpiryTimeDetails, SubscriptionItemExpiryInfo{})
			continue
		}
		result.ItemExpiryTimeDetails = append(result.ItemExpiryTimeDetails, SubscriptionItemExpiryInfo{
			ProductID:  item.ProductId,
			ExpiryTime: item.ExpiryTime,
		})
	}
	return result
}

func subscriptionDeferInfoFromLegacyResponse(resp *androidpublisher.SubscriptionPurchasesDeferResponse) SubscriptionDeferInfo {
	if resp == nil {
		return SubscriptionDeferInfo{}
	}
	return SubscriptionDeferInfo{
		NewExpiryTimeMillis: resp.NewExpiryTimeMillis,
	}
}

func normalizeLegacySubscriptionTarget(packageName, subscriptionID, token string) (string, string, string, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return "", "", "", fmt.Errorf("package name is required")
	}
	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID == "" {
		return "", "", "", fmt.Errorf("subscription id is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", "", fmt.Errorf("purchase token is required")
	}
	return packageName, subscriptionID, token, nil
}

func legacySubscriptionAcknowledgementState(state int64) string {
	switch state {
	case 1:
		return "ACKNOWLEDGED"
	case 0:
		return "PENDING"
	default:
		return fmt.Sprintf("%d", state)
	}
}

func voidedPurchasesListInfoFromResponse(resp *androidpublisher.VoidedPurchasesListResponse) VoidedPurchasesListInfo {
	if resp == nil {
		return VoidedPurchasesListInfo{}
	}
	result := VoidedPurchasesListInfo{
		VoidedPurchases: make([]VoidedPurchaseInfo, 0, len(resp.VoidedPurchases)),
	}
	for _, purchase := range resp.VoidedPurchases {
		result.VoidedPurchases = append(result.VoidedPurchases, voidedPurchaseInfoFromVoidedPurchase(purchase))
	}
	if resp.TokenPagination != nil {
		result.NextToken = resp.TokenPagination.NextPageToken
	}
	return result
}

func voidedPurchaseInfoFromVoidedPurchase(purchase *androidpublisher.VoidedPurchase) VoidedPurchaseInfo {
	if purchase == nil {
		return VoidedPurchaseInfo{}
	}
	return VoidedPurchaseInfo{
		OrderID:            purchase.OrderId,
		PurchaseToken:      purchase.PurchaseToken,
		PurchaseTimeMillis: purchase.PurchaseTimeMillis,
		VoidedTimeMillis:   purchase.VoidedTimeMillis,
		VoidedReason:       purchase.VoidedReason,
		VoidedSource:       purchase.VoidedSource,
		VoidedQuantity:     purchase.VoidedQuantity,
	}
}
