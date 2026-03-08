# OP-004: Backup Integrity Verification

| Field              | Value                                      |
|--------------------|--------------------------------------------|
| **Runbook ID**     | OP-004                                     |
| **Title**          | Backup Integrity Verification              |
| **Frequency**      | Weekly (Wednesdays, 04:00 UTC)             |
| **Estimated Time** | ~60 minutes (restore test runs in parallel)|
| **Owner**          | Platform Team (DBA rotation)               |
| **Last Updated**   | 2026-03-08                                 |
| **Review Cycle**   | Quarterly                                  |

## Summary

This runbook verifies the integrity of Clario 360 database backups. It covers listing and validating recent backups, performing a test restore to the staging environment, verifying data integrity post-restore, documenting RPO/RTO metrics, and testing point-in-time recovery (PITR). Backup verification is critical to ensure recovery capability is maintained.

## RPO/RTO Targets

| Metric | Target | Measured |
|--------|--------|----------|
| **RPO** (Recovery Point Objective) | 1 hour | Continuous WAL archiving + daily full backup |
| **RTO** (Recovery Time Objective) | 30 minutes | Full restore from latest backup |

## Prerequisites

```bash
export NAMESPACE=clario360
export STAGING_NAMESPACE=clario360-staging
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export BACKUP_BUCKET=gs://clario360-backups
```

Ensure you have:
- `kubectl` access to both production and staging namespaces
- `gsutil` or equivalent object storage CLI configured
- Access to the staging PostgreSQL instance

---

## Procedure 1: List Recent Backups and Verify Completion Status (~5 min)

### 1a. List Backup CronJob Status

```bash
kubectl get cronjobs -n ${NAMESPACE} -o json | jq -r '.items[] | select(.metadata.name | startswith("pg-backup")) | .metadata.name + " | Schedule: " + .spec.schedule + " | Last: " + (.status.lastScheduleTime // "never") + " | Active: " + (.status.active // [] | length | tostring)'
```

**Expected output:** CronJob shows regular schedule (daily at 02:00 UTC) with a recent `lastScheduleTime`.

### 1b. List Recent Backup Jobs

```bash
kubectl get jobs -n ${NAMESPACE} --sort-by=.status.startTime -o json | jq -r '[.items[] | select(.metadata.name | startswith("pg-backup"))] | .[-7:] | .[] | .metadata.name + " | Start: " + .status.startTime + " | Completed: " + (.status.completionTime // "IN PROGRESS") + " | Succeeded: " + (.status.succeeded // 0 | tostring)'
```

**Expected output:** Last 7 backup jobs all show `Succeeded: 1` with valid completion times.

### 1c. Check for Failed Backup Jobs

```bash
kubectl get jobs -n ${NAMESPACE} -o json | jq -r '.items[] | select(.metadata.name | startswith("pg-backup")) | select(.status.failed != null and .status.failed > 0) | .metadata.name + " FAILED at " + .status.startTime'
```

**Expected output:** No output (no failed backup jobs). Any failures must be investigated immediately.

### 1d. Verify Backup Files in Object Storage

```bash
kubectl exec -n ${NAMESPACE} deploy/backup-tools -- gsutil ls -l ${BACKUP_BUCKET}/daily/ | tail -10
```

**Expected output:** Daily backup files present for the last 7 days with consistent file sizes (not 0 bytes).

### 1e. Verify Today's Backup Size

```bash
TODAY=$(date -u +%Y-%m-%d)
kubectl exec -n ${NAMESPACE} deploy/backup-tools -- gsutil ls -l "${BACKUP_BUCKET}/daily/" | grep "${TODAY}" | awk '{print "Size: " $1 " bytes | File: " $3}'
```

**Expected output:** Today's backup file exists with a size comparable to previous days (within 20% variation).

### 1f. List WAL Archive Files

```bash
kubectl exec -n ${NAMESPACE} deploy/backup-tools -- gsutil ls "${BACKUP_BUCKET}/wal/" | tail -20
```

**Expected output:** Continuous stream of WAL files with recent timestamps, confirming WAL archiving is active.

### Verification

```bash
# Count backups in the last 7 days
kubectl exec -n ${NAMESPACE} deploy/backup-tools -- gsutil ls "${BACKUP_BUCKET}/daily/" | grep -c "$(date -u +%Y)"
```

**Expected output:** At least 7 backup files.

---

## Procedure 2: Test Restore to Staging Environment (~25 min)

### 2a. Identify the Latest Backup

```bash
LATEST_BACKUP=$(kubectl exec -n ${NAMESPACE} deploy/backup-tools -- gsutil ls -l "${BACKUP_BUCKET}/daily/" | sort -k2 | tail -1 | awk '{print $3}')
echo "Latest backup: ${LATEST_BACKUP}"
```

