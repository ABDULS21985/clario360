# Clario360 Platform — Complete Architecture

## High-Level Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│  BROWSER (Next.js 14 App Router)                                                     │
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────────────────────────────┐  │
│  │ Zustand Stores   │  │ React Query v5   │  │ WebSocket (gorilla/websocket)      │  │
│  │ auth, sidebar,   │  │ Server state     │  │ Real-time notifications, cyber,    │  │
│  │ notifications,   │  │ caching &        │  │ executive alerts                   │  │
│  │ command palette,  │  │ invalidation     │  │ Exponential backoff reconnect      │  │
│  │ realtime         │  │                  │  │                                     │  │
│  └─────────────────┘  └──────────────────┘  └─────────────────────────────────────┘  │
│  Access Token: in-memory only (Zustand)  │  Refresh Token: httpOnly cookie (BFF)     │
│  BFF routes: /api/auth/session, /api/auth/refresh (Next.js API routes → gateway)     │
└──────────────────────────────────┬───────────────────────────────────────────────────┘
                                   │ HTTPS
┌──────────────────────────────────▼───────────────────────────────────────────────────┐
│  API GATEWAY  :8080  (Chi v5)                                                        │
│                                                                                      │
│  Global Middleware Chain:                                                             │
│  Recovery → RequestID → SecurityHeaders → CORS → BodyLimit → Logging                │
│  → OpenTelemetry Tracing → Timeout                                                   │
│                                                                                      │
│  Per-Route Middleware:                                                                │
│  [JWT Auth (RS256)] → ProxyHeaders → Redis Rate Limit (per-tenant sliding window)   │
│  → Prometheus Metrics (14 metrics) → ProxyLogging → Circuit Breaker (gobreaker)     │
│  → Reverse Proxy                                                                     │
│                                                                                      │
│  WebSocket Routes: /ws/v1/{notifications, cyber, visus}                              │
└──┬──────┬────────┬───────┬───────┬────────┬───────┬───────┬───────┬───────┬──────────┘
   │      │        │       │       │        │       │       │       │       │
```

## Backend Microservices

```
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│   IAM    │ │  AUDIT   │ │ WORKFLOW │ │  NOTIF   │ │   FILE   │
│  :8081   │ │  :8084   │ │  :8083   │ │  :8090   │ │  :8091   │
│          │ │          │ │          │ │          │ │          │
│ Users    │ │ Hash-    │ │ BPMN     │ │ Email    │ │ Upload   │
│ Roles    │ │ chain    │ │ Engine   │ │ In-app   │ │ AES Enc  │
│ Tenants  │ │ integrity│ │ Human    │ │ Webhook  │ │ ClamAV   │
│ MFA/TOTP │ │ Export   │ │ tasks    │ │ WebSocket│ │ scan     │
│ OAuth    │ │ to MinIO │ │ Parallel │ │ Slack    │ │ Presigned│
│ API Keys │ │ Masking  │ │ gateways │ │ Jira     │ │ URLs     │
│ JWT RS256│ │          │ │ Timers   │ │ Teams    │ │ MinIO    │
└────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘
     │            │            │            │            │
     │       ┌────▼────────────▼────────────▼────────────▼──────┐
     │       │              Apache Kafka :9092                   │
     │       │      30 topics · KRaft mode · CloudEvents        │
     │       │      Dead-letter queues · Idempotency             │
     │       └──────────────────────────────────────────────────┘
     │            │            │            │            │
