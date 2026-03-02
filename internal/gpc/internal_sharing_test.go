package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestInternalSharingMethods_RejectMissingClient(t *testing.T) {
	var c *Client

	if _, err := c.UploadInternalSharingAPK(context.Background(), "com.example.app", "/tmp/app.apk"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UploadInternalSharingAPK, got %v", err)
	}
	if _, err := c.UploadInternalSharingBundle(context.Background(), "com.example.app", "/tmp/app.aab"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials from UploadInternalSharingBundle, got %v", err)
	}
}

func TestInternalSharingMethods_ValidateArgs(t *testing.T) {
	c := &Client{}

	if _, err := c.UploadInternalSharingAPK(context.Background(), "", "/tmp/app.apk"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected UploadInternalSharingAPK package error: %v", err)
	}
	if _, err := c.UploadInternalSharingAPK(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "apk path is required") {
		t.Fatalf("unexpected UploadInternalSharingAPK path error: %v", err)
	}
	if _, err := c.UploadInternalSharingBundle(context.Background(), "", "/tmp/app.aab"); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("unexpected UploadInternalSharingBundle package error: %v", err)
	}
	if _, err := c.UploadInternalSharingBundle(context.Background(), "com.example.app", ""); err == nil || !strings.Contains(err.Error(), "bundle path is required") {
		t.Fatalf("unexpected UploadInternalSharingBundle path error: %v", err)
	}
}

func TestInternalSharingArtifactInfoFromArtifact(t *testing.T) {
	got := internalSharingArtifactInfoFromArtifact(&androidpublisher.InternalAppSharingArtifact{
		DownloadUrl:            "https://play.google.com/internal-sharing/download",
		CertificateFingerprint: "AB:CD:EF",
		Sha256:                 "abc123",
	})
	if got.DownloadURL != "https://play.google.com/internal-sharing/download" || got.CertificateFingerprint != "AB:CD:EF" || got.SHA256 != "abc123" {
		t.Fatalf("unexpected internal sharing artifact map: %+v", got)
	}
}
