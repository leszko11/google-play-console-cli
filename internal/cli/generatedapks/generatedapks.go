package generatedapks

import (
	"context"
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
	ListGeneratedAPKs(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.GeneratedApksListResponse, error)
	DownloadGeneratedAPK(ctx context.Context, packageName string, versionCode int64, downloadID string) ([]byte, error)
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
		Name:      "generated-apks",
		ShortHelp: "List and download APKs generated from bundles",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
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
		ShortHelp: "List generated APK download metadata",
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

			resp, err := client.ListGeneratedAPKs(requestCtx, pkg, versionCode)
			if err != nil {
				return fmt.Errorf("failed to list generated apks: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"versionCode":   versionCode,
				"generatedApks": resp.GeneratedApks,
			})
		},
	}
}

func newDownloadCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, downloadID, outputPath string
	var versionCode int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&versionCode, "version-code", 0, "Version code of the App Bundle")
	fs.StringVar(&downloadID, "download-id", "", "Generated APK download ID")
	fs.StringVar(&outputPath, "output", "", "Path to write the downloaded APK")

	return &ffcli.Command{
		Name:      "download",
		ShortHelp: "Download one generated APK by download ID",
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
			downloadID = strings.TrimSpace(downloadID)
			if downloadID == "" {
				return fmt.Errorf("--download-id is required")
			}
			outputPath = strings.TrimSpace(outputPath)
			if outputPath == "" {
				return fmt.Errorf("--output is required")
			}

			raw, err := client.DownloadGeneratedAPK(requestCtx, pkg, versionCode, downloadID)
			if err != nil {
				return fmt.Errorf("failed to download generated apk: %w", err)
			}

			if err := writeDownload(outputPath, raw); err != nil {
				return err
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"versionCode": versionCode,
				"downloadId":  downloadID,
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
