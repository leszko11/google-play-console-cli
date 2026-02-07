# gpc (Google Play Console CLI)

A fast, lightweight, and scriptable CLI for Google Play Console. Automate Android app workflows from your terminal.

Inspired by Rudrank Riyaam's [App-Store-Connect-CLI](https://github.com/rudrankriyam/App-Store-Connect-CLI).

## Phase 1 Scope

- Authentication bootstrap with Google service account credentials
- Basic app visibility commands (`apps list`, `apps get`)
- CI quality gates for format, lint, test, and build

## Non-Goals (Phase 1)

- Edit workflows and publishing (`edits`, `tracks`, uploads)
- Store listing/image management
- Monetization and reporting commands

## Build

```bash
go build -o build/gpc .
```

## Testing

```bash
make dev
```

Detailed smoke tests: `docs/TESTING.md`.

## Quickstart

```bash
# Initialize credentials
gpc auth init --service-account /path/to/service-account.json

# Show current auth profile
gpc auth status

# Store package for reusable list/verify flows
gpc apps add-package --package-name com.example.app

# List configured packages
gpc apps list --output json

# Fetch app details for one package
gpc apps get --package-name com.example.app --output json
```

## Command Discovery

```bash
gpc --help
gpc auth --help
gpc apps --help
```
