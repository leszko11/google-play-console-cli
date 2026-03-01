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
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, false)
			if err != nil {
				return err
			}
			defer cancel()
			bundles, err := client.ListBundles(requestCtx, pkg, eid)
			if err != nil {
				return fmt.Errorf("failed to list bundles: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
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
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, true)
			if err != nil {
				return err
			}
			defer cancel()
			bundlePath = strings.TrimSpace(bundlePath)
			if bundlePath == "" {
				return fmt.Errorf("--file is required")
			}
			bundle, err := client.UploadBundle(requestCtx, pkg, eid, bundlePath)
			if err != nil {
				return fmt.Errorf("failed to upload bundle: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"bundle":      bundle,
				"status":      "uploaded",
			})
		},
	}
}

func buildClient(ctx context.Context, deps Deps, packageName, editID string, upload bool) (Client, string, string, context.Context, context.CancelFunc, error) {
	pkg, err := shared.ResolvePackageName(packageName)
	if err != nil {
		return nil, "", "", nil, nil, err
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, "", "", nil, nil, fmt.Errorf("--edit-id is required")
	}

	client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
		LoadConfig: deps.LoadConfig,
		LookupEnv:  deps.LookupEnv,
		NewClient:  deps.NewClient,
		Upload:     upload,
	})
	if err != nil {
		return nil, "", "", nil, nil, err
	}

	return client, pkg, editID, requestCtx, cancel, nil
}
