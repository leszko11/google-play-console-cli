package validate

import (
	"fmt"
	"unicode/utf8"
)

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

const (
	maxTitleChars            = 30
	maxShortDescriptionChars = 80
	maxFullDescriptionChars  = 4000
	maxReleaseNotesChars     = 500
	maxReleaseNameChars      = 50
)

func Title(text string) error {
	return maxRunes("title", text, maxTitleChars)
}

func ShortDescription(text string) error {
	return maxRunes("short description", text, maxShortDescriptionChars)
}

func FullDescription(text string) error {
	return maxRunes("full description", text, maxFullDescriptionChars)
}

func ReleaseNotes(text string) error {
	return maxRunes("release notes", text, maxReleaseNotesChars)
}

func ReleaseName(text string) error {
	return maxRunes("release name", text, maxReleaseNameChars)
}

func maxRunes(label, text string, limit int) error {
	count := utf8.RuneCountInString(text)
	if count > limit {
		return ValidationError{Message: fmt.Sprintf("%s too long: %d characters (max %d)", label, count, limit)}
	}
	return nil
}
