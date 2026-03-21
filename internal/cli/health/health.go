package health

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const reviewSnapshotLimit int64 = 100

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ListTracks(ctx context.Context, packageName, editID string) ([]gpc.TrackInfo, error)
	ListReviews(ctx context.Context, packageName string, maxResults, startIndex int64, token, translationLanguage string, paginate bool) (gpc.ReviewsListInfo, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
}

type healthResult struct {
	PackageName    string        `json:"packageName"`
	Score          int           `json:"score"`
	Grade          string        `json:"grade"`
	ReviewRating   float64       `json:"reviewRating"`
	ReviewTotal    int           `json:"reviewTotal"`
	PendingReplies int           `json:"pendingReplies"`
	TrackCount     int           `json:"trackCount"`
	ActiveRollouts int           `json:"activeRollouts"`
	HaltedTracks   int           `json:"haltedTracks"`
	Checks         []healthCheck `json:"checks"`
}

type healthCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Impact int    `json:"impact"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName, output string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown, yaml")

	return &ffcli.Command{
		Name:      "health",
		ShortHelp: "Show composite health score for an app",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			pkg, err := shared.ResolvePackageName(packageName)
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

			result, err := runHealth(requestCtx, client, pkg)
			if err != nil {
				return fmt.Errorf("failed to collect health data: %w", err)
			}
			switch shared.ResolveOutput(output) {
			case "json":
				return shared.WriteJSON(deps.Stdout, result)
			case "yaml":
				return shared.WriteYAML(deps.Stdout, result)
			case "table":
				return writeTable(deps.Stdout, result)
			case "markdown":
				return writeMarkdown(deps.Stdout, result)
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(output))
			}
		},
	}
}

func runHealth(ctx context.Context, client Client, packageName string) (healthResult, error) {
	result := healthResult{
		PackageName: packageName,
		Score:       100,
	}

	var checks []healthCheck

	// Fetch tracks via temporary edit.
	edit, err := client.CreateEdit(ctx, packageName)
	if err != nil {
		return healthResult{}, fmt.Errorf("create edit: %w", err)
	}

	tracks, err := client.ListTracks(ctx, packageName, edit.ID)
	if err != nil {
		_ = client.DeleteEdit(ctx, packageName, edit.ID)
		return healthResult{}, fmt.Errorf("list tracks: %w", err)
	}
	result.TrackCount = len(tracks)
	for _, track := range tracks {
		for _, release := range track.Releases {
			if release.Status == "inProgress" {
				result.ActiveRollouts++
			}
			if release.Status == "halted" {
				result.HaltedTracks++
			}
		}
	}
	_ = client.DeleteEdit(ctx, packageName, edit.ID)

	// Halted tracks check.
	if result.HaltedTracks > 0 {
		impact := 10 * result.HaltedTracks
		if impact > 30 {
			impact = 30
		}
		checks = append(checks, healthCheck{
			Name:   "halted_tracks",
			Status: "fail",
			Detail: fmt.Sprintf("%d track(s) halted", result.HaltedTracks),
			Impact: impact,
		})
		result.Score -= impact
	} else {
		checks = append(checks, healthCheck{
			Name:   "halted_tracks",
			Status: "pass",
		})
	}

	// Fetch reviews.
	reviews, err := client.ListReviews(ctx, packageName, reviewSnapshotLimit, 0, "", "", false)
	if err != nil {
		return healthResult{}, fmt.Errorf("list reviews: %w", err)
	}
	result.ReviewTotal = len(reviews.Reviews)
	var totalRating int64
	for _, review := range reviews.Reviews {
		if review.StarRating >= 1 && review.StarRating <= 5 {
			totalRating += review.StarRating
		}
		if !review.HasReply {
			result.PendingReplies++
		}
	}
	if result.ReviewTotal > 0 {
		result.ReviewRating = float64(totalRating) / float64(result.ReviewTotal)
	}

	// Rating check.
	if result.ReviewRating < 3.0 && result.ReviewTotal > 0 {
		checks = append(checks, healthCheck{
			Name:   "review_rating",
			Status: "fail",
			Detail: fmt.Sprintf("%.1f average (below 3.0)", result.ReviewRating),
			Impact: 20,
		})
		result.Score -= 20
	} else if result.ReviewRating < 4.0 && result.ReviewTotal > 0 {
		checks = append(checks, healthCheck{
			Name:   "review_rating",
			Status: "warn",
			Detail: fmt.Sprintf("%.1f average (below 4.0)", result.ReviewRating),
			Impact: 10,
		})
		result.Score -= 10
	} else {
		checks = append(checks, healthCheck{
			Name:   "review_rating",
			Status: "pass",
			Detail: fmt.Sprintf("%.1f average", result.ReviewRating),
		})
	}

	// Pending replies check.
	if result.PendingReplies > 10 {
		checks = append(checks, healthCheck{
			Name:   "pending_replies",
			Status: "warn",
			Detail: fmt.Sprintf("%d unreplied reviews", result.PendingReplies),
			Impact: 15,
		})
		result.Score -= 15
	} else {
		checks = append(checks, healthCheck{
			Name:   "pending_replies",
			Status: "pass",
			Detail: fmt.Sprintf("%d unreplied", result.PendingReplies),
		})
	}

	if result.Score < 0 {
		result.Score = 0
	}

	// Determine grade.
	switch {
	case result.Score >= 90:
		result.Grade = "healthy"
	case result.Score >= 70:
		result.Grade = "fair"
	case result.Score >= 50:
		result.Grade = "degraded"
	default:
		result.Grade = "critical"
	}

	result.Checks = checks
	return result, nil
}

func writeTable(out io.Writer, result healthResult) error {
	if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", result.PackageName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "SCORE\t%d\n", result.Score); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "GRADE\t%s\n", result.Grade); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "REVIEW_RATING\t%.1f\n", result.ReviewRating); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PENDING_REPLIES\t%d\n", result.PendingReplies); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "ACTIVE_ROLLOUTS\t%d\n", result.ActiveRollouts); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "HALTED_TRACKS\t%d\n", result.HaltedTracks); err != nil {
		return err
	}
	if len(result.Checks) > 0 {
		if _, err := fmt.Fprintln(out, "CHECK\tSTATUS\tIMPACT\tDETAIL"); err != nil {
			return err
		}
		for _, check := range result.Checks {
			if _, err := fmt.Fprintf(out, "%s\t%s\t%d\t%s\n", check.Name, check.Status, check.Impact, check.Detail); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeMarkdown(out io.Writer, result healthResult) error {
	summaryRows := [][]string{
		{"package", result.PackageName},
		{"score", strconv.Itoa(result.Score)},
		{"grade", result.Grade},
		{"reviewRating", fmt.Sprintf("%.1f", result.ReviewRating)},
		{"pendingReplies", strconv.Itoa(result.PendingReplies)},
		{"activeRollouts", strconv.Itoa(result.ActiveRollouts)},
		{"haltedTracks", strconv.Itoa(result.HaltedTracks)},
	}
	if err := shared.WriteMarkdownTable(out, []string{"field", "value"}, summaryRows); err != nil {
		return err
	}

	if len(result.Checks) > 0 {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		checkRows := make([][]string, 0, len(result.Checks))
		for _, check := range result.Checks {
			checkRows = append(checkRows, []string{check.Name, check.Status, strconv.Itoa(check.Impact), check.Detail})
		}
		if err := shared.WriteMarkdownTable(out, []string{"check", "status", "impact", "detail"}, checkRows); err != nil {
			return err
		}
	}
	return nil
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
