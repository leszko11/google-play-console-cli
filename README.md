# gpc (Google Play Console CLI)

A fast, lightweight, and scriptable CLI for Google Play Console. Automate Android app workflows from your terminal.

Inspired by Rudrank Riyaam's [App-Store-Connect-CLI](https://github.com/rudrankriyam/App-Store-Connect-CLI).

## Current Scope

- Authentication bootstrap with Google service account credentials
- Basic app visibility commands (`apps list`, `apps get`)
- Edit transactions and listing updates (`edits`)
- Testers and country availability inside edits
- Track management inside edits (`tracks list/get/update/promote`)
- Store images inside edits (`edits images list/upload/delete/delete-all`)
- APK expansion files inside edits (`edits expansion-files get/patch/update/upload`)
- Binary uploads in edits (`apks list/upload`, `bundles list/upload`)
- Deobfuscation mapping upload (`deobfuscation upload`)
- End-to-end deploy orchestration (`deploy`)
- Staging release workflows (`release verify`, `release alpha`)
- Reviews management (`reviews list/get/reply`)
- Orders API support (`orders get/batch-get/refund`)
- External transactions API support (`external-transactions get/create/refund`)
- Device tier config support (`device-tier-configs list/get/create`)
- Monetization subscription commands (`subscriptions ...` including offers)
- Monetization one-time product commands (`products ...`)
- Legacy in-app product commands (`iap ...`)
- Purchase lifecycle commands (`purchases ...`)
- Account users management (`users list/create/update/delete`)
- Per-app grants management (`grants create/update/delete`)
- Internal app sharing upload (`internal-sharing upload`)
- CI quality gates for format, lint, test, and build

## Not Yet Implemented

- Reporting surfaces

## Build

```bash
go build -o build/gpc .
```

## Testing

```bash
make dev
```

Detailed smoke tests: `docs/TESTING.md`.
Auth behavior and credential source model: `docs/AUTH.md`.
Google Play API caveats: `docs/API_NOTES.md`.
Endpoint index notes: `docs/openapi/README.md`.
Endpoint coverage report: `docs/openapi/COVERAGE.md`.

GitHub smoke workflow: `.github/workflows/smoke-tests.yml`.
Use workflow-dispatch inputs `run_phase3` / `run_phase5` to enable optional deploy/monetization smoke phases.

## Quickstart

