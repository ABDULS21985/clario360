# IR-009: Data Integrity Issue — Data Corruption

| Field            | Value                                    |
|------------------|------------------------------------------|
| Runbook ID       | IR-009                                   |
| Title            | Data Integrity Issue — Data Corruption   |
| Severity         | P1 — Critical                            |
| Owner            | Platform Team                            |
| Last Updated     | 2026-03-08                               |
| Review Frequency | Quarterly                                |
| Approver         | Platform Lead                            |

---

## Summary

This runbook covers diagnosis and resolution of data integrity issues in the Clario 360 platform. Data corruption can manifest as broken audit hash chains, foreign key constraint violations, character encoding errors, partial writes from interrupted transactions, or inconsistent state across services. Data integrity issues are P1 Critical because they undermine platform trustworthiness and may indicate deeper problems such as application bugs, hardware failures, or security breaches.

**CRITICAL**: Before modifying any data, always create a backup of the affected tables. Never run UPDATE or DELETE statements without a WHERE clause, and always test queries with SELECT first.

---

## Symptoms

- **Alerts**: `AuditHashChainBroken`, `DatabaseConstraintViolation`, `DataConsistencyCheckFailed`
- Audit service reports hash chain verification failures
- Application errors referencing foreign key violations: `ERROR: insert or update on table ... violates foreign key constraint`
- Character encoding errors: `ERROR: invalid byte sequence for encoding "UTF8"`
- API responses returning inconsistent data (e.g., references to deleted entities)
- Reports showing incorrect counts or mismatched totals
- Database constraint violations in application logs
- Partial records: rows with required fields set to NULL
- Grafana `Database Performance` dashboard (`/d/db-performance`) showing increased error rates

---

## Impact Assessment

| Scope                        | Impact                                                            |
|------------------------------|-------------------------------------------------------------------|
| Audit hash chain broken      | Audit trail integrity compromised; regulatory compliance at risk  |
| Foreign key violations       | Data relationships broken; cascading errors in dependent queries  |
| Encoding corruption          | Data display errors; search failures; export corruption           |
| Partial writes               | Incomplete records; business logic errors                        |
| Cross-service inconsistency  | Services hold conflicting state; stale cache references           |
| Index corruption             | Query performance degradation; incorrect query results            |

---

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export GRAFANA_URL=https://grafana.clario360.io
export BACKUP_DIR=/tmp/data-recovery-$(date +%Y%m%d-%H%M%S)
mkdir -p ${BACKUP_DIR}
```

- `kubectl` configured with access to the `clario360` namespace
- `psql` client installed (PostgreSQL 15+)
- Access to database backups (verify backup location before starting)
- Appropriate K8s RBAC permissions (`clario360-admin` role)

---

## Diagnosis Steps

### Step 1: Identify which database is affected

Check for errors in each service's logs:

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "=== ${SVC} ==="
  kubectl logs -n clario360 -l app=${SVC} --since=1h --timestamps | grep -iE "constraint|violation|corrupt|integrity|invalid byte|encoding|hash.*mismatch|hash.*broken" | tail -5
done
```

### Step 2: Check PostgreSQL error logs

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -c "
SELECT log_time, error_severity, message
FROM pg_catalog.pg_stat_activity
WHERE state = 'active'
ORDER BY query_start DESC
LIMIT 20;
"
```

```bash
kubectl logs -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') --since=2h | grep -iE "ERROR|FATAL|PANIC|constraint|violation" | tail -30
```

### Step 3: Verify audit hash chain integrity

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
WITH ordered_logs AS (
  SELECT
    id,
    entry_hash,
    prev_hash,
    LAG(entry_hash) OVER (ORDER BY id) AS expected_prev_hash
  FROM audit_logs
  ORDER BY id
)
SELECT
  id,
  entry_hash,
  prev_hash,
  expected_prev_hash,
  CASE
    WHEN prev_hash IS NULL AND id = (SELECT MIN(id) FROM audit_logs) THEN 'OK (first entry)'
    WHEN prev_hash = expected_prev_hash THEN 'OK'
    ELSE 'BROKEN'
  END AS chain_status
FROM ordered_logs
WHERE prev_hash IS DISTINCT FROM expected_prev_hash
  AND NOT (prev_hash IS NULL AND id = (SELECT MIN(id) FROM audit_logs))
ORDER BY id
LIMIT 20;
"
```

