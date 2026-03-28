package shared

import (
	"fmt"
	"strings"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/config"
)

var authExplainConfigPath = config.Path

type AuthExplainSource struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Selected  bool   `json:"selected,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type AuthExplainEnv struct {
	ServiceAccountPath bool `json:"serviceAccountPath"`
	Profile            bool `json:"profile"`
	StrictAuth         bool `json:"strictAuth"`
	BypassKeychain     bool `json:"bypassKeychain"`
}

type AuthExplainKeychain struct {
	Bypassed  bool   `json:"bypassed"`
	Available bool   `json:"available"`
	Blocked   bool   `json:"blocked,omitempty"`
	Error     string `json:"error,omitempty"`
}

type AuthExplainStrict struct {
	Enabled             bool `json:"enabled"`
	WouldFail           bool `json:"wouldFail"`
	DetectedSourceCount int  `json:"detectedSourceCount"`
}

type AuthExplainSnapshot struct {
	Status           AuthStatusSnapshot  `json:"status"`
	ConfigPath       string              `json:"configPath,omitempty"`
	SelectedProfile  string              `json:"selectedProfile,omitempty"`
	PersistedStorage string              `json:"persistedStorage,omitempty"`
	Env              AuthExplainEnv      `json:"env"`
	Keychain         AuthExplainKeychain `json:"keychain"`
	Sources          []AuthExplainSource `json:"sources"`
	FinalSource      string              `json:"finalSource,omitempty"`
	MixedSourceRisk  bool                `json:"mixedSourceRisk"`
	StrictAuth       AuthExplainStrict   `json:"strictAuth"`
	CIRecommendation string              `json:"ciRecommendation,omitempty"`
}

func BuildAuthExplainSnapshot(cfg config.Config, lookupEnv func(string) string) AuthExplainSnapshot {
	if lookupEnv == nil {
		lookupEnv = func(string) string { return "" }
	}

	status := BuildAuthStatusSnapshot(cfg, lookupEnv)
	selectedProfile := ResolveProfileName(cfg)
	configPath, _ := authExplainConfigPath()
	strictEnabled := ResolveStrictAuth(lookupEnv)
	bypassKeychain := resolveCredentialsShouldBypassKeychain(lookupEnv)
	probe := resolveCredentialsProbeKeychainAccess(lookupEnv)

	var persistedStorage string
	var profilePath string
	if selectedProfile != "" && cfg.Profiles != nil {
		if profile, ok := cfg.Profiles[selectedProfile]; ok {
			persistedStorage = strings.TrimSpace(profile.Storage)
			profilePath = strings.TrimSpace(profile.ServiceAccountPath)
		}
	}

	flagPath := strings.TrimSpace(ActiveGlobalFlags().ServiceAccount)
	envPath := strings.TrimSpace(lookupEnv(EnvServiceAccountPath))

	sources := make([]AuthExplainSource, 0, 4)
	detectedSourceCount := 0

	flagAvailable := flagPath != ""
	if flagAvailable {
		detectedSourceCount++
	}
	sources = append(sources, AuthExplainSource{
		Name:      string(authresolver.SourceFlag),
		Available: flagAvailable,
		Selected:  status.Source == string(authresolver.SourceFlag),
		Detail:    flagPath,
	})

	envAvailable := envPath != ""
	if envAvailable {
		detectedSourceCount++
	}
	sources = append(sources, AuthExplainSource{
		Name:      string(authresolver.SourceEnv),
		Available: envAvailable,
		Selected:  status.Source == string(authresolver.SourceEnv),
		Detail:    envPath,
	})

	keychainAvailable := false
	keychainDetail := ""
	switch {
	case selectedProfile == "":
		keychainDetail = "no profile selected"
	case bypassKeychain:
		keychainDetail = "bypassed via GPC_BYPASS_KEYCHAIN"
	case probe.Blocked:
		keychainDetail = "blocked by system keychain access"
	case probe.Err != nil:
		keychainDetail = probe.Err.Error()
	case !probe.Available:
		keychainDetail = "unavailable"
	default:
		keychainAvailable = persistedStorage == config.StorageKeychain || persistedStorage == "" || status.Source == string(authresolver.SourceKeychain)
		if keychainAvailable {
			detectedSourceCount++
			keychainDetail = "eligible for resolution"
		}
	}
	sources = append(sources, AuthExplainSource{
		Name:      string(authresolver.SourceKeychain),
		Available: keychainAvailable,
		Selected:  status.Source == string(authresolver.SourceKeychain),
		Detail:    keychainDetail,
	})

	configAvailable := profilePath != ""
	if configAvailable {
		detectedSourceCount++
	}
	sources = append(sources, AuthExplainSource{
		Name:      string(authresolver.SourceConfig),
		Available: configAvailable,
		Selected:  status.Source == string(authresolver.SourceConfig),
		Detail:    profilePath,
	})

	mixedSourceRisk := status.Health == string(AuthHealthMixedSources) || detectedSourceCount > 1
	if persistedStorage == config.StoragePath {
		mixedSourceRisk = flagAvailable || envAvailable
	}
	if persistedStorage == config.StorageKeychain {
		mixedSourceRisk = flagAvailable || envAvailable || (!bypassKeychain && profilePath != "")
	}

	return AuthExplainSnapshot{
		Status:           status,
		ConfigPath:       configPath,
		SelectedProfile:  selectedProfile,
		PersistedStorage: persistedStorage,
		Env: AuthExplainEnv{
			ServiceAccountPath: envAvailable,
			Profile:            strings.TrimSpace(lookupEnv(EnvProfile)) != "",
			StrictAuth:         isTruthy(lookupEnv(EnvStrictAuth)),
			BypassKeychain:     bypassKeychain,
		},
		Keychain: AuthExplainKeychain{
			Bypassed:  bypassKeychain,
			Available: probe.Available && !probe.Blocked && probe.Err == nil,
			Blocked:   probe.Blocked,
			Error:     errorString(probe.Err),
		},
		Sources:         sources,
		FinalSource:     status.Source,
		MixedSourceRisk: mixedSourceRisk,
		StrictAuth: AuthExplainStrict{
			Enabled:             strictEnabled,
			WouldFail:           strictEnabled && detectedSourceCount > 1,
			DetectedSourceCount: detectedSourceCount,
		},
		CIRecommendation: buildAuthExplainCIRecommendation(selectedProfile, status),
	}
}

func buildAuthExplainCIRecommendation(profile string, status AuthStatusSnapshot) string {
	parts := []string{"GPC_BYPASS_KEYCHAIN=1"}
	if profile = strings.TrimSpace(profile); profile != "" {
		parts = append(parts, "GPC_PROFILE="+shellQuote(profile))
	}
	serviceAccountPath := strings.TrimSpace(status.ServiceAccountPath)
	if serviceAccountPath == "" {
		serviceAccountPath = "/path/to/service-account.json"
	}
	parts = append(parts, "GPC_SERVICE_ACCOUNT_PATH="+shellQuote(serviceAccountPath))
	parts = append(parts, "GPC_STRICT_AUTH=1")
	parts = append(parts, "gpc auth status --output json")
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return `""`
	}
	escaped := strings.ReplaceAll(value, `"`, `\"`)
	return fmt.Sprintf(`"%s"`, escaped)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