┌────▼─────┐ ┌───▼──────┐ ┌───▼──────┐ ┌───▼──────┐ ┌───▼──────┐
│  CYBER   │ │   DATA   │ │   ACTA   │ │   LEX    │ │  VISUS   │
│  :8085   │ │  :8086   │ │  :8087   │ │  :8088   │ │  :8089   │
│          │ │          │ │          │ │          │ │          │
│ Assets   │ │ Sources  │ │ Meetings │ │Contracts │ │ KPIs     │
│ Threats  │ │ Pipelines│ │ Minutes  │ │ Clauses  │ │Dashboards│
│ Alerts   │ │ Quality  │ │ Actions  │ │ Risk     │ │ Reports  │
│ Rules    │ │ Lineage  │ │Committee │ │Compliance│ │ Widgets  │
│ CTEM     │ │ Dark Data│ │Compliance│ │ NLP      │ │ Cross-   │
│ DSPM     │ │Contradict│ │ AI Gen   │ │ Entity   │ │ suite    │
│ UEBA     │ │ ETL      │ │          │ │ Extract  │ │ aggregat │
│ vCISO    │ │ PII scan │ │          │ │          │ │ Alerts   │
│ MITRE    │ │          │ │          │ │          │ │ Escalate │
│ IoCs     │ │Connectors│ │          │ │          │ │          │
│ Risk     │ │ PG,MySQL │ │          │ │          │ │          │
│ Remediate│ │ CH,Spark │ │          │ │          │ │          │
│ Enrichmt │ │ Hive,HDFS│ │          │ │          │ │          │
│ LLM(5)  │ │ S3,CSV   │ │          │ │          │ │          │
└──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘
```

## 5 Business Suites

| Suite | Service | Database | Purpose |
|---|---|---|---|
| **Cybersecurity** | cyber-service :8085 | `cyber_db` (23 migrations) | Full SOC: asset discovery, threat intel, detection rules (Sigma), CTEM, DSPM, UEBA, vCISO (LLM-powered), MITRE ATT&CK, IoC matching, risk scoring, remediation |
| **Data Intelligence** | data-service :8086 | `data_db` (6 migrations) | Data management: 10+ connectors (PG, MySQL, ClickHouse, Spark, Hive, HDFS, S3, CSV), ETL pipelines, quality rules, lineage graphs, dark data/PII discovery, contradiction detection |
| **Board Governance (Acta)** | acta-service :8087 | `acta_db` (3 migrations) | Board meetings, AI-generated minutes, action items, committees, compliance tracking |
| **Legal Operations (Lex)** | lex-service :8088 | `lex_db` (2 migrations) | Contract management, clause extraction (NLP), risk analysis, compliance monitoring |
| **Executive Intelligence (Visus)** | visus-service :8089 | `visus_db` (3 migrations) | KPI engine, executive dashboards, cross-suite data aggregation, report generation, alert correlation |

## All Backend Services

| Service | Port | Admin Port | Purpose |
|---|---|---|---|
| `api-gateway` | 8080 | 9080 | HTTP gateway: JWT auth, rate limiting, circuit breaker, reverse proxy, WebSocket proxy |
| `iam-service` | 8081 | — | Identity & Access Management: users, roles, permissions, MFA (TOTP), OAuth, API keys, tenant onboarding |
| `event-bus` | — | — | Kafka event routing and processing |
| `workflow-engine` | 8083 | — | Workflow orchestration with human tasks, timers, parallel gateways, event/condition/service tasks |
| `audit-service` | 8084 | — | Centralized audit log with hash-chain integrity, export to MinIO |
| `cyber-service` | 8085 | — | Cybersecurity suite (assets, threats, CTEM, DSPM, vCISO, UEBA, MITRE ATT&CK, threat feeds) |
| `data-service` | 8086 | 9086 | Data Intelligence suite (connectors, pipelines, quality, lineage, dark data, contradictions) |
| `acta-service` | 8087 | 9087 | Board Governance suite (meetings, AI minutes, action items, committees, compliance) |
| `lex-service` | 8088 | 9088 | Legal Operations suite (contracts, clause extraction, risk scoring, compliance) |
| `visus-service` | 8089 | 9089 | Executive Intelligence suite (KPIs, dashboards, reports, cross-suite aggregation) |
| `notification-service` | 8090 | — | Multi-channel notifications: email, in-app, webhooks, WebSocket, Slack/Jira/Teams/ServiceNow |
| `file-service` | 8091 | — | File management: upload, virus scan (ClamAV), encryption (AES), MinIO storage, presigned URLs |
| `migrator` | — | — | Database migration runner (golang-migrate) |
| `data-seeder` | — | — | Development data seeder (500 assets, 200 vulns, etc.) |

## API Gateway Route Table

### HTTP Routes

| URL Prefix | Backend Service | Auth | Rate Limit Group |
|---|---|---|---|
| `/.well-known` | iam-service:8081 | Public | auth |
| `/api/v1/auth` | iam-service:8081 | Public | auth |
| `/api/v1/onboarding` | iam-service:8081 | Public | auth |
| `/api/v1/invitations` | iam-service:8081 | Public | auth |
| `/api/v1/ai` | iam-service:8081 | JWT required | admin |
| `/api/v1/users` | iam-service:8081 | JWT required | write |
| `/api/v1/roles` | iam-service:8081 | JWT required | admin |
| `/api/v1/tenants` | iam-service:8081 | JWT required | admin |
| `/api/v1/api-keys` | iam-service:8081 | JWT required | write |
| `/api/v1/notebooks` | iam-service:8081 | JWT required | write |
| `/api/v1/audit` | audit-service:8084 | JWT required | read |
| `/api/v1/workflows` | workflow-engine:8083 | JWT required | write |
| `/api/v1/notifications` | notification-service:8090 | JWT required | write |
| `/api/v1/integrations` | notification-service:8090 | Public | write |
| `/api/v1/files/upload` | file-service:8091 | JWT required | upload (100MB, 120s timeout) |
| `/api/v1/files` | file-service:8091 | JWT required | read |
| `/api/v1/cyber` | cyber-service:8085 | JWT required | write |
| `/api/v1/rca` | cyber-service:8085 | JWT required | write |
| `/api/v1/data` | data-service:8086 | JWT required | write |
| `/api/v1/acta` | acta-service:8087 | JWT required | write |
| `/api/v1/lex` | lex-service:8088 | JWT required | write |
| `/api/v1/visus` | visus-service:8089 | JWT required | read |

### WebSocket Routes

| WS Prefix | Backend Service | Auth |
|---|---|---|
| `/ws/v1/notifications` | notification-service:8090 | JWT required |
| `/ws/v1/cyber` | cyber-service:8085 | JWT required |
| `/ws/v1/visus` | visus-service:8089 | JWT required |

### Gateway Middleware Chain

```
Recovery → RequestID → SecurityHeaders → CORS → BodyLimit → Logging → OTel Tracing → Timeout
  → (per-route): [ProxyAuth (JWT)] → ProxyHeaders → RateLimit (Redis, per-tenant) → Metrics → ProxyLogging → SpanEnricher
    → Reverse Proxy / WebSocket Proxy
