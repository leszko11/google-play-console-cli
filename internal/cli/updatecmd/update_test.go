package updatecmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateCheckReportsAvailableRelease(t *testing.T) {
	server := newReleaseServer(t, "v0.4.0", "linux", "amd64", archiveTarGZ(t, "gpc", []byte("new-binary")))
	defer server.Close()

	execPath := writeExecutable(t, "old-binary")
	out, err := runUpdateCommand(t, Deps{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return execPath, nil },
		EvalSymlinks:   func(path string) (string, error) { return path, nil },
		CurrentVersion: "v0.3.0",
		RuntimeGOOS:    "linux",
		RuntimeGOARCH:  "amd64",
	}, "--check")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	for _, want := range []string{`"status":"update-available"`, `"currentVersion":"v0.3.0"`, `"targetVersion":"v0.4.0"`, `"checkOnly":true`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %s", want, out)
		}
	}
}

func TestUpdateConfirmReplacesExecutableFromTarGZ(t *testing.T) {
	server := newReleaseServer(t, "v0.4.0", "linux", "amd64", archiveTarGZ(t, "gpc", []byte("new-binary")))
	defer server.Close()

	execPath := writeExecutable(t, "old-binary")
	out, err := runUpdateCommand(t, Deps{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return execPath, nil },
		EvalSymlinks:   func(path string) (string, error) { return path, nil },
		CurrentVersion: "v0.3.0",
		RuntimeGOOS:    "linux",
		RuntimeGOARCH:  "amd64",
	}, "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"updated"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	raw, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read exec: %v", err)
	}
	if string(raw) != "new-binary" {
		t.Fatalf("unexpected executable contents: %q", string(raw))
	}
}

func TestUpdateConfirmReplacesExecutableFromZip(t *testing.T) {
	server := newReleaseServer(t, "v0.4.0", "windows", "amd64", archiveZip(t, "gpc.exe", []byte("new-binary")))
	defer server.Close()

	execPath := filepath.Join(t.TempDir(), "gpc.exe")
	if err := os.WriteFile(execPath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("write exec: %v", err)
	}

	result, err := executeUpdateForTest(server, execPath, "windows", "amd64", "v0.3.0", "v0.4.0")
	if err == nil || !strings.Contains(err.Error(), "self-update is not supported on windows yet") {
		t.Fatalf("unexpected error: %v result=%+v", err, result)
	}
}

func TestUpdateCheckReportsUpToDate(t *testing.T) {
	server := newReleaseServer(t, "v0.3.0", "linux", "amd64", archiveTarGZ(t, "gpc", []byte("same")))
	defer server.Close()

	execPath := writeExecutable(t, "same")
	out, err := runUpdateCommand(t, Deps{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return execPath, nil },
		EvalSymlinks:   func(path string) (string, error) { return path, nil },
		CurrentVersion: "v0.3.0",
		RuntimeGOOS:    "linux",
		RuntimeGOARCH:  "amd64",
	}, "--check")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"up-to-date"`) || !strings.Contains(out, `"needsUpdate":false`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUpdateRejectsHomebrewInstall(t *testing.T) {
	_, err := runUpdateCommand(t, Deps{
		CurrentVersion: "v0.3.0",
		ExecutablePath: func() (string, error) { return "/opt/homebrew/Cellar/gpc/0.3.0/bin/gpc", nil },
		EvalSymlinks:   func(path string) (string, error) { return path, nil },
		RuntimeGOOS:    "darwin",
		RuntimeGOARCH:  "arm64",
	}, "--check")
	if err == nil || !strings.Contains(err.Error(), "brew upgrade gpc") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateRequiresConfirmUnlessCheck(t *testing.T) {
	_, err := runUpdateCommand(t, Deps{}, "--version", "v0.3.0")
	if err == nil || !strings.Contains(err.Error(), "--confirm is required unless --check is set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeReleaseTag(t *testing.T) {
	got, err := normalizeReleaseTag("0.3.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v0.3.0" {
		t.Fatalf("unexpected tag %q", got)
	}
}

func TestParseReleaseVersion(t *testing.T) {
	got, ok := parseReleaseVersion("v1.2.3")
	if !ok || got.Major != 1 || got.Minor != 2 || got.Patch != 3 {
		t.Fatalf("unexpected parsed version: %+v ok=%v", got, ok)
	}
}

func runUpdateCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func executeUpdateForTest(server *httptest.Server, execPath, goos, goarch, current, target string) (updateResult, error) {
	return runUpdate(context.Background(), Deps{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return execPath, nil },
		EvalSymlinks:   func(path string) (string, error) { return path, nil },
		CurrentVersion: current,
		RuntimeGOOS:    goos,
		RuntimeGOARCH:  goarch,
	}, target, false)
}

func newReleaseServer(t *testing.T, tag, goos, goarch string, archive []byte) *httptest.Server {
	t.Helper()

	assetName := releaseArchiveName(tag, goos, goarch)
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	releaseBody := fmt.Sprintf(`{"tag_name":%q,"html_url":"https://example.com/releases/%s","assets":[{"name":%q,"digest":"sha256:%s","browser_download_url":"__DOWNLOAD__"}]}`, tag, tag, assetName, digest)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/leszko11/google-play-console-cli/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, strings.ReplaceAll(releaseBody, "__DOWNLOAD__", server.URL+"/downloads/"+assetName))
		case "/repos/leszko11/google-play-console-cli/releases/tags/" + tag:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, strings.ReplaceAll(releaseBody, "__DOWNLOAD__", server.URL+"/downloads/"+assetName))
		case "/downloads/" + assetName:
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	return server
}

func archiveTarGZ(t *testing.T, name string, raw []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(raw))}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(raw); err != nil {
		t.Fatalf("write tar body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}

func archiveZip(t *testing.T, name string, raw []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write(raw); err != nil {
		t.Fatalf("write zip body: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func writeExecutable(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gpc")
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write exec: %v", err)
	}
	return path
}
