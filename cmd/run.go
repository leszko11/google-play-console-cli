package cmd

import "context"

func Run(args []string) int {
	root := newRootCommand()
	if err := root.ParseAndRun(context.Background(), args); err != nil {
		return handleError(err)
	}

	return ExitCodeOK
}
