package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListOneTimeProducts(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (OneTimeProductsListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductsListInfo{}, fmt.Errorf("package name is required")
	}
	if pageSize < 0 {
		return OneTimeProductsListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}
	pageToken = strings.TrimSpace(pageToken)
	if c == nil || c.service == nil {
		return OneTimeProductsListInfo{}, ErrInvalidCredentials
	}

	if !paginate {
		resp, err := c.oneTimeProductsListCall(ctx, packageName, pageSize, pageToken).Do()
		if err != nil {
			return OneTimeProductsListInfo{}, mapGoogleAPIError(err)
		}
		return oneTimeProductsListInfoFromResponse(resp), nil
	}

	result := OneTimeProductsListInfo{}
	nextToken := pageToken
	for {
		resp, err := c.oneTimeProductsListCall(ctx, packageName, pageSize, nextToken).Do()
		if err != nil {
			return OneTimeProductsListInfo{}, mapGoogleAPIError(err)
		}
		page := oneTimeProductsListInfoFromResponse(resp)
		result.Products = append(result.Products, page.Products...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return OneTimeProductsListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *Client) GetOneTimeProduct(ctx context.Context, packageName, productID string) (OneTimeProductInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return OneTimeProductInfo{}, fmt.Errorf("product id is required")
	}
	if c == nil || c.service == nil {
		return OneTimeProductInfo{}, ErrInvalidCredentials
	}

	product, err := c.service.Monetization.Onetimeproducts.Get(packageName, productID).Context(ctx).Do()
	if err != nil {
		return OneTimeProductInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductInfoFromProduct(product), nil
}

func (c *Client) CreateOneTimeProduct(ctx context.Context, packageName string, product *androidpublisher.OneTimeProduct) (OneTimeProductInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductInfo{}, fmt.Errorf("package name is required")
	}
	if product == nil {
		return OneTimeProductInfo{}, fmt.Errorf("one-time product payload is required")
	}
	productID := strings.TrimSpace(product.ProductId)
	if productID == "" {
		return OneTimeProductInfo{}, fmt.Errorf("one-time product payload must include productId")
	}
	if c == nil || c.service == nil {
		return OneTimeProductInfo{}, ErrInvalidCredentials
	}

	created, err := c.service.Monetization.Onetimeproducts.Patch(packageName, productID, product).
		AllowMissing(true).
		Context(ctx).
		Do()
	if err != nil {
		return OneTimeProductInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductInfoFromProduct(created), nil
}

func (c *Client) UpdateOneTimeProduct(ctx context.Context, packageName, productID string, product *androidpublisher.OneTimeProduct, updateMask string) (OneTimeProductInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return OneTimeProductInfo{}, fmt.Errorf("product id is required")
	}
	if product == nil {
		return OneTimeProductInfo{}, fmt.Errorf("one-time product payload is required")
	}
	updateMask = strings.TrimSpace(updateMask)
	if updateMask == "" {
		return OneTimeProductInfo{}, fmt.Errorf("update mask is required")
	}
	if c == nil || c.service == nil {
		return OneTimeProductInfo{}, ErrInvalidCredentials
	}

	updated, err := c.service.Monetization.Onetimeproducts.Patch(packageName, productID, product).
		UpdateMask(updateMask).
		Context(ctx).
		Do()
	if err != nil {
		return OneTimeProductInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductInfoFromProduct(updated), nil
}

func (c *Client) DeleteOneTimeProduct(ctx context.Context, packageName, productID string) error {
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

	if err := c.service.Monetization.Onetimeproducts.Delete(packageName, productID).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) ListOneTimeProductOffers(ctx context.Context, packageName, productID, purchaseOptionID string, pageSize int64, pageToken string, paginate bool) (OneTimeProductOffersListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("product id is required")
	}
	purchaseOptionID = strings.TrimSpace(purchaseOptionID)
	if purchaseOptionID == "" {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("purchase option id is required")
	}
	if pageSize < 0 {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}
	pageToken = strings.TrimSpace(pageToken)
	if c == nil || c.service == nil {
		return OneTimeProductOffersListInfo{}, ErrInvalidCredentials
	}

	if !paginate {
		resp, err := c.oneTimeProductOffersListCall(ctx, packageName, productID, purchaseOptionID, pageSize, pageToken).Do()
		if err != nil {
			return OneTimeProductOffersListInfo{}, mapGoogleAPIError(err)
		}
		return oneTimeProductOffersListInfoFromResponse(resp), nil
	}

	result := OneTimeProductOffersListInfo{}
	nextToken := pageToken
	for {
		resp, err := c.oneTimeProductOffersListCall(ctx, packageName, productID, purchaseOptionID, pageSize, nextToken).Do()
		if err != nil {
			return OneTimeProductOffersListInfo{}, mapGoogleAPIError(err)
		}
		page := oneTimeProductOffersListInfoFromResponse(resp)
		result.Offers = append(result.Offers, page.Offers...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return OneTimeProductOffersListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *Client) ActivateOneTimeProductOffer(ctx context.Context, packageName, productID, purchaseOptionID, offerID string) (OneTimeProductOfferInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("product id is required")
	}
	purchaseOptionID = strings.TrimSpace(purchaseOptionID)
	if purchaseOptionID == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("purchase option id is required")
	}
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("offer id is required")
	}
	if c == nil || c.service == nil {
		return OneTimeProductOfferInfo{}, ErrInvalidCredentials
	}

	offer, err := c.service.Monetization.Onetimeproducts.PurchaseOptions.Offers.Activate(
		packageName,
		productID,
		purchaseOptionID,
		offerID,
		&androidpublisher.ActivateOneTimeProductOfferRequest{},
	).Context(ctx).Do()
	if err != nil {
		return OneTimeProductOfferInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductOfferInfoFromOffer(offer), nil
}

func (c *Client) DeactivateOneTimeProductOffer(ctx context.Context, packageName, productID, purchaseOptionID, offerID string) (OneTimeProductOfferInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("product id is required")
	}
	purchaseOptionID = strings.TrimSpace(purchaseOptionID)
	if purchaseOptionID == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("purchase option id is required")
	}
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("offer id is required")
	}
	if c == nil || c.service == nil {
		return OneTimeProductOfferInfo{}, ErrInvalidCredentials
	}

	offer, err := c.service.Monetization.Onetimeproducts.PurchaseOptions.Offers.Deactivate(
		packageName,
		productID,
		purchaseOptionID,
		offerID,
		&androidpublisher.DeactivateOneTimeProductOfferRequest{},
	).Context(ctx).Do()
	if err != nil {
		return OneTimeProductOfferInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductOfferInfoFromOffer(offer), nil
}

