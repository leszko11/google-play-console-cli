package release

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeReleaseClient struct {
	verifyErr error

	createEditIDs   []string
	createEditErr   error
	deleteEditErr   error
	validateEditErr error
	commitEditErr   error
	uploadBundle    gpc.BundleInfo
	uploadBundleErr error
	updateTrackErr  error
	getTrackInfo    gpc.TrackInfo
	getTrackErr     error

	commitCalls   int
	deleteCalls   int
	createCalls   int
	lastTrack     gpc.TrackUpdate
	lastTrackName string
}

func (f *fakeReleaseClient) VerifyPackageAccess(_ context.Context, _ string) error {
	return f.verifyErr
}

func (f *fakeReleaseClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	if f.createEditErr != nil {
		return gpc.EditInfo{}, f.createEditErr
	}
	f.createCalls++
	if len(f.createEditIDs) >= f.createCalls {
		return gpc.EditInfo{ID: f.createEditIDs[f.createCalls-1]}, nil
	}
	return gpc.EditInfo{ID: fmt.Sprintf("edit-%d", f.createCalls)}, nil
}

func (f *fakeReleaseClient) DeleteEdit(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return f.deleteEditErr
}

func (f *fakeReleaseClient) ValidateEdit(_ context.Context, _, _ string) error {
	return f.validateEditErr
}

func (f *fakeReleaseClient) CommitEdit(_ context.Context, _, _ string) (gpc.EditInfo, error) {
	if f.commitEditErr != nil {
		return gpc.EditInfo{}, f.commitEditErr
	}
	f.commitCalls++
	return gpc.EditInfo{ID: "committed"}, nil
}

func (f *fakeReleaseClient) UpdateTrack(_ context.Context, _, _, trackName string, update gpc.TrackUpdate) (gpc.TrackInfo, error) {
	f.lastTrackName = trackName
	f.lastTrack = update
	if f.updateTrackErr != nil {
		return gpc.TrackInfo{}, f.updateTrackErr
	}
	return gpc.TrackInfo{Name: trackName}, nil
}

func (f *fakeReleaseClient) UploadBundle(_ context.Context, _, _, _ string) (gpc.BundleInfo, error) {
	if f.uploadBundleErr != nil {
		return gpc.BundleInfo{}, f.uploadBundleErr
	}
	if f.uploadBundle.VersionCode == 0 {
		return gpc.BundleInfo{VersionCode: 123}, nil
	}
	return f.uploadBundle, nil
}

func (f *fakeReleaseClient) GetTrack(_ context.Context, _, _, _ string) (gpc.TrackInfo, error) {
	if f.getTrackErr != nil {
		return gpc.TrackInfo{}, f.getTrackErr
	}
	if f.getTrackInfo.Name == "" {
		return gpc.TrackInfo{
			Name: "alpha",
			Releases: []gpc.TrackReleaseInfo{
				{
					Status:       "completed",
					VersionCodes: []int64{123},
				},
			},
		}, nil
	}
	return f.getTrackInfo, nil
}

type fakeReleaseReportingClient struct {
	queryResults    []gpc.ReportingVitalsQueryResult
	queryErr        error
	queryCalls      int
	capturedPackage []string
	capturedMetrics []gpc.ReportingVitalsMetricSet
}

func (f *fakeReleaseReportingClient) QueryVitalsMetricSet(_ context.Context, packageName string, metricSet gpc.ReportingVitalsMetricSet, _ *gpc.ReportingVitalsQueryRequest) (gpc.ReportingVitalsQueryResult, error) {
	f.queryCalls++
	f.capturedPackage = append(f.capturedPackage, packageName)
	f.capturedMetrics = append(f.capturedMetrics, metricSet)
	if f.queryErr != nil {
		return gpc.ReportingVitalsQueryResult{}, f.queryErr
	}
	if len(f.queryResults) == 0 {
		return gpc.ReportingVitalsQueryResult{}, nil
	}
	if f.queryCalls <= len(f.queryResults) {
		return f.queryResults[f.queryCalls-1], nil
	}
	return f.queryResults[len(f.queryResults)-1], nil
}

