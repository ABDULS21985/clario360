# OP-006: Vault Secret Rotation

| Field | Value |
|-------|-------|
| **Runbook ID** | OP-006 |
| **Title** | Vault Secret Rotation |
| **Frequency** | Quarterly |
| **Owner** | Platform Security Team |
| **Last Updated** | 2026-03-08 |
| **Estimated Duration** | 2–3 hours |
| **Risk Level** | High — services will briefly lose DB/cache connectivity during rolling restarts |
| **Approval Required** | Yes — Change Advisory Board (CAB) approval required |
| **Maintenance Window** | Scheduled maintenance window (Saturday 02:00–06:00 UTC) |

## Summary

This runbook covers quarterly rotation of all secrets managed by HashiCorp Vault for the Clario 360 platform. Secrets rotated include:

1. PostgreSQL database credentials (all 6 databases)
2. JWT signing keys (RS256 key pair)
3. API keys (inter-service and external)
4. Kafka credentials (SASL/SCRAM)
5. Redis password
6. Rolling restart of all services to pick up new secrets

All secrets are stored in Vault and injected into Kubernetes pods via the Vault Agent sidecar. After rotation, services must be restarted in a specific order to avoid dependency failures.

## Prerequisites

```bash
export NAMESPACE=clario360
export VAULT_ADDR=https://vault.clario360.io
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin

# Authenticate to Vault
vault login -method=oidc role=platform-admin
```

Verify Vault connectivity:

```bash
vault status
vault token lookup
```

Confirm you have the required Vault policies:

```bash
vault token capabilities secret/data/clario360/
vault token capabilities database/creds/
vault token capabilities transit/keys/
```

## Pre-Rotation Checks

### 1. Verify all services are healthy before starting

```bash
SERVICES="api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service"

for svc in $SERVICES; do
  echo "=== $svc ==="
  kubectl -n $NAMESPACE rollout status deployment/$svc --timeout=30s
  kubectl -n $NAMESPACE exec deployment/$svc -- wget -qO- http://localhost:8080/healthz
  echo ""
done
```

### 2. Verify current secret versions in Vault

```bash
# Database credentials
for db in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== $db ==="
  vault kv metadata get secret/clario360/database/$db | grep -E "current_version|created_time"
done

# JWT signing keys
vault kv metadata get secret/clario360/auth/jwt-signing-key | grep -E "current_version|created_time"

# API keys
vault kv metadata get secret/clario360/api-keys/internal | grep -E "current_version|created_time"

# Kafka credentials
vault kv metadata get secret/clario360/kafka/credentials | grep -E "current_version|created_time"

# Redis password
vault kv metadata get secret/clario360/redis/password | grep -E "current_version|created_time"
```

### 3. Create a backup snapshot of current Vault state

```bash
vault operator raft snapshot save vault-pre-rotation-$(date +%Y%m%d-%H%M%S).snap
```

### 4. Notify stakeholders

```bash
curl -X POST https://api.clario360.io/v1/notifications/broadcast \
  -H "Authorization: Bearer $(vault kv get -field=token secret/clario360/api-keys/ops-bot)" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "ops-announcements",
    "message": "Scheduled secret rotation starting. Services will experience brief restarts over the next 2 hours.",
    "severity": "info"
  }'
```

---

## Procedure

### Step 1: Rotate Database Credentials

Rotate credentials for each database. Vault's database secrets engine generates new credentials and updates the database user password atomically.

#### 1a. Rotate the PostgreSQL root credential

```bash
vault write -force database/rotate-root/clario360-postgresql
```

#### 1b. Rotate dynamic credentials for each database

```bash
DATABASES="platform_core cyber_db data_db acta_db lex_db visus_db"

for db in $DATABASES; do
  echo "=== Rotating credentials for $db ==="

  # Generate new static credentials
  NEW_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)

  # Update password in PostgreSQL
  kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U postgres -c \
    "ALTER USER ${db}_app PASSWORD '${NEW_PASSWORD}';"

  # Store new password in Vault
  vault kv put secret/clario360/database/$db \
    username="${db}_app" \
    password="${NEW_PASSWORD}" \
    host="$PG_HOST" \
    port="5432" \
    dbname="$db" \
    sslmode="require"

  echo "Rotated credentials for $db"
done
```

