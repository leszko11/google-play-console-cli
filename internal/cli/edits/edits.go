package edits

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"

	_ "image/jpeg"
	_ "image/png"
)

type Client interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	GetEdit(ctx context.Context, packageName, editID string) (gpc.EditInfo, error)
	ValidateEdit(ctx context.Context, packageName, editID string) error
	CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	GetAppDetails(ctx context.Context, packageName, editID string) (gpc.AppDetailsInfo, error)
	UpdateAppDetails(ctx context.Context, packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error)
	ReplaceAppDetails(ctx context.Context, packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error)
	GetTesters(ctx context.Context, packageName, editID, track string) (gpc.TestersInfo, error)
	UpdateTesters(ctx context.Context, packageName, editID, track string, googleGroups []string) (gpc.TestersInfo, error)
	ReplaceTesters(ctx context.Context, packageName, editID, track string, googleGroups []string) (gpc.TestersInfo, error)
	GetCountryAvailability(ctx context.Context, packageName, editID, track string) (gpc.CountryAvailabilityInfo, error)
	GetListing(ctx context.Context, packageName, editID, language string) (gpc.ListingInfo, error)
	ListListings(ctx context.Context, packageName, editID string) ([]gpc.ListingInfo, error)
	UpdateListing(ctx context.Context, packageName, editID, language string, update gpc.ListingUpdate) (gpc.ListingInfo, error)
	ReplaceListing(ctx context.Context, packageName, editID, language string, update gpc.ListingUpdate) (gpc.ListingInfo, error)
	DeleteListing(ctx context.Context, packageName, editID, language string) error
	DeleteAllListings(ctx context.Context, packageName, editID string) error
	ListImages(ctx context.Context, packageName, editID, language, imageType string) ([]gpc.ImageInfo, error)
	UploadImage(ctx context.Context, packageName, editID, language, imageType, imagePath string) (gpc.ImageInfo, error)
	DeleteImage(ctx context.Context, packageName, editID, language, imageType, imageID string) error
	DeleteAllImages(ctx context.Context, packageName, editID, language, imageType string) ([]gpc.ImageInfo, error)
	GetExpansionFile(ctx context.Context, packageName, editID string, apkVersionCode int64, expansionFileType string) (gpc.ExpansionFileInfo, error)
	PatchExpansionFile(ctx context.Context, packageName, editID string, apkVersionCode int64, expansionFileType string, referencesVersion int64) (gpc.ExpansionFileInfo, error)
	UpdateExpansionFile(ctx context.Context, packageName, editID string, apkVersionCode int64, expansionFileType string, referencesVersion int64) (gpc.ExpansionFileInfo, error)
	UploadExpansionFile(ctx context.Context, packageName, editID string, apkVersionCode int64, expansionFileType, filePath string) (gpc.ExpansionFileInfo, error)
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
			newImagesCommand(deps),
			newExpansionFilesCommand(deps),
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
				return shared.UsageErrorf("--edit-id is required")
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
				return shared.UsageErrorf("--edit-id is required")
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
	var confirm, dryRun, changesNotSentForReview bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm committing the edit (required unless --dry-run)")
	fs.BoolVar(&dryRun, "dry-run", false, "Validate the edit without committing it")
	fs.BoolVar(&changesNotSentForReview, "changes-not-sent-for-review", false, "Indicate that the changes in this edit will not be reviewed until they are explicitly sent for review from the Google Play Console UI")

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
				return shared.UsageErrorf("--edit-id is required")
			}
			if !confirm && !dryRun {
				return shared.UsageErrorf("--confirm is required unless --dry-run is set")
			}
			if dryRun {
				if err := client.ValidateEdit(requestCtx, pkg, editID); err != nil {
					return fmt.Errorf("failed to validate edit: %w", err)
				}
				return shared.WriteJSON(deps.Stdout, map[string]any{
					"packageName": pkg,
					"editId":      editID,
					"status":      "dry-run",
					"validated":   true,
				})
			}
			commit, err := shared.CommitEditWithReviewFallback(requestCtx, client, pkg, editID, changesNotSentForReview)
			if err != nil {
				return fmt.Errorf("failed to commit edit: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":             pkg,
				"edit":                    commit.Edit,
				"status":                  "committed",
				"changesNotSentForReview": commit.ChangesNotSentForReview,
				"commitRetried":           commit.RetriedWithChangesNotSentForReview,
			})
		},
	}
}

func newDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	var confirm, dryRun bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deleting the edit (required unless --dry-run)")
	fs.BoolVar(&dryRun, "dry-run", false, "Verify the edit exists without deleting it")

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
				return shared.UsageErrorf("--edit-id is required")
			}
			if !confirm && !dryRun {
				return shared.UsageErrorf("--confirm is required unless --dry-run is set")
			}
			if dryRun {
				edit, err := client.GetEdit(requestCtx, pkg, editID)
				if err != nil {
					return fmt.Errorf("failed to get edit: %w", err)
				}
				return shared.WriteJSON(deps.Stdout, map[string]any{
					"packageName": pkg,
					"edit":        edit,
					"status":      "dry-run",
				})
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
			newListingsBatchUpdateCommand(deps),
			newListingsDeleteCommand(deps),
			newListingsDeleteAllCommand(deps),
		},
	}
}

func newImagesCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "images",
		ShortHelp: "Manage store listing images inside an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newImagesListCommand(deps),
			newImagesUploadCommand(deps),
			newImagesUploadDirCommand(deps),
			newImagesDeleteCommand(deps),
			newImagesDeleteAllCommand(deps),
		},
	}
}

func newExpansionFilesCommand(deps Deps) *ffcli.Command {
	return &ffcli.Command{
		Name:      "expansion-files",
		ShortHelp: "Manage APK expansion files inside an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newExpansionFilesGetCommand(deps),
			newExpansionFilesPatchCommand(deps),
			newExpansionFilesUpdateCommand(deps),
			newExpansionFilesUploadCommand(deps),
		},
	}
}

func newImagesListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, locale, imageType string
	addImageSharedFlags(fs, &packageName, &editID, &locale, &imageType)

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List images for one locale/type inside an edit",
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
				return shared.UsageErrorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return shared.UsageErrorf("--locale is required")
			}
			imageType, err = validateImageType(imageType)
			if err != nil {
				return err
			}

			images, err := client.ListImages(requestCtx, pkg, editID, locale, imageType)
			if err != nil {
				return fmt.Errorf("failed to list images: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"locale":      locale,
				"imageType":   imageType,
				"images":      images,
			})
		},
	}
}

func newImagesUploadCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, locale, imageType, imagePath string
	addImageSharedFlags(fs, &packageName, &editID, &locale, &imageType)
	fs.StringVar(&imagePath, "file", "", "Path to image file (PNG/JPEG)")

	return &ffcli.Command{
		Name:      "upload",
		ShortHelp: "Upload one image inside an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, true)
			if err != nil {
				return err
			}
			defer cancel()

			editID = strings.TrimSpace(editID)
			if editID == "" {
				return shared.UsageErrorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return shared.UsageErrorf("--locale is required")
			}
			imageType, err = validateImageType(imageType)
			if err != nil {
				return err
			}
			imagePath = strings.TrimSpace(imagePath)
			if imagePath == "" {
				return shared.UsageErrorf("--file is required")
			}
			if err := validateImageUploadFile(imageType, imagePath); err != nil {
				return err
			}

			imageInfo, err := client.UploadImage(requestCtx, pkg, editID, locale, imageType, imagePath)
			if err != nil {
				return fmt.Errorf("failed to upload image: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"locale":      locale,
				"imageType":   imageType,
				"image":       imageInfo,
				"status":      "uploaded",
			})
		},
	}
}

func newImagesUploadDirCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("upload-dir", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, locale, imageType, dir, output string
	var replace bool
	addImageSharedFlags(fs, &packageName, &editID, &locale, &imageType)
	fs.StringVar(&dir, "dir", "", "Directory containing image files (PNG/JPEG)")
	fs.BoolVar(&replace, "replace", false, "Delete existing images for this locale/type before uploading")
	fs.StringVar(&output, "output", "", "Output format: json")

	return &ffcli.Command{
		Name:      "upload-dir",
		ShortHelp: "Upload all image files from a directory inside an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if strings.TrimSpace(output) != "" && shared.ResolveOutput(output) != "json" {
				resolved := shared.ResolveOutput(output)
				return shared.UsageErrorf("unsupported output format %q", resolved)
			}

			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, true)
			if err != nil {
				return err
			}
			defer cancel()

			editID = strings.TrimSpace(editID)
			if editID == "" {
				return shared.UsageErrorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return shared.UsageErrorf("--locale is required")
			}
			imageType, err = validateImageType(imageType)
			if err != nil {
				return err
			}
			dir = strings.TrimSpace(dir)
			if dir == "" {
				return shared.UsageErrorf("--dir is required")
			}

			files, err := collectImageUploadFiles(dir)
			if err != nil {
				return err
			}
			for _, imagePath := range files {
				if err := validateImageUploadFile(imageType, imagePath); err != nil {
					return err
				}
			}

			deleted := []gpc.ImageInfo(nil)
			if replace {
				deleted, err = client.DeleteAllImages(requestCtx, pkg, editID, locale, imageType)
				if err != nil {
					return fmt.Errorf("failed to replace images: %w", err)
				}
			}

			uploaded := make([]gpc.ImageInfo, 0, len(files))
			for _, imagePath := range files {
				imageInfo, err := client.UploadImage(requestCtx, pkg, editID, locale, imageType, imagePath)
				if err != nil {
					return fmt.Errorf("failed to upload image %q: %w", imagePath, err)
				}
				uploaded = append(uploaded, imageInfo)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"locale":      locale,
				"imageType":   imageType,
				"dir":         dir,
				"replace":     replace,
				"uploadCount": len(uploaded),
				"images":      uploaded,
				"deleted":     deleted,
				"status":      "uploaded",
			})
		},
	}
}

func newImagesDeleteCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, locale, imageType, imageID string
	addImageSharedFlags(fs, &packageName, &editID, &locale, &imageType)
	fs.StringVar(&imageID, "image-id", "", "Image ID to delete")

	return &ffcli.Command{
		Name:      "delete",
		ShortHelp: "Delete one image inside an edit",
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
				return shared.UsageErrorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return shared.UsageErrorf("--locale is required")
			}
			imageType, err = validateImageType(imageType)
			if err != nil {
				return err
			}
			imageID = strings.TrimSpace(imageID)
			if imageID == "" {
				return shared.UsageErrorf("--image-id is required")
			}

			if err := client.DeleteImage(requestCtx, pkg, editID, locale, imageType, imageID); err != nil {
				return fmt.Errorf("failed to delete image: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"locale":      locale,
				"imageType":   imageType,
				"imageId":     imageID,
				"status":      "deleted",
			})
		},
	}
}

func newImagesDeleteAllCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("delete-all", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, locale, imageType string
	addImageSharedFlags(fs, &packageName, &editID, &locale, &imageType)

	return &ffcli.Command{
		Name:      "delete-all",
		ShortHelp: "Delete all images for one locale/type inside an edit",
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
				return shared.UsageErrorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return shared.UsageErrorf("--locale is required")
			}
			imageType, err = validateImageType(imageType)
			if err != nil {
				return err
			}

			deleted, err := client.DeleteAllImages(requestCtx, pkg, editID, locale, imageType)
			if err != nil {
				return fmt.Errorf("failed to delete all images: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"locale":      locale,
				"imageType":   imageType,
				"images":      deleted,
				"status":      "deleted_all",
			})
		},
	}
}

