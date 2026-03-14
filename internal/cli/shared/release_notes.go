package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
	notes "github.com/leszko11/google-play-console-cli/internal/release/notes"
	"github.com/leszko11/google-play-console-cli/internal/validate"
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

	return parseJSONReleaseNotesRaw(raw)
}

func ParseReleaseNotesInput(filePath, inlineText, defaultLocale string, readFile func(string) ([]byte, error)) ([]gpc.LocalizedText, error) {
	filePath = strings.TrimSpace(filePath)
	inlineText = strings.TrimSpace(inlineText)
	defaultLocale = strings.TrimSpace(defaultLocale)
	if defaultLocale == "" {
		defaultLocale = notes.DefaultLocale
	}
	if filePath != "" && inlineText != "" {
		return nil, UsageErrorf("only one of --release-notes-file or --release-notes-text can be set")
	}
	if filePath == "" && inlineText == "" {
		return nil, nil
	}

	if readFile == nil {
		readFile = os.ReadFile
	}

	if filePath != "" {
		raw, err := readFile(filePath)
		if err != nil {
			return nil, UsageErrorf("failed to read --release-notes-file: %v", err)
		}

		trimmed := strings.TrimSpace(string(raw))
		if trimmed == "" {
			return nil, UsageErrorf("notes file is empty: %s", filePath)
		}

		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			return parseJSONReleaseNotesRaw([]byte(trimmed))
		}

		parsedNotes, err := notes.ParseLocalizedText(trimmed, defaultLocale)
		if err != nil {
			return nil, UsageErrorf("%s", err)
		}
		return toLocalizedText(parsedNotes), nil
	}

	parsedNotes, err := notes.ParseLocalizedText(inlineText, defaultLocale)
	if err != nil {
		return nil, UsageErrorf("%s", err)
	}
	return toLocalizedText(parsedNotes), nil
}

func parseJSONReleaseNotesRaw(raw []byte) ([]gpc.LocalizedText, error) {
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

func toLocalizedText(parsed []notes.LocalizedNote) []gpc.LocalizedText {
	if len(parsed) == 0 {
		return nil
	}

	notesOut := make([]gpc.LocalizedText, 0, len(parsed))
	for _, note := range parsed {
		notesOut = append(notesOut, gpc.LocalizedText{
			Language: note.Locale,
			Text:     note.Text,
		})
	}
	return notesOut
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
		if err := validate.ReleaseNotes(text); err != nil {
			return nil, WrapUsageError(fmt.Errorf("%w for locale %q", err, locale))
		}
		localeKey := strings.ToLower(locale)
		if _, ok := seenLocales[localeKey]; ok {
			return nil, UsageErrorf("duplicate release note locale %q", locale)
		}
		seenLocales[localeKey] = struct{}{}
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
