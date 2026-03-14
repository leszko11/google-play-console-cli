package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/leszko11/google-play-console-cli/internal/validate"
)

func handleError(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return ExitCodeOK
	}

	fmt.Fprintln(os.Stderr, err)
	if shared.ActiveGlobalFlags().BootstrapAssist {
		maybeOfferBootstrapBuild(err)
	}
	return classifyExitCode(err)
}

func classifyExitCode(err error) int {
	var validationErr validate.ValidationError
	if errors.As(err, &validationErr) {
		return ExitCodeValidation
	}
	if shared.IsLikelyUsageError(err) {
		return ExitCodeUsage
	}
	switch {
	case errors.Is(err, gpc.ErrInvalidCredentials):
		return ExitCodeAuth
	case errors.Is(err, gpc.ErrAccessDenied):
		return ExitCodePermission
	case errors.Is(err, gpc.ErrRateLimited):
		return ExitCodeRateLimited
	case errors.Is(err, gpc.ErrPackageNotFound):
		return ExitCodeNotFound
	case errors.Is(err, context.DeadlineExceeded):
		return ExitCodeNetwork
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return ExitCodeNetwork
	}
	return ExitCodeError
}
