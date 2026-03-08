# TS-002: Data Pipeline Failure Investigation

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | TS-002                                     |
| **Title**        | Data Pipeline Failure Investigation        |
| **Severity**     | P2 - High                                  |
| **Services**     | data-service, workflow-engine              |
| **Last Updated** | 2026-03-08                                 |
| **Author**       | Platform Engineering                       |
| **Review Cycle** | Quarterly                                  |

---

## Summary

This runbook covers the investigation and resolution of failed data pipelines in the Clario 360 platform. Data pipelines are orchestrated as Kubernetes jobs by the data-service and workflow-engine. Failures can originate from source data unavailability, target database write permission issues, schema mismatches, resource limits, or transient infrastructure errors. Follow the diagnosis steps to identify the specific failure point, then apply the corresponding resolution.

---

## Symptoms

- Pipeline jobs show `Failed` or `BackoffLimitExceeded` status in Kubernetes.
- Data freshness alerts fire: downstream tables are not being updated on schedule.
- Users report stale or missing data in dashboards and reports.
- Workflow engine shows stuck or failed pipeline step executions.
- Error events appear in the `clario360.pipeline.events` Kafka topic.
- Data-service logs contain errors about connection failures, schema validation, or timeout exceeded.

---

## Diagnosis Steps

### Step 1: Check Pipeline Job Status

```bash
# List all pipeline jobs in the clario360 namespace
kubectl -n clario360 get jobs --sort-by='.status.startTime' | tail -30

# List only failed jobs
kubectl -n clario360 get jobs --field-selector=status.successful=0

# Get detailed information about a specific failed job
kubectl -n clario360 describe job/<pipeline-job-name>

# Check if jobs are hitting backoff limits
kubectl -n clario360 get jobs -o jsonpath='{range .items[?(@.status.failed)]}{.metadata.name}{"\t"}{.status.failed}{"\t"}{.status.conditions[*].reason}{"\n"}{end}'
```

```bash
# List pods created by failed jobs to inspect their status
kubectl -n clario360 get pods --selector=job-name=<pipeline-job-name> -o wide

# Check for pods in CrashLoopBackOff or Error state
kubectl -n clario360 get pods --field-selector=status.phase=Failed -l component=pipeline
```

### Step 2: Check Pipeline Pod Logs

```bash
# Get logs from the most recent failed pipeline pod
kubectl -n clario360 logs job/<pipeline-job-name> --tail=200

# If the pod has restarted multiple times, check previous container logs
kubectl -n clario360 logs job/<pipeline-job-name> --previous --tail=200

# Get logs for all pods of a specific job
kubectl -n clario360 logs -l job-name=<pipeline-job-name> --tail=100

# Search for specific error patterns in pipeline pod logs
kubectl -n clario360 logs job/<pipeline-job-name> --tail=500 | grep -i -E 'error|fatal|panic|timeout|refused'
```

```bash
# Check data-service logs for pipeline orchestration errors
kubectl -n clario360 logs deploy/data-service --tail=300 | grep -i -E 'pipeline|error|failed'

# Check workflow-engine logs for step execution failures
kubectl -n clario360 logs deploy/workflow-engine --tail=300 | grep -i -E 'pipeline|step.*fail|execution.*error'
```

### Step 3: Verify Source Data Availability

```bash
# Check if source databases are reachable from inside the cluster
kubectl -n clario360 run db-check --rm -it --image=postgres:15 -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d data_db -c "SELECT 1;"

# Check if source tables exist and have data
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d data_db
```

```sql
-- List tables in the public schema
SELECT table_name, pg_size_pretty(pg_total_relation_size(quote_ident(table_name))) AS size
FROM information_schema.tables
WHERE table_schema = 'public'
ORDER BY pg_total_relation_size(quote_ident(table_name)) DESC;

-- Check row counts on source tables
SELECT schemaname, relname, n_live_tup
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC;

-- Verify the most recent data timestamp in a source table
SELECT max(created_at), max(updated_at)
FROM pipeline_runs;
```

```bash
# If the source is an external API, test connectivity
kubectl -n clario360 run curl-test --rm -it --image=curlimages/curl -- \
  curl -v --connect-timeout 10 https://<source-api-endpoint>/health
```

### Step 4: Check Target Database Write Permissions

```bash
# Connect to the target database
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d data_db
```

