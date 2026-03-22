package release

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"gopkg.in/yaml.v3"
)

type fullStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type fullResult struct {
	PackageName          string                `json:"packageName"`
	PackageReadiness     string                `json:"packageReadiness,omitempty"`
	Manifest             string                `json:"manifest"`
	Track                string                `json:"track"`
	Status               string                `json:"status"`
	UploadedArtifactType string                `json:"uploadedArtifactType,omitempty"`
	VersionCode          int64                 `json:"versionCode,omitempty"`
	Committed            bool                  `json:"committed"`
	CleanupPerformed     bool                  `json:"cleanupPerformed"`
	VitalsGate           *fullVitalsGateResult `json:"vitalsGate,omitempty"`
	Warnings             []string              `json:"warnings,omitempty"`
	NextSteps            []string              `json:"nextSteps,omitempty"`
	Steps                []fullStep            `json:"steps"`
}

type fullOptions struct {
	PackageName          string
	ManifestPath         string
	Confirm              bool
	DryRun               bool
	AllowProduction      bool
	VitalsGateRaw        string
	VitalsGate           []vitalsGateCondition
	VitalsWait           time.Duration
	AutoHaltOnRegression bool
}

type bootstrapDraftState struct {
	Exists       bool
	TrackName    string
	VersionCodes []int64
}

const (
	fullStatusBootstrapCommitted = "bootstrap_committed"
)

func newFullCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("full", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts fullOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Target package name")
	fs.StringVar(&opts.ManifestPath, "manifest", "", "Path to release manifest (.json/.yaml/.yml)")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm committing release (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Run all steps but delete edit instead of committing")
	fs.BoolVar(&opts.AllowProduction, "allow-production", false, "Allow track=production")
	fs.StringVar(&opts.VitalsGateRaw, "vitals-gate", "", "Comma-separated vitals thresholds (for example: crashRate<2.0,anrRate<0.5)")
	fs.DurationVar(&opts.VitalsWait, "vitals-wait", 0, "Monitor vitals after commit for the given duration")
	fs.BoolVar(&opts.AutoHaltOnRegression, "auto-halt-on-regression", false, "Halt an in-progress rollout if monitored vitals cross the configured thresholds")

	return &ffcli.Command{
		Name:      "full",
		ShortHelp: "Deploy a release from a YAML or JSON manifest",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, manifest, err := validateFullOptions(opts)
			if err != nil {
				return err
			}

			client, requestCtx, cancel, err := buildClient(ctx, deps)
			if err != nil {
				return err
			}
			defer cancel()

			spinner := shared.NewSpinner(deps.Stderr, "Running full release flow")
			err = runFull(ctx, requestCtx, client, deps, deps.Stdout, opts, manifest)
			if err != nil {
				spinner.Fail("Full release flow failed")
				return err
			}
			spinner.Success("Full release flow finished")
			return nil
		},
	}
}

func validateFullOptions(opts fullOptions) (fullOptions, fullManifest, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return fullOptions{}, fullManifest{}, err
	}
	opts.PackageName = pkg
	opts.ManifestPath, err = shared.ResolveProjectPath(opts.ManifestPath, func(cfg config.ProjectConfig) string { return cfg.ReleaseManifest })
	if err != nil {
		return fullOptions{}, fullManifest{}, err
	}
	opts.ManifestPath = strings.TrimSpace(opts.ManifestPath)
	if opts.ManifestPath == "" {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--manifest is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}
	if opts.DryRun && strings.TrimSpace(opts.VitalsGateRaw) != "" {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--vitals-gate cannot be used with --dry-run")
	}
	if opts.VitalsWait < 0 {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--vitals-wait must be greater than or equal to zero")
	}
	if opts.AutoHaltOnRegression && strings.TrimSpace(opts.VitalsGateRaw) == "" {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--auto-halt-on-regression requires --vitals-gate")
	}
	if opts.AutoHaltOnRegression && opts.VitalsWait <= 0 {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--auto-halt-on-regression requires --vitals-wait")
	}
	if strings.TrimSpace(opts.VitalsGateRaw) != "" {
		conditions, err := parseVitalsGate(opts.VitalsGateRaw)
		if err != nil {
			return fullOptions{}, fullManifest{}, shared.UsageErrorf("%v", err)
		}
		opts.VitalsGate = conditions
	}

	manifest, err := loadFullManifest(opts.ManifestPath)
	if err != nil {
		return fullOptions{}, fullManifest{}, err
	}
	if manifest.Track == "production" && !opts.AllowProduction {
		return fullOptions{}, fullManifest{}, shared.UsageErrorf("--allow-production is required when --track=production")
	}
	return opts, manifest, nil
}

