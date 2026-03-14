package release

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
	playdeveloperreporting "google.golang.org/api/playdeveloperreporting/v1beta1"
)

const (
	defaultVitalsGateWindow   = 24 * time.Hour
	defaultVitalsPollInterval = 5 * time.Minute
)

var errVitalsValueUnavailable = errors.New("vitals value unavailable")

type vitalsGateCondition struct {
	Metric      string                       `json:"metric"`
	MetricSet   gpc.ReportingVitalsMetricSet `json:"-"`
	QueryMetric string                       `json:"-"`
	Operator    string                       `json:"operator"`
	Threshold   float64                      `json:"threshold"`
}

type fullVitalsGateCheck struct {
	Metric    string   `json:"metric"`
	Operator  string   `json:"operator"`
	Threshold float64  `json:"threshold"`
	Actual    *float64 `json:"actual,omitempty"`
	Passed    bool     `json:"passed"`
}

type fullVitalsGateResult struct {
	Status      string                `json:"status"`
	Wait        string                `json:"wait,omitempty"`
	AutoHalt    bool                  `json:"autoHaltOnRegression,omitempty"`
	Evaluations int                   `json:"evaluations,omitempty"`
	Halted      bool                  `json:"halted,omitempty"`
	Checks      []fullVitalsGateCheck `json:"checks,omitempty"`
}

func parseVitalsGate(raw string) ([]vitalsGateCondition, error) {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	conditions := make([]vitalsGateCondition, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var operator string
		for _, candidate := range []string{"<=", ">=", "<", ">"} {
			if strings.Contains(part, candidate) {
				operator = candidate
				break
			}
		}
		if operator == "" {
			return nil, fmt.Errorf("invalid vitals gate condition %q", part)
		}

		metricName, thresholdRaw, ok := strings.Cut(part, operator)
		if !ok {
			return nil, fmt.Errorf("invalid vitals gate condition %q", part)
		}

		metricName = strings.TrimSpace(metricName)
		threshold, err := strconv.ParseFloat(strings.TrimSpace(thresholdRaw), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid threshold for %q: %w", metricName, err)
		}

		condition, err := newVitalsGateCondition(metricName, operator, threshold)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
	}

	if len(conditions) == 0 {
		return nil, fmt.Errorf("at least one vitals gate condition is required")
	}
	return conditions, nil
}

func newVitalsGateCondition(metricName, operator string, threshold float64) (vitalsGateCondition, error) {
	switch strings.TrimSpace(metricName) {
	case "crashRate":
		return vitalsGateCondition{
			Metric:      "crashRate",
			MetricSet:   gpc.ReportingVitalsMetricSetCrashRate,
			QueryMetric: "crashRate",
			Operator:    operator,
			Threshold:   threshold,
		}, nil
	case "anrRate":
		return vitalsGateCondition{
			Metric:      "anrRate",
			MetricSet:   gpc.ReportingVitalsMetricSetANRRate,
			QueryMetric: "anrRate",
			Operator:    operator,
			Threshold:   threshold,
		}, nil
	default:
		return vitalsGateCondition{}, fmt.Errorf("unsupported vitals gate metric %q (expected crashRate or anrRate)", metricName)
	}
}

func defaultVitalsGateQuery(now time.Time, metric string) *gpc.ReportingVitalsQueryRequest {
	end := now.UTC()
	start := end.Add(-defaultVitalsGateWindow)
	return &gpc.ReportingVitalsQueryRequest{
		Metrics: []string{metric},
		TimelineSpec: &gpc.ReportingTimelineSpec{
			AggregationPeriod: "FULL_RANGE",
			StartTime: &playdeveloperreporting.GoogleTypeDateTime{
				Year:  int64(start.Year()),
				Month: int64(start.Month()),
				Day:   int64(start.Day()),
			},
			EndTime: &playdeveloperreporting.GoogleTypeDateTime{
				Year:  int64(end.Year()),
				Month: int64(end.Month()),
				Day:   int64(end.Day()),
			},
		},
	}
}

func evaluateVitalsGate(ctx context.Context, client ReportingClient, packageName string, now time.Time, conditions []vitalsGateCondition) ([]fullVitalsGateCheck, error) {
	checks := make([]fullVitalsGateCheck, 0, len(conditions))
	for _, condition := range conditions {
		result, err := client.QueryVitalsMetricSet(ctx, packageName, condition.MetricSet, defaultVitalsGateQuery(now, condition.QueryMetric))
		if err != nil {
			return nil, err
		}
		actual, err := extractVitalsMetricValue(result.Rows, condition.QueryMetric)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errVitalsValueUnavailable, err)
		}
		passed := compareVitalsGate(actual, condition.Operator, condition.Threshold)
		checks = append(checks, fullVitalsGateCheck{
			Metric:    condition.Metric,
			Operator:  condition.Operator,
			Threshold: condition.Threshold,
			Actual:    &actual,
			Passed:    passed,
		})
	}
	return checks, nil
}

func extractVitalsMetricValue(rows []*gpc.ReportingMetricsRow, metric string) (float64, error) {
	for _, row := range rows {
		if row == nil {
			continue
		}
		for _, value := range row.Metrics {
			if value == nil || strings.TrimSpace(value.Metric) != metric || value.DecimalValue == nil {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value.DecimalValue.Value), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid %s value %q: %w", metric, value.DecimalValue.Value, err)
			}
			return parsed, nil
		}
	}
	return 0, fmt.Errorf("no %s metric value returned", metric)
}

func compareVitalsGate(actual float64, operator string, threshold float64) bool {
	switch operator {
	case "<":
		return actual < threshold
	case "<=":
		return actual <= threshold
	case ">":
		return actual > threshold
	case ">=":
		return actual >= threshold
	default:
		return false
	}
}
