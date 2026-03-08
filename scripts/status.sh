#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Enterprise Platform Status Script
# =============================================================================
# Live health dashboard for all Clario 360 services and infrastructure.
#
# Usage:
#   ./scripts/status.sh           # One-shot status table
#   ./scripts/status.sh --watch   # Auto-refresh every 5 seconds (like watch)
#   ./scripts/status.sh --json    # Machine-readable JSON output
#   ./scripts/status.sh --help    # Show usage
# =============================================================================
set -euo pipefail

# ── Directory layout ──────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PID_DIR="${REPO_ROOT}/.dev-pids"

# ── Flags ─────────────────────────────────────────────────────────────────────
OPT_WATCH=false
OPT_JSON=false
WATCH_INTERVAL=5

for arg in "$@"; do
  case "$arg" in
    --watch)        OPT_WATCH=true ;;
    --json)         OPT_JSON=true ;;
    --interval=*)   WATCH_INTERVAL="${arg#--interval=}" ;;
    --help|-h)
      cat <<'HELP'
Usage: ./scripts/status.sh [OPTIONS]

Options:
  --watch           Auto-refresh the status table every 5 seconds (Ctrl+C to exit)
  --interval=N      Set watch refresh interval in seconds (default: 5)
  --json            Print machine-readable JSON (one-shot, ignores --watch)
  --help            Show this message

Exit codes:
  0  All services healthy
  1  One or more services unhealthy or unreachable
  2  No services running
HELP
      exit 0 ;;
    *) echo "Unknown option: $arg" >&2; exit 1 ;;
  esac
done

# ── ANSI colours ──────────────────────────────────────────────────────────────
if [ -t 1 ] && ! $OPT_JSON; then
  C_RESET='\033[0m'
  C_BOLD='\033[1m'
  C_GREEN='\033[0;32m'
  C_YELLOW='\033[0;33m'
  C_CYAN='\033[0;36m'
  C_RED='\033[0;31m'
  C_DIM='\033[2m'
  C_BLUE='\033[0;34m'
else
  C_RESET='' C_BOLD='' C_GREEN='' C_YELLOW='' C_CYAN='' C_RED='' C_DIM='' C_BLUE=''
fi

# ── Service definitions ───────────────────────────────────────────────────────
# Format: "name:http_port:health_path:label"
SERVICES=(
  "api-gateway:8080:/healthz:API Gateway"
  "iam-service:8081:/healthz:IAM Service"
  "workflow-engine:8083:/healthz:Workflow Engine"
  "audit-service:8084:/healthz:Audit Service"
  "cyber-service:8085:/healthz:Cyber Service"
  "data-service:8086:/healthz:Data Service"
  "acta-service:8087:/healthz:Acta Service"
  "lex-service:8088:/healthz:Lex Service"
  "visus-service:8089:/healthz:Visus Service"
  "notification-service:8090:/healthz:Notification Service"
  "file-service:8091:/healthz:File Service"
  "frontend:3000::Next.js Frontend"
)

# Infrastructure checks: "label:check_command"
INFRA=(
  "Postgres:pg_check"
  "Redis:redis_check"
  "Kafka:kafka_check"
  "MinIO:minio_check"
  "Jaeger:jaeger_check"
)

# ── Check helpers ─────────────────────────────────────────────────────────────

# Returns "UP", "DOWN", or "UNKNOWN"
http_check() {
  local url="$1"
  local http_code
  http_code=$(curl -sf -o /dev/null -w "%{http_code}" \
    --connect-timeout 2 --max-time 3 "${url}" 2>/dev/null || echo "000")
  if [ "${http_code}" -ge 200 ] 2>/dev/null && [ "${http_code}" -lt 400 ] 2>/dev/null; then
    echo "UP"
  elif [ "${http_code}" = "000" ]; then
    echo "DOWN"
  else
    echo "DEGRADED"
  fi
}

# Check if a port is open (TCP) without making an HTTP request
port_open() {
  local host="$1" port="$2"
  (echo >/dev/tcp/"${host}"/"${port}") 2>/dev/null && echo "UP" || echo "DOWN"
}

pg_check() {
  if command -v pg_isready &>/dev/null; then
    pg_isready -h localhost -p 5432 -U clario -q 2>/dev/null && echo "UP" || echo "DOWN"
  else
    port_open localhost 5432
  fi
}

redis_check() {
  if command -v redis-cli &>/dev/null; then
    result=$(redis-cli -h localhost -p 6379 ping 2>/dev/null || echo "")
    [ "${result}" = "PONG" ] && echo "UP" || echo "DOWN"
  else
    port_open localhost 6379
  fi
}

kafka_check() {
  port_open localhost 9094
}

minio_check() {
  http_check "http://localhost:9000/minio/health/live"
}

