# TS-009: Audit Hash Chain Integrity Failure

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | TS-009                                                                |
| **Title**          | Audit Hash Chain Integrity Failure                                    |
| **Severity**       | P1 -- Critical                                                        |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | audit-service                                                         |
| **Namespace**      | clario360                                                             |
| **Escalation**     | Platform Engineering Lead -> Compliance Officer -> VP Engineering -> CTO |
| **SLA**            | Acknowledge within 5 minutes, resolve within 1 hour                   |

---

## Summary

This runbook addresses audit hash chain integrity failures in the Clario360 audit service. The audit system uses a hash chain (blockchain-like) where each audit log entry contains a hash of the previous entry (`prev_hash`) and its own hash (`entry_hash`), computed from the entry data plus the previous hash. If any entry is modified, deleted, or inserted out of order, the chain breaks and integrity verification fails. This is a critical compliance issue -- a broken hash chain means the audit trail can no longer be cryptographically proven to be tamper-free.

---

## Symptoms

- Audit service health check or integrity verification endpoint returns a failure.
- Compliance dashboard shows "Audit Chain Integrity: FAILED" status.
- Prometheus alert `audit_chain_integrity_failure` firing.
- Audit service logs contain messages like `hash chain broken`, `entry_hash mismatch`, or `prev_hash does not match`.
- Scheduled integrity verification job reports failures.
- External auditors or compliance scans flag audit trail inconsistencies.
- `entry_hash` in an audit log entry does not match the recomputed hash of its data.
- `prev_hash` in an audit log entry does not match the `entry_hash` of the preceding entry.

---

## Impact Assessment

| Scenario                           | Impact                                                          |
|------------------------------------|-----------------------------------------------------------------|
| Single break point                 | Chain broken from that entry forward; all subsequent entries unverifiable |
| Multiple break points              | Indicates systematic issue (concurrent writes or data corruption) |
| Break at recent entries            | Recent audit trail unverifiable; may affect active compliance audits |
| Break at historical entries        | Historical audit trail compromised; regulatory reporting affected |
| Manual DB modification detected    | Potential security incident; escalate to security team immediately |

**Compliance Note**: A broken audit hash chain may constitute a compliance violation under SOC 2, HIPAA, GDPR, or other regulatory frameworks. Notify the Compliance Officer immediately upon detection.

---

## Prerequisites

- `kubectl` configured with cluster access and `clario360` namespace permissions.
- `psql` client installed locally or ability to exec into a pod with `psql`.
- Access to the audit-service API endpoints.
- A valid admin-level JWT token for calling the audit-service verification API.
- Understanding of the hash chain algorithm used (SHA-256, fields included in hash computation).

---

## Diagnosis Steps

### Step 1: Check Audit Service Health

```bash
kubectl port-forward -n clario360 svc/audit-service 8080:8080 &
PF_PID=$!
sleep 2

curl -s http://localhost:8080/healthz | jq .
curl -s http://localhost:8080/readyz | jq .

kill $PF_PID
```

### Step 2: Run Hash Chain Verification via API

```bash
kubectl port-forward -n clario360 svc/audit-service 8080:8080 &
PF_PID=$!
sleep 2

# Full chain verification
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/audit/verify | jq .

# Verify a specific date range
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/audit/verify?from=2026-03-01T00:00:00Z&to=2026-03-08T23:59:59Z" | jq .

# Verify a specific tenant's chain
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/audit/verify?tenant_id=<TENANT_ID>" | jq .

kill $PF_PID
```

The response should indicate the first broken entry if the chain is invalid.

### Step 3: Check Audit Service Logs for Chain Errors

```bash
kubectl logs -n clario360 -l app=audit-service --tail=500 --timestamps | \
  grep -i -E "hash.*chain|integrity|entry_hash|prev_hash|mismatch|broken|tamper|verification"
```

### Step 4: Identify the Break Point in the Database

