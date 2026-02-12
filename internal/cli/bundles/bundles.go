package bundles

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
	ListBundles(ctx context.Context, packageName, editID string) ([]gpc.BundleInfo, error)
	UploadBundle(ctx context.Context, packageName, editID, bundlePath string) (gpc.BundleInfo, error)
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
		Name:      "bundles",
		ShortHelp: "Manage Android App Bundles in an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
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

func newListCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List bundles in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, err := buildClient(ctx, deps, packageName, editID)
			if err != nil {
				return err
			}
			bundles, err := client.ListBundles(ctx, pkg, eid)
			if err != nil {
				return fmt.Errorf("failed to list bundles: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"bundles":     bundles,
			})
		},
	}
}

func newUploadCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, bundlePath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&bundlePath, "file", "", "Path to .aab file")

	return &ffcli.Command{
		Name:      "upload",
		ShortHelp: "Upload an Android App Bundle to an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, err := buildClient(ctx, deps, packageName, editID)
			if err != nil {
				return err
			}
			bundlePath = strings.TrimSpace(bundlePath)
			if bundlePath == "" {
				return fmt.Errorf("--file is required")
			}
			bundle, err := client.UploadBundle(ctx, pkg, eid, bundlePath)
			if err != nil {
				return fmt.Errorf("failed to upload bundle: %w", err)
			}
			return writeJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"bundle":      bundle,
				"status":      "uploaded",
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName, editID string) (Client, string, string, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, "", "", fmt.Errorf("--package-name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, "", "", fmt.Errorf("--edit-id is required")
	}

	cfg, err := deps.LoadConfig()
	if err != nil {
		return nil, "", "", err
	}
	serviceAccountPath, err := resolveServiceAccountPath(cfg, deps.LookupEnv)
	if err != nil {
		return nil, "", "", err
	}

	client, err := deps.NewClient(ctx, gpc.CredentialInput{ServiceAccountPath: serviceAccountPath})
	if err != nil {
		return nil, "", "", err
	}
	return client, packageName, editID, nil
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
