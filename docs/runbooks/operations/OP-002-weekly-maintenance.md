# OP-002: Weekly Maintenance Procedures

| Field              | Value                                |
|--------------------|--------------------------------------|
| **Runbook ID**     | OP-002                               |
| **Title**          | Weekly Maintenance Procedures        |
| **Frequency**      | Weekly (Sundays, 02:00 UTC)          |
| **Estimated Time** | ~45 minutes                          |
| **Owner**          | Platform Team                        |
| **Last Updated**   | 2026-03-08                           |
| **Review Cycle**   | Quarterly                            |

## Summary

This runbook covers the weekly maintenance procedures for the Clario 360 platform. These tasks ensure optimal database performance, clean Kafka state, controlled log growth, security posture, dependency health, full audit chain integrity, and awareness of performance trends. Execute all sections in order. Each section includes verification steps.

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export API_URL=https://api.clario360.io
export GRAFANA_URL=https://grafana.clario360.io
```

Ensure the daily checks ([OP-001](OP-001-daily-checks.md)) pass before starting weekly maintenance.

---

## Procedure 1: PostgreSQL ANALYZE on All Databases (~10 min)

Running `ANALYZE` updates table statistics used by the query planner, ensuring optimal query performance.

### 1a. Run ANALYZE on All Databases

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ANALYZE ${DB} ==="
  kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -c "ANALYZE VERBOSE;" 2>&1 | tail -5
  echo ""
done
```

**Expected output:** Each database reports `INFO: analyzing ...` messages for its tables, ending without errors.

### 1b. Check Table Statistics Freshness

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} - Tables with stale statistics ==="
  kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -c "
    SELECT schemaname, relname, last_analyze, last_autoanalyze,
           n_live_tup, n_dead_tup,
           CASE WHEN n_live_tup > 0
                THEN round(n_dead_tup::numeric / n_live_tup * 100, 1)
                ELSE 0
           END AS dead_pct
    FROM pg_stat_user_tables
    WHERE last_analyze IS NULL
       OR last_analyze < now() - interval '7 days'
    ORDER BY n_dead_tup DESC
    LIMIT 10;
  "
done
```

**Expected output:** After running ANALYZE, no tables should have `last_analyze` older than 7 days. `dead_pct` above 20% indicates the table needs VACUUM (see [OP-007](OP-007-database-maintenance.md)).

### 1c. Check for Bloated Tables

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} - Bloat check ==="
  kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -c "
    SELECT schemaname, relname,
           pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
           n_dead_tup,
           n_live_tup,
           last_vacuum,
           last_autovacuum
    FROM pg_stat_user_tables
    WHERE n_dead_tup > 10000
    ORDER BY n_dead_tup DESC
    LIMIT 5;
  "
done
```

**Expected output:** Tables with high dead tuple counts should have recent vacuum timestamps. If not, trigger a manual vacuum per [OP-007](OP-007-database-maintenance.md).

### Verification

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
  SELECT count(*) AS tables_analyzed_today
  FROM pg_stat_user_tables
  WHERE last_analyze >= now() - interval '1 hour';
