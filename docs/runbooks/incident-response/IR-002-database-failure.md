# IR-002: Database Connectivity Loss

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | IR-002                                                                |
| **Title**          | Database Connectivity Loss                                            |
| **Severity**       | P1 -- Critical                                                        |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | PostgreSQL (platform_core, cyber_db, data_db, acta_db, lex_db, visus_db) |
| **Namespace**      | clario360                                                             |
| **DB Host**        | postgresql.clario360.svc.cluster.local                                |
| **Escalation**     | Database Engineering Lead -> Platform Engineering Lead -> VP Engineering |
| **SLA**            | Acknowledge within 5 minutes, resolve within 30 minutes               |

---

## Summary

This runbook covers incidents where one or more Clario360 platform services lose connectivity to their PostgreSQL databases. Scenarios include the PostgreSQL pod being unreachable, connection pool exhaustion, replication lag on read replicas, and disk space exhaustion on the database volume.

---

## Symptoms

- Application logs showing `pq: connection refused`, `pq: too many connections`, or `dial tcp: i/o timeout` errors.
- `/readyz` endpoints returning non-200 for services with database dependencies.
- Grafana alerts for database connection pool saturation (active connections near max).
- Grafana alerts for replication lag exceeding threshold.
- Slow query response times across multiple services.
- Disk space alerts on the PostgreSQL PersistentVolume.
- Users experiencing timeouts or HTTP 500 errors on data-dependent operations.

---

## Impact Assessment

| Database        | Dependent Services                          | Business Impact                                  |
|-----------------|---------------------------------------------|--------------------------------------------------|
| platform_core   | iam-service, api-gateway, workflow-engine, audit-service, notification-service | Authentication, authorization, workflows, audit -- platform-wide outage |
| cyber_db        | cyber-service                               | Security monitoring and threat detection halted  |
| data_db         | data-service                                | Data ingestion and querying unavailable          |
| acta_db         | acta-service                                | Document management offline                      |
| lex_db          | lex-service                                 | Legal/regulatory tracking offline                |
| visus_db        | visus-service                               | Dashboards and reporting unavailable             |

---

## Prerequisites

- `kubectl` configured with cluster access and `clario360` namespace permissions.
- `psql` client installed locally or access to a pod with `psql`.
- PostgreSQL superuser credentials (stored in Kubernetes secret `postgresql-credentials`).
- Access to Grafana dashboards for database metrics.
- Knowledge of the database backup and restore procedures.

---

## Diagnosis Steps

### Step 1: Check PostgreSQL Pod Status

```bash
kubectl get pods -n clario360 -l app=postgresql -o wide
```

```bash
kubectl get pods -n clario360 -l app=postgresql -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.phase}{"\t"}{.status.containerStatuses[0].ready}{"\n"}{end}'
```

### Step 2: Check PostgreSQL Pod Events and Logs

```bash
kubectl describe pod -n clario360 -l app=postgresql
```

```bash
kubectl logs -n clario360 -l app=postgresql --tail=300 --timestamps
```

Look for:
- `FATAL: too many connections for role`
- `FATAL: the database system is shutting down`
- `PANIC: could not write to file`
- `LOG: checkpoints are occurring too frequently`
- `FATAL: could not open file` (disk full)

### Step 3: Test Database Connectivity from Within the Cluster

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "SELECT 1 AS connectivity_test;"
```

### Step 4: Check Active Connections per Database

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
SELECT datname, count(*) AS connections,
       (SELECT setting::int FROM pg_settings WHERE name='max_connections') AS max_connections
FROM pg_stat_activity
GROUP BY datname
ORDER BY connections DESC;"
```

### Step 5: Check for Long-Running Queries

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state, wait_event_type, wait_event
FROM pg_stat_activity
WHERE state != 'idle'
  AND query NOT ILIKE '%pg_stat_activity%'
ORDER BY duration DESC
LIMIT 20;"
```

### Step 6: Check for Blocked Queries (Lock Contention)

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
SELECT blocked_locks.pid AS blocked_pid,
       blocked_activity.usename AS blocked_user,
       blocking_locks.pid AS blocking_pid,
       blocking_activity.usename AS blocking_user,
       blocked_activity.query AS blocked_statement,
       blocking_activity.query AS blocking_statement
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
WHERE NOT blocked_locks.granted;"
```

### Step 7: Check Replication Status (If Using Replicas)

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
SELECT client_addr, state, sent_lsn, write_lsn, flush_lsn, replay_lsn,
       (sent_lsn - replay_lsn) AS replication_lag_bytes,
       now() - pg_last_xact_replay_timestamp() AS replay_lag_time
FROM pg_stat_replication;"
```

### Step 8: Check Disk Usage

```bash
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- df -h /var/lib/postgresql/data
```

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
SELECT datname, pg_size_pretty(pg_database_size(datname)) AS size
FROM pg_database
ORDER BY pg_database_size(datname) DESC;"
```

### Step 9: Check PersistentVolumeClaim Status

```bash
kubectl get pvc -n clario360 -l app=postgresql
kubectl describe pvc -n clario360 -l app=postgresql
```

### Step 10: Check Service DNS Resolution

```bash
kubectl run dns-debug --rm -it --restart=Never -n clario360 \
  --image=busybox:1.36 \
  -- nslookup postgresql.clario360.svc.cluster.local
```

---

## Resolution Steps

### Scenario A: PostgreSQL Pod Unreachable / CrashLoopBackOff

**1. Restart the PostgreSQL StatefulSet:**

