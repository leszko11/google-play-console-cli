package shared

import (
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/config"
)

func ResolveProjectPath(localValue string, selectPath func(config.ProjectConfig) string) (string, error) {
	if value := strings.TrimSpace(localValue); value != "" {
		return value, nil
	}
	project, err := config.LoadProject()
	if err != nil {
		return "", err
	}
	if value := strings.TrimSpace(selectPath(project.Config)); value != "" {
		return value, nil
	}
	return "", nil
}

func ResolveDefaultTrack(localValue string) (string, error) {
	if value := strings.TrimSpace(localValue); value != "" {
		return value, nil
	}
	project, err := config.LoadProject()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(project.Config.DefaultTrack), nil
}

func ResolveDefaultLocale(localValue string) (string, error) {
	if value := strings.TrimSpace(localValue); value != "" {
		return value, nil
	}
	project, err := config.LoadProject()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(project.Config.DefaultLocale), nil
}
