# TS-008: Database Connection Pool Exhaustion

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | TS-008                                                                |
| **Title**          | Database Connection Pool Exhaustion                                   |
| **Severity**       | P1 -- Critical                                                        |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | api-gateway, iam-service, audit-service, workflow-engine, notification-service, cyber-service, data-service, acta-service, lex-service, visus-service |
| **Namespace**      | clario360                                                             |
| **Escalation**     | Platform Engineering Lead -> Database Team Lead -> VP Engineering     |
| **SLA**            | Acknowledge within 5 minutes, resolve within 30 minutes               |

---

## Summary

This runbook addresses database connection pool exhaustion in the Clario360 platform. All services connect to PostgreSQL databases (`platform_core`, `cyber_db`, `data_db`, `acta_db`, `lex_db`, `visus_db`) via Go's `database/sql` connection pool. When the pool is exhausted, new database queries block until a connection becomes available or the query times out, causing cascading failures. Root causes include connection leaks (unclosed rows/transactions), long-running queries holding connections, insufficient pool size for the workload, or PostgreSQL reaching its `max_connections` limit.

---

## Symptoms

- Application logs showing `pq: sorry, too many clients already` or `pq: remaining connection slots are reserved for non-replication superuser connections`.
- HTTP 503 or 500 errors from services with messages indicating database unavailability.
- `/readyz` endpoints returning unhealthy because the database check times out.
- Prometheus alert `db_connection_pool_exhausted` or `db_wait_count_high` firing.
- Grafana dashboard showing `db_open_connections` at or near `db_max_open_connections`.
- Rapidly increasing `db_wait_duration_seconds` metric.
- Services timing out on all database-dependent endpoints while health checks on non-database paths succeed.
- Multiple services affected simultaneously (indicates PostgreSQL-level `max_connections` exhaustion).

---

## Impact Assessment

| Condition                          | Impact                                                         |
|------------------------------------|----------------------------------------------------------------|
| Single service pool exhausted      | That service's DB-dependent endpoints fail; others unaffected  |
| Multiple services pool exhausted   | Widespread failures across platform features                   |
| PostgreSQL max_connections reached  | All services fail; complete platform outage for DB operations  |
| Long-running transactions          | Lock contention; blocking other queries; potential deadlocks   |

---

## Prerequisites

- `kubectl` configured with cluster access and `clario360` namespace permissions.
- `psql` client installed locally or ability to exec into a pod with `psql`.
- PostgreSQL superuser or a role with access to `pg_stat_activity` and `pg_terminate_backend`.
- Access to Grafana dashboards for connection pool metrics.

---

## Diagnosis Steps

### Step 1: Check PostgreSQL Total Connection Count

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT
    max_conn,
    used,
    max_conn - used AS available,
    ROUND((used::float / max_conn::float) * 100, 1) AS pct_used
FROM
    (SELECT count(*) AS used FROM pg_stat_activity) t,
    (SELECT setting::int AS max_conn FROM pg_settings WHERE name = 'max_connections') s;
"
```

### Step 2: Check Connections Per Database

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT
    datname AS database,
    count(*) AS connections,
    count(*) FILTER (WHERE state = 'active') AS active,
    count(*) FILTER (WHERE state = 'idle') AS idle,
    count(*) FILTER (WHERE state = 'idle in transaction') AS idle_in_txn,
    count(*) FILTER (WHERE state = 'idle in transaction (aborted)') AS idle_in_txn_aborted,
    count(*) FILTER (WHERE wait_event IS NOT NULL AND state = 'active') AS waiting
FROM pg_stat_activity
WHERE datname IS NOT NULL
GROUP BY datname
ORDER BY connections DESC;
"
```

### Step 3: Check Connections Per Client Application

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT
    application_name,
    client_addr,
    datname,
    count(*) AS connections,
    count(*) FILTER (WHERE state = 'active') AS active,
    count(*) FILTER (WHERE state = 'idle') AS idle,
    count(*) FILTER (WHERE state = 'idle in transaction') AS idle_in_txn
FROM pg_stat_activity
WHERE datname IS NOT NULL
GROUP BY application_name, client_addr, datname
ORDER BY connections DESC;
"
```

### Step 4: Identify the Connection-Leaking Service

Check connection pool metrics from each service:

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $SERVICE ---"
  kubectl port-forward -n clario360 svc/$SERVICE 8080:8080 &
  PF_PID=$!
  sleep 2
  curl -s http://localhost:8080/metrics 2>/dev/null | grep -E "db_open_connections|db_max_open|db_in_use|db_idle|db_wait_count|db_wait_duration|sql_open_connections" | head -10
  kill $PF_PID 2>/dev/null
  echo ""
done
```

### Step 5: Find Long-Running Queries Holding Connections

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT
    pid,
    now() - pg_stat_activity.query_start AS duration,
    state,
    application_name,
    client_addr,
    datname,
    LEFT(query, 120) AS query_preview
FROM pg_stat_activity
WHERE state != 'idle'
  AND query NOT ILIKE '%pg_stat_activity%'
  AND datname IS NOT NULL
