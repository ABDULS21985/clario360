# Clario 360 — AI Agent Build Prompts

## Master Reference

| Field | Detail |
|-------|--------|
| Platform | Clario 360 Enterprise AI Platform |
| Backend | Go (Golang) 1.22+ |
| Database | PostgreSQL 16+ |
| Frontend | Next.js 14+ (App Router, TypeScript, Tailwind CSS, shadcn/ui) |
| Architecture | Kubernetes-native, API-first, Event-driven, Multi-tenant |
| Total Prompts | 50 (10 Batches of 5) |

---

# BATCH 1 — Foundation, Project Scaffolding & Core Infrastructure (Prompts 1–5)

---

## PROMPT 1: Monorepo Scaffolding, Go Project Structure & Development Environment

```
You are a principal engineer building the Clario 360 Enterprise AI Platform — a Saudi-owned, Kubernetes-native, multi-suite enterprise platform. The tech stack is: Go 1.22+ (backend), PostgreSQL 16 (database), Next.js 14 App Router with TypeScript and Tailwind CSS (frontend).

Create the complete monorepo project scaffolding with the following exact structure. Every file must be production-grade — no stubs, no TODOs, no placeholders.

### 1. Root Structure

clario360/
├── docker-compose.yml          # Local dev: PostgreSQL 16, Redis 7, Kafka (KRaft), MinIO, Keycloak
├── docker-compose.test.yml     # Test environment with test databases
├── Makefile                    # Targets: build, test, lint, migrate-up, migrate-down, proto-gen, docker-up, docker-down, seed
├── .env.example                # All environment variables documented
├── .gitignore
├── README.md
├── go.work                     # Go workspace file linking all Go modules
├── go.work.sum
│
├── backend/
│   ├── go.mod                  # Module: github.com/clario360/platform
│   ├── go.sum
│   ├── cmd/
│   │   ├── api-gateway/main.go       # API Gateway service entry point
│   │   ├── iam-service/main.go       # IAM/Auth service entry point
│   │   ├── event-bus/main.go         # Event bus consumer/producer service
│   │   ├── workflow-engine/main.go   # Workflow orchestration service
│   │   ├── audit-service/main.go     # Centralized audit logging service
│   │   ├── cyber-service/main.go     # Cybersecurity Suite service
│   │   ├── data-service/main.go      # Data Suite service
│   │   ├── acta-service/main.go      # Clario Acta service
│   │   ├── lex-service/main.go       # Clario Lex service
│   │   ├── visus-service/main.go     # Clario Visus360 service
│   │   └── migrator/main.go          # Database migration runner
│   │
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go             # Viper-based config loader (env + yaml + defaults)
│   │   ├── server/
│   │   │   └── http.go               # Shared HTTP server setup (Chi router, middleware stack)
│   │   ├── middleware/
│   │   │   ├── auth.go               # JWT validation + RBAC enforcement middleware
│   │   │   ├── tenant.go             # Multi-tenant context extraction middleware
│   │   │   ├── logging.go            # Structured request logging middleware
│   │   │   ├── ratelimit.go          # Per-tenant rate limiting middleware
│   │   │   ├── cors.go               # CORS configuration middleware
│   │   │   ├── requestid.go          # X-Request-ID generation middleware
│   │   │   └── recovery.go           # Panic recovery middleware
│   │   ├── database/
│   │   │   ├── postgres.go           # PostgreSQL connection pool (pgxpool) with health checks
│   │   │   ├── migrations.go         # golang-migrate integration
│   │   │   └── tx.go                 # Transaction helper (RunInTx pattern)
│   │   ├── events/
│   │   │   ├── kafka.go              # Kafka producer/consumer with Sarama
│   │   │   ├── events.go             # Event types, serialization (JSON + Avro)
│   │   │   └── handler.go            # Event handler registry pattern
│   │   ├── auth/
│   │   │   ├── jwt.go                # JWT creation, validation, refresh
│   │   │   ├── rbac.go               # Role/permission checking
│   │   │   └── context.go            # Auth context helpers (UserFromContext, TenantFromContext)
│   │   ├── observability/
│   │   │   ├── logger.go             # Structured logging (zerolog)
│   │   │   ├── metrics.go            # Prometheus metrics registry
│   │   │   └── tracing.go            # OpenTelemetry tracing setup
│   │   ├── errors/
│   │   │   └── errors.go             # Domain error types (NotFound, Unauthorized, Validation, Conflict, Internal)
│   │   └── types/
│   │       ├── pagination.go         # Pagination request/response types
│   │       ├── audit.go              # AuditEntry type used across all services
│   │       └── common.go             # Common types (ID, Timestamp, Metadata)
│   │
│   ├── pkg/                          # Publicly importable packages
│   │   ├── validator/
│   │   │   └── validator.go          # Request validation (go-playground/validator)
│   │   ├── crypto/
│   │   │   └── crypto.go             # AES-256-GCM encryption, PBKDF2, hashing utilities
│   │   └── httpclient/
│   │       └── client.go             # Instrumented HTTP client with retries, circuit breaker
│   │
│   └── migrations/
│       └── 000001_init_schema.up.sql   # Initial migration (created in Prompt 2)
│       └── 000001_init_schema.down.sql
│
├── frontend/
│   ├── package.json
│   ├── tsconfig.json
│   ├── tailwind.config.ts
│   ├── next.config.ts
│   ├── .env.local.example
│   ├── src/
│   │   ├── app/
│   │   │   ├── layout.tsx            # Root layout with providers
│   │   │   ├── page.tsx              # Landing/login redirect
│   │   │   ├── (auth)/
│   │   │   │   ├── login/page.tsx
│   │   │   │   └── layout.tsx
│   │   │   ├── (dashboard)/
│   │   │   │   ├── layout.tsx        # Dashboard layout with sidebar + header
│   │   │   │   ├── page.tsx          # Dashboard home
│   │   │   │   ├── cyber/            # Cybersecurity suite pages (placeholder routes)
│   │   │   │   ├── data/             # Data suite pages
│   │   │   │   ├── acta/             # Acta pages
│   │   │   │   ├── lex/              # Lex pages
│   │   │   │   └── visus/            # Visus360 pages
│   │   │   └── api/                  # Next.js API routes (BFF proxy)
│   │   ├── components/
│   │   │   ├── ui/                   # shadcn/ui components (button, input, dialog, table, etc.)
│   │   │   ├── layout/
│   │   │   │   ├── sidebar.tsx       # Collapsible sidebar with suite navigation
│   │   │   │   ├── header.tsx        # Top bar with user menu, notifications, tenant switcher
│   │   │   │   └── breadcrumbs.tsx
│   │   │   └── shared/               # Shared components (data-table, charts, status-badge, etc.)
│   │   ├── lib/
│   │   │   ├── api.ts                # Axios instance with interceptors (auth, refresh, error handling)
│   │   │   ├── auth.ts               # Auth helpers (getSession, isAuthenticated, hasPermission)
│   │   │   └── utils.ts              # Utility functions
│   │   ├── hooks/
│   │   │   ├── use-auth.ts           # Auth context hook
│   │   │   └── use-api.ts            # SWR/React Query wrapper hook
│   │   ├── stores/
│   │   │   └── auth-store.ts         # Zustand auth store
│   │   └── types/
│   │       ├── api.ts                # API response types
│   │       └── models.ts             # Domain model types (User, Tenant, Role, etc.)
│   └── public/
│       └── clario360-logo.svg
│
├── deploy/
│   ├── helm/
│   │   └── clario360/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       └── templates/             # K8s manifests (deployments, services, ingress, configmaps)
│   ├── terraform/
│   │   ├── modules/
│   │   └── environments/
│   │       ├── dev/
│   │       └── prod/
│   └── docker/
│       ├── Dockerfile.backend         # Multi-stage Go build
│       └── Dockerfile.frontend        # Multi-stage Next.js build
│
└── docs/
    ├── architecture/
    ├── api/
    └── runbooks/

### 2. Implementation Requirements

**go.mod dependencies (use latest stable versions):**
- github.com/go-chi/chi/v5 (HTTP router)
- github.com/jackc/pgx/v5 (PostgreSQL driver)
- github.com/golang-migrate/migrate/v4 (migrations)
- github.com/rs/zerolog (structured logging)
- github.com/IBM/sarama (Kafka client)
- github.com/go-playground/validator/v10
- github.com/golang-jwt/jwt/v5
- github.com/spf13/viper (configuration)
- github.com/prometheus/client_golang (metrics)
- go.opentelemetry.io/otel (tracing)
- github.com/redis/go-redis/v9
- golang.org/x/crypto

**config.go must support:**
- Environment variables (12-factor)
- YAML config file fallback
- Sensible defaults for local development
- Sections: Server, Database, Redis, Kafka, Auth, Observability

**http.go (shared server) must include:**
- Chi router with middleware stack in this order: RequestID → Recovery → CORS → Logging → Auth → RateLimit → Tenant
- Health check endpoints: GET /healthz (liveness), GET /readyz (readiness with DB ping)
- Graceful shutdown with context cancellation

**Every middleware must be fully implemented** — not just function signatures. Auth middleware must validate JWTs, extract claims, and set user/tenant context. Rate limiter must use Redis sliding window.

**docker-compose.yml must include:**
- PostgreSQL 16 with init scripts for creating per-service databases
- Redis 7.2
- Kafka (bitnami/kafka with KRaft mode, no ZooKeeper)
- MinIO (S3-compatible object storage)
- Schema Registry (for Avro event schemas)

**Makefile targets must all work.** `make docker-up` starts all dependencies. `make build` compiles all services. `make test` runs all tests with race detector. `make migrate-up` runs all migrations.

**Dockerfile.backend must:**
- Use multi-stage build (golang:1.22-alpine builder → gcr.io/distroless/static-debian12)
- Accept BUILD_SERVICE arg to compile specific service
- Output < 20MB image
- Run as non-root user

Generate ALL files with complete, production-ready code. No shortcuts.
```

