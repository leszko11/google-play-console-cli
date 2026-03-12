package changelog

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ValidateEdit(ctx context.Context, packageName, editID string) error
	CommitEdit(ctx context.Context, packageName, editID string) (gpc.EditInfo, error)
	GetTrack(ctx context.Context, packageName, editID, trackName string) (gpc.TrackInfo, error)
	UpdateTrack(ctx context.Context, packageName, editID, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error)
}

type syncOptions struct {
	PackageName string
	Track       string
	Dir         string
	Confirm     bool
	DryRun      bool
}

type syncResult struct {
	PackageName string              `json:"packageName"`
	Track       string              `json:"track"`
	Status      string              `json:"status"`
	ReleaseName string              `json:"releaseName,omitempty"`
	Notes       []gpc.LocalizedText `json:"notes,omitempty"`
	Committed   bool                `json:"committed"`
}

func newSyncCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts syncOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Track, "track", "", "Track name")
	fs.StringVar(&opts.Dir, "dir", "", "Changelog directory")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm committing the edit (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Create and validate the edit, then delete it instead of updating Play")

	return &ffcli.Command{
		Name:      "sync",
		ShortHelp: "Sync track release notes from locale-named text files",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateOptions(opts)
			if err != nil {
				return err
			}
			notes, err := loadReleaseNotesDir(opts.Dir)
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
			return runSync(ctx, requestCtx, client, deps.Stdout, opts, notes)
		},
	}
}

func validateOptions(opts syncOptions) (syncOptions, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return syncOptions{}, err
	}
	opts.PackageName = pkg
	opts.Track = strings.TrimSpace(opts.Track)
	opts.Dir = strings.TrimSpace(opts.Dir)
	if opts.Track == "" {
		return syncOptions{}, shared.UsageErrorf("--track is required")
	}
	if opts.Dir == "" {
		return syncOptions{}, shared.UsageErrorf("--dir is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return syncOptions{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}
	return opts, nil
}

func loadReleaseNotesDir(dir string) ([]gpc.LocalizedText, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read changelog directory: %w", err)
	}
	notes := make([]gpc.LocalizedText, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || strings.ToLower(filepath.Ext(entry.Name())) != ".txt" {
			continue
		}
		locale := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(string(raw))
		if text == "" {
			return nil, fmt.Errorf("release notes file %s is empty", entry.Name())
		}
		notes = append(notes, gpc.LocalizedText{Language: locale, Text: text})
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].Language < notes[j].Language })
	if len(notes) == 0 {
		return nil, fmt.Errorf("no release notes files found in %s", dir)
	}
	return notes, nil
}

func runSync(parentCtx, requestCtx context.Context, client Client, out io.Writer, opts syncOptions, notes []gpc.LocalizedText) error {
	result := syncResult{
		PackageName: opts.PackageName,
		Track:       opts.Track,
		Status:      "failed",
		Notes:       notes,
	}

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return err
	}
	fail := func(err error) error {
		_ = client.DeleteEdit(parentCtx, opts.PackageName, edit.ID)
		_ = shared.WriteJSON(out, result)
		return err
	}

	track, err := client.GetTrack(requestCtx, opts.PackageName, edit.ID, opts.Track)
	if err != nil {
		return fail(fmt.Errorf("failed to read track: %w", err))
	}
	release, err := selectRelease(track)
	if err != nil {
		return fail(err)
	}
	result.ReleaseName = release.Name

	if opts.DryRun {
		if err := client.ValidateEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
			return fail(fmt.Errorf("failed to validate edit: %w", err))
		}
		if err := client.DeleteEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
			return fail(fmt.Errorf("failed to delete dry-run edit: %w", err))
		}
		result.Status = "dry-run"
		return shared.WriteJSON(out, result)
	}

	if _, err := client.UpdateTrack(requestCtx, opts.PackageName, edit.ID, opts.Track, gpc.TrackUpdate{
		Status:         release.Status,
		ReleaseName:    release.Name,
		UserFraction:   release.UserFraction,
		VersionCodes:   append([]int64(nil), release.VersionCodes...),
		UpdatePriority: release.UpdatePriority,
		ReleaseNotes:   notes,
	}); err != nil {
		return fail(fmt.Errorf("failed to update track: %w", err))
	}
	if err := client.ValidateEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
		return fail(fmt.Errorf("failed to validate edit: %w", err))
	}
	if _, err := client.CommitEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
		return fail(fmt.Errorf("failed to commit edit: %w", err))
	}
	result.Status = "committed"
	result.Committed = true
	return shared.WriteJSON(out, result)
}

func selectRelease(track gpc.TrackInfo) (gpc.TrackReleaseInfo, error) {
	candidates := make([]gpc.TrackReleaseInfo, 0, len(track.Releases))
	for _, release := range track.Releases {
		if len(release.VersionCodes) > 0 {
			candidates = append(candidates, release)
		}
	}
	switch len(candidates) {
	case 0:
		return gpc.TrackReleaseInfo{}, fmt.Errorf("track %q has no releasable release", track.Name)
	case 1:
		return candidates[0], nil
	default:
		return gpc.TrackReleaseInfo{}, fmt.Errorf("track %q has multiple releases; refusing to update implicitly", track.Name)
	}
}
