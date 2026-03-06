package gpc

import (
	"context"
	"fmt"
	"os"
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

func (c *Client) ListListings(ctx context.Context, packageName, editID string) ([]ListingInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, fmt.Errorf("edit id is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Edits.Listings.List(packageName, editID).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}

	listings := make([]ListingInfo, 0, len(resp.Listings))
	for _, listing := range resp.Listings {
		listings = append(listings, listingInfoFromListing(listing))
	}
	return listings, nil
}

func (c *Client) UpdateListing(ctx context.Context, packageName, editID, language string, update ListingUpdate) (ListingInfo, error) {
	packageName, editID, language, req, err := normalizeListingUpdateRequest(packageName, editID, language, update)
	if err != nil {
		return ListingInfo{}, err
	}
	if c == nil || c.service == nil {
		return ListingInfo{}, ErrInvalidCredentials
	}

	listing, err := c.service.Edits.Listings.Patch(packageName, editID, language, req).Context(ctx).Do()
	if err != nil {
		return ListingInfo{}, mapGoogleAPIError(err)
	}
	return listingInfoFromListing(listing), nil
}

func (c *Client) ReplaceListing(ctx context.Context, packageName, editID, language string, update ListingUpdate) (ListingInfo, error) {
	packageName, editID, language, req, err := normalizeListingUpdateRequest(packageName, editID, language, update)
	if err != nil {
		return ListingInfo{}, err
	}
	if c == nil || c.service == nil {
		return ListingInfo{}, ErrInvalidCredentials
	}

	listing, err := c.service.Edits.Listings.Update(packageName, editID, language, req).Context(ctx).Do()
	if err != nil {
		return ListingInfo{}, mapGoogleAPIError(err)
	}
	return listingInfoFromListing(listing), nil
}

func (c *Client) GetAppDetails(ctx context.Context, packageName, editID string) (AppDetailsInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return AppDetailsInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return AppDetailsInfo{}, fmt.Errorf("edit id is required")
	}
	if c == nil || c.service == nil {
		return AppDetailsInfo{}, ErrInvalidCredentials
	}

	details, err := c.service.Edits.Details.Get(packageName, editID).Context(ctx).Do()
	if err != nil {
		return AppDetailsInfo{}, mapGoogleAPIError(err)
	}
	return appDetailsInfoFromDetails(details), nil
}

func (c *Client) UpdateAppDetails(ctx context.Context, packageName, editID string, update AppDetailsUpdate) (AppDetailsInfo, error) {
	packageName, editID, req, err := normalizeAppDetailsUpdateRequest(packageName, editID, update)
	if err != nil {
		return AppDetailsInfo{}, err
	}
	if c == nil || c.service == nil {
		return AppDetailsInfo{}, ErrInvalidCredentials
	}

	details, err := c.service.Edits.Details.Patch(packageName, editID, req).Context(ctx).Do()
	if err != nil {
		return AppDetailsInfo{}, mapGoogleAPIError(err)
	}
	return appDetailsInfoFromDetails(details), nil
}

func (c *Client) ReplaceAppDetails(ctx context.Context, packageName, editID string, update AppDetailsUpdate) (AppDetailsInfo, error) {
	packageName, editID, req, err := normalizeAppDetailsUpdateRequest(packageName, editID, update)
	if err != nil {
		return AppDetailsInfo{}, err
	}
	if c == nil || c.service == nil {
		return AppDetailsInfo{}, ErrInvalidCredentials
	}

	details, err := c.service.Edits.Details.Update(packageName, editID, req).Context(ctx).Do()
	if err != nil {
		return AppDetailsInfo{}, mapGoogleAPIError(err)
	}
	return appDetailsInfoFromDetails(details), nil
}

func (c *Client) GetTesters(ctx context.Context, packageName, editID, track string) (TestersInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return TestersInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return TestersInfo{}, fmt.Errorf("edit id is required")
	}
	track = strings.TrimSpace(track)
	if track == "" {
		return TestersInfo{}, fmt.Errorf("track is required")
	}
	if c == nil || c.service == nil {
		return TestersInfo{}, ErrInvalidCredentials
	}

	testers, err := c.service.Edits.Testers.Get(packageName, editID, track).Context(ctx).Do()
	if err != nil {
		return TestersInfo{}, mapGoogleAPIError(err)
	}
	return testersInfoFromTesters(track, testers), nil
}

