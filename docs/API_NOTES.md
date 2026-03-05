# Google Play API Notes

Operational caveats and behavior notes for Android Publisher API v3.

## Package Bootstrap Constraint

Play API calls fail for packages that were never initialized in Play Console UI.

- Typical error: package not found (HTTP 404)
- Required first step: upload first APK/AAB once in Play Console UI
- After bootstrap, API flows (`edits`, `deploy`, `tracks`, etc.) become available

## Permission Model

Service account credentials are not enough without Play Console permissions.

- App-level commands require app access for the package
- Account-level commands (`users`, `grants`) require account permissions
- Enable `Google Play Android Developer API` in the same GCP project as the service account

Common failures:

- HTTP 401/403: missing or insufficient permissions
- HTTP 404 with package not found text: uninitialized package or inaccessible package

## Edits Are Transactional

Most publishing operations happen inside an edit:

1. create edit
2. mutate resources (listings, tracks, images, binaries)
3. validate or commit

Notes:

- `edits commit` and `edits delete` are destructive and require explicit confirmation
- stale edit IDs can fail after lifecycle changes
- treat edit IDs as short-lived transaction handles

## Artifact Upload Behavior

- Some apps accept AAB only; APK upload may fail with API error text about APK disallowance
- Use `bundles upload` or `deploy --aab ...` for AAB-only apps

## Release Notes and Locales

- Track/deploy notes support plain text or tagged multi-locale payload files
- Locale normalization and duplicate-locale prevention should be handled before API writes

## Monetization API Split

There are two product surfaces:

- New monetization API: `products`, `subscriptions`
- Legacy IAP API: `iap`

If API messages indicate migration to new publishing API, use `products`/`subscriptions` instead of legacy `iap`.

## Regions Version Resolution

For monetization create/update flows:

- CLI auto-resolves active Play regions version when possible
- payload-specified regions version can override in supported commands

## Pagination and Throughput

- Use `--paginate` for complete list retrieval where supported
- Large lists can increase runtime and API quota usage
- Prefer bounded page sizes for scripts with strict time budgets

## Orders API

- Orders API supports direct order lookup, batch lookup, and refunds
- Discovery docs note a maximum of 1000 orders per batch request
- Order and refund automation counts against Play Developer API quota, so bulk sync jobs should be explicit and rate-aware

## Timeouts

- `--timeout` controls standard API request deadline
- `--upload-timeout` controls upload request deadline

Set explicit values in CI for predictable behavior.
