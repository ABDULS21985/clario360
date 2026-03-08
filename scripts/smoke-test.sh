#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${CLARIO360_BASE_URL:-http://localhost:8080}"
SMOKE_EMAIL="${CLARIO360_SMOKE_EMAIL:-}"
SMOKE_PASSWORD="${CLARIO360_SMOKE_PASSWORD:-}"
GATEWAY_HEALTH_URL="${CLARIO360_GATEWAY_HEALTH_URL:-http://localhost:8080/healthz}"
IAM_HEALTH_URL="${CLARIO360_IAM_HEALTH_URL:-http://localhost:9081/healthz}"
WORKFLOW_HEALTH_URL="${CLARIO360_WORKFLOW_HEALTH_URL:-http://localhost:8083/healthz}"
AUDIT_HEALTH_URL="${CLARIO360_AUDIT_HEALTH_URL:-http://localhost:8084/healthz}"
CYBER_HEALTH_URL="${CLARIO360_CYBER_HEALTH_URL:-http://localhost:8085/healthz}"
DATA_HEALTH_URL="${CLARIO360_DATA_HEALTH_URL:-http://localhost:8086/healthz}"
ACTA_HEALTH_URL="${CLARIO360_ACTA_HEALTH_URL:-http://localhost:8087/healthz}"
LEX_HEALTH_URL="${CLARIO360_LEX_HEALTH_URL:-http://localhost:8088/healthz}"
VISUS_HEALTH_URL="${CLARIO360_VISUS_HEALTH_URL:-http://localhost:8089/healthz}"
NOTIFICATION_HEALTH_URL="${CLARIO360_NOTIFICATION_HEALTH_URL:-http://localhost:8090/healthz}"
FILE_HEALTH_URL="${CLARIO360_FILE_HEALTH_URL:-http://localhost:9091/healthz}"
WS_ORIGIN_URL="${CLARIO360_WS_ORIGIN_URL:-http://localhost:3000}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  echo "[PASS] $1"
}

fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  echo "[FAIL] $1"
}

skip() {
  SKIP_COUNT=$((SKIP_COUNT + 1))
  echo "[SKIP] $1"
}

curl_status() {
  local output_file="$1"
  shift
  curl -sS -o "${output_file}" -w "%{http_code}" "$@" || true
}

assert_error_shape() {
  local body_file="$1"
  jq -e '
    (.error.code | type) == "string" and
    (.error.message | type) == "string" and
    ((.error.details == null) or ((.error.details | type) == "object")) and
    ((.error.request_id == null) or ((.error.request_id | type) == "string"))
  ' "${body_file}" >/dev/null
}

assert_paginated_shape() {
  local body_file="$1"
  jq -e '
    (.data | type) == "array" and
    (.meta.page | type) == "number" and
    (.meta.per_page | type) == "number" and
    (.meta.total | type) == "number" and
    (.meta.total_pages | type) == "number"
  ' "${body_file}" >/dev/null
}

check_health() {
  local name="$1"
  local url="$2"
  local body="${TMP_DIR}/${name}.health.json"
  local status
  status="$(curl_status "${body}" "${url}")"
  if [[ "${status}" == "200" ]]; then
    pass "health ${name} (${url})"
  else
    fail "health ${name} (${url}) returned HTTP ${status}"
  fi
}

check_error_contract() {
  local name="$1"
  local expected_status="$2"
  shift 2
  local body="${TMP_DIR}/${name}.error.json"
  local status
  status="$(curl_status "${body}" "$@")"
  if [[ "${status}" != "${expected_status}" ]]; then
    fail "${name} returned HTTP ${status}, expected ${expected_status}"
    return
  fi
  if assert_error_shape "${body}"; then
    pass "${name} error contract"
  else
    fail "${name} did not return the expected top-level error shape"
  fi
}

