# OP-010: Admin User Operations

| Field | Value |
|-------|-------|
| **Runbook ID** | OP-010 |
| **Title** | Admin User Operations |
| **Frequency** | As needed |
| **Owner** | Platform Security Team |
| **Last Updated** | 2026-03-08 |
| **Estimated Duration** | 5–30 minutes per operation |
| **Risk Level** | Medium — incorrect changes can lock out users or grant excessive permissions |
| **Approval Required** | Yes — all admin user changes require a ticket and peer review |
| **Maintenance Window** | Not required — can be performed anytime |

## Summary

This runbook covers administrative user operations for the Clario 360 platform:

1. Create new admin user
2. Disable/lock user account
3. Reset user password
4. Assign/revoke roles and permissions
5. Audit user access history
6. Manage service accounts
7. Emergency: revoke all sessions for a user

All user operations go through the IAM service API. Direct database modifications are forbidden except in emergency scenarios (documented in Step 7).

### API Endpoints

| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create user | POST | `/v1/users` |
| Get user | GET | `/v1/users/{id}` |
| Update user | PATCH | `/v1/users/{id}` |
| Disable user | POST | `/v1/users/{id}/disable` |
| Enable user | POST | `/v1/users/{id}/enable` |
| Reset password | POST | `/v1/users/{id}/reset-password` |
| List roles | GET | `/v1/roles` |
| Assign roles | POST | `/v1/users/{id}/roles` |
| Revoke roles | DELETE | `/v1/users/{id}/roles/{roleId}` |
| List sessions | GET | `/v1/users/{id}/sessions` |
| Revoke sessions | DELETE | `/v1/users/{id}/sessions` |
| Audit log | GET | `/v1/audit-logs?actor_id={id}` |

## Prerequisites

```bash
export NAMESPACE=clario360
export API_URL=https://api.clario360.io
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export VAULT_ADDR=https://vault.clario360.io

# Authenticate as platform admin
# Option 1: Use personal admin credentials
ADMIN_TOKEN=$(curl -s -X POST "$API_URL/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@clario360.io", "password": "'"$ADMIN_PASSWORD"'"}' | jq -r '.access_token')

# Option 2: Use ops service account token from Vault
ADMIN_TOKEN=$(vault kv get -field=token secret/clario360/api-keys/service/ops-admin)

echo "Auth token obtained: ${ADMIN_TOKEN:0:20}..."
```

Verify admin access:

```bash
curl -s "$API_URL/v1/users/me" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '{id, email, roles}'
```

---

## Step 1: Create New Admin User

### 1a. Create the user account

```bash
NEW_USER_EMAIL="jane.doe@clario360.io"
NEW_USER_FIRST="Jane"
NEW_USER_LAST="Doe"
TENANT_ID="<tenant-uuid>"

curl -s -X POST "$API_URL/v1/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'"$NEW_USER_EMAIL"'",
    "first_name": "'"$NEW_USER_FIRST"'",
    "last_name": "'"$NEW_USER_LAST"'",
    "tenant_id": "'"$TENANT_ID"'",
    "require_password_change": true,
    "send_welcome_email": true
  }' | jq .
```

Save the returned user ID:

```bash
NEW_USER_ID=$(curl -s -X POST "$API_URL/v1/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'"$NEW_USER_EMAIL"'",
    "first_name": "'"$NEW_USER_FIRST"'",
    "last_name": "'"$NEW_USER_LAST"'",
    "tenant_id": "'"$TENANT_ID"'",
    "require_password_change": true,
    "send_welcome_email": true
  }' | jq -r '.id')

echo "Created user: $NEW_USER_ID"
```

### 1b. Set initial password

```bash
INITIAL_PASSWORD=$(openssl rand -base64 16)

curl -s -X POST "$API_URL/v1/users/$NEW_USER_ID/reset-password" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "new_password": "'"$INITIAL_PASSWORD"'",
    "require_change": true
  }' | jq .

echo "Initial password: $INITIAL_PASSWORD"
echo "IMPORTANT: Communicate this to the user via a secure channel (e.g., encrypted email, 1Password share)"
```