func (c *Client) CancelOneTimeProductOffer(ctx context.Context, packageName, productID, purchaseOptionID, offerID string) (OneTimeProductOfferInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("product id is required")
	}
	purchaseOptionID = strings.TrimSpace(purchaseOptionID)
	if purchaseOptionID == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("purchase option id is required")
	}
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return OneTimeProductOfferInfo{}, fmt.Errorf("offer id is required")
	}
	if c == nil || c.service == nil {
		return OneTimeProductOfferInfo{}, ErrInvalidCredentials
	}

	offer, err := c.service.Monetization.Onetimeproducts.PurchaseOptions.Offers.Cancel(
		packageName,
		productID,
		purchaseOptionID,
		offerID,
		&androidpublisher.CancelOneTimeProductOfferRequest{},
	).Context(ctx).Do()
	if err != nil {
		return OneTimeProductOfferInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductOfferInfoFromOffer(offer), nil
}

func (c *Client) oneTimeProductsListCall(ctx context.Context, packageName string, pageSize int64, pageToken string) *androidpublisher.MonetizationOnetimeproductsListCall {
	call := c.service.Monetization.Onetimeproducts.List(packageName).Context(ctx)
	if pageSize > 0 {
		call.PageSize(pageSize)
	}
	if pageToken != "" {
		call.PageToken(pageToken)
	}
	return call
}

func (c *Client) oneTimeProductOffersListCall(ctx context.Context, packageName, productID, purchaseOptionID string, pageSize int64, pageToken string) *androidpublisher.MonetizationOnetimeproductsPurchaseOptionsOffersListCall {
	call := c.service.Monetization.Onetimeproducts.PurchaseOptions.Offers.List(packageName, productID, purchaseOptionID).Context(ctx)
	if pageSize > 0 {
		call.PageSize(pageSize)
	}
	if pageToken != "" {
		call.PageToken(pageToken)
	}
	return call
}

func oneTimeProductsListInfoFromResponse(resp *androidpublisher.ListOneTimeProductsResponse) OneTimeProductsListInfo {
	if resp == nil {
		return OneTimeProductsListInfo{}
	}
	result := OneTimeProductsListInfo{
		Products:      make([]OneTimeProductInfo, 0, len(resp.OneTimeProducts)),
		NextPageToken: resp.NextPageToken,
	}
	for _, product := range resp.OneTimeProducts {
		result.Products = append(result.Products, oneTimeProductInfoFromProduct(product))
	}
	return result
}

func oneTimeProductInfoFromProduct(product *androidpublisher.OneTimeProduct) OneTimeProductInfo {
	if product == nil {
		return OneTimeProductInfo{}
	}
	return OneTimeProductInfo{
		PackageName:         product.PackageName,
		ProductID:           product.ProductId,
		ListingCount:        len(product.Listings),
		PurchaseOptionCount: len(product.PurchaseOptions),
		OfferTagCount:       len(product.OfferTags),
	}
}

func oneTimeProductOffersListInfoFromResponse(resp *androidpublisher.ListOneTimeProductOffersResponse) OneTimeProductOffersListInfo {
	if resp == nil {
		return OneTimeProductOffersListInfo{}
	}
	result := OneTimeProductOffersListInfo{
		Offers:        make([]OneTimeProductOfferInfo, 0, len(resp.OneTimeProductOffers)),
		NextPageToken: resp.NextPageToken,
	}
	for _, offer := range resp.OneTimeProductOffers {
		result.Offers = append(result.Offers, oneTimeProductOfferInfoFromOffer(offer))
	}
	return result
}

func oneTimeProductOfferInfoFromOffer(offer *androidpublisher.OneTimeProductOffer) OneTimeProductOfferInfo {
	if offer == nil {
		return OneTimeProductOfferInfo{}
	}
	return OneTimeProductOfferInfo{
		PackageName:         offer.PackageName,
		ProductID:           offer.ProductId,
		PurchaseOptionID:    offer.PurchaseOptionId,
		OfferID:             offer.OfferId,
		State:               offer.State,
		OfferTagCount:       len(offer.OfferTags),
		RegionalConfigCount: len(offer.RegionalPricingAndAvailabilityConfigs),
	}
}
