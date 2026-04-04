# PROMPT: Backend End-to-End Integration Testing & Bug Fixing

You are testing the Clario360 Go backend at `/Users/mac/clario360/backend`.

## CRITICAL RULES (VIOLATIONS WILL BREAK THE BUILD)

- ALL Go commands: `GOWORK=off go test ./... -count=1`
- Go module: `github.com/clario360/platform` (NOT github.com/clario360/backend)
- Prometheus: ALWAYS `prometheus.NewRegistry()` + `promauto.With(reg)` — NEVER default registry
- Router: Chi v5 — middleware is `func(http.Handler) http.Handler`
- Auth context: `auth.UserFromContext(ctx)` → `*auth.ContextUser{ID, TenantID, Email, Roles []string}`
- Auth context: `auth.TenantFromContext(ctx)` → `string`
- Auth context: `auth.WithUser(ctx, &auth.ContextUser{...})` / `auth.WithTenantID(ctx, id)`
- pgxmock: `pgxmock.NewPool()` returns `PgxPoolIface` — use for DB mocks
- miniredis: `miniredis/v2` is in go.mod — use for Redis-dependent tests
- testcontainers-go: available with postgres, mysql, redpanda modules — use for real DB integration tests
- NEVER use the default Prometheus registry — it causes duplicate registration panics in tests

## PHASE 1: BUILD VERIFICATION & STATIC ANALYSIS

Run these first. Fix every error before proceeding:

```bash
GOWORK=off go build ./...
GOWORK=off go vet ./...
GOWORK=off go test ./... -count=1 -race -timeout 300s
```

Report: total tests before you start, any failures, any race conditions.

## PHASE 2: SERVICE-LEVEL INTEGRATION TESTS

For EACH of the 15 services under `cmd/`, write integration tests that verify the full
request→handler→repository→database→response cycle. Use `httptest.NewServer` with the
real Chi router from each service.

### 2.1 IAM Service (`cmd/iam-service/`)

Test real CRUD operations against the IAM API:

**User CRUD** (`internal/iam/`):
- `POST /api/v1/users` — Create user with `{email, first_name, last_name, password, role_ids}`
  - Verify: 201 response, user in DB, password hashed (not plaintext), audit event emitted
  - Verify: 409 on duplicate email
  - Verify: 400 on missing required fields
  - Verify: tenant_id injected from JWT context (not from request body — mass assignment check)
- `GET /api/v1/users` — List users
  - Verify: only returns users for the authenticated tenant (RLS)
  - Verify: pagination works (`?page=1&per_page=10`), returns correct `meta.total`
  - Verify: search filter (`?search=alice`) works
- `GET /api/v1/users/{id}` — Get user
  - Verify: 404 for non-existent user
  - Verify: 404 (not 403) for user in different tenant (BOLA prevention)
- `PUT /api/v1/users/{id}` — Update user
  - Verify: cannot update `tenant_id`, `password_hash`, `id` (mass assignment)
  - Verify: audit log records old/new values
- `DELETE /api/v1/users/{id}` — Soft delete
  - Verify: sets `deleted_at`, doesn't physically remove row
  - Verify: user no longer appears in GET /users list
  - Verify: cannot delete self

**Role CRUD** (`internal/iam/`):
- Full CRUD cycle: create role → assign to user → verify permissions → remove → delete
- Verify: cannot delete system roles (`is_system_role = true`)
- Verify: permissions field validates against known permission strings

**Auth Flow**:
- `POST /api/v1/auth/login` → verify JWT contains `sub`, `tenant_id`, `roles`, `permissions`, `exp`
- `POST /api/v1/auth/refresh` → verify new access token, old refresh token invalidated
- `POST /api/v1/auth/logout` → verify session deleted from Redis
- MFA flow: enable → verify TOTP → login requires MFA → verify code → success

**Onboarding Flow** (`internal/onboarding/`):
- `POST /api/v1/onboarding/register` → creates tenant + admin user + sends OTP
  - Verify: rate limited (5/hour per IP)
  - Verify: tenant status is "pending_verification"
- `POST /api/v1/onboarding/verify-email` → marks email verified
  - Verify: rate limited (20/10min)
  - Verify: invalid OTP returns 400
- Wizard flow: `/wizard/organization` → `/wizard/branding` → `/wizard/team` → `/wizard/suites` → `/wizard/complete`
  - Verify: each step validates and saves state
  - Verify: `/wizard/complete` triggers provisioning (creates databases, Kafka topics)
