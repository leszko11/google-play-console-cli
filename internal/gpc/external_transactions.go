package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) GetExternalTransaction(ctx context.Context, packageName, externalTransactionID string) (*androidpublisher.ExternalTransaction, error) {
	packageName, externalTransactionID, err := validateExternalTransactionTarget(packageName, externalTransactionID)
	if err != nil {
		return nil, err
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	transaction, err := c.service.Externaltransactions.Getexternaltransaction(
		externalTransactionName(packageName, externalTransactionID),
	).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return transaction, nil
}

func (c *Client) CreateExternalTransaction(ctx context.Context, packageName, externalTransactionID string, transaction *androidpublisher.ExternalTransaction) (*androidpublisher.ExternalTransaction, error) {
	packageName, externalTransactionID, err := validateExternalTransactionTarget(packageName, externalTransactionID)
	if err != nil {
		return nil, err
	}
	if transaction == nil {
		return nil, fmt.Errorf("external transaction payload is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	created, err := c.service.Externaltransactions.Createexternaltransaction(
		externalTransactionParent(packageName),
		transaction,
	).ExternalTransactionId(externalTransactionID).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return created, nil
}

func (c *Client) RefundExternalTransaction(ctx context.Context, packageName, externalTransactionID string, request *androidpublisher.RefundExternalTransactionRequest) (*androidpublisher.ExternalTransaction, error) {
	packageName, externalTransactionID, err := validateExternalTransactionTarget(packageName, externalTransactionID)
	if err != nil {
		return nil, err
	}
	if err := validateRefundExternalTransactionRequest(request); err != nil {
		return nil, err
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	transaction, err := c.service.Externaltransactions.Refundexternaltransaction(
		externalTransactionName(packageName, externalTransactionID),
		request,
	).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}
	return transaction, nil
}

func validateExternalTransactionTarget(packageName, externalTransactionID string) (string, string, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return "", "", fmt.Errorf("package name is required")
	}
	externalTransactionID = strings.TrimSpace(externalTransactionID)
	if externalTransactionID == "" {
		return "", "", fmt.Errorf("external transaction id is required")
	}
	return packageName, externalTransactionID, nil
}

func validateRefundExternalTransactionRequest(request *androidpublisher.RefundExternalTransactionRequest) error {
	if request == nil {
		return fmt.Errorf("refund payload is required")
	}
	if strings.TrimSpace(request.RefundTime) == "" {
		return fmt.Errorf("refund time is required")
	}

	fullRefund := request.FullRefund != nil
	partialRefund := request.PartialRefund != nil
	if fullRefund == partialRefund {
		return fmt.Errorf("exactly one of fullRefund or partialRefund is required")
	}

	if partialRefund {
		if strings.TrimSpace(request.PartialRefund.RefundId) == "" {
			return fmt.Errorf("partial refund id is required")
		}
		if request.PartialRefund.RefundPreTaxAmount == nil {
			return fmt.Errorf("partial refund pre-tax amount is required")
		}
	}

	return nil
}

func externalTransactionParent(packageName string) string {
	return fmt.Sprintf("applications/%s", packageName)
}

func externalTransactionName(packageName, externalTransactionID string) string {
	return fmt.Sprintf("%s/externalTransactions/%s", externalTransactionParent(packageName), externalTransactionID)
}