### 1c. Assign roles to the new user

```bash
# List available roles
curl -s "$API_URL/v1/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {id, name, description}'

# Assign admin role (replace ROLE_ID with actual role UUID)
ROLE_ID="<role-uuid>"

curl -s -X POST "$API_URL/v1/users/$NEW_USER_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_ids": ["'"$ROLE_ID"'"]
  }' | jq .
```

### 1d. Verify the new user

```bash
curl -s "$API_URL/v1/users/$NEW_USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '{id, email, first_name, last_name, status, roles, created_at}'
```

---

## Step 2: Disable/Lock User Account

### 2a. Disable a user account

```bash
USER_ID="<user-uuid>"

curl -s -X POST "$API_URL/v1/users/$USER_ID/disable" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Account disabled per ticket SEC-1234"
  }' | jq .
```

### 2b. Verify account is disabled

```bash
curl -s "$API_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '{id, email, status, disabled_at, disabled_reason}'
```

### 2c. Revoke all active sessions for the disabled user

```bash
curl -s -X DELETE "$API_URL/v1/users/$USER_ID/sessions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
```

### 2d. Re-enable a previously disabled account

```bash
curl -s -X POST "$API_URL/v1/users/$USER_ID/enable" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Account re-enabled per ticket SEC-1235"
  }' | jq .
```

### 2e. Lock account after failed login attempts (verify lock status)

```bash
# Check if account is locked due to brute-force protection
curl -s "$API_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '{id, email, status, locked_until, failed_login_attempts}'

# Unlock a locked account
curl -s -X POST "$API_URL/v1/users/$USER_ID/unlock" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Manual unlock per user request, identity verified"
  }' | jq .
```

---

## Step 3: Reset User Password

### 3a. Admin-initiated password reset

```bash
USER_ID="<user-uuid>"

# Generate a temporary password
TEMP_PASSWORD=$(openssl rand -base64 16)

curl -s -X POST "$API_URL/v1/users/$USER_ID/reset-password" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "new_password": "'"$TEMP_PASSWORD"'",
    "require_change": true
  }' | jq .

echo "Temporary password: $TEMP_PASSWORD"
echo "User must change password on next login."
```

### 3b. Send password reset email (self-service)

```bash
USER_EMAIL="user@example.com"

curl -s -X POST "$API_URL/v1/auth/forgot-password" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'"$USER_EMAIL"'"
  }' | jq .

echo "Password reset email sent to $USER_EMAIL"
```

### 3c. Verify password was changed

```bash
curl -s "$API_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '{id, email, password_changed_at, require_password_change}'
```

### 3d. Force password change for all users in a tenant (security incident)

```bash
TENANT_ID="<tenant-uuid>"

# Get all users in the tenant
USERS=$(curl -s "$API_URL/v1/users?tenant_id=$TENANT_ID&limit=1000" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[].id')

for user_id in $USERS; do
  curl -s -X PATCH "$API_URL/v1/users/$user_id" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"require_password_change": true}' > /dev/null
  echo "Flagged password change for user: $user_id"
done

echo "All users in tenant $TENANT_ID must change password on next login."
```

---

## Step 4: Assign/Revoke Roles and Permissions

### 4a. List all available roles

```bash
curl -s "$API_URL/v1/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {id, name, description, permissions: (.permissions | length)}'
```

### 4b. View a specific role's permissions

```bash
ROLE_ID="<role-uuid>"

curl -s "$API_URL/v1/roles/$ROLE_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '{name, description, permissions}'
```

### 4c. View user's current roles

```bash
USER_ID="<user-uuid>"

curl -s "$API_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '{email, roles: [.roles[] | {id, name}]}'
```

### 4d. Assign roles to a user

```bash
USER_ID="<user-uuid>"
ROLE_IDS='["<role-uuid-1>", "<role-uuid-2>"]'

curl -s -X POST "$API_URL/v1/users/$USER_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_ids": '"$ROLE_IDS"'
  }' | jq .
```

