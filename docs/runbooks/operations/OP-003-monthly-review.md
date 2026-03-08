# OP-003: Monthly Operational Review

| Field              | Value                                    |
|--------------------|------------------------------------------|
| **Runbook ID**     | OP-003                                   |
| **Title**          | Monthly Operational Review               |
| **Frequency**      | Monthly (first Monday of each month)     |
| **Estimated Time** | ~2 hours                                 |
| **Owner**          | Platform Team Lead                       |
| **Last Updated**   | 2026-03-08                               |
| **Review Cycle**   | Quarterly                                |
| **Audience**       | Platform Team, Engineering Leadership    |

## Summary

This runbook defines the monthly operational review for the Clario 360 platform. It produces a comprehensive monthly operations report covering capacity, performance, security, cost, SLA compliance, and incident lessons learned. The output is a written report shared with engineering leadership.

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export API_URL=https://api.clario360.io
export GRAFANA_URL=https://grafana.clario360.io
```

Set the reporting period:

```bash
export REVIEW_START=$(date -u -v-1m +%Y-%m-01T00:00:00Z 2>/dev/null || date -u -d 'last month' +%Y-%m-01T00:00:00Z)
export REVIEW_END=$(date -u +%Y-%m-01T00:00:00Z)
echo "Review period: ${REVIEW_START} to ${REVIEW_END}"
```

---

## Section 1: Capacity Review — CPU, Memory, Disk Trends (~20 min)

### 1a. Node-Level CPU Utilization (30-day trend)

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  "http://localhost:9090/api/v1/query?query=avg_over_time(instance:node_cpu_utilisation:rate5m[30d])*100" \
  2>/dev/null | jq -r '.data.result[] | .metric.instance + ": " + (.value[1] | tonumber | round | tostring) + "% avg CPU"'
```

**Target:** Average CPU utilization below 60%. Nodes consistently above 70% indicate scaling is needed.

### 1b. Node-Level Memory Utilization (30-day trend)

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  "http://localhost:9090/api/v1/query?query=avg_over_time(instance:node_memory_utilisation:ratio[30d])*100" \
  2>/dev/null | jq -r '.data.result[] | .metric.instance + ": " + (.value[1] | tonumber | round | tostring) + "% avg memory"'
```

**Target:** Average memory utilization below 70%. Nodes consistently above 80% need scaling.

### 1c. Per-Service Resource Usage vs Requests/Limits

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(rate(container_cpu_usage_seconds_total{namespace="clario360",container!="POD",container!=""}[30d]))by(container)/sum(kube_pod_container_resource_requests{namespace="clario360",resource="cpu"})by(container)*100' \
  2>/dev/null | jq -r '.data.result[] | .metric.container + " CPU request utilization: " + (.value[1] | tonumber | round | tostring) + "%"'
```

**Target:** CPU request utilization between 50%-80%. Below 30% means over-provisioned; above 90% means under-provisioned.

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(container_memory_working_set_bytes{namespace="clario360",container!="POD",container!=""})by(container)/sum(kube_pod_container_resource_requests{namespace="clario360",resource="memory"})by(container)*100' \
  2>/dev/null | jq -r '.data.result[] | .metric.container + " memory request utilization: " + (.value[1] | tonumber | round | tostring) + "%"'
```

### 1d. Disk Usage Trends

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=kubelet_volume_stats_used_bytes{namespace="clario360"}/kubelet_volume_stats_capacity_bytes{namespace="clario360"}*100' \
  2>/dev/null | jq -r '.data.result[] | .metric.persistentvolumeclaim + ": " + (.value[1] | tonumber | round | tostring) + "% used"'
```

**Target:** All PVCs below 70%. Plan expansion for any above 60% with a growth trend.

### 1e. Database Growth Rate

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -c "
    SELECT '${DB}' AS database,
           pg_size_pretty(pg_database_size('${DB}')) AS current_size,
           pg_database_size('${DB}') AS size_bytes;
  " -t
done
```

Compare with last month's recorded sizes. Calculate growth rate:

```bash
# Record these values each month to track trend
# Growth rate = (current_size - last_month_size) / last_month_size * 100
```

### 1f. Capacity Forecast

Open Grafana capacity planning view:

```
${GRAFANA_URL}/d/cluster-overview?orgId=1&from=now-90d&to=now
```

Document: At current growth rates, how many months until each resource hits 80% utilization?

### Verification

Record all capacity metrics in the monthly report. Flag any resource above 70% utilization for capacity planning action.

---

## Section 2: Performance Review — p50/p95/p99 Latency Trends (~15 min)

### 2a. API Latency Percentiles (30-day)

```bash
for PERCENTILE in 0.5 0.95 0.99; do
  echo "=== p$(echo "${PERCENTILE} * 100" | bc | cut -d. -f1) Latency ==="
  kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
    "http://localhost:9090/api/v1/query?query=histogram_quantile(${PERCENTILE},sum(rate(http_request_duration_seconds_bucket{namespace=\"clario360\"}[30d]))by(le,service))" \
    2>/dev/null | jq -r '.data.result[] | .metric.service + ": " + (.value[1] | tonumber | . * 1000 | round | tostring) + "ms"'
  echo ""
