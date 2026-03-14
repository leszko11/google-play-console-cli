package workflow

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/shlex"
	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var interpolationPattern = regexp.MustCompile(`\$\{([A-Za-z0-9_.-]+)\}`)

type manifest struct {
	Version int               `json:"version" yaml:"version"`
	Vars    map[string]string `json:"vars" yaml:"vars"`
	Steps   []manifestStep    `json:"steps" yaml:"steps"`
}

type manifestStep struct {
	ID    string   `json:"id" yaml:"id"`
	Needs []string `json:"needs" yaml:"needs"`
	Run   string   `json:"run" yaml:"run"`
}

type runOptions struct {
	File    string
	Confirm bool
	DryRun  bool
	Vars    multiValueFlag
	Output  string
}

type workflowResult struct {
	File         string              `json:"file"`
	Status       string              `json:"status"`
	StepCount    int                 `json:"stepCount"`
	Executed     int                 `json:"executed"`
	Interpolated map[string]string   `json:"vars,omitempty"`
	Steps        []workflowStepState `json:"steps"`
}

type workflowStepState struct {
	ID       string   `json:"id"`
	Needs    []string `json:"needs,omitempty"`
	Status   string   `json:"status"`
	Resolved string   `json:"resolved"`
	Stdout   string   `json:"stdout,omitempty"`
	Stderr   string   `json:"stderr,omitempty"`
}

type Deps struct {
	ExecutablePath func() (string, error)
	RunCommand     func(context.Context, string, []string, *bytes.Buffer, *bytes.Buffer) error
	Getwd          func() (string, error)
	Stdout         io.Writer
	Stderr         io.Writer
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)
	return &ffcli.Command{
		Name:      "workflow",
		ShortHelp: "Run declarative gpc workflows from .gpc/workflow.yml",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newRunCommand(deps),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	if deps.ExecutablePath == nil {
		deps.ExecutablePath = os.Executable
	}
	if deps.Getwd == nil {
		deps.Getwd = os.Getwd
	}
	if deps.RunCommand == nil {
		deps.RunCommand = func(ctx context.Context, name string, args []string, stdout, stderr *bytes.Buffer) error {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdout = stdout
			if deps.Stderr != nil {
				cmd.Stderr = io.MultiWriter(deps.Stderr, stderr)
			} else {
				cmd.Stderr = stderr
			}
			return cmd.Run()
		}
	}
	return deps
}

func newRunCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts runOptions
	fs.StringVar(&opts.File, "file", "", "Workflow file path (defaults to nearest .gpc/workflow.yml or .gpc/workflow.yaml)")
	fs.BoolVar(&opts.Confirm, "confirm", false, "Execute the workflow steps")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Plan the workflow without running any steps")
	fs.Var(&opts.Vars, "var", "Workflow variable override in key=value form (repeatable)")
	fs.StringVar(&opts.Output, "output", "", "Output format: json or table")

	return &ffcli.Command{
		Name:      "run",
		ShortHelp: "Execute or plan a workflow manifest",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, err := validateRunOptions(opts, deps.Getwd)
			if err != nil {
				return err
			}

			var wf manifest
			if err := shared.LoadManifest(opts.File, &wf); err != nil {
				return err
			}

			result, err := runWorkflow(ctx, deps, opts, wf)
			writeErr := writeResult(deps.Stdout, opts.Output, result)
			if err != nil {
				if writeErr != nil {
					return writeErr
				}
				return err
			}
			return writeErr
		},
	}
}

func validateRunOptions(opts runOptions, getwd func() (string, error)) (runOptions, error) {
	if opts.Confirm && opts.DryRun {
		return runOptions{}, shared.UsageErrorf("--confirm and --dry-run cannot be used together")
	}
	if !opts.Confirm && !opts.DryRun {
		return runOptions{}, shared.UsageErrorf("--confirm is required unless --dry-run is set")
	}

	path, err := resolveWorkflowPath(opts.File, getwd)
	if err != nil {
		return runOptions{}, err
	}
	opts.File = path

	output := strings.ToLower(strings.TrimSpace(shared.ResolveOutput(opts.Output)))
	switch output {
	case "json", "table":
		opts.Output = output
	default:
		return runOptions{}, shared.UsageErrorf("unsupported output format %q", strings.TrimSpace(output))
	}
	return opts, nil
}

