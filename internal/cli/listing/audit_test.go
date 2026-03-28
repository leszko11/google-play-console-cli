package listing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListingAuditWarnsAndStrictFails(t *testing.T) {
	root := t.TempDir()
	listingDir := filepath.Join(root, "listing")
	screenshotsDir := filepath.Join(root, "screenshots", "de-DE")
	for _, dir := range []string{
		filepath.Join(listingDir, "en-US"),
		screenshotsDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for path, contents := range map[string]string{
		filepath.Join(listingDir, "en-US", "title.txt"):             "Example",
		filepath.Join(listingDir, "en-US", "short-description.txt"): "Example",
		filepath.Join(listingDir, "en-US", "full-description.txt"):  "Too short for a convincing listing body.",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	out, err := runCommand(t, Deps{}, "audit", "--dir", listingDir, "--screenshots-dir", filepath.Join(root, "screenshots"), "--default-locale", "en-US", "--output", "json", "--strict")
	if err == nil || !strings.Contains(err.Error(), "listing audit reported warn findings") {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"status":"warn"`, `title is duplicated in short description`, `default locale screenshot coverage is missing`, `screenshots locale exists without listing files`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestListingAuditReportsErrors(t *testing.T) {
	root := t.TempDir()
	listingDir := filepath.Join(root, "listing", "en-US")
	if err := os.MkdirAll(listingDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(listingDir, "title.txt"), []byte(strings.Repeat("A", 31)), 0o600); err != nil {
		t.Fatalf("write title: %v", err)
	}
	if err := os.WriteFile(filepath.Join(listingDir, "short-description.txt"), []byte("Short enough"), 0o600); err != nil {
		t.Fatalf("write short description: %v", err)
	}

	out, err := runCommand(t, Deps{}, "audit", "--dir", filepath.Join(root, "listing"), "--default-locale", "en-US", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"status":"error"`, `title too long`, `missing required file full-description.txt`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}
