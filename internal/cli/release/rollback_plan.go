package release

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type rollbackPlanOptions struct {
	PackageName string
	Track       string
	VitalsGate  []vitalsGateCondition
	VitalsRaw   string
	Output      string
}

type rollbackPlanRelease struct {
	Name           string  `json:"name,omitempty"`
	Status         string  `json:"status,omitempty"`
	UserFraction   float64 `json:"userFraction,omitempty"`
	VersionCodes   []int64 `json:"versionCodes,omitempty"`
	UpdatePriority int64   `json:"updatePriority,omitempty"`
}

type rollbackPlanResult struct {
	PackageName        string                `json:"packageName"`
	Track              string                `json:"track"`
	Status             string                `json:"status"`
	PlanType           string                `json:"planType,omitempty"`
	RecommendedCommand string                `json:"recommendedCommand,omitempty"`
	ActiveRelease      *rollbackPlanRelease  `json:"activeRelease,omitempty"`
	RollbackTarget     *rollbackPlanRelease  `json:"rollbackTarget,omitempty"`
	PreviousCompleted  *rollbackPlanRelease  `json:"previousCompletedRelease,omitempty"`
	VitalsGate         *fullVitalsGateResult `json:"vitalsGate,omitempty"`
	Warnings           []string              `json:"warnings,omitempty"`
	BlockingIssues     []string              `json:"blockingIssues,omitempty"`
	CleanupPerformed   bool                  `json:"cleanupPerformed"`
}

func newRollbackCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "rollback",
		ShortHelp: "Read-only rollback planning for staged releases",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newRollbackPlanCommand(deps),
		},
	}
}

func newRollbackPlanCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts rollbackPlanOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Target package name")
	fs.StringVar(&opts.Track, "track", "", "Track name (e.g. production)")
	fs.StringVar(&opts.VitalsRaw, "vitals-gate", "", "Comma-separated vitals thresholds (for example: crashRate<2.0,anrRate<0.5)")
	fs.StringVar(&opts.Output, "output", "", "Output format: json or table")

	return &ffcli.Command{
		Name:      "plan",
		ShortHelp: "Inspect track state and suggest the next rollback action",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			resolved, err := validateRollbackPlanOptions(opts)
			if err != nil {
				return err
			}

			result, runErr := runRollbackPlan(ctx, deps, resolved)
			if writeErr := writeRollbackPlanResult(deps.Stdout, shared.ResolveOutput(resolved.Output), result); writeErr != nil {
				return writeErr
			}
			return runErr
		},
	}
}

func validateRollbackPlanOptions(opts rollbackPlanOptions) (rollbackPlanOptions, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return rollbackPlanOptions{}, err
	}

	opts.PackageName = pkg
	opts.Track = strings.TrimSpace(opts.Track)
	if opts.Track == "" {
		return rollbackPlanOptions{}, shared.UsageErrorf("--track is required")
	}
	if strings.TrimSpace(opts.VitalsRaw) != "" {
		conditions, err := parseVitalsGate(opts.VitalsRaw)
		if err != nil {
			return rollbackPlanOptions{}, shared.UsageErrorf("%v", err)
		}
		opts.VitalsGate = conditions
	}
	switch shared.ResolveOutput(opts.Output) {
	case "json", "table":
	default:
		return rollbackPlanOptions{}, shared.UsageErrorf("output must be json or table")
	}
	return opts, nil
}

func runRollbackPlan(ctx context.Context, deps Deps, opts rollbackPlanOptions) (rollbackPlanResult, error) {
	result := rollbackPlanResult{
		PackageName: opts.PackageName,
		Track:       opts.Track,
		Status:      "failed",
		PlanType:    "manual_review_required",
	}

	client, requestCtx, cancel, err := buildClient(ctx, deps)
	if err != nil {
		result.BlockingIssues = append(result.BlockingIssues, err.Error())
		return result, err
	}
	defer cancel()

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		result.BlockingIssues = append(result.BlockingIssues, "failed to create inspection edit")
		return result, fmt.Errorf("failed to create inspection edit: %w", err)
	}

	cleanup := func() error {
		if err := client.DeleteEdit(requestCtx, opts.PackageName, edit.ID); err != nil {
			result.BlockingIssues = appendUniqueString(result.BlockingIssues, "failed to delete inspection edit")
			result.Status = "failed"
			return fmt.Errorf("failed to delete inspection edit: %w", err)
		}
		result.CleanupPerformed = true
		return nil
	}

	track, err := client.GetTrack(requestCtx, opts.PackageName, edit.ID, opts.Track)
	if err != nil {
		result.BlockingIssues = append(result.BlockingIssues, "failed to read track")
		cleanupErr := cleanup()
		if cleanupErr != nil {
			return result, fmt.Errorf("failed to read track: %w; cleanup: %v", err, cleanupErr)
		}
		return result, fmt.Errorf("failed to read track: %w", err)
	}

	if active, index, ok := primaryTrackRelease(track); ok {
		result.ActiveRelease = newRollbackPlanRelease(active)
		if previous, ok := previousCompletedRelease(track, index); ok {
			result.PreviousCompleted = newRollbackPlanRelease(previous)
		}
	}

	if target, err := shared.SelectRolloutToHalt(track); err != nil {
		result.BlockingIssues = append(result.BlockingIssues, err.Error())
	} else {
		result.PlanType = "halt_in_progress_rollout"
		result.RollbackTarget = newRollbackPlanRelease(target)
		result.RecommendedCommand = fmt.Sprintf("gpc rollback --package-name %s --track %s --confirm", opts.PackageName, opts.Track)
	}

	if len(opts.VitalsGate) > 0 {
		gate := &fullVitalsGateResult{}
		reportingClient, reportingCtx, reportingCancel, err := buildReportingClient(ctx, deps)
		if err != nil {
			gate.Status = "unavailable"
			result.Warnings = append(result.Warnings, fmt.Sprintf("vitals gate unavailable: %v", err))
		} else {
			defer reportingCancel()
			checks, evalErr := evaluateVitalsGate(reportingCtx, reportingClient, opts.PackageName, deps.Now(), opts.VitalsGate)
			switch {
			case evalErr != nil:
				gate.Status = "unavailable"
				result.Warnings = append(result.Warnings, fmt.Sprintf("vitals gate unavailable: %v", evalErr))
			case vitalsChecksPassed(checks):
				gate.Status = "passed"
				gate.Checks = checks
			default:
				gate.Status = "blocked"
				gate.Checks = checks
				result.Warnings = append(result.Warnings, "current vitals exceed the requested thresholds")
			}
		}
		result.VitalsGate = gate
	}

	if err := cleanup(); err != nil {
		return result, err
	}

	result.Status = "ok"
	if len(result.Warnings) > 0 || len(result.BlockingIssues) > 0 {
		result.Status = "warn"
	}
	return result, nil
}