- Invitation flow: create invite → validate token → accept → user created with correct tenant

### 2.2 Cyber Service (`cmd/cyber-service/`)

Test real CRUD against cyber domain tables:

**Asset CRUD** (`internal/cyber/asset/`):
- Create asset: `{name, type: "server", ip_address, hostname, os, criticality: "high", tags: ["prod"]}`
  - Verify: UUID generated, tenant_id from context, `created_at` set
  - Verify: Kafka event `cyber.asset.events` published with action "created"
- Bulk create: `POST /api/v1/cyber/assets/bulk-create` with 100 assets
  - Verify: all created, proper error handling for duplicates
- Update asset tags: `PATCH /api/v1/cyber/assets/{id}/tags`
- Delete: verify soft delete, excluded from listings

**Vulnerability CRUD** (`internal/cyber/vulnerability/`):
- Create vulnerability linked to asset
- Status transitions: `open` → `in_progress` → `resolved`
- Verify CVE enrichment data attached

**Alert CRUD** (`internal/cyber/`):
- Create alert, acknowledge, resolve
- Verify severity filtering works

**CTEM Assessment** (`internal/cyber/ctem/`):
- Create assessment → run → verify findings generated → validate findings → export report
- Verify status transitions: `scheduled` → `running` → `completed`

**Detection Rules** (`internal/cyber/detection/`):
- Create Sigma rule, YARA rule, custom rule
- Verify rule validation (malformed rules rejected)

**Risk Scoring** (`internal/cyber/risk/`):
- Verify score calculation after asset/vulnerability changes
- Verify trend data populates

### 2.3 Data Service (`cmd/data-service/`)

**Data Source CRUD** (`internal/data/`):
- Create source with encrypted connection config
- Verify connection test endpoint works
- Verify schema discovery returns columns

**Pipeline CRUD** (`internal/data/pipeline/`):
- Create pipeline with transforms: `[rename, cast, filter, deduplicate, aggregate]`
- Execute pipeline → verify `pipeline_runs` created with status
- Verify transform chain order matters

**Data Quality** (`internal/data/quality/`):
- Create quality rule → execute → verify score
- Verify contradiction detection works

### 2.4 Audit Service (`cmd/audit-service/`)

**Audit Log Operations** (`internal/audit/`):
- Verify events consumed from `platform.audit.events` Kafka topic
- Verify hash chain integrity: each entry's hash includes previous entry's hash
- `POST /api/v1/audit/verify` → verify chain validation works
- Export: verify CSV and JSON formats
- Verify tenant isolation (RLS): tenant A cannot see tenant B's logs
- Verify partition creation for date ranges

### 2.5 Notification Service (`cmd/notification-service/`)

**Notification CRUD** (`internal/notification/`):
- Create notification → verify delivery to correct channels (in-app, email, webhook)
- Mark as read, delete, mark all as read
- Verify unread count accuracy
- WebSocket: connect, receive real-time notification, verify message format
- Webhook delivery: verify HMAC signature on outgoing webhook
- Preferences: update notification preferences, verify channel routing respects them

### 2.6 File Service (`cmd/file-service/`)

**File Operations** (`internal/filemanager/`):
- Upload file → verify stored in MinIO, metadata in PostgreSQL
- Verify virus scan runs (mock ClamAV or use Docker ClamAV on port 3310)
- Verify file blocked when malware detected (quarantined)
- Verify encryption at rest (AES-256-GCM)
- Download → verify decryption works
- Version history: upload v2 of same file, verify versions tracked
- Presigned URLs: verify they expire after configured time

### 2.7 Workflow Engine (`cmd/workflow-engine/`)

**Workflow Operations** (`internal/workflow/`):
- Create workflow definition with steps: `[service_task, human_task, condition, parallel_gateway]`
- Execute workflow → verify step execution order
- Human task: claim → complete with form data → next step executes
- Condition step: verify branching logic
- Timer step: verify timeout handling
- Verify Kafka events emitted at each state transition

### 2.8 API Gateway (`cmd/api-gateway/`)

Already has tests. Verify:
- Circuit breaker trips after 5 failures, resets after timeout
- Rate limiting per tenant per endpoint
- WebSocket proxy works end-to-end
- Health check aggregates all backend statuses
- Proxy headers injected correctly (X-Tenant-ID, X-User-ID)

## PHASE 3: CROSS-SERVICE INTEGRATION TESTS

Create `internal/integration/` package with tests that span multiple services:

### 3.1 Full Tenant Lifecycle

