package release

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type rehearseOptions struct {
	PackageName          string
	ManifestPath         string
	ProbeTrack           bool
	VitalsGateRaw        string
	VitalsGate           []vitalsGateCondition
	VitalsWait           time.Duration
	AutoHaltOnRegression bool
	Strict               bool
	Output               string
}

type rehearseTrackChange struct {
	Field   string `json:"field,omitempty"`
	Action  string `json:"action"`
	Live    any    `json:"live,omitempty"`
	Desired any    `json:"desired,omitempty"`
}

type rehearseTrackPreview struct {
	TrackFound   bool                  `json:"trackFound"`
	ReleaseFound bool                  `json:"releaseFound"`
	ReleaseName  string                `json:"releaseName,omitempty"`
	HasDiff      bool                  `json:"hasDiff"`
	ChangeCount  int                   `json:"changeCount"`
	Changes      []rehearseTrackChange `json:"changes,omitempty"`
}

type rehearseResult struct {
	PackageName            string                `json:"packageName"`
	Manifest               string                `json:"manifest"`
	Track                  string                `json:"track"`
	AuthHealth             string                `json:"authHealth,omitempty"`
	PackageReadiness       string                `json:"packageReadiness,omitempty"`
	BootstrapDraftExists   bool                  `json:"bootstrapDraftExists,omitempty"`
	BootstrapVersionCodes  []int64               `json:"bootstrapVersionCodes,omitempty"`
	RecommendedNextCommand string                `json:"recommendedNextCommand,omitempty"`
	Status                 string                `json:"status"`
	Verify                 *verifyResult         `json:"verify,omitempty"`
	TrackPreview           *rehearseTrackPreview `json:"trackPreview,omitempty"`
	VitalsGate             *fullVitalsGateResult `json:"vitalsGate,omitempty"`
	BlockingIssues         []string              `json:"blockingIssues,omitempty"`
	Warnings               []string              `json:"warnings,omitempty"`
	PlannedSteps           []string              `json:"plannedSteps,omitempty"`
	NextSteps              []string              `json:"nextSteps,omitempty"`
}

func newRehearseCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("rehearse", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts rehearseOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Target package name")
	fs.StringVar(&opts.ManifestPath, "manifest", "", "Path to release manifest (.json/.yaml/.yml)")
	fs.BoolVar(&opts.ProbeTrack, "probe-track", false, "Run the track probe during embedded release verify checks")
	fs.StringVar(&opts.VitalsGateRaw, "vitals-gate", "", "Comma-separated vitals thresholds (for example: crashRate<2.0,anrRate<0.5)")
	fs.DurationVar(&opts.VitalsWait, "vitals-wait", 0, "Planned post-release vitals monitoring duration")
	fs.BoolVar(&opts.AutoHaltOnRegression, "auto-halt-on-regression", false, "Plan to halt an in-progress rollout if monitored vitals cross the configured thresholds")
	fs.BoolVar(&opts.Strict, "strict", false, "Treat warnings as blocking")
	fs.StringVar(&opts.Output, "output", "", "Output format: json or table")

	return &ffcli.Command{
		Name:      "rehearse",
		ShortHelp: "Compose a read-only release readiness rehearsal",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			resolved, manifest, rawManifest, err := validateRehearseOptions(opts)
			if err != nil {
				return err
			}

			result, runErr := runRehearse(ctx, deps, resolved, manifest, rawManifest)
			if writeErr := writeRehearseResult(deps.Stdout, shared.ResolveOutput(resolved.Output), result); writeErr != nil {
				return writeErr
			}
			return runErr
		},
	}
}

