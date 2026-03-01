# Testing Guide

## Local Verification

Run full local checks:

```bash
make dev
```

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
