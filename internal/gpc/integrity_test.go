package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewIntegrityClient_RejectsMissingCredentials(t *testing.T) {
	_, err := NewIntegrityClient(context.Background(), CredentialInput{})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestIntegrityClientDecodeIntegrityToken_RequiresService(t *testing.T) {
	client := &IntegrityClient{}
	_, err := client.DecodeIntegrityToken(context.Background(), "com.example.app", "token")
	if err == nil || !strings.Contains(err.Error(), "service is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegrityClientDecodeIntegrityToken_RejectsEmptyToken(t *testing.T) {
	client := &IntegrityClient{integrity: nil}
	_, err := client.DecodeIntegrityToken(context.Background(), "com.example.app", "   ")
	if err == nil || !strings.Contains(err.Error(), "service is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}
