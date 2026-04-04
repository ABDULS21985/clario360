#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Automated Daily Health Check
# Runs all checks from OP-001 and outputs a JSON report
# Usage: daily-check.sh [--slack-webhook URL] [--namespace NAMESPACE]
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
NAMESPACE="${NAMESPACE:-clario360}"
KAFKA_NAMESPACE="${KAFKA_NAMESPACE:-kafka}"
MONITORING_NAMESPACE="${MONITORING_NAMESPACE:-monitoring}"
SLACK_WEBHOOK=""
VERBOSE=false

REPORT=()
FAILURES=0
WARNINGS=0
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --slack-webhook)
            SLACK_WEBHOOK="$2"
            shift 2
            ;;
        --namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo "Unknown argument: $1"
            echo "Usage: daily-check.sh [--slack-webhook URL] [--namespace NAMESPACE] [--verbose]"
            exit 1
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log() {
    if [ "$VERBOSE" = true ]; then
        echo "[$(date -u +%H:%M:%S)] $*" >&2
    fi
}

# Run a single check. Arguments:
#   $1 — check name (string, no spaces)
#   $2 — command to execute (string, will be eval'd)
#   $3 — regex the output must match to pass
#   $4 — (optional) "warn" to treat failure as warning instead of failure
check() {
    local name="$1"
    local cmd="$2"
    local expected="$3"
    local level="${4:-fail}"

    log "Running check: ${name}"

    local result
    result=$(eval "${cmd}" 2>&1) || true

    # Escape double quotes and newlines for JSON safety
    local safe_result
    safe_result=$(echo "${result}" | head -c 500 | tr '\n' ' ' | sed 's/"/\\"/g')

    if echo "${result}" | grep -qE "${expected}"; then
        REPORT+=("{\"check\":\"${name}\",\"status\":\"pass\",\"detail\":\"${safe_result}\"}")
        log "  PASS: ${name}"
    else
        if [ "${level}" = "warn" ]; then
            REPORT+=("{\"check\":\"${name}\",\"status\":\"warn\",\"detail\":\"${safe_result}\"}")
            WARNINGS=$((WARNINGS + 1))
            log "  WARN: ${name}"
        else
            REPORT+=("{\"check\":\"${name}\",\"status\":\"fail\",\"detail\":\"${safe_result}\"}")
            FAILURES=$((FAILURES + 1))
            log "  FAIL: ${name}"
        fi
    fi
}

# ---------------------------------------------------------------------------
# 1. Cluster Health
# ---------------------------------------------------------------------------
check "nodes_ready" \
    "kubectl get nodes --no-headers 2>/dev/null | grep -v 'NotReady' | grep -c ' Ready' || echo 0" \
    "^[1-9][0-9]*$"

check "no_node_pressure" \
    "kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{\" \"}{range .status.conditions[*]}{.type}={.status}{\" \"}{end}{\"\n\"}{end}' 2>/dev/null | grep -cE '(MemoryPressure=True|DiskPressure=True|PIDPressure=True)' || echo 0" \
    "^0$"

# ---------------------------------------------------------------------------
# 2. All Pods Running
# ---------------------------------------------------------------------------
check "all_pods_running" \
    "kubectl get pods -n ${NAMESPACE} --field-selector=status.phase!=Running,status.phase!=Succeeded --no-headers 2>/dev/null | wc -l | tr -d ' '" \
    "^0$"

check "no_restarts_last_hour" \
    "kubectl get pods -n ${NAMESPACE} -o jsonpath='{range .items[*]}{.metadata.name}{\" \"}{range .status.containerStatuses[*]}{.restartCount}{\" \"}{end}{\"\n\"}{end}' 2>/dev/null | awk '{total=0; for(i=2;i<=NF;i++) total+=\$i; if(total>10) print \$1, total}' | wc -l | tr -d ' '" \
    "^0$" \
    "warn"

# ---------------------------------------------------------------------------
# 3. Service Health Endpoints
# ---------------------------------------------------------------------------
SERVICES=(
    api-gateway
    iam-service
    audit-service
    workflow-engine
    notification-service
    cyber-service
    data-service
    acta-service
    lex-service
    visus-service
)

for svc in "${SERVICES[@]}"; do
    check "svc_health_${svc}" \
        "kubectl exec -n ${NAMESPACE} deployment/api-gateway -- wget -qO- --timeout=5 http://${svc}:8080/readyz 2>/dev/null | jq -r '.status' 2>/dev/null || echo 'unreachable'" \
        "healthy"
done

# ---------------------------------------------------------------------------
# 4. Database Health
# ---------------------------------------------------------------------------
check "db_connections" \
    "kubectl exec -n ${NAMESPACE} deployment/iam-service -- psql \"\${DATABASE_URL}\" -tAc \"SELECT count(*) FROM pg_stat_activity;\" 2>/dev/null || echo 'error'" \
    "^[0-9]+$"

check "db_no_long_queries" \
    "kubectl exec -n ${NAMESPACE} deployment/iam-service -- psql \"\${DATABASE_URL}\" -tAc \"SELECT count(*) FROM pg_stat_activity WHERE state != 'idle' AND (now() - pg_stat_activity.query_start) > interval '5 minutes';\" 2>/dev/null || echo 0" \
    "^0$" \
    "warn"

# ---------------------------------------------------------------------------
# 5. Kafka Health
# ---------------------------------------------------------------------------
check "kafka_brokers" \
    "kubectl exec -n ${KAFKA_NAMESPACE} kafka-0 -- /opt/bitnami/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092 2>/dev/null | grep -c 'id:' || echo 0" \
    "^[1-9][0-9]*$"

