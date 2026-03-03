package gpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var (
	ErrInvalidCredentials = errors.New("missing service account credentials")
	ErrAccessDenied       = errors.New("access denied for package")
	ErrPackageNotFound    = errors.New("package not found")
)

type Client struct {
	service *androidpublisher.Service
}

func NewClient(ctx context.Context, creds CredentialInput, opts ...option.ClientOption) (*Client, error) {
	if strings.TrimSpace(creds.ServiceAccountPath) == "" && len(creds.ServiceAccountJSON) == 0 {
		return nil, ErrInvalidCredentials
	}

	clientOpts := make([]option.ClientOption, 0, 2+len(opts))
	if strings.TrimSpace(creds.ServiceAccountPath) != "" {
		clientOpts = append(clientOpts, option.WithCredentialsFile(creds.ServiceAccountPath))
	}
	if len(creds.ServiceAccountJSON) > 0 {
		clientOpts = append(clientOpts, option.WithCredentialsJSON(creds.ServiceAccountJSON))
	}
	clientOpts = append(clientOpts, opts...)

	svc, err := androidpublisher.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{service: svc}, nil
}

func mapGoogleAPIError(err error) error {
	if err == nil {
		return nil
	}

	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		return mapAPIError(gerr.Code, gerr.Message)
	}

	return err
}

func mapAPIError(statusCode int, msg string) error {
	if isLegacyIAPMigration(msg) {
		return fmt.Errorf("androidpublisher api error (%d): %s\nhint: this app is on the new monetization API. Use `gpc products ...` and `gpc subscriptions ...` instead of legacy `gpc iap ...`.", statusCode, msg)
	}
	if isAPKUploadNotAllowed(msg) {
		return fmt.Errorf("androidpublisher api error (%d): %s\nhint: this app accepts Android App Bundles only. Use `gpc bundles upload ...` or `gpc deploy --aab ...`.", statusCode, msg)
	}

	switch statusCode {
	case 401:
		return fmt.Errorf("androidpublisher api error (%d): %s\n%s", statusCode, msg, permissionSetupHint())
	case 403:
		if isPermissionErrorMessage(msg) {
			return fmt.Errorf("%w: %s\n%s", ErrAccessDenied, msg, permissionSetupHint())
		}
		return fmt.Errorf("androidpublisher api error (%d): %s", statusCode, msg)
	case 404:
		if isPackageBootstrapNotReady(msg) {
			return fmt.Errorf("%w: %s\nhint: this package is not initialized in Google Play yet. Upload the first APK or AAB once in Play Console, then retry. Also verify the service account has access to this app.", ErrPackageNotFound, msg)
		}
		return fmt.Errorf("%w: %s", ErrPackageNotFound, msg)
	default:
		if isPermissionErrorMessage(msg) {
			return fmt.Errorf("androidpublisher api error (%d): %s\n%s", statusCode, msg, permissionSetupHint())
		}
		return fmt.Errorf("androidpublisher api error (%d): %s", statusCode, msg)
	}
}

func isPackageBootstrapNotReady(msg string) bool {
	return strings.Contains(strings.ToLower(msg), "package not found")
}

func isPermissionErrorMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "permission") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "access denied") ||
		strings.Contains(lower, "insufficient") ||
		strings.Contains(lower, "not authorized")
}

func isLegacyIAPMigration(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "migrate to the new publishing api")
}

func isAPKUploadNotAllowed(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "apks are not allowed for this application")
}

func permissionSetupHint() string {
	return "hint: this usually means missing Play Console permissions. In Play Console -> Users and permissions, grant the service account email access to the app (or account-wide for users/grants). Also ensure the Google Play Android Developer API is enabled in the same Google Cloud project as this service account."
}