#### 1c. Verify new database credentials work

```bash
for db in $DATABASES; do
  echo "=== Verifying $db ==="
  DB_USER=$(vault kv get -field=username secret/clario360/database/$db)
  DB_PASS=$(vault kv get -field=password secret/clario360/database/$db)

  kubectl -n $NAMESPACE exec deployment/postgresql -- psql \
    "postgresql://${DB_USER}:${DB_PASS}@localhost:5432/${db}?sslmode=require" \
    -c "SELECT 1 AS connection_test;"

  if [ $? -eq 0 ]; then
    echo "OK: $db credentials verified"
  else
    echo "FAIL: $db credentials verification failed"
    exit 1
  fi
done
```

---

### Step 2: Rotate JWT Signing Keys

The IAM service uses RS256 JWT tokens. Rotation involves generating a new key pair, storing it in Vault, and keeping the old public key for a grace period so existing tokens can still be verified.

#### 2a. Generate new RS256 key pair

```bash
# Generate new private key
openssl genrsa -out /tmp/jwt-private-new.pem 4096

# Extract public key
openssl rsa -in /tmp/jwt-private-new.pem -pubout -out /tmp/jwt-public-new.pem

# Read key contents
JWT_PRIVATE_KEY=$(cat /tmp/jwt-private-new.pem)
JWT_PUBLIC_KEY=$(cat /tmp/jwt-public-new.pem)
```

#### 2b. Archive the current public key for token verification grace period

```bash
# Get current public key before overwriting
CURRENT_PUBLIC_KEY=$(vault kv get -field=public_key secret/clario360/auth/jwt-signing-key)

# Store as previous key (services will check both current and previous for verification)
vault kv put secret/clario360/auth/jwt-previous-key \
  public_key="$CURRENT_PUBLIC_KEY" \
  archived_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  expires_at="$(date -u -d '+24 hours' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v+24H +%Y-%m-%dT%H:%M:%SZ)"
```

#### 2c. Store new JWT signing key in Vault

```bash
vault kv put secret/clario360/auth/jwt-signing-key \
  private_key="$JWT_PRIVATE_KEY" \
  public_key="$JWT_PUBLIC_KEY" \
  algorithm="RS256" \
  key_size="4096" \
  rotated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  rotated_by="$(vault token lookup -format=json | jq -r '.data.display_name')"
```

#### 2d. Clean up local key files

```bash
shred -u /tmp/jwt-private-new.pem /tmp/jwt-public-new.pem
```

---

### Step 3: Rotate API Keys

#### 3a. Rotate inter-service API keys

```bash
INTERNAL_SERVICES="api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service"

for svc in $INTERNAL_SERVICES; do
  NEW_API_KEY=$(openssl rand -hex 32)

  vault kv put secret/clario360/api-keys/service/$svc \
    api_key="$NEW_API_KEY" \
    service_name="$svc" \
    rotated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  echo "Rotated API key for $svc"
done
```

#### 3b. Rotate shared internal communication key

```bash
NEW_INTERNAL_KEY=$(openssl rand -hex 32)

vault kv put secret/clario360/api-keys/internal \
  key="$NEW_INTERNAL_KEY" \
  rotated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

#### 3c. Rotate external-facing API keys (if any)

```bash
# List current external API keys
vault kv list secret/clario360/api-keys/external/

# For each external integration, generate and store a new key
# Coordinate with integration partners before rotating
for integration in $(vault kv list -format=json secret/clario360/api-keys/external/ | jq -r '.[]'); do
  NEW_EXT_KEY=$(openssl rand -hex 32)

  vault kv put secret/clario360/api-keys/external/$integration \
    api_key="$NEW_EXT_KEY" \
    rotated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  echo "Rotated external API key for $integration — notify integration partner"
done
```

---

### Step 4: Rotate Kafka Credentials

#### 4a. Generate new SASL/SCRAM credentials

```bash
NEW_KAFKA_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)

