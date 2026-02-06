package cmd

import "testing"

func TestNewRootCommand(t *testing.T) {
	cmd := newRootCommand()
	if cmd.Name != "gpc" {
		t.Fatalf("expected gpc, got %q", cmd.Name)
	}
}
