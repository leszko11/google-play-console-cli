package completion

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type Deps struct {
	LoadConfig  func() (config.Config, error)
	LoadProject func() (config.ProjectConfigInfo, error)
	Stdout      io.Writer
}

var knownTracks = []string{"alpha", "beta", "internal", "production"}

func NewCommand(root *ffcli.Command, deps ...Deps) *ffcli.Command {
	cfg := Deps{}
	if len(deps) > 0 {
		cfg = deps[0]
	}
	cfg = withDefaults(cfg)

	return &ffcli.Command{
		Name:      "completion",
		ShortHelp: "Generate shell completion script",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			shellCommand("bash", func() string { return bashScript(root) }, cfg.Stdout),
			shellCommand("zsh", func() string { return zshScript(root) }, cfg.Stdout),
			shellCommand("fish", func() string { return fishScript(root) }, cfg.Stdout),
			newValuesCommand(cfg),
		},
	}
}

func withDefaults(deps Deps) Deps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.LoadProject == nil {
		deps.LoadProject = config.LoadProject
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	return deps
}

func shellCommand(name string, render func() string, out io.Writer) *ffcli.Command {
	return &ffcli.Command{
		Name:      name,
		ShortHelp: fmt.Sprintf("Print %s completion script", name),
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			_, err := io.WriteString(out, render())
			return err
		},
	}
}

func newValuesCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("values", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var flagName string
	fs.StringVar(&flagName, "flag", "", "Flag name to resolve completion values for")

	return &ffcli.Command{
		Name:      "values",
		ShortHelp: "Print dynamic completion values",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			values, err := resolveValues(strings.TrimSpace(flagName), deps)
			if err != nil {
				return err
			}
			for _, value := range values {
				if _, err := fmt.Fprintln(deps.Stdout, value); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func resolveValues(flagName string, deps Deps) ([]string, error) {
	switch strings.TrimPrefix(strings.ToLower(strings.TrimSpace(flagName)), "--") {
	case "package-name":
		return packageValues(deps)
	case "track":
		return trackValues(deps)
	default:
		return nil, shared.UsageErrorf("unsupported completion flag %q", flagName)
	}
}

func packageValues(deps Deps) ([]string, error) {
	cfg, err := deps.LoadConfig()
	if err != nil {
		return nil, err
	}
	values := make([]string, 0, len(cfg.Packages)+1)
	values = append(values, cfg.Packages...)

	project, err := deps.LoadProject()
	if err != nil {
		return nil, err
	}
	if pkg := strings.TrimSpace(project.Config.PackageName); pkg != "" {
		values = append(values, pkg)
	}

	return uniqueSorted(values), nil
}

func trackValues(deps Deps) ([]string, error) {
	values := append([]string{}, knownTracks...)
	project, err := deps.LoadProject()
	if err != nil {
		return nil, err
	}
	if track := strings.TrimSpace(project.Config.DefaultTrack); track != "" {
		values = append(values, track)
	}
	return uniqueSorted(values), nil
}

func uniqueSorted(items []string) []string {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}

	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