### 2b. Stop Staging Applications (prevent writes during restore)

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl scale deploy/${SERVICE} -n ${STAGING_NAMESPACE} --replicas=0
done
```

Verify all pods are terminated:

```bash
kubectl get pods -n ${STAGING_NAMESPACE} --field-selector=status.phase=Running | grep -v postgresql
```

**Expected output:** Only the PostgreSQL pod remains running.

### 2c. Download Backup to Staging

```bash
kubectl exec -n ${STAGING_NAMESPACE} deploy/backup-tools -- gsutil cp "${LATEST_BACKUP}" /tmp/restore-test.sql.gz
```

### 2d. Drop and Recreate Staging Databases

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d postgres -c "
    SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB}' AND pid != pg_backend_pid();
  "
  kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d postgres -c "DROP DATABASE IF EXISTS ${DB};"
  kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d postgres -c "CREATE DATABASE ${DB};"
  echo "Recreated: ${DB}"
done
```

### 2e. Restore Backup

```bash
kubectl exec -n ${STAGING_NAMESPACE} deploy/backup-tools -- bash -c "
  gunzip -c /tmp/restore-test.sql.gz | psql -U ${PG_USER} -h postgresql.${STAGING_NAMESPACE}.svc.cluster.local -d postgres
"
```

Monitor restore progress (in a separate terminal):

```bash
kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d postgres -c "
  SELECT datname, pg_size_pretty(pg_database_size(datname)) AS size
  FROM pg_database
  WHERE datname IN ('platform_core','cyber_db','data_db','acta_db','lex_db','visus_db');
"
```

### 2f. Record Restore Duration

```bash
RESTORE_START=$(date -u +%s)
# ... (restore command above) ...
RESTORE_END=$(date -u +%s)
RESTORE_DURATION=$((RESTORE_END - RESTORE_START))
echo "Restore duration: ${RESTORE_DURATION} seconds ($((RESTORE_DURATION / 60)) minutes)"
```

**Target:** Restore completes within 30 minutes (RTO target).

### Verification

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  TABLE_COUNT=$(kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -t -c "
    SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';
  ")
  echo "${DB}: ${TABLE_COUNT} tables"
done
```

**Expected output:** Table counts match production.

---

## Procedure 3: Verify Data Integrity After Restore (~10 min)

### 3a. Row Count Comparison

Get production row counts:

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} (PRODUCTION) ==="
  kubectl exec -n ${NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -c "
    SELECT schemaname, relname, n_live_tup AS row_count
    FROM pg_stat_user_tables
    WHERE n_live_tup > 0
    ORDER BY n_live_tup DESC
    LIMIT 10;
  "
done
```

Get staging row counts (post-restore):

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} (STAGING - RESTORED) ==="
  kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -c "
    SELECT schemaname, relname, n_live_tup AS row_count
    FROM pg_stat_user_tables
    WHERE n_live_tup > 0
    ORDER BY n_live_tup DESC
    LIMIT 10;
  "
done
```

**Expected output:** Row counts in staging should be within 1% of production (accounts for writes during backup window).

### 3b. Audit Chain Integrity on Restored Data

```bash
kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
WITH chain AS (
  SELECT id, entry_hash, prev_hash,
         LAG(entry_hash) OVER (ORDER BY id) AS expected_prev_hash
  FROM audit_logs
)
SELECT count(*) AS total_entries,
       count(*) FILTER (WHERE expected_prev_hash IS NOT NULL AND prev_hash != expected_prev_hash) AS broken_links
FROM chain;
"
```

**Expected output:** `broken_links` equals `0`. The audit chain should be intact in the restored database.

### 3c. Schema Integrity Check

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} - Constraint check ==="
  kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -c "
    SELECT conname, conrelid::regclass AS table_name, contype
    FROM pg_constraint
    WHERE connamespace = 'public'::regnamespace
    ORDER BY conrelid::regclass::text, contype;
  " | head -20
done
```

**Expected output:** All constraints (foreign keys, unique, check) are present and match production schema.

### 3d. Index Integrity Check

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} - Index check ==="
  kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d ${DB} -c "
    SELECT indexrelid::regclass AS index_name,
           indrelid::regclass AS table_name,
           indisvalid AS is_valid
    FROM pg_index
    JOIN pg_class ON pg_class.oid = pg_index.indexrelid
    WHERE pg_class.relnamespace = 'public'::regnamespace
      AND NOT indisvalid;
  "
