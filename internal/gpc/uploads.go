package gpc

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

func (c *Client) ListBundles(ctx context.Context, packageName, editID string) ([]BundleInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, fmt.Errorf("edit id is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Edits.Bundles.List(packageName, editID).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}

	bundles := make([]BundleInfo, 0, len(resp.Bundles))
	for _, bundle := range resp.Bundles {
		bundles = append(bundles, bundleInfoFromBundle(bundle))
	}
	return bundles, nil
}

func (c *Client) UploadBundle(ctx context.Context, packageName, editID, bundlePath string) (BundleInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return BundleInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return BundleInfo{}, fmt.Errorf("edit id is required")
	}
	bundlePath = strings.TrimSpace(bundlePath)
	if bundlePath == "" {
		return BundleInfo{}, fmt.Errorf("bundle path is required")
	}
	if c == nil || c.service == nil {
		return BundleInfo{}, ErrInvalidCredentials
	}

	f, err := os.Open(bundlePath)
	if err != nil {
		return BundleInfo{}, fmt.Errorf("failed to open bundle file: %w", err)
	}
	defer f.Close()

	bundle, err := c.service.Edits.Bundles.Upload(packageName, editID).Media(f).Context(ctx).Do()
	if err != nil {
		return BundleInfo{}, mapGoogleAPIError(err)
	}
	return bundleInfoFromBundle(bundle), nil
}

func (c *Client) ListAPKs(ctx context.Context, packageName, editID string) ([]APKInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return nil, fmt.Errorf("edit id is required")
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Edits.Apks.List(packageName, editID).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}

	apks := make([]APKInfo, 0, len(resp.Apks))
	for _, apk := range resp.Apks {
		apks = append(apks, apkInfoFromAPK(apk))
	}
	return apks, nil
}

func (c *Client) UploadAPK(ctx context.Context, packageName, editID, apkPath string) (APKInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return APKInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return APKInfo{}, fmt.Errorf("edit id is required")
	}
	apkPath = strings.TrimSpace(apkPath)
	if apkPath == "" {
		return APKInfo{}, fmt.Errorf("apk path is required")
	}
	if c == nil || c.service == nil {
		return APKInfo{}, ErrInvalidCredentials
	}

	f, err := os.Open(apkPath)
	if err != nil {
		return APKInfo{}, fmt.Errorf("failed to open apk file: %w", err)
	}
	defer f.Close()

	apk, err := c.service.Edits.Apks.Upload(packageName, editID).Media(f).Context(ctx).Do()
	if err != nil {
		return APKInfo{}, mapGoogleAPIError(err)
	}
	return apkInfoFromAPK(apk), nil
}

func (c *Client) UploadDeobfuscationFile(ctx context.Context, packageName, editID string, versionCode int64, fileType, filePath string) (DeobfuscationFileInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return DeobfuscationFileInfo{}, fmt.Errorf("package name is required")
	}
	editID = strings.TrimSpace(editID)
	if editID == "" {
		return DeobfuscationFileInfo{}, fmt.Errorf("edit id is required")
	}
	if versionCode <= 0 {
		return DeobfuscationFileInfo{}, fmt.Errorf("version code must be greater than zero")
	}
	fileType = strings.TrimSpace(fileType)
	if fileType == "" {
		return DeobfuscationFileInfo{}, fmt.Errorf("deobfuscation file type is required")
	}
	if fileType != "proguard" && fileType != "nativeCode" {
		return DeobfuscationFileInfo{}, fmt.Errorf("unsupported deobfuscation file type %q", fileType)
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return DeobfuscationFileInfo{}, fmt.Errorf("deobfuscation file path is required")
	}
	if c == nil || c.service == nil {
		return DeobfuscationFileInfo{}, ErrInvalidCredentials
	}

	f, err := os.Open(filePath)
	if err != nil {
		return DeobfuscationFileInfo{}, fmt.Errorf("failed to open deobfuscation file: %w", err)
	}
	defer f.Close()

	resp, err := c.service.Edits.Deobfuscationfiles.Upload(packageName, editID, versionCode, fileType).Media(f).Context(ctx).Do()
	if err != nil {
		return DeobfuscationFileInfo{}, mapGoogleAPIError(err)
	}
	return deobfuscationInfoFromResponse(resp), nil
}

func bundleInfoFromBundle(bundle *androidpublisher.Bundle) BundleInfo {
	if bundle == nil {
		return BundleInfo{}
	}
	return BundleInfo{
		VersionCode: bundle.VersionCode,
		SHA1:        bundle.Sha1,
		SHA256:      bundle.Sha256,
	}
}

func apkInfoFromAPK(apk *androidpublisher.Apk) APKInfo {
	if apk == nil {
		return APKInfo{}
	}
	sha1 := ""
	sha256 := ""
	if apk.Binary != nil {
		sha1 = apk.Binary.Sha1
		sha256 = apk.Binary.Sha256
	}
	return APKInfo{
		VersionCode: apk.VersionCode,
		SHA1:        sha1,
		SHA256:      sha256,
	}
}

func deobfuscationInfoFromResponse(resp *androidpublisher.DeobfuscationFilesUploadResponse) DeobfuscationFileInfo {
	if resp == nil || resp.DeobfuscationFile == nil {
		return DeobfuscationFileInfo{}
	}
	return DeobfuscationFileInfo{
		SymbolType: resp.DeobfuscationFile.SymbolType,
	}
}
