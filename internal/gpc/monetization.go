package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

const defaultRegionsVersionVersion = "2022/02"

func (c *Client) resolveRegionsVersion(ctx context.Context, packageName, fallback string) (string, error) {
	if version := strings.TrimSpace(fallback); version != "" {
		return version, nil
	}
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return "", fmt.Errorf("package name is required")
	}
	if c == nil || c.service == nil {
		return "", ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.ConvertRegionPrices(packageName, &androidpublisher.ConvertRegionPricesRequest{
		Price: &androidpublisher.Money{
			CurrencyCode: "USD",
			Units:        1,
		},
	}).Context(ctx).Do()
	if err != nil {
		return "", mapGoogleAPIError(err)
	}
	if resp == nil || resp.RegionVersion == nil {
		return "", fmt.Errorf("failed to resolve regions version")
	}

	version := strings.TrimSpace(resp.RegionVersion.Version)
	if version == "" {
		return "", fmt.Errorf("failed to resolve regions version")
	}
	return version, nil
}
