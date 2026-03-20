# Clario360 Platform - Full-Spectrum Security Architecture Assessment

**Date:** 2026-03-16
**Scope:** Backend (`/backend`) + Frontend (`/frontend`) + Deployment (`/deploy`)
**Methodology:** Code-level static analysis, architecture review, threat modeling
**Classification:** CONFIDENTIAL

---

## A. Executive Security Summary

### Overall Security Posture: STRONG (7.5/10)

The Clario360 platform demonstrates **mature, enterprise-grade security engineering** with defense-in-depth across authentication, authorization, data protection, and infrastructure. The codebase shows intentional security design rather than bolted-on controls.

### Top Security Strengths

1. **In-memory access tokens** - XSS-resistant; never stored in localStorage/sessionStorage
2. **RS256 JWT with single-use refresh token rotation** and reuse detection (auto-revokes all sessions on theft)
3. **Row-Level Security (RLS)** via PostgreSQL `SET LOCAL` transaction-scoped tenant isolation
4. **Immutable audit logs** with SHA-256 hash chain integrity and database trigger enforcement
5. **AES-256-GCM envelope encryption** with DEK/KEK model, memory cleanup, key rotation support
6. **Zero-trust Kubernetes networking** - default-deny NetworkPolicies with microsegmentation
7. **Comprehensive CI/CD security scanning** - govulncheck, gosec, Semgrep, TruffleHog, Trivy, npm audit
8. **Cloud KMS integration** with HSM-backed keys and 90-day automatic rotation
9. **Multi-layer file upload security** - magic byte validation, polyglot detection, ClamAV integration
10. **BOLA/BFLA prevention** with ownership verification and mass assignment field whitelisting

### Top Critical Risks

1. **Webhook SSRF** - Notification webhook delivery has no URL validation (can target internal services, cloud metadata)
2. **Rate limit fail-open** - All rate limits bypass on Redis outage; no in-process fallback
3. **Login lockout threshold too high** - 20 attempts per 15 minutes enables meaningful brute-force
4. **No email verification** on registration - accounts created without proving email ownership
5. **CORS wildcard risk** - Production deployment could inherit `AllowedOrigins=["*"]` with `AllowCredentials=true`

### Overall Maturity Assessment

| Domain | Maturity | Rating |
|--------|----------|--------|
| Authentication | Advanced | 8/10 |
| Authorization | Advanced | 7.5/10 |
| Data Protection | Advanced | 8/10 |
| Input Validation | Advanced | 8/10 |
| Audit & Logging | Advanced | 8.5/10 |
| Infrastructure Security | Advanced | 8/10 |
| CI/CD Security | Advanced | 8/10 |
| Secrets Management | Intermediate | 6.5/10 |
| Incident Response | Basic | 4/10 |
| Operational Security | Intermediate | 6/10 |

---

## B. Current-State Security Architecture

### System Architecture Overview

```
                    Internet
                       |
                 [Ingress/TLS]
                       |
              [API Gateway :8080]
              /    |    |    \
         [IAM] [Audit] [Cyber] [Data] [Acta] [Lex] [Visus] [Files] [Notify] [Workflow]
          :8081  :8084  :8085  :8086  :8087  :8088  :8089   :8091   :8090    :8083
              \    |    |    /
         [PostgreSQL] [Redis] [Kafka] [MinIO]
              :5432    :6379   :9092   :9000
```

### Trust Boundaries

```
TB1: Internet ←→ Ingress Controller (TLS termination)
TB2: Ingress ←→ API Gateway (NetworkPolicy enforced)
TB3: API Gateway ←→ Backend Services (JWT-validated proxy, internal headers)
TB4: Backend Services ←→ Data Stores (RLS-enforced, encrypted connections)
TB5: Frontend (Browser) ←→ BFF (Next.js API Routes) (httpOnly cookies)
TB6: BFF ←→ API Gateway (Bearer tokens, CSRF)
```

### Entry Points

| Entry Point | Protocol | Auth | Rate Limited |
|------------|----------|------|-------------|
| `/api/v1/auth/*` | HTTPS | Public | 20 req/min (IP) |
| `/api/v1/*` (authenticated) | HTTPS | JWT Bearer | Per-tenant sliding window |
| `/ws/v1/*` | WSS | JWT (query/header) | Per-user connection limit |
| `/api/v1/files/upload` | HTTPS | JWT Bearer | Per-tenant, 100MB max |
| `/api/v1/integrations/*` | HTTPS | Public | Per-IP |
| `/.well-known/*` | HTTPS | Public | Standard |
| BFF `/api/auth/*` | HTTPS | httpOnly cookie | Via gateway |

### Authentication Flow

```
1. User submits credentials → POST /api/v1/auth/login
2. IAM validates: lockout check → bcrypt compare → MFA check
3. If MFA enabled: returns mfa_token (5-min TTL) → user submits TOTP
4. IAM generates: access token (15-min, RS256 JWT) + refresh token (7-day)
5. Frontend stores: access token in memory, sends refresh to BFF
6. BFF sets: httpOnly cookie with refresh token (SameSite=strict, path=/api/auth)
7. On 401: Axios interceptor calls BFF /api/auth/refresh → new token pair
8. Refresh: validates JWT → looks up session by hash → reuse detection → rotate
9. On tab focus: proactive silent refresh if token near expiry (30s buffer)
```

### Access Control Model

- **RBAC with wildcard permissions**: `resource:action` format (e.g., `cyber:read`, `data:write`)
- **16 system roles**: super-admin (`*`), tenant-admin, security-analyst, data-engineer, etc.
- **Permission checking**: exact match → `resource:*` → `*:action` → `*` (superuser)
- **Enforcement layers**: Gateway middleware → service middleware → repository (RLS)
- **Frontend**: UI-only permission gating via `PermissionRedirect` / `PermissionGate` components

