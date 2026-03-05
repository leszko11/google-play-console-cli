package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListDeviceTierConfigs(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (*androidpublisher.ListDeviceTierConfigsResponse, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if pageSize < 0 {
		return nil, fmt.Errorf("page size must be greater than or equal to zero")
	}
	pageToken = strings.TrimSpace(pageToken)
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	if !paginate {
		resp, err := c.deviceTierConfigsListCall(ctx, packageName, pageSize, pageToken).Do()
		if err != nil {
			return nil, mapGoogleAPIError(err)
		}
		return resp, nil
	}

	result := &androidpublisher.ListDeviceTierConfigsResponse{
		DeviceTierConfigs: make([]*androidpublisher.DeviceTierConfig, 0),
	}
	nextToken := pageToken
	for {
		resp, err := c.deviceTierConfigsListCall(ctx, packageName, pageSize, nextToken).Do()
		if err != nil {
			return nil, mapGoogleAPIError(err)
		}
		result.DeviceTierConfigs = append(result.DeviceTierConfigs, resp.DeviceTierConfigs...)
		if resp.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if resp.NextPageToken == nextToken {
			return nil, fmt.Errorf("pagination token did not advance")
		}
		nextToken = resp.NextPageToken
	}
}

func (c *Client) GetDeviceTierConfig(ctx context.Context, packageName string, deviceTierConfigID int64) (*androidpublisher.DeviceTierConfig, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if deviceTierConfigID <= 0 {
		return nil, fmt.Errorf("device tier config id must be greater than zero")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	config, err := c.service.Applications.DeviceTierConfigs.Get(packageName, deviceTierConfigID).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return config, nil
}

func (c *Client) CreateDeviceTierConfig(ctx context.Context, packageName string, config *androidpublisher.DeviceTierConfig, allowUnknownDevices bool) (*androidpublisher.DeviceTierConfig, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if config == nil {
		return nil, fmt.Errorf("device tier config payload is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	created, err := c.service.Applications.DeviceTierConfigs.Create(packageName, config).AllowUnknownDevices(allowUnknownDevices).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return created, nil
}

func (c *Client) deviceTierConfigsListCall(ctx context.Context, packageName string, pageSize int64, pageToken string) *androidpublisher.ApplicationsDeviceTierConfigsListCall {
	call := c.service.Applications.DeviceTierConfigs.List(packageName).Context(ctx)
	if pageSize != 0 {
		call.PageSize(pageSize)
	}
	if pageToken != "" {
		call.PageToken(pageToken)
	}
	return call
}