"
```

**Expected output:** Count matches the number of tables in `platform_core`.

---

## Procedure 2: Kafka Consumer Lag Review and Topic Cleanup (~8 min)

### 2a. Consumer Lag Review

```bash
kubectl exec -n kafka deploy/kafka -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --all-groups 2>/dev/null | awk 'NR==1 || $6 > 0' | column -t
```

**Expected output:** Only rows with non-zero lag are shown. Lag should be under 100 for all groups. Sustained lag above 1000 requires investigation.

### 2b. Identify Inactive Consumer Groups

```bash
kubectl exec -n kafka deploy/kafka -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --list --state | grep -i empty
```

```bash
kubectl exec -n kafka deploy/kafka -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --all-groups --state 2>/dev/null | grep -i "empty"
```

**Expected output:** List any consumer groups with `Empty` state. These are groups with no active members. If a group has been empty for more than 2 weeks, it is a candidate for deletion.

### 2c. Delete Stale Consumer Groups (if identified)

Only delete groups confirmed as no longer needed:

```bash
# Replace <GROUP_NAME> with the actual stale consumer group name
kubectl exec -n kafka deploy/kafka -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --delete --group <GROUP_NAME>
```

### 2d. Topic Partition Health

```bash
kubectl exec -n kafka deploy/kafka -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --under-replicated-partitions
```

**Expected output:** No output (no under-replicated partitions).

```bash
kubectl exec -n kafka deploy/kafka -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --unavailable-partitions
```

**Expected output:** No output (no unavailable partitions).

### 2e. Topic Disk Usage

```bash
kubectl exec -n kafka deploy/kafka -- kafka-log-dirs.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --topic-list "audit-events,workflow-events,notification-events,cyber-events,data-events,acta-events,lex-events,visus-events" 2>/dev/null | jq -r '.brokers[] | .broker as $b | .logDirs[] | .partitions[] | "\($b)\t\(.partition)\t\(.size)"' | sort -t$'\t' -k3 -rn | head -20
```

**Expected output:** No single partition should be disproportionately large compared to others on the same topic.

### Verification

```bash
kubectl exec -n kafka deploy/kafka -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --all-groups 2>/dev/null | awk '$6 > 1000 {print "HIGH LAG: " $1 " topic=" $2 " partition=" $3 " lag=" $6}'
```

**Expected output:** No output (no high-lag consumers).

---

## Procedure 3: Log Volume Check and Rotation (~5 min)

### 3a. Check Log Volume by Service

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  POD=$(kubectl get pods -n ${NAMESPACE} -l app=${SERVICE} -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
  if [ -n "${POD}" ]; then
    LOG_SIZE=$(kubectl exec -n ${NAMESPACE} ${POD} -- du -sh /var/log/ 2>/dev/null || echo "N/A")
    echo "${SERVICE}: ${LOG_SIZE}"
  fi
done
```

### 3b. Check Log Volume in Loki/Elasticsearch

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(rate(log_bytes_total{namespace="clario360"}[7d]))by(app)' \
  2>/dev/null | jq -r '.data.result[] | .metric.app + ": " + (.value[1] | tostring) + " bytes/s"'
```

**Expected output:** No single service should produce more than 10x the log volume of others (may indicate debug logging left enabled).

### 3c. Verify Log Retention Policy

```bash
kubectl get configmap -n monitoring fluentd-config -o json | jq -r '.data["fluent.conf"]' | grep -A5 "retention\|rotate\|max_age"
```

**Expected output:** Retention is set to 30 days for standard logs, 90 days for audit logs.

### 3d. Check for Error Log Spikes

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(increase(log_messages_total{level="error",namespace="clario360"}[7d]))by(app)' \
  2>/dev/null | jq -r '.data.result[] | .metric.app + ": " + (.value[1] | tonumber | round | tostring) + " errors in last 7d"' | sort -t: -k2 -rn
```

**Expected output:** Error counts should be stable week-over-week. Investigate any service with a significant increase.

### Verification

Confirm no log PVC is above 70% usage:

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=kubelet_volume_stats_used_bytes{namespace="monitoring"}/kubelet_volume_stats_capacity_bytes{namespace="monitoring"}*100' \
  2>/dev/null | jq -r '.data.result[] | .metric.persistentvolumeclaim + ": " + (.value[1] | tonumber | round | tostring) + "% used"'
```

---

## Procedure 4: Security Scan Review (~5 min)

### 4a. Container Image Vulnerability Scan

```bash
kubectl get vulnerabilityreports -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.labels["trivy-operator.resource.name"] + " | Critical: " + (.report.summary.criticalCount | tostring) + " High: " + (.report.summary.highCount | tostring)'
```

**Expected output:** Zero `Critical` vulnerabilities. `High` vulnerabilities should be tracked and patched within SLA (7 days).

### 4b. Check for Failed Auth Attempts

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT date_trunc('day', created_at) AS day,
       count(*) AS failed_attempts,
       count(DISTINCT jsonb_extract_path_text(metadata, 'ip_address')) AS unique_ips
FROM audit_logs
WHERE action = 'auth.login.failed'
  AND created_at > now() - interval '7 days'
GROUP BY day
ORDER BY day DESC;
"
```

