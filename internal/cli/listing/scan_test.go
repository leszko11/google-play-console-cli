package listing

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
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

func TestValidateListingsDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "en-US", "title.txt"), "Title")
	writeFile(t, filepath.Join(root, "en-US", "short-description.txt"), "Short")
	writeFile(t, filepath.Join(root, "en-US", "full-description.txt"), "Full")
	writePNGImage(t, filepath.Join(root, "en-US", "images", "icon.png"), 512, 512)
	writeJPEGImage(t, filepath.Join(root, "en-US", "images", "phoneScreenshots", "1.jpg"), 1080, 1920)

	summary, err := ValidateListingsDir(root)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if summary.LocaleCount != 1 || summary.ImageCount != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestValidateListingsDirRejectsInvalidImageDimensions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "en-US", "title.txt"), "Title")
	writeFile(t, filepath.Join(root, "en-US", "short-description.txt"), "Short")
	writeFile(t, filepath.Join(root, "en-US", "full-description.txt"), "Full")
	writePNGImage(t, filepath.Join(root, "en-US", "images", "icon.png"), 256, 256)

	_, err := ValidateListingsDir(root)
	if err == nil || !strings.Contains(err.Error(), "icon requires dimensions 512x512") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writePNGImage(t *testing.T, path string, width, height int) {
	t.Helper()
	writeImage(t, path, width, height, func(file *os.File, img image.Image) error {
		return png.Encode(file, img)
	})
}

func writeJPEGImage(t *testing.T, path string, width, height int) {
	t.Helper()
	writeImage(t, path, width, height, func(file *os.File, img image.Image) error {
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	})
}

func writeImage(t *testing.T, path string, width, height int, encode func(file *os.File, img image.Image) error) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}
	defer file.Close()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff})
		}
	}
	if err := encode(file, img); err != nil {
		t.Fatalf("encode image: %v", err)
	}
}