func addImageSharedFlags(fs *flag.FlagSet, packageName, editID, locale, imageType *string) {
	fs.StringVar(packageName, "package-name", "", "Package name")
	fs.StringVar(editID, "edit-id", "", "Edit ID")
	fs.StringVar(locale, "locale", "", "Listing locale (BCP-47, e.g. en-US)")
	fs.StringVar(imageType, "image-type", "", "Image type (icon, featureGraphic, phoneScreenshots, ...)")
}

func newExpansionFilesGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, expansionFileType string
	var apkVersionCode int64
	addExpansionFileSharedFlags(fs, &packageName, &editID, &apkVersionCode, &expansionFileType)

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get expansion file configuration for one APK inside an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()

			editID, expansionFileType, err = validateExpansionFileArgs(editID, apkVersionCode, expansionFileType)
			if err != nil {
				return err
			}

			file, err := client.GetExpansionFile(requestCtx, pkg, editID, apkVersionCode, expansionFileType)
			if err != nil {
				return fmt.Errorf("failed to get expansion file: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":       pkg,
				"editId":            editID,
				"apkVersionCode":    apkVersionCode,
				"expansionFileType": expansionFileType,
				"expansionFile":     file,
			})
		},
	}
}

func newExpansionFilesPatchCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("patch", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, expansionFileType string
	var apkVersionCode, referencesVersion int64
	addExpansionFileSharedFlags(fs, &packageName, &editID, &apkVersionCode, &expansionFileType)
	fs.Int64Var(&referencesVersion, "references-version", 0, "APK version code whose expansion file should be referenced")

	return &ffcli.Command{
		Name:      "patch",
		ShortHelp: "Patch expansion file reference for one APK inside an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()

			editID, expansionFileType, err = validateExpansionFileArgs(editID, apkVersionCode, expansionFileType)
			if err != nil {
				return err
			}
			if referencesVersion <= 0 {
				return shared.UsageErrorf("--references-version must be greater than zero")
			}

			file, err := client.PatchExpansionFile(requestCtx, pkg, editID, apkVersionCode, expansionFileType, referencesVersion)
			if err != nil {
				return fmt.Errorf("failed to patch expansion file: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":       pkg,
				"editId":            editID,
				"apkVersionCode":    apkVersionCode,
				"expansionFileType": expansionFileType,
				"expansionFile":     file,
				"status":            "patched",
			})
		},
	}
}

func newExpansionFilesUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, expansionFileType string
	var apkVersionCode, referencesVersion int64
	addExpansionFileSharedFlags(fs, &packageName, &editID, &apkVersionCode, &expansionFileType)
	fs.Int64Var(&referencesVersion, "references-version", 0, "APK version code whose expansion file should be referenced")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Update expansion file reference for one APK inside an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, false)
			if err != nil {
				return err
			}
			defer cancel()

			editID, expansionFileType, err = validateExpansionFileArgs(editID, apkVersionCode, expansionFileType)
			if err != nil {
				return err
			}
			if referencesVersion <= 0 {
				return shared.UsageErrorf("--references-version must be greater than zero")
			}

			file, err := client.UpdateExpansionFile(requestCtx, pkg, editID, apkVersionCode, expansionFileType, referencesVersion)
			if err != nil {
				return fmt.Errorf("failed to update expansion file: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":       pkg,
				"editId":            editID,
				"apkVersionCode":    apkVersionCode,
				"expansionFileType": expansionFileType,
				"expansionFile":     file,
				"status":            "updated",
			})
		},
	}
}

func newExpansionFilesUploadCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, expansionFileType, filePath string
	var apkVersionCode int64
	addExpansionFileSharedFlags(fs, &packageName, &editID, &apkVersionCode, &expansionFileType)
	fs.StringVar(&filePath, "file", "", "Path to expansion file to upload")

	return &ffcli.Command{
		Name:      "upload",
		ShortHelp: "Upload an expansion file for one APK inside an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName, true)
			if err != nil {
				return err
			}
			defer cancel()

			editID, expansionFileType, err = validateExpansionFileArgs(editID, apkVersionCode, expansionFileType)
			if err != nil {
				return err
			}
			filePath = strings.TrimSpace(filePath)
			if filePath == "" {
				return shared.UsageErrorf("--file is required")
			}

			file, err := client.UploadExpansionFile(requestCtx, pkg, editID, apkVersionCode, expansionFileType, filePath)
			if err != nil {
				return fmt.Errorf("failed to upload expansion file: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":       pkg,
				"editId":            editID,
				"apkVersionCode":    apkVersionCode,
				"expansionFileType": expansionFileType,
				"expansionFile":     file,
				"status":            "uploaded",
			})
		},
	}
}

func addExpansionFileSharedFlags(fs *flag.FlagSet, packageName, editID *string, apkVersionCode *int64, expansionFileType *string) {
	fs.StringVar(packageName, "package-name", "", "Package name")
	fs.StringVar(editID, "edit-id", "", "Edit ID")
	fs.Int64Var(apkVersionCode, "apk-version-code", 0, "APK version code")
	fs.StringVar(expansionFileType, "expansion-file-type", "", "Expansion file type: main or patch")
}

func validateExpansionFileArgs(editID string, apkVersionCode int64, expansionFileType string) (string, string, error) {
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return "", "", shared.UsageErrorf("--edit-id is required")
	}
	if apkVersionCode <= 0 {
		return "", "", shared.UsageErrorf("--apk-version-code must be greater than zero")
	}
	expansionFileType, err := validateExpansionFileType(expansionFileType)
	if err != nil {
		return "", "", err
	}
	return editID, expansionFileType, nil
}