---

## PROMPT 2: PostgreSQL Database Schema — Complete DDL for Platform Core & All Suites

```
You are continuing to build the Clario 360 Enterprise AI Platform. The monorepo structure from Prompt 1 is in place. Now design and implement the COMPLETE PostgreSQL database schema.

### Context
- Each service gets its own database (database-per-service pattern)
- Shared types (users, tenants, roles) live in the `platform_core` database
- Cross-service references use UUIDs — no foreign keys across databases
- All tables use UUID primary keys (gen_random_uuid())
- All tables include: id, created_at, updated_at, created_by, updated_by
- Soft deletes via deleted_at column where appropriate
- Multi-tenant: every business table includes tenant_id (UUID, NOT NULL, indexed)
- Audit columns track the user who made changes

### Database Structure

Generate golang-migrate compatible migration files (SQL) for each database:

#### Database 1: platform_core
```sql
-- Tables needed:
-- tenants: id, name, slug, domain, settings(jsonb), status, subscription_tier, created_at, updated_at
-- users: id, tenant_id, email, password_hash, first_name, last_name, avatar_url, status(active/inactive/suspended), mfa_enabled, mfa_secret, last_login_at, created_at, updated_at, deleted_at
-- roles: id, tenant_id, name, slug, description, is_system_role, permissions(jsonb), created_at, updated_at
-- user_roles: user_id, role_id, tenant_id, assigned_at, assigned_by
-- sessions: id, user_id, tenant_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at
-- api_keys: id, tenant_id, name, key_hash, key_prefix, permissions(jsonb), last_used_at, expires_at, created_at, revoked_at
-- audit_logs: id, tenant_id, user_id, service, action, resource_type, resource_id, old_value(jsonb), new_value(jsonb), ip_address, user_agent, metadata(jsonb), created_at (PARTITIONED BY RANGE on created_at — monthly partitions)
-- notifications: id, tenant_id, user_id, type, title, body, data(jsonb), read_at, created_at
-- system_settings: id, tenant_id, key, value(jsonb), description, updated_by, updated_at
```

#### Database 2: cyber_db
```sql
-- Tables needed:
-- assets: id, tenant_id, name, type(enum: server, endpoint, network_device, cloud_resource, iot_device, application, database, container), ip_address, hostname, mac_address, os, os_version, owner, department, criticality(enum: critical, high, medium, low), status(enum: active, inactive, decommissioned), discovered_at, last_seen_at, metadata(jsonb), tags(text[]), created_at, updated_at, deleted_at
-- asset_relationships: id, tenant_id, source_asset_id, target_asset_id, relationship_type, metadata(jsonb), created_at
-- vulnerabilities: id, tenant_id, asset_id, cve_id, title, description, severity(enum: critical, high, medium, low, info), cvss_score(decimal), cvss_vector, status(enum: open, in_progress, mitigated, resolved, accepted, false_positive), discovered_at, resolved_at, due_date, assigned_to, metadata(jsonb), created_at, updated_at
-- threats: id, tenant_id, type, title, description, severity, confidence_score(decimal), source, indicators(jsonb), mitre_technique_id, mitre_tactic, status(enum: detected, investigating, confirmed, mitigated, false_positive), detected_at, resolved_at, metadata(jsonb), created_at, updated_at
-- threat_indicators: id, tenant_id, threat_id, type(enum: ip, domain, hash_md5, hash_sha1, hash_sha256, url, email, filename, registry_key), value, confidence(decimal), source, first_seen, last_seen, created_at
-- detection_rules: id, tenant_id, name, description, rule_type(enum: sigma, yara, custom, ml_model), rule_content(text), severity, mitre_techniques(text[]), enabled, last_triggered_at, trigger_count, created_by, created_at, updated_at
-- alerts: id, tenant_id, rule_id, title, description, severity, status(enum: new, acknowledged, investigating, resolved, false_positive, escalated), confidence_score(decimal), explanation(jsonb — SHAP/LIME output), contributing_factors(jsonb), affected_assets(uuid[]), assigned_to, acknowledged_at, resolved_at, resolution_notes, created_at, updated_at
-- remediation_actions: id, tenant_id, alert_id, vulnerability_id, type(enum: patch, config_change, block, isolate, custom), status(enum: pending_approval, approved, dry_run, executing, completed, failed, rolled_back), execution_mode(enum: manual, semi_auto, auto), dry_run_result(jsonb), execution_result(jsonb), rollback_data(jsonb), approved_by, executed_by, executed_at, completed_at, created_at, updated_at
-- ctem_assessments: id, tenant_id, name, scope(jsonb), status(enum: scheduled, running, completed, failed), started_at, completed_at, findings_count, critical_count, high_count, medium_count, low_count, report(jsonb), created_by, created_at, updated_at
-- dspm_data_assets: id, tenant_id, name, type, location, classification(enum: public, internal, confidential, restricted), sensitivity_score(decimal), owner, data_types(text[]), risk_score(decimal), last_scanned_at, metadata(jsonb), created_at, updated_at
```

#### Database 3: data_db
```sql
-- Tables needed:
-- data_sources: id, tenant_id, name, type(enum: database, api, file, stream, cloud_storage), connection_config(jsonb — encrypted), status, schema_metadata(jsonb), last_synced_at, sync_frequency, created_by, created_at, updated_at
-- data_models: id, tenant_id, name, description, version, schema_definition(jsonb), source_id, status(enum: draft, active, deprecated), lineage(jsonb), created_by, created_at, updated_at
-- data_quality_rules: id, tenant_id, model_id, column_name, rule_type(enum: not_null, unique, range, regex, referential, custom), rule_config(jsonb), severity, enabled, last_check_at, last_check_result, created_at, updated_at
-- data_quality_results: id, tenant_id, rule_id, model_id, status(enum: passed, failed, warning), records_checked, records_failed, failure_samples(jsonb), checked_at, created_at
-- contradictions: id, tenant_id, type(enum: logical, semantic, analytical, temporal), source_a(jsonb), source_b(jsonb), description, severity, confidence_score(decimal), resolution_guidance(text), status(enum: detected, investigating, resolved, accepted), resolved_by, resolved_at, created_at, updated_at
-- pipelines: id, tenant_id, name, description, type(enum: etl, elt, streaming, batch), source_id, target_id, schedule(text — cron), config(jsonb), status(enum: active, paused, failed, completed), last_run_at, next_run_at, created_by, created_at, updated_at
-- pipeline_runs: id, tenant_id, pipeline_id, status(enum: running, completed, failed, cancelled), started_at, completed_at, records_processed, records_failed, error_log(text), metrics(jsonb), created_at
-- data_lineage: id, tenant_id, source_type, source_id, target_type, target_id, transformation(text), pipeline_id, created_at
-- dark_data_assets: id, tenant_id, location, type, size_bytes, last_accessed_at, classification, risk_score(decimal), owner, governance_status(enum: unmanaged, under_review, governed, archived), discovered_at, created_at, updated_at
-- data_catalogs: id, tenant_id, name, description, schema_info(jsonb), owner, tags(text[]), classification, access_count, last_accessed_at, created_at, updated_at
```

#### Database 4: acta_db
```sql
-- Tables needed:
-- committees: id, tenant_id, name, type, description, chair_user_id, secretary_user_id, members(jsonb), meeting_frequency, status, created_at, updated_at
-- meetings: id, tenant_id, committee_id, title, description, scheduled_at, location, virtual_link, status(enum: scheduled, in_progress, completed, cancelled), duration_minutes, created_by, created_at, updated_at
-- agenda_items: id, tenant_id, meeting_id, title, description, presenter_user_id, duration_minutes, order_index, status(enum: pending, discussed, deferred, approved, rejected), attachments(jsonb), voting_result(jsonb), created_at, updated_at
-- meeting_minutes: id, tenant_id, meeting_id, content(text), ai_summary(text), ai_action_items(jsonb), status(enum: draft, review, approved, published), approved_by, approved_at, created_at, updated_at
-- action_items: id, tenant_id, meeting_id, agenda_item_id, title, description, assigned_to, due_date, status(enum: pending, in_progress, completed, overdue, cancelled), completed_at, created_at, updated_at
-- governance_workflows: id, tenant_id, name, type, definition(jsonb — BPMN-like), status, created_by, created_at, updated_at
-- workflow_instances: id, tenant_id, workflow_id, current_step, data(jsonb), status(enum: active, completed, cancelled, suspended), started_at, completed_at, created_at, updated_at
-- compliance_checks: id, tenant_id, entity_type, entity_id, rule_name, status(enum: compliant, non_compliant, warning, not_applicable), details(jsonb), checked_at, created_at
```

#### Database 5: lex_db
```sql
-- Tables needed:
-- contracts: id, tenant_id, title, type(enum: nda, service_agreement, employment, vendor, license, other), status(enum: draft, review, negotiation, active, expired, terminated), parties(jsonb), effective_date, expiry_date, value(decimal), currency, file_url, metadata(jsonb), created_by, created_at, updated_at
-- contract_clauses: id, tenant_id, contract_id, clause_number, title, content(text), risk_level(enum: high, medium, low, none), ai_analysis(jsonb), ai_risk_flags(jsonb), status(enum: draft, reviewed, approved, flagged), reviewed_by, created_at, updated_at
-- legal_documents: id, tenant_id, title, type, content(text), file_url, status(enum: draft, review, approved, archived), version, parent_id, tags(text[]), created_by, created_at, updated_at
-- compliance_rules: id, tenant_id, name, description, jurisdiction, regulation_reference, rule_logic(jsonb), severity, enabled, created_at, updated_at
-- compliance_alerts: id, tenant_id, rule_id, entity_type, entity_id, title, description, severity, status(enum: new, acknowledged, resolved, dismissed), resolved_by, resolved_at, created_at, updated_at
-- legal_workflows: id, tenant_id, name, type(enum: contract_review, document_approval, compliance_check, dispute_resolution), definition(jsonb), status, created_by, created_at, updated_at
-- legal_workflow_instances: id, tenant_id, workflow_id, entity_type, entity_id, current_step, data(jsonb), status, started_at, completed_at, created_at, updated_at
```

#### Database 6: visus_db
```sql
-- Tables needed:
-- dashboards: id, tenant_id, name, description, layout(jsonb), is_default, owner_user_id, shared_with(jsonb), created_at, updated_at
-- dashboard_widgets: id, tenant_id, dashboard_id, type(enum: kpi_card, line_chart, bar_chart, pie_chart, table, heatmap, gauge, alert_feed, text), title, config(jsonb — data source, query, visualization settings), position(jsonb — x, y, w, h), refresh_interval_seconds, created_at, updated_at
-- kpi_definitions: id, tenant_id, name, description, suite(enum: cyber, data, acta, lex, platform), query_config(jsonb), unit, target_value(decimal), warning_threshold(decimal), critical_threshold(decimal), calculation_type(enum: count, sum, average, percentage, custom), created_at, updated_at
-- kpi_snapshots: id, tenant_id, kpi_id, value(decimal), period_start, period_end, metadata(jsonb), created_at (PARTITIONED BY RANGE on created_at)
-- executive_alerts: id, tenant_id, source_suite, title, description, severity, category(enum: risk, opportunity, anomaly, threshold_breach, compliance), data(jsonb), status(enum: new, viewed, actioned, dismissed), created_at, updated_at
-- reports: id, tenant_id, name, type(enum: scheduled, on_demand, automated), config(jsonb), schedule(text — cron), last_generated_at, file_url, created_by, created_at, updated_at
-- report_snapshots: id, tenant_id, report_id, generated_at, file_url, metadata(jsonb), created_at
```

### Implementation Requirements

1. Generate separate migration files per database: `backend/migrations/{database_name}/000001_init_schema.up.sql` and `.down.sql`
2. Include all indexes: composite indexes on (tenant_id, status), (tenant_id, created_at), and any frequently filtered columns
3. Include GIN indexes on JSONB columns that will be queried
4. Include partial indexes where appropriate (e.g., active records only)
5. Add CHECK constraints for enums using PostgreSQL native enums (CREATE TYPE)
6. Partition audit_logs and kpi_snapshots by month using RANGE partitioning
7. Add trigger functions for updated_at auto-update
8. Add comments on all tables and columns explaining their purpose
9. Include the init script for docker-compose that creates all 6 databases

Generate ALL migration files — complete DDL, not summaries.
```