func (c *Client) UpdateTesters(ctx context.Context, packageName, editID, track string, googleGroups []string) (TestersInfo, error) {
	packageName, editID, track, req, err := normalizeTestersUpdateRequest(packageName, editID, track, googleGroups)
	if err != nil {
		return TestersInfo{}, err
	}
	if c == nil || c.service == nil {
		return TestersInfo{}, ErrInvalidCredentials
	}

	testers, err := c.service.Edits.Testers.Patch(packageName, editID, track, req).Context(ctx).Do()
	if err != nil {
		return TestersInfo{}, mapGoogleAPIError(err)
	}
	return testersInfoFromTesters(track, testers), nil
}

func (c *Client) ReplaceTesters(ctx context.Context, packageName, editID, track string, googleGroups []string) (TestersInfo, error) {
	packageName, editID, track, req, err := normalizeTestersUpdateRequest(packageName, editID, track, googleGroups)
	if err != nil {
		return TestersInfo{}, err
	}
	if c == nil || c.service == nil {
		return TestersInfo{}, ErrInvalidCredentials
	}

	testers, err := c.service.Edits.Testers.Update(packageName, editID, track, req).Context(ctx).Do()
	if err != nil {
		return TestersInfo{}, mapGoogleAPIError(err)
	}
	return testersInfoFromTesters(track, testers), nil
}

func normalizeListingUpdateRequest(packageName, editID, language string, update ListingUpdate) (string, string, string, *androidpublisher.Listing, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return "", "", "", nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return "", "", "", nil, fmt.Errorf("edit id is required")
	}
	language = strings.TrimSpace(language)
	if language == "" {
		return "", "", "", nil, fmt.Errorf("language is required")
	}
	update.Title = strings.TrimSpace(update.Title)
	update.ShortDescription = strings.TrimSpace(update.ShortDescription)
	update.FullDescription = strings.TrimSpace(update.FullDescription)
	if update.Title == "" && update.ShortDescription == "" && update.FullDescription == "" {
		return "", "", "", nil, fmt.Errorf("at least one listing field must be provided")
	}

	return packageName, editID, language, &androidpublisher.Listing{
		Language:         language,
		Title:            update.Title,
		ShortDescription: update.ShortDescription,
		FullDescription:  update.FullDescription,
	}, nil
}

func normalizeAppDetailsUpdateRequest(packageName, editID string, update AppDetailsUpdate) (string, string, *androidpublisher.AppDetails, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return "", "", nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return "", "", nil, fmt.Errorf("edit id is required")
	}
	update.DefaultLanguage = strings.TrimSpace(update.DefaultLanguage)
	update.ContactEmail = strings.TrimSpace(update.ContactEmail)
	update.ContactPhone = strings.TrimSpace(update.ContactPhone)
	update.ContactWebsite = strings.TrimSpace(update.ContactWebsite)
	if update.DefaultLanguage == "" && update.ContactEmail == "" && update.ContactPhone == "" && update.ContactWebsite == "" {
		return "", "", nil, fmt.Errorf("at least one app detail field must be provided")
	}

	return packageName, editID, &androidpublisher.AppDetails{
		DefaultLanguage: update.DefaultLanguage,
		ContactEmail:    update.ContactEmail,
		ContactPhone:    update.ContactPhone,
		ContactWebsite:  update.ContactWebsite,
	}, nil
}

