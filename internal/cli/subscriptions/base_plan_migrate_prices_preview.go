package subscriptions

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

type migratePricesPreviewResult struct {
	Status                      string                                         `json:"status"`
	PackageName                 string                                         `json:"packageName"`
	ProductID                   string                                         `json:"productId"`
	BasePlanID                  string                                         `json:"basePlanId"`
	RegionsVersion              string                                         `json:"regionsVersion"`
	RegionalPriceMigrationCount int                                            `json:"regionalPriceMigrationCount"`
	Warnings                    []string                                       `json:"warnings,omitempty"`
	Request                     *androidpublisher.MigrateBasePlanPricesRequest `json:"request"`
}

func newBasePlansMigratePricesPreviewCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("preview", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, productID, basePlanID, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&productID, "product-id", "", "Subscription product ID")
	fs.StringVar(&basePlanID, "base-plan-id", "", "Base plan ID")
	fs.StringVar(&inputPath, "input", "", "Path to base plan migrate-prices JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "preview",
		ShortHelp: "Preview normalized base plan price migration payload without mutating Play",
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
			payload, err := readBasePlanMigratePricesPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			preview, err := buildMigratePricesPreview(requestCtx, client, pkg, productID, basePlanID, payload)
			if err != nil {
				return err
			}

			switch shared.ResolveOutput("") {
			case "json":
				return shared.WriteJSON(deps.Stdout, preview)
			case "table":
				return writeMigratePricesPreviewTable(deps.Stdout, preview)
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(""))
			}
		},
	}
}

func buildMigratePricesPreview(ctx context.Context, client Client, packageName, productID, basePlanID string, payload *androidpublisher.MigrateBasePlanPricesRequest) (migratePricesPreviewResult, error) {
	if payload == nil {
		return migratePricesPreviewResult{}, fmt.Errorf("migrate prices payload is required")
	}
	if len(payload.RegionalPriceMigrations) == 0 {
		return migratePricesPreviewResult{}, fmt.Errorf("migrate prices payload must include at least one regional price migration")
	}

	regionsVersion := ""
	if payload.RegionsVersion != nil {
		regionsVersion = strings.TrimSpace(payload.RegionsVersion.Version)
	}
	if regionsVersion == "" {
		resolved, err := client.GetLatestRegionsVersion(ctx, packageName)
		if err != nil {
			return migratePricesPreviewResult{}, fmt.Errorf("failed to resolve regions version: %w", err)
		}
		regionsVersion = resolved
	}

	normalized := &androidpublisher.MigrateBasePlanPricesRequest{
		PackageName:             packageName,
		ProductId:               productID,
		BasePlanId:              basePlanID,
		LatencyTolerance:        strings.TrimSpace(payload.LatencyTolerance),
		RegionalPriceMigrations: payload.RegionalPriceMigrations,
		RegionsVersion:          &androidpublisher.RegionsVersion{Version: regionsVersion},
	}

	warnings := make([]string, 0, len(payload.RegionalPriceMigrations))
	for idx, migration := range payload.RegionalPriceMigrations {
		if migration == nil {
			warnings = append(warnings, fmt.Sprintf("migration %d is empty", idx))
			continue
		}
		if strings.TrimSpace(migration.RegionCode) == "" {
			warnings = append(warnings, fmt.Sprintf("migration %d is missing regionCode", idx))
		}
		if strings.TrimSpace(migration.OldestAllowedPriceVersionTime) == "" {
			warnings = append(warnings, fmt.Sprintf("migration %d is missing oldestAllowedPriceVersionTime", idx))
		}
	}

	return migratePricesPreviewResult{
		Status:                      "preview",
		PackageName:                 packageName,
		ProductID:                   productID,
		BasePlanID:                  basePlanID,
		RegionsVersion:              regionsVersion,
		RegionalPriceMigrationCount: len(payload.RegionalPriceMigrations),
		Warnings:                    warnings,
		Request:                     normalized,
	}, nil
}

func writeMigratePricesPreviewTable(out io.Writer, result migratePricesPreviewResult) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", result.PackageName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PRODUCT_ID\t%s\n", result.ProductID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "BASE_PLAN_ID\t%s\n", result.BasePlanID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "REGIONS_VERSION\t%s\n", result.RegionsVersion); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "MIGRATIONS\t%d\n", result.RegionalPriceMigrationCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "REGION\tOLDEST_ALLOWED_PRICE_VERSION_TIME"); err != nil {
		return err
	}
	for _, migration := range result.Request.RegionalPriceMigrations {
		if migration == nil {
			if _, err := fmt.Fprintln(out, "-\t-"); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(out, "%s\t%s\n", migration.RegionCode, migration.OldestAllowedPriceVersionTime); err != nil {
			return err
		}
	}
	for _, warning := range result.Warnings {
		if _, err := fmt.Fprintf(out, "WARNING\t%s\n", warning); err != nil {
			return err
		}
	}
	return nil
}
