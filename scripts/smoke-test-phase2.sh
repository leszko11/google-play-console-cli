#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib/smoke-common.sh
source "${SCRIPT_DIR}/lib/smoke-common.sh"

require_env GPC_SERVICE_ACCOUNT
require_env GPC_TEST_PACKAGE

ensure_auth

EDIT_ID=""
cleanup() {
  if [[ -n "${EDIT_ID}" ]]; then
    run_gpc edits delete --package-name "${GPC_TEST_PACKAGE}" --edit-id "${EDIT_ID}" --confirm >/dev/null || true
  fi
}
trap cleanup EXIT

CREATE_JSON="$(run_gpc edits create --package-name "${GPC_TEST_PACKAGE}")"
EDIT_ID="$(printf '%s' "${CREATE_JSON}" | json_get 'data.get("edit", {}).get("id", "")')"
if [[ -z "${EDIT_ID}" ]]; then
  printf 'failed to extract edit id from: %s\n' "${CREATE_JSON}" >&2
  exit 1
fi

run_gpc edits get --package-name "${GPC_TEST_PACKAGE}" --edit-id "${EDIT_ID}" >/dev/null
run_gpc tracks list --package-name "${GPC_TEST_PACKAGE}" --edit-id "${EDIT_ID}" >/dev/null

VALIDATE_JSON="$(run_gpc edits validate --package-name "${GPC_TEST_PACKAGE}" --edit-id "${EDIT_ID}")"
VALIDATE_STATUS="$(printf '%s' "${VALIDATE_JSON}" | json_get 'data.get("status", "")')"
if [[ "${VALIDATE_STATUS}" != "validated" ]]; then
  printf 'expected validated status, got: %s\n' "${VALIDATE_JSON}" >&2
  exit 1
fi

DELETE_JSON="$(run_gpc edits delete --package-name "${GPC_TEST_PACKAGE}" --edit-id "${EDIT_ID}" --confirm)"
DELETE_STATUS="$(printf '%s' "${DELETE_JSON}" | json_get 'data.get("status", "")')"
if [[ "${DELETE_STATUS}" != "deleted" ]]; then
  printf 'expected deleted status, got: %s\n' "${DELETE_JSON}" >&2
  exit 1
fi

EDIT_ID=""
log "phase2 smoke passed"
