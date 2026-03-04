# Google Play Console CLI Phase 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Deliver a working `gpc` binary with credential setup and basic app visibility commands (`apps list`, `apps get`) plus CI workflows that enforce formatting, linting, tests, and build health.

**Architecture:** Use a thin `ffcli` command layer over internal services (`auth`, `config`, `gpc client`) with strict dependency flow: CLI -> domain service -> API adapter. Keep Phase 1 limited to auth + app discovery primitives, with all logic testable via table-driven unit tests and `httptest`. Because Google Play Developer API does not expose a first-class “list all apps” endpoint, Phase 1 `apps list` will list configured package names and optionally verify access per package.

**Tech Stack:** Go 1.24+, `github.com/peterbourgon/ff/v4`, `google.golang.org/api/androidpublisher/v3`, `go test`, GitHub Actions.

---

## Preconditions

- Use `@using-git-worktrees` before implementation work.
- Execute this plan with `@executing-plans`.
- Keep commits small and task-scoped.

### Task 1: Rewrite README With Scope + Inspiration

**Files:**
- Modify: `README.md`

**Step 1: Write failing content check**

Run: `grep -q "Inspired by Rudrank" README.md`  
Expected: command exits with code `1` (text not present yet).

**Step 2: Replace README with Phase 1-ready documentation**