### Step 4: Find the first broken link in the hash chain

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
WITH ordered_logs AS (
  SELECT
    id,
    created_at,
    action,
    user_id,
    entry_hash,
    prev_hash,
    LAG(entry_hash) OVER (ORDER BY id) AS expected_prev_hash
  FROM audit_logs
  ORDER BY id
)
SELECT id, created_at, action, user_id, prev_hash, expected_prev_hash
FROM ordered_logs
WHERE prev_hash IS DISTINCT FROM expected_prev_hash
  AND NOT (prev_hash IS NULL AND id = (SELECT MIN(id) FROM audit_logs))
ORDER BY id
LIMIT 1;
"
```

### Step 5: Check for foreign key violations across all databases

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} ==="
  kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d ${DB} -c "
  SELECT
    tc.table_name,
    tc.constraint_name,
    tc.constraint_type,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
  FROM information_schema.table_constraints tc
  JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
  JOIN information_schema.constraint_column_usage ccu
    ON ccu.constraint_name = tc.constraint_name
  WHERE tc.constraint_type = 'FOREIGN KEY';
  "
done
```

### Step 6: Find orphaned records (foreign key targets missing)

Example for platform_core (adapt table/column names as needed):

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT 'user_roles with missing user' AS issue, COUNT(*) AS count
FROM user_roles ur
LEFT JOIN users u ON ur.user_id = u.id
WHERE u.id IS NULL
UNION ALL
SELECT 'user_roles with missing role' AS issue, COUNT(*) AS count
FROM user_roles ur
LEFT JOIN roles r ON ur.role_id = r.id
WHERE r.id IS NULL
UNION ALL
SELECT 'sessions with missing user' AS issue, COUNT(*) AS count
FROM sessions s
LEFT JOIN users u ON s.user_id = u.id
WHERE u.id IS NULL
UNION ALL
SELECT 'audit_logs with missing user' AS issue, COUNT(*) AS count
FROM audit_logs al
LEFT JOIN users u ON al.user_id = u.id
WHERE al.user_id IS NOT NULL AND u.id IS NULL;
"
```

### Step 7: Check for encoding issues

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} encoding ==="
  kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d ${DB} -c "
  SELECT datname, encoding, datcollate, datctype
  FROM pg_database WHERE datname = '${DB}';
  "
done
```

### Step 8: Check for NULL values in required columns

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB}: NOT NULL violations ==="
  kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d ${DB} -c "
  SELECT table_name, column_name
  FROM information_schema.columns
  WHERE is_nullable = 'NO'
    AND table_schema = 'public'
    AND column_default IS NULL
    AND column_name NOT IN ('id')
  ORDER BY table_name, ordinal_position;
  "
done
```

### Step 9: Check for duplicate records that should be unique

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT 'duplicate user emails' AS issue, email, COUNT(*) AS count
FROM users
GROUP BY email
HAVING COUNT(*) > 1
UNION ALL
SELECT 'duplicate role names' AS issue, name, COUNT(*) AS count
FROM roles
GROUP BY name
HAVING COUNT(*) > 1;
"
```

### Step 10: Verify index consistency

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB}: Invalid indexes ==="
  kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d ${DB} -c "
  SELECT indexrelid::regclass AS index_name, indisvalid, indisready
  FROM pg_index
  WHERE NOT indisvalid OR NOT indisready;
  "
done
```

### Step 11: Check PostgreSQL data checksums (if enabled)

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -c "SHOW data_checksums;"
```

If checksums are enabled, check for corruption:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -c "
SELECT datname, checksum_failures, checksum_last_failure
FROM pg_stat_database
WHERE checksum_failures > 0;
"
```

---

## Resolution Steps

### Option A: Restore from backup (full database or specific tables)

**Step 1**: List available backups:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- ls -la /var/lib/postgresql/backups/
```

**Step 2**: Create a safety backup of the current (corrupted) state:

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- pg_dump -U ${PG_USER} -h localhost -d ${DB} --no-owner > ${BACKUP_DIR}/${DB}-pre-restore-$(date +%Y%m%d-%H%M%S).sql
done
```

**Step 3**: Restore a specific table from backup (example: audit_logs):

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- pg_dump -U ${PG_USER} -h localhost -d platform_core -t audit_logs --no-owner > ${BACKUP_DIR}/audit_logs-current.sql
```

