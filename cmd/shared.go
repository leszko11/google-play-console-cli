package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func handleError(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return ExitCodeOK
	}

	fmt.Fprintln(os.Stderr, err)
	maybeOfferBootstrapBuild(err)
	return ExitCodeError
}