### 4e. Revoke a role from a user

```bash
USER_ID="<user-uuid>"
ROLE_ID="<role-uuid>"

curl -s -X DELETE "$API_URL/v1/users/$USER_ID/roles/$ROLE_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
```

### 4f. Verify role assignment

```bash
curl -s "$API_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.roles[] | {id, name}'
```

### 4g. Bulk role assignment (e.g., onboarding a department)

```bash
ROLE_ID="<role-uuid>"
USER_IDS=("user-uuid-1" "user-uuid-2" "user-uuid-3")

for user_id in "${USER_IDS[@]}"; do
  curl -s -X POST "$API_URL/v1/users/$user_id/roles" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"role_ids": ["'"$ROLE_ID"'"]}' > /dev/null
  echo "Assigned role $ROLE_ID to user $user_id"
done
```

---

## Step 5: Audit User Access History

### 5a. View audit logs for a specific user

```bash
USER_ID="<user-uuid>"

curl -s "$API_URL/v1/audit-logs?actor_id=$USER_ID&limit=50&sort=-created_at" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {
    id,
    action,
    resource_type,
    resource_id,
    created_at,
    ip_address,
    user_agent: (.user_agent // "N/A" | .[0:50])
  }'
```

### 5b. View login history for a user

```bash
curl -s "$API_URL/v1/audit-logs?actor_id=$USER_ID&action=auth.login&limit=20&sort=-created_at" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {
    created_at,
    action,
    ip_address,
    success: .metadata.success,
    user_agent: (.user_agent // "N/A" | .[0:60])
  }'
```

### 5c. View failed login attempts

```bash
curl -s "$API_URL/v1/audit-logs?actor_id=$USER_ID&action=auth.login_failed&limit=20&sort=-created_at" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {
    created_at,
    ip_address,
    reason: .metadata.reason
  }'
```

### 5d. View permission changes for a user

```bash
curl -s "$API_URL/v1/audit-logs?resource_type=user&resource_id=$USER_ID&action=role.assigned,role.revoked&limit=20&sort=-created_at" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {
    created_at,
    action,
    actor_email: .actor_email,
    old_value,
    new_value
  }'
```

### 5e. Query audit logs directly in PostgreSQL (for complex queries)

```bash
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  SELECT
    al.id,
    al.action,
    al.actor_id,
    u.email AS actor_email,
    al.resource_type,
    al.resource_id,
    al.ip_address,
    al.created_at
  FROM audit_logs al
  LEFT JOIN users u ON al.actor_id = u.id
  WHERE al.actor_id = '<user-uuid>'
  ORDER BY al.created_at DESC
  LIMIT 50;
"
```

### 5f. Check active sessions for a user

```bash
curl -s "$API_URL/v1/users/$USER_ID/sessions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {
    session_id: .id,
    created_at,
    last_active_at,
    ip_address,
    user_agent: (.user_agent // "N/A" | .[0:50]),
    expires_at
  }'
```

---

## Step 6: Manage Service Accounts

Service accounts are non-human accounts used for inter-service communication, CI/CD pipelines, and external integrations.

### 6a. List all service accounts

```bash
curl -s "$API_URL/v1/users?type=service_account&limit=100" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {id, email, name: .first_name, status, created_at, last_login_at}'
```

### 6b. Create a new service account

```bash
SA_NAME="ci-deploy-bot"
SA_DESCRIPTION="CI/CD deployment pipeline service account"

SA_RESPONSE=$(curl -s -X POST "$API_URL/v1/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'"$SA_NAME"'@service.clario360.io",
    "first_name": "'"$SA_NAME"'",
    "last_name": "Service Account",
    "type": "service_account",
    "description": "'"$SA_DESCRIPTION"'",
    "send_welcome_email": false
  }')

SA_ID=$(echo $SA_RESPONSE | jq -r '.id')
echo "Created service account: $SA_ID"
echo "$SA_RESPONSE" | jq .
```

