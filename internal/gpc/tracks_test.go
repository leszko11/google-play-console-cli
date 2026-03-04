package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestTrackMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.ListTracks(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListTracks, got %v", err)
	}
	if _, err := c.GetTrack(context.Background(), "com.example.app", "edit-1", "production"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetTrack, got %v", err)
	}
	if _, err := c.UpdateTrack(context.Background(), "com.example.app", "edit-1", "production", TrackUpdate{
		Status:       "completed",
		VersionCodes: []int64{1},
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UpdateTrack, got %v", err)
	}
}

func TestTrackMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListTracks(context.Background(), "", "edit-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListTracks error: %v", err)
	}
	if _, err := c.GetTrack(context.Background(), "com.example.app", "", "production"); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected GetTrack error: %v", err)
	}
	if _, err := c.GetTrack(context.Background(), "com.example.app", "edit-1", ""); err == nil || !strings.Contains(err.Error(), "track is required") {
		t.Fatalf("unexpected GetTrack error: %v", err)
	}
	if _, err := c.UpdateTrack(context.Background(), "com.example.app", "edit-1", "production", TrackUpdate{}); err == nil || !strings.Contains(err.Error(), "status is required") {
		t.Fatalf("unexpected UpdateTrack error: %v", err)
	}
	if _, err := c.UpdateTrack(context.Background(), "com.example.app", "edit-1", "production", TrackUpdate{Status: "completed"}); err == nil || !strings.Contains(err.Error(), "at least one version code is required") {
		t.Fatalf("unexpected UpdateTrack error: %v", err)
	}
}

func TestTrackInfoFromTrack(t *testing.T) {
	got := trackInfoFromTrack(&androidpublisher.Track{
		Track: "production",
		Releases: []*androidpublisher.TrackRelease{
			{
				Name:                "1.0.0",
				Status:              "completed",
				UserFraction:        0.5,
				VersionCodes:        []int64{123, 456},
				InAppUpdatePriority: 3,
				ReleaseNotes: []*androidpublisher.LocalizedText{
					{
						Language: "en-US",
						Text:     "Bug fixes and stability improvements.",
					},
				},
			},
		},
	})

	if got.Name != "production" {
		t.Fatalf("unexpected track name: %+v", got)
	}
	if len(got.Releases) != 1 {
		t.Fatalf("expected 1 release, got %+v", got)
	}
	if got.Releases[0].Status != "completed" || got.Releases[0].Name != "1.0.0" {
		t.Fatalf("unexpected release map: %+v", got.Releases[0])
	}
	if len(got.Releases[0].VersionCodes) != 2 || got.Releases[0].VersionCodes[0] != 123 {
		t.Fatalf("unexpected version codes map: %+v", got.Releases[0])
	}
	if len(got.Releases[0].ReleaseNotes) != 1 || got.Releases[0].ReleaseNotes[0].Language != "en-US" {
		t.Fatalf("unexpected release notes map: %+v", got.Releases[0].ReleaseNotes)
	}
}

func TestTrackReleaseRequestFromUpdate(t *testing.T) {
	release := trackReleaseRequestFromUpdate(TrackUpdate{
		Status:         "completed",
		ReleaseName:    "1.0.1",
		UserFraction:   0.3,
		VersionCodes:   []int64{123},
		UpdatePriority: 4,
		ReleaseNotes: []LocalizedReleaseNote{
			{
				Language: "en-US",
				Text:     "Polish and bug fixes.",
			},
			{
				Language: "",
				Text:     "ignored because no locale",
			},
		},
	})

	if release == nil {
		t.Fatal("expected release request")
	}
	if release.Status != "completed" || release.Name != "1.0.1" {
		t.Fatalf("unexpected release metadata: %+v", release)
	}
	if len(release.VersionCodes) != 1 || release.VersionCodes[0] != 123 {
		t.Fatalf("unexpected version codes: %+v", release.VersionCodes)
	}
	if release.InAppUpdatePriority != 4 {
		t.Fatalf("expected update priority 4, got %d", release.InAppUpdatePriority)
	}
	if len(release.ReleaseNotes) != 1 || release.ReleaseNotes[0].Language != "en-US" {
		t.Fatalf("unexpected release notes: %+v", release.ReleaseNotes)
	}
}
