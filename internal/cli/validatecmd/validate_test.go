package validatecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/listing"
	"github.com/leszko11/google-play-console-cli/internal/cli/release"
)

type fakeEditClient struct {
	validateErr error
	calls       int
}

func (f *fakeEditClient) ValidateEdit(_ context.Context, _, _ string) error {
	f.calls++
	return f.validateErr
}

func runCommand(t *testing.T, deps Deps, args ...string) (result, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}

	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)

	var res result
	if out.Len() > 0 {
		if decodeErr := json.Unmarshal(out.Bytes(), &res); decodeErr != nil {
			t.Fatalf("decode json: %v\noutput=%s", decodeErr, out.String())
		}
	}
	return res, err
}

func TestValidateCommandSuccess(t *testing.T) {
	client := &fakeEditClient{}
	res, err := runCommand(t, Deps{
		RunReleaseVerify: func(context.Context, release.VerifyOptions) (release.VerifyResult, error) {
			return release.VerifyResult{
				PackageName: "com.example.app",
				Track:       "alpha",
				ProjectDir:  ".",
				ListingDir:  "/tmp/listing",
				NotesMode:   "none",
				Status:      "ok",
				Checks: []release.VerifyCheck{
					{Name: "listing_metadata", Status: "ok", Detail: "listing metadata ready"},
				},
			}, nil
		},
		ValidateListings: func(string) (listing.ValidationSummary, error) {
			return listing.ValidationSummary{LocaleCount: 1, ImageCount: 2}, nil
		},
		ValidateEdit: func(context.Context, string, string) error {
			client.calls++
			return client.validateErr
		},
	}, "--package-name", "com.example.app", "--edit-id", "edit-1", "--notes-mode", "none")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "ok" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if client.calls != 1 {
		t.Fatalf("expected edit validation call, got %d", client.calls)
	}
	assertContainsCheck(t, res.Checks, "listing_assets", "ok")
	assertContainsCheck(t, res.Checks, "edit_validation", "ok")
}

func TestValidateCommandBlocksOnListingAssetFailure(t *testing.T) {
	res, err := runCommand(t, Deps{
		RunReleaseVerify: func(context.Context, release.VerifyOptions) (release.VerifyResult, error) {
			return release.VerifyResult{
				PackageName: "com.example.app",
				Track:       "alpha",
				ProjectDir:  ".",
				ListingDir:  "/tmp/listing",
				NotesMode:   "none",
				Status:      "ok",
				Checks: []release.VerifyCheck{
					{Name: "listing_metadata", Status: "ok", Detail: "listing metadata ready"},
				},
			}, nil
		},
		ValidateListings: func(string) (listing.ValidationSummary, error) {
			return listing.ValidationSummary{}, errors.New("en-US icon: invalid dimensions")
		},
	}, "--package-name", "com.example.app", "--notes-mode", "none")
	if err == nil {
		t.Fatalf("expected validation failure")
	}
	if res.Status != "failed" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(res.BlockingIssues) == 0 || !strings.Contains(res.BlockingIssues[0], "invalid dimensions") {
		t.Fatalf("unexpected blocking issues: %+v", res.BlockingIssues)
	}
	assertContainsCheck(t, res.Checks, "listing_assets", "error")
}

func TestValidateCommandPropagatesReleaseError(t *testing.T) {
	_, err := runCommand(t, Deps{
		RunReleaseVerify: func(context.Context, release.VerifyOptions) (release.VerifyResult, error) {
			return release.VerifyResult{}, errors.New("boom")
		},
	}, "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertContainsCheck(t *testing.T, checks []release.VerifyCheck, name, status string) {
	t.Helper()
	for _, check := range checks {
		if check.Name == name && check.Status == status {
			return
		}
	}
	t.Fatalf("expected check %q with status %q, got %+v", name, status, checks)
}
