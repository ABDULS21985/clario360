#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Enterprise Platform Startup Script
# =============================================================================
# Brings up the complete Clario 360 platform end-to-end:
#   Phase 0: Prerequisite checks
#   Phase 1: JWT key generation
#   Phase 2: Docker infrastructure (Postgres, Redis, Kafka, MinIO, Jaeger)
#   Phase 3: Database creation & schema migrations
#   Phase 4: Dev data seeding (first-run only)
#   Phase 5: Build all Go service binaries
#   Phase 6: Start all backend services
#   Phase 7: Start the Next.js frontend
#   Phase 8: Health-check all endpoints
#   Phase 9: Status summary
#
# Usage:
#   ./scripts/start.sh              # Full startup
#   ./scripts/start.sh --no-build   # Skip binary rebuild
#   ./scripts/start.sh --infra-only # Infrastructure + migrations only
#   ./scripts/start.sh --help       # Show usage
# =============================================================================
set -euo pipefail

# ── Directory layout ──────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BACKEND_DIR="${REPO_ROOT}/backend"
FRONTEND_DIR="${REPO_ROOT}/frontend"
SECRETS_DIR="${REPO_ROOT}/.dev-secrets"
BIN_DIR="${REPO_ROOT}/.dev-bin"
LOG_DIR="${REPO_ROOT}/.dev-logs"
PID_DIR="${REPO_ROOT}/.dev-pids"

mkdir -p "${SECRETS_DIR}" "${BIN_DIR}" "${LOG_DIR}" "${PID_DIR}"

# ── Flags ─────────────────────────────────────────────────────────────────────
OPT_NO_BUILD=false
OPT_INFRA_ONLY=false
OPT_SKIP_FRONTEND=false

for arg in "$@"; do
  case "$arg" in
    --no-build)      OPT_NO_BUILD=true ;;
    --infra-only)    OPT_INFRA_ONLY=true ;;
    --skip-frontend) OPT_SKIP_FRONTEND=true ;;
    --help|-h)
      echo "Usage: $0 [--no-build] [--infra-only] [--skip-frontend]"
      echo ""
      echo "  --no-build       Skip rebuilding Go binaries (use existing .dev-bin/)"
      echo "  --infra-only     Start infra + run migrations only (no services)"
      echo "  --skip-frontend  Do not start the Next.js frontend"
      exit 0 ;;
    *) echo "Unknown option: $arg" >&2; exit 1 ;;
  esac
done

# ── ANSI colours ──────────────────────────────────────────────────────────────
if [ -t 1 ]; then
  C_RESET='\033[0m'
  C_BOLD='\033[1m'
  C_GREEN='\033[0;32m'
  C_YELLOW='\033[0;33m'
  C_CYAN='\033[0;36m'
  C_RED='\033[0;31m'
  C_DIM='\033[2m'
else
  C_RESET='' C_BOLD='' C_GREEN='' C_YELLOW='' C_CYAN='' C_RED='' C_DIM=''
fi

# ── Logging helpers ───────────────────────────────────────────────────────────
phase()   { echo -e "\n${C_BOLD}${C_CYAN}━━━ $* ━━━${C_RESET}"; }
ok()      { echo -e "  ${C_GREEN}✓${C_RESET} $*"; }
warn()    { echo -e "  ${C_YELLOW}⚠${C_RESET} $*"; }
fail()    { echo -e "  ${C_RED}✗${C_RESET} $*" >&2; }
info()    { echo -e "  ${C_DIM}→${C_RESET} $*"; }
header()  { echo -e "\n${C_BOLD}${C_GREEN}╔══════════════════════════════════════════════════════╗${C_RESET}"; \
            echo -e "${C_BOLD}${C_GREEN}║  Clario 360 — Enterprise Platform Startup             ║${C_RESET}"; \
            echo -e "${C_BOLD}${C_GREEN}╚══════════════════════════════════════════════════════╝${C_RESET}\n"; }

# ── Configuration ─────────────────────────────────────────────────────────────
DB_HOST=localhost
DB_PORT=5432
DB_USER=clario
DB_PASS=clario_dev_pass
DB_NAME=clario360          # global DB for audit/notification services
REDIS_HOST=localhost
REDIS_PORT=6379
KAFKA_BROKERS=localhost:9094
JWT_PRIVATE_KEY="${SECRETS_DIR}/jwt-private.pem"
JWT_PUBLIC_KEY="${SECRETS_DIR}/jwt-public.pem"

# Service port assignments (Makefile canonical values)
declare -A SVC_HTTP_PORT=(
  [api-gateway]=8080
  [iam-service]=8081
  [workflow-engine]=8083
  [audit-service]=8084
  [cyber-service]=8085
  [data-service]=8086
  [acta-service]=8087
  [lex-service]=8088
  [visus-service]=8089
  [notification-service]=8090
  [file-service]=8091
)

# Admin/metrics ports (HTTP port + 1000)
declare -A SVC_ADMIN_PORT=(
  [api-gateway]=9080
  [iam-service]=9081
  [workflow-engine]=9083
  [audit-service]=9084
  [cyber-service]=9085
  [data-service]=9086
  [acta-service]=9087
  [lex-service]=9088
  [visus-service]=9089
  [notification-service]=9090
  [file-service]=9091
)

# Service startup order (dependencies first)
SERVICES=(
  iam-service
  audit-service
  notification-service
  workflow-engine
  cyber-service
  data-service
  acta-service
  lex-service
  visus-service
  file-service
  api-gateway
)

