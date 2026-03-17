package reports

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeFinancialClient struct {
	listResult         FinancialObjectList
	listErr            error
	downloadResult     FinancialObjectDownload
	downloadErr        error
	capturedBucket     string
	capturedPrefix     string
	capturedPageSize   int64
	capturedPageToken  string
	capturedObjectName string
}

func (f *fakeFinancialClient) ListObjects(_ context.Context, bucket, prefix string, pageSize int64, pageToken string) (FinancialObjectList, error) {
	f.capturedBucket = bucket
	f.capturedPrefix = prefix
	f.capturedPageSize = pageSize
	f.capturedPageToken = pageToken
	return f.listResult, f.listErr
}

func (f *fakeFinancialClient) DownloadObject(_ context.Context, bucket, objectName string) (FinancialObjectDownload, error) {
	f.capturedBucket = bucket
	f.capturedObjectName = objectName
	return f.downloadResult, f.downloadErr
}

func TestReportsFinancialList_ReturnsObjects(t *testing.T) {
	ff := &fakeFinancialClient{
		listResult: FinancialObjectList{
			Objects: []FinancialObjectInfo{
				{
					Bucket:      "play-financial",
					Name:        "reports/earnings_2026-03.csv",
					Size:        128,
					ContentType: "text/csv",
					Updated:     time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC),
				},
			},
			NextPageToken: "tok-2",
		},
	}
	deps := Deps{
		LoadConfig:         func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:          func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		NewFinancialClient: func(context.Context, gpc.CredentialInput) (FinancialClient, error) { return ff, nil },
	}

	out, err := runReports(t, deps, "financial", "list", "--bucket", "play-financial", "--prefix", "reports/", "--page-size", "100", "--page-token", "tok-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{`"bucket":"play-financial"`, `"name":"reports/earnings_2026-03.csv"`, `"count":1`, `"nextPageToken":"tok-2"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %s", want, out)
		}
	}
	if ff.capturedBucket != "play-financial" || ff.capturedPrefix != "reports/" || ff.capturedPageSize != 100 || ff.capturedPageToken != "tok-1" {
		t.Fatalf("unexpected captured list params: %+v", ff)
	}
}

