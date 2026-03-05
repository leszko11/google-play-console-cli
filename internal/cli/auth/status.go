package auth

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	authresolver "github.com/leszko11/google-play-console-cli/internal/auth"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func NewStatusCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)
	var output string
	fs.StringVar(&output, "output", "", "Output format: json, table")

	return &ffcli.Command{
		Name:      "status",
		ShortHelp: "Show authentication status",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}

			status := buildStatus(cfg, deps.LookupEnv)
			resolvedOutput := shared.ResolveOutput(output)
			switch resolvedOutput {
			case "json":
				return shared.WriteJSON(deps.Stdout, status)
			case "table":
				return writeStatusTable(deps.Stdout, status)
			default:
				return shared.UsageErrorf("unsupported output format %q", strings.TrimSpace(resolvedOutput))
			}
		},
	}
}

type statusPayload struct {
	ActiveProfile      string `json:"activeProfile,omitempty"`
	SelectedProfile    string `json:"selectedProfile,omitempty"`
	Authenticated      bool   `json:"authenticated"`
	Source             string `json:"source,omitempty"`
	StorageBackend     string `json:"storageBackend,omitempty"`
	ServiceAccountPath string `json:"serviceAccountPath,omitempty"`
	LastValidatedAt    string `json:"lastValidatedAt,omitempty"`
	DeveloperID        string `json:"developerId,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
}

func buildStatus(cfg config.Config, lookupEnv func(string) string) statusPayload {
	if lookupEnv == nil {
		lookupEnv = func(string) string { return "" }
	}

	selectedProfile := shared.ResolveProfileName(cfg)
	out := statusPayload{
		ActiveProfile: cfg.ActiveProfile,
		SelectedProfile: selectedProfile,
		Authenticated: false,
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

	resolved, err := shared.ResolveCredentials(cfg, lookupEnv)
	if err != nil {
		if !shared.IsUsageError(err) {
			out.Warnings = append(out.Warnings, err.Error())
		}
		return out
	}
	out.Source = string(resolved.Source)
	out.ServiceAccountPath = resolved.ServiceAccountPath
	out.Warnings = append(out.Warnings, resolved.Warnings...)
	if shared.CredentialLocallyValid(resolved.Input) {
		out.Authenticated = true
	}
	return out
}

func writeStatusTable(out io.Writer, status statusPayload) error {
	if _, err := fmt.Fprintln(out, "FIELD\tVALUE"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "authenticated\t%t\n", status.Authenticated); err != nil {
		return err
	}
	if status.ActiveProfile != "" {
		if _, err := fmt.Fprintf(out, "activeProfile\t%s\n", status.ActiveProfile); err != nil {
			return err
		}
	}
	if status.SelectedProfile != "" {
		if _, err := fmt.Fprintf(out, "selectedProfile\t%s\n", status.SelectedProfile); err != nil {
			return err
		}
	}
	if status.Source != "" {
		if _, err := fmt.Fprintf(out, "source\t%s\n", status.Source); err != nil {
			return err
		}
	}
	if status.StorageBackend != "" {
		if _, err := fmt.Fprintf(out, "storageBackend\t%s\n", status.StorageBackend); err != nil {
			return err
		}
	}
	if status.ServiceAccountPath != "" {
		if _, err := fmt.Fprintf(out, "serviceAccountPath\t%s\n", status.ServiceAccountPath); err != nil {
			return err
		}
	}
	if status.LastValidatedAt != "" {
		if _, err := fmt.Fprintf(out, "lastValidatedAt\t%s\n", status.LastValidatedAt); err != nil {
			return err
		}
	}
	if status.DeveloperID != "" {
		if _, err := fmt.Fprintf(out, "developerId\t%s\n", status.DeveloperID); err != nil {
			return err
		}
	}
	for _, warning := range status.Warnings {
		if _, err := fmt.Fprintf(out, "warning\t%s\n", warning); err != nil {
			return err
		}
	}
	return nil
}
