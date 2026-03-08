# OP-001: Daily Operational Health Checks

| Field              | Value                                |
|--------------------|--------------------------------------|
| **Runbook ID**     | OP-001                               |
| **Title**          | Daily Operational Health Checks      |
| **Frequency**      | Daily (08:00 UTC)                    |
| **Estimated Time** | ~15 minutes                          |
| **Owner**          | Platform Team (On-Call Engineer)     |
| **Last Updated**   | 2026-03-08                           |
| **Review Cycle**   | Quarterly                            |
| **Automation**     | `scripts/daily-check.sh`             |

## Summary

This runbook defines the daily health checks for the Clario 360 platform. It covers 10 verification areas that collectively confirm the platform is operating within normal parameters. The automated script `scripts/daily-check.sh` performs all checks; this runbook documents the manual procedure and expected outputs for when manual verification is required.

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export API_URL=https://api.clario360.io
export GRAFANA_URL=https://grafana.clario360.io
```

---

## Check 1: Kubernetes Cluster Health (~1 min)

Verify all nodes are `Ready` and no nodes are under resource pressure.

```bash
kubectl get nodes -o wide
```

**Expected output:** All nodes show `STATUS: Ready`. No `NotReady`, `SchedulingDisabled`, or pressure conditions.

Check for node conditions:

```bash
kubectl get nodes -o json | jq -r '.items[] | select(.status.conditions[] | select(.type != "Ready" and .status == "True")) | .metadata.name + ": " + (.status.conditions[] | select(.status == "True" and .type != "Ready") | .type)'
```

**Expected output:** Empty (no output). Any output indicates a node under pressure (MemoryPressure, DiskPressure, PIDPressure).

Check cluster component status:

```bash
kubectl get componentstatuses 2>/dev/null || kubectl get --raw='/readyz?verbose' | head -30
```

---

## Check 2: Service Health — All 10 Services (~2 min)

Verify every Clario 360 service is running and healthy.

### 2a. Pod Status

```bash
kubectl get pods -n ${NAMESPACE} -o wide --sort-by=.metadata.name
```

**Expected output:** All pods show `STATUS: Running` and `READY: n/n` (all containers ready). No pods in `CrashLoopBackOff`, `Error`, `Pending`, or `ImagePullBackOff`.

Check for pods not in Running state:

```bash
kubectl get pods -n ${NAMESPACE} --field-selector=status.phase!=Running,status.phase!=Succeeded
```

**Expected output:** `No resources found in clario360 namespace.`

### 2b. Health Endpoints

Loop through all services and verify `/healthz` returns 200:

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  STATUS=$(kubectl exec -n ${NAMESPACE} deploy/${SERVICE} -- wget -q -O - --timeout=5 http://localhost:8080/healthz 2>/dev/null)
  EXITCODE=$?
  if [ ${EXITCODE} -eq 0 ]; then
    echo "OK: ${SERVICE} - ${STATUS}"
  else
    echo "FAIL: ${SERVICE} - exit code ${EXITCODE}"
  fi
done
```

**Expected output:** All 10 services return `OK` with a JSON health response.

