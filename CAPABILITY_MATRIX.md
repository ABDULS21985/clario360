# Clario360 Platform — Comprehensive Capability & Feature Matrix

> **Generated:** 2026-03-08
> **Coverage:** 100% of codebase (backend, frontend, infrastructure, databases)

---

## Table of Contents

1. [Platform Overview](#1-platform-overview)
2. [Service Architecture Matrix](#2-service-architecture-matrix)
3. [API Gateway](#3-api-gateway)
4. [IAM Service — Identity & Access Management](#4-iam-service)
5. [Cyber Service — Security & Risk Management](#5-cyber-service)
6. [Data Service — Data Governance & Quality](#6-data-service)
7. [Workflow Engine — Automation & Orchestration](#7-workflow-engine)
8. [Audit Service — Compliance & Change Logging](#8-audit-service)
9. [Notification Service — Alert & Message Delivery](#9-notification-service)
10. [File Service — File Management & Antivirus](#10-file-service)
11. [Acta Service — Board & Committee Governance](#11-acta-service)
12. [Lex Service — Legal & Contract Management](#12-lex-service)
13. [Visus Service — Visualization & KPI Analytics](#13-visus-service)
14. [Event Bus — Central Event Routing](#14-event-bus)
15. [AI Governance — Model Lifecycle Management](#15-ai-governance)
16. [Frontend Application](#16-frontend-application)
17. [Database Architecture](#17-database-architecture)
18. [Infrastructure & DevOps](#18-infrastructure--devops)
19. [Cross-Cutting Concerns](#19-cross-cutting-concerns)
20. [Statistics Summary](#20-statistics-summary)

---

## 1. Platform Overview

Clario360 is an enterprise security, governance, and data intelligence platform built as a Go microservices backend with a Next.js 14 frontend.

| Dimension | Detail |
|-----------|--------|
| **Backend Language** | Go (module: `github.com/clario360/platform`) |
| **Frontend Framework** | Next.js 14 (App Router), TypeScript, Tailwind CSS |
| **Router** | chi v5 (not gin) |
| **Auth Standard** | RS256 JWT via custom `auth.JWTManager` |
| **Databases** | 8 PostgreSQL databases (multi-tenant, RLS) |
| **Message Broker** | Apache Kafka |
| **Cache / Sessions** | Redis |
| **Object Storage** | MinIO (S3-compatible) |
| **Metrics** | Prometheus (per-service registry) |
| **Tracing** | OpenTelemetry |
| **Frontend State** | Zustand + React Query |
| **UI Components** | shadcn/ui + Recharts |

---

## 2. Service Architecture Matrix

| # | Service | Binary | HTTP Port | Admin Port | Database | Status |
|---|---------|--------|-----------|------------|----------|--------|
| 1 | API Gateway | `cmd/api-gateway` | 8080 | 9090 | — (stateless) | ✅ |
| 2 | IAM Service | `cmd/iam-service` | 8081 | 9081 | `platform_core` | ✅ |
| 3 | Cyber Service | `cmd/cyber-service` | 8085 | 9085 | `cyber_db` | ✅ |
| 4 | Data Service | `cmd/data-service` | 8086 | 9086 | `data_db` | ✅ |
| 5 | Workflow Engine | `cmd/workflow-engine` | 8083 | 9083 | `platform_core` | ✅ |
| 6 | Audit Service | `cmd/audit-service` | 8084 | 9084 | `audit_db` | ✅ |
| 7 | Notification Service | `cmd/notification-service` | 8090 | 9090 | `notification_db` | ✅ |
| 8 | File Service | `cmd/file-service` | 8091 | 9091 | `platform_core` | ✅ |
| 9 | Acta Service | `cmd/acta-service` | 8087 | 9087 | `acta_db` | ✅ |
| 10 | Lex Service | `cmd/lex-service` | 8088 | 9088 | `lex_db` | ✅ |
| 11 | Visus Service | `cmd/visus-service` | 8089 | 9089 | `visus_db` | ✅ |
| 12 | Event Bus | `cmd/event-bus` | — | — | — | ✅ |

### Utility Binaries

| Tool | Binary | Purpose |
|------|--------|---------|
| Migrator | `cmd/migrator` | Database schema migration runner |
| Data Seeder | `cmd/data-seeder` | Development data seeding |
| Seeder | `cmd/seeder` | General-purpose data seeder |

---

## 3. API Gateway

**Package:** `internal/gateway/`

### Capability Matrix

| Category | Feature | Details |
|----------|---------|---------|
| **Routing** | Dynamic service proxy | Routes to 11 backend services via configurable registry |
| **Routing** | WebSocket proxy | `/ws/v1/notifications/`, `/ws/v1/cyber/vciso-chat/` |
| **Routing** | Path-based routing | Prefix matching with strip-prefix rewrite |
| **Security** | JWT validation | RS256 token verification on protected routes |
| **Security** | Optional auth | Public routes with optional token extraction |
| **Security** | CORS | Configurable allowed origins |
| **Security** | Security headers | X-Frame-Options, CSP, HSTS, etc. |
| **Security** | Body size limit | Global default + per-route overrides |
| **Rate Limiting** | Per-tenant buckets | Separate limits for auth, read, write, admin, upload, WebSocket |
| **Rate Limiting** | Redis-backed | Distributed rate limiting across instances |
| **Resilience** | Circuit breaker | Per-service with configurable failure threshold, open timeout |
| **Resilience** | Request timeout | Per-route configurable timeouts |
| **Resilience** | Panic recovery | Middleware-level panic recovery |
| **Observability** | Request ID | Unique ID generation + propagation |
| **Observability** | Structured logging | Request/response logging with redaction |
| **Observability** | Prometheus metrics | 14 custom metrics (request count, latency, rate limit, circuit breaker) |
| **Observability** | Distributed tracing | OpenTelemetry span enrichment |
| **Admin** | Health checks | `/healthz` (liveness), `/readyz` (readiness) |
| **Admin** | Gateway status | `/api/v1/gateway/status` — circuit breaker state per service |

### Rate Limit Buckets

| Bucket | Scope | Default |
|--------|-------|---------|
| Auth | Authentication endpoints | 20/min |
| Read | GET requests | Configurable |
| Write | POST/PUT/DELETE requests | Configurable |
| Admin | Admin endpoints | Configurable |
| Upload | File upload endpoints | Configurable |
| WebSocket | WebSocket connections | Configurable |

---

## 4. IAM Service

**Package:** `internal/iam/`, `internal/onboarding/`

### Capability Matrix

| Category | Feature | Endpoints | Details |
|----------|---------|-----------|---------|
| **Authentication** | Email/password login | `POST /api/v1/auth/login` | bcrypt hashing, session creation |
| **Authentication** | User registration | `POST /api/v1/auth/register` | Email + password with validation |
| **Authentication** | Token refresh | `POST /api/v1/auth/refresh` | Rotate access + refresh tokens |
| **Authentication** | Logout | `POST /api/v1/auth/logout` | Invalidate session + tokens |
| **Authentication** | Password reset request | `POST /api/v1/auth/forgot-password` | Email-based token generation |
| **Authentication** | Password reset | `POST /api/v1/auth/reset-password` | Token-verified password change |
| **MFA** | TOTP verification | `POST /api/v1/auth/verify-mfa` | Time-based one-time password |
| **MFA** | Enable MFA | `POST /api/v1/users/me/mfa/enable` | Generate TOTP secret + QR code |
| **MFA** | Verify MFA setup | `POST /api/v1/users/me/mfa/verify-setup` | Confirm TOTP setup |
| **MFA** | Disable MFA | `POST /api/v1/users/me/mfa/disable` | Remove TOTP with verification |
| **OAuth/OIDC** | Discovery | `GET /.well-known/openid-configuration` | OIDC discovery document |
| **OAuth/OIDC** | JWKS | `GET /.well-known/jwks.json` | Public key set |
| **OAuth/OIDC** | Authorize | `GET /api/v1/auth/oauth/authorize` | Authorization endpoint |
| **OAuth/OIDC** | Token | `POST /api/v1/auth/oauth/token` | Token exchange |
| **OAuth/OIDC** | Userinfo | `GET /api/v1/auth/oauth/userinfo` | User profile endpoint |
| **User Mgmt** | Profile | `GET /api/v1/users/me` | Current user profile |
| **User Mgmt** | Change password | `PUT /api/v1/users/me/password` | Authenticated password change |
| **User Mgmt** | Sessions | `GET/DELETE /api/v1/users/me/sessions` | Session listing + revocation |
| **User Mgmt** | CRUD | `GET/POST/PUT/DELETE /api/v1/users` | Full user lifecycle |
| **User Mgmt** | Status management | `PUT /api/v1/users/{id}/status` | Activate/deactivate/suspend |
| **Role Mgmt** | CRUD | `GET/POST/PUT/DELETE /api/v1/roles` | Role lifecycle |
| **Role Mgmt** | Permission assignment | JSON permission arrays | Wildcard matching (`*`) |
| **Role Mgmt** | User-role binding | `GET/POST/DELETE /api/v1/users/{id}/roles` | Role assignment/removal |
| **Tenant Mgmt** | CRUD | `GET/POST/PUT /api/v1/tenants` | Tenant lifecycle |
| **Tenant Mgmt** | Status management | `PUT /api/v1/tenants/{id}/status` | Activate/suspend/trial |
| **API Keys** | CRUD | `GET/POST/DELETE /api/v1/api-keys` | Key generation + revocation |
| **API Keys** | Permission scoping | JSONB permissions | Per-key permission matrix |
| **Onboarding** | Self-service registration | `POST /api/v1/onboarding/register` | 5/hour rate limit |
| **Onboarding** | Email verification | `POST /api/v1/onboarding/verify-email` | OTP verification |
| **Onboarding** | Setup wizard | `POST /api/v1/onboarding/wizard/*` | Organization, branding, team, suites |
| **Onboarding** | Tenant provisioning | `POST /api/v1/admin/tenants/provision` | DB creation, user setup, schema |
| **Invitations** | Batch invite | `POST /api/v1/invitations` | Bulk team member invitations |
| **Invitations** | Accept invite | `POST /api/v1/invitations/accept` | Token-based acceptance |
| **Notebooks** | JupyterHub OAuth | `/api/v1/notebooks/` | Token exchange for notebook access |

### Session & Token Architecture

| Component | Storage | Lifetime |
|-----------|---------|----------|
| Access token | In-memory (frontend) | Short-lived |
| Refresh token | httpOnly cookie | Configurable (`AUTH_JWT_REFRESH_TOKEN_TTL`) |
| Session record | Redis + PostgreSQL | Until logout/expiry |

---

## 5. Cyber Service

**Package:** `internal/cyber/` (310 .go files — largest service)

### Capability Matrix

| Category | Feature | Endpoints | Details |
|----------|---------|-----------|---------|
| **Asset Management** | Asset CRUD | `GET/POST/PUT/DELETE /api/v1/cyber/assets` | Server, endpoint, cloud, IoT, app, DB, container |
| **Asset Management** | Bulk operations | `POST /bulk`, `PUT /bulk/tags`, `DELETE /bulk` | Mass create, tag, delete |
| **Asset Management** | Asset relationships | `GET/POST/DELETE /api/v1/cyber/assets/{id}/relationships` | hosts, runs_on, connects_to, depends_on |
| **Asset Management** | Asset statistics | `GET /api/v1/cyber/assets/stats` | Count by type, criticality, status |
| **Asset Management** | Tag management | `PATCH /api/v1/cyber/assets/{id}/tags` | Add/remove tags |
| **Vulnerability Mgmt** | Vuln CRUD | `GET/PUT /api/v1/cyber/vulnerabilities` | Track by CVE, severity, status |
| **Vulnerability Mgmt** | Per-asset vulns | `GET/POST/PUT /api/v1/cyber/assets/{id}/vulnerabilities` | Asset-scoped vulnerability tracking |
| **Vulnerability Mgmt** | Stats & aging | `GET /stats`, `GET /aging`, `GET /top-cves` | Vulnerability analytics |
| **Scanning** | Network scan | `POST /api/v1/cyber/assets/scan` | Port scanning, service detection |
| **Scanning** | Cloud scan | via scan config | Cloud asset discovery |
| **Scanning** | Agent collection | via scan config | Endpoint agent data collection |
| **Scanning** | Scan history | `GET /api/v1/cyber/assets/scans` | Scan tracking + cancellation |
| **Alert Mgmt** | Alert listing | `GET /api/v1/cyber/alerts` | Filterable, sortable, paginated |
| **Alert Mgmt** | Alert detail | `GET /api/v1/cyber/alerts/{id}` | Full alert with confidence factors |
| **Alert Mgmt** | Status workflow | `PUT /api/v1/cyber/alerts/{id}/status` | new → acknowledged → investigating → resolved |
| **Alert Mgmt** | Assignment | `PUT /api/v1/cyber/alerts/{id}/assign` | Assign to analyst |
| **Alert Mgmt** | Escalation | `POST /api/v1/cyber/alerts/{id}/escalate` | Escalate to higher tier |
| **Alert Mgmt** | Comments | `GET/POST /api/v1/cyber/alerts/{id}/comments` | Analyst notes |
| **Alert Mgmt** | Timeline | `GET /api/v1/cyber/alerts/{id}/timeline` | Status change history |
| **Alert Mgmt** | Related alerts | `GET /api/v1/cyber/alerts/{id}/related` | Correlation |
| **Alert Mgmt** | Alert merge | `POST /api/v1/cyber/alerts/{id}/merge` | Merge duplicate alerts |
| **Detection Rules** | Rule CRUD | `GET/POST/PUT/DELETE /api/v1/cyber/rules` | Sigma, threshold, correlation, anomaly |
| **Detection Rules** | Rule templates | `GET /api/v1/cyber/rules/templates` | Pre-built rule library |
| **Detection Rules** | Toggle enable | `PUT /api/v1/cyber/rules/{id}/toggle` | Enable/disable rules |
| **Detection Rules** | Rule testing | `POST /api/v1/cyber/rules/{id}/test` | Test against historical data |
| **Detection Rules** | Feedback | `POST /api/v1/cyber/rules/{id}/feedback` | Accuracy feedback loop |
| **Threat Intel** | Threat tracking | `GET/PUT /api/v1/cyber/threats` | Malware, APT, ransomware, etc. |
| **Threat Intel** | Indicator management | `GET/POST /api/v1/cyber/threats/{id}/indicators` | IP, domain, hash, URL indicators |
| **Threat Intel** | Indicator check | `POST /api/v1/cyber/threats/indicators/check` | Check indicators against threat DB |
| **Threat Intel** | Bulk import | `POST /api/v1/cyber/threats/indicators/bulk` | Mass indicator import |
| **MITRE ATT&CK** | Tactic listing | `GET /api/v1/cyber/mitre/tactics` | Full MITRE tactic catalog |
| **MITRE ATT&CK** | Technique listing | `GET /api/v1/cyber/mitre/techniques` | Technique catalog |
| **MITRE ATT&CK** | Coverage analysis | `GET /api/v1/cyber/mitre/coverage` | Detection coverage heatmap |
| **Risk Scoring** | Current score | `GET /api/v1/cyber/risk/score` | Multi-component risk calculation |
| **Risk Scoring** | Trend analysis | `GET /api/v1/cyber/risk/score/trend` | Historical risk trend |
| **Risk Scoring** | Recalculation | `GET /api/v1/cyber/risk/score/recalculate` | Force risk recalculation |
| **Risk Scoring** | Heatmap | `GET /api/v1/cyber/risk/heatmap` | Risk by asset type × severity |
| **Risk Scoring** | Top risks | `GET /api/v1/cyber/risk/top-risks` | Highest risk items |
| **Risk Scoring** | Recommendations | `GET /api/v1/cyber/risk/recommendations` | AI-driven risk reduction advice |
| **Remediation** | Action CRUD | `GET/POST/PUT/DELETE /api/v1/cyber/remediation` | Patch, config change, block, isolate |
| **Remediation** | Approval workflow | `POST /submit`, `POST /approve`, `POST /reject` | Multi-step approval |
| **Remediation** | Dry run | `POST /dry-run`, `GET /dry-run` | Test before execution |
| **Remediation** | Execution | `POST /execute` | Execute remediation action |
| **Remediation** | Verification | `POST /verify` | Post-execution verification |
| **Remediation** | Rollback | `POST /rollback` | Undo executed action |
| **Remediation** | Audit trail | `GET /audit-trail` | Full remediation history |
| **CTEM** | Assessment mgmt | `/api/v1/cyber/ctem/` | Continuous Threat Exposure Management |
| **CTEM** | Finding tracking | Assessment-scoped | Vulnerability, misconfig, attack path findings |
| **CTEM** | Remediation groups | Assessment-scoped | Group findings for batch remediation |
| **CTEM** | Exposure scoring | Per-assessment | Multi-factor exposure calculation |
| **DSPM** | Data asset discovery | `GET /api/v1/cyber/dspm/data-assets` | Data Security Posture Management |
| **DSPM** | Classification | `GET /api/v1/cyber/dspm/classification` | Data sensitivity classification |
| **DSPM** | Scan trigger | `POST /api/v1/cyber/dspm/scan` | On-demand DSPM scan |
| **UEBA** | User profiles | `GET /api/v1/cyber/ueba/users/{id}/profile` | Behavioral baseline + risk score |
| **UEBA** | Activity tracking | `GET /api/v1/cyber/ueba/users/{id}/activity` | User activity timeline |
| **UEBA** | Behavioral alerts | `GET /api/v1/cyber/ueba/alerts` | Anomaly-based alerts |
| **UEBA** | Baseline | `GET /api/v1/cyber/ueba/baseline` | Normal behavior patterns |
| **vCISO** | Executive briefing | `GET /api/v1/cyber/vciso/briefing` | AI-powered security briefing |
| **vCISO** | Recommendations | `GET /api/v1/cyber/vciso/recommendations` | Prioritized recommendations |
| **vCISO** | Risk report | `GET /api/v1/cyber/vciso/report` | Executive risk report |
| **vCISO Chat** | Conversations | `POST/GET /api/v1/cyber/vciso/chat/conversations` | AI chat sessions |
| **vCISO Chat** | Messages | `POST/GET .../conversations/{id}/messages` | Send/receive messages |
| **vCISO Chat** | WebSocket chat | `WS /ws/conversations/{id}` | Real-time chat |
| **Dashboard** | Overview | `GET /api/v1/cyber/dashboard` | Aggregated security metrics |
| **Dashboard** | KPIs | `GET /dashboard/kpis` | Key performance indicators |
| **Dashboard** | Alerts timeline | `GET /dashboard/alerts-timeline` | Alert trend over time |
| **Dashboard** | Severity distribution | `GET /dashboard/severity-distribution` | Alert breakdown by severity |
| **Dashboard** | MTTR | `GET /dashboard/mttr` | Mean time to remediate |
| **Dashboard** | Analyst workload | `GET /dashboard/analyst-workload` | Workload distribution |
| **Dashboard** | Top attacked assets | `GET /dashboard/top-attacked-assets` | Most targeted assets |
| **Dashboard** | MITRE heatmap | `GET /dashboard/mitre-heatmap` | Detection coverage visualization |

### Enrichment Pipeline

| Enricher | Input | Output |
|----------|-------|--------|
| DNS Enrichment | IP address | Hostname resolution |
| CVE Enrichment | CVE ID | Vulnerability details, CVSS |
| Geo Enrichment | IP address | Geolocation data |

### Risk Score Components

| Component | Weight | Source |
|-----------|--------|--------|
| Vulnerability score | Configurable | Open vulnerabilities by severity |
| Threat exposure score | Configurable | Active threats × confidence |
| Configuration score | Configurable | Misconfiguration findings |
| Attack surface score | Configurable | Exposed assets + relationships |
| Compliance score | Configurable | Compliance check results |

---

## 6. Data Service

**Package:** `internal/data/` (209 .go files)

### Capability Matrix

| Category | Feature | Endpoints | Details |
|----------|---------|-----------|---------|
| **Data Sources** | Source CRUD | `GET/POST/PUT/DELETE /api/v1/data/sources` | Database, API, file, stream, cloud storage |
| **Data Sources** | Connection test | `POST /api/v1/data/sources/{id}/test` | Verify connectivity |
| **Data Sources** | Schema discovery | `POST /api/v1/data/sources/{id}/discover` | Auto-detect tables/columns |
| **Data Sources** | Table listing | `GET /api/v1/data/sources/{id}/tables` | Browse discovered tables |
| **Data Sources** | Column listing | `GET .../tables/{tid}/columns` | Column metadata |
| **Data Models** | Model CRUD | `GET/POST/PUT/DELETE /api/v1/data/models` | Schema-driven data models |
| **Data Models** | Version tracking | Version field | Model versioning |
| **Pipelines** | Pipeline CRUD | `GET/POST/PUT/DELETE /api/v1/data/pipelines` | ETL, ELT, streaming, batch |
| **Pipelines** | Run trigger | `POST /api/v1/data/pipelines/{id}/run` | On-demand pipeline execution |
| **Pipelines** | Run history | `GET /api/v1/data/pipelines/{id}/runs` | Execution history + logs |
| **Pipelines** | Run logs | `GET .../runs/{rid}/logs` | Detailed execution logs |
| **Pipelines** | Scheduling | CRON-based | Configurable pipeline schedules |
| **Quality** | Rule CRUD | `GET/POST/PUT/DELETE /api/v1/data/quality/rules` | not_null, unique, range, regex, referential, custom |
| **Quality** | Check results | `GET /api/v1/data/quality/results` | Quality check outcomes |
| **Quality** | Scheduled checks | Background task | Periodic quality validation |
| **Lineage** | Graph visualization | `GET /api/v1/data/lineage/graph` | Data flow DAG |
| **Lineage** | Impact analysis | `GET /api/v1/data/lineage/impact/{id}` | Downstream dependency analysis |
| **Lineage** | Source tracking | `GET /api/v1/data/lineage/sources/{id}` | Upstream source tracing |
| **Dark Data** | Discovery | `GET /api/v1/data/dark-data` | Unmodeled, orphaned, stale data detection |
| **Dark Data** | Classification | `POST /api/v1/data/dark-data/{id}/classify` | Classify dark data items |
| **Contradictions** | Detection | `GET /api/v1/data/contradictions` | Logical, semantic, temporal conflicts |
| **Contradictions** | Resolution | `POST /api/v1/data/contradictions/{id}/resolve` | Mark as resolved |
| **Analytics** | Query execution | `POST /api/v1/data/analytics/query/execute` | Ad-hoc query execution |
| **Analytics** | Saved queries | `GET/POST /api/v1/data/analytics/query` | Save and reuse queries |
| **Dashboard** | Overview | `GET /api/v1/data/dashboard` | Aggregated data metrics |
| **Dashboard** | Quality score | `GET /api/v1/data/dashboard/quality-score` | Overall data quality |
| **Dashboard** | Lineage complexity | `GET /api/v1/data/dashboard/lineage-complexity` | Graph complexity metrics |

### Connector Support

| Connector Type | Protocol |
|----------------|----------|
| SQL Server | MSSQL |
| MySQL | MySQL |
| PostgreSQL | PostgreSQL |
| Oracle | Oracle |
| API | REST/GraphQL |
| File | CSV/JSON/Parquet |
| Stream | Kafka/Event stream |
| Cloud Storage | S3/MinIO/GCS/Azure Blob |

---

## 7. Workflow Engine

**Package:** `internal/workflow/` (41 .go files)

### Capability Matrix

| Category | Feature | Endpoints | Details |
|----------|---------|-----------|---------|
| **Definitions** | Workflow CRUD | `GET/POST/PUT/DELETE /api/v1/workflows/definitions` | Define workflow templates |
| **Definitions** | Templates | `GET /api/v1/workflows/templates` | Pre-built workflow templates |
| **Instances** | Start workflow | `POST /api/v1/workflows/instances` | Launch workflow instance |
| **Instances** | Instance tracking | `GET /api/v1/workflows/instances` | List running/completed instances |
| **Instances** | Cancel workflow | `DELETE /api/v1/workflows/instances/{id}` | Cancel running instance |
| **Instances** | Resume workflow | `POST /api/v1/workflows/instances/{id}/resume` | Resume paused instance |
| **Instances** | Execution history | `GET /api/v1/workflows/instances/{id}/history` | Step-by-step execution log |
| **Human Tasks** | Task listing | `GET /api/v1/workflows/tasks` | List tasks assigned to user |
| **Human Tasks** | Task completion | `POST /api/v1/workflows/tasks/{id}/complete` | Submit form + complete task |
| **Human Tasks** | Task update | `PUT /api/v1/workflows/tasks/{id}` | Update task form data |
| **Human Tasks** | Task skip/reject | `DELETE /api/v1/workflows/tasks/{id}` | Skip or reject task |

### Step Executor Registry

| Executor | Type | Behavior |
|----------|------|----------|
| `service_task` | Automated | Invoke external HTTP service |
| `human_task` | Manual | Wait for user form submission |
| `event_task` | Automated | Emit Kafka event |
| `condition` | Logic | Conditional branching |
| `timer` | Scheduled | Delayed/scheduled execution |

### Background Capabilities

| Feature | Details |
|---------|---------|
| Timer polling | Configurable interval for scheduled steps |
| SLA monitoring | Track overdue tasks |
| Instance recovery | Recover orphaned instances on startup |
| Event correlation | Trigger workflows on domain events |

---

## 8. Audit Service

**Package:** `internal/audit/` (26 .go files)

### Capability Matrix

| Category | Feature | Endpoints | Details |
|----------|---------|-----------|---------|
| **Log Query** | List logs | `GET /api/v1/audit/logs` | Filterable, sortable, paginated |
| **Log Query** | Get log entry | `GET /api/v1/audit/logs/{id}` | Single entry with full detail |
| **Log Query** | Statistics | `GET /api/v1/audit/logs/stats` | Aggregation by action, service, user |
| **Log Query** | Timeline | `GET /api/v1/audit/logs/timeline/{resourceId}` | Resource change history |
| **Export** | CSV/JSON export | `GET /api/v1/audit/logs/export` | Bulk export with format selection |
| **Integrity** | Chain verification | `POST /api/v1/audit/verify` | Blockchain-style hash chain verification |
| **Integrity** | Hash chaining | Automatic | `entry_hash` → `prev_hash` immutable trail |
| **Integrity** | Immutability | DB trigger | UPDATE/DELETE prevented at DB level |
| **Storage** | Table partitioning | Monthly | Automatic monthly partitions |
| **Storage** | Partition mgmt | `GET/POST /api/v1/audit/partitions` | List and create partitions |
| **Ingestion** | Kafka consumer | `platform.audit.events` | Ingest events from all services |
| **Privacy** | Log masking | Automatic | PII redaction before storage |
| **Privacy** | Tenant isolation | RLS | Row-level security per tenant |

---

## 9. Notification Service

**Package:** `internal/notification/` (42 .go files)

### Capability Matrix

| Category | Feature | Endpoints | Details |
|----------|---------|-----------|---------|
| **Notifications** | List | `GET /api/v1/notifications` | Paginated, category-filtered |
| **Notifications** | Unread count | `GET /api/v1/notifications/unread-count` | Badge count |
| **Notifications** | Mark all read | `PUT /api/v1/notifications/read-all` | Bulk read |
| **Notifications** | Mark single read | `PUT /api/v1/notifications/{id}/read` | Single read |
| **Notifications** | Delete | `DELETE /api/v1/notifications/{id}` | Remove notification |
| **Preferences** | Get/Update | `GET/PUT /api/v1/notifications/preferences` | Per-channel, per-category config |
| **Webhooks** | CRUD | `GET/POST/PUT/DELETE /api/v1/notifications/webhooks` | Custom webhook endpoints |
| **Admin** | Test notification | `POST /api/v1/notifications/test` | Send test notification |
| **Admin** | Delivery stats | `GET /api/v1/notifications/delivery-stats` | Channel delivery metrics |
| **Admin** | Retry failed | `POST /api/v1/notifications/retry-failed` | Retry failed deliveries |
| **Real-time** | WebSocket | `/ws/v1/notifications/` | Real-time push delivery |

### Delivery Channels

| Channel | Protocol | Features |
|---------|----------|----------|
| **In-App** | REST API | Read/unread tracking, pagination |
| **Email** | SMTP/SendGrid | Template rendering, HTML/text |
| **WebSocket** | Gorilla WebSocket | Max 10 connections/user, JWT auth |
| **Push** | Mobile push | Mobile notification delivery |
| **Webhook** | HTTP POST | HMAC signing, retry with backoff |

### Background Features

| Feature | Details |
|---------|---------|
| Digest aggregation | Daily/weekly configurable |
| Delivery tracking | Per-channel delivery status |
| Retry with backoff | Exponential backoff for failures |
| Idempotency guard | Prevent duplicate delivery |
| Dead letter queue | Track permanently failed messages |
| Kafka consumer | Cross-suite event ingestion |

---

## 10. File Service

**Package:** `internal/filemanager/` (17 .go files)

### Capability Matrix

| Category | Feature | Endpoints | Details |
|----------|---------|-----------|---------|
| **Upload** | Direct upload | `POST /api/v1/files/upload` | Multipart file upload |
| **Upload** | Presigned upload | `POST /api/v1/files/upload/presigned` | Generate presigned URL |
| **Upload** | Confirm upload | `POST /api/v1/files/upload/confirm` | Confirm presigned upload |
| **Download** | Direct download | `GET /api/v1/files/{id}/download` | Stream file download |
| **Download** | Presigned download | `GET /api/v1/files/{id}/presigned` | Generate presigned URL |
| **Management** | List files | `GET /api/v1/files` | Filterable, paginated |
| **Management** | File metadata | `GET /api/v1/files/{id}` | File details |
| **Management** | Delete file | `DELETE /api/v1/files/{id}` | Soft delete |
| **Management** | Version history | `GET /api/v1/files/{id}/versions` | File versioning |
| **Security** | Virus scan | ClamAV | Automatic scan on upload |
| **Security** | Quarantine | `GET /api/v1/files/quarantine` | List quarantined files |
| **Security** | Resolve quarantine | `POST /api/v1/files/quarantine/{id}/resolve` | Admin quarantine resolution |
| **Security** | Rescan | `POST /api/v1/files/{id}/rescan` | On-demand virus rescan |
| **Security** | Encryption | AES-256 | At-rest file encryption |
| **Audit** | Access log | `GET /api/v1/files/{id}/access-log` | File access audit trail |
| **Admin** | File statistics | `GET /api/v1/files/stats` | Storage usage metrics |
| **Lifecycle** | Auto-cleanup | Daily at 03:00 UTC | Expired file cleanup |

### Storage Architecture

| Component | Details |
|-----------|---------|
| Primary storage | MinIO (S3-compatible) |
| Bucket structure | `clario360-{tenant_id}/` per tenant |
| Quarantine bucket | `clario360-quarantine/` |
| Virus scanner | ClamAV (graceful degradation if unavailable) |

---

## 11. Acta Service

**Package:** `internal/acta/` (67 .go files)

### Capability Matrix

| Category | Feature | Details |
|----------|---------|---------|
| **Committees** | CRUD management | Board, audit, risk, compensation, nomination, executive, governance, ad hoc |
| **Committees** | Member management | Chair, vice-chair, secretary, member, observer roles |
| **Committees** | Quorum tracking | Percentage or fixed-count quorum rules |
| **Meetings** | Scheduling | Create meetings with committee, time, location |
| **Meetings** | Location types | Physical, virtual (with link/platform), hybrid |
| **Meetings** | Attendance tracking | Invited, confirmed, declined, present, absent, proxy, excused |
| **Meetings** | Proxy voting | Authorized proxy attendance |
| **Meetings** | Agenda management | Agenda items with presenters, time allocation, ordering |
| **Meetings** | Voting | Unanimous, majority, two-thirds, roll call |
| **Meetings** | Minutes | Draft, review, revision, approval, publish workflow |
| **Action Items** | Creation | From meetings/agenda items with assignees, due dates |
| **Action Items** | Priority | High, medium, low |
| **Action Items** | Status tracking | Pending, in-progress, completed, overdue, cancelled |
| **Compliance** | Tracking | Governance compliance dashboard |
| **AI** | Meeting summaries | AI-powered meeting summary generation |
| **Internal** | Committee members | `GET /api/v1/internal/committee-members` (service-to-service) |
| **Internal** | Meeting attendees | `GET /api/v1/internal/meeting-attendees` (service-to-service) |

### Background Tasks

| Task | Schedule |
|------|----------|
| Overdue checker | Periodic check for overdue action items |
| Meeting reminders | Scheduled reminder notifications |
| Compliance scheduler | Daily compliance check |

---

## 12. Lex Service

**Package:** `internal/lex/`

### Capability Matrix

| Category | Feature | Details |
|----------|---------|---------|
| **Contracts** | Full lifecycle | Draft → review → negotiation → active → expired/terminated |
| **Contracts** | Type support | Service agreement, NDA, employment, vendor, license, lease, partnership, consulting, procurement, SLA, MOU, amendment, renewal |
| **Contracts** | Version control | Multi-version document tracking |
| **Contracts** | Auto-renewal | Automatic renewal with notice period tracking |
| **Contracts** | Risk analysis | Per-contract risk scoring and level assessment |
| **Clauses** | Extraction | Automatic clause identification from documents |
| **Clauses** | Classification | Indemnification, termination, liability, confidentiality, IP, non-compete, payment, warranty, force majeure, dispute, data protection, etc. |
| **Clauses** | Risk scoring | Per-clause risk level with keywords and recommendations |
| **Clauses** | Review workflow | Pending → reviewed → flagged → accepted → rejected |
| **Analysis** | Contract analysis | Overall risk, missing clauses, key findings, recommendations |
| **Analysis** | Party extraction | Automatic party name/entity extraction |
| **Analysis** | Date/amount extraction | Key dates and financial terms |
| **Documents** | Document library | Policy, regulation, template, memo, opinion, filing, correspondence |
| **Documents** | Confidentiality | Public, internal, confidential, privileged levels |
| **Documents** | Version control | Multi-version document tracking |
| **Compliance** | Monitoring | Compliance rule checking |
| **Compliance** | Alerts | Automated compliance violation alerts |

### Background Tasks

| Task | Details |
|------|---------|
| Expiry monitor | Track contract expiration dates |
| Renewal reminder | Send renewal notifications |
| Compliance monitor | Scheduled compliance checking |
| Event consumer | Cross-suite event synchronization |

---

## 13. Visus Service

**Package:** `internal/visus/` (84 .go files)

### Capability Matrix

| Category | Feature | Details |
|----------|---------|---------|
| **Dashboards** | Custom creation | User-created dashboards with grid layout (12-column) |
| **Dashboards** | Visibility | Private, team, organization, public |
| **Dashboards** | Sharing | Share with specific users |
| **Dashboards** | System dashboards | Pre-built default dashboards |
| **Widgets** | Widget library | 13 types: KPI card, line/bar/area/pie chart, gauge, table, alert feed, text, sparkline, heatmap, status grid, trend indicator |
| **Widgets** | Grid positioning | x, y, width, height on 12-column grid |
| **Widgets** | Auto-refresh | Configurable refresh interval |
| **KPIs** | Definition | Name, category, suite, query endpoint, thresholds |
| **KPIs** | Categories | Security, data, governance, legal, operations, general |
| **KPIs** | Cross-suite | Cyber, data, acta, lex, platform, custom |
| **KPIs** | Thresholds | Warning + critical with direction (higher/lower is better) |
| **KPIs** | Calculation types | Direct, delta, percentage change, average, sum over period |
| **KPIs** | Snapshots | every_15m, hourly, every_4h, daily, weekly frequency |
| **Executive Alerts** | Cross-suite alerts | Risk, compliance, data quality, governance, legal, operational |
| **Executive Alerts** | Deduplication | Key-based dedup with occurrence counting |
| **Executive Alerts** | Status workflow | New → viewed → acknowledged → actioned → dismissed/escalated |
| **Reports** | Types | Executive summary, security posture, data intelligence, governance, legal, custom |
| **Reports** | Scheduling | CRON-based auto-generation |
| **Reports** | Distribution | Auto-send to configured recipients |
| **Reports** | Periods | 7d, 14d, 30d, 90d, quarterly, annual, custom |

---

## 14. Event Bus

**Package:** `cmd/event-bus/`

### Capability Matrix

| Category | Feature | Details |
|----------|---------|---------|
| **Event Routing** | Topic subscription | 16 Kafka topics across all services |
| **Event Routing** | Handler registry | Dynamic handler registration per event type |
| **Reliability** | Idempotency | Redis-backed 24hr guard against duplicate processing |
| **Reliability** | Retry | Exponential backoff for failed handlers |
| **Reliability** | Dead letter queue | `platform.dead-letter` topic for unrecoverable failures |
| **DLQ Management** | List DLQ | `GET /api/v1/events/dead-letter` |
| **DLQ Management** | View entry | `GET /api/v1/events/dead-letter/{id}` |
| **DLQ Management** | Replay event | `POST /api/v1/events/dead-letter/{id}/replay` |
| **DLQ Management** | Delete entry | `DELETE /api/v1/events/dead-letter/{id}` |
| **Observability** | Metrics | Per-handler processing metrics |
| **Observability** | Tracing | Distributed tracing across event chains |

### Kafka Topics

| Topic | Producer | Consumers |
|-------|----------|-----------|
| `platform.iam.events` | IAM Service | Event Bus, Notification |
| `platform.audit.events` | All services | Audit Service |
| `platform.notification.events` | All services | Notification Service |
| `platform.workflow.events` | Workflow Engine | Event Bus |
| `platform.asset.events` | Cyber Service | Event Bus, Data Service |
| `platform.threat.events` | Cyber Service | Event Bus, Notification |
| `platform.alert.events` | Cyber Service | Event Bus, Notification |
| `platform.remediation.events` | Cyber Service | Event Bus, Workflow |
| `platform.datasource.events` | Data Service | Event Bus, Cyber |
| `platform.pipeline.events` | Data Service | Event Bus, Lineage |
| `platform.quality.events` | Data Service | Event Bus, Notification |
| `platform.contradiction.events` | Data Service | Event Bus |
| `platform.lineage.events` | Data Service | Event Bus |
| `platform.acta.events` | Acta Service | Event Bus |
| `platform.lex.events` | Lex Service | Event Bus |
| `platform.visus.events` | Visus Service | Event Bus |

---

## 15. AI Governance

**Package:** `internal/aigovernance/`

### Capability Matrix

| Category | Feature | Details |
|----------|---------|---------|
| **Model Registry** | Model CRUD | Register AI/ML models with type, suite, risk tier |
| **Model Registry** | Model types | Rule-based, statistical, ML classifier/regressor, NLP extractor, anomaly detector, scorer, recommender |
| **Model Registry** | Risk tiers | Low, medium, high, critical |
| **Model Registry** | Lifecycle | Active → deprecated → retired |
| **Versioning** | Version tracking | Per-model version numbers with status |
| **Versioning** | Status flow | Development → staging → shadow → production → retired |
| **Versioning** | Promotion | Promote through lifecycle with audit trail |
| **Versioning** | Rollback | Roll back to previous version |
| **Predictions** | Prediction logging | Input/output/confidence/latency logging (partitioned) |
| **Predictions** | Explanation capture | Structured + text explanations, contributing factors |
| **Predictions** | Feedback loop | Correct/incorrect marking with notes |
| **Shadow Mode** | Shadow comparison | Run shadow model alongside production |
| **Shadow Mode** | Agreement analysis | Agreement rate, divergence samples |
| **Shadow Mode** | Recommendations | Promote, keep shadow, reject, needs review |
| **Drift Detection** | Output drift | Population Stability Index (PSI) for output distribution |
| **Drift Detection** | Confidence drift | PSI for confidence distribution |
| **Drift Detection** | Volume changes | Monitor prediction volume changes |
| **Drift Detection** | Latency monitoring | P95 latency tracking |
| **Drift Detection** | Accuracy tracking | Accuracy change over time |
| **Explainability** | Types | Rule trace, feature importance, statistical deviation, template-based |
| **Explainability** | Explanation templates | Configurable per-model-version |

---

## 16. Frontend Application

**Framework:** Next.js 14 (App Router) | **Language:** TypeScript | **UI:** Tailwind CSS + shadcn/ui

### Page & Route Matrix

| Suite | Route | Page | Features |
|-------|-------|------|----------|
| **Auth** | `/login` | Login | Email/password, MFA prompt |
| **Auth** | `/register` | Registration | Form validation, password strength |
| **Auth** | `/forgot-password` | Password Reset | Email-based reset flow |
| **Auth** | `/reset-password` | Reset Confirm | Token-verified password change |
| **Auth** | `/verify-email` | Email Verify | OTP verification |
| **Auth** | `/invite` | Invitation | Accept team invitation |
| **Onboarding** | `/onboarding/setup` | Setup Wizard | 5-step: org, branding, team, suites, provision |
| **Onboarding** | `/onboarding/verify` | Email Verify | Onboarding email verification |
| **Dashboard** | `/dashboard` | Home Dashboard | KPIs, tasks, alerts, activity timeline |
| **Cyber** | `/cyber` | Cyber Home | Security overview |
| **Cyber** | `/cyber/alerts` | Alert List | Filter, sort, assign, escalate |
| **Cyber** | `/cyber/alerts/[id]` | Alert Detail | Confidence, context, remediation, comments, timeline |
| **Cyber** | `/cyber/assets` | Asset List | Stats, bulk operations |
| **Cyber** | `/cyber/assets/[id]` | Asset Detail | Relationships, vulnerabilities |
| **Cyber** | `/cyber/threats` | Threat Intel | Threat listing + indicators |
| **Cyber** | `/cyber/rules` | Detection Rules | Rule CRUD, templates, testing |
| **Cyber** | `/cyber/mitre` | MITRE ATT&CK | Tactic/technique heatmap |
| **Cyber** | `/cyber/risk-heatmap` | Risk Heatmap | Asset type × severity grid |
| **Cyber** | `/cyber/ctem` | CTEM | Assessment list |
| **Cyber** | `/cyber/ctem/[id]` | CTEM Detail | Phases, findings, exposure score |
| **Cyber** | `/cyber/remediation` | Remediation | Action list |
| **Cyber** | `/cyber/remediation/[id]` | Remediation Detail | Plan, dry-run, execution, verification |
| **Cyber** | `/cyber/dspm` | DSPM | Data security posture |
| **Cyber** | `/cyber/ueba` | UEBA | Behavioral analytics |
| **Cyber** | `/cyber/ueba/profiles/[entityId]` | Entity Profile | Baseline, activity, risk, alerts |
| **Cyber** | `/cyber/ueba/alerts` | UEBA Alerts | Behavioral anomaly alerts |
| **Cyber** | `/cyber/vciso` | vCISO | AI chat + briefing |
| **Data** | `/data` | Data Home | Data governance overview |
| **Data** | `/data/sources` | Sources | Data source connections |
| **Data** | `/data/sources/[id]` | Source Detail | Connection config, schema explorer |
| **Data** | `/data/pipelines` | Pipelines | ETL/ELT pipeline list |
| **Data** | `/data/pipelines/[id]` | Pipeline Detail | Transform builder, lineage |
| **Data** | `/data/models` | Models | Data model list |
| **Data** | `/data/models/[id]` | Model Detail | Schema, quality rules |
| **Data** | `/data/quality` | Quality | Data quality dashboard |
| **Data** | `/data/lineage` | Lineage | Data flow visualization |
| **Data** | `/data/analytics` | Analytics | Query + analytics dashboard |
| **Data** | `/data/dark-data` | Dark Data | Unmodeled data discovery |
| **Data** | `/data/contradictions` | Contradictions | Data conflict detection |
| **Acta** | `/acta` | Governance Home | Board management overview |
| **Acta** | `/acta/committees` | Committees | Committee list |
| **Acta** | `/acta/committees/[id]` | Committee Detail | Members, stats |
| **Acta** | `/acta/meetings` | Meetings | Meeting list + calendar |
| **Acta** | `/acta/meetings/[id]` | Meeting Detail | Attendees, agenda, minutes, voting, actions |
| **Acta** | `/acta/action-items` | Action Items | Cross-meeting action tracking |
| **Acta** | `/acta/compliance` | Compliance | Governance compliance dashboard |
| **Lex** | `/lex` | Legal Home | Legal suite overview |
| **Lex** | `/lex/contracts` | Contracts | Contract list |
| **Lex** | `/lex/contracts/[id]` | Contract Detail | Risk, clauses, versions, alerts |
| **Lex** | `/lex/documents` | Documents | Legal document library |
| **Lex** | `/lex/compliance` | Compliance | Legal compliance dashboard |
| **Visus** | `/visus` | Visus Home | Visualization overview |
| **Visus** | `/visus/kpis` | KPIs | KPI definitions + tracking |
| **Visus** | `/visus/reports` | Reports | Report generation |
| **Visus** | `/visus/alerts` | Executive Alerts | Cross-suite executive alerts |
| **Workflows** | `/workflows` | Workflow List | Workflow instance list |
| **Workflows** | `/workflows/[id]` | Workflow Detail | Step timeline |
| **Workflows** | `/workflows/tasks` | Task List | Human task list with status tabs |
| **Workflows** | `/workflows/tasks/[id]` | Task Detail | Form, claim, complete, reject, delegate |
| **Admin** | `/admin/users` | User Mgmt | Full user CRUD |
| **Admin** | `/admin/roles` | Role Mgmt | Role CRUD with permission tree |
| **Admin** | `/admin/audit` | Audit Logs | Log viewer with filtering |
| **Admin** | `/admin/settings` | Tenant Settings | Tenant administration |
| **Admin** | `/admin/ai-governance` | AI Governance | Model registry |
| **Admin** | `/admin/ai-governance/[modelId]` | Model Detail | Versions, performance, drift |
| **Settings** | `/settings` | User Settings | Profile, password, MFA, sessions, API keys |
| **Settings** | `/settings/notifications` | Notification Prefs | Per-channel/category preferences |
| **Other** | `/notifications` | Notification Center | Full notification management |
| **Other** | `/files` | File Browser | File management |
| **Other** | `/notebooks` | Notebooks | Jupyter notebook integration |

**Total Pages: 70+ unique routes**

### BFF API Routes (Next.js Server)

| Route | Method | Purpose |
|-------|--------|---------|
| `/api/auth/session` | GET | Retrieve session (auto-refresh) |
| `/api/auth/session` | POST | Store tokens in httpOnly cookies |
| `/api/auth/session` | DELETE | Logout (clear cookies) |
| `/api/auth/refresh` | POST | Token refresh via httpOnly cookies |
| `/api/health` | GET | Health check |

### Component Library

| Category | Count | Key Components |
|----------|-------|----------------|
| **UI Primitives** (shadcn) | 33 | Button, Input, Dialog, Card, Table, Tabs, Select, etc. |
| **Shared Charts** | 7 | BarChart, PieChart, AreaChart, LineChart, GaugeChart, ChartContainer, ChartTooltip |
| **Shared Forms** | 7 | FormField (RHF+Zod), SearchInput, Combobox, MultiSelect, DateRangePicker, FileUpload, FormSection |
| **Shared Data Table** | 11 | DataTable, Toolbar, Pagination, ColumnHeader, Filter, RowActions, ActiveFilters, Empty/Error/Skeleton states |
| **Shared Display** | 12 | KPICard, SeverityIndicator, StatusBadge, PriorityIndicator, DetailPanel, ConfirmDialog, Timeline, StatCard, UserAvatar, RelativeTime, TruncatedText, CopyButton |
| **Layout** | 11 | Sidebar, Header, Breadcrumbs, CommandPalette, NotificationDropdown, UserMenu, MobileSidebar, ConnectionBanner |
| **Auth** | 12 | LoginForm, RegisterForm, MFASetup/Disable, PasswordStrength, PermissionGate, SessionExpired |
| **Dashboard** | 6 | WelcomeHeader, KPIGrid, KPICard, RecentAlerts, MyTasks, ActivityTimeline |
| **Workflow** | 17 | TaskClaim, TaskComplete, TaskDelegate, TaskReject, TaskDetail, StepTimeline, etc. |
| **Notifications** | 7 | NotificationCard, NotificationList, CategoryTabs, Actions, Empty |
| **Providers** | 4 | AuthProvider, QueryProvider, ToastProvider, WebSocketProvider |
| **Real-time** | 3 | HighlightAnimation, LiveIndicator, NewDataToast |
| **Common** | 6 | PageHeader, EmptyState, ErrorState, LoadingSkeleton, PermissionRedirect, ConnectionStatusBanner |
| **Total** | **~136** | |

### Custom Hooks

| Hook | Purpose |
|------|---------|
| `useApi` | React Query data fetching with auth |
| `useApiMutation` | React Query mutations (POST/PUT/DELETE) |
| `useAuth` | Auth context (user, tenant, permissions) |
| `useWebSocket` | WebSocket connection with exponential backoff |
| `useRealtimeData` | React Query + WebSocket integration |
| `useDataTable` | Table state (sort, filter, pagination) |
| `useInfiniteScroll` | Infinite scroll pagination |
| `useNotificationActions` | Notification action handlers |
| `useTaskForm` | Workflow task form state |
| `useBadgeCounts` | Badge count polling |
| `useBreadcrumbs` | Breadcrumb path generation |
| `useCommandPalette` | Cmd+K palette state |
| `useSidebar` | Sidebar collapse/expand |
| `useMediaQuery` | Responsive breakpoints |
| `useClipboard` | Copy to clipboard |
| `useCountdown` | Countdown timer |
| `useDebounce` | Debounced value |
| `useIntersectionObserver` | Viewport intersection |
| `useKeyboardShortcut` | Keyboard bindings |
| `useLocalStorage` | Persistent local storage |
| `useToast` | Toast notifications |
| `useCsrf` | CSRF token management |

### Zustand Stores

| Store | State Managed |
|-------|--------------|
| `auth-store` | User, tenant, tokens, permissions (wildcard matching) |
| `sidebar-store` | Collapsed state (localStorage persisted) |
| `notification-store` | Notification state, read/unread counts |
| `command-palette-store` | Palette open/close state |
| `realtime-store` | Topic → React Query key registry for WS invalidation |

---

## 17. Database Architecture

### Database Matrix

| # | Database | Tables | Partitioned Tables | Key Feature |
|---|----------|--------|-------------------|-------------|
| 1 | `platform_core` | 19 | audit_logs, ai_prediction_logs | IAM, tenants, users, roles, sessions, API keys, onboarding, AI governance |
| 2 | `cyber_db` | 23 | security_events, ueba_access_events | Assets, vulns, threats, alerts, rules, CTEM, DSPM, UEBA, vCISO |
| 3 | `data_db` | 8 | — | Sources, models, pipelines, quality, lineage, dark data, contradictions |
| 4 | `audit_db` | 2 | audit_logs | Immutable audit trail with hash chaining |
| 5 | `acta_db` | 6 | — | Committees, members, meetings, attendance, agenda, action items |
| 6 | `lex_db` | 6 | — | Contracts, versions, clauses, analyses, documents |
| 7 | `visus_db` | 6 | — | Dashboards, widgets, KPIs, snapshots, alerts, reports |
| 8 | `notification_db` | 5 | — | Notifications, preferences, delivery log, webhooks, templates |

**Total: 75+ tables across 8 databases, 70 SQL migration files**

### Cross-Database Patterns

| Pattern | Implementation |
|---------|---------------|
| **Tenant isolation** | All tables have `tenant_id` + RLS policies |
| **Immutability** | INSERT-only triggers on audit tables |
| **Partitioning** | Monthly partitions on time-series tables |
| **Soft deletes** | `deleted_at` column where applicable |
| **Hash chaining** | `entry_hash` → `prev_hash` for audit integrity |
| **JSONB storage** | Flexible metadata, configs, unstructured data |
| **GIN indexes** | Fast JSONB and text array queries |
| **Full-text search** | `tsvector` columns on searchable tables |

---

## 18. Infrastructure & DevOps

### Deployment Options

| Platform | Tool | Status |
|----------|------|--------|
| **Local** | Docker Compose | ✅ `docker-compose.yml` |
| **Local** | PM2 | ✅ `ecosystem.local.js` |
| **Kubernetes** | Helm Charts | ✅ `deploy/helm/` |
| **Cloud** | Terraform | ✅ `deploy/terraform/` |
| **Container** | Docker | ✅ `deploy/docker/` (multi-stage builds) |
| **Escrow** | Build verification | ✅ `deploy/escrow/` |

### Docker Compose Services

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| PostgreSQL | postgres:16 | 5432 | Primary database |
| Redis | redis:7 | 6379 | Cache, sessions, rate limiting |
| Kafka | kafka | — | Event streaming |
| MinIO | minio | — | S3-compatible object storage |
| Prometheus | prometheus | — | Metrics collection |

### CI/CD (GitHub Actions)

| Workflow | Triggers | Steps |
|----------|----------|-------|
| Backend CI | Push/PR | Lint → Test → Build → Security scan |
| Frontend CI | Push/PR | Lint → Test → Build → Type check |
| Docker Build | Release | Multi-stage build → Push to registry |
| Helm Lint | Push/PR | Lint + template validation |

### Makefile Targets

| Category | Targets |
|----------|---------|
| **Build** | `build`, `build-{service}` |
| **Run** | `run-all`, `run SERVICE=x`, `dev`, `frontend-dev` |
| **Test** | `test`, `test-cover`, `test-short`, `test-integration`, `test-security`, `test-all`, `e2e-test`, `frontend-test` |
| **Lint** | `lint`, `lint-fix`, `fmt`, `frontend-lint` |
| **Database** | `migrate-up`, `migrate-down`, `migrate-create`, `migrate-status`, `seed` |
| **Docker** | `docker-up`, `docker-down`, `docker-clean`, `docker-build`, `docker-test-up/down` |
| **Generate** | `generate-sdk`, `generate-mocks`, `validate-api`, `proto-gen` |
| **Frontend** | `frontend-install`, `frontend-build` |
| **Infra** | `helm-lint`, `helm-template`, `loadtest` |

### Scripts

| Script | Purpose |
|--------|---------|
| `scripts/start.sh` | Start all services |
| `scripts/stop.sh` | Stop all services |
| `scripts/status.sh` | Check service status |
| `scripts/smoke-test.sh` | End-to-end smoke test |

---

## 19. Cross-Cutting Concerns

### Security

| Feature | Implementation |
|---------|---------------|
| Authentication | RS256 JWT (access + refresh tokens) |
| Authorization | Permission-based with wildcard matching |
| Multi-tenancy | Database-level RLS + middleware tenant extraction |
| Rate limiting | Redis-backed per-tenant per-endpoint |
| CORS | Configurable allowed origins |
| Security headers | X-Frame-Options, CSP, HSTS, X-Content-Type-Options |
| CSRF protection | Token-based CSRF for state-changing operations |
| Input sanitization | XSS prevention on frontend |
| File scanning | ClamAV virus scanning on upload |
| Encryption | AES-256 at-rest file encryption |
| Audit trail | Immutable hash-chained audit logs |
| Password hashing | bcrypt with configurable cost |
| Account lockout | Max login attempts (configurable, default 20) |

### Observability

| Component | Tool | Scope |
|-----------|------|-------|
| Structured logging | zerolog | All services |
| Metrics | Prometheus | Per-service registry (14+ metrics/service) |
| Distributed tracing | OpenTelemetry | Cross-service request tracing |
| Health checks | `/healthz`, `/readyz` | All services (HTTP + admin port) |
| Error tracking | Structured error responses | Standardized error codes |

### Error Response Format

```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable message",
  "details": {},
  "request_id": "unique-request-id"
}
```

### Standard Error Codes

| Code | HTTP Status | Usage |
|------|-------------|-------|
| `VALIDATION_ERROR` | 400 | Invalid input |
| `UNAUTHORIZED` | 401 | Missing/invalid auth |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Duplicate/conflict |
| `RATE_LIMITED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Server error |

### Bootstrap Pattern

All services use `bootstrap.Bootstrap(ctx, *ServiceConfig)` → `*Service` providing:

| Component | Details |
|-----------|---------|
| Logger | Structured logging (zerolog) |
| Metrics | Prometheus registry (per-instance) |
| Tracer | OpenTelemetry tracer |
| Redis | Redis client connection |
| Router | chi v5 HTTP router |
| AdminRouter | Separate admin router (metrics, health) |

---

## 20. Statistics Summary

### Backend

| Metric | Count |
|--------|-------|
| Production services | 11 |
| Utility binaries | 3 |
| Go source files | 830+ |
| Internal packages | 23+ |
| Database schemas | 8 |
| Database tables | 75+ |
| SQL migration files | 70 |
| Kafka topics | 16 |
| API endpoints | 250+ |

### Frontend

| Metric | Count |
|--------|-------|
| Page routes | 70+ |
| React components | ~136 |
| Custom hooks | 23 |
| Zustand stores | 5 |
| Utility libraries | 36+ |
| Type definition files | 7 |
| BFF API routes | 5 |

### Infrastructure

| Metric | Count |
|--------|-------|
| Docker Compose services | 5 |
| Makefile targets | 30+ |
| Deployment platforms | 4 (Docker, K8s/Helm, Terraform, PM2) |
| CI/CD workflows | 4+ |
| Utility scripts | 4 |

### Full Platform Capabilities

| # | Capability Domain | Sub-Features |
|---|-------------------|-------------|
| 1 | **Identity & Access Management** | Login, MFA, OAuth/OIDC, user/role/tenant CRUD, API keys, sessions, onboarding, invitations |
| 2 | **API Gateway** | Routing, rate limiting, circuit breaker, WebSocket proxy, CORS, security headers |
| 3 | **Cybersecurity (SOC)** | Asset management, vulnerability tracking, alert management, detection rules, threat intelligence |
| 4 | **MITRE ATT&CK** | Tactic/technique mapping, coverage analysis, heatmap visualization |
| 5 | **Risk Management** | Multi-component scoring, trend analysis, heatmap, recommendations |
| 6 | **CTEM** | Continuous threat exposure assessment, finding prioritization, remediation grouping |
| 7 | **DSPM** | Data asset discovery, classification, sensitivity scoring, posture scanning |
| 8 | **UEBA** | Behavioral profiling, baseline learning, anomaly detection, risk scoring |
| 9 | **vCISO** | AI executive briefing, recommendations, risk reports, real-time chat |
| 10 | **Remediation** | Approval workflow, dry-run, execution, verification, rollback, audit trail |
| 11 | **Data Governance** | Source connections, schema discovery, model management, version tracking |
| 12 | **Data Pipelines** | ETL/ELT/streaming/batch, scheduling, run tracking, logging |
| 13 | **Data Quality** | Rule definitions, scheduled checks, quality scoring, failure analysis |
| 14 | **Data Lineage** | Graph visualization, impact analysis, upstream/downstream tracing |
| 15 | **Dark Data Discovery** | Unmodeled table detection, orphaned file identification, stale data flagging |
| 16 | **Data Contradictions** | Logical/semantic/temporal conflict detection, resolution tracking |
| 17 | **Workflow Engine** | Definition/instance lifecycle, human tasks, service tasks, event/timer steps, SLA monitoring |
| 18 | **Audit & Compliance** | Immutable logging, hash chain verification, export, partitioning, PII masking |
| 19 | **Notifications** | In-app, email, WebSocket, push, webhook channels; preferences, digest, delivery tracking |
| 20 | **File Management** | Upload/download, virus scanning, quarantine, encryption, versioning, access logging |
| 21 | **Board Governance (Acta)** | Committees, meetings, attendance, agenda, voting, minutes, action items, compliance |
| 22 | **Legal Management (Lex)** | Contracts, clauses, risk analysis, documents, compliance monitoring, renewal tracking |
| 23 | **Visualization (Visus)** | Custom dashboards, widget library, KPI tracking, executive alerts, report generation |
| 24 | **AI Governance** | Model registry, version lifecycle, prediction logging, shadow mode, drift detection, explainability |
| 25 | **Event Bus** | 16-topic routing, idempotency, retry, dead letter queue, cross-suite event coordination |
| 26 | **Real-time Updates** | WebSocket provider, topic-based query invalidation, connection status monitoring |
| 27 | **Frontend Shell** | Sidebar nav, command palette, breadcrumbs, mobile responsive, permission gating |
| 28 | **Tenant Onboarding** | Self-service registration, email verification, setup wizard, auto-provisioning |

---

> **Coverage Verification:** This matrix covers all 11 production services, 3 utility binaries, 8 databases (75+ tables), 70+ frontend routes, ~136 components, 23 hooks, 5 stores, and all infrastructure/deployment configurations present in the Clario360 codebase.