# Update the SCRAM credential in Kafka
kubectl -n kafka exec kafka-0 -- kafka-configs.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --alter \
  --entity-type users \
  --entity-name clario360-app \
  --add-config "SCRAM-SHA-512=[password=${NEW_KAFKA_PASSWORD}]"
```

#### 4b. Store new Kafka credentials in Vault

```bash
vault kv put secret/clario360/kafka/credentials \
  username="clario360-app" \
  password="$NEW_KAFKA_PASSWORD" \
  mechanism="SCRAM-SHA-512" \
  bootstrap_servers="kafka-0.kafka-headless.kafka.svc.cluster.local:9092,kafka-1.kafka-headless.kafka.svc.cluster.local:9092,kafka-2.kafka-headless.kafka.svc.cluster.local:9092" \
  rotated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

#### 4c. Verify Kafka connectivity with new credentials

```bash
kubectl -n kafka exec kafka-0 -- kafka-broker-api-versions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --command-config <(cat <<EOF
security.protocol=SASL_PLAINTEXT
sasl.mechanism=SCRAM-SHA-512
sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="clario360-app" password="${NEW_KAFKA_PASSWORD}";
EOF
) 2>&1 | head -5
```

---

### Step 5: Rotate Redis Password

#### 5a. Set new Redis password

```bash
NEW_REDIS_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)

# Get current password
CURRENT_REDIS_PASSWORD=$(vault kv get -field=password secret/clario360/redis/password)

# Update Redis AUTH password (Redis 6+ ACL method)
kubectl -n $NAMESPACE exec deployment/redis -- redis-cli \
  -a "$CURRENT_REDIS_PASSWORD" \
  ACL SETUSER default ">${NEW_REDIS_PASSWORD}"

# Remove old password
kubectl -n $NAMESPACE exec deployment/redis -- redis-cli \
  -a "$NEW_REDIS_PASSWORD" \
  ACL SETUSER default "!${CURRENT_REDIS_PASSWORD}"
```

#### 5b. Store new Redis password in Vault

```bash
vault kv put secret/clario360/redis/password \
  password="$NEW_REDIS_PASSWORD" \
  host="redis.$NAMESPACE.svc.cluster.local" \
  port="6379" \
  rotated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

#### 5c. Verify Redis connectivity with new password

```bash
kubectl -n $NAMESPACE exec deployment/redis -- redis-cli \
  -a "$NEW_REDIS_PASSWORD" \
  PING
```

Expected output: `PONG`

---

### Step 6: Rolling Restart Services

Restart services in dependency order. The Vault Agent sidecar will inject the updated secrets on pod startup.

#### 6a. Restart infrastructure-facing services first

```bash
# Phase 1: IAM service (handles auth — must be up before others can authenticate)
kubectl -n $NAMESPACE rollout restart deployment/iam-service
kubectl -n $NAMESPACE rollout status deployment/iam-service --timeout=300s

# Phase 2: Core platform services
for svc in audit-service workflow-engine notification-service; do
  kubectl -n $NAMESPACE rollout restart deployment/$svc
  kubectl -n $NAMESPACE rollout status deployment/$svc --timeout=300s
done

# Phase 3: Suite services
for svc in cyber-service data-service acta-service lex-service visus-service; do
  kubectl -n $NAMESPACE rollout restart deployment/$svc
  kubectl -n $NAMESPACE rollout status deployment/$svc --timeout=300s
done

# Phase 4: API gateway (restart last to minimize user-facing impact)
kubectl -n $NAMESPACE rollout restart deployment/api-gateway
kubectl -n $NAMESPACE rollout status deployment/api-gateway --timeout=300s
```

#### 6b. Wait for all pods to be ready

```bash
kubectl -n $NAMESPACE wait --for=condition=ready pod --all --timeout=600s
```

---

### Step 7: Post-Rotation Verification

#### 7a. Verify all services are healthy

```bash
SERVICES="api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service"

