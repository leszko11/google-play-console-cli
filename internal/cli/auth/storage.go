package auth

import (
	"fmt"
	"os"
	"strings"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/config"
)

const (
	storageAuto = "auto"
)

var storageShouldBypassKeychain = authresolver.ShouldBypassKeychain
var storageStoreProfileCredential = authresolver.StoreProfileCredential
var storageRemoveProfileCredential = authresolver.RemoveProfileCredential
var storageIsCredentialNotFound = authresolver.IsCredentialNotFound
var storageIsKeyringUnavailable = authresolver.IsKeyringUnavailable

func normalizeStorageChoice(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = storageAuto
	}
	switch value {
	case storageAuto, config.StorageKeychain, config.StoragePath:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported --storage %q (expected auto, keychain, or path)", raw)
	}
}

func resolveStorageChoice(raw string) string {
	if strings.TrimSpace(raw) == config.StorageKeychain {
		return config.StorageKeychain
	}
	return config.StoragePath
}

func persistedServiceAccountPath(storage, sourcePath string) string {
	sourcePath = strings.TrimSpace(sourcePath)
	switch storage {
	case config.StoragePath:
		return sourcePath
	case config.StorageKeychain:
		if sourcePath == "" {
			return ""
		}
		managed, err := config.IsManagedServiceAccountPath(sourcePath)
		if err == nil && managed {
			return ""
		}
		return sourcePath
	default:
		return sourcePath
	}
}

func storeProfileForBackend(profile string, current config.Profile, storage string, serviceAccountPath string, serviceAccountJSON []byte, managePath bool, lookupEnv func(string) string) (config.Profile, []string, error) {
	warnings := []string{}
	next := current

	switch storage {
	case config.StoragePath:
		path := strings.TrimSpace(serviceAccountPath)
		if managePath {
			managedPath, err := config.WriteManagedServiceAccount(profile, serviceAccountJSON)
			if err != nil {
				return config.Profile{}, nil, fmt.Errorf("failed to store managed service account: %w", err)
			}
			path = managedPath
		}
		next.Storage = config.StoragePath
		next.ServiceAccountPath = path

		if !storageShouldBypassKeychain(lookupEnv) {
			if err := storageRemoveProfileCredential(profile); err != nil &&
				!storageIsCredentialNotFound(err) &&
				!storageIsKeyringUnavailable(err) {
				return config.Profile{}, nil, fmt.Errorf("failed to remove old keychain credential: %w", err)
			}
		}

		if strings.TrimSpace(current.Storage) == config.StoragePath {
			removeManagedPathIfReplaced(current.ServiceAccountPath, path)
		}
		return next, warnings, nil
	case config.StorageKeychain:
		if storageShouldBypassKeychain(lookupEnv) {
			return config.Profile{}, nil, fmt.Errorf("--storage keychain cannot be used while GPC_BYPASS_KEYCHAIN is enabled")
		}
		if err := storageStoreProfileCredential(profile, serviceAccountJSON); err != nil {
			if storageIsKeyringUnavailable(err) {
				return config.Profile{}, nil, fmt.Errorf("system keychain unavailable")
			}
			return config.Profile{}, nil, fmt.Errorf("failed to store profile credential: %w", err)
		}
		next.Storage = config.StorageKeychain
		next.ServiceAccountPath = persistedServiceAccountPath(config.StorageKeychain, serviceAccountPath)

		if strings.TrimSpace(current.Storage) == config.StoragePath {
			removeManagedPathIfReplaced(current.ServiceAccountPath, next.ServiceAccountPath)
		}
		return next, warnings, nil
	default:
		return config.Profile{}, nil, fmt.Errorf("unsupported storage backend %q", storage)
	}
}

func removeManagedPathIfReplaced(previousPath, nextPath string) {
	previousPath = strings.TrimSpace(previousPath)
	nextPath = strings.TrimSpace(nextPath)
	if previousPath == "" || previousPath == nextPath {
		return
	}
	managed, err := config.IsManagedServiceAccountPath(previousPath)
	if err != nil || !managed {
		return
	}
	_ = os.Remove(previousPath)
}

func removeManagedPathIfOwned(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	managed, err := config.IsManagedServiceAccountPath(path)
	if err != nil || !managed {
		return
	}
	_ = os.Remove(path)
}
