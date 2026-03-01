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
gpc auth status
gpc --package-name com.example.app --service-account /path/to/service-account.json --pretty apps get
gpc apps add-package --package-name com.example.app
gpc apps list --output json
gpc apps list --verify
gpc apps get --package-name com.example.app --output json
gpc apps remove-package --package-name com.example.app
gpc edits create --package-name com.example.app
gpc edits get --package-name com.example.app --edit-id <edit-id>
gpc edits listings get --package-name com.example.app --edit-id <edit-id> --locale en-US
gpc edits listings update --package-name com.example.app --edit-id <edit-id> --locale en-US --title "My App Test"
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
  - `gpc edits listings update ...` should return `status: updated` inside an edit.
  - `gpc edits commit ... --confirm` should be required for publishing changes.
  - `gpc tracks list ...` should return track JSON for the given edit.
  - `gpc tracks promote ...` should copy release metadata from source track to target track within the edit.
  - `gpc bundles list ...` and `gpc apks list ...` should return version code arrays for the edit.
  - `gpc bundles upload ...` and `gpc apks upload ...` should return `status: uploaded` when upload succeeds.
  - `gpc deobfuscation upload ...` should return `status: uploaded` with the uploaded mapping `symbolType`.
  - `gpc deploy ... --confirm` should return `status: committed` and include deterministic `steps`.
  - `gpc deploy ... --dry-run` should return `status: dry-run` and delete the temporary edit.
  - `gpc deploy --track production` should fail unless `--allow-production` is set.
