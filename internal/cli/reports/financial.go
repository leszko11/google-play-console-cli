package reports

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type financialGetResult struct {
	Bucket      string              `json:"bucket"`
	Object      string              `json:"object"`
	ContentType string              `json:"contentType,omitempty"`
	Columns     []string            `json:"columns"`
	Rows        []map[string]string `json:"rows"`
	RowCount    int                 `json:"rowCount"`
}

func newFinancialCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "financial",
		ShortHelp: "Financial report commands backed by Cloud Storage",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newFinancialListCommand(deps),
			newFinancialGetCommand(deps),
		},
	}
}

func newFinancialListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var bucket, prefix, pageToken, output string
	var pageSize int64
	fs.StringVar(&bucket, "bucket", "", "Cloud Storage bucket containing financial reports")
	fs.StringVar(&prefix, "prefix", "", "Optional object prefix filter")
	fs.Int64Var(&pageSize, "page-size", 0, "Maximum objects per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown, csv, tsv")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List Cloud Storage financial report objects",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			bucket = strings.TrimSpace(bucket)
			if bucket == "" {
				return shared.UsageErrorf("--bucket is required")
			}
			if pageSize < 0 {
				return shared.UsageErrorf("--page-size must be greater than or equal to zero")
			}

			client, requestCtx, cancel, err := buildFinancialClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			result, err := client.ListObjects(requestCtx, bucket, prefix, pageSize, pageToken)
			if err != nil {
				return fmt.Errorf("failed to list financial report objects: %w", err)
			}

			resolvedOutput := shared.ResolveOutput(output)
			switch resolvedOutput {
			case "json":
				return shared.WriteJSON(deps.Stdout, map[string]any{
					"bucket":        bucket,
					"prefix":        strings.TrimSpace(prefix),
					"objects":       result.Objects,
					"count":         len(result.Objects),
					"nextPageToken": result.NextPageToken,
				})
			case "table":
				return writeFinancialListTable(deps.Stdout, result.Objects)
			case "markdown":
				return writeFinancialListMarkdown(deps.Stdout, result.Objects)
			case "csv", "tsv":
				rows := make([][]string, 0, len(result.Objects))
				for _, object := range result.Objects {
					rows = append(rows, []string{
						object.Bucket,
						object.Name,
						fmt.Sprintf("%d", object.Size),
						object.ContentType,
						object.Updated.Format(timeLayoutRFC3339OrEmpty),
					})
				}
				return shared.WriteDelimited(deps.Stdout, resolvedOutput, []string{"bucket", "name", "size", "contentType", "updated"}, rows)
			default:
				return shared.UsageErrorf("unsupported output format %q", resolvedOutput)
			}
		},
	}
}

func newFinancialGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var gcsURI, bucket, objectName, output string
	fs.StringVar(&gcsURI, "gcs-uri", "", "Cloud Storage object URI in the form gs://bucket/object.csv")
	fs.StringVar(&bucket, "bucket", "", "Cloud Storage bucket containing the report")
	fs.StringVar(&objectName, "object", "", "Cloud Storage object name")
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown, csv, tsv")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Download and normalize a financial report CSV from Cloud Storage",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			location, err := resolveFinancialLocation(gcsURI, bucket, objectName)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := buildFinancialClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			download, err := client.DownloadObject(requestCtx, location.Bucket, location.Object)
			if err != nil {
				return fmt.Errorf("failed to download financial report object: %w", err)
			}

			columns, records, err := parseFinancialCSV(download)
			if err != nil {
				return err
			}
			rows := financialRecordMaps(columns, records)
			resolvedOutput := shared.ResolveOutput(output)
			switch resolvedOutput {
			case "json":
				return shared.WriteJSON(deps.Stdout, financialGetResult{
					Bucket:      download.Bucket,
					Object:      download.Name,
					ContentType: download.ContentType,
					Columns:     columns,
					Rows:        rows,
					RowCount:    len(rows),
				})
			case "table":
				return writeFinancialRowsTable(deps.Stdout, columns, records)
			case "markdown":
				return shared.WriteMarkdownTable(deps.Stdout, columns, records)
			case "csv", "tsv":
				return shared.WriteDelimited(deps.Stdout, resolvedOutput, columns, records)
			default:
				return shared.UsageErrorf("unsupported output format %q", resolvedOutput)
			}
		},
	}
}

type financialLocation struct {
	Bucket string
	Object string
}

const timeLayoutRFC3339OrEmpty = "2006-01-02T15:04:05Z07:00"

func buildFinancialClient(ctx context.Context, deps Deps) (FinancialClient, context.Context, context.CancelFunc, error) {
	client, requestCtx, cancel, err := shared.BuildClient(ctx, shared.BuildClientDeps[FinancialClient]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewFinancialClient,
	})
	if err != nil {
		return nil, nil, func() {}, err
	}
	return client, requestCtx, cancel, nil
}

