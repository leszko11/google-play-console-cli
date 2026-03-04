package release

import (
	"context"
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