func TestReportsFinancialGet_ReturnsNormalizedRows(t *testing.T) {
	ff := &fakeFinancialClient{
		downloadResult: FinancialObjectDownload{
			Bucket:      "play-financial",
			Name:        "reports/sales.csv",
			ContentType: "text/csv",
			Data:        []byte("date,sku,amount\n2026-03-01,sku.one,12.34\n2026-03-02,sku.two,45.67\n"),
		},
	}
	deps := Deps{
		LoadConfig:         func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:          func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		NewFinancialClient: func(context.Context, gpc.CredentialInput) (FinancialClient, error) { return ff, nil },
	}

	out, err := runReports(t, deps, "financial", "get", "--gcs-uri", "gs://play-financial/reports/sales.csv")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{`"bucket":"play-financial"`, `"object":"reports/sales.csv"`, `"rowCount":2`, `"sku":"sku.one"`, `"amount":"45.67"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %s", want, out)
		}
	}
	if ff.capturedBucket != "play-financial" || ff.capturedObjectName != "reports/sales.csv" {
		t.Fatalf("unexpected captured get params: %+v", ff)
	}
}

func TestReportsFinancialGet_SupportsGzip(t *testing.T) {
	ff := &fakeFinancialClient{
		downloadResult: FinancialObjectDownload{
			Bucket:          "play-financial",
			Name:            "reports/sales.csv.gz",
			ContentType:     "application/gzip",
			ContentEncoding: "gzip",
			Data:            gzipCSV(t, "date,amount\n2026-03-01,12.34\n"),
		},
	}
	deps := Deps{
		LoadConfig:         func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:          func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		NewFinancialClient: func(context.Context, gpc.CredentialInput) (FinancialClient, error) { return ff, nil },
	}

	out, err := runReports(t, deps, "financial", "get", "--bucket", "play-financial", "--object", "reports/sales.csv.gz")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"rowCount":1`) || !strings.Contains(out, `"amount":"12.34"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReportsFinancialGet_TableOutput(t *testing.T) {
	ff := &fakeFinancialClient{
		downloadResult: FinancialObjectDownload{
			Bucket:      "play-financial",
			Name:        "reports/sales.csv",
			ContentType: "text/csv",
			Data:        []byte("date,sku,amount\n2026-03-01,sku.one,12.34\n"),
		},
	}
	deps := Deps{
		LoadConfig:         func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:          func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		NewFinancialClient: func(context.Context, gpc.CredentialInput) (FinancialClient, error) { return ff, nil },
	}

	out, err := runReports(t, deps, "financial", "get", "--gcs-uri", "gs://play-financial/reports/sales.csv", "--output", "table")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, "date\tsku\tamount") || !strings.Contains(out, "2026-03-01\tsku.one\t12.34") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReportsFinancialList_MarkdownOutput(t *testing.T) {
	ff := &fakeFinancialClient{
		listResult: FinancialObjectList{
			Objects: []FinancialObjectInfo{
				{
					Bucket:      "play-financial",
					Name:        "reports/earnings_2026-03.csv",
					Size:        128,
					ContentType: "text/csv",
					Updated:     time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}
	deps := Deps{
		LoadConfig:         func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:          func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		NewFinancialClient: func(context.Context, gpc.CredentialInput) (FinancialClient, error) { return ff, nil },
	}

	out, err := runReports(t, deps, "financial", "list", "--bucket", "play-financial", "--output", "markdown")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{
		"| bucket | name | size | contentType | updated |",
		"| play-financial | reports/earnings_2026-03.csv | 128 | text/csv | 2026-03-14T12:00:00Z |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestReportsFinancialList_YAMLOutput(t *testing.T) {
	ff := &fakeFinancialClient{
		listResult: FinancialObjectList{
			Objects: []FinancialObjectInfo{
				{
					Bucket:      "play-financial",
					Name:        "reports/earnings_2026-03.csv",
					Size:        128,
					ContentType: "text/csv",
					Updated:     time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}
	deps := Deps{
		LoadConfig:         func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:          func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		NewFinancialClient: func(context.Context, gpc.CredentialInput) (FinancialClient, error) { return ff, nil },
	}

	out, err := runReports(t, deps, "financial", "list", "--bucket", "play-financial", "--output", "yaml")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{"bucket: play-financial", "count: 1", "- bucket: play-financial"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestReportsFinancialGet_MarkdownOutput(t *testing.T) {
	ff := &fakeFinancialClient{
		downloadResult: FinancialObjectDownload{
			Bucket:      "play-financial",
			Name:        "reports/sales.csv",
			ContentType: "text/csv",
			Data:        []byte("date,sku,amount\n2026-03-01,sku.one,12.34\n"),
		},
	}
	deps := Deps{
		LoadConfig:         func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:          func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		NewFinancialClient: func(context.Context, gpc.CredentialInput) (FinancialClient, error) { return ff, nil },
	}

	out, err := runReports(t, deps, "financial", "get", "--gcs-uri", "gs://play-financial/reports/sales.csv", "--output", "markdown")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{
		"| date | sku | amount |",
		"| 2026-03-01 | sku.one | 12.34 |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestReportsFinancialGet_YAMLOutput(t *testing.T) {
	ff := &fakeFinancialClient{
		downloadResult: FinancialObjectDownload{
			Bucket:      "play-financial",
			Name:        "reports/sales.csv",
			ContentType: "text/csv",
			Data:        []byte("date,sku,amount\n2026-03-01,sku.one,12.34\n"),
		},
	}
	deps := Deps{
		LoadConfig:         func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:          func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		NewFinancialClient: func(context.Context, gpc.CredentialInput) (FinancialClient, error) { return ff, nil },
	}

	out, err := runReports(t, deps, "financial", "get", "--gcs-uri", "gs://play-financial/reports/sales.csv", "--output", "yaml")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{"bucket: play-financial", "object: reports/sales.csv", "sku: sku.one"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestReportsFinancialGet_RequiresLocation(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
		NewFinancialClient: func(context.Context, gpc.CredentialInput) (FinancialClient, error) {
			return &fakeFinancialClient{}, nil
		},
	}

	_, err := runReports(t, deps, "financial", "get")
	if err == nil || !strings.Contains(err.Error(), "either --gcs-uri or both --bucket and --object are required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func gzipCSV(t *testing.T, raw string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := io.WriteString(zw, raw); err != nil {
		t.Fatalf("write gzip payload: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close gzip payload: %v", err)
	}
	return buf.Bytes()
}
