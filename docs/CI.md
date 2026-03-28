# CI Integration Examples

This repo ships GitHub Actions workflows for its own validation, but `gpc` is designed to work the same way in any CI runner:

- install a released `gpc` binary
- write service-account JSON from a masked secret to a temp file
- set `GPC_DEFAULT_OUTPUT=json` for automation-safe output
- set `GPC_BYPASS_KEYCHAIN=1` so CI avoids OS keychains entirely
- run `gpc auth init`, then your read-only verification or release steps

## Shared Environment Contract

Use the same environment variable names across CI providers:

- `GPC_VERSION`: release tag to install, for example `v0.3.0`
- `GPC_SERVICE_ACCOUNT_JSON`: full JSON content of the service-account key
- `GPC_TEST_PACKAGE`: package used for smoke verification
- `GPC_DEFAULT_OUTPUT=json`: keep stdout machine-readable
- `GPC_BYPASS_KEYCHAIN=1`: force deterministic non-keychain auth resolution in CI

The example configs in this repo live under `examples/ci/`:

- `examples/ci/install-gpc.sh`
- `examples/ci/gitlab/.gitlab-ci.yml`
- `examples/ci/circleci/config.yml`
- `examples/ci/bitrise/bitrise.yml`

## GitLab CI

File: `examples/ci/gitlab/.gitlab-ci.yml`

Recommended GitLab CI variables:

- masked variable `GPC_SERVICE_ACCOUNT_JSON`
- variable `GPC_TEST_PACKAGE`
- optional variable `GPC_VERSION` if you want to pin something other than the default in the example

The sample job:

- installs `curl`, `tar`, and certificates
- downloads the released `gpc` binary
- writes the service-account secret to `gpc-sa.json`
- runs `gpc auth init`, `gpc auth status`, and `gpc apps get`

## CircleCI

File: `examples/ci/circleci/config.yml`

Recommended CircleCI project environment variables:

- `GPC_SERVICE_ACCOUNT_JSON`
- `GPC_TEST_PACKAGE`
- optional `GPC_VERSION`

The sample job uses a Linux Docker executor, installs the released binary with the shared helper, and verifies both auth and package access.

## Bitrise

File: `examples/ci/bitrise/bitrise.yml`

Recommended Bitrise Secrets / Env Vars:

- Secret `GPC_SERVICE_ACCOUNT_JSON`
- Env Var `GPC_TEST_PACKAGE`
- optional Env Var `GPC_VERSION`

The sample workflow uses a Script step to install `gpc`, persist the service-account JSON to a temp file, and run the same verification sequence as the Linux examples.

## Extending The Examples

After the initial auth/package smoke step is green, the canonical next commands are:

```bash
gpc doctor --package-name "$GPC_TEST_PACKAGE"
gpc release init --package-name "$GPC_TEST_PACKAGE" --dir ./play
gpc release rehearse --package-name "$GPC_TEST_PACKAGE" --manifest ./play/release.yaml --probe-track
gpc release verify --package-name "$GPC_TEST_PACKAGE" --track internal --aab ./app.aab --notes-file ./play/changelog/internal/en-US.txt
gpc products sync --package-name "$GPC_TEST_PACKAGE" --dry-run
gpc subscriptions sync --package-name "$GPC_TEST_PACKAGE" --dry-run
gpc screenshots sync --package-name "$GPC_TEST_PACKAGE" --dry-run
gpc release full --manifest ./play/release.yaml --dry-run
gpc notify webhook --url "$DEPLOY_WEBHOOK_URL" --event release.completed --input ./release-summary.json
```

Keep CI stages read-only until credentials, package access, and artifact wiring are stable.

Use `gpc release init` once per repo to generate `.gpc.yaml`, `./play/release.yaml`, `./play/appinit.yaml`, and the exported store state. After that, repeat releases can stay on `gpc release full --manifest ./play/release.yaml`.

Use `gpc release rehearse --manifest ./play/release.yaml` as the canonical read-only preflight before any `release full` dry-run or confirm step. It keeps the whole readiness report in one command instead of stitching together `doctor`, `release verify`, and ad hoc track inspection.

If the package is still in Play's draft bootstrap state, edit-only validations can fail with `Only releases with status draft may be created on draft app`. In that case, `gpc release full` handles the internal draft bootstrap release before continuing.

CI guidance by journey:

- Greenfield public app: keep the first Console upload manual and use `./play/MANUAL_FIRST_UPLOAD.md` for the web-only steps.
- Existing Play app: rely on `./play/bootstrap-state.json` plus `gpc doctor` to decide whether bootstrap is already seeded.
- Repeat release / CI: if bootstrap draft already exists, rerun readiness first and rebuild only if another upload is actually needed.
