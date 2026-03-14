package diff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	listingcmd "github.com/leszko11/google-play-console-cli/internal/cli/listing"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	notesgen "github.com/leszko11/google-play-console-cli/internal/release/notes"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ListListings(ctx context.Context, packageName, editID string) ([]gpc.ListingInfo, error)
	ListImages(ctx context.Context, packageName, editID, language, imageType string) ([]gpc.ImageInfo, error)
	ListTracks(ctx context.Context, packageName, editID string) ([]gpc.TrackInfo, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
}

type change struct {
	Scope   string `json:"scope"`
	Target  string `json:"target"`
	Field   string `json:"field,omitempty"`
	Action  string `json:"action"`
	Live    any    `json:"live,omitempty"`
	Desired any    `json:"desired,omitempty"`
}

type listingResult struct {
	PackageName    string   `json:"packageName"`
	Dir            string   `json:"dir"`
	DeleteMissing  bool     `json:"deleteMissing"`
	HasDiff        bool     `json:"hasDiff"`
	LocaleCount    int      `json:"localeCount"`
	RemoteLocales  int      `json:"remoteLocaleCount"`
	ChangeCount    int      `json:"changeCount"`
	UnchangedCount int      `json:"unchangedLocaleCount"`
	Changes        []change `json:"changes"`
}

type trackResult struct {
	PackageName string   `json:"packageName"`
	Track       string   `json:"track"`
	HasDiff     bool     `json:"hasDiff"`
	TrackFound  bool     `json:"trackFound"`
	ReleaseName string   `json:"liveReleaseName,omitempty"`
	ChangeCount int      `json:"changeCount"`
	Changes     []change `json:"changes"`
}

type listingOptions struct {
	PackageName   string
	Dir           string
	DeleteMissing bool
	Output        string
}

type trackOptions struct {
	PackageName        string
	Track              string
	ReleaseStatus      string
	ReleaseName        string
	VersionCodesCSV    string
	UserFraction       float64
	UpdatePriority     int64
	ReleaseNotesFile   string
	ReleaseNotesLocale string
	ReleaseNotesText   string
	Output             string
}

type desiredTrackDraft struct {
	Status         string
	ReleaseName    string
	VersionCodes   []int64
	UserFraction   *float64
	UpdatePriority *int64
	ReleaseNotes   []gpc.LocalizedText
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "diff",
		ShortHelp: "Compare live Play state against local listing or track drafts",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListingCommand(deps),
			newTrackCommand(deps),
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

func newListingCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("listing", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts listingOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Dir, "dir", "", "Listings directory root")
	fs.BoolVar(&opts.DeleteMissing, "delete-missing", false, "Mark remote-only locales as deletions")
	fs.StringVar(&opts.Output, "output", "", "Output format: json or table")

	return &ffcli.Command{
		Name:      "listing",
		ShortHelp: "Compare live listings against a local listing directory",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateListingOptions(opts)
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

			locales, err := listingcmd.ScanListingsDir(opts.Dir)
			if err != nil {
				return err
			}

			result, err := runListingDiff(ctx, requestCtx, client, opts, locales)
			if err != nil {
				return err
			}
			return writeResult(deps.Stdout, opts.Output, result, writeListingTable)
		},
	}
}

func newTrackCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("track", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	opts := trackOptions{
		UserFraction:       -1,
		ReleaseNotesLocale: notesgen.DefaultLocale,
	}
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Track, "track", "", "Track name (e.g. production, internal)")
	fs.StringVar(&opts.ReleaseStatus, "status", "", "Release status (draft, inProgress, halted, completed)")
	fs.StringVar(&opts.ReleaseName, "release-name", "", "Release name")
	fs.StringVar(&opts.VersionCodesCSV, "version-codes", "", "Comma-separated version codes")
	fs.Float64Var(&opts.UserFraction, "user-fraction", -1, "Rollout user fraction (0-1)")
	fs.Int64Var(&opts.UpdatePriority, "update-priority", 0, "In-app update priority (0-5)")
	fs.StringVar(&opts.ReleaseNotesFile, "release-notes-file", "", "Path to release notes file (JSON object/array, tagged blocks, or plain text)")
	fs.StringVar(&opts.ReleaseNotesLocale, "release-notes-locale", notesgen.DefaultLocale, "Release notes locale (BCP-47)")
	fs.StringVar(&opts.ReleaseNotesText, "release-notes-text", "", "Release notes text")
	fs.StringVar(&opts.Output, "output", "", "Output format: json or table")

	return &ffcli.Command{
		Name:      "track",
		ShortHelp: "Compare a draft track release against the live track",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, draft, err := validateTrackOptions(opts)
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

			result, err := runTrackDiff(ctx, requestCtx, client, opts, draft)
			if err != nil {
				return err
			}
			return writeResult(deps.Stdout, opts.Output, result, writeTrackTable)
		},
	}
}