Connect to the platform_core database:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core
```

Find where the chain breaks by comparing each entry's `prev_hash` with the preceding entry's `entry_hash`:

```sql
-- Find break points in the hash chain
WITH ordered_entries AS (
    SELECT
        id,
        sequence_number,
        entry_hash,
        prev_hash,
        created_at,
        tenant_id,
        action,
        LAG(entry_hash) OVER (ORDER BY sequence_number ASC) AS expected_prev_hash
    FROM audit_logs
    ORDER BY sequence_number ASC
)
SELECT
    id,
    sequence_number,
    created_at,
    tenant_id,
    action,
    prev_hash,
    expected_prev_hash,
    entry_hash,
    CASE
        WHEN prev_hash IS NULL AND sequence_number = 1 THEN 'GENESIS (OK)'
        WHEN prev_hash = expected_prev_hash THEN 'VALID'
        ELSE 'BROKEN'
    END AS chain_status
FROM ordered_entries
WHERE prev_hash IS DISTINCT FROM expected_prev_hash
  AND NOT (prev_hash IS NULL AND sequence_number = 1)
ORDER BY sequence_number ASC;
```

### Step 5: Examine the Broken Entry and Its Neighbors

```sql
-- Replace <BROKEN_SEQ> with the sequence_number of the broken entry
SELECT
    id,
    sequence_number,
    entry_hash,
    prev_hash,
    created_at,
    updated_at,
    tenant_id,
    actor_id,
    action,
    resource_type,
    resource_id,
    severity,
    service
FROM audit_logs
WHERE sequence_number BETWEEN (<BROKEN_SEQ> - 2) AND (<BROKEN_SEQ> + 2)
ORDER BY sequence_number ASC;
```

### Step 6: Verify the Entry Hash Is Correctly Computed

```sql
-- Check if the entry_hash matches a recomputation from the entry's fields
-- The exact hash computation depends on the implementation
-- Typically: SHA256(prev_hash + tenant_id + actor_id + action + resource_type + resource_id + timestamp + ...)
SELECT
    id,
    sequence_number,
    entry_hash,
    prev_hash,
    encode(
        sha256(
            (coalesce(prev_hash, '') ||
             tenant_id::text ||
             actor_id::text ||
             action ||
             resource_type ||
             resource_id::text ||
             extract(epoch from created_at)::text
            )::bytea
        ),
        'hex'
    ) AS recomputed_hash,
    CASE
        WHEN entry_hash = encode(
            sha256(
                (coalesce(prev_hash, '') ||
                 tenant_id::text ||
                 actor_id::text ||
                 action ||
                 resource_type ||
                 resource_id::text ||
                 extract(epoch from created_at)::text
                )::bytea
            ),
            'hex'
        ) THEN 'MATCH'
        ELSE 'MISMATCH'
    END AS hash_check
FROM audit_logs
WHERE sequence_number = <BROKEN_SEQ>;
```

**Note**: Adjust the hash computation fields to match your actual implementation. Check the audit-service source code for the exact fields and order.

### Step 7: Check for Manual Database Modifications

```sql
-- Check if updated_at differs from created_at (indicates modification)
SELECT
    id,
    sequence_number,
    created_at,
    updated_at,
    CASE
        WHEN updated_at > created_at + interval '1 second' THEN 'MODIFIED'
        ELSE 'ORIGINAL'
    END AS modification_status
FROM audit_logs
WHERE sequence_number BETWEEN (<BROKEN_SEQ> - 5) AND (<BROKEN_SEQ> + 5)
ORDER BY sequence_number ASC;

-- Check PostgreSQL logs for direct SQL modifications to audit_logs
-- (Requires access to PostgreSQL audit logging if enabled)
```

```bash
kubectl logs -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') --tail=1000 | \
  grep -i -E "UPDATE.*audit_logs|DELETE.*audit_logs|INSERT.*audit_logs" | tail -20
```

### Step 8: Check for Sequence Number Gaps (Deleted Entries)

```sql
-- Find gaps in the sequence_number series
WITH seq_range AS (
    SELECT
        sequence_number,
        LEAD(sequence_number) OVER (ORDER BY sequence_number) AS next_seq
    FROM audit_logs
)
SELECT
    sequence_number AS gap_after,
    next_seq AS gap_before,
    next_seq - sequence_number - 1 AS missing_entries