### 2c. Readiness Endpoints

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  STATUS=$(kubectl exec -n ${NAMESPACE} deploy/${SERVICE} -- wget -q -O - --timeout=5 http://localhost:8080/readyz 2>/dev/null)
  EXITCODE=$?
  if [ ${EXITCODE} -eq 0 ]; then
    echo "READY: ${SERVICE}"
  else
    echo "NOT READY: ${SERVICE} - exit code ${EXITCODE}"
  fi
done
```

**Expected output:** All services return `READY`.

### 2d. Recent Restarts

```bash
kubectl get pods -n ${NAMESPACE} -o json | jq -r '.items[] | select(.status.containerStatuses[]?.restartCount > 0) | .metadata.name + ": " + (.status.containerStatuses[] | .name + "=" + (.restartCount | tostring) + " restarts")'
```

**Expected output:** No output, or stable restart counts (not increasing day-over-day). Investigate any service with restarts that increased since yesterday.

---

## Check 3: Database Health (~2 min)

### 3a. Connection Count

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT datname, numbackends,
       (SELECT setting::int FROM pg_settings WHERE name='max_connections') AS max_conn,
       round(numbackends::numeric / (SELECT setting::int FROM pg_settings WHERE name='max_connections') * 100, 1) AS pct_used
FROM pg_stat_database
WHERE datname IN ('platform_core','cyber_db','data_db','acta_db','lex_db','visus_db')
ORDER BY numbackends DESC;
"
```

**Expected output:** Connection usage below 70% of `max_connections` for all databases. Alert threshold: 80%.

### 3b. Long-Running Queries

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT pid, now() - pg_stat_activity.query_start AS duration, state,
       left(query, 80) AS query_preview
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes'
  AND state != 'idle'
  AND query NOT LIKE '%pg_stat_activity%'
ORDER BY duration DESC;
"
```

**Expected output:** No rows returned. Any query running longer than 5 minutes warrants investigation.

### 3c. Replication Lag (if read replicas configured)

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT client_addr, state,
       pg_wal_lsn_diff(pg_current_wal_lsn(), sent_lsn) AS sent_lag_bytes,
       pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS replay_lag_bytes
FROM pg_stat_replication;
"
```

**Expected output:** `replay_lag_bytes` below 1MB (1048576 bytes). Higher values indicate replication falling behind.

### 3d. Database Size Check

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT datname, pg_size_pretty(pg_database_size(datname)) AS size
FROM pg_database
WHERE datname IN ('platform_core','cyber_db','data_db','acta_db','lex_db','visus_db')
ORDER BY pg_database_size(datname) DESC;
"
```

**Expected output:** Note sizes for trend tracking. Alert if any database grows more than 10% day-over-day unexpectedly.

---

## Check 4: Kafka Consumer Lag (~2 min)

### 4a. List Consumer Groups

```bash
kubectl exec -n kafka deploy/kafka -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --list
```

### 4b. Check Lag for Each Consumer Group

```bash
kubectl exec -n kafka deploy/kafka -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --all-groups 2>/dev/null | grep -E "^(GROUP|clario)" | head -50
```

**Expected output:** `LAG` column should be 0 or near-zero for all consumer groups. Sustained lag above 1000 indicates consumers are falling behind.

### 4c. Check for Under-Replicated Partitions

```bash
kubectl exec -n kafka deploy/kafka -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --under-replicated-partitions
```

**Expected output:** No output (no under-replicated partitions). Any output indicates broker issues.

---

## Check 5: Redis Health (~1 min)

### 5a. Redis Ping

```bash
kubectl exec -n ${NAMESPACE} deploy/redis -- redis-cli ping
```

**Expected output:** `PONG`

### 5b. Redis Info (Key Metrics)

```bash
kubectl exec -n ${NAMESPACE} deploy/redis -- redis-cli info stats | grep -E "keyspace_hits|keyspace_misses|connected_clients|blocked_clients|used_memory_human|evicted_keys"
```

**Expected output:**
- `connected_clients` below 500
- `blocked_clients` at 0
- `evicted_keys` at 0 (indicates sufficient memory)
- Hit ratio: `keyspace_hits / (keyspace_hits + keyspace_misses)` should be above 90%

### 5c. Redis Memory

```bash
kubectl exec -n ${NAMESPACE} deploy/redis -- redis-cli info memory | grep -E "used_memory_human|maxmemory_human|mem_fragmentation_ratio"
```

**Expected output:** `used_memory` well below `maxmemory`. `mem_fragmentation_ratio` between 1.0 and 1.5.

---

## Check 6: Certificate Expiry (~1 min)

Check TLS certificates across the cluster:

```bash
kubectl get certificates -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.name + " | Not After: " + .status.notAfter + " | Ready: " + (.status.conditions[] | select(.type=="Ready") | .status)'
```

**Expected output:** All certificates show `Ready: True` and expiry dates more than 30 days from now.

Check ingress certificate specifically:

```bash
kubectl get secret -n ${NAMESPACE} clario360-tls -o json | jq -r '.data["tls.crt"]' | base64 -d | openssl x509 -noout -dates -subject
```

**Expected output:** `notAfter` date is more than 30 days in the future.

Quick expiry check (certificates expiring within 30 days):

```bash
kubectl get certificates -n ${NAMESPACE} -o json | jq -r --arg cutoff "$(date -u -v+30d '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -d '+30 days' '+%Y-%m-%dT%H:%M:%SZ')" '.items[] | select(.status.notAfter < $cutoff) | "EXPIRING SOON: " + .metadata.name + " expires " + .status.notAfter'
```

**Expected output:** No output (no certificates expiring within 30 days).

---

## Check 7: Backup Status (~1 min)

Verify the most recent backup completed successfully:

```bash
kubectl get cronjobs -n ${NAMESPACE} | grep backup
```

```bash
kubectl get jobs -n ${NAMESPACE} --sort-by=.status.startTime | grep backup | tail -5
```

**Expected output:** Most recent backup job shows `COMPLETIONS: 1/1`.

Check the backup job logs:

```bash
LATEST_BACKUP_JOB=$(kubectl get jobs -n ${NAMESPACE} --sort-by=.status.startTime -o json | jq -r '[.items[] | select(.metadata.name | startswith("pg-backup"))] | last | .metadata.name')
kubectl logs -n ${NAMESPACE} job/${LATEST_BACKUP_JOB} --tail=20
```

**Expected output:** Log shows successful completion with backup file size and upload confirmation.

Verify backup exists in storage:

```bash
kubectl exec -n ${NAMESPACE} deploy/backup-tools -- gsutil ls -l gs://clario360-backups/daily/ | tail -5
```

**Expected output:** Today's backup file is present with a reasonable size (not 0 bytes).

---

## Check 8: Alert Status — Alertmanager (~1 min)

### 8a. Active Alerts

```bash
kubectl exec -n monitoring deploy/alertmanager -- wget -q -O - http://localhost:9093/api/v2/alerts?active=true 2>/dev/null | jq -r '.[] | .labels.alertname + " [" + .labels.severity + "] - " + .annotations.summary'
```

**Expected output:** No critical or warning alerts. Informational alerts may be present.

### 8b. Silenced Alerts

```bash
kubectl exec -n monitoring deploy/alertmanager -- wget -q -O - http://localhost:9093/api/v2/silences?active=true 2>/dev/null | jq -r '.[] | select(.status.state == "active") | .createdBy + ": " + (.matchers[] | .name + "=" + .value) + " until " + .endsAt'
```

**Expected output:** Review any active silences. Silences older than 7 days should be reviewed and removed if no longer needed.

### 8c. Alertmanager Health

```bash
kubectl exec -n monitoring deploy/alertmanager -- wget -q -O - http://localhost:9093/-/healthy 2>/dev/null
```

**Expected output:** `OK`

---

## Check 9: Disk Usage (~2 min)

### 9a. PVC Usage

```bash
kubectl get pvc -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.name + " | " + .status.capacity.storage + " | Phase: " + .status.phase'
```

**Expected output:** All PVCs show `Phase: Bound`.

### 9b. Node Disk Usage

```bash
kubectl top nodes --sort-by=cpu 2>/dev/null || kubectl get nodes -o json | jq -r '.items[] | .metadata.name + " | Allocatable disk: " + .status.allocatable."ephemeral-storage"'
```

### 9c. PostgreSQL Disk Usage

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- df -h /var/lib/postgresql/data
```

**Expected output:** Usage below 70%. Alert threshold: 80%. Critical: 90%.

### 9d. Kafka Disk Usage

```bash
kubectl exec -n kafka kafka-0 -- df -h /var/lib/kafka/data
```

**Expected output:** Usage below 70%.

### 9e. PVC Capacity Check via Prometheus (alternative)

```bash
kubectl exec -n monitoring deploy/prometheus-server -- wget -q -O - 'http://localhost:9090/api/v1/query?query=kubelet_volume_stats_used_bytes/kubelet_volume_stats_capacity_bytes*100' 2>/dev/null | jq -r '.data.result[] | .metric.persistentvolumeclaim + ": " + (.value[1] | tonumber | round | tostring) + "% used"'
```

**Expected output:** All volumes below 70% usage.

---

## Check 10: Audit Trail Integrity (~2 min)

Verify the audit hash chain has not been tampered with.

### 10a. Check Latest Audit Entry Hash

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT id, created_at, entry_hash, prev_hash
FROM audit_logs
ORDER BY id DESC
LIMIT 5;
"
```

### 10b. Verify Chain Integrity (last 1000 entries)

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
WITH chain AS (
  SELECT id, entry_hash, prev_hash,
         LAG(entry_hash) OVER (ORDER BY id) AS expected_prev_hash
  FROM audit_logs
  ORDER BY id DESC
  LIMIT 1000
)
SELECT count(*) AS broken_links
FROM chain
WHERE expected_prev_hash IS NOT NULL
  AND prev_hash != expected_prev_hash;
"
```

**Expected output:** `broken_links` equals `0`. Any non-zero value indicates a broken audit chain and must be investigated immediately per [TS-009](../troubleshooting/TS-009-audit-chain-broken.md).

### 10c. Check Audit Log Volume

```bash
kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT date_trunc('hour', created_at) AS hour, count(*) AS entries
FROM audit_logs
WHERE created_at > now() - interval '24 hours'
GROUP BY hour
ORDER BY hour DESC;
"
```

**Expected output:** Consistent hourly volume. Sudden drops may indicate logging failures; sudden spikes may indicate suspicious activity.

---

## Verification Summary

After completing all 10 checks, summarize findings:

| Check | Area | Status |
|-------|------|--------|
| 1 | Cluster Health | PASS / FAIL |
| 2 | Service Health (10 services) | PASS / FAIL |
| 3 | Database Health | PASS / FAIL |
| 4 | Kafka Consumer Lag | PASS / FAIL |
| 5 | Redis Health | PASS / FAIL |
| 6 | Certificate Expiry | PASS / FAIL |
| 7 | Backup Status | PASS / FAIL |
| 8 | Alert Status | PASS / FAIL |
| 9 | Disk Usage | PASS / FAIL |
| 10 | Audit Trail Integrity | PASS / FAIL |

Record results in the daily operations log. If any check fails, follow the corresponding incident response or troubleshooting runbook.

## Automation

The automated version of this runbook is available at `scripts/daily-check.sh`. It runs as a Kubernetes CronJob defined in `scripts/daily-health-check-cronjob.yaml` and posts results to the `#platform-ops` Slack channel.

To run the automated check manually:

```bash
kubectl create job --from=cronjob/daily-health-check daily-health-check-manual-$(date +%s) -n ${NAMESPACE}
```

## Related Runbooks

- [OP-002: Weekly Maintenance](OP-002-weekly-maintenance.md)
- [IR-001: Service Outage](../incident-response/IR-001-service-outage.md)
- [IR-002: Database Failure](../incident-response/IR-002-database-failure.md)
- [IR-006: Disk Full](../incident-response/IR-006-disk-full.md)
- [TS-009: Audit Chain Broken](../troubleshooting/TS-009-audit-chain-broken.md)

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2026-03-08 | Platform Team | Initial version |
