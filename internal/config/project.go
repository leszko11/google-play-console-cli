package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const projectConfigName = ".gpc.yaml"

type ProjectConfig struct {
	PackageName     string `json:"packageName,omitempty" yaml:"package-name"`
	Profile         string `json:"profile,omitempty" yaml:"profile"`
	Output          string `json:"output,omitempty" yaml:"output"`
	DefaultTrack    string `json:"defaultTrack,omitempty" yaml:"default-track"`
	DefaultLocale   string `json:"defaultLocale,omitempty" yaml:"default-locale"`
	ListingDir      string `json:"listingDir,omitempty" yaml:"listing-dir"`
	ChangelogDir    string `json:"changelogDir,omitempty" yaml:"changelog-dir"`
	AppInitManifest string `json:"appInitManifest,omitempty" yaml:"appinit-manifest"`
	ReleaseManifest string `json:"releaseManifest,omitempty" yaml:"release-manifest"`
}

type ProjectConfigInfo struct {
	Path   string        `json:"path,omitempty"`
	Config ProjectConfig `json:"config,omitempty"`
}

func LoadProject() (ProjectConfigInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ProjectConfigInfo{}, err
	}
	return LoadProjectFromDir(cwd)
}

func LoadProjectFromDir(startDir string) (ProjectConfigInfo, error) {
	startDir = strings.TrimSpace(startDir)
	if startDir == "" {
		return ProjectConfigInfo{}, nil
	}

	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ProjectConfigInfo{}, err
	}

	for {
		path := filepath.Join(dir, projectConfigName)
		raw, readErr := os.ReadFile(path)
		if readErr == nil {
			cfg, err := parseProjectConfig(path, raw)
			if err != nil {
				return ProjectConfigInfo{}, err
			}
			return ProjectConfigInfo{Path: path, Config: cfg}, nil
		}
		if !os.IsNotExist(readErr) {
			return ProjectConfigInfo{}, readErr
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ProjectConfigInfo{}, nil
		}
		dir = parent
	}
}

func parseProjectConfig(path string, raw []byte) (ProjectConfig, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return ProjectConfig{}, nil
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return ProjectConfig{}, err
	}

	baseDir := filepath.Dir(path)
	cfg.PackageName = strings.TrimSpace(cfg.PackageName)
	cfg.Profile = strings.TrimSpace(cfg.Profile)
	cfg.Output = strings.TrimSpace(cfg.Output)
	cfg.DefaultTrack = strings.TrimSpace(cfg.DefaultTrack)
	cfg.DefaultLocale = strings.TrimSpace(cfg.DefaultLocale)
	cfg.ListingDir = resolveProjectPath(baseDir, cfg.ListingDir)
	cfg.ChangelogDir = resolveProjectPath(baseDir, cfg.ChangelogDir)
	cfg.AppInitManifest = resolveProjectPath(baseDir, cfg.AppInitManifest)
	cfg.ReleaseManifest = resolveProjectPath(baseDir, cfg.ReleaseManifest)
	return cfg, nil
}

func resolveProjectPath(baseDir, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(baseDir, value)
}
