package cmd

import "testing"

func TestVersionPayload(t *testing.T) {
	prevVersion, prevCommit, prevDate := Version, Commit, Date
	defer func() {
		Version, Commit, Date = prevVersion, prevCommit, prevDate
	}()

	Version = "1.2.3"
	Commit = "abc123"
	Date = "2026-03-01T12:00:00Z"

	got := versionPayload()
	if got.Version != "1.2.3" || got.Commit != "abc123" || got.Date != "2026-03-01T12:00:00Z" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}
