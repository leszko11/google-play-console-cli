package drift

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	listings []gpc.ListingInfo
	images   map[string][]gpc.ImageInfo
	track    gpc.TrackInfo

	products    gpc.OneTimeProductsListInfo
	productsErr error

	subscriptions    gpc.SubscriptionsListInfo
	subscriptionsErr error

	createEditErr error
	getTrackErr   error
	deleteCalls   int
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	if f.createEditErr != nil {
		return gpc.EditInfo{}, f.createEditErr
	}
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return nil
}

func (f *fakeClient) ListListings(_ context.Context, _, _ string) ([]gpc.ListingInfo, error) {
	return append([]gpc.ListingInfo(nil), f.listings...), nil
}

func (f *fakeClient) ListImages(_ context.Context, _, _, language, imageType string) ([]gpc.ImageInfo, error) {
	return append([]gpc.ImageInfo(nil), f.images[language+"/"+imageType]...), nil
}

func (f *fakeClient) GetTrack(_ context.Context, _, _, _ string) (gpc.TrackInfo, error) {
	if f.getTrackErr != nil {
		return gpc.TrackInfo{}, f.getTrackErr
	}
	if f.track.Name == "" {
		return gpc.TrackInfo{
			Name: "internal",
			Releases: []gpc.TrackReleaseInfo{{
				Status:       "completed",
				VersionCodes: []int64{123},
				ReleaseNotes: []gpc.LocalizedText{{Language: "en-US", Text: "Hello"}},
			}},
		}, nil
	}
	return f.track, nil
}

func (f *fakeClient) ListOneTimeProducts(_ context.Context, _ string, _ int64, _ string, _ bool) (gpc.OneTimeProductsListInfo, error) {
	if f.productsErr != nil {
		return gpc.OneTimeProductsListInfo{}, f.productsErr
	}
	return f.products, nil
}

func (f *fakeClient) ListSubscriptions(_ context.Context, _ string, _ int64, _ string, _ bool) (gpc.SubscriptionsListInfo, error) {
	if f.subscriptionsErr != nil {
		return gpc.SubscriptionsListInfo{}, f.subscriptionsErr
	}
	return f.subscriptions, nil
}

func runCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), append([]string{"report"}, args...))
	return out.String(), err
}

func seedAuth(t *testing.T, deps *Deps, root string) {
	t.Helper()
	saPath := filepath.Join(root, "service-account.json")
	if err := os.WriteFile(saPath, []byte(`{"type":"service_account"}`), 0o600); err != nil {
		t.Fatalf("write service account: %v", err)
	}
	deps.LoadConfig = func() (config.Config, error) {
		return config.Config{
			ActiveProfile: "default",
			Profiles: map[string]config.Profile{
				"default": {ServiceAccountPath: saPath, Storage: config.StoragePath},
			},
		}, nil
	}
	deps.LookupEnv = func(key string) string {
		if key == "GPC_BYPASS_KEYCHAIN" {
			return "1"
		}
		return ""
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeJSONFile(t *testing.T, path string, payload any) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	writeFile(t, path, string(raw))
}

func TestReportRejectsTrackIncludeWithoutDraft(t *testing.T) {
	_, err := runCommand(t, Deps{}, "--package-name", "com.example.app", "--track", "internal", "--include", "track")
	if err == nil || !strings.Contains(err.Error(), "track drift requires desired release flags") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shared.IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}
}

func TestReportSkipsMissingDirsByDefault(t *testing.T) {
	root := t.TempDir()
	deps := Deps{
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}
	seedAuth(t, &deps, root)

	out, err := runCommand(t, deps, "--package-name", "com.example.app", "--track", "internal", "--dir", filepath.Join(root, "play"), "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		`"status":"ok"`,
		`{"name":"listing","status":"skipped"`,
		`{"name":"screenshots","status":"skipped"`,
		`{"name":"changelog","status":"skipped"`,
		`{"name":"products","status":"skipped"`,
		`{"name":"subscriptions","status":"skipped"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestReportExplicitMissingDirErrorsButOtherSurfacesContinue(t *testing.T) {
	root := t.TempDir()
	play := filepath.Join(root, "play")
	writeJSONFile(t, filepath.Join(play, "products", "coins_100.json"), map[string]string{"productId": "coins_100"})

	deps := Deps{
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}
	seedAuth(t, &deps, root)

	out, err := runCommand(t, deps,
		"--package-name", "com.example.app",
		"--track", "internal",
		"--dir", play,
		"--include", "listing,products",
		"--output", "json",
	)
	if err == nil || !strings.Contains(err.Error(), "surface errors") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"name":"listing","status":"error"`) || !strings.Contains(out, `"name":"products","status":"diff"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReportSurfaceFailureIsolation(t *testing.T) {
	root := t.TempDir()
	play := filepath.Join(root, "play")
	writeJSONFile(t, filepath.Join(play, "products", "coins_100.json"), map[string]string{"productId": "coins_100"})
	writeJSONFile(t, filepath.Join(play, "subscriptions", "premium_monthly.json"), map[string]string{"productId": "premium_monthly"})

	deps := Deps{
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{
				productsErr: errors.New("permission denied"),
			}, nil
		},
	}
	seedAuth(t, &deps, root)

	out, err := runCommand(t, deps,
		"--package-name", "com.example.app",
		"--track", "internal",
		"--dir", play,
		"--include", "products,subscriptions",
		"--output", "json",
	)
	if err == nil || !strings.Contains(err.Error(), "surface errors") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"name":"products","status":"error"`) || !strings.Contains(out, `"name":"subscriptions","status":"diff"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestWriteReportFormats(t *testing.T) {
	result := reportResult{
		PackageName: "com.example.app",
		Track:       "internal",
		Status:      "diff",
		HasDiff:     true,
		Surfaces: []surfaceResult{
			{Name: "listing", Status: "diff", HasDiff: true, ChangeCount: 2, Dir: "./play/listing"},
		},
	}

	for _, tc := range []struct {
		format string
		want   string
	}{
		{format: "table", want: "SURFACE\tSTATUS\tHAS_DIFF\tCHANGE_COUNT\tDIR\tDETAIL"},
		{format: "markdown", want: "| surface | status | hasDiff | changeCount | dir | detail |"},
		{format: "yaml", want: "packageName: com.example.app"},
	} {
		var out bytes.Buffer
		if err := writeReport(&out, tc.format, result); err != nil {
			t.Fatalf("%s: %v", tc.format, err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("%s output missing %q: %s", tc.format, tc.want, out.String())
		}
	}
}
