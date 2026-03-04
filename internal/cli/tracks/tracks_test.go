package tracks

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	list         []gpc.TrackInfo
	listErr      error
	get          gpc.TrackInfo
	getErr       error
	getByName    map[string]gpc.TrackInfo
	getErrMap    map[string]error
	update       gpc.TrackInfo
	updateErr    error
	updateByName map[string]gpc.TrackInfo
	updateErrMap map[string]error
	updateFn     func(packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error)
}

func (f fakeClient) ListTracks(_ context.Context, _, _ string) ([]gpc.TrackInfo, error) {
	return f.list, f.listErr
}

func (f fakeClient) GetTrack(_ context.Context, _, _, trackName string) (gpc.TrackInfo, error) {
	if err := f.getErrMap[trackName]; err != nil {
		return gpc.TrackInfo{}, err
	}
	if track, ok := f.getByName[trackName]; ok {
		return track, nil
	}
	return f.get, f.getErr
}

func (f fakeClient) UpdateTrack(_ context.Context, packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
	if f.updateFn != nil {
		return f.updateFn(packageName, editID, trackName, update)
	}
	if err := f.updateErrMap[trackName]; err != nil {
		return gpc.TrackInfo{}, err
	}
	if track, ok := f.updateByName[trackName]; ok {
		return track, nil
	}
	if f.updateErr != nil {
		return gpc.TrackInfo{}, f.updateErr
	}
	return f.update, nil
}

func runTracks(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}

	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func writeReleaseNotesFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "release-notes.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write release notes file: %v", err)
	}
	return path
}

