package release

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifestFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

// jsonEscapePath returns a path with backslashes escaped for embedding in JSON strings.
func jsonEscapePath(s string) string {
	b, _ := json.Marshal(s)
	// Strip surrounding quotes from the marshalled string.
	return string(b[1 : len(b)-1])
}

func writeAssetFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	return path
}

func TestLoadFullManifestYAML(t *testing.T) {
	artifact := writeAssetFile(t, "app.aab")
	mapping := writeAssetFile(t, "mapping.txt")
	path := writeManifestFile(t, "release.yaml", "artifact: "+artifact+"\ntrack: internal\nstatus: completed\nreleaseName: v1\nuserFraction: 0.1\nmappingFile: "+mapping+"\nmappingType: nativeCode\nreleaseNotes:\n  en-US: Hello\n")

	got, err := loadFullManifest(path)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if got.ArtifactPath != artifact || got.ArtifactType != artifactTypeAAB {
		t.Fatalf("unexpected artifact parsing: %+v", got)
	}
	if got.MappingFile != mapping || got.MappingType != mappingTypeNativeCode {
		t.Fatalf("unexpected mapping parsing: %+v", got)
	}
	if len(got.ReleaseNotes) != 1 || got.ReleaseNotes[0].Language != "en-US" || got.ReleaseNotes[0].Text != "Hello" {
		t.Fatalf("unexpected release notes: %+v", got.ReleaseNotes)
	}
}

func TestLoadFullManifestRejectsUnsupportedArtifact(t *testing.T) {
	artifact := writeAssetFile(t, "app.zip")
	path := writeManifestFile(t, "release.json", `{"artifact":"`+jsonEscapePath(artifact)+`","track":"internal","status":"completed"}`)

	_, err := loadFullManifest(path)
	if err == nil || !strings.Contains(err.Error(), "artifact must end with .aab or .apk") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFullManifestRejectsMissingTrack(t *testing.T) {
	artifact := writeAssetFile(t, "app.aab")
	path := writeManifestFile(t, "release.json", `{"artifact":"`+jsonEscapePath(artifact)+`","status":"completed"}`)

	_, err := loadFullManifest(path)
	if err == nil || !strings.Contains(err.Error(), "--track is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
