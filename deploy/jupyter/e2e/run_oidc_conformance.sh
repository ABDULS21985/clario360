#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
CONFORMANCE_DIR="${CONFORMANCE_DIR:-/tmp/oidc-conformance-suite}"
ENV_FILE="${ENV_FILE:-/tmp/clario360-jupyter-e2e.env}"
IAM_PORT="${IAM_PORT:-38081}"
FRONTEND_PORT="${FRONTEND_PORT:-33000}"
CLUSTER_NAME="${CLUSTER_NAME:-clario360-jupyter-e2e}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-clario360-oidc-conformance}"
CONFORMANCE_VENV="${CONFORMANCE_VENV:-/tmp/oidc-conformance-venv}"
MAVEN_CACHE="${MAVEN_CACHE:-/tmp/oidc-conformance-m2}"
RESULTS_DIR="${RESULTS_DIR:-/tmp/oidc-conformance-results}"

FRONTEND_LOG="$(mktemp)"
OVERRIDE_FILE="$(mktemp)"
CONFIG_FILE="$(mktemp)"
IAM_PORT_FORWARD_PID=""
FRONTEND_PID=""

cleanup() {
  if [[ -n "${FRONTEND_PID}" ]]; then
    kill "${FRONTEND_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${IAM_PORT_FORWARD_PID}" ]]; then
    kill "${IAM_PORT_FORWARD_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -d "${CONFORMANCE_DIR}" ]]; then
    (
      cd "${CONFORMANCE_DIR}"
      COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME}" docker compose \
        -f docker-compose-dev-mac.yml \
        -f "${OVERRIDE_FILE}" \
        down -v
    ) >/dev/null 2>&1 || true
  fi
  rm -f "${OVERRIDE_FILE}" "${CONFIG_FILE}" "${FRONTEND_LOG}"
}
trap cleanup EXIT

require_file() {
  [[ -e "$1" ]] || {
    echo "required file not found: $1" >&2
    exit 1
  }
}

require_file "${ENV_FILE}"
source "${ENV_FILE}"
export CLUSTER_NAME

log() {
  printf '\n[%s] %s\n' "$(date '+%H:%M:%S')" "$*"
}

wait_for_http() {
  local url="$1"
  local tries="${2:-90}"
  for _ in $(seq 1 "${tries}"); do
    if curl -k -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "timed out waiting for ${url}" >&2
  return 1
}

require_file "${CONFORMANCE_DIR}/scripts/run-test-plan.py"
mkdir -p "${MAVEN_CACHE}" "${RESULTS_DIR}"

log "Forwarding IAM service for browser and conformance traffic"
kubectl --context "kind-${CLUSTER_NAME}" -n clario360 port-forward service/iam-service "${IAM_PORT}:8081" >/tmp/clario360-oidc-iam-port-forward.log 2>&1 &
IAM_PORT_FORWARD_PID="$!"
sleep 4
wait_for_http "http://localhost:${IAM_PORT}/.well-known/openid-configuration" 90

log "Starting local frontend login surface"
(
  cd "${ROOT_DIR}/frontend"
  AUTH_COOKIE_DOMAIN=localhost \
  AUTH_COOKIE_SECURE=false \
  AUTH_COOKIE_SAMESITE=lax \
  NEXT_PUBLIC_API_URL="http://localhost:${IAM_PORT}" \
  npm run dev -- --hostname 0.0.0.0 --port "${FRONTEND_PORT}"
) >"${FRONTEND_LOG}" 2>&1 &
FRONTEND_PID="$!"
wait_for_http "http://localhost:${FRONTEND_PORT}/login" 180

log "Preparing Python runtime for the conformance runner"
if [[ ! -x "${CONFORMANCE_VENV}/bin/python" ]]; then
  python3 -m venv "${CONFORMANCE_VENV}"
fi
"${CONFORMANCE_VENV}/bin/pip" install --upgrade pip
"${CONFORMANCE_VENV}/bin/pip" install -r "${CONFORMANCE_DIR}/scripts/requirements.txt"

log "Building the OpenID Foundation conformance suite"
(
  cd "${CONFORMANCE_DIR}"
  MAVEN_CACHE="${MAVEN_CACHE}" docker compose -f builder-compose.yml run --rm builder
)

cat > "${OVERRIDE_FILE}" <<'EOF'
services:
  server:
    command: >
      java
      -Xdebug -Xrunjdwp:transport=dt_socket,address=*:9999,server=y,suspend=n
      -jar /server/fapi-test-suite.jar
      -Djdk.tls.maxHandshakeMessageSize=65536
      --fintechlabs.base_url=https://localhost:8443
      --fintechlabs.base_mtls_url=https://localhost:8444
      --fintechlabs.devmode=true
      --fintechlabs.startredir=true
EOF

log "Starting the conformance suite"
(
  cd "${CONFORMANCE_DIR}"
  COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME}" docker compose \
    -f docker-compose-dev-mac.yml \
    -f "${OVERRIDE_FILE}" \
    up -d
)
wait_for_http "https://localhost:8443/api/runner/available" 180

cat > "${CONFIG_FILE}" <<EOF
{
  "alias": "clario360-oidcc",
  "description": "Clario360 IAM OIDC basic authorization code flow",
  "server": {
    "discoveryUrl": "http://localhost:${IAM_PORT}/.well-known/openid-configuration"
  },
  "client": {
    "client_id": "oidc-conformance",
    "client_secret": "${CONFORMANCE_CLIENT_SECRET}",
    "scope": "openid profile email",
    "redirect_uri": "{BASEURL}test/a/clario360-oidcc/callback"
  },
  "browser": [
    {
      "match": "http://localhost:${FRONTEND_PORT}/login*",
      "tasks": [
        {
          "task": "Login",
          "optional": true,
          "match": "http://localhost:${FRONTEND_PORT}/login*",
          "commands": [
            ["text", "id", "email", "${TEST_EMAIL}"],
            ["text", "id", "password", "${TEST_PASSWORD}"],
            ["click", "css", "button[type='submit']"]
          ]
        }
      ]
    },
    {
      "match": "*/test/a/clario360-oidcc/callback*",
      "tasks": [
        {
          "task": "Verify Complete",
          "match": "*/test/a/clario360-oidcc/callback*",
          "commands": [
            ["wait", "id", "submission_complete", 10]
          ]
        }
      ]
    }
  ]
}
EOF

log "Running config certification plan"
(
  cd "${CONFORMANCE_DIR}"
  CONFORMANCE_SERVER='https://localhost:8443/' \
  CONFORMANCE_SERVER_MTLS='https://localhost:8444/' \
  CONFORMANCE_DEV_MODE=true \
  "${CONFORMANCE_VENV}/bin/python" scripts/run-test-plan.py --no-parallel --export-dir "${RESULTS_DIR}" \
    oidcc-config-certification-test-plan \
    "${CONFIG_FILE}"
)

log "Running basic discovery/static-client certification plan"
(
  cd "${CONFORMANCE_DIR}"
  CONFORMANCE_SERVER='https://localhost:8443/' \
  CONFORMANCE_SERVER_MTLS='https://localhost:8444/' \
  CONFORMANCE_DEV_MODE=true \
  "${CONFORMANCE_VENV}/bin/python" scripts/run-test-plan.py --no-parallel --export-dir "${RESULTS_DIR}" \
    'oidcc-basic-certification-test-plan[server_metadata=discovery][client_registration=static_client]' \
    "${CONFIG_FILE}"
)

log "OIDC conformance run complete"
echo "Results exported to ${RESULTS_DIR}"
