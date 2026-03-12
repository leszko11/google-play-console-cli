package registry

import (
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestRegisterAddsCoreCommands(t *testing.T) {
	root := &ffcli.Command{Name: "gpc"}
	Register(root, Deps{})

	got := map[string]bool{}
	for _, c := range root.Subcommands {
		got[c.Name] = true
	}

	for _, name := range []string{"auth", "apps", "edits", "tracks", "apks", "bundles", "deobfuscation", "deploy", "doctor", "e2e", "release", "rollback", "reviews", "orders", "external-transactions", "device-tier-configs", "system-apks", "generated-apks", "app-recoveries", "subscriptions", "purchases", "completion"} {
		if !got[name] {
			t.Fatalf("expected subcommand %q, got %#v", name, got)
		}
	}
}
