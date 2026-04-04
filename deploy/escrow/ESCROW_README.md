# Clario 360 — Source Code Escrow Package

## Overview

This escrow package contains the complete source code, build tools, vendored
dependencies, deployment configurations, and operational documentation for the
**Clario 360 Enterprise AI Governance Platform**. Per RFP §14, this package
enables Clario 360 to be built, deployed, operated, maintained, and extended
independently — without vendor contact and without internet access.

---

## Package Contents

```
escrow-<version>/
├── source/                    Complete source code
│   ├── backend/               Go backend (14 microservices)
│   │   ├── cmd/               Service entry points
│   │   ├── internal/          Service-specific packages
│   │   ├── pkg/               Shared packages
│   │   ├── vendor/            Vendored Go dependencies (offline build)
│   │   ├── migrations/        Database migration files
│   │   └── go.mod / go.sum    Dependency manifest
│   ├── frontend/              Next.js 14 TypeScript frontend
│   │   ├── app/               App Router pages (~50 pages)
│   │   ├── components/        UI components (shadcn/ui + custom)
│   │   ├── lib/               Utility libraries
│   │   ├── stores/            Zustand state management
│   │   └── package.json       npm dependency manifest
│   ├── VERSION                Package version
│   └── COMMIT_SHA             Git commit hash for traceability
│
├── dependencies/              Vendored dependencies for offline builds
│   ├── node_modules/          Cached npm packages
│   ├── npm-cache.tar.gz       npm cache archive (alternative)
│   └── images/                Pre-built Docker images (tar archives)
│       ├── api-gateway-*.tar
│       ├── iam-service-*.tar
│       ├── ... (all services)
│       ├── golang-1.25*-alpine.tar   (base images)
│       ├── node-20-alpine.tar
│       └── postgres-16-alpine.tar
│
├── build/                     Build documentation
│   └── BUILD_INSTRUCTIONS.md  Complete build guide (online & offline)
│
├── deploy/                    Deployment artifacts
│   ├── helm/                  Kubernetes Helm charts
│   │   └── clario360/         Main chart with values for all environments
│   ├── terraform/             Infrastructure as Code
│   │   ├── modules/           Reusable modules (networking, database, etc.)
│   │   └── environments/      Per-environment configurations
│   ├── docker/                Dockerfiles for all services
│   ├── grafana/               Monitoring dashboards
│   ├── prometheus/            Metrics collection configuration
│   ├── migrations/            Database migration files (8 databases)
│   ├── docker-compose.yml     Local development environment
│   ├── docker-compose.test.yml Test environment
│   └── Makefile               Build automation
│
├── docs/                      Documentation
│   ├── ESCROW_README.md       This file
│   ├── api/                   API placeholder or generated references
│   ├── architecture/          Architecture decision records
│   └── runbooks/              Operational runbooks (~35 procedures)
│       ├── deployment/        Release and deployment procedures
│       ├── incident-response/ Incident handling playbooks
│       ├── operations/        Day-to-day operational procedures
│       ├── scaling/           Scaling and performance guides
│       └── troubleshooting/   Diagnostic procedures
│
├── verification/              Package verification scripts
│   ├── verify-integrity.sh    Checks completeness and checksums
│   └── verify-build.sh        Verifies buildability from source
│
├── tools/                     CLI tools for offline environments
│   ├── go1.25*.tar.gz         Go compiler
│   ├── node-v20.11.0.tar.xz   Node.js runtime
│   ├── kubectl                Kubernetes CLI
│   ├── helm                   Helm chart manager
│   └── terraform              Infrastructure provisioning
│
├── manifest.json              Machine-readable package manifest
└── SHA256SUMS                 Checksums for all files
```

---

## Getting Started

### Step 1: Verify Package Integrity

```bash
tar xzf clario360-escrow-<version>.tar.gz
cd escrow-<version>

# Verify GPG signature (if signed)
gpg --verify ../clario360-escrow-<version>.tar.gz.asc \
    ../clario360-escrow-<version>.tar.gz

# Verify all file checksums and completeness
./verification/verify-integrity.sh
```

### Step 2: Review Build Instructions

Read `build/BUILD_INSTRUCTIONS.md` for complete build procedures covering:
- Online builds (with internet access)
- Offline/air-gapped builds (using vendored dependencies)
- Container image builds
- Database setup and migrations

### Step 3: Build from Source

```bash
# Install tools (if in air-gapped environment)
tar xzf tools/go1.25*.tar.gz -C /usr/local
export PATH=$PATH:/usr/local/go/bin

# Build backend
cd source/backend
GOWORK=off go build -mod=vendor ./...

# Build frontend
cd ../frontend
cp -r ../../dependencies/node_modules ./
npm run build
```

