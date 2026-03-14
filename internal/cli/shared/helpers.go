package shared

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

var resolveOutputLookupEnv = os.Getenv
var resolveOutputDetectTTY = detectStdoutTTY
var resolvePackageLookupEnv = os.Getenv
var resolveProfileLookupEnv = os.Getenv
var resolveCredentialsShouldBypassKeychain = authresolver.ShouldBypassKeychain
var resolveCredentialsLoadProfileCredential = authresolver.LoadProfileCredential
var resolveCredentialsIsCredentialNotFound = authresolver.IsCredentialNotFound
var resolveCredentialsIsKeyringUnavailable = authresolver.IsKeyringUnavailable
var resolveCredentialsResolveSource = authresolver.ResolveCredentialSource

type ResolvedCredentials struct {
	Input              gpc.CredentialInput
	Source             authresolver.SourceKind
	Profile            string
	ServiceAccountPath string
	Warnings           []string
}

func ResolvePackageName(localValue string) (string, error) {
	if pkg := strings.TrimSpace(localValue); pkg != "" {
		return pkg, nil
	}
	if pkg := strings.TrimSpace(ActiveGlobalFlags().PackageName); pkg != "" {
		return pkg, nil
	}
	if pkg := strings.TrimSpace(resolvePackageLookupEnv(EnvPackageName)); pkg != "" {
		return pkg, nil
	}
	project, err := config.LoadProject()
	if err != nil {
		return "", err
	}
	if pkg := strings.TrimSpace(project.Config.PackageName); pkg != "" {
		return pkg, nil
	}
	return "", UsageErrorf("--package-name is required")
}

func NormalizeDeveloperID(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	normalized := strings.TrimPrefix(trimmed, "developers/")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return "", UsageErrorf("developer id must not be empty")
	}
	for _, r := range normalized {
		if r < '0' || r > '9' {
			return "", UsageErrorf("developer id must be numeric or developers/<id>")
		}
	}
	return normalized, nil
}

func ResolveDeveloperID(localValue string, cfg config.Config) (string, error) {
	if id := strings.TrimSpace(localValue); id != "" {
		return NormalizeDeveloperID(id)
	}
	resolvedProfile := ResolveProfileName(cfg)
	if resolvedProfile != "" && cfg.Profiles != nil {
		if profile, ok := cfg.Profiles[resolvedProfile]; ok {
			if id := strings.TrimSpace(profile.DeveloperID); id != "" {
				return NormalizeDeveloperID(id)
			}
		}
	}
	return "", UsageErrorf("--developer-id is required (or run `gpc auth init --developer-id <id>` once)")
}

func ResolveOutput(localValue string) string {
	if out := strings.TrimSpace(localValue); out != "" {
		return strings.ToLower(out)
	}
	if out := strings.TrimSpace(ActiveGlobalFlags().Output); out != "" {
		return strings.ToLower(out)
	}
	if out := strings.TrimSpace(resolveOutputLookupEnv(EnvDefaultOutput)); out != "" {
		return strings.ToLower(out)
	}
	project, err := config.LoadProject()
	if err == nil {
		if out := strings.TrimSpace(project.Config.Output); out != "" {
			return strings.ToLower(out)
		}
	}
	if resolveOutputDetectTTY() {
		return "table"
	}
	return "json"
}

func ResolveProfileName(cfg config.Config) string {
	if profile := strings.TrimSpace(ActiveGlobalFlags().Profile); profile != "" {
		return profile
	}
	if profile := strings.TrimSpace(resolveProfileLookupEnv(EnvProfile)); profile != "" {
		return profile
	}
	project, err := config.LoadProject()
	if err == nil {
		if profile := strings.TrimSpace(project.Config.Profile); profile != "" {
			return profile
		}
	}
	return strings.TrimSpace(cfg.ActiveProfile)
}

func ResolveStrictAuth(lookupEnv func(string) string) bool {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}
	if ActiveGlobalFlags().StrictAuth {
		return true
	}
	return isTruthy(lookupEnv(EnvStrictAuth))
}

