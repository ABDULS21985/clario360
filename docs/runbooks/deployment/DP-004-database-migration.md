# DP-004: Run Database Migrations

| Field            | Value                                            |
|------------------|--------------------------------------------------|
| **Runbook ID**   | DP-004                                           |
| **Title**        | Run Database Migrations                          |
| **Author**       | Platform Engineering                             |
| **Last Updated** | 2026-03-08                                       |
| **Severity**     | Standard Change (High Risk)                      |
| **Services**     | All Clario 360 services with database schemas    |
| **Namespace**    | clario360                                        |
| **Approvers**    | Engineering Lead, DBA On-Call                    |
| **Est. Duration**| 15-60 minutes (varies by migration complexity)   |

---

## Summary

This runbook covers running database schema migrations for the Clario 360 platform. Migrations are managed via the Go-based migrator tool (`cmd/migrator/main.go`) and applied as Kubernetes jobs. Each service owns its database schema and migrations.

### Database Inventory

| Database        | Owning Services                                               | Cloud SQL Instance               |
|-----------------|---------------------------------------------------------------|----------------------------------|
| platform_core   | iam-service, audit-service, workflow-engine, notification-service | clario360-prod-platform-core   |
| cyber_db        | cyber-service                                                 | clario360-prod-cyber             |
| data_db         | data-service                                                  | clario360-prod-data              |
| acta_db         | acta-service                                                  | clario360-prod-acta              |
| lex_db          | lex-service                                                   | clario360-prod-lex               |
| visus_db        | visus-service                                                 | clario360-prod-visus             |

### Migration File Convention

Migration files are located at `internal/<service>/migrations/` and follow the naming pattern:

```
YYYYMMDDHHMMSS_description.up.sql
YYYYMMDDHHMMSS_description.down.sql
```

Every `up` migration MUST have a corresponding `down` migration.

---

## Prerequisites

- [ ] `kubectl` configured with production GKE cluster credentials
- [ ] Migration SQL files reviewed by at least one other engineer
- [ ] Both `up` and `down` migration files present and reviewed
- [ ] Migration tested on staging environment
- [ ] Database backup taken (Step 1)
- [ ] Maintenance window communicated (for destructive migrations)
- [ ] Service compatibility verified (new code works with both old and new schema)

### Authenticate

```bash
gcloud container clusters get-credentials clario360-prod \
  --region us-central1 \
  --project clario360-prod
```

### Verify Database Connectivity

```bash
# Verify Cloud SQL proxy is running
kubectl get pods -n clario360 -l app=cloud-sql-proxy
```

---

## Procedure

### Step 1: Pre-Migration Backup

**CRITICAL:** Always take a backup before running migrations. This is your safety net.

#### Automated Backup via gcloud

```bash
# platform_core
gcloud sql backups create \
  --instance=clario360-prod-platform-core \
  --project=clario360-prod \
  --description="Pre-migration backup $(date +%Y%m%d-%H%M%S)"

# cyber_db
gcloud sql backups create \
  --instance=clario360-prod-cyber \
  --project=clario360-prod \
  --description="Pre-migration backup $(date +%Y%m%d-%H%M%S)"

# data_db
gcloud sql backups create \
  --instance=clario360-prod-data \
  --project=clario360-prod \
  --description="Pre-migration backup $(date +%Y%m%d-%H%M%S)"

# acta_db
gcloud sql backups create \
  --instance=clario360-prod-acta \
  --project=clario360-prod \
  --description="Pre-migration backup $(date +%Y%m%d-%H%M%S)"

# lex_db
gcloud sql backups create \
  --instance=clario360-prod-lex \
  --project=clario360-prod \
  --description="Pre-migration backup $(date +%Y%m%d-%H%M%S)"

# visus_db
gcloud sql backups create \
  --instance=clario360-prod-visus \
  --project=clario360-prod \
  --description="Pre-migration backup $(date +%Y%m%d-%H%M%S)"
```

Verify backup completed:

```bash
gcloud sql backups list \
  --instance=clario360-prod-platform-core \
  --project=clario360-prod \
  --limit=3 \
  --format="table(id, windowStartTime, status, description)"
```

Record the backup ID for each database -- you will need this for restore if rollback is required:

```bash
BACKUP_ID_PLATFORM_CORE=$(gcloud sql backups list \
  --instance=clario360-prod-platform-core \
  --project=clario360-prod \
  --limit=1 \
  --format="value(id)")
echo "platform_core backup ID: ${BACKUP_ID_PLATFORM_CORE}"
```

