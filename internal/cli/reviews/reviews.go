package reviews

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	ListReviews(ctx context.Context, packageName string, maxResults, startIndex int64, token, translationLanguage string, paginate bool) (gpc.ReviewsListInfo, error)
	GetReview(ctx context.Context, packageName, reviewID string) (gpc.ReviewInfo, error)
	ReplyReview(ctx context.Context, packageName, reviewID, replyText string) (gpc.ReviewReplyInfo, error)
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
		Name:      "reviews",
		ShortHelp: "Read and reply to Play Store reviews",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newGetCommand(deps),
			newTriageCommand(deps),
			newReplyCommand(deps),
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
	var packageName, token, translationLanguage, output string
	var maxResults, startIndex int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&maxResults, "max-results", 0, "Maximum number of reviews per page")
	fs.Int64Var(&startIndex, "start-index", 0, "Index of first review to return (non-token pagination)")
	fs.StringVar(&token, "token", "", "Pagination token")
	fs.StringVar(&translationLanguage, "translation-language", "", "Language localization code for translated responses")
	fs.StringVar(&output, "output", "", "Output format: json, minimal")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List app reviews",
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
			if startIndex < 0 {
				return fmt.Errorf("--start-index must be greater than or equal to zero")
			}

			result, err := client.ListReviews(
				requestCtx,
				pkg,
				maxResults,
				startIndex,
				token,
				translationLanguage,
				shared.ActiveGlobalFlags().Paginate,
			)
			if err != nil {
				return fmt.Errorf("failed to list reviews: %w", err)
			}

			switch shared.ResolveOutput(output) {
			case "json":
				return shared.WriteJSON(deps.Stdout, map[string]any{
					"packageName": pkg,
					"reviews":     result.Reviews,
					"nextToken":   result.NextToken,
				})
			case "minimal":
				values := make([]string, 0, len(result.Reviews))
				for _, r := range result.Reviews {
					values = append(values, r.ReviewID)
				}
				return shared.WriteMinimal(deps.Stdout, values)
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(output))
			}
		},
	}
}

func newGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, reviewID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&reviewID, "review-id", "", "Review ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get one review by ID",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			reviewID = strings.TrimSpace(reviewID)
			if reviewID == "" {
				return fmt.Errorf("--review-id is required")
			}
			review, err := client.GetReview(requestCtx, pkg, reviewID)
			if err != nil {
				return fmt.Errorf("failed to get review: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"review":      review,
			})
		},
	}
}

type triageEntry struct {
	ReviewID   string `json:"reviewId,omitempty"`
	AuthorName string `json:"authorName,omitempty"`
	StarRating int64  `json:"starRating,omitempty"`
	HasReply   bool   `json:"hasReply,omitempty"`
}

type triageResult struct {
	PackageName       string        `json:"packageName"`
	Status            string        `json:"status"`
	ReviewCount       int           `json:"reviewCount"`
	PendingReplyCount int           `json:"pendingReplyCount"`
	RepliedCount      int           `json:"repliedCount"`
	PendingReply      []triageEntry `json:"pendingReply,omitempty"`
	Replied           []triageEntry `json:"replied,omitempty"`
}

func newTriageCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("triage", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, token, translationLanguage string
	var maxResults, startIndex int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&maxResults, "max-results", 100, "Maximum number of reviews per page")
	fs.Int64Var(&startIndex, "start-index", 0, "Index of first review to return (non-token pagination)")
	fs.StringVar(&token, "token", "", "Pagination token")
	fs.StringVar(&translationLanguage, "translation-language", "", "Language localization code for translated responses")

	return &ffcli.Command{
		Name:      "triage",
		ShortHelp: "Group reviews into pending-reply and replied buckets",
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
			if startIndex < 0 {
				return fmt.Errorf("--start-index must be greater than or equal to zero")
			}

			result, err := client.ListReviews(
				requestCtx,
				pkg,
				maxResults,
				startIndex,
				token,
				translationLanguage,
				shared.ActiveGlobalFlags().Paginate,
			)
			if err != nil {
				return fmt.Errorf("failed to list reviews: %w", err)
			}

			triage := buildTriageResult(pkg, result.Reviews)
			return shared.WriteJSON(deps.Stdout, triage)
		},
	}
}

func buildTriageResult(packageName string, reviews []gpc.ReviewInfo) triageResult {
	result := triageResult{
		PackageName: packageName,
		Status:      "ok",
		ReviewCount: len(reviews),
	}

	for _, review := range reviews {
		entry := triageEntry{
			ReviewID:   review.ReviewID,
			AuthorName: review.AuthorName,
			StarRating: review.StarRating,
			HasReply:   review.HasReply,
		}
		if review.HasReply {
			result.Replied = append(result.Replied, entry)
			continue
		}
		result.PendingReply = append(result.PendingReply, entry)
	}

	slices.SortFunc(result.PendingReply, func(a, b triageEntry) int {
		if a.StarRating != b.StarRating {
			if a.StarRating < b.StarRating {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ReviewID, b.ReviewID)
	})
	slices.SortFunc(result.Replied, func(a, b triageEntry) int {
		if a.StarRating != b.StarRating {
			if a.StarRating < b.StarRating {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ReviewID, b.ReviewID)
	})

	result.PendingReplyCount = len(result.PendingReply)
	result.RepliedCount = len(result.Replied)
	if result.PendingReplyCount > 0 {
		result.Status = "warn"
	}
	return result
}

func newReplyCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("reply", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, reviewID, replyText string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&reviewID, "review-id", "", "Review ID")
	fs.StringVar(&replyText, "reply-text", "", "Reply text")

	return &ffcli.Command{
		Name:      "reply",
		ShortHelp: "Reply to a review",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()
			reviewID = strings.TrimSpace(reviewID)
			if reviewID == "" {
				return fmt.Errorf("--review-id is required")
			}
			replyText = strings.TrimSpace(replyText)
			if replyText == "" {
				return fmt.Errorf("--reply-text is required")
			}
			reply, err := client.ReplyReview(requestCtx, pkg, reviewID, replyText)
			if err != nil {
				return fmt.Errorf("failed to reply to review: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"reviewId":    reviewID,
				"reply":       reply,
				"status":      "replied",
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
