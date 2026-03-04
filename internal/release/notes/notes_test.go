package notes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateFileMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("Line one\nLine two"), 0o600); err != nil {
		t.Fatalf("write notes file: %v", err)
	}

	got, err := Generate(Input{
		Mode:     ModeFile,
		FilePath: path,
		Locale:   "pl-PL",
	}, Deps{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Mode != ModeFile || got.Source != path || got.Locale != "pl-PL" {
		t.Fatalf("unexpected metadata: %+v", got)
	}
	if got.Text != "Line one\nLine two" {
		t.Fatalf("unexpected file text: %q", got.Text)
	}
}

func TestGenerateFileModeEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte(" \n"), 0o600); err != nil {
		t.Fatalf("write notes file: %v", err)
	}

	_, err := Generate(Input{
		Mode:     ModeFile,
		FilePath: path,
	}, Deps{})
	if err == nil || !strings.Contains(err.Error(), "notes file is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateNoneMode(t *testing.T) {
	got, err := Generate(Input{Mode: ModeNone}, Deps{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Mode != ModeNone || got.Text != "" || got.Source != ModeNone {
		t.Fatalf("unexpected output: %+v", got)
	}
}

func TestGenerateGitModeWithTag(t *testing.T) {
	runGitCalls := 0
	runGit := func(_ string, args ...string) (string, error) {
		runGitCalls++
		switch strings.Join(args, " ") {
		case "describe --tags --abbrev=0":
			return "1.2.3\n", nil
		case "log --pretty=%s --max-count 20 1.2.3..HEAD":
			return "Fix crash\nImprove onboarding\n", nil
		default:
			return "", errors.New("unexpected git args")
		}
	}

	got, err := Generate(Input{
		Mode:    ModeGit,
		RepoDir: "/tmp/repo",
	}, Deps{RunGit: runGit})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runGitCalls != 2 {
		t.Fatalf("expected 2 git calls, got %d", runGitCalls)
	}
	if got.Source != "git:1.2.3" {
		t.Fatalf("unexpected source: %+v", got)
	}
	if !strings.Contains(got.Text, "- Fix crash") || !strings.Contains(got.Text, "- Improve onboarding") {
		t.Fatalf("unexpected notes text: %q", got.Text)
	}
}

func TestGenerateGitModeWithoutTag(t *testing.T) {
	runGit := func(_ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "describe --tags --abbrev=0":
			return "", errors.New("no names found")
		case "log --pretty=%s --max-count 20":
			return "Initial release\n", nil
		default:
			return "", errors.New("unexpected git args")
		}
	}

	got, err := Generate(Input{
		Mode: ModeGit,
	}, Deps{RunGit: runGit})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != "git" {
		t.Fatalf("expected source git, got %q", got.Source)
	}
	if got.Locale != DefaultLocale {
		t.Fatalf("expected default locale %q, got %q", DefaultLocale, got.Locale)
	}
	if !strings.Contains(got.Text, "- Initial release") {
		t.Fatalf("unexpected text: %q", got.Text)
	}
}

func TestGenerateGitModeEmptyHistoryFallback(t *testing.T) {
	runGit := func(_ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "describe --tags --abbrev=0":
			return "", errors.New("no names found")
		case "log --pretty=%s --max-count 20":
			return "\n", nil
		default:
			return "", errors.New("unexpected git args")
		}
	}

	got, err := Generate(Input{
		Mode: ModeGit,
	}, Deps{RunGit: runGit})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Text != "- "+DefaultEntry {
		t.Fatalf("unexpected fallback text: %q", got.Text)
	}
}

func TestGenerateUnsupportedMode(t *testing.T) {
	_, err := Generate(Input{Mode: "invalid"}, Deps{})
	if err == nil || !strings.Contains(err.Error(), "unsupported notes mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}