```bash
kubectl rollout restart statefulset/postgresql -n clario360
kubectl rollout status statefulset/postgresql -n clario360 --timeout=180s
```

**2. If the pod fails to start due to corrupted data, check WAL status:**

```bash
kubectl logs -n clario360 -l app=postgresql --tail=100 | grep -i -E "wal|corrupt|recovery"
```

**3. If data corruption is confirmed, restore from backup (coordinate with DBA):**

```bash
# List available backups
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=postgresql-backup -o jsonpath='{.items[0].metadata.name}') -- ls -lt /backups/

# Restore procedure requires DBA coordination -- escalate immediately
```

### Scenario B: Connection Pool Exhaustion

**1. Identify and kill long-running idle connections:**

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'idle'
  AND query_start < now() - interval '10 minutes'
  AND usename != 'clario360_admin';"
```

**2. Kill specific long-running queries (replace `<PID>`):**

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "SELECT pg_terminate_backend(<PID>);"
```

**3. Increase max_connections temporarily (requires restart):**

```bash
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -U clario360_admin -d platform_core -c "ALTER SYSTEM SET max_connections = 300;"

kubectl rollout restart statefulset/postgresql -n clario360
kubectl rollout status statefulset/postgresql -n clario360 --timeout=180s
```

**4. Restart affected application services to reset their connection pools:**

```bash
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout restart deployment/$svc -n clario360
done
```

### Scenario C: Replication Lag

**1. Check replica status and identify the lagging replica:**

(See Diagnosis Step 7 above.)

**2. If the replica is significantly behind, restart it:**

```bash
kubectl delete pod postgresql-replica-0 -n clario360
```

The StatefulSet controller will recreate it and it will re-sync from the primary.

**3. If lag persists, check for heavy write operations on the primary:**

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
SELECT query, calls, total_exec_time, rows, mean_exec_time
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 10;"
```

### Scenario D: Disk Full on Database

**1. Identify the largest tables:**

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)) AS total_size,
       pg_size_pretty(pg_relation_size(schemaname || '.' || tablename)) AS table_size,
       pg_size_pretty(pg_indexes_size(schemaname || '.' || quote_ident(tablename))) AS index_size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname || '.' || tablename) DESC
LIMIT 20;"
```

**2. Clean up old WAL files and run VACUUM:**

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "VACUUM FULL VERBOSE;"
```

**3. Delete old audit log entries if platform_core is the issue (retain last 90 days):**

```bash
kubectl run pg-debug --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
DELETE FROM audit_logs WHERE created_at < now() - interval '90 days';
VACUUM audit_logs;"
```

**4. Expand the PersistentVolumeClaim (if storage class supports expansion):**

```bash
kubectl patch pvc postgresql-data -n clario360 -p '{"spec": {"resources": {"requests": {"storage": "100Gi"}}}}'
```

Verify expansion:

```bash
kubectl get pvc postgresql-data -n clario360 -o jsonpath='{.status.capacity.storage}'
```

---

## Verification

```bash
# 1. PostgreSQL pod is running and ready
kubectl get pods -n clario360 -l app=postgresql

# 2. Connectivity test succeeds
kubectl run pg-verify --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "SELECT 1 AS connectivity_ok;"

# 3. Connection count is within limits
kubectl run pg-verify --rm -it --restart=Never -n clario360 \
  --image=postgres:15 \
  --env="PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h postgresql.clario360.svc.cluster.local -U clario360_admin -d platform_core -c "
SELECT count(*) AS active_connections,
       (SELECT setting::int FROM pg_settings WHERE name='max_connections') AS max_connections
FROM pg_stat_activity;"

# 4. All application services pass readiness checks
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "$svc: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=$svc -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) -- wget -q -O- http://localhost:8080/readyz 2>/dev/null && echo " OK" || echo " FAIL"
done

# 5. Disk usage is below 80%
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- df -h /var/lib/postgresql/data
```

---

## Post-Incident Checklist

- [ ] Confirm PostgreSQL pod is `Running` and `Ready`.
- [ ] Confirm connectivity from all dependent services.
- [ ] Confirm connection count is below 80% of `max_connections`.
- [ ] Confirm disk usage is below 80%.
- [ ] Confirm replication lag (if applicable) is under 1 second.
- [ ] Verify Prometheus/Grafana alerts have cleared.
- [ ] Check that no data was lost during the incident.
- [ ] Notify stakeholders of resolution.
- [ ] Create post-incident review (PIR) ticket.
- [ ] Document root cause and corrective actions.
- [ ] If connection pool exhaustion, review application pool sizes and add PgBouncer if not present.
- [ ] If disk full, set up proactive disk usage alerting at 70% threshold.
- [ ] Schedule a database maintenance window if VACUUM FULL is needed on large tables.

---

## Related Links

| Resource                        | Link                                                         |
|---------------------------------|--------------------------------------------------------------|
| PostgreSQL Troubleshooting      | https://www.postgresql.org/docs/15/monitoring.html           |
| Grafana DB Dashboard            | https://grafana.clario360.internal/d/postgresql               |
| Backup and Restore Procedures   | https://wiki.clario360.internal/db-backup-restore            |
| IR-001 Service Outage           | [IR-001-service-outage.md](./IR-001-service-outage.md)       |
| IR-003 Kafka Failure            | [IR-003-kafka-failure.md](./IR-003-kafka-failure.md)         |
| IR-004 Redis Failure            | [IR-004-redis-failure.md](./IR-004-redis-failure.md)         |
| IR-005 Certificate Expiry       | [IR-005-certificate-expiry.md](./IR-005-certificate-expiry.md) |
