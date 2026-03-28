# Authentication Model

`gpc` supports profile-based authentication for multiple identities.

## Credential Storage

- Profile metadata lives in `~/.gpc/config.json` with `serviceAccountPath`, `storage`, `lastValidatedAt`, and optional `developerId`.
- Managed path backend stores service-account JSON in `~/.gpc/credentials/<profile>.json`.
- Keychain backend stores service-account JSON in system keychain entry `gpc:credential:<profile>`.
- `config.json` never stores the full service-account JSON payload.
- Backward compatibility: existing path-only profiles continue to work; no destructive migration.

Supported storage backends:

- `path`: use `serviceAccountPath` from config as the active credential source.
- `keychain`: use keychain as the active credential source for that profile.
- `auto`: input mode for `auth init` / `setup`; currently resolves to managed path storage for new profiles.

## Profile Selection

Profile resolution order:

1. Global `--profile`
2. `activeProfile` in config

Use:

```bash
gpc auth init --profile work --service-account /path/work-sa.json
gpc auth init --profile work --service-account /path/work-sa.json --storage keychain
gpc auth init --profile personal --service-account /path/personal-sa.json
gpc auth switch --profile work
gpc --profile personal auth status --output json
gpc auth profiles list --output table
```

## Credential Source Resolution

Credential source precedence:

1. `--service-account` flag
2. `GPC_SERVICE_ACCOUNT_PATH`
3. persisted profile backend

For explicit storage profiles, `gpc` does not probe the other backend during normal resolution.
Legacy profiles without `storage` keep the previous behavior:

1. `--service-account` flag
2. `GPC_SERVICE_ACCOUNT_PATH`
3. keychain credential for resolved profile
4. profile `serviceAccountPath` from config

`gpc auth status` reports:

- `source`: resolved credential source (`flag`, `env`, `keychain`, `config`)
- `storageBackend`: effective backend used for the current resolution (`keychain` or `path`)
- `profileStorage`: persisted per-profile storage (`path` or `keychain`) when configured
- `managedCredentialPath`: managed `~/.gpc/credentials/...` path when relevant
- `warnings`: fallback/bypass/unavailable notes

Use `gpc auth explain` when you need the full resolution chain instead of the summarized health view. It is especially useful in CI and shared shells where config, env, and bypass flags can mix:

```bash
gpc auth explain --output table
gpc auth explain --output json
```

`gpc auth explain` includes:

- selected and active profile
- config path in use
- env override presence
- keychain availability and bypass state
- persisted profile storage mode
- final credential source selection
- mixed-source risk
- strict-auth failure risk
- a CI-safe recommended invocation

`authenticated=true` is only returned when credentials are locally valid:

- keychain source: JSON exists and parses
- path source: file exists, is readable, and contains valid JSON

## Strict Source Policy

Enable strict source resolution to prevent ambiguous source mixing:

- Global flag: `--strict-auth`
- Env: `GPC_STRICT_AUTH=1`

When strict mode is enabled and multiple credential sources are present, command execution fails.

## Keychain Controls

- `GPC_BYPASS_KEYCHAIN=1` disables keychain reads/writes for the current process.
- For `path` profiles, bypass is effectively a no-op because those profiles do not use keychain during normal resolution.
- On unsupported systems or unavailable keychain backend, explicit `keychain` profiles can fall back to their saved `serviceAccountPath` metadata and report warnings.
- If a local command appears to stall during auth resolution on macOS, rerun once with `GPC_BYPASS_KEYCHAIN=1` to confirm whether the system keychain is the blocking dependency.

Truthy values for keychain bypass:

- `1`
- `true`
- `yes`
- `on`

## Auth Lifecycle

### Init

`gpc auth init`:

- validates the service-account file is readable JSON
- writes profile metadata to config
- accepts `--storage auto|keychain|path`
- with default `--storage auto`, writes a managed credential copy to `~/.gpc/credentials/<profile>.json`
- with `--storage keychain`, imports JSON into keychain and keeps only non-secret metadata in config
- with `--storage path`, keeps the provided `--service-account` path as the persisted profile path
- sets `activeProfile`

### Switch

`gpc auth switch --profile <name>` updates the persisted default active profile.

### Status

`gpc auth status --output json` is additive and automation-friendly. It includes:

- `activeProfile`
- `selectedProfile`
- `authenticated`
- `source`
- `storageBackend`
- `profileStorage`
- `serviceAccountPath` (when path-backed)
- `managedCredentialPath` (when using a managed `~/.gpc/credentials/...` file)
- `lastValidatedAt`
- `developerId`
- `warnings`

### Developer ID Health

If a profile stores `developerId`, `gpc doctor` now validates that value against the live Play `users.list` surface.

Use:

```bash
gpc doctor
gpc doctor --package-name com.example.app
```

If the configured developer ID is stale or invalid, `doctor` reports a `developer_id` warning and points you back to `gpc auth init --developer-id <id>`.

When you pass `--package-name`, `doctor` also reports package readiness:

- `uninitialized`: the app is not initialized in Play yet
- `draft_bootstrap_required`: the package exists, but Play still requires the first internal draft bootstrap release
- `ready`: package access and metadata edits are available

That readiness state is the same one used by `gpc release init`, `gpc release verify`, and `gpc release full`.

When a project has `release-manifest` configured, `doctor` also reports bootstrap rerun context from `./play/bootstrap-state.json`:

- whether an internal draft bootstrap release is already present
- the version code(s) already associated with that bootstrap draft
- the last known readiness result
- the recommended next command for the current state

### Logout

`gpc auth logout` defaults to removing the selected profile.

Scope controls:

- `gpc auth logout` removes selected profile
- `gpc auth logout --profile <name>` removes a specific profile
- `gpc auth logout --all` removes all profiles/credentials

`--profile` and `--all` are mutually exclusive.
