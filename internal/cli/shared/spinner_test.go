package shared

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSpinnerDisabledDoesNotWrite(t *testing.T) {
	var out bytes.Buffer
	spinner := newSpinner(&out, "Uploading", false, time.Millisecond)
	spinner.Update("Still uploading")
	spinner.Success("Uploaded")

	if out.Len() != 0 {
		t.Fatalf("expected no spinner output, got %q", out.String())
	}
}

func TestSpinnerEnabledWritesFinalMessage(t *testing.T) {
	var out bytes.Buffer
	spinner := newSpinner(&out, "Uploading", true, time.Millisecond)

	time.Sleep(5 * time.Millisecond)
	spinner.Update("Validating")
	time.Sleep(5 * time.Millisecond)
	spinner.Success("Done")

	got := out.String()
	if !strings.Contains(got, "Done") {
		t.Fatalf("expected final spinner message, got %q", got)
	}
}
