package release

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func runRehearseCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), append([]string{"rehearse"}, args...))
	return out.String(), err
}

func writeRehearseWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	if err := os.WriteFile(filepath.Join(root, "gradlew"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write gradlew: %v", err)
	}
	artifact := writeFakeAAB(t, filepath.Join(root, "app.aab"), true)
	mapping := writeReleaseAsset(t, "mapping.txt")
	manifest := writeReleaseManifest(t, artifact, mapping)
	return root, manifest
}

func snapshotFiles(t *testing.T, root string) []string {
	t.Helper()
	paths := []string{}
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, rel)
		return nil
	}); err != nil {
		t.Fatalf("walk files: %v", err)
	}
	sort.Strings(paths)
	return paths
}

func TestReleaseRehearseReady(t *testing.T) {
	root, manifest := writeRehearseWorkspace(t)
	client := &fakeReleaseClient{
		listTracksInfo: []gpc.TrackInfo{{
			Name: "internal",
			Releases: []gpc.TrackReleaseInfo{
				{
					Name:           "1.9.0",
					Status:         "completed",
					VersionCodes:   []int64{120},
					UpdatePriority: 2,
				},
			},
		}},
	}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)

	out, err := runRehearseCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifest, "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		`"status":"ready"`,
		`"packageReadiness":"ready"`,
		`"trackPreview":{"trackFound":true,"releaseFound":true,"releaseName":"1.9.0","hasDiff":true`,
		`"plannedSteps":["preflight_verify","sync_app_details_listing","sync_screenshots","sync_products","sync_subscriptions","deploy_release","post_release_checks"]`,
		`gpc release full --manifest ` + manifest + ` --confirm`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestReleaseRehearseUninitializedPackage(t *testing.T) {
	root, manifest := writeRehearseWorkspace(t)
	client := &fakeReleaseClient{verifyErr: mustPackageNotFoundErr()}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)

	out, err := runRehearseCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifest, "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "blocking issues") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"blocked"`) || !strings.Contains(out, `"packageReadiness":"uninitialized"`) || !strings.Contains(out, `package is not initialized in Google Play yet`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseRehearseDraftBootstrapRequired(t *testing.T) {
	root, manifest := writeRehearseWorkspace(t)
	client := &fakeReleaseClient{
		validateEditErr: errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
		listTracksInfo: []gpc.TrackInfo{{
			Name: "internal",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "draft", VersionCodes: []int64{321}},
			},
		}},
	}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)

	out, err := runRehearseCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifest, "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"packageReadiness":"draft_bootstrap_required"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, `"plannedSteps":["preflight_verify","bootstrap_existing_draft","bootstrap_release","post_bootstrap_recheck"`) {
		t.Fatalf("missing bootstrap steps: %s", out)
	}
}

func TestReleaseRehearseVerifyFailure(t *testing.T) {
	root, manifest := writeRehearseWorkspace(t)
	writeFakeAAB(t, filepath.Join(root, "app.aab"), false)
	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)

	out, err := runRehearseCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifest, "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "blocking issues") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"blocked"`) || !strings.Contains(out, `bundle artifact is not signed`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseRehearseVitalsGateBlocked(t *testing.T) {
	root, manifest := writeRehearseWorkspace(t)
	client := &fakeReleaseClient{}
	reporting := &fakeReleaseReportingClient{
		queryResults: []gpc.ReportingVitalsQueryResult{{
			MetricSet: "crash-rate",
			Rows: []*gpc.ReportingMetricsRow{
				metricsRow("crashRate", "2.4"),
			},
		}},
	}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)
	deps.NewReportingClient = func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
		return reporting, nil
	}

	out, err := runRehearseCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifest, "--vitals-gate", "crashRate<2.0", "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "blocking issues") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"vitalsGate":{"status":"blocked"`) || !strings.Contains(out, `current vitals exceed the requested thresholds`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseRehearseVitalsGateUnavailableWarns(t *testing.T) {
	root, manifest := writeRehearseWorkspace(t)
	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)
	deps.NewReportingClient = func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
		return nil, errors.New("reporting unavailable")
	}

	out, err := runRehearseCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifest, "--vitals-gate", "crashRate<2.0", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"ready"`) || !strings.Contains(out, `vitals gate unavailable`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseRehearseWritesNothingToWorkspace(t *testing.T) {
	root, manifest := writeRehearseWorkspace(t)
	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	seedInitAuth(t, &deps, root)
	before := snapshotFiles(t, root)

	if _, err := runRehearseCommand(t, deps, "--package-name", "com.example.app", "--manifest", manifest, "--output", "json"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := snapshotFiles(t, root)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("workspace files changed\nbefore=%v\nafter=%v", before, after)
	}
}
