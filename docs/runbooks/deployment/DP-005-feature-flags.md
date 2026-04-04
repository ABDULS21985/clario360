# DP-005: Feature Flag Management

| Field            | Value                                            |
|------------------|--------------------------------------------------|
| **Runbook ID**   | DP-005                                           |
| **Title**        | Feature Flag Management                          |
| **Author**       | Platform Engineering                             |
| **Last Updated** | 2026-03-08                                       |
| **Severity**     | Standard Change                                  |
| **Services**     | All Clario 360 platform services                 |
| **Namespace**    | clario360                                        |
| **Approvers**    | Engineering Lead (new flags), SRE (kill switch)  |
| **Est. Duration**| 5-15 minutes per operation                       |

---

## Summary

This runbook covers managing feature flags in the Clario 360 platform. Feature flags control the rollout of new features, enable gradual rollout strategies, allow per-tenant overrides, and provide an emergency kill switch mechanism. Feature flags are stored in the platform configuration and exposed via the API gateway's feature flag service.

### Feature Flag Architecture

- **Storage:** Feature flags are stored in Redis (for runtime evaluation) and PostgreSQL `platform_core` database (for persistence and audit trail).
- **Evaluation:** The `api-gateway` evaluates feature flags on each request and injects the flag state into the request context.
- **Configuration:** Flags can be managed via the admin API, ConfigMap, or direct Redis commands.
- **Scope:** Flags can be global, per-tenant, per-user, or percentage-based.

### Feature Flag States

| State      | Description                                           |
|------------|-------------------------------------------------------|
| `enabled`  | Feature is ON for all matching targets                |
| `disabled` | Feature is OFF for all targets                        |
| `rollout`  | Feature is ON for a percentage of targets             |
| `override` | Feature state is determined by per-tenant/user rules  |

---

## Prerequisites

- [ ] `kubectl` configured with production GKE cluster credentials
- [ ] Admin API access token with `feature-flags:manage` permission
- [ ] Redis CLI access (via kubectl exec into a Redis pod)

### Authenticate

```bash
gcloud container clusters get-credentials clario360-prod \
  --region us-central1 \
  --project clario360-prod
```

### Obtain Admin API Token

```bash
ADMIN_TOKEN=$(curl -s -X POST https://api.clario360.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@clario360.com","password":"<ADMIN_PASSWORD>"}' \
  | jq -r '.access_token')

echo "Token obtained: $(echo $ADMIN_TOKEN | cut -c1-20)..."
```

---

## Procedure

### Operation 1: List Current Feature Flags

#### Via Admin API

```bash
curl -s https://api.clario360.com/api/v1/admin/feature-flags \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq '.flags[] | {name: .name, state: .state, rollout_percentage: .rollout_percentage, description: .description}'
```

#### Via Redis (Direct)

```bash
kubectl exec -n clario360 deploy/redis -- redis-cli KEYS "feature_flag:*"
```

To get details on a specific flag:

```bash
kubectl exec -n clario360 deploy/redis -- redis-cli HGETALL "feature_flag:<FLAG_NAME>"
```

#### Via ConfigMap

```bash
kubectl get configmap feature-flags -n clario360 -o yaml
```

#### Via Database

```bash
kubectl run db-list-flags \
  --namespace=clario360 \
  --image=postgres:15 \
  --restart=Never \
  --rm -it \
  --env="PGPASSWORD=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.host}' | base64 -d) \
  -U $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.username}' | base64 -d) \
  -d platform_core \
  -c "SELECT name, state, rollout_percentage, description, created_at, updated_at FROM feature_flags ORDER BY name;"
```

---

### Operation 2: Enable/Disable a Feature Flag

#### Enable a Feature Flag (API)

```bash
curl -s -X PUT https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "enabled",
    "description": "Enabling <FLAG_NAME> for production release",
    "updated_by": "operator@clario360.com"
  }' | jq .
```

#### Disable a Feature Flag (API)

```bash
curl -s -X PUT https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "disabled",
    "description": "Disabling <FLAG_NAME> due to <reason>",
    "updated_by": "operator@clario360.com"
  }' | jq .
```

#### Enable/Disable via ConfigMap (Requires Pod Restart)

```bash
kubectl edit configmap feature-flags -n clario360
```

Or patch directly:

```bash
kubectl patch configmap feature-flags -n clario360 --type merge \
  -p "{\"data\":{\"<FLAG_NAME>\": \"enabled\"}}"
```

After ConfigMap change, restart the affected services to pick up the new config:

```bash
kubectl rollout restart deployment/api-gateway -n clario360
kubectl rollout status deployment/api-gateway -n clario360 --timeout=120s
```

#### Enable/Disable via Redis (Immediate, No Restart)

```bash
# Enable
kubectl exec -n clario360 deploy/redis -- redis-cli HSET "feature_flag:<FLAG_NAME>" "state" "enabled"

# Disable
kubectl exec -n clario360 deploy/redis -- redis-cli HSET "feature_flag:<FLAG_NAME>" "state" "disabled"
```

#### Verify the Change

```bash
curl -s https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq '{name: .name, state: .state, updated_at: .updated_at}'
```

---

### Operation 3: Gradual Rollout (Percentage-Based)

Set a feature flag to roll out to a percentage of users/tenants:

#### Start at 5%

```bash
curl -s -X PUT https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "rollout",
    "rollout_percentage": 5,
    "description": "Gradual rollout: starting at 5%",
    "updated_by": "operator@clario360.com"
  }' | jq .
```

#### Increase to 25%

```bash
curl -s -X PUT https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "rollout",
    "rollout_percentage": 25,
    "description": "Gradual rollout: increasing to 25%",
    "updated_by": "operator@clario360.com"
  }' | jq .
```

#### Increase to 50%

```bash
curl -s -X PUT https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "rollout",
    "rollout_percentage": 50,
    "description": "Gradual rollout: increasing to 50%",
    "updated_by": "operator@clario360.com"
  }' | jq .
```

#### Increase to 100% (Full Rollout)

```bash
curl -s -X PUT https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "enabled",
    "rollout_percentage": 100,
    "description": "Full rollout: enabling for all users",
    "updated_by": "operator@clario360.com"
  }' | jq .
```

#### Monitor Rollout Metrics

After each percentage increase, monitor for errors and user feedback for at least 15 minutes:

```bash
# Check error rate for the feature
curl -s "https://prometheus.clario360.internal/api/v1/query" \
  --data-urlencode "query=sum(rate(feature_flag_evaluation_total{flag=\"<FLAG_NAME>\",result=\"error\"}[5m]))" \
  | jq '.data.result[0].value[1]'

# Check how many evaluations are returning "true" (feature enabled)
curl -s "https://prometheus.clario360.internal/api/v1/query" \
  --data-urlencode "query=sum(rate(feature_flag_evaluation_total{flag=\"<FLAG_NAME>\",result=\"true\"}[5m])) / sum(rate(feature_flag_evaluation_total{flag=\"<FLAG_NAME>\"}[5m]))" \
  | jq '.data.result[0].value[1]'
# Should approximate the rollout percentage
```

#### Recommended Rollout Schedule

| Day   | Percentage | Duration Before Increase |
|-------|-----------|-------------------------|
| Day 1 | 5%        | 24 hours                |
| Day 2 | 25%       | 24 hours                |
| Day 3 | 50%       | 24 hours                |
| Day 4 | 75%       | 24 hours                |
| Day 5 | 100%      | Permanent               |

---

### Operation 4: Per-Tenant Feature Flag Override

Set a feature flag to be enabled or disabled for a specific tenant, regardless of the global state:

#### Enable for a Specific Tenant

```bash
curl -s -X PUT https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME>/overrides \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "<TENANT_ID>",
    "state": "enabled",
    "reason": "Beta tester tenant - early access to <feature>",
    "updated_by": "operator@clario360.com"
  }' | jq .
```

#### Disable for a Specific Tenant

```bash
curl -s -X PUT https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME>/overrides \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "<TENANT_ID>",
    "state": "disabled",
    "reason": "Tenant reported issues with <feature>",
    "updated_by": "operator@clario360.com"
  }' | jq .
```

#### List All Overrides for a Flag

```bash
curl -s https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME>/overrides \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq '.overrides[] | {tenant_id: .tenant_id, state: .state, reason: .reason}'
```

#### Remove a Tenant Override

```bash
curl -s -X DELETE https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME>/overrides/<TENANT_ID> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq .
```

#### Via Redis (Direct)