func validateRehearseOptions(opts rehearseOptions) (rehearseOptions, fullManifest, rawFullManifest, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return rehearseOptions{}, fullManifest{}, rawFullManifest{}, err
	}
	opts.PackageName = pkg

	opts.ManifestPath, err = shared.ResolveProjectPath(opts.ManifestPath, func(cfg config.ProjectConfig) string { return cfg.ReleaseManifest })
	if err != nil {
		return rehearseOptions{}, fullManifest{}, rawFullManifest{}, err
	}
	opts.ManifestPath = strings.TrimSpace(opts.ManifestPath)
	if opts.ManifestPath == "" {
		return rehearseOptions{}, fullManifest{}, rawFullManifest{}, shared.UsageErrorf("--manifest is required")
	}
	if opts.VitalsWait < 0 {
		return rehearseOptions{}, fullManifest{}, rawFullManifest{}, shared.UsageErrorf("--vitals-wait must be greater than or equal to zero")
	}
	if opts.AutoHaltOnRegression && strings.TrimSpace(opts.VitalsGateRaw) == "" {
		return rehearseOptions{}, fullManifest{}, rawFullManifest{}, shared.UsageErrorf("--auto-halt-on-regression requires --vitals-gate")
	}
	if opts.AutoHaltOnRegression && opts.VitalsWait <= 0 {
		return rehearseOptions{}, fullManifest{}, rawFullManifest{}, shared.UsageErrorf("--auto-halt-on-regression requires --vitals-wait")
	}
	if strings.TrimSpace(opts.VitalsGateRaw) != "" {
		conditions, err := parseVitalsGate(opts.VitalsGateRaw)
		if err != nil {
			return rehearseOptions{}, fullManifest{}, rawFullManifest{}, shared.UsageErrorf("%v", err)
		}
		opts.VitalsGate = conditions
	}
	switch shared.ResolveOutput(opts.Output) {
	case "json", "table":
	default:
		return rehearseOptions{}, fullManifest{}, rawFullManifest{}, shared.UsageErrorf("output must be json or table")
	}

	var raw rawFullManifest
	if err := shared.LoadManifest(opts.ManifestPath, &raw); err != nil {
		return rehearseOptions{}, fullManifest{}, rawFullManifest{}, err
	}
	baseDir := filepath.Dir(opts.ManifestPath)
	raw.ArtifactPath = resolveManifestPath(baseDir, raw.ArtifactPath)
	raw.MappingFile = resolveManifestPath(baseDir, raw.MappingFile)
	raw.NotesFile = resolveManifestPath(baseDir, raw.NotesFile)

	manifest, err := normalizeFullManifest(raw)
	if err != nil {
		return rehearseOptions{}, fullManifest{}, rawFullManifest{}, err
	}
	return opts, manifest, raw, nil
}