echo "=== Health Check ==="
FAILED=0
for svc in $SERVICES; do
  HEALTH=$(kubectl -n $NAMESPACE exec deployment/$svc -- wget -qO- http://localhost:8080/healthz 2>/dev/null)
  READY=$(kubectl -n $NAMESPACE exec deployment/$svc -- wget -qO- http://localhost:8080/readyz 2>/dev/null)
  echo "$svc: health=$HEALTH ready=$READY"
  if [ "$HEALTH" != "ok" ] || [ "$READY" != "ok" ]; then
    echo "  WARNING: $svc is not fully healthy!"
    FAILED=$((FAILED + 1))
  fi
done

if [ $FAILED -gt 0 ]; then
  echo ""
  echo "ALERT: $FAILED service(s) not healthy after rotation. Investigate immediately."
else
  echo ""
  echo "All services healthy."
fi
```

#### 7b. Verify API gateway can reach all backends

```bash
curl -s https://api.clario360.io/healthz | jq .
curl -s https://api.clario360.io/readyz | jq .
```

#### 7c. Verify database connectivity through services

```bash
# Quick functional test: attempt a login
curl -s -X POST https://api.clario360.io/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "healthcheck@clario360.io", "password": "unused"}' \
  -o /dev/null -w "HTTP %{http_code}\n"
# Expect 401 (unauthorized) — NOT 500 (which would indicate DB/service failure)
```

#### 7d. Check for errors in logs

```bash
for svc in $SERVICES; do
  echo "=== $svc recent errors ==="
  kubectl -n $NAMESPACE logs deployment/$svc --since=30m | grep -i -c "error"
done
```

#### 7e. Verify Vault secret versions were incremented

```bash
for db in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== $db ==="
  vault kv metadata get secret/clario360/database/$db | grep "current_version"
done

vault kv metadata get secret/clario360/auth/jwt-signing-key | grep "current_version"
vault kv metadata get secret/clario360/kafka/credentials | grep "current_version"
vault kv metadata get secret/clario360/redis/password | grep "current_version"
```

---

## Rollback Procedure

If services fail to start after rotation, roll back to the previous Vault secret version.

### Roll back a specific secret

```bash
# Example: roll back database credential for platform_core
# Get the previous version number
PREV_VERSION=$(vault kv metadata get -format=json secret/clario360/database/platform_core | jq '.data.versions | to_entries | sort_by(.key | tonumber) | .[-2].key')

# Roll back by reading old version and writing it as new
vault kv get -version=$PREV_VERSION -format=json secret/clario360/database/platform_core | \
  jq -r '.data.data' | \
  vault kv put secret/clario360/database/platform_core -

# Also reset the database password to the old value
OLD_PASSWORD=$(vault kv get -version=$PREV_VERSION -field=password secret/clario360/database/platform_core)
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U postgres -c \
  "ALTER USER platform_core_app PASSWORD '${OLD_PASSWORD}';"
```

### Roll back all secrets using Vault snapshot

```bash
# Restore from the pre-rotation snapshot
vault operator raft snapshot restore vault-pre-rotation-YYYYMMDD-HHMMSS.snap

# Restart all services
SERVICES="iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service api-gateway"
for svc in $SERVICES; do
  kubectl -n $NAMESPACE rollout restart deployment/$svc
done
kubectl -n $NAMESPACE wait --for=condition=ready pod --all --timeout=600s
```

---

## Post-Rotation Cleanup

### Remove previous JWT key after grace period (24 hours)

```bash
# Run this 24 hours after rotation, once all old tokens have expired
vault kv delete secret/clario360/auth/jwt-previous-key
```

### Update rotation log

```bash
vault kv put secret/clario360/rotation-log/$(date +%Y-%m-%d) \
  performed_by="$(vault token lookup -format=json | jq -r '.data.display_name')" \
  performed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  secrets_rotated="database,jwt,api-keys,kafka,redis" \
  status="completed" \
  next_rotation="$(date -u -d '+90 days' +%Y-%m-%d 2>/dev/null || date -u -v+90d +%Y-%m-%d)"
```

---

## Related Links

- [OP-005: Certificate Renewal](OP-005-certificate-renewal.md)
- [IR-008: Security Breach](../incident-response/IR-008-security-breach.md)
- [Vault Documentation](https://vault.clario360.io/ui/)
- [Grafana — Security Overview](https://grafana.clario360.io/d/security-overview)
- [Grafana — Service Health](https://grafana.clario360.io/d/service-health)
