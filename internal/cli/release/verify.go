package release

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	notesgen "github.com/leszko11/google-play-console-cli/internal/release/notes"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var (
	javaMajorRegex     = regexp.MustCompile(`version "([0-9]+)`)
	javaAnyDigitsRegex = regexp.MustCompile(`\b([0-9]{1,2})\b`)
)

type verifyCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Blocking bool   `json:"blocking,omitempty"`
}

type verifyResult struct {
	PackageName    string        `json:"packageName"`
	Track          string        `json:"track"`
	ProjectDir     string        `json:"projectDir"`
	BuildTask      string        `json:"buildTask"`
	NotesMode      string        `json:"notesMode"`
	Status         string        `json:"status"`
	Checks         []verifyCheck `json:"checks"`
	BlockingIssues []string      `json:"blockingIssues,omitempty"`
	Warnings       []string      `json:"warnings,omitempty"`
}

type verifyOptions struct {
	PackageName string
	Track       string
	ProjectDir  string
	BuildTask   string
	ProbeTrack  bool
	NotesMode   string
	NotesFile   string
	NotesLocale string
}

func newVerifyCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var (
		packageName string
		track       string
		projectDir  string
		buildTask   string
		probeTrack  bool
		notesMode   string
		notesFile   string
		notesLocale string
	)

	fs.StringVar(&packageName, "package-name", defaultStagingPackage, "Target package name")
	fs.StringVar(&track, "track", defaultTrack, "Target track name")
	fs.StringVar(&projectDir, "project-dir", ".", "Android project directory")
	fs.StringVar(&buildTask, "build-task", defaultBuildTask, "Gradle build task for release bundle")
	fs.BoolVar(&probeTrack, "probe-track", false, "Create temporary edit and probe target track")
	fs.StringVar(&notesMode, "notes-mode", notesgen.ModeGit, "Release notes mode: git, file, none")
	fs.StringVar(&notesFile, "notes-file", "", "Release notes file path when notes-mode=file")
	fs.StringVar(&notesLocale, "notes-locale", notesgen.DefaultLocale, "Release notes locale")

	return &ffcli.Command{
		Name:      "verify",
		ShortHelp: "Run non-mutating release readiness checks",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts := verifyOptions{
				PackageName: strings.TrimSpace(packageName),
				Track:       strings.TrimSpace(track),
				ProjectDir:  strings.TrimSpace(projectDir),
				BuildTask:   strings.TrimSpace(buildTask),
				ProbeTrack:  probeTrack,
				NotesMode:   strings.TrimSpace(notesMode),
				NotesFile:   strings.TrimSpace(notesFile),
				NotesLocale: strings.TrimSpace(notesLocale),
			}
			result, err := runVerify(ctx, deps, opts)
			_ = shared.WriteJSON(deps.Stdout, result)
			if err != nil {
				return err
			}
			if result.Status != "ok" {
				return fmt.Errorf("release verification failed")
			}
			return nil
		},
	}
}

