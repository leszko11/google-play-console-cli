package reports

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	getResult         gpc.ReportingVitalsMetricSetInfo
	getErr            error
	queryResult       gpc.ReportingVitalsQueryResult
	queryErr          error
	capturedPackage   string
	capturedMetricSet gpc.ReportingVitalsMetricSet
	capturedQuery     *gpc.ReportingVitalsQueryRequest
}

func (f *fakeClient) GetVitalsMetricSet(_ context.Context, packageName string, metricSet gpc.ReportingVitalsMetricSet) (gpc.ReportingVitalsMetricSetInfo, error) {
	f.capturedPackage = packageName
	f.capturedMetricSet = metricSet
	return f.getResult, f.getErr
}

func (f *fakeClient) QueryVitalsMetricSet(_ context.Context, packageName string, metricSet gpc.ReportingVitalsMetricSet, request *gpc.ReportingVitalsQueryRequest) (gpc.ReportingVitalsQueryResult, error) {
	f.capturedPackage = packageName
	f.capturedMetricSet = metricSet
	f.capturedQuery = request
	return f.queryResult, f.queryErr
}

func runReports(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func bindGlobalPackageName(t *testing.T, packageName string) {
	t.Helper()
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	cfg := &shared.GlobalFlags{}
	shared.BindGlobalFlags(fs, cfg)
	cfg.PackageName = packageName
}

func TestReportsVitalsGet_ReturnsMetricSet(t *testing.T) {
	fc := &fakeClient{
		getResult: gpc.ReportingVitalsMetricSetInfo{
			MetricSet:    "crash-rate",
			ResourceName: "apps/com.example.app/crashRateMetricSet",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runReports(t, deps, "vitals", "get", "--package-name", "com.example.app", "--metric-set", "crash-rate")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"metricSet":"crash-rate"`) || !strings.Contains(out, `"resourceName":"apps/com.example.app/crashRateMetricSet"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedPackage != "com.example.app" || fc.capturedMetricSet != gpc.ReportingVitalsMetricSetCrashRate {
		t.Fatalf("unexpected captured call: package=%q metricSet=%q", fc.capturedPackage, fc.capturedMetricSet)
	}
}

func TestReportsVitalsGet_RequiresMetricSet(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runReports(t, deps, "vitals", "get", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "--metric-set is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReportsVitalsQuery_ReturnsRows(t *testing.T) {
	fc := &fakeClient{
		queryResult: gpc.ReportingVitalsQueryResult{
			MetricSet:     "crash-rate",
			ResourceName:  "apps/com.example.app/crashRateMetricSet",
			Rows:          []*gpc.ReportingMetricsRow{{AggregationPeriod: "DAILY"}},
			NextPageToken: "tok-2",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
		Stdin:      strings.NewReader(`{"metrics":["crashRate"],"timelineSpec":{"aggregationPeriod":"DAILY"}}`),
	}

	out, err := runReports(t, deps, "vitals", "query", "--package-name", "com.example.app", "--metric-set", "crash-rate", "--input", "-")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"rowCount":1`) || !strings.Contains(out, `"nextPageToken":"tok-2"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedMetricSet != gpc.ReportingVitalsMetricSetCrashRate || fc.capturedQuery == nil {
		t.Fatalf("unexpected captured query: metricSet=%q payload=%v", fc.capturedMetricSet, fc.capturedQuery)
	}
	if len(fc.capturedQuery.Metrics) != 1 || fc.capturedQuery.Metrics[0] != "crashRate" {
		t.Fatalf("unexpected captured metrics: %#v", fc.capturedQuery.Metrics)
	}
}

func TestReportsVitalsQuery_RequiresInput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runReports(t, deps, "vitals", "query", "--package-name", "com.example.app", "--metric-set", "crash-rate")
	if err == nil || !strings.Contains(err.Error(), "--input is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReportsVitalsGet_UsesGlobalPackageName(t *testing.T) {
	bindGlobalPackageName(t, "com.example.global")
	fc := &fakeClient{
		getResult: gpc.ReportingVitalsMetricSetInfo{MetricSet: "anr-rate"},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	if _, err := runReports(t, deps, "vitals", "get", "--metric-set", "anr-rate"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if fc.capturedPackage != "com.example.global" {
		t.Fatalf("expected global package name, got %q", fc.capturedPackage)
	}
}
