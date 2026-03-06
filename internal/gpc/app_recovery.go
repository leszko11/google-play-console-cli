package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListAppRecoveries(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.ListAppRecoveriesResponse, error) {
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

	resp, err := c.service.Apprecovery.List(packageName).VersionCode(versionCode).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return resp, nil
}

func (c *Client) CreateAppRecovery(ctx context.Context, packageName string, request *androidpublisher.CreateDraftAppRecoveryRequest) (*androidpublisher.AppRecoveryAction, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if request == nil {
		return nil, fmt.Errorf("app recovery payload is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Apprecovery.Create(packageName, request).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return resp, nil
}

func (c *Client) AddAppRecoveryTargeting(ctx context.Context, packageName string, appRecoveryID int64, request *androidpublisher.AddTargetingRequest) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	if appRecoveryID <= 0 {
		return fmt.Errorf("app recovery id must be greater than zero")
	}
	if request == nil {
		return fmt.Errorf("app recovery targeting payload is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if _, err := c.service.Apprecovery.AddTargeting(packageName, appRecoveryID, request).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) CancelAppRecovery(ctx context.Context, packageName string, appRecoveryID int64) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	if appRecoveryID <= 0 {
		return fmt.Errorf("app recovery id must be greater than zero")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if _, err := c.service.Apprecovery.Cancel(packageName, appRecoveryID, &androidpublisher.CancelAppRecoveryRequest{}).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) DeployAppRecovery(ctx context.Context, packageName string, appRecoveryID int64) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	if appRecoveryID <= 0 {
		return fmt.Errorf("app recovery id must be greater than zero")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if _, err := c.service.Apprecovery.Deploy(packageName, appRecoveryID, &androidpublisher.DeployAppRecoveryRequest{}).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}
