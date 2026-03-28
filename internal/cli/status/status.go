package status

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

	"github.com/leszko11/google-play-console-cli/internal/cli/listing"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const reviewSnapshotLimit int64 = 100

type Client interface {
	VerifyPackageAccess(ctx context.Context, packageName string) error
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	ValidateEdit(ctx context.Context, packageName, editID string) error
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ListTracks(ctx context.Context, packageName, editID string) ([]gpc.TrackInfo, error)
	ListReviews(ctx context.Context, packageName string, maxResults, startIndex int64, token, translationLanguage string, paginate bool) (gpc.ReviewsListInfo, error)
	ListSubscriptions(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (gpc.SubscriptionsListInfo, error)
}

type Deps struct {
	LoadConfig  func() (config.Config, error)
	LoadProject func() (config.ProjectConfigInfo, error)
	NewClient   func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv   func(string) string
	Stdout      io.Writer
	Stderr      io.Writer
}

type reviewsSummary struct {
	Total              int            `json:"total"`
	AverageRating      float64        `json:"averageRating"`
	PendingReply       int            `json:"pendingReply"`
	RatingDistribution map[string]int `json:"ratingDistribution,omitempty"`
}

type listingSummary struct {
	DefaultLocale         string   `json:"defaultLocale,omitempty"`
	ListingDir            string   `json:"listingDir,omitempty"`
	ScreenshotsDir        string   `json:"screenshotsDir,omitempty"`
	ChangelogDir          string   `json:"changelogDir,omitempty"`
	ListingDirExists      bool     `json:"listingDirExists"`
	ScreenshotsDirExists  bool     `json:"screenshotsDirExists"`
	ChangelogDirExists    bool     `json:"changelogDirExists"`
	LocaleCount           int      `json:"localeCount,omitempty"`
	ScreenshotLocaleCount int      `json:"screenshotLocaleCount,omitempty"`
	Warnings              []string `json:"warnings,omitempty"`
}

type subscriptionsSummary struct {
	LocalDir       string   `json:"localDir,omitempty"`
	LocalDirExists bool     `json:"localDirExists"`
	LocalFileCount int      `json:"localFileCount,omitempty"`
	RemoteCount    int      `json:"remoteCount,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

type statusResult struct {
	PackageName        string                         `json:"packageName"`
	Status             string                         `json:"status"`
	Tracks             []gpc.TrackInfo                `json:"tracks,omitempty"`
	Reviews            *reviewsSummary                `json:"reviews,omitempty"`
	Auth               *shared.AuthStatusSnapshot     `json:"auth,omitempty"`
	Readiness          *shared.PackageReadinessResult `json:"readiness,omitempty"`
	Listing            *listingSummary                `json:"listing,omitempty"`
	Subscriptions      *subscriptionsSummary          `json:"subscriptions,omitempty"`
	Alerts             []string                       `json:"alerts,omitempty"`
	TrackError         string                         `json:"trackError,omitempty"`
	ReviewsError       string                         `json:"reviewsError,omitempty"`
	ReadinessError     string                         `json:"readinessError,omitempty"`
	ListingError       string                         `json:"listingError,omitempty"`
	SubscriptionsError string                         `json:"subscriptionsError,omitempty"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName, output, includesCSV string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown, yaml, minimal")
	fs.StringVar(&includesCSV, "include", "", "Comma-separated sections: tracks,reviews,auth,readiness,listing,subscriptions")

	return &ffcli.Command{
		Name:      "status",
		ShortHelp: "Summarize tracks and recent review health for an app",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			pkg, err := shared.ResolvePackageName(packageName)
			if err != nil {
				return err
			}
			includes, err := parseIncludes(includesCSV)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
			})
			if err != nil {
				return err
			}
			defer cancel()

			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			projectInfo := config.ProjectConfigInfo{}
			var projectErr error
			if includeEnabled(includes, "listing") || includeEnabled(includes, "subscriptions") {
				projectInfo, projectErr = deps.LoadProject()
			}

			result := runStatus(requestCtx, client, cfg, projectInfo, projectErr, deps.LookupEnv, pkg, includes)
			switch shared.ResolveOutput(output) {
			case "json":
				return shared.WriteJSON(deps.Stdout, result)
			case "yaml":
				return shared.WriteYAML(deps.Stdout, result)
			case "table":
				return writeTable(deps.Stdout, result)
			case "markdown":
				return writeMarkdown(deps.Stdout, result)
			case "minimal":
				return shared.WriteMinimal(deps.Stdout, []string{result.Status})
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(output))
			}
		},
	}
}

