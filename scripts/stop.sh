#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Enterprise Platform Stop Script
# =============================================================================
# Gracefully shuts down all Clario 360 services and optionally the Docker
# infrastructure.
#
# Usage:
#   ./scripts/stop.sh                  # Stop all services (keep infra running)
#   ./scripts/stop.sh --infra          # Also stop Docker infrastructure
#   ./scripts/stop.sh --infra --purge  # Stop infra AND remove volumes (data wipe!)
#   ./scripts/stop.sh --service=name   # Stop a single named service
#   ./scripts/stop.sh --help           # Show usage
# =============================================================================
set -euo pipefail

# ── Directory layout ──────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PID_DIR="${REPO_ROOT}/.dev-pids"
LOG_DIR="${REPO_ROOT}/.dev-logs"

# ── Flags ─────────────────────────────────────────────────────────────────────
OPT_INFRA=false
OPT_PURGE=false
OPT_SINGLE=""
OPT_FORCE=false

for arg in "$@"; do
  case "$arg" in
    --infra)           OPT_INFRA=true ;;
    --purge)           OPT_PURGE=true ;;
    --force)           OPT_FORCE=true ;;
    --service=*)       OPT_SINGLE="${arg#--service=}" ;;
    --help|-h)
      cat <<'HELP'
Usage: ./scripts/stop.sh [OPTIONS]

Options:
  --infra          Also stop Docker infrastructure (Postgres, Redis, Kafka, etc.)
  --purge          Combined with --infra: remove Docker volumes (DESTROYS ALL DATA)
  --service=NAME   Stop only one service by name (e.g. --service=api-gateway)
  --force          Send SIGKILL instead of SIGTERM for stubborn processes
  --help           Show this message

Service names:
  api-gateway, iam-service, workflow-engine, audit-service, cyber-service,
  data-service, acta-service, lex-service, visus-service, notification-service,
  file-service, frontend
HELP
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

phase() { echo -e "\n${C_BOLD}${C_CYAN}━━━ $* ━━━${C_RESET}"; }
ok()    { echo -e "  ${C_GREEN}✓${C_RESET} $*"; }
warn()  { echo -e "  ${C_YELLOW}⚠${C_RESET} $*"; }
info()  { echo -e "  ${C_DIM}→${C_RESET} $*"; }

echo -e "\n${C_BOLD}${C_RED}╔══════════════════════════════════════════════════════╗${C_RESET}"
echo -e "${C_BOLD}${C_RED}║  Clario 360 — Graceful Shutdown                       ║${C_RESET}"
echo -e "${C_BOLD}${C_RED}╚══════════════════════════════════════════════════════╝${C_RESET}\n"

# ── Stop signal helper ────────────────────────────────────────────────────────
# stop_service NAME [GRACE_SECONDS]
#   Reads PID from .dev-pids/<name>.pid, sends SIGTERM, waits up to GRACE_SECONDS,
#   then SIGKILL if still running.
stop_service() {
  local name="$1"
  local grace="${2:-10}"
  local pid_file="${PID_DIR}/${name}.pid"

  if [ ! -f "${pid_file}" ]; then
    warn "${name}: no PID file found, skipping"
    return 0
  fi

  local pid
  pid=$(cat "${pid_file}")

  if ! kill -0 "${pid}" 2>/dev/null; then
    warn "${name}: process ${pid} not running (stale PID file)"
    rm -f "${pid_file}"
    return 0
  fi

  info "${name}: sending ${OPT_FORCE:+SIGKILL}${OPT_FORCE:-SIGTERM} to PID ${pid}"

  if $OPT_FORCE; then
    kill -9 "${pid}" 2>/dev/null || true
  else
    kill -TERM "${pid}" 2>/dev/null || true

    # Wait up to GRACE_SECONDS for clean exit
    local waited=0
    while kill -0 "${pid}" 2>/dev/null && [ "${waited}" -lt "${grace}" ]; do
      sleep 1
      waited=$((waited + 1))
    done

    # Escalate to SIGKILL if still alive
    if kill -0 "${pid}" 2>/dev/null; then
      warn "${name}: still running after ${grace}s, sending SIGKILL"
      kill -9 "${pid}" 2>/dev/null || true
      sleep 1
    fi
  fi

  if kill -0 "${pid}" 2>/dev/null; then
    warn "${name}: could not stop PID ${pid}"
    return 1
  fi

  ok "${name}: stopped (PID ${pid})"
  rm -f "${pid_file}"
  return 0
}

# ── Reverse startup order (gateway first, foundation last) ────────────────────
ALL_SERVICES=(
  api-gateway
  file-service
  visus-service
  lex-service
  acta-service
  data-service
  cyber-service
  workflow-engine
  notification-service
  audit-service
  iam-service
  frontend
)

# ── Stop services ─────────────────────────────────────────────────────────────
phase "Stopping Services"

