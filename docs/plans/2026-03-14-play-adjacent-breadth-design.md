# Play-Adjacent Breadth Expansion Design

Date: 2026-03-14

## Context

`main` now includes PR 84 and remains complete for Android Publisher discovery coverage: `136/136` implemented endpoints in `docs/openapi/COVERAGE.md`.

That means the remaining raw-breadth gap versus `gpd` is not in Android Publisher itself. The gap is in adjacent official Google Play APIs and a few extra reporting surfaces:

- Play Integrity API
- Play Custom App Publishing API
- Games Management / Play Games Services
- Extra Play Developer Reporting endpoints, especially error counts

Current `gpc` strengths should not be diluted while closing that gap:

- explicit long flags
- automation-safe CLI behavior
- stable JSON output
- no hidden prompts in core flows
- strong Android Publisher coverage and repo-managed workflows

## Options

### Option A: Targeted Play-adjacent expansion

Add only the highest-value official APIs that fit `gpc`:

- `integrity decode`
- `custom-apps create`
- reporting error counts

Defer Games Management and Play Games Services until there is real user demand.

Pros:

- closes the most credible breadth gap
- keeps the CLI focused on app publishing and operations
- adds surfaces with obvious operator value

Cons:

- `gpd` can still claim broader total surface area because of games-related APIs

### Option B: Full breadth parity chase

Add everything `gpd` exposes, including:

- Games Management
- Play Games Services grouping tokens
- compatibility aliases for analytics/vitals style entrypoints

Pros:

- strongest raw breadth story
- easiest marketing comparison

Cons:

- weak product fit
- much larger test and docs burden
- introduces low-usage surfaces that make the CLI noisier

### Option C: Alias-first parity

Keep current API coverage mostly unchanged and add command aliases or grouping reshapes that look closer to `gpd`.

Pros:

- low implementation cost

Cons:

- does not materially improve real API breadth
- mostly cosmetic

## Recommendation

Choose Option A.

`gpc` should become broader by adding official Play-adjacent APIs with clear operational value, not by absorbing every niche Google Play surface. The next work should make `gpc` stronger where breadth matters to real publishing teams, while explicitly deferring low-fit APIs.

## Scope

### In scope

- Play Integrity token decoding
- Play Custom App Publishing creation workflow
- Reporting error counts get/query
- docs, tests, CI, and command generation for the new families

### Out of scope for the next wave

- Games Management API
- Play Games Services grouping tokens
- unofficial browser automation or Play Console scraping
- broad command aliasing just to mirror `gpd`

## Public CLI Design

### 1. New `integrity` command family

Initial command:

```bash
gpc integrity decode --token <jwt>
gpc integrity decode --input <file>|-
```

Behavior:

- exactly one of `--token` or `--input`
- decode only; no token generation helpers
- JSON output by default for non-TTY
- usage errors for malformed input

Implementation shape:

- `internal/gpc/integrity.go`
- `internal/cli/integrity/...`

### 2. New `custom-apps` command family

Initial command:

```bash
gpc custom-apps create --developer-id <id> --input <json>|-
```

Optional follow-up commands after create lands cleanly:

```bash
gpc custom-apps list --developer-id <id>
gpc custom-apps get --name <resource-name>
```

Behavior:

- reuse existing auth and `--developer-id` resolution
- file/stdin JSON input like other structured commands
- explicit validation before API calls

Implementation shape:

- `internal/gpc/customapps.go`
- `internal/cli/customapps/...`

### 3. Expand `reports errors`

Add:

```bash
gpc reports errors counts get --package-name <pkg> ...
gpc reports errors counts query --package-name <pkg> --input <json>|-
```

This keeps reporting breadth inside the existing namespace instead of creating an `analytics` family just for parity.

## Delivery Phases

### Phase 1: Reporting counts + Integrity

Reason:

- smallest diff from current architecture
- highest signal improvement for breadth
- easy to explain in README and release notes

Deliverables:

- `reports errors counts get/query`
- `integrity decode`
- tests, docs, command reference regeneration

### Phase 2: Custom App Publishing

Reason:

- broadens the CLI into another official Play surface
- still aligned with enterprise publishing use cases

Deliverables:

- `custom-apps create`
- optional `list` if the API and workflow feel clean enough

### Phase 3: Decide on Games surfaces explicitly

Decision gate:

- only proceed if users actually need games resets or grouping tokens
- otherwise keep this deferred and documented as intentionally unsupported

If accepted later:

- start with one thin family
- keep it clearly separated from core publishing commands

## Repo Design Constraints

- no new credential model
- no output envelope redesign just to mimic `gpd`
- no hidden interactivity
- maintain current TTY-aware output defaults
- keep docs generated from real command help

## Testing Plan

- unit tests for each new `internal/gpc` client surface
- CLI tests for required flags, stdin/file input, and JSON output
- docs sync checks for `docs/COMMANDS.md`
- OpenAPI or API inventory docs updated where applicable
- keep Windows CI coverage because Windows binaries are still released

## Success Criteria

The next breadth wave is successful if:

- `gpc` adds at least two official Play-adjacent API families not currently present
- reporting breadth closes the remaining obvious gap with `gpd`
- the CLI remains coherent and publishing-focused
- CI stays stable across macOS, Linux, and Windows

## Immediate Next Branch

Recommended branch:

`codex/play-adjacent-breadth`

Recommended first implementation slice:

1. `reports errors counts get/query`
2. `integrity decode`
3. `custom-apps create`
