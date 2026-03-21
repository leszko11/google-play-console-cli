package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectFromDirFindsNearestConfig(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "workspace", "mobile")
	nested := filepath.Join(project, "android", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(project, projectConfigName)
	raw := []byte("package-name: com.example.app\nprofile: work\nlisting-dir: ./listing\nscreenshots-dir: ./screenshots\nproducts-dir: ./products\nsubscriptions-dir: ./subscriptions\nandroid-project-dir: ./android\nartifact-path: ./android/app.aab\nnotes-file: ./play/notes.txt\n")
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	info, err := LoadProjectFromDir(nested)
	if err != nil {
		t.Fatal(err)
	}
	if info.Path != cfgPath {
		t.Fatalf("path = %q, want %q", info.Path, cfgPath)
	}
	if info.Config.PackageName != "com.example.app" {
		t.Fatalf("package = %q", info.Config.PackageName)
	}
	if want := filepath.Join(project, "listing"); info.Config.ListingDir != want {
		t.Fatalf("listing dir = %q, want %q", info.Config.ListingDir, want)
	}
	if want := filepath.Join(project, "screenshots"); info.Config.ScreenshotsDir != want {
		t.Fatalf("screenshots dir = %q, want %q", info.Config.ScreenshotsDir, want)
	}
	if want := filepath.Join(project, "products"); info.Config.ProductsDir != want {
		t.Fatalf("products dir = %q, want %q", info.Config.ProductsDir, want)
	}
	if want := filepath.Join(project, "subscriptions"); info.Config.SubscriptionsDir != want {
		t.Fatalf("subscriptions dir = %q, want %q", info.Config.SubscriptionsDir, want)
	}
	if want := filepath.Join(project, "android"); info.Config.AndroidProjectDir != want {
		t.Fatalf("android project dir = %q, want %q", info.Config.AndroidProjectDir, want)
	}
	if want := filepath.Join(project, "android", "app.aab"); info.Config.ArtifactPath != want {
		t.Fatalf("artifact path = %q, want %q", info.Config.ArtifactPath, want)
	}
	if want := filepath.Join(project, "play", "notes.txt"); info.Config.NotesFile != want {
		t.Fatalf("notes file = %q, want %q", info.Config.NotesFile, want)
	}
}

func TestLoadProjectFromDirMissingConfig(t *testing.T) {
	info, err := LoadProjectFromDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if info.Path != "" {
		t.Fatalf("unexpected path %q", info.Path)
	}
}