func runVerify(ctx context.Context, deps Deps, opts verifyOptions) (verifyResult, error) {
	result := verifyResult{
		PackageName: opts.PackageName,
		Track:       opts.Track,
		ProjectDir:  opts.ProjectDir,
		BuildTask:   opts.BuildTask,
		NotesMode:   opts.NotesMode,
		Status:      "failed",
		Checks:      make([]verifyCheck, 0, 8),
	}
	if result.PackageName == "" {
		result.PackageName = defaultStagingPackage
	}
	if result.Track == "" {
		result.Track = defaultTrack
	}
	if result.ProjectDir == "" {
		result.ProjectDir = "."
	}
	if result.BuildTask == "" {
		result.BuildTask = defaultBuildTask
	}
	if result.NotesMode == "" {
		result.NotesMode = notesgen.ModeGit
	}

	if info, err := os.Stat(result.ProjectDir); err != nil || !info.IsDir() {
		issue := fmt.Sprintf("project directory is not accessible: %s", result.ProjectDir)
		result.addBlocking("project_dir", issue)
		return finalizeVerifyResult(result), nil
	}

	javaOut, javaErr := deps.RunCommand(ctx, result.ProjectDir, "java", "-version")
	if javaErr != nil {
		result.addBlocking("java_version", fmt.Sprintf("failed to run java -version: %v", javaErr))
	} else {
		javaMajor, parseErr := parseJavaMajor(javaOut)
		if parseErr != nil {
			result.addBlocking("java_version", fmt.Sprintf("unable to parse java version: %v", parseErr))
		} else if javaMajor < 21 {
			result.addBlocking("java_version", fmt.Sprintf("Java 21+ required, found %d", javaMajor))
		} else {
			result.addOK("java_version", fmt.Sprintf("Java %d detected", javaMajor))
		}
	}

	gradlewPath := filepath.Join(result.ProjectDir, "gradlew")
	if info, err := os.Stat(gradlewPath); err != nil || info.IsDir() {
		result.addBlocking("gradle_wrapper", fmt.Sprintf("gradle wrapper not found at %s", gradlewPath))
	} else {
		result.addOK("gradle_wrapper", gradlewPath)
	}

	taskListOut, taskErr := deps.RunCommand(ctx, result.ProjectDir, "./gradlew", ":app:tasks", "--all")
	if taskErr != nil {
		result.addBlocking("gradle_task", fmt.Sprintf("failed to inspect Gradle tasks: %v", taskErr))
	} else if !strings.Contains(strings.ToLower(taskListOut), strings.ToLower(taskToken(result.BuildTask))) {
		result.addBlocking("gradle_task", fmt.Sprintf("build task %q not found", result.BuildTask))
	} else {
		result.addOK("gradle_task", fmt.Sprintf("task %q is available", result.BuildTask))
	}

	client, requestCtx, cancel, clientErr := buildClient(ctx, deps)
	if clientErr != nil {
		result.addBlocking("service_account", clientErr.Error())
		return finalizeVerifyResult(result), nil
	}
	defer cancel()
	result.addOK("service_account", "service account resolved")

	accessErr := client.VerifyPackageAccess(requestCtx, result.PackageName)
	if accessErr != nil {
		result.addBlocking("package_access", accessErr.Error())
		if errors.Is(accessErr, gpc.ErrPackageNotFound) {
			result.Warnings = append(result.Warnings, "Package is not initialized in Play Console. Upload one artifact manually once, then rerun.")
		}
	} else {
		result.addOK("package_access", "package access verified")
	}

	if opts.ProbeTrack && accessErr == nil {
		edit, err := client.CreateEdit(requestCtx, result.PackageName)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("track probe skipped: failed to create edit: %v", err))
			result.addWarn("track_probe", "failed to create temporary edit")
		} else {
			_, trackErr := client.GetTrack(requestCtx, result.PackageName, edit.ID, result.Track)
			deleteErr := client.DeleteEdit(requestCtx, result.PackageName, edit.ID)
			if trackErr != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("track probe could not read track %q: %v", result.Track, trackErr))
				result.addWarn("track_probe", fmt.Sprintf("unable to read track %q", result.Track))
			} else if deleteErr != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("temporary edit cleanup failed: %v", deleteErr))
				result.addWarn("track_probe", "track exists but temporary edit cleanup failed")
			} else {
				result.addOK("track_probe", fmt.Sprintf("track %q is readable", result.Track))
			}
		}
	} else if opts.ProbeTrack {
		result.addWarn("track_probe", "skipped due to package access failure")
	} else {
		result.addWarn("track_probe", "skipped (enable --probe-track to run)")
	}

	notesResult, notesErr := notesgen.Generate(notesgen.Input{
		Mode:     result.NotesMode,
		FilePath: opts.NotesFile,
		Locale:   opts.NotesLocale,
		RepoDir:  result.ProjectDir,
	}, notesgen.Deps{})
	if notesErr != nil {
		result.addBlocking("notes_source", notesErr.Error())
	} else {
		if notesResult.Mode == notesgen.ModeNone {
			result.addWarn("notes_source", "release notes disabled")
		} else {
			parsedNotes, parseErr := notesgen.ParseLocalizedText(notesResult.Text, notesResult.Locale)
			if parseErr != nil {
				result.addBlocking("notes_source", parseErr.Error())
			} else {
				result.addOK("notes_source", fmt.Sprintf("notes ready (%s, locales=%d)", notesResult.Source, len(parsedNotes)))
			}
		}
	}

	return finalizeVerifyResult(result), nil
}

func finalizeVerifyResult(result verifyResult) verifyResult {
	if len(result.BlockingIssues) == 0 {
		result.Status = "ok"
	} else {
		result.Status = "failed"
	}
	return result
}

func (r *verifyResult) addBlocking(name, detail string) {
	r.Checks = append(r.Checks, verifyCheck{
		Name:     name,
		Status:   "error",
		Detail:   detail,
		Blocking: true,
	})
	r.BlockingIssues = append(r.BlockingIssues, detail)
}

func (r *verifyResult) addOK(name, detail string) {
	r.Checks = append(r.Checks, verifyCheck{
		Name:   name,
		Status: "ok",
		Detail: detail,
	})
}

func (r *verifyResult) addWarn(name, detail string) {
	r.Checks = append(r.Checks, verifyCheck{
		Name:   name,
		Status: "warning",
		Detail: detail,
	})
}

func taskToken(task string) string {
	task = strings.TrimSpace(task)
	if task == "" {
		return ""
	}
	if idx := strings.LastIndex(task, ":"); idx >= 0 {
		return task[idx+1:]
	}
	return task
}

func parseJavaMajor(raw string) (int, error) {
	out := strings.TrimSpace(raw)
	if out == "" {
		return 0, fmt.Errorf("empty output")
	}
	matches := javaMajorRegex.FindStringSubmatch(out)
	if len(matches) >= 2 {
		var major int
		if _, err := fmt.Sscanf(matches[1], "%d", &major); err == nil {
			return major, nil
		}
	}

	if any := javaAnyDigitsRegex.FindStringSubmatch(out); len(any) >= 2 {
		var major int
		if _, err := fmt.Sscanf(any[1], "%d", &major); err == nil {
			return major, nil
		}
	}
	return 0, fmt.Errorf("could not parse java major from output")
}
