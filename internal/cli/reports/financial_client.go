package reports

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"google.golang.org/api/option"
	gcsstorage "google.golang.org/api/storage/v1"
)

type FinancialObjectInfo struct {
	Bucket      string    `json:"bucket"`
	Name        string    `json:"name"`
	Size        int64     `json:"size,omitempty"`
	ContentType string    `json:"contentType,omitempty"`
	Updated     time.Time `json:"updated,omitempty"`
}

type FinancialObjectList struct {
	Objects       []FinancialObjectInfo `json:"objects,omitempty"`
	NextPageToken string                `json:"nextPageToken,omitempty"`
}

type FinancialObjectDownload struct {
	Bucket          string `json:"bucket"`
	Name            string `json:"name"`
	ContentType     string `json:"contentType,omitempty"`
	ContentEncoding string `json:"contentEncoding,omitempty"`
	Data            []byte `json:"-"`
}

type storageFinancialClient struct {
	service *gcsstorage.Service
}

func NewFinancialClient(ctx context.Context, creds gpc.CredentialInput) (FinancialClient, error) {
	if strings.TrimSpace(creds.ServiceAccountPath) == "" && len(creds.ServiceAccountJSON) == 0 {
		return nil, gpc.ErrInvalidCredentials
	}

	opts := []option.ClientOption{option.WithScopes(gcsstorage.DevstorageReadOnlyScope)}
	if strings.TrimSpace(creds.ServiceAccountPath) != "" {
		opts = append(opts, option.WithCredentialsFile(creds.ServiceAccountPath))
	}
	if len(creds.ServiceAccountJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(creds.ServiceAccountJSON))
	}

	service, err := gcsstorage.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &storageFinancialClient{service: service}, nil
}

func (c *storageFinancialClient) ListObjects(ctx context.Context, bucket, prefix string, pageSize int64, pageToken string) (FinancialObjectList, error) {
	if c == nil || c.service == nil {
		return FinancialObjectList{}, fmt.Errorf("cloud storage service is not configured")
	}

	call := c.service.Objects.List(strings.TrimSpace(bucket)).Context(ctx)
	if trimmedPrefix := strings.TrimSpace(prefix); trimmedPrefix != "" {
		call.Prefix(trimmedPrefix)
	}
	if trimmedToken := strings.TrimSpace(pageToken); trimmedToken != "" {
		call.PageToken(trimmedToken)
	}
	if pageSize > 0 {
		call.MaxResults(pageSize)
	}

	resp, err := call.Do()
	if err != nil {
		return FinancialObjectList{}, err
	}

	result := FinancialObjectList{
		Objects:       make([]FinancialObjectInfo, 0, len(resp.Items)),
		NextPageToken: resp.NextPageToken,
	}
	for _, item := range resp.Items {
		if item == nil {
			continue
		}
		result.Objects = append(result.Objects, FinancialObjectInfo{
			Bucket:      item.Bucket,
			Name:        item.Name,
			Size:        int64(item.Size),
			ContentType: item.ContentType,
			Updated:     parseFinancialObjectUpdated(item.Updated),
		})
	}
	return result, nil
}

func (c *storageFinancialClient) DownloadObject(ctx context.Context, bucket, objectName string) (FinancialObjectDownload, error) {
	if c == nil || c.service == nil {
		return FinancialObjectDownload{}, fmt.Errorf("cloud storage service is not configured")
	}

	call := c.service.Objects.Get(strings.TrimSpace(bucket), strings.TrimSpace(objectName)).Context(ctx)
	object, err := call.Do()
	if err != nil {
		return FinancialObjectDownload{}, err
	}

	resp, err := call.Download()
	if err != nil {
		return FinancialObjectDownload{}, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return FinancialObjectDownload{}, err
	}

	return FinancialObjectDownload{
		Bucket:          object.Bucket,
		Name:            object.Name,
		ContentType:     object.ContentType,
		ContentEncoding: object.ContentEncoding,
		Data:            payload,
	}, nil
}

func parseFinancialObjectUpdated(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
