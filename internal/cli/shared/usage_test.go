package shared

import (
	"errors"
	"flag"
	"testing"
)

func TestIsUsageError(t *testing.T) {
	if !IsUsageError(UsageErrorf("--flag is required")) {
		t.Fatal("expected usage error to be detected")
	}
	if !IsUsageError(flag.ErrHelp) {
		t.Fatal("expected flag.ErrHelp to be treated as usage")
	}
	if !IsLikelyUsageError(errors.New("flag provided but not defined: -unknown")) {
		t.Fatal("expected unknown-flag parse error to be treated as usage")
	}
	if IsUsageError(errors.New("runtime failure")) {
		t.Fatal("did not expect runtime error to be treated as usage")
	}
}
