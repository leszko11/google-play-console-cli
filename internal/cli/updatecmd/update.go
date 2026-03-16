package updatecmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const (
	defaultRepoOwner = "leszko11"
	defaultRepoName  = "google-play-console-cli"
	defaultAPIBase   = "https://api.github.com"
)

var releaseVersionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Deps struct {
	HTTPClient     httpDoer
	LookupEnv      func(string) string
	ExecutablePath func() (string, error)
	EvalSymlinks   func(string) (string, error)
	CurrentVersion string
	Stdout         io.Writer
	Stderr         io.Writer
	APIBaseURL     string
	RuntimeGOOS    string
	RuntimeGOARCH  string
}

type releaseResponse struct {
	TagName string         `json:"tag_name"`
	HTMLURL string         `json:"html_url"`
	Assets  []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name               string `json:"name"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type versionTuple struct {
	Major int
	Minor int
	Patch int
}

type updateResult struct {
	CurrentVersion string `json:"currentVersion"`
	TargetVersion  string `json:"targetVersion"`
	Status         string `json:"status"`
	NeedsUpdate    bool   `json:"needsUpdate"`
	CheckOnly      bool   `json:"checkOnly"`
	ExecutablePath string `json:"executablePath,omitempty"`
	AssetName      string `json:"assetName,omitempty"`
	ReleaseURL     string `json:"releaseUrl,omitempty"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var check bool
	var confirm bool
	var version string
	fs.BoolVar(&check, "check", false, "Check for updates without downloading or replacing the binary")
	fs.BoolVar(&confirm, "confirm", false, "Download and replace the current gpc binary")
	fs.StringVar(&version, "version", "", "Install a specific release tag (for example v0.3.0)")

	return &ffcli.Command{
		Name:      "update",
		ShortHelp: "Check for and install newer gpc releases",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if check && confirm {
				return shared.UsageErrorf("--check and --confirm cannot be used together")
			}
			if !check && !confirm {
				return shared.UsageErrorf("--confirm is required unless --check is set")
			}

			result, err := runUpdate(ctx, deps, strings.TrimSpace(version), check)
			if writeErr := shared.WriteJSON(deps.Stdout, result); writeErr != nil {
				return writeErr
			}
			return err
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.HTTPClient == nil {
		deps.HTTPClient = http.DefaultClient
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.Getenv
	}
	if deps.ExecutablePath == nil {
		deps.ExecutablePath = os.Executable
	}
	if deps.EvalSymlinks == nil {
		deps.EvalSymlinks = filepath.EvalSymlinks
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	if deps.APIBaseURL == "" {
		deps.APIBaseURL = defaultAPIBase
	}
	if deps.RuntimeGOOS == "" {
		deps.RuntimeGOOS = runtimeGOOS()
	}
	if deps.RuntimeGOARCH == "" {
		deps.RuntimeGOARCH = runtimeGOARCH()
	}
	return deps
}

func runUpdate(ctx context.Context, deps Deps, requestedVersion string, checkOnly bool) (updateResult, error) {
	currentVersion := strings.TrimSpace(deps.CurrentVersion)
	if currentVersion == "" {
		currentVersion = "dev"
	}
	execPath, err := deps.ExecutablePath()
	if err != nil {
		return updateResult{}, fmt.Errorf("failed to resolve current executable: %w", err)
	}
	resolvedExecPath, err := resolveExecutablePath(execPath, deps.EvalSymlinks)
	if err != nil {
		return updateResult{}, fmt.Errorf("failed to resolve current executable path: %w", err)
	}

	if isHomebrewInstall(resolvedExecPath) {
		return updateResult{}, fmt.Errorf("homebrew-managed install detected at %s\nhint: use `brew upgrade gpc` instead of `gpc update`", resolvedExecPath)
	}
	if deps.RuntimeGOOS == "windows" {
		return updateResult{}, fmt.Errorf("self-update is not supported on windows yet")
	}

	desiredTag := strings.TrimSpace(requestedVersion)
	if desiredTag != "" {
		normalized, err := normalizeReleaseTag(desiredTag)
		if err != nil {
			return updateResult{}, shared.UsageErrorf("%v", err)
		}
		desiredTag = normalized
	}

	requestCtx, cancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
	defer cancel()

	release, err := fetchRelease(requestCtx, deps, desiredTag)
	if err != nil {
		return updateResult{}, err
	}

	result := updateResult{
		CurrentVersion: currentVersion,
		TargetVersion:  strings.TrimSpace(release.TagName),
		CheckOnly:      checkOnly,
		ExecutablePath: resolvedExecPath,
		ReleaseURL:     strings.TrimSpace(release.HTMLURL),
	}

	assetName, archiveURL, expectedSHA, err := selectReleaseAsset(release, deps.RuntimeGOOS, deps.RuntimeGOARCH)
	if err != nil {
		return result, err
	}
	result.AssetName = assetName

	if sameVersion(currentVersion, result.TargetVersion) {
		result.Status = "up-to-date"
		result.NeedsUpdate = false
		return result, nil
	}

	result.Status = "update-available"
	result.NeedsUpdate = true
	if checkOnly {
		return result, nil
	}

	archive, err := downloadAsset(requestCtx, deps, archiveURL)
	if err != nil {
		return result, err
	}
	if err := verifySHA256(archive, expectedSHA); err != nil {
		return result, err
	}

	binaryName := platformBinaryName(deps.RuntimeGOOS)
	rawBinary, err := extractBinary(archive, assetName, binaryName)
	if err != nil {
		return result, err
	}
	if err := replaceExecutable(resolvedExecPath, rawBinary); err != nil {
		return result, err
	}

	result.Status = "updated"
	return result, nil
}

func fetchRelease(ctx context.Context, deps Deps, tag string) (releaseResponse, error) {
	var endpoint string
	if tag == "" {
		endpoint = strings.TrimRight(deps.APIBaseURL, "/") + "/repos/" + defaultRepoOwner + "/" + defaultRepoName + "/releases/latest"
	} else {
		endpoint = strings.TrimRight(deps.APIBaseURL, "/") + "/repos/" + defaultRepoOwner + "/" + defaultRepoName + "/releases/tags/" + tag
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return releaseResponse{}, fmt.Errorf("failed to create release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gpc-self-update")
	if token := resolveGitHubToken(deps.LookupEnv); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := deps.HTTPClient.Do(req)
	if err != nil {
		return releaseResponse{}, fmt.Errorf("failed to check latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return releaseResponse{}, fmt.Errorf("github releases api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return releaseResponse{}, fmt.Errorf("failed to decode release response: %w", err)
	}
	if strings.TrimSpace(release.TagName) == "" {
		return releaseResponse{}, fmt.Errorf("github release response did not include tag_name")
	}
	return release, nil
}

func selectReleaseAsset(release releaseResponse, goos, goarch string) (string, string, string, error) {
	archiveName := releaseArchiveName(strings.TrimSpace(release.TagName), goos, goarch)
	checksumName := fmt.Sprintf("gpc_%s_checksums.txt", strings.TrimSpace(release.TagName))

	var archiveAsset *releaseAsset
	var checksumAsset *releaseAsset
	for i := range release.Assets {
		asset := &release.Assets[i]
		switch strings.TrimSpace(asset.Name) {
		case archiveName:
			archiveAsset = asset
		case checksumName:
			checksumAsset = asset
		}
	}

	if archiveAsset == nil {
		return "", "", "", fmt.Errorf("no release asset found for %s/%s", goos, goarch)
	}
	if url := strings.TrimSpace(archiveAsset.BrowserDownloadURL); url == "" {
		return "", "", "", fmt.Errorf("release asset %q did not include a browser download url", archiveAsset.Name)
	}

	if digest := strings.TrimSpace(archiveAsset.Digest); strings.HasPrefix(digest, "sha256:") {
		return archiveAsset.Name, archiveAsset.BrowserDownloadURL, strings.TrimPrefix(digest, "sha256:"), nil
	}
	if checksumAsset == nil {
		return "", "", "", fmt.Errorf("release did not include checksum metadata for %q", archiveAsset.Name)
	}
	return archiveAsset.Name, archiveAsset.BrowserDownloadURL, "", fmt.Errorf("checksum fallback is not implemented for release asset %q", archiveAsset.Name)
}

func downloadAsset(ctx context.Context, deps Deps, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "gpc-self-update")
	if token := resolveGitHubToken(deps.LookupEnv); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := deps.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download update asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("update asset download returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read update asset: %w", err)
	}
	return raw, nil
}

func verifySHA256(raw []byte, expected string) error {
	expected = strings.TrimSpace(strings.ToLower(expected))
	if expected == "" {
		return fmt.Errorf("missing expected sha256 checksum")
	}
	sum := sha256.Sum256(raw)
	actual := hex.EncodeToString(sum[:])
	if actual != expected {
		return fmt.Errorf("downloaded asset checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func extractBinary(archive []byte, assetName, binaryName string) ([]byte, error) {
	switch {
	case strings.HasSuffix(assetName, ".tar.gz"):
		return extractTarGZBinary(archive, binaryName)
	case strings.HasSuffix(assetName, ".zip"):
		return extractZipBinary(archive, binaryName)
	default:
		return nil, fmt.Errorf("unsupported release archive format %q", assetName)
	}
}

func extractTarGZBinary(archive []byte, binaryName string) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("failed to open tar.gz release archive: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar.gz release archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != binaryName {
			continue
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("failed to extract %s from release archive: %w", binaryName, err)
		}
		return raw, nil
	}
	return nil, fmt.Errorf("release archive did not contain %s", binaryName)
}

func extractZipBinary(archive []byte, binaryName string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip release archive: %w", err)
	}
	for _, file := range reader.File {
		if filepath.Base(file.Name) != binaryName {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open %s from release archive: %w", binaryName, err)
		}
		raw, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to extract %s from release archive: %w", binaryName, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("failed to close %s from release archive: %w", binaryName, closeErr)
		}
		return raw, nil
	}
	return nil, fmt.Errorf("release archive did not contain %s", binaryName)
}

