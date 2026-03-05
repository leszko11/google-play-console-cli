package auth

import (
	"errors"
	"fmt"
)

var (
	ErrNoCredentialSources = errors.New("no credential source found")
	ErrMultipleSources     = errors.New("multiple credential sources found")
)

type SourceKind string

const (
	SourceFlag     SourceKind = "flag"
	SourceEnv      SourceKind = "env"
	SourceConfig   SourceKind = "config"
	SourceKeychain SourceKind = "keychain"
)

type Source struct {
	Kind SourceKind
	Path string
	JSON []byte
}

type Input struct {
	FlagPath     string
	EnvPath      string
	ConfigPath   string
	KeychainJSON []byte
	Strict       bool
}

func ResolveCredentialSource(in Input) (Source, error) {
	normalizedKeychain := []byte{}
	if len(in.KeychainJSON) > 0 {
		normalizedKeychain = []byte(string(in.KeychainJSON))
	}

	sources := make([]Source, 0, 3)
	if in.FlagPath != "" {
		sources = append(sources, Source{Kind: SourceFlag, Path: in.FlagPath})
	}
	if in.EnvPath != "" {
		sources = append(sources, Source{Kind: SourceEnv, Path: in.EnvPath})
	}
	if len(normalizedKeychain) > 0 {
		sources = append(sources, Source{Kind: SourceKeychain, JSON: normalizedKeychain})
	} else if in.ConfigPath != "" {
		sources = append(sources, Source{Kind: SourceConfig, Path: in.ConfigPath})
	}

	if len(sources) == 0 {
		return Source{}, ErrNoCredentialSources
	}

	if in.Strict && len(sources) > 1 {
		return Source{}, fmt.Errorf("%w: found %d", ErrMultipleSources, len(sources))
	}

	// Precedence: flag > env > keychain > config
	if in.FlagPath != "" {
		return Source{Kind: SourceFlag, Path: in.FlagPath}, nil
	}
	if in.EnvPath != "" {
		return Source{Kind: SourceEnv, Path: in.EnvPath}, nil
	}
	if len(normalizedKeychain) > 0 {
		return Source{Kind: SourceKeychain, JSON: normalizedKeychain}, nil
	}
	return Source{Kind: SourceConfig, Path: in.ConfigPath}, nil
}
