# Clario 360 — Enterprise AI Platform

[Logo placeholder]

**A Saudi-owned, enterprise-grade platform providing organizations with a unified, real-time, 360-degree view of risk, security, governance, and enterprise data posture.**

[![CI](https://github.com/clario360/platform/actions/workflows/ci.yml/badge.svg)](https://github.com/clario360/platform/actions/workflows/ci.yml)
[![Security Scan](https://github.com/clario360/platform/actions/workflows/security-scan.yml/badge.svg)](https://github.com/clario360/platform/actions/workflows/security-scan.yml)
[![License: Proprietary](https://img.shields.io/badge/License-Proprietary-red.svg)]()

---

## Platform Overview

Clario 360 is a Kubernetes-native, API-first, event-driven enterprise platform consisting of five integrated suites:

AI capabilities in the current codebase are implemented through governed, explainable rule-based and statistical systems across Acta, Lex, and Cyber. See the [AI Capabilities Matrix](docs/architecture/AI_CAPABILITIES_MATRIX.md) for a suite-by-suite breakdown.

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

## Quick Start (Local Development)

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker and Docker Compose
- Make

### Setup (< 5 minutes)

```bash
# Clone the repository
git clone https://github.com/clario360/platform.git
cd platform

# Copy environment files
cp .env.example .env
cp frontend/.env.local.example frontend/.env.local

# Start all infrastructure (PostgreSQL, Redis, Kafka, MinIO)
make docker-up

# Wait for healthy (~30 seconds)
make docker-wait

# Run database migrations
make migrate-up

# Seed development data (500 assets, 200 vulnerabilities, sample data across all suites)
make seed

# Start all backend services (in parallel)
make run-all
# Or start individual services:
# make run SERVICE=iam-service

# In a separate terminal: start the frontend
cd frontend
npm install
npm run dev
```

### Access

- **Frontend:** http://localhost:3000
- **API Gateway:** http://localhost:8080
- **API Docs (Swagger):** http://localhost:8080/docs
- **Grafana:** http://localhost:3000 (admin/admin) — when running via Docker only
- **Jaeger:** http://localhost:16686
- **MinIO Console:** http://localhost:9001 (minioadmin/minioadmin)
- **Prometheus:** http://localhost:9099

### Default Login

- **Email:** admin@clario360.local
- **Password:** Clario360Admin!2026

---

## Project Structure

```
clario360/
├── backend/                    # Go backend (all services)
│   ├── cmd/                    # Service entry points
│   │   ├── api-gateway/        #   HTTP API gateway (port 8080)
│   │   ├── iam-service/        #   Identity & access management (port 8081)
│   │   ├── event-bus/          #   Kafka event processing
│   │   ├── workflow-engine/    #   Workflow orchestration (port 8083)
│   │   ├── audit-service/      #   Centralized audit logging (port 8084)
│   │   ├── notification-service/ # Notification handling (port 8090)
│   │   ├── file-service/       #   File management (port 8091)
│   │   ├── cyber-service/      #   Cybersecurity suite (port 8085)
│   │   ├── data-service/       #   Data suite (port 8086)
│   │   ├── acta-service/       #   Clario Acta (port 8087)
│   │   ├── lex-service/        #   Clario Lex (port 8088)
│   │   ├── visus-service/      #   Clario Visus360 (port 8089)
│   │   ├── migrator/           #   Database migration tool
│   │   └── data-seeder/        #   Development data seeder
│   ├── internal/               # Private packages (per-service + shared)
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
│   │   ├── onboarding/         #   Tenant provisioning
│   │   ├── integration/        #   External integrations
│   │   └── suiteapi/           #   Suite API utilities
│   ├── pkg/                    # Public packages
│   │   ├── validator/          #   Data validation
│   │   ├── crypto/             #   Cryptographic utilities
│   │   ├── storage/            #   Storage abstraction (MinIO, GCS, S3)
│   │   └── httpclient/         #   HTTP client library
│   ├── migrations/             # Database migrations (8 databases)
│   │   ├── platform_core/      #   Core platform schema
│   │   ├── cyber_db/           #   Cybersecurity suite schema
│   │   ├── data_db/            #   Data suite schema
│   │   ├── acta_db/            #   Acta suite schema
│   │   ├── lex_db/             #   Lex suite schema
│   │   ├── visus_db/           #   Visus360 suite schema
│   │   ├── audit_db/           #   Audit service schema
│   │   └── notification_db/    #   Notification service schema
│   └── e2e_tests/              # End-to-end integration tests
│
├── frontend/                   # Next.js 14 frontend
│   ├── src/app/                # App Router pages
│   │   ├── (auth)/             #   Auth pages (login, register, forgot-password)
│   │   ├── (onboarding)/       #   Onboarding flow (verify, setup)
│   │   ├── (dashboard)/        #   Dashboard pages
│   │   │   ├── admin/          #     Admin panel (users, roles, audit, settings)
│   │   │   ├── cyber/          #     Cybersecurity suite
│   │   │   ├── data/           #     Data suite
│   │   │   ├── acta/           #     Clario Acta
│   │   │   ├── lex/            #     Clario Lex
│   │   │   ├── visus/          #     Visus360
│   │   │   ├── workflows/      #     Workflow management
│   │   │   ├── notifications/  #     Notification center
│   │   │   ├── files/          #     File management
│   │   │   └── settings/       #     User settings
│   │   └── api/                #   BFF API routes (auth, health)
│   ├── src/components/         # UI components
│   │   ├── ui/                 #   shadcn/ui base components
│   │   ├── shared/             #   Shared components (charts, data-table, forms)
│   │   ├── layout/             #   Layout components (sidebar, header, breadcrumbs)
│   │   ├── auth/               #   Authentication components
│   │   ├── dashboard/          #   Dashboard-specific components
│   │   ├── cyber/              #   Cybersecurity suite components
│   │   ├── suites/             #   Suite-specific components
│   │   ├── workflows/          #   Workflow components
│   │   ├── notifications/      #   Notification components
│   │   ├── realtime/           #   Real-time data components
│   │   └── providers/          #   Context providers
│   ├── src/lib/                # Utilities (API client, auth, formatting)
│   ├── src/hooks/              # Custom React hooks (21 hooks)
│   ├── src/stores/             # Zustand state stores (5 stores)
│   └── src/types/              # TypeScript type definitions
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
│   └── escrow/                 #   Source code escrow packaging
│
├── docs/                       # Documentation
│   ├── api/                    #   OpenAPI 3.1 specification (~300 endpoints)
│   ├── architecture/           #   Architecture decision records and diagrams
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
├── Makefile                    # Build, test, run, deploy targets
├── go.work                     # Go workspace configuration
├── .env.example                # Backend environment template
└── README.md                   # This file
```

## Services

| Service | Description | Default Port |
|---------|-------------|-------------|
| api-gateway | HTTP API Gateway with rate limiting, circuit breaker, JWT auth | 8080 |
| iam-service | Identity & Access Management (users, roles, permissions, MFA) | 8081 |
| event-bus | Kafka event processing and routing | — |
| workflow-engine | Workflow orchestration with human task support | 8083 |
| audit-service | Centralized audit logging with hash chain integrity | 8084 |
| notification-service | Multi-channel notifications (email, SMS, WebSocket, push) | 8090 |
| file-service | File management with virus scanning and encryption | 8091 |
| cyber-service | Cybersecurity Suite (assets, threats, CTEM, DSPM, vCISO) | 8085 |
| data-service | Data Intelligence Suite (pipelines, quality, lineage) | 8086 |
| acta-service | Board Governance Suite (meetings, minutes, action items) | 8087 |
| lex-service | Legal Operations Suite (contracts, clauses, compliance) | 8088 |
| visus-service | Executive Intelligence Suite (KPIs, dashboards, reports) | 8089 |

## Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Backend | Go 1.22, Chi Router | API services |
| Frontend | Next.js 14, TypeScript, Tailwind CSS, shadcn/ui | User interface |
| Database | PostgreSQL 16 | Primary data store (8 databases, RLS) |
| Cache | Redis 7 | Session cache, rate limiting, job queues |
| Messaging | Apache Kafka 3.7 (KRaft) | Event-driven integration (21 topics) |
| Object Storage | MinIO (S3-compatible) | File storage with encryption |
| Auth | JWT (RS256), TOTP MFA, bcrypt | Authentication and authorization |
| Secrets | HashiCorp Vault | Dynamic credentials, transit encryption |
| Observability | Prometheus, Grafana, Jaeger, OpenTelemetry | Metrics, dashboards, tracing |
| CI/CD | GitHub Actions, ArgoCD | Build, test, deploy |
| Infrastructure | Terraform, Helm, Kubernetes | Provisioning and orchestration |

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

# End-to-end tests (requires full Docker environment)
make e2e-test

# Security tests (gosec, vulnerability checks)
make test-security

# Load tests (k6)
make loadtest SCENARIO=smoke

# All tests (unit + integration + security)
make test-all

# Frontend tests
make frontend-test
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
4. Add Helm deployment: `deploy/helm/clario360/templates/{service}/`
5. Add to CI matrix: `.github/workflows/ci.yml`
6. Add to docker-compose: `docker-compose.yml`
7. Add to Makefile `SERVICES` list
8. Document API: `docs/api/schemas/{service}.yaml`

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

### Local (Docker Compose)

```bash
make docker-up    # Start infrastructure
make run-all      # Start all services
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

## Documentation

| Document | Description |
|----------|-------------|
| [API Reference](docs/api/openapi.yaml) | Complete OpenAPI 3.1 specification |
| [Operational Runbooks](docs/runbooks/) | Operational procedures |
| [Production Checklist](docs/deployment/PRODUCTION_LAUNCH_CHECKLIST.md) | Go-live verification |
| [AI Capabilities Matrix](docs/architecture/AI_CAPABILITIES_MATRIX.md) | Implemented AI capabilities by suite, implementation type, governance coverage, and user-facing surface |

## Security

- Report vulnerabilities to: security@clario360.sa
- See [SECURITY.md](SECURITY.md) for our security policy
- All AI outputs are explainable and auditable — no black-box models
- Full audit trail with hash chain integrity verification
- Multi-tenant with PostgreSQL Row-Level Security

## License

Proprietary — Clario 360. All rights reserved. See [LICENSE](LICENSE) for terms.
Full source code ownership by Clario 360. See escrow documentation for IP transfer details.
