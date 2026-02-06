package gpc

import (
	"context"
	"errors"
	"net/http"
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
}

func TestVerifyPackageAccess_MapsNotFound(t *testing.T) {
	err := mapAPIError(http.StatusNotFound, "not found")
	if !errors.Is(err, ErrPackageNotFound) {
		t.Fatalf("expected ErrPackageNotFound, got %v", err)
	}
}
