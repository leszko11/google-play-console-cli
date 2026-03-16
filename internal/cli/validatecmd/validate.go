package validatecmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/listing"
	"github.com/leszko11/google-play-console-cli/internal/cli/release"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type EditClient interface {
	ValidateEdit(ctx context.Context, packageName, editID string) error
}

type Deps struct {
	RunReleaseVerify func(context.Context, release.VerifyOptions) (release.VerifyResult, error)
	ValidateListings func(string) (listing.ValidationSummary, error)
	ValidateEdit     func(context.Context, string, string) error
	LoadConfig       func() (config.Config, error)
	NewEditClient    func(context.Context, gpc.CredentialInput) (EditClient, error)
	LookupEnv        func(string) string
	Stdout           io.Writer
	Stderr           io.Writer
}

type result struct {
	PackageName    string                `json:"packageName"`
	Track          string                `json:"track"`
	ProjectDir     string                `json:"projectDir"`
	ArtifactPath   string                `json:"artifactPath,omitempty"`
	ListingDir     string                `json:"listingDir,omitempty"`
	NotesMode      string                `json:"notesMode"`
	EditID         string                `json:"editId,omitempty"`
	Status         string                `json:"status"`
	Checks         []release.VerifyCheck `json:"checks"`
	BlockingIssues []string              `json:"blockingIssues,omitempty"`
	Warnings       []string              `json:"warnings,omitempty"`
}

