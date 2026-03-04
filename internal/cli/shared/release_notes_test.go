package shared

import (
	"os"
	"path/filepath"
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
