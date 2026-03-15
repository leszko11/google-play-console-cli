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
	notificationOptions
	InputPath string
}

type notificationOptions struct {
	URL           string
	Event         string
	Title         string
	Message       string
	RetryAttempts int
	RetryDelay    time.Duration
}

type webhookResult struct {
	Provider     string `json:"provider,omitempty"`
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
			newSlackCommand(deps),
			newDiscordCommand(deps),
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
			return sendNotification(ctx, deps, deliveryOptions{
				Provider:      "webhook",
				URL:           opts.URL,
				Event:         opts.Event,
				RetryAttempts: opts.RetryAttempts,
				RetryDelay:    opts.RetryDelay,
			}, payload)
		},
	}
}

func validateWebhookOptions(opts webhookOptions) (webhookOptions, error) {
	normalized, err := validateNotificationOptions(opts.notificationOptions)
	if err != nil {
		return webhookOptions{}, err
	}
	opts.notificationOptions = normalized
	opts.InputPath = strings.TrimSpace(opts.InputPath)
	if opts.InputPath == "" {
		return webhookOptions{}, shared.UsageErrorf("--input is required")
	}
	return opts, nil
}

func validateNotificationOptions(opts notificationOptions) (notificationOptions, error) {
	opts.URL = strings.TrimSpace(opts.URL)
	if opts.URL == "" {
		return notificationOptions{}, shared.UsageErrorf("--url is required")
	}
	opts.Event = strings.TrimSpace(opts.Event)
	if opts.Event == "" {
		return notificationOptions{}, shared.UsageErrorf("--event is required")
	}
	if opts.RetryAttempts < 0 {
		return notificationOptions{}, shared.UsageErrorf("--retry-attempts must be zero or greater")
	}
	if opts.RetryDelay <= 0 {
		return notificationOptions{}, shared.UsageErrorf("--retry-delay must be greater than zero")
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

type deliveryOptions struct {
	Provider      string
	URL           string
	Event         string
	RetryAttempts int
	RetryDelay    time.Duration
}

func sendNotification(ctx context.Context, deps Deps, opts deliveryOptions, payload []byte) error {
	requestCtx, cancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
	defer cancel()

	var lastResponseBody string
	var lastStatusCode int

	for attempt := 0; attempt <= opts.RetryAttempts; attempt++ {
		req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, opts.URL, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("failed to create %s request: %w", requestLabel(opts.Provider), err)
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
			return fmt.Errorf("failed to deliver %s: %w", deliveryLabel(opts.Provider), err)
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
				Provider:     opts.Provider,
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
			return fmt.Errorf("%s returned status %d: %s", responseLabel(opts.Provider), resp.StatusCode, trimResponse(bodyPreview))
		}
		if sleepErr := deps.Sleep(requestCtx, opts.RetryDelay); sleepErr != nil {
			return sleepErr
		}
	}

	return fmt.Errorf("%s delivery failed after %d attempts (last status %d): %s", opts.Provider, opts.RetryAttempts+1, lastStatusCode, trimResponse(lastResponseBody))
}

func requestLabel(provider string) string {
	if provider == "webhook" {
		return "webhook request"
	}
	return provider + " request"
}

func responseLabel(provider string) string {
	if provider == "webhook" {
		return "webhook"
	}
	return provider + " notification"
}

func deliveryLabel(provider string) string {
	if provider == "webhook" {
		return "webhook"
	}
	return provider + " notification"
}

func newSlackCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("slack", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts notificationOptions
	var inputPath string
	fs.StringVar(&opts.URL, "url", "", "Slack webhook URL")
	fs.StringVar(&opts.Event, "event", "", "Event name metadata")
	fs.StringVar(&opts.Title, "title", "", "Optional notification title (defaults to --event)")
	fs.StringVar(&opts.Message, "message", "", "Notification message text")
	fs.StringVar(&inputPath, "input", "", "Optional path to JSON context payload (use - for stdin)")
	fs.IntVar(&opts.RetryAttempts, "retry-attempts", 0, "Additional retry attempts for network, 429, or 5xx failures")
	fs.DurationVar(&opts.RetryDelay, "retry-delay", defaultRetryDelay, "Delay between retry attempts")

	return &ffcli.Command{
		Name:      "slack",
		ShortHelp: "POST a native Slack webhook message",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateChatOptions(opts)
			if err != nil {
				return err
			}
			contextPreview, err := readOptionalJSONPreview(inputPath, deps.Stdin)
			if err != nil {
				return err
			}
			payload, err := buildSlackPayload(opts, contextPreview)
			if err != nil {
				return err
			}
			return sendNotification(ctx, deps, deliveryOptions{
				Provider:      "slack",
				URL:           opts.URL,
				Event:         opts.Event,
				RetryAttempts: opts.RetryAttempts,
				RetryDelay:    opts.RetryDelay,
			}, payload)
		},
	}
}

func newDiscordCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("discord", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts notificationOptions
	var inputPath string
	fs.StringVar(&opts.URL, "url", "", "Discord webhook URL")
	fs.StringVar(&opts.Event, "event", "", "Event name metadata")
	fs.StringVar(&opts.Title, "title", "", "Optional embed title (defaults to --event)")
	fs.StringVar(&opts.Message, "message", "", "Notification message text")
	fs.StringVar(&inputPath, "input", "", "Optional path to JSON context payload (use - for stdin)")
	fs.IntVar(&opts.RetryAttempts, "retry-attempts", 0, "Additional retry attempts for network, 429, or 5xx failures")
	fs.DurationVar(&opts.RetryDelay, "retry-delay", defaultRetryDelay, "Delay between retry attempts")

	return &ffcli.Command{
		Name:      "discord",
		ShortHelp: "POST a native Discord webhook message",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateChatOptions(opts)
			if err != nil {
				return err
			}
			contextPreview, err := readOptionalJSONPreview(inputPath, deps.Stdin)
			if err != nil {
				return err
			}
			payload, err := buildDiscordPayload(opts, contextPreview)
			if err != nil {
				return err
			}
			return sendNotification(ctx, deps, deliveryOptions{
				Provider:      "discord",
				URL:           opts.URL,
				Event:         opts.Event,
				RetryAttempts: opts.RetryAttempts,
				RetryDelay:    opts.RetryDelay,
			}, payload)
		},
	}
}

func validateChatOptions(opts notificationOptions) (notificationOptions, error) {
	normalized, err := validateNotificationOptions(opts)
	if err != nil {
		return notificationOptions{}, err
	}
	normalized.Message = strings.TrimSpace(normalized.Message)
	if normalized.Message == "" {
		return notificationOptions{}, shared.UsageErrorf("--message is required")
	}
	normalized.Title = strings.TrimSpace(normalized.Title)
	if normalized.Title == "" {
		normalized.Title = normalized.Event
	}
	return normalized, nil
}

func readOptionalJSONPreview(path string, stdin io.Reader) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	raw, err := readWebhookPayload(path, stdin)
	if err != nil {
		return "", err
	}
	return prettyJSONPreview(raw, 1800)
}

func prettyJSONPreview(raw []byte, maxLen int) (string, error) {
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", shared.UsageErrorf("--input must contain valid JSON")
	}
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON preview: %w", err)
	}
	preview := string(formatted)
	if len(preview) <= maxLen {
		return preview, nil
	}
	suffix := "\n... (truncated)"
	if maxLen <= len(suffix) {
		return suffix[:maxLen], nil
	}
	return preview[:maxLen-len(suffix)] + suffix, nil
}

func buildSlackPayload(opts notificationOptions, contextPreview string) ([]byte, error) {
	blocks := []map[string]any{
		{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": "*" + opts.Title + "*",
			},
		},
		{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": opts.Message,
			},
		},
		{
			"type": "context",
			"elements": []map[string]any{
				{
					"type": "mrkdwn",
					"text": "Event: `" + opts.Event + "`",
				},
			},
		},
	}
	if contextPreview != "" {
		blocks = append(blocks, map[string]any{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": "*Context*\n```" + contextPreview + "```",
			},
		})
	}
	return json.Marshal(map[string]any{
		"text":   opts.Message,
		"blocks": blocks,
	})
}

func buildDiscordPayload(opts notificationOptions, contextPreview string) ([]byte, error) {
	fields := []map[string]any{
		{
			"name":   "Event",
			"value":  opts.Event,
			"inline": false,
		},
	}
	if contextPreview != "" {
		fields = append(fields, map[string]any{
			"name":   "Context",
			"value":  "```json\n" + contextPreview + "\n```",
			"inline": false,
		})
	}
	return json.Marshal(map[string]any{
		"content": opts.Message,
		"embeds": []map[string]any{
			{
				"title":       opts.Title,
				"description": opts.Message,
				"fields":      fields,
			},
		},
	})
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