---

## PROMPT 3: IAM Service — Authentication, Authorization, Multi-Tenancy & Session Management

```
You are continuing to build the Clario 360 Enterprise AI Platform. The project structure (Prompt 1) and database schema (Prompt 2) are complete. Now build the IAM Service — the identity and access management backbone of the entire platform.

### Service: backend/cmd/iam-service/main.go

This service handles ALL authentication and authorization for the platform. It must be production-grade, secure, and handle the following:

### 1. API Endpoints (Chi router, JSON request/response)

**Auth Endpoints (public — no auth middleware):**
```
POST   /api/v1/auth/register          # Register new user (first user in tenant becomes admin)
POST   /api/v1/auth/login              # Login with email/password → returns access_token + refresh_token
POST   /api/v1/auth/refresh            # Refresh access token using refresh_token
POST   /api/v1/auth/logout             # Invalidate refresh token
POST   /api/v1/auth/forgot-password    # Send password reset email (log to console in dev)
POST   /api/v1/auth/reset-password     # Reset password with token
POST   /api/v1/auth/verify-mfa         # Verify MFA TOTP code after login
```

**User Management (authenticated, admin-only where noted):**
```
GET    /api/v1/users                   # List users in tenant (paginated, filterable) [admin]
GET    /api/v1/users/:id               # Get user by ID [admin or self]
PUT    /api/v1/users/:id               # Update user profile [admin or self]
DELETE /api/v1/users/:id               # Soft-delete user [admin]
PUT    /api/v1/users/:id/status        # Activate/suspend user [admin]
GET    /api/v1/users/me                # Get current user profile
PUT    /api/v1/users/me/password       # Change own password
POST   /api/v1/users/me/mfa/enable     # Enable MFA (returns QR code URI)
POST   /api/v1/users/me/mfa/disable    # Disable MFA (requires current TOTP)
```

**Role Management (authenticated, admin-only):**
```
GET    /api/v1/roles                   # List roles in tenant
POST   /api/v1/roles                   # Create custom role
GET    /api/v1/roles/:id               # Get role with permissions
PUT    /api/v1/roles/:id               # Update role
DELETE /api/v1/roles/:id               # Delete custom role (not system roles)
POST   /api/v1/users/:id/roles         # Assign role to user
DELETE /api/v1/users/:id/roles/:roleId # Remove role from user
```

**Tenant Management (authenticated, super-admin only):**
```
GET    /api/v1/tenants                 # List all tenants [super-admin]
POST   /api/v1/tenants                 # Create tenant [super-admin]
GET    /api/v1/tenants/:id             # Get tenant details [super-admin or tenant admin]
PUT    /api/v1/tenants/:id             # Update tenant settings [super-admin or tenant admin]
```

**API Key Management (authenticated):**
```
GET    /api/v1/api-keys                # List API keys for tenant
POST   /api/v1/api-keys               # Create API key (return key only once)
DELETE /api/v1/api-keys/:id            # Revoke API key
```

### 2. Internal Architecture

```
backend/internal/iam/
├── handler/
│   ├── auth_handler.go         # Auth endpoint handlers
│   ├── user_handler.go         # User CRUD handlers
│   ├── role_handler.go         # Role management handlers
│   ├── tenant_handler.go       # Tenant management handlers
│   └── apikey_handler.go       # API key handlers
├── service/
│   ├── auth_service.go         # Business logic: login, register, token refresh, MFA
│   ├── user_service.go         # User CRUD business logic
│   ├── role_service.go         # Role/permission business logic
│   ├── tenant_service.go       # Tenant management logic
│   └── apikey_service.go       # API key generation and validation
├── repository/
│   ├── user_repo.go            # User database operations
│   ├── role_repo.go            # Role database operations
│   ├── session_repo.go         # Session/refresh token operations
│   ├── tenant_repo.go          # Tenant database operations
│   └── apikey_repo.go          # API key database operations
├── model/
│   ├── user.go                 # User domain model
│   ├── role.go                 # Role domain model with Permission constants
│   ├── tenant.go               # Tenant domain model
│   ├── session.go              # Session model
│   └── apikey.go               # API key model
└── dto/
    ├── auth_dto.go             # Request/response DTOs for auth endpoints
    ├── user_dto.go             # User DTOs
    ├── role_dto.go             # Role DTOs
    └── tenant_dto.go           # Tenant DTOs
