# OpenAPI Endpoint Index

This directory tracks a lightweight endpoint index for Android Publisher API v3.

Files:

- `paths.txt`: normalized endpoint list for quick grep/review
- `COVERAGE.md`: generated implemented-vs-missing endpoint report based on live `internal/gpc` service usage

## Regenerate `paths.txt`

Offline refresh from a local discovery file:

```bash
python3 scripts/update-openapi-paths.py \
  --source /path/to/androidpublisher-v3-discovery.json
```

Online refresh (fetch current discovery doc directly):

```bash
python3 scripts/update-openapi-paths.py --fetch
```

Custom output:

```bash
python3 scripts/update-openapi-paths.py --fetch --output /tmp/paths.txt
```

Generate the coverage report:

```bash
python3 scripts/generate-openapi-coverage.py
```

## Format

Each non-comment line in `paths.txt` is:

```text
<HTTP_METHOD> <PATH> [<DISCOVERY_METHOD_ID>]
```

Examples:

```text
GET androidpublisher/v3/applications/{packageName}/edits/{editId} [androidpublisher.edits.get]
POST androidpublisher/v3/applications/{packageName}/edits/{editId}:commit [androidpublisher.edits.commit]
```
