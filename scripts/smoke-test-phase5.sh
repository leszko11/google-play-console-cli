#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib/smoke-common.sh
source "${SCRIPT_DIR}/lib/smoke-common.sh"

require_env GPC_SERVICE_ACCOUNT
require_env GPC_TEST_PACKAGE

ensure_auth

PAGE_SIZE="${GPC_PHASE5_PAGE_SIZE:-50}"

assert_json_list_field() {
  local json_payload="$1"
  local field_name="$2"
  local is_list
  is_list="$(printf '%s' "${json_payload}" | json_get "isinstance(data.get('${field_name}'), list)")"
  if [[ "${is_list}" != "True" ]]; then
    printf 'expected %s to be a JSON list, got: %s\n' "${field_name}" "${json_payload}" >&2
    exit 1
  fi
}

maybe_skip_purchase_error() {
  local output="$1"
  local strict="${GPC_STRICT_PHASE5_PURCHASES:-0}"

  if [[ "${output}" == *"package not found"* ]] || [[ "${output}" == *"No application was found for the given package name"* ]]; then
    if [[ "${strict}" == "1" ]]; then
      printf 'phase5 strict mode: purchase endpoint returned package-not-found: %s\n' "${output}" >&2
      exit 1
    fi
    log "phase5 purchase endpoint skipped (package-not-found in this account/package context)"
    return 0
  fi

  if [[ "${output}" == *"missing Play Console permissions"* ]] || [[ "${output}" == *"access denied"* ]]; then
    if [[ "${strict}" == "1" ]]; then
      printf 'phase5 strict mode: purchase endpoint returned permissions error: %s\n' "${output}" >&2
      exit 1
    fi
    log "phase5 purchase endpoint skipped (insufficient account scope)"
    return 0
  fi

  return 1
}

SUBSCRIPTIONS_LIST_JSON="$(run_gpc subscriptions list --package-name "${GPC_TEST_PACKAGE}" --page-size "${PAGE_SIZE}")"
assert_json_list_field "${SUBSCRIPTIONS_LIST_JSON}" "subscriptions"

SUBSCRIPTIONS_PAGINATED_JSON="$(run_gpc --paginate subscriptions list --package-name "${GPC_TEST_PACKAGE}" --page-size "${PAGE_SIZE}")"
assert_json_list_field "${SUBSCRIPTIONS_PAGINATED_JSON}" "subscriptions"

PRODUCTS_LIST_JSON="$(run_gpc products list --package-name "${GPC_TEST_PACKAGE}" --page-size "${PAGE_SIZE}")"
assert_json_list_field "${PRODUCTS_LIST_JSON}" "products"

PRODUCTS_PAGINATED_JSON="$(run_gpc --paginate products list --package-name "${GPC_TEST_PACKAGE}" --page-size "${PAGE_SIZE}")"
assert_json_list_field "${PRODUCTS_PAGINATED_JSON}" "products"

if [[ -n "${GPC_TEST_SUBSCRIPTION_PRODUCT_ID:-}" ]]; then
  SUBSCRIPTION_GET_JSON="$(run_gpc subscriptions get --package-name "${GPC_TEST_PACKAGE}" --product-id "${GPC_TEST_SUBSCRIPTION_PRODUCT_ID}")"
  SUBSCRIPTION_ID="$(printf '%s' "${SUBSCRIPTION_GET_JSON}" | json_get 'data.get("subscription", {}).get("productId", "")')"
  if [[ "${SUBSCRIPTION_ID}" != "${GPC_TEST_SUBSCRIPTION_PRODUCT_ID}" ]]; then
    printf 'expected subscription product id %s, got: %s\n' "${GPC_TEST_SUBSCRIPTION_PRODUCT_ID}" "${SUBSCRIPTION_GET_JSON}" >&2
    exit 1
  fi
fi

if [[ -n "${GPC_TEST_PRODUCT_ID:-}" ]]; then
  PRODUCT_GET_JSON="$(run_gpc products get --package-name "${GPC_TEST_PACKAGE}" --product-id "${GPC_TEST_PRODUCT_ID}")"
  PRODUCT_ID="$(printf '%s' "${PRODUCT_GET_JSON}" | json_get 'data.get("product", {}).get("productId", "")')"
  if [[ "${PRODUCT_ID}" != "${GPC_TEST_PRODUCT_ID}" ]]; then
    printf 'expected one-time product id %s, got: %s\n' "${GPC_TEST_PRODUCT_ID}" "${PRODUCT_GET_JSON}" >&2
    exit 1
  fi
fi

set +e
VOIDED_OUTPUT="$(run_gpc purchases voided list --package-name "${GPC_TEST_PACKAGE}" --max-results "${PAGE_SIZE}" 2>&1)"
VOIDED_STATUS=$?
set -e

if [[ ${VOIDED_STATUS} -eq 0 ]]; then
  assert_json_list_field "${VOIDED_OUTPUT}" "voidedPurchases"
else
  maybe_skip_purchase_error "${VOIDED_OUTPUT}" || {
    printf 'phase5 purchases voided list failed: %s\n' "${VOIDED_OUTPUT}" >&2
    exit 1
  }
fi

if [[ -n "${GPC_TEST_SUBSCRIPTION_TOKEN:-}" ]]; then
  SUB_PURCHASE_JSON="$(run_gpc purchases subscriptions get --package-name "${GPC_TEST_PACKAGE}" --token "${GPC_TEST_SUBSCRIPTION_TOKEN}")"
  SUB_STATE="$(printf '%s' "${SUB_PURCHASE_JSON}" | json_get 'data.get("purchase", {}).get("subscriptionState", "")')"
  if [[ -z "${SUB_STATE}" ]]; then
    printf 'expected subscription purchase state, got: %s\n' "${SUB_PURCHASE_JSON}" >&2
    exit 1
  fi

  if [[ -n "${GPC_TEST_SUBSCRIPTION_ETAG:-}" ]]; then
    DEFER_JSON="$(run_gpc purchases subscriptions defer --package-name "${GPC_TEST_PACKAGE}" --token "${GPC_TEST_SUBSCRIPTION_TOKEN}" --etag "${GPC_TEST_SUBSCRIPTION_ETAG}" --defer-duration 604800s --validate-only)"
    DEFER_STATUS="$(printf '%s' "${DEFER_JSON}" | json_get 'data.get("status", "")')"
    if [[ "${DEFER_STATUS}" != "validated" ]]; then
      printf 'expected purchases subscriptions defer validate-only status=validated, got: %s\n' "${DEFER_JSON}" >&2
      exit 1
    fi
  fi
fi

if [[ -n "${GPC_TEST_PRODUCT_ID:-}" && -n "${GPC_TEST_PRODUCT_TOKEN:-}" ]]; then
  PRODUCT_PURCHASE_JSON="$(run_gpc purchases products get --package-name "${GPC_TEST_PACKAGE}" --product-id "${GPC_TEST_PRODUCT_ID}" --token "${GPC_TEST_PRODUCT_TOKEN}")"
  PURCHASE_PRODUCT_ID="$(printf '%s' "${PRODUCT_PURCHASE_JSON}" | json_get 'data.get("purchase", {}).get("productId", "")')"
  if [[ -n "${PURCHASE_PRODUCT_ID}" && "${PURCHASE_PRODUCT_ID}" != "${GPC_TEST_PRODUCT_ID}" ]]; then
    printf 'expected purchase product id %s, got: %s\n' "${GPC_TEST_PRODUCT_ID}" "${PRODUCT_PURCHASE_JSON}" >&2
    exit 1
  fi
fi

log "phase5 smoke passed"
