# OP-007: PostgreSQL Database Maintenance

| Field | Value |
|-------|-------|
| **Runbook ID** | OP-007 |
| **Title** | PostgreSQL Database Maintenance |
| **Frequency** | Weekly / Monthly / Quarterly (see schedule below) |
| **Owner** | Platform Database Team |
| **Last Updated** | 2026-03-08 |
| **Estimated Duration** | Weekly: 30 min, Monthly: 1–2 hours, Quarterly: 2–4 hours |
| **Risk Level** | Low (ANALYZE/VACUUM) to Medium (REINDEX) |
| **Approval Required** | No for weekly/monthly; Yes (CAB) for quarterly REINDEX |
| **Maintenance Window** | Weekly: any off-peak; Quarterly REINDEX: scheduled window |

## Summary

This runbook covers recurring PostgreSQL maintenance tasks for the Clario 360 platform databases:

| Schedule | Tasks |
|----------|-------|
| **Weekly** | ANALYZE on all databases |
| **Monthly** | VACUUM ANALYZE on all databases, audit log partition management |
| **Quarterly** | REINDEX CONCURRENTLY on large tables, bloat analysis, index audit |

### Databases

| Database | Primary Service | Description |
|----------|----------------|-------------|
| `platform_core` | iam-service, workflow-engine | Users, roles, tenants, workflows |
| `cyber_db` | cyber-service | Cybersecurity findings, scans |
| `data_db` | data-service | Data governance, lineage |
| `acta_db` | acta-service | Document management, compliance |
| `lex_db` | lex-service | Legal/regulatory tracking |
| `visus_db` | visus-service | Reporting, dashboards |

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export PG_PORT=5432

# All databases
export DATABASES="platform_core cyber_db data_db acta_db lex_db visus_db"
```

Verify database connectivity:

```bash
for db in $DATABASES; do
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "SELECT version();" -t | head -1
  echo "=== $db: connected ==="
done
```

---

## Weekly: ANALYZE All Databases

**Purpose:** Update table statistics so the query planner can generate optimal execution plans. This is lightweight and does not lock tables.

**When:** Every Sunday at 03:00 UTC (or via CronJob).

### Procedure

```bash
for db in $DATABASES; do
  echo "=== ANALYZE $db ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "ANALYZE VERBOSE;" 2>&1 | tail -5
  echo ""
done
```

### Verify statistics are fresh

```bash
for db in $DATABASES; do
  echo "=== $db: stale statistics check ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT schemaname, relname, last_analyze, last_autoanalyze,
           n_live_tup, n_dead_tup
    FROM pg_stat_user_tables
    WHERE last_analyze < NOW() - INTERVAL '7 days'
       OR last_analyze IS NULL
    ORDER BY n_live_tup DESC
    LIMIT 10;
  "
done
```

---

## Monthly: VACUUM ANALYZE All Databases

**Purpose:** Reclaim storage from dead tuples and update statistics. `VACUUM` does not lock tables for reads/writes (non-FULL variant). Run on the first Saturday of each month.

**When:** First Saturday of each month at 02:00 UTC.

### Procedure

```bash
for db in $DATABASES; do
  echo "=== VACUUM ANALYZE $db — started at $(date -u +%H:%M:%S) ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "VACUUM (VERBOSE, ANALYZE);" 2>&1 | tail -10
  echo "=== Completed at $(date -u +%H:%M:%S) ==="
  echo ""
done
```

### Check dead tuple counts after VACUUM

```bash
for db in $DATABASES; do
  echo "=== $db: dead tuple report ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT schemaname, relname,
           n_live_tup,
           n_dead_tup,
           CASE WHEN n_live_tup > 0
                THEN round(100.0 * n_dead_tup / n_live_tup, 2)
                ELSE 0
           END AS dead_pct,
           last_vacuum,
           last_autovacuum
    FROM pg_stat_user_tables
    WHERE n_dead_tup > 1000
    ORDER BY n_dead_tup DESC
    LIMIT 10;
  "