ORDER BY duration DESC
LIMIT 20;
"
```

### Step 6: Find Idle-in-Transaction Connections (Likely Leaks)

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT
    pid,
    now() - xact_start AS transaction_duration,
    now() - state_change AS time_in_state,
    application_name,
    client_addr,
    datname,
    LEFT(query, 120) AS last_query
FROM pg_stat_activity
WHERE state = 'idle in transaction'
ORDER BY xact_start ASC
LIMIT 20;
"
```

Connections that have been `idle in transaction` for more than 5 minutes are almost certainly leaked.

### Step 7: Check for Lock Contention

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT
    blocked_locks.pid AS blocked_pid,
    blocked_activity.application_name AS blocked_app,
    blocking_locks.pid AS blocking_pid,
    blocking_activity.application_name AS blocking_app,
    blocked_activity.state AS blocked_state,
    now() - blocked_activity.query_start AS blocked_duration,
    LEFT(blocked_activity.query, 100) AS blocked_query,
    LEFT(blocking_activity.query, 100) AS blocking_query
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
    AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
    AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
    AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
    AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
    AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
    AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
    AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
    AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
    AND blocking_locks.pid != blocked_locks.pid
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted
ORDER BY blocked_duration DESC;
"
```

### Step 8: Check Service-Side Connection Pool Configuration

```bash
# Check environment variables for pool configuration
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $SERVICE ---"
  kubectl get deployment $SERVICE -n clario360 -o jsonpath='{.spec.template.spec.containers[0].env}' 2>/dev/null | \
    jq -r '.[] | select(.name | test("DB|POOL|PG|DATABASE|CONN"; "i")) | "\(.name)=\(.value // .valueFrom)"' 2>/dev/null
  echo ""
done
```

### Step 9: Check Application Logs for Connection Errors

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $SERVICE ---"
  kubectl logs -n clario360 -l app=$SERVICE --tail=100 --timestamps | \
    grep -i -E "too many clients|connection pool|connection refused|connection reset|dial tcp.*refused|max_connections" | tail -3
  echo ""
done
```

### Step 10: Check PostgreSQL Configuration Limits

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT name, setting, unit, short_desc
FROM pg_settings
WHERE name IN (
    'max_connections',
    'superuser_reserved_connections',
    'idle_in_transaction_session_timeout',
    'statement_timeout',
    'lock_timeout'
)
ORDER BY name;
"
```

---

## Resolution Steps

### Resolution A: Kill Idle-in-Transaction Connections (Immediate Relief)

Terminate connections that have been idle in transaction for more than 5 minutes:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT pg_terminate_backend(pid), pid, application_name, datname,
       now() - xact_start AS txn_duration
FROM pg_stat_activity
WHERE state = 'idle in transaction'
  AND now() - xact_start > interval '5 minutes';
"
```

### Resolution B: Kill Long-Running Queries

Terminate queries that have been running for more than 10 minutes:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT pg_terminate_backend(pid), pid, application_name, datname,
       now() - query_start AS query_duration,
       LEFT(query, 100) AS query_preview
FROM pg_stat_activity
WHERE state = 'active'
  AND query NOT ILIKE '%pg_stat_activity%'
  AND now() - query_start > interval '10 minutes';
"
```

### Resolution C: Restart the Connection-Leaking Service

If a specific service is leaking connections, restart it to force pool reset:

```bash
kubectl rollout restart deployment/<SERVICE> -n clario360
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

### Resolution D: Increase Service Connection Pool Size

If the pool is correctly configured but too small for the workload:

```bash
kubectl set env deployment/<SERVICE> -n clario360 \
  DB_MAX_OPEN_CONNS=50 \
  DB_MAX_IDLE_CONNS=25 \
  DB_CONN_MAX_LIFETIME=300s \
  DB_CONN_MAX_IDLE_TIME=60s

kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

### Resolution E: Increase PostgreSQL max_connections

If the server-side limit is hit across all services:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
ALTER SYSTEM SET max_connections = 300;
"

# Requires PostgreSQL restart to take effect
kubectl rollout restart statefulset/postgresql -n clario360
kubectl rollout status statefulset/postgresql -n clario360 --timeout=300s
```

**Warning**: Increasing `max_connections` significantly increases PostgreSQL shared memory usage. Each connection uses approximately 5-10 MB of RAM. Ensure the PostgreSQL pod has sufficient memory.

### Resolution F: Set Idle-in-Transaction Timeout (Prevent Future Leaks)

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
-- Set server-side timeout to automatically kill idle-in-transaction sessions
ALTER SYSTEM SET idle_in_transaction_session_timeout = '300000';  -- 5 minutes in ms
SELECT pg_reload_conf();

-- Verify the setting
SHOW idle_in_transaction_session_timeout;
"
```

### Resolution G: Set Statement Timeout for All Connections

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
ALTER SYSTEM SET statement_timeout = '60000';  -- 60 seconds in ms
SELECT pg_reload_conf();

SHOW statement_timeout;
"
```

### Resolution H: Deploy PgBouncer for Connection Pooling

If connection exhaustion is a recurring issue, deploy PgBouncer as a connection multiplexer:

```bash
# Check if PgBouncer is already deployed
kubectl get deployment pgbouncer -n clario360 2>/dev/null

# If not, apply PgBouncer deployment
kubectl apply -n clario360 -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pgbouncer
  namespace: clario360
  labels:
    app: pgbouncer
spec:
  replicas: 2
  selector:
    matchLabels:
      app: pgbouncer
  template:
    metadata:
      labels:
        app: pgbouncer
    spec:
      containers:
      - name: pgbouncer
        image: edoburu/pgbouncer:1.22.0
        ports:
        - containerPort: 5432
        env:
        - name: DATABASE_URL
          value: "postgresql://clario360@postgresql.clario360.svc.cluster.local:5432"
        - name: POOL_MODE
          value: "transaction"
        - name: DEFAULT_POOL_SIZE
          value: "20"
        - name: MAX_CLIENT_CONN
          value: "500"
        - name: MAX_DB_CONNECTIONS
          value: "100"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: pgbouncer
  namespace: clario360
spec:
  selector:
    app: pgbouncer
  ports:
  - port: 5432
    targetPort: 5432
EOF

# Update services to use PgBouncer instead of direct PostgreSQL
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl set env deployment/$SERVICE -n clario360 DB_HOST=pgbouncer.clario360.svc.cluster.local
done
```

### Resolution I: Emergency -- Terminate All Idle Connections

If the platform is in a complete outage due to connection exhaustion:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT pg_terminate_backend(pid), pid, application_name, state, datname
FROM pg_stat_activity
WHERE state IN ('idle', 'idle in transaction', 'idle in transaction (aborted)')
  AND pid != pg_backend_pid()
  AND datname IS NOT NULL;
"
```

---

## Verification

After applying a resolution, verify connection pool health:

```bash
# 1. Check total PostgreSQL connections have decreased
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT
    max_conn,
    used,
    max_conn - used AS available,
    ROUND((used::float / max_conn::float) * 100, 1) AS pct_used
FROM
    (SELECT count(*) AS used FROM pg_stat_activity) t,
    (SELECT setting::int AS max_conn FROM pg_settings WHERE name = 'max_connections') s;
"

# 2. Verify no idle-in-transaction connections remain
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT count(*) AS idle_in_txn FROM pg_stat_activity WHERE state = 'idle in transaction';
"

# 3. Verify service health endpoints pass
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl port-forward -n clario360 svc/$SERVICE 8080:8080 &
  PF_PID=$!
  sleep 2
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/readyz)
  echo "$SERVICE: $STATUS"
  kill $PF_PID 2>/dev/null
done

# 4. Check connection pool metrics from the previously affected service
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2
curl -s http://localhost:8080/metrics | grep -E "db_open_connections|db_max_open|db_in_use|db_idle|db_wait_count"
kill $PF_PID

# 5. Verify no connection errors in recent logs
kubectl logs -n clario360 -l app=<SERVICE> --since=5m --timestamps | grep -i -c -E "too many clients|connection pool|connection refused"

# 6. Confirm connections per database are balanced
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT datname, count(*) AS connections FROM pg_stat_activity WHERE datname IS NOT NULL GROUP BY datname ORDER BY connections DESC;
"
```

---

## Post-Incident Checklist

- [ ] Confirm total PostgreSQL connections are below 70% of `max_connections`.
- [ ] Confirm zero `idle in transaction` connections older than 5 minutes.
- [ ] Confirm all service `/readyz` endpoints return HTTP 200.
- [ ] Verify Prometheus connection pool alerts have cleared in Alertmanager.
- [ ] Check Grafana connection pool dashboard shows normal values.
- [ ] If a connection leak was identified, create a ticket to fix the code.
- [ ] If `idle_in_transaction_session_timeout` was set, verify it persists across PostgreSQL restarts.
- [ ] Review service connection pool configuration across all services for consistency.
- [ ] Consider deploying PgBouncer if not already in place.
- [ ] Document root cause and corrective actions.
- [ ] Add connection pool exhaustion to load testing scenarios.
- [ ] Review and update connection pool alerting thresholds.

---

## Related Links

| Resource                         | Link                                                                     |
|----------------------------------|--------------------------------------------------------------------------|
| PostgreSQL Connection Docs       | https://www.postgresql.org/docs/16/runtime-config-connection.html        |
| Go database/sql Pool Docs        | https://pkg.go.dev/database/sql#DB.SetMaxOpenConns                       |
| PgBouncer Documentation          | https://www.pgbouncer.org/config.html                                    |
| Grafana Dashboards               | https://grafana.clario360.internal/dashboards                            |
| Alertmanager                     | https://alertmanager.clario360.internal                                  |
| IR-002 Database Failure          | [../incident-response/IR-002-database-failure.md](../incident-response/IR-002-database-failure.md) |
| TS-006 Search Problems           | [TS-006-search-not-returning.md](./TS-006-search-not-returning.md)       |
| TS-007 High CPU Usage            | [TS-007-high-cpu-usage.md](./TS-007-high-cpu-usage.md)                   |