# ── PID helpers ───────────────────────────────────────────────────────────────
pid_file() { echo "${PID_DIR}/${1}.pid"; }
save_pid()  { echo "$2" > "$(pid_file "$1")"; }
read_pid()  { local f; f="$(pid_file "$1")"; [ -f "$f" ] && cat "$f"; }

# ── Docker helpers ────────────────────────────────────────────────────────────
container_healthy() {
  local name="$1"
  local status
  status=$(docker inspect --format='{{.State.Health.Status}}' "$name" 2>/dev/null || echo "missing")
  [ "$status" = "healthy" ]
}

container_running() {
  docker ps --format='{{.Names}}' 2>/dev/null | grep -qx "$1"
}

wait_healthy() {
  local name="$1" timeout="${2:-90}" elapsed=0
  while [ "$elapsed" -lt "$timeout" ]; do
    if container_healthy "$name"; then return 0; fi
    sleep 3; elapsed=$((elapsed + 3))
    echo -ne "  ${C_DIM}Waiting for ${name} (${elapsed}s)...${C_RESET}\r"
  done
  echo ""
  return 1
}

# ── Port check helper ─────────────────────────────────────────────────────────
port_open() { nc -z localhost "$1" 2>/dev/null; }

wait_port() {
  local port="$1" name="$2" timeout="${3:-30}" elapsed=0
  while [ "$elapsed" -lt "$timeout" ]; do
    if port_open "$port"; then return 0; fi
    sleep 1; elapsed=$((elapsed + 1))
  done
  return 1
}

# ── Database helpers (via docker exec) ───────────────────────────────────────
PG_CONTAINER=clario360-postgres

pg_exec() {
  # Run SQL against a specific database
  local db="$1"; shift
  docker exec -i "${PG_CONTAINER}" psql -U "${DB_USER}" -d "$db" -v ON_ERROR_STOP=0 -q "$@"
}

db_exists() {
  docker exec "${PG_CONTAINER}" psql -U "${DB_USER}" -tAc \
    "SELECT 1 FROM pg_database WHERE datname='$1'" 2>/dev/null | grep -q 1
}

create_db() {
  local dbname="$1"
  if ! db_exists "$dbname"; then
    docker exec "${PG_CONTAINER}" psql -U "${DB_USER}" -c \
      "CREATE DATABASE ${dbname};" 2>/dev/null
    docker exec "${PG_CONTAINER}" psql -U "${DB_USER}" -d "$dbname" -c \
      "CREATE EXTENSION IF NOT EXISTS pgcrypto; CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";" 2>/dev/null
    ok "Created database: ${dbname}"
  else
    info "Database exists: ${dbname}"
  fi
}

table_exists() {
  local db="$1" table="$2"
  docker exec "${PG_CONTAINER}" psql -U "${DB_USER}" -d "$db" -tAc \
    "SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='${table}'" \
    2>/dev/null | grep -q 1
}

run_migration() {
  # run_migration <db> <sql_file> <sentinel_table>
  local db="$1" file="$2" sentinel="${3:-}"
  if [ -n "$sentinel" ] && table_exists "$db" "$sentinel"; then
    info "Already migrated: $(basename "$file") (table '${sentinel}' exists)"
    return 0
  fi
  if docker exec -i "${PG_CONTAINER}" psql -U "${DB_USER}" -d "$db" \
       -v ON_ERROR_STOP=0 -q < "$file" 2>&1 | grep -iE 'ERROR' | grep -v 'already exists' | head -3; then
    warn "Migration had warnings: $(basename "$file")"
  fi
  ok "Migrated: $(basename "$file") → ${db}"
}

# =============================================================================
# PHASE 0: Prerequisites
# =============================================================================
phase "PHASE 0 — Prerequisite Checks"

MISSING=0
check_cmd() {
  if command -v "$1" &>/dev/null; then
    ok "$1 found: $(command -v "$1")"
  else
    fail "$1 not found — please install it"
    MISSING=$((MISSING + 1))
  fi
}

check_cmd docker
check_cmd openssl
check_cmd node
check_cmd npm

# Go (for building services)
if [ "${OPT_NO_BUILD}" = false ] && ! command -v go &>/dev/null; then
  fail "go not found — cannot build services (use --no-build if binaries exist)"
  MISSING=$((MISSING + 1))
fi

if [ "${MISSING}" -gt 0 ]; then
  fail "${MISSING} prerequisite(s) missing. Aborting."
  exit 1
fi

# Verify Docker daemon is running
if ! docker info &>/dev/null; then
  fail "Docker daemon is not running. Please start Docker."
  exit 1
fi
ok "Docker daemon is running"

# =============================================================================
# PHASE 1: JWT Key Generation
# =============================================================================
phase "PHASE 1 — JWT Key Generation"

if [ -f "${JWT_PRIVATE_KEY}" ] && [ -f "${JWT_PUBLIC_KEY}" ]; then
  ok "JWT keys already exist at ${SECRETS_DIR}/"
else
  info "Generating RSA-2048 JWT key pair..."
  openssl genrsa -out "${JWT_PRIVATE_KEY}" 2048 2>/dev/null
  openssl rsa -in "${JWT_PRIVATE_KEY}" -pubout -out "${JWT_PUBLIC_KEY}" 2>/dev/null
  chmod 600 "${JWT_PRIVATE_KEY}"
  ok "Generated JWT key pair at ${SECRETS_DIR}/"
fi

# Generate dev encryption keys (deterministic for dev, stored in secrets dir)
ENC_KEY_FILE="${SECRETS_DIR}/encryption.key"
if [ ! -f "${ENC_KEY_FILE}" ]; then
  openssl rand -base64 32 > "${ENC_KEY_FILE}"
  ok "Generated encryption key"
