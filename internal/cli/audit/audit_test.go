package audit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixedClock() time.Time {
	return time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
}

func runCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	deps.Clock = fixedClock
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestLogRequiresCommand(t *testing.T) {
	dir := t.TempDir()
	_, err := runCommand(t, Deps{HomeDir: dir}, "log")
	if err == nil || !strings.Contains(err.Error(), "--command is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogAndShow(t *testing.T) {
	dir := t.TempDir()
	deps := Deps{HomeDir: dir}

	// Log an entry
	out, err := runCommand(t, deps, "log", "--command", "deploy", "--package-name", "com.example.app", "--detail", "v1.0")
	if err != nil {
		t.Fatalf("log failed: %v", err)
	}
	if !strings.Contains(out, "logged: deploy") {
		t.Fatalf("unexpected output: %s", out)
	}

	// Verify file was created
	path := filepath.Join(dir, ".gpc", "audit.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("audit file not created: %v", err)
	}
	if !strings.Contains(string(data), `"command":"deploy"`) {
		t.Fatalf("entry not in file: %s", data)
	}

	// Show entries
	out, err = runCommand(t, deps, "show", "--output", "json")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if !strings.Contains(out, `"command":"deploy"`) {
		t.Fatalf("missing entry in show: %s", out)
	}
}

func TestShowEmptyLog(t *testing.T) {
	dir := t.TempDir()
	out, err := runCommand(t, Deps{HomeDir: dir}, "show", "--output", "json")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestShowEmptyLogTableOutput(t *testing.T) {
	dir := t.TempDir()
	out, err := runCommand(t, Deps{HomeDir: dir}, "show", "--output", "table")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if !strings.Contains(out, "TIMESTAMP\tCOMMAND\tPACKAGE\tUSER\tDETAIL") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestShowLastN(t *testing.T) {
	dir := t.TempDir()
	deps := Deps{HomeDir: dir}

	// Log multiple entries
	for _, cmd := range []string{"deploy", "rollback", "publish"} {
		_, err := runCommand(t, deps, "log", "--command", cmd)
		if err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	// Show last 2
	out, err := runCommand(t, deps, "show", "--output", "json", "--last", "2")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if strings.Contains(out, `"command":"deploy"`) {
		t.Fatalf("should not contain first entry: %s", out)
	}
	if !strings.Contains(out, `"command":"rollback"`) || !strings.Contains(out, `"command":"publish"`) {
		t.Fatalf("missing last entries: %s", out)
	}
}

func TestShowTableOutput(t *testing.T) {
	dir := t.TempDir()
	deps := Deps{HomeDir: dir}

	_, err := runCommand(t, deps, "log", "--command", "deploy", "--package-name", "com.example.app")
	if err != nil {
		t.Fatalf("log failed: %v", err)
	}

	out, err := runCommand(t, deps, "show", "--output", "table")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	for _, want := range []string{"TIMESTAMP", "COMMAND", "PACKAGE", "deploy", "com.example.app"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}