```bash
# Set override
kubectl exec -n clario360 deploy/redis -- redis-cli HSET "feature_flag:<FLAG_NAME>:overrides" "<TENANT_ID>" "enabled"

# Get override
kubectl exec -n clario360 deploy/redis -- redis-cli HGET "feature_flag:<FLAG_NAME>:overrides" "<TENANT_ID>"

# Remove override
kubectl exec -n clario360 deploy/redis -- redis-cli HDEL "feature_flag:<FLAG_NAME>:overrides" "<TENANT_ID>"

# List all overrides
kubectl exec -n clario360 deploy/redis -- redis-cli HGETALL "feature_flag:<FLAG_NAME>:overrides"
```

---

### Operation 5: Clean Up Old Feature Flags

Feature flags that have been fully rolled out (100% enabled) for more than 30 days should be cleaned up. This involves removing the flag evaluation from code and deleting the flag configuration.

#### Step 1: Identify Stale Flags

```bash
curl -s https://api.clario360.com/api/v1/admin/feature-flags \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq '.flags[] | select(.state == "enabled" and .rollout_percentage == 100) | select((.updated_at | fromdateiso8601) < (now - 2592000)) | {name: .name, state: .state, updated_at: .updated_at}'
```

Or via database:

```bash
kubectl run db-stale-flags \
  --namespace=clario360 \
  --image=postgres:15 \
  --restart=Never \
  --rm -it \
  --env="PGPASSWORD=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.host}' | base64 -d) \
  -U $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.username}' | base64 -d) \
  -d platform_core \
  -c "SELECT name, state, rollout_percentage, updated_at FROM feature_flags WHERE state = 'enabled' AND rollout_percentage = 100 AND updated_at < NOW() - INTERVAL '30 days' ORDER BY updated_at;"
```

#### Step 2: Remove Flag from Code

Search the codebase for references to the flag:

```bash
cd /Users/mac/clario360
grep -r "<FLAG_NAME>" --include="*.go" --include="*.ts" --include="*.tsx" -l
```

Remove all feature flag checks from the code, keeping only the "enabled" code path. This should be done as a normal code change through the standard PR process.

#### Step 3: Delete the Flag Configuration

After the code change has been deployed:

```bash
curl -s -X DELETE https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq .
```

Clean up Redis:

```bash
kubectl exec -n clario360 deploy/redis -- redis-cli DEL "feature_flag:<FLAG_NAME>"
kubectl exec -n clario360 deploy/redis -- redis-cli DEL "feature_flag:<FLAG_NAME>:overrides"
```

Clean up database:

```bash
kubectl run db-delete-flag \
  --namespace=clario360 \
  --image=postgres:15 \
  --restart=Never \
  --rm -it \
  --env="PGPASSWORD=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.host}' | base64 -d) \
  -U $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.username}' | base64 -d) \
  -d platform_core \
  -c "DELETE FROM feature_flags WHERE name = '<FLAG_NAME>';"
```

---

### Operation 6: Emergency Kill Switch

The kill switch immediately disables a feature that is causing production issues. This is designed for maximum speed -- use Redis directly for instant effect.

#### Step 1: Kill the Feature (Immediate)

```bash
# Disable via Redis (takes effect immediately, no restart needed)
kubectl exec -n clario360 deploy/redis -- redis-cli HSET "feature_flag:<FLAG_NAME>" "state" "disabled"
```

#### Step 2: Confirm the Kill

```bash
# Verify Redis state
kubectl exec -n clario360 deploy/redis -- redis-cli HGETALL "feature_flag:<FLAG_NAME>"

# Verify via API
curl -s https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq '{name: .name, state: .state}'
```

#### Step 3: Persist the Kill to Database

The Redis change is immediate but volatile. Persist it to the database:

```bash
curl -s -X PUT https://api.clario360.com/api/v1/admin/feature-flags/<FLAG_NAME> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "disabled",
    "description": "EMERGENCY KILL SWITCH: Disabled due to <reason>. Incident: INC-XXXX",
    "updated_by": "operator@clario360.com"
  }' | jq .
```

#### Step 4: Verify Impact

