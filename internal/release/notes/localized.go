package notes

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

var localeTagPattern = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$`)

type LocalizedNote struct {
	Locale string
	Text   string
}

func ParseLocalizedInput(filePath, inlineText, defaultLocale string, readFile func(string) ([]byte, error)) ([]LocalizedNote, error) {
	filePath = strings.TrimSpace(filePath)
	inlineText = strings.TrimSpace(inlineText)
	defaultLocale = strings.TrimSpace(defaultLocale)
	if defaultLocale == "" {
		defaultLocale = DefaultLocale
	}
	if filePath != "" && inlineText != "" {
		return nil, fmt.Errorf("only one of --release-notes-file or --release-notes-text can be set")
	}

	if readFile == nil {
		readFile = os.ReadFile
	}

	if filePath != "" {
		raw, err := readFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read release notes file: %w", err)
		}
		text := strings.TrimSpace(string(raw))
		if text == "" {
			return nil, fmt.Errorf("notes file is empty: %s", filePath)
		}
		return ParseLocalizedText(text, defaultLocale)
	}

	return ParseLocalizedText(inlineText, defaultLocale)
}

func ParseLocalizedText(raw, defaultLocale string) ([]LocalizedNote, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, nil
	}

	defaultLocale = strings.TrimSpace(defaultLocale)
	if defaultLocale == "" {
		defaultLocale = DefaultLocale
	}

	notes := make([]LocalizedNote, 0, 8)
	seenLocales := map[string]struct{}{}
	position := 0
	for position < len(text) {
		trimmed := strings.TrimLeftFunc(text[position:], unicode.IsSpace)
		position += len(text[position:]) - len(trimmed)
		if position >= len(text) {
			break
		}
		if text[position] != '<' {
			if len(notes) == 0 {
				return []LocalizedNote{
					{
						Locale: defaultLocale,
						Text:   text,
					},
				}, nil
			}
			return nil, fmt.Errorf("release notes tagged format is invalid: unexpected text outside locale blocks")
		}

		tagEndOffset := strings.IndexByte(text[position:], '>')
		if tagEndOffset <= 0 {
			if len(notes) == 0 {
				return []LocalizedNote{
					{
						Locale: defaultLocale,
						Text:   text,
					},
				}, nil
			}
			return nil, fmt.Errorf("release notes tagged format is invalid: malformed locale tag")
		}
		tagEnd := position + tagEndOffset

		locale := strings.TrimSpace(text[position+1 : tagEnd])
		if strings.HasPrefix(locale, "/") || !localeTagPattern.MatchString(locale) {
			if len(notes) == 0 {
				return []LocalizedNote{
					{
						Locale: defaultLocale,
						Text:   text,
					},
				}, nil
			}
			return nil, fmt.Errorf("release notes tagged format is invalid: malformed locale tag")
		}

		contentStart := tagEnd + 1
		closeTag := "</" + locale + ">"
		closeOffset := strings.Index(text[contentStart:], closeTag)
		if closeOffset < 0 {
			return nil, fmt.Errorf("release notes tagged format is invalid: missing closing tag for locale %q", locale)
		}
		contentEnd := contentStart + closeOffset
		noteText := strings.TrimSpace(text[contentStart:contentEnd])
		if noteText == "" {
			return nil, fmt.Errorf("release notes tagged format is invalid: locale %q has empty text", locale)
		}

		localeKey := strings.ToLower(locale)
		if _, exists := seenLocales[localeKey]; exists {
			return nil, fmt.Errorf("release notes tagged format is invalid: duplicate locale %q", locale)
		}
		seenLocales[localeKey] = struct{}{}

		notes = append(notes, LocalizedNote{
			Locale: locale,
			Text:   noteText,
		})
		position = contentEnd + len(closeTag)
	}

	return notes, nil
}
