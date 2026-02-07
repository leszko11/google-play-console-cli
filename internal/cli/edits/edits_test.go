package edits

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	create    gpc.EditInfo
	createErr error
	get       gpc.EditInfo
	getErr    error
	validate  error
	commit    gpc.EditInfo
	commitErr error
	deleteErr error
}

func (f fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	return f.create, f.createErr
}
func (f fakeClient) GetEdit(_ context.Context, _, _ string) (gpc.EditInfo, error) {
	return f.get, f.getErr
}
func (f fakeClient) ValidateEdit(_ context.Context, _, _ string) error { return f.validate }
func (f fakeClient) CommitEdit(_ context.Context, _, _ string) (gpc.EditInfo, error) {
	return f.commit, f.commitErr
}
func (f fakeClient) DeleteEdit(_ context.Context, _, _ string) error { return f.deleteErr }

func runEdits(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
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

func TestEditsCreate(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{create: gpc.EditInfo{ID: "edit-1"}}, nil
		},
	}

	out, err := runEdits(t, deps, "create", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"id":"edit-1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsValidate(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	out, err := runEdits(t, deps, "validate", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"validated"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsCommit_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{commitErr: errors.New("conflict")}, nil
		},
	}

	_, err := runEdits(t, deps, "commit", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to commit edit") {
		t.Fatalf("unexpected error: %v", err)
	}
}
