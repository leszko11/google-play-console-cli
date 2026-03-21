package rollback

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	createEdit    gpc.EditInfo
	createEditErr error

	getTrack    gpc.TrackInfo
	getTrackErr error

	updateTrack    gpc.TrackInfo
	updateTrackErr error
	lastTrackName  string
	lastUpdate     gpc.TrackUpdate

	validateErr error
	commitErr   error
	deleteErr   error
	deleteCalls int
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	if f.createEditErr != nil {
		return gpc.EditInfo{}, f.createEditErr
	}
	if f.createEdit.ID == "" {
		return gpc.EditInfo{ID: "edit-1"}, nil
	}
	return f.createEdit, nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return f.deleteErr
}

func (f *fakeClient) ValidateEdit(_ context.Context, _, _ string) error {
	return f.validateErr
}

func (f *fakeClient) CommitEdit(_ context.Context, _, _ string, _ bool) (gpc.EditInfo, error) {
	if f.commitErr != nil {
		return gpc.EditInfo{}, f.commitErr
	}
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) GetTrack(_ context.Context, _, _, _ string) (gpc.TrackInfo, error) {
	if f.getTrackErr != nil {
		return gpc.TrackInfo{}, f.getTrackErr
	}
	if f.getTrack.Name == "" {
		return gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{
					Status:       "inProgress",
					UserFraction: 0.1,
					VersionCodes: []int64{123},
				},
			},
		}, nil
	}
	return f.getTrack, nil
}

func (f *fakeClient) UpdateTrack(_ context.Context, _, _, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
	f.lastTrackName = trackName
	f.lastUpdate = update
	if f.updateTrackErr != nil {
		return gpc.TrackInfo{}, f.updateTrackErr
	}
	if f.updateTrack.Name == "" {
		return gpc.TrackInfo{Name: trackName}, nil
	}
	return f.updateTrack, nil
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func runCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		}
	}

	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestRollbackRequiresTrack(t *testing.T) {
	_, err := runCommand(t, Deps{}, "--package-name", "com.example.app", "--confirm")
	if err == nil || !strings.Contains(err.Error(), "--track is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shared.IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}
}

func TestRollbackRequiresConfirmUnlessDryRun(t *testing.T) {
	_, err := runCommand(t, Deps{}, "--package-name", "com.example.app", "--track", "production")
	if err == nil || !strings.Contains(err.Error(), "--confirm is required unless --dry-run is set") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shared.IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}
}

func TestRollbackCommitSuccess(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps,
		"--package-name", "com.example.app",
		"--track", "production",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"committed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.lastTrackName != "production" {
		t.Fatalf("unexpected track name: %q", client.lastTrackName)
	}
	if client.lastUpdate.Status != "halted" {
		t.Fatalf("unexpected track status: %+v", client.lastUpdate)
	}
	if len(client.lastUpdate.VersionCodes) != 1 || client.lastUpdate.VersionCodes[0] != 123 {
		t.Fatalf("unexpected version codes: %+v", client.lastUpdate.VersionCodes)
	}
	if client.lastUpdate.UserFraction != 0.1 {
		t.Fatalf("unexpected user fraction: %+v", client.lastUpdate)
	}
}

func TestRollbackDryRunDeletesEdit(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps,
		"--package-name", "com.example.app",
		"--track", "production",
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected one delete call, got %d", client.deleteCalls)
	}
}

func TestRollbackFailsWhenNoInProgressRelease(t *testing.T) {
	client := &fakeClient{
		getTrack: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "draft", VersionCodes: []int64{123}},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps,
		"--package-name", "com.example.app",
		"--track", "production",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), `track "production" has no in-progress release to halt`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected cleanup delete, got %d", client.deleteCalls)
	}
}

func TestRollbackFailsWhenCompletedRelease(t *testing.T) {
	client := &fakeClient{
		getTrack: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "completed", UserFraction: 1, VersionCodes: []int64{123}},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps,
		"--package-name", "com.example.app",
		"--track", "production",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "cannot halt a completed rollout") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRollbackFailsWhenMultipleInProgressReleases(t *testing.T) {
	client := &fakeClient{
		getTrack: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "inProgress", UserFraction: 0.1, VersionCodes: []int64{123}},
				{Status: "inProgress", UserFraction: 0.2, VersionCodes: []int64{124}},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps,
		"--package-name", "com.example.app",
		"--track", "production",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), `track "production" has multiple in-progress releases; refusing to halt implicitly`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRollbackFailurePerformsCleanup(t *testing.T) {
	client := &fakeClient{updateTrackErr: errors.New("conflict")}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps,
		"--package-name", "com.example.app",
		"--track", "production",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "failed to update track") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"failed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected cleanup delete, got %d", client.deleteCalls)
	}
}
