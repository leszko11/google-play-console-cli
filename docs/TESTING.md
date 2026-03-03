# Testing Guide

## Local Verification

Run full local checks:

```bash
make dev
```

## Smoke Harness Scripts

Phase scripts:

```bash
scripts/smoke-test-phase1.sh
scripts/smoke-test-phase2.sh
scripts/smoke-test-phase3.sh
scripts/smoke-test-all.sh
```

Required environment variables:

- `GPC_SERVICE_ACCOUNT`: absolute path to service-account JSON file.
- `GPC_TEST_PACKAGE`: package used for verification.

Developer account ID (for `users`/`grants` commands):

- Copy numeric ID from Play Console URL segment `developers/<id>`.
- Example: `https://play.google.com/console/u/1/developers/9023817352750250026/app-list`.

Optional environment variables:

- `GPC_BIN`: compiled `gpc` binary path. If unset, scripts use `mise x go@1.24 -- go run .`.
- `GPC_CONFIG_PATH`: config path for isolated smoke runs.
- `GPC_ENABLE_PHASE3=1`: enables phase 3 in `smoke-test-all.sh`.
- `GPC_TEST_AAB` or `GPC_TEST_APK`: exactly one is required for phase 3.
- `GPC_EXPECT_VERSION_CODE`: optional assertion for phase-3 version code.
- `GPC_MAPPING_FILE`, `GPC_MAPPING_TYPE`: optional mapping upload in phase 3.

Example:

```bash
export GPC_SERVICE_ACCOUNT=/absolute/path/to/sa.json
export GPC_TEST_PACKAGE=com.example.app
export GPC_ENABLE_PHASE3=0
scripts/smoke-test-all.sh
```

## GitHub Smoke Workflow

Workflow: `.github/workflows/smoke-tests.yml`

CI secret and variable contract:

- Secret `GPC_SERVICE_ACCOUNT_JSON`: full JSON content (not path).
- Variable `GPC_TEST_PACKAGE`: dedicated test package name.

The workflow writes credentials to a temp file and exports:

```bash
echo "$GPC_SERVICE_ACCOUNT_JSON" > "$RUNNER_TEMP/gpc-sa.json"
chmod 600 "$RUNNER_TEMP/gpc-sa.json"
export GPC_SERVICE_ACCOUNT="$RUNNER_TEMP/gpc-sa.json"
```

Phase 3 can be enabled via workflow-dispatch input `run_phase3=true`, with `aab_path` or `apk_path`. Use `expected_version_code` when your test artifact is built with CI-derived monotonic version code.

## CLI Smoke Commands