```

### 3. Security Requirements

- Passwords: bcrypt with cost 12; minimum 12 characters, must include upper, lower, digit, special
- JWT: RS256 signing; access token TTL = 15 minutes; refresh token TTL = 7 days
- Refresh tokens: stored as bcrypt hash in sessions table; rotated on each refresh (old token invalidated)
- MFA: TOTP (RFC 6238) with 30-second window; recovery codes generated on enable (10 codes, hashed, single-use)
- Rate limiting: 5 failed login attempts → 15-minute lockout per IP+email combination (tracked in Redis)
- API keys: 32-byte random, prefixed with "clario_" + 6-char prefix for identification; stored as SHA-256 hash
- All sensitive operations emit audit events to Kafka topic "audit.iam"
- Password reset tokens: 32-byte random, SHA-256 hashed in DB, 1-hour expiry

### 4. Default System Roles (seeded on tenant creation)

```go
var SystemRoles = []Role{
    {Name: "Super Admin", Slug: "super-admin", Permissions: ["*"]},
    {Name: "Tenant Admin", Slug: "tenant-admin", Permissions: ["tenant:*", "users:*", "roles:*"]},
    {Name: "Security Analyst", Slug: "security-analyst", Permissions: ["cyber:read", "cyber:write", "alerts:*"]},
    {Name: "Security Manager", Slug: "security-manager", Permissions: ["cyber:*", "remediation:approve"]},
    {Name: "Data Engineer", Slug: "data-engineer", Permissions: ["data:*", "pipelines:*"]},
    {Name: "Data Steward", Slug: "data-steward", Permissions: ["data:read", "quality:*", "lineage:*"]},
    {Name: "Legal Analyst", Slug: "legal-analyst", Permissions: ["lex:read", "lex:write"]},
    {Name: "Board Secretary", Slug: "board-secretary", Permissions: ["acta:*"]},
    {Name: "Executive", Slug: "executive", Permissions: ["visus:*", "reports:read"]},
    {Name: "Viewer", Slug: "viewer", Permissions: ["*:read"]},
}
```

### 5. Events Published to Kafka

```
Topic: platform.iam.events
Events: user.registered, user.login.success, user.login.failed, user.logout, user.updated, user.deleted, user.mfa.enabled, user.mfa.disabled, role.created, role.assigned, role.removed, apikey.created, apikey.revoked, tenant.created, tenant.updated
```

### 6. Tests

Write comprehensive tests:
- Unit tests for service layer (mock repository with interfaces)
- Integration tests for repository layer (using testcontainers-go with real PostgreSQL)
- Handler tests (httptest with test JWT tokens)
- Test coverage target: ≥ 80%

Generate ALL files with complete, working code. Every handler, service method, repository query, DTO, and model must be fully implemented.
```