done
```

---

## Monthly: Audit Log Partition Management

**Purpose:** The `audit_logs` table in `platform_core` is partitioned by month. Each month, create the next month's partition before it is needed.

### Check existing partitions

```bash
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  SELECT inhrelid::regclass AS partition_name,
         pg_size_pretty(pg_relation_size(inhrelid)) AS size
  FROM pg_inherits
  WHERE inhparent = 'public.audit_logs'::regclass
  ORDER BY inhrelid::regclass;
"
```

### Create next month's partition

```bash
# Calculate next month boundaries
NEXT_MONTH_START=$(date -u -d "$(date +%Y-%m-01) +1 month" +%Y-%m-%d 2>/dev/null || date -u -v1d -v+1m +%Y-%m-%d)
NEXT_MONTH_END=$(date -u -d "$NEXT_MONTH_START +1 month" +%Y-%m-%d 2>/dev/null || date -u -j -f "%Y-%m-%d" "$NEXT_MONTH_START" -v+1m +%Y-%m-%d)
PARTITION_NAME="audit_logs_$(date -u -d "$NEXT_MONTH_START" +%Y_%m 2>/dev/null || date -u -j -f "%Y-%m-%d" "$NEXT_MONTH_START" +%Y_%m)"

echo "Creating partition: $PARTITION_NAME ($NEXT_MONTH_START to $NEXT_MONTH_END)"

kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  CREATE TABLE IF NOT EXISTS public.${PARTITION_NAME}
    PARTITION OF public.audit_logs
    FOR VALUES FROM ('${NEXT_MONTH_START}') TO ('${NEXT_MONTH_END}');
"
```

### Verify the new partition

```bash
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  SELECT inhrelid::regclass AS partition_name,
         pg_size_pretty(pg_relation_size(inhrelid)) AS size
  FROM pg_inherits
  WHERE inhparent = 'public.audit_logs'::regclass
  ORDER BY inhrelid::regclass;
"
```

### Detach old partitions (older than 13 months, per retention policy)

```bash
CUTOFF_DATE=$(date -u -d "13 months ago" +%Y_%m 2>/dev/null || date -u -v-13m +%Y_%m)
echo "Detaching partitions older than $CUTOFF_DATE"

kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  SELECT inhrelid::regclass AS partition_name
  FROM pg_inherits
  WHERE inhparent = 'public.audit_logs'::regclass
    AND inhrelid::regclass::text < 'audit_logs_${CUTOFF_DATE}'
  ORDER BY inhrelid::regclass;
"

# For each partition to detach (review output above first):
# kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c \
#   "ALTER TABLE public.audit_logs DETACH PARTITION public.audit_logs_YYYY_MM;"
# kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c \
#   "DROP TABLE public.audit_logs_YYYY_MM;"
```

---

## Quarterly: REINDEX CONCURRENTLY on Large Tables

**Purpose:** Rebuild bloated indexes without locking the table. `REINDEX CONCURRENTLY` creates a new index alongside the old one, then swaps them.

**When:** First Saturday of each quarter at 02:00 UTC during maintenance window.

### Identify large/bloated indexes

```bash
for db in $DATABASES; do
  echo "=== $db: index sizes ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT schemaname, tablename, indexname,
           pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
           idx_scan AS index_scans
    FROM pg_stat_user_indexes
    JOIN pg_index USING (indexrelid)
    ORDER BY pg_relation_size(indexrelid) DESC
    LIMIT 15;
  "
done
```

### Check index bloat

```bash
for db in $DATABASES; do
  echo "=== $db: index bloat estimate ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT
      nspname AS schema,
      relname AS table,
      indexrelname AS index,
      pg_size_pretty(pg_relation_size(i.indexrelid)) AS index_size,
      idx_scan,
      CASE WHEN idx_scan = 0 THEN 'UNUSED' ELSE 'ACTIVE' END AS status
    FROM pg_stat_user_indexes ui
    JOIN pg_index i ON ui.indexrelid = i.indexrelid
    WHERE pg_relation_size(i.indexrelid) > 10485760
    ORDER BY pg_relation_size(i.indexrelid) DESC
    LIMIT 20;
  "
