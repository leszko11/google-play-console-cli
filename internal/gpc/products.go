package gpc

import (
	"context"
	"fmt"
	"slices"
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

	result := OneTimeProductsListInfo{Products: make([]OneTimeProductInfo, 0)}
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

func (c *Client) GetOneTimeProductDiagnostic(ctx context.Context, packageName, productID string) (OneTimeProductDiagnosticInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductDiagnosticInfo{}, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return OneTimeProductDiagnosticInfo{}, fmt.Errorf("product id is required")
	}
	if c == nil || c.service == nil {
		return OneTimeProductDiagnosticInfo{}, ErrInvalidCredentials
	}

	product, err := c.service.Monetization.Onetimeproducts.Get(packageName, productID).Context(ctx).Do()
	if err != nil {
		return OneTimeProductDiagnosticInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductDiagnosticInfoFromProduct(product), nil
}

func (c *Client) BatchGetOneTimeProducts(ctx context.Context, packageName string, productIDs []string) (OneTimeProductsListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductsListInfo{}, fmt.Errorf("package name is required")
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
		return OneTimeProductsListInfo{}, fmt.Errorf("at least one product id is required")
	}
	if len(filteredProductIDs) > 100 {
		return OneTimeProductsListInfo{}, fmt.Errorf("product id count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return OneTimeProductsListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Onetimeproducts.BatchGet(packageName).ProductIds(filteredProductIDs...).Context(ctx).Do()
	if err != nil {
		return OneTimeProductsListInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductsListInfoFromResponse(&androidpublisher.ListOneTimeProductsResponse{
		OneTimeProducts: resp.OneTimeProducts,
	}), nil
}

func (c *Client) BatchUpdateOneTimeProducts(ctx context.Context, packageName string, requests []*androidpublisher.UpdateOneTimeProductRequest) (OneTimeProductsListInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OneTimeProductsListInfo{}, fmt.Errorf("package name is required")
	}
	filteredRequests := make([]*androidpublisher.UpdateOneTimeProductRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return OneTimeProductsListInfo{}, fmt.Errorf("at least one batch update request is required")
	}
	if len(filteredRequests) > 100 {
		return OneTimeProductsListInfo{}, fmt.Errorf("batch update request count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return OneTimeProductsListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Onetimeproducts.BatchUpdate(packageName, &androidpublisher.BatchUpdateOneTimeProductsRequest{
		Requests: filteredRequests,
	}).Context(ctx).Do()
	if err != nil {
		return OneTimeProductsListInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductsListInfoFromResponse(&androidpublisher.ListOneTimeProductsResponse{
		OneTimeProducts: resp.OneTimeProducts,
	}), nil
}

func (c *Client) BatchDeleteOneTimeProducts(ctx context.Context, packageName string, requests []*androidpublisher.DeleteOneTimeProductRequest) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	filteredRequests := make([]*androidpublisher.DeleteOneTimeProductRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return fmt.Errorf("at least one batch delete request is required")
	}
	if len(filteredRequests) > 100 {
		return fmt.Errorf("batch delete request count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Monetization.Onetimeproducts.BatchDelete(packageName, &androidpublisher.BatchDeleteOneTimeProductsRequest{
		Requests: filteredRequests,
	}).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
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
	updateMask := oneTimeProductUpdateMask(product)
	if updateMask == "" {
		return OneTimeProductInfo{}, fmt.Errorf("one-time product payload must include at least one mutable field")
	}
	regionsVersion, err := c.resolveRegionsVersion(ctx, packageName, regionsVersionFromOneTimeProduct(product))
	if err != nil {
		return OneTimeProductInfo{}, err
	}

	created, err := c.service.Monetization.Onetimeproducts.Patch(packageName, productID, product).
		AllowMissing(true).
		UpdateMask(updateMask).
		RegionsVersionVersion(regionsVersion).
		Context(ctx).
		Do()
	if err != nil {
		return OneTimeProductInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductInfoFromProduct(created), nil
}

func oneTimeProductUpdateMask(product *androidpublisher.OneTimeProduct) string {
	if product == nil {
		return ""
	}
	paths := make([]string, 0, 5)
	if len(product.Listings) > 0 {
		paths = append(paths, "listings")
	}
	if len(product.PurchaseOptions) > 0 {
		paths = append(paths, "purchaseOptions")
	}
	if len(product.OfferTags) > 0 {
		paths = append(paths, "offerTags")
	}
	if product.RestrictedPaymentCountries != nil {
		paths = append(paths, "restrictedPaymentCountries")
	}
	if product.TaxAndComplianceSettings != nil {
		paths = append(paths, "taxAndComplianceSettings")
	}
	if len(paths) == 0 {
		return ""
	}
	slices.Sort(paths)
	return strings.Join(paths, ",")
}

func regionsVersionFromOneTimeProduct(product *androidpublisher.OneTimeProduct) string {
	if product == nil || product.RegionsVersion == nil {
		return ""
	}
	version := strings.TrimSpace(product.RegionsVersion.Version)
	return version
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
	regionsVersion, err := c.resolveRegionsVersion(ctx, packageName, regionsVersionFromOneTimeProduct(product))
	if err != nil {
		return OneTimeProductInfo{}, err
	}

	updated, err := c.service.Monetization.Onetimeproducts.Patch(packageName, productID, product).
		UpdateMask(updateMask).
		RegionsVersionVersion(regionsVersion).
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

	result := OneTimeProductOffersListInfo{Offers: make([]OneTimeProductOfferInfo, 0)}
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

func (c *Client) BatchGetOneTimeProductOffers(ctx context.Context, packageName, productID, purchaseOptionID string, offerIDs []string) (OneTimeProductOffersListInfo, error) {
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

	requests := make([]*androidpublisher.GetOneTimeProductOfferRequest, 0, len(offerIDs))
	for _, rawID := range offerIDs {
		offerID := strings.TrimSpace(rawID)
		if offerID == "" {
			continue
		}
		requests = append(requests, &androidpublisher.GetOneTimeProductOfferRequest{
			PackageName:      packageName,
			ProductId:        productID,
			PurchaseOptionId: purchaseOptionID,
			OfferId:          offerID,
		})
	}
	if len(requests) == 0 {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("at least one offer id is required")
	}
	if len(requests) > 100 {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("offer id count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return OneTimeProductOffersListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Onetimeproducts.PurchaseOptions.Offers.BatchGet(
		packageName,
		productID,
		purchaseOptionID,
		&androidpublisher.BatchGetOneTimeProductOffersRequest{
			Requests: requests,
		},
	).Context(ctx).Do()
	if err != nil {
		return OneTimeProductOffersListInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductOffersListInfoFromResponse(&androidpublisher.ListOneTimeProductOffersResponse{
		OneTimeProductOffers: resp.OneTimeProductOffers,
	}), nil
}

func (c *Client) BatchUpdateOneTimeProductOffers(ctx context.Context, packageName, productID, purchaseOptionID string, requests []*androidpublisher.UpdateOneTimeProductOfferRequest) (OneTimeProductOffersListInfo, error) {
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
	filteredRequests := make([]*androidpublisher.UpdateOneTimeProductOfferRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("at least one batch update request is required")
	}
	if len(filteredRequests) > 100 {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("batch update request count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return OneTimeProductOffersListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Onetimeproducts.PurchaseOptions.Offers.BatchUpdate(
		packageName,
		productID,
		purchaseOptionID,
		&androidpublisher.BatchUpdateOneTimeProductOffersRequest{
			Requests: filteredRequests,
		},
	).Context(ctx).Do()
	if err != nil {
		return OneTimeProductOffersListInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductOffersListInfoFromResponse(&androidpublisher.ListOneTimeProductOffersResponse{
		OneTimeProductOffers: resp.OneTimeProductOffers,
	}), nil
}

func (c *Client) BatchUpdateOneTimeProductOfferStates(ctx context.Context, packageName, productID, purchaseOptionID string, requests []*androidpublisher.UpdateOneTimeProductOfferStateRequest) (OneTimeProductOffersListInfo, error) {
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
	filteredRequests := make([]*androidpublisher.UpdateOneTimeProductOfferStateRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("at least one batch state update request is required")
	}
	if len(filteredRequests) > 100 {
		return OneTimeProductOffersListInfo{}, fmt.Errorf("batch state update request count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return OneTimeProductOffersListInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Onetimeproducts.PurchaseOptions.Offers.BatchUpdateStates(
		packageName,
		productID,
		purchaseOptionID,
		&androidpublisher.BatchUpdateOneTimeProductOfferStatesRequest{
			Requests: filteredRequests,
		},
	).Context(ctx).Do()
	if err != nil {
		return OneTimeProductOffersListInfo{}, mapGoogleAPIError(err)
	}
	return oneTimeProductOffersListInfoFromResponse(&androidpublisher.ListOneTimeProductOffersResponse{
		OneTimeProductOffers: resp.OneTimeProductOffers,
	}), nil
}

func (c *Client) BatchDeleteOneTimeProductOffers(ctx context.Context, packageName, productID, purchaseOptionID string, requests []*androidpublisher.DeleteOneTimeProductOfferRequest) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}
	purchaseOptionID = strings.TrimSpace(purchaseOptionID)
	if purchaseOptionID == "" {
		return fmt.Errorf("purchase option id is required")
	}
	filteredRequests := make([]*androidpublisher.DeleteOneTimeProductOfferRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		filteredRequests = append(filteredRequests, request)
	}
	if len(filteredRequests) == 0 {
		return fmt.Errorf("at least one batch delete request is required")
	}
	if len(filteredRequests) > 100 {
		return fmt.Errorf("batch delete request count must be less than or equal to 100")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Monetization.Onetimeproducts.PurchaseOptions.Offers.BatchDelete(
		packageName,
		productID,
		purchaseOptionID,
		&androidpublisher.BatchDeleteOneTimeProductOffersRequest{
			Requests: filteredRequests,
		},
	).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
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

func (c *Client) ActivateOneTimeProductPurchaseOption(ctx context.Context, packageName, productID, purchaseOptionID string) ([]OneTimeProductInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("product id is required")
	}
	purchaseOptionID = strings.TrimSpace(purchaseOptionID)
	if purchaseOptionID == "" {
		return nil, fmt.Errorf("purchase option id is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Onetimeproducts.PurchaseOptions.BatchUpdateStates(
		packageName,
		productID,
		&androidpublisher.BatchUpdatePurchaseOptionStatesRequest{
			Requests: []*androidpublisher.UpdatePurchaseOptionStateRequest{
				{
					ActivatePurchaseOptionRequest: &androidpublisher.ActivatePurchaseOptionRequest{
						PackageName:      packageName,
						ProductId:        productID,
						PurchaseOptionId: purchaseOptionID,
					},
				},
			},
		},
	).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return oneTimeProductInfosFromSlice(resp.OneTimeProducts), nil
}

func (c *Client) DeactivateOneTimeProductPurchaseOption(ctx context.Context, packageName, productID, purchaseOptionID string) ([]OneTimeProductInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("product id is required")
	}
	purchaseOptionID = strings.TrimSpace(purchaseOptionID)
	if purchaseOptionID == "" {
		return nil, fmt.Errorf("purchase option id is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.Onetimeproducts.PurchaseOptions.BatchUpdateStates(
		packageName,
		productID,
		&androidpublisher.BatchUpdatePurchaseOptionStatesRequest{
			Requests: []*androidpublisher.UpdatePurchaseOptionStateRequest{
				{
					DeactivatePurchaseOptionRequest: &androidpublisher.DeactivatePurchaseOptionRequest{
						PackageName:      packageName,
						ProductId:        productID,
						PurchaseOptionId: purchaseOptionID,
					},
				},
			},
		},
	).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return oneTimeProductInfosFromSlice(resp.OneTimeProducts), nil
}

func (c *Client) DeleteOneTimeProductPurchaseOption(ctx context.Context, packageName, productID, purchaseOptionID string, force bool) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}
	purchaseOptionID = strings.TrimSpace(purchaseOptionID)
	if purchaseOptionID == "" {
		return fmt.Errorf("purchase option id is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	req := &androidpublisher.BatchDeletePurchaseOptionsRequest{
		Requests: []*androidpublisher.DeletePurchaseOptionRequest{
			{
				PackageName:      packageName,
				ProductId:        productID,
				PurchaseOptionId: purchaseOptionID,
				Force:            force,
			},
		},
	}
	if err := c.service.Monetization.Onetimeproducts.PurchaseOptions.BatchDelete(packageName, productID, req).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
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

func oneTimeProductDiagnosticInfoFromProduct(product *androidpublisher.OneTimeProduct) OneTimeProductDiagnosticInfo {
	if product == nil {
		return OneTimeProductDiagnosticInfo{}
	}

	purchaseOptions := make([]OneTimeProductPurchaseOptionDiagnosticInfo, 0, len(product.PurchaseOptions))
	regionSet := map[string]struct{}{}
	availableRegionSet := map[string]struct{}{}
	activePurchaseOptionCount := 0

	for _, option := range product.PurchaseOptions {
		if option == nil {
			continue
		}

		if option.State == "ACTIVE" {
			activePurchaseOptionCount++
		}

		availableRegionCount := 0
		for _, cfg := range option.RegionalPricingAndAvailabilityConfigs {
			if cfg == nil {
				continue
			}
			regionCode := strings.TrimSpace(cfg.RegionCode)
			if regionCode != "" {
				regionSet[regionCode] = struct{}{}
			}
			if regionCode != "" && (cfg.Availability == "AVAILABLE" || cfg.Availability == "AVAILABLE_IF_RELEASED" || cfg.Availability == "AVAILABLE_FOR_OFFERS_ONLY") {
				availableRegionSet[regionCode] = struct{}{}
				availableRegionCount++
			}
		}

		purchaseOptions = append(purchaseOptions, OneTimeProductPurchaseOptionDiagnosticInfo{
			PurchaseOptionID:     option.PurchaseOptionId,
			State:                option.State,
			RegionalConfigCount:  len(option.RegionalPricingAndAvailabilityConfigs),
			AvailableRegionCount: availableRegionCount,
			OfferTagCount:        len(option.OfferTags),
		})
	}

	return OneTimeProductDiagnosticInfo{
		PackageName:               product.PackageName,
		ProductID:                 product.ProductId,
		ListingCount:              len(product.Listings),
		PurchaseOptionCount:       len(product.PurchaseOptions),
		OfferTagCount:             len(product.OfferTags),
		RegionCount:               len(regionSet),
		AvailableRegionCount:      len(availableRegionSet),
		ActivePurchaseOptionCount: activePurchaseOptionCount,
		PurchaseOptions:           purchaseOptions,
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

func oneTimeProductInfosFromSlice(products []*androidpublisher.OneTimeProduct) []OneTimeProductInfo {
	if len(products) == 0 {
		return nil
	}
	out := make([]OneTimeProductInfo, 0, len(products))
	for _, product := range products {
		out = append(out, oneTimeProductInfoFromProduct(product))
	}
	return out
}
