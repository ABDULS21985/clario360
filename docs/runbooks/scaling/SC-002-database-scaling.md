# SC-002: PostgreSQL Database Scaling

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | SC-002                                     |
| **Title**        | PostgreSQL Database Scaling                |
| **Category**     | Scaling                                    |
| **Severity**     | High                                       |
| **Author**       | Platform Engineering                       |
| **Created**      | 2026-03-08                                 |
| **Last Updated** | 2026-03-08                                 |
| **Review Cycle** | Quarterly                                  |
| **Platform**     | GCP (GKE)                                  |
| **Namespace**    | clario360                                  |

---

## Summary

This runbook covers scaling PostgreSQL databases for the Clario 360 platform. It addresses adding read replicas for horizontal read scaling, configuring PgBouncer connection pooling, tuning `max_connections`, vertical scaling of the PostgreSQL pod, and monitoring replication lag. The platform uses the following databases:

| Database        | Primary Use                        | Service(s)                    |
|-----------------|------------------------------------|-------------------------------|
| platform_core   | IAM, audit, workflow, notifications| iam-service, audit-service, workflow-engine, notification-service |
| cyber_db        | Cybersecurity findings and scans   | cyber-service                 |
| data_db         | Data governance and catalogs       | data-service                  |
| acta_db         | Document and records management    | acta-service                  |
| lex_db          | Regulatory and legal compliance    | lex-service                   |
| visus_db        | Reporting and analytics            | visus-service                 |

PostgreSQL host: `postgresql.clario360.svc.cluster.local`

---

## Prerequisites

- `kubectl` CLI configured with cluster credentials
- `psql` client available (version 14+)
- Access to the PostgreSQL superuser credentials (stored in Kubernetes secret `postgresql-credentials`)
- Familiarity with PostgreSQL replication concepts
- Backup of all databases completed before any scaling operation

### Verify Access

```bash
# Get PostgreSQL credentials
export PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 \
  -o jsonpath='{.data.postgres-password}' | base64 -d)

# Test connectivity via port-forward
kubectl port-forward svc/postgresql -n clario360 5432:5432 &

# Verify connection
psql -h 127.0.0.1 -U postgres -d platform_core -c "SELECT version();"
```

---

## Procedure

### Step 1: Assess Current Database State

```bash
# Connect to PostgreSQL
kubectl port-forward svc/postgresql -n clario360 5432:5432 &
export PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 \
  -o jsonpath='{.data.postgres-password}' | base64 -d)

# Check current connections across all databases
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT datname, numbackends, xact_commit, xact_rollback, blks_hit, blks_read,
       tup_returned, tup_fetched, tup_inserted, tup_updated, tup_deleted
FROM pg_stat_database
WHERE datname IN ('platform_core','cyber_db','data_db','acta_db','lex_db','visus_db')
ORDER BY numbackends DESC;
"

# Check current max_connections setting
psql -h 127.0.0.1 -U postgres -d platform_core -c "SHOW max_connections;"

# Check active connections vs limit
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT count(*) AS active_connections,
       (SELECT setting::int FROM pg_settings WHERE name = 'max_connections') AS max_connections,
       round(count(*)::numeric / (SELECT setting::int FROM pg_settings WHERE name = 'max_connections') * 100, 2) AS usage_pct
FROM pg_stat_activity;
"

# Check database sizes
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT datname,
       pg_size_pretty(pg_database_size(datname)) AS size
FROM pg_database
WHERE datname IN ('platform_core','cyber_db','data_db','acta_db','lex_db','visus_db')
ORDER BY pg_database_size(datname) DESC;
"

# Check for long-running queries
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state, datname
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '30 seconds'
  AND state != 'idle'
ORDER BY duration DESC;
"
```

### Step 2: Add Read Replicas

#### 2a: Deploy a PostgreSQL Read Replica StatefulSet

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgresql-replica
  namespace: clario360
  labels:
    app: postgresql
    role: replica
