package reports

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	appsResult          gpc.ReportingAppsListInfo
	appsErr             error
	anomaliesResult     gpc.ReportingAnomaliesListInfo
	anomaliesErr        error
	errorIssuesResult   gpc.ReportingErrorIssuesListInfo
	errorIssuesErr      error
	errorReportsResult  gpc.ReportingErrorReportsListInfo
	errorReportsErr     error
	getResult           gpc.ReportingVitalsMetricSetInfo
	getErr              error
	queryResult         gpc.ReportingVitalsQueryResult
	queryErr            error
	capturedPackage     string
	capturedMetricSet   gpc.ReportingVitalsMetricSet
	capturedQuery       *gpc.ReportingVitalsQueryRequest
	capturedFilter      string
	capturedOrderBy     string
	capturedPageSize    int64
	capturedPageToken   string
	capturedPaginate    bool
	capturedInterval    gpc.ReportingInterval
	capturedSampleLimit int64
}

func (f *fakeClient) SearchApps(_ context.Context, pageSize int64, pageToken string, paginate bool) (gpc.ReportingAppsListInfo, error) {
	f.capturedPageSize = pageSize
	f.capturedPageToken = pageToken
	f.capturedPaginate = paginate
	return f.appsResult, f.appsErr
}

func (f *fakeClient) ListAnomalies(_ context.Context, packageName, filter string, pageSize int64, pageToken string, paginate bool) (gpc.ReportingAnomaliesListInfo, error) {
	f.capturedPackage = packageName
	f.capturedFilter = filter
	f.capturedPageSize = pageSize
	f.capturedPageToken = pageToken
	f.capturedPaginate = paginate
	return f.anomaliesResult, f.anomaliesErr
}

func (f *fakeClient) SearchErrorIssues(_ context.Context, packageName, filter, orderBy string, interval gpc.ReportingInterval, pageSize, sampleErrorReportLimit int64, pageToken string, paginate bool) (gpc.ReportingErrorIssuesListInfo, error) {
	f.capturedPackage = packageName
	f.capturedFilter = filter
	f.capturedOrderBy = orderBy
	f.capturedInterval = interval
	f.capturedPageSize = pageSize
	f.capturedPageToken = pageToken
	f.capturedPaginate = paginate
	f.capturedSampleLimit = sampleErrorReportLimit
	return f.errorIssuesResult, f.errorIssuesErr
}

func (f *fakeClient) SearchErrorReports(_ context.Context, packageName, filter string, interval gpc.ReportingInterval, pageSize int64, pageToken string, paginate bool) (gpc.ReportingErrorReportsListInfo, error) {
	f.capturedPackage = packageName
	f.capturedFilter = filter
	f.capturedInterval = interval
	f.capturedPageSize = pageSize
	f.capturedPageToken = pageToken
	f.capturedPaginate = paginate
	return f.errorReportsResult, f.errorReportsErr
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
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		}
	}
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