func validateListingOptions(opts listingOptions) (listingOptions, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return listingOptions{}, err
	}
	opts.PackageName = pkg

	opts.Dir, err = shared.ResolveProjectPath(opts.Dir, func(cfg config.ProjectConfig) string { return cfg.ListingDir })
	if err != nil {
		return listingOptions{}, err
	}
	opts.Dir = strings.TrimSpace(opts.Dir)
	if opts.Dir == "" {
		return listingOptions{}, shared.UsageErrorf("--dir is required")
	}
	if _, err := resolveOutput(opts.Output); err != nil {
		return listingOptions{}, err
	}
	return opts, nil
}

func validateTrackOptions(opts trackOptions) (trackOptions, desiredTrackDraft, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return trackOptions{}, desiredTrackDraft{}, err
	}
	opts.PackageName = pkg
	opts.Track, err = shared.ResolveDefaultTrack(opts.Track)
	if err != nil {
		return trackOptions{}, desiredTrackDraft{}, err
	}
	opts.Track = strings.TrimSpace(opts.Track)
	if opts.Track == "" {
		return trackOptions{}, desiredTrackDraft{}, shared.UsageErrorf("--track is required")
	}
	opts.ReleaseStatus = strings.TrimSpace(opts.ReleaseStatus)
	if opts.ReleaseStatus == "" {
		return trackOptions{}, desiredTrackDraft{}, shared.UsageErrorf("--status is required")
	}
	versionCodes, err := parseVersionCodes(opts.VersionCodesCSV)
	if err != nil {
		return trackOptions{}, desiredTrackDraft{}, err
	}
	if opts.UserFraction > 1 || (opts.UserFraction >= 0 && opts.UserFraction <= 0) {
		return trackOptions{}, desiredTrackDraft{}, shared.UsageErrorf("--user-fraction must be within (0,1] when set")
	}
	if opts.UpdatePriority < 0 || opts.UpdatePriority > 5 {
		return trackOptions{}, desiredTrackDraft{}, shared.UsageErrorf("--update-priority must be between 0 and 5")
	}
	if _, err := resolveOutput(opts.Output); err != nil {
		return trackOptions{}, desiredTrackDraft{}, err
	}

	releaseNotes, err := shared.ParseReleaseNotesInput(
		opts.ReleaseNotesFile,
		opts.ReleaseNotesText,
		opts.ReleaseNotesLocale,
		os.ReadFile,
	)
	if err != nil {
		return trackOptions{}, desiredTrackDraft{}, err
	}

	draft := desiredTrackDraft{
		Status:       opts.ReleaseStatus,
		ReleaseName:  strings.TrimSpace(opts.ReleaseName),
		VersionCodes: versionCodes,
		ReleaseNotes: normalizeNotes(releaseNotes),
	}
	if opts.UserFraction >= 0 {
		draft.UserFraction = &opts.UserFraction
	}
	if opts.UpdatePriority > 0 {
		draft.UpdatePriority = &opts.UpdatePriority
	}
	return opts, draft, nil
}

