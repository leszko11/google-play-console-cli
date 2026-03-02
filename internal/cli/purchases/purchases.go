package purchases

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
	GetProductPurchase(ctx context.Context, packageName, productID, token string) (gpc.ProductPurchaseInfo, error)
	AcknowledgeProductPurchase(ctx context.Context, packageName, productID, token, developerPayload string) error
	ConsumeProductPurchase(ctx context.Context, packageName, productID, token string) error

	GetSubscriptionPurchase(ctx context.Context, packageName, token string) (gpc.SubscriptionPurchaseInfo, error)
	CancelSubscriptionPurchase(ctx context.Context, packageName, token, cancellationType string) error
	RevokeSubscriptionPurchase(ctx context.Context, packageName, token, refundType string) error

	ListVoidedPurchases(ctx context.Context, packageName string, query gpc.VoidedPurchasesQuery) (gpc.VoidedPurchasesListInfo, error)
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
		Name:      "purchases",
		ShortHelp: "Manage one-time and subscription purchases",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newProductsCommand(deps),
			newSubscriptionsCommand(deps),
			newVoidedCommand(deps),
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

func newProductsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "products",
		ShortHelp: "Inspect and mutate one-time product purchases",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newProductsGetCommand(deps),
			newProductsAcknowledgeCommand(deps),
			newProductsConsumeCommand(deps),
		},
	}
}

func newProductsGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, token string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&token, "token", "", "Purchase token")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get one-time product purchase details",
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
			token = strings.TrimSpace(token)
			if token == "" {
				return fmt.Errorf("--token is required")
			}

			purchase, err := client.GetProductPurchase(requestCtx, pkg, productID, token)
			if err != nil {
				return fmt.Errorf("failed to get product purchase: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"purchase":    purchase,
			})
		},
	}
}

func newProductsAcknowledgeCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("acknowledge", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, token, developerPayload string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.StringVar(&developerPayload, "developer-payload", "", "Optional developer payload")

	return &ffcli.Command{
		Name:      "acknowledge",
		ShortHelp: "Acknowledge a one-time product purchase",
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
			token = strings.TrimSpace(token)
			if token == "" {
				return fmt.Errorf("--token is required")
			}

			if err := client.AcknowledgeProductPurchase(requestCtx, pkg, productID, token, developerPayload); err != nil {
				return fmt.Errorf("failed to acknowledge product purchase: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"token":       token,
				"status":      "acknowledged",
			})
		},
	}
}

func newProductsConsumeCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("consume", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, token string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.BoolVar(&confirm, "confirm", false, "Confirm consuming the purchase (required)")

	return &ffcli.Command{
		Name:      "consume",
		ShortHelp: "Consume a one-time product purchase",
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
			token = strings.TrimSpace(token)
			if token == "" {
				return fmt.Errorf("--token is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to consume purchase token %q", token)
			}

			if err := client.ConsumeProductPurchase(requestCtx, pkg, productID, token); err != nil {
				return fmt.Errorf("failed to consume product purchase: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"token":       token,
				"status":      "consumed",
			})
		},
	}
}

func newSubscriptionsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "subscriptions",
		ShortHelp: "Inspect and mutate subscription purchases",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newSubscriptionsGetCommand(deps),
			newSubscriptionsCancelCommand(deps),
			newSubscriptionsRevokeCommand(deps),
		},
	}
}

func newSubscriptionsGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, token string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&token, "token", "", "Purchase token")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get subscription purchase details",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			token = strings.TrimSpace(token)
			if token == "" {
				return fmt.Errorf("--token is required")
			}

			purchase, err := client.GetSubscriptionPurchase(requestCtx, pkg, token)
			if err != nil {
				return fmt.Errorf("failed to get subscription purchase: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"purchase":    purchase,
			})
		},
	}
}

func newSubscriptionsCancelCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, token, cancellationType string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.StringVar(&cancellationType, "cancellation-type", gpc.CancellationTypeUserRequestedStopRenewals, "Cancellation type: USER_REQUESTED_STOP_RENEWALS or DEVELOPER_REQUESTED_STOP_PAYMENTS")
	fs.BoolVar(&confirm, "confirm", false, "Confirm canceling the subscription purchase (required)")

	return &ffcli.Command{
		Name:      "cancel",
		ShortHelp: "Cancel a subscription purchase",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			token = strings.TrimSpace(token)
			if token == "" {
				return fmt.Errorf("--token is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to cancel subscription purchase %q", token)
			}

			if err := client.CancelSubscriptionPurchase(requestCtx, pkg, token, cancellationType); err != nil {
				return fmt.Errorf("failed to cancel subscription purchase: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"token":            token,
				"cancellationType": cancellationType,
				"status":           "canceled",
			})
		},
	}
}

func newSubscriptionsRevokeCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("revoke", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, token, refundType string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.StringVar(&refundType, "refund-type", gpc.RevocationRefundTypeFull, "Refund type: full or prorated")
	fs.BoolVar(&confirm, "confirm", false, "Confirm revoking the subscription purchase (required)")

	return &ffcli.Command{
		Name:      "revoke",
		ShortHelp: "Revoke a subscription purchase",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			token = strings.TrimSpace(token)
			if token == "" {
				return fmt.Errorf("--token is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to revoke subscription purchase %q", token)
			}

			if err := client.RevokeSubscriptionPurchase(requestCtx, pkg, token, refundType); err != nil {
				return fmt.Errorf("failed to revoke subscription purchase: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"token":       token,
				"refundType":  refundType,
				"status":      "revoked",
			})
		},
	}
}

func newVoidedCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "voided",
		ShortHelp: "Inspect voided purchases",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newVoidedListCommand(deps),
		},
	}
}

func newVoidedListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, token string
	var maxResults, startIndex, startTime, endTime, listType int64
	var includeQuantityBasedPartialRefund bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&maxResults, "max-results", 0, "Maximum number of results")
	fs.Int64Var(&startIndex, "start-index", 0, "Index of first result (indexed pagination)")
	fs.StringVar(&token, "token", "", "Token from previous paginated response")
	fs.Int64Var(&startTime, "start-time", 0, "Start time in milliseconds since epoch")
	fs.Int64Var(&endTime, "end-time", 0, "End time in milliseconds since epoch")
	fs.Int64Var(&listType, "type", 0, "0 for one-time products, 1 for products and subscriptions")
	fs.BoolVar(&includeQuantityBasedPartialRefund, "include-quantity-based-partial-refund", false, "Include quantity-based partial refunds")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List voided purchases",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			result, err := client.ListVoidedPurchases(requestCtx, pkg, gpc.VoidedPurchasesQuery{
				MaxResults:                        maxResults,
				StartIndex:                        startIndex,
				Token:                             token,
				StartTime:                         startTime,
				EndTime:                           endTime,
				Type:                              listType,
				IncludeQuantityBasedPartialRefund: includeQuantityBasedPartialRefund,
				Paginate:                          shared.ActiveGlobalFlags().Paginate,
			})
			if err != nil {
				return fmt.Errorf("failed to list voided purchases: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":     pkg,
				"voidedPurchases": result.VoidedPurchases,
				"nextToken":       result.NextToken,
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
