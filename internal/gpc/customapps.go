package gpc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/api/option"
	playcustomapp "google.golang.org/api/playcustomapp/v1"
)

type CustomAppOrganizationInfo struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type CustomAppInfo struct {
	Title         string                      `json:"title"`
	LanguageCode  string                      `json:"languageCode,omitempty"`
	PackageName   string                      `json:"packageName,omitempty"`
	Organizations []CustomAppOrganizationInfo `json:"organizations,omitempty"`
}

type CustomAppsClient struct {
	customApps *playcustomapp.Service
}

func NewCustomAppsClient(ctx context.Context, creds CredentialInput, opts ...option.ClientOption) (*CustomAppsClient, error) {
	if strings.TrimSpace(creds.ServiceAccountPath) == "" && len(creds.ServiceAccountJSON) == 0 {
		return nil, ErrInvalidCredentials
	}

	clientOpts := make([]option.ClientOption, 0, 3+len(opts))
	if strings.TrimSpace(creds.ServiceAccountPath) != "" {
		clientOpts = append(clientOpts, option.WithCredentialsFile(creds.ServiceAccountPath))
	}
	if len(creds.ServiceAccountJSON) > 0 {
		clientOpts = append(clientOpts, option.WithCredentialsJSON(creds.ServiceAccountJSON))
	}
	clientOpts = append(clientOpts, option.WithScopes(playcustomapp.AndroidpublisherScope))
	clientOpts = append(clientOpts, opts...)

	svc, err := playcustomapp.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, err
	}
	return &CustomAppsClient{customApps: svc}, nil
}

func (c *CustomAppsClient) CreateCustomApp(ctx context.Context, developerID string, app *playcustomapp.CustomApp) (CustomAppInfo, error) {
	accountID, err := normalizeDeveloperAccountID(developerID)
	if err != nil {
		return CustomAppInfo{}, err
	}
	if app == nil {
		return CustomAppInfo{}, fmt.Errorf("custom app payload is required")
	}
	if err := validateCustomApp(app); err != nil {
		return CustomAppInfo{}, err
	}
	if c == nil || c.customApps == nil {
		return CustomAppInfo{}, ErrInvalidCredentials
	}

	created, err := c.customApps.Accounts.CustomApps.Create(accountID, app).Context(ctx).Do()
	if err != nil {
		return CustomAppInfo{}, mapGoogleAPIErrorWithService("playcustomapp", err, false)
	}
	return customAppInfoFromResource(created), nil
}

func normalizeDeveloperAccountID(developerID string) (int64, error) {
	developerID = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(developerID), "developers/"))
	if developerID == "" {
		return 0, fmt.Errorf("developer id is required")
	}
	accountID, err := strconv.ParseInt(developerID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("developer id must be numeric")
	}
	return accountID, nil
}

func validateCustomApp(app *playcustomapp.CustomApp) error {
	if strings.TrimSpace(app.Title) == "" {
		return fmt.Errorf("custom app title is required")
	}
	if strings.TrimSpace(app.LanguageCode) == "" {
		return fmt.Errorf("custom app languageCode is required")
	}
	for idx, org := range app.Organizations {
		if org == nil || strings.TrimSpace(org.OrganizationId) == "" {
			return fmt.Errorf("organizationId is required for organizations[%d]", idx)
		}
	}
	return nil
}

func customAppInfoFromResource(app *playcustomapp.CustomApp) CustomAppInfo {
	if app == nil {
		return CustomAppInfo{}
	}

	info := CustomAppInfo{
		Title:        strings.TrimSpace(app.Title),
		LanguageCode: strings.TrimSpace(app.LanguageCode),
		PackageName:  strings.TrimSpace(app.PackageName),
	}
	if len(app.Organizations) > 0 {
		info.Organizations = make([]CustomAppOrganizationInfo, 0, len(app.Organizations))
		for _, org := range app.Organizations {
			if org == nil {
				continue
			}
			info.Organizations = append(info.Organizations, CustomAppOrganizationInfo{
				ID:   strings.TrimSpace(org.OrganizationId),
				Name: strings.TrimSpace(org.OrganizationName),
			})
		}
	}
	return info
}