```bash
# Check that the feature is no longer being served
curl -s "https://prometheus.clario360.internal/api/v1/query" \
  --data-urlencode "query=sum(rate(feature_flag_evaluation_total{flag=\"<FLAG_NAME>\",result=\"true\"}[1m]))" \
  | jq '.data.result[0].value[1]'
# Expected: 0 or very close to 0

# Check error rates are dropping
curl -s "https://prometheus.clario360.internal/api/v1/query" \
  --data-urlencode "query=sum(rate(http_requests_total{namespace=\"clario360\",code=~\"5..\"}[1m]))" \
  | jq '.data.result[0].value[1]'
```

#### Step 5: Notify the Team

```bash
curl -X POST https://hooks.slack.com/services/TXXXXX/BXXXXX/XXXXXXX \
  -H "Content-Type: application/json" \
  -d "{\"text\":\"EMERGENCY: Feature flag <FLAG_NAME> has been KILLED in production. Reason: <reason>. Incident: INC-XXXX\"}"
```

---

## Creating a New Feature Flag

To create a new feature flag:

```bash
curl -s -X POST https://api.clario360.com/api/v1/admin/feature-flags \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "<FLAG_NAME>",
    "state": "disabled",
    "rollout_percentage": 0,
    "description": "<description of what this flag controls>",
    "owner": "<team or person>",
    "created_by": "operator@clario360.com",
    "metadata": {
      "jira_ticket": "CLARIO-XXXX",
      "expected_cleanup_date": "2026-06-01"
    }
  }' | jq .
```

### Flag Naming Convention

```
<domain>.<feature>.<variant>
```

Examples:
- `cyber.threat-intel.ai-enrichment`
- `workflow.parallel-execution.enabled`
- `dashboard.new-charts.enabled`
- `iam.passkey-auth.enabled`

---

## Verification

| Check                          | Command                                                           | Expected                          |
|-------------------------------|-------------------------------------------------------------------|-----------------------------------|
| Flag state correct            | `curl ... /api/v1/admin/feature-flags/<FLAG_NAME>`                | Shows expected state              |
| Redis state matches           | `kubectl exec ... redis-cli HGETALL "feature_flag:<FLAG_NAME>"`   | Matches API state                 |
| Database state matches        | `SELECT * FROM feature_flags WHERE name = '<FLAG_NAME>'`          | Matches API state                 |
| Evaluation metrics correct    | Prometheus `feature_flag_evaluation_total{flag="<FLAG_NAME>"}`    | Matches expected percentage       |
| No errors from flag evaluation| Application logs                                                  | No flag evaluation errors         |
| Overrides applied correctly   | `curl ... /api/v1/admin/feature-flags/<FLAG_NAME>/overrides`      | Shows expected overrides          |

---

## Troubleshooting

### Feature Flag Not Taking Effect

1. Check Redis connectivity:

```bash
kubectl exec -n clario360 deploy/redis -- redis-cli PING
# Expected: PONG
```

2. Check if the service is reading from Redis or ConfigMap:

```bash
kubectl logs -n clario360 deploy/api-gateway --tail=50 \
  | grep -i "feature.flag\|feature_flag"
```

3. Check for override conflicts:

```bash
kubectl exec -n clario360 deploy/redis -- redis-cli HGETALL "feature_flag:<FLAG_NAME>:overrides"
```

### Redis and Database Out of Sync

Sync Redis from the database (source of truth):

```bash
# Get the flag state from the database
kubectl run db-get-flag \
  --namespace=clario360 \
  --image=postgres:15 \
  --restart=Never \
  --rm -it \
  --env="PGPASSWORD=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.password}' | base64 -d)" \
  -- psql -h $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.host}' | base64 -d) \
  -U $(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.username}' | base64 -d) \
  -d platform_core \
  -c "SELECT name, state, rollout_percentage FROM feature_flags WHERE name = '<FLAG_NAME>';"

# Update Redis to match
kubectl exec -n clario360 deploy/redis -- redis-cli HSET "feature_flag:<FLAG_NAME>" "state" "<STATE_FROM_DB>" "rollout_percentage" "<PERCENTAGE_FROM_DB>"
```

---

## Related Links

- [DP-001: Deploy a New Version](./DP-001-new-release.md)
- [DP-002: Rollback to Previous Version](./DP-002-rollback.md)
- [DP-003: Emergency Hotfix Procedure](./DP-003-hotfix.md)
- [DP-004: Run Database Migrations](./DP-004-database-migration.md)
- Grafana Feature Flags Dashboard: `https://grafana.clario360.internal/d/feature-flags`
- Prometheus: `https://prometheus.clario360.internal`
