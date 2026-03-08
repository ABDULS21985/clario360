# TS-004: Authentication Issue Debugging

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | TS-004                                     |
| **Title**        | Authentication Issue Debugging             |
| **Severity**     | P1 - Critical                              |
| **Services**     | iam-service, api-gateway                   |
| **Last Updated** | 2026-03-08                                 |
| **Author**       | Platform Engineering                       |
| **Review Cycle** | Quarterly                                  |

---

## Summary

This runbook covers the investigation and resolution of authentication and authorization failures in the Clario 360 platform. The platform uses RS256 JWT tokens issued by the iam-service, with signing keys managed by HashiCorp Vault. Access tokens are held in-memory on the frontend, and refresh tokens are stored in httpOnly cookies managed by the BFF (Backend for Frontend) layer. Failures may arise from expired or invalid JWT tokens, IAM service outages, Vault key unavailability, CORS/cookie misconfigurations, or rate limiting on auth endpoints.

---

## Symptoms

- Users cannot log in: login form returns 401 or 403 errors.
- Users are unexpectedly logged out or sessions expire prematurely.
- API requests return `401 Unauthorized` despite the user being authenticated.
- API requests return `403 Forbidden` for users who should have the required permissions.
- Frontend shows "Session expired" or "Please log in again" messages repeatedly.
- IAM service logs show JWT verification failures.
- API gateway logs show token validation errors.
- Vault audit logs show access denied for signing key reads.

---

## Diagnosis Steps

### Step 1: Check JWT Token Validity

```bash
# Extract a JWT token from a failing request (from browser DevTools, logs, or test)
# Decode the JWT header and payload (without verification) using base64
# Replace <token> with the actual JWT string

# Decode the header
echo "<token>" | cut -d'.' -f1 | base64 -d 2>/dev/null | python3 -m json.tool

# Decode the payload
echo "<token>" | cut -d'.' -f1,2 | cut -d'.' -f2 | base64 -d 2>/dev/null | python3 -m json.tool
```

Check the decoded payload for:
- `exp` (expiration): Is the token expired? Compare with current Unix timestamp.
- `iat` (issued at): When was the token issued?
- `iss` (issuer): Should be the iam-service URL.
- `sub` (subject): The user ID.
- `tenant_id`: The tenant the token is scoped to.
- `permissions`: Array of permission strings.

```bash
# Check current Unix timestamp for comparison
date +%s

# Convert a Unix timestamp to human-readable
date -d @<timestamp> 2>/dev/null || date -r <timestamp>
```

```bash
# Verify the token against the iam-service from inside the cluster
kubectl -n clario360 run curl-test --rm -it --image=curlimages/curl -- \
  curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer <token>" \
  http://iam-service.clario360.svc.cluster.local:8080/api/v1/auth/verify
```

### Step 2: Check IAM Service Health and Logs

```bash
# Check IAM service pod status
kubectl -n clario360 get pods -l app=iam-service -o wide

# Check IAM service health endpoints
kubectl -n clario360 exec deploy/iam-service -- curl -s http://localhost:8080/healthz
kubectl -n clario360 exec deploy/iam-service -- curl -s http://localhost:8080/readyz

# Check IAM service logs for authentication errors
kubectl -n clario360 logs deploy/iam-service --tail=500 | grep -i -E 'auth|token|jwt|error|fail|denied|invalid|expired'

# Check for recent restarts or crashloops
kubectl -n clario360 describe deploy/iam-service | grep -A 5 'Conditions'
kubectl -n clario360 get events --sort-by='.lastTimestamp' | grep iam-service | tail -20

# Check IAM service metrics
kubectl -n clario360 port-forward svc/iam-service 8081:8080
curl -s http://localhost:8081/metrics | grep -E 'auth_requests|token_issued|token_verified|token_rejected'
```

```bash
# Check api-gateway logs for token validation errors
kubectl -n clario360 logs deploy/api-gateway --tail=500 | grep -i -E 'jwt|token|auth|401|403|unauthorized|forbidden'
```

### Step 3: Check Vault for Signing Key Availability

```bash
# Check Vault pod status
kubectl -n clario360 get pods -l app=vault -o wide

# Check Vault seal status
kubectl -n clario360 exec -it vault-0 -- vault status

# Check if the JWT signing key is accessible
kubectl -n clario360 exec -it vault-0 -- vault kv get secret/clario360/jwt/signing-key

# Check Vault audit logs for access denied
kubectl -n clario360 logs vault-0 --tail=200 | grep -i -E 'denied|error|permission|policy'

# Verify the Vault policy allows iam-service access
kubectl -n clario360 exec -it vault-0 -- vault policy read iam-service

# Check Vault token used by iam-service
kubectl -n clario360 get secret iam-service-vault-token -o jsonpath='{.data.token}' | base64 -d

# Test Vault connectivity from iam-service pod
kubectl -n clario360 exec deploy/iam-service -- curl -s http://vault.clario360.svc.cluster.local:8200/v1/sys/health
```