**Expected output:** Failed login attempts should be consistent. A spike of more than 3x the average may indicate a brute-force attack.

### 4c. Network Policy Audit

```bash
kubectl get networkpolicies -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.name + " | Pod Selector: " + (.spec.podSelector.matchLabels | to_entries | map(.key + "=" + .value) | join(","))'
```

**Expected output:** All 10 services have network policies applied. No `default-allow-all` policies.

### 4d. RBAC Audit

```bash
kubectl get rolebindings -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.name + " | " + (.subjects[]? | .kind + "/" + .name) + " -> " + .roleRef.name'
```

**Expected output:** Only expected service accounts and operator roles are bound. No unexpected bindings.

### Verification

```bash
kubectl get vulnerabilityreports -n ${NAMESPACE} -o json | jq '[.items[].report.summary.criticalCount] | add'
```

**Expected output:** `0` (zero critical vulnerabilities).

---

## Procedure 5: Dependency Vulnerability Check (~5 min)

### 5a. Backend Dependencies (Go)

```bash
kubectl exec -n ${NAMESPACE} deploy/api-gateway -- cat /app/go.sum 2>/dev/null | wc -l
```

Check for known vulnerabilities using the Go vulnerability database:

```bash
# Run from CI or a workstation with Go installed
cd /Users/mac/clario360/backend && GOWORK=off govulncheck ./...
```

### 5b. Frontend Dependencies (Node.js)

```bash
cd /Users/mac/clario360/frontend && npm audit --production 2>/dev/null | tail -20
```

**Expected output:** `0 vulnerabilities` or only low-severity advisories. Critical or high vulnerabilities must be patched within SLA.

### 5c. Container Base Image Updates

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  IMAGE=$(kubectl get deploy -n ${NAMESPACE} ${SERVICE} -o jsonpath='{.spec.template.spec.containers[0].image}')
  echo "${SERVICE}: ${IMAGE}"
done
```

**Expected output:** All images should use recent base image tags. Flag any image using a tag older than 30 days for update.

### Verification

Document the count of critical and high vulnerabilities and compare with the previous week. Escalate any new critical vulnerability.

---

## Procedure 6: Full Audit Chain Integrity Verification (~5 min)

The daily check (OP-001) verifies the last 1000 entries. The weekly check verifies the entire chain.

### 6a. Full Chain Verification

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
WITH chain AS (
  SELECT id, entry_hash, prev_hash,
         LAG(entry_hash) OVER (ORDER BY id) AS expected_prev_hash
  FROM audit_logs
)
SELECT count(*) AS total_entries,
       count(*) FILTER (WHERE expected_prev_hash IS NOT NULL AND prev_hash != expected_prev_hash) AS broken_links,
       min(id) FILTER (WHERE expected_prev_hash IS NOT NULL AND prev_hash != expected_prev_hash) AS first_broken_id
FROM chain;
"
```

**Expected output:** `broken_links` equals `0`. If non-zero, note the `first_broken_id` and follow [TS-009](../troubleshooting/TS-009-audit-chain-broken.md).

### 6b. Verify Audit Log Growth Consistency

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT date_trunc('day', created_at) AS day,
       count(*) AS entries,
       count(DISTINCT tenant_id) AS active_tenants,
       count(DISTINCT actor_id) AS active_users
FROM audit_logs
WHERE created_at > now() - interval '7 days'
GROUP BY day
ORDER BY day DESC;
"
```

**Expected output:** Consistent daily volumes. Significant deviations (>50% change) warrant investigation.

### 6c. Verify No Gaps in Audit Sequence

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
WITH numbered AS (
  SELECT id, LAG(id) OVER (ORDER BY id) AS prev_id
  FROM audit_logs
)
SELECT count(*) AS gaps
FROM numbered
WHERE prev_id IS NOT NULL AND id != prev_id + 1;
"
```