### Step 4: Verify Build

```bash
./verification/verify-build.sh
```

### Step 5: Deploy

Choose your deployment target:

| Target | Guide |
|--------|-------|
| Local (Docker Compose) | `deploy/docker-compose.yml` |
| Kubernetes (Helm) | `deploy/helm/clario360/` |
| Cloud (Terraform + Helm) | `deploy/terraform/` then `deploy/helm/` |

---

## Key Reference Documents

| Document | Location | Purpose |
|----------|----------|---------|
| Architecture References | `docs/architecture/` | Current architecture notes, solution design, and capability matrices |
| Gateway Route Map | `source/backend/internal/gateway/config/routes.go` | Checked-in gateway route ownership and service routing |
| Backend Service Entry Points | `source/backend/cmd/` | Service handlers and live API entrypoints |
| Helm Chart Values | `deploy/helm/clario360/values.yaml` | All configurable deployment parameters |
| Terraform Modules | `deploy/terraform/modules/` | Infrastructure provisioning (GCP) |
| Operational Runbooks | `docs/runbooks/` | ~35 procedures for platform operations |
| AI Governance | `docs/architecture/AI_CAPABILITIES_MATRIX.md` | Implemented AI capabilities and governance coverage |

---

## Platform Architecture

### Services

| Service | Language | Description |
|---------|----------|-------------|
| api-gateway | Go | Request routing, rate limiting, circuit breaker, WebSocket proxy |
| iam-service | Go | Identity management, RS256 JWT authentication, RBAC |
| event-bus | Go | Event streaming and inter-service messaging (Kafka) |
| workflow-engine | Go | Workflow orchestration with human task management |
| audit-service | Go | Immutable audit logging with cryptographic hash chain |
| cyber-service | Go | Cybersecurity risk management and vulnerability tracking |
| data-service | Go | Data management, quality monitoring, lineage |
| acta-service | Go | Contract and asset lifecycle tracking |
| lex-service | Go | Legal compliance and regulatory obligation management |
| visus-service | Go | Dashboard visualization and report generation |
| file-service | Go | Encrypted file storage with virus scanning |
| notification-service | Go | Multi-channel notifications (email, SMS, WebSocket) |
| migrator | Go | Database schema migration runner |
| frontend | TypeScript | Next.js 14 dashboard (App Router, Tailwind, shadcn/ui) |

### Databases (PostgreSQL 16)

| Database | Purpose |
|----------|---------|
| platform_core | Users, tenants, roles, permissions, settings |
| cyber_db | Vulnerabilities, risks, assets, security controls |
| data_db | Datasets, data quality, lineage tracking |
| acta_db | Contracts, vendors, licenses, SLAs |
| audit_db | Immutable audit event chain |
| notification_db | Notification channels, preferences, delivery |
| lex_db | Regulations, compliance mappings, obligations |
| visus_db | Dashboards, reports, widget configurations |

### Infrastructure Dependencies

| Component | Version | Purpose |
|-----------|---------|---------|
| PostgreSQL | 16+ | Primary data store (per-service databases) |
| Redis | 7+ | Caching, rate limiting, session storage |
| Apache Kafka | 3.7+ | Event streaming, inter-service messaging |
| Kubernetes | 1.28+ | Container orchestration |
| Prometheus | 2.50+ | Metrics collection |
| Grafana | 10.3+ | Monitoring dashboards |

---

## Support Independence

This package provides everything needed to:

1. **Build** the platform from source code (no internet required)
2. **Deploy** to any Kubernetes cluster (cloud or on-premises)
3. **Operate** with monitoring, alerting, and operational runbooks
4. **Troubleshoot** using diagnostic procedures and runbooks
5. **Back up and restore** all data and configurations
6. **Scale** services horizontally based on load
7. **Update** database schemas via versioned migrations
8. **Extend** with new features following established patterns
9. **Maintain compliance** with ISO 27001, NCA ECC, SAMA CSF frameworks

---

## Compliance

The platform implements security controls aligned with:

- **ISO 27001:2022** — Information security management
- **NCA ECC** — National Cybersecurity Authority Essential Controls
- **SAMA CSF** — Saudi Arabian Monetary Authority Cyber Security Framework
- **NDMO** — National Data Management Office data governance requirements

Data residency: All data stored in Saudi Arabia (GCP ME-CENTRAL2 region).
Encryption: AES-256-GCM at rest, TLS 1.3 in transit.

---

## Contact

For questions about this escrow package, contact the escrow administrator
or refer to the escrow agreement for release conditions and procedures.