func runListingDiff(parentCtx, requestCtx context.Context, client Client, opts listingOptions, locales []listingcmd.LocaleData) (listingResult, error) {
	result := listingResult{
		PackageName:   opts.PackageName,
		Dir:           opts.Dir,
		DeleteMissing: opts.DeleteMissing,
		LocaleCount:   len(locales),
		Changes:       []change{},
	}

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return result, fmt.Errorf("failed to create edit: %w", err)
	}
	defer func() {
		_ = deleteEdit(parentCtx, client, opts.PackageName, edit.ID)
	}()

	remoteListings, err := client.ListListings(requestCtx, opts.PackageName, edit.ID)
	if err != nil {
		return result, fmt.Errorf("failed to list live listings: %w", err)
	}
	result.RemoteLocales = len(remoteListings)

	remoteByLocale := make(map[string]gpc.ListingInfo, len(remoteListings))
	for _, listing := range remoteListings {
		remoteByLocale[strings.TrimSpace(listing.Language)] = listing
	}

	changes := make([]change, 0)
	unchanged := 0
	for _, locale := range locales {
		remote, exists := remoteByLocale[locale.Locale]
		if !exists {
			changes = append(changes, change{
				Scope:   "listing",
				Target:  locale.Locale,
				Action:  "create_locale",
				Desired: locale.Listing,
			})
			delete(remoteByLocale, locale.Locale)
			continue
		}

		localeChanges, err := compareListingLocale(requestCtx, client, opts.PackageName, edit.ID, locale, remote)
		if err != nil {
			return result, err
		}
		if len(localeChanges) == 0 {
			unchanged++
		}
		changes = append(changes, localeChanges...)
		delete(remoteByLocale, locale.Locale)
	}

	remainingLocales := make([]string, 0, len(remoteByLocale))
	for locale := range remoteByLocale {
		remainingLocales = append(remainingLocales, locale)
	}
	sort.Strings(remainingLocales)
	for _, locale := range remainingLocales {
		action := "remote_only_locale"
		if opts.DeleteMissing {
			action = "delete_locale"
		}
		changes = append(changes, change{
			Scope:  "listing",
			Target: locale,
			Action: action,
			Live:   remoteByLocale[locale],
		})
	}

	sortChanges(changes)
	result.Changes = changes
	result.ChangeCount = len(changes)
	result.HasDiff = len(changes) > 0
	result.UnchangedCount = unchanged
	return result, nil
}

func compareListingLocale(ctx context.Context, client Client, packageName, editID string, locale listingcmd.LocaleData, remote gpc.ListingInfo) ([]change, error) {
	changes := make([]change, 0)
	if remote.Title != locale.Listing.Title {
		changes = append(changes, change{
			Scope:   "listing",
			Target:  locale.Locale,
			Field:   "title",
			Action:  "update",
			Live:    remote.Title,
			Desired: locale.Listing.Title,
		})
	}
	if remote.ShortDescription != locale.Listing.ShortDescription {
		changes = append(changes, change{
			Scope:   "listing",
			Target:  locale.Locale,
			Field:   "shortDescription",
			Action:  "update",
			Live:    remote.ShortDescription,
			Desired: locale.Listing.ShortDescription,
		})
	}
	if remote.FullDescription != locale.Listing.FullDescription {
		changes = append(changes, change{
			Scope:   "listing",
			Target:  locale.Locale,
			Field:   "fullDescription",
			Action:  "update",
			Live:    remote.FullDescription,
			Desired: locale.Listing.FullDescription,
		})
	}

	imageTypes := sortedImageTypes(locale.Images)
	for _, imageType := range imageTypes {
		remoteImages, err := client.ListImages(ctx, packageName, editID, locale.Locale, imageType)
		if err != nil {
			return nil, fmt.Errorf("failed to list live images for %q/%q: %w", locale.Locale, imageType, err)
		}
		liveHashes := normalizeRemoteImages(remoteImages)
		desiredHashes, err := hashFiles(locale.Images[imageType])
		if err != nil {
			return nil, fmt.Errorf("failed to hash local images for %q/%q: %w", locale.Locale, imageType, err)
		}
		if !reflect.DeepEqual(liveHashes, desiredHashes) {
			changes = append(changes, change{
				Scope:   "images",
				Target:  locale.Locale,
				Field:   imageType,
				Action:  "replace",
				Live:    liveHashes,
				Desired: desiredHashes,
			})
		}
	}

	return changes, nil
}

