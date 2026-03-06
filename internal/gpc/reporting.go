package gpc

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

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
