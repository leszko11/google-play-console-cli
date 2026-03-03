package products

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
	ListOneTimeProducts(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (gpc.OneTimeProductsListInfo, error)
	GetOneTimeProduct(ctx context.Context, packageName, productID string) (gpc.OneTimeProductInfo, error)
	CreateOneTimeProduct(ctx context.Context, packageName string, product *androidpublisher.OneTimeProduct) (gpc.OneTimeProductInfo, error)
	UpdateOneTimeProduct(ctx context.Context, packageName, productID string, product *androidpublisher.OneTimeProduct, updateMask string) (gpc.OneTimeProductInfo, error)
	DeleteOneTimeProduct(ctx context.Context, packageName, productID string) error
	ListOneTimeProductOffers(ctx context.Context, packageName, productID, purchaseOptionID string, pageSize int64, pageToken string, paginate bool) (gpc.OneTimeProductOffersListInfo, error)
	BatchGetOneTimeProductOffers(ctx context.Context, packageName, productID, purchaseOptionID string, offerIDs []string) (gpc.OneTimeProductOffersListInfo, error)
	BatchUpdateOneTimeProductOffers(ctx context.Context, packageName, productID, purchaseOptionID string, requests []*androidpublisher.UpdateOneTimeProductOfferRequest) (gpc.OneTimeProductOffersListInfo, error)
	BatchDeleteOneTimeProductOffers(ctx context.Context, packageName, productID, purchaseOptionID string, requests []*androidpublisher.DeleteOneTimeProductOfferRequest) error
	ActivateOneTimeProductOffer(ctx context.Context, packageName, productID, purchaseOptionID, offerID string) (gpc.OneTimeProductOfferInfo, error)
	DeactivateOneTimeProductOffer(ctx context.Context, packageName, productID, purchaseOptionID, offerID string) (gpc.OneTimeProductOfferInfo, error)
	CancelOneTimeProductOffer(ctx context.Context, packageName, productID, purchaseOptionID, offerID string) (gpc.OneTimeProductOfferInfo, error)
	ActivateOneTimeProductPurchaseOption(ctx context.Context, packageName, productID, purchaseOptionID string) ([]gpc.OneTimeProductInfo, error)
	DeactivateOneTimeProductPurchaseOption(ctx context.Context, packageName, productID, purchaseOptionID string) ([]gpc.OneTimeProductInfo, error)
	DeleteOneTimeProductPurchaseOption(ctx context.Context, packageName, productID, purchaseOptionID string, force bool) error
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
		Name:      "products",
		ShortHelp: "Manage monetization one-time products",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newGetCommand(deps),
			newCreateCommand(deps),
			newUpdateCommand(deps),
			newDeleteCommand(deps),
			newOffersCommand(deps),
			newPurchaseOptionsCommand(deps),
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
	fs.Int64Var(&pageSize, "page-size", 0, "Maximum one-time products per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List one-time products",
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
			result, err := client.ListOneTimeProducts(requestCtx, pkg, pageSize, pageToken, shared.ActiveGlobalFlags().Paginate)
			if err != nil {
				return fmt.Errorf("failed to list one-time products: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"products":      result.Products,
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
	fs.StringVar(&productID, "product-id", "", "One-time product ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get a one-time product by product ID",
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
			product, err := client.GetOneTimeProduct(requestCtx, pkg, productID)
			if err != nil {
				return fmt.Errorf("failed to get one-time product: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"product":     product,
			})
		},
	}
}

func newCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&inputPath, "input", "", "Path to one-time product JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a one-time product (patch with allowMissing=true)",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			product, err := readOneTimeProductPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			created, err := client.CreateOneTimeProduct(requestCtx, pkg, product)
			if err != nil {
				return fmt.Errorf("failed to create one-time product: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"product":     created,
				"status":      "created",
			})
		},
	}
}

func newUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, inputPath, updateMask string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&inputPath, "input", "", "Path to one-time product JSON payload (use - for stdin)")
	fs.StringVar(&updateMask, "update-mask", "", "Comma-separated list of fields to update")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update a one-time product",
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
			product, err := readOneTimeProductPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			updated, err := client.UpdateOneTimeProduct(requestCtx, pkg, productID, product, updateMask)
			if err != nil {
				return fmt.Errorf("failed to update one-time product: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"product":     updated,
				"status":      "updated",
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
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the one-time product (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete a one-time product",
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
				return fmt.Errorf("--confirm is required to delete one-time product %q", productID)
			}
			if err := client.DeleteOneTimeProduct(requestCtx, pkg, productID); err != nil {
				return fmt.Errorf("failed to delete one-time product: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"productId":   productID,
				"status":      "deleted",
			})
		},
	}
}

func newOffersCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "offers",
		ShortHelp: "Manage one-time product offers under purchase options",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newOffersListCommand(deps),
			newOffersBatchGetCommand(deps),
			newOffersBatchUpdateCommand(deps),
			newOffersBatchDeleteCommand(deps),
			newOffersActivateCommand(deps),
			newOffersDeactivateCommand(deps),
			newOffersCancelCommand(deps),
		},
	}
}

func newOffersListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID, pageToken string
	var pageSize int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")
	fs.Int64Var(&pageSize, "page-size", 0, "Maximum offers per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List offers for a one-time product purchase option",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			if pageSize < 0 {
				return fmt.Errorf("--page-size must be greater than or equal to zero")
			}

			result, err := client.ListOneTimeProductOffers(requestCtx, pkg, productID, purchaseOptionID, pageSize, pageToken, shared.ActiveGlobalFlags().Paginate)
			if err != nil {
				return fmt.Errorf("failed to list one-time product offers: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"offers":           result.Offers,
				"nextPageToken":    result.NextPageToken,
			})
		},
	}
}

func newOffersBatchGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("batch-get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID, offerIDsCSV string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")
	fs.StringVar(&offerIDsCSV, "offer-ids", "", "Comma-separated offer IDs")

	return &ffcli.Command{
		Name:      "batch-get",
		ShortHelp: "Batch-get one-time product offers",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			offerIDsCSV = strings.TrimSpace(offerIDsCSV)
			if offerIDsCSV == "" {
				return fmt.Errorf("--offer-ids is required")
			}
			offerIDs := strings.Split(offerIDsCSV, ",")

			result, err := client.BatchGetOneTimeProductOffers(requestCtx, pkg, productID, purchaseOptionID, offerIDs)
			if err != nil {
				return fmt.Errorf("failed to batch-get one-time product offers: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"offers":           result.Offers,
			})
		},
	}
}

func newOffersBatchUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("batch-update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")
	fs.StringVar(&inputPath, "input", "", "Path to one-time product offers batch update JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "batch-update",
		ShortHelp: "Batch create or update one-time product offers",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			payload, err := readOneTimeProductOffersBatchUpdatePayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}

			result, err := client.BatchUpdateOneTimeProductOffers(requestCtx, pkg, productID, purchaseOptionID, payload.Requests)
			if err != nil {
				return fmt.Errorf("failed to batch-update one-time product offers: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"offers":           result.Offers,
				"status":           "updated",
			})
		},
	}
}

func newOffersBatchDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("batch-delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID, inputPath string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")
	fs.StringVar(&inputPath, "input", "", "Path to one-time product offers batch delete JSON payload (use - for stdin)")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the offers (required)")

	return &ffcli.Command{
		Name:      "batch-delete",
		ShortHelp: "Batch delete one-time product offers",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			payload, err := readOneTimeProductOffersBatchDeletePayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to batch-delete offers for purchase option %q", purchaseOptionID)
			}

			if err := client.BatchDeleteOneTimeProductOffers(requestCtx, pkg, productID, purchaseOptionID, payload.Requests); err != nil {
				return fmt.Errorf("failed to batch-delete one-time product offers: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"deletedCount":     len(payload.Requests),
				"status":           "deleted",
			})
		},
	}
}

func newOffersActivateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("activate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID, offerID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")
	fs.StringVar(&offerID, "offer-id", "", "Offer ID")

	return &ffcli.Command{
		Name:      "activate",
		ShortHelp: "Activate a one-time product offer",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			offerID = strings.TrimSpace(offerID)
			if offerID == "" {
				return fmt.Errorf("--offer-id is required")
			}

			offer, err := client.ActivateOneTimeProductOffer(requestCtx, pkg, productID, purchaseOptionID, offerID)
			if err != nil {
				return fmt.Errorf("failed to activate one-time product offer: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"offer":            offer,
				"status":           "activated",
			})
		},
	}
}

func newOffersDeactivateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("deactivate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID, offerID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")
	fs.StringVar(&offerID, "offer-id", "", "Offer ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deactivating the offer (required)")

	return &ffcli.Command{
		Name:      "deactivate",
		ShortHelp: "Deactivate a one-time product offer",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			offerID = strings.TrimSpace(offerID)
			if offerID == "" {
				return fmt.Errorf("--offer-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to deactivate offer %q", offerID)
			}

			offer, err := client.DeactivateOneTimeProductOffer(requestCtx, pkg, productID, purchaseOptionID, offerID)
			if err != nil {
				return fmt.Errorf("failed to deactivate one-time product offer: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"offer":            offer,
				"status":           "deactivated",
			})
		},
	}
}

func newOffersCancelCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID, offerID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")
	fs.StringVar(&offerID, "offer-id", "", "Offer ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm canceling the offer (required)")

	return &ffcli.Command{
		Name:      "cancel",
		ShortHelp: "Cancel a one-time product pre-order offer",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			offerID = strings.TrimSpace(offerID)
			if offerID == "" {
				return fmt.Errorf("--offer-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to cancel offer %q", offerID)
			}

			offer, err := client.CancelOneTimeProductOffer(requestCtx, pkg, productID, purchaseOptionID, offerID)
			if err != nil {
				return fmt.Errorf("failed to cancel one-time product offer: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"offer":            offer,
				"status":           "canceled",
			})
		},
	}
}

func newPurchaseOptionsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "purchase-options",
		ShortHelp: "Manage one-time product purchase options",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newPurchaseOptionsActivateCommand(deps),
			newPurchaseOptionsDeactivateCommand(deps),
			newPurchaseOptionsDeleteCommand(deps),
		},
	}
}

func newPurchaseOptionsActivateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("activate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")

	return &ffcli.Command{
		Name:      "activate",
		ShortHelp: "Activate a one-time product purchase option",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}

			products, err := client.ActivateOneTimeProductPurchaseOption(requestCtx, pkg, productID, purchaseOptionID)
			if err != nil {
				return fmt.Errorf("failed to activate one-time product purchase option: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"products":         products,
				"status":           "activated",
			})
		},
	}
}

func newPurchaseOptionsDeactivateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("deactivate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deactivating the purchase option (required)")

	return &ffcli.Command{
		Name:      "deactivate",
		ShortHelp: "Deactivate a one-time product purchase option",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to deactivate purchase option %q", purchaseOptionID)
			}

			products, err := client.DeactivateOneTimeProductPurchaseOption(requestCtx, pkg, productID, purchaseOptionID)
			if err != nil {
				return fmt.Errorf("failed to deactivate one-time product purchase option: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"products":         products,
				"status":           "deactivated",
			})
		},
	}
}

func newPurchaseOptionsDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, purchaseOptionID string
	var confirm, force bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "One-time product ID")
	fs.StringVar(&purchaseOptionID, "purchase-option-id", "", "Purchase option ID")
	fs.BoolVar(&force, "force", false, "Delete even when managed externally")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the purchase option (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete a one-time product purchase option",
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
			purchaseOptionID = strings.TrimSpace(purchaseOptionID)
			if purchaseOptionID == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to delete purchase option %q", purchaseOptionID)
			}

			if err := client.DeleteOneTimeProductPurchaseOption(requestCtx, pkg, productID, purchaseOptionID, force); err != nil {
				return fmt.Errorf("failed to delete one-time product purchase option: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"productId":        productID,
				"purchaseOptionId": purchaseOptionID,
				"force":            force,
				"status":           "deleted",
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

func readOneTimeProductPayload(inputPath string, stdin io.Reader) (*androidpublisher.OneTimeProduct, error) {
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

	var product androidpublisher.OneTimeProduct
	if err := json.Unmarshal(raw, &product); err != nil {
		return nil, fmt.Errorf("invalid one-time product JSON payload: %w", err)
	}
	return &product, nil
}

func readOneTimeProductOffersBatchUpdatePayload(inputPath string, stdin io.Reader) (*androidpublisher.BatchUpdateOneTimeProductOffersRequest, error) {
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

	var payload androidpublisher.BatchUpdateOneTimeProductOffersRequest
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid one-time product offers batch update JSON payload: %w", err)
	}
	return &payload, nil
}

func readOneTimeProductOffersBatchDeletePayload(inputPath string, stdin io.Reader) (*androidpublisher.BatchDeleteOneTimeProductOffersRequest, error) {
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

	var payload androidpublisher.BatchDeleteOneTimeProductOffersRequest
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid one-time product offers batch delete JSON payload: %w", err)
	}
	return &payload, nil
}
