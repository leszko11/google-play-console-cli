# Draft-App Track Auto-Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an explicit opt-in path that rewrites the internal track release from `completed` to `draft` when Play rejects a draft-app commit for that exact reason.

**Architecture:** Keep the mutation logic centralized in `internal/cli/shared/edit_commit.go`, behind an explicit `AutoFixDraftTrack` option. Reuse the existing commit fallback flow, inspect the `internal` track only when the draft-app error occurs, rewrite that release to `draft`, then retry the commit. Expose the opt-in flag only on the metadata/edit commit commands that already share this fallback path.

**Tech Stack:** Go, `ffcli`, existing shared edit commit helper, command-specific JSON outputs, generated command docs.

---

### Task 1: Add failing shared helper and command tests

**Files:**
- Modify: `internal/cli/shared/edit_commit_test.go`
- Modify: `internal/cli/edits/edits_test.go`

**Step 1: Write the failing tests**

Cover:
- direct draft-app auto-fix and retry success
- auto-fix after the `changesNotSentForReview=true` retry path
- auto-fix failure messaging
- `gpc edits commit --auto-fix-draft-track` output

**Step 2: Run focused tests**

Run: `env GPC_BYPASS_KEYCHAIN=1 go test ./internal/cli/shared ./internal/cli/edits -run 'CommitEditWith|EditsCommit' -v`
Expected: FAIL before implementation, PASS after implementation.

### Task 2: Implement shared auto-fix logic and wire flags

**Files:**
- Modify: `internal/cli/shared/edit_commit.go`
- Modify: `internal/cli/edits/edits.go`
- Modify: `internal/cli/listing/sync.go`
- Modify: `internal/cli/screenshots/screenshots.go`
- Modify: `internal/cli/changelog/sync.go`

**Step 1: Extend the shared helper**

Add an options struct with `AutoFixDraftTrack`.

**Step 2: Add the internal-track rewrite path**

Only when:
- the commit error matches the draft-app conflict
- the caller enabled auto-fix
- the client can inspect/update tracks
- the `internal` track has exactly one `completed` release with version codes

**Step 3: Retry commit after fix**

Preserve the existing `changesNotSentForReview` fallback semantics and retry with the current resolved review flag.

**Step 4: Surface the explicit flag**

Add `--auto-fix-draft-track` to the commands above and include `draftTrackAutoFixed` in JSON output when used.

### Task 3: Refresh docs and verify

**Files:**
- Modify: `docs/COMMANDS.md`

**Step 1: Run package tests**

Run: `env GPC_BYPASS_KEYCHAIN=1 go test ./internal/cli/shared ./internal/cli/edits ./internal/cli/listing ./internal/cli/screenshots ./internal/cli/changelog -v`

**Step 2: Run full suite**

Run: `env GPC_BYPASS_KEYCHAIN=1 make test`

**Step 3: Refresh generated command docs**

Run: `GPC_BIN=./build/gpc python3 scripts/generate-command-docs.py`

**Step 4: Check docs**

Run: `python3 scripts/check-command-docs.py`

**Step 5: Commit**

```bash
git add internal/cli/shared/edit_commit.go internal/cli/shared/edit_commit_test.go internal/cli/edits/edits.go internal/cli/edits/edits_test.go internal/cli/listing/sync.go internal/cli/screenshots/screenshots.go internal/cli/changelog/sync.go docs/COMMANDS.md docs/plans/2026-03-22-auto-fix-draft-app-tracks.md
git commit -m "feat: add draft app track auto-fix"
```