func runFull(parentCtx, requestCtx context.Context, client Client, deps Deps, out io.Writer, opts fullOptions, manifest fullManifest) error {
	result := fullResult{
		PackageName: opts.PackageName,
		Manifest:    opts.ManifestPath,
		Track:       manifest.Track,
		Status:      "failed",
		Steps:       make([]fullStep, 0, 16),
	}

	verify, _ := runVerify(parentCtx, deps, verifyOptions{
		PackageName: opts.PackageName,
		Track:       manifest.Track,
		ProjectDir:  "",
		BuildTask:   "",
		AABPath:     manifest.ArtifactPath,
		ProbeTrack:  true,
		NotesMode:   "none",
		NotesFile:   "",
	})
	result.PackageReadiness = verify.PackageReadiness
	if verify.Status != "ok" && verify.PackageReadiness != string(shared.PackageReadinessDraftBootstrapRequired) {
		result.Steps = append(result.Steps, fullStep{Name: "preflight_verify", Status: "error", Error: "release verification failed"})
		result.NextSteps = append(result.NextSteps, "Run `gpc release verify --package-name "+opts.PackageName+" --track "+manifest.Track+" --aab "+manifest.ArtifactPath+"` or `gpc doctor --package-name "+opts.PackageName+"` for details.")
		_ = shared.WriteJSON(out, result)
		return fmt.Errorf("release verification failed")
	}
	result.Steps = append(result.Steps, fullStep{Name: "preflight_verify", Status: "ok"})

	if opts.DryRun {
		result.Status = "dry-run"
		for _, step := range plannedFullSteps(result.PackageReadiness) {
			result.Steps = append(result.Steps, fullStep{Name: step, Status: "planned"})
		}
		return shared.WriteJSON(out, result)
	}

	if result.PackageReadiness == string(shared.PackageReadinessUninitialized) {
		result.Status = "manual_required"
		result.NextSteps = append(result.NextSteps, "Run `gpc release init --package-name "+opts.PackageName+" --dir ./play` to generate the first-upload bridge.")
		result.Steps = append(result.Steps, fullStep{Name: "package_readiness", Status: "error", Error: "package is not initialized in Google Play yet"})
		_ = shared.WriteJSON(out, result)
		return fmt.Errorf("package is not initialized in Google Play yet")
	}

	if result.PackageReadiness == string(shared.PackageReadinessDraftBootstrapRequired) {
		existingDraft, err := detectExistingBootstrapDraft(requestCtx, client, opts.PackageName)
		if err != nil {
			result.Steps = append(result.Steps, fullStep{Name: "bootstrap_existing_draft", Status: "error", Error: err.Error()})
			_ = shared.WriteJSON(out, result)
			return err
		}
		if existingDraft.Exists {
			result.Steps = append(result.Steps, fullStep{Name: "bootstrap_existing_draft", Status: "ok"})
			writeFullProgress(deps.Stderr, "bootstrap_release", "reusing existing internal draft bootstrap release")
			if len(existingDraft.VersionCodes) > 0 {
				result.NextSteps = append(result.NextSteps, "Existing internal draft release already uses versionCode(s) "+joinVersionCodes(existingDraft.VersionCodes)+".")
			}
		} else {
			writeFullProgress(deps.Stderr, "bootstrap_release", "uploading internal draft bootstrap release")
			bootstrapManifest := manifest
			bootstrapManifest.Track = "internal"
			bootstrapManifest.Status = "draft"
			if err := runManifestDeploy(parentCtx, requestCtx, client, deps, opts, bootstrapManifest, &result, "bootstrap_"); err != nil {
				_ = shared.WriteJSON(out, result)
				return err
			}
			result.Steps = append(result.Steps, fullStep{Name: "bootstrap_release", Status: "ok"})
		}
		writeFullProgress(deps.Stderr, "post_bootstrap_recheck", "waiting for Play to leave draft bootstrap state")
		bootstrapReadiness, err := waitForReadyAfterBootstrap(requestCtx, deps, client, opts.PackageName)
		if err != nil {
			result.Steps = append(result.Steps, fullStep{Name: "post_bootstrap_recheck", Status: "error", Error: err.Error()})
			_ = shared.WriteJSON(out, result)
			return err
		}
		result.PackageReadiness = string(bootstrapReadiness.Status)
		if bootstrapReadiness.Status != shared.PackageReadinessReady {
			result.Status = fullStatusBootstrapCommitted
			result.Steps = append(result.Steps, fullStep{Name: "post_bootstrap_recheck", Status: "warning", Error: bootstrapReadiness.Detail})
			if bootstrapReadiness.Warning != "" {
				result.Warnings = append(result.Warnings, bootstrapReadiness.Warning)
			}
			result.NextSteps = append(result.NextSteps, "The draft bootstrap release was committed. Wait for Play to finish processing it, then rerun `gpc release full --manifest "+opts.ManifestPath+" --confirm`.")
			_ = shared.WriteJSON(out, result)
			return fmt.Errorf("bootstrap release committed, but package is still in Play's draft bootstrap state")
		}
		result.Steps = append(result.Steps, fullStep{Name: "post_bootstrap_recheck", Status: "ok"})
	}

	writeFullProgress(deps.Stderr, "content_sync", "applying app details, listing, screenshots, products, and subscriptions")
	if err := runContentSync(parentCtx, deps, opts.PackageName, opts.ManifestPath, &result); err != nil {
		_ = shared.WriteJSON(out, result)
		return err
	}

	writeFullProgress(deps.Stderr, "final_deploy", "uploading the requested track release")
	if err := runManifestDeploy(parentCtx, requestCtx, client, deps, opts, manifest, &result, "deploy_"); err != nil {
		_ = shared.WriteJSON(out, result)
		return err
	}

	writeFullProgress(deps.Stderr, "post_release_checks", "running post-release verification")
	if err := runPostReleaseChecks(requestCtx, client, deps, opts.PackageName, manifest.Track, &result); err != nil {
		result.Warnings = append(result.Warnings, err.Error())
	}
	if len(opts.VitalsGate) > 0 && opts.VitalsWait > 0 {
		reportingClient, _, reportingCancel, err := buildReportingClient(parentCtx, deps)
		if err != nil {
			result.Status = "failed"
			result.Steps = append(result.Steps, fullStep{Name: "create_reporting_client", Status: "error", Error: err.Error()})
			_ = shared.WriteJSON(out, result)
			return err
		}
		defer reportingCancel()
		if err := monitorVitalsGate(parentCtx, deps, client, reportingClient, &result, opts, manifest, result.VersionCode); err != nil {
			_ = shared.WriteJSON(out, result)
			return err
		}
	}
	return shared.WriteJSON(out, result)
}

