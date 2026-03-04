package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
)

func handleError(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return ExitCodeOK
	}

	fmt.Fprintln(os.Stderr, err)
	if shared.ActiveGlobalFlags().BootstrapAssist {
		maybeOfferBootstrapBuild(err)
	}
	if shared.IsLikelyUsageError(err) {
		return ExitCodeUsage
	}
	return ExitCodeError
}
