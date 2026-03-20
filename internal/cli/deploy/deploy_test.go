package deploy

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	createEdit    gpc.EditInfo
	createEditErr error

	uploadBundle    gpc.BundleInfo
	uploadBundleErr error
	uploadAPK       gpc.APKInfo
	uploadAPKErr    error

	uploadMappingErr  error
	lastMappingType   string
	mappingUploadCall int

	updateTrackErr error
	lastTrackName  string
	lastTrack      gpc.TrackUpdate

	validateErr error
	commitErr   error
	deleteErr   error
	deleteCalls int
}

func (f *fakeClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	if f.createEditErr != nil {
		return gpc.EditInfo{}, f.createEditErr
	}
	if f.createEdit.ID == "" {
		return gpc.EditInfo{ID: "edit-1"}, nil
	}
	return f.createEdit, nil
}

func (f *fakeClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return f.deleteErr
}

func (f *fakeClient) ValidateEdit(_ context.Context, _, _ string) error {
	return f.validateErr
}

func (f *fakeClient) CommitEdit(_ context.Context, _, _ string, _ bool) (gpc.EditInfo, error) {
	if f.commitErr != nil {
		return gpc.EditInfo{}, f.commitErr
	}
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakeClient) UpdateTrack(_ context.Context, _, _, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
	f.lastTrackName = trackName
	f.lastTrack = update
	if f.updateTrackErr != nil {
		return gpc.TrackInfo{}, f.updateTrackErr
	}
	return gpc.TrackInfo{Name: trackName}, nil
}

func (f *fakeClient) UploadBundle(_ context.Context, _, _, _ string) (gpc.BundleInfo, error) {
	if f.uploadBundleErr != nil {
		return gpc.BundleInfo{}, f.uploadBundleErr
	}
	if f.uploadBundle.VersionCode == 0 {
		return gpc.BundleInfo{VersionCode: 123}, nil
	}
	return f.uploadBundle, nil
}

func (f *fakeClient) UploadAPK(_ context.Context, _, _, _ string) (gpc.APKInfo, error) {
	if f.uploadAPKErr != nil {
		return gpc.APKInfo{}, f.uploadAPKErr
	}
	if f.uploadAPK.VersionCode == 0 {
		return gpc.APKInfo{VersionCode: 123}, nil
	}
	return f.uploadAPK, nil
}

func (f *fakeClient) UploadDeobfuscationFile(_ context.Context, _, _ string, _ int64, fileType, _ string) (gpc.DeobfuscationFileInfo, error) {
	f.mappingUploadCall++
	f.lastMappingType = fileType
	if f.uploadMappingErr != nil {
		return gpc.DeobfuscationFileInfo{}, f.uploadMappingErr
	}
	return gpc.DeobfuscationFileInfo{SymbolType: fileType}, nil
}

func runDeploy(t *testing.T, deps Deps, args ...string) (string, error) {
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

func writeTempFile(t *testing.T, name string) string {
	return writeTempFileWithContents(t, name, "x")
}

func writeTempFileWithContents(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func writeReleaseNotesFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "release-notes.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write release notes file: %v", err)
	}
	return path
}

func TestDeployAABCommitSuccess(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"committed"`) || !strings.Contains(out, `"committed":true`) || !strings.Contains(out, `"uploadedArtifactType":"aab"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.lastTrackName != "internal" {
		t.Fatalf("expected internal track, got %q", client.lastTrackName)
	}
	if len(client.lastTrack.VersionCodes) != 1 || client.lastTrack.VersionCodes[0] != 123 {
		t.Fatalf("unexpected version codes: %+v", client.lastTrack.VersionCodes)
	}
}

func TestDeployRequiresExactlyOneArtifact(t *testing.T) {
	deps := Deps{}

	_, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--track", "internal",
		"--status", "completed",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "exactly one of --aab or --apk is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shared.IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}

	_, err = runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--apk", writeTempFile(t, "app.apk"),
		"--track", "internal",
		"--status", "completed",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "exactly one of --aab or --apk is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeployRequiresConfirmUnlessDryRun(t *testing.T) {
	deps := Deps{}

	_, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
	)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required unless --dry-run is set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeployDryRunDeletesEditAndSkipsCommit(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) || !strings.Contains(out, `"committed":false`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected one delete call for dry-run, got %d", client.deleteCalls)
	}
}