func resolveFinancialLocation(gcsURI, bucket, objectName string) (financialLocation, error) {
	gcsURI = strings.TrimSpace(gcsURI)
	bucket = strings.TrimSpace(bucket)
	objectName = strings.TrimSpace(objectName)

	if gcsURI != "" && (bucket != "" || objectName != "") {
		return financialLocation{}, shared.UsageErrorf("--gcs-uri cannot be combined with --bucket or --object")
	}
	if gcsURI != "" {
		if !strings.HasPrefix(gcsURI, "gs://") {
			return financialLocation{}, shared.UsageErrorf("--gcs-uri must start with gs://")
		}
		trimmed := strings.TrimPrefix(gcsURI, "gs://")
		bucketPart, objectPart, ok := strings.Cut(trimmed, "/")
		if !ok || strings.TrimSpace(bucketPart) == "" || strings.TrimSpace(objectPart) == "" {
			return financialLocation{}, shared.UsageErrorf("--gcs-uri must include both bucket and object")
		}
		return financialLocation{Bucket: bucketPart, Object: objectPart}, nil
	}
	if bucket == "" || objectName == "" {
		return financialLocation{}, shared.UsageErrorf("either --gcs-uri or both --bucket and --object are required")
	}
	return financialLocation{Bucket: bucket, Object: objectName}, nil
}

func parseFinancialCSV(download FinancialObjectDownload) ([]string, [][]string, error) {
	raw, err := maybeGunzipFinancial(download)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode financial report payload: %w", err)
	}

	reader := csv.NewReader(bytes.NewReader(raw))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse financial report csv: %w", err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("financial report csv is empty")
	}

	columns := normalizeFinancialColumns(records[0])
	rows := make([][]string, 0, max(0, len(records)-1))
	for _, record := range records[1:] {
		columns = ensureFinancialColumns(columns, len(record))
	}
	for _, record := range records[1:] {
		row := make([]string, len(columns))
		copy(row, record)
		rows = append(rows, row)
	}
	return columns, rows, nil
}

func maybeGunzipFinancial(download FinancialObjectDownload) ([]byte, error) {
	encoding := strings.ToLower(strings.TrimSpace(download.ContentEncoding))
	name := strings.ToLower(strings.TrimSpace(download.Name))
	if encoding != "gzip" && !strings.HasSuffix(name, ".gz") {
		return download.Data, nil
	}

	reader, err := gzip.NewReader(bytes.NewReader(download.Data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func normalizeFinancialColumns(header []string) []string {
	columns := make([]string, len(header))
	for i, column := range header {
		column = strings.TrimSpace(column)
		if i == 0 {
			column = strings.TrimPrefix(column, "\ufeff")
		}
		if column == "" {
			column = fmt.Sprintf("column_%d", i+1)
		}
		columns[i] = column
	}
	return columns
}

func ensureFinancialColumns(columns []string, size int) []string {
	for len(columns) < size {
		columns = append(columns, fmt.Sprintf("column_%d", len(columns)+1))
	}
	return columns
}

func financialRecordMaps(columns []string, records [][]string) []map[string]string {
	rows := make([]map[string]string, 0, len(records))
	for _, record := range records {
		row := make(map[string]string, len(columns))
		for i, column := range columns {
			if i < len(record) {
				row[column] = record[i]
			} else {
				row[column] = ""
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func writeFinancialListTable(out io.Writer, objects []FinancialObjectInfo) error {
	if _, err := fmt.Fprintln(out, "bucket\tname\tsize\tcontentType\tupdated"); err != nil {
		return err
	}
	for _, object := range objects {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%d\t%s\t%s\n", object.Bucket, object.Name, object.Size, object.ContentType, object.Updated.Format(timeLayoutRFC3339OrEmpty)); err != nil {
			return err
		}
	}
	return nil
}

func writeFinancialListMarkdown(out io.Writer, objects []FinancialObjectInfo) error {
	rows := make([][]string, 0, len(objects))
	for _, object := range objects {
		rows = append(rows, []string{
			object.Bucket,
			object.Name,
			fmt.Sprintf("%d", object.Size),
			object.ContentType,
			object.Updated.Format(timeLayoutRFC3339OrEmpty),
		})
	}
	return shared.WriteMarkdownTable(out, []string{"bucket", "name", "size", "contentType", "updated"}, rows)
}

func writeFinancialRowsTable(out io.Writer, columns []string, rows [][]string) error {
	if _, err := fmt.Fprintln(out, strings.Join(columns, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(out, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return nil
}
