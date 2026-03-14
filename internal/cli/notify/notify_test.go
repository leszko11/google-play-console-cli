package notify

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func runNotify(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	if deps.Stdin == nil {
		deps.Stdin = bytes.NewBuffer(nil)
	}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestWebhookSuccessPostsRawJSON(t *testing.T) {
	var capturedBody string
	var capturedEvent string
	var capturedContentType string

	deps := Deps{
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			raw, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			capturedBody = string(raw)
			capturedEvent = req.Header.Get("X-GPC-Event")
			capturedContentType = req.Header.Get("Content-Type")
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			}, nil
		}),
		Stdin: bytes.NewBufferString(`{"release":"alpha"}`),
	}

	out, err := runNotify(t, deps, "webhook", "--url", "https://example.com/webhook", "--event", "release.completed", "--input", "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBody != `{"release":"alpha"}` {
		t.Fatalf("expected raw payload, got %q", capturedBody)
	}
	if capturedEvent != "release.completed" {
		t.Fatalf("expected event header, got %q", capturedEvent)
	}
	if capturedContentType != "application/json" {
		t.Fatalf("expected application/json, got %q", capturedContentType)
	}
	if !strings.Contains(out, `"status":"delivered"`) || !strings.Contains(out, `"attempts":1`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestWebhookRetriesOnServerError(t *testing.T) {
	attempts := 0
	deps := Deps{
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader("temporary")),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}),
		Sleep: func(context.Context, time.Duration) error { return nil },
		Stdin: bytes.NewBufferString(`{"hello":"world"}`),
	}

	out, err := runNotify(t, deps, "webhook", "--url", "https://example.com/webhook", "--event", "reviews.summary", "--input", "-", "--retry-attempts", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if !strings.Contains(out, `"attempts":2`) {
		t.Fatalf("expected second-attempt success, got %s", out)
	}
}

func TestWebhookDoesNotRetryClientError(t *testing.T) {
	attempts := 0
	deps := Deps{
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("bad payload")),
			}, nil
		}),
		Sleep: func(context.Context, time.Duration) error { return nil },
		Stdin: bytes.NewBufferString(`{"hello":"world"}`),
	}

	_, err := runNotify(t, deps, "webhook", "--url", "https://example.com/webhook", "--event", "reviews.summary", "--input", "-", "--retry-attempts", "3")
	if err == nil || !strings.Contains(err.Error(), "webhook returned status 400") {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected no retries on 4xx, got %d attempts", attempts)
	}
}

func TestWebhookRetriesNetworkError(t *testing.T) {
	attempts := 0
	deps := Deps{
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return nil, errors.New("dial tcp timeout")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}),
		Sleep: func(context.Context, time.Duration) error { return nil },
		Stdin: bytes.NewBufferString(`{"hello":"world"}`),
	}

	out, err := runNotify(t, deps, "webhook", "--url", "https://example.com/webhook", "--event", "release.failed", "--input", "-", "--retry-attempts", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected retry on network error, got %d attempts", attempts)
	}
	if !strings.Contains(out, `"attempts":2`) {
		t.Fatalf("expected success after retry, got %s", out)
	}
}

func TestWebhookRequiresValidJSON(t *testing.T) {
	deps := Deps{
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			t.Fatal("request should not be sent")
			return nil, nil
		}),
		Stdin: bytes.NewBufferString(`not-json`),
	}

	_, err := runNotify(t, deps, "webhook", "--url", "https://example.com/webhook", "--event", "release.failed", "--input", "-")
	if err == nil || !strings.Contains(err.Error(), "--input must contain valid JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhookRequiresFlags(t *testing.T) {
	deps := Deps{
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			t.Fatal("request should not be sent")
			return nil, nil
		}),
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "missing url", args: []string{"webhook", "--event", "release.completed", "--input", "-"}, want: "--url is required"},
		{name: "missing event", args: []string{"webhook", "--url", "https://example.com", "--input", "-"}, want: "--event is required"},
		{name: "missing input", args: []string{"webhook", "--url", "https://example.com", "--event", "release.completed"}, want: "--input is required"},
		{name: "negative retries", args: []string{"webhook", "--url", "https://example.com", "--event", "release.completed", "--input", "-", "--retry-attempts", "-1"}, want: "--retry-attempts must be zero or greater"},
		{name: "bad retry delay", args: []string{"webhook", "--url", "https://example.com", "--event", "release.completed", "--input", "-", "--retry-delay", "0s"}, want: "--retry-delay must be greater than zero"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runNotify(t, deps, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}
