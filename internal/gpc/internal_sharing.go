package gpc

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"
)

func (c *Client) UploadInternalSharingAPK(ctx context.Context, packageName, apkPath string) (InternalSharingArtifactInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return InternalSharingArtifactInfo{}, fmt.Errorf("package name is required")
	}
	apkPath = strings.TrimSpace(apkPath)
	if apkPath == "" {
		return InternalSharingArtifactInfo{}, fmt.Errorf("apk path is required")
	}
	if c == nil || c.service == nil {
		return InternalSharingArtifactInfo{}, ErrInvalidCredentials
	}

	file, err := os.Open(apkPath)
	if err != nil {
		return InternalSharingArtifactInfo{}, fmt.Errorf("failed to open apk file: %w", err)
	}
	defer file.Close()

	artifact, err := c.service.Internalappsharingartifacts.Uploadapk(packageName).
		Media(file, googleapi.ContentType("application/octet-stream")).
		Context(ctx).
		Do()
	if err != nil {
		return InternalSharingArtifactInfo{}, mapGoogleAPIError(err)
	}
	return internalSharingArtifactInfoFromArtifact(artifact), nil
}

func (c *Client) UploadInternalSharingBundle(ctx context.Context, packageName, bundlePath string) (InternalSharingArtifactInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return InternalSharingArtifactInfo{}, fmt.Errorf("package name is required")
	}
	bundlePath = strings.TrimSpace(bundlePath)
	if bundlePath == "" {
		return InternalSharingArtifactInfo{}, fmt.Errorf("bundle path is required")
	}
	if c == nil || c.service == nil {
		return InternalSharingArtifactInfo{}, ErrInvalidCredentials
	}

	file, err := os.Open(bundlePath)
	if err != nil {
		return InternalSharingArtifactInfo{}, fmt.Errorf("failed to open bundle file: %w", err)
	}
	defer file.Close()

	artifact, err := c.service.Internalappsharingartifacts.Uploadbundle(packageName).
		Media(file, googleapi.ContentType("application/octet-stream")).
		Context(ctx).
		Do()
	if err != nil {
		return InternalSharingArtifactInfo{}, mapGoogleAPIError(err)
	}
	return internalSharingArtifactInfoFromArtifact(artifact), nil
}

func internalSharingArtifactInfoFromArtifact(artifact *androidpublisher.InternalAppSharingArtifact) InternalSharingArtifactInfo {
	if artifact == nil {
		return InternalSharingArtifactInfo{}
	}
	return InternalSharingArtifactInfo{
		DownloadURL:            artifact.DownloadUrl,
		CertificateFingerprint: artifact.CertificateFingerprint,
		SHA256:                 artifact.Sha256,
	}
}