func runRehearse(ctx context.Context, deps Deps, opts rehearseOptions, manifest fullManifest, rawManifest rawFullManifest) (rehearseResult, error) {
	result := rehearseResult{
		PackageName:            opts.PackageName,
		Manifest:               opts.ManifestPath,
		Track:                  manifest.Track,
		Status:                 "blocked",
		PlannedSteps:           plannedRehearseSteps(string(shared.PackageReadinessReady), opts),
		RecommendedNextCommand: shared.RecommendedReleaseCommand(opts.PackageName, string(shared.PackageReadinessReady), filepath.Dir(opts.ManifestPath), opts.ManifestPath),
	}

	authSnapshot := shared.BuildAuthStatusSnapshot(loadConfigOrZero(deps.LoadConfig), deps.LookupEnv)
	result.AuthHealth = authSnapshot.Health
	if !authSnapshot.Authenticated {
		blocking := authSnapshot.HealthDetail
		if strings.TrimSpace(blocking) == "" {
			blocking = "no locally valid service-account credential is configured for the selected profile"
		}
		result.BlockingIssues = append(result.BlockingIssues, blocking)
		if strings.TrimSpace(authSnapshot.FixCommand) != "" {
			result.NextSteps = appendDedup(result.NextSteps, authSnapshot.FixCommand)
		}
	}
	result.Warnings = appendDedup(result.Warnings, authSnapshot.Warnings...)

	verifyOpts := verifyOptions{
		PackageName: opts.PackageName,
		Track:       manifest.Track,
		AABPath:     manifest.ArtifactPath,
		ProbeTrack:  opts.ProbeTrack,
		NotesMode:   rehearseNotesMode(rawManifest),
		NotesFile:   rawManifest.NotesFile,
		NotesLocale: rawManifest.NotesLocale,
		NotesText:   rawManifest.NotesText,
	}
	verifyRes, _ := runVerify(ctx, deps, verifyOpts)
	cleanedVerify := sanitizeVerifyForRehearse(verifyRes, rawManifest, opts)
	result.Verify = &cleanedVerify
	result.PackageReadiness = cleanedVerify.PackageReadiness
	result.PlannedSteps = plannedRehearseSteps(result.PackageReadiness, opts)
	result.RecommendedNextCommand = shared.RecommendedReleaseCommand(opts.PackageName, result.PackageReadiness, filepath.Dir(opts.ManifestPath), opts.ManifestPath)
	result.BlockingIssues = appendDedup(result.BlockingIssues, cleanedVerify.BlockingIssues...)
	result.Warnings = appendDedup(result.Warnings, cleanedVerify.Warnings...)

	client, requestCtx, cancel, clientErr := buildClient(ctx, deps)
	if clientErr == nil {
		defer cancel()
		readiness, readinessErr := shared.DetectPackageReadiness(requestCtx, client, opts.PackageName)
		if readinessErr != nil {
			result.BlockingIssues = appendDedup(result.BlockingIssues, readinessErr.Error())
		} else {
			result.PackageReadiness = string(readiness.Status)
			result.PlannedSteps = plannedRehearseSteps(result.PackageReadiness, opts)
			result.RecommendedNextCommand = shared.RecommendedReleaseCommand(opts.PackageName, result.PackageReadiness, filepath.Dir(opts.ManifestPath), opts.ManifestPath)
			if readiness.Warning != "" {
				result.Warnings = appendDedup(result.Warnings, readiness.Warning)
			}
			if readiness.NextStep != "" {
				result.NextSteps = appendDedup(result.NextSteps, readiness.NextStep)
			}
			if readiness.Status != shared.PackageReadinessReady {
				result.Warnings = appendDedup(result.Warnings, readiness.Detail)
			}
			if readiness.Status == shared.PackageReadinessUninitialized {
				result.BlockingIssues = appendDedup(result.BlockingIssues, readiness.Detail)
			}
		}

		if result.PackageReadiness != string(shared.PackageReadinessUninitialized) {
			if bootstrap, err := shared.DetectBootstrapDraftState(requestCtx, client, opts.PackageName); err == nil {
				result.BootstrapDraftExists = bootstrap.Exists
				result.BootstrapVersionCodes = append([]int64(nil), bootstrap.VersionCodes...)
			}
			if preview, err := runRehearseTrackPreview(ctx, requestCtx, client, opts.PackageName, manifest); err != nil {
				result.Warnings = appendDedup(result.Warnings, fmt.Sprintf("track preview unavailable: %v", err))
			} else {
				result.TrackPreview = preview
			}
		}
	} else {
		result.BlockingIssues = appendDedup(result.BlockingIssues, clientErr.Error())
	}

	if len(opts.VitalsGate) > 0 {
		gate := &fullVitalsGateResult{}
		reportingClient, reportingCtx, reportingCancel, err := buildReportingClient(ctx, deps)
		if err != nil {
			gate.Status = "unavailable"
			result.Warnings = appendDedup(result.Warnings, fmt.Sprintf("vitals gate unavailable: %v", err))
		} else {
			defer reportingCancel()
			checks, evalErr := evaluateVitalsGate(reportingCtx, reportingClient, opts.PackageName, deps.Now(), opts.VitalsGate)
			switch {
			case evalErr != nil:
				gate.Status = "unavailable"
				result.Warnings = appendDedup(result.Warnings, fmt.Sprintf("vitals gate unavailable: %v", evalErr))
			case rehearseVitalsChecksPassed(checks):
				gate.Status = "passed"
				gate.Checks = checks
			default:
				gate.Status = "blocked"
				gate.Checks = checks
				result.BlockingIssues = appendDedup(result.BlockingIssues, "current vitals exceed the requested thresholds")
				result.NextSteps = appendDedup(result.NextSteps, "Investigate current crash/ANR trends or relax `--vitals-gate` before releasing.")
			}
		}
		result.VitalsGate = gate
	}

	if cleanedVerify.Status != "ok" {
		result.NextSteps = appendDedup(result.NextSteps, fmt.Sprintf("Run `gpc release verify --package-name %s --track %s --aab %s` for the full verification report.", opts.PackageName, manifest.Track, manifest.ArtifactPath))
	}
	if result.PackageReadiness == string(shared.PackageReadinessReady) && len(result.BlockingIssues) == 0 {
		result.NextSteps = appendDedup(result.NextSteps, fmt.Sprintf("Run `gpc release full --manifest %s --confirm` when you are ready to release.", opts.ManifestPath))
	}

	if opts.Strict && len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			result.BlockingIssues = appendDedup(result.BlockingIssues, "strict mode: "+warning)
		}
	}

	if len(result.BlockingIssues) == 0 {
		result.Status = "ready"
		return result, nil
	}
	result.Status = "blocked"
	return result, fmt.Errorf("release rehearsal found blocking issues")
}

