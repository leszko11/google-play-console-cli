package shared

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

var resolveOutputLookupEnv = os.Getenv
var resolveOutputDetectTTY = detectStdoutTTY

func ResolvePackageName(localValue string) (string, error) {
	if pkg := strings.TrimSpace(localValue); pkg != "" {
		return pkg, nil
	}
	if pkg := strings.TrimSpace(ActiveGlobalFlags().PackageName); pkg != "" {
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
	if cfg.ActiveProfile != "" && cfg.Profiles != nil {
		if profile, ok := cfg.Profiles[cfg.ActiveProfile]; ok {
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
	if resolveOutputDetectTTY() {
		return "table"
	}
	return "json"
}

func ResolveServiceAccountPath(cfg config.Config, lookupEnv func(string) string) (string, error) {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}

	var cfgPath string
	if cfg.ActiveProfile != "" && cfg.Profiles != nil {
		if profile, ok := cfg.Profiles[cfg.ActiveProfile]; ok {
			cfgPath = strings.TrimSpace(profile.ServiceAccountPath)
		}
	}

	source, err := authresolver.ResolveCredentialSource(authresolver.Input{
		FlagPath:   strings.TrimSpace(ActiveGlobalFlags().ServiceAccount),
		EnvPath:    strings.TrimSpace(lookupEnv(EnvServiceAccountPath)),
		ConfigPath: cfgPath,
	})
	if err != nil {
		if errors.Is(err, authresolver.ErrNoCredentialSources) {
			return "", UsageErrorf("no service account configured")
		}
		return "", err
	}
	return source.Path, nil
}

func WriteJSON(out io.Writer, v any) error {
	if out == nil {
		out = os.Stdout
	}
	b, err := RenderJSON(v, ActiveGlobalFlags().Pretty)
	if err != nil {
		return err
	}
	_, err = out.Write(b)
	return err
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

	serviceAccountPath, err := ResolveServiceAccountPath(cfg, deps.LookupEnv)
	if err != nil {
		cancel()
		return zero, nil, noopCancel, err
	}

	client, err := deps.NewClient(requestCtx, gpc.CredentialInput{ServiceAccountPath: serviceAccountPath})
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
