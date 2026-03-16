package integrity

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	playintegrity "google.golang.org/api/playintegrity/v1"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	result        gpc.IntegrityDecodeInfo
	err           error
	capturedPkg   string
	capturedToken string
}

func (f *fakeClient) DecodeIntegrityToken(_ context.Context, packageName, integrityToken string) (gpc.IntegrityDecodeInfo, error) {
	f.capturedPkg = packageName
	f.capturedToken = integrityToken
	return f.result, f.err
}

func runIntegrity(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		}
	}
	if deps.Stdin == nil {
		deps.Stdin = bytes.NewBuffer(nil)
	}
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func TestIntegrityDecode_WithToken_ReturnsDecodedPayload(t *testing.T) {
	fc := &fakeClient{
		result: gpc.IntegrityDecodeInfo{
			TokenPayloadExternal: &playintegrity.TokenPayloadExternal{
				RequestDetails: &playintegrity.RequestDetails{
					RequestHash: "hash-1",
				},
			},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
	}

	out, err := runIntegrity(t, deps, "decode", "--package-name", "com.example.app", "--token", "token-123")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if fc.capturedPkg != "com.example.app" || fc.capturedToken != "token-123" {
		t.Fatalf("unexpected decode args: pkg=%q token=%q", fc.capturedPkg, fc.capturedToken)
	}
	for _, want := range []string{`"packageName":"com.example.app"`, `"requestHash":"hash-1"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %s", want, out)
		}
	}
}

func TestIntegrityDecode_ReadsTokenFromStdin(t *testing.T) {
	fc := &fakeClient{
		result: gpc.IntegrityDecodeInfo{
			TokenPayloadExternal: &playintegrity.TokenPayloadExternal{},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
		Stdin: strings.NewReader("stdin-token\n"),
	}

	_, err := runIntegrity(t, deps, "decode", "--package-name", "com.example.app", "--input", "-")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if fc.capturedToken != "stdin-token" {
		t.Fatalf("expected trimmed stdin token, got %q", fc.capturedToken)
	}
}

func TestIntegrityDecode_RejectsBothTokenAndInput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runIntegrity(t, deps, "decode", "--package-name", "com.example.app", "--token", "token-123", "--input", "token.txt")
	if err == nil || !strings.Contains(err.Error(), "exactly one of --token or --input must be set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegrityDecode_RejectsMissingTokenInput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{}, nil
		},
	}

	_, err := runIntegrity(t, deps, "decode", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "exactly one of --token or --input must be set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegrityDecode_RejectsEmptyInputPayload(t *testing.T) {
	fc := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fc, nil
		},
		Stdin: strings.NewReader("   \n"),
	}

	_, err := runIntegrity(t, deps, "decode", "--package-name", "com.example.app", "--input", "-")
	if err == nil || !strings.Contains(err.Error(), "integrity token must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.capturedToken != "" {
		t.Fatalf("expected no API call for empty input, got token %q", fc.capturedToken)
	}
}

func TestIntegrityDecode_ReturnsDecodeError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return &fakeClient{err: errors.New("permission denied")}, nil
		},
	}

	_, err := runIntegrity(t, deps, "decode", "--package-name", "com.example.app", "--token", "token-123")
	if err == nil || !strings.Contains(err.Error(), "failed to decode integrity token") {
		t.Fatalf("unexpected error: %v", err)
	}
}
