package shared

import (
	"os"
	"path/filepath"
	"testing"
)

type manifestFixture struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

func TestLoadManifestJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(`{"name":"demo","count":2}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	var got manifestFixture
	if err := LoadManifest(path, &got); err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if got.Name != "demo" || got.Count != 2 {
		t.Fatalf("unexpected manifest: %+v", got)
	}
}

func TestLoadManifestYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(path, []byte("name: demo\ncount: 3\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	var got manifestFixture
	if err := LoadManifest(path, &got); err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if got.Name != "demo" || got.Count != 3 {
		t.Fatalf("unexpected manifest: %+v", got)
	}
}

func TestLoadManifestRejectsUnsupportedExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.txt")
	if err := os.WriteFile(path, []byte("name=demo"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	var got manifestFixture
	if err := LoadManifest(path, &got); err == nil {
		t.Fatal("expected unsupported extension error")
	}
}
