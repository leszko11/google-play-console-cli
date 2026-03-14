package gpc

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"google.golang.org/api/option"
	playdeveloperreporting "google.golang.org/api/playdeveloperreporting/v1beta1"
)

type ReportingClient struct {
	reporting *playdeveloperreporting.Service
}

type ReportingVitalsMetricSet string

const (
	ReportingVitalsMetricSetANRRate                     ReportingVitalsMetricSet = "anr-rate"
	ReportingVitalsMetricSetCrashRate                   ReportingVitalsMetricSet = "crash-rate"
	ReportingVitalsMetricSetErrorCounts                 ReportingVitalsMetricSet = "error-counts"
	ReportingVitalsMetricSetExcessiveWakeupRate         ReportingVitalsMetricSet = "excessive-wakeup-rate"
	ReportingVitalsMetricSetLMKRate                     ReportingVitalsMetricSet = "lmk-rate"
	ReportingVitalsMetricSetSlowRenderingRate           ReportingVitalsMetricSet = "slow-rendering-rate"
	ReportingVitalsMetricSetSlowStartRate               ReportingVitalsMetricSet = "slow-start-rate"
	ReportingVitalsMetricSetStuckBackgroundWakelockRate ReportingVitalsMetricSet = "stuck-background-wakelock-rate"
)

var reportingVitalsMetricSetResources = map[ReportingVitalsMetricSet]string{
	ReportingVitalsMetricSetANRRate:                     "anrRateMetricSet",
	ReportingVitalsMetricSetCrashRate:                   "crashRateMetricSet",
	ReportingVitalsMetricSetErrorCounts:                 "errorCountMetricSet",
	ReportingVitalsMetricSetExcessiveWakeupRate:         "excessiveWakeupRateMetricSet",
	ReportingVitalsMetricSetLMKRate:                     "lmkRateMetricSet",
	ReportingVitalsMetricSetSlowRenderingRate:           "slowRenderingRateMetricSet",
	ReportingVitalsMetricSetSlowStartRate:               "slowStartRateMetricSet",
	ReportingVitalsMetricSetStuckBackgroundWakelockRate: "stuckBackgroundWakelockRateMetricSet",
}

type ReportingFreshnessInfo = playdeveloperreporting.GooglePlayDeveloperReportingV1beta1FreshnessInfo
type ReportingTimelineSpec = playdeveloperreporting.GooglePlayDeveloperReportingV1beta1TimelineSpec
type ReportingMetricsRow = playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricsRow
type ReportingApp = playdeveloperreporting.GooglePlayDeveloperReportingV1beta1App
type ReportingAnomaly = playdeveloperreporting.GooglePlayDeveloperReportingV1beta1Anomaly
type ReportingErrorIssue = playdeveloperreporting.GooglePlayDeveloperReportingV1beta1ErrorIssue
type ReportingErrorReport = playdeveloperreporting.GooglePlayDeveloperReportingV1beta1ErrorReport

type ReportingAppsListInfo struct {
	Apps          []*ReportingApp `json:"apps,omitempty"`
	NextPageToken string          `json:"nextPageToken,omitempty"`
}

type ReportingAnomaliesListInfo struct {
	Anomalies     []*ReportingAnomaly `json:"anomalies,omitempty"`
	NextPageToken string              `json:"nextPageToken,omitempty"`
}

type ReportingInterval struct {
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
}

type ReportingErrorIssuesListInfo struct {
	Issues        []*ReportingErrorIssue `json:"issues,omitempty"`
	NextPageToken string                 `json:"nextPageToken,omitempty"`
}

type ReportingErrorReportsListInfo struct {
	Reports       []*ReportingErrorReport `json:"reports,omitempty"`
	NextPageToken string                  `json:"nextPageToken,omitempty"`
}

type ReportingVitalsMetricSetInfo struct {
	MetricSet     string                  `json:"metricSet"`
	ResourceName  string                  `json:"resourceName"`
	FreshnessInfo *ReportingFreshnessInfo `json:"freshnessInfo,omitempty"`
}

