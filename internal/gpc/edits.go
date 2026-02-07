package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) CreateEdit(ctx context.Context, packageName string) (EditInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return EditInfo{}, fmt.Errorf("package name is required")
	}
	if c == nil || c.service == nil {
		return EditInfo{}, ErrInvalidCredentials
	}

	edit, err := c.service.Edits.Insert(packageName, &androidpublisher.AppEdit{}).Context(ctx).Do()
	if err != nil {
		return EditInfo{}, mapGoogleAPIError(err)
	}

	return editInfoFromAppEdit(edit), nil
}

func (c *Client) GetEdit(ctx context.Context, packageName, editID string) (EditInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return EditInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return EditInfo{}, fmt.Errorf("edit id is required")
	}
	if c == nil || c.service == nil {
		return EditInfo{}, ErrInvalidCredentials
	}

	edit, err := c.service.Edits.Get(packageName, editID).Context(ctx).Do()
	if err != nil {
		return EditInfo{}, mapGoogleAPIError(err)
	}

	return editInfoFromAppEdit(edit), nil
}

func (c *Client) ValidateEdit(ctx context.Context, packageName, editID string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return fmt.Errorf("edit id is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if _, err := c.service.Edits.Validate(packageName, editID).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) CommitEdit(ctx context.Context, packageName, editID string) (EditInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return EditInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return EditInfo{}, fmt.Errorf("edit id is required")
	}
	if c == nil || c.service == nil {
		return EditInfo{}, ErrInvalidCredentials
	}

	edit, err := c.service.Edits.Commit(packageName, editID).Context(ctx).Do()
	if err != nil {
		return EditInfo{}, mapGoogleAPIError(err)
	}
	return editInfoFromAppEdit(edit), nil
}

func (c *Client) DeleteEdit(ctx context.Context, packageName, editID string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return fmt.Errorf("edit id is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Edits.Delete(packageName, editID).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) GetListing(ctx context.Context, packageName, editID, language string) (ListingInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return ListingInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return ListingInfo{}, fmt.Errorf("edit id is required")
	}
	language = strings.TrimSpace(language)
	if language == "" {
		return ListingInfo{}, fmt.Errorf("language is required")
	}
	if c == nil || c.service == nil {
		return ListingInfo{}, ErrInvalidCredentials
	}

	listing, err := c.service.Edits.Listings.Get(packageName, editID, language).Context(ctx).Do()
	if err != nil {
		return ListingInfo{}, mapGoogleAPIError(err)
	}
	return listingInfoFromListing(listing), nil
}

func (c *Client) UpdateListing(ctx context.Context, packageName, editID, language string, update ListingUpdate) (ListingInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return ListingInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return ListingInfo{}, fmt.Errorf("edit id is required")
	}
	language = strings.TrimSpace(language)
	if language == "" {
		return ListingInfo{}, fmt.Errorf("language is required")
	}
	update.Title = strings.TrimSpace(update.Title)
	update.ShortDescription = strings.TrimSpace(update.ShortDescription)
	update.FullDescription = strings.TrimSpace(update.FullDescription)
	if update.Title == "" && update.ShortDescription == "" && update.FullDescription == "" {
		return ListingInfo{}, fmt.Errorf("at least one listing field must be provided")
	}
	if c == nil || c.service == nil {
		return ListingInfo{}, ErrInvalidCredentials
	}

	req := &androidpublisher.Listing{
		Language:         language,
		Title:            update.Title,
		ShortDescription: update.ShortDescription,
		FullDescription:  update.FullDescription,
	}
	listing, err := c.service.Edits.Listings.Patch(packageName, editID, language, req).Context(ctx).Do()
	if err != nil {
		return ListingInfo{}, mapGoogleAPIError(err)
	}
	return listingInfoFromListing(listing), nil
}

func editInfoFromAppEdit(edit *androidpublisher.AppEdit) EditInfo {
	if edit == nil {
		return EditInfo{}
	}
	return EditInfo{
		ID:                edit.Id,
		ExpiryTimeSeconds: edit.ExpiryTimeSeconds,
	}
}

func listingInfoFromListing(listing *androidpublisher.Listing) ListingInfo {
	if listing == nil {
		return ListingInfo{}
	}
	return ListingInfo{
		Language:         listing.Language,
		Title:            listing.Title,
		ShortDescription: listing.ShortDescription,
		FullDescription:  listing.FullDescription,
	}
}
