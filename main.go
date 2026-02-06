package main

import (
	"os"

	"github.com/leszko11/google-play-console-cli/cmd"
)

func main() {
	os.Exit(cmd.Run(os.Args[1:]))
}