### Data Protection Mechanisms

- **Encryption at rest**: AES-256-GCM for data source configs, integration credentials, MFA secrets
- **Password hashing**: bcrypt cost 12 (configurable)
- **API key storage**: SHA-256 hash only (secret never stored)
- **Token storage**: Refresh token SHA-256 hash in sessions table
- **PII masking**: Role-based audit log masking (IP, email, user-agent)
- **Log redaction**: Automatic field-level redaction (password, token, secret, key, credit_card, ssn)

### External Dependencies

| Dependency | Purpose | Security Implications |
|-----------|---------|----------------------|
| PostgreSQL | Primary data store | RLS, encrypted connections |
| Redis | Rate limiting, session cache, MFA tokens | Ephemeral data, short TTLs |
| Kafka | Event streaming, audit ingestion | SASL_SSL in production |
| MinIO | Object storage (files) | TLS, access key auth |
| ClamAV | Antivirus scanning | Optional, fail-closed |
| GCP Cloud KMS | Key management | HSM-backed, auto-rotation |

---

## C. Threat Model

### Assets

| Asset | Classification | Value |
|-------|---------------|-------|
| User credentials (passwords, MFA secrets) | Critical | Account takeover |
| JWT signing keys (RSA private key) | Critical | Universal token forgery |
| Encryption keys (AES-256) | Critical | Data breach |
| PII (names, emails, IPs) | High | Privacy/compliance violation |
| Audit logs | High | Forensic integrity |
| Data source connection strings | High | Lateral movement |
| API keys | High | Unauthorized access |
| Business data (cyber, compliance, governance) | High | Competitive/regulatory risk |
| Integration credentials (Slack, Jira, SMTP) | Medium | Impersonation |
| Session tokens | Medium | Session hijacking |

### Actors & Threat Agents

| Actor | Motivation | Capability |
|-------|-----------|-----------|
| External attacker | Data theft, ransomware | Network-level, web application attacks |
| Malicious tenant user | Privilege escalation, cross-tenant access | Authenticated, limited permissions |
| Malicious admin | Data exfiltration | Full tenant access |
| Insider (developer) | IP theft | Code/infrastructure access |
| Automated bot | Credential stuffing, scraping | High volume, distributed |
| Supply chain attacker | Backdoor | Dependency manipulation |

### STRIDE Threat Analysis

| Threat | Category | Attack Surface | Mitigations Present | Residual Risk |
|--------|----------|---------------|-------------------|---------------|
| Token forgery | Spoofing | JWT signing | RS256 with managed keys | Low |
| Cross-tenant data access | Tampering | Database queries | RLS + query filters | Low |
| Audit log tampering | Repudiation | Audit database | Immutable trigger + hash chain | Low |
| PII exposure in logs | Information Disclosure | Application logs | Field-level redaction | Medium |
| Webhook SSRF | Elevation of Privilege | Notification service | **MISSING** | **High** |
| Brute-force login | Spoofing | Auth endpoint | Rate limiting (20/15min) | Medium |
| Session fixation | Spoofing | Session management | Token rotation, fingerprinting | Low |
| XSS via stored content | Tampering | Rich text fields | Sanitization + CSP | Low |
| SQL injection | Tampering | All query endpoints | Parameterized queries + pattern detection | Very Low |
| Dependency vulnerability | All | Build pipeline | govulncheck, npm audit, Semgrep | Low |

### Attack Surfaces

```
External Attack Surface:
├── API Gateway (all /api/v1/* endpoints)
├── WebSocket endpoints (/ws/v1/*)
├── Public auth endpoints (login, register, forgot-password)
├── OAuth callback handler
├── File upload endpoint (100MB max)
└── Integration webhook receivers

Internal Attack Surface:
├── Service-to-service communication (internal headers)
├── Database connections (PostgreSQL, Redis)
├── Kafka message bus
├── MinIO object storage
└── DNS resolution (SSRF vector)

Client-Side Attack Surface:
├── Browser JavaScript (in-memory tokens)
├── Cookie storage (httpOnly refresh tokens)
├── localStorage (wizard drafts, UI state)
├── WebSocket connections
└── OAuth state parameter
```

### Likely Attack Paths

1. **Credential Stuffing → Account Takeover**: Distributed bot attack bypasses IP-based rate limiting (20 req/min/IP). Lockout at 20 attempts allows meaningful password guessing.
2. **Webhook SSRF → Internal Service Access**: Register webhook pointing to `http://169.254.169.254/latest/meta-data/iam/security-credentials/` to steal cloud credentials.
3. **Cross-Tenant via API Key**: `ValidateKey()` may not enforce tenant isolation explicitly - key valid for any tenant context.
4. **Session Replay**: Refresh tokens not bound to IP/fingerprint - stolen cookie reusable from different device.
5. **OAuth State Manipulation**: Base64 state parameter decoded without cryptographic validation - redirect manipulation possible.

---

## D. Security Findings

### CRITICAL Severity

#### D1. Webhook SSRF - No URL Validation on Delivery
- **Affected Area**: `internal/notification/handler/webhook_helpers.go:20`
- **Evidence**: `deliverWebhookTest()` calls `http.NewRequest(http.MethodPost, url, ...)` with user-supplied URL. No private IP blocking, no metadata endpoint protection, no scheme restriction.
- **Risk**: Authenticated user creates webhook targeting `http://169.254.169.254/latest/meta-data/` (AWS), `http://localhost:8081/api/v1/internal/users` (IAM), or `http://localhost:6379` (Redis) to exfiltrate data or execute commands.
- **Attack Scenario**: Attacker registers as tenant user → creates webhook with URL `http://169.254.169.254/latest/meta-data/iam/security-credentials/` → triggers notification → receives cloud IAM credentials in webhook response → escalates to full cloud access.
- **Remediation**: Apply the existing SSRF protection from `internal/security/ssrf.go` to webhook delivery. Validate URL against private IP ranges, cloud metadata endpoints, and internal service ports. Use the `SSRFValidator` with DNS rebinding protection.
- **Priority**: P0 - Immediate