func resolveWorkflowPath(localValue string, getwd func() (string, error)) (string, error) {
	if value := strings.TrimSpace(localValue); value != "" {
		return value, nil
	}
	if getwd == nil {
		getwd = os.Getwd
	}
	cwd, err := getwd()
	if err != nil {
		return "", err
	}
	dir, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	for {
		for _, name := range []string{
			filepath.Join(".gpc", "workflow.yml"),
			filepath.Join(".gpc", "workflow.yaml"),
		} {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", shared.UsageErrorf("--file is required or add .gpc/workflow.yml")
		}
		dir = parent
	}
}

func runWorkflow(ctx context.Context, deps Deps, opts runOptions, wf manifest) (workflowResult, error) {
	ordered, vars, err := planWorkflow(wf, opts.Vars)
	if err != nil {
		return workflowResult{}, err
	}

	result := workflowResult{
		File:         opts.File,
		Status:       "ok",
		StepCount:    len(ordered),
		Interpolated: vars,
		Steps:        make([]workflowStepState, 0, len(ordered)),
	}

	executablePath, err := deps.ExecutablePath()
	if err != nil {
		return workflowResult{}, fmt.Errorf("resolve gpc executable: %w", err)
	}

	if opts.DryRun {
		result.Status = "dry-run"
		for _, step := range ordered {
			resolved, _, err := interpolateStep(step.Run, vars)
			if err != nil {
				return workflowResult{}, err
			}
			result.Steps = append(result.Steps, workflowStepState{
				ID:       step.ID,
				Needs:    step.Needs,
				Status:   "planned",
				Resolved: resolved,
			})
		}
		return result, nil
	}

	for _, step := range ordered {
		resolved, args, err := interpolateStep(step.Run, vars)
		if err != nil {
			return workflowResult{}, err
		}
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		stepState := workflowStepState{
			ID:       step.ID,
			Needs:    step.Needs,
			Status:   "ok",
			Resolved: resolved,
		}
		if err := deps.RunCommand(ctx, executablePath, args, stdout, stderr); err != nil {
			stepState.Status = "error"
			stepState.Stdout = strings.TrimSpace(stdout.String())
			stepState.Stderr = strings.TrimSpace(stderr.String())
			result.Steps = append(result.Steps, stepState)
			result.Status = "failed"
			result.Executed = len(result.Steps) - 1
			return result, fmt.Errorf("workflow step %q failed: %w", step.ID, err)
		}
		stepState.Stdout = strings.TrimSpace(stdout.String())
		stepState.Stderr = strings.TrimSpace(stderr.String())
		result.Steps = append(result.Steps, stepState)
		result.Executed++
	}

	return result, nil
}