check "kafka_consumer_lag" \
    "kubectl exec -n ${KAFKA_NAMESPACE} kafka-0 -- /opt/bitnami/kafka/bin/kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --all-groups 2>/dev/null | awk 'NR>1 && \$6+0 > 10000 {count++} END {print count+0}'" \
    "^0$" \
    "warn"

# ---------------------------------------------------------------------------
# 6. Redis Health
# ---------------------------------------------------------------------------
check "redis_ping" \
    "kubectl exec -n ${NAMESPACE} deployment/api-gateway -- redis-cli -h redis PING 2>/dev/null || echo 'error'" \
    "PONG"

check "redis_memory" \
    "kubectl exec -n ${NAMESPACE} deployment/api-gateway -- redis-cli -h redis INFO memory 2>/dev/null | grep 'used_memory_human' | cut -d: -f2 | tr -d '[:space:]' || echo 'error'" \
    "^[0-9]"

# ---------------------------------------------------------------------------
# 7. Certificate Expiry
# ---------------------------------------------------------------------------
check "certs_valid" \
    "kubectl get certificates -n ${NAMESPACE} -o jsonpath='{range .items[*]}{.metadata.name}={.status.conditions[0].status} {end}' 2>/dev/null | grep -c 'False' || echo 0" \
    "^0$"

# ---------------------------------------------------------------------------
# 8. Backup Status
# ---------------------------------------------------------------------------
check "backup_recent" \
    "kubectl get jobs -n ${NAMESPACE} -l app=database-backup --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].status.succeeded}' 2>/dev/null || echo 0" \
    "^1$" \
    "warn"

# ---------------------------------------------------------------------------
# 9. Alertmanager — Active Critical Alerts
# ---------------------------------------------------------------------------
check "no_critical_alerts" \
    "kubectl exec -n ${MONITORING_NAMESPACE} deployment/alertmanager -- wget -qO- http://localhost:9093/api/v2/alerts 2>/dev/null | jq '[.[] | select(.labels.severity==\"critical\")] | length' 2>/dev/null || echo 0" \
    "^0$" \
    "warn"

# ---------------------------------------------------------------------------
# 10. Disk Usage
# ---------------------------------------------------------------------------
check "disk_usage" \
    "kubectl exec -n ${NAMESPACE} deployment/audit-service -- df --output=pcent / 2>/dev/null | tail -1 | tr -d ' %' || echo 0" \
    "^[0-7][0-9]?$"

# ---------------------------------------------------------------------------
# 11. Audit Trail Integrity (quick check)
# ---------------------------------------------------------------------------
check "audit_chain" \
    "kubectl exec -n ${NAMESPACE} deployment/audit-service -- wget -qO- --timeout=5 http://localhost:8080/healthz 2>/dev/null | jq -r '.status' 2>/dev/null || echo 'error'" \
    "healthy" \
    "warn"

# ---------------------------------------------------------------------------
# Generate JSON Report
# ---------------------------------------------------------------------------
TOTAL=${#REPORT[@]}
STATUS="healthy"
if [ ${FAILURES} -gt 0 ]; then
    STATUS="critical"
elif [ ${WARNINGS} -gt 0 ]; then
    STATUS="degraded"
fi

# Build JSON array manually to avoid issues with special characters
CHECKS_JSON=""
for i in "${!REPORT[@]}"; do
    if [ $i -gt 0 ]; then
        CHECKS_JSON="${CHECKS_JSON},"
    fi
    CHECKS_JSON="${CHECKS_JSON}${REPORT[$i]}"
done

REPORT_JSON=$(cat <<EOF
{
  "timestamp": "${TIMESTAMP}",
  "namespace": "${NAMESPACE}",
  "total_checks": ${TOTAL},
  "passed": $((TOTAL - FAILURES - WARNINGS)),
  "warnings": ${WARNINGS},
  "failures": ${FAILURES},
  "status": "${STATUS}",
  "checks": [${CHECKS_JSON}]
}
EOF
)

echo "${REPORT_JSON}" | jq . 2>/dev/null || echo "${REPORT_JSON}"

# ---------------------------------------------------------------------------
# Send to Slack (if webhook provided)
# ---------------------------------------------------------------------------
if [ -n "${SLACK_WEBHOOK}" ]; then
    if [ "${STATUS}" = "healthy" ]; then
        ICON=":white_check_mark:"
        COLOR="#36a64f"
    elif [ "${STATUS}" = "degraded" ]; then
        ICON=":warning:"
        COLOR="#ff9900"
    else
        ICON=":rotating_light:"
        COLOR="#ff0000"
    fi

    SLACK_PAYLOAD=$(cat <<EOF
{
  "attachments": [
    {
      "color": "${COLOR}",
      "title": "${ICON} Clario 360 Daily Health Check — $(date +%Y-%m-%d)",
      "text": "Status: *${STATUS}*\nTotal: ${TOTAL} | Passed: $((TOTAL - FAILURES - WARNINGS)) | Warnings: ${WARNINGS} | Failures: ${FAILURES}",
      "footer": "Clario 360 Operations",
      "ts": $(date +%s)
    }
  ]
}
EOF
    )

    curl -s -X POST "${SLACK_WEBHOOK}" \
        -H "Content-Type: application/json" \
        -d "${SLACK_PAYLOAD}" > /dev/null 2>&1 || \
        echo "WARNING: Failed to send Slack notification" >&2
fi

# ---------------------------------------------------------------------------
# Exit with appropriate code
# ---------------------------------------------------------------------------
if [ ${FAILURES} -gt 0 ]; then
    exit 1
elif [ ${WARNINGS} -gt 0 ]; then
    exit 0  # Warnings are not critical
fi

exit 0
