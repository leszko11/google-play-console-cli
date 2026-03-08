package reports

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	SearchApps(ctx context.Context, pageSize int64, pageToken string, paginate bool) (gpc.ReportingAppsListInfo, error)
	ListAnomalies(ctx context.Context, packageName, filter string, pageSize int64, pageToken string, paginate bool) (gpc.ReportingAnomaliesListInfo, error)
	GetVitalsMetricSet(ctx context.Context, packageName string, metricSet gpc.ReportingVitalsMetricSet) (gpc.ReportingVitalsMetricSetInfo, error)
	QueryVitalsMetricSet(ctx context.Context, packageName string, metricSet gpc.ReportingVitalsMetricSet, request *gpc.ReportingVitalsQueryRequest) (gpc.ReportingVitalsQueryResult, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	return &ffcli.Command{
		Name:      "reports",
		ShortHelp: "Google Play Developer Reporting commands",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newAppsCommand(deps),
			newAnomaliesCommand(deps),
			newVitalsCommand(deps),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewClient == nil {
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (Client, error) {
			return gpc.NewReportingClient(ctx, creds)
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
	}
	if deps.Stdin == nil {
		deps.Stdin = os.Stdin
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return deps
}

func newVitalsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "vitals",
		ShortHelp: "Vitals metric set reporting commands",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newVitalsGetCommand(deps),
			newVitalsQueryCommand(deps),
		},
	}
}

func newAppsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "apps",
		ShortHelp: "Reporting app discovery commands",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newAppsListCommand(deps),
		},
	}
}

func newAppsListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var pageToken string
	var pageSize int64
	fs.Int64Var(&pageSize, "page-size", 0, "Maximum apps per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List reporting-accessible apps",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, requestCtx, cancel, err := buildReportingClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()
			if pageSize < 0 {
				return shared.UsageErrorf("--page-size must be greater than or equal to zero")
			}
			result, err := client.SearchApps(requestCtx, pageSize, pageToken, shared.ActiveGlobalFlags().Paginate)
			if err != nil {
				return fmt.Errorf("failed to list reporting apps: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"apps":          result.Apps,
				"count":         len(result.Apps),
				"nextPageToken": result.NextPageToken,
			})
		},
	}
}

func newAnomaliesCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "anomalies",
		ShortHelp: "Reporting anomaly commands",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newAnomaliesListCommand(deps),
		},
	}
}

func newAnomaliesListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, filter, pageToken string
	var pageSize int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&filter, "filter", "", "Anomaly filter expression")
	fs.Int64Var(&pageSize, "page-size", 0, "Maximum anomalies per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List reporting anomalies for an app",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			if pageSize < 0 {
				return shared.UsageErrorf("--page-size must be greater than or equal to zero")
			}
			result, err := client.ListAnomalies(requestCtx, pkg, filter, pageSize, pageToken, shared.ActiveGlobalFlags().Paginate)
			if err != nil {
				return fmt.Errorf("failed to list anomalies: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"anomalies":     result.Anomalies,
				"count":         len(result.Anomalies),
				"nextPageToken": result.NextPageToken,
			})
		},
	}
}

func newVitalsGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, metricSetRaw string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&metricSetRaw, "metric-set", "", "Vitals metric set")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get vitals metric set freshness metadata",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			metricSet, err := resolveMetricSet(metricSetRaw)
			if err != nil {
				return err
			}
			result, err := client.GetVitalsMetricSet(requestCtx, pkg, metricSet)
			if err != nil {
				return fmt.Errorf("failed to get vitals metric set: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"metricSet":   result,
			})
		},
	}
}

func newVitalsQueryCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, metricSetRaw, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&metricSetRaw, "metric-set", "", "Vitals metric set")
	fs.StringVar(&inputPath, "input", "", "Path to vitals query JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "query",
		ShortHelp: "Query vitals metric set rows",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			metricSet, err := resolveMetricSet(metricSetRaw)
			if err != nil {
				return err
			}
			payload, err := readVitalsQueryPayload(inputPath, deps.Stdin)
			if err != nil {
				return err
			}
			result, err := client.QueryVitalsMetricSet(requestCtx, pkg, metricSet, payload)
			if err != nil {
				return fmt.Errorf("failed to query vitals metric set: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"metricSet":     result.MetricSet,
				"resourceName":  result.ResourceName,
				"rows":          result.Rows,
				"rowCount":      len(result.Rows),
				"nextPageToken": result.NextPageToken,
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName string) (Client, string, context.Context, context.CancelFunc, error) {
	pkg, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return nil, "", nil, func() {}, err
	}

	client, requestCtx, cancel, err := shared.BuildClient(ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
	})
	if err != nil {
		return nil, "", nil, func() {}, err
	}

	return client, pkg, requestCtx, cancel, nil
}

func buildReportingClient(ctx context.Context, deps Deps) (Client, context.Context, context.CancelFunc, error) {
	client, requestCtx, cancel, err := shared.BuildClient(ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
	})
	if err != nil {
		return nil, nil, func() {}, err
	}
	return client, requestCtx, cancel, nil
}

func resolveMetricSet(raw string) (gpc.ReportingVitalsMetricSet, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", shared.UsageErrorf("--metric-set is required")
	}
	metricSet, err := gpc.ParseReportingVitalsMetricSet(trimmed)
	if err != nil {
		return "", shared.UsageErrorf("%v", err)
	}
	return metricSet, nil
}

func readVitalsQueryPayload(path string, stdin io.Reader) (*gpc.ReportingVitalsQueryRequest, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, shared.UsageErrorf("--input is required")
	}

	var (
		raw []byte
		err error
	)
	if path == "-" {
		raw, err = io.ReadAll(stdin)
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read vitals query payload: %w", err)
	}

	var payload gpc.ReportingVitalsQueryRequest
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode vitals query payload: %w", err)
	}
	return &payload, nil
}