func plannedFullSteps(readiness string) []string {
	steps := []string{"preflight_verify"}
	if readiness == string(shared.PackageReadinessDraftBootstrapRequired) {
		steps = append(steps, "bootstrap_existing_draft", "bootstrap_release", "post_bootstrap_recheck")
	}
	return append(steps, "sync_app_details_listing", "sync_screenshots", "sync_products", "sync_subscriptions", "deploy_release", "post_release_checks")
}

func waitForReadyAfterBootstrap(ctx context.Context, deps Deps, client Client, packageName string) (shared.PackageReadinessResult, error) {
	const attempts = 8
	for attempt := 0; attempt < attempts; attempt++ {
		readiness, err := shared.DetectPackageReadiness(ctx, client, packageName)
		if err != nil {
			return shared.PackageReadinessResult{}, err
		}
		if readiness.Status == shared.PackageReadinessReady {
			return readiness, nil
		}
		if attempt == attempts-1 {
			return readiness, nil
		}
		if err := deps.Sleep(ctx, 5*time.Second); err != nil {
			return shared.PackageReadinessResult{}, err
		}
	}
	return shared.PackageReadinessResult{}, nil
}

func runContentSync(ctx context.Context, deps Deps, packageName, manifestPath string, result *fullResult) error {
	appInitManifest, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.AppInitManifest })
	if err != nil {
		return err
	}
	if strings.TrimSpace(appInitManifest) != "" {
		listingManifest, err := writeFilteredAppInitManifest(appInitManifest, map[string]bool{"appDetails": true, "listing": true})
		if err != nil {
			return err
		}
		if listingManifest != "" {
			defer os.Remove(listingManifest)
			if err := deps.RunAppInit(ctx, []string{"--package-name", packageName, "--manifest", listingManifest, "--confirm"}); err != nil {
				result.Steps = append(result.Steps, fullStep{Name: "sync_app_details_listing", Status: "error", Error: err.Error()})
				return err
			}
			result.Steps = append(result.Steps, fullStep{Name: "sync_app_details_listing", Status: "ok"})
		} else {
			result.Steps = append(result.Steps, fullStep{Name: "sync_app_details_listing", Status: "skipped"})
		}
	} else {
		result.Steps = append(result.Steps, fullStep{Name: "sync_app_details_listing", Status: "skipped"})
	}

	screenshotsDir, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.ScreenshotsDir })
	if err != nil {
		return err
	}
	if dirHasFiles(screenshotsDir) {
		if err := deps.RunScreenshotsSync(ctx, []string{"--package-name", packageName, "--dir", screenshotsDir, "--confirm"}); err != nil {
			result.Steps = append(result.Steps, fullStep{Name: "sync_screenshots", Status: "error", Error: err.Error()})
			return err
		}
		result.Steps = append(result.Steps, fullStep{Name: "sync_screenshots", Status: "ok"})
	} else {
		result.Steps = append(result.Steps, fullStep{Name: "sync_screenshots", Status: "skipped"})
	}

	productsDir, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.ProductsDir })
	if err != nil {
		return err
	}
	if dirHasFiles(productsDir) {
		if err := deps.RunProductsSync(ctx, []string{"--package-name", packageName, "--dir", productsDir, "--confirm"}); err != nil {
			result.Steps = append(result.Steps, fullStep{Name: "sync_products", Status: "error", Error: err.Error()})
			return err
		}
		result.Steps = append(result.Steps, fullStep{Name: "sync_products", Status: "ok"})
	} else {
		result.Steps = append(result.Steps, fullStep{Name: "sync_products", Status: "skipped"})
	}

	subscriptionsDir, err := shared.ResolveProjectPath("", func(cfg config.ProjectConfig) string { return cfg.SubscriptionsDir })
	if err != nil {
		return err
	}
	if dirHasFiles(subscriptionsDir) {
		if err := deps.RunSubscriptionsSync(ctx, []string{"--package-name", packageName, "--dir", subscriptionsDir, "--confirm"}); err != nil {
			result.Steps = append(result.Steps, fullStep{Name: "sync_subscriptions", Status: "error", Error: err.Error()})
			return err
		}
		result.Steps = append(result.Steps, fullStep{Name: "sync_subscriptions", Status: "ok"})
	} else {
		result.Steps = append(result.Steps, fullStep{Name: "sync_subscriptions", Status: "skipped"})
	}

	_ = manifestPath
	return nil
}

