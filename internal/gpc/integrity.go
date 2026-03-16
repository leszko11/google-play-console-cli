package gpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/option"
	playintegrity "google.golang.org/api/playintegrity/v1"
)

type IntegrityTokenPayload = playintegrity.TokenPayloadExternal

type IntegrityDecodeInfo struct {
	TokenPayloadExternal *IntegrityTokenPayload `json:"tokenPayloadExternal,omitempty"`
}

type IntegrityClient struct {
	integrity *playintegrity.Service
}

func NewIntegrityClient(ctx context.Context, creds CredentialInput, opts ...option.ClientOption) (*IntegrityClient, error) {
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

	svc, err := playintegrity.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, err
	}

	return &IntegrityClient{integrity: svc}, nil
}

func (c *IntegrityClient) DecodeIntegrityToken(ctx context.Context, packageName, integrityToken string) (IntegrityDecodeInfo, error) {
	if c == nil || c.integrity == nil {
		return IntegrityDecodeInfo{}, errors.New("playintegrity service is not configured")
	}

	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return IntegrityDecodeInfo{}, fmt.Errorf("package name is required")
	}

	integrityToken = strings.TrimSpace(integrityToken)
	if integrityToken == "" {
		return IntegrityDecodeInfo{}, fmt.Errorf("integrity token must not be empty")
	}

	resp, err := c.integrity.V1.DecodeIntegrityToken(packageName, &playintegrity.DecodeIntegrityTokenRequest{
		IntegrityToken: integrityToken,
	}).Context(ctx).Do()
	if err != nil {
		return IntegrityDecodeInfo{}, mapGoogleAPIErrorWithService("playintegrity", err, false)
	}

	return IntegrityDecodeInfo{
		TokenPayloadExternal: resp.TokenPayloadExternal,
	}, nil
}