### Step 4: Check CORS and Cookie Settings

```bash
# Check the api-gateway / ingress CORS configuration
kubectl -n clario360 get ingress -o yaml | grep -A 20 'annotations'

# Check api-gateway CORS configuration
kubectl -n clario360 get configmap api-gateway-config -o yaml | grep -i -E 'cors|origin|cookie|samesite'

# Test CORS preflight from outside the cluster
kubectl -n clario360 port-forward svc/api-gateway 8080:8080

curl -v -X OPTIONS http://localhost:8080/api/v1/auth/login \
  -H "Origin: https://app.clario360.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type,Authorization"

# Verify the response includes correct CORS headers:
# Access-Control-Allow-Origin: https://app.clario360.com
# Access-Control-Allow-Credentials: true
# Access-Control-Allow-Headers: Content-Type,Authorization
# Access-Control-Allow-Methods: GET,POST,PUT,DELETE,OPTIONS
```

```bash
# Check cookie settings on the BFF auth endpoints
curl -v -X POST http://localhost:8080/api/auth/session \
  -H "Content-Type: application/json" \
  -d '{"username":"test@example.com","password":"test"}' 2>&1 | grep -i set-cookie

# Verify Set-Cookie attributes:
# - HttpOnly: true (prevents XS access)
# - Secure: true (HTTPS only)
# - SameSite: Strict or Lax
# - Path: /
# - Domain: correct domain
```

```bash
# Check ingress TLS configuration (cookies with Secure flag require HTTPS)
kubectl -n clario360 get ingress -o yaml | grep -A 10 'tls'
```

### Step 5: Check Rate Limiting on Auth Endpoints

```bash
# Check api-gateway rate limit configuration
kubectl -n clario360 get configmap api-gateway-config -o yaml | grep -i -E 'rate|limit|throttle'

# Check rate limit metrics
kubectl -n clario360 port-forward svc/api-gateway 8080:8080
curl -s http://localhost:8080/metrics | grep -E 'rate_limit|throttle|rejected'

# Check Redis for rate limit keys
kubectl -n clario360 exec -it deploy/redis -- redis-cli KEYS 'ratelimit:*auth*'
kubectl -n clario360 exec -it deploy/redis -- redis-cli KEYS 'ratelimit:*/login*'

# Check current rate limit counters for a specific IP or tenant
kubectl -n clario360 exec -it deploy/redis -- redis-cli GET 'ratelimit:auth:login:<ip-or-tenant>'

# Check TTL on rate limit keys (when will they reset)
kubectl -n clario360 exec -it deploy/redis -- redis-cli TTL 'ratelimit:auth:login:<ip-or-tenant>'
```

```bash
# Test if auth endpoints are being rate limited
for i in $(seq 1 20); do
  echo "Request $i: $(curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8080/api/v1/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"username":"test@example.com","password":"test"}')"
done
# If you see 429 responses, rate limiting is active
```

### Step 6: Check IAM Service Database

```bash
# Connect to the platform_core database
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core
```

```sql
-- Check if the user account is locked or disabled
SELECT id, email, status, locked_at, failed_login_attempts, last_login_at
FROM users
WHERE email = '<user-email>';

-- Check active sessions for the user
SELECT id, user_id, created_at, expires_at, revoked
FROM sessions
WHERE user_id = '<user-id>'
ORDER BY created_at DESC
LIMIT 10;

-- Check if refresh tokens exist and are valid
SELECT id, user_id, created_at, expires_at, revoked
FROM refresh_tokens
WHERE user_id = '<user-id>'
ORDER BY created_at DESC
LIMIT 10;

-- Check for signing key records
SELECT id, algorithm, created_at, expires_at, is_active
FROM signing_keys
ORDER BY created_at DESC
LIMIT 5;
```

---

## Resolution Steps

### Resolution: Refresh Token / Clear User Session

```bash
# If the user's access token is expired but refresh token is valid,
# the frontend should automatically refresh. If it doesn't, clear the session:

# Revoke all sessions for a specific user
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core -c \
  "UPDATE sessions SET revoked = true WHERE user_id = '<user-id>';"

kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core -c \
  "UPDATE refresh_tokens SET revoked = true WHERE user_id = '<user-id>';"

# Clear the user's cached auth data from Redis
kubectl -n clario360 exec -it deploy/redis -- redis-cli DEL "session:<user-id>"
kubectl -n clario360 exec -it deploy/redis -- redis-cli DEL "auth:user:<user-id>"

# The user will need to log in again
```

