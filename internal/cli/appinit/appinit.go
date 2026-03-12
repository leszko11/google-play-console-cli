package appinit

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/listing"
	"github.com/leszko11/google-play-console-cli/internal/cli/monetization"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const (
	appInitStatusCompleted = "completed"
	appInitStatusDryRun    = "dry-run"
	appInitStatusFailed    = "failed"
)

type Client interface {
	listing.Client
	monetization.Client
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	ValidateEdit(ctx context.Context, packageName, editID string) error
	CommitEdit(ctx context.Context, packageName, editID string) (gpc.EditInfo, error)
	UpdateAppDetails(ctx context.Context, packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
}

type stepResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type options struct {
	PackageName  string
	ManifestPath string
	Confirm      bool
	DryRun       bool
}

type appInitManifest struct {
	AppDetails   *appDetailsSection   `json:"appDetails" yaml:"appDetails"`
	Listing      *listingSection      `json:"listing" yaml:"listing"`
	Monetization *monetizationSection `json:"monetization" yaml:"monetization"`
}

type appDetailsSection struct {
	DefaultLanguage string `json:"defaultLanguage" yaml:"defaultLanguage"`
	ContactEmail    string `json:"contactEmail" yaml:"contactEmail"`
	ContactPhone    string `json:"contactPhone" yaml:"contactPhone"`
	ContactWebsite  string `json:"contactWebsite" yaml:"contactWebsite"`
}

type listingSection struct {
	Dir           string `json:"dir" yaml:"dir"`
	DeleteMissing bool   `json:"deleteMissing" yaml:"deleteMissing"`
}

type monetizationSection struct {
	Manifest string `json:"manifest" yaml:"manifest"`
	Activate bool   `json:"activate" yaml:"activate"`
}

type result struct {
	PackageName       string       `json:"packageName"`
	Manifest          string       `json:"manifest"`
	Status            string       `json:"status"`
	PlannedSections   []string     `json:"plannedSections,omitempty"`
	CompletedSections []string     `json:"completedSections,omitempty"`
	Steps             []stepResult `json:"steps"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("appinit", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts options
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.ManifestPath, "manifest", "", "Path to app bootstrap manifest (.json/.yaml/.yml)")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Confirm applying bootstrap changes (required unless --dry-run)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Validate all sections and use dry-run flows where supported")

	return &ffcli.Command{
		Name:      "appinit",
		ShortHelp: "Bootstrap app store presence from a manifest",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, manifest, err := validateOptions(opts)
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

			return runAppInit(ctx, requestCtx, client, deps.Stdout, opts, manifest)
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

func validateOptions(opts options) (options, appInitManifest, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return options{}, appInitManifest{}, err
	}
	opts.PackageName = pkg
	opts.ManifestPath = strings.TrimSpace(opts.ManifestPath)
	if opts.ManifestPath == "" {
		return options{}, appInitManifest{}, shared.UsageErrorf("--manifest is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return options{}, appInitManifest{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}

	manifest, err := loadManifest(opts.ManifestPath)
	if err != nil {
		return options{}, appInitManifest{}, err
	}
	return opts, manifest, nil
}

func loadManifest(path string) (appInitManifest, error) {
	var manifest appInitManifest
	if err := shared.LoadManifest(path, &manifest); err != nil {
		return appInitManifest{}, err
	}
	baseDir := filepath.Dir(path)

	if manifest.AppDetails != nil {
		manifest.AppDetails.DefaultLanguage = strings.TrimSpace(manifest.AppDetails.DefaultLanguage)
		manifest.AppDetails.ContactEmail = strings.TrimSpace(manifest.AppDetails.ContactEmail)
		manifest.AppDetails.ContactPhone = strings.TrimSpace(manifest.AppDetails.ContactPhone)
		manifest.AppDetails.ContactWebsite = strings.TrimSpace(manifest.AppDetails.ContactWebsite)
		if manifest.AppDetails.DefaultLanguage == "" && manifest.AppDetails.ContactEmail == "" && manifest.AppDetails.ContactPhone == "" && manifest.AppDetails.ContactWebsite == "" {
			return appInitManifest{}, shared.UsageErrorf("appDetails must include at least one field")
		}
	}
	if manifest.Listing != nil {
		manifest.Listing.Dir = resolveManifestPath(baseDir, manifest.Listing.Dir)
		if strings.TrimSpace(manifest.Listing.Dir) == "" {
			return appInitManifest{}, shared.UsageErrorf("listing.dir is required")
		}
	}
	if manifest.Monetization != nil {
		manifest.Monetization.Manifest = resolveManifestPath(baseDir, manifest.Monetization.Manifest)
		if strings.TrimSpace(manifest.Monetization.Manifest) == "" {
			return appInitManifest{}, shared.UsageErrorf("monetization.manifest is required")
		}
	}
	if manifest.AppDetails == nil && manifest.Listing == nil && manifest.Monetization == nil {
		return appInitManifest{}, shared.UsageErrorf("manifest must include at least one of: appDetails, listing, monetization")
	}

	return manifest, nil
}

func resolveManifestPath(baseDir, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(baseDir, value)
}

func runAppInit(parentCtx, requestCtx context.Context, client Client, out io.Writer, opts options, manifest appInitManifest) error {
	result := result{
		PackageName:     opts.PackageName,
		Manifest:        opts.ManifestPath,
		Status:          appInitStatusFailed,
		PlannedSections: plannedSections(manifest),
		Steps:           make([]stepResult, 0, 6),
	}

	fail := func(step string, err error) error {
		result.Steps = append(result.Steps, stepResult{Name: step, Status: "error", Error: err.Error()})
		_ = shared.WriteJSON(out, result)
		return err
	}

	if manifest.AppDetails != nil {
		if err := runAppDetails(parentCtx, requestCtx, client, opts.PackageName, *manifest.AppDetails, opts.DryRun); err != nil {
			return fail("app_details", err)
		}
		result.CompletedSections = append(result.CompletedSections, "appDetails")
		result.Steps = append(result.Steps, stepResult{Name: "app_details", Status: "ok"})
	}

	if manifest.Listing != nil {
		locales, err := listing.ScanListingsDir(manifest.Listing.Dir)
		if err != nil {
			return fail("listing_scan", err)
		}
		result.Steps = append(result.Steps, stepResult{Name: "listing_scan", Status: "ok"})

		var listingOut bytes.Buffer
		if err := listing.RunSync(parentCtx, requestCtx, client, &listingOut, listing.SyncOptions{
			PackageName:   opts.PackageName,
			Dir:           manifest.Listing.Dir,
			Confirm:       !opts.DryRun,
			DryRun:        opts.DryRun,
			DeleteMissing: manifest.Listing.DeleteMissing,
		}, locales); err != nil {
			return fail("listing_sync", fmt.Errorf("%w", err))
		}
		result.CompletedSections = append(result.CompletedSections, "listing")
		result.Steps = append(result.Steps, stepResult{Name: "listing_sync", Status: "ok"})
	}

	if manifest.Monetization != nil {
		monetizationManifest, err := monetization.LoadManifest(manifest.Monetization.Manifest)
		if err != nil {
			return fail("monetization_manifest", err)
		}
		result.Steps = append(result.Steps, stepResult{Name: "monetization_manifest", Status: "ok"})

		var monetizationOut bytes.Buffer
		if err := monetization.RunSetup(requestCtx, client, &monetizationOut, monetization.SetupOptions{
			PackageName:  opts.PackageName,
			ManifestPath: manifest.Monetization.Manifest,
			Confirm:      !opts.DryRun,
			DryRun:       opts.DryRun,
			Activate:     manifest.Monetization.Activate,
		}, monetizationManifest); err != nil {
			return fail("monetization_setup", fmt.Errorf("%w", err))
		}
		result.CompletedSections = append(result.CompletedSections, "monetization")
		result.Steps = append(result.Steps, stepResult{Name: "monetization_setup", Status: "ok"})
	}

	if opts.DryRun {
		result.Status = appInitStatusDryRun
	} else {
		result.Status = appInitStatusCompleted
	}
	return shared.WriteJSON(out, result)
}

func plannedSections(manifest appInitManifest) []string {
	sections := make([]string, 0, 3)
	if manifest.AppDetails != nil {
		sections = append(sections, "appDetails")
	}
	if manifest.Listing != nil {
		sections = append(sections, "listing")
	}
	if manifest.Monetization != nil {
		sections = append(sections, "monetization")
	}
	return sections
}

func runAppDetails(parentCtx, requestCtx context.Context, client Client, packageName string, details appDetailsSection, dryRun bool) error {
	edit, err := client.CreateEdit(requestCtx, packageName)
	if err != nil {
		return fmt.Errorf("failed to create edit for app details: %w", err)
	}
	fail := func(err error) error {
		cleanupCtx, cleanupCancel := shared.ContextWithTimeout(parentCtx, shared.ActiveGlobalFlags().Timeout)
		_ = client.DeleteEdit(cleanupCtx, packageName, edit.ID)
		cleanupCancel()
		return err
	}

	if _, err := client.UpdateAppDetails(requestCtx, packageName, edit.ID, gpc.AppDetailsUpdate{
		DefaultLanguage: details.DefaultLanguage,
		ContactEmail:    details.ContactEmail,
		ContactPhone:    details.ContactPhone,
		ContactWebsite:  details.ContactWebsite,
	}); err != nil {
		return fail(fmt.Errorf("failed to update app details: %w", err))
	}
	if err := client.ValidateEdit(requestCtx, packageName, edit.ID); err != nil {
		return fail(fmt.Errorf("failed to validate app details edit: %w", err))
	}
	if dryRun {
		if err := client.DeleteEdit(requestCtx, packageName, edit.ID); err != nil {
			return fail(fmt.Errorf("failed to delete app details dry-run edit: %w", err))
		}
		return nil
	}
	if _, err := client.CommitEdit(requestCtx, packageName, edit.ID); err != nil {
		return fail(fmt.Errorf("failed to commit app details edit: %w", err))
	}
	return nil
}
