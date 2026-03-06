package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewReportingClient_RejectsMissingCredentials(t *testing.T) {
	_, err := NewReportingClient(context.Background(), CredentialInput{})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestParseReportingVitalsMetricSet(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ReportingVitalsMetricSet
	}{
		{name: "anr", input: "anr-rate", want: ReportingVitalsMetricSetANRRate},
		{name: "crash", input: "crash-rate", want: ReportingVitalsMetricSetCrashRate},
		{name: "error-counts", input: "error-counts", want: ReportingVitalsMetricSetErrorCounts},
		{name: "excessive-wakeup", input: "excessive-wakeup-rate", want: ReportingVitalsMetricSetExcessiveWakeupRate},
		{name: "lmk", input: "lmk-rate", want: ReportingVitalsMetricSetLMKRate},
		{name: "slow-rendering", input: "slow-rendering-rate", want: ReportingVitalsMetricSetSlowRenderingRate},
		{name: "slow-start", input: "slow-start-rate", want: ReportingVitalsMetricSetSlowStartRate},
		{name: "stuck-wakelock", input: "stuck-background-wakelock-rate", want: ReportingVitalsMetricSetStuckBackgroundWakelockRate},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseReportingVitalsMetricSet(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestParseReportingVitalsMetricSet_RejectsUnknown(t *testing.T) {
	_, err := ParseReportingVitalsMetricSet("unknown")
	if err == nil || !strings.Contains(err.Error(), "unsupported metric set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReportingVitalsMetricSetResourceName(t *testing.T) {
	got, err := reportingVitalsMetricSetResourceName("com.example.app", ReportingVitalsMetricSetCrashRate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "apps/com.example.app/crashRateMetricSet" {
		t.Fatalf("unexpected resource name: %q", got)
	}
}

func TestReportingClientGetVitalsMetricSet_RequiresService(t *testing.T) {
	client := &ReportingClient{}
	_, err := client.GetVitalsMetricSet(context.Background(), "com.example.app", ReportingVitalsMetricSetCrashRate)
	if err == nil || !strings.Contains(err.Error(), "service is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReportingClientQueryVitalsMetricSet_RequiresService(t *testing.T) {
	client := &ReportingClient{}
	_, err := client.QueryVitalsMetricSet(context.Background(), "com.example.app", ReportingVitalsMetricSetCrashRate, &ReportingVitalsQueryRequest{})
	if err == nil || !strings.Contains(err.Error(), "service is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}