### Resolution: Unlock User Account

```bash
kubectl -n clario360 exec -it deploy/postgresql -- psql -U clario360 -d platform_core -c \
  "UPDATE users SET status = 'active', locked_at = NULL, failed_login_attempts = 0 WHERE email = '<user-email>';"
```

### Resolution: Rotate JWT Signing Key

Use this when the Vault signing key is compromised or unavailable.

```bash
# Generate a new RSA key pair
kubectl -n clario360 run keygen --rm -it --image=alpine -- sh -c '
  apk add --no-cache openssl &&
  openssl genrsa -out /tmp/private.pem 2048 &&
  openssl rsa -in /tmp/private.pem -pubout -out /tmp/public.pem &&
  echo "=== PRIVATE KEY ===" &&
  cat /tmp/private.pem &&
  echo "=== PUBLIC KEY ===" &&
  cat /tmp/public.pem
'

# Store the new key pair in Vault
kubectl -n clario360 exec -it vault-0 -- vault kv put secret/clario360/jwt/signing-key \
  private_key=@/tmp/private.pem \
  public_key=@/tmp/public.pem \
  algorithm=RS256 \
  rotated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Restart IAM service to pick up the new key
kubectl -n clario360 rollout restart deploy/iam-service
kubectl -n clario360 rollout status deploy/iam-service

# Restart api-gateway to pick up the new public key for verification
kubectl -n clario360 rollout restart deploy/api-gateway
kubectl -n clario360 rollout status deploy/api-gateway

# IMPORTANT: All existing tokens signed with the old key will become invalid.
# All users will need to re-authenticate.
```

### Resolution: Clear Auth Cache

```bash
# Clear all auth-related keys from Redis
kubectl -n clario360 exec -it deploy/redis -- redis-cli EVAL \
  "local keys = redis.call('keys', ARGV[1]) for i=1,#keys do redis.call('del', keys[i]) end return #keys" \
  0 'auth:*'

kubectl -n clario360 exec -it deploy/redis -- redis-cli EVAL \
  "local keys = redis.call('keys', ARGV[1]) for i=1,#keys do redis.call('del', keys[i]) end return #keys" \
  0 'session:*'

kubectl -n clario360 exec -it deploy/redis -- redis-cli EVAL \
  "local keys = redis.call('keys', ARGV[1]) for i=1,#keys do redis.call('del', keys[i]) end return #keys" \
  0 'permissions:*'

# Restart IAM service to reinitialize its caches
kubectl -n clario360 rollout restart deploy/iam-service
kubectl -n clario360 rollout status deploy/iam-service
```

### Resolution: Fix CORS Configuration

```bash
# Update the api-gateway CORS configuration
kubectl -n clario360 patch configmap api-gateway-config --type='merge' -p='{
  "data": {
    "CORS_ALLOWED_ORIGINS": "https://app.clario360.com,https://admin.clario360.com",
    "CORS_ALLOWED_METHODS": "GET,POST,PUT,DELETE,PATCH,OPTIONS",
    "CORS_ALLOWED_HEADERS": "Content-Type,Authorization,X-Request-ID,X-Tenant-ID",
    "CORS_ALLOW_CREDENTIALS": "true",
    "CORS_MAX_AGE": "3600"
  }
}'

# Restart api-gateway to apply
kubectl -n clario360 rollout restart deploy/api-gateway
kubectl -n clario360 rollout status deploy/api-gateway
```

```bash
# If the issue is with ingress annotations, update them
kubectl -n clario360 annotate ingress clario360-ingress --overwrite \
  nginx.ingress.kubernetes.io/cors-allow-origin="https://app.clario360.com,https://admin.clario360.com" \
  nginx.ingress.kubernetes.io/cors-allow-methods="GET,POST,PUT,DELETE,PATCH,OPTIONS" \
  nginx.ingress.kubernetes.io/cors-allow-headers="Content-Type,Authorization,X-Request-ID,X-Tenant-ID" \
  nginx.ingress.kubernetes.io/cors-allow-credentials="true" \
  nginx.ingress.kubernetes.io/enable-cors="true"
```

### Resolution: Clear Rate Limits