### 6c. Generate API key for service account

```bash
SA_API_KEY=$(curl -s -X POST "$API_URL/v1/users/$SA_ID/api-keys" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "'"$SA_NAME"'-key",
    "expires_at": "'"$(date -u -d '+365 days' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v+365d +%Y-%m-%dT%H:%M:%SZ)"'"
  }' | jq -r '.api_key')

echo "API Key: $SA_API_KEY"
echo "IMPORTANT: Store this key securely. It cannot be retrieved again."

# Store in Vault
vault kv put secret/clario360/service-accounts/$SA_NAME \
  api_key="$SA_API_KEY" \
  user_id="$SA_ID" \
  created_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

### 6d. Assign minimal roles to service account

```bash
# Service accounts should have the minimum permissions required
DEPLOY_ROLE_ID="<deploy-role-uuid>"

curl -s -X POST "$API_URL/v1/users/$SA_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role_ids": ["'"$DEPLOY_ROLE_ID"'"]}' | jq .
```

### 6e. Rotate service account API key

```bash
SA_ID="<service-account-uuid>"

# List current API keys
curl -s "$API_URL/v1/users/$SA_ID/api-keys" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {id, name, created_at, expires_at}'

# Generate new key
NEW_SA_KEY=$(curl -s -X POST "$API_URL/v1/users/$SA_ID/api-keys" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rotated-key-'"$(date +%Y%m%d)"'",
    "expires_at": "'"$(date -u -d '+365 days' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v+365d +%Y-%m-%dT%H:%M:%SZ)"'"
  }' | jq -r '.api_key')

