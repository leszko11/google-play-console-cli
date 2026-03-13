package release

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkLoadFullManifest(b *testing.B) {
	root := b.TempDir()
	artifactPath := filepath.Join(root, "app.aab")
	mappingPath := filepath.Join(root, "mapping.txt")
	manifestPath := filepath.Join(root, "release.yaml")
	if err := os.WriteFile(artifactPath, []byte("bundle"), 0o600); err != nil {
		b.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(mappingPath, []byte("mapping"), 0o600); err != nil {
		b.Fatalf("write mapping: %v", err)
	}
	manifest := []byte(`
track: internal
releaseName: "1.2.3"
artifact: ` + artifactPath + `
status: completed
releaseNotes:
  en-US: Hello
mappingFile: ` + mappingPath + `
`)
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		b.Fatalf("write manifest: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := loadFullManifest(manifestPath); err != nil {
			b.Fatalf("load manifest: %v", err)
		}
	}
}