fi
ENCRYPTION_KEY=$(cat "${ENC_KEY_FILE}")

DATA_ENC_KEY_FILE="${SECRETS_DIR}/data-encryption.key"
if [ ! -f "${DATA_ENC_KEY_FILE}" ]; then
  openssl rand -base64 32 > "${DATA_ENC_KEY_FILE}"
  ok "Generated data service encryption key"
fi
DATA_ENCRYPTION_KEY=$(cat "${DATA_ENC_KEY_FILE}")

FILE_ENC_KEY_FILE="${SECRETS_DIR}/file-encryption.key"
if [ ! -f "${FILE_ENC_KEY_FILE}" ]; then
  openssl rand -base64 32 > "${FILE_ENC_KEY_FILE}"
  ok "Generated file service encryption key"
fi
FILE_ENCRYPTION_KEY=$(cat "${FILE_ENC_KEY_FILE}")

WEBHOOK_SECRET_FILE="${SECRETS_DIR}/webhook-hmac.key"
if [ ! -f "${WEBHOOK_SECRET_FILE}" ]; then
  openssl rand -hex 32 > "${WEBHOOK_SECRET_FILE}"
fi
WEBHOOK_HMAC_SECRET=$(cat "${WEBHOOK_SECRET_FILE}")

JWT_PRIVATE_PEM=$(cat "${JWT_PRIVATE_KEY}")
JWT_PUBLIC_PEM=$(cat "${JWT_PUBLIC_KEY}")

# =============================================================================
# PHASE 2: Docker Infrastructure
# =============================================================================
phase "PHASE 2 — Docker Infrastructure"

cd "${REPO_ROOT}"

# Start only the essential services (skip grafana/prometheus/clamav/keycloak for speed)
INFRA_SERVICES="postgres redis kafka minio jaeger"

info "Starting infrastructure services: ${INFRA_SERVICES}"
docker compose up -d ${INFRA_SERVICES} 2>&1 | grep -v "^#" || true

# Wait for critical services to become healthy
echo ""
info "Waiting for PostgreSQL to be healthy..."
if wait_healthy "${PG_CONTAINER}" 60; then
  ok "PostgreSQL is healthy"
else
  warn "PostgreSQL health check timed out — proceeding anyway"
fi

info "Waiting for Redis..."
if wait_healthy "clario360-redis" 30; then
  ok "Redis is healthy"
else
  # Redis might not have healthcheck; check port instead
  if wait_port ${REDIS_PORT} "Redis" 15; then
    ok "Redis port ${REDIS_PORT} is open"
  else
    warn "Redis not detected on port ${REDIS_PORT} — services may fail"
  fi
fi

info "Waiting for Kafka..."
# Kafka/Redpanda may already be running; check port
if port_open 9094; then
  ok "Kafka port 9094 is reachable"
else
  if wait_healthy "clario360-kafka" 90; then
    ok "Kafka is healthy"
  else
    warn "Kafka health check timed out — Kafka-dependent features may be unavailable"
  fi
fi

info "Waiting for MinIO..."
if wait_port 9000 "MinIO" 30; then
  ok "MinIO port 9000 is open"
else
  warn "MinIO not available — file service may run in degraded mode"
fi

ok "Infrastructure ready"

# =============================================================================
# PHASE 3: Database Creation & Migrations
# =============================================================================
phase "PHASE 3 — Database Creation & Schema Migrations"

info "Ensuring all service databases exist..."

# These databases are created by init-databases.sql on first docker-compose start
for db in platform_core cyber_db data_db acta_db lex_db visus_db "${DB_NAME}"; do
  create_db "$db"
done

echo ""
info "Running migrations for platform_core..."
MDIR="${BACKEND_DIR}/migrations/platform_core"
run_migration "platform_core" "${MDIR}/000001_init_schema.up.sql"       "tenants"
run_migration "platform_core" "${MDIR}/000002_rls.up.sql"               "tenants"
run_migration "platform_core" "${MDIR}/000003_tenant_onboarding.up.sql" "tenant_onboarding"
run_migration "platform_core" "${MDIR}/000004_ai_governance_schema.up.sql" "ai_models"
run_migration "platform_core" "${MDIR}/000005_ai_governance_rls.up.sql" "ai_models"

echo ""
info "Running migrations for cyber_db..."
MDIR="${BACKEND_DIR}/migrations/cyber_db"
run_migration "cyber_db" "${MDIR}/000001_init_schema.up.sql"
run_migration "cyber_db" "${MDIR}/000002_asset_inventory.up.sql"
run_migration "cyber_db" "${MDIR}/000003_threat_detection_engine.up.sql"
run_migration "cyber_db" "${MDIR}/000004_ctem_assessment_engine.up.sql"
run_migration "cyber_db" "${MDIR}/000005_risk_scoring_engine.up.sql"
run_migration "cyber_db" "${MDIR}/000006_asset_inventory_owner_text.up.sql"
run_migration "cyber_db" "${MDIR}/000007_remediation_dspm_vciso.up.sql"
run_migration "cyber_db" "${MDIR}/000008_remediation_prompt20_compat.up.sql"
run_migration "cyber_db" "${MDIR}/000009_dspm_score_precision_compat.up.sql"
run_migration "cyber_db" "${MDIR}/000010_dspm_legacy_schema_compat.up.sql"
run_migration "cyber_db" "${MDIR}/000011_rls.up.sql"

