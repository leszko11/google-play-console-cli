# gpc (Google Play Console CLI)

A fast, lightweight, and scriptable CLI for Google Play Console. Automate Android app workflows from your terminal.

Inspired by Rudrank Riyaam's [App-Store-Connect-CLI](https://github.com/rudrankriyam/App-Store-Connect-CLI).

## Current Scope

- Authentication bootstrap with Google service account credentials
- Basic app visibility commands (`apps list`, `apps get`)
- Edit transactions and listing updates (`edits`)
- Track management inside edits (`tracks list/get/update/promote`)
- Binary uploads in edits (`apks list/upload`, `bundles list/upload`)
- CI quality gates for format, lint, test, and build

## Not Yet Implemented

- Deobfuscation mapping uploads
- End-to-end submit orchestration
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

# Start an edit transaction
gpc edits create --package-name com.example.app

# Update listing title in an edit (publish with edits commit)
gpc edits listings update \
  --package-name com.example.app \
  --edit-id <edit-id> \
  --locale en-US \
  --title "My App Title"

# Commit or delete edits are destructive and require explicit confirmation
gpc edits commit --package-name com.example.app --edit-id <edit-id> --confirm
gpc edits delete --package-name com.example.app --edit-id <edit-id> --confirm

# Inspect tracks inside an edit
gpc tracks list --package-name com.example.app --edit-id <edit-id>
gpc tracks get --package-name com.example.app --edit-id <edit-id> --track production
gpc tracks promote --package-name com.example.app --edit-id <edit-id> --from-track internal --to-track production

# List/upload binaries in an edit
gpc bundles list --package-name com.example.app --edit-id <edit-id>
gpc bundles upload --package-name com.example.app --edit-id <edit-id> --file /path/to/app.aab
gpc apks list --package-name com.example.app --edit-id <edit-id>
gpc apks upload --package-name com.example.app --edit-id <edit-id> --file /path/to/app.apk
```

## Command Discovery

```bash
gpc --help
gpc auth --help
gpc apps --help
gpc edits --help
gpc tracks --help
gpc bundles --help
gpc apks --help
```