jaeger_check() {
  port_open localhost 16686
}

get_pid() {
  local name="$1"
  local pid_file="${PID_DIR}/${name}.pid"
  if [ -f "${pid_file}" ]; then
    local pid
    pid=$(cat "${pid_file}" 2>/dev/null || echo "")
    if [ -n "${pid}" ] && kill -0 "${pid}" 2>/dev/null; then
      echo "${pid}"
    else
      echo ""
    fi
  fi
}

# Memory usage for a PID in MB (macOS + Linux compatible)
get_mem_mb() {
  local pid="$1"
  if [[ "$(uname)" == "Darwin" ]]; then
    # macOS: ps rss is in KB
    ps -o rss= -p "${pid}" 2>/dev/null | awk '{printf "%.0f", $1/1024}'
  else
    # Linux: /proc/pid/status VmRSS in kB
    awk '/VmRSS/{printf "%.0f", $2/1024}' "/proc/${pid}/status" 2>/dev/null || echo "?"
  fi
}

# ── JSON output mode ──────────────────────────────────────────────────────────
if $OPT_JSON; then
  output=""
  any_down=false
  all_down=true

  output+='{"timestamp":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'","services":['

  first=true
  for svc_def in "${SERVICES[@]}"; do
    IFS=: read -r name port path label <<< "${svc_def}"
    pid=$(get_pid "${name}")

    if [ -n "${path}" ]; then
      state=$(http_check "http://localhost:${port}${path}")
    else
      state=$(port_open localhost "${port}")
    fi

    [ "${state}" = "UP" ] && all_down=false
    [ "${state}" != "UP" ] && any_down=true

    $first || output+=","
    first=false
    output+='{"name":"'"${name}"'","label":"'"${label}"'","port":'"${port}"',"status":"'"${state}"'","pid":'"${pid:-null}"'}'
  done

  output+='],"infrastructure":['

  first=true
  for infra_def in "${INFRA[@]}"; do
    IFS=: read -r label check_fn <<< "${infra_def}"
    state=$("${check_fn}")

    [ "${state}" = "UP" ] && all_down=false
    [ "${state}" != "UP" ] && any_down=true

    $first || output+=","
    first=false
    output+='{"label":"'"${label}"'","status":"'"${state}"'"}'
  done
  output+=']}'

  echo "${output}"

  $all_down && exit 2
  $any_down && exit 1
  exit 0
fi