echo ""
info "Running migrations for data_db..."
MDIR="${BACKEND_DIR}/migrations/data_db"
run_migration "data_db" "${MDIR}/000001_init_schema.up.sql"
run_migration "data_db" "${MDIR}/000002_prompt23_data_source_engine.up.sql"
run_migration "data_db" "${MDIR}/000003_prompt24_pipeline_quality_contradictions.up.sql"
run_migration "data_db" "${MDIR}/000004_prompt25_lineage_darkdata_analytics_dashboard.up.sql"
run_migration "data_db" "${MDIR}/000005_rls.up.sql"
run_migration "data_db" "${MDIR}/000006_prompt51_connector_types.up.sql"

echo ""
info "Running migrations for acta_db..."
MDIR="${BACKEND_DIR}/migrations/acta_db"
run_migration "acta_db" "${MDIR}/000001_init_schema.up.sql"
run_migration "acta_db" "${MDIR}/000002_rls.up.sql"

echo ""
info "Running migrations for lex_db..."
MDIR="${BACKEND_DIR}/migrations/lex_db"
run_migration "lex_db" "${MDIR}/000001_init_schema.up.sql"
run_migration "lex_db" "${MDIR}/000002_rls.up.sql"

echo ""
info "Running migrations for visus_db..."
MDIR="${BACKEND_DIR}/migrations/visus_db"
run_migration "visus_db" "${MDIR}/000001_init_schema.up.sql"
run_migration "visus_db" "${MDIR}/000002_modular_schema_compat.up.sql"
run_migration "visus_db" "${MDIR}/000003_rls.up.sql"

echo ""
info "Running migrations for ${DB_NAME} (audit/notification)..."
MDIR="${BACKEND_DIR}/migrations/audit_db"
if [ -f "${MDIR}/000001_init_schema.up.sql" ]; then
  run_migration "${DB_NAME}" "${MDIR}/000001_init_schema.up.sql"
  run_migration "${DB_NAME}" "${MDIR}/000002_rls.up.sql"
fi
MDIR="${BACKEND_DIR}/migrations/notification_db"
if [ -f "${MDIR}/000001_init_schema.up.sql" ]; then
  run_migration "${DB_NAME}" "${MDIR}/000001_init_schema.up.sql"
  run_migration "${DB_NAME}" "${MDIR}/000002_rls.up.sql"
fi

ok "All migrations complete"

# =============================================================================
# PHASE 4: Dev Data Seeding (first run only)
# =============================================================================
phase "PHASE 4 — Dev Data Seeding"

USER_EXISTS=$(docker exec "${PG_CONTAINER}" psql -U "${DB_USER}" -d platform_core -tAc \
  "SELECT COUNT(*) FROM users WHERE email='admin@clario.dev'" 2>/dev/null || echo "0")
USER_EXISTS=$(echo "${USER_EXISTS}" | tr -d '[:space:]')

if [ "${USER_EXISTS}" = "0" ] || [ -z "${USER_EXISTS}" ]; then
  info "Seeding development tenant and admin user..."
  docker exec -i "${PG_CONTAINER}" psql -U "${DB_USER}" -d platform_core -q << 'ENDSQL'
-- Dev tenant
INSERT INTO tenants (id, name, slug, domain, status, subscription_tier)
VALUES (
  'aaaaaaaa-0000-0000-0000-000000000001',
  'Clario Dev',
  'clario-dev',
  'clario.dev',
  'active',
  'enterprise'
) ON CONFLICT (slug) DO NOTHING;

-- System roles for dev tenant
INSERT INTO roles (tenant_id, name, slug, description, is_system_role, permissions)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', 'Super Admin',       'super-admin',       'Full system access',       true, '["*"]'),
  ('aaaaaaaa-0000-0000-0000-000000000001', 'Tenant Admin',      'tenant-admin',      'Tenant administration',    true, '["tenant:*","users:*","roles:*","apikeys:*"]'),
  ('aaaaaaaa-0000-0000-0000-000000000001', 'Security Analyst',  'security-analyst',  'Cyber security ops',       true, '["cyber:read","cyber:write","alerts:*","remediation:read"]'),
  ('aaaaaaaa-0000-0000-0000-000000000001', 'Data Engineer',     'data-engineer',     'Data pipeline access',     true, '["data:*","pipelines:*","quality:read","lineage:read"]'),
  ('aaaaaaaa-0000-0000-0000-000000000001', 'Compliance Officer','compliance-officer','Compliance & quality',      true, '["*:read","quality:*","lineage:read"]'),
  ('aaaaaaaa-0000-0000-0000-000000000001', 'Viewer',            'viewer',            'Read-only access',         true, '["*:read"]')
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- Dev admin user (password: Cl@rio360Dev!)
INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, status, mfa_enabled, created_by)
VALUES (
  'bbbbbbbb-0000-0000-0000-000000000001',
  'aaaaaaaa-0000-0000-0000-000000000001',
  'admin@clario.dev',
  '$2a$12$cFmphkbPbg.iJLfQml3mBOKK.lFRu1FVrKa6gu2t7jbn5xkVdL3pm',
  'Admin',
  'Dev',
  'active',
  false,
  'bbbbbbbb-0000-0000-0000-000000000001'
) ON CONFLICT DO NOTHING;

-- Assign super-admin role
INSERT INTO user_roles (user_id, role_id, tenant_id)
SELECT 'bbbbbbbb-0000-0000-0000-000000000001', r.id, 'aaaaaaaa-0000-0000-0000-000000000001'
FROM roles r
WHERE r.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001' AND r.slug = 'super-admin'
ON CONFLICT DO NOTHING;
ENDSQL
  ok "Dev data seeded — admin@clario.dev / Cl@rio360Dev!"
else
  ok "Dev data already present (admin@clario.dev)"
fi

if [ "${OPT_INFRA_ONLY}" = true ]; then
  echo ""
  ok "Infrastructure-only mode — done."
  echo ""
  echo -e "${C_BOLD}  Databases ready. Services not started (--infra-only).${C_RESET}"
  exit 0
