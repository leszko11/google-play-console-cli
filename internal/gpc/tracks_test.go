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
	if _, err := c.PatchTrack(context.Background(), "com.example.app", "edit-1", "production", TrackUpdate{
		Status:       "completed",
		VersionCodes: []int64{1},
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from PatchTrack, got %v", err)
	}
	if _, err := c.CreateTrack(context.Background(), "com.example.app", "edit-1", TrackCreate{Track: "production"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateTrack, got %v", err)
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
	if _, err := c.PatchTrack(context.Background(), "com.example.app", "edit-1", "production", TrackUpdate{}); err == nil || !strings.Contains(err.Error(), "status is required") {
		t.Fatalf("unexpected PatchTrack error: %v", err)
	}
	if _, err := c.CreateTrack(context.Background(), "com.example.app", "edit-1", TrackCreate{}); err == nil || !strings.Contains(err.Error(), "track is required") {
		t.Fatalf("unexpected CreateTrack error: %v", err)
	}
	if _, err := c.UpdateTrack(context.Background(), "com.example.app", "edit-1", "production", TrackUpdate{Status: "completed"}); err == nil || !strings.Contains(err.Error(), "at least one version code is required") {
		t.Fatalf("unexpected UpdateTrack error: %v", err)
	}
	if _, err := c.PatchTrack(context.Background(), "com.example.app", "edit-1", "production", TrackUpdate{Status: "completed"}); err == nil || !strings.Contains(err.Error(), "at least one version code is required") {
		t.Fatalf("unexpected PatchTrack error: %v", err)
	}
	if _, err := c.CreateTrack(context.Background(), "com.example.app", "edit-1", TrackCreate{Track: "production", FormFactor: "fridge"}); err == nil || !strings.Contains(err.Error(), "form factor must be one of") {
		t.Fatalf("unexpected CreateTrack form factor error: %v", err)
	}
	if _, err := c.UpdateTrack(context.Background(), "com.example.app", "edit-1", "production", TrackUpdate{
		Status:       "completed",
		VersionCodes: []int64{1},
		ReleaseNotes: []LocalizedText{{Language: "en-US", Text: "   "}},
	}); err == nil || !strings.Contains(err.Error(), "release note text is required") {
		t.Fatalf("unexpected UpdateTrack error: %v", err)
	}
	if _, err := c.PatchTrack(context.Background(), "com.example.app", "edit-1", "production", TrackUpdate{
		Status:       "completed",
		VersionCodes: []int64{1},
		ReleaseNotes: []LocalizedText{{Language: "en-US", Text: "   "}},
	}); err == nil || !strings.Contains(err.Error(), "release note text is required") {
		t.Fatalf("unexpected PatchTrack error: %v", err)
	}
	if _, err := c.CreateTrack(context.Background(), "com.example.app", "edit-1", TrackCreate{Track: "production", Type: "open"}); err == nil || !strings.Contains(err.Error(), "track type must be one of") {
		t.Fatalf("unexpected CreateTrack type error: %v", err)
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
					{Language: "en-US", Text: "Release note"},
					{Language: "pl-PL", Text: "Notatki wydania"},
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
	if len(got.Releases[0].ReleaseNotes) != 2 || got.Releases[0].ReleaseNotes[0].Language != "en-US" {
		t.Fatalf("unexpected release notes map: %+v", got.Releases[0])
	}
}