func runStatus(ctx context.Context, client Client, cfg config.Config, projectInfo config.ProjectConfigInfo, projectErr error, lookupEnv func(string) string, packageName string, includes []string) statusResult {
	result := statusResult{
		PackageName: packageName,
		Status:      "ok",
	}

	if includeEnabled(includes, "tracks") {
		edit, err := client.CreateEdit(ctx, packageName)
		if err != nil {
			result.TrackError = fmt.Sprintf("failed to create temporary edit: %v", err)
		} else {
			tracks, listErr := client.ListTracks(ctx, packageName, edit.ID)
			if listErr != nil {
				result.TrackError = fmt.Sprintf("failed to list tracks: %v", listErr)
			} else {
				result.Tracks = normalizeTracks(tracks)
			}
			if deleteErr := client.DeleteEdit(ctx, packageName, edit.ID); deleteErr != nil {
				result.Alerts = append(result.Alerts, fmt.Sprintf("track cleanup failed: %v", deleteErr))
			}
		}
	}

	if includeEnabled(includes, "reviews") {
		reviews, err := client.ListReviews(ctx, packageName, reviewSnapshotLimit, 0, "", "", false)
		if err != nil {
			result.ReviewsError = fmt.Sprintf("failed to list reviews: %v", err)
		} else {
			result.Reviews = summarizeReviews(reviews.Reviews)
			if result.Reviews.PendingReply > 0 {
				result.Alerts = append(result.Alerts, fmt.Sprintf("%d unreplied reviews", result.Reviews.PendingReply))
			}
		}
	}

	if includeEnabled(includes, "auth") {
		auth := shared.BuildAuthStatusSnapshot(cfg, lookupEnv)
		result.Auth = &auth
		if !auth.Authenticated || auth.Health != string(shared.AuthHealthReady) {
			result.Alerts = append(result.Alerts, "auth needs attention")
		}
	}

	if includeEnabled(includes, "readiness") {
		readiness, err := shared.DetectPackageReadiness(ctx, client, packageName)
		if err != nil {
			result.ReadinessError = err.Error()
		} else {
			result.Readiness = &readiness
			if readiness.Status != shared.PackageReadinessReady {
				result.Alerts = append(result.Alerts, fmt.Sprintf("package readiness: %s", readiness.Status))
			}
		}
	}

	if includeEnabled(includes, "listing") {
		summary := buildListingSummary(projectInfo, projectErr)
		result.Listing = &summary
		if projectErr != nil {
			result.ListingError = projectErr.Error()
		}
		if len(summary.Warnings) > 0 {
			result.Alerts = append(result.Alerts, "listing workspace needs attention")
		}
	}

	if includeEnabled(includes, "subscriptions") {
		summary, err := buildSubscriptionsSummary(ctx, client, packageName, projectInfo, projectErr)
		if err != nil {
			result.SubscriptionsError = err.Error()
		} else {
			result.Subscriptions = &summary
			if len(summary.Warnings) > 0 {
				result.Alerts = append(result.Alerts, "subscriptions workspace needs attention")
			}
		}
	}

	if result.TrackError != "" || result.ReviewsError != "" || result.ReadinessError != "" || result.ListingError != "" || result.SubscriptionsError != "" || len(result.Alerts) > 0 {
		result.Status = "warn"
	}
	return result
}

func summarizeReviews(reviews []gpc.ReviewInfo) *reviewsSummary {
	summary := &reviewsSummary{
		Total:              len(reviews),
		RatingDistribution: map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0},
	}
	if len(reviews) == 0 {
		return summary
	}

	var totalRating int64
	for _, review := range reviews {
		rating := review.StarRating
		if rating >= 1 && rating <= 5 {
			summary.RatingDistribution[strconv.FormatInt(rating, 10)]++
			totalRating += rating
		}
		if !review.HasReply {
			summary.PendingReply++
		}
	}
	summary.AverageRating = float64(totalRating) / float64(len(reviews))
	return summary
}

