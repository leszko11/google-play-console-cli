package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

func maybeOfferBootstrapBuild(err error) {
	if !errors.Is(err, gpc.ErrPackageNotFound) {
		return
	}
	if !isInteractiveTerminal(os.Stdin, os.Stderr) {
		return
	}

	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		return
	}
	projectDir, gradlewPath, ok := detectAndroidGradleProject(cwd)
	if !ok {
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Detected Android Gradle project in %s\n", projectDir)
	shouldBuild := promptYesNo(
		reader,
		os.Stderr,
		"Build a bootstrap artifact now for manual Play Console upload?",
		false,
	)
	if !shouldBuild {
		return
	}

	artifactType := promptChoice(
		reader,
		os.Stderr,
		"Artifact type",
		[]string{"aab", "apk"},
		"aab",
	)
	module := promptText(reader, os.Stderr, "Gradle module", "app")
	variant := promptText(reader, os.Stderr, "Build variant", "release")

	task, taskErr := gradleTaskFor(artifactType, module, variant)
	if taskErr != nil {
		fmt.Fprintf(os.Stderr, "failed to compute Gradle task: %v\n", taskErr)
		return
	}

	fmt.Fprintf(os.Stderr, "Running %s %s\n", gradlewPath, task)
	if runErr := runGradleTask(projectDir, gradlewPath, task); runErr != nil {
		fmt.Fprintf(os.Stderr, "bootstrap build failed: %v\n", runErr)
		return
	}

	artifactPath, artifactErr := findBuiltArtifact(projectDir, module, variant, artifactType)
	if artifactErr != nil {
		fmt.Fprintf(os.Stderr, "build finished but could not find artifact: %v\n", artifactErr)
		return
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Bootstrap artifact ready: %s\n", artifactPath)
	fmt.Fprintln(os.Stderr, "Upload this file once in Play Console UI to initialize the package, then rerun gpc.")
}

func isInteractiveTerminal(stdin, stderr *os.File) bool {
	return isCharDevice(stdin) && isCharDevice(stderr)
}

func isCharDevice(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func detectAndroidGradleProject(cwd string) (string, string, bool) {
	candidates := []string{
		cwd,
		filepath.Join(cwd, "android"),
	}
	for _, dir := range candidates {
		wrapper := filepath.Join(dir, "gradlew")
		info, err := os.Stat(wrapper)
		if err == nil && !info.IsDir() {
			return dir, wrapper, true
		}
	}
	return "", "", false
}

func promptYesNo(reader *bufio.Reader, out io.Writer, question string, defaultYes bool) bool {
	defaultValue := "y/N"
	if defaultYes {
		defaultValue = "Y/n"
	}
	for {
		fmt.Fprintf(out, "%s [%s]: ", question, defaultValue)
		value := strings.ToLower(strings.TrimSpace(readLine(reader)))
		if value == "" {
			return defaultYes
		}
		if value == "y" || value == "yes" {
			return true
		}
		if value == "n" || value == "no" {
			return false
		}
		fmt.Fprintln(out, "Please answer yes or no.")
	}
}

func promptChoice(reader *bufio.Reader, out io.Writer, question string, choices []string, defaultValue string) string {
	allowed := map[string]string{}
	for _, choice := range choices {
		allowed[strings.ToLower(choice)] = choice
	}

	for {
		fmt.Fprintf(out, "%s [%s] (default: %s): ", question, strings.Join(choices, "/"), defaultValue)
		value := strings.ToLower(strings.TrimSpace(readLine(reader)))
		if value == "" {
			return defaultValue
		}
		if mapped, ok := allowed[value]; ok {
			return mapped
		}
		fmt.Fprintf(out, "Invalid choice %q. Allowed: %s\n", value, strings.Join(choices, ", "))
	}
}

func promptText(reader *bufio.Reader, out io.Writer, question, defaultValue string) string {
	fmt.Fprintf(out, "%s (default: %s): ", question, defaultValue)
	value := strings.TrimSpace(readLine(reader))
	if value == "" {
		return defaultValue
	}
	return value
}

func readLine(reader *bufio.Reader) string {
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return ""
	}
	return line
}

func gradleTaskFor(artifactType, module, variant string) (string, error) {
	module = strings.TrimSpace(strings.TrimPrefix(module, ":"))
	if module == "" {
		return "", fmt.Errorf("module is required")
	}
	variantTaskToken := toGradleTaskToken(variant)
	if variantTaskToken == "" {
		return "", fmt.Errorf("variant is required")
	}

	switch strings.ToLower(strings.TrimSpace(artifactType)) {
	case "aab":
		return fmt.Sprintf(":%s:bundle%s", module, variantTaskToken), nil
	case "apk":
		return fmt.Sprintf(":%s:assemble%s", module, variantTaskToken), nil
	default:
		return "", fmt.Errorf("unsupported artifact type %q", artifactType)
	}
}

func toGradleTaskToken(value string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	})
	if len(parts) == 0 {
		return ""
	}
	var token strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		token.WriteRune(unicode.ToUpper(runes[0]))
		for i := 1; i < len(runes); i++ {
			token.WriteRune(runes[i])
		}
	}
	return token.String()
}

func runGradleTask(projectDir, gradlewPath, task string) error {
	var cmd *exec.Cmd
	if isExecutable(gradlewPath) {
		cmd = exec.Command(gradlewPath, task)
	} else {
		cmd = exec.Command("bash", gradlewPath, task)
	}
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0o111 != 0
}

type artifactCandidate struct {
	path    string
	modTime int64
}

func findBuiltArtifact(projectDir, module, variant, artifactType string) (string, error) {
	module = strings.TrimSpace(strings.TrimPrefix(module, ":"))
	variant = strings.ToLower(strings.TrimSpace(variant))
	if module == "" || variant == "" {
		return "", fmt.Errorf("module and variant are required")
	}

	var baseDir string
	var suffix string
	switch strings.ToLower(strings.TrimSpace(artifactType)) {
	case "aab":
		baseDir = filepath.Join(projectDir, module, "build", "outputs", "bundle")
		suffix = ".aab"
	case "apk":
		baseDir = filepath.Join(projectDir, module, "build", "outputs", "apk")
		suffix = ".apk"
	default:
		return "", fmt.Errorf("unsupported artifact type %q", artifactType)
	}

	var candidates []artifactCandidate
	walkErr := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), suffix) {
			return nil
		}
		if !pathMatchesVariant(path, variant) {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		candidates = append(candidates, artifactCandidate{
			path:    path,
			modTime: info.ModTime().UnixNano(),
		})
		return nil
	})
	if walkErr != nil {
		return "", walkErr
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no %s artifact found under %s", suffix, baseDir)
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].modTime == candidates[j].modTime {
			return candidates[i].path > candidates[j].path
		}
		return candidates[i].modTime > candidates[j].modTime
	})
	return candidates[0].path, nil
}

func pathMatchesVariant(path, variant string) bool {
	lowerPath := strings.ToLower(path)
	return strings.Contains(lowerPath, string(filepath.Separator)+variant+string(filepath.Separator)) ||
		strings.Contains(lowerPath, "-"+variant+".")
}