### HIGH Severity

#### D2. Rate Limiting Fails Open on Redis Outage
- **Affected Area**: `internal/gateway/ratelimit/limiter.go:53-71`, `internal/middleware/ratelimit.go:56`
- **Evidence**: `if l.rdb == nil { return Result{Allowed: true} }` and error path returns `Allowed: true`
- **Risk**: Redis outage (crash, network partition, OOM) disables ALL rate limiting platform-wide. Authentication brute-force, API abuse, and DoS become unthrottled.
- **Attack Scenario**: Attacker induces Redis memory exhaustion via cache poisoning → Redis crashes → all rate limits disabled → automated credential stuffing at full speed.
- **Remediation**: Implement in-process fallback using `golang.org/x/time/rate` (token bucket) with conservative defaults. Cache recent rate limit state locally. Alert on Redis unavailability.
- **Priority**: P0 - Immediate

#### D3. Login Lockout Threshold Too Permissive
- **Affected Area**: `internal/iam/service/auth_service.go:30`
- **Evidence**: `loginLockoutMax = 20` with 15-minute window. IP-based limiting at 20 req/min in gateway.
- **Risk**: 20 attempts per email within 15 minutes allows ~80 password attempts per hour per targeted account. With a small dictionary or leaked credential list, this is exploitable.
- **Attack Scenario**: Attacker obtains user email list (from data breach) → rotates across IPs (residential proxy) → attempts top 20 passwords per account → achieves 1-3% hit rate on large user bases.
- **Remediation**: Reduce lockout threshold to 5 attempts. Implement progressive delays (1s, 2s, 4s, 8s...) after 3 failures. Add CAPTCHA challenge after 3 failed attempts. Implement anomaly detection for distributed attacks.
- **Priority**: P1 - Short-term

#### D4. No Email Verification on Registration
- **Affected Area**: `internal/iam/service/auth_service.go:105-170`
- **Evidence**: `Register()` creates user with `status: active` immediately. No verification email sent. No `pending_verification` status applied despite the status being defined in the model.
- **Risk**: Attackers can register with arbitrary emails (including victim emails), creating phishing-capable accounts or polluting tenant namespaces.
- **Attack Scenario**: Attacker registers with `ceo@targetcompany.com` → appears as legitimate user in tenant → social engineers other users via internal notifications.
- **Remediation**: Set initial status to `pending_verification`. Send verification email with time-limited token. Only activate account after email confirmation.
- **Priority**: P1 - Short-term

#### D5. CORS Configuration Not Production-Hardened
- **Affected Area**: `cmd/api-gateway/main.go:137-144`, `internal/middleware/cors.go`
- **Evidence**: `AllowedOrigins` loaded from config with `AllowCredentials: true`. Default includes `localhost:3000, localhost:8080`. If production misconfigured with wildcard, enables credential theft.
- **Risk**: `Access-Control-Allow-Origin: *` combined with `Access-Control-Allow-Credentials: true` allows any website to make authenticated API calls on behalf of users.
- **Remediation**: Add startup validation that rejects wildcard origins when credentials are enabled. Require explicit origin allowlist in production configuration. Log warnings for localhost origins in non-dev environments.
- **Priority**: P1 - Short-term

#### D6. API Key Tenant Isolation Gap
- **Affected Area**: `internal/iam/service/apikey_service.go:196-216`
- **Evidence**: `ValidateKey()` looks up key by hash but does not explicitly verify the key's `tenant_id` matches the request's tenant context.
- **Risk**: An API key created in Tenant A could potentially be used to access Tenant B's resources if the gateway doesn't enforce tenant matching.
- **Remediation**: Add explicit tenant ID comparison in `ValidateKey()`: verify `key.TenantID == requestTenantID`.
- **Priority**: P1 - Short-term

### MEDIUM Severity

#### D7. OAuth State Parameter Not Cryptographically Protected
- **Affected Area**: Frontend `app/(auth)/callback/page.tsx:66`, `components/auth/oauth-providers.tsx`
- **Evidence**: State is `btoa(JSON.stringify({ provider, redirect_to: "/dashboard" }))` - plain base64, no HMAC or nonce.
- **Risk**: Attacker can craft arbitrary state parameter to redirect users to malicious URLs after OAuth flow.
- **Remediation**: Include a cryptographic nonce (stored in session) in the state parameter. Validate nonce on callback. Sign state with HMAC.
- **Priority**: P2

#### D8. No Password History Enforcement
- **Affected Area**: `internal/iam/service/auth_service.go:416-444`
- **Evidence**: `ResetPassword()` validates password strength but does not check against previous passwords.
- **Risk**: Users can cycle back to previously compromised passwords.
- **Remediation**: Store last N password hashes (e.g., 12). Compare new password against history before accepting.
- **Priority**: P2

#### D9. Refresh Token Not Bound to Client Context
- **Affected Area**: `internal/iam/service/auth_service.go:331-373`
- **Evidence**: Session stores `IPAddress` and `UserAgent` but `RefreshToken()` does not validate these against the current request.
- **Risk**: Stolen refresh token (via cookie exfiltration) can be used from any IP/device.
- **Remediation**: Optionally validate IP subnet (Class C) and user-agent on refresh. Implement step-up authentication for IP changes.
- **Priority**: P2

