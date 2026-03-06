package systemapks

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

type Client interface {
	ListSystemAPKVariants(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.SystemApksListResponse, error)
	GetSystemAPKVariant(ctx context.Context, packageName string, versionCode, variantID int64) (*androidpublisher.Variant, error)
	CreateSystemAPKVariant(ctx context.Context, packageName string, versionCode int64, variant *androidpublisher.Variant) (*androidpublisher.Variant, error)
	DownloadSystemAPKVariant(ctx context.Context, packageName string, versionCode, variantID int64) ([]byte, error)
}

type Deps struct {
	LoadConfig func() (config.Config, error)
	NewClient  func(context.Context, gpc.CredentialInput) (Client, error)
	LookupEnv  func(string) string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "system-apks",
		ShortHelp: "Manage generated system APK variants",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newGetCommand(deps),
			newCreateCommand(deps),
			newDownloadCommand(deps),
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
	if deps.Stdin == nil {
		deps.Stdin = os.Stdin
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return deps
}

func newListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName string
	var versionCode int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&versionCode, "version-code", 0, "Version code of the App Bundle")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List generated system APK variants",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			if err := validateVersionCode(versionCode); err != nil {
				return err
			}

			resp, err := client.ListSystemAPKVariants(requestCtx, pkg, versionCode)
			if err != nil {
				return fmt.Errorf("failed to list system apk variants: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"versionCode": versionCode,
				"variants":    resp.Variants,
			})
		},
	}
}

func newGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName string
	var versionCode, variantID int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&versionCode, "version-code", 0, "Version code of the App Bundle")
	fs.Int64Var(&variantID, "variant-id", 0, "System APK variant ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get a generated system APK variant",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			if err := validateVersionCode(versionCode); err != nil {
				return err
			}
			if err := validateVariantID(variantID); err != nil {
				return err
			}

			variant, err := client.GetSystemAPKVariant(requestCtx, pkg, versionCode, variantID)
			if err != nil {
				return fmt.Errorf("failed to get system apk variant: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"versionCode": versionCode,
				"variant":     variant,
			})
		},
	}
}

func newCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, inputPath string
	var versionCode int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&versionCode, "version-code", 0, "Version code of the App Bundle")
	fs.StringVar(&inputPath, "input", "", "Path to system APK variant JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a generated system APK variant",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			if err := validateVersionCode(versionCode); err != nil {
				return err
			}

			variant, err := readVariantPayload(inputPath, deps.Stdin)
			if err != nil {
				return err
			}

			created, err := client.CreateSystemAPKVariant(requestCtx, pkg, versionCode, variant)
			if err != nil {
				return fmt.Errorf("failed to create system apk variant: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"versionCode": versionCode,
				"variant":     created,
			})
		},
	}
}

func newDownloadCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, outputPath string
	var versionCode, variantID int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&versionCode, "version-code", 0, "Version code of the App Bundle")
	fs.Int64Var(&variantID, "variant-id", 0, "System APK variant ID")
	fs.StringVar(&outputPath, "output", "", "Path to write the downloaded APK")

	return &ffcli.Command{
		Name:      "download",
		ShortHelp: "Download a generated system APK variant",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			if err := validateVersionCode(versionCode); err != nil {
				return err
			}
			if err := validateVariantID(variantID); err != nil {
				return err
			}

			outputPath = strings.TrimSpace(outputPath)
			if outputPath == "" {
				return fmt.Errorf("--output is required")
			}

			raw, err := client.DownloadSystemAPKVariant(requestCtx, pkg, versionCode, variantID)
			if err != nil {
				return fmt.Errorf("failed to download system apk variant: %w", err)
			}

			if err := writeDownload(outputPath, raw); err != nil {
				return err
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"versionCode": versionCode,
				"variantId":   variantID,
				"outputPath":  outputPath,
				"sizeBytes":   len(raw),
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName string) (Client, string, context.Context, context.CancelFunc, error) {
	pkg, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return nil, "", nil, nil, err
	}

	client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
		Upload:     false,
	})
	if err != nil {
		return nil, "", nil, nil, err
	}
	return client, pkg, requestCtx, cancel, nil
}

func validateVersionCode(versionCode int64) error {
	if versionCode <= 0 {
		return fmt.Errorf("--version-code must be greater than zero")
	}
	return nil
}

func validateVariantID(variantID int64) error {
	if variantID <= 0 {
		return fmt.Errorf("--variant-id must be greater than zero")
	}
	return nil
}

func readVariantPayload(inputPath string, stdin io.Reader) (*androidpublisher.Variant, error) {
	var variant androidpublisher.Variant
	if err := readJSONPayload(inputPath, stdin, &variant); err != nil {
		return nil, fmt.Errorf("invalid system APK variant JSON payload: %w", err)
	}
	return &variant, nil
}

func readJSONPayload(inputPath string, stdin io.Reader, dst any) error {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return fmt.Errorf("--input is required")
	}

	var raw []byte
	var err error
	if inputPath == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		raw, err = io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("failed to read --input from stdin: %w", err)
		}
	} else {
		raw, err = os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("failed to read --input: %w", err)
		}
	}

	if err := json.Unmarshal(raw, dst); err != nil {
		return err
	}
	return nil
}

func writeDownload(outputPath string, raw []byte) error {
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("refusing to overwrite existing file %q", outputPath)
		}
		return fmt.Errorf("failed to open --output: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(raw); err != nil {
		return fmt.Errorf("failed to write --output: %w", err)
	}
	return nil
}