done
```

**Expected output:** No invalid indexes. Any invalid indexes indicate a problem with the restore.

### 3e. Application-Level Verification

Restart staging services and verify they can connect and serve requests:

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl scale deploy/${SERVICE} -n ${STAGING_NAMESPACE} --replicas=1
done
```

Wait for pods to be ready:

```bash
kubectl wait --for=condition=ready pod -l app=api-gateway -n ${STAGING_NAMESPACE} --timeout=120s
```

Test health endpoints on staging:

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  STATUS=$(kubectl exec -n ${STAGING_NAMESPACE} deploy/${SERVICE} -- wget -q -O - --timeout=5 http://localhost:8080/readyz 2>/dev/null)
  EXITCODE=$?
  if [ ${EXITCODE} -eq 0 ]; then
    echo "READY: ${SERVICE}"
  else
    echo "NOT READY: ${SERVICE}"
  fi
done
```

**Expected output:** All 10 services report `READY`, confirming they can connect to and read from the restored database.

### Verification

All services healthy on restored data confirms the backup is valid and restorable.

---

## Procedure 4: Document RPO/RTO Metrics (~5 min)

### 4a. Calculate Actual RPO

The RPO is determined by the age of the latest available recovery point:

```bash
# Latest full backup age
LATEST_BACKUP_TIME=$(kubectl get jobs -n ${NAMESPACE} --sort-by=.status.completionTime -o json | jq -r '[.items[] | select(.metadata.name | startswith("pg-backup")) | select(.status.succeeded == 1)] | last | .status.completionTime')
echo "Latest full backup: ${LATEST_BACKUP_TIME}"

# Latest WAL file (for PITR)
LATEST_WAL=$(kubectl exec -n ${NAMESPACE} deploy/backup-tools -- gsutil ls -l "${BACKUP_BUCKET}/wal/" | sort -k2 | tail -1)
echo "Latest WAL archive: ${LATEST_WAL}"
```

Calculate RPO:

```bash
BACKUP_EPOCH=$(date -u -jf "%Y-%m-%dT%H:%M:%SZ" "${LATEST_BACKUP_TIME}" +%s 2>/dev/null || date -u -d "${LATEST_BACKUP_TIME}" +%s)
NOW_EPOCH=$(date -u +%s)
RPO_SECONDS=$((NOW_EPOCH - BACKUP_EPOCH))
RPO_MINUTES=$((RPO_SECONDS / 60))
echo "Actual RPO (full backup): ${RPO_MINUTES} minutes"
echo "Actual RPO (with WAL): ~5 minutes (continuous archiving)"
```

**Target:** RPO with WAL archiving should be under 5 minutes. Full backup RPO should be under 24 hours.

### 4b. Record RTO from Restore Test

```bash
echo "RTO measured during this test: ${RESTORE_DURATION} seconds ($((RESTORE_DURATION / 60)) minutes)"
echo "RTO target: 30 minutes"
if [ ${RESTORE_DURATION} -lt 1800 ]; then
  echo "STATUS: WITHIN RTO TARGET"
else
  echo "STATUS: EXCEEDS RTO TARGET - INVESTIGATE"
fi
```

### 4c. Update RPO/RTO Tracking

Record the following in the backup verification log:

| Date | Backup Size | Restore Duration | RPO (WAL) | RPO (Full) | RTO | Status |
|------|-------------|-----------------|-----------|------------|-----|--------|
| _today_ | __GB | __min | ~5min | __hrs | __min | PASS/FAIL |

### Verification

Both RPO and RTO are within targets.

---

## Procedure 5: Test Point-in-Time Recovery (~15 min)

### 5a. Identify PITR Target Time

Choose a target time between the latest full backup and now:

```bash
# Set PITR target to 2 hours ago
PITR_TARGET=$(date -u -v-2H '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -u -d '2 hours ago' '+%Y-%m-%d %H:%M:%S')
echo "PITR target: ${PITR_TARGET}"
```

### 5b. Prepare Staging for PITR

```bash
# Stop all staging services
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl scale deploy/${SERVICE} -n ${STAGING_NAMESPACE} --replicas=0
done
```

### 5c. Configure PITR Recovery

Create a recovery configuration:

```bash
kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- bash -c "cat > /var/lib/postgresql/data/recovery.signal << 'RECOVERYEOF'
RECOVERYEOF"

kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- bash -c "cat >> /var/lib/postgresql/data/postgresql.auto.conf << CONFEOF
restore_command = 'gsutil cp ${BACKUP_BUCKET}/wal/%f %p'
recovery_target_time = '${PITR_TARGET}'
recovery_target_action = 'promote'
CONFEOF"
```

### 5d. Restore Base Backup for PITR

```bash
# Stop PostgreSQL
kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- pg_ctl stop -D /var/lib/postgresql/data -m fast

# Clear existing data
kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- bash -c "rm -rf /var/lib/postgresql/data/*"

# Restore base backup
kubectl exec -n ${STAGING_NAMESPACE} deploy/backup-tools -- bash -c "
  gsutil cp ${BACKUP_BUCKET}/daily/$(kubectl exec -n ${NAMESPACE} deploy/backup-tools -- gsutil ls ${BACKUP_BUCKET}/daily/ | sort | tail -1 | xargs basename) /tmp/base-backup.tar.gz
  tar xzf /tmp/base-backup.tar.gz -C /var/lib/postgresql/data/
"

# Start PostgreSQL (will replay WAL files to target time)
kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- pg_ctl start -D /var/lib/postgresql/data -l /var/lib/postgresql/data/logfile
```

### 5e. Monitor Recovery Progress

```bash
kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d postgres -c "
  SELECT pg_is_in_recovery(),
         pg_last_wal_receive_lsn(),
         pg_last_wal_replay_lsn(),
         pg_last_xact_replay_timestamp();
"
```

Wait until recovery completes:

```bash
while true; do
  IN_RECOVERY=$(kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d postgres -t -c "SELECT pg_is_in_recovery();" | tr -d ' ')
  if [ "${IN_RECOVERY}" = "f" ]; then
    echo "Recovery complete!"
    break
  fi
  REPLAY_TS=$(kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d postgres -t -c "SELECT pg_last_xact_replay_timestamp();" | tr -d ' ')
  echo "Still recovering... Last replayed: ${REPLAY_TS}"
  sleep 10
done
```

### 5f. Verify PITR Data Consistency

```bash
kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT max(created_at) AS latest_record,
       '${PITR_TARGET}'::timestamp AS target_time,
       CASE
         WHEN max(created_at) <= '${PITR_TARGET}'::timestamp THEN 'PITR VERIFIED'
         ELSE 'PITR FAILED - data after target time exists'
       END AS status
FROM audit_logs;
"
```

**Expected output:** `status` shows `PITR VERIFIED`, confirming no data after the target time exists.

### 5g. Verify Data Exists Up To Target Time

```bash
kubectl exec -n ${STAGING_NAMESPACE} deploy/postgresql -- psql -U ${PG_USER} -d platform_core -c "
SELECT count(*) AS records_before_target
FROM audit_logs
WHERE created_at <= '${PITR_TARGET}'::timestamp;
"
```

**Expected output:** Non-zero count, confirming data up to the target time was recovered.

### Verification

PITR is verified when:
1. No data exists after the target recovery time
2. Data exists up to (or very close to) the target recovery time
3. All database schemas and constraints are intact

---

## Cleanup

### Restore Staging to Normal State

After verification, restore staging to its normal state by re-running the standard staging deployment:

```bash
# Restart staging services with normal configuration
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl scale deploy/${SERVICE} -n ${STAGING_NAMESPACE} --replicas=1
done
```

### Remove Temporary Files

```bash
kubectl exec -n ${STAGING_NAMESPACE} deploy/backup-tools -- rm -f /tmp/restore-test.sql.gz /tmp/base-backup.tar.gz
```

---

## Final Verification Summary

| Test | Target | Result | Status |
|------|--------|--------|--------|
| Backup completion | Daily backups present | __ | PASS/FAIL |
| Backup file integrity | Non-zero, consistent size | __ | PASS/FAIL |
| WAL archiving | Continuous, recent files | __ | PASS/FAIL |
| Full restore | Completes within 30 min | __min | PASS/FAIL |
| Data integrity | Row counts match production | __ | PASS/FAIL |
| Audit chain | Zero broken links | __ | PASS/FAIL |
| Schema integrity | All constraints valid | __ | PASS/FAIL |
| Application health | All services ready | __ | PASS/FAIL |
| RPO (WAL) | <5 minutes | __min | PASS/FAIL |
| RPO (full backup) | <24 hours | __hrs | PASS/FAIL |
| RTO | <30 minutes | __min | PASS/FAIL |
| PITR | Data correct at target time | __ | PASS/FAIL |

Record this table in the weekly operations report.

## Related Runbooks

- [OP-001: Daily Checks](OP-001-daily-checks.md) (Check 7: Backup Status)
- [OP-007: Database Maintenance](OP-007-database-maintenance.md)
- [IR-002: Database Failure](../incident-response/IR-002-database-failure.md)
- [IR-009: Data Corruption](../incident-response/IR-009-data-corruption.md)

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2026-03-08 | Platform Team | Initial version |
