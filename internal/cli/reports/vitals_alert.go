package reports

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/release"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type vitalsAlertResult struct {
	Status        string                    `json:"status"`
	PackageName   string                    `json:"packageName"`
	FailOn        string                    `json:"failOn"`
	BreachCount   int                       `json:"breachCount"`
	Checks        []release.VitalsGateCheck `json:"checks"`
	Notifications []alertDeliveryResult     `json:"notifications,omitempty"`
}

type alertDeliveryResult struct {
	Provider   string `json:"provider"`
	URL        string `json:"url,omitempty"`
	Status     string `json:"status"`
	StatusCode int    `json:"statusCode,omitempty"`
}

func newVitalsAlertCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("alert", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName, vitalsGate, failOn, slackWebhook, webhook string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&vitalsGate, "vitals-gate", "", "Vitals threshold gate, for example: crashRate<2.0,anrRate<0.5")
	fs.StringVar(&failOn, "fail-on", "warning", "Exit non-zero on warning, critical, or none")
	fs.StringVar(&slackWebhook, "slack-webhook", "", "Slack webhook URL for breach notifications")
	fs.StringVar(&webhook, "webhook", "", "Generic webhook URL for breach notifications")

	return &ffcli.Command{
		Name:      "alert",
		ShortHelp: "Evaluate vitals thresholds and optionally deliver breach alerts",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			conditions, err := release.ParseVitalsGate(vitalsGate)
			if err != nil {
				return shared.UsageErrorf("%v", err)
			}
			failOn, err = normalizeFailOn(failOn)
			if err != nil {
				return err
			}

			checks, err := release.EvaluateVitalsGate(requestCtx, client, pkg, deps.Now(), conditions)
			if err != nil {
				return fmt.Errorf("failed to evaluate vitals alert: %w", err)
			}

			result := buildVitalsAlertResult(pkg, failOn, checks)
			if result.Status != "ok" {
				result.Notifications, err = sendVitalsAlertNotifications(requestCtx, deps, result, slackWebhook, webhook)
				if err != nil {
					return err
				}
			}

			switch shared.ResolveOutput("") {
			case "json":
				if err := shared.WriteJSON(deps.Stdout, result); err != nil {
					return err
				}
			case "table":
				if err := writeVitalsAlertTable(deps.Stdout, result); err != nil {
					return err
				}
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(""))
			}

			if shouldFailVitalsAlert(result.Status, failOn) {
				return fmt.Errorf("vitals alert status %s breached fail-on=%s", result.Status, failOn)
			}
			return nil
		},
	}
}

func normalizeFailOn(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "warning", "critical", "none":
		return strings.ToLower(strings.TrimSpace(raw)), nil
	default:
		return "", shared.UsageErrorf("--fail-on must be one of warning, critical, none")
	}
}

func buildVitalsAlertResult(packageName, failOn string, checks []release.VitalsGateCheck) vitalsAlertResult {
	result := vitalsAlertResult{
		Status:      "ok",
		PackageName: packageName,
		FailOn:      failOn,
		Checks:      checks,
	}
	for _, check := range checks {
		if check.Passed {
			continue
		}
		result.BreachCount++
	}
	switch {
	case result.BreachCount == 0:
		result.Status = "ok"
	case result.BreachCount == 1:
		result.Status = "warning"
	default:
		result.Status = "critical"
	}
	return result
}

func shouldFailVitalsAlert(status, failOn string) bool {
	switch failOn {
	case "none":
		return false
	case "critical":
		return status == "critical"
	default:
		return status == "warning" || status == "critical"
	}
}

func sendVitalsAlertNotifications(ctx context.Context, deps Deps, result vitalsAlertResult, slackWebhook, webhook string) ([]alertDeliveryResult, error) {
	deliveries := make([]alertDeliveryResult, 0, 2)

	if url := strings.TrimSpace(slackWebhook); url != "" {
		payload, err := json.Marshal(map[string]string{
			"text": buildVitalsAlertSlackMessage(result),
		})
		if err != nil {
			return nil, err
		}
		delivery, err := postAlertPayload(ctx, deps.HTTPClient, url, payload)
		if err != nil {
			return nil, fmt.Errorf("failed to deliver Slack alert: %w", err)
		}
		deliveries = append(deliveries, alertDeliveryResult{
			Provider:   "slack",
			URL:        url,
			Status:     "delivered",
			StatusCode: delivery,
		})
	}

	if url := strings.TrimSpace(webhook); url != "" {
		payload, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		delivery, err := postAlertPayload(ctx, deps.HTTPClient, url, payload)
		if err != nil {
			return nil, fmt.Errorf("failed to deliver webhook alert: %w", err)
		}
		deliveries = append(deliveries, alertDeliveryResult{
			Provider:   "webhook",
			URL:        url,
			Status:     "delivered",
			StatusCode: delivery,
		})
	}

	return deliveries, nil
}

func postAlertPayload(ctx context.Context, client httpDoer, url string, payload []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GPC-Source", "gpc")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return 0, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp.StatusCode, nil
}

func buildVitalsAlertSlackMessage(result vitalsAlertResult) string {
	parts := []string{
		fmt.Sprintf("GPC vitals alert %s for %s", strings.ToUpper(result.Status), result.PackageName),
	}
	for _, check := range result.Checks {
		actual := "n/a"
		if check.Actual != nil {
			actual = fmt.Sprintf("%.3f", *check.Actual)
		}
		state := "pass"
		if !check.Passed {
			state = "breach"
		}
		parts = append(parts, fmt.Sprintf("%s %s %.3f actual=%s %s", check.Metric, check.Operator, check.Threshold, actual, state))
	}
	return strings.Join(parts, "\n")
}

func writeVitalsAlertTable(out io.Writer, result vitalsAlertResult) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", result.PackageName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "FAIL_ON\t%s\n", result.FailOn); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "BREACHES\t%d\n", result.BreachCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "CHECK\tOPERATOR\tTHRESHOLD\tACTUAL\tPASSED"); err != nil {
		return err
	}
	for _, check := range result.Checks {
		actual := "-"
		if check.Actual != nil {
			actual = fmt.Sprintf("%.3f", *check.Actual)
		}
		if _, err := fmt.Fprintf(out, "%s\t%s\t%.3f\t%s\t%t\n", check.Metric, check.Operator, check.Threshold, actual, check.Passed); err != nil {
			return err
		}
	}
	for _, notification := range result.Notifications {
		if _, err := fmt.Fprintf(out, "NOTIFICATION\t%s\t%s\t%d\n", notification.Provider, notification.Status, notification.StatusCode); err != nil {
			return err
		}
	}
	return nil
}