fi

# =============================================================================
# PHASE 5: Build All Go Service Binaries
# =============================================================================
phase "PHASE 5 — Build Go Service Binaries"

cd "${BACKEND_DIR}"

if [ "${OPT_NO_BUILD}" = true ]; then
  warn "Skipping build (--no-build). Using existing binaries in ${BIN_DIR}/"
else
  BUILD_SERVICES=(
    api-gateway iam-service workflow-engine audit-service
    cyber-service data-service acta-service lex-service
    visus-service notification-service file-service
  )
  BUILD_FAILED=()
  for svc in "${BUILD_SERVICES[@]}"; do
    info "Building ${svc}..."
    if GOWORK=off go build -o "${BIN_DIR}/${svc}" "./cmd/${svc}" 2>"${LOG_DIR}/${svc}-build.log"; then
      ok "Built ${svc}"
    else
      fail "Failed to build ${svc} — see ${LOG_DIR}/${svc}-build.log"
      BUILD_FAILED+=("${svc}")
    fi
  done

  if [ "${#BUILD_FAILED[@]}" -gt 0 ]; then
    fail "Build failures: ${BUILD_FAILED[*]}"
    echo "  Run: tail -50 ${LOG_DIR}/<service>-build.log for details"
    exit 1
  fi
fi

ok "All binaries ready in ${BIN_DIR}/"

# =============================================================================
# PHASE 6: Start Backend Services
# =============================================================================
phase "PHASE 6 — Start Backend Services"

# ── Shared environment ────────────────────────────────────────────────────────
export DATABASE_HOST="${DB_HOST}"
export DATABASE_PORT="${DB_PORT}"
export DATABASE_USER="${DB_USER}"
export DATABASE_PASSWORD="${DB_PASS}"
export DATABASE_NAME="${DB_NAME}"
export DATABASE_SSL_MODE="disable"
export DATABASE_MAX_OPEN_CONNS="25"
export DATABASE_MAX_IDLE_CONNS="10"
export DATABASE_CONN_MAX_LIFETIME="5m"

export REDIS_HOST="${REDIS_HOST}"
export REDIS_PORT="${REDIS_PORT}"
export REDIS_PASSWORD=""
export REDIS_DB="0"

export KAFKA_BROKERS="${KAFKA_BROKERS}"
export KAFKA_GROUP_ID="clario360"
export KAFKA_AUTO_OFFSET_RESET="earliest"

export AUTH_RSA_PRIVATE_KEY_PEM="${JWT_PRIVATE_PEM}"
export AUTH_RSA_PUBLIC_KEY_PEM="${JWT_PUBLIC_PEM}"
export AUTH_JWT_ISSUER="clario360"
export AUTH_JWT_ACCESS_TOKEN_TTL="15m"
export AUTH_JWT_REFRESH_TOKEN_TTL="168h"
export AUTH_BCRYPT_COST="12"

export ENCRYPTION_KEY="${ENCRYPTION_KEY}"

export MINIO_ENDPOINT="localhost:9000"
export MINIO_ACCESS_KEY="clario_minio"
export MINIO_SECRET_KEY="clario_minio_secret"
export MINIO_USE_SSL="false"
export MINIO_BUCKET="clario360"

export OBSERVABILITY_LOG_LEVEL="info"
export OBSERVABILITY_LOG_FORMAT="json"
export OBSERVABILITY_OTLP_ENDPOINT=""   # disable tracing export in dev (Jaeger still gets logs via OTEL if set)
export ENVIRONMENT="development"

# start_service <name>
start_service() {
  local name="$1"
  local bin="${BIN_DIR}/${name}"
  local log="${LOG_DIR}/${name}.log"
  local pidfile
  pidfile="$(pid_file "$name")"

  if [ ! -f "${bin}" ]; then
    warn "Binary not found: ${bin} — skipping ${name}"
    return 1
  fi

  # Check if already running
  if [ -f "${pidfile}" ]; then
    local existing_pid
    existing_pid=$(cat "${pidfile}" 2>/dev/null)
    if kill -0 "${existing_pid}" 2>/dev/null; then
      warn "${name} already running (PID ${existing_pid})"
      return 0
    fi
  fi

  nohup "${bin}" > "${log}" 2>&1 &
  local pid=$!
  save_pid "${name}" "${pid}"
  ok "Started ${name} (PID ${pid}) → ${log}"
}

# ── api-gateway ───────────────────────────────────────────────────────────────
GW_HTTP_PORT=${SVC_HTTP_PORT[api-gateway]}
GW_ADMIN_PORT=${SVC_ADMIN_PORT[api-gateway]}
export GW_HTTP_PORT GW_ADMIN_PORT
export GW_ENVIRONMENT="development"
export GW_CORS_ALLOWED_ORIGINS="http://localhost:3000,http://localhost:3001"
export GW_READ_TIMEOUT_SEC="15"
export GW_WRITE_TIMEOUT_SEC="60"
export GW_PROXY_TIMEOUT_SEC="30"
# Route gateway to correct service ports
export GW_SVC_URL_IAM="http://localhost:${SVC_HTTP_PORT[iam-service]}"
export GW_SVC_URL_AUDIT="http://localhost:${SVC_HTTP_PORT[audit-service]}"
export GW_SVC_URL_WORKFLOW="http://localhost:${SVC_HTTP_PORT[workflow-engine]}"
export GW_SVC_URL_NOTIFICATION="http://localhost:${SVC_HTTP_PORT[notification-service]}"
export GW_SVC_URL_FILE="http://localhost:${SVC_HTTP_PORT[file-service]}"
export GW_SVC_URL_CYBER="http://localhost:${SVC_HTTP_PORT[cyber-service]}"
export GW_SVC_URL_DATA="http://localhost:${SVC_HTTP_PORT[data-service]}"
export GW_SVC_URL_ACTA="http://localhost:${SVC_HTTP_PORT[acta-service]}"
export GW_SVC_URL_LEX="http://localhost:${SVC_HTTP_PORT[lex-service]}"
export GW_SVC_URL_VISUS="http://localhost:${SVC_HTTP_PORT[visus-service]}"