**Expected output:** The `gaps` count is `0`, confirming no audit entries have been deleted.

### Verification

Record the total entry count and broken link count in the weekly operations report.

---

## Procedure 7: Performance Trend Review in Grafana (~7 min)

### 7a. API Latency Trends (7-day)

Open the API Performance dashboard:

```
${GRAFANA_URL}/d/api-performance?orgId=1&from=now-7d&to=now
```

Or query via Prometheus:

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=histogram_quantile(0.95,sum(rate(http_request_duration_seconds_bucket{namespace="clario360"}[7d]))by(le,service))' \
  2>/dev/null | jq -r '.data.result[] | .metric.service + " p95: " + .value[1] + "s"'
```

**Expected output:** p95 latency below 500ms for all services. Increasing trends over the week warrant investigation.

### 7b. Error Rate Trends (7-day)

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(rate(http_requests_total{namespace="clario360",code=~"5.."}[7d]))by(service)/sum(rate(http_requests_total{namespace="clario360"}[7d]))by(service)*100' \
  2>/dev/null | jq -r '.data.result[] | .metric.service + " error rate: " + (.value[1] | tonumber | . * 100 | round / 100 | tostring) + "%"'
```

**Expected output:** Error rate below 0.1% for all services.

### 7c. Database Query Duration Trends

Open the Database Performance dashboard:

```
${GRAFANA_URL}/d/db-performance?orgId=1&from=now-7d&to=now
```

Or query via Prometheus:

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=histogram_quantile(0.95,sum(rate(pg_query_duration_seconds_bucket[7d]))by(le,datname))' \
  2>/dev/null | jq -r '.data.result[] | .metric.datname + " p95 query duration: " + .value[1] + "s"'
```

**Expected output:** p95 query duration below 100ms. Increasing trends indicate potential query optimization needed.

### 7d. Resource Utilization Trends

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=avg(rate(container_cpu_usage_seconds_total{namespace="clario360"}[7d]))by(container)*100' \
  2>/dev/null | jq -r '.data.result[] | .metric.container + " avg CPU: " + (.value[1] | tonumber | . * 100 | round / 100 | tostring) + "%"'
```

**Expected output:** Average CPU below 60% for all services. Services consistently above 70% are candidates for scaling.

### 7e. Kafka Throughput Trends

Open the Kafka Overview dashboard:

```
${GRAFANA_URL}/d/kafka-overview?orgId=1&from=now-7d&to=now
```

### Verification

Document observations in the weekly operations report. Flag any metrics showing degradation trends over 3+ consecutive weeks.

---

## Post-Maintenance Checklist

After completing all procedures, run the daily health checks to verify the platform is in a healthy state:

```bash
# Run OP-001 daily checks or the automated script
bash scripts/daily-check.sh
```

Record the following in the weekly operations report:

| Procedure | Status | Notes |
|-----------|--------|-------|
| PostgreSQL ANALYZE | PASS / FAIL | |
| Kafka Consumer Lag Review | PASS / FAIL | |
| Log Volume Check | PASS / FAIL | |
| Security Scan Review | PASS / FAIL | |
| Dependency Vulnerability Check | PASS / FAIL | |
| Audit Chain Integrity | PASS / FAIL | |
| Performance Trend Review | PASS / FAIL | |

## Related Runbooks

- [OP-001: Daily Checks](OP-001-daily-checks.md)
- [OP-003: Monthly Review](OP-003-monthly-review.md)
- [OP-007: Database Maintenance](OP-007-database-maintenance.md)
- [OP-008: Kafka Maintenance](OP-008-kafka-maintenance.md)
- [OP-009: Log Management](OP-009-log-management.md)
- [TS-009: Audit Chain Broken](../troubleshooting/TS-009-audit-chain-broken.md)

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2026-03-08 | Platform Team | Initial version |
