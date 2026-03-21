package cmd

import (
	"context"
	"flag"
	"os"
	"strings"
	"testing"
)

func TestNewRootCommand(t *testing.T) {
	cmd := newRootCommand()
	if cmd.Name != "gpc" {
		t.Fatalf("expected gpc, got %q", cmd.Name)
	}
}

func TestRootVersionFlagPrintsJSON(t *testing.T) {
	prevVersion, prevCommit, prevDate := Version, Commit, Date
	Version, Commit, Date = "1.0.0", "abc123", "2026-03-01T12:00:00Z"
	defer func() { Version, Commit, Date = prevVersion, prevCommit, prevDate }()

	oldArgs := os.Args
	os.Args = []string{"gpc", "--version"}
	defer func() { os.Args = oldArgs }()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer stdoutR.Close()
	oldStdout := os.Stdout
	os.Stdout = stdoutW
	defer func() { os.Stdout = oldStdout }()

	exit := Run([]string{"--version"})
	_ = stdoutW.Close()
	if exit != ExitCodeOK {
		t.Fatalf("expected exit 0, got %d", exit)
	}

	buf := make([]byte, 4096)
	n, _ := stdoutR.Read(buf)
	out := string(buf[:n])
	if !strings.Contains(out, `"version":"1.0.0"`) ||
		!strings.Contains(out, `"commit":"abc123"`) ||
		!strings.Contains(out, `"date":"2026-03-01T12:00:00Z"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestRootUnknownCommandPrintsSuggestions(t *testing.T) {
	cmd := newRootCommand()

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	defer stderrR.Close()

	oldStderr := os.Stderr
	os.Stderr = stderrW
	defer func() { os.Stderr = oldStderr }()

	err = cmd.Exec(context.Background(), []string{"statu"})
	_ = stderrW.Close()
	if err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := stderrR.Read(buf)
	out := string(buf[:n])
	if !strings.Contains(out, `Unknown command "statu". Did you mean:`) {
		t.Fatalf("unexpected stderr: %s", out)
	}
	if !strings.Contains(out, "gpc status") {
		t.Fatalf("expected status suggestion, got: %s", out)
	}
}
