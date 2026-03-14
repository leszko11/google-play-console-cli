package cmd

import (
	"context"
	"errors"
	"flag"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/leszko11/google-play-console-cli/internal/validate"
)

func TestHandleErrorExitCodes(t *testing.T) {
	if got := handleError(flag.ErrHelp); got != ExitCodeOK {
		t.Fatalf("expected help to return %d, got %d", ExitCodeOK, got)
	}
	if got := handleError(shared.UsageErrorf("--flag is required")); got != ExitCodeUsage {
		t.Fatalf("expected usage error to return %d, got %d", ExitCodeUsage, got)
	}
	if got := handleError(gpc.ErrInvalidCredentials); got != ExitCodeAuth {
		t.Fatalf("expected auth error to return %d, got %d", ExitCodeAuth, got)
	}
	if got := handleError(gpc.ErrAccessDenied); got != ExitCodePermission {
		t.Fatalf("expected permission error to return %d, got %d", ExitCodePermission, got)
	}
	if got := handleError(validate.ReleaseNotes(strings.Repeat("a", 501))); got != ExitCodeValidation {
		t.Fatalf("expected validation error to return %d, got %d", ExitCodeValidation, got)
	}
	if got := handleError(shared.WrapUsageError(validate.ReleaseNotes(strings.Repeat("a", 501)))); got != ExitCodeValidation {
		t.Fatalf("expected wrapped validation error to return %d, got %d", ExitCodeValidation, got)
	}
	if got := handleError(gpc.ErrRateLimited); got != ExitCodeRateLimited {
		t.Fatalf("expected rate-limited error to return %d, got %d", ExitCodeRateLimited, got)
	}
	if got := handleError(&url.Error{Op: "Get", URL: "https://example.com", Err: &net.DNSError{Err: "timeout", IsTimeout: true}}); got != ExitCodeNetwork {
		t.Fatalf("expected network error to return %d, got %d", ExitCodeNetwork, got)
	}
	if got := handleError(context.DeadlineExceeded); got != ExitCodeNetwork {
		t.Fatalf("expected timeout error to return %d, got %d", ExitCodeNetwork, got)
	}
	if got := handleError(gpc.ErrPackageNotFound); got != ExitCodeNotFound {
		t.Fatalf("expected not-found error to return %d, got %d", ExitCodeNotFound, got)
	}
	if got := handleError(errors.New("runtime failure")); got != ExitCodeError {
		t.Fatalf("expected runtime error to return %d, got %d", ExitCodeError, got)
	}
}