Include:
- Project purpose and non-goals for Phase 1
- Install/build quickstart
- Minimal command examples (`auth init`, `auth status`, `apps list`, `apps get`)
- Explicit inspiration credit to [Rudrank’s App-Store-Connect-CLI](https://github.com/rudrankriyam/App-Store-Connect-CLI)

Minimal skeleton to include:

```markdown
# gpc (Google Play Console CLI)

Inspired by Rudrank Riyaam's App-Store-Connect-CLI.

## Phase 1 Scope
- Authentication bootstrap
- App visibility commands
- CI-backed quality gates
```

**Step 3: Verify README requirements**

Run: `grep -n "Inspired by Rudrank" README.md`  
Expected: one matching line is printed.

**Step 4: Run markdown sanity check**

Run: `awk 'NF{p=1} END{exit !p}' README.md`  
Expected: exit code `0`.

**Step 5: Commit**

```bash
git add README.md
git commit -m "docs: rewrite readme and credit project inspiration"
```

### Task 2: Add AGENTS + CLAUDE Hand-off Docs

**Files:**
- Create: `AGENTS.md`
- Create: `CLAUDE.md`

**Step 1: Write failing file existence check**

Run: `test -f AGENTS.md && test -f CLAUDE.md`  
Expected: exit code `1` (files missing).

**Step 2: Create `AGENTS.md`**

Add concise contributor guardrails:
- Core principles (explicit flags, JSON-first, no prompts, pagination)
- Command discovery examples
- Build/test commands
- Testing discipline (TDD + table tests + stderr assertions)
- Security note for service account credentials

**Step 3: Create `CLAUDE.md`**

Set content to:

```markdown
@AGENTS.md
```

**Step 4: Verify file contents**

Run: `grep -n "^@AGENTS.md$" CLAUDE.md`  
Expected: exact one-line match.

**Step 5: Commit**

```bash
git add AGENTS.md CLAUDE.md
git commit -m "docs: add agent operating instructions"
```

### Task 3: Scaffold Go Module + Root Command Skeleton

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/errors.go`
- Create: `cmd/root.go`
- Create: `cmd/run.go`
- Create: `cmd/shared.go`
- Create: `cmd/root_test.go`
- Create: `internal/cli/registry/registry.go`

**Step 1: Write failing root command test**

`cmd/root_test.go`:

```go
func TestNewRootCommand(t *testing.T) {
	cmd := newRootCommand()
	if cmd.Name != "gpc" {
		t.Fatalf("expected gpc, got %q", cmd.Name)
	}
}
```

**Step 2: Run test to verify failure**

Run: `go test ./cmd -run TestNewRootCommand -v`  
Expected: FAIL (`undefined: newRootCommand`).

**Step 3: Implement minimal command skeleton**

`cmd/root.go` minimal target:

```go
func newRootCommand() *ffcli.Command {
	return &ffcli.Command{Name: "gpc", ShortHelp: "Google Play Console CLI"}
}
```

Wire `main.go` -> `cmd.Run(os.Args[1:])`.

**Step 4: Re-run tests**

Run: `go test ./cmd -run TestNewRootCommand -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add go.mod main.go cmd internal/cli/registry/registry.go
git commit -m "feat: scaffold module and root command"
```

### Task 4: Add Shared CLI Utilities (Flags, Output, Timeout)

**Files:**
- Create: `internal/cli/shared/flags.go`
- Create: `internal/cli/shared/output.go`
- Create: `internal/cli/shared/context.go`
- Create: `internal/cli/shared/usage.go`
- Create: `internal/cli/shared/output_test.go`
- Create: `internal/cli/shared/context_test.go`

**Step 1: Write failing output formatting tests**

`internal/cli/shared/output_test.go`:

```go
func TestRenderJSON_MinifiedByDefault(t *testing.T) {
	out, err := RenderJSON(map[string]any{"a": 1}, false)
	if err != nil { t.Fatal(err) }
	if string(out) != "{\"a\":1}\n" {
		t.Fatalf("unexpected output: %q", out)
	}
}
```

**Step 2: Run tests and confirm failure**

Run: `go test ./internal/cli/shared -run RenderJSON -v`  
Expected: FAIL (`undefined: RenderJSON`).

**Step 3: Implement minimal utilities**

Provide:
- `RenderJSON(v any, pretty bool) ([]byte, error)`
- timeout helpers for standard and upload contexts
- common global flag binding helpers (`--package-name`, `--output`, `--timeout`, `--debug`)

**Step 4: Re-run shared tests**

Run: `go test ./internal/cli/shared -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/shared
git commit -m "feat: add shared cli format and context helpers"
```

### Task 5: Implement Config + Credential Resolution Core

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/auth/auth.go`
- Create: `internal/auth/auth_test.go`

**Step 1: Write failing precedence tests**

`internal/auth/auth_test.go` should verify:
- flag path beats env path
- env beats config
- strict mode fails when multiple sources are set

Example:

```go
func TestResolveCredentialSource_StrictModeConflict(t *testing.T) {
	_, err := ResolveCredentialSource(Input{
		FlagPath: "/tmp/a.json",
		EnvPath:  "/tmp/b.json",
		Strict:   true,
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}
```

**Step 2: Run tests and verify failure**

Run: `go test ./internal/auth -v`  
Expected: FAIL (`undefined: ResolveCredentialSource`).

**Step 3: Implement minimal auth + config**

Implement:
- config load/save at `~/.gpc/config.json` (and override via `GPC_CONFIG_PATH`)
- credential source resolver with deterministic precedence
- lightweight credential metadata persistence (path, profile, last validated timestamp)

**Step 4: Re-run auth/config tests**

Run: `go test ./internal/auth ./internal/config -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/auth internal/config
git commit -m "feat: add config and credential source resolution"
```

### Task 6: Build Android Publisher Client Adapter + Access Verification

**Files:**
- Create: `internal/gpc/client.go`
- Create: `internal/gpc/apps.go`
- Create: `internal/gpc/types.go`
- Create: `internal/gpc/client_test.go`

**Step 1: Write failing API adapter tests**

`internal/gpc/client_test.go` should verify:
- client creation rejects missing credentials
- package access check returns actionable error text for `403` and `404`

Example:

```go
func TestVerifyPackageAccess_MapsForbidden(t *testing.T) {
	err := mapAPIError(http.StatusForbidden, "forbidden")
	if !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("expected ErrAccessDenied, got %v", err)
	}
}
```

**Step 2: Run test and verify failure**

Run: `go test ./internal/gpc -v`  
Expected: FAIL (`undefined: mapAPIError`).

**Step 3: Implement minimal adapter**

Implement:
- `NewClient(ctx context.Context, creds CredentialInput, opts ...option.ClientOption) (*Client, error)`
- `VerifyPackageAccess(ctx context.Context, packageName string) error`
- `GetApp(ctx context.Context, packageName string) (AppInfo, error)`

Note: There is no direct Android Publisher endpoint for listing all apps. Keep listing logic in CLI/config layer.

**Step 4: Re-run tests**

Run: `go test ./internal/gpc -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/gpc
git commit -m "feat: add android publisher client adapter"
```

### Task 7: Implement `auth` Commands (init/status/logout/switch)

**Files:**
- Create: `internal/cli/auth/init.go`
- Create: `internal/cli/auth/status.go`
- Create: `internal/cli/auth/logout.go`
- Create: `internal/cli/auth/switch.go`
- Create: `internal/cli/auth/auth_test.go`

**Step 1: Write failing command tests**

`internal/cli/auth/auth_test.go` should cover:
- `auth init --service-account` stores profile metadata
- `auth status` prints active profile JSON
- `auth logout` removes active profile

**Step 2: Run tests to verify failure**

Run: `go test ./internal/cli/auth -v`  
Expected: FAIL (missing command constructors).

**Step 3: Implement minimal command handlers**

Add constructors:
- `func NewInitCommand(deps Deps) *ffcli.Command`
- `func NewStatusCommand(deps Deps) *ffcli.Command`
- `func NewSwitchCommand(deps Deps) *ffcli.Command`
- `func NewLogoutCommand(deps Deps) *ffcli.Command`

Make `auth init` call `VerifyPackageAccess` only when `--package-name` is supplied.

**Step 4: Re-run auth CLI tests**

Run: `go test ./internal/cli/auth -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/auth
git commit -m "feat: implement auth command set"
```

### Task 8: Implement `apps` Commands for Basic Visibility

**Files:**
- Create: `internal/cli/apps/list.go`
- Create: `internal/cli/apps/get.go`
- Create: `internal/cli/apps/apps_test.go`
- Modify: `internal/config/config.go`

**Step 1: Write failing app command tests**

`internal/cli/apps/apps_test.go` should verify:
- `apps list` returns configured package names (JSON by default)
- `apps list --verify` checks access and includes status field per package
- `apps get --package-name com.example.app` returns app info or clear API error

**Step 2: Run tests to verify failure**

Run: `go test ./internal/cli/apps -v`  
Expected: FAIL (missing commands and config fields).

**Step 3: Implement minimal commands + config additions**

Add config shape:

```go
type Config struct {
	ActiveProfile string   `json:"activeProfile"`
	Packages      []string `json:"packages"`
}
```

Implement:
- `apps list` (config-backed listing)
- `apps get` (remote lookup for explicit package)
- output via shared renderer (`json`, `table`, `markdown`)

**Step 4: Re-run tests**

Run: `go test ./internal/cli/apps -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/apps internal/config/config.go
git commit -m "feat: add apps list/get phase1 commands"
```

### Task 9: Register Commands + Add Shell Completion

**Files:**
- Modify: `internal/cli/registry/registry.go`
- Create: `internal/cli/completion/completion.go`
- Create: `internal/cli/registry/registry_test.go`

**Step 1: Write failing registration test**

Test that root registers:
- `auth`
- `apps`
- `completion`

**Step 2: Run test and verify failure**

Run: `go test ./internal/cli/registry -v`  
Expected: FAIL (incomplete registry).

**Step 3: Implement registration + completion command**

Wire commands in deterministic order and add `completion {bash|zsh|fish}` output.

**Step 4: Re-run registry tests**

Run: `go test ./internal/cli/registry -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/registry internal/cli/completion
git commit -m "feat: register phase1 commands and shell completion"
```

### Task 10: Add Testing Workflows + Quality Gates

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`
- Create: `Makefile`

**Step 1: Write failing workflow structure checks**

Run:

```bash
test -f .github/workflows/ci.yml && test -f .github/workflows/release.yml && test -f Makefile
```

Expected: exit code `1` (missing files).

**Step 2: Implement required workflows**

`ci.yml`:
- triggers: push + pull_request on `main`
- matrix: `ubuntu-latest`, `macos-latest`
- steps: checkout -> setup-go -> `go test ./...` -> `go build ./...`

`release.yml`:
- trigger: semver tag push
- dry-run build job initially (no publishing in Phase 1)

`Makefile`:
- `build`, `test`, `lint`, `format`, `dev`

**Step 3: Verify local quality gate commands**

Run:
- `make test`
- `make build`

Expected: both commands exit with code `0`.

**Step 4: Verify workflow YAML contains required triggers**

Run: `grep -n "pull_request" .github/workflows/ci.yml`  
Expected: one matching line.

**Step 5: Commit**

```bash
git add .github/workflows Makefile
git commit -m "ci: add phase1 test and build workflows"
```

### Task 11: Phase 1 End-to-End Verification + Release Notes Stub

**Files:**
- Create: `docs/TESTING.md`
- Modify: `README.md`

**Step 1: Write failing smoke command checklist test**

Run: `grep -q "gpc auth init" docs/TESTING.md`  
Expected: exit code `1`.

**Step 2: Add `docs/TESTING.md` with executable checks**

Include:

```bash
go run . --help
go run . auth status
go run . apps list --output json
go run . apps get --package-name com.example.app --output json
```

Document expected outcomes for no-auth, invalid-auth, and valid-auth scenarios.

**Step 3: Re-run doc check**

Run: `grep -n "gpc auth init" docs/TESTING.md`  
Expected: one matching line.

**Step 4: Run full local verification**

Run: `make dev`  
Expected: format/lint/tests/build all pass.

**Step 5: Commit**

```bash
git add docs/TESTING.md README.md
git commit -m "docs: add phase1 testing guide and smoke commands"
```

---

## Definition of Done (Phase 1)

- `go run . --help` prints root command with `auth`, `apps`, and `completion`.
- `gpc auth init --service-account <path>` stores profile metadata.
- `gpc apps list` returns configured packages; `--verify` checks access.
- `gpc apps get --package-name <pkg>` performs remote lookup with clear errors.
- `make test` and `make build` pass locally.
- GitHub Actions CI runs on PRs and pushes with test/build jobs.

## Risks to Re-check During Execution

- Android Publisher API credentials and package access can fail with ambiguous `403`; keep user-facing error mapping explicit.
- There is no single endpoint for “all apps visible to this service account”; do not overpromise this behavior in docs/help.
- Keep Phase 1 narrow; defer edits/upload flows to Phase 2.
