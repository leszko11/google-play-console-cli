package internalsharing

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
	UploadInternalSharingAPK(ctx context.Context, packageName, apkPath string) (gpc.InternalSharingArtifactInfo, error)
	UploadInternalSharingBundle(ctx context.Context, packageName, bundlePath string) (gpc.InternalSharingArtifactInfo, error)
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
		Name:      "internal-sharing",
		ShortHelp: "Upload artifacts for internal app sharing",
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

	var packageName, apkPath, aabPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&apkPath, "apk", "", "Path to .apk file")
	fs.StringVar(&aabPath, "aab", "", "Path to .aab file")

	return &ffcli.Command{
		Name:      "upload",
		ShortHelp: "Upload one APK or AAB for internal app sharing",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			apkPath = strings.TrimSpace(apkPath)
			aabPath = strings.TrimSpace(aabPath)
			if (apkPath == "" && aabPath == "") || (apkPath != "" && aabPath != "") {
				return fmt.Errorf("exactly one of --apk or --aab is required")
			}

			var (
				artifactType string
				artifact     gpc.InternalSharingArtifactInfo
			)
			spinner := shared.NewSpinner(deps.Stderr, "Uploading internal sharing artifact")

			if apkPath != "" {
				if err := validateReadableFile(apkPath, "--apk"); err != nil {
					return err
				}
				artifact, err = client.UploadInternalSharingAPK(requestCtx, pkg, apkPath)
				artifactType = "apk"
			} else {
				if err := validateReadableFile(aabPath, "--aab"); err != nil {
					return err
				}
				artifact, err = client.UploadInternalSharingBundle(requestCtx, pkg, aabPath)
				artifactType = "aab"
			}
			if err != nil {
				spinner.Fail("Internal sharing upload failed")
				return fmt.Errorf("failed to upload internal sharing artifact: %w", err)
			}
			spinner.Success("Internal sharing artifact uploaded")

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":  pkg,
				"artifactType": artifactType,
				"artifact":     artifact,
				"status":       "uploaded",
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
		Upload:     true,
	})
	if err != nil {
		return nil, "", nil, nil, err
	}
	return client, pkg, requestCtx, cancel, nil
}

func validateReadableFile(path, flagName string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist: %s", flagName, path)
		}
		return fmt.Errorf("failed to stat %s: %w", flagName, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s must be a file, got directory: %s", flagName, path)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s is not readable: %w", flagName, err)
	}
	return file.Close()
}
