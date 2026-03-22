# Fastlane Diff Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `gpc migrate fastlane diff` to compare Fastlane metadata against live Play listings and live track release notes without importing anything first.

**Architecture:** Extend the existing `migrate fastlane` command tree with a sibling `diff` subcommand. Reuse the Fastlane scanning code already in `internal/cli/migrate`, then compare the scanned desired state against live Play data through the same edit-backed client flow used by `diff listing` and `diff track`.

**Tech Stack:** Go, `ffcli`, existing `internal/cli/migrate`, `internal/cli/diff`, `internal/cli/listing`, shared config/auth helpers, table-driven tests.

---

### Task 1: Add failing CLI tests for Fastlane diff

**Files:**
- Modify: `internal/cli/migrate/migrate_test.go`

**Step 1: Write the failing test**

Add tests that:
- run `gpc migrate fastlane diff`
- verify listing text and image drift are reported from Fastlane fixtures
- verify release note drift is reported for the requested track/version code
- verify `json`, `table`, `markdown`, and `yaml` outputs behave consistently enough for existing CLI expectations

**Step 2: Run test to verify it fails**

Run: `env GPC_BYPASS_KEYCHAIN=1 go test ./internal/cli/migrate -run FastlaneDiff -v`
Expected: FAIL because `fastlane diff` does not exist yet.

**Step 3: Commit**

No commit yet. Continue once failure is confirmed.

### Task 2: Implement the Fastlane diff command and comparison logic

**Files:**
- Modify: `internal/cli/migrate/migrate.go`

**Step 1: Add command wiring**

Add `fastlane diff` alongside `fastlane import`, with flags for:
- `--from-dir`
- `--package-name`
- `--track`
- `--version-code`
- `--output`

**Step 2: Add desired-state conversion**

Convert scanned `localeImport` values into:
- listing-like desired state for title, short description, full description, images
- normalized release-note desired state for locales with changelog text

**Step 3: Add live comparison flow**

Create one edit, fetch live listings and tracks, compare:
- listing locale creation/update/delete drift
- image replacement drift
- release-note drift on the target track’s current release

**Step 4: Add result rendering**

Return structured output with summary counts plus detailed changes.
Support `json`, `table`, `markdown`, and `yaml`.

**Step 5: Run focused tests**

Run: `env GPC_BYPASS_KEYCHAIN=1 go test ./internal/cli/migrate -run FastlaneDiff -v`
Expected: PASS

### Task 3: Update user-facing docs and generated command reference

**Files:**
- Modify: `README.md`
- Modify: `docs/COMMANDS.md`

**Step 1: Document the new command**

Add one concise README entry and example showing Fastlane diff usage.

**Step 2: Refresh generated command docs**

Run: `python3 scripts/generate-command-docs.py`

**Step 3: Verify docs are current**

Run: `python3 scripts/check-command-docs.py`
Expected: PASS

### Task 4: Run full verification and prepare merge

**Files:**
- Modify: `internal/cli/migrate/migrate.go`
- Modify: `internal/cli/migrate/migrate_test.go`
- Modify: `README.md`
- Modify: `docs/COMMANDS.md`

**Step 1: Run package tests**

Run: `env GPC_BYPASS_KEYCHAIN=1 go test ./internal/cli/migrate ./internal/cli/diff ./internal/cli/listing -v`
Expected: PASS

**Step 2: Run repo test suite**

Run: `env GPC_BYPASS_KEYCHAIN=1 make test`
Expected: PASS

**Step 3: Run formatter if needed**

Run: `gofmt -w internal/cli/migrate/migrate.go internal/cli/migrate/migrate_test.go`

**Step 4: Commit**

```bash
git add internal/cli/migrate/migrate.go internal/cli/migrate/migrate_test.go README.md docs/COMMANDS.md docs/plans/2026-03-22-fastlane-diff.md
git commit -m "feat: add fastlane diff command"
```
