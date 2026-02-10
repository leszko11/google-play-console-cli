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

const envServiceAccountPath = "GPC_SERVICE_ACCOUNT_PATH"

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	GetEdit(ctx context.Context, packageName, editID string) (gpc.EditInfo, error)
	ValidateEdit(ctx context.Context, packageName, editID string) error
	CommitEdit(ctx context.Context, packageName, editID string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	GetListing(ctx context.Context, packageName, editID, language string) (gpc.ListingInfo, error)
	UpdateListing(ctx context.Context, packageName, editID, language string, update gpc.ListingUpdate) (gpc.ListingInfo, error)
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
			client, pkg, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			edit, err := client.CreateEdit(ctx, pkg)
			if err != nil {
				return fmt.Errorf("failed to create edit: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
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
			client, pkg, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			edit, err := client.GetEdit(ctx, pkg, editID)
			if err != nil {
				return fmt.Errorf("failed to get edit: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
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
			client, pkg, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			if err := client.ValidateEdit(ctx, pkg, editID); err != nil {
				return fmt.Errorf("failed to validate edit: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
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
			client, pkg, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to commit edit %q", editID)
			}
			edit, err := client.CommitEdit(ctx, pkg, editID)
			if err != nil {
				return fmt.Errorf("failed to commit edit: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
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
			client, pkg, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to delete edit %q", editID)
			}
			if err := client.DeleteEdit(ctx, pkg, editID); err != nil {
				return fmt.Errorf("failed to delete edit: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
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
			newListingsGetCommand(deps),
			newListingsUpdateCommand(deps),
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
			client, pkg, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return fmt.Errorf("--locale is required")
			}
			listing, err := client.GetListing(ctx, pkg, editID, locale)
			if err != nil {
				return fmt.Errorf("failed to get listing: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
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
			client, pkg, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return fmt.Errorf("--locale is required")
			}
			listing, err := client.UpdateListing(ctx, pkg, editID, locale, gpc.ListingUpdate{
				Title:            title,
				ShortDescription: shortDescription,
				FullDescription:  fullDescription,
			})
			if err != nil {
				return fmt.Errorf("failed to update listing: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"listing":     listing,
				"status":      "updated",
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName string) (Client, string, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, "", fmt.Errorf("--package-name is required")
	}
	cfg, err := deps.LoadConfig()
	if err != nil {
		return nil, "", err
	}
	serviceAccountPath, err := resolveServiceAccountPath(cfg, deps.LookupEnv)
	if err != nil {
		return nil, "", err
	}
	client, err := deps.NewClient(ctx, gpc.CredentialInput{ServiceAccountPath: serviceAccountPath})
	if err != nil {
		return nil, "", err
	}
	return client, packageName, nil
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

func writeJSON(out io.Writer, v any) error {
	b, err := shared.RenderJSON(v, false)
	if err != nil {
		return err
	}
	_, err = out.Write(b)
	return err
}
