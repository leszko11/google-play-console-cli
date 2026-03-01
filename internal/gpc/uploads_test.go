package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestUploadMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.ListBundles(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListBundles, got %v", err)
	}
	if _, err := c.UploadBundle(context.Background(), "com.example.app", "edit-1", "/tmp/app.aab"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UploadBundle, got %v", err)
	}
	if _, err := c.ListAPKs(context.Background(), "com.example.app", "edit-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from ListAPKs, got %v", err)
	}
	if _, err := c.UploadAPK(context.Background(), "com.example.app", "edit-1", "/tmp/app.apk"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UploadAPK, got %v", err)
	}
}

func TestUploadMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.ListBundles(context.Background(), "", "edit-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListBundles error: %v", err)
	}
	if _, err := c.UploadBundle(context.Background(), "com.example.app", "", "/tmp/app.aab"); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected UploadBundle error: %v", err)
	}
	if _, err := c.UploadBundle(context.Background(), "com.example.app", "edit-1", ""); err == nil || !strings.Contains(err.Error(), "bundle path is required") {
		t.Fatalf("unexpected UploadBundle error: %v", err)
	}
	if _, err := c.ListAPKs(context.Background(), "", "edit-1"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected ListAPKs error: %v", err)
	}
	if _, err := c.UploadAPK(context.Background(), "com.example.app", "", "/tmp/app.apk"); err == nil || !strings.Contains(err.Error(), "edit id is required") {
		t.Fatalf("unexpected UploadAPK error: %v", err)
	}
	if _, err := c.UploadAPK(context.Background(), "com.example.app", "edit-1", ""); err == nil || !strings.Contains(err.Error(), "apk path is required") {
		t.Fatalf("unexpected UploadAPK error: %v", err)
	}
}

func TestUploadMethods_FileErrors(t *testing.T) {
	c := &Client{service: &androidpublisher.Service{}}

	if _, err := c.UploadBundle(context.Background(), "com.example.app", "edit-1", "/path/does/not/exist.aab"); err == nil || !strings.Contains(err.Error(), "failed to open bundle file") {
		t.Fatalf("unexpected UploadBundle file error: %v", err)
	}
	if _, err := c.UploadAPK(context.Background(), "com.example.app", "edit-1", "/path/does/not/exist.apk"); err == nil || !strings.Contains(err.Error(), "failed to open apk file") {
		t.Fatalf("unexpected UploadAPK file error: %v", err)
	}
}

func TestBundleInfoFromBundle(t *testing.T) {
	got := bundleInfoFromBundle(&androidpublisher.Bundle{
		VersionCode: 42,
		Sha1:        "sha1",
		Sha256:      "sha256",
	})
	if got.VersionCode != 42 || got.SHA1 != "sha1" || got.SHA256 != "sha256" {
		t.Fatalf("unexpected bundle mapping: %+v", got)
	}
}

func TestAPKInfoFromAPK(t *testing.T) {
	got := apkInfoFromAPK(&androidpublisher.Apk{
		VersionCode: 99,
		Binary: &androidpublisher.ApkBinary{
			Sha1:   "sha1",
			Sha256: "sha256",
		},
	})
	if got.VersionCode != 99 || got.SHA1 != "sha1" || got.SHA256 != "sha256" {
		t.Fatalf("unexpected apk mapping: %+v", got)
	}
}

func TestAPKInfoFromAPK_NilBinary(t *testing.T) {
	got := apkInfoFromAPK(&androidpublisher.Apk{
		VersionCode: 11,
	})
	if got.VersionCode != 11 || got.SHA1 != "" || got.SHA256 != "" {
		t.Fatalf("unexpected apk map for nil binary: %+v", got)
	}
}
