package cmd

import (
	"errors"
	"flag"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
)

func TestHandleErrorExitCodes(t *testing.T) {
	if got := handleError(flag.ErrHelp); got != ExitCodeOK {
		t.Fatalf("expected help to return %d, got %d", ExitCodeOK, got)
	}
	if got := handleError(shared.UsageErrorf("--flag is required")); got != ExitCodeUsage {
		t.Fatalf("expected usage error to return %d, got %d", ExitCodeUsage, got)
	}
	if got := handleError(errors.New("runtime failure")); got != ExitCodeError {
		t.Fatalf("expected runtime error to return %d, got %d", ExitCodeError, got)
	}
}
