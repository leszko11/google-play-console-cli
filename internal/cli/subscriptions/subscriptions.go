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
	BatchGetSubscriptions(ctx context.Context, packageName string, productIDs []string) (gpc.SubscriptionsListInfo, error)
	CreateSubscription(ctx context.Context, packageName string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error)
	BatchUpdateSubscriptions(ctx context.Context, packageName string, requests []*androidpublisher.UpdateSubscriptionRequest) (gpc.SubscriptionsListInfo, error)
	UpdateSubscription(ctx context.Context, packageName, productID string, subscription *androidpublisher.Subscription) (gpc.SubscriptionInfo, error)
	DeleteSubscription(ctx context.Context, packageName, productID string) error
	ArchiveSubscription(ctx context.Context, packageName, productID string) error
	ActivateSubscriptionBasePlan(ctx context.Context, packageName, productID, basePlanID string) ([]gpc.SubscriptionInfo, error)
	DeactivateSubscriptionBasePlan(ctx context.Context, packageName, productID, basePlanID string) ([]gpc.SubscriptionInfo, error)
	DeleteSubscriptionBasePlan(ctx context.Context, packageName, productID, basePlanID string) error

	ListSubscriptionOffers(ctx context.Context, packageName, productID, basePlanID string, pageSize int64, pageToken string, paginate bool) (gpc.SubscriptionOffersListInfo, error)
	GetSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string) (gpc.SubscriptionOfferInfo, error)
	BatchGetSubscriptionOffers(ctx context.Context, packageName, productID, basePlanID string, offerIDs []string) (gpc.SubscriptionOffersListInfo, error)
	ActivateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string) (gpc.SubscriptionOfferInfo, error)
	DeactivateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string) (gpc.SubscriptionOfferInfo, error)
	CreateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID string, offer *androidpublisher.SubscriptionOffer) (gpc.SubscriptionOfferInfo, error)
	UpdateSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string, offer *androidpublisher.SubscriptionOffer, updateMask string) (gpc.SubscriptionOfferInfo, error)
	DeleteSubscriptionOffer(ctx context.Context, packageName, productID, basePlanID, offerID string) error
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
			newBatchGetCommand(deps),
			newCreateCommand(deps),
			newBatchUpdateCommand(deps),
			newUpdateCommand(deps),
			newDeleteCommand(deps),
			newArchiveCommand(deps),
			newBasePlansCommand(deps),
			newOffersCommand(deps),
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

func newBatchGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("batch-get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productIDsCSV string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productIDsCSV, "product-ids", "", "Comma-separated subscription product IDs")

	return &ffcli.Command{
		Name:      "batch-get",
		ShortHelp: "Batch-get subscriptions by product IDs",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			productIDsCSV = strings.TrimSpace(productIDsCSV)
			if productIDsCSV == "" {
				return fmt.Errorf("--product-ids is required")
			}
			productIDs := strings.Split(productIDsCSV, ",")

			result, err := client.BatchGetSubscriptions(requestCtx, pkg, productIDs)
			if err != nil {
				return fmt.Errorf("failed to batch-get subscriptions: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"subscriptions": result.Subscriptions,
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

func newBatchUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("batch-update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&inputPath, "input", "", "Path to subscriptions batch update JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "batch-update",
		ShortHelp: "Batch create or update subscriptions",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			payload, err := readSubscriptionBatchUpdatePayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}

			result, err := client.BatchUpdateSubscriptions(requestCtx, pkg, payload.Requests)
			if err != nil {
				return fmt.Errorf("failed to batch-update subscriptions: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"subscriptions": result.Subscriptions,
				"status":        "updated",
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

func newBasePlansCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "base-plans",
		ShortHelp: "Manage subscription base plans",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newBasePlansActivateCommand(deps),
			newBasePlansDeactivateCommand(deps),
			newBasePlansDeleteCommand(deps),
		},
	}
}

func newBasePlansActivateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("activate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")

	return &ffcli.Command{
		Name:      "activate",
		ShortHelp: "Activate a subscription base plan",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}

			subscriptions, err := client.ActivateSubscriptionBasePlan(requestCtx, pkg, productID, basePlanID)
			if err != nil {
				return fmt.Errorf("failed to activate subscription base plan: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"productId":     productID,
				"basePlanId":    basePlanID,
				"subscriptions": subscriptions,
				"status":        "activated",
			})
		},
	}
}

func newBasePlansDeactivateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("deactivate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deactivating the base plan (required)")

	return &ffcli.Command{
		Name:      "deactivate",
		ShortHelp: "Deactivate a subscription base plan",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to deactivate base plan %q", basePlanID)
			}

			subscriptions, err := client.DeactivateSubscriptionBasePlan(requestCtx, pkg, productID, basePlanID)
			if err != nil {
				return fmt.Errorf("failed to deactivate subscription base plan: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"productId":     productID,
				"basePlanId":    basePlanID,
				"subscriptions": subscriptions,
				"status":        "deactivated",
			})
		},
	}
}

func newBasePlansDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the base plan (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete a subscription base plan",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to delete base plan %q", basePlanID)
			}

			if err := client.DeleteSubscriptionBasePlan(requestCtx, pkg, productID, basePlanID); err != nil {
				return fmt.Errorf("failed to delete subscription base plan: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"basePlanId":  basePlanID,
				"status":      "deleted",
			})
		},
	}
}

func newOffersCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "offers",
		ShortHelp: "Manage subscription offers under base plans",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newOffersListCommand(deps),
			newOffersGetCommand(deps),
			newOffersBatchGetCommand(deps),
			newOffersActivateCommand(deps),
			newOffersDeactivateCommand(deps),
			newOffersCreateCommand(deps),
			newOffersUpdateCommand(deps),
			newOffersDeleteCommand(deps),
		},
	}
}

func newOffersListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID, pageToken string
	var pageSize int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.Int64Var(&pageSize, "page-size", 0, "Maximum offers per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List offers under a subscription base plan",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			if pageSize < 0 {
				return fmt.Errorf("--page-size must be greater than or equal to zero")
			}

			result, err := client.ListSubscriptionOffers(requestCtx, pkg, productID, basePlanID, pageSize, pageToken, shared.ActiveGlobalFlags().Paginate)
			if err != nil {
				return fmt.Errorf("failed to list subscription offers: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"offers":        result.Offers,
				"nextPageToken": result.NextPageToken,
			})
		},
	}
}

func newOffersGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID, offerID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.StringVar(&offerID, "offer-id", "", "Offer ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get one subscription offer",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			offerID = strings.TrimSpace(offerID)
			if offerID == "" {
				return fmt.Errorf("--offer-id is required")
			}

			offer, err := client.GetSubscriptionOffer(requestCtx, pkg, productID, basePlanID, offerID)
			if err != nil {
				return fmt.Errorf("failed to get subscription offer: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"offer":       offer,
			})
		},
	}
}

func newOffersCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.StringVar(&inputPath, "input", "", "Path to subscription offer JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a subscription offer",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			offer, err := readSubscriptionOfferPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}

			created, err := client.CreateSubscriptionOffer(requestCtx, pkg, productID, basePlanID, offer)
			if err != nil {
				return fmt.Errorf("failed to create subscription offer: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"offer":       created,
				"status":      "created",
			})
		},
	}
}

func newOffersBatchGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("batch-get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID, offerIDsCSV string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.StringVar(&offerIDsCSV, "offer-ids", "", "Comma-separated offer IDs")

	return &ffcli.Command{
		Name:      "batch-get",
		ShortHelp: "Batch-get subscription offers",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			offerIDsCSV = strings.TrimSpace(offerIDsCSV)
			if offerIDsCSV == "" {
				return fmt.Errorf("--offer-ids is required")
			}
			offerIDs := strings.Split(offerIDsCSV, ",")

			result, err := client.BatchGetSubscriptionOffers(requestCtx, pkg, productID, basePlanID, offerIDs)
			if err != nil {
				return fmt.Errorf("failed to batch-get subscription offers: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"basePlanId":  basePlanID,
				"offers":      result.Offers,
			})
		},
	}
}

func newOffersActivateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("activate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID, offerID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.StringVar(&offerID, "offer-id", "", "Offer ID")

	return &ffcli.Command{
		Name:      "activate",
		ShortHelp: "Activate a subscription offer",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			offerID = strings.TrimSpace(offerID)
			if offerID == "" {
				return fmt.Errorf("--offer-id is required")
			}

			offer, err := client.ActivateSubscriptionOffer(requestCtx, pkg, productID, basePlanID, offerID)
			if err != nil {
				return fmt.Errorf("failed to activate subscription offer: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"basePlanId":  basePlanID,
				"offer":       offer,
				"status":      "activated",
			})
		},
	}
}

func newOffersDeactivateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("deactivate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID, offerID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.StringVar(&offerID, "offer-id", "", "Offer ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deactivating the offer (required)")

	return &ffcli.Command{
		Name:      "deactivate",
		ShortHelp: "Deactivate a subscription offer",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			offerID = strings.TrimSpace(offerID)
			if offerID == "" {
				return fmt.Errorf("--offer-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to deactivate offer %q", offerID)
			}

			offer, err := client.DeactivateSubscriptionOffer(requestCtx, pkg, productID, basePlanID, offerID)
			if err != nil {
				return fmt.Errorf("failed to deactivate subscription offer: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"basePlanId":  basePlanID,
				"offer":       offer,
				"status":      "deactivated",
			})
		},
	}
}

func newOffersUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID, offerID, inputPath, updateMask string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.StringVar(&offerID, "offer-id", "", "Offer ID")
	fs.StringVar(&inputPath, "input", "", "Path to subscription offer JSON payload (use - for stdin)")
	fs.StringVar(&updateMask, "update-mask", "", "Comma-separated list of fields to update")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update a subscription offer",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			offerID = strings.TrimSpace(offerID)
			if offerID == "" {
				return fmt.Errorf("--offer-id is required")
			}
			offer, err := readSubscriptionOfferPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}

			updated, err := client.UpdateSubscriptionOffer(requestCtx, pkg, productID, basePlanID, offerID, offer, updateMask)
			if err != nil {
				return fmt.Errorf("failed to update subscription offer: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"offer":       updated,
				"status":      "updated",
			})
		},
	}
}

func newOffersDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID, offerID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.StringVar(&offerID, "offer-id", "", "Offer ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the offer (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete a subscription offer",
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
			basePlanID = strings.TrimSpace(basePlanID)
			if basePlanID == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			offerID = strings.TrimSpace(offerID)
			if offerID == "" {
				return fmt.Errorf("--offer-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to delete offer %q", offerID)
			}

			if err := client.DeleteSubscriptionOffer(requestCtx, pkg, productID, basePlanID, offerID); err != nil {
				return fmt.Errorf("failed to delete subscription offer: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"basePlanId":  basePlanID,
				"offerId":     offerID,
				"status":      "deleted",
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

func readSubscriptionOfferPayload(inputPath string, stdin io.Reader) (*androidpublisher.SubscriptionOffer, error) {
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

	var offer androidpublisher.SubscriptionOffer
	if err := json.Unmarshal(raw, &offer); err != nil {
		return nil, fmt.Errorf("invalid subscription offer JSON payload: %w", err)
	}
	return &offer, nil
}

func readSubscriptionBatchUpdatePayload(inputPath string, stdin io.Reader) (*androidpublisher.BatchUpdateSubscriptionsRequest, error) {
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

	var payload androidpublisher.BatchUpdateSubscriptionsRequest
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid subscriptions batch update JSON payload: %w", err)
	}
	return &payload, nil
}