func runManifestDeploy(parentCtx, requestCtx context.Context, client Client, deps Deps, opts fullOptions, manifest fullManifest, result *fullResult, prefix string) error {
	var currentEditID string
	fail := func(step string, err error) error {
		result.Steps = append(result.Steps, fullStep{Name: prefix + step, Status: "error", Error: err.Error()})
		if currentEditID != "" {
			cleanupCtx, cleanupCancel := shared.ContextWithTimeout(parentCtx, shared.ActiveGlobalFlags().Timeout)
			cleanupErr := client.DeleteEdit(cleanupCtx, opts.PackageName, currentEditID)
			cleanupCancel()
			if cleanupErr == nil {
				result.CleanupPerformed = true
				result.Steps = append(result.Steps, fullStep{Name: prefix + "cleanup_delete_edit", Status: "ok"})
			}
		}
		return err
	}

	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return fail("create_edit", fmt.Errorf("failed to create edit: %w", err))
	}
	currentEditID = edit.ID
	result.Steps = append(result.Steps, fullStep{Name: prefix + "create_edit", Status: "ok"})

	writeFullProgress(deps.Stderr, prefix+"upload_artifact", "uploading artifact")
	uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(parentCtx, shared.ActiveGlobalFlags().UploadTimeout)
	versionCode, err := uploadManifestArtifact(uploadCtx, client, opts.PackageName, currentEditID, manifest)
	uploadCancel()
	if err != nil {
		return fail("upload_artifact", normalizeUploadError(err))
	}
	result.VersionCode = versionCode
	result.UploadedArtifactType = manifest.ArtifactType
	result.Steps = append(result.Steps, fullStep{Name: prefix + "upload_artifact", Status: "ok"})

	if manifest.MappingFile != "" {
		writeFullProgress(deps.Stderr, prefix+"upload_mapping", "uploading mapping file")
		mapCtx, mapCancel := shared.ContextWithUploadTimeout(parentCtx, shared.ActiveGlobalFlags().UploadTimeout)
		_, err := client.UploadDeobfuscationFile(mapCtx, opts.PackageName, currentEditID, versionCode, manifest.MappingType, manifest.MappingFile)
		mapCancel()
		if err != nil {
			return fail("upload_mapping", fmt.Errorf("failed to upload mapping file: %w", err))
		}
		result.Steps = append(result.Steps, fullStep{Name: prefix + "upload_mapping", Status: "ok"})
	}

	_, err = client.UpdateTrack(requestCtx, opts.PackageName, currentEditID, manifest.Track, gpc.TrackUpdate{
		Status:         manifest.Status,
		ReleaseName:    manifest.ReleaseName,
		UserFraction:   manifest.UserFraction,
		VersionCodes:   []int64{versionCode},
		UpdatePriority: manifest.UpdatePriority,
		ReleaseNotes:   manifest.ReleaseNotes,
	})
	if err != nil {
		return fail("update_track", fmt.Errorf("failed to update track: %w", err))
	}
	result.Steps = append(result.Steps, fullStep{Name: prefix + "update_track", Status: "ok"})

	writeFullProgress(deps.Stderr, prefix+"validate_edit", "validating Play edit")
	if err := client.ValidateEdit(requestCtx, opts.PackageName, currentEditID); err != nil {
		return fail("validate_edit", fmt.Errorf("failed to validate edit: %w", err))
	}
	result.Steps = append(result.Steps, fullStep{Name: prefix + "validate_edit", Status: "ok"})

	if prefix == "deploy_" && len(opts.VitalsGate) > 0 {
		reportingClient, reportingCtx, reportingCancel, err := buildReportingClient(parentCtx, deps)
		if err != nil {
			return fail("create_reporting_client", err)
		}
		checks, err := evaluateVitalsGate(reportingCtx, reportingClient, opts.PackageName, deps.Now(), opts.VitalsGate)
		reportingCancel()
		if err != nil {
			return fail("vitals_gate_precommit", fmt.Errorf("failed to evaluate vitals gate: %w", err))
		}
		result.VitalsGate = &fullVitalsGateResult{
			Status:      "passed",
			Wait:        opts.VitalsWait.String(),
			AutoHalt:    opts.AutoHaltOnRegression,
			Evaluations: 1,
			Checks:      checks,
		}
		if hasVitalsGateFailure(checks) {
			result.VitalsGate.Status = "blocked"
			return fail("vitals_gate_precommit", fmt.Errorf("vitals gate failed"))
		}
		result.Steps = append(result.Steps, fullStep{Name: prefix + "vitals_gate_precommit", Status: "ok"})
	}

	commit, err := shared.CommitEditWithReviewFallback(requestCtx, client, opts.PackageName, currentEditID, false)
	if err != nil {
		return fail("commit_edit", fmt.Errorf("failed to commit edit: %w", err))
	}
	result.Committed = true
	if commit.ChangesNotSentForReview {
		result.Warnings = append(result.Warnings, "commit used changesNotSentForReview fallback")
	}
	result.Status = "committed"
	result.Steps = append(result.Steps, fullStep{Name: prefix + "commit_edit", Status: "ok"})
	return nil
}