FROM seq_range
WHERE next_seq - sequence_number > 1
ORDER BY sequence_number ASC;
```

### Step 9: Check for Concurrent Write Race Conditions

```sql
-- Look for entries with identical or very close timestamps
-- that may have been written by different service instances
SELECT
    id,
    sequence_number,
    entry_hash,
    prev_hash,
    created_at,
    service,
    actor_id,
    action
FROM audit_logs
WHERE created_at BETWEEN
    (SELECT created_at - interval '1 second' FROM audit_logs WHERE sequence_number = <BROKEN_SEQ>)
    AND
    (SELECT created_at + interval '1 second' FROM audit_logs WHERE sequence_number = <BROKEN_SEQ>)
ORDER BY sequence_number ASC;

-- Check how many audit-service replicas were running at the time
```

```bash
kubectl get events -n clario360 --field-selector involvedObject.name=audit-service --sort-by='.lastTimestamp' | tail -20
```

### Step 10: Count Total Break Points

```sql
WITH ordered_entries AS (
    SELECT
        sequence_number,
        entry_hash,
        prev_hash,
        LAG(entry_hash) OVER (ORDER BY sequence_number ASC) AS expected_prev_hash
    FROM audit_logs
)
SELECT
    count(*) AS total_break_points
FROM ordered_entries
WHERE prev_hash IS DISTINCT FROM expected_prev_hash
  AND NOT (prev_hash IS NULL AND sequence_number = (SELECT MIN(sequence_number) FROM audit_logs));
```

---

## Resolution Steps

### Resolution A: Recalculate Hashes from the Break Point Forward

**Warning**: This operation modifies audit entries. Document the reason, get compliance approval, and create a backup first.

**1. Create a backup of the affected entries:**

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
CREATE TABLE audit_logs_backup_$(date +%Y%m%d%H%M%S) AS
SELECT * FROM audit_logs
WHERE sequence_number >= <BROKEN_SEQ>
ORDER BY sequence_number ASC;
"
```

**2. Use the audit-service repair API (preferred):**

```bash
kubectl port-forward -n clario360 svc/audit-service 8080:8080 &
PF_PID=$!
sleep 2

# Trigger hash chain recalculation from the break point
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"from_sequence\": <BROKEN_SEQ>, \"reason\": \"Hash chain repair - TS-009 incident\"}" \
  http://localhost:8080/api/v1/audit/repair | jq .

kill $PF_PID
```

**3. If no repair API exists, recalculate directly in SQL:**

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core <<'SQL'
-- Recalculate hash chain from the break point
-- IMPORTANT: Adjust the hash computation to match your implementation
DO $$
DECLARE
    rec RECORD;
    current_prev_hash TEXT;
    new_hash TEXT;
BEGIN
    -- Get the entry_hash of the entry BEFORE the break point
    SELECT entry_hash INTO current_prev_hash
    FROM audit_logs
    WHERE sequence_number = (<BROKEN_SEQ> - 1);

    -- If the break point is the first entry, prev_hash is empty
    IF current_prev_hash IS NULL THEN
        current_prev_hash := '';
    END IF;

    -- Iterate through all entries from the break point forward
    FOR rec IN
        SELECT id, sequence_number, tenant_id, actor_id, action,
               resource_type, resource_id, created_at
        FROM audit_logs
        WHERE sequence_number >= <BROKEN_SEQ>
        ORDER BY sequence_number ASC
    LOOP
        -- Compute new hash (adjust fields to match your implementation)
        new_hash := encode(
            sha256(
                (current_prev_hash ||
                 rec.tenant_id::text ||
                 rec.actor_id::text ||
                 rec.action ||
                 rec.resource_type ||
                 rec.resource_id::text ||
                 extract(epoch from rec.created_at)::text
                )::bytea
            ),
            'hex'
        );

        -- Update the entry
        UPDATE audit_logs
        SET prev_hash = current_prev_hash,
            entry_hash = new_hash
        WHERE id = rec.id;

        -- Set prev_hash for the next iteration
        current_prev_hash := new_hash;
    END LOOP;

    RAISE NOTICE 'Hash chain recalculated from sequence %', <BROKEN_SEQ>;
