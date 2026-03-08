#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
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
SLACK_SIGNING_SECRET_FILE="${CLARIO360_SLACK_SIGNING_SECRET_FILE:-${REPO_ROOT}/.dev-secrets/slack-signing-secret.key}"
SLACK_SIGNING_SECRET="${CLARIO360_SLACK_SIGNING_SECRET:-}"
if [[ -z "${SLACK_SIGNING_SECRET}" && -f "${SLACK_SIGNING_SECRET_FILE}" ]]; then
  SLACK_SIGNING_SECRET="$(tr -d '\n' < "${SLACK_SIGNING_SECRET_FILE}")"
fi

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

assert_array_data_shape() {
  local body_file="$1"
  jq -e '(.data | type) == "array"' "${body_file}" >/dev/null
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

check_array_data_endpoint_via_auth() {
  local name="$1"
  local path="$2"
  local body="${TMP_DIR}/${name}.json"
  local status
  status="$(curl_status "${body}" -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}${path}")"
  if [[ "${status}" != "200" ]]; then
    fail "${name} returned HTTP ${status}"
    return
  fi
  if assert_array_data_shape "${body}"; then
    pass "${name} data array contract"
  else
    fail "${name} did not return a top-level data array"
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

create_integration() {
  local name="$1"
  local payload="$2"
  local body="${TMP_DIR}/${name}.create.json"
  local status
  status="$(curl_status "${body}" -X POST -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" -d "${payload}" "${BASE_URL}/api/v1/integrations")"
  if [[ "${status}" != "201" ]]; then
    fail "${name} create returned HTTP ${status}"
    return 1
  fi
  jq -r '.data.id // empty' "${body}"
}

delete_integration() {
  local name="$1"
  local integration_id="$2"
  local body="${TMP_DIR}/${name}.delete.json"
  local status
  status="$(curl_status "${body}" -X DELETE -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/api/v1/integrations/${integration_id}")"
  if [[ "${status}" == "204" ]]; then
    pass "${name} delete"
  else
    fail "${name} delete returned HTTP ${status}"
  fi
}

sign_slack_request() {
  local body_file="$1"
  local timestamp="$2"
  python3 - "$SLACK_SIGNING_SECRET" "$timestamp" "$body_file" <<'PY'
import hashlib
import hmac
import pathlib
import sys

secret = sys.argv[1].encode()
timestamp = sys.argv[2]
body = pathlib.Path(sys.argv[3]).read_bytes()
base = b"v0:" + timestamp.encode() + b":" + body
print("v0=" + hmac.new(secret, base, hashlib.sha256).hexdigest())
PY
}

sign_jira_request() {
  local secret="$1"
  local body_file="$2"
  python3 - "$secret" "$body_file" <<'PY'
import hashlib
import hmac
import pathlib
import sys

secret = sys.argv[1].encode()
body = pathlib.Path(sys.argv[2]).read_bytes()
print("sha256=" + hmac.new(secret, body, hashlib.sha256).hexdigest())
PY
}

check_slack_events_signature() {
  if [[ -z "${SLACK_SIGNING_SECRET}" ]]; then
    skip "slack events verification (no local signing secret available)"
    return
  fi

  local body="${TMP_DIR}/slack.events.body.json"
  printf '{"type":"url_verification","challenge":"smoke-challenge"}' > "${body}"
  local timestamp
  timestamp="$(date +%s)"
  local signature
  signature="$(sign_slack_request "${body}" "${timestamp}")"

  local valid_body="${TMP_DIR}/slack.events.valid.json"
  local valid_status
  valid_status="$(curl_status "${valid_body}" -X POST \
    -H "Content-Type: application/json" \
    -H "X-Slack-Request-Timestamp: ${timestamp}" \
    -H "X-Slack-Signature: ${signature}" \
    --data-binary "@${body}" \
    "${BASE_URL}/api/v1/integrations/slack/events")"
  if [[ "${valid_status}" == "200" ]] && jq -e '.challenge == "smoke-challenge"' "${valid_body}" >/dev/null; then
    pass "slack events valid signature"
  else
    fail "slack events valid signature returned HTTP ${valid_status}"
  fi

  check_error_contract "slack-events-invalid-signature" "401" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "X-Slack-Request-Timestamp: ${timestamp}" \
    -H "X-Slack-Signature: v0=invalid" \
    --data-binary "@${body}" \
    "${BASE_URL}/api/v1/integrations/slack/events"
}

