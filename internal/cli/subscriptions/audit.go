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
	"strconv"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

type auditFinding struct {
	Severity  string `json:"severity"`
	ProductID string `json:"productId,omitempty"`
	Path      string `json:"path,omitempty"`
	Message   string `json:"message"`
}

type auditResult struct {
	Status            string         `json:"status"`
	Dir               string         `json:"dir"`
	FileCount         int            `json:"fileCount"`
	Findings          []auditFinding `json:"findings,omitempty"`
	SubscriptionCount int            `json:"subscriptionCount"`
}

type auditSubscriptionFile struct {
	Path           string
	Wrapped        bool
	RegionsVersion string
	Subscription   *androidpublisher.Subscription
	Raw            map[string]any
}

func newAuditCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var dir, output string
	var strict bool
	fs.StringVar(&dir, "dir", "", "Directory containing exported subscription JSON files")
	fs.BoolVar(&strict, "strict", false, "Fail on warnings and errors")
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown, yaml")

	return &ffcli.Command{
		Name:      "audit",
		ShortHelp: "Audit subscription sync files before mutation",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			var err error
			dir, err = shared.ResolveProjectPath(dir, func(cfg config.ProjectConfig) string { return cfg.SubscriptionsDir })
			if err != nil {
				return err
			}
			if strings.TrimSpace(dir) == "" {
				return shared.UsageErrorf("--dir is required")
			}

			result, err := runAudit(dir)
			if err != nil {
				return err
			}

			switch shared.ResolveOutput(output) {
			case "json":
				if err := shared.WriteJSON(deps.Stdout, result); err != nil {
					return err
				}
			case "yaml":
				if err := shared.WriteYAML(deps.Stdout, result); err != nil {
					return err
				}
			case "table":
				if err := writeAuditTable(deps.Stdout, result); err != nil {
					return err
				}
			case "markdown":
				if err := writeAuditMarkdown(deps.Stdout, result); err != nil {
					return err
				}
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(output))
			}

			if strict && result.Status != "ok" {
				return fmt.Errorf("subscriptions audit reported %s findings", result.Status)
			}
			return nil
		},
	}
}

func runAudit(dir string) (auditResult, error) {
	result := auditResult{
		Status: "ok",
		Dir:    strings.TrimSpace(dir),
	}

	entries, err := os.ReadDir(result.Dir)
	if err != nil {
		return auditResult{}, fmt.Errorf("read subscriptions directory: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		files = append(files, filepath.Join(result.Dir, entry.Name()))
	}
	sort.Strings(files)
	result.FileCount = len(files)
	if len(files) == 0 {
		result.Findings = append(result.Findings, auditFinding{
			Severity: "error",
			Path:     result.Dir,
			Message:  "no subscription JSON files found",
		})
		result.Status = "error"
		return result, nil
	}

	seenProductIDs := map[string]string{}
	for _, path := range files {
		file, findings := inspectSubscriptionFile(path)
		result.Findings = append(result.Findings, findings...)
		if file.Subscription == nil {
			continue
		}
		result.SubscriptionCount++
		productID := strings.TrimSpace(file.Subscription.ProductId)
		if productID == "" {
			continue
		}
		if previous, ok := seenProductIDs[productID]; ok {
			result.Findings = append(result.Findings, auditFinding{
				Severity:  "error",
				ProductID: productID,
				Path:      path,
				Message:   fmt.Sprintf("duplicate productId also declared in %s", previous),
			})
			continue
		}
		seenProductIDs[productID] = path
	}

	result.Status = auditStatus(result.Findings)
	return result, nil
}

func inspectSubscriptionFile(path string) (auditSubscriptionFile, []auditFinding) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return auditSubscriptionFile{}, []auditFinding{{
			Severity: "error",
			Path:     path,
			Message:  fmt.Sprintf("failed to read file: %v", err),
		}}
	}

	file := auditSubscriptionFile{Path: path}
	findings := []auditFinding{}

	_ = json.Unmarshal(raw, &file.Raw)

	var wrapped syncSubscriptionFile
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Subscription != nil {
		file.Wrapped = true
		file.RegionsVersion = strings.TrimSpace(wrapped.RegionsVersion)
		file.Subscription = wrapped.Subscription
	} else {
		var subscription androidpublisher.Subscription
		if err := json.Unmarshal(raw, &subscription); err != nil {
			return auditSubscriptionFile{}, []auditFinding{{
				Severity: "error",
				Path:     path,
				Message:  fmt.Sprintf("invalid subscription JSON payload: %v", err),
			}}
		}
		file.Subscription = &subscription
	}

	if file.Subscription == nil {
		return file, findings
	}
	productID := strings.TrimSpace(file.Subscription.ProductId)
	if productID == "" {
		findings = append(findings, auditFinding{
			Severity: "error",
			Path:     path,
			Message:  "subscription productId is required",
		})
	} else if filepath.Base(path) != productID+".json" {
		findings = append(findings, auditFinding{
			Severity:  "warning",
			ProductID: productID,
			Path:      path,
			Message:   "filename does not match <productId>.json",
		})
	}

	if len(file.Subscription.BasePlans) == 0 {
		findings = append(findings, auditFinding{
			Severity:  "error",
			ProductID: productID,
			Path:      path,
			Message:   "subscription has no base plans",
		})
	} else {
		seenBasePlans := map[string]struct{}{}
		for _, basePlan := range file.Subscription.BasePlans {
			basePlanID := strings.TrimSpace(basePlan.BasePlanId)
			if basePlanID == "" {
				findings = append(findings, auditFinding{
					Severity:  "error",
					ProductID: productID,
					Path:      path,
					Message:   "base plan is missing basePlanId",
				})
				continue
			}
			if _, ok := seenBasePlans[basePlanID]; ok {
				findings = append(findings, auditFinding{
					Severity:  "error",
					ProductID: productID,
					Path:      path,
					Message:   fmt.Sprintf("duplicate base plan id %q", basePlanID),
				})
				continue
			}
			seenBasePlans[basePlanID] = struct{}{}
		}
	}

	if len(file.Subscription.Listings) == 0 {
		findings = append(findings, auditFinding{
			Severity:  "error",
			ProductID: productID,
			Path:      path,
			Message:   "subscription has no listings",
		})
	}

	if !file.Wrapped || file.RegionsVersion == "" {
		findings = append(findings, auditFinding{
			Severity:  "warning",
			ProductID: productID,
			Path:      path,
			Message:   "regionsVersion wrapper is missing; sync will resolve the latest regions version automatically",
		})
	}

	if updateMask := subscriptionSyncUpdateMask(file.Subscription); updateMask == "" {
		findings = append(findings, auditFinding{
			Severity:  "warning",
			ProductID: productID,
			Path:      path,
			Message:   "effective update mask is empty; sync would not update any mutable fields",
		})
	}

	findings = append(findings, auditOfferFindings(productID, path, file.Raw)...)
	return file, findings
}

