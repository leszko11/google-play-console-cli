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
	SourceFlag   SourceKind = "flag"
	SourceEnv    SourceKind = "env"
	SourceConfig SourceKind = "config"
)

type Source struct {
	Kind SourceKind
	Path string
}

type Input struct {
	FlagPath   string
	EnvPath    string
	ConfigPath string
	Strict     bool
}

func ResolveCredentialSource(in Input) (Source, error) {
	sources := make([]Source, 0, 3)
	if in.FlagPath != "" {
		sources = append(sources, Source{Kind: SourceFlag, Path: in.FlagPath})
	}
	if in.EnvPath != "" {
		sources = append(sources, Source{Kind: SourceEnv, Path: in.EnvPath})
	}
	if in.ConfigPath != "" {
		sources = append(sources, Source{Kind: SourceConfig, Path: in.ConfigPath})
	}

	if len(sources) == 0 {
		return Source{}, ErrNoCredentialSources
	}

	if in.Strict && len(sources) > 1 {
		return Source{}, fmt.Errorf("%w: found %d", ErrMultipleSources, len(sources))
	}

	// Precedence: flag > env > config
	if in.FlagPath != "" {
		return Source{Kind: SourceFlag, Path: in.FlagPath}, nil
	}
	if in.EnvPath != "" {
		return Source{Kind: SourceEnv, Path: in.EnvPath}, nil
	}
	return Source{Kind: SourceConfig, Path: in.ConfigPath}, nil
}
