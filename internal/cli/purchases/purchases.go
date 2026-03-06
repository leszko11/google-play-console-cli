package purchases

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	GetProductPurchase(ctx context.Context, packageName, productID, token string) (gpc.ProductPurchaseInfo, error)
	GetProductPurchaseV2(ctx context.Context, packageName, token string) (gpc.ProductPurchaseV2Info, error)
	AcknowledgeProductPurchase(ctx context.Context, packageName, productID, token, developerPayload string) error
	ConsumeProductPurchase(ctx context.Context, packageName, productID, token string) error

	GetSubscriptionPurchase(ctx context.Context, packageName, token string) (gpc.SubscriptionPurchaseInfo, error)
	CancelSubscriptionPurchase(ctx context.Context, packageName, token, cancellationType string) error
	DeferSubscriptionPurchase(ctx context.Context, packageName, token, etag, deferDuration string, validateOnly bool) (gpc.SubscriptionDeferInfo, error)
	RevokeSubscriptionPurchase(ctx context.Context, packageName, token, refundType string) error
	GetLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string) (gpc.SubscriptionPurchaseInfo, error)
	AcknowledgeLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token, developerPayload string) error
	CancelLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string) error
	DeferLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string, expectedExpiryTimeMillis, desiredExpiryTimeMillis int64) (gpc.SubscriptionDeferInfo, error)
	RefundLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string) error
	RevokeLegacySubscriptionPurchase(ctx context.Context, packageName, subscriptionID, token string) error

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
			newProductsV2Command(deps),
			newSubscriptionsCommand(deps),
			newSubscriptionsLegacyCommand(deps),
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

func newProductsV2Command(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "products-v2",
		ShortHelp: "Inspect one-time product purchases via Purchases.Productsv2",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newProductsV2GetCommand(deps),
		},
	}
}

func newProductsV2GetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, token string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&token, "token", "", "Purchase token")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get one-time product purchase details (v2)",
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

			purchase, err := client.GetProductPurchaseV2(requestCtx, pkg, token)
			if err != nil {
				return wrapPurchaseEndpointError("failed to get product purchase (v2)", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"purchase":    purchase,
			})
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
				return wrapPurchaseEndpointError("failed to get product purchase", err)
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
				return wrapPurchaseEndpointError("failed to acknowledge product purchase", err)
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
				return wrapPurchaseEndpointError("failed to consume product purchase", err)
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
			newSubscriptionsDeferCommand(deps),
			newSubscriptionsRevokeCommand(deps),
		},
	}
}

func newSubscriptionsLegacyCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "subscriptions-legacy",
		ShortHelp: "Inspect and mutate legacy subscription purchases",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newSubscriptionsLegacyGetCommand(deps),
			newSubscriptionsLegacyAcknowledgeCommand(deps),
			newSubscriptionsLegacyCancelCommand(deps),
			newSubscriptionsLegacyDeferCommand(deps),
			newSubscriptionsLegacyRefundCommand(deps),
			newSubscriptionsLegacyRevokeCommand(deps),
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
				return wrapPurchaseEndpointError("failed to get subscription purchase", err)
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
				return wrapPurchaseEndpointError("failed to cancel subscription purchase", err)
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
				return wrapPurchaseEndpointError("failed to revoke subscription purchase", err)
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

func newSubscriptionsDeferCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("defer", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, token, etag, deferDuration string
	var validateOnly, confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.StringVar(&etag, "etag", "", "Current subscription etag from purchases subscriptions get")
	fs.StringVar(&deferDuration, "defer-duration", "", "Deferral duration (protobuf format, for example 604800s)")
	fs.BoolVar(&validateOnly, "validate-only", false, "Validate deferral request without applying changes")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deferring the subscription purchase (required unless --validate-only)")

	return &ffcli.Command{
		Name:      "defer",
		ShortHelp: "Defer a subscription renewal",
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
			etag = strings.TrimSpace(etag)
			if etag == "" {
				return fmt.Errorf("--etag is required")
			}
			deferDuration = strings.TrimSpace(deferDuration)
			if deferDuration == "" {
				return fmt.Errorf("--defer-duration is required")
			}
			if !validateOnly && !confirm {
				return fmt.Errorf("--confirm is required to defer subscription purchase %q", token)
			}

			result, err := client.DeferSubscriptionPurchase(requestCtx, pkg, token, etag, deferDuration, validateOnly)
			if err != nil {
				return wrapPurchaseEndpointError("failed to defer subscription purchase", err)
			}

			status := "deferred"
			if validateOnly {
				status = "validated"
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":           pkg,
				"token":                 token,
				"etag":                  etag,
				"deferDuration":         deferDuration,
				"validateOnly":          validateOnly,
				"itemExpiryTimeDetails": result.ItemExpiryTimeDetails,
				"status":                status,
			})
		},
	}
}

func newSubscriptionsLegacyGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, subscriptionID, token string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&subscriptionID, "subscription-id", "", "Legacy subscription product ID")
	fs.StringVar(&token, "token", "", "Purchase token")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get legacy subscription purchase details",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			subscriptionID, token, err = requireSubscriptionLegacyTarget(subscriptionID, token)
			if err != nil {
				return err
			}

			purchase, err := client.GetLegacySubscriptionPurchase(requestCtx, pkg, subscriptionID, token)
			if err != nil {
				return wrapPurchaseEndpointError("failed to get legacy subscription purchase", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"api":            "legacy",
				"packageName":    pkg,
				"subscriptionId": subscriptionID,
				"purchase":       purchase,
			})
		},
	}
}

func newSubscriptionsLegacyAcknowledgeCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("acknowledge", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, subscriptionID, token, developerPayload string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&subscriptionID, "subscription-id", "", "Legacy subscription product ID")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.StringVar(&developerPayload, "developer-payload", "", "Optional developer payload")

	return &ffcli.Command{
		Name:      "acknowledge",
		ShortHelp: "Acknowledge a legacy subscription purchase",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			subscriptionID, token, err = requireSubscriptionLegacyTarget(subscriptionID, token)
			if err != nil {
				return err
			}

			if err := client.AcknowledgeLegacySubscriptionPurchase(requestCtx, pkg, subscriptionID, token, developerPayload); err != nil {
				return wrapPurchaseEndpointError("failed to acknowledge legacy subscription purchase", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"api":            "legacy",
				"packageName":    pkg,
				"subscriptionId": subscriptionID,
				"token":          token,
				"status":         "acknowledged",
			})
		},
	}
}

func newSubscriptionsLegacyCancelCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, subscriptionID, token string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&subscriptionID, "subscription-id", "", "Legacy subscription product ID")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.BoolVar(&confirm, "confirm", false, "Confirm canceling the legacy subscription purchase (required)")

	return &ffcli.Command{
		Name:      "cancel",
		ShortHelp: "Cancel a legacy subscription purchase",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			subscriptionID, token, err = requireSubscriptionLegacyTarget(subscriptionID, token)
			if err != nil {
				return err
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to cancel legacy subscription purchase %q", token)
			}

			if err := client.CancelLegacySubscriptionPurchase(requestCtx, pkg, subscriptionID, token); err != nil {
				return wrapPurchaseEndpointError("failed to cancel legacy subscription purchase", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"api":            "legacy",
				"packageName":    pkg,
				"subscriptionId": subscriptionID,
				"token":          token,
				"status":         "canceled",
			})
		},
	}
}

func newSubscriptionsLegacyDeferCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("defer", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, subscriptionID, token string
	var expectedExpiryTimeMillis, desiredExpiryTimeMillis string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&subscriptionID, "subscription-id", "", "Legacy subscription product ID")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.StringVar(&expectedExpiryTimeMillis, "expected-expiry-time-millis", "", "Current expiry time in milliseconds since epoch")
	fs.StringVar(&desiredExpiryTimeMillis, "desired-expiry-time-millis", "", "Desired next expiry time in milliseconds since epoch")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deferring the legacy subscription purchase (required)")

	return &ffcli.Command{
		Name:      "defer",
		ShortHelp: "Defer a legacy subscription renewal",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			subscriptionID, token, err = requireSubscriptionLegacyTarget(subscriptionID, token)
			if err != nil {
				return err
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to defer legacy subscription purchase %q", token)
			}
			expectedMillis, err := parsePositiveMillisFlag("--expected-expiry-time-millis", expectedExpiryTimeMillis)
			if err != nil {
				return err
			}
			desiredMillis, err := parsePositiveMillisFlag("--desired-expiry-time-millis", desiredExpiryTimeMillis)
			if err != nil {
				return err
			}

			result, err := client.DeferLegacySubscriptionPurchase(requestCtx, pkg, subscriptionID, token, expectedMillis, desiredMillis)
			if err != nil {
				return wrapPurchaseEndpointError("failed to defer legacy subscription purchase", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"api":                      "legacy",
				"packageName":              pkg,
				"subscriptionId":           subscriptionID,
				"token":                    token,
				"expectedExpiryTimeMillis": expectedMillis,
				"desiredExpiryTimeMillis":  desiredMillis,
				"newExpiryTimeMillis":      result.NewExpiryTimeMillis,
				"status":                   "deferred",
			})
		},
	}
}

func newSubscriptionsLegacyRefundCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("refund", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, subscriptionID, token string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&subscriptionID, "subscription-id", "", "Legacy subscription product ID")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.BoolVar(&confirm, "confirm", false, "Confirm refunding the legacy subscription purchase (required)")

	return &ffcli.Command{
		Name:      "refund",
		ShortHelp: "Refund a legacy subscription purchase",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			subscriptionID, token, err = requireSubscriptionLegacyTarget(subscriptionID, token)
			if err != nil {
				return err
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to refund legacy subscription purchase %q", token)
			}

			if err := client.RefundLegacySubscriptionPurchase(requestCtx, pkg, subscriptionID, token); err != nil {
				return wrapPurchaseEndpointError("failed to refund legacy subscription purchase", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"api":            "legacy",
				"packageName":    pkg,
				"subscriptionId": subscriptionID,
				"token":          token,
				"status":         "refunded",
			})
		},
	}
}

func newSubscriptionsLegacyRevokeCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("revoke", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, subscriptionID, token string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&subscriptionID, "subscription-id", "", "Legacy subscription product ID")
	fs.StringVar(&token, "token", "", "Purchase token")
	fs.BoolVar(&confirm, "confirm", false, "Confirm revoking the legacy subscription purchase (required)")

	return &ffcli.Command{
		Name:      "revoke",
		ShortHelp: "Revoke a legacy subscription purchase",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			subscriptionID, token, err = requireSubscriptionLegacyTarget(subscriptionID, token)
			if err != nil {
				return err
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to revoke legacy subscription purchase %q", token)
			}

			if err := client.RevokeLegacySubscriptionPurchase(requestCtx, pkg, subscriptionID, token); err != nil {
				return wrapPurchaseEndpointError("failed to revoke legacy subscription purchase", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"api":            "legacy",
				"packageName":    pkg,
				"subscriptionId": subscriptionID,
				"token":          token,
				"status":         "revoked",
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
				return wrapPurchaseEndpointError("failed to list voided purchases", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":     pkg,
				"voidedPurchases": result.VoidedPurchases,
				"nextToken":       result.NextToken,
			})
		},
	}
}

func requireSubscriptionLegacyTarget(subscriptionID, token string) (string, string, error) {
	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID == "" {
		return "", "", fmt.Errorf("--subscription-id is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", fmt.Errorf("--token is required")
	}
	return subscriptionID, token, nil
}

func parsePositiveMillisFlag(flagName, raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", flagName)
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", flagName)
	}
	return value, nil
}

func wrapPurchaseEndpointError(prefix string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gpc.ErrPackageNotFound) {
		return fmt.Errorf("%s: %w\nhint: purchases endpoints can return package-not-found when billing access or purchase history is unavailable for this package. Verify package name, financial permissions, and purchase token source", prefix, err)
	}
	return fmt.Errorf("%s: %w", prefix, err)
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
