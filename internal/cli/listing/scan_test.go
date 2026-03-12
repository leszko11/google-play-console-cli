package listing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestScanListingsDir(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "en-US", "title.txt"), "Title")
	writeFile(t, filepath.Join(root, "en-US", "short-description.txt"), "Short")
	writeFile(t, filepath.Join(root, "en-US", "full-description.txt"), "Full")
	writeFile(t, filepath.Join(root, "en-US", "images", "phoneScreenshots", "2.png"), "two")
	writeFile(t, filepath.Join(root, "en-US", "images", "phoneScreenshots", "1.png"), "one")
	writeFile(t, filepath.Join(root, "en-US", "images", "featureGraphic.png"), "feature")
	writeFile(t, filepath.Join(root, "ja-JP", "title.txt"), "タイトル")
	writeFile(t, filepath.Join(root, "ja-JP", "short-description.txt"), "短い説明")
	writeFile(t, filepath.Join(root, "ja-JP", "full-description.txt"), "詳しい説明")

	locales, err := scanListingsDir(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(locales) != 2 {
		t.Fatalf("expected 2 locales, got %d", len(locales))
	}
	if locales[0].Locale != "en-US" || locales[1].Locale != "ja-JP" {
		t.Fatalf("unexpected locale order: %+v", locales)
	}
	if locales[0].Listing.Title != "Title" || locales[0].Listing.ShortDescription != "Short" || locales[0].Listing.FullDescription != "Full" {
		t.Fatalf("unexpected listing payload: %+v", locales[0].Listing)
	}
	if got := locales[0].Images["phoneScreenshots"]; len(got) != 2 || !strings.HasSuffix(got[0], "1.png") || !strings.HasSuffix(got[1], "2.png") {
		t.Fatalf("unexpected screenshot paths: %+v", got)
	}
	if got := locales[0].Images["featureGraphic"]; len(got) != 1 || !strings.HasSuffix(got[0], "featureGraphic.png") {
		t.Fatalf("unexpected single-file image paths: %+v", got)
	}
}

func TestScanListingsDirRequiresAllTextFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "en-US", "title.txt"), "Title")
	writeFile(t, filepath.Join(root, "en-US", "short-description.txt"), "Short")

	_, err := scanListingsDir(root)
	if err == nil || !strings.Contains(err.Error(), "missing required file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanListingsDirRejectsUnknownImageDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "en-US", "title.txt"), "Title")
	writeFile(t, filepath.Join(root, "en-US", "short-description.txt"), "Short")
	writeFile(t, filepath.Join(root, "en-US", "full-description.txt"), "Full")
	writeFile(t, filepath.Join(root, "en-US", "images", "tabletScreenshots", "1.png"), "bad")

	_, err := scanListingsDir(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported image directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanListingsDirRejectsDuplicateSingleImageType(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "en-US", "title.txt"), "Title")
	writeFile(t, filepath.Join(root, "en-US", "short-description.txt"), "Short")
	writeFile(t, filepath.Join(root, "en-US", "full-description.txt"), "Full")
	writeFile(t, filepath.Join(root, "en-US", "images", "icon.png"), "one")
	writeFile(t, filepath.Join(root, "en-US", "images", "icon.jpg"), "two")

	_, err := scanListingsDir(root)
	if err == nil || !strings.Contains(err.Error(), `multiple files provided for image type "icon"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