```
Register → Verify Email → Complete Wizard → Provision (creates DBs + topics) →
Create Users → Assign Roles → Login → Access Resources → Deprovision → Verify Cleanup
```

### 3.2 Security Event Chain

```
Failed Login (5x) → Account Lockout → Escalation Event →
Audit Log Entry → Notification to Admin → WebSocket Push
```

### 3.3 File Upload Pipeline

```
Upload File → ClamAV Scan → Store in MinIO → Metadata in PostgreSQL →
Kafka Event → Audit Log → Notification to Uploader
```

### 3.4 Cyber Alert Pipeline

```
Detection Rule Fires → Alert Created → Kafka Event →
Notification Service → WebSocket → Audit Trail
```

### 3.5 Multi-Tenant Isolation

For every endpoint that returns data:

```
Create resource as Tenant A → Attempt access as Tenant B → Verify 404 (not 403)
Verify SQL queries include `WHERE tenant_id = $1 AND deleted_at IS NULL`
Verify Redis keys are tenant-scoped
Verify Kafka events include tenant_id
```

## PHASE 4: SECURITY TESTS

### 4.1 OWASP API Security Top 10

For EACH endpoint:
- **BOLA (API1)**: Access resource with wrong tenant → expect 404
- **Auth bypass (API2)**: Request without JWT → expect 401; expired JWT → expect 401
- **Mass assignment (API3)**: Send `tenant_id`, `id`, `password_hash` in body → verify ignored/rejected
- **Rate limiting (API4)**: Exceed rate limit → expect 429 with Retry-After header
- **BFLA (API5)**: Analyst role tries admin endpoint → expect 403
- **SSRF (API7)**: Webhook URL with `http://169.254.169.254/` → expect blocked
- **Injection (API8)**: SQL injection in query params, path params, JSON body → expect blocked
- **XSS (API8)**: `<script>` tags in string fields → expect sanitized/rejected

### 4.2 Security Middleware Tests (`internal/security/`)

Test the full middleware chain in order:

```
SecurityHeaders → CSRF → APIRateLimit → SanitizeBody → APISecurityMiddleware → ContentTypeEnforcement
```

Verify each middleware passes the request to the next, and blocks when it should.

### 4.3 Session Security

- Verify session fixation prevention (session ID rotates on auth state change)
- Verify concurrent session limit (default: 5)
- Verify idle timeout (30 min) and absolute timeout (24 hours)
- Verify session destroyed on password change

## PHASE 5: KAFKA EVENT TESTS

For each of the 28 Kafka topics, verify:
- Producer serializes events correctly (JSON schema matches consumer expectations)
- Consumer deserializes and processes events (idempotency guard works)
- Dead-letter queue: malformed events go to `platform.dead-letter`
- Event ordering: verify events processed in order within a partition

Topics to test: `platform.iam.events`, `platform.audit.events`, `platform.notification.events`,
`platform.file.events`, `platform.workflow.events`, `cyber.asset.events`, `cyber.alert.events`,
`cyber.vulnerability.events`, `data.pipeline.events`, `data.quality.events`

## PHASE 6: DATABASE TESTS

### 6.1 Migration Verification

- Run all migrations forward
- Run all migrations backward
- Re-run forward → verify idempotent

### 6.2 RLS (Row-Level Security)

For every tenant-scoped table across all 7 databases:
- Set `app.current_tenant_id` in session → verify only tenant's rows returned
- Unset tenant → verify no rows returned (not all rows)

### 6.3 Indexes

- Verify every foreign key has an index
- Verify every `WHERE tenant_id = $1` query uses an index (EXPLAIN ANALYZE)

## PHASE 7: BUG FIXING

For EVERY test failure:
1. Read the failing test and the source code it tests
2. Identify the root cause with absolute certainty (read all related code)
3. Fix the source code (not the test) unless the test is wrong
4. Re-run the specific test to verify the fix
5. Re-run the full test suite to verify no regressions

Report every bug found with:
- File path and line number
- What was wrong
- What the fix was
- Before/after code

## PHASE 8: FINAL VERIFICATION

```bash
GOWORK=off go build ./...            # Must pass
GOWORK=off go vet ./...              # Must pass
GOWORK=off go test ./... -count=1 -race -timeout 600s  # Must pass, zero race conditions
```

Report:
- Total tests: before → after
- New test files created (list each)
- Bugs found and fixed (list each)
- Test coverage per package (if possible)
- Any architectural concerns discovered
