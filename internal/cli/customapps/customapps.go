package customapps

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	playcustomapp "google.golang.org/api/playcustomapp/v1"
)

type Client interface {
	CreateCustomApp(ctx context.Context, developerID string, app *playcustomapp.CustomApp) (gpc.CustomAppInfo, error)
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
		Name:      "custom-apps",
		ShortHelp: "Create custom private apps for managed Play organizations",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newCreateCommand(deps),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewClient == nil {
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (Client, error) {
			return gpc.NewCustomAppsClient(ctx, creds)
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

func newCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var developerID string
	var inputPath string
	fs.StringVar(&developerID, "developer-id", "", "Developer account ID (numeric or developers/<id>)")
	fs.StringVar(&inputPath, "input", "", "Path to custom app JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a custom app",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			resolvedDeveloperID, err := resolveDeveloperID(deps, developerID)
			if err != nil {
				return err
			}
			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			app, err := readCustomAppPayload(inputPath, deps.Stdin)
			if err != nil {
				return err
			}
			created, err := client.CreateCustomApp(requestCtx, resolvedDeveloperID, app)
			if err != nil {
				return fmt.Errorf("failed to create custom app: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"developerId": resolvedDeveloperID,
				"customApp":   created,
				"status":      "created",
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps) (Client, context.Context, context.CancelFunc, error) {
	client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return client, requestCtx, cancel, nil
}

func resolveDeveloperID(deps Deps, localValue string) (string, error) {
	cfg, err := deps.LoadConfig()
	if err != nil {
		return "", err
	}
	return shared.ResolveDeveloperID(localValue, cfg)
}

func readCustomAppPayload(inputPath string, stdin io.Reader) (*playcustomapp.CustomApp, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return nil, fmt.Errorf("--input is required")
	}

	var raw []byte
	var err error
	if inputPath == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		raw, err = io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read --input from stdin: %w", err)
		}
	} else {
		raw, err = os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read --input: %w", err)
		}
	}

	var app playcustomapp.CustomApp
	if err := json.Unmarshal(raw, &app); err != nil {
		return nil, fmt.Errorf("invalid custom app JSON payload: %w", err)
	}
	return &app, nil
}
