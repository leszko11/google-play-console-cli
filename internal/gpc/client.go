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
	switch statusCode {
	case 403:
		return fmt.Errorf("%w: %s", ErrAccessDenied, msg)
	case 404:
		if isPackageBootstrapNotReady(msg) {
			return fmt.Errorf("%w: %s\nhint: this package is not initialized in Google Play yet. Upload the first APK or AAB once in Play Console, then retry. Also verify the service account has access to this app.", ErrPackageNotFound, msg)
		}
		return fmt.Errorf("%w: %s", ErrPackageNotFound, msg)
	default:
		return fmt.Errorf("androidpublisher api error (%d): %s", statusCode, msg)
	}
}

func isPackageBootstrapNotReady(msg string) bool {
	return strings.Contains(strings.ToLower(msg), "package not found")
}
