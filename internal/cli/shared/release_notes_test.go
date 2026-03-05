package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeReleaseNotesFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "release-notes.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write release notes fixture: %v", err)
	}
	return path
}

func TestParseReleaseNotesFile_AllowsObjectPayload(t *testing.T) {
	path := writeReleaseNotesFixture(t, `{"pl-PL":"Wersja testowa","en-US":"Test release"}`)

	notes, err := ParseReleaseNotesFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[0].Language != "en-US" || notes[0].Text != "Test release" {
		t.Fatalf("expected sorted en-US note first, got %+v", notes[0])
	}
	if notes[1].Language != "pl-PL" {
		t.Fatalf("expected pl-PL second, got %+v", notes[1])
	}
}

func TestParseReleaseNotesFile_AllowsArrayPayload(t *testing.T) {
	path := writeReleaseNotesFixture(t, `[{"language":"de-DE","text":"Hinweis"},{"language":"en-US","text":"Release note"}]`)

	notes, err := ParseReleaseNotesFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[0].Language != "de-DE" || notes[1].Language != "en-US" {
		t.Fatalf("expected deterministic ordering, got %+v", notes)
	}
}

func TestParseReleaseNotesFile_RejectsDuplicateLocale(t *testing.T) {
	path := writeReleaseNotesFixture(t, `[{"language":"en-US","text":"A"},{"language":"en-US","text":"B"}]`)

	_, err := ParseReleaseNotesFile(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}
}

func TestParseReleaseNotesFile_RejectsDuplicateLocaleCaseInsensitive(t *testing.T) {
	path := writeReleaseNotesFixture(t, `[{"language":"en-US","text":"A"},{"language":"en-us","text":"B"}]`)

	_, err := ParseReleaseNotesFile(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "duplicate release note locale") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseReleaseNotesFile_RejectsInvalidShape(t *testing.T) {
	path := writeReleaseNotesFixture(t, `{"en-US":{"text":"invalid"}}`)

	_, err := ParseReleaseNotesFile(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsUsageError(err) {
		t.Fatalf("expected usage error, got %T: %v", err, err)
	}
}

func TestParseReleaseNotesFile_EmptyPathIsNoop(t *testing.T) {
	notes, err := ParseReleaseNotesFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notes != nil {
		t.Fatalf("expected nil notes, got %+v", notes)
	}
}

func TestParseReleaseNotesInput_ParsesTaggedFile(t *testing.T) {
	path := writeReleaseNotesFixture(t, `<en-US>
Bug fixes.
</en-US>
<pl-PL>
Poprawki.
</pl-PL>`)

	notes, err := ParseReleaseNotesInput(path, "", "en-US", os.ReadFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[0].Language != "en-US" || notes[1].Language != "pl-PL" {
		t.Fatalf("unexpected notes: %+v", notes)
	}
}

func TestParseReleaseNotesInput_ParsesInlinePlainTextWithDefaultLocale(t *testing.T) {
	notes, err := ParseReleaseNotesInput("", "Single note", "cs-CZ", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Language != "cs-CZ" || notes[0].Text != "Single note" {
		t.Fatalf("unexpected note: %+v", notes[0])
	}
}
