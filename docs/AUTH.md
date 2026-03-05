# Authentication Model

`gpc` supports profile-based authentication for multiple identities.

## Credential Storage

- Preferred backend: system keychain entry `gpc:credential:<profile>`.
- Stored value: full service account JSON payload bytes.
- Config fallback: profile metadata in `~/.gpc/config.json` with `serviceAccountPath`, `lastValidatedAt`, and optional `developerId`.
- Backward compatibility: existing path-only profiles continue to work; no destructive migration.

## Profile Selection

Profile resolution order:

1. Global `--profile`
2. `activeProfile` in config

Use:

```bash
gpc auth init --profile work --service-account /path/work-sa.json
gpc auth init --profile personal --service-account /path/personal-sa.json
gpc auth switch --profile work
gpc --profile personal auth status --output json
gpc auth profiles list --output table
```

## Credential Source Resolution

Credential source precedence:

1. `--service-account` flag
2. `GPC_SERVICE_ACCOUNT_PATH`
3. keychain credential for resolved profile
4. profile `serviceAccountPath` from config

`gpc auth status` reports:

- `source`: resolved credential source (`flag`, `env`, `keychain`, `config`)
- `storageBackend`: effective backend (`keychain` or `config`)
- `warnings`: fallback/bypass/unavailable notes

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
- On unsupported systems or unavailable keychain backend, `gpc` falls back to config-path metadata and reports warnings.

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
- stores JSON in keychain when available (unless bypassed)
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
- `serviceAccountPath` (when path-backed)
- `lastValidatedAt`
- `developerId`
- `warnings`

### Logout

`gpc auth logout` defaults to removing the selected profile.

Scope controls:

- `gpc auth logout` removes selected profile
- `gpc auth logout --profile <name>` removes a specific profile
- `gpc auth logout --all` removes all profiles/credentials

`--profile` and `--all` are mutually exclusive.
