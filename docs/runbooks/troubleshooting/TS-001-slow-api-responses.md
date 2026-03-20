# TS-001: API Latency Investigation

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | TS-001                                     |
| **Title**        | API Latency Investigation                  |
| **Severity**     | P2 - High                                  |
| **Services**     | api-gateway, all backend services          |
| **Last Updated** | 2026-03-08                                 |
| **Author**       | Platform Engineering                       |
| **Review Cycle** | Quarterly                                  |

---

## Summary

This runbook covers the investigation and resolution of elevated API response times across the Clario 360 platform. Slow API responses may stem from database query performance degradation, low Redis cache hit rates, resource exhaustion on service pods, Kafka consumer lag causing back-pressure, or misconfigured connection pools. Follow the diagnosis steps in order to isolate the root cause, then apply the corresponding resolution.

---

## Symptoms

- API response times exceed p95 SLA thresholds (typically > 500ms for reads, > 1000ms for writes).
- Users report sluggish UI interactions, timeouts, or spinner states that persist.
- Grafana API Performance dashboard shows elevated latency across one or more services.
- HTTP 504 Gateway Timeout errors appear in api-gateway logs.
- Increased error rates on downstream service calls visible in distributed traces.

---

## Diagnosis Steps

### Step 1: Check Grafana API Performance Dashboard

Open the Grafana API Performance dashboard to identify which services and endpoints exhibit elevated latency.

```
# Port-forward to Grafana if not exposed externally
kubectl -n monitoring port-forward svc/grafana 3000:3000

# Open in browser: http://localhost:3000/d/api-performance/api-performance
# Filter by namespace: clario360
# Look at panels: Request Duration p50/p95/p99, Error Rate, Throughput
```

Key observations:
- Is latency elevated across all services, or isolated to specific ones?
- Did latency spike at a specific time? Correlate with deployments or traffic changes.
- Are error rates increasing alongside latency (indicating timeouts or failures)?

### Step 2: Check API Gateway Metrics Directly

```bash
# Check api-gateway request duration metrics via Prometheus
kubectl -n clario360 port-forward svc/api-gateway 8080:8080

# Query the metrics endpoint
curl -s http://localhost:8080/metrics | grep 'http_request_duration_seconds'

# Look for high-latency buckets:
curl -s http://localhost:8080/metrics | grep 'http_request_duration_seconds_bucket{.*le="1"'
```

```bash
# Check api-gateway health
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/readyz
```

### Step 3: Check Individual Service Health

```bash
# Verify all services are healthy
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $svc ---"
  kubectl -n clario360 exec deploy/$svc -- curl -s http://localhost:8080/healthz
  kubectl -n clario360 exec deploy/$svc -- curl -s http://localhost:8080/readyz
done
```

### Step 4: Find Slow Queries in pg_stat_statements

```bash
# Connect to the PostgreSQL instance
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core
```

```sql
-- Find the top 20 slowest queries by mean execution time
SELECT
    queryid,
    substring(query, 1, 120) AS short_query,
    calls,
    round(mean_exec_time::numeric, 2) AS mean_ms,
    round(max_exec_time::numeric, 2) AS max_ms,
    round(total_exec_time::numeric, 2) AS total_ms,
    rows
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;
```

```sql
-- Find queries with high total execution time (biggest overall impact)
SELECT
    queryid,
    substring(query, 1, 120) AS short_query,
    calls,
    round(mean_exec_time::numeric, 2) AS mean_ms,
    round(total_exec_time::numeric, 2) AS total_ms,
    rows
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 20;
```

```sql
-- Check for sequential scans on large tables
SELECT
    schemaname,
    relname AS table_name,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch,
    n_live_tup AS row_count
FROM pg_stat_user_tables
WHERE seq_scan > 0
ORDER BY seq_tup_read DESC
LIMIT 20;
```

```sql
-- Check active connections and long-running queries
SELECT
    pid,
    now() - pg_stat_activity.query_start AS duration,
    state,
    substring(query, 1, 100) AS query
FROM pg_stat_activity
WHERE state != 'idle'
    AND query NOT LIKE '%pg_stat_activity%'
ORDER BY duration DESC
LIMIT 20;
```

Repeat for each database if the issue is service-specific:

```bash
# Connect to other databases as needed
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d cyber_db
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d data_db
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d acta_db
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d lex_db
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d visus_db
```

