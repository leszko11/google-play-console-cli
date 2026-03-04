package tracks

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
	ListTracks(ctx context.Context, packageName, editID string) ([]gpc.TrackInfo, error)
	GetTrack(ctx context.Context, packageName, editID, trackName string) (gpc.TrackInfo, error)
	UpdateTrack(ctx context.Context, packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error)
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
		Name:      "tracks",
		ShortHelp: "Manage release tracks inside an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newGetCommand(deps),
			newUpdateCommand(deps),
			newPromoteCommand(deps),
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
	var packageName, editID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List tracks in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, false)
			if err != nil {
				return err
			}
			defer cancel()
			tracks, err := client.ListTracks(requestCtx, pkg, eid)
			if err != nil {
				return fmt.Errorf("failed to list tracks: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"tracks":      tracks,
			})
		},
	}
}

func newGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, trackName string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&trackName, "track", "", "Track name (e.g. production, internal)")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get a single track in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, false)
			if err != nil {
				return err
			}
			defer cancel()
			trackName = strings.TrimSpace(trackName)
			if trackName == "" {
				return fmt.Errorf("--track is required")
			}

			track, err := client.GetTrack(requestCtx, pkg, eid, trackName)
			if err != nil {
				return fmt.Errorf("failed to get track: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"track":       track,
			})
		},
	}
}

func newUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, trackName, status, releaseName, versionCodesCSV, releaseNotesFile string
	var userFraction float64
	var updatePriority int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&trackName, "track", "", "Track name (e.g. production, internal)")
	fs.StringVar(&status, "status", "", "Release status (draft, inProgress, halted, completed)")
	fs.StringVar(&releaseName, "release-name", "", "Release name")
	fs.StringVar(&versionCodesCSV, "version-codes", "", "Comma-separated version codes")
	fs.StringVar(&releaseNotesFile, "release-notes-file", "", "Path to release notes JSON payload (object or array)")
	fs.Float64Var(&userFraction, "user-fraction", -1, "Rollout user fraction (0-1)")
	fs.Int64Var(&updatePriority, "update-priority", 0, "In-app update priority (0-5)")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update a track release in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, false)
			if err != nil {
				return err
			}
			defer cancel()

			trackName = strings.TrimSpace(trackName)
			if trackName == "" {
				return fmt.Errorf("--track is required")
			}
			status = strings.TrimSpace(status)
			if status == "" {
				return fmt.Errorf("--status is required")
			}

			versionCodes, err := parseVersionCodes(versionCodesCSV)
			if err != nil {
				return err
			}
			if userFraction > 1 || (userFraction >= 0 && userFraction <= 0) {
				return fmt.Errorf("--user-fraction must be within (0,1] when set")
			}
			if updatePriority < 0 || updatePriority > 5 {
				return fmt.Errorf("--update-priority must be between 0 and 5")
			}
			releaseNotes, err := shared.ParseReleaseNotesFile(releaseNotesFile)
			if err != nil {
				return err
			}

			track, err := client.UpdateTrack(requestCtx, pkg, eid, trackName, gpc.TrackUpdate{
				Status:         status,
				ReleaseName:    releaseName,
				UserFraction:   userFraction,
				VersionCodes:   versionCodes,
				UpdatePriority: updatePriority,
				ReleaseNotes:   releaseNotes,
			})
			if err != nil {
				return fmt.Errorf("failed to update track: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"track":       track,
				"status":      "updated",
			})
		},
	}
}

func newPromoteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, fromTrack, toTrack, status, releaseName string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&fromTrack, "from-track", "", "Source track name")
	fs.StringVar(&toTrack, "to-track", "", "Target track name")
	fs.StringVar(&status, "status", "", "Override release status")
	fs.StringVar(&releaseName, "release-name", "", "Override release name")

	return &ffcli.Command{
		Name:      "promote",
		ShortHelp: "Promote a release from one track to another in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, false)
			if err != nil {
				return err
			}
			defer cancel()

			fromTrack = strings.TrimSpace(fromTrack)
			if fromTrack == "" {
				return fmt.Errorf("--from-track is required")
			}
			toTrack = strings.TrimSpace(toTrack)
			if toTrack == "" {
				return fmt.Errorf("--to-track is required")
			}
			if fromTrack == toTrack {
				return fmt.Errorf("--from-track and --to-track must be different")
			}

			sourceTrack, err := client.GetTrack(requestCtx, pkg, eid, fromTrack)
			if err != nil {
				return fmt.Errorf("failed to read source track: %w", err)
			}
			if len(sourceTrack.Releases) == 0 {
				return fmt.Errorf("source track %q has no releases to promote", fromTrack)
			}

			sourceRelease := sourceTrack.Releases[0]
			if len(sourceRelease.VersionCodes) == 0 {
				return fmt.Errorf("source track %q has no version codes to promote", fromTrack)
			}

			promotedStatus := strings.TrimSpace(status)
			if promotedStatus == "" {
				promotedStatus = sourceRelease.Status
			}
			promotedReleaseName := strings.TrimSpace(releaseName)
			if promotedReleaseName == "" {
				promotedReleaseName = sourceRelease.Name
			}

			targetTrack, err := client.UpdateTrack(requestCtx, pkg, eid, toTrack, gpc.TrackUpdate{
				Status:         promotedStatus,
				ReleaseName:    promotedReleaseName,
				UserFraction:   sourceRelease.UserFraction,
				VersionCodes:   sourceRelease.VersionCodes,
				UpdatePriority: sourceRelease.UpdatePriority,
				ReleaseNotes:   sourceRelease.ReleaseNotes,
			})
			if err != nil {
				return fmt.Errorf("failed to promote track: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"editId":        eid,
				"fromTrack":     fromTrack,
				"toTrack":       toTrack,
				"sourceTrack":   sourceTrack,
				"targetTrack":   targetTrack,
				"releaseName":   promotedReleaseName,
				"releaseStatus": promotedStatus,
				"status":        "promoted",
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName, editID string, upload bool) (Client, string, string, context.Context, context.CancelFunc, error) {
	pkg, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return nil, "", "", nil, nil, err
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, "", "", nil, nil, fmt.Errorf("--edit-id is required")
	}

	client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
		Upload:     upload,
	})
	if err != nil {
		return nil, "", "", nil, nil, err
	}

	return client, pkg, editID, requestCtx, cancel, nil
}

func parseVersionCodes(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("--version-codes is required")
	}

	parts := strings.Split(raw, ",")
	versionCodes := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		code, err := strconv.ParseInt(part, 10, 64)
		if err != nil || code <= 0 {
			return nil, fmt.Errorf("invalid version code %q", part)
		}
		versionCodes = append(versionCodes, code)
	}

	if len(versionCodes) == 0 {
		return nil, fmt.Errorf("--version-codes must include at least one valid integer")
	}
	return versionCodes, nil
}
