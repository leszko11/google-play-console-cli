#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib/smoke-common.sh
source "${SCRIPT_DIR}/lib/smoke-common.sh"

require_env GPC_SERVICE_ACCOUNT
require_env GPC_TEST_PACKAGE

ensure_auth

STATUS_JSON="$(run_gpc auth status)"
AUTHENTICATED="$(printf '%s' "${STATUS_JSON}" | json_get 'data.get("authenticated", False)')"
if [[ "${AUTHENTICATED}" != "True" ]]; then
  printf 'expected authenticated=true, got status: %s\n' "${STATUS_JSON}" >&2
  exit 1
fi

run_gpc apps add-package --package-name "${GPC_TEST_PACKAGE}" >/dev/null

LIST_JSON="$(run_gpc apps list --verify)"
PACKAGE_STATUS="$(printf '%s' "${LIST_JSON}" | python3 -c 'import json,os,sys; items=json.load(sys.stdin); pkg=os.environ["GPC_TEST_PACKAGE"]; found=[x for x in items if x.get("packageName")==pkg]; print(found[0].get("status","missing") if found else "missing")')"
if [[ "${PACKAGE_STATUS}" != "ok" ]]; then
  printf 'expected package status ok in apps list --verify, got: %s\n' "${LIST_JSON}" >&2
  exit 1
fi

GET_JSON="$(run_gpc apps get --package-name "${GPC_TEST_PACKAGE}")"
PACKAGE_NAME="$(printf '%s' "${GET_JSON}" | json_get 'data.get("packageName", "")')"
if [[ "${PACKAGE_NAME}" != "${GPC_TEST_PACKAGE}" ]]; then
  printf 'expected packageName=%s, got: %s\n' "${GPC_TEST_PACKAGE}" "${GET_JSON}" >&2
  exit 1
fi

log "phase1 smoke passed"
