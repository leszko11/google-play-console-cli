<p align="center">
  <img src="https://fonts.gstatic.com/s/i/productlogos/play_console/v10/192px.svg" width="96" alt="Google Play Console" />
</p>

<h1 align="center">gpc</h1>

<p align="center">
  <strong>Google Play Console from your terminal.</strong><br/>
  Ship Android apps, manage releases, sync listings, handle monetization &mdash; no browser required.
</p>

<p align="center">
  <a href="https://github.com/leszko11/google-play-console-cli/releases"><img src="https://img.shields.io/github/v/release/leszko11/google-play-console-cli?style=for-the-badge&color=00ADD8&label=Release" alt="Latest Release" /></a>
  <a href="https://github.com/leszko11/google-play-console-cli/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/leszko11/google-play-console-cli/ci.yml?style=for-the-badge&label=CI" alt="CI Status" /></a>
  <a href="https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go"><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go" alt="Go 1.24+" /></a>
  <a href="https://img.shields.io/badge/API_Coverage-136/136-brightgreen?style=for-the-badge"><img src="https://img.shields.io/badge/API_Coverage-136/136-brightgreen?style=for-the-badge" alt="API Coverage 136/136" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue?style=for-the-badge" alt="MIT License" /></a>
</p>

<p align="center">
  <a href="#quickstart">Quickstart</a> &bull;
  <a href="#install">Install</a> &bull;
  <a href="#what-can-it-do">What Can It Do</a> &bull;
  <a href="#common-workflows">Common Workflows</a> &bull;
  <a href="#gpc-vs-raw-google-play-apis">gpc vs Raw APIs</a> &bull;
  <a href="#docs">Docs</a>
</p>

---

## Why gpc?

Working with Google Play Console usually means clicking through a slow web UI or wrestling with raw REST APIs, JSON payloads, and transactional edit sessions. `gpc` wraps all of that into a single binary with human-friendly commands, sane defaults, and first-class CI/CD support.

- **One binary, zero runtime deps** &mdash; download and go
- **136/136 Android Publisher discovery endpoints covered** &mdash; plus Play reporting, billing, and games workflows
- **CI-native** &mdash; JSON output by default in non-TTY, explicit flags for everything, no interactive prompts
- **Multi-profile auth** &mdash; switch between service accounts in one command
- **Dry-run everything** &mdash; preview any destructive operation before committing

Inspired by Rudrank Riyam's [App-Store-Connect-CLI](https://github.com/rudrankriyam/App-Store-Connect-CLI).

---

## Quickstart

```bash
# Install
brew install leszko11/tap/gpc

# Authenticate
gpc auth init --service-account /path/to/service-account.json

# Verify access
gpc doctor --package-name com.example.app

# Save a package for later commands
gpc apps add-package --package-name com.example.app

# Deploy a bundle to internal testing
gpc deploy \
  --package-name com.example.app \
  --aab ./app.aab \
  --track internal \
  --status completed \
  --release-notes-text "Bug fixes and stability improvements." \
  --confirm
```

That's it. Five commands from zero to deployed.

---

## Install

### Homebrew

```bash
brew install leszko11/tap/gpc
```

### Go install

```bash
go install github.com/leszko11/google-play-console-cli@latest
```

### GitHub Releases