func replaceExecutable(execPath string, raw []byte) error {
	info, err := os.Stat(execPath)
	if err != nil {
		return fmt.Errorf("failed to stat current executable: %w", err)
	}
	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o755
	}

	tmpPath := execPath + ".tmp"
	if err := os.WriteFile(tmpPath, raw, mode); err != nil {
		return fmt.Errorf("failed to stage updated binary: %w", err)
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to set updated binary permissions: %w", err)
	}
	if err := os.Rename(tmpPath, execPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace current binary: %w", err)
	}
	return nil
}

func resolveExecutablePath(execPath string, evalSymlinks func(string) (string, error)) (string, error) {
	if execPath == "" {
		return "", fmt.Errorf("current executable path is empty")
	}
	if evalSymlinks == nil {
		return execPath, nil
	}
	resolved, err := evalSymlinks(execPath)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(resolved) == "" {
		return execPath, nil
	}
	return resolved, nil
}

func releaseArchiveName(tag, goos, goarch string) string {
	suffix := ".tar.gz"
	if goos == "windows" {
		suffix = ".zip"
	}
	return fmt.Sprintf("gpc_%s_%s_%s%s", tag, goos, goarch, suffix)
}

func platformBinaryName(goos string) string {
	if goos == "windows" {
		return "gpc.exe"
	}
	return "gpc"
}