#### D10. WebSocket Messages Not Validated at Gateway
- **Affected Area**: `internal/gateway/proxy/websocket_proxy.go:138-206`
- **Evidence**: Messages forwarded as-is between client and backend. No size limits, schema validation, or injection filtering.
- **Risk**: Malicious client sends oversized messages (memory exhaustion) or injection payloads through WebSocket proxy.
- **Remediation**: Add configurable message size limits (e.g., 64KB). Implement message rate limiting per connection. Log oversized message attempts.
- **Priority**: P2

#### D11. MFA Recovery Code Timing Side-Channel
- **Affected Area**: `internal/iam/service/auth_service.go:302-310`
- **Evidence**: Recovery codes checked sequentially with `bcrypt.CompareHashAndPassword()` in a loop. First match returns success - timing reveals code position.
- **Risk**: Statistical timing analysis could narrow recovery code guesses. Impact is LOW due to bcrypt's inherent timing properties (constant-ish comparison).
- **Remediation**: Always iterate all codes regardless of match. Use constant-time result aggregation.
- **Priority**: P3

#### D12. Hardcoded Development Defaults in Configuration
- **Affected Area**: `internal/config/config.go:147,166-167,182-183,187`
- **Evidence**: Default database password `clario_dev_pass`, JWT secret `change-me-in-production-use-256-bit-key`, MinIO creds `clario_minio/clario_minio_secret`, encryption key `0123456789abcdef...`
- **Risk**: If production deployment fails to set environment variables, services start with known credentials.
- **Remediation**: Remove defaults for security-critical config. Fail startup if required vars are missing in production. Add `Validate()` function that checks environment.
- **Priority**: P2

#### D13. `GetByEmailGlobal()` Cross-Tenant Email Enumeration
- **Affected Area**: `internal/iam/repository/user_repo.go:120-150`
- **Evidence**: Login without `tenant_id` calls `GetByEmailGlobal()` which searches across all tenants.
- **Risk**: Attacker can determine if an email exists in ANY tenant by attempting login without specifying tenant.
- **Remediation**: Always return generic "invalid credentials" regardless of email existence. Add consistent timing to login responses. Consider requiring tenant selection before login.
- **Priority**: P2

### LOW Severity

#### D14. Task Draft Data in localStorage
- **Affected Area**: Frontend `app/(dashboard)/workflows/tasks/[id]/task-detail-page-client.tsx`
- **Evidence**: Task form drafts persisted in localStorage, which may contain sensitive workflow data.
- **Remediation**: Implement encryption for localStorage data or use sessionStorage with shorter lifecycle.
- **Priority**: P3

#### D15. CSP `style-src 'unsafe-inline'` in Production
- **Affected Area**: Frontend `middleware/security.ts:73`
- **Evidence**: `style-src 'self' 'unsafe-inline'` allows inline styles in production (required by Tailwind CSS).
- **Risk**: Reduces CSP effectiveness against style-injection attacks.
- **Remediation**: Migrate to nonce-based style injection or use Tailwind's purge output as external stylesheets.
- **Priority**: P3

#### D16. Audit Log Partitions Not Pre-Created Beyond 2026
- **Affected Area**: `migrations/audit_db/000001_init_schema.up.sql`
- **Evidence**: Partitions only pre-created for 2025-2026. After 2026, logs fall into default partition (performance degradation).
- **Remediation**: Implement automated monthly partition creation via cron job or maintenance task.
- **Priority**: P3

---

## E. Backend Security Architecture (Target State)

### Authentication/Authorization Model

```
┌─────────────────────────────────────────────────────────┐
│                    API Gateway                          │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌───────────┐  │
│  │Recovery │→│RequestID │→│SecHeaders│→│   CORS    │  │
│  └─────────┘ └──────────┘ └──────────┘ └───────────┘  │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌───────────┐  │
│  │BodyLimit│→│  Logging │→│ Tracing  │→│  Timeout  │  │
│  └─────────┘ └──────────┘ └──────────┘ └───────────┘  │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌───────────┐  │
│  │RateLimit│→│ProxyAuth │→│  Tenant  │→│   CSRF    │  │
│  └─────────┘ └──────────┘ └──────────┘ └───────────┘  │
│  ┌─────────┐                                           │
│  │  SSRF   │→ Reverse Proxy / WebSocket Proxy          │
│  └─────────┘                                           │
└─────────────────────────────────────────────────────────┘
```

### Recommended Security Layers

1. **Input Layer**: Body size limit → JSON schema validation → field sanitization → injection detection
2. **Auth Layer**: JWT validation (RS256) → tenant extraction → permission check → CSRF validation
3. **Business Layer**: BOLA ownership check → data-level authorization → field-level masking
4. **Data Layer**: Parameterized queries → RLS enforcement → encryption at rest → audit logging
5. **Output Layer**: Error sanitization → response filtering → security headers → CORS

### API Protection Model (Target)

```yaml
Authentication:
  primary: RS256 JWT (access token, 15-min TTL)
  secondary: API key (SHA-256 hashed, scoped permissions)
  refresh: Single-use rotating refresh tokens with reuse detection
  mfa: TOTP + bcrypt-hashed recovery codes

Authorization:
  model: RBAC with wildcard permission matching
  enforcement:
    - Gateway: RequirePermission middleware
    - Service: Business logic permission checks
    - Database: RLS tenant isolation
  escalation_prevention:
    - BOLA checks on object access
    - Mass assignment field whitelisting
    - Admin operation audit logging

Rate Limiting:
  auth_endpoints: 5 req/min per email, 20 req/min per IP
  read_endpoints: 100 req/min per tenant
  write_endpoints: 50 req/min per tenant
  upload_endpoints: 10 req/min per tenant
  websocket: 5 connections per user
  fallback: In-process token bucket (10 req/s) on Redis failure

Input Validation:
  body: go-playground/validator struct tags + DisallowUnknownFields
  query: Strict type parsing with bounds checking
  path: UUID format enforcement
  files: Magic byte + extension + size + ClamAV
  urls: SSRF validator (private IP, metadata, DNS rebinding)

Secrets Management:
  jwt_keys: Cloud KMS or Vault-managed RSA keys
  encryption_keys: Envelope encryption with KMS-backed KEK
  api_keys: crypto/rand generation, SHA-256 storage
  config: Environment variables via Kubernetes Secrets (external-secrets-operator)
  rotation: 90-day key rotation via KMS, JWT key rotation with grace period
```

