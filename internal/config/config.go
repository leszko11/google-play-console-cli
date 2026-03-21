package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const envConfigPath = "GPC_CONFIG_PATH"

const (
	StorageKeychain = "keychain"
	StoragePath     = "path"
)

type Profile struct {
	ServiceAccountPath string `json:"serviceAccountPath,omitempty"`
	Storage            string `json:"storage,omitempty"`
	LastValidatedAt    string `json:"lastValidatedAt,omitempty"`
	DeveloperID        string `json:"developerId,omitempty"`
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

func BaseDir() (string, error) {
	p, err := Path()
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

func ManagedCredentialsDir() (string, error) {
	baseDir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "credentials"), nil
}

func ManagedServiceAccountPath(profile string) (string, error) {
	dir, err := ManagedCredentialsDir()
	if err != nil {
		return "", err
	}
	name := sanitizeProfileName(profile)
	if name == "" {
		name = "default"
	}
	return filepath.Join(dir, name+".json"), nil
}

func EnsureManagedCredentialsDir() (string, error) {
	dir, err := ManagedCredentialsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func WriteManagedServiceAccount(profile string, payload []byte) (string, error) {
	if _, err := EnsureManagedCredentialsDir(); err != nil {
		return "", err
	}
	path, err := ManagedServiceAccountPath(profile)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(append([]byte(nil), payload...), '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func IsManagedServiceAccountPath(path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, nil
	}
	managedDir, err := ManagedCredentialsDir()
	if err != nil {
		return false, err
	}
	managedDir, err = filepath.Abs(managedDir)
	if err != nil {
		return false, err
	}
	cleaned, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(managedDir, cleaned)
	if err != nil {
		return false, err
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

func sanitizeProfileName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		" ", "-",
	)
	return replacer.Replace(value)
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

	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, append(raw, '\n'), 0o600)
}
