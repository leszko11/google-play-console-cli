package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListTracks(ctx context.Context, packageName, editID string) ([]TrackInfo, error) {
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

	resp, err := c.service.Edits.Tracks.List(packageName, editID).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}

	tracks := make([]TrackInfo, 0, len(resp.Tracks))
	for _, track := range resp.Tracks {
		tracks = append(tracks, trackInfoFromTrack(track))
	}
	return tracks, nil
}

func (c *Client) GetTrack(ctx context.Context, packageName, editID, trackName string) (TrackInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return TrackInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return TrackInfo{}, fmt.Errorf("edit id is required")
	}
	trackName = strings.TrimSpace(trackName)
	if trackName == "" {
		return TrackInfo{}, fmt.Errorf("track is required")
	}
	if c == nil || c.service == nil {
		return TrackInfo{}, ErrInvalidCredentials
	}

	track, err := c.service.Edits.Tracks.Get(packageName, editID, trackName).Context(ctx).Do()
	if err != nil {
		return TrackInfo{}, mapGoogleAPIError(err)
	}
	return trackInfoFromTrack(track), nil
}

func (c *Client) UpdateTrack(ctx context.Context, packageName, editID, trackName string, update TrackUpdate) (TrackInfo, error) {
	packageName, editID, trackName, req, err := normalizeTrackUpdateRequest(packageName, editID, trackName, update)
	if err != nil {
		return TrackInfo{}, err
	}
	if c == nil || c.service == nil {
		return TrackInfo{}, ErrInvalidCredentials
	}

	track, err := c.service.Edits.Tracks.Update(packageName, editID, trackName, req).Context(ctx).Do()
	if err != nil {
		return TrackInfo{}, mapGoogleAPIError(err)
	}
	return trackInfoFromTrack(track), nil
}

func (c *Client) PatchTrack(ctx context.Context, packageName, editID, trackName string, update TrackUpdate) (TrackInfo, error) {
	packageName, editID, trackName, req, err := normalizeTrackUpdateRequest(packageName, editID, trackName, update)
	if err != nil {
		return TrackInfo{}, err
	}
	if c == nil || c.service == nil {
		return TrackInfo{}, ErrInvalidCredentials
	}

	track, err := c.service.Edits.Tracks.Patch(packageName, editID, trackName, req).Context(ctx).Do()
	if err != nil {
		return TrackInfo{}, mapGoogleAPIError(err)
	}
	return trackInfoFromTrack(track), nil
}

func (c *Client) CreateTrack(ctx context.Context, packageName, editID string, create TrackCreate) (TrackInfo, error) {
	packageName, editID, req, err := normalizeTrackCreateRequest(packageName, editID, create)
	if err != nil {
		return TrackInfo{}, err
	}
	if c == nil || c.service == nil {
		return TrackInfo{}, ErrInvalidCredentials
	}

	track, err := c.service.Edits.Tracks.Create(packageName, editID, req).Context(ctx).Do()
	if err != nil {
		return TrackInfo{}, mapGoogleAPIError(err)
	}
	return trackInfoFromTrack(track), nil
}

func normalizeTrackUpdateRequest(packageName, editID, trackName string, update TrackUpdate) (string, string, string, *androidpublisher.Track, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return "", "", "", nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return "", "", "", nil, fmt.Errorf("edit id is required")
	}
	trackName = strings.TrimSpace(trackName)
	if trackName == "" {
		return "", "", "", nil, fmt.Errorf("track is required")
	}

	release, err := normalizeTrackRelease(update)
	if err != nil {
		return "", "", "", nil, err
	}

	return packageName, editID, trackName, &androidpublisher.Track{
		Track:    trackName,
		Releases: []*androidpublisher.TrackRelease{release},
	}, nil
}

