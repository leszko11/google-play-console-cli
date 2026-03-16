package games

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	ListAchievementDefinitions(ctx context.Context, language string, pageSize int64, pageToken string) (gpc.GamesAchievementsListInfo, error)
	ListEventDefinitions(ctx context.Context, language string, pageSize int64, pageToken string) (gpc.GamesEventsListInfo, error)
	ListLeaderboards(ctx context.Context, language string, pageSize int64, pageToken string) (gpc.GamesLeaderboardsListInfo, error)
	GetLeaderboard(ctx context.Context, leaderboardID, language string) (gpc.GamesLeaderboardInfo, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
}

type listOptions struct {
	Language  string
	PageSize  int64
	PageToken string
	Output    string
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "games",
		ShortHelp: "Inspect Play Games Services achievements, events, and leaderboards",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newAchievementsCommand(deps),
			newEventsCommand(deps),
			newLeaderboardsCommand(deps),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewClient == nil {
		deps.NewClient = func(ctx context.Context, creds gpc.CredentialInput) (Client, error) {
			return gpc.NewGamesClient(ctx, creds)
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

func newAchievementsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "achievements",
		ShortHelp: "List Play Games achievement definitions",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newAchievementsListCommand(deps),
		},
	}
}

func newAchievementsListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var opts listOptions
	bindListFlags(fs, &opts)

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List Play Games achievement definitions",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateListOptions(opts)
			if err != nil {
				return err
			}
			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			result, err := client.ListAchievementDefinitions(requestCtx, opts.Language, opts.PageSize, opts.PageToken)
			if err != nil {
				return fmt.Errorf("failed to list achievement definitions: %w", err)
			}
			return writeAchievementsOutput(deps.Stdout, opts.Output, result)
		},
	}
}

func newEventsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "events",
		ShortHelp: "List Play Games event definitions",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newEventsListCommand(deps),
		},
	}
}

func newEventsListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var opts listOptions
	bindListFlags(fs, &opts)

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List Play Games event definitions",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateListOptions(opts)
			if err != nil {
				return err
			}
			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			result, err := client.ListEventDefinitions(requestCtx, opts.Language, opts.PageSize, opts.PageToken)
			if err != nil {
				return fmt.Errorf("failed to list event definitions: %w", err)
			}
			return writeEventsOutput(deps.Stdout, opts.Output, result)
		},
	}
}

func newLeaderboardsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "leaderboards",
		ShortHelp: "List and inspect Play Games leaderboards",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newLeaderboardsListCommand(deps),
			newLeaderboardsGetCommand(deps),
		},
	}
}

func newLeaderboardsListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var opts listOptions
	bindListFlags(fs, &opts)

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List Play Games leaderboards",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateListOptions(opts)
			if err != nil {
				return err
			}
			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			result, err := client.ListLeaderboards(requestCtx, opts.Language, opts.PageSize, opts.PageToken)
			if err != nil {
				return fmt.Errorf("failed to list leaderboards: %w", err)
			}
			return writeLeaderboardsListOutput(deps.Stdout, opts.Output, result)
		},
	}
}

func newLeaderboardsGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var leaderboardID string
	var language string
	var output string
	fs.StringVar(&leaderboardID, "leaderboard-id", "", "Leaderboard ID")
	fs.StringVar(&language, "language", "", "Preferred language for localized strings")
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get Play Games leaderboard metadata",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			leaderboardID = strings.TrimSpace(leaderboardID)
			if leaderboardID == "" {
				return shared.UsageErrorf("--leaderboard-id is required")
			}
			output, err := resolveOutput(output)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			result, err := client.GetLeaderboard(requestCtx, leaderboardID, language)
			if err != nil {
				return fmt.Errorf("failed to get leaderboard: %w", err)
			}
			return writeLeaderboardGetOutput(deps.Stdout, output, result)
		},
	}
}

func bindListFlags(fs *flag.FlagSet, opts *listOptions) {
	if opts == nil {
		return
	}
	fs.StringVar(&opts.Language, "language", "", "Preferred language for localized strings")
	fs.Int64Var(&opts.PageSize, "page-size", 0, "Maximum items per page")
	fs.StringVar(&opts.PageToken, "page-token", "", "Page token for the next page")
	fs.StringVar(&opts.Output, "output", "", "Output format: json, table, markdown")
}

