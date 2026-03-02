package subscriptions

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
	ListSubscriptions(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (gpc.SubscriptionsListInfo, error)
	GetSubscription(ctx context.Context, packageName, productID string) (gpc.SubscriptionInfo, error)
	CreateSubscription(ctx context.Context, packageName string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error)
	UpdateSubscription(ctx context.Context, packageName, productID string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error)
	DeleteSubscription(ctx context.Context, packageName, productID string) error
	ArchiveSubscription(ctx context.Context, packageName, productID string) error
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
		Name:      "subscriptions",
		ShortHelp: "Manage monetization subscriptions",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newGetCommand(deps),
			newCreateCommand(deps),
			newUpdateCommand(deps),
			newDeleteCommand(deps),
			newArchiveCommand(deps),
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
	var packageName, pageToken string
	var pageSize int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&pageSize, "page-size", 0, "Maximum subscriptions per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List subscriptions",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			if pageSize < 0 {
				return fmt.Errorf("--page-size must be greater than or equal to zero")
			}
			result, err := client.ListSubscriptions(requestCtx, pkg, pageSize, pageToken, shared.ActiveGlobalFlags().Paginate)
			if err != nil {
				return fmt.Errorf("failed to list subscriptions: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"subscriptions": result.Subscriptions,
				"nextPageToken": result.NextPageToken,
			})
		},
	}
}

func newGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get a subscription by product ID",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			productID = strings.TrimSpace(productID)
			if productID == "" {
				return fmt.Errorf("--product-id is required")
			}
			subscription, err := client.GetSubscription(requestCtx, pkg, productID)
			if err != nil {
				return fmt.Errorf("failed to get subscription: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":  pkg,
				"subscription": subscription,
			})
		},
	}
}

func newCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&inputPath, "input", "", "Path to subscription JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a subscription",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			subscription, err := readSubscriptionPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			created, err := client.CreateSubscription(requestCtx, pkg, subscription)
			if err != nil {
				return fmt.Errorf("failed to create subscription: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":  pkg,
				"subscription": created,
				"status":       "created",
			})
		},
	}
}

func newUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&inputPath, "input", "", "Path to subscription JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update a subscription",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			productID = strings.TrimSpace(productID)
			if productID == "" {
				return fmt.Errorf("--product-id is required")
			}
			subscription, err := readSubscriptionPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			updated, err := client.UpdateSubscription(requestCtx, pkg, productID, subscription)
			if err != nil {
				return fmt.Errorf("failed to update subscription: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":  pkg,
				"subscription": updated,
				"status":       "updated",
			})
		},
	}
}

func newDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the subscription (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete a subscription",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			productID = strings.TrimSpace(productID)
			if productID == "" {
				return fmt.Errorf("--product-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to delete subscription %q", productID)
			}
			if err := client.DeleteSubscription(requestCtx, pkg, productID); err != nil {
				return fmt.Errorf("failed to delete subscription: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"status":      "deleted",
			})
		},
	}
}

func newArchiveCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("archive", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm archiving the subscription (required)")

	return &ffcli.Command{
		Name:      "archive",
		ShortHelp: "Archive a subscription",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			productID = strings.TrimSpace(productID)
			if productID == "" {
				return fmt.Errorf("--product-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to archive subscription %q", productID)
			}
			if err := client.ArchiveSubscription(requestCtx, pkg, productID); err != nil {
				return fmt.Errorf("failed to archive subscription: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"status":      "archived",
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
		Upload:     false,
	})
	if err != nil {
		return nil, "", nil, nil, err
	}
	return client, pkg, requestCtx, cancel, nil
}

func readSubscriptionPayload(inputPath string, stdin io.Reader) (*androidpublisher.Subscription, error) {
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

	var subscription androidpublisher.Subscription
	if err := json.Unmarshal(raw, &subscription); err != nil {
		return nil, fmt.Errorf("invalid subscription JSON payload: %w", err)
	}
	return &subscription, nil
}