func writeFullProgress(stderr io.Writer, step, detail string) {
	if stderr == nil {
		return
	}
	step = strings.TrimSpace(step)
	detail = strings.TrimSpace(detail)
	if step == "" && detail == "" {
		return
	}
	if detail == "" {
		_, _ = fmt.Fprintf(stderr, "[release full] %s\n", step)
		return
	}
	_, _ = fmt.Fprintf(stderr, "[release full] %s: %s\n", step, detail)
}

func normalizeUploadError(err error) error {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "http2: client connection lost") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "connection reset by peer") {
		return fmt.Errorf("retryable upload failure: artifact upload lost its network connection; rerun the same `gpc release full` command (%w)", err)
	}
	return err
}

func detectExistingBootstrapDraft(ctx context.Context, client Client, packageName string) (bootstrapDraftState, error) {
	edit, err := client.CreateEdit(ctx, packageName)
	if err != nil {
		return bootstrapDraftState{}, fmt.Errorf("failed to inspect internal bootstrap track: %w", err)
	}
	defer client.DeleteEdit(ctx, packageName, edit.ID)

	track, err := client.GetTrack(ctx, packageName, edit.ID, "internal")
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "not found") {
			return bootstrapDraftState{}, nil
		}
		return bootstrapDraftState{}, fmt.Errorf("failed to inspect internal bootstrap track: %w", err)
	}

	for _, release := range track.Releases {
		if strings.EqualFold(strings.TrimSpace(release.Status), "draft") {
			return bootstrapDraftState{
				Exists:       true,
				TrackName:    track.Name,
				VersionCodes: append([]int64(nil), release.VersionCodes...),
			}, nil
		}
	}
	return bootstrapDraftState{}, nil
}

