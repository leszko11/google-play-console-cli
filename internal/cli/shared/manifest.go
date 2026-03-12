package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadManifest(path string, out any) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return UsageErrorf("manifest path is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return UsageErrorf("failed to read manifest: %v", err)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("parse JSON manifest: %w", err)
		}
		return nil
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("parse YAML manifest: %w", err)
		}
		return nil
	default:
		return UsageErrorf("manifest file must end with .json, .yaml, or .yml")
	}
}