```bash
# Check build metadata
gpc --version

# Initialize credentials
gpc auth init --service-account /path/to/service-account.json

# Initialize additional profiles (multi-identity)
gpc auth init --profile work --service-account /path/to/work-service-account.json
gpc auth init --profile personal --service-account /path/to/personal-service-account.json

# Optional: save your developer account ID once for developer-level commands
gpc auth init --service-account /path/to/service-account.json --developer-id <developer-id>

# Show current auth profile
gpc auth status
gpc auth profiles list --output table

# Profile override without mutating persisted active profile
gpc --profile work auth status --output json

# Strict auth source policy (fails on mixed sources)
gpc --strict-auth apps get --package-name com.example.app

# Global flags are available from root and apply to all commands
gpc --package-name com.example.app --service-account /path/to/service-account.json --timeout 90s --pretty apps get

# Store package for reusable list/verify flows
gpc apps add-package --package-name com.example.app

# List configured packages
gpc apps list --output json

# Fetch app details for one package
gpc apps get --package-name com.example.app --output json

# Start an edit transaction
gpc edits create --package-name com.example.app

# Read or update app details in an edit
gpc edits details get --package-name com.example.app --edit-id <edit-id>
gpc edits details update \
  --package-name com.example.app \
  --edit-id <edit-id> \
  --contact-email support@example.com

# Manage testers and country availability in an edit
gpc edits testers get --package-name com.example.app --edit-id <edit-id> --track internal
gpc edits testers update --package-name com.example.app --edit-id <edit-id> --track internal --google-groups qa-team@example.com
gpc edits country-availability get --package-name com.example.app --edit-id <edit-id> --track production

# Update listing title in an edit (publish with edits commit)
gpc edits listings update \
  --package-name com.example.app \
  --edit-id <edit-id> \
  --locale en-US \
  --title "My App Title"

# List / delete localized listings in an edit
gpc edits listings list --package-name com.example.app --edit-id <edit-id>
gpc edits listings delete --package-name com.example.app --edit-id <edit-id> --locale en-US
gpc edits listings delete-all --package-name com.example.app --edit-id <edit-id>

# Manage store images in an edit
gpc edits images list --package-name com.example.app --edit-id <edit-id> --locale en-US --image-type phoneScreenshots
gpc edits images upload --package-name com.example.app --edit-id <edit-id> --locale en-US --image-type icon --file /path/to/icon-512.png
gpc edits images delete --package-name com.example.app --edit-id <edit-id> --locale en-US --image-type phoneScreenshots --image-id <image-id>
gpc edits images delete-all --package-name com.example.app --edit-id <edit-id> --locale en-US --image-type phoneScreenshots

# Manage APK expansion files in an edit
gpc edits expansion-files get --package-name com.example.app --edit-id <edit-id> --apk-version-code 123 --expansion-file-type main
gpc edits expansion-files patch --package-name com.example.app --edit-id <edit-id> --apk-version-code 123 --expansion-file-type main --references-version 122
gpc edits expansion-files update --package-name com.example.app --edit-id <edit-id> --apk-version-code 123 --expansion-file-type patch --references-version 122
gpc edits expansion-files upload --package-name com.example.app --edit-id <edit-id> --apk-version-code 123 --expansion-file-type main --file /path/to/main.obb

# Local image validation (pre-API)
# - Supported image types: featureGraphic, icon, phoneScreenshots, promoGraphic,
#   sevenInchScreenshots, tenInchScreenshots, tvBanner, tvScreenshots, wearScreenshots
# - Supported file formats: PNG, JPG, JPEG
# - icon: PNG and exactly 512x512
# - featureGraphic: exactly 1024x500
# - tvBanner: exactly 1280x720
# - screenshot types: each side in range 320-3840

# Commit or delete edits are destructive and require explicit confirmation
gpc edits commit --package-name com.example.app --edit-id <edit-id> --confirm
gpc edits delete --package-name com.example.app --edit-id <edit-id> --confirm

# List/get/reply to user reviews
gpc reviews list --package-name com.example.app --max-results 50
gpc reviews get --package-name com.example.app --review-id <review-id>
gpc reviews reply --package-name com.example.app --review-id <review-id> --reply-text "Thanks for your feedback!"

# Inspect or refund Play orders
gpc orders get --package-name com.example.app --order-id GPA.1234-5678-9012-34567
gpc orders batch-get --package-name com.example.app --order-ids GPA.1234-5678-9012-34567,GPA.1234-5678-9012-34568
gpc orders refund --package-name com.example.app --order-id GPA.1234-5678-9012-34567 --confirm
gpc orders refund --package-name com.example.app --order-id GPA.1234-5678-9012-34567 --revoke --confirm

# Report and refund external transactions
gpc external-transactions get --package-name com.example.app --external-transaction-id ext-monthly-20260305
gpc external-transactions create --package-name com.example.app --external-transaction-id ext-monthly-20260305 --input /path/to/external-transaction.json
gpc external-transactions refund --package-name com.example.app --external-transaction-id ext-monthly-20260305 --input /path/to/external-transaction-refund.json --confirm

# Manage device tier configs used for app bundle targeting
gpc device-tier-configs list --package-name com.example.app --page-size 100
gpc device-tier-configs get --package-name com.example.app --device-tier-config-id 123
gpc device-tier-configs create --package-name com.example.app --input /path/to/device-tier-config.json
gpc device-tier-configs create --package-name com.example.app --input /path/to/device-tier-config.json --allow-unknown-devices

# List/get subscription products
gpc subscriptions list --package-name com.example.app --page-size 100
gpc subscriptions get --package-name com.example.app --product-id premium_monthly
gpc subscriptions batch-get --package-name com.example.app --product-ids premium_monthly,premium_yearly

# Create/update subscriptions from JSON payload files
gpc subscriptions create --package-name com.example.app --input /path/to/subscription.json
gpc subscriptions batch-update --package-name com.example.app --input /path/to/subscriptions-batch-update.json
gpc subscriptions update --package-name com.example.app --product-id premium_monthly --input /path/to/subscription.json
# Note: create/update auto-resolve the active Play regions version via API.

# Delete/archive subscriptions (explicit confirmation required)
gpc subscriptions delete --package-name com.example.app --product-id premium_monthly --confirm
gpc subscriptions archive --package-name com.example.app --product-id premium_monthly --confirm

# Manage subscription base plans
gpc subscriptions base-plans activate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly
gpc subscriptions base-plans deactivate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --confirm
gpc subscriptions base-plans delete --package-name com.example.app --product-id premium_monthly --base-plan-id legacy --confirm
gpc subscriptions base-plans migrate-prices --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --input /path/to/base-plan-migrate-prices.json --confirm
gpc subscriptions base-plans batch-migrate-prices --package-name com.example.app --product-id premium_monthly --input /path/to/base-plans-batch-migrate-prices.json --confirm

# Manage subscription offers
gpc subscriptions offers list --package-name com.example.app --product-id premium_monthly --base-plan-id monthly
gpc subscriptions offers get --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro
gpc subscriptions offers batch-get --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-ids intro,loyalty
gpc subscriptions offers batch-update --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --input /path/to/subscription-offers-batch-update.json
gpc subscriptions offers batch-update-states --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --input /path/to/subscription-offers-batch-update-states.json --confirm
gpc subscriptions offers activate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro
gpc subscriptions offers deactivate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro --confirm
gpc subscriptions offers create --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --input /path/to/offer.json
gpc subscriptions offers update --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro --input /path/to/offer.json --update-mask phases,regionalConfigs
gpc subscriptions offers delete --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro --confirm
# Note: offer create/update auto-resolve the active Play regions version via API.

# One-time product management
gpc products list --package-name com.example.app --page-size 100
gpc products get --package-name com.example.app --product-id coins_100
gpc products batch-get --package-name com.example.app --product-ids coins_100,coins_500
gpc products create --package-name com.example.app --input /path/to/one-time-product.json
gpc products batch-update --package-name com.example.app --input /path/to/one-time-products-batch-update.json
gpc products update --package-name com.example.app --product-id coins_100 --input /path/to/one-time-product.json --update-mask listings,purchaseOptions
gpc products batch-delete --package-name com.example.app --input /path/to/one-time-products-batch-delete.json --confirm
gpc products delete --package-name com.example.app --product-id coins_100 --confirm
# Note: create/update auto-resolve the active Play regions version via API (or use payload `regionsVersion.version` when provided).

# One-time product offers management
gpc products offers list --package-name com.example.app --product-id coins_100 --purchase-option-id buy
gpc products offers batch-get --package-name com.example.app --product-id coins_100 --purchase-option-id buy --offer-ids offer_intro,offer_sale
gpc products offers batch-update --package-name com.example.app --product-id coins_100 --purchase-option-id buy --input /path/to/offers-batch-update.json
gpc products offers batch-update-states --package-name com.example.app --product-id coins_100 --purchase-option-id buy --input /path/to/offers-batch-update-states.json --confirm
gpc products offers batch-delete --package-name com.example.app --product-id coins_100 --purchase-option-id buy --input /path/to/offers-batch-delete.json --confirm
gpc products offers activate --package-name com.example.app --product-id coins_100 --purchase-option-id buy --offer-id offer_intro
gpc products offers deactivate --package-name com.example.app --product-id coins_100 --purchase-option-id buy --offer-id offer_intro --confirm
gpc products offers cancel --package-name com.example.app --product-id coins_100 --purchase-option-id buy --offer-id offer_preorder --confirm

# One-time product purchase option state management
gpc products purchase-options activate --package-name com.example.app --product-id coins_100 --purchase-option-id buy
gpc products purchase-options deactivate --package-name com.example.app --product-id coins_100 --purchase-option-id buy --confirm
gpc products purchase-options delete --package-name com.example.app --product-id coins_100 --purchase-option-id buy --confirm

# Legacy in-app product management
gpc iap list --package-name com.example.app --max-results 100
gpc iap get --package-name com.example.app --sku coins_100
gpc iap batch-get --package-name com.example.app --skus coins_100,coins_500
gpc iap create --package-name com.example.app --input /path/to/inappproduct.json
gpc iap batch-update --package-name com.example.app --input /path/to/inappproducts-batch-update.json
gpc iap update --package-name com.example.app --sku coins_100 --input /path/to/inappproduct.json
gpc iap batch-delete --package-name com.example.app --input /path/to/inappproducts-batch-delete.json --confirm
gpc iap delete --package-name com.example.app --sku coins_100 --confirm

# Purchase management
gpc purchases products get --package-name com.example.app --product-id premium --token <purchase-token>
gpc purchases products-v2 get --package-name com.example.app --token <purchase-token>
gpc purchases products acknowledge --package-name com.example.app --product-id premium --token <purchase-token>
gpc purchases products consume --package-name com.example.app --product-id premium --token <purchase-token> --confirm
gpc purchases subscriptions get --package-name com.example.app --token <subscription-token>
gpc purchases subscriptions cancel --package-name com.example.app --token <subscription-token> --confirm
gpc purchases subscriptions defer --package-name com.example.app --token <subscription-token> --etag <etag> --defer-duration 604800s --confirm
gpc purchases subscriptions defer --package-name com.example.app --token <subscription-token> --etag <etag> --defer-duration 604800s --validate-only
gpc purchases subscriptions revoke --package-name com.example.app --token <subscription-token> --refund-type full --confirm
gpc purchases voided list --package-name com.example.app --max-results 100

# Account users management (developer-level, not package-level)
# --developer-id can be omitted if saved via `gpc auth init --developer-id ...`
gpc users list --developer-id <developer-id>
gpc users create --developer-id <developer-id> --input /path/to/user.json
gpc users update --name developers/<developer-id>/users/<email> --input /path/to/user.json --update-mask expirationTime
gpc users delete --name developers/<developer-id>/users/<email> --confirm

# Shortcut form for update/delete (uses stored auth developer ID)
gpc users update --user-email dev@example.com --input /path/to/user.json --update-mask expirationTime
gpc users delete --user-email dev@example.com --confirm

# Per-app grants management under a user resource
gpc grants create --parent developers/<developer-id>/users/<email> --input /path/to/grant.json
gpc grants update --name developers/<developer-id>/users/<email>/grants/<package-name> --input /path/to/grant.json --update-mask appLevelPermissions
gpc grants delete --name developers/<developer-id>/users/<email>/grants/<package-name> --confirm

# Shortcut form (uses stored auth developer ID)
gpc grants create --user-email dev@example.com --input /path/to/grant.json
gpc grants update --user-email dev@example.com --package-name com.example.app --input /path/to/grant.json --update-mask appLevelPermissions
gpc grants delete --user-email dev@example.com --package-name com.example.app --confirm

# Internal app sharing upload (no edit required)
gpc internal-sharing upload --package-name com.example.app --apk /path/to/app.apk
gpc internal-sharing upload --package-name com.example.app --aab /path/to/app.aab

# Inspect tracks inside an edit
gpc tracks list --package-name com.example.app --edit-id <edit-id>
gpc tracks get --package-name com.example.app --edit-id <edit-id> --track production
gpc tracks promote --package-name com.example.app --edit-id <edit-id> --from-track internal --to-track production
gpc tracks update --package-name com.example.app --edit-id <edit-id> --track alpha --status draft --version-codes 123456789 --release-notes-file /path/to/release-notes.txt

# List/upload binaries in an edit
gpc bundles list --package-name com.example.app --edit-id <edit-id>
gpc bundles upload --package-name com.example.app --edit-id <edit-id> --file /path/to/app.aab
gpc apks list --package-name com.example.app --edit-id <edit-id>
gpc apks upload --package-name com.example.app --edit-id <edit-id> --file /path/to/app.apk

# Upload deobfuscation mapping (proguard/nativeCode)
gpc deobfuscation upload \
  --package-name com.example.app \
  --edit-id <edit-id> \
  --version-code <version-code> \
  --type proguard \
  --file /path/to/mapping.txt

# Deploy in one flow (create edit -> upload -> track update -> validate -> commit)
gpc deploy \
  --package-name com.example.app \
  --aab /path/to/app.aab \
  --track internal \
  --status completed \
  --update-priority 3 \
  --release-notes-locale en-US \
  --release-notes-file /path/to/whats-new.txt \
  --confirm

# Multi-locale release notes format (works with --release-notes-file, --notes-file)
# If file content is plain text, locale defaults to --release-notes-locale / --notes-locale.
cat <<'EOF' >/path/to/release-notes.txt
<pl-PL>
Poprawki bledow i ulepszenia stabilnosci.
</pl-PL>
<cs-CZ>
Opravy chyb a vylepseni stability.
</cs-CZ>
<de-DE>
Fehlerbehebungen und Stabilitaetsverbesserungen.
</de-DE>
<en-GB>
Bug fixes and stability improvements.
</en-GB>
<en-US>
Bug fixes and stability improvements.
</en-US>
<fr-FR>
Corrections de bugs et ameliorations de la stabilite.
</fr-FR>
<it-IT>
Correzioni di bug e miglioramenti della stabilita.
</it-IT>
<sk>
Opravy chyb a zlepsenia stability.
</sk>
EOF

# Dry-run deploy (deletes edit instead of commit)
gpc deploy \
  --package-name com.example.app \
  --aab /path/to/app.aab \
  --track internal \
  --status completed \
  --dry-run

# Preflight release checks for staging -> alpha
gpc release verify \
  --package-name com.example.app.staging \
  --project-dir /path/to/android-project \
  --build-task :app:bundleStagingRelease \
  --notes-mode git

# One-command staging alpha release flow
gpc release alpha \
  --package-name com.example.app.staging \
  --project-dir /path/to/android-project \
  --track alpha \
  --status completed \
  --notes-mode git \
  --confirm
```