func joinVersionCodes(values []int64) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

func runPostReleaseChecks(ctx context.Context, client Client, deps Deps, packageName, track string, result *fullResult) error {
	edit, err := client.CreateEdit(ctx, packageName)
	if err != nil {
		result.Steps = append(result.Steps, fullStep{Name: "post_release_checks", Status: "error", Error: err.Error()})
		return err
	}
	defer client.DeleteEdit(ctx, packageName, edit.ID)
	if _, err := client.GetTrack(ctx, packageName, edit.ID, track); err != nil {
		result.Steps = append(result.Steps, fullStep{Name: "post_release_track", Status: "error", Error: err.Error()})
		return err
	}
	result.Steps = append(result.Steps, fullStep{Name: "post_release_track", Status: "ok"})
	if _, err := client.ListOneTimeProducts(ctx, packageName, 1, "", false); err == nil {
		result.Steps = append(result.Steps, fullStep{Name: "post_release_products", Status: "ok"})
	} else {
		result.Warnings = append(result.Warnings, err.Error())
	}
	if _, err := client.ListSubscriptions(ctx, packageName, 1, "", false); err == nil {
		result.Steps = append(result.Steps, fullStep{Name: "post_release_subscriptions", Status: "ok"})
	} else {
		result.Warnings = append(result.Warnings, err.Error())
	}
	reportingClient, reportingCtx, reportingCancel, err := buildReportingClient(ctx, deps)
	if err == nil {
		defer reportingCancel()
		if _, err := reportingClient.SearchApps(reportingCtx, 10, "", false); err == nil {
			result.Steps = append(result.Steps, fullStep{Name: "post_release_reporting", Status: "ok"})
		} else {
			result.Warnings = append(result.Warnings, err.Error())
		}
	}
	return nil
}

func writeFilteredAppInitManifest(path string, include map[string]bool) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := yaml.Unmarshal(raw, &payload); err != nil {
		return "", err
	}
	filtered := map[string]any{}
	for key, value := range payload {
		if include[key] {
			filtered[key] = value
		}
	}
	if len(filtered) == 0 {
		return "", nil
	}
	tmp, err := os.CreateTemp("", "gpc-release-appinit-*.yaml")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	encoded, err := yaml.Marshal(filtered)
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(encoded); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

func dirHasFiles(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) > 0
}

func uploadManifestArtifact(ctx context.Context, client Client, packageName, editID string, manifest fullManifest) (int64, error) {
	switch manifest.ArtifactType {
	case artifactTypeAAB:
		bundle, err := client.UploadBundle(ctx, packageName, editID, manifest.ArtifactPath)
		if err != nil {
			return 0, fmt.Errorf("failed to upload aab: %w", err)
		}
		if bundle.VersionCode <= 0 {
			return 0, fmt.Errorf("uploaded aab did not return a valid version code")
		}
		return bundle.VersionCode, nil
	case artifactTypeAPK:
		apk, err := client.UploadAPK(ctx, packageName, editID, manifest.ArtifactPath)
		if err != nil {
			return 0, fmt.Errorf("failed to upload apk: %w", err)
		}
		if apk.VersionCode <= 0 {
			return 0, fmt.Errorf("uploaded apk did not return a valid version code")
		}
		return apk.VersionCode, nil
	default:
		return 0, fmt.Errorf("unsupported artifact type %q", manifest.ArtifactType)
	}
}