```bash
kubectl cp ${BACKUP_DIR}/audit_logs-backup.sql clario360/$(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}'):/tmp/audit_logs-backup.sql
```

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
BEGIN;
ALTER TABLE audit_logs RENAME TO audit_logs_corrupted;
\i /tmp/audit_logs-backup.sql
COMMIT;
"
```

**Step 4**: Verify the restored table:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT COUNT(*) AS total_rows FROM audit_logs;
"
```

### Option B: Repair audit hash chain

**Step 1**: Backup the audit_logs table:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- pg_dump -U ${PG_USER} -h localhost -d platform_core -t audit_logs --no-owner > ${BACKUP_DIR}/audit_logs-before-repair.sql
```

**Step 2**: Identify the break point:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
WITH ordered_logs AS (
  SELECT
    id,
    entry_hash,
    prev_hash,
    LAG(entry_hash) OVER (ORDER BY id) AS expected_prev_hash
  FROM audit_logs
  ORDER BY id
)
SELECT id, prev_hash, expected_prev_hash
FROM ordered_logs
WHERE prev_hash IS DISTINCT FROM expected_prev_hash
  AND NOT (prev_hash IS NULL AND id = (SELECT MIN(id) FROM audit_logs))
ORDER BY id
LIMIT 1;
"
```

**Step 3**: Fix the broken link by updating prev_hash to match the previous entry's entry_hash:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
UPDATE audit_logs
SET prev_hash = (
  SELECT entry_hash
  FROM audit_logs AS prev
  WHERE prev.id = (
    SELECT MAX(id) FROM audit_logs WHERE id < <BROKEN_ENTRY_ID>
  )
)
WHERE id = <BROKEN_ENTRY_ID>;
"
```

**Step 4**: If multiple entries need hash recalculation, rebuild the chain from the break point. This requires the application's hash function. Use the audit-service's repair endpoint if available:

```bash
curl -s -X POST http://$(kubectl get pod -n clario360 -l app=audit-service -o jsonpath='{.items[0].status.podIP}'):8080/api/v1/audit/repair-chain \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -d '{"from_id": <BROKEN_ENTRY_ID>}'
```

**Step 5**: Verify the chain is now intact:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
WITH ordered_logs AS (
  SELECT
    id,
    entry_hash,
    prev_hash,
    LAG(entry_hash) OVER (ORDER BY id) AS expected_prev_hash
  FROM audit_logs
  ORDER BY id
)
SELECT COUNT(*) AS broken_links
FROM ordered_logs
WHERE prev_hash IS DISTINCT FROM expected_prev_hash
  AND NOT (prev_hash IS NULL AND id = (SELECT MIN(id) FROM audit_logs));
"
```

Expected: `broken_links = 0`.

### Option C: Fix orphaned records (foreign key violations)

**Step 1**: Backup the affected tables:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- pg_dump -U ${PG_USER} -h localhost -d platform_core -t user_roles -t sessions --no-owner > ${BACKUP_DIR}/orphan-tables-backup.sql
```

**Step 2**: Delete orphaned user_roles referencing deleted users:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
BEGIN;
DELETE FROM user_roles
WHERE user_id NOT IN (SELECT id FROM users);
DELETE FROM user_roles
WHERE role_id NOT IN (SELECT id FROM roles);
COMMIT;
"
```

**Step 3**: Delete orphaned sessions:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
DELETE FROM sessions
WHERE user_id NOT IN (SELECT id FROM users);
"
```

### Option D: Fix encoding issues

**Step 1**: Identify rows with invalid encoding:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT id, octet_length(details::text) AS bytes, char_length(details::text) AS chars
FROM audit_logs
WHERE octet_length(details::text) != char_length(details::text)
LIMIT 20;
"
```

**Step 2**: Fix rows with encoding issues by converting to valid UTF-8:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
UPDATE audit_logs
SET details = convert_from(convert_to(details::text, 'UTF8'), 'UTF8')::jsonb
WHERE id = <AFFECTED_ROW_ID>;
"
```

### Option E: Rebuild invalid indexes

**Step 1**: Identify invalid indexes:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT indexrelid::regclass AS index_name, indrelid::regclass AS table_name
FROM pg_index
WHERE NOT indisvalid;
"
```

