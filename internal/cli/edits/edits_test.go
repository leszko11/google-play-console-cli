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
	create           gpc.EditInfo
	createErr        error
	get              gpc.EditInfo
	getErr           error
	validate         error
	commit           gpc.EditInfo
	commitErr        error
	deleteErr        error
	appDetails       gpc.AppDetailsInfo
	appDetailsErr    error
	updateDetailsErr error
	updateDetailsFn  func(packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error)
	listing          gpc.ListingInfo
	listings         []gpc.ListingInfo
	listErr          error
	updateErr        error
	delListErr       error
	delAllErr        error
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
func (f fakeClient) GetAppDetails(_ context.Context, _, _ string) (gpc.AppDetailsInfo, error) {
	return f.appDetails, f.appDetailsErr
}
func (f fakeClient) UpdateAppDetails(_ context.Context, packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error) {
	if f.updateDetailsFn != nil {
		return f.updateDetailsFn(packageName, editID, update)
	}
	return f.appDetails, f.updateDetailsErr
}
func (f fakeClient) GetListing(_ context.Context, _, _, _ string) (gpc.ListingInfo, error) {
	return f.listing, f.listErr
}
func (f fakeClient) ListListings(_ context.Context, _, _ string) ([]gpc.ListingInfo, error) {
	return f.listings, f.listErr
}
func (f fakeClient) UpdateListing(_ context.Context, _, _, _ string, _ gpc.ListingUpdate) (gpc.ListingInfo, error) {
	return f.listing, f.updateErr
}
func (f fakeClient) DeleteListing(_ context.Context, _, _, _ string) error  { return f.delListErr }
func (f fakeClient) DeleteAllListings(_ context.Context, _, _ string) error { return f.delAllErr }

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

	_, err := runEdits(t, deps, "commit", "--package-name", "com.example.app", "--edit-id", "edit-1", "--confirm")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to commit edit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsCommit_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{commit: gpc.EditInfo{ID: "edit-1"}}, nil
		},
	}

	_, err := runEdits(t, deps, "commit", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsDelete_RequiresConfirm(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(t, deps, "delete", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsListingsGet(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{listing: gpc.ListingInfo{Language: "en-US", Title: "PeakMe"}}, nil
		},
	}

	out, err := runEdits(t, deps, "listings", "get", "--package-name", "com.example.app", "--edit-id", "edit-1", "--locale", "en-US")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"title":"PeakMe"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsDetailsGet_ReturnsDetails(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				appDetails: gpc.AppDetailsInfo{
					DefaultLanguage: "en-US",
					ContactEmail:    "support@example.com",
				},
			}, nil
		},
	}

	out, err := runEdits(t, deps, "details", "get", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"defaultLanguage":"en-US"`) || !strings.Contains(out, `"contactEmail":"support@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsDetailsGet_RequiresEditID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(t, deps, "details", "get", "--package-name", "com.example.app")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--edit-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsDetailsUpdate_ReturnsStatusUpdated(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateDetailsFn: func(_, _ string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error) {
					if update.ContactEmail != "support@example.com" {
						t.Fatalf("unexpected contact email: %q", update.ContactEmail)
					}
					return gpc.AppDetailsInfo{
						DefaultLanguage: "en-US",
						ContactEmail:    update.ContactEmail,
					}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"details",
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--contact-email", "support@example.com",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"contactEmail":"support@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsDetailsUpdate_ReturnsAPIError(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{updateDetailsErr: errors.New("invalid details payload")}, nil
		},
	}

	_, err := runEdits(
		t,
		deps,
		"details",
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--contact-email", "support@example.com",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to update app details") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsListingsList(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{listings: []gpc.ListingInfo{{Language: "en-US", Title: "PeakMe"}, {Language: "pl-PL", Title: "PeakMe PL"}}}, nil
		},
	}

	out, err := runEdits(t, deps, "listings", "list", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"language":"en-US"`) || !strings.Contains(out, `"language":"pl-PL"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsListingsUpdate_ReturnsStatusUpdated(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{listing: gpc.ListingInfo{Language: "en-US", Title: "PeakMe Test"}}, nil
		},
	}

	out, err := runEdits(t, deps, "listings", "update", "--package-name", "com.example.app", "--edit-id", "edit-1", "--locale", "en-US", "--title", "PeakMe Test")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsListingsDelete_ReturnsStatusDeleted(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	out, err := runEdits(t, deps, "listings", "delete", "--package-name", "com.example.app", "--edit-id", "edit-1", "--locale", "en-US")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deleted"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsListingsDeleteAll_ReturnsStatusDeletedAll(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	out, err := runEdits(t, deps, "listings", "delete-all", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deleted_all"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsListingsDelete_RequiresLocale(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(t, deps, "listings", "delete", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--locale is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
