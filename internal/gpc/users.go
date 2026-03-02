package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListUsers(ctx context.Context, developerID string, pageSize int64, pageToken string, paginate bool) (UsersListInfo, error) {
	parent, err := normalizeDeveloperParent(developerID)
	if err != nil {
		return UsersListInfo{}, err
	}
	if pageSize < 0 {
		return UsersListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}
	pageToken = strings.TrimSpace(pageToken)
	if c == nil || c.service == nil {
		return UsersListInfo{}, ErrInvalidCredentials
	}

	if !paginate {
		resp, err := c.usersListCall(ctx, parent, pageSize, pageToken).Do()
		if err != nil {
			return UsersListInfo{}, mapGoogleAPIError(err)
		}
		return usersListInfoFromResponse(resp), nil
	}

	result := UsersListInfo{}
	nextToken := pageToken
	for {
		resp, err := c.usersListCall(ctx, parent, pageSize, nextToken).Do()
		if err != nil {
			return UsersListInfo{}, mapGoogleAPIError(err)
		}
		page := usersListInfoFromResponse(resp)
		result.Users = append(result.Users, page.Users...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return UsersListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *Client) CreateUser(ctx context.Context, developerID string, user *androidpublisher.User) (UserInfo, error) {
	parent, err := normalizeDeveloperParent(developerID)
	if err != nil {
		return UserInfo{}, err
	}
	if user == nil {
		return UserInfo{}, fmt.Errorf("user payload is required")
	}
	if c == nil || c.service == nil {
		return UserInfo{}, ErrInvalidCredentials
	}

	created, err := c.service.Users.Create(parent, user).Context(ctx).Do()
	if err != nil {
		return UserInfo{}, mapGoogleAPIError(err)
	}
	return userInfoFromUser(created), nil
}

func (c *Client) UpdateUser(ctx context.Context, name string, user *androidpublisher.User, updateMask string) (UserInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return UserInfo{}, fmt.Errorf("name is required")
	}
	if user == nil {
		return UserInfo{}, fmt.Errorf("user payload is required")
	}
	updateMask = strings.TrimSpace(updateMask)
	if c == nil || c.service == nil {
		return UserInfo{}, ErrInvalidCredentials
	}

	call := c.service.Users.Patch(name, user).Context(ctx)
	if updateMask != "" {
		call.UpdateMask(updateMask)
	}
	updated, err := call.Do()
	if err != nil {
		return UserInfo{}, mapGoogleAPIError(err)
	}
	return userInfoFromUser(updated), nil
}

func (c *Client) DeleteUser(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Users.Delete(name).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func normalizeDeveloperParent(developerID string) (string, error) {
	developerID = strings.TrimSpace(developerID)
	if developerID == "" {
		return "", fmt.Errorf("developer id is required")
	}
	if strings.HasPrefix(developerID, "developers/") {
		return developerID, nil
	}
	return "developers/" + developerID, nil
}

func (c *Client) usersListCall(ctx context.Context, parent string, pageSize int64, pageToken string) *androidpublisher.UsersListCall {
	call := c.service.Users.List(parent).Context(ctx)
	if pageSize > 0 {
		call.PageSize(pageSize)
	}
	if pageToken != "" {
		call.PageToken(pageToken)
	}
	return call
}

func usersListInfoFromResponse(resp *androidpublisher.ListUsersResponse) UsersListInfo {
	if resp == nil {
		return UsersListInfo{}
	}
	result := UsersListInfo{
		Users:         make([]UserInfo, 0, len(resp.Users)),
		NextPageToken: resp.NextPageToken,
	}
	for _, user := range resp.Users {
		result.Users = append(result.Users, userInfoFromUser(user))
	}
	return result
}

func userInfoFromUser(user *androidpublisher.User) UserInfo {
	if user == nil {
		return UserInfo{}
	}
	return UserInfo{
		Name:                        user.Name,
		Email:                       user.Email,
		AccessState:                 user.AccessState,
		ExpirationTime:              user.ExpirationTime,
		Partial:                     user.Partial,
		DeveloperAccountPermissions: append([]string(nil), user.DeveloperAccountPermissions...),
		GrantCount:                  len(user.Grants),
	}
}
