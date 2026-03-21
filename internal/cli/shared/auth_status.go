package shared

import (
	"fmt"
	"slices"
	"strings"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/config"
)

var authStatusKeychainAvailable = authresolver.KeychainAvailable

type AuthStatusSnapshot struct {
	ActiveProfile         string   `json:"activeProfile,omitempty"`
	SelectedProfile       string   `json:"selectedProfile,omitempty"`
	Authenticated         bool     `json:"authenticated"`
	Source                string   `json:"source,omitempty"`
	StorageBackend        string   `json:"storageBackend,omitempty"`
	ProfileStorage        string   `json:"profileStorage,omitempty"`
	ServiceAccountPath    string   `json:"serviceAccountPath,omitempty"`
	ManagedCredentialPath string   `json:"managedCredentialPath,omitempty"`
	LastValidatedAt       string   `json:"lastValidatedAt,omitempty"`
	DeveloperID           string   `json:"developerId,omitempty"`
	Warnings              []string `json:"warnings,omitempty"`
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
	out.ProfileStorage = strings.TrimSpace(profile.Storage)
	if managed, err := config.IsManagedServiceAccountPath(profile.ServiceAccountPath); err == nil && managed {
		out.ManagedCredentialPath = strings.TrimSpace(profile.ServiceAccountPath)
	}

	resolved, err := ResolveCredentials(cfg, lookupEnv)
	if err != nil {
		if !IsUsageError(err) {
			out.Warnings = append(out.Warnings, err.Error())
		}
	} else {
		out.Source = string(resolved.Source)
		out.ServiceAccountPath = resolved.ServiceAccountPath
		if out.ProfileStorage == "" {
			out.ProfileStorage = resolved.ProfileStorage
		}
		if out.ManagedCredentialPath == "" {
			if managed, err := config.IsManagedServiceAccountPath(resolved.ServiceAccountPath); err == nil && managed {
				out.ManagedCredentialPath = resolved.ServiceAccountPath
			}
		}
		out.Warnings = append(out.Warnings, resolved.Warnings...)
		if CredentialLocallyValid(resolved.Input) {
			out.Authenticated = true
		}
	}

	switch out.Source {
	case string(authresolver.SourceKeychain):
		out.StorageBackend = config.StorageKeychain
	case string(authresolver.SourceFlag), string(authresolver.SourceEnv), string(authresolver.SourceConfig):
		if strings.TrimSpace(out.ServiceAccountPath) != "" {
			out.StorageBackend = config.StoragePath
		}
	}

	if out.StorageBackend == "" {
		switch out.ProfileStorage {
		case config.StoragePath:
			out.StorageBackend = config.StoragePath
		case config.StorageKeychain:
			if resolveCredentialsShouldBypassKeychain(lookupEnv) && strings.TrimSpace(profile.ServiceAccountPath) != "" {
				out.StorageBackend = config.StoragePath
				appendAuthStatusWarning(&out.Warnings, "keychain bypassed via GPC_BYPASS_KEYCHAIN")
			} else {
				out.StorageBackend = config.StorageKeychain
			}
		default:
			keychainAvailable, keychainErr := authStatusKeychainAvailable(lookupEnv)
			switch {
			case resolveCredentialsShouldBypassKeychain(lookupEnv):
				out.StorageBackend = config.StoragePath
				appendAuthStatusWarning(&out.Warnings, "keychain bypassed via GPC_BYPASS_KEYCHAIN")
			case keychainAvailable:
				out.StorageBackend = config.StorageKeychain
			default:
				out.StorageBackend = config.StoragePath
				if keychainErr != nil {
					appendAuthStatusWarning(&out.Warnings, fmt.Sprintf("keychain error: %v", keychainErr))
				} else {
					appendAuthStatusWarning(&out.Warnings, "system keychain unavailable; using config/environment/flags")
				}
			}
		}
	}

	return out
}

func appendAuthStatusWarning(warnings *[]string, warning string) {
	warning = strings.TrimSpace(warning)
	if warning == "" || slices.Contains(*warnings, warning) {
		return
	}
	*warnings = append(*warnings, warning)
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
	if status.ProfileStorage != "" {
		parts = append(parts, "profileStorage="+status.ProfileStorage)
	}
	parts = append(parts, fmt.Sprintf("authenticated=%t", status.Authenticated))
	if status.DeveloperID != "" {
		parts = append(parts, "developerId="+status.DeveloperID)
	}
	return strings.Join(parts, " ")
}
