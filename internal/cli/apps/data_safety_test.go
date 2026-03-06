package apps

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func TestAppsDataSafety_RequiresInput(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/sa.json"},
				},
			}, nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runApps(t, deps, "data-safety", "--package-name", "com.example.app")
	if err == nil || !strings.Contains(err.Error(), "--input is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppsDataSafety_ReadsCSVFile(t *testing.T) {
	var capturedPackage string
	var capturedCSV string
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/sa.json"},
				},
			}, nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				setDataSafetyFn: func(_ context.Context, packageName, safetyLabelsCSV string) error {
					capturedPackage = packageName
					capturedCSV = safetyLabelsCSV
					return nil
				},
			}, nil
		},
	}

	input := filepath.Join(t.TempDir(), "labels.csv")
	if err := os.WriteFile(input, []byte("question,answer\nfoo,bar\n"), 0o600); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}

	out, err := runApps(t, deps, "data-safety", "--package-name", "com.example.app", "--input", input)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if capturedPackage != "com.example.app" {
		t.Fatalf("unexpected package: %q", capturedPackage)
	}
	if capturedCSV != "question,answer\nfoo,bar\n" {
		t.Fatalf("unexpected CSV payload: %q", capturedCSV)
	}
}

func TestAppsDataSafety_ReadsStdin(t *testing.T) {
	var capturedCSV string
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/sa.json"},
				},
			}, nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				setDataSafetyFn: func(_ context.Context, _, safetyLabelsCSV string) error {
					capturedCSV = safetyLabelsCSV
					return nil
				},
			}, nil
		},
		Stdin: bytes.NewBufferString("question,answer\nstdin,yes\n"),
	}

	if _, err := runApps(t, deps, "data-safety", "--package-name", "com.example.app", "--input", "-"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if capturedCSV != "question,answer\nstdin,yes\n" {
		t.Fatalf("unexpected CSV payload: %q", capturedCSV)
	}
}

func TestAppsDataSafety_UsesGlobalPackageName(t *testing.T) {
	t.Helper()
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	cfg := &shared.GlobalFlags{}
	shared.BindGlobalFlags(fs, cfg)
	cfg.PackageName = "com.example.global"

	var capturedPackage string
	deps := Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/sa.json"},
				},
			}, nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				setDataSafetyFn: func(_ context.Context, packageName, _ string) error {
					capturedPackage = packageName
					return nil
				},
			}, nil
		},
		Stdin: bytes.NewBufferString("question,answer\nstdin,yes\n"),
	}

	if _, err := runApps(t, deps, "data-safety", "--input", "-"); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if capturedPackage != "com.example.global" {
		t.Fatalf("unexpected package: %q", capturedPackage)
	}
}