func runTrackDiff(parentCtx, requestCtx context.Context, client Client, opts trackOptions, draft desiredTrackDraft) (trackResult, error) {
	result := trackResult{
		PackageName: opts.PackageName,
		Track:       opts.Track,
		Changes:     []change{},
	}

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return result, fmt.Errorf("failed to create edit: %w", err)
	}
	defer func() {
		_ = deleteEdit(parentCtx, client, opts.PackageName, edit.ID)
	}()

	tracks, err := client.ListTracks(requestCtx, opts.PackageName, edit.ID)
	if err != nil {
		return result, fmt.Errorf("failed to list live tracks: %w", err)
	}

	var current *gpc.TrackInfo
	for i := range tracks {
		if strings.TrimSpace(tracks[i].Name) == opts.Track {
			current = &tracks[i]
			break
		}
	}
	if current == nil {
		result.Changes = []change{{
			Scope:   "track",
			Target:  opts.Track,
			Action:  "create_track",
			Desired: draft,
		}}
		result.HasDiff = true
		result.TrackFound = false
		result.ChangeCount = 1
		return result, nil
	}

	result.TrackFound = true
	if len(current.Releases) == 0 {
		result.Changes = []change{{
			Scope:   "track",
			Target:  opts.Track,
			Action:  "create_release",
			Desired: draft,
		}}
		result.HasDiff = true
		result.ChangeCount = 1
		return result, nil
	}

	live := current.Releases[0]
	result.ReleaseName = live.Name
	changes := compareTrackDraft(opts.Track, live, draft)
	sortChanges(changes)
	result.Changes = changes
	result.ChangeCount = len(changes)
	result.HasDiff = len(changes) > 0
	return result, nil
}

func compareTrackDraft(trackName string, live gpc.TrackReleaseInfo, draft desiredTrackDraft) []change {
	changes := make([]change, 0)
	if live.Status != draft.Status {
		changes = append(changes, change{
			Scope:   "track",
			Target:  trackName,
			Field:   "status",
			Action:  "update",
			Live:    live.Status,
			Desired: draft.Status,
		})
	}
	if live.Name != draft.ReleaseName {
		changes = append(changes, change{
			Scope:   "track",
			Target:  trackName,
			Field:   "releaseName",
			Action:  "update",
			Live:    live.Name,
			Desired: draft.ReleaseName,
		})
	}
	if !reflect.DeepEqual(normalizeVersionCodes(live.VersionCodes), normalizeVersionCodes(draft.VersionCodes)) {
		changes = append(changes, change{
			Scope:   "track",
			Target:  trackName,
			Field:   "versionCodes",
			Action:  "update",
			Live:    normalizeVersionCodes(live.VersionCodes),
			Desired: normalizeVersionCodes(draft.VersionCodes),
		})
	}
	if !equalOptionalFloat(live.UserFraction, draft.UserFraction) {
		changes = append(changes, change{
			Scope:   "track",
			Target:  trackName,
			Field:   "userFraction",
			Action:  "update",
			Live:    floatOrNil(live.UserFraction, live.UserFraction > 0),
			Desired: draft.UserFraction,
		})
	}
	if !equalOptionalInt(live.UpdatePriority, draft.UpdatePriority) {
		changes = append(changes, change{
			Scope:   "track",
			Target:  trackName,
			Field:   "updatePriority",
			Action:  "update",
			Live:    intOrNil(live.UpdatePriority, live.UpdatePriority > 0),
			Desired: draft.UpdatePriority,
		})
	}
	liveNotes := normalizeNotes(live.ReleaseNotes)
	if !reflect.DeepEqual(liveNotes, draft.ReleaseNotes) {
		changes = append(changes, change{
			Scope:   "track",
			Target:  trackName,
			Field:   "releaseNotes",
			Action:  "update",
			Live:    sliceOrNil(liveNotes),
			Desired: sliceOrNil(draft.ReleaseNotes),
		})
	}
	return changes
}

func deleteEdit(ctx context.Context, client Client, packageName, editID string) error {
	cleanupCtx, cleanupCancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
	defer cleanupCancel()
	if err := client.DeleteEdit(cleanupCtx, packageName, editID); err != nil {
		return fmt.Errorf("failed to clean up edit: %w", err)
	}
	return nil
}

func writeResult(out io.Writer, output string, payload any, tableWriter func(io.Writer, any) error) error {
	format, err := resolveOutput(output)
	if err != nil {
		return err
	}
	switch format {
	case "json":
		return shared.WriteJSON(out, payload)
	case "table":
		return tableWriter(out, payload)
	default:
		return shared.UsageErrorf("unsupported output format %q", format)
	}
}

func writeListingTable(out io.Writer, payload any) error {
	result, ok := payload.(listingResult)
	if !ok {
		return fmt.Errorf("unexpected listing payload type %T", payload)
	}
	status := "no-diff"
	if result.HasDiff {
		status = "diff"
	}
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", result.PackageName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "DIR\t%s\n", result.Dir); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "SCOPE\tTARGET\tFIELD\tACTION\tLIVE\tDESIRED"); err != nil {
		return err
	}
	for _, entry := range result.Changes {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\n", entry.Scope, entry.Target, entry.Field, entry.Action, formatValue(entry.Live), formatValue(entry.Desired)); err != nil {
			return err
		}
	}
	return nil
}