func TestDeployRequiresAllowProductionForProductionTrack(t *testing.T) {
	deps := Deps{}

	_, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "production",
		"--status", "completed",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "--allow-production is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeployFailurePerformsCleanupByDefault(t *testing.T) {
	client := &fakeClient{updateTrackErr: errors.New("conflict")}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "failed to update track") {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected cleanup delete call, got %d", client.deleteCalls)
	}
	if !strings.Contains(out, `"cleanupPerformed":true`) || !strings.Contains(out, `"status":"failed"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestDeployFailureCanDisableCleanup(t *testing.T) {
	client := &fakeClient{updateTrackErr: errors.New("conflict")}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--confirm",
		"--cleanup-on-failure=false",
	)
	if err == nil || !strings.Contains(err.Error(), "failed to update track") {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.deleteCalls != 0 {
		t.Fatalf("expected cleanup to be disabled, got delete calls %d", client.deleteCalls)
	}
	if !strings.Contains(out, `"cleanupPerformed":false`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestDeployMappingDefaultsToProguard(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--mapping-file", writeTempFile(t, "mapping.txt"),
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if client.mappingUploadCall != 1 {
		t.Fatalf("expected mapping upload call, got %d", client.mappingUploadCall)
	}
	if client.lastMappingType != "proguard" {
		t.Fatalf("expected default mapping type proguard, got %q", client.lastMappingType)
	}
}

func TestDeployPropagatesUpdatePriorityAndReleaseNotes(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--update-priority", "5",
		"--release-notes-locale", "pl-PL",
		"--release-notes-text", "Poprawki bledow",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if client.lastTrack.UpdatePriority != 5 {
		t.Fatalf("expected update priority 5, got %d", client.lastTrack.UpdatePriority)
	}
	if len(client.lastTrack.ReleaseNotes) != 1 {
		t.Fatalf("expected one release note, got %+v", client.lastTrack.ReleaseNotes)
	}
	if client.lastTrack.ReleaseNotes[0].Language != "pl-PL" || client.lastTrack.ReleaseNotes[0].Text != "Poprawki bledow" {
		t.Fatalf("unexpected release note payload: %+v", client.lastTrack.ReleaseNotes[0])
	}
}

func TestDeployRejectsInvalidUpdatePriority(t *testing.T) {
	_, err := runDeploy(
		t,
		Deps{},
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--update-priority", "7",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "--update-priority must be between 0 and 5") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeployRejectsBothReleaseNotesSources(t *testing.T) {
	_, err := runDeploy(
		t,
		Deps{},
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--release-notes-file", writeTempFile(t, "notes.txt"),
		"--release-notes-text", "Hello",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "only one of --release-notes-file or --release-notes-text can be set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeployParsesMultipleReleaseNotesFromTaggedFile(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	notesPath := writeTempFileWithContents(t, "notes.txt", `<pl-PL>
Poprawki bledow i ulepszenia stabilnosci.
</pl-PL>
<en-US>
Bug fixes and stability improvements.
</en-US>`)

	_, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--release-notes-file", notesPath,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if len(client.lastTrack.ReleaseNotes) != 2 {
		t.Fatalf("expected 2 release notes, got %+v", client.lastTrack.ReleaseNotes)
	}
	if client.lastTrack.ReleaseNotes[0].Language != "pl-PL" || client.lastTrack.ReleaseNotes[1].Language != "en-US" {
		t.Fatalf("unexpected release notes locales: %+v", client.lastTrack.ReleaseNotes)
	}
}

func TestDeployValidatesArtifactFileBeforeAPI(t *testing.T) {
	clientCreated := false
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			clientCreated = true
			return &fakeClient{}, nil
		},
	}

	_, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", filepath.Join(t.TempDir(), "missing.aab"),
		"--track", "internal",
		"--status", "completed",
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "artifact does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientCreated {
		t.Fatal("expected client not to be created when artifact file validation fails")
	}
}

func TestDeployPassesReleaseNotesToTrackUpdate(t *testing.T) {
	client := &fakeClient{}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	_, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--release-notes-file", writeReleaseNotesFile(t, `{"pl-PL":"Notatki wydania","en-US":"Release notes"}`),
		"--confirm",
	)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if len(client.lastTrack.ReleaseNotes) != 2 {
		t.Fatalf("expected release notes to be passed to track update, got %+v", client.lastTrack.ReleaseNotes)
	}
	if client.lastTrack.ReleaseNotes[0].Language != "en-US" || client.lastTrack.ReleaseNotes[1].Language != "pl-PL" {
		t.Fatalf("expected deterministic locale order, got %+v", client.lastTrack.ReleaseNotes)
	}
}

func TestDeployRejectsInvalidReleaseNotesFile(t *testing.T) {
	clientCreated := false
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			clientCreated = true
			return &fakeClient{}, nil
		},
	}

	_, err := runDeploy(
		t,
		deps,
		"--package-name", "com.example.app",
		"--aab", writeTempFile(t, "app.aab"),
		"--track", "internal",
		"--status", "completed",
		"--release-notes-file", writeReleaseNotesFile(t, `{"en-US":{"text":"invalid"}}`),
		"--confirm",
	)
	if err == nil || !strings.Contains(err.Error(), "--release-notes-file must be either a JSON object or array") {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientCreated {
		t.Fatal("expected client not to be created when release notes validation fails")
	}
}
