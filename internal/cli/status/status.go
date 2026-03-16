package status

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

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

type reviewsSummary struct {
	Total              int            `json:"total"`
	AverageRating      float64        `json:"averageRating"`
	PendingReply       int            `json:"pendingReply"`
	RatingDistribution map[string]int `json:"ratingDistribution,omitempty"`
}

type statusResult struct {
	PackageName  string          `json:"packageName"`
	Status       string          `json:"status"`
	Tracks       []gpc.TrackInfo `json:"tracks,omitempty"`
	Reviews      *reviewsSummary `json:"reviews,omitempty"`
	Alerts       []string        `json:"alerts,omitempty"`
	TrackError   string          `json:"trackError,omitempty"`
	ReviewsError string          `json:"reviewsError,omitempty"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName, output string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown")

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

			client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
			})
			if err != nil {
				return err
			}
			defer cancel()

			result := runStatus(requestCtx, client, pkg)
			switch shared.ResolveOutput(output) {
			case "json":
				return shared.WriteJSON(deps.Stdout, result)
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

func runStatus(ctx context.Context, client Client, packageName string) statusResult {
	result := statusResult{
		PackageName: packageName,
		Status:      "ok",
	}

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

	reviews, err := client.ListReviews(ctx, packageName, reviewSnapshotLimit, 0, "", "", false)
	if err != nil {
		result.ReviewsError = fmt.Sprintf("failed to list reviews: %v", err)
	} else {
		result.Reviews = summarizeReviews(reviews.Reviews)
		if result.Reviews.PendingReply > 0 {
			result.Alerts = append(result.Alerts, fmt.Sprintf("%d unreplied reviews", result.Reviews.PendingReply))
		}
	}

	if result.TrackError != "" || result.ReviewsError != "" || len(result.Alerts) > 0 {
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
	} else {
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
	} else {
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