func validateListOptions(opts listOptions) (listOptions, error) {
	if opts.PageSize < 0 {
		return listOptions{}, shared.UsageErrorf("--page-size must be greater than or equal to zero")
	}
	output, err := resolveOutput(opts.Output)
	if err != nil {
		return listOptions{}, err
	}
	opts.Language = strings.TrimSpace(opts.Language)
	opts.PageToken = strings.TrimSpace(opts.PageToken)
	opts.Output = output
	return opts, nil
}

func buildClient(ctx context.Context, deps Deps) (Client, context.Context, context.CancelFunc, error) {
	return shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
	})
}

func resolveOutput(local string) (string, error) {
	output := shared.ResolveOutput(local)
	switch output {
	case "json", "table", "markdown":
		return output, nil
	default:
		return "", shared.UsageErrorf("output must be json, table, or markdown")
	}
}

func writeAchievementsOutput(out io.Writer, output string, result gpc.GamesAchievementsListInfo) error {
	switch output {
	case "json":
		return shared.WriteJSON(out, result)
	case "table":
		if _, err := fmt.Fprintln(out, "ID\tNAME\tTYPE\tINITIAL_STATE\tXP\tTOTAL_STEPS"); err != nil {
			return err
		}
		for _, item := range result.Achievements {
			if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%d\t%d\n", item.ID, item.Name, item.AchievementType, item.InitialState, item.ExperiencePoints, item.TotalSteps); err != nil {
				return err
			}
		}
		return nil
	case "markdown":
		rows := make([][]string, 0, len(result.Achievements))
		for _, item := range result.Achievements {
			rows = append(rows, []string{item.ID, item.Name, item.AchievementType, item.InitialState, strconv.FormatInt(item.ExperiencePoints, 10), strconv.FormatInt(item.TotalSteps, 10)})
		}
		return shared.WriteMarkdownTable(out, []string{"id", "name", "type", "initialState", "xp", "totalSteps"}, rows)
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}

func writeEventsOutput(out io.Writer, output string, result gpc.GamesEventsListInfo) error {
	switch output {
	case "json":
		return shared.WriteJSON(out, result)
	case "table":
		if _, err := fmt.Fprintln(out, "ID\tDISPLAY_NAME\tVISIBILITY\tCHILD_EVENTS"); err != nil {
			return err
		}
		for _, item := range result.Events {
			if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%d\n", item.ID, item.DisplayName, item.Visibility, item.ChildEventCount); err != nil {
				return err
			}
		}
		return nil
	case "markdown":
		rows := make([][]string, 0, len(result.Events))
		for _, item := range result.Events {
			rows = append(rows, []string{item.ID, item.DisplayName, item.Visibility, strconv.Itoa(item.ChildEventCount)})
		}
		return shared.WriteMarkdownTable(out, []string{"id", "displayName", "visibility", "childEvents"}, rows)
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}

func writeLeaderboardsListOutput(out io.Writer, output string, result gpc.GamesLeaderboardsListInfo) error {
	switch output {
	case "json":
		return shared.WriteJSON(out, result)
	case "table":
		if _, err := fmt.Fprintln(out, "ID\tNAME\tORDER"); err != nil {
			return err
		}
		for _, item := range result.Leaderboards {
			if _, err := fmt.Fprintf(out, "%s\t%s\t%s\n", item.ID, item.Name, item.Order); err != nil {
				return err
			}
		}
		return nil
	case "markdown":
		rows := make([][]string, 0, len(result.Leaderboards))
		for _, item := range result.Leaderboards {
			rows = append(rows, []string{item.ID, item.Name, item.Order})
		}
		return shared.WriteMarkdownTable(out, []string{"id", "name", "order"}, rows)
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}

func writeLeaderboardGetOutput(out io.Writer, output string, result gpc.GamesLeaderboardInfo) error {
	switch output {
	case "json":
		return shared.WriteJSON(out, result)
	case "table":
		if _, err := fmt.Fprintln(out, "FIELD\tVALUE"); err != nil {
			return err
		}
		for _, row := range [][]string{{"id", result.ID}, {"name", result.Name}, {"order", result.Order}} {
			if _, err := fmt.Fprintf(out, "%s\t%s\n", row[0], row[1]); err != nil {
				return err
			}
		}
		return nil
	case "markdown":
		return shared.WriteMarkdownTable(out, []string{"field", "value"}, [][]string{{"id", result.ID}, {"name", result.Name}, {"order", result.Order}})
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}
