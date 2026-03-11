package release

import (
	"context"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func TestValidatePromoteOptions(t *testing.T) {
	err := validatePromoteOptions(promoteOptions{
		PackageName: "com.example.app",
		FromTrack:   "alpha",
		ToTrack:     "production",
		Confirm:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = validatePromoteOptions(promoteOptions{
		PackageName: "com.example.app",
		FromTrack:   "alpha",
		ToTrack:     "alpha",
		Confirm:     true,
	})
	if err == nil || !strings.Contains(err.Error(), "--from-track and --to-track must be different") || !shared.IsUsageError(err) {
		t.Fatalf("unexpected same-track validation error: %v", err)
	}

	err = validatePromoteOptions(promoteOptions{
		PackageName: "com.example.app",
		FromTrack:   "alpha",
		ToTrack:     "production",
	})
	if err == nil || !strings.Contains(err.Error(), "--confirm is required unless --dry-run is set") || !shared.IsUsageError(err) {
		t.Fatalf("unexpected confirm validation error: %v", err)
	}
}

func TestLatestPromotableRelease(t *testing.T) {
	release, err := latestPromotableRelease(gpc.TrackInfo{
		Name: "alpha",
		Releases: []gpc.TrackReleaseInfo{
			{Status: "completed"},
			{Status: "inProgress", VersionCodes: []int64{123, 124}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(release.VersionCodes) != 2 || release.VersionCodes[0] != 123 {
		t.Fatalf("unexpected release: %+v", release)
	}
}

func TestLatestPromotableRelease_NoReleases(t *testing.T) {
	_, err := latestPromotableRelease(gpc.TrackInfo{Name: "alpha"})
	if err == nil || !strings.Contains(err.Error(), `source track "alpha" has no releases to promote`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLatestPromotableRelease_NoVersionCodes(t *testing.T) {
	_, err := latestPromotableRelease(gpc.TrackInfo{
		Name: "alpha",
		Releases: []gpc.TrackReleaseInfo{
			{Status: "draft"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), `source track "alpha" has no releasable versions to promote`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPromoteSuccess(t *testing.T) {
	client := &fakeReleaseClient{
		createEditIDs: []string{"edit-promote"},
		getTrackInfo: gpc.TrackInfo{
			Name: "alpha",
			Releases: []gpc.TrackReleaseInfo{
				{
					Name:           "1.2.3",
					Status:         "completed",
					UserFraction:   0.5,
					VersionCodes:   []int64{456},
					UpdatePriority: 3,
					ReleaseNotes: []gpc.LocalizedText{
						{Language: "en-US", Text: "Notes"},
					},
				},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	result, err := runPromote(context.Background(), deps, promoteOptions{
		PackageName: "com.example.app",
		FromTrack:   "alpha",
		ToTrack:     "production",
		Confirm:     true,
	})
	if err != nil {
		t.Fatalf("expected success, got %v result=%+v", err, result)
	}
	if result.Status != "committed" || !result.Committed {
		t.Fatalf("unexpected promote result: %+v", result)
	}
	if client.lastTrackName != "production" {
		t.Fatalf("unexpected target track: %s", client.lastTrackName)
	}
	if client.lastTrack.Status != "completed" || client.lastTrack.ReleaseName != "1.2.3" {
		t.Fatalf("unexpected copied track payload: %+v", client.lastTrack)
	}
	if client.lastTrack.UserFraction != 0.5 || client.lastTrack.UpdatePriority != 3 || len(client.lastTrack.VersionCodes) != 1 || client.lastTrack.VersionCodes[0] != 456 {
		t.Fatalf("unexpected copied rollout payload: %+v", client.lastTrack)
	}
	if len(client.lastTrack.ReleaseNotes) != 1 || client.lastTrack.ReleaseNotes[0].Language != "en-US" {
		t.Fatalf("unexpected copied release notes: %+v", client.lastTrack.ReleaseNotes)
	}
}

func TestRunPromoteDryRun(t *testing.T) {
	client := &fakeReleaseClient{
		createEditIDs: []string{"edit-promote"},
		getTrackInfo: gpc.TrackInfo{
			Name: "alpha",
			Releases: []gpc.TrackReleaseInfo{
				{
					Status:       "completed",
					VersionCodes: []int64{456},
				},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	result, err := runPromote(context.Background(), deps, promoteOptions{
		PackageName: "com.example.app",
		FromTrack:   "alpha",
		ToTrack:     "beta",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("expected dry-run success, got %v result=%+v", err, result)
	}
	if result.Status != "dry-run" || result.Committed {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}
	if !result.CleanupPerformed {
		t.Fatalf("expected cleanup performed, got %+v", result)
	}
	assertContainsPromoteStep(t, result.Steps, "delete_edit_dry_run", "ok")
}

func TestRunPromoteSourceTrackError(t *testing.T) {
	client := &fakeReleaseClient{createEditIDs: []string{"edit-promote"}, getTrackErr: context.DeadlineExceeded}
	deps := baseReleaseDeps(t, client)

	result, err := runPromote(context.Background(), deps, promoteOptions{
		PackageName: "com.example.app",
		FromTrack:   "alpha",
		ToTrack:     "beta",
		Confirm:     true,
	})
	if err == nil || !strings.Contains(err.Error(), "failed to read source track") {
		t.Fatalf("unexpected error: %v result=%+v", err, result)
	}
	assertContainsPromoteStep(t, result.Steps, "read_source_track", "error")
}

func TestRunPromoteSourceTrackNoReleases(t *testing.T) {
	client := &fakeReleaseClient{
		createEditIDs: []string{"edit-promote"},
		getTrackInfo:  gpc.TrackInfo{Name: "alpha"},
	}
	deps := baseReleaseDeps(t, client)

	result, err := runPromote(context.Background(), deps, promoteOptions{
		PackageName: "com.example.app",
		FromTrack:   "alpha",
		ToTrack:     "beta",
		Confirm:     true,
	})
	if err == nil || !strings.Contains(err.Error(), `source track "alpha" has no releases to promote`) {
		t.Fatalf("unexpected error: %v result=%+v", err, result)
	}
	assertContainsPromoteStep(t, result.Steps, "select_source_release", "error")
}

func TestRunPromoteStatusAndNameOverride(t *testing.T) {
	client := &fakeReleaseClient{
		createEditIDs: []string{"edit-promote"},
		getTrackInfo: gpc.TrackInfo{
			Name: "alpha",
			Releases: []gpc.TrackReleaseInfo{
				{
					Name:         "1.2.3",
					Status:       "completed",
					VersionCodes: []int64{456},
				},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	result, err := runPromote(context.Background(), deps, promoteOptions{
		PackageName: "com.example.app",
		FromTrack:   "alpha",
		ToTrack:     "production",
		Status:      "inProgress",
		ReleaseName: "1.2.3 rollout",
		Confirm:     true,
	})
	if err != nil {
		t.Fatalf("expected success, got %v result=%+v", err, result)
	}
	if result.ReleaseStatus != "inProgress" || result.ReleaseName != "1.2.3 rollout" {
		t.Fatalf("unexpected promote result: %+v", result)
	}
	if client.lastTrack.Status != "inProgress" || client.lastTrack.ReleaseName != "1.2.3 rollout" {
		t.Fatalf("unexpected override payload: %+v", client.lastTrack)
	}
}

func assertContainsPromoteStep(t *testing.T, steps []promoteStep, name, status string) {
	t.Helper()
	for _, step := range steps {
		if step.Name == name && step.Status == status {
			return
		}
	}
	t.Fatalf("missing step %q with status %q: %+v", name, status, steps)
}