func hasVitalsGateFailure(checks []fullVitalsGateCheck) bool {
	for _, check := range checks {
		if !check.Passed {
			return true
		}
	}
	return false
}

func monitorVitalsGate(parentCtx context.Context, deps Deps, client Client, reportingClient ReportingClient, result *fullResult, opts fullOptions, manifest fullManifest, versionCode int64) error {
	waitCtx, cancel := context.WithTimeout(parentCtx, opts.VitalsWait)
	defer cancel()

	for {
		checks, err := evaluateVitalsGate(waitCtx, reportingClient, opts.PackageName, deps.Now(), opts.VitalsGate)
		if err == nil {
			result.VitalsGate.Evaluations++
			result.VitalsGate.Checks = checks
			if hasVitalsGateFailure(checks) {
				result.VitalsGate.Status = "regression"
				result.Steps = append(result.Steps, fullStep{Name: "vitals_gate_monitor", Status: "error", Error: "threshold crossed"})
				if opts.AutoHaltOnRegression {
					if err := haltRollout(waitCtx, client, opts.PackageName, manifest, versionCode); err != nil {
						result.Steps = append(result.Steps, fullStep{Name: "halt_rollout", Status: "error", Error: err.Error()})
						return fmt.Errorf("vitals regression detected: %w", err)
					}
					result.VitalsGate.Status = "halted"
					result.VitalsGate.Halted = true
					result.Status = "halted"
					result.Steps = append(result.Steps, fullStep{Name: "halt_rollout", Status: "ok"})
				} else {
					result.Status = "regression"
				}
				return fmt.Errorf("vitals regression detected")
			}
		} else if !errors.Is(err, errVitalsValueUnavailable) {
			result.Status = "failed"
			result.Steps = append(result.Steps, fullStep{Name: "vitals_gate_monitor", Status: "error", Error: err.Error()})
			return fmt.Errorf("failed to monitor vitals gate: %w", err)
		}

		if waitCtx.Err() != nil {
			if result.VitalsGate != nil {
				result.VitalsGate.Status = "passed"
			}
			result.Steps = append(result.Steps, fullStep{Name: "vitals_gate_monitor", Status: "ok"})
			return nil
		}

		if err := deps.Sleep(waitCtx, defaultVitalsPollInterval); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				if result.VitalsGate != nil {
					result.VitalsGate.Status = "passed"
				}
				result.Steps = append(result.Steps, fullStep{Name: "vitals_gate_monitor", Status: "ok"})
				return nil
			}
			result.Status = "failed"
			result.Steps = append(result.Steps, fullStep{Name: "vitals_gate_monitor", Status: "error", Error: err.Error()})
			return err
		}
	}
}

func haltRollout(ctx context.Context, client Client, packageName string, manifest fullManifest, versionCode int64) error {
	if strings.TrimSpace(manifest.Status) != "inProgress" {
		return fmt.Errorf("auto halt is only supported for inProgress releases")
	}

	edit, err := client.CreateEdit(ctx, packageName)
	if err != nil {
		return fmt.Errorf("failed to create halt edit: %w", err)
	}
	editID := edit.ID
	cleanup := true
	defer func() {
		if cleanup {
			_ = client.DeleteEdit(ctx, packageName, editID)
		}
	}()

	_, err = client.UpdateTrack(ctx, packageName, editID, manifest.Track, gpc.TrackUpdate{
		Status:         "halted",
		ReleaseName:    manifest.ReleaseName,
		UserFraction:   manifest.UserFraction,
		VersionCodes:   []int64{versionCode},
		UpdatePriority: manifest.UpdatePriority,
		ReleaseNotes:   manifest.ReleaseNotes,
	})
	if err != nil {
		return fmt.Errorf("failed to halt rollout: %w", err)
	}
	if err := client.ValidateEdit(ctx, packageName, editID); err != nil {
		return fmt.Errorf("failed to validate halt edit: %w", err)
	}
	if _, err := client.CommitEdit(ctx, packageName, editID, false); err != nil {
		return fmt.Errorf("failed to commit halt edit: %w", err)
	}
	cleanup = false
	return nil
}
