package gpc

import (
	"context"
	"fmt"
	"io"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListSystemAPKVariants(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.SystemApksListResponse, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if versionCode <= 0 {
		return nil, fmt.Errorf("version code must be greater than zero")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Systemapks.Variants.List(packageName, versionCode).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return resp, nil
}

func (c *Client) GetSystemAPKVariant(ctx context.Context, packageName string, versionCode, variantID int64) (*androidpublisher.Variant, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if versionCode <= 0 {
		return nil, fmt.Errorf("version code must be greater than zero")
	}
	if variantID <= 0 {
		return nil, fmt.Errorf("variant id must be greater than zero")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	variant, err := c.service.Systemapks.Variants.Get(packageName, versionCode, variantID).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return variant, nil
}

func (c *Client) CreateSystemAPKVariant(ctx context.Context, packageName string, versionCode int64, variant *androidpublisher.Variant) (*androidpublisher.Variant, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if versionCode <= 0 {
		return nil, fmt.Errorf("version code must be greater than zero")
	}
	if variant == nil {
		return nil, fmt.Errorf("system apk variant payload is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	created, err := c.service.Systemapks.Variants.Create(packageName, versionCode, variant).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return created, nil
}

func (c *Client) DownloadSystemAPKVariant(ctx context.Context, packageName string, versionCode, variantID int64) ([]byte, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if versionCode <= 0 {
		return nil, fmt.Errorf("version code must be greater than zero")
	}
	if variantID <= 0 {
		return nil, fmt.Errorf("variant id must be greater than zero")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Systemapks.Variants.Download(packageName, versionCode, variantID).Context(ctx).Download()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read system apk download: %w", err)
	}
	return raw, nil
}