# ── iam-service (ports hardcoded in main.go; env vars for config only) ────────
# (No HTTP port env var — port 8081 is hardcoded; admin port fixed at 9081)

# ── workflow-engine ───────────────────────────────────────────────────────────
export WF_HTTP_PORT=${SVC_HTTP_PORT[workflow-engine]}
export WF_SERVICE_URLS="notification=http://localhost:${SVC_HTTP_PORT[notification-service]},cyber=http://localhost:${SVC_HTTP_PORT[cyber-service]}"

# ── audit-service ─────────────────────────────────────────────────────────────
export AUDIT_HTTP_PORT=${SVC_HTTP_PORT[audit-service]}
export AUDIT_MINIO_ENDPOINT="localhost:9000"
export AUDIT_MINIO_ACCESS_KEY="clario_minio"
export AUDIT_MINIO_SECRET_KEY="clario_minio_secret"
export AUDIT_MINIO_BUCKET="audit-exports"

# ── cyber-service ─────────────────────────────────────────────────────────────
export CYBER_HTTP_PORT=${SVC_HTTP_PORT[cyber-service]}
export CYBER_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/cyber_db?sslmode=disable"
export CYBER_REDIS_URL="redis://${REDIS_HOST}:${REDIS_PORT}/1"
export CYBER_KAFKA_BROKERS="${KAFKA_BROKERS}"
export CYBER_KAFKA_GROUP_ID="cyber-service"
export CYBER_JWT_PUBLIC_KEY_PATH="${JWT_PUBLIC_KEY}"

# ── data-service ──────────────────────────────────────────────────────────────
export DATA_HTTP_PORT=${SVC_HTTP_PORT[data-service]}
export DATA_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/data_db?sslmode=disable"
export DATA_REDIS_URL="redis://${REDIS_HOST}:${REDIS_PORT}/2"
export DATA_KAFKA_BROKERS="${KAFKA_BROKERS}"
export DATA_KAFKA_GROUP_ID="data-service"
export DATA_JWT_PUBLIC_KEY_PATH="${JWT_PUBLIC_KEY}"
export DATA_ENCRYPTION_KEY="${DATA_ENCRYPTION_KEY}"
export DATA_MINIO_ENDPOINT="localhost:9000"
export DATA_MINIO_ACCESS_KEY="clario_minio"
export DATA_MINIO_SECRET_KEY="clario_minio_secret"

# ── acta-service ──────────────────────────────────────────────────────────────
export ACTA_HTTP_PORT=${SVC_HTTP_PORT[acta-service]}
export ACTA_ADMIN_PORT=${SVC_ADMIN_PORT[acta-service]}
export ACTA_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/acta_db?sslmode=disable"
export ACTA_REDIS_ADDR="${REDIS_HOST}:${REDIS_PORT}"
export ACTA_KAFKA_BROKERS="${KAFKA_BROKERS}"
export ACTA_SEED_DEMO_DATA="false"

# ── lex-service ───────────────────────────────────────────────────────────────
export LEX_HTTP_PORT=${SVC_HTTP_PORT[lex-service]}
export LEX_ADMIN_PORT=${SVC_ADMIN_PORT[lex-service]}
export LEX_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/lex_db?sslmode=disable"
export LEX_REDIS_ADDR="${REDIS_HOST}:${REDIS_PORT}"
export LEX_KAFKA_BROKERS="${KAFKA_BROKERS}"
export LEX_SEED_DEMO_DATA="false"

# ── visus-service ─────────────────────────────────────────────────────────────
export VISUS_HTTP_PORT=${SVC_HTTP_PORT[visus-service]}
export VISUS_ADMIN_PORT=${SVC_ADMIN_PORT[visus-service]}
export VISUS_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/visus_db?sslmode=disable"
export VISUS_REDIS_ADDR="${REDIS_HOST}:${REDIS_PORT}"
export VISUS_KAFKA_BROKERS="${KAFKA_BROKERS}"
export VISUS_SEED_DEMO_DATA="false"
# Internal suite URLs for VISUS cross-suite data fetching
export VISUS_SUITE_CYBER_URL="http://localhost:${SVC_HTTP_PORT[cyber-service]}"
export VISUS_SUITE_DATA_URL="http://localhost:${SVC_HTTP_PORT[data-service]}"
export VISUS_SUITE_ACTA_URL="http://localhost:${SVC_HTTP_PORT[acta-service]}"
export VISUS_SUITE_LEX_URL="http://localhost:${SVC_HTTP_PORT[lex-service]}"

