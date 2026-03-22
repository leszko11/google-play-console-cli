package screenshots

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	createEditErr error
	validateErr   error
	deleteErr     error
	deleteCalls   int
	uploadCalls   int
	commitCalls   []bool
	commitErrs    []error
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	if f.createEditErr != nil {
		return gpc.EditInfo{}, f.createEditErr
	}
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return f.deleteErr
}

func (f *fakeClient) ValidateEdit(_ context.Context, _, _ string) error {
	return f.validateErr
}

func (f *fakeClient) CommitEdit(_ context.Context, _, _ string, changesNotSentForReview bool) (gpc.EditInfo, error) {
	f.commitCalls = append(f.commitCalls, changesNotSentForReview)
	if len(f.commitErrs) > 0 {
		err := f.commitErrs[0]
		f.commitErrs = f.commitErrs[1:]
		if err != nil {
			return gpc.EditInfo{}, err
		}
	}
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) DeleteAllImages(_ context.Context, _, _, _, _ string) ([]gpc.ImageInfo, error) {
	return []gpc.ImageInfo{{ID: "old-1"}}, nil
}

func (f *fakeClient) UploadImage(_ context.Context, _, _, _, _, _ string) (gpc.ImageInfo, error) {
	f.uploadCalls++
	return gpc.ImageInfo{ID: "img"}, nil
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

func TestScanScreenshotsDir_SimpleLayout(t *testing.T) {
	root := t.TempDir()
	writePNGFile(t, filepath.Join(root, "en-US", "phone", "02.png"), 320, 320)
	writePNGFile(t, filepath.Join(root, "en-US", "phone", "01.png"), 320, 320)

	locales, err := scanScreenshotsDir(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(locales) != 1 {
		t.Fatalf("expected one locale, got %d", len(locales))
	}
	got := locales[0].Images["phoneScreenshots"]
	if len(got) != 2 || !strings.HasSuffix(got[0], "01.png") || !strings.HasSuffix(got[1], "02.png") {
		t.Fatalf("unexpected files: %+v", got)
	}
}

func TestScanScreenshotsDir_ListingLayout(t *testing.T) {
	root := t.TempDir()
	writePNGFile(t, filepath.Join(root, "pl-PL", "images", "phoneScreenshots", "01.png"), 320, 320)

	locales, err := scanScreenshotsDir(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(locales) != 1 || len(locales[0].Images["phoneScreenshots"]) != 1 {
		t.Fatalf("unexpected locales: %+v", locales)
	}
}

func TestNormalizeScreenshotDirName_Alias(t *testing.T) {
	got, err := normalizeScreenshotDirName("seven-inch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sevenInchScreenshots" {
		t.Fatalf("unexpected image type: %q", got)
	}
}

func TestScreenshotsSyncDryRun(t *testing.T) {
	root := t.TempDir()
	writePNGFile(t, filepath.Join(root, "en-US", "phone", "01.png"), 320, 320)

	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--dir", root, "--dry-run")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.uploadCalls != 0 || len(client.commitCalls) != 0 {
		t.Fatalf("dry-run should not mutate: uploads=%d commits=%d", client.uploadCalls, len(client.commitCalls))
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected one cleanup delete, got %d", client.deleteCalls)
	}
}

func TestScreenshotsSyncCommitRetry(t *testing.T) {
	root := t.TempDir()
	writePNGFile(t, filepath.Join(root, "en-US", "phone", "01.png"), 320, 320)

	client := &fakeClient{
		commitErrs: []error{
			errors.New("androidpublisher api error (400): Changes cannot be sent for review automatically. Please set the query parameter changesNotSentForReview to true."),
			nil,
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--dir", root, "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if len(client.commitCalls) != 2 || client.commitCalls[0] || !client.commitCalls[1] {
		t.Fatalf("unexpected commit calls: %+v", client.commitCalls)
	}
	for _, want := range []string{`"status":"committed"`, `"commitRetried":true`, `"changesNotSentForReview":true`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestScreenshotsSyncDraftTrackConflictHint(t *testing.T) {
	root := t.TempDir()
	writePNGFile(t, filepath.Join(root, "en-US", "phone", "01.png"), 320, 320)

	client := &fakeClient{
		commitErrs: []error{
			errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--dir", root, "--confirm")
	if err == nil || !strings.Contains(err.Error(), `track release with status "completed"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScreenshotsSyncDryRunDraftTrackConflictHint(t *testing.T) {
	root := t.TempDir()
	writePNGFile(t, filepath.Join(root, "en-US", "phone", "01.png"), 320, 320)

	client := &fakeClient{
		validateErr: errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runCommand(t, deps, "sync", "--package-name", "com.example.app", "--dir", root, "--dry-run")
	if err == nil || !strings.Contains(err.Error(), "draft bootstrap state") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writePNGFile(t *testing.T, path string, width, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode image: %v", err)
	}
}
