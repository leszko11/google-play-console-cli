# AGENTS.md

A fast, lightweight, AI-agent-friendly CLI for Google Play Console. Built in Go with `ffcli`.

## Core Principles

- Explicit flags: prefer `--package-name` over short aliases.
- TTY-aware output defaults: `table` for interactive terminals and `json` for non-interactive output.
- JSON-first safety for automation (`--output json` and `GPC_DEFAULT_OUTPUT` for explicit control).
- No interactive prompts for core flows (opt-in only via explicit flags).
- Keep stdout for data and stderr for errors/help.

## Command Discovery

Use `--help` at every level:

```bash
gpc --help
gpc auth --help
gpc apps --help
```

## Build and Test

```bash
make build
make test
make lint
make format
```

## Testing Discipline

- TDD by default: start with a failing test.
- Prefer table-driven tests for CLI and domain logic.
- Assert stderr messages for usage and validation failures.
- Keep tests deterministic with temp dirs and env isolation.

## Security

- Never commit service account credentials.
- Keep credentials in local config/keychain and CI secrets only.