func loadConfigOrZero(load func() (config.Config, error)) config.Config {
	if load == nil {
		return config.Config{}
	}
	cfg, err := load()
	if err != nil {
		return config.Config{}
	}
	return cfg
}

func plannedRehearseSteps(readiness string, opts rehearseOptions) []string {
	steps := plannedFullSteps(readiness)
	if opts.VitalsWait > 0 {
		steps = append(steps, "monitor_vitals")
	}
	if opts.AutoHaltOnRegression {
		steps = append(steps, "auto_halt_on_regression")
	}
	return steps
}

func rehearseNotesMode(raw rawFullManifest) string {
	switch {
	case strings.TrimSpace(raw.NotesFile) != "":
		return "file"
	case strings.TrimSpace(raw.NotesText) != "":
		return "git"
	case len(raw.ReleaseNotes) > 0:
		return "none"
	default:
		return "none"
	}
}

func sanitizeVerifyForRehearse(result verifyResult, raw rawFullManifest, opts rehearseOptions) verifyResult {
	filteredChecks := make([]verifyCheck, 0, len(result.Checks))
	for _, check := range result.Checks {
		if check.Name == "track_probe" && !opts.ProbeTrack {
			continue
		}
		if check.Name == "notes_source" && len(raw.ReleaseNotes) > 0 {
			continue
		}
		filteredChecks = append(filteredChecks, check)
	}
	result.Checks = filteredChecks
	if !opts.ProbeTrack {
		result.Warnings = filterString(result.Warnings, "skipped (enable --probe-track to run)")
	}
	if len(raw.ReleaseNotes) > 0 {
		result.Warnings = filterString(result.Warnings, "release notes disabled")
	}
	result.Status = "ok"
	if len(result.BlockingIssues) > 0 {
		result.Status = "failed"
	}
	return result
}

func filterString(values []string, match string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.Contains(value, match) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func runRehearseTrackPreview(parentCtx, requestCtx context.Context, client Client, packageName string, manifest fullManifest) (*rehearseTrackPreview, error) {
	preview := &rehearseTrackPreview{Changes: []rehearseTrackChange{}}
	edit, err := client.CreateEdit(requestCtx, packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to create edit: %w", err)
	}
	defer func() {
		_ = deleteRehearseEdit(parentCtx, client, packageName, edit.ID)
	}()

	tracks, err := client.ListTracks(requestCtx, packageName, edit.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", err)
	}

	var current *gpc.TrackInfo
	for i := range tracks {
		if strings.TrimSpace(tracks[i].Name) == manifest.Track {
			current = &tracks[i]
			break
		}
	}
	if current == nil {
		preview.HasDiff = true
		preview.Changes = []rehearseTrackChange{{Action: "create_track", Desired: manifest.Track}}
		preview.ChangeCount = len(preview.Changes)
		return preview, nil
	}

	preview.TrackFound = true
	if len(current.Releases) == 0 {
		preview.HasDiff = true
		preview.Changes = []rehearseTrackChange{{Action: "create_release", Desired: plannedReleaseSummary(manifest)}}
		preview.ChangeCount = len(preview.Changes)
		return preview, nil
	}

	preview.ReleaseFound = true
	live := current.Releases[0]
	preview.ReleaseName = live.Name
	changes := compareRehearseTrack(live, manifest)
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Field != changes[j].Field {
			return changes[i].Field < changes[j].Field
		}
		return changes[i].Action < changes[j].Action
	})
	preview.Changes = changes
	preview.ChangeCount = len(changes)
	preview.HasDiff = len(changes) > 0
	return preview, nil
}

