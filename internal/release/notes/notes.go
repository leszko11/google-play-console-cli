package notes

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	ModeGit  = "git"
	ModeFile = "file"
	ModeNone = "none"

	DefaultLocale = "en-US"
	DefaultEntry  = "Maintenance and stability improvements."
)

type Input struct {
	Mode       string
	RepoDir    string
	FilePath   string
	Text       string
	Locale     string
	MaxEntries int
}

type Result struct {
	Mode    string   `json:"mode"`
	Source  string   `json:"source"`
	Locale  string   `json:"locale,omitempty"`
	Text    string   `json:"text,omitempty"`
	Entries []string `json:"entries,omitempty"`
}

type Deps struct {
	ReadFile func(string) ([]byte, error)
	RunGit   func(repoDir string, args ...string) (string, error)
}

func Generate(input Input, deps Deps) (Result, error) {
	deps = withDefaults(deps)

	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = ModeGit
	}
	locale := strings.TrimSpace(input.Locale)
	if locale == "" {
		locale = DefaultLocale
	}

	switch mode {
	case ModeNone:
		return Result{
			Mode:   ModeNone,
			Source: ModeNone,
		}, nil
	case ModeFile:
		path := strings.TrimSpace(input.FilePath)
		if path == "" {
			return Result{}, fmt.Errorf("--notes-file is required when --notes-mode=file")
		}
		raw, err := deps.ReadFile(path)
		if err != nil {
			return Result{}, fmt.Errorf("failed to read notes file: %w", err)
		}
		text := strings.TrimSpace(string(raw))
		if text == "" {
			return Result{}, fmt.Errorf("notes file is empty: %s", path)
		}
		return Result{
			Mode:   ModeFile,
			Source: path,
			Locale: locale,
			Text:   text,
		}, nil
	case ModeGit:
		if strings.TrimSpace(input.Text) != "" {
			return Result{
				Mode:   ModeGit,
				Source: "manual-text",
				Locale: locale,
				Text:   strings.TrimSpace(input.Text),
			}, nil
		}
		return buildFromGit(input, deps.RunGit, locale)
	default:
		return Result{}, fmt.Errorf("unsupported notes mode %q (allowed: git, file, none)", input.Mode)
	}
}

func withDefaults(deps Deps) Deps {
	if deps.ReadFile == nil {
		deps.ReadFile = os.ReadFile
	}
	if deps.RunGit == nil {
		deps.RunGit = runGit
	}
	return deps
}

func buildFromGit(input Input, runGitFn func(repoDir string, args ...string) (string, error), locale string) (Result, error) {
	maxEntries := input.MaxEntries
	if maxEntries <= 0 {
		maxEntries = 20
	}

	baseRef := ""
	describeOut, describeErr := runGitFn(input.RepoDir, "describe", "--tags", "--abbrev=0")
	if describeErr == nil {
		baseRef = strings.TrimSpace(describeOut)
	}

	logArgs := []string{"log", "--pretty=%s", "--max-count", strconv.Itoa(maxEntries)}
	if baseRef != "" {
		logArgs = append(logArgs, baseRef+"..HEAD")
	}
	logOut, err := runGitFn(input.RepoDir, logArgs...)
	if err != nil {
		return Result{}, fmt.Errorf("failed to collect git commit messages: %w", err)
	}

	entries := parseLines(logOut)
	if len(entries) == 0 {
		entries = []string{DefaultEntry}
	}

	text := formatAsBullets(entries)
	source := "git"
	if baseRef != "" {
		source = "git:" + baseRef
	}

	return Result{
		Mode:    ModeGit,
		Source:  source,
		Locale:  locale,
		Text:    text,
		Entries: entries,
	}, nil
}

func parseLines(raw string) []string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func formatAsBullets(entries []string) string {
	var buf bytes.Buffer
	for i, entry := range entries {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("- ")
		buf.WriteString(entry)
	}
	return buf.String()
}

func runGit(repoDir string, args ...string) (string, error) {
	gitArgs := args
	if strings.TrimSpace(repoDir) != "" {
		gitArgs = append([]string{"-C", repoDir}, args...)
	}
	cmd := exec.Command("git", gitArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errOutput := strings.TrimSpace(stderr.String())
		if errOutput == "" {
			errOutput = err.Error()
		}
		return "", fmt.Errorf("%s", errOutput)
	}
	return stdout.String(), nil
}