### Recommended Additions

1. **Startup configuration validator**: Fail-fast if security-critical env vars use defaults
2. **Request signing for service-to-service**: HMAC-signed internal headers to prevent header injection
3. **Webhook SSRF protection**: Apply `SSRFValidator` to all outbound HTTP requests
4. **Circuit breaker for auth**: Prevent cascade failures in auth service
5. **Token revocation list**: Short-lived in-memory cache for revoked access tokens (before expiry)

---

## F. Frontend Security Architecture (Target State)

### Route Protection Model

```
┌──────────────────────────────────────────────────┐
│                Next.js Edge Middleware            │
│  1. Public path bypass (login, register, etc.)   │
│  2. Access token cookie check                    │
│  3. Token expiry validation (30s buffer)         │
│  4. Silent refresh attempt on near-expiry        │
│  5. Redirect to /login if unauthenticated        │
│  6. Security headers injection                   │
└──────────────────────────────────────────────────┘
           │
┌──────────────────────────────────────────────────┐
│              Component-Level Guards               │
│  PermissionRedirect: Route-level gating          │
│  PermissionGate: Element-level visibility        │
│  hasSuiteAccess: Suite-level feature gating      │
│  NOTE: UI-only - backend is the security boundary│
└──────────────────────────────────────────────────┘
```

### Secure Token Strategy

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐
│   Browser    │     │  BFF (Next) │     │  API Gateway │
│  Memory Only │────→│ httpOnly    │────→│  Bearer JWT  │
│  (accessTkn) │     │ Cookie      │     │  Validation  │
│              │     │ (refreshTkn)│     │              │
│ - Never in   │     │ - SameSite  │     │ - RS256      │
│   localStorage│    │   strict    │     │ - Expiry     │
│ - Lost on    │     │ - Path      │     │ - Tenant     │
│   refresh    │     │   /api/auth │     │ - Permission │
│ - 15min TTL  │     │ - Secure    │     │              │
└─────────────┘     │   (prod)    │     └──────────────┘
                     └─────────────┘
```

### CSP and Browser Security Controls (Target)

```
Content-Security-Policy:
  default-src: 'self'
  script-src: 'self' (NO unsafe-eval in production)
  style-src: 'self' 'unsafe-inline' (Tailwind requirement)
  img-src: 'self' data: https:
  connect-src: 'self' {API_URL} wss:
  worker-src: 'self' blob:
  frame-ancestors: 'none'
  base-uri: 'self'
  form-action: 'self'
  object-src: 'none'
  upgrade-insecure-requests (production)

Additional Headers:
  Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
  X-Content-Type-Options: nosniff
  X-Frame-Options: DENY
  X-XSS-Protection: 0
  Referrer-Policy: strict-origin-when-cross-origin
  Permissions-Policy: camera=(), microphone=(), geolocation=(), payment=()
  Cross-Origin-Embedder-Policy: require-corp (production)
  Cross-Origin-Opener-Policy: same-origin (production)
  Cross-Origin-Resource-Policy: same-origin (production)
```

### Frontend-to-Backend Trust Model

- Frontend NEVER trusts its own permission checks as security boundaries
- All sensitive operations validated by backend middleware
- Access tokens carry claims for UI optimization only
- Permission cache refreshed on every token refresh
- Session-expired events trigger immediate cleanup and redirect
- Error responses never rendered as HTML (JSON-only API consumption)

### Recommended Frontend Additions

1. **Subresource Integrity (SRI)**: Add integrity hashes to external script/style loads
2. **OAuth state nonce**: Generate cryptographic nonce, validate on callback
3. **Sensitive form auto-clear**: Clear password fields on blur/timeout
4. **CSP violation reporting**: Configure `report-uri` / `report-to` endpoint
5. **Client-side anomaly detection**: Detect and report unusual API patterns

---

## G. Data Flow Security Architecture

### Data Flow Map

```
┌─────────────────────────────────────────────────────────────────────┐
│                           DATA ENTRY                                │
│                                                                     │
│  [User Browser]──HTTPS──→[Ingress/TLS]──→[API Gateway]             │
│       │                                      │                      │
│   Form Input                          ┌──────┴───────┐             │
│   File Upload                         │              │             │
│   WebSocket                    Input Validation  Auth Check        │
│   OAuth Callback               (Zod/validator)  (JWT/CSRF)        │
│                                       │              │             │
│                                ┌──────┴──────────────┘             │
│                                │                                    │
│                         [Backend Service]                           │
│                                │                                    │
│                    ┌───────────┼───────────┐                       │
│                    │           │           │                        │
│              Sanitization  Validation  Authorization               │
│              (XSS/SQLi)   (Business)  (RBAC/BOLA)                 │
│                    │           │           │                        │
│                    └───────────┼───────────┘                       │
│                                │                                    │
├─────────────────────────────────────────────────────────────────────┤
│                         DATA PROCESSING                             │
│                                                                     │
│  [Business Logic]──→[Encryption]──→[Data Store]                    │
│       │                  │              │                           │
│   Transform         AES-256-GCM    PostgreSQL                     │
│   Compute            Envelope      (RLS enforced)                 │
│   Aggregate          Encryption    Redis (ephemeral)              │
│                                    MinIO (files)                   │
│                                    Kafka (events)                  │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                          DATA STORAGE                               │
│                                                                     │
│  [PostgreSQL]                    [MinIO]                           │
│  - RLS tenant isolation          - UUID-based keys                 │
│  - Encrypted connections         - Tenant-scoped paths             │
│  - Column-level encryption       - AES-256 at rest                 │
│  - Soft deletes (audit trail)    - Checksum verification          │
│  - Immutable audit logs          - ClamAV scanned                 │
│  - Hash chain integrity          - Magic byte validated           │
│                                                                     │
│  [Redis]                         [Kafka]                           │
│  - Short TTLs (5min-24h)         - SASL_SSL in production         │
│  - Hashed keys (no PII)          - Audit events                   │
│  - Rate limit counters           - Lifecycle events               │
│  - MFA pending tokens            - Notification delivery          │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                           DATA EXIT                                 │
│                                                                     │
│  [API Response]                  [Webhook Delivery]                │
│  - Error sanitization            - HMAC signature                  │
│  - PII masking (role-based)      - ⚠ SSRF UNPROTECTED            │
│  - Field-level filtering                                           │
│  - No stack traces               [Email/SMTP]                     │
│                                  - TLS in transit                  │
│  [File Download]                 - Template-based                  │
│  - UUID-based access                                               │
│  - Checksum header               [Audit Export]                   │
│  - Content-Disposition           - Hash chain verified             │
│  - Sanitized filename            - Role-based masking             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Trust Boundary Controls

