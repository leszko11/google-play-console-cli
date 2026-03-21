package release

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	playdeveloperreporting "google.golang.org/api/playdeveloperreporting/v1beta1"
)

func (f *fakeReleaseClient) UploadAPK(_ context.Context, _, _, _ string) (gpc.APKInfo, error) {
	if f.uploadBundleErr != nil {
		return gpc.APKInfo{}, f.uploadBundleErr
	}
	return gpc.APKInfo{VersionCode: 321}, nil
}

func (f *fakeReleaseClient) UploadDeobfuscationFile(_ context.Context, _, _ string, _ int64, _, _ string) (gpc.DeobfuscationFileInfo, error) {
	return gpc.DeobfuscationFileInfo{SymbolType: "proguard"}, nil
}

func writeReleaseAsset(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	return path
}

func writeReleaseManifest(t *testing.T, artifact, mapping string) string {
	t.Helper()
	contents := `artifact: ` + artifact + `
track: internal
status: completed
releaseName: "v2.1.0"
userFraction: 0.1
mappingFile: ` + mapping + `
mappingType: proguard
releaseNotes:
  en-US: "Bug fixes"
  ja-JP: "改善"
`
	path := filepath.Join(t.TempDir(), "release.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func runFullCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestReleaseFullCommitSuccess(t *testing.T) {
	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	artifact := writeFakeAAB(t, filepath.Join(t.TempDir(), "app.aab"), true)
	mapping := writeReleaseAsset(t, "mapping.txt")
	manifest := writeReleaseManifest(t, artifact, mapping)

	out, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--manifest", manifest, "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"committed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.lastTrackName != "internal" || client.lastTrack.ReleaseName != "v2.1.0" {
		t.Fatalf("unexpected track update: %+v", client.lastTrack)
	}
	if len(client.lastTrack.ReleaseNotes) != 2 {
		t.Fatalf("unexpected notes: %+v", client.lastTrack.ReleaseNotes)
	}
}

func TestReleaseFullDryRunDeletesEdit(t *testing.T) {
	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	artifact := writeFakeAAB(t, filepath.Join(t.TempDir(), "app.aab"), true)
	mapping := writeReleaseAsset(t, "mapping.txt")
	manifest := writeReleaseManifest(t, artifact, mapping)

	out, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--manifest", manifest, "--dry-run")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.commitCalls != 0 {
		t.Fatalf("dry-run should not commit, got %d commits", client.commitCalls)
	}
}

func TestReleaseFullRequiresAllowProduction(t *testing.T) {
	client := &fakeReleaseClient{}
	deps := baseReleaseDeps(t, client)
	artifact := writeReleaseAsset(t, "app.aab")
	manifest := writeManifestFile(t, "release.yaml", "artifact: "+artifact+"\ntrack: production\nstatus: completed\n")

	_, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--manifest", manifest, "--confirm")
	if err == nil || !strings.Contains(err.Error(), "--allow-production is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseFullRejectsMissingManifest(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return config.Config{}, nil },
	}
	_, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--confirm")
	if err == nil || !strings.Contains(err.Error(), "--manifest is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseFullBlocksCommitWhenVitalsGateFails(t *testing.T) {
	client := &fakeReleaseClient{}
	reporting := &fakeReleaseReportingClient{
		queryResults: []gpc.ReportingVitalsQueryResult{
			{
				MetricSet: "crash-rate",
				Rows: []*gpc.ReportingMetricsRow{
					metricsRow("crashRate", "2.4"),
				},
			},
			{
				MetricSet: "anr-rate",
				Rows: []*gpc.ReportingMetricsRow{
					metricsRow("anrRate", "0.3"),
				},
			},
		},
	}
	deps := baseReleaseDeps(t, client)
	deps.NewReportingClient = func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
		return reporting, nil
	}
	artifact := writeFakeAAB(t, filepath.Join(t.TempDir(), "app.aab"), true)
	mapping := writeReleaseAsset(t, "mapping.txt")
	manifest := writeReleaseManifest(t, artifact, mapping)

	out, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--manifest", manifest, "--confirm", "--vitals-gate", "crashRate<2.0,anrRate<0.5")
	if err == nil || !strings.Contains(err.Error(), "vitals gate failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"failed"`) || !strings.Contains(out, `"vitalsGate":{"status":"blocked"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.commitCalls != 0 {
		t.Fatalf("expected commit to be blocked, got %d commits", client.commitCalls)
	}
	if reporting.queryCalls != 2 {
		t.Fatalf("expected 2 vitals queries, got %d", reporting.queryCalls)
	}
}

func TestReleaseFullAutoHaltsRegressionDuringWait(t *testing.T) {
	client := &fakeReleaseClient{}
	reporting := &fakeReleaseReportingClient{
		queryResults: []gpc.ReportingVitalsQueryResult{
			{
				MetricSet: "crash-rate",
				Rows: []*gpc.ReportingMetricsRow{
					metricsRow("crashRate", "1.4"),
				},
			},
			{
				MetricSet: "crash-rate",
				Rows: []*gpc.ReportingMetricsRow{
					metricsRow("crashRate", "2.6"),
				},
			},
		},
	}
	deps := baseReleaseDeps(t, client)
	deps.NewReportingClient = func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
		return reporting, nil
	}
	artifact := writeFakeAAB(t, filepath.Join(t.TempDir(), "app.aab"), true)
	manifest := writeManifestFile(t, "release.yaml", "artifact: "+artifact+"\ntrack: internal\nstatus: inProgress\nuserFraction: 0.1\n")

	out, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--manifest", manifest, "--confirm", "--vitals-gate", "crashRate<2.0", "--vitals-wait", "10m", "--auto-halt-on-regression")
	if err == nil || !strings.Contains(err.Error(), "vitals regression detected") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"halted"`) || !strings.Contains(out, `"halted":true`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.createCalls < 2 {
		t.Fatalf("expected at least release + halt edits, got %d create calls", client.createCalls)
	}
	if client.commitCalls < 2 {
		t.Fatalf("expected release commit and halt commit, got %d commits", client.commitCalls)
	}
	if client.lastTrack.Status != "halted" {
		t.Fatalf("expected halted track update, got %+v", client.lastTrack)
	}
}

func TestReleaseFullStopsAfterBootstrapWhenPlayStillReportsDraftState(t *testing.T) {
	client := &fakeReleaseClient{
		validateEditErrs: []error{
			errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
			nil,
			errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
			errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
			errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
			errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
			errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
		},
	}
	deps := baseReleaseDeps(t, client)
	deps.RunAppInit = func(context.Context, []string) error {
		t.Fatal("appinit should not run while package remains in draft bootstrap state")
		return nil
	}
	artifact := writeFakeAAB(t, filepath.Join(t.TempDir(), "app.aab"), true)
	mapping := writeReleaseAsset(t, "mapping.txt")
	manifest := writeReleaseManifest(t, artifact, mapping)

	out, err := runFullCommand(t, deps, "full", "--package-name", "com.example.app", "--manifest", manifest, "--confirm")
	if err == nil || !strings.Contains(err.Error(), "bootstrap release committed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"bootstrap_committed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.commitCalls != 1 {
		t.Fatalf("expected only bootstrap commit, got %d commits", client.commitCalls)
	}
}

func metricsRow(metric, value string) *gpc.ReportingMetricsRow {
	return &gpc.ReportingMetricsRow{
		AggregationPeriod: "FULL_RANGE",
		Metrics: []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricValue{
			{
				Metric: metric,
				DecimalValue: &playdeveloperreporting.GoogleTypeDecimal{
					Value: value,
				},
			},
		},
	}
}
