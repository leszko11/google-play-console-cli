package completion

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestScriptsIncludeRegisteredCommandsAndFlags(t *testing.T) {
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	fs.String("fields", "", "projection")
	root := &ffcli.Command{
		Name:    "gpc",
		FlagSet: fs,
		Subcommands: []*ffcli.Command{
			{
				Name: "appinit",
				Subcommands: []*ffcli.Command{
					{Name: "export"},
				},
			},
			{
				Name: "products",
				Subcommands: []*ffcli.Command{
					{Name: "sync"},
				},
			},
			{
				Name: "reports",
				Subcommands: []*ffcli.Command{
					{
						Name: "errors",
						Subcommands: []*ffcli.Command{
							{Name: "issues"},
						},
					},
				},
			},
		},
	}

	for name, script := range map[string]string{
		"bash": bashScript(root),
		"zsh":  zshScript(root),
		"fish": fishScript(root),
	} {
		wants := []string{"appinit", "export", "products", "sync", "reports", "errors"}
		if name == "fish" {
			wants = append(wants, "-l fields", "_gpc_completion_values package-name", "_gpc_completion_values track")
		} else {
			wants = append(wants, "--fields", "gpc completion values --flag", "--package-name", "--track")
		}
		for _, want := range wants {
			if !strings.Contains(script, want) {
				t.Fatalf("%s script missing %q:\n%s", name, want, script)
			}
		}
	}
}

func TestScriptsSkipFlagValuesWhileWalkingPath(t *testing.T) {
	fs := flag.NewFlagSet("gpc", flag.ContinueOnError)
	root := &ffcli.Command{
		Name:    "gpc",
		FlagSet: fs,
		Subcommands: []*ffcli.Command{
			{
				Name: "tracks",
				Subcommands: []*ffcli.Command{
					func() *ffcli.Command {
						getFS := flag.NewFlagSet("get", flag.ContinueOnError)
						getFS.String("package-name", "", "Package name")
						getFS.String("track", "", "Track name")
						return &ffcli.Command{Name: "get", FlagSet: getFS}
					}(),
				},
			},
		},
	}

	bash := bashScript(root)
	if !strings.Contains(bash, "\"tracks get|--package-name\") i=$((i+2)); continue ;;") {
		t.Fatalf("bash script missing value-skip case:\n%s", bash)
	}
	zsh := zshScript(root)
	if !strings.Contains(zsh, "\"tracks get|--package-name\") ((i+=2)); continue ;;") {
		t.Fatalf("zsh script missing value-skip case:\n%s", zsh)
	}
}

func TestCompletionValuesPackageNameIncludesConfigAndProject(t *testing.T) {
	var out bytes.Buffer
	cmd := NewCommand(nil, Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{Packages: []string{"com.example.app", "com.example.other"}}, nil
		},
		LoadProject: func() (config.ProjectConfigInfo, error) {
			return config.ProjectConfigInfo{Config: config.ProjectConfig{PackageName: "com.example.app"}}, nil
		},
		Stdout: &out,
	})

	if err := cmd.ParseAndRun(context.Background(), []string{"values", "--flag", "package-name"}); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	want := "com.example.app\ncom.example.other\n"
	if out.String() != want {
		t.Fatalf("unexpected output %q, want %q", out.String(), want)
	}
}

func TestCompletionValuesTrackIncludesKnownTracksAndProjectDefault(t *testing.T) {
	var out bytes.Buffer
	cmd := NewCommand(nil, Deps{
		LoadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		LoadProject: func() (config.ProjectConfigInfo, error) {
			return config.ProjectConfigInfo{Config: config.ProjectConfig{DefaultTrack: "qa"}}, nil
		},
		Stdout: &out,
	})

	if err := cmd.ParseAndRun(context.Background(), []string{"values", "--flag", "track"}); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	want := "alpha\nbeta\ninternal\nproduction\nqa\n"
	if out.String() != want {
		t.Fatalf("unexpected output %q, want %q", out.String(), want)
	}
}

func TestCompletionValuesRejectsUnsupportedFlag(t *testing.T) {
	cmd := NewCommand(nil, Deps{
		LoadConfig: func() (config.Config, error) { return config.Config{}, nil },
		LoadProject: func() (config.ProjectConfigInfo, error) {
			return config.ProjectConfigInfo{}, nil
		},
		Stdout: &bytes.Buffer{},
	})

	err := cmd.ParseAndRun(context.Background(), []string{"values", "--flag", "output"})
	if err == nil || !strings.Contains(err.Error(), "unsupported completion flag") {
		t.Fatalf("unexpected error: %v", err)
	}
}