func primaryTrackRelease(track gpc.TrackInfo) (gpc.TrackReleaseInfo, int, bool) {
	for i, release := range track.Releases {
		if len(release.VersionCodes) > 0 {
			return release, i, true
		}
	}
	if len(track.Releases) == 0 {
		return gpc.TrackReleaseInfo{}, -1, false
	}
	return track.Releases[0], 0, true
}

func previousCompletedRelease(track gpc.TrackInfo, activeIndex int) (gpc.TrackReleaseInfo, bool) {
	start := activeIndex + 1
	if start < 0 {
		start = 0
	}
	for i := start; i < len(track.Releases); i++ {
		release := track.Releases[i]
		if strings.TrimSpace(release.Status) == "completed" || release.UserFraction >= 1 {
			if len(release.VersionCodes) == 0 {
				continue
			}
			return release, true
		}
	}
	return gpc.TrackReleaseInfo{}, false
}

func newRollbackPlanRelease(release gpc.TrackReleaseInfo) *rollbackPlanRelease {
	return &rollbackPlanRelease{
		Name:           release.Name,
		Status:         release.Status,
		UserFraction:   release.UserFraction,
		VersionCodes:   append([]int64(nil), release.VersionCodes...),
		UpdatePriority: release.UpdatePriority,
	}
}

func vitalsChecksPassed(checks []fullVitalsGateCheck) bool {
	for _, check := range checks {
		if !check.Passed {
			return false
		}
	}
	return true
}

func writeRollbackPlanResult(out io.Writer, output string, result rollbackPlanResult) error {
	switch output {
	case "json":
		return shared.WriteJSON(out, result)
	case "table":
		return writeRollbackPlanTable(out, result)
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}

func writeRollbackPlanTable(out io.Writer, result rollbackPlanResult) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", result.PackageName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "TRACK\t%s\n", result.Track); err != nil {
		return err
	}
	if result.PlanType != "" {
		if _, err := fmt.Fprintf(out, "PLAN_TYPE\t%s\n", result.PlanType); err != nil {
			return err
		}
	}
	if result.RecommendedCommand != "" {
		if _, err := fmt.Fprintf(out, "RECOMMENDED_COMMAND\t%s\n", result.RecommendedCommand); err != nil {
			return err
		}
	}
	if result.ActiveRelease != nil {
		if _, err := fmt.Fprintf(out, "ACTIVE_RELEASE\t%s\n", formatRollbackPlanRelease(*result.ActiveRelease)); err != nil {
			return err
		}
	}
	if result.RollbackTarget != nil {
		if _, err := fmt.Fprintf(out, "ROLLBACK_TARGET\t%s\n", formatRollbackPlanRelease(*result.RollbackTarget)); err != nil {
			return err
		}
	}
	if result.PreviousCompleted != nil {
		if _, err := fmt.Fprintf(out, "PREVIOUS_COMPLETED_RELEASE\t%s\n", formatRollbackPlanRelease(*result.PreviousCompleted)); err != nil {
			return err
		}
	}
	if result.VitalsGate != nil {
		if _, err := fmt.Fprintf(out, "VITALS_GATE\t%s\n", result.VitalsGate.Status); err != nil {
			return err
		}
		if len(result.VitalsGate.Checks) > 0 {
			if _, err := fmt.Fprintln(out, "VITALS_CHECK\tOPERATOR\tTHRESHOLD\tACTUAL\tPASSED"); err != nil {
				return err
			}
			for _, check := range result.VitalsGate.Checks {
				if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%t\n", check.Metric, check.Operator, formatThreshold(check.Threshold), formatActual(check.Actual), check.Passed); err != nil {
					return err
				}
			}
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
	return nil
}

func formatRollbackPlanRelease(release rollbackPlanRelease) string {
	parts := []string{}
	if strings.TrimSpace(release.Name) != "" {
		parts = append(parts, "name="+release.Name)
	}
	if strings.TrimSpace(release.Status) != "" {
		parts = append(parts, "status="+release.Status)
	}
	if release.UserFraction > 0 {
		parts = append(parts, "userFraction="+formatThreshold(release.UserFraction))
	}
	if len(release.VersionCodes) > 0 {
		parts = append(parts, "versionCodes="+shared.JoinVersionCodes(release.VersionCodes))
	}
	if release.UpdatePriority > 0 {
		parts = append(parts, fmt.Sprintf("updatePriority=%d", release.UpdatePriority))
	}
	return strings.Join(parts, " ")
}

func formatThreshold(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", value), "0"), ".")
}

func formatActual(value *float64) string {
	if value == nil {
		return ""
	}
	return formatThreshold(*value)
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