func normalizeTrackRelease(update TrackUpdate) (*androidpublisher.TrackRelease, error) {
	update.Status = strings.TrimSpace(update.Status)
	if update.Status == "" {
		return nil, fmt.Errorf("status is required")
	}
	if len(update.VersionCodes) == 0 {
		return nil, fmt.Errorf("at least one version code is required")
	}

	releaseNotes := make([]*androidpublisher.LocalizedText, 0, len(update.ReleaseNotes))
	for _, note := range update.ReleaseNotes {
		language := strings.TrimSpace(note.Language)
		if language == "" {
			return nil, fmt.Errorf("release note language is required")
		}
		text := strings.TrimSpace(note.Text)
		if text == "" {
			return nil, fmt.Errorf("release note text is required for locale %q", language)
		}
		releaseNotes = append(releaseNotes, &androidpublisher.LocalizedText{
			Language: language,
			Text:     text,
		})
	}

	release := &androidpublisher.TrackRelease{
		Status:       update.Status,
		Name:         strings.TrimSpace(update.ReleaseName),
		VersionCodes: update.VersionCodes,
	}
	if update.UserFraction >= 0 {
		release.UserFraction = update.UserFraction
	}
	if update.UpdatePriority > 0 {
		release.InAppUpdatePriority = update.UpdatePriority
	}
	if len(releaseNotes) > 0 {
		release.ReleaseNotes = releaseNotes
	}
	return release, nil
}

func normalizeTrackCreateRequest(packageName, editID string, create TrackCreate) (string, string, *androidpublisher.TrackConfig, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return "", "", nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return "", "", nil, fmt.Errorf("edit id is required")
	}
	create.Track = strings.TrimSpace(create.Track)
	if create.Track == "" {
		return "", "", nil, fmt.Errorf("track is required")
	}

	formFactor, err := normalizeTrackFormFactor(create.FormFactor)
	if err != nil {
		return "", "", nil, err
	}
	trackType, err := normalizeTrackType(create.Type)
	if err != nil {
		return "", "", nil, err
	}

	return packageName, editID, &androidpublisher.TrackConfig{
		Track:      create.Track,
		FormFactor: formFactor,
		Type:       trackType,
	}, nil
}

func normalizeTrackFormFactor(value string) (string, error) {
	normalized := compactTrackEnum(value)
	switch normalized {
	case "", "DEFAULT":
		return "DEFAULT", nil
	case "WEAR":
		return "WEAR", nil
	case "AUTOMOTIVE":
		return "AUTOMOTIVE", nil
	default:
		return "", fmt.Errorf("form factor must be one of: default, wear, automotive")
	}
}

func normalizeTrackType(value string) (string, error) {
	normalized := compactTrackEnum(value)
	switch normalized {
	case "", "CLOSEDTESTING":
		return "CLOSED_TESTING", nil
	default:
		return "", fmt.Errorf("track type must be one of: closed-testing")
	}
}

func compactTrackEnum(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, " ", "")
	return strings.ToUpper(value)
}

func trackInfoFromTrack(track *androidpublisher.Track) TrackInfo {
	if track == nil {
		return TrackInfo{}
	}

	releases := make([]TrackReleaseInfo, 0, len(track.Releases))
	for _, release := range track.Releases {
		releases = append(releases, trackReleaseInfoFromRelease(release))
	}

	return TrackInfo{
		Name:     track.Track,
		Releases: releases,
	}
}

func trackReleaseInfoFromRelease(release *androidpublisher.TrackRelease) TrackReleaseInfo {
	if release == nil {
		return TrackReleaseInfo{}
	}
	releaseNotes := make([]LocalizedText, 0, len(release.ReleaseNotes))
	for _, note := range release.ReleaseNotes {
		if note == nil {
			continue
		}
		releaseNotes = append(releaseNotes, LocalizedText{
			Language: note.Language,
			Text:     note.Text,
		})
	}

	return TrackReleaseInfo{
		Name:           release.Name,
		Status:         release.Status,
		UserFraction:   release.UserFraction,
		VersionCodes:   release.VersionCodes,
		UpdatePriority: release.InAppUpdatePriority,
		ReleaseNotes:   releaseNotes,
	}
}