type ReportingVitalsQueryRequest struct {
	Dimensions   []string               `json:"dimensions,omitempty"`
	Filter       string                 `json:"filter,omitempty"`
	Metrics      []string               `json:"metrics,omitempty"`
	PageSize     int64                  `json:"pageSize,omitempty"`
	PageToken    string                 `json:"pageToken,omitempty"`
	TimelineSpec *ReportingTimelineSpec `json:"timelineSpec,omitempty"`
	UserCohort   string                 `json:"userCohort,omitempty"`
}

type ReportingVitalsQueryResult struct {
	MetricSet     string                 `json:"metricSet"`
	ResourceName  string                 `json:"resourceName"`
	Rows          []*ReportingMetricsRow `json:"rows,omitempty"`
	NextPageToken string                 `json:"nextPageToken,omitempty"`
}

func NewReportingClient(ctx context.Context, creds CredentialInput, opts ...option.ClientOption) (*ReportingClient, error) {
	if strings.TrimSpace(creds.ServiceAccountPath) == "" && len(creds.ServiceAccountJSON) == 0 {
		return nil, ErrInvalidCredentials
	}

	clientOpts := make([]option.ClientOption, 0, 2+len(opts))
	if strings.TrimSpace(creds.ServiceAccountPath) != "" {
		clientOpts = append(clientOpts, option.WithCredentialsFile(creds.ServiceAccountPath))
	}
	if len(creds.ServiceAccountJSON) > 0 {
		clientOpts = append(clientOpts, option.WithCredentialsJSON(creds.ServiceAccountJSON))
	}
	clientOpts = append(clientOpts, opts...)

	svc, err := playdeveloperreporting.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, err
	}

	return &ReportingClient{reporting: svc}, nil
}

func SupportedReportingVitalsMetricSets() []string {
	sets := make([]string, 0, len(reportingVitalsMetricSetResources))
	for metricSet := range reportingVitalsMetricSetResources {
		sets = append(sets, string(metricSet))
	}
	slices.Sort(sets)
	return sets
}

func ParseReportingVitalsMetricSet(raw string) (ReportingVitalsMetricSet, error) {
	metricSet := ReportingVitalsMetricSet(strings.TrimSpace(raw))
	if _, ok := reportingVitalsMetricSetResources[metricSet]; ok {
		return metricSet, nil
	}
	return "", fmt.Errorf("unsupported metric set %q (expected one of: %s)", raw, strings.Join(SupportedReportingVitalsMetricSets(), ", "))
}

