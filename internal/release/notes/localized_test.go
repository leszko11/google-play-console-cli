package notes

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLocalizedInputInlineTextSingleLocale(t *testing.T) {
	notes, err := ParseLocalizedInput("", "Bug fixes and stability improvements.", "en-US", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Locale != "en-US" || notes[0].Text != "Bug fixes and stability improvements." {
		t.Fatalf("unexpected note: %+v", notes[0])
	}
}

func TestParseLocalizedInputTaggedText(t *testing.T) {
	raw := `<pl-PL>
Poprawki bledow i ulepszenia stabilnosci.
</pl-PL>
<en-US>
Bug fixes and stability improvements.
</en-US>`
	notes, err := ParseLocalizedInput("", raw, "en-US", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[0].Locale != "pl-PL" || notes[1].Locale != "en-US" {
		t.Fatalf("unexpected locales: %+v", notes)
	}
}

func TestParseLocalizedInputTaggedTextWithTrailingContent(t *testing.T) {
	raw := `<en-US>
Bug fixes and stability improvements.
</en-US>
trailing`
	_, err := ParseLocalizedInput("", raw, "en-US", nil)
	if err == nil || !strings.Contains(err.Error(), "unexpected text outside locale blocks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseLocalizedInputTaggedTextDuplicateLocale(t *testing.T) {
	raw := `<en-US>
One.
</en-US>
<en-US>
Two.
</en-US>`
	_, err := ParseLocalizedInput("", raw, "en-US", nil)
	if err == nil || !strings.Contains(err.Error(), "duplicate locale") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseLocalizedInputEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	_, err := ParseLocalizedInput(path, "", "en-US", func(string) ([]byte, error) {
		return []byte(" \n"), nil
	})
	if err == nil || !strings.Contains(err.Error(), "notes file is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseLocalizedInputReadFileError(t *testing.T) {
	_, err := ParseLocalizedInput("/tmp/notes.txt", "", "en-US", func(string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	if err == nil || !strings.Contains(err.Error(), "failed to read release notes file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
