package reviews

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
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
	var packageName, token, translationLanguage string
	var maxResults, startIndex int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&maxResults, "max-results", 0, "Maximum number of reviews per page")
	fs.Int64Var(&startIndex, "start-index", 0, "Index of first review to return (non-token pagination)")
	fs.StringVar(&token, "token", "", "Pagination token")
	fs.StringVar(&translationLanguage, "translation-language", "", "Language localization code for translated responses")

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

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"reviews":     result.Reviews,
				"nextToken":   result.NextToken,
			})
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