# ── notification-service ──────────────────────────────────────────────────────
export NOTIF_HTTP_PORT=${SVC_HTTP_PORT[notification-service]}
export NOTIF_EMAIL_PROVIDER="smtp"
export NOTIF_SMTP_HOST="localhost"
export NOTIF_SMTP_PORT="1025"
export NOTIF_SMTP_TLS_ENABLED="false"
export NOTIF_WEBHOOK_HMAC_SECRET="${WEBHOOK_HMAC_SECRET}"
export NOTIF_WS_ALLOWED_ORIGINS="http://localhost:3000"
export NOTIF_IAM_SERVICE_URL="http://localhost:${SVC_HTTP_PORT[iam-service]}"
export NOTIF_DATA_SERVICE_URL="http://localhost:${SVC_HTTP_PORT[data-service]}"
export NOTIF_ACTA_SERVICE_URL="http://localhost:${SVC_HTTP_PORT[acta-service]}"
export NOTIF_CYBER_SERVICE_URL="http://localhost:${SVC_HTTP_PORT[cyber-service]}"
export NOTIF_ENVIRONMENT="development"

# ── file-service ──────────────────────────────────────────────────────────────
# File service uses its own required env vars
export FILE_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/platform_core?sslmode=disable"
export FILE_REDIS_URL="redis://${REDIS_HOST}:${REDIS_PORT}/3"
export FILE_KAFKA_BROKERS="${KAFKA_BROKERS}"
export FILE_KAFKA_GROUP_ID="file-service"
export FILE_JWT_PUBLIC_KEY_PATH="${JWT_PUBLIC_KEY}"
export FILE_MINIO_ENDPOINT="localhost:9000"
export FILE_MINIO_ACCESS_KEY="clario_minio"
export FILE_MINIO_SECRET_KEY="clario_minio_secret"
export FILE_MINIO_USE_SSL="false"
export FILE_MINIO_BUCKET_PREFIX="clario360"
export FILE_ENCRYPTION_MASTER_KEY="${FILE_ENCRYPTION_KEY}"
export FILE_CLAMAV_ADDRESS="localhost:3310"
export FILE_ENVIRONMENT="development"

# ── Start all services ────────────────────────────────────────────────────────
for svc in "${SERVICES[@]}"; do
  start_service "${svc}"
  sleep 0.5   # brief pause between starts to avoid port-binding races
done

# =============================================================================
# PHASE 7: Start Next.js Frontend
# =============================================================================
phase "PHASE 7 — Start Frontend"

if [ "${OPT_SKIP_FRONTEND}" = false ]; then
  # Ensure .env.local is configured
  ENVLOCAL="${FRONTEND_DIR}/.env.local"
  if [ ! -f "${ENVLOCAL}" ]; then
    cat > "${ENVLOCAL}" << EOF
NEXT_PUBLIC_API_URL=http://localhost:${SVC_HTTP_PORT[api-gateway]}
NEXT_PUBLIC_APP_NAME=Clario 360
NEXT_PUBLIC_APP_URL=http://localhost:3000
NEXT_PUBLIC_APP_VERSION=0.1.0
AUTH_COOKIE_NAME=clario360
AUTH_COOKIE_SECURE=false
AUTH_COOKIE_DOMAIN=localhost
AUTH_COOKIE_SAMESITE=strict
AUTH_ACCESS_TOKEN_MAX_AGE=900
AUTH_REFRESH_TOKEN_MAX_AGE=604800
EOF
    ok "Created ${ENVLOCAL}"
  else
    # Update API URL to point at gateway if it's pointing elsewhere
    if grep -q "NEXT_PUBLIC_API_URL=http://localhost:8081" "${ENVLOCAL}"; then
      sed -i '' \
        "s|NEXT_PUBLIC_API_URL=http://localhost:8081|NEXT_PUBLIC_API_URL=http://localhost:${SVC_HTTP_PORT[api-gateway]}|" \
        "${ENVLOCAL}" 2>/dev/null || true
      ok "Updated .env.local to point to API gateway"
    fi
  fi

  FRONTEND_LOG="${LOG_DIR}/frontend.log"
  FRONTEND_PID_FILE="${PID_DIR}/frontend.pid"

  if [ -f "${FRONTEND_PID_FILE}" ]; then
    existing=$(cat "${FRONTEND_PID_FILE}" 2>/dev/null)
    if kill -0 "${existing}" 2>/dev/null; then
      warn "Frontend already running (PID ${existing})"
    else
      rm -f "${FRONTEND_PID_FILE}"
    fi
  fi

  if [ ! -f "${FRONTEND_PID_FILE}" ] || ! kill -0 "$(cat "${FRONTEND_PID_FILE}" 2>/dev/null)" 2>/dev/null; then
    cd "${FRONTEND_DIR}"
    # Install dependencies if node_modules missing
    if [ ! -d "node_modules" ]; then
      info "Installing frontend dependencies..."
      npm install --silent
    fi
    nohup npm run dev > "${FRONTEND_LOG}" 2>&1 &
    FRONTEND_PID=$!
    echo "${FRONTEND_PID}" > "${FRONTEND_PID_FILE}"
    ok "Started frontend (PID ${FRONTEND_PID}) → ${FRONTEND_LOG}"
  fi
  cd "${REPO_ROOT}"
fi

# =============================================================================
# PHASE 8: Health Checks
# =============================================================================
phase "PHASE 8 — Waiting for Services to Become Ready"

# Wait slightly for services to finish initializing
sleep 3

declare -A SVC_HEALTH_STATUS=()

check_service_health() {
  local name="$1" port="$2" path="${3:-/healthz}"
  local url="http://localhost:${port}${path}"
  local http_code
  http_code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 "${url}" 2>/dev/null || echo "000")
  if [ "${http_code}" = "200" ] || [ "${http_code}" = "204" ]; then
    SVC_HEALTH_STATUS["${name}"]="healthy"
    return 0
  else
    SVC_HEALTH_STATUS["${name}"]="degraded (HTTP ${http_code})"
    return 1
  fi
}

MAX_WAIT=45
info "Polling service health endpoints (up to ${MAX_WAIT}s per service)..."

