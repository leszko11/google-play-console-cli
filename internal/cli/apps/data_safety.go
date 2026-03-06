package apps

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewDataSafetyCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("data-safety", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&inputPath, "input", "", "Path to Data Safety CSV payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "data-safety",
		ShortHelp: "Write the Data Safety CSV declaration for an app",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			pkg, err := shared.ResolvePackageName(packageName)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
			})
			if err != nil {
				return err
			}
			defer cancel()

			safetyLabelsCSV, err := readDataSafetyInput(inputPath, deps.Stdin)
			if err != nil {
				return err
			}

			if err := client.SetDataSafety(requestCtx, pkg, safetyLabelsCSV); err != nil {
				return fmt.Errorf("failed to update data safety: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"status":      "updated",
			})
		},
	}
}

func readDataSafetyInput(inputPath string, stdin io.Reader) (string, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return "", fmt.Errorf("--input is required")
	}

	var raw []byte
	var err error
	if inputPath == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		raw, err = io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read --input from stdin: %w", err)
		}
	} else {
		raw, err = os.ReadFile(inputPath)
		if err != nil {
			return "", fmt.Errorf("failed to read --input: %w", err)
		}
	}

	if len(strings.TrimSpace(string(raw))) == 0 {
		return "", fmt.Errorf("data safety CSV must not be empty")
	}
	return string(raw), nil
}