END $$;
SQL
```

### Resolution B: Fix Concurrent Write Race Conditions with Advisory Locks

If the root cause is concurrent writes from multiple audit-service replicas:

**1. Scale audit-service to a single replica temporarily:**

```bash
kubectl scale deployment/audit-service -n clario360 --replicas=1
kubectl rollout status deployment/audit-service -n clario360 --timeout=120s
```

**2. Add advisory locking to prevent concurrent chain writes:**

This requires a code change. As an immediate database-level mitigation, add a serialization constraint:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core <<'SQL'
-- Create a function that acquires an advisory lock before inserting audit entries
CREATE OR REPLACE FUNCTION audit_chain_lock() RETURNS trigger AS $$
BEGIN
    -- Acquire an exclusive advisory lock (key = hash of 'audit_chain')
    PERFORM pg_advisory_xact_lock(hashtext('audit_chain'));

    -- Set prev_hash from the most recent entry
    SELECT entry_hash INTO NEW.prev_hash
    FROM audit_logs
    ORDER BY sequence_number DESC
    LIMIT 1;

    -- If this is the first entry, prev_hash remains NULL
    IF NEW.prev_hash IS NULL THEN
        NEW.prev_hash := '';
    END IF;

    -- Compute entry_hash (adjust fields to match your implementation)
    NEW.entry_hash := encode(
        sha256(
            (coalesce(NEW.prev_hash, '') ||
             NEW.tenant_id::text ||
             NEW.actor_id::text ||
             NEW.action ||
             NEW.resource_type ||
             NEW.resource_id::text ||
             extract(epoch from NEW.created_at)::text
            )::bytea
        ),
        'hex'
    );

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply the trigger (drops existing if present)
DROP TRIGGER IF EXISTS audit_chain_lock_trigger ON audit_logs;
CREATE TRIGGER audit_chain_lock_trigger
    BEFORE INSERT ON audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION audit_chain_lock();
SQL
```

**3. After the code fix is deployed, scale back up:**

```bash
kubectl scale deployment/audit-service -n clario360 --replicas=3
kubectl rollout status deployment/audit-service -n clario360 --timeout=120s
```

### Resolution C: Handle Deleted Entries (Sequence Gaps)

If entries were deleted (sequence gaps found in Step 8):

**1. Recalculate the chain across the gap:**

The chain must be recalculated from the entry after each gap. Use the same procedure as Resolution A, starting from the first entry after each gap.

**2. Add a deletion protection rule:**

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
-- Create a rule that prevents deleting audit entries
CREATE OR REPLACE RULE prevent_audit_delete AS
ON DELETE TO audit_logs
DO INSTEAD NOTHING;

-- Or use a trigger that raises an error
CREATE OR REPLACE FUNCTION prevent_audit_modification() RETURNS trigger AS \$\$
BEGIN
    RAISE EXCEPTION 'Audit log entries cannot be modified or deleted';
END;
\$\$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS prevent_audit_update_trigger ON audit_logs;
CREATE TRIGGER prevent_audit_update_trigger
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();
"
```

### Resolution D: Investigate Potential Tampering

If manual database modification is suspected:

```bash
# 1. Preserve evidence
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
CREATE TABLE audit_chain_evidence_$(date +%Y%m%d%H%M%S) AS
SELECT *, now() AS evidence_captured_at
FROM audit_logs
ORDER BY sequence_number ASC;
"

# 2. Check who has direct database access
kubectl get rolebinding -n clario360 -o wide
kubectl get clusterrolebinding -o wide | grep clario360

# 3. Check PostgreSQL user activity
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
SELECT rolname, rolcanlogin, rolsuper, rolcreatedb
FROM pg_roles
WHERE rolcanlogin = true
ORDER BY rolname;
"

