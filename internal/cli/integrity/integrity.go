package integrity

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	DecodeIntegrityToken(ctx context.Context, packageName, integrityToken string) (gpc.IntegrityDecodeInfo, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "integrity",
		ShortHelp: "Decode and inspect Play Integrity tokens",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newDecodeCommand(deps),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewClient == nil {
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (Client, error) {
			return gpc.NewIntegrityClient(ctx, creds)
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
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

func newDecodeCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("decode", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName string
	var token string
	var inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&token, "token", "", "Integrity token")
	fs.StringVar(&inputPath, "input", "", "Path to a file containing the integrity token (use - for stdin)")

	return &ffcli.Command{
		Name:      "decode",
		ShortHelp: "Decode a Play Integrity token",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			resolvedToken, err := resolveIntegrityToken(token, inputPath, deps.Stdin)
			if err != nil {
				return err
			}

			result, err := client.DecodeIntegrityToken(requestCtx, pkg, resolvedToken)
			if err != nil {
				return fmt.Errorf("failed to decode integrity token: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":          pkg,
				"tokenPayloadExternal": result.TokenPayloadExternal,
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName string) (Client, string, context.Context, context.CancelFunc, error) {
	pkg, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return nil, "", nil, nil, err
	}
	client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
	})
	if err != nil {
		return nil, "", nil, nil, err
	}
	return client, pkg, requestCtx, cancel, nil
}

func resolveIntegrityToken(token, inputPath string, stdin io.Reader) (string, error) {
	token = strings.TrimSpace(token)
	inputPath = strings.TrimSpace(inputPath)

	switch {
	case token != "" && inputPath != "":
		return "", shared.UsageErrorf("exactly one of --token or --input must be set")
	case token == "" && inputPath == "":
		return "", shared.UsageErrorf("exactly one of --token or --input must be set")
	case token != "":
		return token, nil
	}

	raw, err := readInput(inputPath, stdin)
	if err != nil {
		return "", err
	}
	resolved := strings.TrimSpace(string(raw))
	if resolved == "" {
		return "", shared.UsageErrorf("integrity token must not be empty")
	}
	return resolved, nil
}

func readInput(path string, stdin io.Reader) ([]byte, error) {
	switch path {
	case "-":
		if stdin == nil {
			stdin = os.Stdin
		}
		raw, err := io.ReadAll(stdin)
		if err != nil {
			return nil, shared.UsageErrorf("failed to read --input from stdin: %v", err)
		}
		return raw, nil
	default:
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, shared.UsageErrorf("failed to read --input: %v", err)
		}
		return raw, nil
	}
}
