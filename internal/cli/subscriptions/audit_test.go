package subscriptions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubscriptionsAuditReportsErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "monthly.json"), []byte(`{"productId":"monthly","basePlans":[{"basePlanId":"monthly"}],"listings":[{"languageCode":"en-US","title":"Monthly"}]}`), 0o600); err != nil {
		t.Fatalf("write monthly.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "duplicate.json"), []byte(`{"subscription":{"productId":"monthly","basePlans":[{"basePlanId":"monthly"}],"listings":[{"languageCode":"en-US","title":"Duplicate"}]},"regionsVersion":"2026/01"}`), 0o600); err != nil {
		t.Fatalf("write duplicate.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.json"), []byte(`{"subscription":`), 0o600); err != nil {
		t.Fatalf("write broken.json: %v", err)
	}

	out, err := runSubscriptions(t, Deps{}, "audit", "--dir", root, "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"status":"error"`, `duplicate productId`, `invalid subscription JSON payload`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestSubscriptionsAuditStrictFailsOnWarnings(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "raw-input.json"), []byte(`{"productId":"monthly","basePlans":[{"basePlanId":"monthly"}],"listings":[{"languageCode":"en-US","title":"Monthly"}]}`), 0o600); err != nil {
		t.Fatalf("write raw-input.json: %v", err)
	}

	out, err := runSubscriptions(t, Deps{}, "audit", "--dir", root, "--output", "json", "--strict")
	if err == nil || !strings.Contains(err.Error(), "subscriptions audit reported warn findings") {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"status":"warn"`, `regionsVersion wrapper is missing`, `filename does not match \u003cproductId\u003e.json`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}