**Step 2**: Rebuild invalid indexes concurrently (does not lock the table):

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
REINDEX INDEX CONCURRENTLY <INDEX_NAME>;
"
```

**Step 3**: Rebuild all indexes in a database (if multiple are affected):

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
REINDEX DATABASE CONCURRENTLY platform_core;
"
```

### Option F: Clear stale cache after data repair

After fixing any data, clear Redis cache to prevent stale references:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=redis -o jsonpath='{.items[0].metadata.name}') -- redis-cli -h redis FLUSHALL
```

Restart affected services to clear in-memory caches:

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout restart deployment/${SVC} -n clario360
done
```

Wait for all rollouts:

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout status deployment/${SVC} -n clario360 --timeout=120s
done
```

---

## Verification

### Step 1: Verify audit hash chain integrity

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
WITH ordered_logs AS (
  SELECT
    id,
    entry_hash,
    prev_hash,
    LAG(entry_hash) OVER (ORDER BY id) AS expected_prev_hash
  FROM audit_logs
  ORDER BY id
)
SELECT COUNT(*) AS broken_links
FROM ordered_logs
WHERE prev_hash IS DISTINCT FROM expected_prev_hash
  AND NOT (prev_hash IS NULL AND id = (SELECT MIN(id) FROM audit_logs));
"
```

Expected: `broken_links = 0`.

### Step 2: Verify no foreign key violations

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT 'orphaned user_roles' AS check,
  COUNT(*) AS violations
FROM user_roles ur
LEFT JOIN users u ON ur.user_id = u.id
WHERE u.id IS NULL
UNION ALL
SELECT 'orphaned sessions' AS check,
  COUNT(*) AS violations
FROM sessions s
LEFT JOIN users u ON s.user_id = u.id
WHERE u.id IS NULL;
"
```

Expected: All violation counts are `0`.

### Step 3: Verify all indexes are valid

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} ==="
  kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d ${DB} -c "
  SELECT COUNT(*) AS invalid_indexes FROM pg_index WHERE NOT indisvalid;
  "
done
```

Expected: `invalid_indexes = 0` for all databases.

### Step 4: Verify all services are healthy

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "${SVC}: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=${SVC} -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "UNHEALTHY"
done
```

### Step 5: Verify services pass readiness checks

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "${SVC}: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=${SVC} -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/readyz 2>/dev/null || echo "NOT READY"
done
```

### Step 6: Run a write test across databases

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo -n "${DB}: "
  kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d ${DB} -c "CREATE TABLE _integrity_test (id int); DROP TABLE _integrity_test; SELECT 'OK';" -t | tr -d ' \n'
  echo ""
done
```

Expected: All databases return `OK`.

---

## Post-Incident Checklist

- [ ] Document the root cause of the data corruption (application bug, hardware, concurrent writes, etc.)
- [ ] Verify all backups completed before the restoration are retained
- [ ] Confirm the corrupted table backup is preserved for forensic analysis
- [ ] Review application code that writes to the affected tables for potential race conditions
- [ ] Add or improve database constraints to prevent the specific corruption pattern
- [ ] Consider enabling PostgreSQL data checksums if not already enabled
- [ ] Review transaction isolation levels for affected write paths
- [ ] Verify automated backup schedule is current and backups are restorable
- [ ] Schedule a full database consistency check across all databases
- [ ] If hash chain was broken, investigate whether it was caused by a security incident (see IR-008)
- [ ] Update monitoring alerts for data integrity checks
- [ ] Create a Jira ticket for any application-level fixes needed
- [ ] Update this runbook with any new patterns discovered

---

## Related Links

- [IR-008: Suspected Security Incident](IR-008-security-breach.md) -- Data corruption may indicate a breach
- [OP-004: Backup Verification](../operations/OP-004-backup-verification.md) -- Backup integrity procedures
- [OP-007: Database Maintenance](../operations/OP-007-database-maintenance.md) -- VACUUM, REINDEX procedures
- [TS-009: Audit Chain Broken](../troubleshooting/TS-009-audit-chain-broken.md) -- Audit-specific troubleshooting
- [DP-004: Database Migration](../deployment/DP-004-database-migration.md) -- Migration procedures
- Grafana Database Performance: `${GRAFANA_URL}/d/db-performance`
- PostgreSQL Data Checksums: https://www.postgresql.org/docs/15/checksums.html
- PostgreSQL REINDEX: https://www.postgresql.org/docs/15/sql-reindex.html
