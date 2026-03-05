package externaltransactions

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"google.golang.org/api/androidpublisher/v3"
)

type fakeClient struct {
	transaction           *androidpublisher.ExternalTransaction
	getErr                error
	createErr             error
	refundErr             error
	capturedExternalTxnID string
	capturedCreatePayload *androidpublisher.ExternalTransaction
	capturedRefundPayload *androidpublisher.RefundExternalTransactionRequest
}

func (f *fakeClient) GetExternalTransaction(_ context.Context, _ string, externalTransactionID string) (*androidpublisher.ExternalTransaction, error) {
	f.capturedExternalTxnID = externalTransactionID
	return f.transaction, f.getErr
}

func (f *fakeClient) CreateExternalTransaction(_ context.Context, _ string, externalTransactionID string, transaction *androidpublisher.ExternalTransaction) (*androidpublisher.ExternalTransaction, error) {
	f.capturedExternalTxnID = externalTransactionID
	f.capturedCreatePayload = transaction
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.transaction != nil {
		return f.transaction, nil
	}
	return transaction, nil
}

func (f *fakeClient) RefundExternalTransaction(_ context.Context, _ string, externalTransactionID string, request *androidpublisher.RefundExternalTransactionRequest) (*androidpublisher.ExternalTransaction, error) {
	f.capturedExternalTxnID = externalTransactionID
	f.capturedRefundPayload = request
	return f.transaction, f.refundErr
}

func runExternalTransactions(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func bindGlobalPackageName(t *testing.T, packageName string) {
	t.Helper()
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	cfg := &shared.GlobalFlags{}
	shared.BindGlobalFlags(fs, cfg)
	cfg.PackageName = packageName
}

func writePayloadFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write payload: %v", err)
	}
	return path
}

func TestExternalTransactionsGet_ReturnsTransaction(t *testing.T) {
	fc := &fakeClient{
		transaction: &androidpublisher.ExternalTransaction{
			ExternalTransactionId: "ext-1",
			PackageName:           "com.example.app",
			TransactionState:      "TRANSACTION_REPORTED",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	out, err := runExternalTransactions(t, deps, "get", "--package-name", "com.example.app", "--external-transaction-id", "ext-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"externalTransactionId":"ext-1"`) || !strings.Contains(out, `"transactionState":"TRANSACTION_REPORTED"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestExternalTransactionsCreate_RequiresID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	payload := writePayloadFile(t, "external-transaction.json", `{"transactionTime":"2026-03-05T10:00:00Z"}`)
	_, err := runExternalTransactions(t, deps, "create", "--package-name", "com.example.app", "--input", payload)
	if err == nil || !strings.Contains(err.Error(), "--external-transaction-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExternalTransactionsCreate_ReadsPayload(t *testing.T) {
	fc := &fakeClient{
		transaction: &androidpublisher.ExternalTransaction{
			ExternalTransactionId: "ext-1",
			TransactionState:      "TRANSACTION_REPORTED",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	payload := writePayloadFile(t, "external-transaction.json", `{
		"transactionTime":"2026-03-05T10:00:00Z",
		"originalPreTaxAmount":{"currency":"USD","priceMicros":"9990000"},
		"originalTaxAmount":{"currency":"USD","priceMicros":"1000000"},
		"userTaxAddress":{"regionCode":"US"},
		"oneTimeTransaction":{"externalTransactionToken":"token-1"}
	}`)
	out, err := runExternalTransactions(t, deps, "create", "--package-name", "com.example.app", "--external-transaction-id", "ext-1", "--input", payload)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"externalTransactionId":"ext-1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedExternalTxnID != "ext-1" || fc.capturedCreatePayload == nil || fc.capturedCreatePayload.OneTimeTransaction == nil {
		t.Fatalf("unexpected captured create payload: %#v", fc.capturedCreatePayload)
	}
}

func TestExternalTransactionsRefund_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	payload := writePayloadFile(t, "refund.json", `{"refundTime":"2026-03-05T10:00:00Z","fullRefund":{}}`)
	_, err := runExternalTransactions(t, deps, "refund", "--package-name", "com.example.app", "--external-transaction-id", "ext-1", "--input", payload)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExternalTransactionsRefund_ReadsPayload(t *testing.T) {
	fc := &fakeClient{
		transaction: &androidpublisher.ExternalTransaction{
			ExternalTransactionId: "ext-1",
			TransactionState:      "TRANSACTION_CANCELED",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	payload := writePayloadFile(t, "refund.json", `{"refundTime":"2026-03-05T10:00:00Z","fullRefund":{}}`)
	out, err := runExternalTransactions(t, deps, "refund", "--package-name", "com.example.app", "--external-transaction-id", "ext-1", "--input", payload, "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"transactionState":"TRANSACTION_CANCELED"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fc.capturedExternalTxnID != "ext-1" || fc.capturedRefundPayload == nil || fc.capturedRefundPayload.FullRefund == nil {
		t.Fatalf("unexpected refund payload: %#v", fc.capturedRefundPayload)
	}
}

func TestExternalTransactionsGet_UsesGlobalPackageName(t *testing.T) {
	bindGlobalPackageName(t, "com.example.global")
	fc := &fakeClient{
		transaction: &androidpublisher.ExternalTransaction{ExternalTransactionId: "ext-1"},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return fc, nil },
	}

	if _, err := runExternalTransactions(t, deps, "get", "--external-transaction-id", "ext-1"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
}
