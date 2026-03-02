package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListIAPs(ctx context.Context, packageName string, maxResults int64, pageToken string, paginate bool) (IAPsListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return IAPsListInfo{}, fmt.Errorf("package name is required")
	}
	if maxResults < 0 {
		return IAPsListInfo{}, fmt.Errorf("max results must be greater than or equal to zero")
	}
	pageToken = strings.TrimSpace(pageToken)
	if c == nil || c.service == nil {
		return IAPsListInfo{}, ErrInvalidCredentials
	}

	if !paginate {
		resp, err := c.iapsListCall(ctx, packageName, maxResults, pageToken).Do()
		if err != nil {
			return IAPsListInfo{}, mapGoogleAPIError(err)
		}
		return iapsListInfoFromResponse(resp), nil
	}

	result := IAPsListInfo{}
	nextToken := pageToken
	for {
		resp, err := c.iapsListCall(ctx, packageName, maxResults, nextToken).Do()
		if err != nil {
			return IAPsListInfo{}, mapGoogleAPIError(err)
		}
		page := iapsListInfoFromResponse(resp)
		result.Products = append(result.Products, page.Products...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return IAPsListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *Client) GetIAP(ctx context.Context, packageName, sku string) (IAPInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return IAPInfo{}, fmt.Errorf("package name is required")
	}
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return IAPInfo{}, fmt.Errorf("sku is required")
	}
	if c == nil || c.service == nil {
		return IAPInfo{}, ErrInvalidCredentials
	}

	product, err := c.service.Inappproducts.Get(packageName, sku).Context(ctx).Do()
	if err != nil {
		return IAPInfo{}, mapGoogleAPIError(err)
	}
	return iapInfoFromProduct(product), nil
}

func (c *Client) CreateIAP(ctx context.Context, packageName string, product *androidpublisher.InAppProduct) (IAPInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return IAPInfo{}, fmt.Errorf("package name is required")
	}
	if product == nil {
		return IAPInfo{}, fmt.Errorf("in-app product payload is required")
	}
	if c == nil || c.service == nil {
		return IAPInfo{}, ErrInvalidCredentials
	}

	created, err := c.service.Inappproducts.Insert(packageName, product).Context(ctx).Do()
	if err != nil {
		return IAPInfo{}, mapGoogleAPIError(err)
	}
	return iapInfoFromProduct(created), nil
}

func (c *Client) UpdateIAP(ctx context.Context, packageName, sku string, product *androidpublisher.InAppProduct) (IAPInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return IAPInfo{}, fmt.Errorf("package name is required")
	}
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return IAPInfo{}, fmt.Errorf("sku is required")
	}
	if product == nil {
		return IAPInfo{}, fmt.Errorf("in-app product payload is required")
	}
	if c == nil || c.service == nil {
		return IAPInfo{}, ErrInvalidCredentials
	}

	updated, err := c.service.Inappproducts.Patch(packageName, sku, product).Context(ctx).Do()
	if err != nil {
		return IAPInfo{}, mapGoogleAPIError(err)
	}
	return iapInfoFromProduct(updated), nil
}

func (c *Client) DeleteIAP(ctx context.Context, packageName, sku string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return fmt.Errorf("sku is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Inappproducts.Delete(packageName, sku).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) iapsListCall(ctx context.Context, packageName string, maxResults int64, pageToken string) *androidpublisher.InappproductsListCall {
	call := c.service.Inappproducts.List(packageName).Context(ctx)
	if maxResults > 0 {
		call.MaxResults(maxResults)
	}
	if pageToken != "" {
		call.Token(pageToken)
	}
	return call
}

func iapsListInfoFromResponse(resp *androidpublisher.InappproductsListResponse) IAPsListInfo {
	if resp == nil {
		return IAPsListInfo{}
	}
	nextToken := ""
	if resp.TokenPagination != nil {
		nextToken = resp.TokenPagination.NextPageToken
	}
	result := IAPsListInfo{
		Products:      make([]IAPInfo, 0, len(resp.Inappproduct)),
		NextPageToken: nextToken,
	}
	for _, product := range resp.Inappproduct {
		result.Products = append(result.Products, iapInfoFromProduct(product))
	}
	return result
}

func iapInfoFromProduct(product *androidpublisher.InAppProduct) IAPInfo {
	if product == nil {
		return IAPInfo{}
	}
	return IAPInfo{
		PackageName:  product.PackageName,
		SKU:          product.Sku,
		Status:       product.Status,
		PurchaseType: product.PurchaseType,
		ListingCount: len(product.Listings),
		PriceCount:   len(product.Prices),
	}
}