func normalizeTracks(tracks []gpc.TrackInfo) []gpc.TrackInfo {
	out := append([]gpc.TrackInfo(nil), tracks...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func parseIncludes(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{"tracks", "reviews"}, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		switch part {
		case "tracks", "reviews", "auth", "readiness", "listing", "subscriptions":
		default:
			return nil, shared.UsageErrorf("unsupported include %q", part)
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil, shared.UsageErrorf("--include must name at least one section")
	}
	return out, nil
}

func includeEnabled(includes []string, name string) bool {
	for _, include := range includes {
		if include == name {
			return true
		}
	}
	return false
}

func buildListingSummary(projectInfo config.ProjectConfigInfo, projectErr error) listingSummary {
	summary := listingSummary{
		DefaultLocale:  projectInfo.Config.DefaultLocale,
		ListingDir:     projectInfo.Config.ListingDir,
		ScreenshotsDir: projectInfo.Config.ScreenshotsDir,
		ChangelogDir:   projectInfo.Config.ChangelogDir,
	}
	if projectErr != nil {
		summary.Warnings = append(summary.Warnings, fmt.Sprintf("project config unavailable: %v", projectErr))
		return summary
	}
	if summary.DefaultLocale == "" {
		summary.Warnings = append(summary.Warnings, "default locale is not configured in .gpc.yaml")
	}
	if summary.ListingDir == "" {
		summary.Warnings = append(summary.Warnings, "listing dir is not configured")
	} else if info, err := os.Stat(summary.ListingDir); err == nil && info.IsDir() {
		summary.ListingDirExists = true
		if locales, err := listing.ScanListingsDir(summary.ListingDir); err == nil {
			summary.LocaleCount = len(locales)
		}
	} else {
		summary.Warnings = append(summary.Warnings, "listing dir is missing")
	}
	if summary.ScreenshotsDir == "" {
		summary.Warnings = append(summary.Warnings, "screenshots dir is not configured")
	} else if locales, err := scanLocaleDirs(summary.ScreenshotsDir); err == nil {
		summary.ScreenshotsDirExists = true
		summary.ScreenshotLocaleCount = len(locales)
	} else {
		summary.Warnings = append(summary.Warnings, "screenshots dir is missing")
	}
	if summary.ChangelogDir == "" {
		summary.Warnings = append(summary.Warnings, "changelog dir is not configured")
	} else if info, err := os.Stat(summary.ChangelogDir); err == nil && info.IsDir() {
		summary.ChangelogDirExists = true
	} else {
		summary.Warnings = append(summary.Warnings, "changelog dir is missing")
	}
	return summary
}

func buildSubscriptionsSummary(ctx context.Context, client Client, packageName string, projectInfo config.ProjectConfigInfo, projectErr error) (subscriptionsSummary, error) {
	summary := subscriptionsSummary{
		LocalDir: projectInfo.Config.SubscriptionsDir,
	}
	if projectErr != nil {
		summary.Warnings = append(summary.Warnings, fmt.Sprintf("project config unavailable: %v", projectErr))
	} else if summary.LocalDir == "" {
		summary.Warnings = append(summary.Warnings, "subscriptions dir is not configured")
	} else if info, err := os.Stat(summary.LocalDir); err == nil && info.IsDir() {
		summary.LocalDirExists = true
		count, countErr := countJSONFiles(summary.LocalDir)
		if countErr == nil {
			summary.LocalFileCount = count
			if count == 0 {
				summary.Warnings = append(summary.Warnings, "subscriptions dir has no JSON files")
			}
		}
	} else {
		summary.Warnings = append(summary.Warnings, "subscriptions dir is missing")
	}

	result, err := client.ListSubscriptions(ctx, packageName, 0, "", true)
	if err != nil {
		return subscriptionsSummary{}, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	summary.RemoteCount = len(result.Subscriptions)
	return summary, nil
}

func scanLocaleDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			out = append(out, entry.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func countJSONFiles(root string) (int, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}
	return count, nil
}

func writeTable(out io.Writer, result statusResult) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", result.PackageName); err != nil {
		return err
	}

	if result.TrackError != "" {
		if _, err := fmt.Fprintf(out, "TRACK_ERROR\t%s\n", result.TrackError); err != nil {
			return err
		}
	} else if result.Tracks != nil {
		if _, err := fmt.Fprintln(out, "TRACK\tRELEASE_STATUS\tUSER_FRACTION\tVERSION_CODES"); err != nil {
			return err
		}
		for _, track := range result.Tracks {
			if len(track.Releases) == 0 {
				if _, err := fmt.Fprintf(out, "%s\t-\t-\t-\n", track.Name); err != nil {
					return err
				}
				continue
			}
			for _, release := range track.Releases {
				if _, err := fmt.Fprintf(out, "%s\t%s\t%.3f\t%s\n", track.Name, release.Status, release.UserFraction, joinVersionCodes(release.VersionCodes)); err != nil {
					return err
				}
			}
		}
	}

	if result.ReviewsError != "" {
		if _, err := fmt.Fprintf(out, "REVIEWS_ERROR\t%s\n", result.ReviewsError); err != nil {
			return err
		}
	} else if result.Reviews != nil {
		if _, err := fmt.Fprintf(out, "REVIEW_TOTAL\t%d\n", result.Reviews.Total); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "REVIEW_AVERAGE\t%.3f\n", result.Reviews.AverageRating); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "REVIEW_PENDING_REPLY\t%d\n", result.Reviews.PendingReply); err != nil {
			return err
		}
	}

	for _, alert := range result.Alerts {
		if _, err := fmt.Fprintf(out, "ALERT\t%s\n", alert); err != nil {
			return err
		}
	}
	if result.Auth != nil {
		if _, err := fmt.Fprintf(out, "AUTH\t%s\t%s\t%t\n", result.Auth.Health, result.Auth.Source, result.Auth.Authenticated); err != nil {
			return err
		}
	}
	if result.Readiness != nil {
		if _, err := fmt.Fprintf(out, "READINESS\t%s\t%s\n", result.Readiness.Status, result.Readiness.Detail); err != nil {
			return err
		}
	}
	if result.ReadinessError != "" {
		if _, err := fmt.Fprintf(out, "READINESS_ERROR\t%s\n", result.ReadinessError); err != nil {
			return err
		}
	}
	if result.Listing != nil {
		if _, err := fmt.Fprintf(out, "LISTING\t%s\tlocales=%d\tscreenshotLocales=%d\n", result.Listing.DefaultLocale, result.Listing.LocaleCount, result.Listing.ScreenshotLocaleCount); err != nil {
			return err
		}
	}
	if result.ListingError != "" {
		if _, err := fmt.Fprintf(out, "LISTING_ERROR\t%s\n", result.ListingError); err != nil {
			return err
		}
	}
	if result.Listing != nil {
		for _, warning := range result.Listing.Warnings {
			if _, err := fmt.Fprintf(out, "LISTING_WARNING\t%s\n", warning); err != nil {
				return err
			}
		}
	}
	if result.Subscriptions != nil {
		if _, err := fmt.Fprintf(out, "SUBSCRIPTIONS\tremote=%d\tlocal=%d\n", result.Subscriptions.RemoteCount, result.Subscriptions.LocalFileCount); err != nil {
			return err
		}
		for _, warning := range result.Subscriptions.Warnings {
			if _, err := fmt.Fprintf(out, "SUBSCRIPTIONS_WARNING\t%s\n", warning); err != nil {
				return err
			}
		}
	}
	if result.SubscriptionsError != "" {
		if _, err := fmt.Fprintf(out, "SUBSCRIPTIONS_ERROR\t%s\n", result.SubscriptionsError); err != nil {
			return err
		}
	}
	return nil
}

