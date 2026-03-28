package listing

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/validate"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type auditFinding struct {
	Severity string `json:"severity"`
	Locale   string `json:"locale,omitempty"`
	Field    string `json:"field,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type auditResult struct {
	Status                string         `json:"status"`
	Dir                   string         `json:"dir"`
	ScreenshotsDir        string         `json:"screenshotsDir,omitempty"`
	DefaultLocale         string         `json:"defaultLocale,omitempty"`
	LocaleCount           int            `json:"localeCount"`
	ScreenshotLocaleCount int            `json:"screenshotLocaleCount,omitempty"`
	Findings              []auditFinding `json:"findings,omitempty"`
}

func newAuditCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var dir, screenshotsDir, defaultLocale, output string
	var strict bool
	fs.StringVar(&dir, "dir", "", "Directory containing listing locale folders")
	fs.StringVar(&screenshotsDir, "screenshots-dir", "", "Directory containing screenshot locale folders")
	fs.StringVar(&defaultLocale, "default-locale", "", "Default locale to require in the listing set")
	fs.BoolVar(&strict, "strict", false, "Fail on warnings and errors")
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown, yaml")

	return &ffcli.Command{
		Name:      "audit",
		ShortHelp: "Audit listing files and screenshot coverage before sync",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			var err error
			dir, err = shared.ResolveProjectPath(dir, func(cfg config.ProjectConfig) string { return cfg.ListingDir })
			if err != nil {
				return err
			}
			if strings.TrimSpace(dir) == "" {
				return shared.UsageErrorf("--dir is required")
			}
			screenshotsDir, err = shared.ResolveProjectPath(screenshotsDir, func(cfg config.ProjectConfig) string { return cfg.ScreenshotsDir })
			if err != nil && strings.TrimSpace(screenshotsDir) == "" {
				screenshotsDir = ""
			}
			defaultLocale, err = shared.ResolveDefaultLocale(defaultLocale)
			if err != nil && strings.TrimSpace(defaultLocale) == "" {
				defaultLocale = ""
			}

			result, err := runListingAudit(dir, screenshotsDir, defaultLocale)
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
				return fmt.Errorf("listing audit reported %s findings", result.Status)
			}
			return nil
		},
	}
}

func runListingAudit(dir, screenshotsDir, defaultLocale string) (auditResult, error) {
	result := auditResult{
		Status:         "ok",
		Dir:            strings.TrimSpace(dir),
		ScreenshotsDir: strings.TrimSpace(screenshotsDir),
		DefaultLocale:  strings.TrimSpace(defaultLocale),
	}

	entries, err := os.ReadDir(result.Dir)
	if err != nil {
		return auditResult{}, fmt.Errorf("read listing directory: %w", err)
	}

	listingLocales := make([]string, 0, len(entries))
	listingLocaleSet := map[string]struct{}{}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		locale := entry.Name()
		listingLocales = append(listingLocales, locale)
		listingLocaleSet[locale] = struct{}{}
		findings := auditListingLocale(filepath.Join(result.Dir, locale), locale)
		result.Findings = append(result.Findings, findings...)
	}
	sort.Strings(listingLocales)
	result.LocaleCount = len(listingLocales)
	if result.LocaleCount == 0 {
		result.Findings = append(result.Findings, auditFinding{
			Severity: "error",
			Message:  "no locale directories found",
			Path:     result.Dir,
		})
	}

	if result.DefaultLocale == "" {
		result.Findings = append(result.Findings, auditFinding{
			Severity: "warning",
			Field:    "defaultLocale",
			Message:  "default locale is not configured",
		})
	} else if _, ok := listingLocaleSet[result.DefaultLocale]; !ok {
		result.Findings = append(result.Findings, auditFinding{
			Severity: "error",
			Locale:   result.DefaultLocale,
			Field:    "defaultLocale",
			Message:  "default locale is missing from listing files",
		})
	}

	screenshotLocales := []string{}
	if result.ScreenshotsDir != "" {
		locales, scanErr := scanLocaleDirs(result.ScreenshotsDir)
		if scanErr != nil {
			result.Findings = append(result.Findings, auditFinding{
				Severity: "warning",
				Field:    "screenshotsDir",
				Path:     result.ScreenshotsDir,
				Message:  fmt.Sprintf("screenshots directory unavailable: %v", scanErr),
			})
		} else {
			screenshotLocales = locales
		}
	}
	result.ScreenshotLocaleCount = len(screenshotLocales)
	screenshotLocaleSet := map[string]struct{}{}
	for _, locale := range screenshotLocales {
		screenshotLocaleSet[locale] = struct{}{}
	}

	if result.DefaultLocale != "" {
		if _, ok := screenshotLocaleSet[result.DefaultLocale]; !ok {
			result.Findings = append(result.Findings, auditFinding{
				Severity: "warning",
				Locale:   result.DefaultLocale,
				Field:    "screenshots",
				Message:  "default locale screenshot coverage is missing",
			})
		}
	}
	for _, locale := range listingLocales {
		if _, ok := screenshotLocaleSet[locale]; !ok {
			result.Findings = append(result.Findings, auditFinding{
				Severity: "warning",
				Locale:   locale,
				Field:    "screenshots",
				Message:  "listing locale exists without screenshots",
			})
		}
	}
	for _, locale := range screenshotLocales {
		if _, ok := listingLocaleSet[locale]; !ok {
			result.Findings = append(result.Findings, auditFinding{
				Severity: "warning",
				Locale:   locale,
				Field:    "listing",
				Message:  "screenshots locale exists without listing files",
			})
		}
	}

	result.Status = listingAuditStatus(result.Findings)
	return result, nil
}

func auditListingLocale(dir, locale string) []auditFinding {
	findings := []auditFinding{}

	title, titleMissing, titlePath := readAuditTextFile(filepath.Join(dir, "title.txt"))
	shortDescription, shortMissing, shortPath := readAuditTextFile(filepath.Join(dir, "short-description.txt"))
	fullDescription, fullMissing, fullPath := readAuditTextFile(filepath.Join(dir, "full-description.txt"))

	if titleMissing != "" {
		findings = append(findings, auditFinding{Severity: "error", Locale: locale, Field: "title", Path: titlePath, Message: titleMissing})
	} else {
		findings = append(findings, validateListingField(locale, "title", titlePath, title, validate.Title)...)
		if utf8.RuneCountInString(title) < 5 {
			findings = append(findings, auditFinding{Severity: "warning", Locale: locale, Field: "title", Path: titlePath, Message: "title looks suspiciously short"})
		}
	}

	if shortMissing != "" {
		findings = append(findings, auditFinding{Severity: "error", Locale: locale, Field: "shortDescription", Path: shortPath, Message: shortMissing})
	} else {
		findings = append(findings, validateListingField(locale, "shortDescription", shortPath, shortDescription, validate.ShortDescription)...)
		if utf8.RuneCountInString(shortDescription) < 15 {
			findings = append(findings, auditFinding{Severity: "warning", Locale: locale, Field: "shortDescription", Path: shortPath, Message: "short description looks suspiciously short"})
		}
	}

	if fullMissing != "" {
		findings = append(findings, auditFinding{Severity: "error", Locale: locale, Field: "fullDescription", Path: fullPath, Message: fullMissing})
	} else {
		findings = append(findings, validateListingField(locale, "fullDescription", fullPath, fullDescription, validate.FullDescription)...)
		if utf8.RuneCountInString(fullDescription) < 40 {
			findings = append(findings, auditFinding{Severity: "warning", Locale: locale, Field: "fullDescription", Path: fullPath, Message: "full description looks suspiciously short"})
		}
	}

	if strings.EqualFold(strings.TrimSpace(title), strings.TrimSpace(shortDescription)) && title != "" {
		findings = append(findings, auditFinding{
			Severity: "warning",
			Locale:   locale,
			Field:    "shortDescription",
			Path:     shortPath,
			Message:  "title is duplicated in short description",
		})
	}

	return findings
}

func readAuditTextFile(path string) (value string, message string, resolvedPath string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Sprintf("missing required file %s", filepath.Base(path)), path
		}
		return "", err.Error(), path
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", fmt.Sprintf("required file %s is empty", filepath.Base(path)), path
	}
	return text, "", path
}

func validateListingField(locale, field, path, value string, check func(string) error) []auditFinding {
	if err := check(value); err != nil {
		return []auditFinding{{
			Severity: "error",
			Locale:   locale,
			Field:    field,
			Path:     path,
			Message:  err.Error(),
		}}
	}
	return nil
}

func listingAuditStatus(findings []auditFinding) string {
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

func scanLocaleDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	locales := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			locales = append(locales, entry.Name())
		}
	}
	sort.Strings(locales)
	return locales, nil
}

func writeAuditTable(out io.Writer, result auditResult) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "DIR\t%s\n", result.Dir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "DEFAULT_LOCALE\t%s\n", result.DefaultLocale); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "LOCALES\t%d\n", result.LocaleCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "SCREENSHOT_LOCALES\t%d\n", result.ScreenshotLocaleCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "SEVERITY\tLOCALE\tFIELD\tMESSAGE"); err != nil {
		return err
	}
	for _, finding := range result.Findings {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", finding.Severity, finding.Locale, finding.Field, finding.Message); err != nil {
			return err
		}
	}
	return nil
}

func writeAuditMarkdown(out io.Writer, result auditResult) error {
	summaryRows := [][]string{
		{"status", result.Status},
		{"dir", result.Dir},
		{"defaultLocale", result.DefaultLocale},
		{"localeCount", strconv.Itoa(result.LocaleCount)},
		{"screenshotLocaleCount", strconv.Itoa(result.ScreenshotLocaleCount)},
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
		rows = append(rows, []string{finding.Severity, finding.Locale, finding.Field, finding.Message})
	}
	return shared.WriteMarkdownTable(out, []string{"severity", "locale", "field", "message"}, rows)
}