| Boundary | Control | Implementation |
|----------|---------|----------------|
| Internet → Ingress | TLS 1.2+ | Kubernetes Ingress with cert-manager |
| Ingress → Gateway | NetworkPolicy | Only ingress-nginx namespace allowed |
| Gateway → Services | JWT validation + internal headers | ProxyAuth middleware |
| Service → Database | RLS + parameterized queries | SET LOCAL per transaction |
| Service → Redis | Password auth | REDIS_PASSWORD env var |
| Service → Kafka | SASL_SSL | Username/password + TLS |
| Service → MinIO | Access key auth | TLS in production |
| Service → External | SSRF validation (partial) | SSRFValidator on data connectors |

---

## H. Security Control Matrix

| Control Domain | Current Status | Identified Gap | Recommended Control | Priority |
|---------------|---------------|---------------|-------------------|----------|
| **Authentication** | RS256 JWT, bcrypt, MFA | No email verification | Email verification flow | P1 |
| **Session Management** | Rotating refresh tokens, reuse detection | No client binding | IP/fingerprint validation on refresh | P2 |
| **Authorization** | RBAC + RLS + BOLA | API key tenant isolation gap | Explicit tenant check in ValidateKey | P1 |
| **Rate Limiting** | Redis sliding window | Fail-open, high thresholds | In-process fallback, lower thresholds | P0 |
| **Input Validation** | Struct validation, sanitization | WebSocket messages unvalidated | Message size/schema validation | P2 |
| **SSRF Protection** | Full protection on data connectors | Webhook delivery unprotected | Apply SSRFValidator to webhooks | P0 |
| **XSS Prevention** | CSP, sanitization, React escaping | CSP allows unsafe-inline styles | Nonce-based style injection | P3 |
| **CSRF Protection** | Double-submit cookie, SameSite | OAuth state not signed | HMAC-signed OAuth state | P2 |
| **SQL Injection** | Parameterized queries + pattern detection | None identified | Maintain current controls | N/A |
| **File Upload** | Magic byte, size, ClamAV, polyglot | None significant | Maintain current controls | N/A |
| **Encryption at Rest** | AES-256-GCM envelope encryption | No key rotation automation | Automated KMS rotation | P2 |
| **Encryption in Transit** | TLS at ingress, SASL_SSL Kafka | No mTLS between services | Service mesh or mTLS | P3 |
| **Secrets Management** | Env vars, K8s Secrets | Hardcoded dev defaults | Fail-fast on missing prod secrets | P2 |
| **Audit Logging** | Immutable, hash chain, partitioned | No real-time alerting | SIEM integration | P2 |
| **Log Security** | Field-level redaction | Audit stores unmasked email/IP | Application-level masking before storage | P3 |
| **CORS** | Origin whitelist | Wildcard + credentials risk | Startup validation in production | P1 |
| **Container Security** | Distroless, non-root | No runtime security | Falco or similar runtime monitoring | P3 |
| **Network Security** | Default-deny NetworkPolicies | No mTLS | Service mesh (Istio/Linkerd) | P3 |
| **Dependency Security** | govulncheck, npm audit, Trivy | Manual updates | Dependabot automated PRs | P2 |
| **Password Policy** | 12 char, complexity, blacklist | No password history | Store and check last 12 hashes | P2 |
| **Account Security** | Lockout (20 attempts) | No login anomaly alerts | Suspicious login notification service | P2 |
| **Incident Response** | Audit trail available | No playbooks or alerting | Incident response runbooks + PagerDuty | P2 |
| **Backup Security** | Not assessed | Unknown encryption status | Encrypted backups with tested restore | P2 |

---

## I. Target-State Holistic Security Architecture

