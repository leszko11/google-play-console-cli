package listing

import (
	"fmt"

	"github.com/leszko11/google-play-console-cli/internal/cli/edits"
)

type ValidationSummary struct {
	LocaleCount int `json:"localeCount"`
	ImageCount  int `json:"imageCount"`
}

func ValidateListingsDir(root string) (ValidationSummary, error) {
	locales, err := ScanListingsDir(root)
	if err != nil {
		return ValidationSummary{}, err
	}

	summary := ValidationSummary{LocaleCount: len(locales)}
	for _, locale := range locales {
		for imageType, paths := range locale.Images {
			for _, imagePath := range paths {
				if err := edits.ValidateImageUploadFile(imageType, imagePath); err != nil {
					return ValidationSummary{}, fmt.Errorf("%s %s: %w", locale.Locale, imageType, err)
				}
				summary.ImageCount++
			}
		}
	}

	return summary, nil
}
