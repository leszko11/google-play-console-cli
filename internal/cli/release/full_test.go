package release

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func (f *fakeReleaseClient) UploadAPK(_ context.Context, _, _, _ string) (gpc.APKInfo, error) {
	if f.uploadBundleErr != nil {
		return gpc.APKInfo{}, f.uploadBundleErr
	}
	return gpc.APKInfo{VersionCode: 321}, nil
}

func (f *fakeReleaseClient) UploadDeobfuscationFile(_ context.Context, _, _ string, _ int64, _, _ string) (gpc.DeobfuscationFileInfo, error) {
	return gpc.DeobfuscationFileInfo{SymbolType: "proguard"}, nil
}

func writeReleaseAsset(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	return path
}

func writeReleaseManifest(t *testing.T, artifact, mapping string) string {
	t.Helper()
	contents := `artifact: ` + artifact + `
track: internal
status: completed
releaseName: "v2.1.0"
userFraction: 0.1
mappingFile: ` + mapping + `
mappingType: proguard
releaseNotes:
  en-US: "Bug fixes"
  ja-JP: "改善"
`
	path := filepath.Join(t.TempDir(), "release.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func runFullCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestReleaseFullCommitSuccess(t *testing.T) {
	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	artifact := writeReleaseAsset(t, "app.aab")
	mapping := writeReleaseAsset(t, "mapping.txt")
	manifest := writeReleaseManifest(t, artifact, mapping)

	out, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--manifest", manifest, "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"committed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.lastTrackName != "internal" || client.lastTrack.ReleaseName != "v2.1.0" {
		t.Fatalf("unexpected track update: %+v", client.lastTrack)
	}
	if len(client.lastTrack.ReleaseNotes) != 2 {
		t.Fatalf("unexpected notes: %+v", client.lastTrack.ReleaseNotes)
	}
}

func TestReleaseFullDryRunDeletesEdit(t *testing.T) {
	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	artifact := writeReleaseAsset(t, "app.aab")
	mapping := writeReleaseAsset(t, "mapping.txt")
	manifest := writeReleaseManifest(t, artifact, mapping)

	out, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--manifest", manifest, "--dry-run")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected delete call, got %d", client.deleteCalls)
	}
}

func TestReleaseFullRequiresAllowProduction(t *testing.T) {
	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	artifact := writeReleaseAsset(t, "app.aab")
	manifest := writeManifestFile(t, "release.yaml", "artifact: "+artifact+"\ntrack: production\nstatus: completed\n")

	_, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--manifest", manifest, "--confirm")
	if err == nil || !strings.Contains(err.Error(), "--allow-production is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseFullRejectsMissingManifest(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return config.Config{}, nil },
	}
	_, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--confirm")
	if err == nil || !strings.Contains(err.Error(), "--manifest is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