func baseReleaseDeps(t *testing.T, client *fakeReleaseClient) Deps {
	t.Helper()
	return Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				ActiveProfile: "default",
				Profiles: map[string]config.Profile{
					"default": {ServiceAccountPath: "/tmp/service-account.json"},
				},
			}, nil
		},
		NewClient: func(context.Context, gpc.CredentialInput) (Client, error) {
			return client, nil
		},
		NewReportingClient: func(context.Context, gpc.CredentialInput) (ReportingClient, error) {
			return &fakeReleaseReportingClient{}, nil
		},
		LookupEnv: func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		},
		RunCommand: func(_ context.Context, _ string, name string, args ...string) (string, error) {
			switch {
			case name == "java":
				return `openjdk version "21.0.10"`, nil
			case name == "./gradlew" && strings.Join(args, " ") == ":app:tasks --all":
				return "bundleStagingRelease - Assembles bundle for variant stagingRelease", nil
			case name == "env":
				return "BUILD OK", nil
			default:
				return "", nil
			}
		},
		Now: func() time.Time {
			return time.Unix(1_750_000_000, 0)
		},
		Sleep: func(_ context.Context, _ time.Duration) error {
			return nil
		},
	}
}

func mustPackageNotFoundErr() error {
	return fmt.Errorf("%w: Package not found", gpc.ErrPackageNotFound)
}

func assertContainsStep(t *testing.T, steps []alphaStep, name string, status string) {
	t.Helper()
	for _, step := range steps {
		if step.Name == name && step.Status == status {
			return
		}
	}
	t.Fatalf("missing step %q with status %q: %+v", name, status, steps)
}

func writeFakeAAB(t *testing.T, path string, signed bool) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fake aab: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	writeEntry := func(name, contents string) {
		t.Helper()
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := io.WriteString(entry, contents); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}

	writeEntry("BundleConfig.pb", "bundle-config")
	writeEntry("base/manifest/AndroidManifest.xml", "<manifest package=\"com.example.app\" />")
	if signed {
		writeEntry("META-INF/BUNDLE.RSA", "signed")
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return path
}

func TestValidateAlphaOptions(t *testing.T) {
	err := validateAlphaOptions(alphaOptions{
		PackageName:    "com.example.app",
		Track:          "alpha",
		ReleaseStatus:  "completed",
		Confirm:        true,
		UpdatePriority: 2,
		UserFraction:   -1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = validateAlphaOptions(alphaOptions{
		PackageName:   "com.example.app",
		Track:         "production",
		ReleaseStatus: "completed",
		Confirm:       true,
		UserFraction:  -1,
	})
	if err == nil || !strings.Contains(err.Error(), "--allow-production is required") {
		t.Fatalf("unexpected production validation error: %v", err)
	}
	if !shared.IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}

	err = validateAlphaOptions(alphaOptions{
		PackageName:    "com.example.app",
		Track:          "alpha",
		ReleaseStatus:  "completed",
		Confirm:        true,
		UpdatePriority: 9,
		UserFraction:   -1,
	})
	if err == nil || !strings.Contains(err.Error(), "--update-priority must be between 0 and 5") {
		t.Fatalf("unexpected priority validation error: %v", err)
	}
	if !shared.IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}
}

func TestResolveVersionInfo(t *testing.T) {
	code, name, err := resolveVersionInfo(alphaOptions{
		VersionCode: 12345,
		VersionName: "1.2.3",
	}, func(string) string { return "" }, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 12345 || name != "1.2.3" {
		t.Fatalf("unexpected explicit version info: %d %q", code, name)
	}

	code, _, err = resolveVersionInfo(alphaOptions{}, func(key string) string {
		if key == "APP_VERSION_CODE" {
			return "200"
		}
		return ""
	}, nil)
	if err != nil || code != 200 {
		t.Fatalf("expected env version code, got %d err=%v", code, err)
	}

	_, _, err = resolveVersionInfo(alphaOptions{}, func(string) string { return "invalid" }, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid APP_VERSION_CODE") {
		t.Fatalf("unexpected invalid env error: %v", err)
	}
}

func TestMustPackageNotFoundErr(t *testing.T) {
	if !errors.Is(mustPackageNotFoundErr(), gpc.ErrPackageNotFound) {
		t.Fatal("expected package-not-found wrapping")
	}
}
