package edits

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	create             gpc.EditInfo
	createErr          error
	get                gpc.EditInfo
	getErr             error
	getFn              func(packageName, editID string) (gpc.EditInfo, error)
	validate           error
	validateFn         func(packageName, editID string) error
	commit             gpc.EditInfo
	commitErr          error
	commitFn           func(packageName, editID string) (gpc.EditInfo, error)
	deleteErr          error
	deleteFn           func(packageName, editID string) error
	appDetails         gpc.AppDetailsInfo
	appDetailsErr      error
	updateDetailsErr   error
	updateDetailsFn    func(packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error)
	replaceDetailsErr  error
	replaceDetailsFn   func(packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error)
	testers            gpc.TestersInfo
	testersErr         error
	updateTestersErr   error
	updateTestersFn    func(packageName, editID, track string, googleGroups []string) (gpc.TestersInfo, error)
	replaceTestersErr  error
	replaceTestersFn   func(packageName, editID, track string, googleGroups []string) (gpc.TestersInfo, error)
	countryAvail       gpc.CountryAvailabilityInfo
	countryAvailErr    error
	listing            gpc.ListingInfo
	listings           []gpc.ListingInfo
	listErr            error
	updateErr          error
	updateListingFn    func(packageName, editID, language string, update gpc.ListingUpdate) (gpc.ListingInfo, error)
	replaceErr         error
	replaceListingFn   func(packageName, editID, language string, update gpc.ListingUpdate) (gpc.ListingInfo, error)
	delListErr         error
	delAllErr          error
	images             []gpc.ImageInfo
	image              gpc.ImageInfo
	imagesErr          error
	uploadImageErr     error
	deleteImageErr     error
	deleteAllImgErr    error
	uploadImageFn      func(packageName, editID, language, imageType, imagePath string) (gpc.ImageInfo, error)
	expansionFile      gpc.ExpansionFileInfo
	expansionFileErr   error
	uploadExpansionErr error
	patchExpansionErr  error
	updateExpansionErr error
	getExpansionFn     func(packageName, editID string, apkVersionCode int64, expansionFileType string) (gpc.ExpansionFileInfo, error)
	patchExpansionFn   func(packageName, editID string, apkVersionCode int64, expansionFileType string, referencesVersion int64) (gpc.ExpansionFileInfo, error)
	updateExpansionFn  func(packageName, editID string, apkVersionCode int64, expansionFileType string, referencesVersion int64) (gpc.ExpansionFileInfo, error)
	uploadExpansionFn  func(packageName, editID string, apkVersionCode int64, expansionFileType, filePath string) (gpc.ExpansionFileInfo, error)
}

