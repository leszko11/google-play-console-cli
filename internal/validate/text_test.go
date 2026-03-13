package validate

import (
	"strings"
	"testing"
)

func TestTitleLength(t *testing.T) {
	if err := Title(strings.Repeat("a", 30)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Title(strings.Repeat("a", 31)); err == nil || err.Error() != "title too long: 31 characters (max 30)" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseNotesLengthCountsRunes(t *testing.T) {
	valid := strings.Repeat("ą", 500)
	if err := ReleaseNotes(valid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	invalid := strings.Repeat("ą", 501)
	if err := ReleaseNotes(invalid); err == nil || err.Error() != "release notes too long: 501 characters (max 500)" {
		t.Fatalf("unexpected error: %v", err)
	}
}
