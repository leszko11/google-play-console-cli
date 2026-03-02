package iap

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
	ListIAPs(ctx context.Context, packageName string, maxResults int64, pageToken string, paginate bool) (gpc.IAPsListInfo, error)
	GetIAP(ctx context.Context, packageName, sku string) (gpc.IAPInfo, error)
	CreateIAP(ctx context.Context, packageName string, product *androidpublisher.InAppProduct) (gpc.IAPInfo, error)
	UpdateIAP(ctx context.Context, packageName, sku string, product *androidpublisher.InAppProduct) (gpc.IAPInfo, error)
	DeleteIAP(ctx context.Context, packageName, sku string) error
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
		Name:      "iap",
		ShortHelp: "Manage legacy in-app products",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newGetCommand(deps),
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
	var packageName, pageToken string
	var maxResults int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&maxResults, "max-results", 0, "Maximum in-app products per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List legacy in-app products",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			if maxResults < 0 {
				return fmt.Errorf("--max-results must be greater than or equal to zero")
			}
			result, err := client.ListIAPs(requestCtx, pkg, maxResults, pageToken, shared.ActiveGlobalFlags().Paginate)
			if err != nil {
				return fmt.Errorf("failed to list in-app products: %w", err)
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
	var packageName, sku string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&sku, "sku", "", "In-app product SKU")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get a legacy in-app product by SKU",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			sku = strings.TrimSpace(sku)
			if sku == "" {
				return fmt.Errorf("--sku is required")
			}
			product, err := client.GetIAP(requestCtx, pkg, sku)
			if err != nil {
				return fmt.Errorf("failed to get in-app product: %w", err)
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
	fs.StringVar(&inputPath, "input", "", "Path to in-app product JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a legacy in-app product",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			product, err := readIAPPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			created, err := client.CreateIAP(requestCtx, pkg, product)
			if err != nil {
				return fmt.Errorf("failed to create in-app product: %w", err)
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
	var packageName, sku, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&sku, "sku", "", "In-app product SKU")
	fs.StringVar(&inputPath, "input", "", "Path to in-app product JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update a legacy in-app product",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			sku = strings.TrimSpace(sku)
			if sku == "" {
				return fmt.Errorf("--sku is required")
			}
			product, err := readIAPPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			updated, err := client.UpdateIAP(requestCtx, pkg, sku, product)
			if err != nil {
				return fmt.Errorf("failed to update in-app product: %w", err)
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
	var packageName, sku string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&sku, "sku", "", "In-app product SKU")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the in-app product (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete a legacy in-app product",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			sku = strings.TrimSpace(sku)
			if sku == "" {
				return fmt.Errorf("--sku is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to delete in-app product %q", sku)
			}
			if err := client.DeleteIAP(requestCtx, pkg, sku); err != nil {
				return fmt.Errorf("failed to delete in-app product: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"sku":         sku,
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

func readIAPPayload(inputPath string, stdin io.Reader) (*androidpublisher.InAppProduct, error) {
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

	var product androidpublisher.InAppProduct
	if err := json.Unmarshal(raw, &product); err != nil {
		return nil, fmt.Errorf("invalid in-app product JSON payload: %w", err)
	}
	return &product, nil
}
