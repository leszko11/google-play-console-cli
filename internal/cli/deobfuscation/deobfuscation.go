package deobfuscation

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

const (
	deobfuscationTypeProguard = "proguard"
	deobfuscationTypeNative   = "nativeCode"
)

type Client interface {
	UploadDeobfuscationFile(ctx context.Context, packageName, editID string, versionCode int64, fileType, filePath string) (gpc.DeobfuscationFileInfo, error)
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
		Name:      "deobfuscation",
		ShortHelp: "Manage deobfuscation files in an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newUploadCommand(deps),
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

func newUploadCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var (
		packageName string
		editID      string
		versionCode int64
		fileType    string
		filePath    string
	)

	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.Int64Var(&versionCode, "version-code", 0, "Version code associated with the mapping file")
	fs.StringVar(&fileType, "type", "", "Deobfuscation file type: proguard or nativeCode")
	fs.StringVar(&filePath, "file", "", "Path to deobfuscation file")

	return &ffcli.Command{
		Name:      "upload",
		ShortHelp: "Upload a deobfuscation file to an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			pkg, err := shared.ResolvePackageName(packageName)
			if err != nil {
				return err
			}

			editID = strings.TrimSpace(editID)
			if editID == "" {
				return fmt.Errorf("--edit-id is required")
			}
			if versionCode <= 0 {
				return fmt.Errorf("--version-code must be greater than zero")
			}

			fileType = strings.TrimSpace(fileType)
			if fileType == "" {
				return fmt.Errorf("--type is required")
			}
			if fileType != deobfuscationTypeProguard && fileType != deobfuscationTypeNative {
				return fmt.Errorf("--type must be one of: %s, %s", deobfuscationTypeProguard, deobfuscationTypeNative)
			}

			filePath = strings.TrimSpace(filePath)
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}
			if err := validateReadableFile(filePath); err != nil {
				return err
			}

			client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
				Upload:     true,
			})
			if err != nil {
				return err
			}
			defer cancel()

			spinner := shared.NewSpinner(deps.Stderr, "Uploading deobfuscation file")
			result, err := client.UploadDeobfuscationFile(requestCtx, pkg, editID, versionCode, fileType, filePath)
			if err != nil {
				spinner.Fail("Deobfuscation upload failed")
				return fmt.Errorf("failed to upload deobfuscation file: %w", err)
			}
			spinner.Success("Deobfuscation file uploaded")

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":       pkg,
				"editId":            editID,
				"versionCode":       versionCode,
				"type":              fileType,
				"deobfuscationFile": result,
				"status":            "uploaded",
			})
		},
	}
}

func validateReadableFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("--file does not exist: %s", path)
		}
		return fmt.Errorf("failed to stat --file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("--file must be a file, got directory: %s", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("--file is not readable: %w", err)
	}
	return f.Close()
}
