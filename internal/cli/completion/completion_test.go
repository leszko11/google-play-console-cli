package completion

import (
	"flag"
	"strings"
	"testing"

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
			wants = append(wants, "-l fields")
		} else {
			wants = append(wants, "--fields")
		}
		for _, want := range wants {
			if !strings.Contains(script, want) {
				t.Fatalf("%s script missing %q:\n%s", name, want, script)
			}
		}
	}
}