### Step 5: Check Redis Cache Hit Rate

```bash
# Connect to Redis
kubectl -n clario360 exec -it deploy/redis -- redis-cli

# Check overall cache statistics
INFO stats

# Look for keyspace_hits and keyspace_misses
INFO stats | grep -E 'keyspace_hits|keyspace_misses'
```

```bash
# Calculate hit rate directly
kubectl -n clario360 exec -it deploy/redis -- redis-cli INFO stats | grep -E 'keyspace_hits|keyspace_misses'

# Hit rate = keyspace_hits / (keyspace_hits + keyspace_misses) * 100
# A healthy hit rate is > 90%. Below 80% indicates caching issues.
```

```bash
# Check Redis memory usage
kubectl -n clario360 exec -it deploy/redis -- redis-cli INFO memory

# Check for evictions (indicates memory pressure)
kubectl -n clario360 exec -it deploy/redis -- redis-cli INFO stats | grep evicted_keys

# Check connected clients
kubectl -n clario360 exec -it deploy/redis -- redis-cli INFO clients

# Check slow log for Redis itself
kubectl -n clario360 exec -it deploy/redis -- redis-cli SLOWLOG GET 10
```

### Step 6: Check Resource Utilization

```bash
# Check CPU and memory for all pods in the clario360 namespace
kubectl -n clario360 top pods

# Check node-level resources
kubectl top nodes

# Check specific service resource usage and limits
kubectl -n clario360 describe deploy/api-gateway | grep -A 5 'Limits\|Requests'

# Check for pods in resource pressure (OOMKilled, throttled)
kubectl -n clario360 get pods -o wide
kubectl -n clario360 get events --sort-by='.lastTimestamp' | grep -i -E 'oom|throttl|evict|kill'
```

```bash
# Check for CPU throttling on specific services
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $svc ---"
  kubectl -n clario360 top pod -l app=$svc
done
```

### Step 7: Check Kafka Consumer Lag

```bash
# List all consumer groups
kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --list

# Check consumer lag for each group
kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-audit-service

kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-notification-service

kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-workflow-engine

kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-data-service
```

A LAG value greater than 10,000 on any partition indicates the consumer is falling behind. Total lag across all partitions greater than 50,000 warrants immediate attention.

### Step 8: Check Connection Pool Saturation

```bash
# Check PostgreSQL connection count
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core -c \
  "SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname ORDER BY count DESC;"

# Check max connections setting
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core -c \
  "SHOW max_connections;"

# Check if connections are close to the limit
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core -c \
  "SELECT count(*) AS active_connections, (SELECT setting::int FROM pg_settings WHERE name='max_connections') AS max_connections FROM pg_stat_activity;"
```

---

## Resolution Steps

### Root Cause: Database Query Performance

1. **Add missing indexes** based on the sequential scan analysis:

```bash
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core
```

```sql
-- Identify missing indexes on frequently queried columns
SELECT
    schemaname,
    relname,
    seq_scan - idx_scan AS too_much_seq,
    CASE
        WHEN seq_scan - idx_scan > 0 THEN 'Missing Index?'
        ELSE 'OK'
    END AS status,
    pg_size_pretty(pg_relation_size(relid)) AS table_size
FROM pg_stat_user_tables
WHERE seq_scan > 100
ORDER BY too_much_seq DESC;

-- Example: Create an index (adjust table/column based on findings)
-- CREATE INDEX CONCURRENTLY idx_tablename_columnname ON schema.tablename(columnname);
```

2. **Kill long-running queries** if they are blocking:

```sql
-- Terminate a specific long-running query
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE duration > interval '5 minutes'
    AND state = 'active'
    AND query NOT LIKE '%pg_stat_activity%';
```

3. **Reset pg_stat_statements** after fixing to establish a new baseline:

```sql
SELECT pg_stat_statements_reset();
```

### Root Cause: Low Redis Cache Hit Rate

1. **Check and increase Redis memory limit** if evictions are occurring:

```bash
kubectl -n clario360 exec -it deploy/redis -- redis-cli CONFIG GET maxmemory
kubectl -n clario360 exec -it deploy/redis -- redis-cli CONFIG SET maxmemory 2gb
```

2. **Verify TTL settings** on cached keys:

```bash
# Check TTL on sample keys
kubectl -n clario360 exec -it deploy/redis -- redis-cli TTL "session:*"
kubectl -n clario360 exec -it deploy/redis -- redis-cli TTL "cache:tenant:*"
```

3. **Flush stale cache** if cache is corrupted or contains stale data:

```bash
# Flush specific key patterns (NOT FLUSHALL)
kubectl -n clario360 exec -it deploy/redis -- redis-cli --scan --pattern 'cache:*' | head -100

# Delete specific key patterns
kubectl -n clario360 exec -it deploy/redis -- redis-cli EVAL "local keys = redis.call('keys', ARGV[1]) for i=1,#keys do redis.call('del', keys[i]) end return #keys" 0 'cache:stale-prefix:*'
```

4. **Restart affected service** to reinitialize cache connections:

```bash
kubectl -n clario360 rollout restart deploy/<service-name>
kubectl -n clario360 rollout status deploy/<service-name>
```

### Root Cause: Resource Exhaustion

1. **Scale horizontally** (increase replicas):

```bash
kubectl -n clario360 scale deploy/<service-name> --replicas=4
kubectl -n clario360 rollout status deploy/<service-name>
```

2. **Scale vertically** (increase resource limits):

```bash
kubectl -n clario360 patch deploy/<service-name> --type='json' -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "500m"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/cpu", "value": "1000m"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "512Mi"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "1Gi"}
]'
```

### Root Cause: Kafka Consumer Lag

1. **Scale the consumer service** to increase parallelism:

```bash
kubectl -n clario360 scale deploy/<consumer-service> --replicas=4
```

2. **Check topic partition count** and increase if needed:

```bash
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --topic clario360.events

# Increase partitions if consumer count exceeds partition count
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --alter --topic clario360.events --partitions 12
```

### Root Cause: Connection Pool Saturation

1. **Increase PostgreSQL max_connections** (requires restart):

```bash
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core -c \
  "ALTER SYSTEM SET max_connections = 300;"

# Restart PostgreSQL to apply
kubectl -n clario360 rollout restart statefulset/postgresql
```

2. **Tune service-level pool settings** via environment variables:

```bash
kubectl -n clario360 set env deploy/<service-name> \
  DB_MAX_OPEN_CONNS=25 \
  DB_MAX_IDLE_CONNS=10 \
  DB_CONN_MAX_LIFETIME=300s
kubectl -n clario360 rollout status deploy/<service-name>
```

---

## Verification

After applying the resolution, verify the fix:

```bash
# 1. Check service health
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $svc ---"
  kubectl -n clario360 exec deploy/$svc -- curl -s http://localhost:8080/healthz
done

# 2. Run a quick API latency test from inside the cluster
kubectl -n clario360 run curl-test --rm -it --image=curlimages/curl -- \
  curl -w "\n  DNS: %{time_namelookup}s\n  Connect: %{time_connect}s\n  TTFB: %{time_starttransfer}s\n  Total: %{time_total}s\n" \
  -o /dev/null -s http://api-gateway.clario360.svc.cluster.local:8080/healthz

# 3. Check Grafana dashboard for latency reduction
# Open: http://localhost:3000/d/api-performance/api-performance
# Confirm p95 latency has returned below SLA thresholds.

# 4. Verify Kafka consumer lag has decreased
kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-audit-service

# 5. Verify Redis hit rate has improved
kubectl -n clario360 exec -it deploy/redis -- redis-cli INFO stats | grep -E 'keyspace_hits|keyspace_misses'

# 6. Verify connection pool usage has stabilized
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core -c \
  "SELECT count(*) AS active_connections FROM pg_stat_activity;"
```

---

## Related Links

- [TS-002: Data Pipeline Failure Investigation](./TS-002-failed-pipelines.md)
- [TS-003: Kafka Event Loss Investigation](./TS-003-missing-events.md)
- [TS-004: Authentication Issue Debugging](./TS-004-auth-failures.md)
- [TS-005: WebSocket Connectivity Issues](./TS-005-websocket-disconnects.md)
- Grafana API Performance Dashboard: `/d/api-performance/api-performance`
- PostgreSQL Monitoring Dashboard: `/d/pg-monitoring/postgresql`
- Kafka Consumer Lag Dashboard: `/d/kafka-lag/kafka-consumer-lag`