## Auth Profiles and Credential Sources

Credential source precedence:

1. `--service-account`
2. `GPC_SERVICE_ACCOUNT_PATH`
3. keychain credential for selected profile
4. profile `serviceAccountPath` in config

Profile selection precedence:

1. `--profile`
2. config `activeProfile`

Useful commands:

```bash
gpc auth profiles list --output table
gpc auth switch --profile work
gpc --profile personal auth status --output json
gpc auth logout --profile work
gpc auth logout --all
```

Keychain controls:

- `GPC_BYPASS_KEYCHAIN=1` disables keychain usage for current process.
- if keychain is unavailable, CLI falls back to config-path metadata and reports warnings in auth output.

Strict source policy:

- `--strict-auth` or `GPC_STRICT_AUTH=1` fails when multiple credential sources are present.

## Bootstrap New Apps

Google Play API can only manage packages that were initialized with at least one artifact uploaded in Play Console UI.

When `gpc` hits a `package not found` error in an interactive terminal and detects an Android Gradle project (`./gradlew` or `./android/gradlew`), it offers a guided build flow:
- asks for `aab`/`apk`, module, and variant
- runs the Gradle task
- prints the built artifact path you can upload manually in Play Console for one-time bootstrap

## Finding Developer ID

- In Play Console, open any URL under your account and copy the number in `developers/<id>`.
- Example: `https://play.google.com/console/u/1/developers/<developer-id>/app-list` → developer ID is `<developer-id>`.
- You can store it once with `gpc auth init --developer-id <id>` so `gpc users list/create` can use it by default.

