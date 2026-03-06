# Clario 360 — Enterprise AI Platform

Saudi-owned, Kubernetes-native, multi-suite enterprise AI platform.

## Tech Stack

- **Backend:** Go 1.22+, Chi router, PostgreSQL 16, Redis, Kafka
- **Frontend:** Next.js 14 (App Router), TypeScript, Tailwind CSS, shadcn/ui
- **Infrastructure:** Kubernetes, Helm, Terraform, Docker

## Quick Start

```bash
# 1. Copy environment configuration
cp .env.example .env

# 2. Start infrastructure dependencies
make docker-up

# 3. Run database migrations
make migrate-up

# 4. Start the API gateway
make dev

# 5. (In another terminal) Start the frontend
make frontend-install
make frontend-dev
```

## Project Structure

```
clario360/
├── backend/         # Go services (monorepo)
│   ├── cmd/         # Service entry points
│   ├── internal/    # Shared internal packages
│   ├── pkg/         # Public packages
│   └── migrations/  # SQL migrations
├── frontend/        # Next.js application
├── deploy/          # Helm, Terraform, Dockerfiles
└── docs/            # Documentation
```

## Make Targets

```bash
make help            # Show all available targets
make build           # Build all backend services
make test            # Run tests with race detector
make lint            # Run golangci-lint
make docker-up       # Start local dependencies
make docker-down     # Stop local dependencies
make migrate-up      # Run database migrations
make migrate-down    # Rollback last migration
make seed            # Seed development data
```

## Services

| Service | Description | Default Port |
|---------|-------------|-------------|
| api-gateway | HTTP API Gateway | 8080 |
| iam-service | Identity & Access Management | 8081 |
| event-bus | Kafka event processing | — |
| workflow-engine | Workflow orchestration | 8083 |
| audit-service | Centralized audit logging | 8084 |
| cyber-service | Cybersecurity Suite | 8085 |
| data-service | Data Suite | 8086 |
| acta-service | Clario Acta | 8087 |
| lex-service | Clario Lex | 8088 |
| visus-service | Clario Visus360 | 8089 |
# clario360
