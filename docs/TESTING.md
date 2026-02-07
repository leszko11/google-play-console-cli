# Testing Guide

## Local Verification

Run full local checks:

```bash
make dev
```

## CLI Smoke Commands

```bash
gpc --help
gpc auth init --service-account /path/to/service-account.json
gpc auth status
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
gpc edits delete --package-name com.example.app --edit-id <edit-id>
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