if [ -n "${OPT_SINGLE}" ]; then
  # Single-service mode
  stop_service "${OPT_SINGLE}" 15
else
  # Full shutdown — stop in reverse dependency order
  ERRORS=0
  for svc in "${ALL_SERVICES[@]}"; do
    # Frontend gets a shorter grace period (Next.js exits quickly)
    grace=10
    [ "${svc}" = "frontend" ] && grace=5
    # Gateway gets longer (drain in-flight requests)
    [ "${svc}" = "api-gateway" ] && grace=20

    stop_service "${svc}" "${grace}" || ERRORS=$((ERRORS + 1))
  done

  if [ "${ERRORS}" -gt 0 ]; then
    warn "${ERRORS} service(s) could not be stopped cleanly"
  fi
fi

# ── Clean up any leftover PID files for dead processes ────────────────────────
if [ -d "${PID_DIR}" ]; then
  for pf in "${PID_DIR}"/*.pid; do
    [ -f "${pf}" ] || continue
    pid=$(cat "${pf}" 2>/dev/null || echo "")
    if [ -z "${pid}" ] || ! kill -0 "${pid}" 2>/dev/null; then
      svcname="$(basename "${pf}" .pid)"
      warn "${svcname}: removing stale PID file"
      rm -f "${pf}"
    fi
  done
fi

# ── Rotate logs ───────────────────────────────────────────────────────────────
phase "Rotating Logs"
if [ -d "${LOG_DIR}" ] && ls "${LOG_DIR}"/*.log 1>/dev/null 2>&1; then
  TS=$(date +%Y%m%d-%H%M%S)
  ARCHIVE_DIR="${LOG_DIR}/archive-${TS}"
  mkdir -p "${ARCHIVE_DIR}"
  mv "${LOG_DIR}"/*.log "${ARCHIVE_DIR}/" 2>/dev/null || true
  ok "Logs archived to ${ARCHIVE_DIR}"
else
  info "No active log files to archive"
fi

# ── Optionally stop Docker infrastructure ────────────────────────────────────
if $OPT_INFRA; then
  phase "Stopping Docker Infrastructure"
  if $OPT_PURGE; then
    echo -e "\n  ${C_RED}${C_BOLD}⚠  WARNING: --purge will DELETE all database volumes!${C_RESET}"
    echo -e "  ${C_RED}This will destroy all data in Postgres, Redis, and MinIO.${C_RESET}"
    echo ""
    read -r -p "  Type YES to confirm: " confirm
    if [ "${confirm}" != "YES" ]; then
      warn "Purge cancelled — infrastructure left running"
    else
      info "Stopping and removing all containers and volumes..."
      docker compose -f "${REPO_ROOT}/docker-compose.yml" down -v --remove-orphans 2>&1 | \
        sed 's/^/    /'
      ok "Docker infrastructure stopped and volumes removed"
    fi
  else
    info "Stopping Docker containers (data preserved)..."
    docker compose -f "${REPO_ROOT}/docker-compose.yml" down --remove-orphans 2>&1 | \
      sed 's/^/    /'
    ok "Docker infrastructure stopped (volumes retained)"
  fi
fi

# ── Final summary ─────────────────────────────────────────────────────────────
echo ""
echo -e "${C_BOLD}${C_CYAN}╔══════════════════════════════════════════════════════╗${C_RESET}"
echo -e "${C_BOLD}${C_CYAN}║  Shutdown Complete                                    ║${C_RESET}"
echo -e "${C_BOLD}${C_CYAN}╚══════════════════════════════════════════════════════╝${C_RESET}"
echo ""

# Report any processes still running on service ports
LINGERING=()
for port in 8080 8081 8083 8084 8085 8086 8087 8088 8089 8090 8091 3002 3000; do
  pid=$(lsof -ti tcp:"${port}" 2>/dev/null || true)
  if [ -n "${pid}" ]; then
    LINGERING+=("port ${port} → PID ${pid}")
  fi
done

if [ ${#LINGERING[@]} -gt 0 ]; then
  echo -e "  ${C_YELLOW}Lingering processes on service ports:${C_RESET}"
  for entry in "${LINGERING[@]}"; do
    echo -e "    ${C_YELLOW}${entry}${C_RESET}"
  done
  echo ""
  echo -e "  Run ${C_BOLD}./scripts/stop.sh --force${C_RESET} to SIGKILL all remaining processes"
else
  echo -e "  ${C_GREEN}All service ports are free.${C_RESET}"
fi

if $OPT_INFRA; then
  echo ""
  echo -e "  To restart: ${C_BOLD}./scripts/start.sh${C_RESET}"
else
  echo ""
  echo -e "  Infrastructure (Postgres/Redis/Kafka/MinIO) is still running."
  echo -e "  To restart services: ${C_BOLD}./scripts/start.sh --no-build${C_RESET}"
  echo -e "  To also stop infra:  ${C_BOLD}./scripts/stop.sh --infra${C_RESET}"
fi
echo ""
