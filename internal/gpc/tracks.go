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
	update.Status = strings.TrimSpace(update.Status)
	if update.Status == "" {
		return TrackInfo{}, fmt.Errorf("status is required")
	}
	if len(update.VersionCodes) == 0 {
		return TrackInfo{}, fmt.Errorf("at least one version code is required")
	}
	if c == nil || c.service == nil {
		return TrackInfo{}, ErrInvalidCredentials
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

	req := &androidpublisher.Track{
		Track:    trackName,
		Releases: []*androidpublisher.TrackRelease{release},
	}

	track, err := c.service.Edits.Tracks.Update(packageName, editID, trackName, req).Context(ctx).Do()
	if err != nil {
		return TrackInfo{}, mapGoogleAPIError(err)
	}
	return trackInfoFromTrack(track), nil
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
	return TrackReleaseInfo{
		Name:           release.Name,
		Status:         release.Status,
		UserFraction:   release.UserFraction,
		VersionCodes:   release.VersionCodes,
		UpdatePriority: release.InAppUpdatePriority,
	}
}