### Step 2: Review Migration SQL (Up and Down)

Review the migration files before applying:

```bash
# List pending migration files for the service
# Replace <service> with: iam-service, cyber-service, data-service, etc.
SERVICE="iam-service"
kubectl run migration-review-${SERVICE} \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/${SERVICE}:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --version
```

Review the actual SQL files in the repository:

```bash
cd /Users/mac/clario360

# List migration files
ls -la internal/iam/migrations/
ls -la internal/cyber/migrations/
ls -la internal/data/migrations/
ls -la internal/acta/migrations/
ls -la internal/lex/migrations/
ls -la internal/visus/migrations/
```

**Review checklist:**

- [ ] `up.sql`: Schema changes are additive where possible (add columns, not remove)
- [ ] `up.sql`: New columns have DEFAULT values if the column is NOT NULL
- [ ] `up.sql`: Large table ALTERs use `ALTER TABLE ... ADD COLUMN ... DEFAULT ...` (not backfill in the same transaction)
- [ ] `down.sql`: Reverses the `up.sql` changes completely
- [ ] `down.sql`: Handles the case where `up.sql` was partially applied
- [ ] No data destruction in `up.sql` (DROP TABLE, DROP COLUMN only in cleanup migrations)
- [ ] Index creation uses `CREATE INDEX CONCURRENTLY` where applicable

### Step 3: Check Current Migration Version

```bash
# platform_core
kubectl run migration-version-platform-core \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/iam-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --version

# cyber_db
kubectl run migration-version-cyber-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/cyber-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 cyber-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --version

# data_db
kubectl run migration-version-data-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/data-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 data-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --version

# acta_db
kubectl run migration-version-acta-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/acta-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 acta-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --version

# lex_db
kubectl run migration-version-lex-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/lex-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 lex-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --version

# visus_db
kubectl run migration-version-visus-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/visus-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 visus-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --version
```

Record the current version for each database. You will need this for rollback.

### Step 4: Run Migration (Up)

Apply the migration for each database that needs it. Replace the service name and database credentials secret accordingly.

#### platform_core

```bash
kubectl run migration-up-platform-core \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/iam-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=up
```

#### cyber_db

```bash
kubectl run migration-up-cyber-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/cyber-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 cyber-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=up
```

#### data_db

```bash
kubectl run migration-up-data-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/data-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 data-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=up
```

#### acta_db

```bash
kubectl run migration-up-acta-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/acta-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 acta-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=up
```

#### lex_db

```bash
kubectl run migration-up-lex-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/lex-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 lex-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=up
```

#### visus_db

```bash
kubectl run migration-up-visus-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/visus-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 visus-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=up
```

### Step 5: Verify Migration Applied

Check the new migration version for each database:

```bash
# platform_core
kubectl run migration-verify-platform-core \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/iam-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --version

# Repeat for other databases as needed (see Step 3 for commands)
```

Verify the schema changes are present by connecting to the database and inspecting:

```bash
# Connect to platform_core via Cloud SQL proxy
kubectl run db-shell-platform-core \
  --namespace=clario360 \
  --image=postgres:15 \
  --restart=Never \
  --rm -it \
  --env="PGPASSWORD=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.host}' | base64 -d) \
  -U $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.username}' | base64 -d) \
  -d platform_core \
  -c "\dt" \
  -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 5;"
```

### Step 6: Test Rollback Migration (Dry Run on Staging)

**IMPORTANT:** Always verify the down migration works on staging before considering it a viable rollback option for production.

```bash
# On staging: run the down migration
kubectl run migration-down-staging-platform-core \
  --namespace=clario360-staging \
  --image=gcr.io/clario360-prod/iam-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360-staging platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=down --steps=1

# Verify the schema reverted
kubectl run migration-version-staging-platform-core \
  --namespace=clario360-staging \
  --image=gcr.io/clario360-prod/iam-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360-staging platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --version

# Re-apply the up migration on staging to restore it
kubectl run migration-up-staging-platform-core \
  --namespace=clario360-staging \
  --image=gcr.io/clario360-prod/iam-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360-staging platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=up
```

### Step 7: Verify Service Compatibility with New Schema

After migration, verify all services that use the migrated database are still healthy:

```bash
# For platform_core database (shared by multiple services)
for svc in iam-service audit-service workflow-engine notification-service; do
  echo "=== $svc ==="
  kubectl rollout status deployment/$svc -n clario360 --timeout=60s
  kubectl exec -n clario360 deploy/$svc -- wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "UNHEALTHY"
done

# For individual databases
for pair in "cyber-service:cyber" "data-service:data" "acta-service:acta" "lex-service:lex" "visus-service:visus"; do
  svc="${pair%%:*}"
  echo "=== $svc ==="
  kubectl rollout status deployment/$svc -n clario360 --timeout=60s
  kubectl exec -n clario360 deploy/$svc -- wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "UNHEALTHY"
done
```

Check for any database connection errors in the logs:

```bash
kubectl logs -n clario360 -l app.kubernetes.io/part-of=clario360 \
  --since=10m --tail=100 \
  | grep -i "database\|migration\|schema\|sql\|connection refused\|relation.*does not exist"
```

---

## Rollback

### Option A: Down Migration (Preferred)

Roll back the migration using the down migration file:

```bash
# Example for platform_core
kubectl run migration-rollback-platform-core \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/iam-service:latest \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=down --steps=1
```

Use `--steps=N` to roll back N migrations. Default (without `--steps`) rolls back all migrations.

### Option B: Database Restore from Backup (Last Resort)

If the down migration fails or causes data issues, restore from the pre-migration backup:

```bash
# WARNING: This will cause downtime. Scale down services first.
for deploy in iam-service audit-service workflow-engine notification-service; do
  kubectl scale deployment/$deploy -n clario360 --replicas=0
done

# Restore from backup
gcloud sql backups restore ${BACKUP_ID_PLATFORM_CORE} \
  --restore-instance=clario360-prod-platform-core \
  --project=clario360-prod \
  --quiet

# Wait for restore to complete
gcloud sql operations list \
  --instance=clario360-prod-platform-core \
  --project=clario360-prod \
  --limit=1 \
  --format="table(name, operationType, status)"

# Scale services back up
for deploy in iam-service audit-service workflow-engine notification-service; do
  kubectl scale deployment/$deploy -n clario360 --replicas=3
done
```

---

## Verification

| Check                          | Command                                                           | Expected                          |
|-------------------------------|-------------------------------------------------------------------|-----------------------------------|
| Migration version correct     | `go run cmd/migrator/main.go --version`                           | Shows expected version number     |
| Schema changes present        | `psql -c "\dt"` or `psql -c "\d <table>"`                        | New tables/columns visible        |
| No dirty migration state      | `SELECT dirty FROM schema_migrations`                             | `false`                           |
| Services healthy              | `kubectl get pods -n clario360 --field-selector=status.phase!=Running` | No resources found           |
| No DB errors in logs          | `kubectl logs ... \| grep -i "sql\|database\|migration"`         | No error entries                  |
| Backup exists                 | `gcloud sql backups list --instance=...`                          | Recent backup present             |

---

## Troubleshooting

### Migration Stuck in "Dirty" State

If a migration failed partway through, the schema_migrations table may be in a dirty state:

```bash
kubectl run db-fix-dirty \
  --namespace=clario360 \
  --image=postgres:15 \
  --restart=Never \
  --rm -it \
  --env="PGPASSWORD=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.host}' | base64 -d) \
  -U $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.username}' | base64 -d) \
  -d platform_core \
  -c "UPDATE schema_migrations SET dirty = false;"
```

Then manually inspect the schema state and decide whether to re-run the up migration or run the down migration first.

### Migration Timeout

For large tables, migrations may take longer than expected. Increase the pod timeout:

```bash
kubectl run migration-up-long \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/iam-service:latest \
  --restart=Never \
  --rm -it \
  --request-timeout=3600 \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  --env="MIGRATION_TIMEOUT=3600s" \
  -- go run cmd/migrator/main.go --direction=up
```

### Lock Wait Timeout

If the migration is blocked by active queries:

```bash
# Check for blocking queries
kubectl run db-check-locks \
  --namespace=clario360 \
  --image=postgres:15 \
  --restart=Never \
  --rm -it \
  --env="PGPASSWORD=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.host}' | base64 -d) \
  -U $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.username}' | base64 -d) \
  -d platform_core \
  -c "SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state FROM pg_stat_activity WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes' ORDER BY duration DESC;"
```

---

## Related Links

- [DP-001: Deploy a New Version](./DP-001-new-release.md)
- [DP-002: Rollback to Previous Version](./DP-002-rollback.md)
- [DP-003: Emergency Hotfix Procedure](./DP-003-hotfix.md)
- [DP-005: Feature Flag Management](./DP-005-feature-flags.md)
- Cloud SQL Console: `https://console.cloud.google.com/sql/instances?project=clario360-prod`
- Migration Documentation: `https://docs.clario360.internal/database/migrations`
