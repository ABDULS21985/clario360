# Clario 360 — Enterprise AI Platform

[Logo placeholder]

**A Saudi-owned, enterprise-grade platform providing organizations with a unified, real-time, 360-degree view of risk, security, governance, and enterprise data posture.**

[![CI](https://github.com/clario360/platform/actions/workflows/ci.yml/badge.svg)](https://github.com/clario360/platform/actions/workflows/ci.yml)
[![Security Scan](https://github.com/clario360/platform/actions/workflows/security-scan.yml/badge.svg)](https://github.com/clario360/platform/actions/workflows/security-scan.yml)
[![License: Proprietary](https://img.shields.io/badge/License-Proprietary-red.svg)]()

---

## Platform Overview

Clario 360 is a Kubernetes-native, API-first, event-driven enterprise platform consisting of five integrated suites:

AI capabilities in the current codebase are implemented through governed, explainable rule-based and statistical systems across Acta, Lex, and Cyber. See the [AI Capabilities Matrix](docs/architecture/AI_CAPABILITIES_MATRIX.md) and the [AI Model Catalog](docs/AI_MODEL_CATALOG.md) for details.

| Suite | Purpose | Key Capabilities |
|-------|---------|-----------------|
| **Cybersecurity** | Proactive threat detection and exposure management | Asset discovery, AI-powered threat detection, CTEM, governed remediation, DSPM, Virtual CISO |
| **Data Intelligence** | Trusted enterprise data and governed analytics | Data source management, pipeline orchestration, quality monitoring, contradiction detection, lineage |
| **Board Governance (Acta)** | Board committee operations automation | Meeting management, AI minutes generation, action item tracking, governance compliance |
| **Legal Operations (Lex)** | Contract lifecycle and compliance management | Contract analysis, clause extraction, risk scoring, expiry monitoring, compliance alerts |
| **Executive Intelligence (Visus360)** | Strategic command center | Cross-suite dashboards, KPI engine, executive alerts, report generation |

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ FRONTEND (Next.js 14 · TypeScript · Tailwind CSS)           │
├─────────────────────────────────────────────────────────────┤
│ API GATEWAY (Go · Chi Router · Rate Limiting · JWT Auth)    │
├─────────┬──────────┬──────────┬──────────┬─────────────────┤
│ Cyber   │ Data     │ Acta     │ Lex      │ Visus360        │
│ Service │ Service  │ Service  │ Service  │ Service         │
├─────────┴──────────┴──────────┴──────────┴─────────────────┤
│ PLATFORM CORE                                               │
│ IAM · Audit · Workflow · Notifications · File Storage       │
├─────────────────────────────────────────────────────────────┤
│ INFRASTRUCTURE                                              │
│ PostgreSQL · Redis · Kafka · MinIO · Vault · Prometheus     │
├─────────────────────────────────────────────────────────────┤
│ KUBERNETES (On-Prem · Private Cloud · KSA Public · Air-Gap) │
└─────────────────────────────────────────────────────────────┘
```

---

## Quick Start (Local Development)

### Prerequisites

- Go 1.25+
- Node.js 20+
- Docker and Docker Compose
- PM2 (`npm install -g pm2`)
- `openssl`

### Option 1: Startup Script (Recommended)

`./scripts/start.sh` is the most reliable end-to-end local bootstrap path. It generates dev JWT/encryption keys, starts Docker infrastructure, runs migrations, seeds data, builds and starts all services:

```bash
./scripts/start.sh
```

Useful variants:

```bash
./scripts/start.sh --infra-only     # Infrastructure + migrations only
./scripts/start.sh --no-build       # Skip binary rebuild
./scripts/start.sh --skip-frontend  # Backend only

./scripts/status.sh                 # Check all services
./scripts/status.sh --watch         # Continuous monitoring
./scripts/stop.sh                   # Stop all services
./scripts/stop.sh --infra           # Stop infrastructure only
```

### Option 2: PM2 Ecosystem

```bash
# Start infrastructure
make docker-up
make docker-wait

# Run migrations and seed
make migrate-up
make seed

# Start all 12 services via PM2
pm2 start ecosystem.config.js

# Management
pm2 status                              # Check service health
pm2 logs                                # Tail all logs
pm2 logs clario360-api-gateway          # Tail specific service logs
pm2 restart clario360-iam-service       # Restart a specific service
pm2 stop ecosystem.config.js            # Stop all Clario360 services
pm2 delete ecosystem.config.js          # Remove from PM2 process list
```

### Option 3: Makefile

```bash
make docker-up      # Start infrastructure
make docker-wait    # Wait for healthy
make migrate-up     # Run migrations
make seed           # Seed dev data
make run-all        # Start all backend services

# In a separate terminal:
cd frontend && npm install && npm run dev
```

---

## Access

### Default Runtime (`scripts/start.sh`)

| Surface | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| API Gateway | http://localhost:8080 |
| Gateway Health | http://localhost:8080/healthz |
| MinIO Console | http://localhost:9001 |
| Jaeger | http://localhost:16686 |

### PM2 Ecosystem Runtime

| Surface | URL |
|---------|-----|
| Frontend | http://localhost:3002 |
| API Gateway | http://localhost:8080 |
| MinIO Console | http://localhost:9001 |

### Full Docker Compose Add-ons

| Surface | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin / clario360 |
| Prometheus | http://localhost:9099 | — |
| Keycloak | http://localhost:8180 | admin / admin |
| Schema Registry | http://localhost:8081 | — |

> If you run the full Docker Compose stack, Grafana claims port 3000. Run the frontend on a different port (e.g., `PORT=3001 npm run dev`).

### Default Dev Credentials

| Surface | Username / Email | Password |
|---------|-----------------|----------|
| Application admin | `admin@clario.dev` | `Cl@rio360Dev!` |
| MinIO console | `clario_minio` | `clario_minio_secret` |
| Grafana | `admin` | `clario360` |
| Keycloak | `admin` | `admin` |

---

## Services

| Service | Description | HTTP Port | Admin Port |
|---------|-------------|-----------|------------|
| api-gateway | HTTP API Gateway with rate limiting, circuit breaker, JWT auth | 8080 | 9080 |
| iam-service | Identity & Access Management (users, roles, permissions, MFA) | 8081 | 9081 |
| event-bus | Kafka event processing and routing | — | — |
| workflow-engine | Workflow orchestration with human task support | 8083 | — |
| audit-service | Centralized audit logging with hash chain integrity | 8084 | — |
| notification-service | Multi-channel notifications (email, SMS, WebSocket, push) | 8090 | — |
| file-service | File management with virus scanning and encryption | 8091 | — |
| cyber-service | Cybersecurity Suite (assets, threats, CTEM, DSPM, vCISO) | 8085 | — |
| data-service | Data Intelligence Suite (pipelines, quality, lineage) | 8086 | 9086 |
| acta-service | Board Governance Suite (meetings, minutes, action items) | 8087 | 9087 |
| lex-service | Legal Operations Suite (contracts, clauses, compliance) | 8088 | 9088 |
| visus-service | Executive Intelligence Suite (KPIs, dashboards, reports) | 8089 | 9089 |

---

## Project Structure

```
clario360/
├── backend/                    # Go backend (all services)
│   ├── cmd/                    # Service entry points (23 binaries)
│   │   ├── api-gateway/        #   HTTP API gateway (port 8080)
│   │   ├── iam-service/        #   Identity & access management (port 8081)
│   │   ├── event-bus/          #   Kafka event processing
│   │   ├── workflow-engine/    #   Workflow orchestration (port 8083)
│   │   ├── audit-service/      #   Centralized audit logging (port 8084)
│   │   ├── audit-isolation/    #   Isolated audit processing
│   │   ├── notification-service/ # Notification handling (port 8090)
│   │   ├── file-service/       #   File management (port 8091)
│   │   ├── cyber-service/      #   Cybersecurity suite (port 8085)
│   │   ├── data-service/       #   Data suite (port 8086)
│   │   ├── acta-service/       #   Clario Acta (port 8087)
│   │   ├── lex-service/        #   Clario Lex (port 8088)
│   │   ├── visus-service/      #   Clario Visus360 (port 8089)
│   │   ├── migrator/           #   Database migration tool
│   │   ├── data-seeder/        #   Development data seeder
│   │   ├── seeder/             #   General-purpose seeder
│   │   ├── system-seeder/      #   System data seeder
│   │   └── prompt59-seeder/    #   Prompt-based data seeder
│   ├── internal/               # Private packages (28 packages)
│   │   ├── config/             #   Configuration loading
│   │   ├── server/             #   Shared HTTP server setup
│   │   ├── middleware/         #   HTTP middleware stack
│   │   ├── database/           #   PostgreSQL pool, migrations, query builder
│   │   ├── events/             #   Kafka producer/consumer, CloudEvents
│   │   ├── auth/               #   JWT, RBAC, context helpers
│   │   ├── observability/      #   Logging, metrics, tracing
│   │   ├── security/           #   Security hardening, CSRF, sanitization
│   │   ├── errors/             #   Structured error handling
│   │   ├── types/              #   Shared domain types
│   │   ├── iam/                #   IAM service internals
│   │   ├── audit/              #   Audit service internals
│   │   ├── workflow/           #   Workflow engine internals
│   │   ├── notification/       #   Notification service internals
│   │   ├── filemanager/        #   File management internals
│   │   ├── gateway/            #   API gateway internals
│   │   ├── cyber/              #   Cybersecurity suite internals
│   │   ├── data/               #   Data suite internals
│   │   ├── acta/               #   Acta suite internals
│   │   ├── lex/                #   Lex suite internals
│   │   ├── visus/              #   Visus360 suite internals
│   │   ├── aigovernance/       #   AI governance framework
│   │   ├── notebook/           #   Notebook/analysis engine
│   │   ├── prd/                #   PRD generation utilities
│   │   ├── rca/                #   Root cause analysis
│   │   ├── onboarding/         #   Tenant provisioning
│   │   ├── integration/        #   External integrations
│   │   └── suiteapi/           #   Suite API utilities
│   ├── pkg/                    # Public packages
│   │   ├── validator/          #   Data validation
│   │   ├── crypto/             #   Cryptographic utilities
│   │   ├── storage/            #   Storage abstraction (MinIO, GCS, S3)
│   │   └── httpclient/         #   HTTP client library
│   ├── integration_tests/      # Cross-service and tenant isolation tests
│   ├── frameworks/             # Framework and compliance helpers (SOC2)
│   └── migrations/             # Database migrations (8 databases)
│       ├── platform_core/      #   Core platform schema
│       ├── cyber_db/           #   Cybersecurity suite schema
│       ├── data_db/            #   Data suite schema
│       ├── acta_db/            #   Acta suite schema
│       ├── lex_db/             #   Lex suite schema
│       ├── visus_db/           #   Visus360 suite schema
│       ├── audit_db/           #   Audit service schema
│       └── notification_db/    #   Notification service schema
│
├── frontend/                   # Next.js 14 frontend
│   ├── src/app/                # App Router pages
│   │   ├── (auth)/             #   Auth pages (login, register, forgot-password)
│   │   ├── (onboarding)/       #   Onboarding flow (verify, setup)
│   │   ├── (dashboard)/        #   Dashboard pages
│   │   │   ├── admin/          #     Admin panel
│   │   │   │   ├── ai-governance/ #   AI model governance
│   │   │   │   ├── api-keys/   #     API key management
│   │   │   │   ├── audit/      #     Audit logs
│   │   │   │   ├── integrations/ #   External integrations
│   │   │   │   ├── invitations/ #    User invitations
│   │   │   │   ├── notifications/ #  Notification management
│   │   │   │   ├── roles/      #     Role management
│   │   │   │   ├── settings/   #     Admin settings
│   │   │   │   ├── tenants/    #     Tenant management
│   │   │   │   ├── users/      #     User management
│   │   │   │   └── workflows/  #     Workflow administration
│   │   │   ├── cyber/          #     Cybersecurity suite
│   │   │   ├── data/           #     Data suite
│   │   │   ├── acta/           #     Clario Acta
│   │   │   ├── lex/            #     Clario Lex
│   │   │   ├── visus/          #     Visus360
│   │   │   ├── notebooks/      #     Notebook/analysis views
│   │   │   ├── workflows/      #     Workflow management
│   │   │   ├── notifications/  #     Notification center
│   │   │   ├── files/          #     File management
│   │   │   ├── dashboard/      #     Main dashboard
│   │   │   └── settings/       #     User settings
│   │   └── api/                #   BFF API routes (auth session, refresh, health)
│   ├── src/components/         # UI components (12 directories)
│   │   ├── ui/                 #   shadcn/ui base components
│   │   ├── shared/             #   Shared components (charts, data-table, forms)
│   │   ├── common/             #   Common reusable components
│   │   ├── layout/             #   Layout components (sidebar, header, breadcrumbs)
│   │   ├── auth/               #   Authentication components
│   │   ├── dashboard/          #   Dashboard-specific components
│   │   ├── cyber/              #   Cybersecurity suite components
│   │   ├── suites/             #   Suite-specific components
│   │   ├── workflows/          #   Workflow components
│   │   ├── notifications/      #   Notification components
│   │   ├── realtime/           #   Real-time data components
│   │   └── providers/          #   Context providers
│   ├── src/lib/                # Utilities (API client, auth, formatting, security)
│   ├── src/hooks/              # Custom React hooks (35 hooks)
│   ├── src/stores/             # Zustand state stores (6 stores)
│   └── src/types/              # TypeScript type definitions (13 type modules)
│
├── sdks/                       # Client SDKs
│   └── python/                 #   Python SDK and CLI (clario360 package)
│
├── scripts/                    # Development & operations scripts
│   ├── start.sh                #   Full platform startup
│   ├── stop.sh                 #   Stop all services
│   ├── status.sh               #   Check service status
│   └── smoke-test.sh           #   Smoke test runner
│
├── deploy/                     # Deployment artifacts
│   ├── helm/clario360/         #   Helm chart (K8s manifests for all services)
│   ├── terraform/              #   Infrastructure as Code
│   │   ├── environments/       #     Per-environment configs
│   │   ├── modules/            #     Reusable Terraform modules
│   │   └── scripts/            #     Automation scripts
│   ├── docker/                 #   Dockerfiles (backend multi-stage, frontend)
│   ├── grafana/                #   Grafana dashboards and provisioning
│   ├── prometheus/             #   Prometheus configuration and rules
│   ├── monitoring/             #   Additional monitoring configuration
│   ├── jupyter/                #   Jupyter notebook setup
│   └── escrow/                 #   Source code escrow packaging
│
├── docs/                       # Documentation
│   ├── api/                    #   OpenAPI 3.1 specification
│   ├── architecture/           #   Architecture decision records and diagrams
│   ├── prd/                    #   Product requirements documents
│   ├── prompts/                #   AI prompt templates
│   ├── cpu-inference-client-package/ # CPU inference client docs
│   └── runbooks/               #   Operational runbooks
│       ├── deployment/         #     Deployment procedures
│       ├── incident-response/  #     Incident response guides
│       ├── operations/         #     Operations procedures
│       ├── scaling/            #     Scaling guides
│       ├── troubleshooting/    #     Troubleshooting guides
│       └── scripts/            #     Helper scripts
│
├── docker-compose.yml          # Local development infrastructure
├── docker-compose.test.yml     # Test-specific infrastructure
├── ecosystem.local.js          # PM2 ecosystem config (all services)
├── Makefile                    # Build, test, run, deploy targets
├── go.work                     # Go workspace configuration
├── .env.example                # Backend environment template
├── CHANGELOG.md                # Version changelog
├── CONTRIBUTING.md             # Contribution guidelines
├── SECURITY.md                 # Security policy
├── LICENSE                     # License terms
└── README.md                   # This file
```

## Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Backend | Go 1.25, Chi Router, pgx | API services |
| Frontend | Next.js 14.1.4, React 18, TypeScript 5, Tailwind CSS, shadcn/ui | User interface |
| Database | PostgreSQL 16 | Primary data store (8 databases, RLS) |
| Cache | Redis 7 | Session cache, rate limiting, job queues |
| Messaging | Apache Kafka (KRaft mode) | Event-driven integration |
| Object Storage | MinIO (S3-compatible) | File storage with encryption |
| Auth | JWT (RS256), TOTP MFA, bcrypt | Authentication and authorization |
| Secrets | HashiCorp Vault | Dynamic credentials, transit encryption |
| Observability | Prometheus, Grafana, Jaeger, OpenTelemetry | Metrics, dashboards, tracing |
| CI/CD | GitHub Actions, ArgoCD | Build, test, deploy |
| Infrastructure | Terraform, Helm, Kubernetes | Provisioning and orchestration |

---

## Development

### Running Tests

```bash
# Unit tests (fast, no external dependencies)
make test

# Unit tests with coverage report
make test-cover

# Short tests only (skip integration)
make test-short

# Integration tests (requires Docker — spins up test databases)
make test-integration

# Security tests (gosec, vulnerability checks)
make test-security

# All tests (unit + integration + security)
make test-all

# Frontend tests
make frontend-test

# Smoke test (requires running services)
./scripts/smoke-test.sh
```

### Code Quality

```bash
# Go linting
make lint

# Go formatting
make fmt

# Frontend linting
make frontend-lint

# Validate OpenAPI spec
make validate-api
```

### Code Generation

```bash
# Generate TypeScript types from OpenAPI spec
make generate-sdk

# Generate Go mocks for testing
make generate-mocks
```

### Database Migrations

```bash
# Create new migration
make migrate-create NAME=add_new_table

# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check migration status
make migrate-status
```

### Adding a New Service

1. Create entry point: `backend/cmd/{service-name}/main.go`
2. Create internal package: `backend/internal/{service}/`
3. Add database (if needed): create migration directory in `backend/migrations/`
4. Add to PM2 ecosystem: `ecosystem.local.js`
5. Add Helm deployment: `deploy/helm/clario360/templates/{service}/`
6. Add to CI matrix: `.github/workflows/ci.yml`
7. Add to docker-compose: `docker-compose.yml`
8. Add to Makefile `SERVICES` list
9. Document API: `docs/api/schemas/{service}.yaml`

---

## Deployment

### Production (KSA Public Cloud)

```bash
cd deploy/terraform/environments/production
terraform init && terraform apply
# Then: ArgoCD syncs Helm chart automatically
```

### Air-Gapped

```bash
# On internet-connected machine:
deploy/airgap/create-bundle.sh v1.0.0

# Transfer bundle to air-gapped environment, then:
deploy/airgap/deploy.sh
```

### Docker Images

```bash
# Build all Docker images
make docker-build

# Build a specific service image
make docker-build-api-gateway
```

### Helm

```bash
# Lint Helm chart
make helm-lint

# Render Helm templates locally
make helm-template
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [API Reference](docs/api/) | OpenAPI 3.1 specification |
| [Architecture](docs/architecture/) | Architecture decision records and diagrams |
| [AI Capabilities Matrix](docs/architecture/AI_CAPABILITIES_MATRIX.md) | AI capabilities by suite, governance coverage |
| [AI Model Catalog](docs/AI_MODEL_CATALOG.md) | Registered AI/ML model catalog |
| [Feature Parity](docs/architecture/FRONTEND_BACKEND_FEATURE_PARITY.md) | Frontend-backend feature alignment |
| [Product Requirements](docs/prd/) | Product requirements documents |
| [Operational Runbooks](docs/runbooks/) | Deployment, incident response, scaling guides |
| [CPU Inference Client](docs/cpu-inference-client-package/) | CPU inference client package docs |
| [Terraform](deploy/terraform/README.md) | Infrastructure provisioning docs |
| [Python SDK](sdks/python/README.md) | Python SDK and CLI docs |
| [Backend Testing](TESTING-BACKEND.md) | Backend test strategy and patterns |
| [Frontend Testing](TESTING-FRONTEND.md) | Frontend test strategy and patterns |
| [Changelog](CHANGELOG.md) | Version history and release notes |
| [Contributing](CONTRIBUTING.md) | Contribution guidelines |

---

## Security

- Report vulnerabilities to: security@clario360.sa
- See [SECURITY.md](SECURITY.md) for our security policy
- All AI outputs are explainable and auditable — no black-box models
- Full audit trail with hash chain integrity verification
- Multi-tenant with PostgreSQL Row-Level Security
- Tenant isolation enforced at database and service level

## License

Proprietary — Clario 360. All rights reserved. See [LICENSE](LICENSE) for terms.
Full source code ownership by Clario 360. See escrow documentation for IP transfer details.