func validateExpansionFileType(expansionFileType string) (string, error) {
	expansionFileType = strings.ToLower(strings.TrimSpace(expansionFileType))
	switch expansionFileType {
	case "main", "patch":
		return expansionFileType, nil
	default:
		return "", shared.UsageErrorf("--expansion-file-type must be one of: main, patch")
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
				return shared.UsageErrorf("--edit-id is required")
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
	var packageName, editID, method string
	var defaultLanguage, contactEmail, contactPhone, contactWebsite string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&method, "method", "patch", "Update method: patch or update")
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
				return shared.UsageErrorf("--edit-id is required")
			}
			method, err = parseEditMutationMethod(method)
			if err != nil {
				return err
			}
			update := gpc.AppDetailsUpdate{
				DefaultLanguage: defaultLanguage,
				ContactEmail:    contactEmail,
				ContactPhone:    contactPhone,
				ContactWebsite:  contactWebsite,
			}
			var details gpc.AppDetailsInfo
			switch method {
			case "patch":
				details, err = client.UpdateAppDetails(requestCtx, pkg, editID, update)
			case "update":
				details, err = client.ReplaceAppDetails(requestCtx, pkg, editID, update)
			}
			if err != nil {
				return fmt.Errorf("failed to update app details: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"method":      method,
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
				return shared.UsageErrorf("--edit-id is required")
			}
			track = strings.TrimSpace(track)
			if track == "" {
				return shared.UsageErrorf("--track is required")
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
	var packageName, editID, track, groupsCSV, method string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&track, "track", "", "Track name (e.g. internal, closed)")
	fs.StringVar(&groupsCSV, "google-groups", "", "Comma-separated Google Group email addresses")
	fs.StringVar(&method, "method", "patch", "Update method: patch or update")

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
				return shared.UsageErrorf("--edit-id is required")
			}
			track = strings.TrimSpace(track)
			if track == "" {
				return shared.UsageErrorf("--track is required")
			}
			googleGroups := parseCommaSeparated(groupsCSV)
			if len(googleGroups) == 0 {
				return shared.UsageErrorf("--google-groups is required")
			}
			method, err = parseEditMutationMethod(method)
			if err != nil {
				return err
			}
			var testers gpc.TestersInfo
			switch method {
			case "patch":
				testers, err = client.UpdateTesters(requestCtx, pkg, editID, track, googleGroups)
			case "update":
				testers, err = client.ReplaceTesters(requestCtx, pkg, editID, track, googleGroups)
			}
			if err != nil {
				return fmt.Errorf("failed to update testers: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"method":      method,
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
				return shared.UsageErrorf("--edit-id is required")
			}
			track = strings.TrimSpace(track)
			if track == "" {
				return shared.UsageErrorf("--track is required")
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
				return shared.UsageErrorf("--edit-id is required")
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

func parseEditMutationMethod(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "patch":
		return "patch", nil
	case "update":
		return "update", nil
	default:
		return "", shared.UsageErrorf("--method must be one of: patch, update")
	}
}

var validImageTypes = map[string]struct{}{
	"featureGraphic":       {},
	"icon":                 {},
	"phoneScreenshots":     {},
	"promoGraphic":         {},
	"sevenInchScreenshots": {},
	"tenInchScreenshots":   {},
	"tvBanner":             {},
	"tvScreenshots":        {},
	"wearScreenshots":      {},
}

const validImageTypesHelp = "featureGraphic, icon, phoneScreenshots, promoGraphic, sevenInchScreenshots, tenInchScreenshots, tvBanner, tvScreenshots, wearScreenshots"

func validateImageType(raw string) (string, error) {
	imageType := strings.TrimSpace(raw)
	if imageType == "" {
		return "", shared.UsageErrorf("--image-type is required")
	}
	if _, ok := validImageTypes[imageType]; !ok {
		return "", shared.UsageErrorf("--image-type must be one of: %s", validImageTypesHelp)
	}
	return imageType, nil
}

func validateImageUploadFile(imageType, imagePath string) error {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(imagePath)))
	switch ext {
	case ".png", ".jpg", ".jpeg":
	default:
		return shared.UsageErrorf("--file must use one of: .png, .jpg, .jpeg")
	}

	stat, err := os.Stat(imagePath)
	if err != nil {
		return fmt.Errorf("stat --file: %w", err)
	}
	if stat.IsDir() {
		return shared.UsageErrorf("--file must point to an image file, got directory")
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("open --file: %w", err)
	}
	defer file.Close()

	cfg, format, err := image.DecodeConfig(file)
	if err != nil {
		return shared.UsageErrorf("--file must be a valid PNG or JPEG image: %v", err)
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "png" && format != "jpeg" {
		return shared.UsageErrorf("--file must be PNG or JPEG, got %q", format)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return shared.UsageErrorf("--file has invalid dimensions %dx%d", cfg.Width, cfg.Height)
	}

	if imageType == "icon" {
		if format != "png" {
			return shared.UsageErrorf("--image-type icon requires a PNG file")
		}
		if cfg.Width != 512 || cfg.Height != 512 {
			return shared.UsageErrorf("--image-type icon requires dimensions 512x512, got %dx%d", cfg.Width, cfg.Height)
		}
	}
	if imageType == "featureGraphic" && (cfg.Width != 1024 || cfg.Height != 500) {
		return shared.UsageErrorf("--image-type featureGraphic requires dimensions 1024x500, got %dx%d", cfg.Width, cfg.Height)
	}
	if imageType == "tvBanner" && (cfg.Width != 1280 || cfg.Height != 720) {
		return shared.UsageErrorf("--image-type tvBanner requires dimensions 1280x720, got %dx%d", cfg.Width, cfg.Height)
	}
	if isScreenshotImageType(imageType) && (cfg.Width < 320 || cfg.Width > 3840 || cfg.Height < 320 || cfg.Height > 3840) {
		return shared.UsageErrorf("--image-type %s requires width/height in range 320-3840, got %dx%d", imageType, cfg.Width, cfg.Height)
	}

	return nil
}