### End-to-End Security Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        EXTERNAL PERIMETER                           │
│  [WAF] → [DDoS Protection] → [CDN/Edge] → [Load Balancer]         │
│  - OWASP Core Rule Set      - Rate limiting    - TLS 1.3          │
│  - Bot detection             - Geo-blocking     - Certificate      │
│  - Custom rules              - IP reputation      rotation         │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────┴─────────────────────────────────────┐
│                     KUBERNETES CLUSTER                               │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                  Ingress Controller                          │   │
│  │  - TLS termination (cert-manager)                           │   │
│  │  - Request size limits                                      │   │
│  │  - WebSocket upgrade support                                │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                              │ (NetworkPolicy: ingress-nginx only)  │
│  ┌──────────────────────────┴───────────────────────────────────┐   │
│  │                    API Gateway                               │   │
│  │  Recovery → RequestID → SecurityHeaders → CORS → BodyLimit  │   │
│  │  → Logging → Tracing → Timeout → RateLimit → CSRF          │   │
│  │  → ProxyAuth (RS256 JWT) → Tenant → SSRF → Proxy           │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                              │ (NetworkPolicy: gateway only)        │
│  ┌──────────────────────────┴───────────────────────────────────┐   │
│  │                  Backend Services                            │   │
│  │  ┌─────┐ ┌─────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐     │   │
│  │  │ IAM │ │Audit│ │Cyber │ │ Data │ │ Acta │ │ Lex  │ ... │   │
│  │  └──┬──┘ └──┬──┘ └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘     │   │
│  │     │       │       │        │        │        │           │   │
│  │  Per-service: Input Validation → Business Auth → BOLA      │   │
│  │  → Data Access (RLS) → Encryption → Audit Event            │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                              │ (NetworkPolicy: backend only)        │
│  ┌──────────────────────────┴───────────────────────────────────┐   │
│  │                   Data Tier                                  │   │
│  │  [PostgreSQL]    [Redis]      [Kafka]       [MinIO]         │   │
│  │  - RLS enabled   - AUTH       - SASL_SSL    - TLS           │   │
│  │  - TLS required  - No persist - Topic ACLs  - Bucket policy │   │
│  │  - Column encrypt - Short TTL  - Schema reg  - AES-256      │   │
│  │  - Audit triggers                                            │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                  Security Infrastructure                     │   │
│  │  [Cloud KMS]     [Vault]      [SIEM]        [Monitoring]   │   │
│  │  - Key rotation  - Dynamic    - Log ingest   - Prometheus   │   │
│  │  - HSM-backed      secrets    - Alerting     - Grafana      │   │
│  │  - Audit trail   - PKI mgmt   - Correlation  - AlertManager│   │
│  │                  - Lease TTL  - Dashboards   - PagerDuty    │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                  Frontend (Next.js)                          │   │
│  │  - In-memory access tokens (XSS-proof)                      │   │
│  │  - httpOnly refresh cookies (SameSite=strict)               │   │
│  │  - BFF pattern for auth flows                               │   │
│  │  - CSP with nonce-based scripts                             │   │
│  │  - CSRF double-submit tokens                                │   │
│  │  - Permission-gated UI (backend-enforced)                   │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### CI/CD Security Pipeline (Target)

```
Developer → Git Push → [Pre-commit Hooks]
                            │
                     ┌──────┴──────┐
                     │  CI Pipeline │
                     │              │
                     │ 1. Lint/Format│
                     │ 2. Unit Tests │
                     │ 3. SAST       │
                     │    - gosec    │
                     │    - Semgrep  │
                     │ 4. Secret Scan│
                     │    - TruffleHog│
                     │ 5. Dep Audit  │
                     │    - govulncheck│
                     │    - npm audit │
                     │ 6. License    │
                     │ 7. Build      │
                     │ 8. Container  │
                     │    - Trivy    │
                     │ 9. Integration│
                     │    Tests     │
                     │ 10. DAST     │
                     │    (staging) │
                     └──────┬──────┘
                            │
                     [Staging Deploy]
                            │
                     [Security Review]
                            │
                     [Production Deploy]
                            │
                     [Post-Deploy Verify]
```

### Incident Response Architecture

```
Detection Sources:
├── Application Logs (structured JSON)
├── Audit Trail (hash-chain verified)
├── Security Metrics (Prometheus)
├── Rate Limit Events
├── Authentication Failures
├── SSRF/XSS Detection Events
└── ClamAV Malware Alerts

→ [Log Aggregation] → [SIEM/Correlation] → [Alert Rules]

Alert Escalation:
├── P0 (Critical): PagerDuty immediate page
│   - Mass session revocation triggered
│   - Token reuse detected
│   - Audit chain integrity failure
│   - SSRF attempt to metadata endpoint
│
├── P1 (High): Slack + PagerDuty (15min)
│   - Rate limit bypass detected
│   - Unusual cross-tenant query patterns
│   - Failed MFA attempts > threshold
│   - Malware detected in upload
│
├── P2 (Medium): Slack notification
│   - Login anomaly (new IP/device)
│   - Password reset surge
│   - API key created with broad scope
│
└── P3 (Low): Dashboard metric
    - Failed login attempts
    - Rate limit hits
    - CORS violations
```

---

## J. Prioritized Remediation Roadmap

### Immediate Actions (Week 1-2)

| # | Action | Finding | Effort | Impact |
|---|--------|---------|--------|--------|
| 1 | Apply `SSRFValidator` to webhook delivery | D1 | 2-4 hours | Eliminates SSRF attack vector |
| 2 | Add in-process rate limit fallback | D2 | 4-8 hours | Prevents abuse on Redis outage |
| 3 | Reduce login lockout to 5 attempts | D3 | 30 min | Reduces brute-force window |
| 4 | Add CORS startup validation (reject wildcard+credentials) | D5 | 1-2 hours | Prevents credential theft |

### Short-Term Actions (Weeks 3-6)