# Update Vault with new key
SA_NAME=$(curl -s "$API_URL/v1/users/$SA_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.first_name')

vault kv put secret/clario360/service-accounts/$SA_NAME \
  api_key="$NEW_SA_KEY" \
  user_id="$SA_ID" \
  rotated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "New API key stored in Vault. Update all consumers, then delete the old key."

# Delete old key (after confirming new key works)
OLD_KEY_ID="<old-key-uuid>"
curl -s -X DELETE "$API_URL/v1/users/$SA_ID/api-keys/$OLD_KEY_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
```

### 6f. Disable a service account

```bash
curl -s -X POST "$API_URL/v1/users/$SA_ID/disable" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Service account decommissioned per ticket OPS-5678"}' | jq .
```

---

## Step 7: Emergency — Revoke All Sessions for a User

Use this procedure when a user account may be compromised. This immediately invalidates all active sessions, API keys, and JWT tokens.

### 7a. Identify the compromised user

```bash
# By email
USER_ID=$(curl -s "$API_URL/v1/users?email=compromised@example.com" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

echo "User ID: $USER_ID"
```

### 7b. Revoke all sessions via API

```bash
curl -s -X DELETE "$API_URL/v1/users/$USER_ID/sessions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

echo "All sessions revoked for user $USER_ID"
```

### 7c. Disable the account

```bash
curl -s -X POST "$API_URL/v1/users/$USER_ID/disable" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Emergency: suspected account compromise — incident INC-9999"}' | jq .
```

### 7d. Revoke API keys

```bash
# List and revoke all API keys for the user
API_KEYS=$(curl -s "$API_URL/v1/users/$USER_ID/api-keys" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[].id')

for key_id in $API_KEYS; do
  curl -s -X DELETE "$API_URL/v1/users/$USER_ID/api-keys/$key_id" \
    -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
  echo "Revoked API key: $key_id"
done
```

### 7e. Invalidate cached sessions in Redis

```bash
REDIS_PASSWORD=$(vault kv get -field=password secret/clario360/redis/password)

# Delete all session keys for this user from Redis
kubectl -n $NAMESPACE exec deployment/redis -- redis-cli \
  -a "$REDIS_PASSWORD" \
  --scan --pattern "session:$USER_ID:*" | while read key; do
  kubectl -n $NAMESPACE exec deployment/redis -- redis-cli \
    -a "$REDIS_PASSWORD" \
    DEL "$key"
  echo "Deleted Redis session: $key"
done

# Also invalidate any cached tokens
kubectl -n $NAMESPACE exec deployment/redis -- redis-cli \
  -a "$REDIS_PASSWORD" \
  --scan --pattern "token:*:$USER_ID" | while read key; do
  kubectl -n $NAMESPACE exec deployment/redis -- redis-cli \
    -a "$REDIS_PASSWORD" \
    DEL "$key"
  echo "Deleted Redis token: $key"
done
```

### 7f. Direct database invalidation (if API is unavailable)

> **Use only if the IAM service API is down.**

```bash
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  -- Disable the user account
  UPDATE users
  SET status = 'disabled',
      disabled_at = NOW(),
      disabled_reason = 'Emergency: suspected compromise — incident INC-9999',
      updated_at = NOW()
  WHERE id = '$USER_ID';

  -- Invalidate all sessions
  DELETE FROM sessions WHERE user_id = '$USER_ID';

  -- Invalidate all refresh tokens
  DELETE FROM refresh_tokens WHERE user_id = '$USER_ID';

  -- Revoke all API keys
  UPDATE api_keys
  SET revoked_at = NOW(),
      revoked_reason = 'Emergency revocation — incident INC-9999'
  WHERE user_id = '$USER_ID';

  -- Log the emergency action
  INSERT INTO audit_logs (action, actor_id, resource_type, resource_id, metadata, created_at)
  VALUES (
    'emergency.session_revocation',
    '00000000-0000-0000-0000-000000000000',
    'user',
    '$USER_ID',
    '{\"reason\": \"Direct DB intervention — API unavailable\", \"incident\": \"INC-9999\"}'::jsonb,
    NOW()
  );
"
```

### 7g. Verify all sessions are revoked

```bash
# Check via API
curl -s "$API_URL/v1/users/$USER_ID/sessions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '. | length'
# Expected: 0

# Check Redis
kubectl -n $NAMESPACE exec deployment/redis -- redis-cli \
  -a "$REDIS_PASSWORD" \
  --scan --pattern "*$USER_ID*" | wc -l
# Expected: 0

# Check database
kubectl -n $NAMESPACE exec deployment/postgresql -- psql -U $PG_USER -d platform_core -c "
  SELECT count(*) AS active_sessions FROM sessions WHERE user_id = '$USER_ID';
  SELECT count(*) AS active_tokens FROM refresh_tokens WHERE user_id = '$USER_ID';
  SELECT status, disabled_at FROM users WHERE id = '$USER_ID';
"
```

### 7h. Notify security team

```bash
curl -X POST https://api.clario360.io/v1/notifications/broadcast \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "security-incidents",
    "message": "EMERGENCY: All sessions revoked for user '"$USER_ID"'. Account disabled. Incident: INC-9999. Investigate login history and data access immediately.",
    "severity": "critical"
  }'
```

---

## Verification

After any user management operation:

```bash
# 1. Verify user state
USER_ID="<user-uuid>"

curl -s "$API_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '{
    id,
    email,
    status,
    roles: [.roles[].name],
    last_login_at,
    password_changed_at,
    disabled_at,
    require_password_change
  }'

# 2. Check audit log for the operation
curl -s "$API_URL/v1/audit-logs?resource_type=user&resource_id=$USER_ID&limit=5&sort=-created_at" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | {action, actor_email, created_at}'

# 3. Verify IAM service is healthy
kubectl -n $NAMESPACE exec deployment/iam-service -- wget -qO- http://localhost:8080/healthz
```

---

## Related Links

- [OP-006: Secret Rotation](OP-006-secret-rotation.md)
- [IR-008: Security Breach](../incident-response/IR-008-security-breach.md)
- [TS-004: Auth Failures](../troubleshooting/TS-004-auth-failures.md)
- [Grafana — Security Overview](https://grafana.clario360.io/d/security-overview)
- [Grafana — Service Health](https://grafana.clario360.io/d/service-health)