func writeTrackTable(out io.Writer, payload any) error {
	result, ok := payload.(trackResult)
	if !ok {
		return fmt.Errorf("unexpected track payload type %T", payload)
	}
	status := "no-diff"
	if result.HasDiff {
		status = "diff"
	}
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", result.PackageName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "TRACK\t%s\n", result.Track); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "TRACK_FOUND\t%t\n", result.TrackFound); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "SCOPE\tTARGET\tFIELD\tACTION\tLIVE\tDESIRED"); err != nil {
		return err
	}
	for _, entry := range result.Changes {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\n", entry.Scope, entry.Target, entry.Field, entry.Action, formatValue(entry.Live), formatValue(entry.Desired)); err != nil {
			return err
		}
	}
	return nil
}

func resolveOutput(local string) (string, error) {
	output := shared.ResolveOutput(local)
	switch output {
	case "json", "table":
		return output, nil
	default:
		return "", shared.UsageErrorf("output must be json or table")
	}
}

func formatValue(v any) string {
	if v == nil {
		return "-"
	}
	switch typed := v.(type) {
	case string:
		if typed == "" {
			return "-"
		}
		return typed
	case *float64:
		if typed == nil {
			return "-"
		}
		return strconv.FormatFloat(*typed, 'f', -1, 64)
	case *int64:
		if typed == nil {
			return "-"
		}
		return strconv.FormatInt(*typed, 10)
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(raw)
}

func parseVersionCodes(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, shared.UsageErrorf("--version-codes is required")
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
			return nil, shared.UsageErrorf("invalid version code %q", part)
		}
		versionCodes = append(versionCodes, code)
	}
	if len(versionCodes) == 0 {
		return nil, shared.UsageErrorf("--version-codes must include at least one valid integer")
	}
	return versionCodes, nil
}

func sortChanges(changes []change) {
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Scope != changes[j].Scope {
			return changes[i].Scope < changes[j].Scope
		}
		if changes[i].Target != changes[j].Target {
			return changes[i].Target < changes[j].Target
		}
		if changes[i].Field != changes[j].Field {
			return changes[i].Field < changes[j].Field
		}
		return changes[i].Action < changes[j].Action
	})
}

func sortedImageTypes(images map[string][]string) []string {
	keys := make([]string, 0, len(images))
	for key := range images {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeRemoteImages(images []gpc.ImageInfo) []string {
	out := make([]string, 0, len(images))
	for _, image := range images {
		switch {
		case strings.TrimSpace(image.SHA256) != "":
			out = append(out, strings.ToLower(strings.TrimSpace(image.SHA256)))
		case strings.TrimSpace(image.SHA1) != "":
			out = append(out, "sha1:"+strings.ToLower(strings.TrimSpace(image.SHA1)))
		default:
			out = append(out, "id:"+strings.TrimSpace(image.ID))
		}
	}
	sort.Strings(out)
	return out
}

func hashFiles(paths []string) ([]string, error) {
	hashes := make([]string, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(raw)
		hashes = append(hashes, hex.EncodeToString(sum[:]))
	}
	sort.Strings(hashes)
	return hashes, nil
}

func normalizeVersionCodes(values []int64) []int64 {
	out := append([]int64(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func normalizeNotes(values []gpc.LocalizedText) []gpc.LocalizedText {
	out := append([]gpc.LocalizedText(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Language != out[j].Language {
			return out[i].Language < out[j].Language
		}
		return out[i].Text < out[j].Text
	})
	return out
}

func equalOptionalFloat(live float64, desired *float64) bool {
	if desired == nil {
		return live == 0
	}
	return live == *desired
}

func equalOptionalInt(live int64, desired *int64) bool {
	if desired == nil {
		return live == 0
	}
	return live == *desired
}

func floatOrNil(value float64, set bool) *float64 {
	if !set {
		return nil
	}
	return &value
}

func intOrNil(value int64, set bool) *int64 {
	if !set {
		return nil
	}
	return &value
}

func sliceOrNil[T any](values []T) any {
	if len(values) == 0 {
		return nil
	}
	return values
}
