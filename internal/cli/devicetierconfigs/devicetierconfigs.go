package devicetierconfigs

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
	ListDeviceTierConfigs(ctx context.Context, packageName string, pageSize int64, pageToken string, paginate bool) (*androidpublisher.ListDeviceTierConfigsResponse, error)
	GetDeviceTierConfig(ctx context.Context, packageName string, deviceTierConfigID int64) (*androidpublisher.DeviceTierConfig, error)
	CreateDeviceTierConfig(ctx context.Context, packageName string, config *androidpublisher.DeviceTierConfig, allowUnknownDevices bool) (*androidpublisher.DeviceTierConfig, error)
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
		Name:      "device-tier-configs",
		ShortHelp: "Manage application device tier configs",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newListCommand(deps),
			newGetCommand(deps),
			newCreateCommand(deps),
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
	var packageName, pageToken string
	var pageSize int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&pageSize, "page-size", 0, "Maximum device tier configs per page")
	fs.StringVar(&pageToken, "page-token", "", "Page token for the next page")

	return &ffcli.Command{
		Name:      "list",
		ShortHelp: "List device tier configs",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			if pageSize < 0 {
				return fmt.Errorf("--page-size must be greater than or equal to zero")
			}

			resp, err := client.ListDeviceTierConfigs(requestCtx, pkg, pageSize, pageToken, shared.ActiveGlobalFlags().Paginate)
			if err != nil {
				return fmt.Errorf("failed to list device tier configs: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":       pkg,
				"deviceTierConfigs": resp.DeviceTierConfigs,
				"nextPageToken":     resp.NextPageToken,
			})
		},
	}
}

func newGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName string
	var deviceTierConfigID int64
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.Int64Var(&deviceTierConfigID, "device-tier-config-id", 0, "Device tier config ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get a device tier config by ID",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			if deviceTierConfigID <= 0 {
				return fmt.Errorf("--device-tier-config-id must be greater than zero")
			}

			config, err := client.GetDeviceTierConfig(requestCtx, pkg, deviceTierConfigID)
			if err != nil {
				return fmt.Errorf("failed to get device tier config: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"deviceTierConfig": config,
			})
		},
	}
}

func newCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, inputPath string
	var allowUnknownDevices bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&inputPath, "input", "", "Path to device tier config JSON payload (use - for stdin)")
	fs.BoolVar(&allowUnknownDevices, "allow-unknown-devices", false, "Allow device IDs unknown to Play's device catalog")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create a device tier config",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			config, err := readDeviceTierConfigPayload(inputPath, deps.Stdin)
			if err != nil {
				return err
			}

			created, err := client.CreateDeviceTierConfig(requestCtx, pkg, config, allowUnknownDevices)
			if err != nil {
				return fmt.Errorf("failed to create device tier config: %w", err)
			}
			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":      pkg,
				"deviceTierConfig": created,
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

func readDeviceTierConfigPayload(inputPath string, stdin io.Reader) (*androidpublisher.DeviceTierConfig, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return nil, fmt.Errorf("--input is required")
	}

	var raw []byte
	var err error
	if inputPath == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		raw, err = io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read --input from stdin: %w", err)
		}
	} else {
		raw, err = os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read --input: %w", err)
		}
	}

	var config androidpublisher.DeviceTierConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("invalid device tier config JSON payload: %w", err)
	}
	return &config, nil
}