check_paginated_endpoint() {
  local name="$1"
  local path="$2"
  local body="${TMP_DIR}/${name}.json"
  local status
  status="$(curl_status "${body}" -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}${path}")"
  if [[ "${status}" != "200" ]]; then
    fail "${name} returned HTTP ${status}"
    return
  fi
  if assert_paginated_shape "${body}"; then
    pass "${name} paginated contract"
  else
    fail "${name} did not return data[] with meta.{page,per_page,total,total_pages}"
  fi
}

check_object_endpoint() {
  local name="$1"
  local path="$2"
  local body="${TMP_DIR}/${name}.json"
  local status
  status="$(curl_status "${body}" -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}${path}")"
  if [[ "${status}" == "200" ]]; then
    pass "${name} object contract"
  else
    fail "${name} returned HTTP ${status}"
  fi
}

check_array_data_endpoint() {
  local name="$1"
  local path="$2"
  local body="${TMP_DIR}/${name}.json"
  local status
  status="$(curl_status "${body}" -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}${path}")"
  if [[ "${status}" != "200" ]]; then
    fail "${name} returned HTTP ${status}"
    return
  fi
  if jq -e '(.data | type) == "array"' "${body}" >/dev/null; then
    pass "${name} array data contract"
  else
    fail "${name} did not return a top-level data array"
  fi
}

check_top_level_array_endpoint() {
  local name="$1"
  local path="$2"
  local body="${TMP_DIR}/${name}.json"
  local status
  status="$(curl_status "${body}" -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}${path}")"
  if [[ "${status}" != "200" ]]; then
    fail "${name} returned HTTP ${status}"
    return
  fi
  if jq -e '. | type == "array"' "${body}" >/dev/null; then
    pass "${name} top-level array contract"
  else
    fail "${name} did not return a top-level array"
  fi
}

check_websocket_upgrade() {
  local headers_file="${TMP_DIR}/ws.headers"
  curl -sS --http1.1 --max-time 5 -D "${headers_file}" -o /dev/null \
    -H "Origin: ${WS_ORIGIN_URL}" \
    -H "Connection: Upgrade" \
    -H "Upgrade: websocket" \
    -H "Sec-WebSocket-Version: 13" \
    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
    "${BASE_URL}/ws/v1/notifications?token=${TOKEN}" >/dev/null 2>&1 || true

  if grep -qi '^HTTP/1.1 101' "${headers_file}"; then
    pass "notifications websocket upgrade"
  else
    fail "notifications websocket upgrade did not return HTTP 101"
  fi
}

main() {
  require_cmd curl
  require_cmd jq
  require_cmd python3

  echo "Running Clario360 smoke tests against ${BASE_URL}"

  check_health "gateway" "${GATEWAY_HEALTH_URL}"
  check_health "iam" "${IAM_HEALTH_URL}"
  check_health "workflow" "${WORKFLOW_HEALTH_URL}"
  check_health "audit" "${AUDIT_HEALTH_URL}"
  check_health "cyber" "${CYBER_HEALTH_URL}"
  check_health "data" "${DATA_HEALTH_URL}"
  check_health "acta" "${ACTA_HEALTH_URL}"
  check_health "lex" "${LEX_HEALTH_URL}"
  check_health "visus" "${VISUS_HEALTH_URL}"
  check_health "notification" "${NOTIFICATION_HEALTH_URL}"
  check_health "file" "${FILE_HEALTH_URL}"

  check_error_contract "gateway-not-found" "404" "${BASE_URL}/api/v1/definitely-missing"
  check_error_contract "gateway-unauthorized" "401" "${BASE_URL}/api/v1/users/me"

  TOKEN=""
  if [[ -n "${SMOKE_EMAIL}" && -n "${SMOKE_PASSWORD}" ]]; then
    local login_body="${TMP_DIR}/login.json"
    local payload
    payload="$(jq -cn --arg email "${SMOKE_EMAIL}" --arg password "${SMOKE_PASSWORD}" '{email: $email, password: $password}')"
    local status
    status="$(curl_status "${login_body}" -X POST -H "Content-Type: application/json" -d "${payload}" "${BASE_URL}/api/v1/auth/login")"
    if [[ "${status}" == "200" ]]; then
      TOKEN="$(jq -r '.access_token // empty' "${login_body}")"
      if [[ -n "${TOKEN}" ]]; then
        pass "gateway authentication"
      else
        fail "gateway authentication succeeded without an access_token"
      fi
    else
      fail "gateway authentication returned HTTP ${status}"
    fi
  else
    skip "gateway authentication (set CLARIO360_SMOKE_EMAIL and CLARIO360_SMOKE_PASSWORD to enable)"
  fi

  if [[ -n "${TOKEN}" ]]; then
    local date_from
    local date_to
    date_from="$(python3 - <<'PY'
from datetime import datetime, timedelta, timezone
print((datetime.now(timezone.utc) - timedelta(days=7)).isoformat().replace("+00:00", "Z"))
PY
)"
    date_to="$(python3 - <<'PY'
