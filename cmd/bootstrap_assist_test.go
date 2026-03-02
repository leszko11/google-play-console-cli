package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGradleTaskFor(t *testing.T) {
	t.Run("aab", func(t *testing.T) {
		got, err := gradleTaskFor("aab", "app", "release")
		if err != nil {
			t.Fatalf("gradleTaskFor returned error: %v", err)
		}
		if got != ":app:bundleRelease" {
			t.Fatalf("unexpected task: %s", got)
		}
	})

	t.Run("apk with separators", func(t *testing.T) {
		got, err := gradleTaskFor("apk", ":mobile", "prod-release")
		if err != nil {
			t.Fatalf("gradleTaskFor returned error: %v", err)
		}
		if got != ":mobile:assembleProdRelease" {
			t.Fatalf("unexpected task: %s", got)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		if _, err := gradleTaskFor("ipa", "app", "release"); err == nil {
			t.Fatal("expected error for unsupported artifact type")
		}
	})
}

func TestDetectAndroidGradleProject(t *testing.T) {
	tmp := t.TempDir()
	androidDir := filepath.Join(tmp, "android")
	if err := os.MkdirAll(androidDir, 0o755); err != nil {
		t.Fatalf("mkdir android: %v", err)
	}

	rootWrapper := filepath.Join(tmp, "gradlew")
	if err := os.WriteFile(rootWrapper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write root gradlew: %v", err)
	}
	if _, _, ok := detectAndroidGradleProject(tmp); !ok {
		t.Fatal("expected root gradlew to be detected")
	}

	if err := os.Remove(rootWrapper); err != nil {
		t.Fatalf("remove root gradlew: %v", err)
	}
	androidWrapper := filepath.Join(androidDir, "gradlew")
	if err := os.WriteFile(androidWrapper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write android gradlew: %v", err)
	}
	projectDir, _, ok := detectAndroidGradleProject(tmp)
	if !ok {
		t.Fatal("expected android/gradlew to be detected")
	}
	if projectDir != androidDir {
		t.Fatalf("expected projectDir %q, got %q", androidDir, projectDir)
	}
}

func TestFindBuiltArtifact(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "app", "build", "outputs", "bundle", "release")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("mkdir bundle output: %v", err)
	}

	first := filepath.Join(base, "app-release.aab")
	second := filepath.Join(base, "app-release-v2.aab")
	if err := os.WriteFile(first, []byte("aab-1"), 0o644); err != nil {
		t.Fatalf("write first artifact: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(second, []byte("aab-2"), 0o644); err != nil {
		t.Fatalf("write second artifact: %v", err)
	}

	got, err := findBuiltArtifact(tmp, "app", "release", "aab")
	if err != nil {
		t.Fatalf("findBuiltArtifact returned error: %v", err)
	}
	if got != second {
		t.Fatalf("expected latest artifact %q, got %q", second, got)
	}
}