done
```

**Targets:**
- p50 below 100ms
- p95 below 500ms
- p99 below 1000ms

### 2b. Per-Service Throughput (requests/sec)

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(rate(http_requests_total{namespace="clario360"}[30d]))by(service)' \
  2>/dev/null | jq -r '.data.result[] | .metric.service + ": " + (.value[1] | tonumber | . * 100 | round / 100 | tostring) + " req/s"'
```

### 2c. Error Rates by Service (30-day)

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(rate(http_requests_total{namespace="clario360",code=~"5.."}[30d]))by(service)/sum(rate(http_requests_total{namespace="clario360"}[30d]))by(service)*100' \
  2>/dev/null | jq -r '.data.result[] | .metric.service + " error rate: " + (.value[1] | tonumber | . * 1000 | round / 1000 | tostring) + "%"'
```

**Target:** Error rate below 0.1% for all services.

### 2d. Slowest API Endpoints

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=topk(10,histogram_quantile(0.95,sum(rate(http_request_duration_seconds_bucket{namespace="clario360"}[30d]))by(le,handler,service)))' \
  2>/dev/null | jq -r '.data.result[] | .metric.service + " " + .metric.handler + ": " + (.value[1] | tonumber | . * 1000 | round | tostring) + "ms"' | sort -t: -k2 -rn | head -10
```

**Target:** No endpoint above 2000ms p95. Endpoints above this threshold should be optimized.

### 2e. Database Query Performance (30-day)

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} - Top 5 Slowest Queries ==="
  kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -c "
    SELECT round(mean_exec_time::numeric, 2) AS avg_ms,
           calls,
           round(total_exec_time::numeric, 2) AS total_ms,
           left(query, 100) AS query_preview
    FROM pg_stat_statements
    ORDER BY mean_exec_time DESC
    LIMIT 5;
  "
done
```

### Verification

Open Grafana and visually confirm trends match the queried data:

```
${GRAFANA_URL}/d/api-performance?orgId=1&from=now-30d&to=now
```

---

## Section 3: Security Review — Auth Failures, Blocked Requests (~15 min)

### 3a. Failed Authentication Attempts (30-day)

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT date_trunc('week', created_at) AS week,
       count(*) AS failed_logins,
       count(DISTINCT jsonb_extract_path_text(metadata, 'ip_address')) AS unique_ips,
       count(DISTINCT jsonb_extract_path_text(metadata, 'email')) AS unique_emails
FROM audit_logs
WHERE action = 'auth.login.failed'
  AND created_at BETWEEN '${REVIEW_START}' AND '${REVIEW_END}'
GROUP BY week
ORDER BY week;
"
```

**Target:** Stable or declining trend. A sudden spike indicates potential brute-force activity.

### 3b. Blocked Requests (Rate Limiting)

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(increase(http_requests_total{namespace="clario360",code="429"}[30d]))by(service)' \
  2>/dev/null | jq -r '.data.result[] | .metric.service + ": " + (.value[1] | tonumber | round | tostring) + " rate-limited requests"'
```

### 3c. WAF/Firewall Events

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(increase(waf_blocked_requests_total[30d]))by(rule)' \
  2>/dev/null | jq -r '.data.result[] | .metric.rule + ": " + (.value[1] | tonumber | round | tostring) + " blocked"'
```

### 3d. Privilege Escalation Attempts

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT action, count(*) AS attempts
FROM audit_logs
WHERE action IN ('auth.privilege_escalation', 'rbac.unauthorized', 'auth.token.invalid')
  AND created_at BETWEEN '${REVIEW_START}' AND '${REVIEW_END}'