```bash
gpc --version
gpc --help
gpc auth init --service-account /path/to/service-account.json
gpc auth init --service-account /path/to/service-account.json --developer-id <developer-id>
gpc auth status
gpc --package-name com.example.app --service-account /path/to/service-account.json --pretty apps get
gpc apps add-package --package-name com.example.app
gpc apps list --output json
gpc apps list --verify
gpc apps get --package-name com.example.app --output json
gpc apps remove-package --package-name com.example.app
gpc edits create --package-name com.example.app
gpc edits get --package-name com.example.app --edit-id <edit-id>
gpc edits details get --package-name com.example.app --edit-id <edit-id>
gpc edits details update --package-name com.example.app --edit-id <edit-id> --contact-email support@example.com
gpc edits testers get --package-name com.example.app --edit-id <edit-id> --track internal
gpc edits testers update --package-name com.example.app --edit-id <edit-id> --track internal --google-groups qa-team@example.com
gpc edits country-availability get --package-name com.example.app --edit-id <edit-id> --track production
gpc edits listings list --package-name com.example.app --edit-id <edit-id>
gpc edits listings get --package-name com.example.app --edit-id <edit-id> --locale en-US
gpc edits listings update --package-name com.example.app --edit-id <edit-id> --locale en-US --title "My App Test"
gpc edits listings delete --package-name com.example.app --edit-id <edit-id> --locale en-US
gpc edits listings delete-all --package-name com.example.app --edit-id <edit-id>
gpc edits images list --package-name com.example.app --edit-id <edit-id> --locale en-US --image-type phoneScreenshots
gpc edits images upload --package-name com.example.app --edit-id <edit-id> --locale en-US --image-type icon --file /path/to/icon-512.png
gpc edits images delete --package-name com.example.app --edit-id <edit-id> --locale en-US --image-type phoneScreenshots --image-id <image-id>
gpc edits images delete-all --package-name com.example.app --edit-id <edit-id> --locale en-US --image-type phoneScreenshots
gpc edits validate --package-name com.example.app --edit-id <edit-id>
gpc edits commit --package-name com.example.app --edit-id <edit-id> --confirm
gpc edits delete --package-name com.example.app --edit-id <edit-id> --confirm
gpc tracks list --package-name com.example.app --edit-id <edit-id>
gpc tracks get --package-name com.example.app --edit-id <edit-id> --track production
gpc tracks update --package-name com.example.app --edit-id <edit-id> --track internal --status completed --version-codes 123456
gpc tracks promote --package-name com.example.app --edit-id <edit-id> --from-track internal --to-track production
gpc bundles list --package-name com.example.app --edit-id <edit-id>
gpc bundles upload --package-name com.example.app --edit-id <edit-id> --file /path/to/app.aab
gpc apks list --package-name com.example.app --edit-id <edit-id>
gpc apks upload --package-name com.example.app --edit-id <edit-id> --file /path/to/app.apk
gpc deobfuscation upload --package-name com.example.app --edit-id <edit-id> --version-code <version-code> --type proguard --file /path/to/mapping.txt
gpc deploy --package-name com.example.app --aab /path/to/app.aab --track internal --status completed --confirm
gpc deploy --package-name com.example.app --aab /path/to/app.aab --track internal --status completed --dry-run
gpc reviews list --package-name com.example.app --max-results 50
gpc reviews get --package-name com.example.app --review-id <review-id>
gpc reviews reply --package-name com.example.app --review-id <review-id> --reply-text "Thanks for your feedback!"
gpc subscriptions list --package-name com.example.app --page-size 100
gpc --paginate subscriptions list --package-name com.example.app --page-size 100
gpc subscriptions get --package-name com.example.app --product-id premium_monthly
gpc subscriptions batch-get --package-name com.example.app --product-ids premium_monthly,premium_yearly
gpc subscriptions create --package-name com.example.app --input /path/to/subscription.json
gpc subscriptions batch-update --package-name com.example.app --input /path/to/subscriptions-batch-update.json
gpc subscriptions update --package-name com.example.app --product-id premium_monthly --input /path/to/subscription.json
gpc subscriptions delete --package-name com.example.app --product-id premium_monthly --confirm
gpc subscriptions archive --package-name com.example.app --product-id premium_monthly --confirm
gpc subscriptions base-plans activate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly
gpc subscriptions base-plans deactivate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --confirm
gpc subscriptions base-plans delete --package-name com.example.app --product-id premium_monthly --base-plan-id legacy --confirm
gpc subscriptions offers list --package-name com.example.app --product-id premium_monthly --base-plan-id monthly
gpc subscriptions offers get --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro
gpc subscriptions offers batch-get --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-ids intro,loyalty
gpc subscriptions offers batch-update --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --input /path/to/subscription-offers-batch-update.json
gpc subscriptions offers activate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro
gpc subscriptions offers deactivate --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro --confirm
gpc subscriptions offers create --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --input /path/to/offer.json
gpc subscriptions offers update --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro --input /path/to/offer.json --update-mask phases,regionalConfigs
gpc subscriptions offers delete --package-name com.example.app --product-id premium_monthly --base-plan-id monthly --offer-id intro --confirm
gpc products list --package-name com.example.app --page-size 100
gpc products get --package-name com.example.app --product-id coins_100
gpc products batch-get --package-name com.example.app --product-ids coins_100,coins_500
gpc products create --package-name com.example.app --input /path/to/one-time-product.json
gpc products batch-update --package-name com.example.app --input /path/to/one-time-products-batch-update.json
gpc products update --package-name com.example.app --product-id coins_100 --input /path/to/one-time-product.json --update-mask listings,purchaseOptions
gpc products batch-delete --package-name com.example.app --input /path/to/one-time-products-batch-delete.json --confirm
gpc products delete --package-name com.example.app --product-id coins_100 --confirm
gpc products offers list --package-name com.example.app --product-id coins_100 --purchase-option-id buy
gpc products offers batch-get --package-name com.example.app --product-id coins_100 --purchase-option-id buy --offer-ids offer_intro,offer_sale
gpc products offers batch-update --package-name com.example.app --product-id coins_100 --purchase-option-id buy --input /path/to/offers-batch-update.json
gpc products offers batch-delete --package-name com.example.app --product-id coins_100 --purchase-option-id buy --input /path/to/offers-batch-delete.json --confirm
gpc products offers activate --package-name com.example.app --product-id coins_100 --purchase-option-id buy --offer-id offer_intro
gpc products offers deactivate --package-name com.example.app --product-id coins_100 --purchase-option-id buy --offer-id offer_intro --confirm
gpc products offers cancel --package-name com.example.app --product-id coins_100 --purchase-option-id buy --offer-id offer_preorder --confirm
gpc products purchase-options activate --package-name com.example.app --product-id coins_100 --purchase-option-id buy
gpc products purchase-options deactivate --package-name com.example.app --product-id coins_100 --purchase-option-id buy --confirm
gpc products purchase-options delete --package-name com.example.app --product-id coins_100 --purchase-option-id buy --confirm
gpc iap list --package-name com.example.app --max-results 100
gpc iap get --package-name com.example.app --sku coins_100
gpc iap batch-get --package-name com.example.app --skus coins_100,coins_500
gpc iap create --package-name com.example.app --input /path/to/inappproduct.json
gpc iap batch-update --package-name com.example.app --input /path/to/inappproducts-batch-update.json
gpc iap update --package-name com.example.app --sku coins_100 --input /path/to/inappproduct.json
gpc iap batch-delete --package-name com.example.app --input /path/to/inappproducts-batch-delete.json --confirm
gpc iap delete --package-name com.example.app --sku coins_100 --confirm
gpc purchases products get --package-name com.example.app --product-id premium --token <purchase-token>
gpc purchases products acknowledge --package-name com.example.app --product-id premium --token <purchase-token>
gpc purchases products consume --package-name com.example.app --product-id premium --token <purchase-token> --confirm
gpc purchases subscriptions get --package-name com.example.app --token <subscription-token>
gpc purchases subscriptions cancel --package-name com.example.app --token <subscription-token> --confirm
gpc purchases subscriptions defer --package-name com.example.app --token <subscription-token> --etag <etag> --defer-duration 604800s --confirm
gpc purchases subscriptions defer --package-name com.example.app --token <subscription-token> --etag <etag> --defer-duration 604800s --validate-only
gpc purchases subscriptions revoke --package-name com.example.app --token <subscription-token> --refund-type full --confirm
gpc purchases voided list --package-name com.example.app --max-results 100
gpc --paginate purchases voided list --package-name com.example.app --max-results 100
gpc users list --developer-id <developer-id>
gpc users list
gpc users create --developer-id <developer-id> --input /path/to/user.json
gpc users create --input /path/to/user.json
gpc users update --name developers/<developer-id>/users/<email> --input /path/to/user.json --update-mask expirationTime
gpc users update --user-email dev@example.com --input /path/to/user.json --update-mask expirationTime
gpc users delete --name developers/<developer-id>/users/<email> --confirm
gpc users delete --user-email dev@example.com --confirm
gpc grants create --parent developers/<developer-id>/users/<email> --input /path/to/grant.json
gpc grants update --name developers/<developer-id>/users/<email>/grants/<package-name> --input /path/to/grant.json --update-mask appLevelPermissions
gpc grants delete --name developers/<developer-id>/users/<email>/grants/<package-name> --confirm
gpc grants create --user-email dev@example.com --input /path/to/grant.json
gpc grants update --user-email dev@example.com --package-name com.example.app --input /path/to/grant.json --update-mask appLevelPermissions
gpc grants delete --user-email dev@example.com --package-name com.example.app --confirm
gpc internal-sharing upload --package-name com.example.app --apk /path/to/app.apk
gpc internal-sharing upload --package-name com.example.app --aab /path/to/app.aab
```