func ValidateImageUploadFile(imageType, imagePath string) error {
	return validateImageUploadFile(imageType, imagePath)
}

func collectImageUploadFiles(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("stat --dir: %w", err)
	}
	if !info.IsDir() {
		return nil, shared.UsageErrorf("--dir must point to a directory")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read --dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			return nil, shared.UsageErrorf("--dir must contain image files only; nested directory %q is not supported", entry.Name())
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		switch ext {
		case ".png", ".jpg", ".jpeg":
			files = append(files, filepath.Join(dir, entry.Name()))
		default:
			return nil, shared.UsageErrorf("--dir contains unsupported file %q; only .png, .jpg, and .jpeg are allowed", entry.Name())
		}
	}

	sort.Strings(files)
	if len(files) == 0 {
		return nil, shared.UsageErrorf("--dir must contain at least one image file")
	}
	return files, nil
}

func isScreenshotImageType(imageType string) bool {
	switch imageType {
	case "phoneScreenshots", "sevenInchScreenshots", "tenInchScreenshots", "tvScreenshots", "wearScreenshots":
		return true
	default:
		return false
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
				return shared.UsageErrorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return shared.UsageErrorf("--locale is required")
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
	var packageName, editID, locale, title, shortDescription, fullDescription, method string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&locale, "locale", "", "Listing locale (BCP-47, e.g. en-US)")
	fs.StringVar(&method, "method", "patch", "Update method: patch or update")
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
				return shared.UsageErrorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return shared.UsageErrorf("--locale is required")
			}
			method, err = parseEditMutationMethod(method)
			if err != nil {
				return err
			}
			update := gpc.ListingUpdate{
				Title:            title,
				ShortDescription: shortDescription,
				FullDescription:  fullDescription,
			}
			var listing gpc.ListingInfo
			switch method {
			case "patch":
				listing, err = client.UpdateListing(requestCtx, pkg, editID, locale, update)
			case "update":
				listing, err = client.ReplaceListing(requestCtx, pkg, editID, locale, update)
			}
			if err != nil {
				return fmt.Errorf("failed to update listing: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      editID,
				"method":      method,
				"listing":     listing,
				"status":      "updated",
			})
		},
	}
}

type listingBatchInput struct {
	Title            string `json:"title,omitempty"`
	ShortDescription string `json:"shortDescription,omitempty"`
	FullDescription  string `json:"fullDescription,omitempty"`
}

type listingBatchItem struct {
	Locale string
	Input  listingBatchInput
}

type listingBatchResult struct {
	Locale  string            `json:"locale"`
	Status  string            `json:"status"`
	Input   listingBatchInput `json:"input,omitempty"`
	Listing *gpc.ListingInfo  `json:"listing,omitempty"`
	Error   string            `json:"error,omitempty"`
}

func newListingsBatchUpdateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("batch-update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, fromDir, localesCSV string
	var dryRun, continueOnError bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&fromDir, "from-dir", "", "Directory containing per-locale JSON files (<locale>.json)")
	fs.StringVar(&localesCSV, "locales", "", "Optional comma-separated locale filter")
	fs.BoolVar(&dryRun, "dry-run", false, "Preview updates without calling the API")
	fs.BoolVar(&continueOnError, "continue-on-error", true, "Continue processing locales after errors")

	return &ffcli.Command{
		Name:      "batch-update",
		ShortHelp: "Batch update listing fields from per-locale JSON files",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			pkg, err := shared.ResolvePackageName(packageName)
			if err != nil {
				return err
			}
			editID = strings.TrimSpace(editID)
			if editID == "" {
				return shared.UsageErrorf("--edit-id is required")
			}
			fromDir = strings.TrimSpace(fromDir)
			if fromDir == "" {
				return shared.UsageErrorf("--from-dir is required")
			}

			filter := parseCommaSeparated(localesCSV)
			items, err := loadListingBatchItems(fromDir, filter)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return shared.UsageErrorf("no locale JSON payloads found in %s", fromDir)
			}

			var (
				client     Client
				requestCtx context.Context
				cancel     context.CancelFunc
			)
			if !dryRun {
				client, _, requestCtx, cancel, err = buildClient(ctx, deps, packageName, false)
				if err != nil {
					return err
				}
				defer cancel()
			}

			results := make([]listingBatchResult, 0, len(items))
			failed := 0

			for _, item := range items {
				result := listingBatchResult{
					Locale: item.Locale,
					Input:  item.Input,
				}

				update := gpc.ListingUpdate{
					Title:            item.Input.Title,
					ShortDescription: item.Input.ShortDescription,
					FullDescription:  item.Input.FullDescription,
				}

				if update.Title == "" && update.ShortDescription == "" && update.FullDescription == "" {
					result.Status = "error"
					result.Error = "at least one listing field must be provided"
					failed++
					results = append(results, result)
					if !continueOnError {
						break
					}
					continue
				}

				if dryRun {
					result.Status = "planned"
					results = append(results, result)
					continue
				}

				listing, updateErr := client.UpdateListing(requestCtx, pkg, editID, item.Locale, update)
				if updateErr != nil {
					result.Status = "error"
					result.Error = updateErr.Error()
					failed++
					results = append(results, result)
					if !continueOnError {
						break
					}
					continue
				}

				result.Status = "updated"
				result.Listing = &listing
				results = append(results, result)
			}

			payload := map[string]any{
				"packageName":       pkg,
				"editId":            editID,
				"fromDir":           fromDir,
				"dryRun":            dryRun,
				"continueOnError":   continueOnError,
				"results":           results,
				"failedLocaleCount": failed,
			}
			if err := shared.WriteJSON(deps.Stdout, payload); err != nil {
				return err
			}
			if failed > 0 {
				return fmt.Errorf("%d listing locale updates failed", failed)
			}
			return nil
		},
	}
}

func loadListingBatchItems(fromDir string, filter []string) ([]listingBatchItem, error) {
	entries, err := os.ReadDir(fromDir)
	if err != nil {
		return nil, shared.UsageErrorf("failed to read --from-dir %q: %v", fromDir, err)
	}

	filterSet := make(map[string]struct{}, len(filter))
	for _, locale := range filter {
		filterSet[locale] = struct{}{}
	}

	items := make([]listingBatchItem, 0, len(entries))
	foundByLocale := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}

		locale := strings.TrimSpace(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		if locale == "" {
			continue
		}
		if len(filterSet) > 0 {
			if _, ok := filterSet[locale]; !ok {
				continue
			}
		}

		path := filepath.Join(fromDir, entry.Name())
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, shared.UsageErrorf("failed to read %s: %v", path, readErr)
		}

		var input listingBatchInput
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, shared.UsageErrorf("invalid JSON in %s: %v", path, err)
		}
		input.Title = strings.TrimSpace(input.Title)
		input.ShortDescription = strings.TrimSpace(input.ShortDescription)
		input.FullDescription = strings.TrimSpace(input.FullDescription)

		items = append(items, listingBatchItem{
			Locale: locale,
			Input:  input,
		})
		foundByLocale[locale] = struct{}{}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Locale < items[j].Locale
	})

	if len(filterSet) > 0 {
		missing := make([]string, 0)
		for locale := range filterSet {
			if _, ok := foundByLocale[locale]; !ok {
				missing = append(missing, locale)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			return nil, shared.UsageErrorf("requested locales not found in --from-dir: %s", strings.Join(missing, ","))
		}
	}

	return items, nil
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
				return shared.UsageErrorf("--edit-id is required")
			}
			locale = strings.TrimSpace(locale)
			if locale == "" {
				return shared.UsageErrorf("--locale is required")
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
				return shared.UsageErrorf("--edit-id is required")
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
