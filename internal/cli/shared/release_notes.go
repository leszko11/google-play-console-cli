package shared

import (
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func ParseReleaseNotesFile(path string) ([]gpc.LocalizedText, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, UsageErrorf("failed to read --release-notes-file: %v", err)
	}

	var mapPayload map[string]string
	if err := json.Unmarshal(raw, &mapPayload); err == nil && mapPayload != nil {
		if len(mapPayload) == 0 {
			return nil, UsageErrorf("--release-notes-file must contain at least one locale entry")
		}
		notes := make([]gpc.LocalizedText, 0, len(mapPayload))
		for locale, text := range mapPayload {
			notes = append(notes, gpc.LocalizedText{Language: locale, Text: text})
		}
		return normalizeReleaseNotes(notes)
	}

	var arrayPayload []gpc.LocalizedText
	if err := json.Unmarshal(raw, &arrayPayload); err == nil && arrayPayload != nil {
		if len(arrayPayload) == 0 {
			return nil, UsageErrorf("--release-notes-file must contain at least one locale entry")
		}
		return normalizeReleaseNotes(arrayPayload)
	}

	return nil, UsageErrorf("--release-notes-file must be either a JSON object or array")
}

func normalizeReleaseNotes(notes []gpc.LocalizedText) ([]gpc.LocalizedText, error) {
	normalized := make([]gpc.LocalizedText, 0, len(notes))
	seenLocales := map[string]struct{}{}

	for _, note := range notes {
		locale := strings.TrimSpace(note.Language)
		if locale == "" {
			return nil, UsageErrorf("release note locale must not be empty")
		}
		text := strings.TrimSpace(note.Text)
		if text == "" {
			return nil, UsageErrorf("release note text must not be empty for locale %q", locale)
		}
		if _, ok := seenLocales[locale]; ok {
			return nil, UsageErrorf("duplicate release note locale %q", locale)
		}
		seenLocales[locale] = struct{}{}
		normalized = append(normalized, gpc.LocalizedText{
			Language: locale,
			Text:     text,
		})
	}

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Language < normalized[j].Language
	})

	return normalized, nil
}