done
```

### Perform REINDEX CONCURRENTLY

```bash
# Reindex the largest tables in each database
# REINDEX CONCURRENTLY does not block reads or writes

# platform_core: users, audit_logs, tenants, workflows
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  REINDEX INDEX CONCURRENTLY idx_users_email;
  REINDEX INDEX CONCURRENTLY idx_users_tenant_id;
  REINDEX INDEX CONCURRENTLY idx_audit_logs_created_at;
  REINDEX INDEX CONCURRENTLY idx_audit_logs_tenant_id;
  REINDEX INDEX CONCURRENTLY idx_audit_logs_actor_id;
  REINDEX INDEX CONCURRENTLY idx_workflows_tenant_id;
  REINDEX INDEX CONCURRENTLY idx_workflows_status;
"

# cyber_db
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d cyber_db -c "
  REINDEX INDEX CONCURRENTLY idx_findings_severity;
  REINDEX INDEX CONCURRENTLY idx_findings_tenant_id;
  REINDEX INDEX CONCURRENTLY idx_scans_status;
"

# data_db
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d data_db -c "
  REINDEX INDEX CONCURRENTLY idx_datasets_tenant_id;
  REINDEX INDEX CONCURRENTLY idx_lineage_source_id;
  REINDEX INDEX CONCURRENTLY idx_lineage_target_id;
"

# acta_db
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d acta_db -c "
  REINDEX INDEX CONCURRENTLY idx_documents_tenant_id;
  REINDEX INDEX CONCURRENTLY idx_documents_status;
"

# lex_db
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d lex_db -c "
  REINDEX INDEX CONCURRENTLY idx_regulations_jurisdiction;
  REINDEX INDEX CONCURRENTLY idx_obligations_status;
"

# visus_db
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d visus_db -c "
  REINDEX INDEX CONCURRENTLY idx_reports_tenant_id;
  REINDEX INDEX CONCURRENTLY idx_dashboards_tenant_id;
"
```

---

## Check Table Sizes and Bloat

Use this at any time to understand database size distribution.

### Table sizes across all databases

```bash
for db in $DATABASES; do
  echo "=== $db: top 15 tables by size ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT schemaname, relname AS table_name,
           pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
           pg_size_pretty(pg_relation_size(relid)) AS table_size,
           pg_size_pretty(pg_total_relation_size(relid) - pg_relation_size(relid)) AS index_size,
           n_live_tup AS live_rows,
           n_dead_tup AS dead_rows
    FROM pg_stat_user_tables
    ORDER BY pg_total_relation_size(relid) DESC
    LIMIT 15;
  "
done
```

### Table bloat estimation

```bash
for db in $DATABASES; do
  echo "=== $db: table bloat estimate ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT
      schemaname, relname,
      n_live_tup,
      n_dead_tup,
      CASE WHEN n_live_tup > 0
           THEN round(100.0 * n_dead_tup / (n_live_tup + n_dead_tup), 2)
           ELSE 0
      END AS bloat_pct,
      pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
      last_vacuum,
      last_autovacuum
    FROM pg_stat_user_tables
    WHERE n_dead_tup > 10000
       OR (n_live_tup > 0 AND n_dead_tup::float / n_live_tup > 0.1)
    ORDER BY n_dead_tup DESC
    LIMIT 20;
  "
done
```

### Database-level sizes

```bash
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d postgres -c "
  SELECT datname,
         pg_size_pretty(pg_database_size(datname)) AS size
  FROM pg_database
  WHERE datname IN ('platform_core', 'cyber_db', 'data_db', 'acta_db', 'lex_db', 'visus_db')
  ORDER BY pg_database_size(datname) DESC;