## Expected Outcomes

- No auth configured:
  - `gpc auth status` should report `authenticated: false`.
  - `gpc apps get` should fail with a clear auth/config error.
- Invalid credentials:
  - `gpc auth init --service-account <bad-file>` should fail with a validation/auth error.
- Valid credentials:
  - `gpc auth init --service-account <valid-file>` should succeed.
  - `gpc apps get --package-name <valid-package>` should return app JSON with `packageName`.
  - `gpc edits create --package-name <valid-package>` should return a JSON edit object with `id`.
  - `gpc edits details get ...` should return app details for the edit.
  - `gpc edits details update ...` should return `status: updated` inside an edit.
  - `gpc edits testers get ...` should return testers for the selected track.
  - `gpc edits testers update ...` should return `status: updated`.
  - `gpc edits country-availability get ...` should return country targeting for the track.
  - `gpc edits listings list ...` should return localized listings for the edit.
  - `gpc edits listings update ...` should return `status: updated` inside an edit.
  - `gpc edits listings delete ...` should return `status: deleted`.
  - `gpc edits listings delete-all ...` should return `status: deleted_all`.
  - `gpc edits images list ...` should return image metadata for locale + image type.
  - `gpc edits images upload ...` should return `status: uploaded`; invalid type/dimensions should fail before API call.
  - `gpc edits images delete ...` should return `status: deleted`.
  - `gpc edits images delete-all ...` should return `status: deleted_all`.
  - `gpc edits commit ... --confirm` should be required for publishing changes.
  - `gpc tracks list ...` should return track JSON for the given edit.
  - `gpc tracks promote ...` should copy release metadata from source track to target track within the edit.
  - `gpc bundles list ...` and `gpc apks list ...` should return version code arrays for the edit.
  - `gpc bundles upload ...` and `gpc apks upload ...` should return `status: uploaded` when upload succeeds.
  - `gpc deobfuscation upload ...` should return `status: uploaded` with the uploaded mapping `symbolType`.
  - `gpc deploy ... --confirm` should return `status: committed` and include deterministic `steps`.
  - `gpc deploy ... --dry-run` should return `status: dry-run` and delete the temporary edit.
  - `gpc deploy --track production` should fail unless `--allow-production` is set.
  - `gpc reviews list ...` should return a review array and optional `nextToken`.
  - `gpc reviews get ...` should return a single review.
  - `gpc reviews reply ...` should return `status: replied`.
  - `gpc subscriptions list ...` should return subscription products and optional `nextPageToken`.
  - `gpc --paginate subscriptions list ...` should aggregate all pages and return empty `nextPageToken`.
  - `gpc subscriptions get ...` should return a subscription for the requested `productId`.
  - `gpc subscriptions batch-get ...` should return the requested subscriptions in one call.
  - `gpc subscriptions create ...` should return `status: created`.
  - `gpc subscriptions batch-update ...` should return `status: updated`.
  - `gpc subscriptions update ...` should return `status: updated`.
  - `gpc subscriptions delete ... --confirm` should return `status: deleted`.
  - `gpc subscriptions archive ... --confirm` should return `status: archived`.
  - `gpc subscriptions base-plans activate ...` should return `status: activated`.
  - `gpc subscriptions base-plans deactivate ... --confirm` should return `status: deactivated`.
  - `gpc subscriptions base-plans delete ... --confirm` should return `status: deleted`.
  - `gpc subscriptions offers list ...` should return offers and optional `nextPageToken`.
  - `gpc subscriptions offers get ...` should return one offer.
  - `gpc subscriptions offers batch-get ...` should return the requested offers in one call.
  - `gpc subscriptions offers batch-update ...` should return `status: updated`.
  - `gpc subscriptions offers activate ...` should return `status: activated`.
  - `gpc subscriptions offers deactivate ... --confirm` should return `status: deactivated`.
  - `gpc subscriptions offers create ...` should return `status: created`.
  - `gpc subscriptions offers update ...` should return `status: updated`.
  - `gpc subscriptions offers delete ... --confirm` should return `status: deleted`.
  - `gpc products list ...` should return one-time products and optional `nextPageToken`.
  - `gpc products get ...` should return one one-time product.
  - `gpc products batch-get ...` should return the requested one-time products in one call.
  - `gpc products create ...` should return `status: created`.
  - `gpc products batch-update ...` should return `status: updated`.
  - `gpc products update ...` should return `status: updated`.
  - `gpc products batch-delete ... --confirm` should return `status: deleted`.
  - `gpc products delete ... --confirm` should return `status: deleted`.
  - `gpc products offers list ...` should return offers and optional `nextPageToken`.
  - `gpc products offers batch-get ...` should return the requested offers in one call.
  - `gpc products offers batch-update ...` should return `status: updated`.
  - `gpc products offers batch-delete ... --confirm` should return `status: deleted`.
  - `gpc products offers activate ...` should return `status: activated`.
  - `gpc products offers deactivate ... --confirm` should return `status: deactivated`.
  - `gpc products offers cancel ... --confirm` should return `status: canceled`.
  - `gpc products purchase-options activate ...` should return `status: activated`.
  - `gpc products purchase-options deactivate ... --confirm` should return `status: deactivated`.
  - `gpc products purchase-options delete ... --confirm` should return `status: deleted`.
  - `gpc iap list ...` should return legacy in-app products and optional `nextPageToken`.
  - `gpc iap get ...` should return one legacy in-app product.
  - `gpc iap batch-get ...` should return requested legacy in-app products.
  - `gpc iap create ...` should return `status: created`.
  - `gpc iap batch-update ...` should return `status: updated`.
  - `gpc iap update ...` should return `status: updated`.
  - `gpc iap batch-delete ... --confirm` should return `status: deleted`.
  - `gpc iap delete ... --confirm` should return `status: deleted`.
  - `gpc purchases products get ...` should return one-time purchase details.
  - `gpc purchases products acknowledge ...` should return `status: acknowledged`.
  - `gpc purchases products consume ... --confirm` should return `status: consumed`.
  - `gpc purchases subscriptions get ...` should return subscription purchase details.
  - `gpc purchases subscriptions cancel ... --confirm` should return `status: canceled`.
  - `gpc purchases subscriptions defer ... --confirm` should return `status: deferred`.
  - `gpc purchases subscriptions defer ... --validate-only` should return `status: validated`.
  - `gpc purchases subscriptions revoke ... --confirm` should return `status: revoked`.
  - `gpc purchases voided list ...` should return voided purchases and optional `nextToken`.
  - `gpc --paginate purchases voided list ...` should aggregate all pages and return empty `nextToken`.
  - `gpc users list ...` should return account users with optional `nextPageToken` (use `--developer-id` or a stored `auth init --developer-id` default).
  - `gpc users create ...` should return `status: created`.
  - `gpc users update ...` should return `status: updated`.
  - `gpc users delete ... --confirm` should return `status: deleted`.
  - `gpc grants create ...` should return `status: created`.
  - `gpc grants update ...` should return `status: updated`.
  - `gpc grants delete ... --confirm` should return `status: deleted`.
  - `gpc internal-sharing upload --apk ...` should return `status: uploaded` and a download URL.
  - `gpc internal-sharing upload --aab ...` should return `status: uploaded` and a download URL.