func normalizeReleaseTag(raw string) (string, error) {
	match := releaseVersionPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return "", fmt.Errorf("--version must look like vMAJOR.MINOR.PATCH")
	}
	return "v" + match[1] + "." + match[2] + "." + match[3], nil
}

func sameVersion(current, target string) bool {
	current = strings.TrimSpace(current)
	target = strings.TrimSpace(target)
	if current == "" || target == "" {
		return false
	}
	return normalizeComparableVersion(current) == normalizeComparableVersion(target)
}

func normalizeComparableVersion(raw string) string {
	if normalized, err := normalizeReleaseTag(raw); err == nil {
		return normalized
	}
	return strings.TrimSpace(raw)
}

func isHomebrewInstall(execPath string) bool {
	lower := strings.ToLower(filepath.ToSlash(execPath))
	return strings.Contains(lower, "/cellar/") || strings.Contains(lower, "/homebrew/")
}

func resolveGitHubToken(lookupEnv func(string) string) string {
	if lookupEnv == nil {
		return ""
	}
	if token := strings.TrimSpace(lookupEnv("GITHUB_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(lookupEnv("GH_TOKEN"))
}

func parseReleaseVersion(raw string) (versionTuple, bool) {
	match := releaseVersionPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return versionTuple{}, false
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return versionTuple{}, false
	}
	minor, err := strconv.Atoi(match[2])
	if err != nil {
		return versionTuple{}, false
	}
	patch, err := strconv.Atoi(match[3])
	if err != nil {
		return versionTuple{}, false
	}
	return versionTuple{Major: major, Minor: minor, Patch: patch}, true
}
