#!/usr/bin/env bash

set -euo pipefail

COMMON_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${COMMON_DIR}/../.." && pwd)"

log() {
  printf '[smoke] %s\n' "$*"
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    printf 'required environment variable is missing: %s\n' "$name" >&2
    exit 1
  fi
}

ensure_config_path() {
  if [[ -n "${GPC_CONFIG_PATH:-}" ]]; then
    return
  fi

  local cfg
  cfg="$(mktemp "${TMPDIR:-/tmp}/gpc-smoke-config.XXXXXX.json")"
  export GPC_CONFIG_PATH="$cfg"
  log "using config path: ${GPC_CONFIG_PATH}"
}

run_gpc() {
  if [[ -n "${GPC_BIN:-}" ]]; then
    "${GPC_BIN}" "$@"
    return
  fi

  (
    cd "${REPO_DIR}"
    mise x go@1.24 -- go run . "$@"
  )
}

ensure_auth() {
  require_env GPC_SERVICE_ACCOUNT
  ensure_config_path
  run_gpc auth init --service-account "${GPC_SERVICE_ACCOUNT}" >/dev/null
}

json_get() {
  local expr="$1"
  python3 -c "import json,sys; data=json.load(sys.stdin); value=${expr}; print(value if value is not None else '')"
}