```bash
# Remove rate limit keys for a specific IP/tenant
kubectl -n clario360 exec -it deploy/redis -- redis-cli DEL 'ratelimit:auth:login:<ip-or-tenant>'

# Remove all auth-related rate limit keys
kubectl -n clario360 exec -it deploy/redis -- redis-cli EVAL \
  "local keys = redis.call('keys', ARGV[1]) for i=1,#keys do redis.call('del', keys[i]) end return #keys" \
  0 'ratelimit:*auth*'

# If rate limits are too aggressive, update the configuration
kubectl -n clario360 patch configmap api-gateway-config --type='merge' -p='{
  "data": {
    "AUTH_RATE_LIMIT_PER_MINUTE": "30",
    "AUTH_RATE_LIMIT_BURST": "10"
  }
}'

kubectl -n clario360 rollout restart deploy/api-gateway
kubectl -n clario360 rollout status deploy/api-gateway
```

### Resolution: Fix Vault Connectivity

```bash
# If Vault is sealed, unseal it
kubectl -n clario360 exec -it vault-0 -- vault operator unseal <unseal-key-1>
kubectl -n clario360 exec -it vault-0 -- vault operator unseal <unseal-key-2>
kubectl -n clario360 exec -it vault-0 -- vault operator unseal <unseal-key-3>

# Verify Vault is unsealed
kubectl -n clario360 exec -it vault-0 -- vault status

# If the iam-service Vault token has expired, renew it
kubectl -n clario360 exec -it vault-0 -- vault token renew <iam-service-token>

# Or create a new token with the iam-service policy
kubectl -n clario360 exec -it vault-0 -- vault token create \
  -policy=iam-service \
  -period=768h \
  -display-name=iam-service

# Update the iam-service secret with the new token
kubectl -n clario360 create secret generic iam-service-vault-token \
  --from-literal=token=<new-token> \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart iam-service to use the new token
kubectl -n clario360 rollout restart deploy/iam-service
kubectl -n clario360 rollout status deploy/iam-service
```

---

## Verification

After applying the resolution, verify the fix:

```bash
# 1. Check IAM service health
kubectl -n clario360 exec deploy/iam-service -- curl -s http://localhost:8080/healthz
kubectl -n clario360 exec deploy/iam-service -- curl -s http://localhost:8080/readyz

# 2. Check api-gateway health
kubectl -n clario360 exec deploy/api-gateway -- curl -s http://localhost:8080/healthz

# 3. Test authentication end-to-end from inside the cluster
kubectl -n clario360 run auth-test --rm -it --image=curlimages/curl -- sh -c '
  # Step 1: Login
  RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    http://iam-service.clario360.svc.cluster.local:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"admin@clario360.com\",\"password\":\"<test-password>\"}")
  HTTP_CODE=$(echo "$RESPONSE" | tail -1)
  BODY=$(echo "$RESPONSE" | head -1)
  echo "Login HTTP Code: $HTTP_CODE"
  echo "Login Response: $BODY"

  # Step 2: Extract token and verify
  TOKEN=$(echo "$BODY" | grep -o "\"access_token\":\"[^\"]*\"" | cut -d\" -f4)
  if [ -n "$TOKEN" ]; then
    echo "Token obtained, verifying..."
    curl -s -o /dev/null -w "Verify HTTP Code: %{http_code}\n" \
      -H "Authorization: Bearer $TOKEN" \
      http://api-gateway.clario360.svc.cluster.local:8080/api/v1/users/me
  fi
'

# 4. Verify CORS headers
kubectl -n clario360 port-forward svc/api-gateway 8080:8080 &
curl -s -D - -o /dev/null -X OPTIONS http://localhost:8080/api/v1/auth/login \
  -H "Origin: https://app.clario360.com" \
  -H "Access-Control-Request-Method: POST" | grep -i 'access-control'
kill %1

# 5. Verify Vault is accessible
kubectl -n clario360 exec -it vault-0 -- vault status
kubectl -n clario360 exec -it vault-0 -- vault kv get secret/clario360/jwt/signing-key

# 6. Verify no auth errors in recent logs
kubectl -n clario360 logs deploy/iam-service --since=5m | grep -i -E 'error|fail' | wc -l
kubectl -n clario360 logs deploy/api-gateway --since=5m | grep -i -E '401|403|unauthorized' | wc -l

# 7. Verify rate limits are not blocking legitimate traffic
kubectl -n clario360 exec -it deploy/redis -- redis-cli KEYS 'ratelimit:*auth*'
```

---

## Related Links

- [TS-001: API Latency Investigation](./TS-001-slow-api-responses.md)
- [TS-002: Data Pipeline Failure Investigation](./TS-002-failed-pipelines.md)
- [TS-003: Kafka Event Loss Investigation](./TS-003-missing-events.md)
- [TS-005: WebSocket Connectivity Issues](./TS-005-websocket-disconnects.md)
- Grafana Auth Dashboard: `/d/auth-monitoring/authentication-monitoring`
- Vault UI: `https://vault.clario360.internal`
- JWT.io (for manual token inspection): https://jwt.io