# ── Table rendering ───────────────────────────────────────────────────────────
render_status() {
  local timestamp
  timestamp=$(date '+%Y-%m-%d %H:%M:%S')

  # Clear screen in watch mode
  $OPT_WATCH && tput clear 2>/dev/null || true

  echo -e "${C_BOLD}${C_CYAN}╔══════════════════════════════════════════════════════════════════╗${C_RESET}"
  printf "${C_BOLD}${C_CYAN}║  %-64s║${C_RESET}\n" "Clario 360 — Platform Status  ${timestamp}"
  echo -e "${C_BOLD}${C_CYAN}╚══════════════════════════════════════════════════════════════════╝${C_RESET}"

  # ── Infrastructure ────────────────────────────────────────────────────────
  echo -e "\n${C_BOLD}  Infrastructure${C_RESET}"
  echo -e "  ${C_DIM}$(printf '─%.0s' {1..62})${C_RESET}"
  printf "  ${C_BOLD}%-22s %-10s %-8s${C_RESET}\n" "Component" "Status" "Port"
  echo -e "  ${C_DIM}$(printf '─%.0s' {1..62})${C_RESET}"

  infra_all_up=true
  for infra_def in "${INFRA[@]}"; do
    IFS=: read -r label check_fn <<< "${infra_def}"
    state=$("${check_fn}")

    case "$state" in
      UP)      icon="${C_GREEN}●${C_RESET}" badge="${C_GREEN}UP${C_RESET}" ;;
      DOWN)    icon="${C_RED}●${C_RESET}"   badge="${C_RED}DOWN${C_RESET}" infra_all_up=false ;;
      *)       icon="${C_YELLOW}●${C_RESET}" badge="${C_YELLOW}${state}${C_RESET}" infra_all_up=false ;;
    esac

    # Map label to port for display
    case "${label}" in
      Postgres) port_display="5432" ;;
      Redis)    port_display="6379" ;;
      Kafka)    port_display="9094" ;;
      MinIO)    port_display="9000" ;;
      Jaeger)   port_display="16686" ;;
      *)        port_display="—" ;;
    esac

    printf "  ${icon} %-20s %-20b %s\n" "${label}" "${badge}" "${port_display}"
  done

  # ── Services ──────────────────────────────────────────────────────────────
  echo -e "\n${C_BOLD}  Services${C_RESET}"
  echo -e "  ${C_DIM}$(printf '─%.0s' {1..62})${C_RESET}"
  printf "  ${C_BOLD}%-26s %-10s %-6s %-10s %-8s${C_RESET}\n" "Service" "Status" "Port" "PID" "Mem(MB)"
  echo -e "  ${C_DIM}$(printf '─%.0s' {1..62})${C_RESET}"

  svc_up=0
  svc_total=0
  for svc_def in "${SERVICES[@]}"; do
    IFS=: read -r name port path label <<< "${svc_def}"
    svc_total=$((svc_total + 1))

    pid=$(get_pid "${name}")

    if [ -n "${path}" ]; then
      state=$(http_check "http://localhost:${port}${path}")
    else
      # Frontend: just check port is open
      state=$(port_open localhost "${port}")
    fi

    mem_display="—"
    if [ -n "${pid}" ]; then
      mem=$(get_mem_mb "${pid}")
      [ -n "${mem}" ] && mem_display="${mem}"
    fi

    pid_display="${pid:-—}"

    case "${state}" in
      UP)
        icon="${C_GREEN}●${C_RESET}"
        badge="${C_GREEN}UP${C_RESET}      "
        svc_up=$((svc_up + 1))
        ;;
      DOWN)
        icon="${C_RED}●${C_RESET}"
        badge="${C_RED}DOWN${C_RESET}    "
        ;;
      DEGRADED)
        icon="${C_YELLOW}●${C_RESET}"
        badge="${C_YELLOW}DEGRADED${C_RESET}"
        ;;
      *)
        icon="${C_DIM}●${C_RESET}"
        badge="${C_DIM}UNKNOWN${C_RESET} "
        ;;
    esac

    printf "  ${icon} %-24s %-20b %-6s %-10s %s\n" \
      "${label}" "${badge}" "${port}" "${pid_display}" "${mem_display}"
  done

  echo -e "  ${C_DIM}$(printf '─%.0s' {1..62})${C_RESET}"

  # ── Summary ───────────────────────────────────────────────────────────────
  svc_down=$((svc_total - svc_up))
  echo ""

  if [ "${svc_up}" -eq "${svc_total}" ] && $infra_all_up; then
    echo -e "  ${C_BOLD}${C_GREEN}✓ All ${svc_total} services + infrastructure healthy${C_RESET}"
  elif [ "${svc_up}" -eq 0 ]; then
    echo -e "  ${C_BOLD}${C_RED}✗ No services running  (run ./scripts/start.sh)${C_RESET}"
  else
    echo -e "  ${C_BOLD}${C_YELLOW}⚠ ${svc_up}/${svc_total} services UP — ${svc_down} service(s) down${C_RESET}"
    if ! $infra_all_up; then
      echo -e "  ${C_BOLD}${C_RED}✗ Infrastructure issues detected${C_RESET}"
    fi
  fi

  # ── Useful endpoints ──────────────────────────────────────────────────────
  echo ""
  echo -e "  ${C_BOLD}Endpoints:${C_RESET}"
  echo -e "  ${C_DIM}→${C_RESET} Frontend:      ${C_BLUE}http://localhost:3000${C_RESET}"
  echo -e "  ${C_DIM}→${C_RESET} API Gateway:   ${C_BLUE}http://localhost:8080${C_RESET}"
  echo -e "  ${C_DIM}→${C_RESET} MinIO Console: ${C_BLUE}http://localhost:9001${C_RESET}  (admin/clario_minio_secret)"
  echo -e "  ${C_DIM}→${C_RESET} Jaeger UI:     ${C_BLUE}http://localhost:16686${C_RESET}"
  echo -e "  ${C_DIM}→${C_RESET} Gateway /metrics: ${C_BLUE}http://localhost:9080/metrics${C_RESET}"

  if $OPT_WATCH; then
    echo ""
    echo -e "  ${C_DIM}Refreshing every ${WATCH_INTERVAL}s — Ctrl+C to exit${C_RESET}"
  fi
}

# ── Main ──────────────────────────────────────────────────────────────────────
if $OPT_WATCH; then
  # Trap Ctrl+C for clean exit
  trap 'echo -e "\n"; exit 0' INT

  while true; do
    render_status
    sleep "${WATCH_INTERVAL}"
  done
else
  render_status

  # Exit code reflects overall health
  any_down=false
  all_down=true

  for svc_def in "${SERVICES[@]}"; do
    IFS=: read -r name port path label <<< "${svc_def}"
    if [ -n "${path}" ]; then
      state=$(http_check "http://localhost:${port}${path}")
    else
      state=$(port_open localhost "${port}")
    fi
    [ "${state}" = "UP" ] && all_down=false
    [ "${state}" != "UP" ] && any_down=true
  done

  $all_down && exit 2
  $any_down && exit 1
  exit 0
fi
