package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const (
	defaultRetryDelay = time.Second
	maxBodyPreview    = 4096
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Deps struct {
	HTTPClient httpDoer
	Sleep      func(context.Context, time.Duration) error
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

type webhookOptions struct {
	URL           string
	Event         string
	InputPath     string
	RetryAttempts int
	RetryDelay    time.Duration
}

type webhookResult struct {
	URL          string `json:"url"`
	Event        string `json:"event"`
	StatusCode   int    `json:"statusCode"`
	Status       string `json:"status"`
	Attempts     int    `json:"attempts"`
	PayloadBytes int    `json:"payloadBytes"`
	ResponseBody string `json:"responseBody,omitempty"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "notify",
		ShortHelp: "Notification delivery helpers",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newWebhookCommand(deps),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.HTTPClient == nil {
		deps.HTTPClient = http.DefaultClient
	}
	if deps.Sleep == nil {
		deps.Sleep = sleepContext
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

func newWebhookCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("webhook", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts webhookOptions
	fs.StringVar(&opts.URL, "url", "", "Webhook URL")
	fs.StringVar(&opts.Event, "event", "", "Event name metadata")
	fs.StringVar(&opts.InputPath, "input", "", "Path to JSON payload file (use - for stdin)")
	fs.IntVar(&opts.RetryAttempts, "retry-attempts", 0, "Additional retry attempts for network, 429, or 5xx failures")
	fs.DurationVar(&opts.RetryDelay, "retry-delay", defaultRetryDelay, "Delay between retry attempts")

	return &ffcli.Command{
		Name:      "webhook",
		ShortHelp: "POST a JSON payload to a webhook endpoint",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateWebhookOptions(opts)
			if err != nil {
				return err
			}
			payload, err := readWebhookPayload(opts.InputPath, deps.Stdin)
			if err != nil {
				return err
			}
			return runWebhook(ctx, deps, opts, payload)
		},
	}
}

func validateWebhookOptions(opts webhookOptions) (webhookOptions, error) {
	opts.URL = strings.TrimSpace(opts.URL)
	if opts.URL == "" {
		return webhookOptions{}, shared.UsageErrorf("--url is required")
	}
	opts.Event = strings.TrimSpace(opts.Event)
	if opts.Event == "" {
		return webhookOptions{}, shared.UsageErrorf("--event is required")
	}
	opts.InputPath = strings.TrimSpace(opts.InputPath)
	if opts.InputPath == "" {
		return webhookOptions{}, shared.UsageErrorf("--input is required")
	}
	if opts.RetryAttempts < 0 {
		return webhookOptions{}, shared.UsageErrorf("--retry-attempts must be zero or greater")
	}
	if opts.RetryDelay <= 0 {
		return webhookOptions{}, shared.UsageErrorf("--retry-delay must be greater than zero")
	}
	return opts, nil
}

func readWebhookPayload(path string, stdin io.Reader) ([]byte, error) {
	var raw []byte
	var err error

	switch strings.TrimSpace(path) {
	case "-":
		if stdin == nil {
			stdin = os.Stdin
		}
		raw, err = io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read --input from stdin: %w", err)
		}
	default:
		raw, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read --input: %w", err)
		}
	}

	if !json.Valid(raw) {
		return nil, shared.UsageErrorf("--input must contain valid JSON")
	}
	return raw, nil
}

func runWebhook(ctx context.Context, deps Deps, opts webhookOptions, payload []byte) error {
	requestCtx, cancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
	defer cancel()

	var lastResponseBody string
	var lastStatusCode int

	for attempt := 0; attempt <= opts.RetryAttempts; attempt++ {
		req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, opts.URL, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("failed to create webhook request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GPC-Event", opts.Event)
		req.Header.Set("X-GPC-Source", "gpc")

		resp, err := deps.HTTPClient.Do(req)
		if err != nil {
			if attempt < opts.RetryAttempts {
				if sleepErr := deps.Sleep(requestCtx, opts.RetryDelay); sleepErr != nil {
					return sleepErr
				}
				continue
			}
			return fmt.Errorf("failed to deliver webhook: %w", err)
		}

		bodyPreview, readErr := readBodyPreview(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("failed to read webhook response: %w", readErr)
		}

		lastStatusCode = resp.StatusCode
		lastResponseBody = bodyPreview
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return shared.WriteJSON(deps.Stdout, webhookResult{
				URL:          opts.URL,
				Event:        opts.Event,
				StatusCode:   resp.StatusCode,
				Status:       "delivered",
				Attempts:     attempt + 1,
				PayloadBytes: len(payload),
				ResponseBody: bodyPreview,
			})
		}

		if !shouldRetryStatus(resp.StatusCode) || attempt >= opts.RetryAttempts {
			return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, trimResponse(bodyPreview))
		}
		if sleepErr := deps.Sleep(requestCtx, opts.RetryDelay); sleepErr != nil {
			return sleepErr
		}
	}

	return fmt.Errorf("webhook delivery failed after %d attempts (last status %d): %s", opts.RetryAttempts+1, lastStatusCode, trimResponse(lastResponseBody))
}

func readBodyPreview(body io.Reader) (string, error) {
	if body == nil {
		return "", nil
	}
	limited := io.LimitReader(body, maxBodyPreview)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func trimResponse(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "empty response body"
	}
	return value
}

func shouldRetryStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
