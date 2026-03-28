package release

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func runRollbackPlanCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), append([]string{"rollback", "plan"}, args...))
	return out.String(), err
}

func TestReleaseRollbackPlanInProgressRelease(t *testing.T) {
	client := &fakeReleaseClient{
		getTrackInfo: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{
					Name:           "2.1.0 rollout",
					Status:         "inProgress",
					UserFraction:   0.1,
					VersionCodes:   []int64{321},
					UpdatePriority: 3,
				},
				{
					Name:         "2.0.0",
					Status:       "completed",
					VersionCodes: []int64{320},
				},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	out, err := runRollbackPlanCommand(t, deps, "--package-name", "com.example.app", "--track", "production", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		`"status":"ok"`,
		`"planType":"halt_in_progress_rollout"`,
		`"recommendedCommand":"gpc rollback --package-name com.example.app --track production --confirm"`,
		`"activeRelease":{"name":"2.1.0 rollout","status":"inProgress","userFraction":0.1,"versionCodes":[321],"updatePriority":3}`,
		`"previousCompletedRelease":{"name":"2.0.0","status":"completed","versionCodes":[320]}`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected inspection edit cleanup, got %d", client.deleteCalls)
	}
}

func TestReleaseRollbackPlanCompletedOnlyTrack(t *testing.T) {
	client := &fakeReleaseClient{
		getTrackInfo: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{
					Name:         "2.1.0",
					Status:       "completed",
					UserFraction: 1,
					VersionCodes: []int64{321},
				},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	out, err := runRollbackPlanCommand(t, deps, "--package-name", "com.example.app", "--track", "production", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"warn"`) || !strings.Contains(out, `cannot halt a completed rollout on track \"production\"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if strings.Contains(out, `"recommendedCommand"`) {
		t.Fatalf("did not expect recommended command: %s", out)
	}
}

func TestReleaseRollbackPlanMultipleInProgressReleases(t *testing.T) {
	client := &fakeReleaseClient{
		getTrackInfo: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "inProgress", UserFraction: 0.1, VersionCodes: []int64{321}},
				{Status: "inProgress", UserFraction: 0.2, VersionCodes: []int64{322}},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	out, err := runRollbackPlanCommand(t, deps, "--package-name", "com.example.app", "--track", "production", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `multiple in-progress releases`) || !strings.Contains(out, `"status":"warn"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseRollbackPlanMissingVersionCodes(t *testing.T) {
	client := &fakeReleaseClient{
		getTrackInfo: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "inProgress"},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	out, err := runRollbackPlanCommand(t, deps, "--package-name", "com.example.app", "--track", "production", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `has no version codes`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseRollbackPlanTrackFetchFailure(t *testing.T) {
	client := &fakeReleaseClient{getTrackErr: errors.New("permission denied")}
	deps := baseReleaseDeps(t, client)

	out, err := runRollbackPlanCommand(t, deps, "--package-name", "com.example.app", "--track", "production", "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "failed to read track") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"failed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected cleanup delete, got %d", client.deleteCalls)
	}
}

func TestReleaseRollbackPlanCleanupFailure(t *testing.T) {
	client := &fakeReleaseClient{
		deleteEditErr: errors.New("delete failed"),
		getTrackInfo: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "inProgress", UserFraction: 0.1, VersionCodes: []int64{321}},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	out, err := runRollbackPlanCommand(t, deps, "--package-name", "com.example.app", "--track", "production", "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "failed to delete inspection edit") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"failed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseRollbackPlanVitalsGateBlocked(t *testing.T) {
	client := &fakeReleaseClient{
		getTrackInfo: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "inProgress", UserFraction: 0.1, VersionCodes: []int64{321}},
			},
		},
	}
	reporting := &fakeReleaseReportingClient{
		queryResults: []gpc.ReportingVitalsQueryResult{
			{
				MetricSet: "crash-rate",
				Rows: []*gpc.ReportingMetricsRow{
					metricsRow("crashRate", "2.4"),
				},
			},
		},
	}
	deps := baseReleaseDeps(t, client)
	deps.NewReportingClient = func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
		return reporting, nil
	}

	out, err := runRollbackPlanCommand(t, deps,
		"--package-name", "com.example.app",
		"--track", "production",
		"--vitals-gate", "crashRate<2.0",
		"--output", "json",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"vitalsGate":{"status":"blocked"`) || !strings.Contains(out, `"actual":2.4`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseRollbackPlanRejectsInvalidVitalsGate(t *testing.T) {
	_, err := runRollbackPlanCommand(t, Deps{}, "--package-name", "com.example.app", "--track", "production", "--vitals-gate", "oops")
	if err == nil || !strings.Contains(err.Error(), "invalid vitals gate condition") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shared.IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}
}

func TestReleaseRollbackPlanTableOutput(t *testing.T) {
	client := &fakeReleaseClient{
		getTrackInfo: gpc.TrackInfo{
			Name: "production",
			Releases: []gpc.TrackReleaseInfo{
				{Status: "inProgress", UserFraction: 0.1, VersionCodes: []int64{321}},
			},
		},
	}
	deps := baseReleaseDeps(t, client)

	out, err := runRollbackPlanCommand(t, deps, "--package-name", "com.example.app", "--track", "production", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"STATUS\tok", "PLAN_TYPE\thalt_in_progress_rollout", "RECOMMENDED_COMMAND\tgpc rollback --package-name com.example.app --track production --confirm"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}