func (f fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	return f.create, f.createErr
}
func (f fakeClient) GetEdit(_ context.Context, packageName, editID string) (gpc.EditInfo, error) {
	if f.getFn != nil {
		return f.getFn(packageName, editID)
	}
	return f.get, f.getErr
}
func (f fakeClient) ValidateEdit(_ context.Context, packageName, editID string) error {
	if f.validateFn != nil {
		return f.validateFn(packageName, editID)
	}
	return f.validate
}
func (f fakeClient) CommitEdit(_ context.Context, packageName, editID string) (gpc.EditInfo, error) {
	if f.commitFn != nil {
		return f.commitFn(packageName, editID)
	}
	return f.commit, f.commitErr
}
func (f fakeClient) DeleteEdit(_ context.Context, packageName, editID string) error {
	if f.deleteFn != nil {
		return f.deleteFn(packageName, editID)
	}
	return f.deleteErr
}
func (f fakeClient) GetAppDetails(_ context.Context, _, _ string) (gpc.AppDetailsInfo, error) {
	return f.appDetails, f.appDetailsErr
}
func (f fakeClient) UpdateAppDetails(_ context.Context, packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error) {
	if f.updateDetailsFn != nil {
		return f.updateDetailsFn(packageName, editID, update)
	}
	return f.appDetails, f.updateDetailsErr
}
func (f fakeClient) ReplaceAppDetails(_ context.Context, packageName, editID string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error) {
	if f.replaceDetailsFn != nil {
		return f.replaceDetailsFn(packageName, editID, update)
	}
	return f.appDetails, f.replaceDetailsErr
}
func (f fakeClient) GetTesters(_ context.Context, _, _, _ string) (gpc.TestersInfo, error) {
	return f.testers, f.testersErr
}
func (f fakeClient) UpdateTesters(_ context.Context, packageName, editID, track string, googleGroups []string) (gpc.TestersInfo, error) {
	if f.updateTestersFn != nil {
		return f.updateTestersFn(packageName, editID, track, googleGroups)
	}
	return f.testers, f.updateTestersErr
}
func (f fakeClient) ReplaceTesters(_ context.Context, packageName, editID, track string, googleGroups []string) (gpc.TestersInfo, error) {
	if f.replaceTestersFn != nil {
		return f.replaceTestersFn(packageName, editID, track, googleGroups)
	}
	return f.testers, f.replaceTestersErr
}
func (f fakeClient) GetCountryAvailability(_ context.Context, _, _, _ string) (gpc.CountryAvailabilityInfo, error) {
	return f.countryAvail, f.countryAvailErr
}
func (f fakeClient) GetListing(_ context.Context, _, _, _ string) (gpc.ListingInfo, error) {
	return f.listing, f.listErr
}
func (f fakeClient) ListListings(_ context.Context, _, _ string) ([]gpc.ListingInfo, error) {
	return f.listings, f.listErr
}
func (f fakeClient) UpdateListing(_ context.Context, packageName, editID, language string, update gpc.ListingUpdate) (gpc.ListingInfo, error) {
	if f.updateListingFn != nil {
		return f.updateListingFn(packageName, editID, language, update)
	}
	return f.listing, f.updateErr
}
func (f fakeClient) ReplaceListing(_ context.Context, packageName, editID, language string, update gpc.ListingUpdate) (gpc.ListingInfo, error) {
	if f.replaceListingFn != nil {
		return f.replaceListingFn(packageName, editID, language, update)
	}
	return f.listing, f.replaceErr
}
func (f fakeClient) DeleteListing(_ context.Context, _, _, _ string) error  { return f.delListErr }
func (f fakeClient) DeleteAllListings(_ context.Context, _, _ string) error { return f.delAllErr }
func (f fakeClient) ListImages(_ context.Context, _, _, _, _ string) ([]gpc.ImageInfo, error) {
	return f.images, f.imagesErr
}
func (f fakeClient) UploadImage(_ context.Context, packageName, editID, language, imageType, imagePath string) (gpc.ImageInfo, error) {
	if f.uploadImageFn != nil {
		return f.uploadImageFn(packageName, editID, language, imageType, imagePath)
	}
	return f.image, f.uploadImageErr
}
func (f fakeClient) DeleteImage(_ context.Context, _, _, _, _, _ string) error {
	return f.deleteImageErr
}
func (f fakeClient) DeleteAllImages(_ context.Context, _, _, _, _ string) ([]gpc.ImageInfo, error) {
	return f.images, f.deleteAllImgErr
}
func (f fakeClient) GetExpansionFile(_ context.Context, packageName, editID string, apkVersionCode int64, expansionFileType string) (gpc.ExpansionFileInfo, error) {
	if f.getExpansionFn != nil {
		return f.getExpansionFn(packageName, editID, apkVersionCode, expansionFileType)
	}
	return f.expansionFile, f.expansionFileErr
}
func (f fakeClient) PatchExpansionFile(_ context.Context, packageName, editID string, apkVersionCode int64, expansionFileType string, referencesVersion int64) (gpc.ExpansionFileInfo, error) {
	if f.patchExpansionFn != nil {
		return f.patchExpansionFn(packageName, editID, apkVersionCode, expansionFileType, referencesVersion)
	}
	return f.expansionFile, f.patchExpansionErr
}
func (f fakeClient) UpdateExpansionFile(_ context.Context, packageName, editID string, apkVersionCode int64, expansionFileType string, referencesVersion int64) (gpc.ExpansionFileInfo, error) {
	if f.updateExpansionFn != nil {
		return f.updateExpansionFn(packageName, editID, apkVersionCode, expansionFileType, referencesVersion)
	}
	return f.expansionFile, f.updateExpansionErr
}
func (f fakeClient) UploadExpansionFile(_ context.Context, packageName, editID string, apkVersionCode int64, expansionFileType, filePath string) (gpc.ExpansionFileInfo, error) {
	if f.uploadExpansionFn != nil {
		return f.uploadExpansionFn(packageName, editID, apkVersionCode, expansionFileType, filePath)
	}
	return f.expansionFile, f.uploadExpansionErr
}

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

