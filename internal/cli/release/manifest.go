package release

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/leszko11/google-play-console-cli/internal/validate"
)

const (
	artifactTypeAAB       = "aab"
	artifactTypeAPK       = "apk"
	mappingTypeProguard   = "proguard"
	mappingTypeNativeCode = "nativeCode"
)

type rawFullManifest struct {
	ArtifactPath   string            `json:"artifact" yaml:"artifact"`
	Track          string            `json:"track" yaml:"track"`
	Status         string            `json:"status" yaml:"status"`
	ReleaseName    string            `json:"releaseName" yaml:"releaseName"`
	UserFraction   *float64          `json:"userFraction" yaml:"userFraction"`
	MappingFile    string            `json:"mappingFile" yaml:"mappingFile"`
	MappingType    string            `json:"mappingType" yaml:"mappingType"`
	UpdatePriority int64             `json:"updatePriority" yaml:"updatePriority"`
	ReleaseNotes   map[string]string `json:"releaseNotes" yaml:"releaseNotes"`
}

type fullManifest struct {
	ArtifactType   string
	ArtifactPath   string
	Track          string
	Status         string
	ReleaseName    string
	UserFraction   float64
	MappingFile    string
	MappingType    string
	UpdatePriority int64
	ReleaseNotes   []gpc.LocalizedText
}

func loadFullManifest(path string) (fullManifest, error) {
	var raw rawFullManifest
	if err := shared.LoadManifest(path, &raw); err != nil {
		return fullManifest{}, err
	}
	return normalizeFullManifest(raw)
}

func normalizeFullManifest(raw rawFullManifest) (fullManifest, error) {
	raw.ArtifactPath = strings.TrimSpace(raw.ArtifactPath)
	if raw.ArtifactPath == "" {
		return fullManifest{}, shared.UsageErrorf("--artifact is required in manifest")
	}
	if err := validateReadableFile(raw.ArtifactPath, "artifact"); err != nil {
		return fullManifest{}, err
	}

	artifactType, err := detectArtifactType(raw.ArtifactPath)
	if err != nil {
		return fullManifest{}, err
	}

	raw.Track = strings.TrimSpace(raw.Track)
	if raw.Track == "" {
		return fullManifest{}, shared.UsageErrorf("--track is required")
	}
	raw.Status = strings.TrimSpace(raw.Status)
	if raw.Status == "" {
		return fullManifest{}, shared.UsageErrorf("--status is required")
	}

	raw.MappingFile = strings.TrimSpace(raw.MappingFile)
	raw.MappingType = strings.TrimSpace(raw.MappingType)
	if raw.MappingFile == "" && raw.MappingType != "" {
		return fullManifest{}, shared.UsageErrorf("--mapping-type requires --mapping-file")
	}
	if raw.MappingFile != "" {
		if err := validateReadableFile(raw.MappingFile, "mapping file"); err != nil {
			return fullManifest{}, err
		}
		if raw.MappingType == "" {
			raw.MappingType = mappingTypeProguard
		}
		if raw.MappingType != mappingTypeProguard && raw.MappingType != mappingTypeNativeCode {
			return fullManifest{}, shared.UsageErrorf("--mapping-type must be one of: %s, %s", mappingTypeProguard, mappingTypeNativeCode)
		}
	}

	userFraction := -1.0
	if raw.UserFraction != nil {
		userFraction = *raw.UserFraction
		if userFraction > 1 || userFraction <= 0 {
			return fullManifest{}, shared.UsageErrorf("--user-fraction must be within (0,1] when set")
		}
	}
	if raw.UpdatePriority < 0 || raw.UpdatePriority > 5 {
		return fullManifest{}, shared.UsageErrorf("--update-priority must be between 0 and 5")
	}
	raw.ReleaseName = strings.TrimSpace(raw.ReleaseName)
	if raw.ReleaseName != "" {
		if err := validate.ReleaseName(raw.ReleaseName); err != nil {
			return fullManifest{}, shared.UsageErrorf("%s", err)
		}
	}

	notes := make([]gpc.LocalizedText, 0, len(raw.ReleaseNotes))
	for locale, text := range raw.ReleaseNotes {
		locale = strings.TrimSpace(locale)
		text = strings.TrimSpace(text)
		if locale == "" || text == "" {
			return fullManifest{}, shared.UsageErrorf("releaseNotes entries must include non-empty locale and text")
		}
		if err := validate.ReleaseNotes(text); err != nil {
			return fullManifest{}, shared.UsageErrorf("%s for locale %q", err, locale)
		}
		notes = append(notes, gpc.LocalizedText{Language: locale, Text: text})
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].Language < notes[j].Language })

	return fullManifest{
		ArtifactType:   artifactType,
		ArtifactPath:   raw.ArtifactPath,
		Track:          raw.Track,
		Status:         raw.Status,
		ReleaseName:    raw.ReleaseName,
		UserFraction:   userFraction,
		MappingFile:    raw.MappingFile,
		MappingType:    raw.MappingType,
		UpdatePriority: raw.UpdatePriority,
		ReleaseNotes:   notes,
	}, nil
}

func detectArtifactType(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".aab":
		return artifactTypeAAB, nil
	case ".apk":
		return artifactTypeAPK, nil
	default:
		return "", shared.UsageErrorf("artifact must end with .aab or .apk")
	}
}

func validateReadableFile(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return shared.UsageErrorf("%s does not exist: %s", label, path)
		}
		return err
	}
	if info.IsDir() {
		return shared.UsageErrorf("%s must be a file, got directory: %s", label, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return shared.UsageErrorf("%s is not readable: %v", label, err)
	}
	return file.Close()
}
