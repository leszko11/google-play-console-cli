package listing

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkScanListingsDir(b *testing.B) {
	root := b.TempDir()
	for _, locale := range []string{"en-US", "fr-FR", "de-DE"} {
		localeDir := filepath.Join(root, locale)
		if err := os.MkdirAll(filepath.Join(localeDir, "images", "phoneScreenshots"), 0o755); err != nil {
			b.Fatalf("mkdir: %v", err)
		}
		for name, value := range map[string]string{
			"title.txt":             "Title",
			"short-description.txt": "Short description",
			"full-description.txt":  "Full description",
		} {
			if err := os.WriteFile(filepath.Join(localeDir, name), []byte(value), 0o600); err != nil {
				b.Fatalf("write %s: %v", name, err)
			}
		}
		if err := os.WriteFile(filepath.Join(localeDir, "images", "phoneScreenshots", "01.png"), []byte("img"), 0o600); err != nil {
			b.Fatalf("write image: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := scanListingsDir(root); err != nil {
			b.Fatalf("scan listings: %v", err)
		}
	}
}
