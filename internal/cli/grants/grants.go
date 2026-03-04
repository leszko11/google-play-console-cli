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
	var parent, inputPath, developerID, userEmail string
	fs.StringVar(&parent, "parent", "", "User resource name (developers/<developer-id>/users/<email>)")
	fs.StringVar(&developerID, "developer-id", "", "Developer account ID (numeric or developers/<id>)")
	fs.StringVar(&userEmail, "user-email", "", "User email for parent name synthesis (requires --developer-id or stored auth developer ID)")
	fs.StringVar(&inputPath, "input", "", "Path to grant JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a grant under a user",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			resolvedParent, err := resolveGrantParent(deps, parent, developerID, userEmail)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			grant, err := readGrantPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			created, err := client.CreateGrant(requestCtx, resolvedParent, grant)
			if err != nil {
				return fmt.Errorf("failed to create grant: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"parent": resolvedParent,
				"grant":  created,
				"status": "created",
			})
		},
	}
}

func newUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var name, inputPath, updateMask, developerID, userEmail, packageName string
	fs.StringVar(&name, "name", "", "Grant resource name (developers/<developer-id>/users/<email>/grants/<package-name>)")
	fs.StringVar(&developerID, "developer-id", "", "Developer account ID (numeric or developers/<id>)")
	fs.StringVar(&userEmail, "user-email", "", "User email for grant name synthesis")
	fs.StringVar(&packageName, "package-name", "", "Package name for grant name synthesis")
	fs.StringVar(&inputPath, "input", "", "Path to grant JSON payload (use - for stdin)")
	fs.StringVar(&updateMask, "update-mask", "", "Comma-separated list of fields to update")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update a grant",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			resolvedName, err := resolveGrantName(deps, name, developerID, userEmail, packageName)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			grant, err := readGrantPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			updated, err := client.UpdateGrant(requestCtx, resolvedName, grant, updateMask)
			if err != nil {
				return fmt.Errorf("failed to update grant: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"name":   resolvedName,
				"grant":  updated,
				"status": "updated",
			})
		},
	}
}

func newDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var name, developerID, userEmail, packageName string
	var confirm bool
	fs.StringVar(&name, "name", "", "Grant resource name (developers/<developer-id>/users/<email>/grants/<package-name>)")
	fs.StringVar(&developerID, "developer-id", "", "Developer account ID (numeric or developers/<id>)")
	fs.StringVar(&userEmail, "user-email", "", "User email for grant name synthesis")
	fs.StringVar(&packageName, "package-name", "", "Package name for grant name synthesis")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the grant (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete a grant",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			resolvedName, err := resolveGrantName(deps, name, developerID, userEmail, packageName)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			if !confirm {
				return fmt.Errorf("--confirm is required to delete grant %q", resolvedName)
			}
			if err := client.DeleteGrant(requestCtx, resolvedName); err != nil {
				return fmt.Errorf("failed to delete grant: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"name":   resolvedName,
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

func resolveGrantParent(deps Deps, parent, developerID, userEmail string) (string, error) {
	parent = strings.TrimSpace(parent)
	if parent != "" {
		return parent, nil
	}

	userEmail = strings.TrimSpace(userEmail)
	if userEmail == "" {
		return "", fmt.Errorf("--parent is required (or provide --user-email with --developer-id)")
	}
	resolvedDeveloperID, err := resolveDeveloperID(deps, developerID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("developers/%s/users/%s", resolvedDeveloperID, userEmail), nil
}

func resolveGrantName(deps Deps, name, developerID, userEmail, packageName string) (string, error) {
	name = strings.TrimSpace(name)
	if name != "" {
		return name, nil
	}

	userEmail = strings.TrimSpace(userEmail)
	if userEmail == "" {
		return "", fmt.Errorf("--name is required (or provide --user-email and --package-name)")
	}
	resolvedPackageName, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return "", fmt.Errorf("--name is required (or provide --user-email and --package-name): %w", err)
	}
	resolvedDeveloperID, err := resolveDeveloperID(deps, developerID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("developers/%s/users/%s/grants/%s", resolvedDeveloperID, userEmail, resolvedPackageName), nil
}

func resolveDeveloperID(deps Deps, localValue string) (string, error) {
	cfg, err := deps.LoadConfig()
	if err != nil {
		return "", err
	}
	return shared.ResolveDeveloperID(localValue, cfg)
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
