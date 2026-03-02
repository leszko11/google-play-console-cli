package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) CreateGrant(ctx context.Context, parent string, grant *androidpublisher.Grant) (GrantInfo, error) {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		return GrantInfo{}, fmt.Errorf("parent is required")
	}
	if grant == nil {
		return GrantInfo{}, fmt.Errorf("grant payload is required")
	}
	if c == nil || c.service == nil {
		return GrantInfo{}, ErrInvalidCredentials
	}

	created, err := c.service.Grants.Create(parent, grant).Context(ctx).Do()
	if err != nil {
		return GrantInfo{}, mapGoogleAPIError(err)
	}
	return grantInfoFromGrant(created), nil
}

func (c *Client) UpdateGrant(ctx context.Context, name string, grant *androidpublisher.Grant, updateMask string) (GrantInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return GrantInfo{}, fmt.Errorf("name is required")
	}
	if grant == nil {
		return GrantInfo{}, fmt.Errorf("grant payload is required")
	}
	updateMask = strings.TrimSpace(updateMask)
	if c == nil || c.service == nil {
		return GrantInfo{}, ErrInvalidCredentials
	}

	call := c.service.Grants.Patch(name, grant).Context(ctx)
	if updateMask != "" {
		call.UpdateMask(updateMask)
	}
	updated, err := call.Do()
	if err != nil {
		return GrantInfo{}, mapGoogleAPIError(err)
	}
	return grantInfoFromGrant(updated), nil
}

func (c *Client) DeleteGrant(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Grants.Delete(name).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func grantInfoFromGrant(grant *androidpublisher.Grant) GrantInfo {
	if grant == nil {
		return GrantInfo{}
	}
	return GrantInfo{
		Name:                grant.Name,
		PackageName:         grant.PackageName,
		AppLevelPermissions: append([]string(nil), grant.AppLevelPermissions...),
		PermissionCount:     len(grant.AppLevelPermissions),
	}
}
