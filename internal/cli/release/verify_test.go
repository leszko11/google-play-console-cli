package release

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVerifyMissingPackage(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, projectDir+"/gradlew", "#!/bin/bash\n")

	client := &fakeReleaseClient{
		verifyErr: mustPackageNotFoundErr(),
	}
	deps := baseReleaseDeps(t, client)

	result, err := runVerify(context.Background(), deps, verifyOptions{
		PackageName: "com.example.staging",
		Track:       "alpha",
		ProjectDir:  projectDir,
		BuildTask:   defaultBuildTask,
		NotesMode:   "none",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %+v", result)
	}
	if len(result.BlockingIssues) == 0 {
		t.Fatalf("expected blocking issues, got %+v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected bootstrap warning for package-not-found, got %+v", result)
	}
}

func TestRunVerifyRejectsOldJava(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, projectDir+"/gradlew", "#!/bin/bash\n")

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	deps.RunCommand = func(_ context.Context, _ string, name string, args ...string) (string, error) {
		switch {
		case name == "java":
			return `openjdk version "17.0.1"`, nil
		case name == "./gradlew" && strings.Join(args, " ") == ":app:tasks --all":
			return "bundleStagingRelease - Assembles bundle for variant stagingRelease", nil
		default:
			return "", nil
		}
	}

	result, err := runVerify(context.Background(), deps, verifyOptions{
		PackageName: "com.example.app",
		Track:       "alpha",
		ProjectDir:  projectDir,
		BuildTask:   defaultBuildTask,
		NotesMode:   "none",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status with old java, got %+v", result)
	}
	found := false
	for _, issue := range result.BlockingIssues {
		if strings.Contains(issue, "Java 21+") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Java 21 blocking issue, got %+v", result.BlockingIssues)
	}
}

func TestRunVerifySuccess(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, projectDir+"/gradlew", "#!/bin/bash\n")

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)

	result, err := runVerify(context.Background(), deps, verifyOptions{
		PackageName: "com.example.app",
		Track:       "alpha",
		ProjectDir:  projectDir,
		BuildTask:   defaultBuildTask,
		NotesMode:   "none",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok status, got %+v", result)
	}
}

func TestRunVerifyBlocksOnInvalidNotesFormat(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, projectDir+"/gradlew", "#!/bin/bash\n")
	mustWriteFile(t, projectDir+"/notes.txt", "<en-US>\nBroken notes without closing tag\n")

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)

	result, err := runVerify(context.Background(), deps, verifyOptions{
		PackageName: "com.example.app",
		Track:       "alpha",
		ProjectDir:  projectDir,
		BuildTask:   defaultBuildTask,
		NotesMode:   "file",
		NotesFile:   projectDir + "/notes.txt",
		NotesLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %+v", result)
	}
	found := false
	for _, issue := range result.BlockingIssues {
		if strings.Contains(issue, "missing closing tag") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected invalid notes format blocking issue, got %+v", result.BlockingIssues)
	}
}

func TestRunVerifyValidatesExplicitAAB(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")
	aabPath := writeFakeAAB(t, filepath.Join(projectDir, "artifact.aab"), true)

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)

	result, err := runVerify(context.Background(), deps, verifyOptions{
		PackageName: "com.example.app",
		Track:       "alpha",
		ProjectDir:  projectDir,
		BuildTask:   defaultBuildTask,
		AABPath:     aabPath,
		NotesMode:   "none",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok status, got %+v", result)
	}
	found := false
	for _, check := range result.Checks {
		if check.Name == "bundle_artifact" && check.Status == "ok" && strings.Contains(check.Detail, aabPath) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected bundle_artifact success check, got %+v", result.Checks)
	}
}

func TestRunVerifyBlocksOnUnsignedAAB(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")
	aabPath := writeFakeAAB(t, filepath.Join(projectDir, "artifact.aab"), false)

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)

	result, err := runVerify(context.Background(), deps, verifyOptions{
		PackageName: "com.example.app",
		Track:       "alpha",
		ProjectDir:  projectDir,
		BuildTask:   defaultBuildTask,
		AABPath:     aabPath,
		NotesMode:   "none",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %+v", result)
	}
	found := false
	for _, issue := range result.BlockingIssues {
		if strings.Contains(issue, "bundle artifact is not signed") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unsigned bundle blocking issue, got %+v", result.BlockingIssues)
	}
}

func TestRunVerifyBlocksOnInvalidListingMetadata(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")
	mustWriteFile(t, filepath.Join(projectDir, ".gpc.yaml"), "listing-dir: ./listing\n")
	mustWriteFile(t, filepath.Join(projectDir, "listing", "en-US", "title.txt"), "Example App")
	mustWriteFile(t, filepath.Join(projectDir, "listing", "en-US", "short-description.txt"), "Short")

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)

	result, err := runVerify(context.Background(), deps, verifyOptions{
		PackageName: "com.example.app",
		Track:       "alpha",
		ProjectDir:  projectDir,
		BuildTask:   defaultBuildTask,
		NotesMode:   "none",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %+v", result)
	}
	found := false
	for _, issue := range result.BlockingIssues {
		if strings.Contains(issue, "missing required file full-description.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected listing metadata issue, got %+v", result.BlockingIssues)
	}
}

func TestRunVerifyValidatesListingMetadataWhenConfigured(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")
	mustWriteFile(t, filepath.Join(projectDir, ".gpc.yaml"), "listing-dir: ./listing\n")
	mustWriteFile(t, filepath.Join(projectDir, "listing", "en-US", "title.txt"), "Example App")
	mustWriteFile(t, filepath.Join(projectDir, "listing", "en-US", "short-description.txt"), "Short summary")
	mustWriteFile(t, filepath.Join(projectDir, "listing", "en-US", "full-description.txt"), "Long description")

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)

	result, err := runVerify(context.Background(), deps, verifyOptions{
		PackageName: "com.example.app",
		Track:       "alpha",
		ProjectDir:  projectDir,
		BuildTask:   defaultBuildTask,
		NotesMode:   "none",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok status, got %+v", result)
	}
	found := false
	for _, check := range result.Checks {
		if check.Name == "listing_metadata" && check.Status == "ok" && strings.Contains(check.Detail, "locales=1") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected listing metadata success, got %+v", result.Checks)
	}
}

func TestRunVerifyBlocksOnTooLongInlineNotes(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)

	result, err := runVerify(context.Background(), deps, verifyOptions{
		PackageName: "com.example.app",
		Track:       "alpha",
		ProjectDir:  projectDir,
		BuildTask:   defaultBuildTask,
		NotesMode:   "git",
		NotesText:   strings.Repeat("a", 501),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %+v", result)
	}
	found := false
	for _, issue := range result.BlockingIssues {
		if strings.Contains(issue, "release notes too long") && strings.Contains(issue, "en-US") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected release notes length blocking issue, got %+v", result.BlockingIssues)
	}
}
