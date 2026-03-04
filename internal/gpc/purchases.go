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
