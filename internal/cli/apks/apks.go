package apks

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
)

type Client interface {
	ListAPKs(ctx context.Context, packageName, editID string) ([]gpc.APKInfo, error)
	UploadAPK(ctx context.Context, packageName, editID, apkPath string) (gpc.APKInfo, error)
	AddExternallyHostedAPK(ctx context.Context, packageName, editID string, apk *androidpublisher.ExternallyHostedApk) (gpc.ExternallyHostedAPKInfo, error)
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
		Name:      "apks",
		ShortHelp: "Manage APK uploads in an edit",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newUploadCommand(deps),
			newAddExternallyHostedCommand(deps),
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
		ShortHelp: "List APKs in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, false)
			if err != nil {
				return err
			}
			defer cancel()
			apks, err := client.ListAPKs(requestCtx, pkg, eid)
			if err != nil {
				return fmt.Errorf("failed to list apks: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"apks":        apks,
			})
		},
	}
}

func newUploadCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, apkPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&apkPath, "file", "", "Path to .apk file")

	return &ffcli.Command{
		Name:      "upload",
		ShortHelp: "Upload an APK to an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, true)
			if err != nil {
				return err
			}
			defer cancel()
			apkPath = strings.TrimSpace(apkPath)
			if apkPath == "" {
				return fmt.Errorf("--file is required")
			}
			apk, err := client.UploadAPK(requestCtx, pkg, eid, apkPath)
			if err != nil {
				return fmt.Errorf("failed to upload apk: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"apk":         apk,
				"status":      "uploaded",
			})
		},
	}
}

func newAddExternallyHostedCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("add-externally-hosted", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, editID, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&editID, "edit-id", "", "Edit ID")
	fs.StringVar(&inputPath, "input", "", "Path to externally hosted APK JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "add-externally-hosted",
		ShortHelp: "Register an externally hosted APK in an edit",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, eid, requestCtx, cancel, err := buildClient(ctx, deps, packageName, editID, false)
			if err != nil {
				return err
			}
			defer cancel()

			apk, err := readExternallyHostedAPKPayload(inputPath, os.Stdin)
			if err != nil {
				return err
			}
			result, err := client.AddExternallyHostedAPK(requestCtx, pkg, eid, apk)
			if err != nil {
				return fmt.Errorf("failed to add externally hosted apk: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"editId":      eid,
				"apk":         result,
				"status":      "added",
			})
		},
	}
}

func readExternallyHostedAPKPayload(path string, stdin io.Reader) (*androidpublisher.ExternallyHostedApk, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("--input is required")
	}

	var (
		raw []byte
		err error
	)
	if path == "-" {
		raw, err = io.ReadAll(stdin)
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read externally hosted apk payload: %w", err)
	}

	var payload androidpublisher.ExternallyHostedApk
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode externally hosted apk payload: %w", err)
	}
	return &payload, nil
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