spec:
  serviceName: postgresql-replica
  replicas: 2
  selector:
    matchLabels:
      app: postgresql
      role: replica
  template:
    metadata:
      labels:
        app: postgresql
        role: replica
    spec:
      containers:
      - name: postgresql
        image: postgres:16-alpine
        ports:
        - containerPort: 5432
          name: postgresql
        env:
        - name: PGDATA
          value: /var/lib/postgresql/data/pgdata
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgresql-credentials
              key: postgres-password
        - name: PGUSER
          value: "replicator"
        - name: PGPASSWORD
          valueFrom:
            secretKeyRef:
              name: postgresql-credentials
              key: replication-password
        command:
        - bash
        - -c
        - |
          # Wait for primary to be available
          until pg_isready -h postgresql.clario360.svc.cluster.local -U postgres; do
            echo "Waiting for primary..."
            sleep 2
          done
          # Take base backup from primary
          pg_basebackup -h postgresql.clario360.svc.cluster.local \
            -U replicator -D /var/lib/postgresql/data/pgdata \
            -Fp -Xs -P -R
          # Start PostgreSQL in replica mode
          exec postgres -c hot_standby=on \
            -c primary_conninfo="host=postgresql.clario360.svc.cluster.local port=5432 user=replicator password=$(cat /run/secrets/replication-password)"
        resources:
          requests:
            cpu: "2"
            memory: 4Gi
          limits:
            cpu: "4"
            memory: 8Gi
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: premium-rwo
      resources:
        requests:
          storage: 100Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgresql-replica
  namespace: clario360
  labels:
    app: postgresql
    role: replica
spec:
  ports:
  - port: 5432
    targetPort: 5432
  selector:
    app: postgresql
    role: replica
  type: ClusterIP
EOF
```

#### 2b: Configure Replication User on Primary

```bash
# Create the replication user on the primary (if not already exists)
psql -h 127.0.0.1 -U postgres -d platform_core -c "
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'replicator') THEN
    CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD 'CHANGE_ME_REPLICATION_PASSWORD';
  END IF;
END
\$\$;
"

# Update pg_hba.conf to allow replication connections
# (This is typically done via ConfigMap for the PostgreSQL pod)
kubectl edit configmap postgresql-config -n clario360
# Add to pg_hba.conf section:
# host replication replicator 10.0.0.0/8 md5

# Reload PostgreSQL configuration
psql -h 127.0.0.1 -U postgres -d platform_core -c "SELECT pg_reload_conf();"
```

#### 2c: Verify Replication is Active

```bash
# Check replication status on primary
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT client_addr, state, sent_lsn, write_lsn, flush_lsn, replay_lsn,
       (sent_lsn - replay_lsn) AS replication_lag_bytes,
       now() - reply_time AS reply_lag
FROM pg_stat_replication;
"

# Check replica is in recovery mode (run on replica)
kubectl exec -n clario360 postgresql-replica-0 -- \
  psql -U postgres -c "SELECT pg_is_in_recovery();"

# Verify replica can serve read queries
kubectl exec -n clario360 postgresql-replica-0 -- \
  psql -U postgres -d platform_core -c "SELECT count(*) FROM pg_stat_user_tables;"
```

### Step 3: Configure PgBouncer Connection Pooling

#### 3a: Deploy PgBouncer

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: pgbouncer-config
  namespace: clario360
data:
  pgbouncer.ini: |
    [databases]
    platform_core = host=postgresql.clario360.svc.cluster.local port=5432 dbname=platform_core
    platform_core_ro = host=postgresql-replica.clario360.svc.cluster.local port=5432 dbname=platform_core
    cyber_db = host=postgresql.clario360.svc.cluster.local port=5432 dbname=cyber_db
    cyber_db_ro = host=postgresql-replica.clario360.svc.cluster.local port=5432 dbname=cyber_db
    data_db = host=postgresql.clario360.svc.cluster.local port=5432 dbname=data_db
    data_db_ro = host=postgresql-replica.clario360.svc.cluster.local port=5432 dbname=data_db
    acta_db = host=postgresql.clario360.svc.cluster.local port=5432 dbname=acta_db
    acta_db_ro = host=postgresql-replica.clario360.svc.cluster.local port=5432 dbname=acta_db
    lex_db = host=postgresql.clario360.svc.cluster.local port=5432 dbname=lex_db
    lex_db_ro = host=postgresql-replica.clario360.svc.cluster.local port=5432 dbname=lex_db
    visus_db = host=postgresql.clario360.svc.cluster.local port=5432 dbname=visus_db
    visus_db_ro = host=postgresql-replica.clario360.svc.cluster.local port=5432 dbname=visus_db

    [pgbouncer]
    listen_addr = 0.0.0.0
    listen_port = 6432
    auth_type = md5
    auth_file = /etc/pgbouncer/userlist.txt
    pool_mode = transaction
    default_pool_size = 25
    min_pool_size = 5
    reserve_pool_size = 5
    reserve_pool_timeout = 3
    max_client_conn = 1000
    max_db_connections = 50
    server_idle_timeout = 300
    server_lifetime = 3600
    server_connect_timeout = 15
    query_wait_timeout = 120
    client_idle_timeout = 0
    log_connections = 1
    log_disconnections = 1
    log_pooler_errors = 1
    stats_period = 60
    admin_users = postgres
    stats_users = postgres,monitoring
  userlist.txt: |
    "postgres" "SCRAM-SHA-256$4096:CHANGE_ME_HASH"
    "app_user" "SCRAM-SHA-256$4096:CHANGE_ME_HASH"
    "monitoring" "SCRAM-SHA-256$4096:CHANGE_ME_HASH"
---
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
        - containerPort: 6432
          name: pgbouncer
        volumeMounts:
        - name: config
          mountPath: /etc/pgbouncer
          readOnly: true
        resources:
          requests:
            cpu: 250m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          tcpSocket:
            port: 6432
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          tcpSocket:
            port: 6432
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: pgbouncer-config
---
apiVersion: v1
kind: Service
metadata:
  name: pgbouncer
  namespace: clario360
  labels:
    app: pgbouncer
spec:
  ports:
  - port: 6432
    targetPort: 6432
    name: pgbouncer
  selector:
    app: pgbouncer
  type: ClusterIP
EOF
```

