package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const envConfigPath = "GPC_CONFIG_PATH"

type Profile struct {
	ServiceAccountPath string `json:"serviceAccountPath,omitempty"`
	LastValidatedAt    string `json:"lastValidatedAt,omitempty"`
}

type Config struct {
	ActiveProfile string             `json:"activeProfile,omitempty"`
	Packages      []string           `json:"packages,omitempty"`
	Profiles      map[string]Profile `json:"profiles,omitempty"`
}

func Path() (string, error) {
	if fromEnv := os.Getenv(envConfigPath); fromEnv != "" {
		return fromEnv, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".gpc", "config.json"), nil
}

func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return Config{}, err
	}

	raw, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return Config{}, nil
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Save(cfg Config) error {
	p, err := Path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, append(raw, '\n'), 0o600)
}
