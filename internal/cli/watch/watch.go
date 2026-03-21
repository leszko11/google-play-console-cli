package watch

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

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
	// Clock allows tests to inject a fake time source.
	Clock func() time.Time
	// PollOnce when true runs a single iteration then exits (for testing).
	PollOnce bool
}

type snapshot struct {
	Timestamp      string          `json:"timestamp"`
	PackageName    string          `json:"packageName"`
	Tracks         []gpc.TrackInfo `json:"tracks,omitempty"`
	ReviewTotal    int             `json:"reviewTotal"`
	AverageRating  float64         `json:"averageRating"`
	PendingReplies int             `json:"pendingReplies"`
	TrackError     string          `json:"trackError,omitempty"`
	ReviewsError   string          `json:"reviewsError,omitempty"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var packageName, output string
	var intervalSec int
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&output, "output", "", "Output format: json, table")
	fs.IntVar(&intervalSec, "interval", 300, "Poll interval in seconds")

	return &ffcli.Command{
		Name:      "watch",
		ShortHelp: "Monitor release status and review health at regular intervals",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			pkg, err := shared.ResolvePackageName(packageName)
			if err != nil {
				return err
			}

			if intervalSec < 10 {
				return shared.UsageErrorf("--interval must be at least 10 seconds")
			}

			resolvedOutput := shared.ResolveOutput(output)
			if resolvedOutput != "json" && resolvedOutput != "table" {
				return shared.UsageErrorf("watch supports only json and table output formats")
			}

			client, _, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
			})
			if err != nil {
				return err
			}
			defer cancel()

			interval := time.Duration(intervalSec) * time.Second
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			// Run first poll immediately
			snap := poll(ctx, client, pkg, deps.Clock)
			if err := writeSnapshot(deps.Stdout, resolvedOutput, snap); err != nil {
				return err
			}

			if deps.PollOnce {
				return nil
			}

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					snap = poll(ctx, client, pkg, deps.Clock)
					if err := writeSnapshot(deps.Stdout, resolvedOutput, snap); err != nil {
						return err
					}
				}
			}
		},
	}
}

func poll(ctx context.Context, client Client, packageName string, clock func() time.Time) snapshot {
	snap := snapshot{
		Timestamp:   clock().UTC().Format(time.RFC3339),
		PackageName: packageName,
	}

	edit, err := client.CreateEdit(ctx, packageName)
	if err != nil {
		snap.TrackError = fmt.Sprintf("failed to create edit: %v", err)
	} else {
		tracks, listErr := client.ListTracks(ctx, packageName, edit.ID)
		if listErr != nil {
			snap.TrackError = fmt.Sprintf("failed to list tracks: %v", listErr)
		} else {
			sort.Slice(tracks, func(i, j int) bool {
				return tracks[i].Name < tracks[j].Name
			})
			snap.Tracks = tracks
		}
		_ = client.DeleteEdit(ctx, packageName, edit.ID)
	}

	reviews, err := client.ListReviews(ctx, packageName, reviewSnapshotLimit, 0, "", "", false)
	if err != nil {
		snap.ReviewsError = fmt.Sprintf("failed to list reviews: %v", err)
	} else {
		snap.ReviewTotal = len(reviews.Reviews)
		var totalRating int64
		for _, review := range reviews.Reviews {
			if review.StarRating >= 1 && review.StarRating <= 5 {
				totalRating += review.StarRating
			}
			if !review.HasReply {
				snap.PendingReplies++
			}
		}
		if snap.ReviewTotal > 0 {
			snap.AverageRating = float64(totalRating) / float64(snap.ReviewTotal)
		}
	}

	return snap
}

func writeSnapshot(out io.Writer, format string, snap snapshot) error {
	switch format {
	case "json":
		return shared.WriteJSON(out, snap)
	case "table":
		if _, err := fmt.Fprintf(out, "[%s] %s\n", snap.Timestamp, snap.PackageName); err != nil {
			return err
		}
		if snap.TrackError != "" {
			if _, err := fmt.Fprintf(out, "  TRACK_ERROR\t%s\n", snap.TrackError); err != nil {
				return err
			}
		} else {
			for _, track := range snap.Tracks {
				for _, release := range track.Releases {
					fraction := "-"
					if release.UserFraction > 0 {
						fraction = fmt.Sprintf("%.1f%%", release.UserFraction*100)
					}
					codes := make([]string, 0, len(release.VersionCodes))
					for _, code := range release.VersionCodes {
						codes = append(codes, strconv.FormatInt(code, 10))
					}
					if _, err := fmt.Fprintf(out, "  %s\t%s\t%s\t%s\n", track.Name, release.Status, fraction, strings.Join(codes, ",")); err != nil {
						return err
					}
				}
			}
		}
		if snap.ReviewsError != "" {
			if _, err := fmt.Fprintf(out, "  REVIEWS_ERROR\t%s\n", snap.ReviewsError); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(out, "  REVIEWS\t%d total\tavg %.1f\t%d pending reply\n", snap.ReviewTotal, snap.AverageRating, snap.PendingReplies); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		return nil
	default:
		return shared.UsageErrorf("unsupported output format %q", format)
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
	if deps.Clock == nil {
		deps.Clock = time.Now
	}
	return deps
}