"
```

---

## Check for Unused Indexes

Unused indexes waste disk space and slow down writes. Review and drop indexes that have zero scans.

```bash
for db in $DATABASES; do
  echo "=== $db: unused indexes (0 scans since last stats reset) ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT schemaname, relname AS table_name,
           indexrelname AS index_name,
           pg_size_pretty(pg_relation_size(i.indexrelid)) AS index_size,
           idx_scan AS times_used
    FROM pg_stat_user_indexes ui
    JOIN pg_index i ON ui.indexrelid = i.indexrelid
    WHERE idx_scan = 0
      AND NOT indisunique
      AND NOT indisprimary
      AND pg_relation_size(i.indexrelid) > 1048576
    ORDER BY pg_relation_size(i.indexrelid) DESC
    LIMIT 20;
  "
done
```

Check when stats were last reset (to contextualize "unused"):

```bash
for db in $DATABASES; do
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT stats_reset FROM pg_stat_bgwriter;
  "
done
```

> **Note:** Only drop unused indexes after verifying stats have been accumulating for at least one full business cycle (30+ days). To drop:
> ```bash
> kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d <database> -c \
>   "DROP INDEX CONCURRENTLY <index_name>;"
> ```

---

## Check pg_stat_statements for Slow Queries

Requires the `pg_stat_statements` extension to be enabled.

### Top 20 queries by total execution time

```bash
for db in $DATABASES; do
  echo "=== $db: top 20 slowest queries ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT
      queryid,
      calls,
      round(total_exec_time::numeric, 2) AS total_time_ms,
      round(mean_exec_time::numeric, 2) AS mean_time_ms,
      round(max_exec_time::numeric, 2) AS max_time_ms,
      rows,
      left(query, 120) AS query_preview
    FROM pg_stat_statements
    WHERE dbid = (SELECT oid FROM pg_database WHERE datname = '$db')
    ORDER BY total_exec_time DESC
    LIMIT 20;
  "
done
```

### Queries with highest mean execution time

```bash
for db in $DATABASES; do
  echo "=== $db: queries with highest mean time (>100ms, >10 calls) ==="
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d $db -c "
    SELECT
      queryid,
      calls,
      round(mean_exec_time::numeric, 2) AS mean_time_ms,
      round(max_exec_time::numeric, 2) AS max_time_ms,
      rows,
      left(query, 120) AS query_preview
    FROM pg_stat_statements
    WHERE dbid = (SELECT oid FROM pg_database WHERE datname = '$db')
      AND mean_exec_time > 100
      AND calls > 10
    ORDER BY mean_exec_time DESC
    LIMIT 20;
  "
done
```

### Reset pg_stat_statements (after quarterly review)

```bash
# Only reset after documenting/exporting the current stats
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  SELECT pg_stat_statements_reset();
"
```

---

## Verification

After completing maintenance tasks, verify database health:

```bash
# 1. Check all services can connect
SERVICES="api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service"
for svc in $SERVICES; do
  READY=$(kubectl -n $NAMESPACE exec deployment/$svc -- wget -qO- http://localhost:8080/readyz 2>/dev/null)
  echo "$svc readiness: $READY"
done

# 2. Check PostgreSQL is accepting connections
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  SELECT count(*) AS active_connections FROM pg_stat_activity WHERE state = 'active';
"

# 3. Check for any long-running queries (maintenance leftovers)
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  SELECT pid, now() - pg_stat_activity.query_start AS duration,
         state, left(query, 80) AS query
  FROM pg_stat_activity
  WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes'
    AND state != 'idle'
  ORDER BY duration DESC;
"

# 4. Check Grafana database dashboard
echo "Review: https://grafana.clario360.io/d/db-performance"
```

---

## Related Links

- [OP-001: Daily Checks](OP-001-daily-checks.md)
- [OP-004: Backup Verification](OP-004-backup-verification.md)
- [IR-002: Database Failure](../incident-response/IR-002-database-failure.md)
- [SC-002: Database Scaling](../scaling/SC-002-database-scaling.md)
- [TS-008: Connection Pool Exhaustion](../troubleshooting/TS-008-connection-pool-exhaustion.md)
- [Grafana — Database Performance](https://grafana.clario360.io/d/db-performance)