func normalizeTestersUpdateRequest(packageName, editID, track string, googleGroups []string) (string, string, string, *androidpublisher.Testers, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return "", "", "", nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return "", "", "", nil, fmt.Errorf("edit id is required")
	}
	track = strings.TrimSpace(track)
	if track == "" {
		return "", "", "", nil, fmt.Errorf("track is required")
	}

	sanitizedGroups := make([]string, 0, len(googleGroups))
	for _, group := range googleGroups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		sanitizedGroups = append(sanitizedGroups, group)
	}
	if len(sanitizedGroups) == 0 {
		return "", "", "", nil, fmt.Errorf("at least one google group is required")
	}

	return packageName, editID, track, &androidpublisher.Testers{GoogleGroups: sanitizedGroups}, nil
}

func (c *Client) GetCountryAvailability(ctx context.Context, packageName, editID, track string) (CountryAvailabilityInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return CountryAvailabilityInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return CountryAvailabilityInfo{}, fmt.Errorf("edit id is required")
	}
	track = strings.TrimSpace(track)
	if track == "" {
		return CountryAvailabilityInfo{}, fmt.Errorf("track is required")
	}
	if c == nil || c.service == nil {
		return CountryAvailabilityInfo{}, ErrInvalidCredentials
	}

	availability, err := c.service.Edits.Countryavailability.Get(packageName, editID, track).Context(ctx).Do()
	if err != nil {
		return CountryAvailabilityInfo{}, mapGoogleAPIError(err)
	}
	return countryAvailabilityInfoFromTrackCountryAvailability(track, availability), nil
}

