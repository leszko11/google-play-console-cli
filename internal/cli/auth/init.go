package auth

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewInitCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var (
		serviceAccountPath string
		profile            string
		packageName        string
		developerID        string
	)

	fs.StringVar(&serviceAccountPath, "service-account", "", "Path to service account JSON")
	fs.StringVar(&profile, "profile", "default", "Auth profile name")
	fs.StringVar(&packageName, "package-name", "", "Verify package access for this package")
	fs.StringVar(&developerID, "developer-id", "", "Optional developer account ID (numeric or developers/<id>)")

	return &ffcli.Command{
		Name:      "init",
		ShortHelp: "Initialize authentication profile",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if strings.TrimSpace(serviceAccountPath) == "" {
				serviceAccountPath = strings.TrimSpace(shared.ActiveGlobalFlags().ServiceAccount)
			}
			if strings.TrimSpace(serviceAccountPath) == "" {
				serviceAccountPath = strings.TrimSpace(deps.LookupEnv(shared.EnvServiceAccountPath))
			}
			if strings.TrimSpace(serviceAccountPath) == "" {
				return fmt.Errorf("--service-account is required or set %s", shared.EnvServiceAccountPath)
			}

			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			if cfg.Profiles == nil {
				cfg.Profiles = make(map[string]config.Profile)
			}
			current := cfg.Profiles[profile]

			resolvedDeveloperID, err := resolveDeveloperIDForProfile(deps, developerID, current.DeveloperID)
			if err != nil {
				return err
			}

			if strings.TrimSpace(packageName) != "" {
				requestCtx, cancel := shared.ContextWithTimeout(ctx, shared.ActiveGlobalFlags().Timeout)
				defer cancel()

				client, err := deps.NewClient(requestCtx, gpc.CredentialInput{ServiceAccountPath: serviceAccountPath})
				if err != nil {
					return err
				}
				if err := client.VerifyPackageAccess(requestCtx, packageName); err != nil {
					return fmt.Errorf("failed to verify package access: %w", err)
				}
			}

			cfg.Profiles[profile] = config.Profile{
				ServiceAccountPath: serviceAccountPath,
				LastValidatedAt:    deps.Now().UTC().Format(time.RFC3339),
				DeveloperID:        resolvedDeveloperID,
			}
			cfg.ActiveProfile = profile

			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}

			out := map[string]any{
				"activeProfile":      cfg.ActiveProfile,
				"serviceAccountPath": serviceAccountPath,
			}
			if resolvedDeveloperID != "" {
				out["developerId"] = resolvedDeveloperID
			}
			return shared.WriteJSON(deps.Stdout, out)
		},
	}
}

func resolveDeveloperIDForProfile(deps Deps, provided, existing string) (string, error) {
	if normalized, err := shared.NormalizeDeveloperID(provided); err != nil {
		return "", err
	} else if normalized != "" {
		return normalized, nil
	}
	if normalized, err := shared.NormalizeDeveloperID(existing); err != nil {
		return "", err
	} else if normalized != "" {
		return normalized, nil
	}

	prompted, err := deps.PromptID(deps.Stdin, deps.Stderr)
	if err != nil {
		return "", err
	}
	return shared.NormalizeDeveloperID(prompted)
}

func promptDeveloperID(stdin io.Reader, stderr io.Writer) (string, error) {
	if !isInteractive(stdin) {
		return "", nil
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	if _, err := fmt.Fprintln(stderr, "Optional: provide a Google Play developer ID to save as default for developer-level commands."); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(stderr, "Find it in Play Console URL: https://play.google.com/console/u/0/developers/<developer-id>/..."); err != nil {
		return "", err
	}
	if _, err := fmt.Fprint(stderr, "Developer ID (press Enter to skip): "); err != nil {
		return "", err
	}

	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to read developer id: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func isInteractive(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
