package subscriptions

import (
	"context"
	"encoding/json"
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

type syncSubscriptionFile struct {
	Subscription   *androidpublisher.Subscription `json:"subscription"`
	RegionsVersion string                         `json:"regionsVersion,omitempty"`
}

func newSyncCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var opts syncOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Dir, "dir", "", "Directory containing exported subscription JSON files")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm applying subscription changes (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Plan subscription changes without mutating Play")
	fs.BoolVar(&opts.DeleteMissing, "delete-missing", false, "Delete remote subscriptions not present in the local directory")

	return &ffcli.Command{
		Name:      "sync",
		ShortHelp: "Sync subscriptions from exported JSON files",
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
			subscriptions, err := readSyncSubscriptionsDir(opts.Dir)
			if err != nil {
				return err
			}
			return runSync(requestCtx, client, deps.Stdout, opts, subscriptions)
		},
	}
}

func readSyncSubscriptionsDir(dir string) ([]syncSubscriptionFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read subscriptions directory: %w", err)
	}
	files := make([]syncSubscriptionFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		item, err := readSyncSubscriptionFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		files = append(files, item)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Subscription.ProductId < files[j].Subscription.ProductId })
	if len(files) == 0 {
		return nil, fmt.Errorf("no subscription JSON files found in %s", dir)
	}
	return files, nil
}

func readSyncSubscriptionFile(path string) (syncSubscriptionFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return syncSubscriptionFile{}, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var wrapped syncSubscriptionFile
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Subscription != nil {
		return wrapped, nil
	}
	var subscription androidpublisher.Subscription
	if err := json.Unmarshal(raw, &subscription); err != nil {
		return syncSubscriptionFile{}, fmt.Errorf("invalid subscription JSON payload: %w", err)
	}
	return syncSubscriptionFile{Subscription: &subscription}, nil
}

func runSync(requestCtx context.Context, client Client, out io.Writer, opts syncOptions, subscriptions []syncSubscriptionFile) error {
	result := syncResult{
		PackageName: opts.PackageName,
		Dir:         opts.Dir,
		Status:      "dry-run",
	}

	remote, err := client.ListSubscriptions(requestCtx, opts.PackageName, 0, "", true)
	if err != nil {
		return fmt.Errorf("failed to list remote subscriptions: %w", err)
	}
	remoteSet := make(map[string]struct{}, len(remote.Subscriptions))
	for _, item := range remote.Subscriptions {
		remoteSet[item.ProductID] = struct{}{}
	}
	defaultRegionsVersion := ""

	requests := make([]*androidpublisher.UpdateSubscriptionRequest, 0, len(subscriptions))
	localSet := make(map[string]struct{}, len(subscriptions))
	for _, item := range subscriptions {
		productID := item.Subscription.ProductId
		if strings.TrimSpace(item.RegionsVersion) == "" {
			if defaultRegionsVersion == "" {
				resolved, err := client.GetLatestRegionsVersion(requestCtx, opts.PackageName)
				if err != nil {
					return fmt.Errorf("failed to resolve regions version: %w", err)
				}
				defaultRegionsVersion = resolved
			}
			item.RegionsVersion = defaultRegionsVersion
		}
		localSet[productID] = struct{}{}
		updateMask := subscriptionSyncUpdateMask(item.Subscription)
		requests = append(requests, &androidpublisher.UpdateSubscriptionRequest{
			AllowMissing:   true,
			RegionsVersion: &androidpublisher.RegionsVersion{Version: item.RegionsVersion},
			Subscription:   item.Subscription,
			UpdateMask:     updateMask,
		})
		if _, ok := remoteSet[productID]; ok {
			result.Updated = append(result.Updated, productID)
			result.Planned = append(result.Planned, "update "+productID)
			continue
		}
		result.Created = append(result.Created, productID)
		result.Planned = append(result.Planned, "create "+productID)
	}
	if opts.DeleteMissing {
		for _, item := range remote.Subscriptions {
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

	for start := 0; start < len(requests); start += 100 {
		end := start + 100
		if end > len(requests) {
			end = len(requests)
		}
		if _, err := client.BatchUpdateSubscriptions(requestCtx, opts.PackageName, requests[start:end]); err != nil {
			return fmt.Errorf("failed to upsert subscriptions: %w", err)
		}
	}
	for _, productID := range result.Deleted {
		if err := client.DeleteSubscription(requestCtx, opts.PackageName, productID); err != nil {
			return fmt.Errorf("failed to delete subscription %q: %w", productID, err)
		}
	}
	result.Status = "committed"
	return shared.WriteJSON(out, result)
}

func subscriptionSyncUpdateMask(subscription *androidpublisher.Subscription) string {
	if subscription == nil {
		return ""
	}
	paths := make([]string, 0, 4)
	if len(subscription.BasePlans) > 0 {
		paths = append(paths, "basePlans")
	}
	if len(subscription.Listings) > 0 {
		paths = append(paths, "listings")
	}
	if subscription.RestrictedPaymentCountries != nil {
		paths = append(paths, "restrictedPaymentCountries")
	}
	if subscription.TaxAndComplianceSettings != nil {
		paths = append(paths, "taxAndComplianceSettings")
	}
	return strings.Join(paths, ",")
}