func (c *Client) ListImages(ctx context.Context, packageName, editID, language, imageType string) ([]ImageInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, fmt.Errorf("edit id is required")
	}
	language = strings.TrimSpace(language)
	if language == "" {
		return nil, fmt.Errorf("language is required")
	}
	imageType = strings.TrimSpace(imageType)
	if imageType == "" {
		return nil, fmt.Errorf("image type is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Edits.Images.List(packageName, editID, language, imageType).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	images := make([]ImageInfo, 0, len(resp.Images))
	for _, image := range resp.Images {
		images = append(images, imageInfoFromImage(image))
	}
	return images, nil
}

func (c *Client) UploadImage(ctx context.Context, packageName, editID, language, imageType, imagePath string) (ImageInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return ImageInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return ImageInfo{}, fmt.Errorf("edit id is required")
	}
	language = strings.TrimSpace(language)
	if language == "" {
		return ImageInfo{}, fmt.Errorf("language is required")
	}
	imageType = strings.TrimSpace(imageType)
	if imageType == "" {
		return ImageInfo{}, fmt.Errorf("image type is required")
	}
	imagePath = strings.TrimSpace(imagePath)
	if imagePath == "" {
		return ImageInfo{}, fmt.Errorf("image path is required")
	}
	if c == nil || c.service == nil {
		return ImageInfo{}, ErrInvalidCredentials
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return ImageInfo{}, fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	resp, err := c.service.Edits.Images.Upload(packageName, editID, language, imageType).Media(file).Context(ctx).Do()
	if err != nil {
		return ImageInfo{}, mapGoogleAPIError(err)
	}
	return imageInfoFromImage(resp.Image), nil
}

func (c *Client) DeleteImage(ctx context.Context, packageName, editID, language, imageType, imageID string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return fmt.Errorf("edit id is required")
	}
	language = strings.TrimSpace(language)
	if language == "" {
		return fmt.Errorf("language is required")
	}
	imageType = strings.TrimSpace(imageType)
	if imageType == "" {
		return fmt.Errorf("image type is required")
	}
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return fmt.Errorf("image id is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Edits.Images.Delete(packageName, editID, language, imageType, imageID).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) DeleteAllImages(ctx context.Context, packageName, editID, language, imageType string) ([]ImageInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, fmt.Errorf("edit id is required")
	}
	language = strings.TrimSpace(language)
	if language == "" {
		return nil, fmt.Errorf("language is required")
	}
	imageType = strings.TrimSpace(imageType)
	if imageType == "" {
		return nil, fmt.Errorf("image type is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Edits.Images.Deleteall(packageName, editID, language, imageType).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	deleted := make([]ImageInfo, 0, len(resp.Deleted))
	for _, image := range resp.Deleted {
		deleted = append(deleted, imageInfoFromImage(image))
	}
	return deleted, nil
}

func (c *Client) GetExpansionFile(ctx context.Context, packageName, editID string, apkVersionCode int64, expansionFileType string) (ExpansionFileInfo, error) {
	packageName, editID, apkVersionCode, expansionFileType, err := normalizeExpansionFileTarget(packageName, editID, apkVersionCode, expansionFileType)
	if err != nil {
		return ExpansionFileInfo{}, err
	}
	if c == nil || c.service == nil {
		return ExpansionFileInfo{}, ErrInvalidCredentials
	}

	file, err := c.service.Edits.Expansionfiles.Get(packageName, editID, apkVersionCode, expansionFileType).Context(ctx).Do()
	if err != nil {
		return ExpansionFileInfo{}, mapGoogleAPIError(err)
	}
	return expansionFileInfoFromExpansionFile(file), nil
}

func (c *Client) PatchExpansionFile(ctx context.Context, packageName, editID string, apkVersionCode int64, expansionFileType string, referencesVersion int64) (ExpansionFileInfo, error) {
	packageName, editID, apkVersionCode, expansionFileType, err := normalizeExpansionFileTarget(packageName, editID, apkVersionCode, expansionFileType)
	if err != nil {
		return ExpansionFileInfo{}, err
	}
	if referencesVersion <= 0 {
		return ExpansionFileInfo{}, fmt.Errorf("references version must be greater than zero")
	}
	if c == nil || c.service == nil {
		return ExpansionFileInfo{}, ErrInvalidCredentials
	}

	file, err := c.service.Edits.Expansionfiles.Patch(packageName, editID, apkVersionCode, expansionFileType, &androidpublisher.ExpansionFile{
		ReferencesVersion: referencesVersion,
	}).Context(ctx).Do()
	if err != nil {
		return ExpansionFileInfo{}, mapGoogleAPIError(err)
	}
	return expansionFileInfoFromExpansionFile(file), nil
}

func (c *Client) UpdateExpansionFile(ctx context.Context, packageName, editID string, apkVersionCode int64, expansionFileType string, referencesVersion int64) (ExpansionFileInfo, error) {
	packageName, editID, apkVersionCode, expansionFileType, err := normalizeExpansionFileTarget(packageName, editID, apkVersionCode, expansionFileType)
	if err != nil {
		return ExpansionFileInfo{}, err
	}
	if referencesVersion <= 0 {
		return ExpansionFileInfo{}, fmt.Errorf("references version must be greater than zero")
	}
	if c == nil || c.service == nil {
		return ExpansionFileInfo{}, ErrInvalidCredentials
	}

	file, err := c.service.Edits.Expansionfiles.Update(packageName, editID, apkVersionCode, expansionFileType, &androidpublisher.ExpansionFile{
		ReferencesVersion: referencesVersion,
	}).Context(ctx).Do()
	if err != nil {
		return ExpansionFileInfo{}, mapGoogleAPIError(err)
	}
	return expansionFileInfoFromExpansionFile(file), nil
}

func (c *Client) UploadExpansionFile(ctx context.Context, packageName, editID string, apkVersionCode int64, expansionFileType, filePath string) (ExpansionFileInfo, error) {
	packageName, editID, apkVersionCode, expansionFileType, err := normalizeExpansionFileTarget(packageName, editID, apkVersionCode, expansionFileType)
	if err != nil {
		return ExpansionFileInfo{}, err
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return ExpansionFileInfo{}, fmt.Errorf("expansion file path is required")
	}
	if c == nil || c.service == nil {
		return ExpansionFileInfo{}, ErrInvalidCredentials
	}

	file, err := os.Open(filePath)
	if err != nil {
		return ExpansionFileInfo{}, fmt.Errorf("open expansion file: %w", err)
	}
	defer file.Close()

	resp, err := c.service.Edits.Expansionfiles.Upload(packageName, editID, apkVersionCode, expansionFileType).Media(file).Context(ctx).Do()
	if err != nil {
		return ExpansionFileInfo{}, mapGoogleAPIError(err)
	}
	return expansionFileInfoFromExpansionFile(resp.ExpansionFile), nil
}

func (c *Client) DeleteListing(ctx context.Context, packageName, editID, language string) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return fmt.Errorf("edit id is required")
	}
	language = strings.TrimSpace(language)
	if language == "" {
		return fmt.Errorf("language is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Edits.Listings.Delete(packageName, editID, language).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func (c *Client) DeleteAllListings(ctx context.Context, packageName, editID string) error {
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

	if err := c.service.Edits.Listings.Deleteall(packageName, editID).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func appDetailsInfoFromDetails(details *androidpublisher.AppDetails) AppDetailsInfo {
	if details == nil {
		return AppDetailsInfo{}
	}
	return AppDetailsInfo{
		DefaultLanguage: details.DefaultLanguage,
		ContactEmail:    details.ContactEmail,
		ContactPhone:    details.ContactPhone,
		ContactWebsite:  details.ContactWebsite,
	}
}

func testersInfoFromTesters(track string, testers *androidpublisher.Testers) TestersInfo {
	info := TestersInfo{Track: track}
	if testers == nil {
		return info
	}
	info.GoogleGroups = append(info.GoogleGroups, testers.GoogleGroups...)
	return info
}

func countryAvailabilityInfoFromTrackCountryAvailability(track string, availability *androidpublisher.TrackCountryAvailability) CountryAvailabilityInfo {
	info := CountryAvailabilityInfo{Track: track}
	if availability == nil {
		return info
	}
	info.RestOfWorld = availability.RestOfWorld
	info.SyncWithProduction = availability.SyncWithProduction
	for _, country := range availability.Countries {
		if country == nil {
			continue
		}
		info.Countries = append(info.Countries, CountryTargetedInfo{CountryCode: country.CountryCode})
	}
	return info
}

func imageInfoFromImage(image *androidpublisher.Image) ImageInfo {
	if image == nil {
		return ImageInfo{}
	}
	return ImageInfo{
		ID:     image.Id,
		SHA1:   image.Sha1,
		SHA256: image.Sha256,
		URL:    image.Url,
	}
}

func normalizeExpansionFileTarget(packageName, editID string, apkVersionCode int64, expansionFileType string) (string, string, int64, string, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return "", "", 0, "", fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return "", "", 0, "", fmt.Errorf("edit id is required")
	}
	if apkVersionCode <= 0 {
		return "", "", 0, "", fmt.Errorf("apk version code must be greater than zero")
	}
	expansionFileType, err := normalizeExpansionFileType(expansionFileType)
	if err != nil {
		return "", "", 0, "", err
	}
	return packageName, editID, apkVersionCode, expansionFileType, nil
}

func normalizeExpansionFileType(expansionFileType string) (string, error) {
	expansionFileType = strings.ToLower(strings.TrimSpace(expansionFileType))
	switch expansionFileType {
	case "main", "patch":
		return expansionFileType, nil
	default:
		return "", fmt.Errorf("expansion file type must be one of: main, patch")
	}
}

func expansionFileInfoFromExpansionFile(file *androidpublisher.ExpansionFile) ExpansionFileInfo {
	if file == nil {
		return ExpansionFileInfo{}
	}
	return ExpansionFileInfo{
		FileSize:          file.FileSize,
		ReferencesVersion: file.ReferencesVersion,
	}
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
