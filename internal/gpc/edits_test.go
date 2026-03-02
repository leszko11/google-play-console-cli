package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestEditMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.CreateEdit(context.Background(), "com.example.app"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CreateEdit, got %v", err)
	}
	if _, err := c.GetEdit(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetEdit, got %v", err)
	}
	if err := c.ValidateEdit(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ValidateEdit, got %v", err)
	}
	if _, err := c.CommitEdit(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from CommitEdit, got %v", err)
	}
	if err := c.DeleteEdit(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteEdit, got %v", err)
	}
	if _, err := c.ListListings(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListListings, got %v", err)
	}
	if err := c.DeleteListing(context.Background(), "com.example.app", "edit-1", "en-US"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteListing, got %v", err)
	}
	if err := c.DeleteAllListings(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from DeleteAllListings, got %v", err)
	}
	if _, err := c.GetAppDetails(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetAppDetails, got %v", err)
	}
	if _, err := c.UpdateAppDetails(context.Background(), "com.example.app", "edit-1", AppDetailsUpdate{ContactEmail: "help@example.com"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UpdateAppDetails, got %v", err)
	}
	if _, err := c.GetTesters(context.Background(), "com.example.app", "edit-1", "internal"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetTesters, got %v", err)
	}
	if _, err := c.UpdateTesters(context.Background(), "com.example.app", "edit-1", "internal", []string{"qa@example.com"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UpdateTesters, got %v", err)
	}
	if _, err := c.GetCountryAvailability(context.Background(), "com.example.app", "edit-1", "production"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from GetCountryAvailability, got %v", err)
	}
}

func TestEditMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.CreateEdit(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected CreateEdit error: %v", err)
	}
	if _, err := c.GetEdit(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected GetEdit error: %v", err)
	}
	if err := c.ValidateEdit(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected ValidateEdit error: %v", err)
	}
	if _, err := c.CommitEdit(context.Background(), "", "edit-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected CommitEdit error: %v", err)
	}
	if err := c.DeleteEdit(context.Background(), "", "edit-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected DeleteEdit error: %v", err)
	}
}

func TestEditInfoFromAppEdit(t *testing.T) {
	got := editInfoFromAppEdit(&androidpublisher.AppEdit{
		Id:                "edit-123",
		ExpiryTimeSeconds: "1234567890",
	})
	if got.ID != "edit-123" || got.ExpiryTimeSeconds != "1234567890" {
		t.Fatalf("unexpected mapped edit: %+v", got)
	}
}

func TestListingMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.GetListing(context.Background(), "com.example.app", "", "en-US"); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected GetListing error: %v", err)
	}
	if _, err := c.GetListing(context.Background(), "com.example.app", "edit-1", ""); err == nil || !strings.Contains(err.Error(), "language is required") {
		t.Fatalf("unexpected GetListing error: %v", err)
	}
	if _, err := c.UpdateListing(context.Background(), "com.example.app", "edit-1", "en-US", ListingUpdate{}); err == nil || !strings.Contains(err.Error(), "at least one listing field must be provided") {
		t.Fatalf("unexpected UpdateListing error: %v", err)
	}
	if _, err := c.ListListings(context.Background(), "", "edit-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListListings package error: %v", err)
	}
	if _, err := c.ListListings(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected ListListings edit id error: %v", err)
	}
	if err := c.DeleteListing(context.Background(), "com.example.app", "edit-1", ""); err == nil || !strings.Contains(err.Error(), "language is required") {
		t.Fatalf("unexpected DeleteListing language error: %v", err)
	}
	if err := c.DeleteAllListings(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected DeleteAllListings edit id error: %v", err)
	}
	if _, err := c.GetAppDetails(context.Background(), "", "edit-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected GetAppDetails package error: %v", err)
	}
	if _, err := c.GetAppDetails(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected GetAppDetails edit id error: %v", err)
	}
	if _, err := c.UpdateAppDetails(context.Background(), "com.example.app", "edit-1", AppDetailsUpdate{}); err == nil || !strings.Contains(err.Error(), "at least one app detail field must be provided") {
		t.Fatalf("unexpected UpdateAppDetails empty update error: %v", err)
	}
	if _, err := c.GetTesters(context.Background(), "com.example.app", "edit-1", ""); err == nil || !strings.Contains(err.Error(), "track is required") {
		t.Fatalf("unexpected GetTesters track error: %v", err)
	}
	if _, err := c.UpdateTesters(context.Background(), "com.example.app", "edit-1", "internal", nil); err == nil || !strings.Contains(err.Error(), "at least one google group is required") {
		t.Fatalf("unexpected UpdateTesters groups error: %v", err)
	}
	if _, err := c.GetCountryAvailability(context.Background(), "com.example.app", "edit-1", ""); err == nil || !strings.Contains(err.Error(), "track is required") {
		t.Fatalf("unexpected GetCountryAvailability track error: %v", err)
	}
}

func TestListingInfoFromListing(t *testing.T) {
	got := listingInfoFromListing(&androidpublisher.Listing{
		Language:         "en-US",
		Title:            "PeakMe",
		ShortDescription: "Short",
		FullDescription:  "Full",
	})
	if got.Language != "en-US" || got.Title != "PeakMe" {
		t.Fatalf("unexpected listing map: %+v", got)
	}
}

func TestAppDetailsInfoFromDetails(t *testing.T) {
	got := appDetailsInfoFromDetails(&androidpublisher.AppDetails{
		DefaultLanguage: "en-US",
		ContactEmail:    "support@example.com",
		ContactPhone:    "+48123456789",
		ContactWebsite:  "https://example.com",
	})
	if got.DefaultLanguage != "en-US" || got.ContactEmail != "support@example.com" || got.ContactWebsite != "https://example.com" {
		t.Fatalf("unexpected app details map: %+v", got)
	}
}

func TestTestersInfoFromTesters(t *testing.T) {
	got := testersInfoFromTesters("internal", &androidpublisher.Testers{
		GoogleGroups: []string{"qa-team@example.com"},
	})
	if got.Track != "internal" || len(got.GoogleGroups) != 1 || got.GoogleGroups[0] != "qa-team@example.com" {
		t.Fatalf("unexpected testers map: %+v", got)
	}
}

func TestCountryAvailabilityInfoFromTrackCountryAvailability(t *testing.T) {
	got := countryAvailabilityInfoFromTrackCountryAvailability("production", &androidpublisher.TrackCountryAvailability{
		RestOfWorld:        true,
		SyncWithProduction: false,
		Countries: []*androidpublisher.TrackTargetedCountry{
			{CountryCode: "PL"},
			{CountryCode: "US"},
		},
	})
	if got.Track != "production" || !got.RestOfWorld || len(got.Countries) != 2 {
		t.Fatalf("unexpected country availability map: %+v", got)
	}
}