```

Circuit Breaker: sony/gobreaker per service (failure threshold, open timeout, half-open successes configurable)

## Shared Infrastructure

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         SHARED INFRASTRUCTURE                                    │
│                                                                                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐               │
│  │  PostgreSQL 16   │  │   Redis 7.2      │  │  Apache Kafka    │               │
│  │  :5432           │  │   :6379          │  │  :9092 (KRaft)   │               │
│  │                  │  │                  │  │                  │               │
│  │  8 databases:    │  │  Sessions        │  │  30 topics       │               │
│  │  • platform_core │  │  Rate limiting   │  │  CloudEvents     │               │
│  │  • cyber_db      │  │  Caching         │  │  Dead-letter Qs  │               │
│  │  • data_db       │  │  Queue buffers   │  │  Idempotency     │               │
│  │  • acta_db       │  │                  │  │  Schema Registry │               │
│  │  • lex_db        │  │                  │  │  :8081           │               │
│  │  • visus_db      │  │                  │  │                  │               │
│  │  • audit_db      │  │                  │  │                  │               │
│  │  • notification  │  │                  │  │                  │               │
│  │    _db           │  │                  │  │                  │               │
│  │                  │  │                  │  │                  │               │
│  │  RLS per tenant  │  │                  │  │                  │               │
│  │  pgcrypto        │  │                  │  │                  │               │
│  │  uuid-ossp       │  │                  │  │                  │               │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘               │
│                                                                                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐               │
│  │   MinIO          │  │   ClamAV         │  │   Keycloak       │               │
│  │   :9000 / :9001  │  │   :3310          │  │   :8180          │               │
│  │                  │  │                  │  │                  │               │
│  │  File storage    │  │  Virus scanning  │  │  External IdP    │               │
│  │  Audit exports   │  │  Upload safety   │  │  OIDC/SAML       │               │
│  │  Presigned URLs  │  │                  │  │  OAuth flows      │               │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘               │
│                                                                                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐               │
│  │  Prometheus      │  │   Grafana        │  │   Jaeger         │               │
│  │  :9099           │  │   :3000          │  │   :16686         │               │
│  │                  │  │                  │  │                  │               │
│  │  Metrics scrape  │  │  Dashboards      │  │  Distributed     │               │
│  │  Alert rules     │  │  Per-service     │  │  tracing (OTLP)  │               │
│  │  ServiceMonitor  │  │  panels          │  │  :4317           │               │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘               │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Databases and Migrations

There are **8 separate PostgreSQL databases** (database-per-service pattern), all using Row-Level Security (RLS) with `app.current_tenant_id` session variable.

| Database | Service | Migration Count | Key Domain |
|---|---|---|---|
| `platform_core` | iam-service, file-service | 11 migrations | Users, roles, tenants, file storage, AI governance models, compute benchmarks |
| `cyber_db` | cyber-service | 23 migrations | Assets, vulnerabilities, threats, alerts, rules, CTEM, DSPM, vCISO governance, UEBA, threat feeds, MITRE, RLS |
| `data_db` | data-service | 6 migrations | Data sources, pipelines, quality, lineage, dark data, connector types |
| `acta_db` | acta-service | 3 migrations | Meetings, committees, minutes, action items, compliance |
| `lex_db` | lex-service | 2 migrations | Contracts, clauses, documents, compliance |
| `visus_db` | visus-service | 3 migrations | KPIs, dashboards, reports, executive views |
| `audit_db` | audit-service | 2 migrations | Audit log with hash chain, RLS |
| `notification_db` | notification-service | 3 migrations | Notifications, webhooks, integrations |

All databases are initialized by `deploy/docker/init-databases.sql` on first Docker start, with `pgcrypto` and `uuid-ossp` extensions.

## Security Architecture

```
┌─ AUTHENTICATION ──────────────────────────────────────────────────┐
│  • RS256 JWT (15min access token, 168h refresh token)            │
│  • Access token: in-memory only (never localStorage/cookies)      │
│  • Refresh token: httpOnly secure cookie via BFF                  │
│  • MFA: TOTP (RFC 6238) with backup codes                        │
│  • OAuth: Keycloak (OIDC/SAML) external IdP support              │
│  • API keys: per-tenant, scoped permissions                       │
│  • Lockout: 20 failed attempts max                                │
└───────────────────────────────────────────────────────────────────┘

