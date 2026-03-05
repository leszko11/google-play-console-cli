package externaltransactions

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
	GetExternalTransaction(ctx context.Context, packageName, externalTransactionID string) (*androidpublisher.ExternalTransaction, error)
	CreateExternalTransaction(ctx context.Context, packageName, externalTransactionID string, transaction *androidpublisher.ExternalTransaction) (*androidpublisher.ExternalTransaction, error)
	RefundExternalTransaction(ctx context.Context, packageName, externalTransactionID string, request *androidpublisher.RefundExternalTransactionRequest) (*androidpublisher.ExternalTransaction, error)
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
		Name:      "external-transactions",
		ShortHelp: "Report and refund external transactions",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newGetCommand(deps),
			newCreateCommand(deps),
			newRefundCommand(deps),
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

func newGetCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, externalTransactionID string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&externalTransactionID, "external-transaction-id", "", "External transaction ID")

	return &ffcli.Command{
		Name:      "get",
		ShortHelp: "Get an external transaction by ID",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			externalTransactionID = strings.TrimSpace(externalTransactionID)
			if externalTransactionID == "" {
				return fmt.Errorf("--external-transaction-id is required")
			}

			transaction, err := client.GetExternalTransaction(requestCtx, pkg, externalTransactionID)
			if err != nil {
				return fmt.Errorf("failed to get external transaction: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":         pkg,
				"externalTransaction": transaction,
			})
		},
	}
}

func newCreateCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, externalTransactionID, inputPath string
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&externalTransactionID, "external-transaction-id", "", "External transaction ID")
	fs.StringVar(&inputPath, "input", "", "Path to external transaction JSON payload (use - for stdin)")

	return &ffcli.Command{
		Name:      "create",
		ShortHelp: "Create an external transaction report",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			externalTransactionID = strings.TrimSpace(externalTransactionID)
			if externalTransactionID == "" {
				return fmt.Errorf("--external-transaction-id is required")
			}

			transaction, err := readExternalTransactionPayload(inputPath, deps.Stdin)
			if err != nil {
				return err
			}

			created, err := client.CreateExternalTransaction(requestCtx, pkg, externalTransactionID, transaction)
			if err != nil {
				return fmt.Errorf("failed to create external transaction: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":         pkg,
				"externalTransaction": created,
			})
		},
	}
}

func newRefundCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("refund", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var packageName, externalTransactionID, inputPath string
	var confirm bool
	fs.StringVar(&packageName, "package-name", "", "Package name")
	fs.StringVar(&externalTransactionID, "external-transaction-id", "", "External transaction ID")
	fs.StringVar(&inputPath, "input", "", "Path to refund request JSON payload (use - for stdin)")
	fs.BoolVar(&confirm, "confirm", false, "Confirm refunding the external transaction (required)")

	return &ffcli.Command{
		Name:      "refund",
		ShortHelp: "Refund an external transaction",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			client, pkg, requestCtx, cancel, err := buildClient(ctx, deps, packageName)
			if err != nil {
				return err
			}
			defer cancel()

			externalTransactionID = strings.TrimSpace(externalTransactionID)
			if externalTransactionID == "" {
				return fmt.Errorf("--external-transaction-id is required")
			}
			if !confirm {
				return fmt.Errorf("--confirm is required to refund external transaction %q", externalTransactionID)
			}

			request, err := readRefundExternalTransactionPayload(inputPath, deps.Stdin)
			if err != nil {
				return err
			}

			transaction, err := client.RefundExternalTransaction(requestCtx, pkg, externalTransactionID, request)
			if err != nil {
				return fmt.Errorf("failed to refund external transaction: %w", err)
			}

			return shared.WriteJSON(deps.Stdout, map[string]any{
				"packageName":         pkg,
				"externalTransaction": transaction,
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

func readExternalTransactionPayload(inputPath string, stdin io.Reader) (*androidpublisher.ExternalTransaction, error) {
	var transaction androidpublisher.ExternalTransaction
	if err := readJSONPayload(inputPath, stdin, &transaction); err != nil {
		return nil, fmt.Errorf("invalid external transaction JSON payload: %w", err)
	}
	return &transaction, nil
}

func readRefundExternalTransactionPayload(inputPath string, stdin io.Reader) (*androidpublisher.RefundExternalTransactionRequest, error) {
	var request androidpublisher.RefundExternalTransactionRequest
	if err := readJSONPayload(inputPath, stdin, &request); err != nil {
		return nil, fmt.Errorf("invalid refund JSON payload: %w", err)
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