func deleteRehearseEdit(ctx context.Context, client Client, packageName, editID string) error {
	cleanupCtx, cleanupCancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
	defer cleanupCancel()
	if err := client.DeleteEdit(cleanupCtx, packageName, editID); err != nil {
		return fmt.Errorf("failed to delete edit: %w", err)
	}
	return nil
}

func compareRehearseTrack(live gpc.TrackReleaseInfo, manifest fullManifest) []rehearseTrackChange {
	changes := make([]rehearseTrackChange, 0, 5)
	if live.Status != manifest.Status {
		changes = append(changes, rehearseTrackChange{Field: "status", Action: "update", Live: live.Status, Desired: manifest.Status})
	}
	if live.Name != manifest.ReleaseName {
		changes = append(changes, rehearseTrackChange{Field: "releaseName", Action: "update", Live: live.Name, Desired: manifest.ReleaseName})
	}
	if !equalManifestFloat(live.UserFraction, manifest.UserFraction) {
		changes = append(changes, rehearseTrackChange{Field: "userFraction", Action: "update", Live: manifestFloatOrNil(live.UserFraction, live.UserFraction > 0), Desired: manifestFloatOrNil(manifest.UserFraction, manifest.UserFraction > 0)})
	}
	if live.UpdatePriority != manifest.UpdatePriority {
		changes = append(changes, rehearseTrackChange{Field: "updatePriority", Action: "update", Live: manifestIntOrNil(live.UpdatePriority, live.UpdatePriority > 0), Desired: manifestIntOrNil(manifest.UpdatePriority, manifest.UpdatePriority > 0)})
	}
	liveNotes := normalizeManifestNotes(live.ReleaseNotes)
	desiredNotes := normalizeManifestNotes(manifest.ReleaseNotes)
	if !reflect.DeepEqual(liveNotes, desiredNotes) {
		changes = append(changes, rehearseTrackChange{Field: "releaseNotes", Action: "update", Live: manifestSliceOrNil(liveNotes), Desired: manifestSliceOrNil(desiredNotes)})
	}
	return changes
}

func normalizeManifestNotes(values []gpc.LocalizedText) []gpc.LocalizedText {
	out := append([]gpc.LocalizedText(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Language != out[j].Language {
			return out[i].Language < out[j].Language
		}
		return out[i].Text < out[j].Text
	})
	return out
}

func equalManifestFloat(live, desired float64) bool {
	if desired < 0 {
		return live == 0
	}
	return live == desired
}

func manifestFloatOrNil(value float64, set bool) *float64 {
	if !set {
		return nil
	}
	return &value
}

func manifestIntOrNil(value int64, set bool) *int64 {
	if !set {
		return nil
	}
	return &value
}

func manifestSliceOrNil[T any](values []T) any {
	if len(values) == 0 {
		return nil
	}
	return values
}

func plannedReleaseSummary(manifest fullManifest) map[string]any {
	return map[string]any{
		"status":         manifest.Status,
		"releaseName":    manifest.ReleaseName,
		"userFraction":   manifestFloatOrNil(manifest.UserFraction, manifest.UserFraction > 0),
		"updatePriority": manifestIntOrNil(manifest.UpdatePriority, manifest.UpdatePriority > 0),
		"releaseNotes":   manifestSliceOrNil(normalizeManifestNotes(manifest.ReleaseNotes)),
	}
}

