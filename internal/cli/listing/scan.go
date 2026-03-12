package listing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

var screenshotImageTypes = map[string]struct{}{
	"phoneScreenshots":     {},
	"sevenInchScreenshots": {},
	"tenInchScreenshots":   {},
	"tvScreenshots":        {},
	"wearScreenshots":      {},
}

var singleFileImageTypes = map[string]struct{}{
	"icon":           {},
	"featureGraphic": {},
	"tvBanner":       {},
	"promoGraphic":   {},
}

type localeData struct {
	Locale  string
	Listing gpc.ListingUpdate
	Images  map[string][]string
}

func scanListingsDir(root string) ([]localeData, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("listings directory is required")
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read listings directory: %w", err)
	}

	locales := make([]localeData, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		localePath := filepath.Join(root, entry.Name())
		locale, err := scanLocaleDir(entry.Name(), localePath)
		if err != nil {
			return nil, err
		}
		locales = append(locales, locale)
	}

	sort.Slice(locales, func(i, j int) bool {
		return locales[i].Locale < locales[j].Locale
	})
	if len(locales) == 0 {
		return nil, fmt.Errorf("no locale directories found in %s", root)
	}
	return locales, nil
}

func scanLocaleDir(locale, dir string) (localeData, error) {
	title, err := readRequiredFile(filepath.Join(dir, "title.txt"))
	if err != nil {
		return localeData{}, fmt.Errorf("%s: %w", locale, err)
	}
	shortDescription, err := readRequiredFile(filepath.Join(dir, "short-description.txt"))
	if err != nil {
		return localeData{}, fmt.Errorf("%s: %w", locale, err)
	}
	fullDescription, err := readRequiredFile(filepath.Join(dir, "full-description.txt"))
	if err != nil {
		return localeData{}, fmt.Errorf("%s: %w", locale, err)
	}

	images, err := scanImagesDir(filepath.Join(dir, "images"))
	if err != nil {
		return localeData{}, fmt.Errorf("%s: %w", locale, err)
	}

	return localeData{
		Locale: locale,
		Listing: gpc.ListingUpdate{
			Title:            title,
			ShortDescription: shortDescription,
			FullDescription:  fullDescription,
		},
		Images: images,
	}, nil
}

func readRequiredFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("missing required file %s", filepath.Base(path))
		}
		return "", err
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", fmt.Errorf("required file %s is empty", filepath.Base(path))
	}
	return text, nil
}

func scanImagesDir(dir string) (map[string][]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]string{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("images must be a directory")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	images := map[string][]string{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			imageType := entry.Name()
			if _, ok := screenshotImageTypes[imageType]; !ok {
				return nil, fmt.Errorf("unsupported image directory %q", imageType)
			}
			files, err := collectFiles(path)
			if err != nil {
				return nil, err
			}
			if len(files) > 0 {
				images[imageType] = files
			}
			continue
		}

		imageType := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if _, ok := singleFileImageTypes[imageType]; !ok {
			return nil, fmt.Errorf("unsupported image file %q", entry.Name())
		}
		if _, exists := images[imageType]; exists {
			return nil, fmt.Errorf("multiple files provided for image type %q", imageType)
		}
		images[imageType] = []string{path}
	}

	return images, nil
}

func collectFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("nested directories are not supported in %s", dir)
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}
