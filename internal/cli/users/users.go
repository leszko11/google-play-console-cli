package users

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
	ListUsers(ctx context.Context, developerID string, pageSize int64, pageToken string, paginate bool) (gpc.UsersListInfo, error)
	CreateUser(ctx context.Context, developerID string, user *androidpublisher.User) (gpc.UserInfo, error)
	UpdateUser(ctx context.Context, name string, user *androidpublisher.User, updateMask string) (gpc.UserInfo, error)
	DeleteUser(ctx context.Context, name string) error
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
		Name:      "users",
		ShortHelp: "Manage Play Console account users",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
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

func newListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var developerID, pageToken string
	var pageSize int64
	fs.StringVar(&developerID, "developer-id", "", "Developer account ID (numeric or developers/<id>)")
	fs.Int64Var(&pageSize, "page-size", 0, "Maximum users per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List account users",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			developerID = strings.TrimSpace(developerID)
			if developerID == "" {
				return fmt.Errorf("--developer-id is required")
			}
			if pageSize < 0 {
				return fmt.Errorf("--page-size must be greater than or equal to zero")
			}

			result, err := client.ListUsers(requestCtx, developerID, pageSize, pageToken, shared.ActiveGlobalFlags().Paginate)
			if err != nil {
				return fmt.Errorf("failed to list users: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"developerId":   developerID,
				"users":         result.Users,
				"nextPageToken": result.NextPageToken,
			})
		},
	}
}

func newCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var developerID, inputPath string
	fs.StringVar(&developerID, "developer-id", "", "Developer account ID (numeric or developers/<id>)")
	fs.StringVar(&inputPath, "input", "", "Path to user JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create an account user",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			developerID = strings.TrimSpace(developerID)
			if developerID == "" {
				return fmt.Errorf("--developer-id is required")
			}
			user, err := readUserPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			created, err := client.CreateUser(requestCtx, developerID, user)
			if err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"developerId": developerID,
				"user":        created,
				"status":      "created",
			})
		},
	}
}

func newUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var name, inputPath, updateMask string
	fs.StringVar(&name, "name", "", "User resource name (developers/<developer-id>/users/<email>)")
	fs.StringVar(&inputPath, "input", "", "Path to user JSON payload (use - for stdin)")
	fs.StringVar(&updateMask, "update-mask", "", "Comma-separated list of fields to update")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update an account user",
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
			user, err := readUserPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			updated, err := client.UpdateUser(requestCtx, name, user, updateMask)
			if err != nil {
				return fmt.Errorf("failed to update user: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"name":   name,
				"user":   updated,
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
	fs.StringVar(&name, "name", "", "User resource name (developers/<developer-id>/users/<email>)")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the user (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete an account user",
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
				return fmt.Errorf("--confirm is required to delete user %q", name)
			}
			if err := client.DeleteUser(requestCtx, name); err != nil {
				return fmt.Errorf("failed to delete user: %w", err)
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

func readUserPayload(inputPath string, stdin io.Reader) (*androidpublisher.User, error) {
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

	var user androidpublisher.User
	if err := json.Unmarshal(raw, &user); err != nil {
		return nil, fmt.Errorf("invalid user JSON payload: %w", err)
	}
	return &user, nil
}
