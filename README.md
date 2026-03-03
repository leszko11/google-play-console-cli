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
- Binary uploads in edits (`apks list/upload`, `bundles list/upload`)
- Deobfuscation mapping upload (`deobfuscation upload`)
- End-to-end deploy orchestration (`deploy`)
- Reviews management (`reviews list/get/reply`)
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

GitHub smoke workflow: `.github/workflows/smoke-tests.yml`.

## Quickstart

```bash
# Check build metadata
gpc --version

# Initialize credentials
gpc auth init --service-account /path/to/service-account.json

# Show current auth profile
gpc auth status

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

# List/get subscription products
gpc subscriptions list --package-name com.example.app --page-size 100
gpc subscriptions get --package-name com.example.app --product-id premium_monthly
gpc subscriptions batch-get --package-name com.example.app --product-ids premium_monthly,premium_yearly

# Create/update subscriptions from JSON payload files
gpc subscriptions create --package-name com.example.app --input /path/to/subscription.json
gpc subscriptions batch-update --package-name com.example.app --input /path/to/subscriptions-batch-update.json
gpc subscriptions update --package-name com.example.app --product-id premium_monthly --input /path/to/subscription.json

# Delete/archive subscriptions (explicit confirmation required)
gpc subscriptions delete --package-name com.example.app --product-id premium_monthly --confirm
gpc subscriptions archive --package-name com.example.app --product-id premium_monthly --confirm

# Manage subscription base plans
gpc subscriptions base-plans activate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly
gpc subscriptions base-plans deactivate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --confirm
gpc subscriptions base-plans delete --package-name com.example.app --product-id premium_monthly --base-plan-id legacy --confirm

# Manage subscription offers
gpc subscriptions offers list --package-name com.example.app --product-id premium_monthly --base-plan-id monthly
gpc subscriptions offers get --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro
gpc subscriptions offers batch-get --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-ids intro,loyalty
gpc subscriptions offers batch-update --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --input /path/to/subscription-offers-batch-update.json
gpc subscriptions offers activate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro
gpc subscriptions offers deactivate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro --confirm
gpc subscriptions offers create --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --input /path/to/offer.json
gpc subscriptions offers update --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro --input /path/to/offer.json --update-mask phases,regionalConfigs
gpc subscriptions offers delete --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro --confirm

# One-time product management
gpc products list --package-name com.example.app --page-size 100
gpc products get --package-name com.example.app --product-id coins_100
gpc products batch-get --package-name com.example.app --product-ids coins_100,coins_500
gpc products create --package-name com.example.app --input /path/to/one-time-product.json
gpc products batch-update --package-name com.example.app --input /path/to/one-time-products-batch-update.json
gpc products update --package-name com.example.app --product-id coins_100 --input /path/to/one-time-product.json --update-mask listings,purchaseOptions
gpc products batch-delete --package-name com.example.app --input /path/to/one-time-products-batch-delete.json --confirm
gpc products delete --package-name com.example.app --product-id coins_100 --confirm

# One-time product offers management
gpc products offers list --package-name com.example.app --product-id coins_100 --purchase-option-id buy
gpc products offers batch-get --package-name com.example.app --product-id coins_100 --purchase-option-id buy --offer-ids offer_intro,offer_sale
gpc products offers batch-update --package-name com.example.app --product-id coins_100 --purchase-option-id buy --input /path/to/offers-batch-update.json
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
gpc purchases products acknowledge --package-name com.example.app --product-id premium --token <purchase-token>
gpc purchases products consume --package-name com.example.app --product-id premium --token <purchase-token> --confirm
gpc purchases subscriptions get --package-name com.example.app --token <subscription-token>
gpc purchases subscriptions cancel --package-name com.example.app --token <subscription-token> --confirm
gpc purchases subscriptions defer --package-name com.example.app --token <subscription-token> --etag <etag> --defer-duration 604800s --confirm
gpc purchases subscriptions defer --package-name com.example.app --token <subscription-token> --etag <etag> --defer-duration 604800s --validate-only
gpc purchases subscriptions revoke --package-name com.example.app --token <subscription-token> --refund-type full --confirm
gpc purchases voided list --package-name com.example.app --max-results 100

# Account users management (developer-level, not package-level)
gpc users list --developer-id <developer-id>
gpc users create --developer-id <developer-id> --input /path/to/user.json
gpc users update --name developers/<developer-id>/users/<email> --input /path/to/user.json --update-mask expirationTime
gpc users delete --name developers/<developer-id>/users/<email> --confirm

# Per-app grants management under a user resource
gpc grants create --parent developers/<developer-id>/users/<email> --input /path/to/grant.json
gpc grants update --name developers/<developer-id>/users/<email>/grants/<package-name> --input /path/to/grant.json --update-mask appLevelPermissions
gpc grants delete --name developers/<developer-id>/users/<email>/grants/<package-name> --confirm

# Internal app sharing upload (no edit required)
gpc internal-sharing upload --package-name com.example.app --apk /path/to/app.apk
gpc internal-sharing upload --package-name com.example.app --aab /path/to/app.aab

# Inspect tracks inside an edit
gpc tracks list --package-name com.example.app --edit-id <edit-id>
gpc tracks get --package-name com.example.app --edit-id <edit-id> --track production
gpc tracks promote --package-name com.example.app --edit-id <edit-id> --from-track internal --to-track production

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
  --confirm

# Dry-run deploy (deletes edit instead of commit)
gpc deploy \
  --package-name com.example.app \
  --aab /path/to/app.aab \
  --track internal \
  --status completed \
  --dry-run
```

## Bootstrap New Apps

Google Play API can only manage packages that were initialized with at least one artifact uploaded in Play Console UI.

When `gpc` hits a `package not found` error in an interactive terminal and detects an Android Gradle project (`./gradlew` or `./android/gradlew`), it offers a guided build flow:
- asks for `aab`/`apk`, module, and variant
- runs the Gradle task
- prints the built artifact path you can upload manually in Play Console for one-time bootstrap

## Global Flags

- `--package-name`: default package for commands that support package-level operations.
- `--service-account`: credential path override (`flag > env > config`).
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
gpc reviews --help
gpc subscriptions --help
gpc products --help
gpc iap --help
gpc purchases --help
gpc users --help
gpc grants --help
gpc internal-sharing --help
```