```sql
-- Check if the pipeline user has write permissions
SELECT grantee, privilege_type, table_name
FROM information_schema.table_privileges
WHERE grantee = 'clario360' AND table_schema = 'public'
ORDER BY table_name;

-- Test write access explicitly
BEGIN;
INSERT INTO pipeline_runs (id, status, started_at) VALUES (gen_random_uuid(), 'test', now());
ROLLBACK;

-- Check for table locks that might block writes
SELECT
    l.locktype,
    l.relation::regclass,
    l.mode,
    l.granted,
    a.query,
    a.pid
FROM pg_locks l
JOIN pg_stat_activity a ON l.pid = a.pid
WHERE l.relation IS NOT NULL
    AND NOT l.granted
ORDER BY a.query_start;

-- Check disk space on the database
SELECT
    pg_database.datname,
    pg_size_pretty(pg_database_size(pg_database.datname)) AS size
FROM pg_database
ORDER BY pg_database_size(pg_database.datname) DESC;
```

### Step 5: Check for Schema Mismatches

```bash
# Connect to the target database
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d data_db
```

```sql
-- Check current table schema
SELECT
    column_name,
    data_type,
    character_maximum_length,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = '<target_table_name>'
ORDER BY ordinal_position;

-- Check for recent schema changes (via migration history if available)
SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 10;

-- Check for NOT NULL constraint violations that might cause failures
SELECT
    tc.constraint_name,
    tc.table_name,
    kcu.column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
WHERE tc.constraint_type = 'NOT NULL'
    OR tc.constraint_type = 'CHECK'
ORDER BY tc.table_name;

-- Check for foreign key constraints that might fail on insert
SELECT
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY';
```

### Step 6: Check Resource Limits and Events

```bash
# Check if pipeline pods are being OOMKilled or throttled
kubectl -n clario360 get events --sort-by='.lastTimestamp' | grep -i -E 'oom|kill|evict|fail|backoff' | tail -20

# Check resource usage of running pipeline pods
kubectl -n clario360 top pods -l component=pipeline

# Check resource limits defined on pipeline jobs
kubectl -n clario360 get jobs -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.template.spec.containers[0].resources}{"\n"}{end}' | tail -20
```

### Step 7: Check Pipeline Kafka Events

```bash
# Check for pipeline failure events in Kafka
kubectl -n kafka exec -it kafka-0 -- kafka-console-consumer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.pipeline.events \
  --from-beginning \
  --max-messages 20 \
  --property print.timestamp=true \
  --property print.key=true

# Check the pipeline dead letter queue for failed events
kubectl -n kafka exec -it kafka-0 -- kafka-console-consumer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.pipeline.events.dlq \
  --from-beginning \
  --max-messages 20
```

---

## Resolution Steps

### Resolution: Retry Failed Jobs

```bash
# Delete the failed job and let the controller recreate it (if managed by a CronJob)
kubectl -n clario360 delete job/<failed-pipeline-job-name>

# If the pipeline is triggered manually, recreate the job
kubectl -n clario360 get job/<failed-pipeline-job-name> -o yaml | \
  grep -v 'status:' | \
  grep -v 'creationTimestamp:' | \
  grep -v 'uid:' | \
  grep -v 'resourceVersion:' | \
  kubectl apply -f -

# Alternatively, trigger a pipeline run via the data-service API
kubectl -n clario360 port-forward svc/data-service 8080:8080
curl -X POST http://localhost:8080/api/v1/pipelines/<pipeline-id>/trigger \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>"

# Monitor the new job
kubectl -n clario360 get jobs -w
kubectl -n clario360 logs -f job/<new-pipeline-job-name>
```

### Resolution: Fix Data Format / Schema Mismatch

1. **If columns were added to the source but not the target:**

```bash
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d data_db
```

```sql
-- Add the missing column to the target table
ALTER TABLE <target_table> ADD COLUMN <column_name> <data_type>;

-- Or if a column type changed
ALTER TABLE <target_table> ALTER COLUMN <column_name> TYPE <new_data_type> USING <column_name>::<new_data_type>;
```

2. **If the pipeline expects a different data format, update the pipeline configuration:**