GROUP BY action
ORDER BY attempts DESC;
"
```

**Target:** Zero privilege escalation attempts. Any non-zero count requires investigation.

### 3e. Certificate Status Review

```bash
kubectl get certificates -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.name + " | Expires: " + .status.notAfter + " | Ready: " + (.status.conditions[] | select(.type=="Ready") | .status)'
```

### 3f. Vulnerability Summary

```bash
kubectl get vulnerabilityreports -n ${NAMESPACE} -o json | jq '{
  total: (.items | length),
  critical: [.items[].report.summary.criticalCount] | add,
  high: [.items[].report.summary.highCount] | add,
  medium: [.items[].report.summary.mediumCount] | add,
  low: [.items[].report.summary.lowCount] | add
}'
```

**Target:** Zero critical vulnerabilities. Track high vulnerabilities with remediation timeline.

### Verification

Document all security findings. Escalate any critical items to the security team.

---

## Section 4: Cost Review — Resource Utilization Efficiency (~10 min)

### 4a. Namespace Resource Consumption

```bash
kubectl top pods -n ${NAMESPACE} --sort-by=cpu | head -20
```

```bash
kubectl top pods -n ${NAMESPACE} --sort-by=memory | head -20
```

### 4b. Over-Provisioned Services

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(rate(container_cpu_usage_seconds_total{namespace="clario360",container!="POD",container!=""}[30d]))by(container)/sum(kube_pod_container_resource_limits{namespace="clario360",resource="cpu"})by(container)*100' \
  2>/dev/null | jq -r '.data.result[] | select((.value[1] | tonumber) < 20) | .metric.container + " uses only " + (.value[1] | tonumber | round | tostring) + "% of CPU limit - OVER-PROVISIONED"'
```

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(container_memory_working_set_bytes{namespace="clario360",container!="POD",container!=""})by(container)/sum(kube_pod_container_resource_limits{namespace="clario360",resource="memory"})by(container)*100' \
  2>/dev/null | jq -r '.data.result[] | select((.value[1] | tonumber) < 20) | .metric.container + " uses only " + (.value[1] | tonumber | round | tostring) + "% of memory limit - OVER-PROVISIONED"'
```

**Action:** Services using less than 20% of their CPU or memory limits should have their resource requests/limits reduced.

### 4c. PVC Usage vs Provisioned

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=kubelet_volume_stats_used_bytes{namespace="clario360"}/kubelet_volume_stats_capacity_bytes{namespace="clario360"}*100' \
  2>/dev/null | jq -r '.data.result[] | select((.value[1] | tonumber) < 30) | .metric.persistentvolumeclaim + ": " + (.value[1] | tonumber | round | tostring) + "% used - consider downsizing"'
```

### 4d. Node Count and Utilization

```bash
kubectl get nodes -o json | jq -r '.items | length | tostring + " nodes in cluster"'
kubectl top nodes 2>/dev/null
```

### 4e. Cost Optimization Recommendations

Based on the above data, document:

1. Services to right-size (reduce requests/limits)
2. PVCs to resize
3. Node pool adjustments (scale down if nodes are under-utilized)
4. Reserved instance or committed use discount opportunities

### Verification

Calculate estimated monthly savings from recommended optimizations.

---

## Section 5: SLA Compliance Report (~10 min)

### 5a. Overall Availability

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=(1-sum(rate(http_requests_total{namespace="clario360",code=~"5.."}[30d]))/sum(rate(http_requests_total{namespace="clario360"}[30d])))*100' \
  2>/dev/null | jq -r '.data.result[] | "Overall availability: " + (.value[1] | tostring) + "%"'
```

**Target:** 99.9% availability (SLA commitment). Monthly error budget: 43.2 minutes of downtime.

### 5b. Per-Service Availability

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=(1-sum(rate(http_requests_total{namespace="clario360",code=~"5.."}[30d]))by(service)/sum(rate(http_requests_total{namespace="clario360"}[30d]))by(service))*100' \
  2>/dev/null | jq -r '.data.result[] | .metric.service + ": " + (.value[1] | tostring) + "% available"'
```

### 5c. Error Budget Consumption

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(increase(http_requests_total{namespace="clario360",code=~"5.."}[30d]))/sum(increase(http_requests_total{namespace="clario360"}[30d]))*100' \
  2>/dev/null | jq -r '.data.result[] | "Error budget consumed: " + (.value[1] | tonumber | . * 1000 | round / 1000 | tostring) + "% (budget: 0.1%)"'
```

### 5d. Downtime Events

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT created_at, action, severity,
       jsonb_extract_path_text(metadata, 'service') AS service,
       jsonb_extract_path_text(metadata, 'duration_seconds') AS duration_s,
       jsonb_extract_path_text(metadata, 'description') AS description
FROM audit_logs
WHERE action LIKE 'incident.%'
  AND created_at BETWEEN '${REVIEW_START}' AND '${REVIEW_END}'
ORDER BY created_at;
"
```

### 5e. Latency SLA Compliance

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - \
  'http://localhost:9090/api/v1/query?query=sum(rate(http_request_duration_seconds_bucket{namespace="clario360",le="0.5"}[30d]))/sum(rate(http_request_duration_seconds_count{namespace="clario360"}[30d]))*100' \
  2>/dev/null | jq -r '.data.result[] | "Requests under 500ms: " + (.value[1] | tonumber | round | tostring) + "% (target: 95%)"'