func (c *ReportingClient) SearchApps(ctx context.Context, pageSize int64, pageToken string, paginate bool) (ReportingAppsListInfo, error) {
	if c == nil || c.reporting == nil {
		return ReportingAppsListInfo{}, errors.New("playdeveloperreporting service is not configured")
	}
	if pageSize < 0 {
		return ReportingAppsListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}

	pageToken = strings.TrimSpace(pageToken)
	if !paginate {
		resp, err := c.reporting.Apps.Search().PageSize(pageSize).PageToken(pageToken).Context(ctx).Do()
		if err != nil {
			return ReportingAppsListInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingAppsListInfoFromResponse(resp), nil
	}

	result := ReportingAppsListInfo{Apps: make([]*ReportingApp, 0)}
	nextToken := pageToken
	for {
		resp, err := c.reporting.Apps.Search().PageSize(pageSize).PageToken(nextToken).Context(ctx).Do()
		if err != nil {
			return ReportingAppsListInfo{}, mapReportingGoogleAPIError(err)
		}
		page := reportingAppsListInfoFromResponse(resp)
		result.Apps = append(result.Apps, page.Apps...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return ReportingAppsListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *ReportingClient) ListAnomalies(ctx context.Context, packageName, filter string, pageSize int64, pageToken string, paginate bool) (ReportingAnomaliesListInfo, error) {
	if c == nil || c.reporting == nil {
		return ReportingAnomaliesListInfo{}, errors.New("playdeveloperreporting service is not configured")
	}
	if pageSize < 0 {
		return ReportingAnomaliesListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}

	parent, err := reportingAppParent(packageName)
	if err != nil {
		return ReportingAnomaliesListInfo{}, err
	}
	filter = strings.TrimSpace(filter)
	pageToken = strings.TrimSpace(pageToken)

	if !paginate {
		call := c.reporting.Anomalies.List(parent).PageSize(pageSize).PageToken(pageToken).Context(ctx)
		if filter != "" {
			call.Filter(filter)
		}
		resp, err := call.Do()
		if err != nil {
			return ReportingAnomaliesListInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingAnomaliesListInfoFromResponse(resp), nil
	}

	result := ReportingAnomaliesListInfo{Anomalies: make([]*ReportingAnomaly, 0)}
	nextToken := pageToken
	for {
		call := c.reporting.Anomalies.List(parent).PageSize(pageSize).PageToken(nextToken).Context(ctx)
		if filter != "" {
			call.Filter(filter)
		}
		resp, err := call.Do()
		if err != nil {
			return ReportingAnomaliesListInfo{}, mapReportingGoogleAPIError(err)
		}
		page := reportingAnomaliesListInfoFromResponse(resp)
		result.Anomalies = append(result.Anomalies, page.Anomalies...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return ReportingAnomaliesListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *ReportingClient) SearchErrorIssues(ctx context.Context, packageName, filter, orderBy string, interval ReportingInterval, pageSize, sampleErrorReportLimit int64, pageToken string, paginate bool) (ReportingErrorIssuesListInfo, error) {
	if c == nil || c.reporting == nil {
		return ReportingErrorIssuesListInfo{}, errors.New("playdeveloperreporting service is not configured")
	}
	if pageSize < 0 {
		return ReportingErrorIssuesListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}
	if sampleErrorReportLimit < 0 {
		return ReportingErrorIssuesListInfo{}, fmt.Errorf("sample error report limit must be greater than or equal to zero")
	}

	parent, err := reportingAppParent(packageName)
	if err != nil {
		return ReportingErrorIssuesListInfo{}, err
	}
	filter = strings.TrimSpace(filter)
	orderBy = strings.TrimSpace(orderBy)
	pageToken = strings.TrimSpace(pageToken)

	call := c.reporting.Vitals.Errors.Issues.Search(parent).PageSize(pageSize).PageToken(pageToken).SampleErrorReportLimit(sampleErrorReportLimit).Context(ctx)
	if filter != "" {
		call.Filter(filter)
	}
	if orderBy != "" {
		call.OrderBy(orderBy)
	}
	applyReportingIssuesInterval(call, interval)

	if !paginate {
		resp, err := call.Do()
		if err != nil {
			return ReportingErrorIssuesListInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingErrorIssuesListInfoFromResponse(resp), nil
	}

	result := ReportingErrorIssuesListInfo{Issues: make([]*ReportingErrorIssue, 0)}
	nextToken := pageToken
	for {
		call := c.reporting.Vitals.Errors.Issues.Search(parent).PageSize(pageSize).PageToken(nextToken).SampleErrorReportLimit(sampleErrorReportLimit).Context(ctx)
		if filter != "" {
			call.Filter(filter)
		}
		if orderBy != "" {
			call.OrderBy(orderBy)
		}
		applyReportingIssuesInterval(call, interval)
		resp, err := call.Do()
		if err != nil {
			return ReportingErrorIssuesListInfo{}, mapReportingGoogleAPIError(err)
		}
		page := reportingErrorIssuesListInfoFromResponse(resp)
		result.Issues = append(result.Issues, page.Issues...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return ReportingErrorIssuesListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *ReportingClient) SearchErrorReports(ctx context.Context, packageName, filter string, interval ReportingInterval, pageSize int64, pageToken string, paginate bool) (ReportingErrorReportsListInfo, error) {
	if c == nil || c.reporting == nil {
		return ReportingErrorReportsListInfo{}, errors.New("playdeveloperreporting service is not configured")
	}
	if pageSize < 0 {
		return ReportingErrorReportsListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}

	parent, err := reportingAppParent(packageName)
	if err != nil {
		return ReportingErrorReportsListInfo{}, err
	}
	filter = strings.TrimSpace(filter)
	pageToken = strings.TrimSpace(pageToken)

	call := c.reporting.Vitals.Errors.Reports.Search(parent).PageSize(pageSize).PageToken(pageToken).Context(ctx)
	if filter != "" {
		call.Filter(filter)
	}
	applyReportingReportsInterval(call, interval)

	if !paginate {
		resp, err := call.Do()
		if err != nil {
			return ReportingErrorReportsListInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingErrorReportsListInfoFromResponse(resp), nil
	}

	result := ReportingErrorReportsListInfo{Reports: make([]*ReportingErrorReport, 0)}
	nextToken := pageToken
	for {
		call := c.reporting.Vitals.Errors.Reports.Search(parent).PageSize(pageSize).PageToken(nextToken).Context(ctx)
		if filter != "" {
			call.Filter(filter)
		}
		applyReportingReportsInterval(call, interval)
		resp, err := call.Do()
		if err != nil {
			return ReportingErrorReportsListInfo{}, mapReportingGoogleAPIError(err)
		}
		page := reportingErrorReportsListInfoFromResponse(resp)
		result.Reports = append(result.Reports, page.Reports...)
		if page.NextPageToken == "" {
			result.NextPageToken = ""
			return result, nil
		}
		if page.NextPageToken == nextToken {
			return ReportingErrorReportsListInfo{}, fmt.Errorf("pagination token did not advance")
		}
		nextToken = page.NextPageToken
	}
}

func (c *ReportingClient) GetVitalsMetricSet(ctx context.Context, packageName string, metricSet ReportingVitalsMetricSet) (ReportingVitalsMetricSetInfo, error) {
	if c == nil || c.reporting == nil {
		return ReportingVitalsMetricSetInfo{}, errors.New("playdeveloperreporting service is not configured")
	}

	resourceName, err := reportingVitalsMetricSetResourceName(packageName, metricSet)
	if err != nil {
		return ReportingVitalsMetricSetInfo{}, err
	}

	switch metricSet {
	case ReportingVitalsMetricSetANRRate:
		resp, err := c.reporting.Vitals.Anrrate.Get(resourceName).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsMetricSetInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsMetricSetInfo(metricSet, resp.Name, resp.FreshnessInfo), nil
	case ReportingVitalsMetricSetCrashRate:
		resp, err := c.reporting.Vitals.Crashrate.Get(resourceName).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsMetricSetInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsMetricSetInfo(metricSet, resp.Name, resp.FreshnessInfo), nil
	case ReportingVitalsMetricSetErrorCounts:
		resp, err := c.reporting.Vitals.Errors.Counts.Get(resourceName).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsMetricSetInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsMetricSetInfo(metricSet, resp.Name, resp.FreshnessInfo), nil
	case ReportingVitalsMetricSetExcessiveWakeupRate:
		resp, err := c.reporting.Vitals.Excessivewakeuprate.Get(resourceName).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsMetricSetInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsMetricSetInfo(metricSet, resp.Name, resp.FreshnessInfo), nil
	case ReportingVitalsMetricSetLMKRate:
		resp, err := c.reporting.Vitals.Lmkrate.Get(resourceName).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsMetricSetInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsMetricSetInfo(metricSet, resp.Name, resp.FreshnessInfo), nil
	case ReportingVitalsMetricSetSlowRenderingRate:
		resp, err := c.reporting.Vitals.Slowrenderingrate.Get(resourceName).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsMetricSetInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsMetricSetInfo(metricSet, resp.Name, resp.FreshnessInfo), nil
	case ReportingVitalsMetricSetSlowStartRate:
		resp, err := c.reporting.Vitals.Slowstartrate.Get(resourceName).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsMetricSetInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsMetricSetInfo(metricSet, resp.Name, resp.FreshnessInfo), nil
	case ReportingVitalsMetricSetStuckBackgroundWakelockRate:
		resp, err := c.reporting.Vitals.Stuckbackgroundwakelockrate.Get(resourceName).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsMetricSetInfo{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsMetricSetInfo(metricSet, resp.Name, resp.FreshnessInfo), nil
	default:
		return ReportingVitalsMetricSetInfo{}, fmt.Errorf("unsupported metric set %q", metricSet)
	}
}

func (c *ReportingClient) QueryVitalsMetricSet(ctx context.Context, packageName string, metricSet ReportingVitalsMetricSet, request *ReportingVitalsQueryRequest) (ReportingVitalsQueryResult, error) {
	if c == nil || c.reporting == nil {
		return ReportingVitalsQueryResult{}, errors.New("playdeveloperreporting service is not configured")
	}

	resourceName, err := reportingVitalsMetricSetResourceName(packageName, metricSet)
	if err != nil {
		return ReportingVitalsQueryResult{}, err
	}

	payload := reportingVitalsQueryRequestOrDefault(request)
	switch metricSet {
	case ReportingVitalsMetricSetANRRate:
		resp, err := c.reporting.Vitals.Anrrate.Query(resourceName, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryAnrRateMetricSetRequest{
			Dimensions:   payload.Dimensions,
			Filter:       payload.Filter,
			Metrics:      payload.Metrics,
			PageSize:     payload.PageSize,
			PageToken:    payload.PageToken,
			TimelineSpec: payload.TimelineSpec,
			UserCohort:   payload.UserCohort,
		}).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsQueryResult{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsQueryResult(metricSet, resourceName, resp.Rows, resp.NextPageToken), nil
	case ReportingVitalsMetricSetCrashRate:
		resp, err := c.reporting.Vitals.Crashrate.Query(resourceName, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryCrashRateMetricSetRequest{
			Dimensions:   payload.Dimensions,
			Filter:       payload.Filter,
			Metrics:      payload.Metrics,
			PageSize:     payload.PageSize,
			PageToken:    payload.PageToken,
			TimelineSpec: payload.TimelineSpec,
			UserCohort:   payload.UserCohort,
		}).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsQueryResult{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsQueryResult(metricSet, resourceName, resp.Rows, resp.NextPageToken), nil
	case ReportingVitalsMetricSetErrorCounts:
		resp, err := c.reporting.Vitals.Errors.Counts.Query(resourceName, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryErrorCountMetricSetRequest{
			Dimensions:   payload.Dimensions,
			Filter:       payload.Filter,
			Metrics:      payload.Metrics,
			PageSize:     payload.PageSize,
			PageToken:    payload.PageToken,
			TimelineSpec: payload.TimelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsQueryResult{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsQueryResult(metricSet, resourceName, resp.Rows, resp.NextPageToken), nil
	case ReportingVitalsMetricSetExcessiveWakeupRate:
		resp, err := c.reporting.Vitals.Excessivewakeuprate.Query(resourceName, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryExcessiveWakeupRateMetricSetRequest{
			Dimensions:   payload.Dimensions,
			Filter:       payload.Filter,
			Metrics:      payload.Metrics,
			PageSize:     payload.PageSize,
			PageToken:    payload.PageToken,
			TimelineSpec: payload.TimelineSpec,
			UserCohort:   payload.UserCohort,
		}).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsQueryResult{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsQueryResult(metricSet, resourceName, resp.Rows, resp.NextPageToken), nil
	case ReportingVitalsMetricSetLMKRate:
		resp, err := c.reporting.Vitals.Lmkrate.Query(resourceName, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryLmkRateMetricSetRequest{
			Dimensions:   payload.Dimensions,
			Filter:       payload.Filter,
			Metrics:      payload.Metrics,
			PageSize:     payload.PageSize,
			PageToken:    payload.PageToken,
			TimelineSpec: payload.TimelineSpec,
			UserCohort:   payload.UserCohort,
		}).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsQueryResult{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsQueryResult(metricSet, resourceName, resp.Rows, resp.NextPageToken), nil
	case ReportingVitalsMetricSetSlowRenderingRate:
		resp, err := c.reporting.Vitals.Slowrenderingrate.Query(resourceName, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QuerySlowRenderingRateMetricSetRequest{
			Dimensions:   payload.Dimensions,
			Filter:       payload.Filter,
			Metrics:      payload.Metrics,
			PageSize:     payload.PageSize,
			PageToken:    payload.PageToken,
			TimelineSpec: payload.TimelineSpec,
			UserCohort:   payload.UserCohort,
		}).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsQueryResult{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsQueryResult(metricSet, resourceName, resp.Rows, resp.NextPageToken), nil
	case ReportingVitalsMetricSetSlowStartRate:
		resp, err := c.reporting.Vitals.Slowstartrate.Query(resourceName, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QuerySlowStartRateMetricSetRequest{
			Dimensions:   payload.Dimensions,
			Filter:       payload.Filter,
			Metrics:      payload.Metrics,
			PageSize:     payload.PageSize,
			PageToken:    payload.PageToken,
			TimelineSpec: payload.TimelineSpec,
			UserCohort:   payload.UserCohort,
		}).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsQueryResult{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsQueryResult(metricSet, resourceName, resp.Rows, resp.NextPageToken), nil
	case ReportingVitalsMetricSetStuckBackgroundWakelockRate:
		resp, err := c.reporting.Vitals.Stuckbackgroundwakelockrate.Query(resourceName, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryStuckBackgroundWakelockRateMetricSetRequest{
			Dimensions:   payload.Dimensions,
			Filter:       payload.Filter,
			Metrics:      payload.Metrics,
			PageSize:     payload.PageSize,
			PageToken:    payload.PageToken,
			TimelineSpec: payload.TimelineSpec,
			UserCohort:   payload.UserCohort,
		}).Context(ctx).Do()
		if err != nil {
			return ReportingVitalsQueryResult{}, mapReportingGoogleAPIError(err)
		}
		return reportingVitalsQueryResult(metricSet, resourceName, resp.Rows, resp.NextPageToken), nil
	default:
		return ReportingVitalsQueryResult{}, fmt.Errorf("unsupported metric set %q", metricSet)
	}
}

func reportingVitalsMetricSetResourceName(packageName string, metricSet ReportingVitalsMetricSet) (string, error) {
	trimmedPackage := strings.TrimSpace(packageName)
	if trimmedPackage == "" {
		return "", fmt.Errorf("package name is required")
	}
	resourceSuffix, ok := reportingVitalsMetricSetResources[metricSet]
	if !ok {
		return "", fmt.Errorf("unsupported metric set %q", metricSet)
	}
	return fmt.Sprintf("apps/%s/%s", trimmedPackage, resourceSuffix), nil
}

func reportingVitalsMetricSetInfo(metricSet ReportingVitalsMetricSet, resourceName string, freshness *ReportingFreshnessInfo) ReportingVitalsMetricSetInfo {
	return ReportingVitalsMetricSetInfo{
		MetricSet:     string(metricSet),
		ResourceName:  resourceName,
		FreshnessInfo: freshness,
	}
}

func reportingVitalsQueryResult(metricSet ReportingVitalsMetricSet, resourceName string, rows []*ReportingMetricsRow, nextPageToken string) ReportingVitalsQueryResult {
	return ReportingVitalsQueryResult{
		MetricSet:     string(metricSet),
		ResourceName:  resourceName,
		Rows:          rows,
		NextPageToken: nextPageToken,
	}
}

func reportingVitalsQueryRequestOrDefault(request *ReportingVitalsQueryRequest) ReportingVitalsQueryRequest {
	if request == nil {
		return ReportingVitalsQueryRequest{}
	}
	return ReportingVitalsQueryRequest{
		Dimensions:   append([]string(nil), request.Dimensions...),
		Filter:       request.Filter,
		Metrics:      append([]string(nil), request.Metrics...),
		PageSize:     request.PageSize,
		PageToken:    request.PageToken,
		TimelineSpec: request.TimelineSpec,
		UserCohort:   request.UserCohort,
	}
}

func mapReportingGoogleAPIError(err error) error {
	return mapGoogleAPIErrorWithService("playdeveloperreporting", err, false)
}

func reportingAppParent(packageName string) (string, error) {
	trimmed := strings.TrimSpace(packageName)
	if trimmed == "" {
		return "", fmt.Errorf("package name is required")
	}
	return "apps/" + trimmed, nil
}

func reportingAppsListInfoFromResponse(resp *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1SearchAccessibleAppsResponse) ReportingAppsListInfo {
	if resp == nil {
		return ReportingAppsListInfo{}
	}
	return ReportingAppsListInfo{
		Apps:          resp.Apps,
		NextPageToken: resp.NextPageToken,
	}
}

func reportingAnomaliesListInfoFromResponse(resp *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1ListAnomaliesResponse) ReportingAnomaliesListInfo {
	if resp == nil {
		return ReportingAnomaliesListInfo{}
	}
	return ReportingAnomaliesListInfo{
		Anomalies:     resp.Anomalies,
		NextPageToken: resp.NextPageToken,
	}
}

func reportingErrorIssuesListInfoFromResponse(resp *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1SearchErrorIssuesResponse) ReportingErrorIssuesListInfo {
	if resp == nil {
		return ReportingErrorIssuesListInfo{}
	}
	return ReportingErrorIssuesListInfo{
		Issues:        resp.ErrorIssues,
		NextPageToken: resp.NextPageToken,
	}
}

func reportingErrorReportsListInfoFromResponse(resp *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1SearchErrorReportsResponse) ReportingErrorReportsListInfo {
	if resp == nil {
		return ReportingErrorReportsListInfo{}
	}
	return ReportingErrorReportsListInfo{
		Reports:       resp.ErrorReports,
		NextPageToken: resp.NextPageToken,
	}
}

func applyReportingIssuesInterval(call *playdeveloperreporting.VitalsErrorsIssuesSearchCall, interval ReportingInterval) {
	if call == nil {
		return
	}
	start := strings.TrimSpace(interval.StartTime)
	if start != "" {
		if parsed, err := parseReportingIntervalTime(start); err == nil {
			call.IntervalStartTimeYear(int64(parsed.Year()))
			call.IntervalStartTimeMonth(int64(parsed.Month()))
			call.IntervalStartTimeDay(int64(parsed.Day()))
			call.IntervalStartTimeHours(int64(parsed.Hour()))
			call.IntervalStartTimeMinutes(int64(parsed.Minute()))
			call.IntervalStartTimeSeconds(int64(parsed.Second()))
			call.IntervalStartTimeNanos(int64(parsed.Nanosecond()))
			call.IntervalStartTimeTimeZoneId(parsed.Location().String())
		}
	}
	end := strings.TrimSpace(interval.EndTime)
	if end != "" {
		if parsed, err := parseReportingIntervalTime(end); err == nil {
			call.IntervalEndTimeYear(int64(parsed.Year()))
			call.IntervalEndTimeMonth(int64(parsed.Month()))
			call.IntervalEndTimeDay(int64(parsed.Day()))
			call.IntervalEndTimeHours(int64(parsed.Hour()))
			call.IntervalEndTimeMinutes(int64(parsed.Minute()))
			call.IntervalEndTimeSeconds(int64(parsed.Second()))
			call.IntervalEndTimeNanos(int64(parsed.Nanosecond()))
			call.IntervalEndTimeTimeZoneId(parsed.Location().String())
		}
	}
}

func applyReportingReportsInterval(call *playdeveloperreporting.VitalsErrorsReportsSearchCall, interval ReportingInterval) {
	if call == nil {
		return
	}
	start := strings.TrimSpace(interval.StartTime)
	if start != "" {
		if parsed, err := parseReportingIntervalTime(start); err == nil {
			call.IntervalStartTimeYear(int64(parsed.Year()))
			call.IntervalStartTimeMonth(int64(parsed.Month()))
			call.IntervalStartTimeDay(int64(parsed.Day()))
			call.IntervalStartTimeHours(int64(parsed.Hour()))
			call.IntervalStartTimeMinutes(int64(parsed.Minute()))
			call.IntervalStartTimeSeconds(int64(parsed.Second()))
			call.IntervalStartTimeNanos(int64(parsed.Nanosecond()))
			call.IntervalStartTimeTimeZoneId(parsed.Location().String())
		}
	}
	end := strings.TrimSpace(interval.EndTime)
	if end != "" {
		if parsed, err := parseReportingIntervalTime(end); err == nil {
			call.IntervalEndTimeYear(int64(parsed.Year()))
			call.IntervalEndTimeMonth(int64(parsed.Month()))
			call.IntervalEndTimeDay(int64(parsed.Day()))
			call.IntervalEndTimeHours(int64(parsed.Hour()))
			call.IntervalEndTimeMinutes(int64(parsed.Minute()))
			call.IntervalEndTimeSeconds(int64(parsed.Second()))
			call.IntervalEndTimeNanos(int64(parsed.Nanosecond()))
			call.IntervalEndTimeTimeZoneId(parsed.Location().String())
		}
	}
}

func parseReportingIntervalTime(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.TrimSpace(raw))
}