from datetime import datetime, timezone
print(datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"))
PY
)"

    check_object_endpoint "users-me" "/api/v1/users/me"
    check_top_level_array_endpoint "users-me-sessions" "/api/v1/users/me/sessions"
    check_object_endpoint "notifications-unread-count" "/api/v1/notifications/unread-count"
    check_object_endpoint "notifications-preferences" "/api/v1/notifications/preferences"
    check_object_endpoint "cyber-vciso-briefing" "/api/v1/cyber/vciso/briefing"
    check_object_endpoint "cyber-vciso-posture-summary" "/api/v1/cyber/vciso/posture-summary"
    check_object_endpoint "lex-dashboard" "/api/v1/lex/dashboard"
    check_object_endpoint "data-quality-dashboard" "/api/v1/data/quality/dashboard"
    check_array_data_endpoint "data-quality-score-trend" "/api/v1/data/quality/score/trend?days=30"
    check_top_level_array_endpoint "notebooks-servers" "/api/v1/notebooks/servers"

    check_paginated_endpoint "notifications" "/api/v1/notifications?page=1&per_page=5"
    check_paginated_endpoint "files" "/api/v1/files?page=1&per_page=5"
    check_paginated_endpoint "cyber-alerts" "/api/v1/cyber/alerts?page=1&per_page=5"
    check_paginated_endpoint "cyber-ueba-profiles" "/api/v1/cyber/ueba/profiles?page=1&per_page=5"
    check_paginated_endpoint "cyber-ueba-alerts" "/api/v1/cyber/ueba/alerts?page=1&per_page=5"
    check_paginated_endpoint "cyber-ueba-timeline" "/api/v1/cyber/ueba/profiles/test-entity/timeline?page=1&per_page=5"
    check_paginated_endpoint "data-sources" "/api/v1/data/sources?page=1&per_page=5"
    check_paginated_endpoint "acta-committees" "/api/v1/acta/committees?page=1&per_page=5"
    check_paginated_endpoint "lex-contracts" "/api/v1/lex/contracts?page=1&per_page=5"
    check_paginated_endpoint "visus-alerts" "/api/v1/visus/alerts?page=1&per_page=5"
    check_paginated_endpoint "workflow-instances" "/api/v1/workflows/instances?page=1&per_page=5"
    check_paginated_endpoint "ai-models" "/api/v1/ai/models?page=1&per_page=5"
    check_paginated_endpoint "audit-logs" "/api/v1/audit/logs?date_from=${date_from}&date_to=${date_to}&page=1&per_page=5"

    check_websocket_upgrade
  else
    skip "authenticated endpoint sweep (no access token available)"
    skip "notifications websocket upgrade (no access token available)"
  fi

  echo
  echo "Smoke summary: pass=${PASS_COUNT} fail=${FAIL_COUNT} skip=${SKIP_COUNT}"
  if [[ "${FAIL_COUNT}" -gt 0 ]]; then
    exit 1
  fi
}

main "$@"
