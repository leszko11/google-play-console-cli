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

const envServiceAccountPath = "GPC_SERVICE_ACCOUNT_PATH"

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
			client, pkg, eid, err := buildClient(ctx, deps, packageName, editID)
			if err != nil {
				return err
			}
			tracks, err := client.ListTracks(ctx, pkg, eid)
			if err != nil {
				return fmt.Errorf("failed to list tracks: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
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
			client, pkg, eid, err := buildClient(ctx, deps, packageName, editID)
			if err != nil {
				return err
			}
			trackName = strings.TrimSpace(trackName)
			if trackName == "" {
				return fmt.Errorf("--track is required")
			}

			track, err := client.GetTrack(ctx, pkg, eid, trackName)
			if err != nil {
				return fmt.Errorf("failed to get track: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
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
	var packageName, editID, trackName, status, releaseName, versionCodesCSV string
	var userFraction float64
	var updatePriority int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&trackName, "track", "", "Track name (e.g. production, internal)")
	fs.StringVar(&status, "status", "", "Release status (draft, inProgress, halted, completed)")
	fs.StringVar(&releaseName, "release-name", "", "Release name")
	fs.StringVar(&versionCodesCSV, "version-codes", "", "Comma-separated version codes")
	fs.Float64Var(&userFraction, "user-fraction", -1, "Rollout user fraction (0-1)")
	fs.Int64Var(&updatePriority, "update-priority", 0, "In-app update priority (0-5)")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update a track release in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, err := buildClient(ctx, deps, packageName, editID)
			if err != nil {
				return err
			}

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

			track, err := client.UpdateTrack(ctx, pkg, eid, trackName, gpc.TrackUpdate{
				Status:         status,
				ReleaseName:    releaseName,
				UserFraction:   userFraction,
				VersionCodes:   versionCodes,
				UpdatePriority: updatePriority,
			})
			if err != nil {
				return fmt.Errorf("failed to update track: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"track":       track,
				"status":      "updated",
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName, editID string) (Client, string, string, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, "", "", fmt.Errorf("--package-name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, "", "", fmt.Errorf("--edit-id is required")
	}

	cfg, err := deps.LoadConfig()
	if err != nil {
		return nil, "", "", err
	}
	serviceAccountPath, err := resolveServiceAccountPath(cfg, deps.LookupEnv)
	if err != nil {
		return nil, "", "", err
	}

	client, err := deps.NewClient(ctx, gpc.CredentialInput{ServiceAccountPath: serviceAccountPath})
	if err != nil {
		return nil, "", "", err
	}

	return client, packageName, editID, nil
}

func resolveServiceAccountPath(cfg config.Config, lookupEnv func(string) string) (string, error) {
	if cfg.ActiveProfile != "" && cfg.Profiles != nil {
		if profile, ok := cfg.Profiles[cfg.ActiveProfile]; ok && profile.ServiceAccountPath != "" {
			return profile.ServiceAccountPath, nil
		}
	}

	if envPath := strings.TrimSpace(lookupEnv(envServiceAccountPath)); envPath != "" {
		return envPath, nil
	}

	return "", fmt.Errorf("no service account configured")
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

func writeJSON(out io.Writer, v any) error {
	b, err := shared.RenderJSON(v, false)
	if err != nil {
		return err
	}
	_, err = out.Write(b)
	return err
}
