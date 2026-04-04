# OP-009: Log Rotation, Retention & Archival

| Field | Value |
|-------|-------|
| **Runbook ID** | OP-009 |
| **Title** | Log Rotation, Retention & Archival |
| **Frequency** | Weekly |
| **Owner** | Platform Infrastructure Team |
| **Last Updated** | 2026-03-08 |
| **Estimated Duration** | 30–60 minutes |
| **Risk Level** | Low |
| **Approval Required** | No |
| **Maintenance Window** | Not required — non-disruptive |

## Summary

This runbook covers log lifecycle management for the Clario 360 platform:

1. Check log volume sizes across services
2. Verify log rotation is working (Kubernetes-level and application-level)
3. Archive old logs to object storage (GCS)
4. Clean up expired log archives
5. Verify centralized logging (Loki) is receiving logs
6. Check log-based alerting rules

### Log Architecture

| Component | Log Method | Destination | Retention |
|-----------|-----------|-------------|-----------|
| Application services | stdout/stderr | Collected by Promtail | 30 days in Loki |
| PostgreSQL | File-based | PV on postgres pod | 14 days local |
| Kafka brokers | File-based | PV on broker pods | 7 days local |
| Redis | stdout | Collected by Promtail | 30 days in Loki |
| Nginx ingress | stdout | Collected by Promtail | 30 days in Loki |
| Audit logs | PostgreSQL table | `platform_core.audit_logs` | 13 months (see OP-007) |

### Log Levels

All Clario 360 services use structured JSON logging with levels: `debug`, `info`, `warn`, `error`, `fatal`.

## Prerequisites

```bash
export NAMESPACE=clario360
export KAFKA_NS=kafka
export LOKI_URL=http://loki-gateway.monitoring.svc.cluster.local
export GCS_BUCKET=clario360-log-archives
export GRAFANA_URL=https://grafana.clario360.io
```

Verify tools:

```bash
kubectl -n $NAMESPACE get pods --no-headers | wc -l
gsutil ls gs://$GCS_BUCKET/ 2>/dev/null && echo "GCS accessible" || echo "GCS not accessible"
```

---

## Step 1: Check Log Volume Sizes Across Services

### 1a. Check container log sizes on each node

```bash
# Get all nodes
NODES=$(kubectl get nodes -o jsonpath='{.items[*].metadata.name}')

for node in $NODES; do
  echo "=== Node: $node ==="
  kubectl debug node/$node -it --image=busybox -- sh -c \
    "du -sh /host/var/log/containers/*clario360* 2>/dev/null | sort -rh | head -20" 2>/dev/null
  echo ""
done
```

### 1b. Check per-pod log volume

```bash
SERVICES="api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service"

for svc in $SERVICES; do
  PODS=$(kubectl -n $NAMESPACE get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}')
  for pod in $PODS; do
    SIZE=$(kubectl -n $NAMESPACE exec $pod -- sh -c 'du -sh /proc/1/fd/1 2>/dev/null || echo "N/A"' 2>/dev/null)
    echo "$pod: $SIZE"
  done
done
```

### 1c. Check log rates via Loki (lines per minute per service)

```bash
# Query Loki for log rates over the last hour
for svc in $SERVICES; do
  RATE=$(curl -s -G "$LOKI_URL/loki/api/v1/query" \
    --data-urlencode "query=sum(rate({namespace=\"$NAMESPACE\", app=\"$svc\"}[5m]))" \
    2>/dev/null | jq -r '.data.result[0].value[1] // "0"')
  echo "$svc: ${RATE} lines/sec"
done
```

### 1d. Check PostgreSQL log size

```bash
kubectl -n $NAMESPACE exec deployment/postgresql -- sh -c \
  "du -sh /var/lib/postgresql/data/log/ 2>/dev/null || du -sh /var/log/postgresql/ 2>/dev/null || echo 'Log dir not found'"
```

### 1e. Check Kafka broker log sizes

```bash
for broker in kafka-0 kafka-1 kafka-2; do
  echo "=== $broker ==="
  kubectl -n $KAFKA_NS exec $broker -- du -sh /opt/kafka/logs/ 2>/dev/null || \
  kubectl -n $KAFKA_NS exec $broker -- du -sh /var/log/kafka/ 2>/dev/null || \
  echo "Log dir not found"
done
```

---

## Step 2: Verify Log Rotation Is Working

### 2a. Check Kubernetes container log rotation (kubelet configuration)

```bash
# Kubelet default: 10MB max size, 5 files retained
# Verify on a node
NODE=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
kubectl debug node/$NODE -it --image=busybox -- sh -c \
  "cat /host/var/lib/kubelet/config.yaml 2>/dev/null | grep -A2 -i containerLog" 2>/dev/null
```

### 2b. Verify rotated log files exist

