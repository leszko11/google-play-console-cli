package shared

import (
	"fmt"
	"strings"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/config"
)

type AuthStatusSnapshot struct {
	ActiveProfile      string   `json:"activeProfile,omitempty"`
	SelectedProfile    string   `json:"selectedProfile,omitempty"`
	Authenticated      bool     `json:"authenticated"`
	Source             string   `json:"source,omitempty"`
	StorageBackend     string   `json:"storageBackend,omitempty"`
	ServiceAccountPath string   `json:"serviceAccountPath,omitempty"`
	LastValidatedAt    string   `json:"lastValidatedAt,omitempty"`
	DeveloperID        string   `json:"developerId,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
}

func BuildAuthStatusSnapshot(cfg config.Config, lookupEnv func(string) string) AuthStatusSnapshot {
	if lookupEnv == nil {
		lookupEnv = func(string) string { return "" }
	}

	selectedProfile := ResolveProfileName(cfg)
	out := AuthStatusSnapshot{
		ActiveProfile:   cfg.ActiveProfile,
		SelectedProfile: selectedProfile,
		Authenticated:   false,
	}

	profile, ok := cfg.Profiles[selectedProfile]
	if !ok {
		profile = config.Profile{}
	}
	out.LastValidatedAt = profile.LastValidatedAt
	out.DeveloperID = profile.DeveloperID

	keychainAvailable, keychainErr := authresolver.KeychainAvailable(lookupEnv)
	switch {
	case authresolver.ShouldBypassKeychain(lookupEnv):
		out.StorageBackend = "config"
		out.Warnings = append(out.Warnings, "keychain bypassed via GPC_BYPASS_KEYCHAIN")
	case keychainAvailable:
		out.StorageBackend = "keychain"
	case keychainErr != nil:
		out.StorageBackend = "config"
		out.Warnings = append(out.Warnings, fmt.Sprintf("keychain error: %v", keychainErr))
	default:
		out.StorageBackend = "config"
		out.Warnings = append(out.Warnings, "system keychain unavailable; using config/environment/flags")
	}

	resolved, err := ResolveCredentials(cfg, lookupEnv)
	if err != nil {
		if !IsUsageError(err) {
			out.Warnings = append(out.Warnings, err.Error())
		}
		return out
	}
	out.Source = string(resolved.Source)
	out.ServiceAccountPath = resolved.ServiceAccountPath
	out.Warnings = append(out.Warnings, resolved.Warnings...)
	if CredentialLocallyValid(resolved.Input) {
		out.Authenticated = true
	}
	return out
}

func AuthStatusSummary(status AuthStatusSnapshot) string {
	parts := make([]string, 0, 6)
	if status.SelectedProfile != "" {
		parts = append(parts, "profile="+status.SelectedProfile)
	}
	if status.Source != "" {
		parts = append(parts, "source="+status.Source)
	}
	if status.StorageBackend != "" {
		parts = append(parts, "backend="+status.StorageBackend)
	}
	parts = append(parts, fmt.Sprintf("authenticated=%t", status.Authenticated))
	if status.DeveloperID != "" {
		parts = append(parts, "developerId="+status.DeveloperID)
	}
	return strings.Join(parts, " ")
}
