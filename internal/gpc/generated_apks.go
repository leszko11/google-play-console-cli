package gpc

import (
	"context"
	"fmt"
	"io"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListGeneratedAPKs(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.GeneratedApksListResponse, error) {
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

	resp, err := c.service.Generatedapks.List(packageName, versionCode).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return resp, nil
}

func (c *Client) DownloadGeneratedAPK(ctx context.Context, packageName string, versionCode int64, downloadID string) ([]byte, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if versionCode <= 0 {
		return nil, fmt.Errorf("version code must be greater than zero")
	}
	downloadID = strings.TrimSpace(downloadID)
	if downloadID == "" {
		return nil, fmt.Errorf("download id is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Generatedapks.Download(packageName, versionCode, downloadID).Context(ctx).Download()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read generated apk download: %w", err)
	}
	return raw, nil
}