func auditOfferFindings(productID, path string, raw map[string]any) []auditFinding {
	findings := []auditFinding{}
	subscriptionNode := raw
	if wrapped, ok := raw["subscription"].(map[string]any); ok {
		subscriptionNode = wrapped
	}
	basePlans, ok := subscriptionNode["basePlans"].([]any)
	if !ok {
		return findings
	}

	for _, rawBasePlan := range basePlans {
		basePlan, ok := rawBasePlan.(map[string]any)
		if !ok {
			continue
		}
		basePlanID, _ := basePlan["basePlanId"].(string)
		offers, ok := basePlan["offers"].([]any)
		if !ok {
			continue
		}
		seenOfferIDs := map[string]struct{}{}
		for _, rawOffer := range offers {
			offer, ok := rawOffer.(map[string]any)
			if !ok {
				continue
			}
			offerID, _ := offer["offerId"].(string)
			offerID = strings.TrimSpace(offerID)
			if offerID == "" {
				continue
			}
			if _, exists := seenOfferIDs[offerID]; exists {
				findings = append(findings, auditFinding{
					Severity:  "error",
					ProductID: productID,
					Path:      path,
					Message:   fmt.Sprintf("duplicate offer id %q in base plan %q", offerID, basePlanID),
				})
				continue
			}
			seenOfferIDs[offerID] = struct{}{}
		}
	}

	return findings
}

func auditStatus(findings []auditFinding) string {
	status := "ok"
	for _, finding := range findings {
		switch finding.Severity {
		case "error":
			return "error"
		case "warning":
			status = "warn"
		}
	}
	return status
}

func writeAuditTable(out io.Writer, result auditResult) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "DIR\t%s\n", result.Dir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "FILES\t%d\n", result.FileCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "SUBSCRIPTIONS\t%d\n", result.SubscriptionCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "SEVERITY\tPRODUCT_ID\tMESSAGE"); err != nil {
		return err
	}
	for _, finding := range result.Findings {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\n", finding.Severity, finding.ProductID, finding.Message); err != nil {
			return err
		}
	}
	return nil
}

func writeAuditMarkdown(out io.Writer, result auditResult) error {
	summaryRows := [][]string{
		{"status", result.Status},
		{"dir", result.Dir},
		{"fileCount", strconv.Itoa(result.FileCount)},
		{"subscriptionCount", strconv.Itoa(result.SubscriptionCount)},
	}
	if err := shared.WriteMarkdownTable(out, []string{"field", "value"}, summaryRows); err != nil {
		return err
	}
	if len(result.Findings) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	rows := make([][]string, 0, len(result.Findings))
	for _, finding := range result.Findings {
		rows = append(rows, []string{finding.Severity, finding.ProductID, finding.Message})
	}
	return shared.WriteMarkdownTable(out, []string{"severity", "productId", "message"}, rows)
}
