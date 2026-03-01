package cmd

import (
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