┌─ AUTHORIZATION ───────────────────────────────────────────────────┐
│  • RBAC: roles → permissions (wildcard support: "cyber.*")       │
│  • Row-Level Security: PostgreSQL RLS on every table              │
│  • Tenant isolation: app.current_tenant_id session var            │
│  • Gateway: per-route auth requirements (public vs JWT)           │
│  • Frontend: PermissionRedirect guards on routes                  │
└───────────────────────────────────────────────────────────────────┘

┌─ DATA SECURITY ───────────────────────────────────────────────────┐
│  • File encryption: AES-256 at rest (MinIO)                       │
│  • Virus scanning: ClamAV on every upload                         │
│  • Audit trail: hash-chain integrity (tamper-proof)               │
│  • PII detection & masking in data discovery                      │
│  • CSRF protection, XSS sanitization, injection prevention        │
│  • SSRF protection, rate limiting, security headers               │
└───────────────────────────────────────────────────────────────────┘
```

## Authentication Flow

```
Browser
  │
  ├── Access Token (in-memory via Zustand auth-store)
  │   └── Attached as Authorization: Bearer <token> by axios interceptor
  │
  └── Refresh Token (httpOnly cookie via Next.js BFF)
      ├── POST /api/auth/session  → Next.js BFF → calls gateway /api/v1/auth/refresh
      └── POST /api/auth/refresh  → Next.js BFF

API Gateway (port 8080)
  └── ProxyAuth middleware → validates RS256 JWT → sets user/tenant in context
      └── Forward to upstream service with X-User-ID, X-Tenant-ID, X-User-Roles headers

IAM Service (port 8081)
  ├── /api/v1/auth/login → bcrypt verify → issue RS256 JWT (15m access, 168h refresh)
  ├── /api/v1/auth/mfa  → TOTP (pquerna/otp) verify
  ├── /api/v1/auth/refresh → rotate refresh token → issue new access token
  └── Shared JWT key: AUTH_RSA_PRIVATE_KEY_PEM / AUTH_RSA_PUBLIC_KEY_PEM (PEM content in env)
```

## AI/ML Capabilities

```
┌─ LLM INTEGRATION (vCISO) ────────────────────────────────────────┐
│  Providers: OpenAI │ Anthropic │ Azure OpenAI │ LlamaCPP │ BitNet│
│  Intent classifier → Entity extractor → Tool router               │
│  Context manager → Response formatter → Suggestion engine         │
│  Predictive analytics: risk forecasting, threat prediction        │
└───────────────────────────────────────────────────────────────────┘

┌─ AI GOVERNANCE ───────────────────────────────────────────────────┐
│  Model registry → Lifecycle (promote/rollback)                    │
│  Shadow testing → Drift detection (PSI)                           │
│  Compute benchmarking → Explainability (natural language)         │
│  Prediction logging → Validation framework                        │
└───────────────────────────────────────────────────────────────────┘

