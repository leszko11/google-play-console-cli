package workflow

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkflowRunRequiresConfirmUnlessDryRun(t *testing.T) {
	manifestPath := writeWorkflowManifest(t, `
version: 1
steps:
  - id: apps
    run: apps list --package-name com.example.app
`)

	_, err := runWorkflowCommand(t, Deps{}, "run", "--file", manifestPath)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required unless --dry-run is set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunDryRunPlansInterpolatedStepsInDependencyOrder(t *testing.T) {
	manifestPath := writeWorkflowManifest(t, `
version: 1
vars:
  packageName: com.example.app
steps:
  - id: export
    run: bootstrap --package-name ${packageName} --dir ./play
  - id: verify
    needs: [export]
    run: release verify --package-name ${packageName} --aab ./app.aab
`)

	out, err := runWorkflowCommand(t, Deps{}, "run", "--file", manifestPath, "--dry-run")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if !strings.Contains(out, `"status":"dry-run"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, `"resolved":"bootstrap --package-name com.example.app --dir ./play"`) {
		t.Fatalf("expected interpolated bootstrap command, got %s", out)
	}
	verifyIndex := strings.Index(out, `"id":"verify"`)
	exportIndex := strings.Index(out, `"id":"export"`)
	if exportIndex < 0 || verifyIndex < 0 || exportIndex > verifyIndex {
		t.Fatalf("expected export before verify, got %s", out)
	}
}

func TestWorkflowRunExecutesStepsAndCapturesStdout(t *testing.T) {
	manifestPath := writeWorkflowManifest(t, `
version: 1
vars:
  packageName: com.example.app
steps:
  - id: export
    run: bootstrap --package-name ${packageName} --dir ./play
  - id: verify
    needs: [export]
    run: release verify --package-name ${packageName} --aab ./app.aab
`)

	var calls [][]string
	deps := Deps{
		ExecutablePath: func() (string, error) { return "/usr/local/bin/gpc", nil },
		RunCommand: func(_ context.Context, name string, args []string, stdout, stderr *bytes.Buffer) error {
			calls = append(calls, append([]string{name}, args...))
			switch {
			case len(args) > 0 && args[0] == "bootstrap":
				_, _ = stdout.WriteString(`{"status":"ok","step":"bootstrap"}`)
			case len(args) > 0 && args[0] == "release":
				_, _ = stdout.WriteString(`{"status":"ok","step":"verify"}`)
			default:
				t.Fatalf("unexpected args: %v", args)
			}
			return nil
		},
	}

	out, err := runWorkflowCommand(t, deps, "run", "--file", manifestPath, "--confirm")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if strings.Join(calls[0], " ") != "/usr/local/bin/gpc bootstrap --package-name com.example.app --dir ./play" {
		t.Fatalf("unexpected first call: %v", calls[0])
	}
	if strings.Join(calls[1], " ") != "/usr/local/bin/gpc release verify --package-name com.example.app --aab ./app.aab" {
		t.Fatalf("unexpected second call: %v", calls[1])
	}
	if !strings.Contains(out, `"status":"ok"`) || !strings.Contains(out, `\"step\":\"verify\"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestWorkflowRunRejectsMissingVariables(t *testing.T) {
	manifestPath := writeWorkflowManifest(t, `
version: 1
steps:
  - id: verify
    run: release verify --package-name ${packageName} --aab ./app.aab
`)

	_, err := runWorkflowCommand(t, Deps{}, "run", "--file", manifestPath, "--dry-run")
	if err == nil || !strings.Contains(err.Error(), `unknown workflow variable "packageName"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunRejectsDependencyCycles(t *testing.T) {
	manifestPath := writeWorkflowManifest(t, `
version: 1
steps:
  - id: one
    needs: [two]
    run: apps list
  - id: two
    needs: [one]
    run: apps get --package-name com.example.app
`)

	_, err := runWorkflowCommand(t, Deps{}, "run", "--file", manifestPath, "--dry-run")
	if err == nil || !strings.Contains(err.Error(), "workflow contains cyclic or unresolved step dependencies") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunWritesFailureResultWhenStepFails(t *testing.T) {
	manifestPath := writeWorkflowManifest(t, `
version: 1
steps:
  - id: verify
    run: release verify --package-name com.example.app --aab ./app.aab
`)

	deps := Deps{
		ExecutablePath: func() (string, error) { return "/usr/local/bin/gpc", nil },
		RunCommand: func(_ context.Context, name string, args []string, stdout, stderr *bytes.Buffer) error {
			_, _ = stderr.WriteString("boom")
			return errors.New("exit status 5")
		},
	}

	out, err := runWorkflowCommand(t, deps, "run", "--file", manifestPath, "--confirm")
	if err == nil || !strings.Contains(err.Error(), `workflow step "verify" failed`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"status":"failed"`) || !strings.Contains(out, `"stderr":"boom"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func runWorkflowCommand(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var stdout bytes.Buffer
	deps.Stdout = &stdout
	if deps.Stderr == nil {
		deps.Stderr = &bytes.Buffer{}
	}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return stdout.String(), err
}

func writeWorkflowManifest(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}