func TestEditsCommit_DryRunValidatesWithoutCommitting(t *testing.T) {
	var validateCalls, commitCalls int
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				validateFn: func(packageName, editID string) error {
					validateCalls++
					if packageName != "com.example.app" || editID != "edit-1" {
						t.Fatalf("unexpected validate target %q %q", packageName, editID)
					}
					return nil
				},
				commitFn: func(string, string) (gpc.EditInfo, error) {
					commitCalls++
					return gpc.EditInfo{}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(t, deps, "commit", "--package-name", "com.example.app", "--edit-id", "edit-1", "--dry-run")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if validateCalls != 1 {
		t.Fatalf("expected one validate call, got %d", validateCalls)
	}
	if commitCalls != 0 {
		t.Fatalf("expected no commit calls in dry-run, got %d", commitCalls)
	}
	if !strings.Contains(out, `"status":"dry-run"`) || !strings.Contains(out, `"validated":true`) {
		t.Fatalf("unexpected output: %s", out)
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

func TestEditsDelete_DryRunChecksEditWithoutDeleting(t *testing.T) {
	var getCalls, deleteCalls int
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				getFn: func(packageName, editID string) (gpc.EditInfo, error) {
					getCalls++
					if packageName != "com.example.app" || editID != "edit-1" {
						t.Fatalf("unexpected get target %q %q", packageName, editID)
					}
					return gpc.EditInfo{ID: "edit-1", ExpiryTimeSeconds: "1712345678"}, nil
				},
				deleteFn: func(string, string) error {
					deleteCalls++
					return nil
				},
			}, nil
		},
	}

	out, err := runEdits(t, deps, "delete", "--package-name", "com.example.app", "--edit-id", "edit-1", "--dry-run")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if getCalls != 1 {
		t.Fatalf("expected one get call, got %d", getCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("expected no delete calls in dry-run, got %d", deleteCalls)
	}
	if !strings.Contains(out, `"status":"dry-run"`) || !strings.Contains(out, `"id":"edit-1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsListingsGet(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{listing: gpc.ListingInfo{Language: "en-US", Title: "Example App"}}, nil
		},
	}

	out, err := runEdits(t, deps, "listings", "get", "--package-name", "com.example.app", "--edit-id", "edit-1", "--locale", "en-US")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"title":"Example App"`) {
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

func TestEditsDetailsUpdate_MethodUpdateUsesReplaceEndpoint(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				replaceDetailsFn: func(_, _ string, update gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error) {
					if update.ContactEmail != "support@example.com" {
						t.Fatalf("unexpected contact email: %q", update.ContactEmail)
					}
					return gpc.AppDetailsInfo{ContactEmail: update.ContactEmail}, nil
				},
				updateDetailsFn: func(_, _ string, _ gpc.AppDetailsUpdate) (gpc.AppDetailsInfo, error) {
					t.Fatal("expected replace details method, got patch update")
					return gpc.AppDetailsInfo{}, nil
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
		"--method", "update",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"method":"update"`) {
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

func TestEditsTestersGet_ReturnsTesters(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				testers: gpc.TestersInfo{
					Track:        "internal",
					GoogleGroups: []string{"qa-team@example.com"},
				},
			}, nil
		},
	}

	out, err := runEdits(t, deps, "testers", "get", "--package-name", "com.example.app", "--edit-id", "edit-1", "--track", "internal")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"track":"internal"`) || !strings.Contains(out, `"qa-team@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsTestersUpdate_ReturnsStatusUpdated(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateTestersFn: func(_, _, track string, googleGroups []string) (gpc.TestersInfo, error) {
					if track != "internal" {
						t.Fatalf("unexpected track: %q", track)
					}
					if len(googleGroups) != 2 || googleGroups[0] != "qa-team@example.com" || googleGroups[1] != "beta@example.com" {
						t.Fatalf("unexpected google groups: %#v", googleGroups)
					}
					return gpc.TestersInfo{Track: track, GoogleGroups: googleGroups}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"testers",
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--track", "internal",
		"--google-groups", "qa-team@example.com,beta@example.com",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) || !strings.Contains(out, `"qa-team@example.com"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsTestersUpdate_MethodUpdateUsesReplaceEndpoint(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				replaceTestersFn: func(_, _, track string, googleGroups []string) (gpc.TestersInfo, error) {
					if track != "internal" {
						t.Fatalf("unexpected track: %q", track)
					}
					return gpc.TestersInfo{Track: track, GoogleGroups: googleGroups}, nil
				},
				updateTestersFn: func(_, _, _ string, _ []string) (gpc.TestersInfo, error) {
					t.Fatal("expected replace testers method, got patch update")
					return gpc.TestersInfo{}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"testers",
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--track", "internal",
		"--google-groups", "qa-team@example.com",
		"--method", "update",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"method":"update"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsTestersUpdate_RequiresGoogleGroups(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(
		t,
		deps,
		"testers",
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--track", "internal",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--google-groups is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsCountryAvailabilityGet_ReturnsAvailability(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				countryAvail: gpc.CountryAvailabilityInfo{
					Track:       "production",
					RestOfWorld: true,
					Countries: []gpc.CountryTargetedInfo{
						{CountryCode: "PL"},
						{CountryCode: "US"},
					},
				},
			}, nil
		},
	}

	out, err := runEdits(t, deps, "country-availability", "get", "--package-name", "com.example.app", "--edit-id", "edit-1", "--track", "production")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"restOfWorld":true`) || !strings.Contains(out, `"countryCode":"PL"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsCountryAvailabilityGet_RequiresTrack(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(t, deps, "country-availability", "get", "--package-name", "com.example.app", "--edit-id", "edit-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--track is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsListingsList(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{listings: []gpc.ListingInfo{{Language: "en-US", Title: "Example App"}, {Language: "pl-PL", Title: "Example App PL"}}}, nil
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
			return fakeClient{listing: gpc.ListingInfo{Language: "en-US", Title: "Example App Test"}}, nil
		},
	}

	out, err := runEdits(t, deps, "listings", "update", "--package-name", "com.example.app", "--edit-id", "edit-1", "--locale", "en-US", "--title", "Example App Test")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsListingsUpdate_MethodUpdateUsesReplaceEndpoint(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				replaceListingFn: func(_, _, locale string, update gpc.ListingUpdate) (gpc.ListingInfo, error) {
					if locale != "en-US" {
						t.Fatalf("unexpected locale: %q", locale)
					}
					return gpc.ListingInfo{Language: locale, Title: update.Title}, nil
				},
				updateListingFn: func(_, _, _ string, _ gpc.ListingUpdate) (gpc.ListingInfo, error) {
					t.Fatal("expected replace listing method, got patch update")
					return gpc.ListingInfo{}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"listings",
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--locale", "en-US",
		"--title", "Example App Test",
		"--method", "update",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"method":"update"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsListingsUpdate_RejectsInvalidMethod(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(
		t,
		deps,
		"listings",
		"update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--locale", "en-US",
		"--title", "Example App Test",
		"--method", "replace",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--method must be one of: patch, update") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsListingsBatchUpdate_DryRun(t *testing.T) {
	dir := t.TempDir()
	writeListingLocaleFile(t, dir, "en-US", `{"title":"Title EN","shortDescription":"Short EN"}`)
	writeListingLocaleFile(t, dir, "pl-PL", `{"title":"Title PL","fullDescription":"Full PL"}`)

	updateCalls := 0
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateListingFn: func(_, _, _ string, _ gpc.ListingUpdate) (gpc.ListingInfo, error) {
					updateCalls++
					return gpc.ListingInfo{}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"listings",
		"batch-update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--from-dir", dir,
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if updateCalls != 0 {
		t.Fatalf("expected no API calls in dry-run, got %d", updateCalls)
	}
	if !strings.Contains(out, `"status":"planned"`) || !strings.Contains(out, `"locale":"en-US"`) || !strings.Contains(out, `"locale":"pl-PL"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsListingsBatchUpdate_PartialFailureReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeListingLocaleFile(t, dir, "en-US", `{"title":"Title EN"}`)
	writeListingLocaleFile(t, dir, "pl-PL", `{"title":"Title PL"}`)

	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				updateListingFn: func(_, _, locale string, update gpc.ListingUpdate) (gpc.ListingInfo, error) {
					if locale == "pl-PL" {
						return gpc.ListingInfo{}, errors.New("boom")
					}
					return gpc.ListingInfo{Language: locale, Title: update.Title}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"listings",
		"batch-update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--from-dir", dir,
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "listing locale updates failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"locale":"en-US"`) || !strings.Contains(out, `"status":"updated"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, `"locale":"pl-PL"`) || !strings.Contains(out, `"status":"error"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsListingsBatchUpdate_LocalesFilterDeterministic(t *testing.T) {
	dir := t.TempDir()
	writeListingLocaleFile(t, dir, "fr-FR", `{"title":"FR"}`)
	writeListingLocaleFile(t, dir, "en-US", `{"title":"EN"}`)
	writeListingLocaleFile(t, dir, "pl-PL", `{"title":"PL"}`)

	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"listings",
		"batch-update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--from-dir", dir,
		"--locales", "pl-PL,en-US",
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if strings.Contains(out, `"locale":"fr-FR"`) {
		t.Fatalf("did not expect filtered-out locale in output: %s", out)
	}
	enIdx := strings.Index(out, `"locale":"en-US"`)
	plIdx := strings.Index(out, `"locale":"pl-PL"`)
	if enIdx == -1 || plIdx == -1 {
		t.Fatalf("expected locales missing from output: %s", out)
	}
	if enIdx > plIdx {
		t.Fatalf("expected deterministic sorted output, got: %s", out)
	}
}

func TestEditsListingsBatchUpdate_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	writeListingLocaleFile(t, dir, "en-US", `{"title":`)

	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(
		t,
		deps,
		"listings",
		"batch-update",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--from-dir", dir,
		"--dry-run",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("unexpected error: %v", err)
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

func TestEditsImagesList_ReturnsImages(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				images: []gpc.ImageInfo{
					{ID: "img-1", URL: "https://example.com/1.png"},
					{ID: "img-2", URL: "https://example.com/2.png"},
				},
			}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"images",
		"list",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--locale", "en-US",
		"--image-type", "phoneScreenshots",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"imageType":"phoneScreenshots"`) || !strings.Contains(out, `"id":"img-1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsImagesUpload_ReturnsUploaded(t *testing.T) {
	imagePath := writePNG(t, 320, 320)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				uploadImageFn: func(_, _, _, imageType, gotPath string) (gpc.ImageInfo, error) {
					if imageType != "phoneScreenshots" {
						t.Fatalf("unexpected image type: %q", imageType)
					}
					if gotPath != imagePath {
						t.Fatalf("unexpected image path: %q", gotPath)
					}
					return gpc.ImageInfo{ID: "img-uploaded"}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"images",
		"upload",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--locale", "en-US",
		"--image-type", "phoneScreenshots",
		"--file", imagePath,
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"uploaded"`) || !strings.Contains(out, `"id":"img-uploaded"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsImagesUpload_RejectsWrongDimensionsForIcon(t *testing.T) {
	imagePath := writePNG(t, 320, 320)
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(
		t,
		deps,
		"images",
		"upload",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--locale", "en-US",
		"--image-type", "icon",
		"--file", imagePath,
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires dimensions 512x512") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsImagesDelete_ReturnsDeleted(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"images",
		"delete",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--locale", "en-US",
		"--image-type", "phoneScreenshots",
		"--image-id", "img-1",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deleted"`) || !strings.Contains(out, `"imageId":"img-1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsImagesDeleteAll_ReturnsDeletedAll(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{images: []gpc.ImageInfo{{ID: "img-1"}}}, nil
		},
	}

	out, err := runEdits(
		t,
		deps,
		"images",
		"delete-all",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--locale", "en-US",
		"--image-type", "phoneScreenshots",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"deleted_all"`) || !strings.Contains(out, `"id":"img-1"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsImagesList_RequiresImageType(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(
		t,
		deps,
		"images",
		"list",
		"--package-name", "com.example.app",
		"--edit-id", "edit-1",
		"--locale", "en-US",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--image-type is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsExpansionFilesGet_ReturnsExpansionFile(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{expansionFile: gpc.ExpansionFileInfo{FileSize: 123456}}, nil
		},
	}

	out, err := runEdits(t, deps, "expansion-files", "get", "--package-name", "com.example.app", "--edit-id", "edit-1", "--apk-version-code", "123", "--expansion-file-type", "main")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"fileSize":123456`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsExpansionFilesPatch_ReturnsPatched(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				patchExpansionFn: func(_ string, _ string, apkVersionCode int64, expansionFileType string, referencesVersion int64) (gpc.ExpansionFileInfo, error) {
					if expansionFileType != "patch" || apkVersionCode != 123 || referencesVersion != 456 {
						t.Fatalf("unexpected patch args: type=%s apk=%d ref=%d", expansionFileType, apkVersionCode, referencesVersion)
					}
					return gpc.ExpansionFileInfo{ReferencesVersion: referencesVersion}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(t, deps, "expansion-files", "patch", "--package-name", "com.example.app", "--edit-id", "edit-1", "--apk-version-code", "123", "--expansion-file-type", "patch", "--references-version", "456")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"patched"`) || !strings.Contains(out, `"referencesVersion":456`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestEditsExpansionFilesUpdate_RequiresReferencesVersion(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{}, nil
		},
	}

	_, err := runEdits(t, deps, "expansion-files", "update", "--package-name", "com.example.app", "--edit-id", "edit-1", "--apk-version-code", "123", "--expansion-file-type", "main")
	if err == nil || !strings.Contains(err.Error(), "--references-version must be greater than zero") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditsExpansionFilesUpload_ReturnsUploaded(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "main.obb")
	if err := os.WriteFile(filePath, []byte("obb-data"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return fakeClient{
				uploadExpansionFn: func(_ string, _ string, apkVersionCode int64, expansionFileType, uploadPath string) (gpc.ExpansionFileInfo, error) {
					if expansionFileType != "main" || apkVersionCode != 123 || uploadPath != filePath {
						t.Fatalf("unexpected upload args: type=%s apk=%d path=%s", expansionFileType, apkVersionCode, uploadPath)
					}
					return gpc.ExpansionFileInfo{FileSize: 8}, nil
				},
			}, nil
		},
	}

	out, err := runEdits(t, deps, "expansion-files", "upload", "--package-name", "com.example.app", "--edit-id", "edit-1", "--apk-version-code", "123", "--expansion-file-type", "main", "--file", filePath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"uploaded"`) || !strings.Contains(out, `"fileSize":8`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func writeListingLocaleFile(t *testing.T, dir, locale, content string) {
	t.Helper()
	path := filepath.Join(dir, locale+".json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write listing file: %v", err)
	}
}

func writePNG(t *testing.T, width, height int) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 0x22, G: 0x44, B: 0x88, A: 0xff})
		}
	}

	path := filepath.Join(t.TempDir(), "test.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	return path
}
