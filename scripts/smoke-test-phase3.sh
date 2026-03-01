#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib/smoke-common.sh
source "${SCRIPT_DIR}/lib/smoke-common.sh"

require_env GPC_SERVICE_ACCOUNT
require_env GPC_TEST_PACKAGE

ensure_auth

TRACK="${GPC_TEST_TRACK:-internal}"
RELEASE_STATUS="${GPC_RELEASE_STATUS:-completed}"
RELEASE_NAME="${GPC_RELEASE_NAME:-CLI E2E ${GITHUB_RUN_NUMBER:-local}}"

AAB_PATH="${GPC_TEST_AAB:-}"
APK_PATH="${GPC_TEST_APK:-}"

if [[ -z "${AAB_PATH}" && -z "${APK_PATH}" ]]; then
  printf 'phase3 requires one artifact path via GPC_TEST_AAB or GPC_TEST_APK\n' >&2
  exit 1
fi
if [[ -n "${AAB_PATH}" && -n "${APK_PATH}" ]]; then
  printf 'phase3 requires exactly one artifact path; got both GPC_TEST_AAB and GPC_TEST_APK\n' >&2
  exit 1
fi

DEPLOY_ARGS=(
  deploy
  --package-name "${GPC_TEST_PACKAGE}"
  --track "${TRACK}"
  --status "${RELEASE_STATUS}"
  --release-name "${RELEASE_NAME}"
  --confirm
)

if [[ -n "${AAB_PATH}" ]]; then
  DEPLOY_ARGS+=(--aab "${AAB_PATH}")
else
  DEPLOY_ARGS+=(--apk "${APK_PATH}")
fi

if [[ -n "${GPC_MAPPING_FILE:-}" ]]; then
  DEPLOY_ARGS+=(--mapping-file "${GPC_MAPPING_FILE}")
fi
if [[ -n "${GPC_MAPPING_TYPE:-}" ]]; then
  DEPLOY_ARGS+=(--mapping-type "${GPC_MAPPING_TYPE}")
fi
if [[ "${TRACK}" == "production" && "${GPC_ALLOW_PRODUCTION:-0}" == "1" ]]; then
  DEPLOY_ARGS+=(--allow-production)
fi

DEPLOY_JSON="$(run_gpc "${DEPLOY_ARGS[@]}")"
DEPLOY_STATUS="$(printf '%s' "${DEPLOY_JSON}" | json_get 'data.get("status", "")')"
COMMITTED="$(printf '%s' "${DEPLOY_JSON}" | json_get 'data.get("committed", False)')"
VERSION_CODE="$(printf '%s' "${DEPLOY_JSON}" | json_get 'data.get("versionCode", 0)')"

if [[ "${DEPLOY_STATUS}" != "committed" ]]; then
  printf 'expected deploy status committed, got: %s\n' "${DEPLOY_JSON}" >&2
  exit 1
fi
if [[ "${COMMITTED}" != "True" ]]; then
  printf 'expected committed=true, got: %s\n' "${DEPLOY_JSON}" >&2
  exit 1
fi
if ! [[ "${VERSION_CODE}" =~ ^[0-9]+$ ]] || [[ "${VERSION_CODE}" -le 0 ]]; then
  printf 'expected positive versionCode, got: %s\n' "${DEPLOY_JSON}" >&2
  exit 1
fi

if [[ -n "${GPC_EXPECT_VERSION_CODE:-}" && "${VERSION_CODE}" != "${GPC_EXPECT_VERSION_CODE}" ]]; then
  printf 'expected versionCode=%s, got=%s\n' "${GPC_EXPECT_VERSION_CODE}" "${VERSION_CODE}" >&2
  exit 1
fi

log "phase3 smoke passed (versionCode=${VERSION_CODE})"