## Permission Errors

When CLI output includes `access denied` or `missing Play Console permissions`:

- Open Play Console -> `Users and permissions`.
- Add or edit your service account user and grant required permissions:
  - app-level access for package commands (`apps`, `edits`, `tracks`, `deploy`, `reviews`, monetization),
  - account-level access for `users`/`grants` commands.
- In Google Cloud Console, verify `Google Play Android Developer API` is enabled in the same project that owns the service account JSON key.
- Wait a minute for propagation, then retry.

## Global Flags

- `--package-name`: default package for commands that support package-level operations.
- `--service-account`: credential override (`flag > env > keychain > config`).
- `--profile`: select auth profile for this command invocation.
- `--strict-auth`: fail when credentials resolve from multiple sources.
- `--output`: default output format for commands that support output variants.
- `--pretty`: pretty-print JSON output.
- `--timeout`: timeout for standard API requests.
- `--upload-timeout`: timeout for upload API requests.
- `--paginate`: fetch all pages on paginated endpoints (enabled per-command where supported).

## Command Discovery

```bash
gpc --help
gpc auth --help
gpc apps --help
gpc edits --help
gpc tracks --help
gpc bundles --help
gpc apks --help
gpc deobfuscation --help
gpc deploy --help
gpc release --help
gpc reviews --help
gpc orders --help
gpc external-transactions --help
gpc device-tier-configs --help
gpc subscriptions --help
gpc products --help
gpc iap --help
gpc purchases --help
gpc users --help
gpc grants --help
gpc internal-sharing --help
gpc completion --help
```
