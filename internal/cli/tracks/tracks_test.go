package tracks

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	list      []gpc.TrackInfo
	listErr   error
	get       gpc.TrackInfo
	getErr    error
	update    gpc.TrackInfo
	updateErr error
}

func (f fakeClient) ListTracks(_ context.Context, _, _ string) ([]gpc.TrackInfo, error) {
	return f.list, f.listErr
}

func (f fakeClient) GetTrack(_ context.Context, _, _, _ string) (gpc.TrackInfo, error) {
	return f.get, f.getErr
}

func (f fakeClient) UpdateTrack(_ context.Context, _, _, _ string, _ gpc.TrackUpdate) (gpc.TrackInfo, error) {
	return f.update, f.updateErr
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
