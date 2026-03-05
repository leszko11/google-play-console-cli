package orders

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
	GetOrder(ctx context.Context, packageName, orderID string) (gpc.OrderInfo, error)
	BatchGetOrders(ctx context.Context, packageName string, orderIDs []string) ([]gpc.OrderInfo, error)
	RefundOrder(ctx context.Context, packageName, orderID string, revoke bool) error
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
		Name:      "orders",
		ShortHelp: "Inspect and refund Play orders",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newGetCommand(deps),
			newBatchGetCommand(deps),
			newRefundCommand(deps),
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

func newGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, orderID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&orderID, "order-id", "", "Order ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get one Play order by ID",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			orderID = strings.TrimSpace(orderID)
			if orderID == "" {
				return fmt.Errorf("--order-id is required")
			}

			order, err := client.GetOrder(requestCtx, pkg, orderID)
			if err != nil {
				return fmt.Errorf("failed to get order: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"order":       order,
			})
		},
	}
}

func newBatchGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("batch-get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, orderIDsRaw string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&orderIDsRaw, "order-ids", "", "Comma-separated order IDs")

	return &ffcli.Command{
		Name:      "batch-get",
		ShortHelp: "Get multiple Play orders by ID",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			orderIDs := splitCSV(orderIDsRaw)
			if len(orderIDs) == 0 {
				return fmt.Errorf("--order-ids is required")
			}

			orders, err := client.BatchGetOrders(requestCtx, pkg, orderIDs)
			if err != nil {
				return fmt.Errorf("failed to batch get orders: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"orders":      orders,
				"count":       len(orders),
			})
		},
	}
}

func newRefundCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("refund", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, orderID string
	var revoke, confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&orderID, "order-id", "", "Order ID")
	fs.BoolVar(&revoke, "revoke", false, "Also revoke the purchase entitlement")
	fs.BoolVar(&confirm, "confirm", false, "Confirm refunding the order (required)")

	return &ffcli.Command{
		Name:      "refund",
		ShortHelp: "Refund a Play order",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			orderID = strings.TrimSpace(orderID)
			if orderID == "" {
				return fmt.Errorf("--order-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to refund order %q", orderID)
			}

			if err := client.RefundOrder(requestCtx, pkg, orderID, revoke); err != nil {
				return fmt.Errorf("failed to refund order: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"orderId":     orderID,
				"revoke":      revoke,
				"status":      "refunded",
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

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
