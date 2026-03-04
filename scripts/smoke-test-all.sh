#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib/smoke-common.sh
source "${SCRIPT_DIR}/lib/smoke-common.sh"

ensure_config_path

"${SCRIPT_DIR}/smoke-test-phase1.sh"
"${SCRIPT_DIR}/smoke-test-phase2.sh"

if [[ "${GPC_ENABLE_PHASE3:-0}" == "1" ]]; then
  "${SCRIPT_DIR}/smoke-test-phase3.sh"
else
  log "phase3 smoke skipped (set GPC_ENABLE_PHASE3=1 to enable)"
fi

if [[ "${GPC_ENABLE_PHASE5:-0}" == "1" ]]; then
  "${SCRIPT_DIR}/smoke-test-phase5.sh"
else
  log "phase5 smoke skipped (set GPC_ENABLE_PHASE5=1 to enable)"
fi

log "all requested smoke phases passed"
