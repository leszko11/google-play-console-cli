package grants

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
	"google.golang.org/api/androidpublisher/v3"
)

type Client interface {
	CreateGrant(ctx context.Context, parent string, grant *androidpublisher.Grant) (gpc.GrantInfo, error)
	UpdateGrant(ctx context.Context, name string, grant *androidpublisher.Grant, updateMask string) (gpc.GrantInfo, error)
	DeleteGrant(ctx context.Context, name string) error
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "grants",
		ShortHelp: "Manage per-app user grants",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newCreateCommand(deps),
			newUpdateCommand(deps),
			newDeleteCommand(deps),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewClient == nil {
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (Client, error) {
			return gpc.NewClient(ctx, creds)
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
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
	var parent, inputPath string
	fs.StringVar(&parent, "parent", "", "User resource name (developers/<developer-id>/users/<email>)")
	fs.StringVar(&inputPath, "input", "", "Path to grant JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a grant under a user",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			parent = strings.TrimSpace(parent)
			if parent == "" {
				return fmt.Errorf("--parent is required")
			}
			grant, err := readGrantPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			created, err := client.CreateGrant(requestCtx, parent, grant)
			if err != nil {
				return fmt.Errorf("failed to create grant: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"parent": parent,
				"grant":  created,
				"status": "created",
			})
		},
	}
}

func newUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var name, inputPath, updateMask string
	fs.StringVar(&name, "name", "", "Grant resource name (developers/<developer-id>/users/<email>/grants/<package-name>)")
	fs.StringVar(&inputPath, "input", "", "Path to grant JSON payload (use - for stdin)")
	fs.StringVar(&updateMask, "update-mask", "", "Comma-separated list of fields to update")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update a grant",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			grant, err := readGrantPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			updated, err := client.UpdateGrant(requestCtx, name, grant, updateMask)
			if err != nil {
				return fmt.Errorf("failed to update grant: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"name":   name,
				"grant":  updated,
				"status": "updated",
			})
		},
	}
}

func newDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var name string
	var confirm bool
	fs.StringVar(&name, "name", "", "Grant resource name (developers/<developer-id>/users/<email>/grants/<package-name>)")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the grant (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete a grant",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to delete grant %q", name)
			}
			if err := client.DeleteGrant(requestCtx, name); err != nil {
				return fmt.Errorf("failed to delete grant: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"name":   name,
				"status": "deleted",
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps) (Client, context.Context, context.CancelFunc, error) {
	client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
		Upload:     false,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return client, requestCtx, cancel, nil
}

func readGrantPayload(inputPath string, stdin io.Reader) (*androidpublisher.Grant, error) {
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

	var grant androidpublisher.Grant
	if err := json.Unmarshal(raw, &grant); err != nil {
		return nil, fmt.Errorf("invalid grant JSON payload: %w", err)
	}
	return &grant, nil
}