```bash
# Edit the pipeline ConfigMap
kubectl -n clario360 edit configmap/<pipeline-config-name>

# Or patch it directly
kubectl -n clario360 patch configmap/<pipeline-config-name> --type='merge' -p='{"data":{"FIELD_MAPPING":"<updated-mapping>"}}'

# Restart the data-service to pick up config changes
kubectl -n clario360 rollout restart deploy/data-service
kubectl -n clario360 rollout status deploy/data-service
```

3. **If data contains NULL values violating NOT NULL constraints:**

```sql
-- Option A: Allow NULL on the column
ALTER TABLE <target_table> ALTER COLUMN <column_name> DROP NOT NULL;

-- Option B: Set a default value
ALTER TABLE <target_table> ALTER COLUMN <column_name> SET DEFAULT '<default_value>';
```

### Resolution: Increase Timeout / Resource Limits

```bash
# Increase job deadline (activeDeadlineSeconds)
kubectl -n clario360 patch job/<pipeline-job-name> --type='json' -p='[
  {"op": "replace", "path": "/spec/activeDeadlineSeconds", "value": 3600}
]'

# For managed CronJobs, update the template
kubectl -n clario360 patch cronjob/<pipeline-cronjob-name> --type='json' -p='[
  {"op": "replace", "path": "/spec/jobTemplate/spec/activeDeadlineSeconds", "value": 3600},
  {"op": "replace", "path": "/spec/jobTemplate/spec/template/spec/containers/0/resources/requests/memory", "value": "1Gi"},
  {"op": "replace", "path": "/spec/jobTemplate/spec/template/spec/containers/0/resources/limits/memory", "value": "2Gi"},
  {"op": "replace", "path": "/spec/jobTemplate/spec/template/spec/containers/0/resources/requests/cpu", "value": "500m"},
  {"op": "replace", "path": "/spec/jobTemplate/spec/template/spec/containers/0/resources/limits/cpu", "value": "1000m"}
]'

# Increase backoff limit if transient errors are expected
kubectl -n clario360 patch cronjob/<pipeline-cronjob-name> --type='json' -p='[
  {"op": "replace", "path": "/spec/jobTemplate/spec/backoffLimit", "value": 5}
]'
```

### Resolution: Fix Database Connectivity

```bash
# If PostgreSQL is unreachable, check the StatefulSet
kubectl -n clario360 get statefulset postgresql
kubectl -n clario360 describe statefulset postgresql
kubectl -n clario360 logs statefulset/postgresql --tail=100

# Restart PostgreSQL if needed
kubectl -n clario360 rollout restart statefulset/postgresql
kubectl -n clario360 rollout status statefulset/postgresql

# Verify connectivity after restart
kubectl -n clario360 run db-check --rm -it --image=postgres:15 -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d data_db -c "SELECT 1;"
```

### Resolution: Fix Write Permissions

```bash
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d data_db
```

```sql
-- Grant necessary permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO clario360;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO clario360;
GRANT USAGE ON SCHEMA public TO clario360;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO clario360;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO clario360;
```

---

## Verification

After applying the resolution, verify the fix:

```bash
# 1. Check that the retried/new job completes successfully
kubectl -n clario360 get jobs | grep <pipeline-job-name>
kubectl -n clario360 logs job/<pipeline-job-name> --tail=50

# 2. Verify data was written to the target table
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d data_db -c \
  "SELECT count(*), max(created_at) FROM <target_table>;"

# 3. Check data-service health
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/readyz

# 4. Verify no more failed jobs remain
kubectl -n clario360 get jobs --field-selector=status.successful=0

# 5. Check pipeline events topic for success events
kubectl -n kafka exec -it kafka-0 -- kafka-console-consumer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.pipeline.events \
  --from-beginning \
  --max-messages 5 \
  --property print.timestamp=true

# 6. Verify CronJob schedule is active for recurring pipelines
kubectl -n clario360 get cronjobs
```

---

## Related Links

- [TS-001: API Latency Investigation](./TS-001-slow-api-responses.md)
- [TS-003: Kafka Event Loss Investigation](./TS-003-missing-events.md)
- [TS-004: Authentication Issue Debugging](./TS-004-auth-failures.md)
- [TS-005: WebSocket Connectivity Issues](./TS-005-websocket-disconnects.md)
- Grafana Pipeline Dashboard: `/d/pipeline-monitoring/data-pipelines`
- Kubernetes Jobs Documentation: https://kubernetes.io/docs/concepts/workloads/controllers/job/