┌─ NLP / ANALYTICS ────────────────────────────────────────────────┐
│  Acta: AI-generated meeting minutes, action extraction            │
│  Lex: clause extraction, entity extraction, missing clause detect │
│  Cyber: Sigma rule evaluation, anomaly detection, UEBA            │
│  Data: contradiction detection, entity linking                    │
│  Visus: narrative report generation                               │
└───────────────────────────────────────────────────────────────────┘
```

## Frontend Architecture

### App Router Pages

```
app/
├── (auth)/                     # Unauthenticated pages
│   ├── login/
│   ├── register/
│   ├── forgot-password/
│   ├── reset-password/
│   ├── verify-email/
│   ├── invite/
│   └── callback/               # OAuth callback
├── (onboarding)/               # Post-registration onboarding
├── (dashboard)/                # Protected dashboard pages
│   ├── page.tsx                # Root dashboard (/)
│   ├── dashboard/              # Dashboard home
│   ├── cyber/                  # Cybersecurity suite
│   │   ├── alerts/             # Alert management
│   │   ├── assets/             # Asset inventory
│   │   ├── threats/            # Threat intelligence
│   │   ├── rules/              # Detection rules (Sigma)
│   │   ├── threat-feeds/       # Threat feed management
│   │   ├── indicators/         # IoC indicators
│   │   ├── ctem/               # Continuous Threat Exposure Mgmt
│   │   ├── dspm/               # Data Security Posture Mgmt
│   │   ├── ueba/               # User behavior analytics
│   │   ├── vciso/              # Virtual CISO
│   │   ├── remediation/        # Remediation tracking
│   │   ├── risk-heatmap/       # Risk heatmap
│   │   ├── mitre/              # MITRE ATT&CK mapping
│   │   ├── events/             # Security events
│   │   └── analytics/          # Cyber analytics
│   ├── data/                   # Data Intelligence suite
│   │   ├── sources/            # Data source management
│   │   ├── pipelines/          # Pipeline management
│   │   ├── quality/            # Data quality monitoring
│   │   ├── models/             # Data models
│   │   ├── lineage/            # Data lineage
│   │   ├── dark-data/          # Dark data discovery
│   │   ├── contradictions/     # Contradiction detection
│   │   └── analytics/          # Data analytics
│   ├── acta/                   # Board Governance
│   │   ├── meetings/
│   │   ├── committees/
│   │   ├── action-items/
│   │   └── compliance/
│   ├── lex/                    # Legal Operations
│   │   ├── contracts/
│   │   ├── documents/
│   │   └── compliance/
│   ├── visus/                  # Executive Intelligence
│   │   ├── kpis/
│   │   ├── alerts/
│   │   └── reports/
│   ├── admin/                  # Admin panel
│   │   ├── users/
│   │   ├── roles/
│   │   ├── audit/
│   │   ├── ai-governance/
│   │   ├── tenants/
│   │   ├── api-keys/
│   │   ├── integrations/
│   │   ├── invitations/
│   │   ├── notifications/
│   │   ├── settings/
│   │   └── workflows/
│   ├── workflows/              # Workflow management
│   ├── notifications/          # Notification center
│   ├── files/                  # File management
│   ├── notebooks/              # Jupyter notebooks
│   └── settings/               # User settings
└── api/                        # BFF Next.js API routes
    ├── auth/session/           # Session endpoint
    ├── auth/refresh/           # Token refresh
    └── health/                 # Health check