Download the archive for your platform from [Releases](https://github.com/leszko11/google-play-console-cli/releases), then verify with the published checksums.

```bash
VERSION=v0.5.0
curl -LO "https://github.com/leszko11/google-play-console-cli/releases/download/${VERSION}/gpc_${VERSION}_darwin_arm64.tar.gz"
curl -LO "https://github.com/leszko11/google-play-console-cli/releases/download/${VERSION}/gpc_${VERSION}_checksums.txt"
shasum -a 256 --check gpc_${VERSION}_checksums.txt
tar -xzf "gpc_${VERSION}_darwin_arm64.tar.gz"
mv gpc /usr/local/bin/gpc
```

### Docker

```bash
docker run --rm -v ~/.gpc:/home/gpc/.gpc ghcr.io/leszko11/gpc apps list
```

### Shell Completion

```bash
# Bash
gpc completion bash > ~/.local/share/bash-completion/completions/gpc

# Zsh
mkdir -p ~/.zfunc && gpc completion zsh > ~/.zfunc/_gpc

# Fish
gpc completion fish > ~/.config/fish/completions/gpc.fish
```

| Platform       | Architecture    | Format   |
|----------------|-----------------|----------|
| macOS          | arm64, amd64    | tar.gz   |
| Linux          | arm64, amd64    | tar.gz   |
| Windows        | amd64           | zip      |

---

## What Can It Do

`gpc` covers the full surface of Google Play Console operations across **6 Google API families**.

### Releases & Publishing

| Command | Description |
|---------|-------------|
| `deploy` | End-to-end: upload bundle &rarr; update track &rarr; validate &rarr; commit |
| `release alpha` | One-command staging release with build verification |
| `release full` | Vitals-gated staged rollout with `--auto-halt-on-regression` |
| `release promote` | Promote releases between tracks |
| `rollback` | Revert a track to previous release |
| `publish alpha/production` | Composite shortcut with built-in bundle processing wait |
| `validate` | Pre-submission gate for listings, assets, and edit validation |

### Store Content

| Command | Description |
|---------|-------------|
| `listing sync` | Sync local listing directory to Play Console |
| `screenshots sync` | Batch sync screenshot directories per locale |
| `changelog sync` | Push release notes from local files |
| `edits listings` | CRUD for localized store listings |
| `edits images` | Upload, list, delete store images |
| `diff listing` | Preview listing drift before syncing |

### Monetization

| Command | Description |
|---------|-------------|
| `subscriptions` | Full lifecycle: create, update, archive, sync with base plans and offers |
| `products` | One-time products with purchase options and offer management |
| `monetization setup` | Manifest-driven provisioning |
| `monetization sync` | Directory-driven sync from YAML manifests |
| `iap` | Legacy in-app product management |
| `purchases` | Validate, acknowledge, consume, defer, revoke, refund |
| `orders` | Query and refund Play Billing orders |

### Analytics & Reporting

| Command | Description |
|---------|-------------|
| `reports vitals` | Crash rate, ANR rate, and custom metric queries |
| `reports anomalies` | Anomaly detection alerts |
| `reports errors` | Error issue and report search |
| `reports financial` | Cloud Storage financial report download and normalization |
| `reports summary` | Operator dashboard: visibility, anomalies, vitals freshness |
| `reviews triage` | Group reviews into pending-reply and replied buckets |

### Account & Access

| Command | Description |
|---------|-------------|
| `users` | Manage developer account users (create, update, delete) |
| `grants` | Per-app permission grants |
| `auth profiles` | Multi-identity service account management |

### Everything Else

| Command | Description |
|---------|-------------|
| `bootstrap` / `appinit` | Export Play state to local workspace |
| `setup --auto` | One-shot GCP + auth + workspace provisioning |
| `workflow run` | Declarative multi-step automation from `.gpc/workflow.yml` |
| `notify` | Webhook, Slack, and Discord notification delivery |
| `doctor` | Diagnostic pass for auth, access, and fixture health |
| `games` | Play Games Services: achievements, events, leaderboards |
| `migrate fastlane import` | Import Fastlane metadata into gpc workspace layout |
| `integrity decode` | Decode Android Integrity API tokens |
| `update` | Self-update to latest release |

<details>
<summary><strong>Full command reference (180+ subcommands)</strong></summary>

Use `--help` at every level:

```bash
gpc --help
gpc auth --help
gpc edits --help
gpc subscriptions offers --help
```

See [docs/COMMANDS.md](docs/COMMANDS.md) for the complete auto-generated reference.

</details>

---

## Common Workflows

### CI/CD: Deploy on merge

```bash
# Non-TTY defaults to JSON output. Explicit flags for automation safety.
gpc deploy \
  --package-name "$PACKAGE_NAME" \
  --service-account "$SA_PATH" \
  --aab ./app/build/outputs/bundle/release/app-release.aab \
  --track internal \
  --status completed \
  --release-notes-text "$COMMIT_MSG" \
  --output json \
  --confirm
```

### Staged rollout with vitals gating

```bash
gpc release full \
  --package-name com.example.app \
  --manifest ./release.yaml \
  --vitals-gate 'crashRate<2.0,anrRate<0.5' \
  --vitals-wait 24h \
  --auto-halt-on-regression \
  --confirm
```

### Sync store listings from local files

```bash
# Preview first
gpc diff listing --package-name com.example.app --dir ./store/listing --delete-missing

# Apply
gpc listing sync --package-name com.example.app --dir ./store/listing --confirm
```

### Multi-locale release notes

```bash
cat <<'EOF' > release-notes.txt
<en-US>
Bug fixes and stability improvements.
</en-US>
<pl-PL>
Poprawki bledow i ulepszenia stabilnosci.
</pl-PL>
<de-DE>
Fehlerbehebungen und Stabilitaetsverbesserungen.
</de-DE>
EOF

gpc deploy --package-name com.example.app --aab ./app.aab \
  --track production --status inProgress \
  --release-notes-file release-notes.txt --confirm
```

### Manage subscriptions from YAML

```bash
# Dry-run to preview changes
gpc monetization sync \
  --package-name com.example.app \
  --manifest ./monetization.yaml \
  --dry-run

# Apply with activation
gpc monetization sync \
  --package-name com.example.app \
  --manifest ./monetization.yaml \
  --confirm --activate
```

### Switch between accounts

```bash
gpc auth init --profile work --service-account /path/to/work.json
gpc auth init --profile personal --service-account /path/to/personal.json

gpc auth switch --profile work
gpc --profile personal apps list   # one-off override without switching
```

### Declarative workflow automation

```yaml
# .gpc/workflow.yml
version: 1
vars:
  packageName: com.example.app
  slackWebhook: https://hooks.slack.com/services/...
steps:
  - id: deploy-internal
    run: >
      gpc deploy --package-name ${packageName} --aab ./app.aab
      --track internal --status completed --confirm
  - id: notify-team
    needs: [deploy-internal]
    run: >
      gpc notify slack --url ${slackWebhook}
      --event release.completed
      --message "Internal build deployed"
```

```bash
gpc workflow run --var packageName=com.example.app --confirm
```

---

## gpc vs Raw Google Play APIs

Without `gpc`, every Play Console operation requires managing OAuth tokens, constructing JSON payloads, handling edit transactions, and chaining multiple API calls. The left-hand snippets below are simplified examples of that manual HTTP plumbing for comparison only. When using `gpc`, you run the command on the right.

### Deploy a bundle

<table>
<tr><th width="50%">Manual API flow (5+ requests)</th><th width="50%">gpc (1 command)</th></tr>
<tr>
<td>

```bash
# 1. Create edit
curl -X POST .../edits \
  -H "Authorization: Bearer $TOKEN"

# 2. Upload bundle
curl -X POST .../edits/$EDIT/bundles \
  -H "Content-Type: application/octet-stream" \
  --data-binary @app.aab

# 3. Update track
curl -X PUT .../edits/$EDIT/tracks/internal \
  -d '{"releases":[{"versionCodes":["42"],
       "status":"completed"}]}'

# 4. Validate edit
curl -X POST .../edits/$EDIT:validate

# 5. Commit edit
curl -X POST .../edits/$EDIT:commit
```

</td>
<td>

```bash
gpc deploy \
  --package-name com.example.app \
  --aab ./app.aab \
  --track internal \
  --status completed \
  --confirm
```

</td>
</tr>
</table>

### Manage subscriptions

<table>
<tr><th width="50%">Manual API flow</th><th width="50%">gpc</th></tr>
<tr>
<td>

```bash
# Resolve regions version first
curl .../monetization/convertRegionPrices

# Create subscription
curl -X POST \
  .../monetization/subscriptions \
  -d '{ 130 lines of JSON... }'

# Activate base plan
curl -X POST \
  .../basePlans/$BP:activate

# Create offer with more JSON...
curl -X POST \
  .../basePlans/$BP/offers \
  -d '{ 80 lines of JSON... }'
```

</td>
<td>

```bash
# Regions version auto-resolved
gpc monetization sync \
  --package-name com.example.app \
  --manifest ./monetization.yaml \
  --confirm --activate
```

</td>
</tr>
</table>

### Android Publisher coverage at a glance

| Metric | Count |
|--------|------:|
| Total discovery endpoints | 136 |
| Implemented endpoints | 136 |
| Missing endpoints | 0 |
| Detected service method IDs | 136 |
| Unmatched service method IDs | 0 |

Full endpoint mapping: [docs/openapi/COVERAGE.md](docs/openapi/COVERAGE.md)

---

## Configuration

### Global Flags

```
--package-name       App package name
--service-account    Path to service account JSON
--profile            Auth profile override
--output             Output format: json, table, markdown, yaml
--fields             JSON field projection (comma-separated)
--timeout            API request timeout (default: 30s)
--pretty             Pretty-print JSON output
--strict-auth        Fail on mixed credential sources
--debug              Enable debug logging
```

### Environment Variables

```bash
GPC_PACKAGE_NAME=com.example.app      # Default package
GPC_SERVICE_ACCOUNT_PATH=/path/to.json # Default credentials
GPC_DEFAULT_OUTPUT=json                # Default output format
GPC_PROFILE=work                       # Default profile
GPC_STRICT_AUTH=1                      # Strict source policy
GPC_BYPASS_KEYCHAIN=1                  # Disable keychain
GPC_CONFIG_PATH=~/.gpc/config.json     # Config location
```

### Output Behavior

| Context     | Default Format |
|-------------|---------------|
| Interactive TTY | `table` |
| Non-interactive / pipe | `json` |
| `--output` flag | Explicit override |
| `GPC_DEFAULT_OUTPUT` | Explicit override |

---

## Auth

`gpc` uses Google service account credentials with a multi-profile system.

```bash
# Initialize (default: managed file-based storage)
gpc auth init --service-account /path/to/sa.json

# Or use OS keychain
gpc auth init --service-account /path/to/sa.json --storage keychain

# Check status
gpc auth status

# Multiple profiles
gpc auth init --profile ci --service-account /path/to/ci-sa.json
gpc auth switch --profile ci
gpc auth profiles list
```

**Credential resolution order:**
1. `--service-account` flag
2. `GPC_SERVICE_ACCOUNT_PATH` env
3. Persisted profile backend (file or keychain)

Full auth documentation: [docs/AUTH.md](docs/AUTH.md)

---

## Build & Development

```bash
make build      # Build binary to ./build/gpc
make test       # Run tests
make lint       # go vet
make format     # gofmt
make coverage   # Generate coverage report
make dev        # Full CI pipeline locally
```

---

## Docs

| Document | Description |
|----------|-------------|
| [COMMANDS.md](docs/COMMANDS.md) | Auto-generated command reference |
| [AUTH.md](docs/AUTH.md) | Authentication model and credential sources |
| [CI.md](docs/CI.md) | CI/CD examples (GitHub Actions, GitLab, CircleCI, Bitrise) |
| [API_NOTES.md](docs/API_NOTES.md) | Google Play API caveats and gotchas |
| [TESTING.md](docs/TESTING.md) | Smoke test documentation |
| [RELEASING.md](docs/RELEASING.md) | Release process |
| [openapi/COVERAGE.md](docs/openapi/COVERAGE.md) | API endpoint coverage report |
| [llms.txt](llms.txt) | AI agent entrypoint |

---

## License

[MIT](LICENSE) &copy; 2026 [Lukasz Lech](https://github.com/leszko11)

---

<p align="center">
  <sub>Built with <a href="https://github.com/peterbourgon/ff">ff</a> and <a href="https://pkg.go.dev/google.golang.org/api">google.golang.org/api</a></sub>
</p>