check_teams_invalid_token() {
  check_error_contract "teams-invalid-jwt" "401" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer invalid.jwt.token" \
    -d '{"type":"message","text":"help","recipient":{"id":"bot"},"serviceUrl":"https://smba.trafficmanager.net/emea/","conversation":{"id":"conv"}}' \
    "${BASE_URL}/api/v1/integrations/teams/messages"
}

check_oauth_start() {
  local name="$1"
  local session_path="$2"
  local payload="$3"
  local session_body="${TMP_DIR}/${name}.oauth-session.json"
  local session_status
  session_status="$(curl_status "${session_body}" -X POST -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" -d "${payload}" "${BASE_URL}${session_path}")"
  if [[ "${session_status}" == "503" ]] && jq -e '.error.code and .error.message' "${session_body}" >/dev/null; then
    pass "${name} oauth session unavailable contract"
    return
  fi
  if [[ "${session_status}" != "200" ]]; then
    fail "${name} oauth session returned HTTP ${session_status}"
    return
  fi

  local start_url
  start_url="$(jq -r '.data.url // empty' "${session_body}")"
  if [[ -z "${start_url}" ]]; then
    fail "${name} oauth session did not return a start url"
    return
  fi

  local start_body="${TMP_DIR}/${name}.oauth-start.json"
  local start_status
  start_status="$(curl_status "${start_body}" "${start_url}")"
  if [[ "${start_status}" == "302" ]]; then
    pass "${name} oauth start redirect"
    return
  fi
  fail "${name} oauth start returned HTTP ${start_status}"
}

check_jira_webhooks() {
  local secret="smoke-jira-secret"
  local body="${TMP_DIR}/jira.webhook.body.json"
  printf '{"webhookEvent":"jira:issue_updated","issue":{"id":"999999","key":"SEC-999","fields":{"status":{"name":"Done"}}},"changelog":{"items":[{"field":"status","toString":"Done"}]}}' > "${body}"

  check_error_contract "jira-webhook-invalid-signature" "401" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "X-Hub-Signature: sha256=invalid" \
    --data-binary "@${body}" \
    "${BASE_URL}/api/v1/integrations/jira/webhook"

  local payload
  payload="$(jq -cn --arg secret "${secret}" '{
    type: "jira",
    name: "Smoke Jira Webhook",
    description: "Temporary smoke-test Jira integration",
    config: {
      base_url: "https://example.atlassian.net",
      project_key: "SEC",
      auth_token: "smoke-token",
      webhook_secret: $secret
    },
    event_filters: []
  }')"
  local integration_id
  integration_id="$(create_integration "jira-webhook-smoke" "${payload}")" || return
  if [[ -z "${integration_id}" ]]; then
    fail "jira-webhook-smoke create did not return an integration id"
    return
  fi
  pass "jira-webhook-smoke create"

  local signature
  signature="$(sign_jira_request "${secret}" "${body}")"
  local valid_body="${TMP_DIR}/jira.webhook.valid.json"
  local valid_status
  valid_status="$(curl_status "${valid_body}" -X POST \
    -H "Content-Type: application/json" \
    -H "X-Hub-Signature: ${signature}" \
    --data-binary "@${body}" \
    "${BASE_URL}/api/v1/integrations/jira/webhook")"
  if [[ "${valid_status}" == "200" ]]; then
    pass "jira webhook valid signature"
  else
    fail "jira webhook valid signature returned HTTP ${valid_status}"
  fi

  delete_integration "jira-webhook-smoke" "${integration_id}"
}