```

### Component Library

```
components/
├── ui/                  # shadcn/ui base components (60+ primitives)
├── shared/              # KPI cards, charts, data tables, forms, modals, timeline, detail panel
│   ├── charts/          # bar-chart, area-chart, line-chart, pie-chart, gauge-chart (recharts)
│   ├── forms/           # search-input, form-field, multi-select, combobox, date-range-picker, file-upload
│   └── ...              # severity-indicator, status-badge, timeline, confirm-dialog
├── layout/              # Sidebar, Header, Breadcrumbs, CommandPalette, NotificationDropdown, UserMenu
├── auth/                # Login form, register form, MFA setup dialog, OAuth buttons
├── cyber/               # Cybersecurity-specific components
├── suites/              # Suite-specific shared components
├── workflows/           # Workflow task forms, instance viewer
├── notifications/       # Notification list, category tabs, preference settings
├── realtime/            # ConnectionStatusBanner, real-time data components
└── providers/           # WebSocketProvider, QueryClientProvider, ThemeProvider
```

### State Management

| Store | Purpose |
|---|---|
| `auth-store.ts` | Auth state: user, permissions, access token (in-memory), tenant |
| `sidebar-store.ts` | Sidebar collapsed/expanded state (localStorage persist) |
| `notification-store.ts` | Notification count, WebSocket notification state |
| `command-palette-store.ts` | Command palette open/close, search state |
| `realtime-store.ts` | Topic → queryKey registry for WebSocket-triggered React Query invalidations |

## Deployment Architecture

```
┌─ LOCAL DEV ─────────────────────┐   ┌─ PRODUCTION ─────────────────┐
│  PM2 (ecosystem.local.js)       │   │  Kubernetes (Helm chart)     │
│  Docker Compose (infra only)    │   │  Terraform (infra-as-code)   │
│  Hot-reload for all services    │   │                              │
│  Frontend: localhost:3000       │   │  11 service deployments      │
│  Gateway: localhost:8080        │   │  Network policies            │
│                                 │   │  Resource quotas             │
│                                 │   │  Priority classes            │
│                                 │   │  CronJobs (scheduled tasks)  │
│                                 │   │  ServiceMonitor (Prometheus) │
│                                 │   │  Migration & Seed jobs       │
│                                 │   │  LLM inference deployment    │
│                                 │   │                              │
│                                 │   │  Environments:               │
│                                 │   │  dev → staging → production  │
└─────────────────────────────────┘   └──────────────────────────────┘
```

### Docker Compose Services

| Container | Image | Port | Purpose |
|---|---|---|---|
| `clario360-postgres` | postgres:16-alpine | 5432 | Primary database (all 8 DBs) |
| `clario360-redis` | redis:7.2-alpine | 6379 | Cache, rate limiting, sessions |
| `clario360-kafka` | bitnami kafka:4.0.0 (KRaft) | 9092, 9094 | Event streaming (30 topics) |
| `schema-registry` | cp-schema-registry:7.6.0 | 8081 | Kafka schema registry |
| `clario360-minio` | minio/minio:latest | 9000, 9001 | S3-compatible object storage |
| `clario360-keycloak` | keycloak:24.0 | 8180 | Identity provider (OIDC/SAML) |
| `clario360-prometheus` | prom/prometheus:v2.53.0 | 9099 | Metrics collection |
| `clario360-grafana` | grafana/grafana:11.1.0 | 3000 | Dashboards |
| `clario360-clamav` | clamav/clamav:stable | 3310 | Virus scanning |
| `clario360-jaeger` | jaegertracing/all-in-one:1.58 | 4317, 16686 | Distributed tracing |

## Data Flow Summary

```
User Request → Next.js BFF (auth) → API Gateway :8080
  → JWT validation → Rate limit check → Circuit breaker
  → Route to target microservice
  → Service processes request (with tenant RLS)
  → Emits Kafka event (CloudEvents format)
  → Consumer(s) in other services react:
      • Audit service logs action
      • Notification service sends alerts
      • Workflow engine triggers tasks
      • Visus aggregates metrics
  → Response returned through gateway to browser
  → WebSocket pushes real-time updates to connected clients
```

## Technology Stack Summary

### Backend
- **Language:** Go 1.25
- **Router:** Chi v5
- **Database:** PostgreSQL 16 (pgx/v5 driver), Row-Level Security
- **Cache:** Redis 7.2 (go-redis/v9)
- **Messaging:** Apache Kafka (sarama), CloudEvents
- **Object Storage:** MinIO (minio-go/v7)
- **Auth:** RS256 JWT (golang-jwt/jwt/v5), TOTP MFA (pquerna/otp)
- **Observability:** Prometheus, OpenTelemetry (OTLP), Zerolog
- **Testing:** testcontainers-go

### Frontend
- **Framework:** Next.js 14 (App Router)
- **Language:** TypeScript 5
- **UI:** Tailwind CSS 3, shadcn/ui (60+ components)
- **State:** Zustand 4, TanStack Query 5
- **Forms:** react-hook-form 7, Zod 3
- **Charts:** Recharts 2, D3 7
- **Testing:** Vitest, MSW, Playwright (E2E)

### Infrastructure
- **Container Orchestration:** Kubernetes (Helm)
- **Infrastructure-as-Code:** Terraform
- **CI/CD:** Docker multi-stage builds
- **Monitoring:** Prometheus + Grafana + Jaeger
