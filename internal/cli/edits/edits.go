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