#### 3b: Update Service Connection Strings

Update each service's environment variables to use PgBouncer instead of direct PostgreSQL connections.

```bash
# Update api-gateway to use PgBouncer
kubectl set env deployment/api-gateway -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432

# Update iam-service
kubectl set env deployment/iam-service -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432

# Update audit-service
kubectl set env deployment/audit-service -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432

# Update workflow-engine
kubectl set env deployment/workflow-engine -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432

# Update notification-service
kubectl set env deployment/notification-service -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432

# Update cyber-service
kubectl set env deployment/cyber-service -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432 \
  DATABASE_NAME=cyber_db

# Update data-service
kubectl set env deployment/data-service -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432 \
  DATABASE_NAME=data_db

# Update acta-service
kubectl set env deployment/acta-service -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432 \
  DATABASE_NAME=acta_db

# Update lex-service
kubectl set env deployment/lex-service -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432 \
  DATABASE_NAME=lex_db

# Update visus-service
kubectl set env deployment/visus-service -n clario360 \
  DATABASE_HOST=pgbouncer.clario360.svc.cluster.local \
  DATABASE_PORT=6432 \
  DATABASE_NAME=visus_db
```

#### 3c: Verify PgBouncer is Working

```bash
# Port-forward to PgBouncer
kubectl port-forward svc/pgbouncer -n clario360 6432:6432 &

# Connect through PgBouncer
psql -h 127.0.0.1 -p 6432 -U postgres -d platform_core -c "SELECT 1;"

# Check PgBouncer stats
psql -h 127.0.0.1 -p 6432 -U postgres -d pgbouncer -c "SHOW POOLS;"
psql -h 127.0.0.1 -p 6432 -U postgres -d pgbouncer -c "SHOW STATS;"
psql -h 127.0.0.1 -p 6432 -U postgres -d pgbouncer -c "SHOW CLIENTS;"
psql -h 127.0.0.1 -p 6432 -U postgres -d pgbouncer -c "SHOW SERVERS;"
```

### Step 4: Adjust max_connections

```bash
# Check current value
psql -h 127.0.0.1 -U postgres -d platform_core -c "SHOW max_connections;"

# Calculate recommended max_connections:
# Formula: (num_services * max_pool_per_service * max_replicas) + reserved_superuser + pgbouncer_overhead
# Example: (10 * 10 * 8) + 10 + 50 = 860 -> round to 1000

# Update PostgreSQL ConfigMap (adjust value as needed)
kubectl edit configmap postgresql-config -n clario360
# Set: max_connections = 1000

# Restart PostgreSQL to apply (requires downtime)
kubectl rollout restart statefulset/postgresql -n clario360

# Verify the new setting
psql -h 127.0.0.1 -U postgres -d platform_core -c "SHOW max_connections;"

# Also adjust shared_buffers if increasing connections significantly
# Recommended: shared_buffers = 25% of total memory
psql -h 127.0.0.1 -U postgres -d platform_core -c "SHOW shared_buffers;"
```

### Step 5: Vertical Scaling (Increase CPU/Memory for PostgreSQL Pod)

