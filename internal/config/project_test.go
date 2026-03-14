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
	raw := []byte("package-name: com.example.app\nprofile: work\nlisting-dir: ./listing\n")
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