| # | Action | Finding | Effort | Impact |
|---|--------|---------|--------|--------|
| 5 | Implement email verification flow | D4 | 2-3 days | Prevents account impersonation |
| 6 | Add explicit tenant check in `ValidateKey()` | D6 | 2-4 hours | Closes cross-tenant API key risk |
| 7 | Add configuration validator (fail on dev defaults in prod) | D12 | 4-8 hours | Prevents credential exposure |
| 8 | Sign OAuth state parameter with HMAC | D7 | 4-8 hours | Prevents redirect manipulation |
| 9 | Add progressive login delays (1s, 2s, 4s...) | D3 | 4-8 hours | Slows automated attacks |
| 10 | Add WebSocket message size limits | D10 | 2-4 hours | Prevents memory exhaustion |

### Medium-Term Actions (Months 2-3)

| # | Action | Finding | Effort | Impact |
|---|--------|---------|--------|--------|
| 11 | Implement password history (last 12) | D8 | 1-2 days | Prevents password cycling |
| 12 | Add refresh token client binding (IP subnet + UA) | D9 | 2-3 days | Reduces session hijacking |
| 13 | Implement SIEM integration for audit events | H-matrix | 3-5 days | Enables real-time alerting |
| 14 | Add `GetByEmailGlobal` timing normalization | D13 | 4-8 hours | Prevents email enumeration |
| 15 | Implement Dependabot for automated dependency updates | H-matrix | 2-4 hours | Reduces supply chain risk |
| 16 | Add automated audit partition creation | D16 | 4-8 hours | Prevents performance degradation |
| 17 | Implement suspicious login notifications | H-matrix | 2-3 days | Alerts users to compromise |

### Long-Term Hardening (Months 4-6+)

| # | Action | Finding | Effort | Impact |
|---|--------|---------|--------|--------|
| 18 | Deploy WAF (OWASP Core Rule Set) | Architecture | 1-2 weeks | External attack surface reduction |
| 19 | Implement service mesh (mTLS between services) | H-matrix | 2-4 weeks | Zero-trust internal networking |
| 20 | Add DAST scanning to staging pipeline | SDLC | 1-2 weeks | Runtime vulnerability detection |
| 21 | Implement runtime security monitoring (Falco) | H-matrix | 1-2 weeks | Container threat detection |
| 22 | Create incident response playbooks | Operations | 2-3 weeks | Reduce MTTR |
| 23 | Migrate CSP to nonce-based script/style injection | D15 | 1-2 weeks | Stronger CSP enforcement |
| 24 | Implement HashiCorp Vault for dynamic secrets | Architecture | 2-4 weeks | Centralized secret management |
| 25 | Add token revocation list (in-memory, short TTL) | Architecture | 1-2 days | Immediate access revocation |

---

## Top 10 Critical Security Risks

| Rank | Risk | Severity | Exploitability | Impact |
|------|------|----------|---------------|--------|
| 1 | **Webhook SSRF** - internal service/cloud metadata access | Critical | Easy (authenticated user) | Cloud credential theft |
| 2 | **Rate limit fail-open** - all limits bypass on Redis outage | High | Medium (requires Redis failure) | Brute-force, abuse |
| 3 | **Login lockout too permissive** - 20 attempts/15min | High | Easy (automated) | Account takeover |
| 4 | **No email verification** - unverified account creation | High | Easy (no auth needed) | Impersonation, phishing |
| 5 | **CORS wildcard risk** - misconfiguration enables credential theft | High | Medium (requires misconfig) | Session hijacking |
| 6 | **API key tenant gap** - missing explicit tenant validation | High | Medium (requires valid key) | Cross-tenant access |
| 7 | **OAuth state unsigned** - redirect manipulation | Medium | Medium (social engineering) | Phishing |
| 8 | **Refresh token unbound** - no IP/fingerprint validation | Medium | Medium (requires token theft) | Session hijacking |
| 9 | **WebSocket messages unvalidated** - no size/schema limits | Medium | Easy (authenticated user) | DoS, injection |
| 10 | **Hardcoded dev defaults** - known credentials if env vars missing | Medium | Easy (if misconfigured) | Full system access |

## Top 10 Architectural Security Improvements

| Rank | Improvement | Current State | Target State | Business Impact |
|------|------------|--------------|-------------|----------------|
| 1 | **In-process rate limit fallback** | Redis-only, fail-open | Hybrid (Redis + local token bucket) | Resilient abuse prevention |
| 2 | **SSRF protection on all outbound HTTP** | Data connectors only | All outbound requests (webhooks, integrations) | Eliminates internal network exposure |
| 3 | **Production config validation** | Defaults allow startup | Fail-fast on missing critical secrets | Prevents accidental credential exposure |
| 4 | **SIEM integration** | Audit logs stored, no alerting | Real-time correlation and alerting | Reduces incident detection time |
| 5 | **Email verification flow** | Immediate activation | Verification-gated activation | Prevents impersonation attacks |
| 6 | **Service mesh (mTLS)** | Plain HTTP between services | Mutual TLS with identity verification | Zero-trust internal network |
| 7 | **WAF deployment** | No external WAF | OWASP CRS at edge | Filters known attack patterns |
| 8 | **Dynamic secrets (Vault)** | Static env vars, K8s Secrets | Short-lived, auto-rotated credentials | Limits credential exposure window |
| 9 | **Runtime security monitoring** | No container monitoring | Falco/Sysdig runtime policies | Detects container breakout attempts |
| 10 | **Incident response playbooks** | No formal procedures | Documented runbooks + automated response | Reduces MTTR from hours to minutes |

---

*Assessment conducted via static code analysis of the Clario360 repository. Findings are based on code review and architectural analysis. Dynamic testing (DAST/penetration testing) is recommended to validate findings and discover runtime-specific vulnerabilities.*