type options struct {
	PackageName string
	Track       string
	ProjectDir  string
	BuildTask   string
	AABPath     string
	ProbeTrack  bool
	NotesMode   string
	NotesFile   string
	NotesLocale string
	NotesText   string
	EditID      string
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var (
		packageName string
		track       string
		projectDir  string
		buildTask   string
		aabPath     string
		probeTrack  bool
		notesMode   string
		notesFile   string
		notesLocale string
		notesText   string
		editID      string
	)

	fs.StringVar(&packageName, "package-name", "", "Target package name")
	fs.StringVar(&track, "track", "alpha", "Target track name")
	fs.StringVar(&projectDir, "project-dir", ".", "Android project directory")
	fs.StringVar(&buildTask, "build-task", ":app:bundleStagingRelease", "Gradle build task for release bundle")
	fs.StringVar(&aabPath, "aab", "", "Path to prebuilt .aab for artifact validation")
	fs.BoolVar(&probeTrack, "probe-track", false, "Create temporary edit and probe target track")
	fs.StringVar(&notesMode, "notes-mode", "git", "Release notes mode: git, file, none")
	fs.StringVar(&notesFile, "notes-file", "", "Release notes file path when notes-mode=file")
	fs.StringVar(&notesLocale, "notes-locale", "en-US", "Release notes locale")
	fs.StringVar(&notesText, "notes-text", "", "Inline release notes text override")
	fs.StringVar(&editID, "edit-id", "", "Existing edit ID to validate")

	return &ffcli.Command{
		Name:      "validate",
		ShortHelp: "Run pre-submission validation checks",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			res, err := run(ctx, deps, options{
				PackageName: strings.TrimSpace(packageName),
				Track:       strings.TrimSpace(track),
				ProjectDir:  strings.TrimSpace(projectDir),
				BuildTask:   strings.TrimSpace(buildTask),
				AABPath:     strings.TrimSpace(aabPath),
				ProbeTrack:  probeTrack,
				NotesMode:   strings.TrimSpace(notesMode),
				NotesFile:   strings.TrimSpace(notesFile),
				NotesLocale: strings.TrimSpace(notesLocale),
				NotesText:   strings.TrimSpace(notesText),
				EditID:      strings.TrimSpace(editID),
			})
			if err != nil {
				return err
			}

			switch shared.ResolveOutput("") {
			case "json":
				if err := shared.WriteJSON(deps.Stdout, res); err != nil {
					return err
				}
			case "table":
				if err := writeTable(deps.Stdout, res); err != nil {
					return err
				}
			default:
				return shared.UsageErrorf("unsupported output format %q", shared.ResolveOutput(""))
			}

			if res.Status != "ok" {
				return fmt.Errorf("validation failed")
			}
			return nil
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.RunReleaseVerify == nil {
		deps.RunReleaseVerify = func(ctx context.Context, opts release.VerifyOptions) (release.VerifyResult, error) {
			return release.RunVerify(ctx, release.Deps{}, opts)
		}
	}
	if deps.ValidateListings == nil {
		deps.ValidateListings = listing.ValidateListingsDir
	}
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewEditClient == nil {
		deps.NewEditClient = func(ctx context.Context, creds gpc.CredentialInput) (EditClient, error) {
			return gpc.NewClient(ctx, creds)
		}
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
	}
	if deps.ValidateEdit == nil {
		deps.ValidateEdit = func(ctx context.Context, packageName, editID string) error {
			client, requestCtx, cancel, err := shared.BuildClient[EditClient](ctx, shared.BuildClientDeps[EditClient]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewEditClient,
			})
			if err != nil {
				return err
			}
			defer cancel()
			return client.ValidateEdit(requestCtx, packageName, editID)
		}
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return deps
}

func run(ctx context.Context, deps Deps, opts options) (result, error) {
	verifyResult, err := deps.RunReleaseVerify(ctx, release.VerifyOptions{
		PackageName: opts.PackageName,
		Track:       opts.Track,
		ProjectDir:  opts.ProjectDir,
		BuildTask:   opts.BuildTask,
		AABPath:     opts.AABPath,
		ProbeTrack:  opts.ProbeTrack,
		NotesMode:   opts.NotesMode,
		NotesFile:   opts.NotesFile,
		NotesLocale: opts.NotesLocale,
		NotesText:   opts.NotesText,
	})
	if err != nil {
		return result{}, err
	}

	res := result{
		PackageName:    verifyResult.PackageName,
		Track:          verifyResult.Track,
		ProjectDir:     verifyResult.ProjectDir,
		ArtifactPath:   verifyResult.ArtifactPath,
		ListingDir:     verifyResult.ListingDir,
		NotesMode:      verifyResult.NotesMode,
		EditID:         opts.EditID,
		Status:         verifyResult.Status,
		Checks:         append([]release.VerifyCheck{}, verifyResult.Checks...),
		BlockingIssues: append([]string{}, verifyResult.BlockingIssues...),
		Warnings:       append([]string{}, verifyResult.Warnings...),
	}

	switch {
	case strings.TrimSpace(res.ListingDir) == "":
		res.addWarn("listing_assets", "skipped (no local listing metadata configured)")
	case hasSuccessfulCheck(res.Checks, "listing_metadata"):
		summary, err := deps.ValidateListings(res.ListingDir)
		if err != nil {
			res.addBlocking("listing_assets", err.Error())
		} else {
			res.addOK("listing_assets", fmt.Sprintf("listing assets ready (dir=%s, locales=%d, images=%d)", res.ListingDir, summary.LocaleCount, summary.ImageCount))
		}
	default:
		res.addWarn("listing_assets", "skipped due to listing metadata errors")
	}

	if opts.EditID == "" {
		res.addWarn("edit_validation", "skipped (provide --edit-id to validate an existing edit)")
		return finalizeResult(res), nil
	}

	if err := deps.ValidateEdit(ctx, res.PackageName, opts.EditID); err != nil {
		res.addBlocking("edit_validation", fmt.Sprintf("failed to validate edit: %v", err))
	} else {
		res.addOK("edit_validation", fmt.Sprintf("edit %q is valid", opts.EditID))
	}

	return finalizeResult(res), nil
}

func finalizeResult(res result) result {
	if len(res.BlockingIssues) == 0 {
		res.Status = "ok"
	} else {
		res.Status = "failed"
	}
	return res
}

func hasSuccessfulCheck(checks []release.VerifyCheck, name string) bool {
	for _, check := range checks {
		if check.Name == name && check.Status == "ok" {
			return true
		}
	}
	return false
}

func (r *result) addBlocking(name, detail string) {
	r.Checks = append(r.Checks, release.VerifyCheck{
		Name:     name,
		Status:   "error",
		Detail:   detail,
		Blocking: true,
	})
	r.BlockingIssues = append(r.BlockingIssues, detail)
}

func (r *result) addWarn(name, detail string) {
	r.Checks = append(r.Checks, release.VerifyCheck{
		Name:   name,
		Status: "warning",
		Detail: detail,
	})
	r.Warnings = append(r.Warnings, detail)
}

func (r *result) addOK(name, detail string) {
	r.Checks = append(r.Checks, release.VerifyCheck{
		Name:   name,
		Status: "ok",
		Detail: detail,
	})
}

func writeTable(out io.Writer, res result) error {
	if _, err := fmt.Fprintf(out, "STATUS\t%s\n", res.Status); err != nil {
		return err
	}
	if res.PackageName != "" {
		if _, err := fmt.Fprintf(out, "PACKAGE\t%s\n", res.PackageName); err != nil {
			return err
		}
	}
	if res.Track != "" {
		if _, err := fmt.Fprintf(out, "TRACK\t%s\n", res.Track); err != nil {
			return err
		}
	}
	if res.EditID != "" {
		if _, err := fmt.Fprintf(out, "EDIT\t%s\n", res.EditID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(out, "CHECK\tSTATUS\tBLOCKING\tDETAIL"); err != nil {
		return err
	}
	for _, check := range res.Checks {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%t\t%s\n", check.Name, check.Status, check.Blocking, check.Detail); err != nil {
			return err
		}
	}
	return nil
}
