package gpc

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestNewClient_RejectsMissingCredentials(t *testing.T) {
	_, err := NewClient(context.Background(), CredentialInput{})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestVerifyPackageAccess_MapsForbidden(t *testing.T) {
	err := mapAPIError(http.StatusForbidden, "forbidden")
	if !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("expected ErrAccessDenied, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing Play Console permissions") {
		t.Fatalf("expected permission hint for forbidden error, got %v", err)
	}
}

func TestVerifyPackageAccess_MapsNotFound(t *testing.T) {
	err := mapAPIError(http.StatusNotFound, "Package not found: com.example.app.")
	if !errors.Is(err, ErrPackageNotFound) {
		t.Fatalf("expected ErrPackageNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "hint: this package is not initialized in Google Play yet") {
		t.Fatalf("expected bootstrap hint for package-not-found error, got %v", err)
	}
}

func TestVerifyPackageAccess_MapsOtherNotFoundWithoutBootstrapHint(t *testing.T) {
	err := mapAPIError(http.StatusNotFound, "Edit not found")
	if !errors.Is(err, ErrPackageNotFound) {
		t.Fatalf("expected ErrPackageNotFound, got %v", err)
	}
	if strings.Contains(err.Error(), "this package is not initialized in Google Play yet") {
		t.Fatalf("expected no bootstrap hint for non-package not found error, got %v", err)
	}
}

func TestMapAPIError_MapsUnauthorizedWithPermissionHint(t *testing.T) {
	err := mapAPIError(http.StatusUnauthorized, "invalid credentials")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing Play Console permissions") {
		t.Fatalf("expected permission hint for unauthorized error, got %v", err)
	}
}

func TestMapAPIError_PermissionLikeMessageAddsHint(t *testing.T) {
	err := mapAPIError(http.StatusBadRequest, "The caller does not have permission to access this resource.")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing Play Console permissions") {
		t.Fatalf("expected permission hint for permission-like message, got %v", err)
	}
}

func TestMapAPIError_ForbiddenWithoutPermissionSignalHasNoPermissionHint(t *testing.T) {
	err := mapAPIError(http.StatusForbidden, "Version code 1772439125 has already been used.")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrAccessDenied) {
		t.Fatalf("expected non-permission 403 to avoid ErrAccessDenied wrapping, got %v", err)
	}
	if strings.Contains(err.Error(), "missing Play Console permissions") {
		t.Fatalf("expected no permission hint for non-permission 403, got %v", err)
	}
}

func TestMapAPIError_LegacyIAPMigrationHint(t *testing.T) {
	err := mapAPIError(http.StatusForbidden, "Please migrate to the new publishing API.")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "gpc products") {
		t.Fatalf("expected migration hint, got %v", err)
	}
}

func TestMapAPIError_APKUploadNotAllowedHint(t *testing.T) {
	err := mapAPIError(http.StatusForbidden, "APKs are not allowed for this application.")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "accepts Android App Bundles only") {
		t.Fatalf("expected APK upload hint, got %v", err)
	}
}