func planWorkflow(wf manifest, overrides []string) ([]manifestStep, map[string]string, error) {
	if wf.Version != 0 && wf.Version != 1 {
		return nil, nil, shared.UsageErrorf("workflow version must be 1")
	}
	if len(wf.Steps) == 0 {
		return nil, nil, shared.UsageErrorf("workflow must contain at least one step")
	}

	vars := make(map[string]string, len(wf.Vars)+len(overrides))
	for key, value := range wf.Vars {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			return nil, nil, shared.UsageErrorf("workflow vars must not contain empty keys")
		}
		vars[trimmed] = value
	}
	for _, raw := range overrides {
		key, value, ok := strings.Cut(raw, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, nil, shared.UsageErrorf("--var must use key=value form")
		}
		vars[key] = value
	}

	stepByID := make(map[string]manifestStep, len(wf.Steps))
	inDegree := make(map[string]int, len(wf.Steps))
	dependents := make(map[string][]string, len(wf.Steps))
	orderedIDs := make([]string, 0, len(wf.Steps))
	for _, step := range wf.Steps {
		step.ID = strings.TrimSpace(step.ID)
		step.Run = strings.TrimSpace(step.Run)
		if step.ID == "" {
			return nil, nil, shared.UsageErrorf("workflow step id is required")
		}
		if step.Run == "" {
			return nil, nil, shared.UsageErrorf("workflow step %q must define run", step.ID)
		}
		if _, exists := stepByID[step.ID]; exists {
			return nil, nil, shared.UsageErrorf("workflow step ids must be unique: %q", step.ID)
		}
		cleanNeeds := make([]string, 0, len(step.Needs))
		for _, need := range step.Needs {
			trimmed := strings.TrimSpace(need)
			if trimmed == "" {
				continue
			}
			if trimmed == step.ID {
				return nil, nil, shared.UsageErrorf("workflow step %q cannot depend on itself", step.ID)
			}
			cleanNeeds = append(cleanNeeds, trimmed)
		}
		step.Needs = cleanNeeds
		stepByID[step.ID] = step
		inDegree[step.ID] = len(step.Needs)
		orderedIDs = append(orderedIDs, step.ID)
	}

	for _, step := range wf.Steps {
		current := stepByID[strings.TrimSpace(step.ID)]
		for _, need := range current.Needs {
			if _, exists := stepByID[need]; !exists {
				return nil, nil, shared.UsageErrorf("workflow step %q depends on unknown step %q", current.ID, need)
			}
			dependents[need] = append(dependents[need], current.ID)
		}
	}

	queue := make([]string, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	ordered := make([]manifestStep, 0, len(wf.Steps))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ordered = append(ordered, stepByID[id])
		for _, dependent := range dependents[id] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}
	if len(ordered) != len(wf.Steps) {
		return nil, nil, shared.UsageErrorf("workflow contains cyclic or unresolved step dependencies")
	}
	return ordered, vars, nil
}

func interpolateStep(raw string, vars map[string]string) (string, []string, error) {
	matches := interpolationPattern.FindAllStringSubmatch(raw, -1)
	for _, match := range matches {
		key := match[1]
		if _, ok := vars[key]; !ok {
			return "", nil, shared.UsageErrorf("unknown workflow variable %q", key)
		}
	}
	resolved := interpolationPattern.ReplaceAllStringFunc(raw, func(match string) string {
		key := interpolationPattern.FindStringSubmatch(match)[1]
		return vars[key]
	})

	args, err := shlex.Split(resolved)
	if err != nil {
		return "", nil, shared.UsageErrorf("parse workflow command: %v", err)
	}
	if len(args) == 0 {
		return "", nil, shared.UsageErrorf("workflow step command must not be empty")
	}
	if args[0] == "gpc" {
		args = args[1:]
	}
	if len(args) == 0 {
		return "", nil, shared.UsageErrorf("workflow step command must include a gpc subcommand")
	}
	if args[0] == "workflow" {
		return "", nil, shared.UsageErrorf("workflow steps cannot invoke workflow commands recursively")
	}
	return resolved, args, nil
}

func writeResult(out io.Writer, output string, result workflowResult) error {
	switch output {
	case "", "json":
		return shared.WriteJSON(out, result)
	case "table":
		return writeTable(out, result)
	default:
		return shared.UsageErrorf("unsupported output format %q", output)
	}
}

func writeTable(out io.Writer, result workflowResult) error {
	if _, err := fmt.Fprintln(out, "STEP\tSTATUS\tNEEDS\tCOMMAND"); err != nil {
		return err
	}
	for _, step := range result.Steps {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", step.ID, step.Status, strings.Join(step.Needs, ","), step.Resolved); err != nil {
			return err
		}
	}
	return nil
}

type multiValueFlag []string

func (m *multiValueFlag) String() string {
	if m == nil {
		return ""
	}
	return strings.Join(*m, ",")
}

func (m *multiValueFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