func TestReportsAppsList_ReturnsApps(t *testing.T) {
	fc := &fakeClient{
		appsResult: gpc.ReportingAppsListInfo{
			Apps: []*gpc.ReportingApp{
				{Name: "apps/com.example.app", PackageName: "com.example.app", DisplayName: "Example App"},
			},
			NextPageToken: "tok-2",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runReports(t, deps, "apps", "list", "--page-size", "50", "--page-token", "tok-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"count":1`) || !strings.Contains(out, `"packageName":"com.example.app"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedPageSize != 50 || fc.capturedPageToken != "tok-1" || fc.capturedPaginate {
		t.Fatalf("unexpected captured pagination: size=%d token=%q paginate=%t", fc.capturedPageSize, fc.capturedPageToken, fc.capturedPaginate)
	}
}

func TestReportsAnomaliesList_ReturnsAnomalies(t *testing.T) {
	fc := &fakeClient{
		anomaliesResult: gpc.ReportingAnomaliesListInfo{
			Anomalies: []*gpc.ReportingAnomaly{
				{Name: "apps/com.example.app/anomalies/a1", MetricSet: "apps/com.example.app/crashRateMetricSet"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runReports(t, deps, "anomalies", "list", "--package-name", "com.example.app", "--filter", `activeBetween("2026-03-01T00:00:00Z", UNBOUNDED)`, "--page-size", "10")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"count":1`) || !strings.Contains(out, `"metricSet":"apps/com.example.app/crashRateMetricSet"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedPackage != "com.example.app" || !strings.Contains(fc.capturedFilter, "activeBetween") {
		t.Fatalf("unexpected captured anomalies call: package=%q filter=%q", fc.capturedPackage, fc.capturedFilter)
	}
}

func TestReportsErrorIssuesList_ReturnsIssues(t *testing.T) {
	fc := &fakeClient{
		errorIssuesResult: gpc.ReportingErrorIssuesListInfo{
			Issues: []*gpc.ReportingErrorIssue{
				{Name: "apps/com.example.app/errorIssues/1"},
			},
			NextPageToken: "tok-2",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runReports(t, deps, "errors", "issues", "list", "--package-name", "com.example.app", "--filter", `errorIssueType = CRASH`, "--order-by", "errorReportCount desc", "--start-time", "2026-03-01T00:00:00Z", "--end-time", "2026-03-07T00:00:00Z", "--sample-error-report-limit", "1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"count":1`) || !strings.Contains(out, `"nextPageToken":"tok-2"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedOrderBy != "errorReportCount desc" || fc.capturedSampleLimit != 1 {
		t.Fatalf("unexpected captured issue params: order=%q sample=%d", fc.capturedOrderBy, fc.capturedSampleLimit)
	}
	if fc.capturedInterval.StartTime != "2026-03-01T00:00:00Z" || fc.capturedInterval.EndTime != "2026-03-07T00:00:00Z" {
		t.Fatalf("unexpected interval: %+v", fc.capturedInterval)
	}
}

func TestReportsErrorReportsList_ReturnsReports(t *testing.T) {
	fc := &fakeClient{
		errorReportsResult: gpc.ReportingErrorReportsListInfo{
			Reports: []*gpc.ReportingErrorReport{
				{Name: "apps/com.example.app/errorReports/1"},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runReports(t, deps, "errors", "reports", "list", "--package-name", "com.example.app", "--filter", `errorReportId = 1`, "--page-size", "10")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"count":1`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedFilter != "errorReportId = 1" || fc.capturedPageSize != 10 {
		t.Fatalf("unexpected captured report params: filter=%q pageSize=%d", fc.capturedFilter, fc.capturedPageSize)
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

func TestReportsSummary_ReturnsAggregate(t *testing.T) {
	fc := &fakeClient{
		appsResult: gpc.ReportingAppsListInfo{
			Apps: []*gpc.ReportingApp{
				{Name: "apps/com.example.app", PackageName: "com.example.app", DisplayName: "Example App"},
			},
		},
		anomaliesResult: gpc.ReportingAnomaliesListInfo{},
		getResult: gpc.ReportingVitalsMetricSetInfo{
			MetricSet:    "crash-rate",
			ResourceName: "apps/com.example.app/crashRateMetricSet",
		},
		queryResult: gpc.ReportingVitalsQueryResult{
			MetricSet:     "crash-rate",
			ResourceName:  "apps/com.example.app/crashRateMetricSet",
			Rows:          []*gpc.ReportingMetricsRow{{AggregationPeriod: "DAILY"}},
			NextPageToken: "",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runReports(t, deps, "summary", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{`"status":"ok"`, `"packageName":"com.example.app"`, `"visible":true`, `"count":0`, `"rowCount":1`, `"metricSet":"crash-rate"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %s", want, out)
		}
	}
	if fc.capturedMetricSet != gpc.ReportingVitalsMetricSetCrashRate {
		t.Fatalf("expected crash-rate metric set, got %q", fc.capturedMetricSet)
	}
	if fc.capturedQuery == nil || len(fc.capturedQuery.Metrics) != 1 || fc.capturedQuery.Metrics[0] != "crashRate" {
		t.Fatalf("expected default crash-rate query payload, got %#v", fc.capturedQuery)
	}
}

func TestReportsSummary_WarnsWhenAnomaliesPresent(t *testing.T) {
	fc := &fakeClient{
		appsResult: gpc.ReportingAppsListInfo{
			Apps: []*gpc.ReportingApp{
				{Name: "apps/com.example.app", PackageName: "com.example.app"},
			},
		},
		anomaliesResult: gpc.ReportingAnomaliesListInfo{
			Anomalies: []*gpc.ReportingAnomaly{{Name: "apps/com.example.app/anomalies/a1"}},
		},
		getResult: gpc.ReportingVitalsMetricSetInfo{MetricSet: "anr-rate"},
		queryResult: gpc.ReportingVitalsQueryResult{
			MetricSet: "anr-rate",
			Rows:      []*gpc.ReportingMetricsRow{},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runReports(t, deps, "summary", "--package-name", "com.example.app", "--metric-set", "anr-rate")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"warn"`) || !strings.Contains(out, `"count":1`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReportsSummary_WarnsWhenPackageNotVisible(t *testing.T) {
	fc := &fakeClient{
		appsResult: gpc.ReportingAppsListInfo{
			Apps: []*gpc.ReportingApp{
				{Name: "apps/com.other.app", PackageName: "com.other.app"},
			},
		},
		anomaliesResult: gpc.ReportingAnomaliesListInfo{},
		getResult:       gpc.ReportingVitalsMetricSetInfo{MetricSet: "crash-rate"},
		queryResult:     gpc.ReportingVitalsQueryResult{MetricSet: "crash-rate"},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runReports(t, deps, "summary", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"warn"`) || !strings.Contains(out, `"visible":false`) {
		t.Fatalf("unexpected output: %s", out)
	}
}
