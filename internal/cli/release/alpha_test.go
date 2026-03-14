package release

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func TestRunAlphaSuccess(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")
	aabPath := filepath.Join(projectDir, "app", "build", "outputs", "bundle", "stagingRelease", "app-staging-release.aab")
	writeFakeAAB(t, aabPath, true)

	client := &fakeReleaseClient{
		createEditIDs: []string{"edit-1", "edit-2"},
		uploadBundle:  gpc.BundleInfo{VersionCode: 456},
		getTrackInfo: gpc.TrackInfo{
			Name: "alpha",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "completed", VersionCodes: []int64{456}},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	result, err := runAlpha(context.Background(), deps, alphaOptions{
		PackageName:      "com.example.app",
		Track:            "alpha",
		ReleaseStatus:    "completed",
		ProjectDir:       projectDir,
		BuildTask:        defaultBuildTask,
		AABPath:          aabPath,
		SkipBuild:        true,
		Confirm:          true,
		CleanupOnFailure: true,
		VersionCode:      456,
		NotesMode:        "none",
		UpdatePriority:   3,
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v result=%+v", err, result)
	}
	if result.Status != "committed" || !result.Committed {
		t.Fatalf("expected committed status, got %+v", result)
	}
	if result.UploadedVersionCode != 456 {
		t.Fatalf("unexpected uploaded version code: %+v", result)
	}
	if client.lastTrack.UpdatePriority != 3 || client.lastTrackName != "alpha" {
		t.Fatalf("unexpected track update payload: %+v", client.lastTrack)
	}
	assertContainsStep(t, result.Steps, "preflight_verify", "ok")
	assertContainsStep(t, result.Steps, "post_deploy_verify", "ok")
}

func TestRunAlphaParsesTaggedReleaseNotesFile(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")
	aabPath := filepath.Join(projectDir, "app", "build", "outputs", "bundle", "stagingRelease", "app-staging-release.aab")
	writeFakeAAB(t, aabPath, true)
	notesPath := filepath.Join(projectDir, "release-notes.txt")
	mustWriteFile(t, notesPath, `<en-US>
Bug fixes and stability improvements.
</en-US>
<pl-PL>
Poprawki bledow i ulepszenia stabilnosci.
</pl-PL>`)

	client := &fakeReleaseClient{
		createEditIDs: []string{"edit-1", "edit-2"},
		uploadBundle:  gpc.BundleInfo{VersionCode: 456},
		getTrackInfo: gpc.TrackInfo{
			Name: "alpha",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "completed", VersionCodes: []int64{456}},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	result, err := runAlpha(context.Background(), deps, alphaOptions{
		PackageName:      "com.example.app",
		Track:            "alpha",
		ReleaseStatus:    "completed",
		ProjectDir:       projectDir,
		BuildTask:        defaultBuildTask,
		AABPath:          aabPath,
		SkipBuild:        true,
		Confirm:          true,
		CleanupOnFailure: true,
		VersionCode:      456,
		NotesMode:        "file",
		NotesFile:        notesPath,
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v result=%+v", err, result)
	}
	if len(client.lastTrack.ReleaseNotes) != 2 {
		t.Fatalf("expected 2 release notes, got %+v", client.lastTrack.ReleaseNotes)
	}
	if client.lastTrack.ReleaseNotes[0].Language != "en-US" || client.lastTrack.ReleaseNotes[1].Language != "pl-PL" {
		t.Fatalf("unexpected release note payload: %+v", client.lastTrack.ReleaseNotes)
	}
}

func TestRunAlphaBuildFailure(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	deps.RunCommand = func(_ context.Context, _ string, name string, args ...string) (string, error) {
		switch {
		case name == "java":
			return `openjdk version "21.0.10"`, nil
		case name == "./gradlew" && strings.Join(args, " ") == ":app:tasks --all":
			return "bundleStagingRelease", nil
		case name == "env":
			return "", errors.New("gradle failed")
		default:
			return "", nil
		}
	}

	result, err := runAlpha(context.Background(), deps, alphaOptions{
		PackageName:      "com.example.app",
		Track:            "alpha",
		ReleaseStatus:    "completed",
		ProjectDir:       projectDir,
		BuildTask:        defaultBuildTask,
		SkipBuild:        false,
		Confirm:          true,
		CleanupOnFailure: true,
		VersionCode:      123,
		NotesMode:        "none",
	})
	if err == nil || !strings.Contains(err.Error(), "Gradle build failed") {
		t.Fatalf("expected build failure, got err=%v result=%+v", err, result)
	}
	assertContainsStep(t, result.Steps, "build_artifact", "error")
}

func TestRunAlphaDeployFailureCleansUp(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")
	aabPath := filepath.Join(projectDir, "artifact.aab")
	writeFakeAAB(t, aabPath, true)

	client := &fakeReleaseClient{
		createEditIDs:  []string{"edit-1"},
		uploadBundle:   gpc.BundleInfo{VersionCode: 111},
		updateTrackErr: errors.New("conflict"),
	}
	deps := baseReleaseDeps(t, client)

	result, err := runAlpha(context.Background(), deps, alphaOptions{
		PackageName:      "com.example.app",
		Track:            "alpha",
		ReleaseStatus:    "completed",
		ProjectDir:       projectDir,
		BuildTask:        defaultBuildTask,
		AABPath:          aabPath,
		SkipBuild:        true,
		Confirm:          true,
		CleanupOnFailure: true,
		VersionCode:      111,
		NotesMode:        "none",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to update track") {
		t.Fatalf("expected update_track error, got err=%v result=%+v", err, result)
	}
	if !result.CleanupPerformed {
		t.Fatalf("expected cleanup performed, got %+v", result)
	}
	assertContainsStep(t, result.Steps, "cleanup_delete_edit", "ok")
}

func TestRunAlphaDryRun(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")
	aabPath := filepath.Join(projectDir, "artifact.aab")
	writeFakeAAB(t, aabPath, true)

	client := &fakeReleaseClient{
		createEditIDs: []string{"edit-1"},
		uploadBundle:  gpc.BundleInfo{VersionCode: 222},
	}
	deps := baseReleaseDeps(t, client)

	result, err := runAlpha(context.Background(), deps, alphaOptions{
		PackageName:      "com.example.app",
		Track:            "alpha",
		ReleaseStatus:    "completed",
		ProjectDir:       projectDir,
		BuildTask:        defaultBuildTask,
		AABPath:          aabPath,
		SkipBuild:        true,
		DryRun:           true,
		CleanupOnFailure: true,
		VersionCode:      222,
		NotesMode:        "none",
	})
	if err != nil {
		t.Fatalf("expected dry-run success, got %v result=%+v", err, result)
	}
	if result.Status != "dry-run" || result.Committed {
		t.Fatalf("expected dry-run status, got %+v", result)
	}
	assertContainsStep(t, result.Steps, "delete_edit_dry_run", "ok")
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestRunAlphaPreflightVerifyUsesNotesTextAndAAB(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFile(t, filepath.Join(projectDir, "gradlew"), "#!/bin/bash\n")
	aabPath := writeFakeAAB(t, filepath.Join(projectDir, "artifact.aab"), true)

	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)

	result, err := runAlpha(context.Background(), deps, alphaOptions{
		PackageName:      "com.example.app",
		Track:            "alpha",
		ReleaseStatus:    "completed",
		ProjectDir:       projectDir,
		BuildTask:        defaultBuildTask,
		AABPath:          aabPath,
		SkipBuild:        true,
		Confirm:          true,
		CleanupOnFailure: true,
		VersionCode:      456,
		NotesMode:        "git",
		NotesText:        strings.Repeat("a", 501),
	})
	if err == nil || !strings.Contains(err.Error(), "release verification failed") {
		t.Fatalf("expected preflight verify failure, got err=%v result=%+v", err, result)
	}
	assertContainsStep(t, result.Steps, "preflight_verify", "error")
	if result.Verify == nil || result.Verify.Status != "failed" {
		t.Fatalf("expected embedded verify failure, got %+v", result.Verify)
	}
}