func ResolveCredentials(cfg config.Config, lookupEnv func(string) string) (ResolvedCredentials, error) {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}

	profileName := ResolveProfileName(cfg)
	var cfgPath string
	if profileName != "" && cfg.Profiles != nil {
		if profile, ok := cfg.Profiles[profileName]; ok {
			cfgPath = strings.TrimSpace(profile.ServiceAccountPath)
		}
	}

	warnings := []string{}
	var keychainJSON []byte
	if profileName != "" {
		if resolveCredentialsShouldBypassKeychain(lookupEnv) {
			warnings = append(warnings, "keychain bypassed via GPC_BYPASS_KEYCHAIN")
		} else {
			payload, err := resolveCredentialsLoadProfileCredential(profileName)
			if err == nil {
				keychainJSON = payload
			} else if resolveCredentialsIsCredentialNotFound(err) {
				// Profile may still have config path fallback.
			} else if resolveCredentialsIsKeyringUnavailable(err) {
				warnings = append(warnings, "system keychain unavailable; using config/environment/flags")
			} else {
				return ResolvedCredentials{}, err
			}
		}
	}

	source, err := resolveCredentialsResolveSource(authresolver.Input{
		FlagPath:     strings.TrimSpace(ActiveGlobalFlags().ServiceAccount),
		EnvPath:      strings.TrimSpace(lookupEnv(EnvServiceAccountPath)),
		ConfigPath:   cfgPath,
		KeychainJSON: keychainJSON,
		Strict:       ResolveStrictAuth(lookupEnv),
	})
	if err != nil {
		if errors.Is(err, authresolver.ErrNoCredentialSources) {
			return ResolvedCredentials{}, fmt.Errorf("%w: no service account configured", gpc.ErrInvalidCredentials)
		}
		if errors.Is(err, authresolver.ErrMultipleSources) {
			return ResolvedCredentials{}, UsageErrorf("%v", err)
		}
		return ResolvedCredentials{}, err
	}

	input := gpc.CredentialInput{
		ServiceAccountPath: source.Path,
		ServiceAccountJSON: source.JSON,
	}
	return ResolvedCredentials{
		Input:              input,
		Source:             source.Kind,
		Profile:            profileName,
		ServiceAccountPath: source.Path,
		Warnings:           warnings,
	}, nil
}

func ResolveServiceAccountPath(cfg config.Config, lookupEnv func(string) string) (string, error) {
	resolved, err := ResolveCredentials(cfg, lookupEnv)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(resolved.ServiceAccountPath) == "" {
		return "", UsageErrorf("resolved credentials from %q do not expose --service-account path", resolved.Source)
	}
	return resolved.ServiceAccountPath, nil
}

func WriteJSON(out io.Writer, v any) error {
	if out == nil {
		out = os.Stdout
	}
	if fields := strings.TrimSpace(ActiveGlobalFlags().Fields); fields != "" {
		projected, err := ProjectFields(v, fields)
		if err != nil {
			return UsageErrorf("%v", err)
		}
		v = projected
	}
	b, err := RenderJSON(v, ActiveGlobalFlags().Pretty)
	if err != nil {
		return err
	}
	_, err = out.Write(b)
	return err
}

func WriteDelimited(out io.Writer, format string, header []string, rows [][]string) error {
	if out == nil {
		out = os.Stdout
	}

	writer := csv.NewWriter(out)
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv":
		writer.Comma = ','
	case "tsv":
		writer.Comma = '\t'
	default:
		return UsageErrorf("unsupported delimited output format %q", format)
	}

	if err := writer.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

type BuildClientDeps[T any] struct {
	LoadConfig func() (config.Config, error)
	LookupEnv  func(string) string
	NewClient  func(context.Context, gpc.CredentialInput) (T, error)
	Upload     bool
}

func BuildClient[T any](ctx context.Context, deps BuildClientDeps[T]) (T, context.Context, context.CancelFunc, error) {
	var zero T
	noopCancel := func() {}

	requestCtx, cancel := ContextWithTimeout(ctx, ActiveGlobalFlags().Timeout)
	if deps.Upload {
		requestCtx, cancel = ContextWithUploadTimeout(ctx, ActiveGlobalFlags().UploadTimeout)
	}

	cfg, err := deps.LoadConfig()
	if err != nil {
		cancel()
		return zero, nil, noopCancel, err
	}

	resolved, err := ResolveCredentials(cfg, deps.LookupEnv)
	if err != nil {
		cancel()
		return zero, nil, noopCancel, err
	}

	client, err := deps.NewClient(requestCtx, resolved.Input)
	if err != nil {
		cancel()
		return zero, nil, noopCancel, err
	}

	return client, requestCtx, cancel, nil
}

func detectStdoutTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func CredentialLocallyValid(input gpc.CredentialInput) bool {
	if len(input.ServiceAccountJSON) > 0 {
		return json.Valid(input.ServiceAccountJSON)
	}
	path := strings.TrimSpace(input.ServiceAccountPath)
	if path == "" {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return json.Valid(raw)
}

func isTruthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