func writeMarkdown(out io.Writer, result statusResult) error {
	summaryRows := [][]string{
		{"status", result.Status},
		{"package", result.PackageName},
	}
	if err := shared.WriteMarkdownTable(out, []string{"field", "value"}, summaryRows); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	if result.TrackError != "" {
		if err := shared.WriteMarkdownTable(out, []string{"trackError"}, [][]string{{result.TrackError}}); err != nil {
			return err
		}
	} else if result.Tracks != nil {
		trackRows := make([][]string, 0, len(result.Tracks))
		for _, track := range result.Tracks {
			if len(track.Releases) == 0 {
				trackRows = append(trackRows, []string{track.Name, "-", "-", "-"})
				continue
			}
			for _, release := range track.Releases {
				trackRows = append(trackRows, []string{
					track.Name,
					release.Status,
					fmt.Sprintf("%.3f", release.UserFraction),
					joinVersionCodes(release.VersionCodes),
				})
			}
		}
		if err := shared.WriteMarkdownTable(out, []string{"track", "releaseStatus", "userFraction", "versionCodes"}, trackRows); err != nil {
			return err
		}
	}

	if result.ReviewsError != "" {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		if err := shared.WriteMarkdownTable(out, []string{"reviewsError"}, [][]string{{result.ReviewsError}}); err != nil {
			return err
		}
	} else if result.Reviews != nil {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		reviewRows := [][]string{
			{"total", strconv.Itoa(result.Reviews.Total)},
			{"average", fmt.Sprintf("%.3f", result.Reviews.AverageRating)},
			{"pendingReply", strconv.Itoa(result.Reviews.PendingReply)},
		}
		if err := shared.WriteMarkdownTable(out, []string{"field", "value"}, reviewRows); err != nil {
			return err
		}
	}

	if len(result.Alerts) > 0 {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		alertRows := make([][]string, 0, len(result.Alerts))
		for _, alert := range result.Alerts {
			alertRows = append(alertRows, []string{alert})
		}
		if err := shared.WriteMarkdownTable(out, []string{"alert"}, alertRows); err != nil {
			return err
		}
	}
	if result.Auth != nil {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		authRows := [][]string{
			{"health", result.Auth.Health},
			{"source", result.Auth.Source},
			{"authenticated", fmt.Sprintf("%t", result.Auth.Authenticated)},
		}
		if err := shared.WriteMarkdownTable(out, []string{"auth", "value"}, authRows); err != nil {
			return err
		}
	}
	if result.Readiness != nil || result.ReadinessError != "" {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		if result.ReadinessError != "" {
			if err := shared.WriteMarkdownTable(out, []string{"readinessError"}, [][]string{{result.ReadinessError}}); err != nil {
				return err
			}
		} else {
			if err := shared.WriteMarkdownTable(out, []string{"field", "value"}, [][]string{{"status", string(result.Readiness.Status)}, {"detail", result.Readiness.Detail}}); err != nil {
				return err
			}
		}
	}
	if result.Listing != nil {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		listingRows := [][]string{
			{"defaultLocale", result.Listing.DefaultLocale},
			{"localeCount", strconv.Itoa(result.Listing.LocaleCount)},
			{"screenshotLocaleCount", strconv.Itoa(result.Listing.ScreenshotLocaleCount)},
		}
		if err := shared.WriteMarkdownTable(out, []string{"listing", "value"}, listingRows); err != nil {
			return err
		}
		if len(result.Listing.Warnings) > 0 {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			rows := make([][]string, 0, len(result.Listing.Warnings))
			for _, warning := range result.Listing.Warnings {
				rows = append(rows, []string{warning})
			}
			if err := shared.WriteMarkdownTable(out, []string{"listingWarning"}, rows); err != nil {
				return err
			}
		}
	}
	if result.ListingError != "" {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		if err := shared.WriteMarkdownTable(out, []string{"listingError"}, [][]string{{result.ListingError}}); err != nil {
			return err
		}
	}
	if result.Subscriptions != nil || result.SubscriptionsError != "" {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		if result.SubscriptionsError != "" {
			if err := shared.WriteMarkdownTable(out, []string{"subscriptionsError"}, [][]string{{result.SubscriptionsError}}); err != nil {
				return err
			}
		} else {
			rows := [][]string{
				{"remoteCount", strconv.Itoa(result.Subscriptions.RemoteCount)},
				{"localFileCount", strconv.Itoa(result.Subscriptions.LocalFileCount)},
			}
			if err := shared.WriteMarkdownTable(out, []string{"subscriptions", "value"}, rows); err != nil {
				return err
			}
			if len(result.Subscriptions.Warnings) > 0 {
				if _, err := fmt.Fprintln(out); err != nil {
					return err
				}
				warningRows := make([][]string, 0, len(result.Subscriptions.Warnings))
				for _, warning := range result.Subscriptions.Warnings {
					warningRows = append(warningRows, []string{warning})
				}
				if err := shared.WriteMarkdownTable(out, []string{"subscriptionsWarning"}, warningRows); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func joinVersionCodes(versionCodes []int64) string {
	if len(versionCodes) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(versionCodes))
	for _, code := range versionCodes {
		parts = append(parts, strconv.FormatInt(code, 10))
	}
	return strings.Join(parts, ",")
}

func withDefaults(deps Deps) Deps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.LoadProject == nil {
		deps.LoadProject = config.LoadProject
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