func TestTracksList_ReturnsTracks(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				list: []gpc.TrackInfo{
					{Name: "internal"},
					{Name: "production"},
				},
			}, nil
		},
	}

	out, err := runTracks(t, deps, "list", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"name":"internal"`) || !strings.Contains(out, `"name":"production"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestTracksGet_RequiresTrack(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runTracks(t, deps, "get", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--track is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracksGet_ReturnsTrack(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{get: gpc.TrackInfo{Name: "production"}}, nil
		},
	}

	out, err := runTracks(t, deps, "get", "--package-name", "com.example.app", "--edit-id", "edit-1", "--track", "production")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"name":"production"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestTracksUpdate_RejectsInvalidVersionCodes(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runTracks(
		t,
		deps,
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--track", "production",
		"--status", "completed",
		"--version-codes", "x",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid version code") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracksUpdate_ReturnsUpdatedStatus(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				update: gpc.TrackInfo{Name: "internal"},
			}, nil
		},
	}

	out, err := runTracks(
		t,
		deps,
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--track", "internal",
		"--status", "completed",
		"--version-codes", "1002003,1002004",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"name":"internal"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestTracksUpdate_ParsesReleaseNotesFile(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateFn: func(_, _, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
					if trackName != "internal" {
						t.Fatalf("unexpected track name: %q", trackName)
					}
					if len(update.ReleaseNotes) != 2 {
						t.Fatalf("expected two release notes, got %+v", update.ReleaseNotes)
					}
					if update.ReleaseNotes[0].Language != "en-US" || update.ReleaseNotes[1].Language != "pl-PL" {
						t.Fatalf("expected deterministic locale order, got %+v", update.ReleaseNotes)
					}
					return gpc.TrackInfo{Name: "internal"}, nil
				},
			}, nil
		},
	}

	out, err := runTracks(
		t,
		deps,
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--track", "internal",
		"--status", "completed",
		"--version-codes", "1002003",
		"--release-notes-file", writeReleaseNotesFile(t, `{"pl-PL":"Notatki wydania","en-US":"Release notes"}`),
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestTracksUpdate_RejectsInvalidReleaseNotesFile(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runTracks(
		t,
		deps,
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--track", "internal",
		"--status", "completed",
		"--version-codes", "1002003",
		"--release-notes-file", writeReleaseNotesFile(t, `{"en-US":{"text":"invalid"}}`),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--release-notes-file must be either a JSON object or array") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracksUpdate_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateErr: errors.New("conflict"),
			}, nil
		},
	}

	_, err := runTracks(
		t,
		deps,
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--track", "internal",
		"--status", "completed",
		"--version-codes", "1002003",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to update track") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracksPromote_RequiresFromAndToTracks(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runTracks(t, deps, "promote", "--package-name", "com.example.app", "--edit-id", "edit-1", "--to-track", "production")
	if err == nil || !strings.Contains(err.Error(), "--from-track is required") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = runTracks(t, deps, "promote", "--package-name", "com.example.app", "--edit-id", "edit-1", "--from-track", "internal")
	if err == nil || !strings.Contains(err.Error(), "--to-track is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracksPromote_RequiresDifferentTracks(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runTracks(t, deps, "promote", "--package-name", "com.example.app", "--edit-id", "edit-1", "--from-track", "internal", "--to-track", "internal")
	if err == nil || !strings.Contains(err.Error(), "must be different") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracksPromote_FailsWhenSourceTrackHasNoReleases(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				getByName: map[string]gpc.TrackInfo{
					"internal": {Name: "internal"},
				},
			}, nil
		},
	}

	_, err := runTracks(t, deps, "promote", "--package-name", "com.example.app", "--edit-id", "edit-1", "--from-track", "internal", "--to-track", "production")
	if err == nil || !strings.Contains(err.Error(), "has no releases") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracksPromote_FailsWhenReadingSourceTrackFails(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				getErrMap: map[string]error{
					"internal": errors.New("forbidden"),
				},
			}, nil
		},
	}

	_, err := runTracks(t, deps, "promote", "--package-name", "com.example.app", "--edit-id", "edit-1", "--from-track", "internal", "--to-track", "production")
	if err == nil || !strings.Contains(err.Error(), "failed to read source track") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracksPromote_FailsWhenUpdateFails(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				getByName: map[string]gpc.TrackInfo{
					"internal": {
						Name: "internal",
						Releases: []gpc.TrackReleaseInfo{
							{
								Status:       "draft",
								VersionCodes: []int64{1},
							},
						},
					},
				},
				updateErrMap: map[string]error{
					"production": errors.New("conflict"),
				},
			}, nil
		},
	}

	_, err := runTracks(t, deps, "promote", "--package-name", "com.example.app", "--edit-id", "edit-1", "--from-track", "internal", "--to-track", "production")
	if err == nil || !strings.Contains(err.Error(), "failed to promote track") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracksPromote_ReturnsPromotedStatus(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				getByName: map[string]gpc.TrackInfo{
					"internal": {
						Name: "internal",
						Releases: []gpc.TrackReleaseInfo{
							{
								Name:           "1 (1.0)",
								Status:         "draft",
								VersionCodes:   []int64{1},
								UpdatePriority: 2,
							},
						},
					},
				},
				updateFn: func(_, _, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
					if trackName != "production" {
						t.Fatalf("expected promote target to be production, got %q", trackName)
					}
					if update.Status != "completed" {
						t.Fatalf("expected overridden status completed, got %q", update.Status)
					}
					if update.ReleaseName != "Promoted 1.0" {
						t.Fatalf("expected overridden release name, got %q", update.ReleaseName)
					}
					if len(update.VersionCodes) != 1 || update.VersionCodes[0] != 1 {
						t.Fatalf("unexpected version codes: %+v", update.VersionCodes)
					}
					return gpc.TrackInfo{Name: "production"}, nil
				},
			}, nil
		},
	}

	out, err := runTracks(
		t,
		deps,
		"promote",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--from-track", "internal",
		"--to-track", "production",
		"--status", "completed",
		"--release-name", "Promoted 1.0",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"promoted"`) || !strings.Contains(out, `"toTrack":"production"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}