declare -A HEALTH_ENDPOINTS=(
  [api-gateway]="/healthz"
  [iam-service]="/healthz"
  [workflow-engine]="/healthz"
  [audit-service]="/healthz"
  [cyber-service]="/healthz"
  [data-service]="/healthz"
  [acta-service]="/healthz"
  [lex-service]="/healthz"
  [visus-service]="/healthz"
  [notification-service]="/healthz"
  [file-service]="/healthz"
)

for svc in "${SERVICES[@]}"; do
  port="${SVC_HTTP_PORT[$svc]}"
  path="${HEALTH_ENDPOINTS[$svc]:-/healthz}"
  elapsed=0
  ready=false
  while [ "$elapsed" -lt "$MAX_WAIT" ]; do
    if check_service_health "$svc" "$port" "$path" 2>/dev/null; then
      ready=true; break
    fi
    sleep 2; elapsed=$((elapsed + 2))
    echo -ne "  ${C_DIM}Waiting for ${svc}:${port} (${elapsed}s)...${C_RESET}\r"
  done
  echo ""
  if [ "$ready" = true ]; then
    ok "${svc} → http://localhost:${port}"
  else
    # Service may still be starting (some take longer); mark as degraded but continue
    SVC_HEALTH_STATUS["${svc}"]="${SVC_HEALTH_STATUS[$svc]:-starting}"
    warn "${svc} not yet healthy (port ${port}) — may still be starting"
  fi
done

# Frontend health check
if [ "${OPT_SKIP_FRONTEND}" = false ]; then
  info "Waiting for frontend..."
  elapsed=0
  while [ "$elapsed" -lt 60 ]; do
    if curl -s -o /dev/null -w '%{http_code}' --max-time 3 "http://localhost:3000" 2>/dev/null | grep -q "200\|304"; then
      ok "Frontend → http://localhost:3000"; break
    fi
    sleep 2; elapsed=$((elapsed + 2))
    echo -ne "  ${C_DIM}Waiting for frontend (${elapsed}s)...${C_RESET}\r"
  done
  echo ""
fi

# =============================================================================
# PHASE 9: Status Summary
# =============================================================================
phase "PHASE 9 — Platform Status"

echo ""
echo -e "${C_BOLD}  ┌────────────────────────────────────────────────────────┐${C_RESET}"
echo -e "${C_BOLD}  │  Service              Port   Status                    │${C_RESET}"
echo -e "${C_BOLD}  ├────────────────────────────────────────────────────────┤${C_RESET}"

for svc in "${SERVICES[@]}"; do
  port="${SVC_HTTP_PORT[$svc]}"
  status="${SVC_HEALTH_STATUS[$svc]:-unknown}"
  if [ "$status" = "healthy" ]; then
    icon="${C_GREEN}●${C_RESET}"
    label="${C_GREEN}healthy${C_RESET}"
  elif echo "$status" | grep -qi "starting"; then
    icon="${C_YELLOW}◌${C_RESET}"
    label="${C_YELLOW}starting${C_RESET}"
  else
    icon="${C_RED}○${C_RESET}"
    label="${C_RED}${status}${C_RESET}"
  fi
  printf "  │  ${icon} %-22s%-7s%-24b │\n" "${svc}" ":${port}" "${label}"
done

if [ "${OPT_SKIP_FRONTEND}" = false ]; then
  printf "  │  ${C_GREEN}●${C_RESET} %-22s%-7s%-24b │\n" "frontend" ":3000" "${C_GREEN}next.js dev${C_RESET}"
fi

echo -e "${C_BOLD}  └────────────────────────────────────────────────────────┘${C_RESET}"

echo ""
echo -e "${C_BOLD}  Infrastructure${C_RESET}"
echo -e "  ${C_DIM}PostgreSQL${C_RESET}       → localhost:5432"
echo -e "  ${C_DIM}Redis${C_RESET}            → localhost:6379"
echo -e "  ${C_DIM}Kafka/Redpanda${C_RESET}   → localhost:9094"
echo -e "  ${C_DIM}MinIO API${C_RESET}        → localhost:9000"
echo -e "  ${C_DIM}MinIO Console${C_RESET}    → http://localhost:9001  (clario_minio / clario_minio_secret)"
echo -e "  ${C_DIM}Jaeger Tracing${C_RESET}   → http://localhost:16686"

echo ""
echo -e "${C_BOLD}  Dev Credentials${C_RESET}"
echo -e "  ${C_DIM}App URL${C_RESET}          → ${C_CYAN}http://localhost:3000${C_RESET}"
echo -e "  ${C_DIM}API Gateway${C_RESET}      → ${C_CYAN}http://localhost:8080${C_RESET}"
echo -e "  ${C_DIM}Email${C_RESET}            → admin@clario.dev"
echo -e "  ${C_DIM}Password${C_RESET}         → Cl@rio360Dev!"
echo -e "  ${C_DIM}Tenant${C_RESET}           → clario-dev"
echo -e "  ${C_DIM}Role${C_RESET}             → super-admin"

echo ""
echo -e "${C_BOLD}  Secrets & Logs${C_RESET}"
echo -e "  ${C_DIM}JWT keys${C_RESET}         → ${SECRETS_DIR}/"
echo -e "  ${C_DIM}Service logs${C_RESET}     → ${LOG_DIR}/"
echo -e "  ${C_DIM}PIDs${C_RESET}             → ${PID_DIR}/"

echo ""
echo -e "  Run ${C_CYAN}./scripts/stop.sh${C_RESET} to gracefully stop all services."
echo -e "  Run ${C_CYAN}./scripts/status.sh${C_RESET} to check live service health."
echo ""