```

**Target:** 95% of requests complete within 500ms.

### Verification

Produce an SLA summary table:

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| Availability | 99.9% | __%    | PASS/FAIL |
| p95 Latency  | <500ms | __ms  | PASS/FAIL |
| Error Rate   | <0.1%  | __%   | PASS/FAIL |

---

## Section 6: Incident Post-Mortem Review (~15 min)

### 6a. List Incidents from the Month

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT id, created_at, severity,
       jsonb_extract_path_text(metadata, 'title') AS title,
       jsonb_extract_path_text(metadata, 'status') AS status,
       jsonb_extract_path_text(metadata, 'ttd_minutes') AS time_to_detect,
       jsonb_extract_path_text(metadata, 'ttr_minutes') AS time_to_resolve
FROM audit_logs
WHERE action = 'incident.created'
  AND created_at BETWEEN '${REVIEW_START}' AND '${REVIEW_END}'
ORDER BY severity, created_at;
"
```

### 6b. Incident Metrics

Document:

| Metric | Value |
|--------|-------|
| Total incidents | __ |
| P1 (Critical) | __ |
| P2 (High) | __ |
| P3 (Medium) | __ |
| P4 (Low) | __ |
| Mean Time to Detect (MTTD) | __ min |
| Mean Time to Resolve (MTTR) | __ min |
| Repeat incidents | __ |

### 6c. Review Action Items from Post-Mortems

Check that action items from previous post-mortems have been completed:

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT id, created_at,
       jsonb_extract_path_text(metadata, 'incident_id') AS incident_id,
       jsonb_extract_path_text(metadata, 'action_item') AS action_item,
       jsonb_extract_path_text(metadata, 'status') AS status,
       jsonb_extract_path_text(metadata, 'assignee') AS assignee
FROM audit_logs
WHERE action = 'incident.action_item'
  AND jsonb_extract_path_text(metadata, 'status') != 'completed'
ORDER BY created_at;
"
```

**Target:** All action items from incidents older than 30 days should be completed.

### Verification

Confirm all P1/P2 incidents have post-mortems written and action items tracked.

---

## Section 7: Generate Monthly Operations Report (~15 min)

### 7a. Report Template

Generate the monthly report with the following structure:

```
# Monthly Operations Report — [Month Year]

## Executive Summary
- Overall platform availability: __%
- Total incidents: __ (P1: __, P2: __, P3: __, P4: __)
- Key highlights: ...
- Key risks: ...

## Capacity
- CPU utilization trend: __% avg (target: <60%)
- Memory utilization trend: __% avg (target: <70%)
- Disk growth rate: __GB/month
- Capacity actions needed: ...

## Performance
- p50/p95/p99 API latency: __ms / __ms / __ms
- Error rate: __%
- Slowest endpoints: ...
- Database query trends: ...

## Security
- Failed auth attempts: __
- Rate-limited requests: __
- Vulnerabilities: Critical: __ High: __
- Security actions needed: ...

## Cost
- Over-provisioned services: ...
- Recommended right-sizing: ...
- Estimated savings: $__/month

## SLA Compliance
- Availability: __% (target: 99.9%)
- Error budget remaining: __%
- Latency SLA: __% under 500ms (target: 95%)

## Incidents
- Total: __
- MTTD: __ min
- MTTR: __ min
- Outstanding action items: __

## Action Items for Next Month
1. ...
2. ...
3. ...
```

### 7b. Save Report

Save the report to the shared drive:

```bash
# Generate report filename
REPORT_FILE="monthly-ops-report-$(date -u +%Y-%m).md"
echo "Save report as: ${REPORT_FILE}"
```

### 7c. Distribute Report

Send the report to the distribution list:
- Platform Team
- Engineering Leadership
- Security Team (security section only)

### Verification

Confirm the report has been saved and distributed. Update the report index with the new entry.

---

## Post-Review Actions

1. Create tickets for any identified issues
2. Update capacity planning forecasts
3. Schedule follow-up meetings for any critical items
4. Update dashboards or alerts based on findings
5. Update this runbook if new review areas are identified

## Related Runbooks

- [OP-001: Daily Checks](OP-001-daily-checks.md)
- [OP-002: Weekly Maintenance](OP-002-weekly-maintenance.md)
- [OP-004: Backup Verification](OP-004-backup-verification.md)
- [SC-005: Capacity Planning](../scaling/SC-005-capacity-planning.md)

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2026-03-08 | Platform Team | Initial version |