---

## PROMPT 4: API Gateway, Rate Limiting, Request Routing & Service Discovery

```
You are continuing to build the Clario 360 Enterprise AI Platform. The IAM service (Prompt 3) is complete. Now build the API Gateway — the single entry point for all client requests.

### Service: backend/cmd/api-gateway/main.go

The API Gateway is a reverse proxy that routes requests to the correct backend service, enforces authentication, applies rate limiting, and provides observability.

### 1. Architecture

The gateway does NOT contain business logic. It:
1. Receives all incoming HTTP requests
2. Validates JWT tokens (calling IAM service for verification if needed, or validating locally with public key)
3. Extracts tenant context from token claims
4. Applies per-tenant rate limiting
5. Routes requests to the correct backend service based on URL prefix
6. Adds tracing headers (X-Request-ID, X-Tenant-ID, X-User-ID)
7. Logs all requests with structured logging
8. Collects Prometheus metrics (request count, latency histogram, error rate by service)

### 2. Route Configuration

```go
// Route configuration — maps URL prefixes to backend service addresses
var routes = []RouteConfig{
    {Prefix: "/api/v1/auth", Service: "iam-service", StripPrefix: false, Public: true},
    {Prefix: "/api/v1/users", Service: "iam-service", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/roles", Service: "iam-service", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/tenants", Service: "iam-service", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/api-keys", Service: "iam-service", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/audit", Service: "audit-service", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/workflows", Service: "workflow-engine", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/cyber", Service: "cyber-service", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/data", Service: "data-service", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/acta", Service: "acta-service", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/lex", Service: "lex-service", StripPrefix: false, Public: false},
    {Prefix: "/api/v1/visus", Service: "visus-service", StripPrefix: false, Public: false},
}
```

### 3. Implementation Details

```
backend/internal/gateway/
├── proxy/
│   ├── reverse_proxy.go        # httputil.ReverseProxy wrapper with circuit breaker
│   ├── router.go               # Route registration and matching
│   └── service_registry.go     # Service URL resolution (env-based for now, K8s DNS in prod)
├── ratelimit/
│   ├── limiter.go              # Redis-based sliding window rate limiter
│   └── config.go               # Per-tenant rate limit configuration (default: 1000 req/min, burst: 100)
├── middleware/
│   ├── proxy_auth.go           # JWT validation middleware (validates locally using RSA public key)
│   ├── proxy_metrics.go        # Prometheus metrics collection per route
│   ├── proxy_logging.go        # Structured request/response logging (exclude sensitive headers)
│   └── proxy_headers.go        # Add X-Request-ID, X-Tenant-ID, X-User-ID, X-Forwarded-For
├── health/
│   └── checker.go              # Aggregated health check (pings all backend services)
└── config/
    └── routes.go               # Route configuration loader (YAML + env override)
```

### 4. Rate Limiting Implementation

Use Redis sorted sets (ZRANGEBYSCORE) for sliding window rate limiting:
- Key: `ratelimit:{tenant_id}:{endpoint_group}`
- Default limits: 1000 requests/minute per tenant, 100 requests/second burst
- Different limits for different endpoint groups:
  - Auth endpoints: 20/minute per IP (brute force protection)
  - Read endpoints: 2000/minute per tenant
  - Write endpoints: 500/minute per tenant
  - Admin endpoints: 100/minute per tenant
- Return headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
- Return 429 Too Many Requests with Retry-After header when exceeded

### 5. Circuit Breaker

Implement circuit breaker per backend service:
- Closed → Open: 5 consecutive failures or 50% failure rate in 10-second window
- Open → Half-Open: after 30 seconds
- Half-Open → Closed: 3 consecutive successes
- When open: return 503 Service Unavailable with the service name

### 6. Prometheus Metrics

```
clario360_gateway_requests_total{service, method, status_code, tenant_id}
clario360_gateway_request_duration_seconds{service, method} (histogram)
clario360_gateway_active_connections{service}
clario360_gateway_circuit_breaker_state{service} (0=closed, 1=half-open, 2=open)
clario360_gateway_rate_limit_exceeded_total{tenant_id, endpoint_group}
```

### 7. WebSocket Support

The gateway must support WebSocket proxying for real-time features:
- Cybersecurity suite: real-time alert streaming
- Visus360: live dashboard updates
- Path: /ws/v1/{service}/* — proxied to backend service's WebSocket handler

### 8. CORS Configuration

```go
cors.Options{
    AllowedOrigins:   []string{"https://*.clario360.com", "http://localhost:3000"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID", "X-Tenant-ID"},
    ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
    AllowCredentials: true,
    MaxAge:           3600,
}
```

### 9. Request/Response Logging

Log every request with:
- Request: method, path, query params, content-length, user-agent, IP, tenant_id, user_id, request_id
- Response: status code, content-length, latency_ms
- DO NOT log: Authorization header values, request/response bodies (too large, sensitive data risk)
- Log format: JSON (zerolog)

Generate ALL files with complete, working code including circuit breaker, rate limiter, proxy, metrics, and health check implementations.
```

---

## PROMPT 5: Event Bus Infrastructure — Kafka Producers, Consumers, Event Schema Registry & Dead Letter Queue

```
You are continuing to build the Clario 360 Enterprise AI Platform. The API Gateway (Prompt 4) is operational. Now build the event bus infrastructure that enables asynchronous, event-driven communication between all services.

### Design Principles
- Every state change in any service publishes a domain event
- Events are the single source of truth for cross-service communication
- No service calls another service's database directly
- Events use CloudEvents specification (https://cloudevents.io/) as the envelope
- Event payloads are JSON (with Avro schema registry for validation)
- At-least-once delivery with idempotent consumers

### 1. Event Infrastructure

```
backend/internal/events/
├── producer.go             # Kafka producer with retry, batching, and partitioning
├── consumer.go             # Kafka consumer group with offset management
├── consumer_group.go       # Consumer group manager (start/stop/health)
├── event.go                # CloudEvents-compliant event envelope
├── registry.go             # Event handler registry (topic → handler mapping)
├── middleware.go            # Consumer middleware: logging, metrics, retry, DLQ
├── dead_letter.go          # Dead letter queue producer and consumer
├── serializer.go           # JSON serialization/deserialization with schema validation
├── topics.go               # Topic name constants and configuration
└── health.go               # Kafka health check (broker connectivity, consumer lag)
```

### 2. CloudEvents Envelope

```go
type Event struct {
    // CloudEvents required attributes
    ID              string            `json:"id"`               // UUID
    Source          string            `json:"source"`           // e.g., "clario360/cyber-service"
    SpecVersion     string            `json:"specversion"`      // "1.0"
    Type            string            `json:"type"`             // e.g., "com.clario360.cyber.alert.created"
    
    // CloudEvents optional attributes
    DataContentType string            `json:"datacontenttype"`  // "application/json"
    Subject         string            `json:"subject,omitempty"` // Resource ID
    Time            time.Time         `json:"time"`
    
    // Clario 360 extensions
    TenantID        string            `json:"tenantid"`
    UserID          string            `json:"userid,omitempty"`
    CorrelationID   string            `json:"correlationid"`    // For request tracing
    CausationID     string            `json:"causationid,omitempty"` // ID of event that caused this one
    
    // Payload
    Data            json.RawMessage   `json:"data"`
}
```

### 3. Topic Architecture

```go
// Topic naming: {domain}.{entity}.{action}
var Topics = struct {
    // Platform
    IAMEvents           string  // "platform.iam.events"
    AuditEvents         string  // "platform.audit.events"
    NotificationEvents  string  // "platform.notification.events"
    WorkflowEvents      string  // "platform.workflow.events"
    
    // Cybersecurity
    AssetEvents         string  // "cyber.asset.events"
    ThreatEvents        string  // "cyber.threat.events"
    AlertEvents         string  // "cyber.alert.events"
    RemediationEvents   string  // "cyber.remediation.events"
    
    // Data
    DataSourceEvents    string  // "data.source.events"
    PipelineEvents      string  // "data.pipeline.events"
    QualityEvents       string  // "data.quality.events"
    ContradictionEvents string  // "data.contradiction.events"
    LineageEvents       string  // "data.lineage.events"
    
    // Enterprise
    ActaEvents          string  // "enterprise.acta.events"
    LexEvents           string  // "enterprise.lex.events"
    VisusEvents         string  // "enterprise.visus.events"
    
    // Dead Letter
    DeadLetter          string  // "platform.dead-letter"
}
```

### 4. Producer Implementation

```go
type Producer interface {
    Publish(ctx context.Context, topic string, event Event) error
    PublishBatch(ctx context.Context, topic string, events []Event) error
    Close() error
}
```

Requirements:
- Partition by tenant_id (ensures ordering within a tenant)
- Idempotent producer (enable.idempotence=true)
- Retries: 3 with exponential backoff (100ms, 500ms, 2s)
- Acks: all (wait for all in-sync replicas)
- Compression: snappy
- Batch size: 16KB with 10ms linger
- Include OpenTelemetry trace context in event headers

### 5. Consumer Implementation

```go
type Consumer interface {
    Subscribe(topics []string, handler EventHandler) error
    Start(ctx context.Context) error
    Stop() error
    Health() HealthStatus
}

type EventHandler interface {
    Handle(ctx context.Context, event Event) error
    EventTypes() []string  // Which event types this handler processes
}
```

Requirements:
- Consumer groups per service (group.id = service name)
- Manual offset commit after successful processing
- Concurrency: configurable workers per partition (default: 1 for ordering)
- Retry policy: 3 retries with exponential backoff before sending to DLQ
- Dead Letter Queue: failed events written to "platform.dead-letter" topic with original topic, error, and retry count in headers
- Graceful shutdown: finish processing current batch before stopping
- Consumer lag metric exposed to Prometheus

### 6. Consumer Middleware Chain

```go
type ConsumerMiddleware func(EventHandler) EventHandler

// Built-in middleware:
func WithLogging(logger zerolog.Logger) ConsumerMiddleware      // Log event type, tenant, duration
func WithMetrics(registry *prometheus.Registry) ConsumerMiddleware // Track processing time, success/failure
func WithRetry(maxRetries int, backoff BackoffPolicy) ConsumerMiddleware // Retry failed handlers
func WithDeadLetter(dlqProducer Producer) ConsumerMiddleware    // Send to DLQ after max retries
func WithIdempotency(store IdempotencyStore) ConsumerMiddleware // Skip already-processed events (Redis-based)
func WithTracing(tracer trace.Tracer) ConsumerMiddleware        // OpenTelemetry span per event
```

### 7. Dead Letter Queue Consumer

A separate service/goroutine that:
- Reads from "platform.dead-letter" topic
- Stores failed events in PostgreSQL (dead_letter_events table in platform_core)
- Provides API endpoints for inspecting and replaying failed events:
  ```
  GET    /api/v1/events/dead-letter          # List DLQ events (paginated, filterable)
  GET    /api/v1/events/dead-letter/:id      # Get DLQ event details
  POST   /api/v1/events/dead-letter/:id/replay  # Replay event back to original topic
  DELETE /api/v1/events/dead-letter/:id      # Acknowledge/delete DLQ event
  ```

### 8. Idempotency Store

Redis-based idempotency check:
- Key: `idempotent:{consumer_group}:{event_id}`
- TTL: 7 days
- Before processing, check if event_id exists in Redis
- After successful processing, store event_id in Redis

### 9. Service Integration Pattern

Show a complete example of how a service (e.g., cyber-service) both produces and consumes events:

```go
// In cyber-service/main.go:
// 1. Create producer
// 2. Create consumer
// 3. Register event handlers
// 4. Start consumer in goroutine
// 5. Use producer in service layer when state changes occur
// 6. Graceful shutdown stops consumer, flushes producer
```

### 10. Tests

- Unit tests for producer (mock Kafka with sarama/mocks)
- Unit tests for consumer middleware chain
- Integration test with embedded Kafka (testcontainers)
- Test DLQ flow: publish event → handler fails 3 times → event appears in DLQ
- Test idempotency: same event processed twice → handler called only once

Generate ALL files with complete, production-ready code. Every function must be fully implemented.
```

---

## QA Checklist — Batch 1

After executing Prompts 1–5 with an AI agent, verify:

| Check | Verification Command |
|-------|---------------------|
| Project compiles | `make build` succeeds for all services |
| Docker environment starts | `make docker-up` — all containers healthy |
| Migrations run | `make migrate-up` — all 6 databases created with tables |
| IAM service starts | `go run cmd/iam-service/main.go` — health check returns 200 |
| User registration works | POST /api/v1/auth/register returns JWT |
| Login works | POST /api/v1/auth/login returns access + refresh tokens |
| API Gateway routes | Requests to /api/v1/users proxied to IAM service |
| Rate limiter works | Exceed limit → 429 response with Retry-After |
| Events publish | Login event appears on platform.iam.events topic |
| Tests pass | `make test` — ≥ 80% coverage on service layer |

---

*Batch 2 (Prompts 6–10) covers: Audit Service, Workflow Engine, Observability Stack, Centralized Notification Service, and Shared File Storage/Encryption Service.*
