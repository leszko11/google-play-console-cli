package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type BootstrapState struct {
	PackageName              string  `json:"packageName,omitempty"`
	PackageReadiness         string  `json:"packageReadiness,omitempty"`
	BootstrapDraftExists     bool    `json:"bootstrapDraftExists,omitempty"`
	BootstrapVersionCodes    []int64 `json:"bootstrapVersionCodes,omitempty"`
	LastBootstrapCommittedAt string  `json:"lastBootstrapCommittedAt,omitempty"`
	LastReadinessRecheck     string  `json:"lastReadinessRecheck,omitempty"`
}

type BootstrapStateInfo struct {
	Path   string
	Exists bool
	State  BootstrapState
}

type BootstrapDraftInfo struct {
	Exists       bool    `json:"exists,omitempty"`
	TrackName    string  `json:"trackName,omitempty"`
	VersionCodes []int64 `json:"versionCodes,omitempty"`
}

type BootstrapDraftClient interface {
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	DeleteEdit(ctx context.Context, packageName, editID string) error
	GetTrack(ctx context.Context, packageName, editID, trackName string) (gpc.TrackInfo, error)
}

func BootstrapStatePathForWorkspace(workspaceDir string) string {
	workspaceDir = strings.TrimSpace(workspaceDir)
	if workspaceDir == "" {
		return ""
	}
	return filepath.Join(workspaceDir, "bootstrap-state.json")
}

func BootstrapStatePathFromReleaseManifest(manifestPath string) string {
	manifestPath = strings.TrimSpace(manifestPath)
	if manifestPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(manifestPath), "bootstrap-state.json")
}

func ReadBootstrapState(path string) (BootstrapStateInfo, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return BootstrapStateInfo{}, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return BootstrapStateInfo{Path: path}, nil
		}
		return BootstrapStateInfo{}, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return BootstrapStateInfo{Path: path, Exists: true}, nil
	}

	var state BootstrapState
	if err := json.Unmarshal(raw, &state); err != nil {
		return BootstrapStateInfo{}, err
	}
	state.PackageName = strings.TrimSpace(state.PackageName)
	state.PackageReadiness = strings.TrimSpace(state.PackageReadiness)
	state.LastBootstrapCommittedAt = strings.TrimSpace(state.LastBootstrapCommittedAt)
	state.LastReadinessRecheck = strings.TrimSpace(state.LastReadinessRecheck)
	state.BootstrapVersionCodes = append([]int64(nil), state.BootstrapVersionCodes...)
	return BootstrapStateInfo{Path: path, Exists: true, State: state}, nil
}

func WriteBootstrapState(path string, state BootstrapState) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload := BootstrapState{
		PackageName:              strings.TrimSpace(state.PackageName),
		PackageReadiness:         strings.TrimSpace(state.PackageReadiness),
		BootstrapDraftExists:     state.BootstrapDraftExists,
		BootstrapVersionCodes:    append([]int64(nil), state.BootstrapVersionCodes...),
		LastBootstrapCommittedAt: strings.TrimSpace(state.LastBootstrapCommittedAt),
		LastReadinessRecheck:     strings.TrimSpace(state.LastReadinessRecheck),
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

func DetectBootstrapDraftState(ctx context.Context, client BootstrapDraftClient, packageName string) (BootstrapDraftInfo, error) {
	edit, err := client.CreateEdit(ctx, packageName)
	if err != nil {
		return BootstrapDraftInfo{}, fmt.Errorf("failed to inspect internal bootstrap track: %w", err)
	}
	defer client.DeleteEdit(ctx, packageName, edit.ID)

	track, err := client.GetTrack(ctx, packageName, edit.ID, "internal")
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "not found") {
			return BootstrapDraftInfo{}, nil
		}
		return BootstrapDraftInfo{}, fmt.Errorf("failed to inspect internal bootstrap track: %w", err)
	}

	for _, release := range track.Releases {
		if strings.EqualFold(strings.TrimSpace(release.Status), "draft") {
			return BootstrapDraftInfo{
				Exists:       true,
				TrackName:    track.Name,
				VersionCodes: append([]int64(nil), release.VersionCodes...),
			}, nil
		}
	}
	return BootstrapDraftInfo{}, nil
}

func JoinVersionCodes(values []int64) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

func RecommendedReleaseCommand(packageName, readiness, workspaceDir, releaseManifestPath string) string {
	packageName = strings.TrimSpace(packageName)
	readiness = strings.TrimSpace(readiness)
	workspaceDir = strings.TrimSpace(workspaceDir)
	releaseManifestPath = strings.TrimSpace(releaseManifestPath)
	if releaseManifestPath == "" && workspaceDir != "" {
		releaseManifestPath = filepath.Join(workspaceDir, "release.yaml")
	}

	switch PackageReadiness(readiness) {
	case PackageReadinessUninitialized:
		if packageName == "" {
			return ""
		}
		if workspaceDir == "" {
			workspaceDir = "./play"
		}
		return fmt.Sprintf("gpc release init --package-name %s --dir %s", packageName, workspaceDir)
	case PackageReadinessDraftBootstrapRequired, PackageReadinessReady:
		if releaseManifestPath != "" {
			return fmt.Sprintf("gpc release full --manifest %s --confirm", releaseManifestPath)
		}
		if packageName != "" {
			return fmt.Sprintf("gpc release init --package-name %s --dir ./play", packageName)
		}
	}
	if packageName != "" {
		return fmt.Sprintf("gpc doctor --package-name %s", packageName)
	}
	return ""
}
