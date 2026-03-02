package edits

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	GetEdit(ctx context.Context, packageName, editID string) (gpc.EditInfo, error)
	ValidateEdit(ctx context.Context, packageName, editID string) error
	CommitEdit(ctx context.Context, packageName, editID string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	GetAppDetails(ctx context.Context, packageName, editID string) (gpc.AppDetailsInfo, error)
	UpdateAppDetails(ctx context.Context, packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error)
	GetTesters(ctx context.Context, packageName, editID, track string) (gpc.TestersInfo, error)
	UpdateTesters(ctx context.Context, packageName, editID, track string, googleGroups []string) (gpc.TestersInfo, error)
	GetCountryAvailability(ctx context.Context, packageName, editID, track string) (gpc.CountryAvailabilityInfo, error)
	GetListing(ctx context.Context, packageName, editID, language string) (gpc.ListingInfo, error)
	ListListings(ctx context.Context, packageName, editID string) ([]gpc.ListingInfo, error)
	UpdateListing(ctx context.Context, packageName, editID, language string, update gpc.ListingUpdate) (gpc.ListingInfo, error)
	DeleteListing(ctx context.Context, packageName, editID, language string) error
	DeleteAllListings(ctx context.Context, packageName, editID string) error
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
		Name:      "edits",
		ShortHelp: "Manage Google Play edit transactions",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newCreateCommand(deps),
			newGetCommand(deps),
			newValidateCommand(deps),
			newCommitCommand(deps),
			newDeleteCommand(deps),
			newDetailsCommand(deps),
			newTestersCommand(deps),
			newCountryAvailabilityCommand(deps),
			newListingsCommand(deps),
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

func newCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName string
	fs.StringVar(&packageName, "package-name", "", "Package name")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a new edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			edit, err := client.CreateEdit(requestCtx, pkg)
			if err != nil {
				return fmt.Errorf("failed to create edit: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"edit":        edit,
			})
		},
	}
}

func newGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get edit details",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			edit, err := client.GetEdit(requestCtx, pkg, editID)
			if err != nil {
				return fmt.Errorf("failed to get edit: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"edit":        edit,
			})
		},
	}
}

func newValidateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")

	return &ffcli.Command{
		Name:      "validate",
		ShortHelp: "Validate an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			if err := client.ValidateEdit(requestCtx, pkg, editID); err != nil {
				return fmt.Errorf("failed to validate edit: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"status":      "validated",
			})
		},
	}
}

func newCommitCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("commit", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm committing the edit (required)")

	return &ffcli.Command{
		Name:      "commit",
		ShortHelp: "Commit an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to commit edit %q", editID)
			}
			edit, err := client.CommitEdit(requestCtx, pkg, editID)
			if err != nil {
				return fmt.Errorf("failed to commit edit: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"edit":        edit,
				"status":      "committed",
			})
		},
	}
}

func newDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the edit (required)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to delete edit %q", editID)
			}
			if err := client.DeleteEdit(requestCtx, pkg, editID); err != nil {
				return fmt.Errorf("failed to delete edit: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"status":      "deleted",
			})
		},
	}
}

func newListingsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "listings",
		ShortHelp: "Manage listing changes inside an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListingsListCommand(deps),
			newListingsGetCommand(deps),
			newListingsUpdateCommand(deps),
			newListingsDeleteCommand(deps),
			newListingsDeleteAllCommand(deps),
		},
	}
}

func newDetailsCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "details",
		ShortHelp: "Manage app details inside an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newDetailsGetCommand(deps),
			newDetailsUpdateCommand(deps),
		},
	}
}

func newDetailsGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get app details in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			details, err := client.GetAppDetails(requestCtx, pkg, editID)
			if err != nil {
				return fmt.Errorf("failed to get app details: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"details":     details,
			})
		},
	}
}

func newDetailsUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	var defaultLanguage, contactEmail, contactPhone, contactWebsite string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&defaultLanguage, "default-language", "", "Default listing language (BCP-47, e.g. en-US)")
	fs.StringVar(&contactEmail, "contact-email", "", "Contact email address")
	fs.StringVar(&contactPhone, "contact-phone", "", "Contact phone number")
	fs.StringVar(&contactWebsite, "contact-website", "", "Contact website URL")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update app details in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			details, err := client.UpdateAppDetails(requestCtx, pkg, editID, gpc.AppDetailsUpdate{
				DefaultLanguage: defaultLanguage,
				ContactEmail:    contactEmail,
				ContactPhone:    contactPhone,
				ContactWebsite:  contactWebsite,
			})
			if err != nil {
				return fmt.Errorf("failed to update app details: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"details":     details,
				"status":      "updated",
			})
		},
	}
}

func newTestersCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "testers",
		ShortHelp: "Manage testers for a track inside an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newTestersGetCommand(deps),
			newTestersUpdateCommand(deps),
		},
	}
}

func newTestersGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, track string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&track, "track", "", "Track name (e.g. internal, closed)")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get tester Google Groups for a track in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			track = strings.TrimSpace(track)
			if track == "" {
				return fmt.Errorf("--track is required")
			}
			testers, err := client.GetTesters(requestCtx, pkg, editID, track)
			if err != nil {
				return fmt.Errorf("failed to get testers: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"testers":     testers,
			})
		},
	}
}

func newTestersUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, track, groupsCSV string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&track, "track", "", "Track name (e.g. internal, closed)")
	fs.StringVar(&groupsCSV, "google-groups", "", "Comma-separated Google Group email addresses")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update tester Google Groups for a track in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			track = strings.TrimSpace(track)
			if track == "" {
				return fmt.Errorf("--track is required")
			}
			googleGroups := parseCommaSeparated(groupsCSV)
			if len(googleGroups) == 0 {
				return fmt.Errorf("--google-groups is required")
			}
			testers, err := client.UpdateTesters(requestCtx, pkg, editID, track, googleGroups)
			if err != nil {
				return fmt.Errorf("failed to update testers: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"testers":     testers,
				"status":      "updated",
			})
		},
	}
}

func newCountryAvailabilityCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "country-availability",
		ShortHelp: "Inspect track country availability inside an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newCountryAvailabilityGetCommand(deps),
		},
	}
}

func newCountryAvailabilityGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, track string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&track, "track", "", "Track name (e.g. production)")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get country availability for a track in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			track = strings.TrimSpace(track)
			if track == "" {
				return fmt.Errorf("--track is required")
			}
			availability, err := client.GetCountryAvailability(requestCtx, pkg, editID, track)
			if err != nil {
				return fmt.Errorf("failed to get country availability: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":         pkg,
				"editId":              editID,
				"countryAvailability": availability,
			})
		},
	}
}

func newListingsListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List localized listings in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			listings, err := client.ListListings(requestCtx, pkg, editID)
			if err != nil {
				return fmt.Errorf("failed to list listings: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"listings":    listings,
			})
		},
	}
}

func parseCommaSeparated(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func newListingsGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, locale string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&locale, "locale", "", "Listing locale (BCP-47, e.g. en-US)")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get listing in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return fmt.Errorf("--locale is required")
			}
			listing, err := client.GetListing(requestCtx, pkg, editID, locale)
			if err != nil {
				return fmt.Errorf("failed to get listing: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"listing":     listing,
			})
		},
	}
}

func newListingsUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, locale, title, shortDescription, fullDescription string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&locale, "locale", "", "Listing locale (BCP-47, e.g. en-US)")
	fs.StringVar(&title, "title", "", "Localized app title")
	fs.StringVar(&shortDescription, "short-description", "", "Localized short description")
	fs.StringVar(&fullDescription, "full-description", "", "Localized full description")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update listing fields in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return fmt.Errorf("--locale is required")
			}
			listing, err := client.UpdateListing(requestCtx, pkg, editID, locale, gpc.ListingUpdate{
				Title:            title,
				ShortDescription: shortDescription,
				FullDescription:  fullDescription,
			})
			if err != nil {
				return fmt.Errorf("failed to update listing: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"listing":     listing,
				"status":      "updated",
			})
		},
	}
}

func newListingsDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, locale string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&locale, "locale", "", "Listing locale (BCP-47, e.g. en-US)")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete one localized listing in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return fmt.Errorf("--locale is required")
			}
			if err := client.DeleteListing(requestCtx, pkg, editID, locale); err != nil {
				return fmt.Errorf("failed to delete listing: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"locale":      locale,
				"status":      "deleted",
			})
		},
	}
}

func newListingsDeleteAllCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete-all", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")

	return &ffcli.Command{
		Name:      "delete-all",
		ShortHelp: "Delete all localized listings in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			if err := client.DeleteAllListings(requestCtx, pkg, editID); err != nil {
				return fmt.Errorf("failed to delete all listings: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"status":      "deleted_all",
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName string, upload bool) (Client, string, context.Context, context.CancelFunc, error) {
	pkg, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return nil, "", nil, nil, err
	}

	client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
		Upload:     upload,
	})
	if err != nil {
		return nil, "", nil, nil, err
	}

	return client, pkg, requestCtx, cancel, nil
}