func rehearseVitalsChecksPassed(checks []fullVitalsGateCheck) bool {
	for _, check := range checks {
		if !check.Passed {
			return false
		}
	}
	return true
}

func appendDedup(base []string, values ...string) []string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		exists := false
		for _, existing := range base {
			if existing == value {
				exists = true
				break
			}
		}
		if !exists {
			base = append(base, value)
		}
	}
	return base
}

func writeRehearseResult(out io.Writer, output string, result rehearseResult) error {
	switch output {
	case "json":
		return shared.WriteJSON(out, result)
	case "table":
		return writeRehearseTable(out, result)
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}

func writeRehearseTable(out io.Writer, result rehearseResult) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", result.PackageName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "MANIFEST\t%s\n", result.Manifest); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "TRACK\t%s\n", result.Track); err != nil {
		return err
	}
	if result.AuthHealth != "" {
		if _, err := fmt.Fprintf(out, "AUTH_HEALTH\t%s\n", result.AuthHealth); err != nil {
			return err
		}
	}
	if result.PackageReadiness != "" {
		if _, err := fmt.Fprintf(out, "PACKAGE_READINESS\t%s\n", result.PackageReadiness); err != nil {
			return err
		}
	}
	if result.RecommendedNextCommand != "" {
		if _, err := fmt.Fprintf(out, "RECOMMENDED_NEXT_COMMAND\t%s\n", result.RecommendedNextCommand); err != nil {
			return err
		}
	}
	if result.TrackPreview != nil {
		if _, err := fmt.Fprintf(out, "TRACK_PREVIEW\ttrackFound=%t releaseFound=%t hasDiff=%t changeCount=%d\n", result.TrackPreview.TrackFound, result.TrackPreview.ReleaseFound, result.TrackPreview.HasDiff, result.TrackPreview.ChangeCount); err != nil {
			return err
		}
		if len(result.TrackPreview.Changes) > 0 {
			if _, err := fmt.Fprintln(out, "TRACK_FIELD\tACTION\tLIVE\tDESIRED"); err != nil {
				return err
			}
			for _, change := range result.TrackPreview.Changes {
				if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", change.Field, change.Action, formatRehearseValue(change.Live), formatRehearseValue(change.Desired)); err != nil {
					return err
				}
			}
		}
	}
	if result.VitalsGate != nil {
		if _, err := fmt.Fprintf(out, "VITALS_GATE\t%s\n", result.VitalsGate.Status); err != nil {
			return err
		}
		for _, check := range result.VitalsGate.Checks {
			if _, err := fmt.Fprintf(out, "vitalsCheck\t%s %s %s actual=%s passed=%t\n", check.Metric, check.Operator, rehearseFormatFloat(check.Threshold), rehearseFormatActual(check.Actual), check.Passed); err != nil {
				return err
			}
		}
	}
	for _, step := range result.PlannedSteps {
		if _, err := fmt.Fprintf(out, "plannedStep\t%s\n", step); err != nil {
			return err
		}
	}
	for _, warning := range result.Warnings {
		if _, err := fmt.Fprintf(out, "warning\t%s\n", warning); err != nil {
			return err
		}
	}
	for _, issue := range result.BlockingIssues {
		if _, err := fmt.Fprintf(out, "blockingIssue\t%s\n", issue); err != nil {
			return err
		}
	}
	for _, step := range result.NextSteps {
		if _, err := fmt.Fprintf(out, "nextStep\t%s\n", step); err != nil {
			return err
		}
	}
	return nil
}

func formatRehearseValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(raw)
	}
}

func rehearseFormatFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", value), "0"), ".")
}

func rehearseFormatActual(value *float64) string {
	if value == nil {
		return ""
	}
	return rehearseFormatFloat(*value)
}