check_servicenow_webhooks() {
  local secret="smoke-servicenow-secret"
  local body="${TMP_DIR}/servicenow.webhook.body.json"
  printf '{"result":{"sys_id":"abc123","state":"Resolved","number":"INC000999"}}' > "${body}"

  check_error_contract "servicenow-webhook-invalid-token" "401" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "X-ServiceNow-Token: invalid" \
    --data-binary "@${body}" \
    "${BASE_URL}/api/v1/integrations/servicenow/webhook"

  local payload
  payload="$(jq -cn --arg secret "${secret}" '{
    type: "servicenow",
    name: "Smoke ServiceNow Webhook",
    description: "Temporary smoke-test ServiceNow integration",
    config: {
      instance_url: "https://example.service-now.com",
      auth_type: "basic",
      username: "smoke-user",
      password: "smoke-pass",
      webhook_secret: $secret
    },
    event_filters: []
  }')"
  local integration_id
  integration_id="$(create_integration "servicenow-webhook-smoke" "${payload}")" || return
  if [[ -z "${integration_id}" ]]; then
    fail "servicenow-webhook-smoke create did not return an integration id"
    return
  fi
  pass "servicenow-webhook-smoke create"

  local valid_body="${TMP_DIR}/servicenow.webhook.valid.json"
  local valid_status
  valid_status="$(curl_status "${valid_body}" -X POST \
    -H "Content-Type: application/json" \
    -H "X-ServiceNow-Token: ${secret}" \
    --data-binary "@${body}" \
    "${BASE_URL}/api/v1/integrations/servicenow/webhook")"
  if [[ "${valid_status}" == "200" ]]; then
    pass "servicenow webhook valid token"
  else
    fail "servicenow webhook valid token returned HTTP ${valid_status}"
  fi

  delete_integration "servicenow-webhook-smoke" "${integration_id}"
}

check_integration_management_flow() {
  check_paginated_endpoint "integrations" "/api/v1/integrations?page=1&per_page=5"
  check_array_data_endpoint_via_auth "integration-providers" "/api/v1/integrations/providers"
  check_oauth_start "slack" "/api/v1/integrations/slack/oauth/session" '{"name":"Smoke Slack Install"}'
  check_oauth_start "jira" "/api/v1/integrations/jira/oauth/session" '{"name":"Smoke Jira Install","project_key":"SEC"}'

  local payload
  payload="$(jq -cn --arg url "${BASE_URL}/api/v1/integrations/webhook/test-receiver" '{
    type: "webhook",
    name: "Smoke Webhook Integration",
    description: "Temporary smoke-test webhook integration",
    config: {
      url: $url,
      method: "POST",
      headers: {"X-Smoke-Test": "true"},
      secret: "smoke-webhook-secret",
      content_type: "application/json"
    },
    event_filters: []
  }')"

  local integration_id
  integration_id="$(create_integration "webhook-smoke" "${payload}")" || return
  if [[ -z "${integration_id}" ]]; then
    fail "webhook-smoke create did not return an integration id"
    return
  fi
  pass "webhook-smoke create"

  local get_body="${TMP_DIR}/webhook-smoke.get.json"
  local get_status
  get_status="$(curl_status "${get_body}" -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/api/v1/integrations/${integration_id}")"
  if [[ "${get_status}" == "200" ]]; then
    pass "webhook-smoke get"
  else
    fail "webhook-smoke get returned HTTP ${get_status}"
  fi

  local test_body="${TMP_DIR}/webhook-smoke.test.json"
  local test_status
  test_status="$(curl_status "${test_body}" -X POST -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/api/v1/integrations/${integration_id}/test")"
  if [[ "${test_status}" == "200" ]]; then
    pass "webhook-smoke test"
  else
    fail "webhook-smoke test returned HTTP ${test_status}"
  fi

  local delivery_body="${TMP_DIR}/webhook-smoke.deliveries.json"
  local delivery_status
  delivery_status="$(curl_status "${delivery_body}" -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/api/v1/integrations/${integration_id}/deliveries?page=1&per_page=5")"
  if [[ "${delivery_status}" == "200" ]] && assert_paginated_shape "${delivery_body}"; then
    pass "webhook-smoke deliveries"
  else
    fail "webhook-smoke deliveries returned HTTP ${delivery_status}"
  fi

  delete_integration "webhook-smoke" "${integration_id}"
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
    check_integration_management_flow
    check_slack_events_signature
    check_teams_invalid_token
    check_jira_webhooks
    check_servicenow_webhooks

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