# 4. Escalate to security team
echo "SECURITY ESCALATION REQUIRED: Potential audit log tampering detected. See TS-009 incident."
```

---

## Verification

After applying a resolution, verify chain integrity:

```bash
# 1. Run full chain verification via API
kubectl port-forward -n clario360 svc/audit-service 8080:8080 &
PF_PID=$!
sleep 2

curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/audit/verify | jq .

kill $PF_PID

# 2. Verify chain directly in database
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
WITH ordered_entries AS (
    SELECT
        sequence_number,
        entry_hash,
        prev_hash,
        LAG(entry_hash) OVER (ORDER BY sequence_number ASC) AS expected_prev_hash
    FROM audit_logs
)
SELECT
    count(*) FILTER (WHERE prev_hash IS DISTINCT FROM expected_prev_hash
        AND NOT (prev_hash IS NULL AND sequence_number = (SELECT MIN(sequence_number) FROM audit_logs))) AS break_points,
    count(*) AS total_entries
FROM ordered_entries;
"

# 3. Verify no sequence gaps
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d platform_core -c "
WITH seq_range AS (
    SELECT sequence_number, LEAD(sequence_number) OVER (ORDER BY sequence_number) AS next_seq
    FROM audit_logs
)
SELECT count(*) AS gaps FROM seq_range WHERE next_seq - sequence_number > 1;
"

# 4. Test that new entries maintain the chain
kubectl port-forward -n clario360 svc/audit-service 8080:8080 &
PF_PID=$!
sleep 2

# Create a test audit entry
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "chain.verify", "resource_type": "system", "resource_id": "test", "severity": "info"}' \
  http://localhost:8080/api/v1/audit | jq .

# Verify the chain is still valid after the new entry
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/audit/verify | jq .

kill $PF_PID

# 5. Check audit service logs for any remaining errors
kubectl logs -n clario360 -l app=audit-service --since=5m --timestamps | \
  grep -i -c -E "hash.*chain|integrity|mismatch|broken"

# 6. Verify audit service health
kubectl port-forward -n clario360 svc/audit-service 8080:8080 &
PF_PID=$!
sleep 2
curl -sf http://localhost:8080/healthz && echo "HEALTHY" || echo "UNHEALTHY"
curl -sf http://localhost:8080/readyz && echo "READY" || echo "NOT READY"
kill $PF_PID
```

---

## Post-Incident Checklist

- [ ] Confirm hash chain verification passes end-to-end (zero break points).
- [ ] Confirm no sequence number gaps exist.
- [ ] Confirm new audit entries are correctly chained.
- [ ] Confirm audit-service `/healthz` and `/readyz` return HTTP 200.
- [ ] Verify Prometheus `audit_chain_integrity_failure` alert has cleared.
- [ ] Notify the Compliance Officer of the incident and resolution.
- [ ] Document the root cause, affected sequence range, and resolution applied.
- [ ] If hash recalculation was performed, document the backup table name and location.
- [ ] If tampering is suspected, escalate to the security team and preserve evidence.
- [ ] If concurrent write race condition was the cause, verify advisory locks are in place.
- [ ] If audit-service was scaled to 1 replica, verify it has been scaled back up.
- [ ] Add the deletion protection trigger if not already present.
- [ ] Review and update the scheduled integrity verification job frequency.
- [ ] Create a ticket to add hash chain verification to the CI/CD pipeline.

---

## Related Links

| Resource                         | Link                                                                     |
|----------------------------------|--------------------------------------------------------------------------|
| Audit Service Source Code        | https://github.com/clario360/platform/tree/main/internal/audit           |
| SOC 2 Audit Trail Requirements  | https://wiki.clario360.internal/compliance/soc2-audit-trail              |
| Grafana Dashboards               | https://grafana.clario360.internal/dashboards                            |
| Alertmanager                     | https://alertmanager.clario360.internal                                  |
| IR-001 Service Outage            | [../incident-response/IR-001-service-outage.md](../incident-response/IR-001-service-outage.md) |
| TS-008 Connection Pool           | [TS-008-connection-pool-exhaustion.md](./TS-008-connection-pool-exhaustion.md) |
