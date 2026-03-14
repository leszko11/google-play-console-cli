package products

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

type syncOptions struct {
	PackageName   string
	Dir           string
	Confirm       bool
	DryRun        bool
	DeleteMissing bool
}

type syncResult struct {
	PackageName string   `json:"packageName"`
	Dir         string   `json:"dir"`
	Status      string   `json:"status"`
	Created     []string `json:"created,omitempty"`
	Updated     []string `json:"updated,omitempty"`
	Deleted     []string `json:"deleted,omitempty"`
	Planned     []string `json:"planned,omitempty"`
}

func newSyncCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var opts syncOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Dir, "dir", "", "Directory containing exported product JSON files")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm applying product changes (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Plan product changes without mutating Play")
	fs.BoolVar(&opts.DeleteMissing, "delete-missing", false, "Delete remote products not present in the local directory")

	return &ffcli.Command{
		Name:      "sync",
		ShortHelp: "Sync one-time products from exported JSON files",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts.Dir = strings.TrimSpace(opts.Dir)
			if opts.Dir == "" {
				return fmt.Errorf("--dir is required")
			}
			if !opts.DryRun && !opts.Confirm {
				return fmt.Errorf("--confirm is required unless --dry-run is set")
			}
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, opts.PackageName)
			if err != nil {
				return err
			}
			defer cancel()
			opts.PackageName = pkg
			products, err := readSyncProductsDir(opts.Dir)
			if err != nil {
				return err
			}
			return runSync(ctx, requestCtx, client, deps.Stdout, opts, products)
		},
	}
}

func readSyncProductsDir(dir string) ([]*androidpublisher.OneTimeProduct, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read products directory: %w", err)
	}
	products := make([]*androidpublisher.OneTimeProduct, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		product, err := readOneTimeProductPayload(filepath.Join(dir, entry.Name()), nil)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	sort.Slice(products, func(i, j int) bool { return products[i].ProductId < products[j].ProductId })
	if len(products) == 0 {
		return nil, fmt.Errorf("no product JSON files found in %s", dir)
	}
	return products, nil
}

func runSync(parentCtx, requestCtx context.Context, client Client, out io.Writer, opts syncOptions, products []*androidpublisher.OneTimeProduct) error {
	result := syncResult{
		PackageName: opts.PackageName,
		Dir:         opts.Dir,
		Status:      "dry-run",
	}

	remote, err := client.ListOneTimeProducts(requestCtx, opts.PackageName, 0, "", true)
	if err != nil {
		return fmt.Errorf("failed to list remote products: %w", err)
	}
	remoteSet := make(map[string]struct{}, len(remote.Products))
	for _, item := range remote.Products {
		remoteSet[item.ProductID] = struct{}{}
	}

	localSet := make(map[string]struct{}, len(products))
	for _, product := range products {
		localSet[product.ProductId] = struct{}{}
		if _, ok := remoteSet[product.ProductId]; ok {
			result.Updated = append(result.Updated, product.ProductId)
			result.Planned = append(result.Planned, "update "+product.ProductId)
			continue
		}
		result.Created = append(result.Created, product.ProductId)
		result.Planned = append(result.Planned, "create "+product.ProductId)
	}
	if opts.DeleteMissing {
		for _, item := range remote.Products {
			if _, ok := localSet[item.ProductID]; ok {
				continue
			}
			result.Deleted = append(result.Deleted, item.ProductID)
			result.Planned = append(result.Planned, "delete "+item.ProductID)
		}
		sort.Strings(result.Deleted)
	}
	if opts.DryRun {
		return shared.WriteJSON(out, result)
	}

	for _, product := range products {
		if _, err := client.CreateOneTimeProduct(requestCtx, opts.PackageName, product); err != nil {
			return fmt.Errorf("failed to upsert product %q: %w", product.ProductId, err)
		}
	}
	for _, productID := range result.Deleted {
		if err := client.DeleteOneTimeProduct(requestCtx, opts.PackageName, productID); err != nil {
			return fmt.Errorf("failed to delete product %q: %w", productID, err)
		}
	}
	result.Status = "committed"
	return shared.WriteJSON(out, result)
}