```bash
# Check current resource usage
kubectl top pod -n clario360 -l app=postgresql

# Check current resource requests/limits
kubectl get statefulset postgresql -n clario360 -o jsonpath='{.spec.template.spec.containers[0].resources}'

# Scale vertically by patching the StatefulSet
kubectl patch statefulset postgresql -n clario360 --type merge -p '{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "postgresql",
          "resources": {
            "requests": {
              "cpu": "4",
              "memory": "16Gi"
            },
            "limits": {
              "cpu": "8",
              "memory": "32Gi"
            }
          }
        }]
      }
    }
  }
}'

# This triggers a rolling restart of PostgreSQL pods
kubectl rollout status statefulset/postgresql -n clario360

# After restart, update PostgreSQL memory-related settings
psql -h 127.0.0.1 -U postgres -d platform_core -c "
ALTER SYSTEM SET shared_buffers = '8GB';
ALTER SYSTEM SET effective_cache_size = '24GB';
ALTER SYSTEM SET work_mem = '64MB';
ALTER SYSTEM SET maintenance_work_mem = '2GB';
ALTER SYSTEM SET wal_buffers = '64MB';
SELECT pg_reload_conf();
"

# Verify settings took effect
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT name, setting, unit
FROM pg_settings
WHERE name IN ('shared_buffers','effective_cache_size','work_mem','maintenance_work_mem','wal_buffers','max_connections');
"
```

### Step 6: Increase Storage for PostgreSQL

```bash
# Check current PVC usage
kubectl exec -n clario360 postgresql-0 -- df -h /var/lib/postgresql/data

# Resize PVC (requires StorageClass to support volume expansion)
kubectl patch pvc data-postgresql-0 -n clario360 --type merge -p '{
  "spec": {
    "resources": {
      "requests": {
        "storage": "200Gi"
      }
    }
  }
}'

# Verify the resize (may take a few minutes)
kubectl get pvc data-postgresql-0 -n clario360 -w

# Confirm new size
kubectl exec -n clario360 postgresql-0 -- df -h /var/lib/postgresql/data
```

### Step 7: Monitor Replication Lag

```bash
# Check replication lag in bytes
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT
  client_addr,
  state,
  pg_wal_lsn_diff(pg_current_wal_lsn(), sent_lsn) AS send_lag_bytes,
  pg_wal_lsn_diff(sent_lsn, write_lsn) AS write_lag_bytes,
  pg_wal_lsn_diff(write_lsn, flush_lsn) AS flush_lag_bytes,
  pg_wal_lsn_diff(flush_lsn, replay_lsn) AS replay_lag_bytes,
  pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS total_lag_bytes,
  pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)) AS total_lag_pretty
FROM pg_stat_replication
ORDER BY total_lag_bytes DESC;
"

# Check replication lag in seconds (requires pg_stat_replication_slots on PG 14+)
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT
  slot_name,
  active,
  pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn)) AS lag
FROM pg_replication_slots;
"

# Continuous monitoring (run for 60 seconds)
for i in $(seq 1 12); do
  psql -h 127.0.0.1 -U postgres -d platform_core -t -c "
    SELECT now()::timestamp(0),
           client_addr,
           pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)) AS lag
    FROM pg_stat_replication;
  "
  sleep 5
done
```

---

## Verification

After completing the scaling procedure, confirm the following:

1. All databases are accessible and healthy:
   ```bash
   for db in platform_core cyber_db data_db acta_db lex_db visus_db; do
     echo "=== ${db} ==="
     psql -h 127.0.0.1 -U postgres -d ${db} -c "SELECT 1 AS healthy;"
   done
   ```

2. Read replicas are in sync (lag < 1MB):
   ```bash
   psql -h 127.0.0.1 -U postgres -d platform_core -c "
   SELECT client_addr,
          pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)) AS lag
   FROM pg_stat_replication;
   "
   ```

3. PgBouncer pools are healthy:
   ```bash
   psql -h 127.0.0.1 -p 6432 -U postgres -d pgbouncer -c "SHOW POOLS;"
   ```

4. Connection count is within limits:
   ```bash
   psql -h 127.0.0.1 -U postgres -d platform_core -c "
   SELECT count(*) AS connections,
          (SELECT setting::int FROM pg_settings WHERE name='max_connections') AS max
   FROM pg_stat_activity;
   "
   ```

5. All services can connect through PgBouncer:
   ```bash
   for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
     echo "=== ${svc} ==="
     kubectl logs -n clario360 -l app=${svc} --tail=5 --since=2m | grep -i "database\|postgres\|connect"
   done
   ```

---

## Related Links

| Document                  | Link                                             |
|---------------------------|--------------------------------------------------|
| Horizontal Pod Scaling    | [SC-001-horizontal-scaling.md](SC-001-horizontal-scaling.md) |
| Kafka Scaling             | [SC-003-kafka-scaling.md](SC-003-kafka-scaling.md)           |
| Capacity Planning         | [SC-005-capacity-planning.md](SC-005-capacity-planning.md)   |
| PostgreSQL Documentation  | https://www.postgresql.org/docs/16/high-availability.html    |
| PgBouncer Documentation   | https://www.pgbouncer.org/config.html                        |
