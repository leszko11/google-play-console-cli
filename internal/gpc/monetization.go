package gpc

import (
	"context"
	"fmt"
	"slices"
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

func (c *Client) GetMonetizationRegions(ctx context.Context, packageName string) (MonetizationRegionsInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return MonetizationRegionsInfo{}, fmt.Errorf("package name is required")
	}
	if c == nil || c.service == nil {
		return MonetizationRegionsInfo{}, ErrInvalidCredentials
	}

	resp, err := c.service.Monetization.ConvertRegionPrices(packageName, &androidpublisher.ConvertRegionPricesRequest{
		Price: &androidpublisher.Money{
			CurrencyCode: "USD",
			Units:        1,
		},
	}).Context(ctx).Do()
	if err != nil {
		return MonetizationRegionsInfo{}, mapGoogleAPIError(err)
	}
	if resp == nil || resp.RegionVersion == nil {
		return MonetizationRegionsInfo{}, fmt.Errorf("failed to resolve monetization regions")
	}

	regions := make([]MonetizationRegionInfo, 0, len(resp.ConvertedRegionPrices))
	for regionCode, region := range resp.ConvertedRegionPrices {
		code := strings.TrimSpace(regionCode)
		if code == "" {
			code = strings.TrimSpace(region.RegionCode)
		}
		if code == "" || region.Price == nil {
			continue
		}
		regions = append(regions, MonetizationRegionInfo{
			RegionCode:   code,
			CurrencyCode: strings.TrimSpace(region.Price.CurrencyCode),
		})
	}
	slices.SortFunc(regions, func(a, b MonetizationRegionInfo) int {
		return strings.Compare(a.RegionCode, b.RegionCode)
	})

	return MonetizationRegionsInfo{
		RegionsVersion: strings.TrimSpace(resp.RegionVersion.Version),
		Regions:        regions,
	}, nil
}

func (c *Client) GetLatestRegionsVersion(ctx context.Context, packageName string) (string, error) {
	return c.resolveRegionsVersion(ctx, packageName, "")
}
