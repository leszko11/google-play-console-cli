package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/99designs/keyring"
)

const (
	EnvBypassKeychain = "GPC_BYPASS_KEYCHAIN"
	keyringService    = "gpc"
	keyringItemPrefix = "gpc:credential:"
)

var ErrCredentialNotFound = errors.New("credential not found")

var keyringOpener = func() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName: keyringService,
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.WinCredBackend,
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			keyring.KeyCtlBackend,
		},
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         false,
		KeychainAccessibleWhenUnlocked: true,
	})
}

func ShouldBypassKeychain(lookupEnv func(string) string) bool {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}
	return truthy(lookupEnv(EnvBypassKeychain))
}

func KeychainAvailable(lookupEnv func(string) string) (bool, error) {
	if ShouldBypassKeychain(lookupEnv) {
		return false, nil
	}
	_, err := keyringOpener()
	if err == nil {
		return true, nil
	}
	if IsKeyringUnavailable(err) {
		return false, nil
	}
	return false, err
}

func StoreProfileCredential(profile string, serviceAccountJSON []byte) error {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return fmt.Errorf("profile name is required")
	}
	payload := strings.TrimSpace(string(serviceAccountJSON))
	if payload == "" {
		return fmt.Errorf("service account json is required")
	}
	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("service account json is invalid")
	}

	kr, err := keyringOpener()
	if err != nil {
		return err
	}
	return kr.Set(keyring.Item{
		Key:  keyringKey(profile),
		Data: []byte(payload),
	})
}

func LoadProfileCredential(profile string) ([]byte, error) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return nil, fmt.Errorf("profile name is required")
	}

	kr, err := keyringOpener()
	if err != nil {
		return nil, err
	}
	item, err := kr.Get(keyringKey(profile))
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}
	data := strings.TrimSpace(string(item.Data))
	if data == "" {
		return nil, ErrCredentialNotFound
	}
	if !json.Valid([]byte(data)) {
		return nil, fmt.Errorf("invalid keychain payload for profile %q", profile)
	}
	return []byte(data), nil
}

func RemoveProfileCredential(profile string) error {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return fmt.Errorf("profile name is required")
	}

	kr, err := keyringOpener()
	if err != nil {
		return err
	}
	if err := kr.Remove(keyringKey(profile)); err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return ErrCredentialNotFound
		}
		return err
	}
	return nil
}

func RemoveAllProfileCredentials() error {
	kr, err := keyringOpener()
	if err != nil {
		return err
	}
	keys, err := kr.Keys()
	if err != nil {
		return err
	}
	for _, key := range keys {
		if !strings.HasPrefix(key, keyringItemPrefix) {
			continue
		}
		if err := kr.Remove(key); err != nil {
			return err
		}
	}
	return nil
}

func ListKeychainProfiles() ([]string, error) {
	kr, err := keyringOpener()
	if err != nil {
		return nil, err
	}
	keys, err := kr.Keys()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	profiles := make([]string, 0, len(keys))
	for _, key := range keys {
		if !strings.HasPrefix(key, keyringItemPrefix) {
			continue
		}
		profile := strings.TrimSpace(strings.TrimPrefix(key, keyringItemPrefix))
		if profile == "" {
			continue
		}
		if _, ok := seen[profile]; ok {
			continue
		}
		seen[profile] = struct{}{}
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)
	return profiles, nil
}

func IsCredentialNotFound(err error) bool {
	return errors.Is(err, ErrCredentialNotFound)
}

func IsKeyringUnavailable(err error) bool {
	return errors.Is(err, keyring.ErrNoAvailImpl)
}

func keyringKey(profile string) string {
	return keyringItemPrefix + strings.TrimSpace(profile)
}

func truthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
