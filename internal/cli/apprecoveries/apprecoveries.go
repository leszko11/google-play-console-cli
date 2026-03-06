package apprecoveries

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
	ListAppRecoveries(ctx context.Context, packageName string, versionCode int64) (*androidpublisher.ListAppRecoveriesResponse, error)
	CreateAppRecovery(ctx context.Context, packageName string, request *androidpublisher.CreateDraftAppRecoveryRequest) (*androidpublisher.AppRecoveryAction, error)
	AddAppRecoveryTargeting(ctx context.Context, packageName string, appRecoveryID int64, request *androidpublisher.AddTargetingRequest) error
	CancelAppRecovery(ctx context.Context, packageName string, appRecoveryID int64) error
	DeployAppRecovery(ctx context.Context, packageName string, appRecoveryID int64) error
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
		Name:      "app-recoveries",
		ShortHelp: "Manage Play app recovery actions",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newCreateCommand(deps),
			newAddTargetingCommand(deps),
			newCancelCommand(deps),
			newDeployCommand(deps),
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
	fs.Int64Var(&versionCode, "version-code", 0, "Version code targeted by the recovery actions")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List app recovery actions for one version code",
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

			resp, err := client.ListAppRecoveries(requestCtx, pkg, versionCode)
			if err != nil {
				return fmt.Errorf("failed to list app recoveries: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":     pkg,
				"versionCode":     versionCode,
				"recoveryActions": resp.RecoveryActions,
			})
		},
	}
}

func newCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&inputPath, "input", "", "Path to app recovery JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a draft app recovery action",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			request, err := readCreatePayload(inputPath, deps.Stdin)
			if err != nil {
				return err
			}

			action, err := client.CreateAppRecovery(requestCtx, pkg, request)
			if err != nil {
				return fmt.Errorf("failed to create app recovery: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName": pkg,
				"recovery":    action,
			})
		},
	}
}

func newAddTargetingCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("add-targeting", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, inputPath string
	var appRecoveryID int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&appRecoveryID, "app-recovery-id", 0, "App recovery action ID")
	fs.StringVar(&inputPath, "input", "", "Path to targeting update JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "add-targeting",
		ShortHelp: "Add targeting to an app recovery action",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			if err := validateAppRecoveryID(appRecoveryID); err != nil {
				return err
			}

			request, err := readAddTargetingPayload(inputPath, deps.Stdin)
			if err != nil {
				return err
			}

			if err := client.AddAppRecoveryTargeting(requestCtx, pkg, appRecoveryID, request); err != nil {
				return fmt.Errorf("failed to add app recovery targeting: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"appRecoveryId": appRecoveryID,
				"status":        "targeting-updated",
			})
		},
	}
}

func newCancelCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName string
	var appRecoveryID int64
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&appRecoveryID, "app-recovery-id", 0, "App recovery action ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm canceling the app recovery action (required)")

	return &ffcli.Command{
		Name:      "cancel",
		ShortHelp: "Cancel an app recovery action",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			if err := validateAppRecoveryID(appRecoveryID); err != nil {
				return err
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to cancel app recovery %q", fmt.Sprint(appRecoveryID))
			}

			if err := client.CancelAppRecovery(requestCtx, pkg, appRecoveryID); err != nil {
				return fmt.Errorf("failed to cancel app recovery: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"appRecoveryId": appRecoveryID,
				"status":        "canceled",
			})
		},
	}
}

func newDeployCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName string
	var appRecoveryID int64
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&appRecoveryID, "app-recovery-id", 0, "App recovery action ID")
	fs.BoolVar(&confirm, "confirm", false, "Confirm deploying the app recovery action (required)")

	return &ffcli.Command{
		Name:      "deploy",
		ShortHelp: "Deploy an app recovery action",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			if err := validateAppRecoveryID(appRecoveryID); err != nil {
				return err
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to deploy app recovery %q", fmt.Sprint(appRecoveryID))
			}

			if err := client.DeployAppRecovery(requestCtx, pkg, appRecoveryID); err != nil {
				return fmt.Errorf("failed to deploy app recovery: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":   pkg,
				"appRecoveryId": appRecoveryID,
				"status":        "deployed",
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

func validateAppRecoveryID(appRecoveryID int64) error {
	if appRecoveryID <= 0 {
		return fmt.Errorf("--app-recovery-id must be greater than zero")
	}
	return nil
}

func readCreatePayload(inputPath string, stdin io.Reader) (*androidpublisher.CreateDraftAppRecoveryRequest, error) {
	var request androidpublisher.CreateDraftAppRecoveryRequest
	if err := readJSONPayload(inputPath, stdin, &request); err != nil {
		return nil, fmt.Errorf("invalid app recovery JSON payload: %w", err)
	}
	return &request, nil
}

func readAddTargetingPayload(inputPath string, stdin io.Reader) (*androidpublisher.AddTargetingRequest, error) {
	var request androidpublisher.AddTargetingRequest
	if err := readJSONPayload(inputPath, stdin, &request); err != nil {
		return nil, fmt.Errorf("invalid app recovery targeting JSON payload: %w", err)
	}
	return &request, nil
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
