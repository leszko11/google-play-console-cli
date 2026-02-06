package gpc

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) VerifyPackageAccess(ctx context.Context, packageName string) error {
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}
	if strings.TrimSpace(packageName) == "" {
		return fmt.Errorf("package name is required")
	}

	_, err := c.service.Reviews.List(packageName).MaxResults(1).Context(ctx).Do()
	if err != nil {
		return mapGoogleAPIError(err)
	}

	return nil
}

func (c *Client) GetApp(ctx context.Context, packageName string) (AppInfo, error) {
	if err := c.VerifyPackageAccess(ctx, packageName); err != nil {
		return AppInfo{}, err
	}

	return AppInfo{
		PackageName: packageName,
	}, nil
}