```bash
kubectl debug node/$NODE -it --image=busybox -- sh -c \
  "ls -la /host/var/log/containers/ | grep clario360 | head -20" 2>/dev/null
```

Look for files with `.log`, `.log.1`, `.log.2` suffixes indicating rotation is active.

### 2c. Check PostgreSQL log rotation

```bash
kubectl -n $NAMESPACE exec deployment/postgresql -- sh -c "
  ls -lt /var/lib/postgresql/data/log/ 2>/dev/null | head -10 || \
  ls -lt /var/log/postgresql/ 2>/dev/null | head -10
"

# Verify log_rotation_age and log_rotation_size
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U clario360_admin -d platform_core -c "
  SHOW log_rotation_age;
  SHOW log_rotation_size;
  SHOW log_filename;
"
```

Expected: `log_rotation_age = 1d`, `log_rotation_size = 100MB`.

### 2d. Check Kafka log4j rotation

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- sh -c "
  ls -lt /opt/kafka/logs/*.log* 2>/dev/null | head -10 || \
  ls -lt /var/log/kafka/*.log* 2>/dev/null | head -10
"

# Check log4j appender configuration
kubectl -n $KAFKA_NS exec kafka-0 -- cat /opt/kafka/config/log4j.properties 2>/dev/null | \
  grep -E "MaxFileSize|MaxBackupIndex|File=" || echo "log4j config not found at default path"
```

### 2e. Verify Promtail is running and tailing logs

```bash
kubectl -n monitoring get pods -l app=promtail
kubectl -n monitoring get daemonset promtail -o jsonpath='{.status.numberReady}/{.status.desiredNumberScheduled}'
echo ""

# Check Promtail targets
PROMTAIL_POD=$(kubectl -n monitoring get pods -l app=promtail -o jsonpath='{.items[0].metadata.name}')
kubectl -n monitoring exec $PROMTAIL_POD -- wget -qO- http://localhost:3101/targets 2>/dev/null | \
  jq '.[] | select(.labels.namespace=="clario360") | {job: .labels.job, state: .targetState}' 2>/dev/null | head -30
```

---

## Step 3: Archive Old Logs to Object Storage

### 3a. Export logs from Loki to GCS (logs older than 14 days)

```bash
ARCHIVE_DATE=$(date -u -d "14 days ago" +%Y-%m-%d 2>/dev/null || date -u -v-14d +%Y-%m-%d)
ARCHIVE_FILE="clario360-logs-before-${ARCHIVE_DATE}.json.gz"

echo "Archiving logs older than $ARCHIVE_DATE"

# Query Loki for old logs and export
for svc in $SERVICES; do
  echo "=== Archiving $svc logs ==="
  curl -s -G "$LOKI_URL/loki/api/v1/query_range" \
    --data-urlencode "query={namespace=\"$NAMESPACE\", app=\"$svc\"}" \
    --data-urlencode "start=$(date -u -d '30 days ago' +%s 2>/dev/null || date -u -v-30d +%s)000000000" \
    --data-urlencode "end=$(date -u -d '14 days ago' +%s 2>/dev/null || date -u -v-14d +%s)000000000" \
    --data-urlencode "limit=10000" \
    -o /tmp/logs-${svc}-${ARCHIVE_DATE}.json 2>/dev/null

  gzip /tmp/logs-${svc}-${ARCHIVE_DATE}.json
  gsutil cp /tmp/logs-${svc}-${ARCHIVE_DATE}.json.gz \
    gs://$GCS_BUCKET/$(date +%Y)/$(date +%m)/${svc}-${ARCHIVE_DATE}.json.gz

  rm -f /tmp/logs-${svc}-${ARCHIVE_DATE}.json.gz
  echo "Archived $svc logs to gs://$GCS_BUCKET/$(date +%Y)/$(date +%m)/"
done
```

### 3b. Archive PostgreSQL logs

```bash
PG_POD=$(kubectl -n $NAMESPACE get pods -l app=postgresql -o jsonpath='{.items[0].metadata.name}')
PG_LOG_DIR="/var/lib/postgresql/data/log"

# List old PostgreSQL log files (older than 7 days)
kubectl -n $NAMESPACE exec $PG_POD -- find $PG_LOG_DIR -name "*.log" -mtime +7 -type f 2>/dev/null | while read logfile; do
  BASENAME=$(basename $logfile)
  kubectl -n $NAMESPACE exec $PG_POD -- cat $logfile | gzip > /tmp/pg-${BASENAME}.gz
  gsutil cp /tmp/pg-${BASENAME}.gz gs://$GCS_BUCKET/postgresql/$(date +%Y)/$(date +%m)/
  rm -f /tmp/pg-${BASENAME}.gz
  echo "Archived $BASENAME"
done

# Remove archived logs from the pod
kubectl -n $NAMESPACE exec $PG_POD -- find $PG_LOG_DIR -name "*.log" -mtime +7 -type f -delete 2>/dev/null
echo "Cleaned up old PostgreSQL logs from pod"
```

### 3c. Archive Kafka broker logs

```bash
for broker in kafka-0 kafka-1 kafka-2; do
  echo "=== Archiving $broker logs ==="
  KAFKA_LOG_DIR="/opt/kafka/logs"

  kubectl -n $KAFKA_NS exec $broker -- find $KAFKA_LOG_DIR -name "*.log.*" -mtime +3 -type f 2>/dev/null | while read logfile; do
    BASENAME=$(basename $logfile)
    kubectl -n $KAFKA_NS exec $broker -- cat $logfile | gzip > /tmp/${broker}-${BASENAME}.gz
    gsutil cp /tmp/${broker}-${BASENAME}.gz gs://$GCS_BUCKET/kafka/$(date +%Y)/$(date +%m)/
    rm -f /tmp/${broker}-${BASENAME}.gz
    echo "Archived $broker/$BASENAME"
  done

  kubectl -n $KAFKA_NS exec $broker -- find $KAFKA_LOG_DIR -name "*.log.*" -mtime +3 -type f -delete 2>/dev/null
done
```

---

## Step 4: Clean Up Expired Log Archives

### 4a. List archived logs by age

```bash
echo "=== Archives by month ==="
gsutil ls -l gs://$GCS_BUCKET/ | head -20

echo ""
echo "=== Total archive size ==="
gsutil du -sh gs://$GCS_BUCKET/
```

### 4b. Check lifecycle policy on the GCS bucket

```bash
gsutil lifecycle get gs://$GCS_BUCKET/
```

Expected lifecycle policy (auto-delete after 365 days):

```json
{
  "rule": [
    {
      "action": {"type": "Delete"},
      "condition": {"age": 365}
    },
    {
      "action": {"type": "SetStorageClass", "storageClass": "COLDLINE"},
      "condition": {"age": 90}
    }
  ]
}
```

### 4c. Set lifecycle policy (if not configured)

```bash
cat > /tmp/lifecycle.json << 'EOF'
{
  "rule": [
    {
      "action": {"type": "SetStorageClass", "storageClass": "COLDLINE"},
      "condition": {"age": 90}
    },
    {
      "action": {"type": "Delete"},
      "condition": {"age": 365}
    }
  ]
}
EOF

gsutil lifecycle set /tmp/lifecycle.json gs://$GCS_BUCKET/
rm -f /tmp/lifecycle.json
```

### 4d. Manually delete archives older than retention period

```bash
# Delete archives older than 365 days
CUTOFF=$(date -u -d "365 days ago" +%Y-%m-%d 2>/dev/null || date -u -v-365d +%Y-%m-%d)
CUTOFF_YEAR=$(echo $CUTOFF | cut -d- -f1)
CUTOFF_MONTH=$(echo $CUTOFF | cut -d- -f2)

echo "Deleting archives older than $CUTOFF"
gsutil -m rm -r gs://$GCS_BUCKET/$((CUTOFF_YEAR - 1))/ 2>/dev/null || echo "No archives from $((CUTOFF_YEAR - 1))"
```

---

## Step 5: Verify Centralized Logging (Loki) Is Receiving Logs

### 5a. Check Loki is running

```bash
kubectl -n monitoring get pods -l app=loki
kubectl -n monitoring get svc loki-gateway
```

### 5b. Query Loki for recent logs from each service

```bash
for svc in $SERVICES; do
  COUNT=$(curl -s -G "$LOKI_URL/loki/api/v1/query" \
    --data-urlencode "query=count_over_time({namespace=\"$NAMESPACE\", app=\"$svc\"}[5m])" \
    2>/dev/null | jq -r '.data.result[0].value[1] // "0"')
  if [ "$COUNT" = "0" ] || [ -z "$COUNT" ]; then
    echo "WARNING: $svc — no logs received in last 5 minutes!"
  else
    echo "OK: $svc — $COUNT log lines in last 5 minutes"
  fi
done
```

### 5c. Check for gaps in log ingestion

```bash
echo "=== Log ingestion rate over last 24 hours (per hour) ==="
curl -s -G "$LOKI_URL/loki/api/v1/query_range" \
  --data-urlencode "query=sum(count_over_time({namespace=\"$NAMESPACE\"}[1h]))" \
  --data-urlencode "start=$(date -u -d '24 hours ago' +%s 2>/dev/null || date -u -v-24H +%s)000000000" \
  --data-urlencode "end=$(date -u +%s)000000000" \
  --data-urlencode "step=3600" \
  2>/dev/null | jq -r '.data.result[0].values[] | "\(.[0] | tonumber | strftime("%Y-%m-%d %H:%M")) — \(.[1]) lines"'
```

Look for hours with zero or abnormally low log counts.

### 5d. Check Loki storage usage

```bash
kubectl -n monitoring exec deployment/loki -- df -h /loki 2>/dev/null || \
kubectl -n monitoring exec deployment/loki -- df -h /data 2>/dev/null || \
echo "Check Loki PVC usage manually"

# Check PVC usage
kubectl -n monitoring get pvc -l app=loki
```

### 5e. Verify Loki data source in Grafana

```bash
curl -s -u admin:$(kubectl -n monitoring get secret grafana -o jsonpath='{.data.admin-password}' | base64 -d) \
  "$GRAFANA_URL/api/datasources" | jq '.[] | select(.type=="loki") | {name, url, access}'
```

---

## Step 6: Check Log-Based Alerting Rules

### 6a. List alerting rules that query Loki

```bash
# Check Prometheus/Grafana alerting rules that use LogQL
kubectl -n monitoring get prometheusrule -o yaml 2>/dev/null | grep -A5 "loki\|logql" || \
echo "PrometheusRule CRDs not found — checking Grafana alert rules"

# Check Grafana alert rules
curl -s -u admin:$(kubectl -n monitoring get secret grafana -o jsonpath='{.data.admin-password}' | base64 -d) \
  "$GRAFANA_URL/api/v1/provisioning/alert-rules" | jq '.[].title' 2>/dev/null
```

### 6b. Verify critical log-based alerts are firing correctly

Expected alerting rules:

| Alert Name | Query | Threshold | Severity |
|-----------|-------|-----------|----------|
| `HighErrorRate` | `sum(rate({namespace="clario360"} \|= "error" [5m])) by (app)` | > 10/min | warning |
| `ServiceNotLogging` | `absent_over_time({namespace="clario360", app=~".+"}[15m])` | absent | critical |
| `FatalLogDetected` | `{namespace="clario360"} \|= "fatal"` | any | critical |
| `AuthFailureSpike` | `sum(rate({namespace="clario360", app="iam-service"} \|= "authentication failed" [5m]))` | > 50/min | warning |
| `DiskPressureLog` | `{namespace="clario360"} \|= "disk pressure"` | any | warning |

### 6c. Test an alert (dry run)

```bash
# Query Loki for error logs in the last hour to verify the alert query works
curl -s -G "$LOKI_URL/loki/api/v1/query" \
  --data-urlencode 'query=sum(rate({namespace="clario360"} |= "error" [5m])) by (app)' \
  2>/dev/null | jq '.data.result[] | {app: .metric.app, error_rate: .value[1]}'
```

### 6d. Check alert notification channels

```bash
curl -s -u admin:$(kubectl -n monitoring get secret grafana -o jsonpath='{.data.admin-password}' | base64 -d) \
  "$GRAFANA_URL/api/v1/provisioning/contact-points" | jq '.[].name' 2>/dev/null
```

---

## Verification

After completing all log management tasks:

```bash
# 1. All services are logging to Loki
echo "=== Log ingestion check ==="
for svc in $SERVICES; do
  COUNT=$(curl -s -G "$LOKI_URL/loki/api/v1/query" \
    --data-urlencode "query=count_over_time({namespace=\"$NAMESPACE\", app=\"$svc\"}[5m])" \
    2>/dev/null | jq -r '.data.result[0].value[1] // "0"')
  echo "$svc: $COUNT lines (last 5m)"
done

# 2. Promtail pods healthy
echo ""
echo "=== Promtail health ==="
kubectl -n monitoring get pods -l app=promtail -o wide

# 3. Disk usage acceptable on nodes
echo ""
echo "=== Node log disk usage ==="
kubectl get nodes -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | while read node; do
  kubectl debug node/$node -it --image=busybox -- sh -c \
    "df -h /host/var/log | tail -1" 2>/dev/null
  echo " ($node)"
done

# 4. Archive bucket accessible
echo ""
echo "=== Archive status ==="
gsutil du -sh gs://$GCS_BUCKET/

# 5. Check Grafana dashboard
echo ""
echo "Review log dashboards: $GRAFANA_URL/d/service-health"
```

---

## Related Links

- [OP-001: Daily Checks](OP-001-daily-checks.md)
- [OP-007: Database Maintenance](OP-007-database-maintenance.md) (audit log partitions)
- [IR-006: Disk Full](../incident-response/IR-006-disk-full.md)
- [TS-001: Slow API Responses](../troubleshooting/TS-001-slow-api-responses.md) (log correlation)
- [Grafana — Service Health](https://grafana.clario360.io/d/service-health)
- [Grafana — Cluster Overview](https://grafana.clario360.io/d/cluster-overview)
