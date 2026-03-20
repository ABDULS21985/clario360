# Clario 360 — Build Instructions

This document provides complete instructions for building the Clario 360 platform
from source code, both with and without internet access.

## Prerequisites

| Tool       | Version | Included in `tools/` | Purpose                    |
|------------|---------|----------------------|----------------------------|
| Go         | 1.22+   | Yes                  | Backend service compilation |
| Node.js    | 20 LTS  | Yes                  | Frontend build              |
| Docker     | 24+     | No (host install)    | Container image builds      |
| Make       | any     | No (system package)  | Build automation            |
| PostgreSQL | 16+     | No (runtime dep)     | Database server             |
| Redis      | 7+      | No (runtime dep)     | Cache and rate limiting     |
| Kafka      | 3.7+    | No (runtime dep)     | Event streaming             |

## Installing Included Tools (Air-Gapped)

```bash
# Go compiler
tar xzf tools/go1.22.0.tar.gz -C /usr/local
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go

# Node.js
tar xJf tools/node-v20.11.0.tar.xz -C /usr/local --strip-components=1

# kubectl, helm, terraform
cp tools/kubectl tools/helm tools/terraform /usr/local/bin/
chmod +x /usr/local/bin/{kubectl,helm,terraform}

# Verify installations
go version        # go1.22.0
node --version    # v20.11.0
kubectl version --client
helm version
terraform version
```

---

## Building from Source (Online)

When you have internet access, building is straightforward:

```bash
cd source/

# Build all Go backend services
cd backend
GOWORK=off go build ./...
cd ..

# Build frontend
cd frontend
npm ci
npm run build
cd ..
```

### Using the Makefile

```bash
cd source/
make build          # Builds all Go services to bin/
make test           # Runs all unit tests
make lint           # Runs Go linter
make docker-build   # Builds all Docker images
```

---

## Building from Source (Offline / Air-Gapped)

### Backend Services

All Go dependencies are vendored in `source/backend/vendor/`. Use `-mod=vendor`
to build without internet access:

```bash
cd source/backend

# Build individual services
GOWORK=off go build -mod=vendor -o ../../bin/api-gateway      ./cmd/api-gateway/
GOWORK=off go build -mod=vendor -o ../../bin/iam-service      ./cmd/iam-service/
GOWORK=off go build -mod=vendor -o ../../bin/event-bus        ./cmd/event-bus/
GOWORK=off go build -mod=vendor -o ../../bin/workflow-engine  ./cmd/workflow-engine/
GOWORK=off go build -mod=vendor -o ../../bin/audit-service    ./cmd/audit-service/
GOWORK=off go build -mod=vendor -o ../../bin/cyber-service    ./cmd/cyber-service/
GOWORK=off go build -mod=vendor -o ../../bin/data-service     ./cmd/data-service/
GOWORK=off go build -mod=vendor -o ../../bin/acta-service     ./cmd/acta-service/
GOWORK=off go build -mod=vendor -o ../../bin/lex-service      ./cmd/lex-service/
GOWORK=off go build -mod=vendor -o ../../bin/visus-service    ./cmd/visus-service/
GOWORK=off go build -mod=vendor -o ../../bin/file-service     ./cmd/file-service/
GOWORK=off go build -mod=vendor -o ../../bin/notification-service ./cmd/notification-service/
GOWORK=off go build -mod=vendor -o ../../bin/migrator         ./cmd/migrator/

# Build all at once
GOWORK=off go build -mod=vendor -o /dev/null ./...
```

### Frontend

```bash
cd source/frontend

# Copy vendored node_modules (from dependencies/)
cp -r ../../dependencies/node_modules ./

# Build production bundle
npm run build

# Output is in .next/ directory
```

---

## Building Container Images

### Option 1: Use Pre-Built Images

Pre-built container images are included in `dependencies/images/`:

```bash
# Load all application images
for img in dependencies/images/*-*.tar; do
    docker load -i "$img"
    echo "Loaded: $img"
done

# Verify
docker images | grep clario360
```

### Option 2: Build from Source (Offline)

```bash
# First, load base images
docker load -i dependencies/images/golang-1.22-alpine.tar
docker load -i dependencies/images/node-20-alpine.tar
docker load -i dependencies/images/gcr.io-distroless-static-debian12-nonroot.tar

# Build backend services
SERVICES=(api-gateway iam-service event-bus workflow-engine audit-service
          cyber-service data-service acta-service lex-service visus-service
          file-service notification-service migrator)

for svc in "${SERVICES[@]}"; do
    docker build \
        -f deploy/docker/Dockerfile.backend \
        --build-arg BUILD_SERVICE="$svc" \
        -t "clario360/${svc}:$(cat source/VERSION)" \
        source/
done

# Build frontend
docker build \
    -f deploy/docker/Dockerfile.frontend \
    -t "clario360/frontend:$(cat source/VERSION)" \
    source/frontend/

# Build migrator
docker build \
    -f deploy/docker/Dockerfile.migrator \
    -t "clario360/migrator:$(cat source/VERSION)" \
    source/
```

### Option 3: Build with Docker Compose

```bash
cp deploy/docker-compose.yml .
docker compose build
```

---

## Running Tests

### Unit Tests (No Infrastructure Required)

```bash
cd source/backend
GOWORK=off go test -mod=vendor -short ./...
```

### Integration Tests (Requires PostgreSQL, Redis, Kafka)

```bash
# Start test infrastructure
docker compose -f deploy/docker-compose.test.yml up -d

# Run integration tests
cd source/backend
GOWORK=off go test -mod=vendor -tags=integration ./integration_tests/...

# Tear down
docker compose -f deploy/docker-compose.test.yml down -v
```

### Frontend Tests

```bash
cd source/frontend
npm test          # Unit tests with Vitest
npm run test:ci   # CI mode (no watch)
```

---

## Database Setup

### Running Migrations

```bash
# Using the migrator binary
./bin/migrator \
    --migrations-path=deploy/migrations \
    --database-url="postgres://user:pass@localhost:5432/platform_core?sslmode=disable"

# Or run SQL files directly
for db in platform_core cyber_db data_db acta_db audit_db notification_db lex_db visus_db; do
    createdb "$db"
    for migration in deploy/migrations/${db}/*up.sql; do
        psql -d "$db" -f "$migration"
    done
done
```

### Database Schema Reference

Each database has ordered migration files:
- `000001_*.up.sql` — Creates initial schema
- `000002_*.up.sql` — Adds indexes, constraints
- `000003_*.up.sql` — Subsequent schema changes
- Corresponding `*.down.sql` files for rollback

---

## Deployment

### Local Development

```bash
docker compose -f deploy/docker-compose.yml up -d
```

### Kubernetes (Helm)

```bash
# Install/upgrade
helm upgrade --install clario360 deploy/helm/clario360/ \
    -f deploy/helm/clario360/values-production.yaml \
    --namespace clario360 \
    --create-namespace

# Verify
kubectl -n clario360 get pods
```

### Infrastructure (Terraform)

```bash
cd deploy/terraform/environments/production
terraform init
terraform plan
terraform apply
```

---

## Verification

After building, run the verification suite to confirm everything is correct:

```bash
# Verify package integrity (checksums, file presence)
./verification/verify-integrity.sh

# Verify build capability (compile, test)
./verification/verify-build.sh
```

Both scripts exit with code 0 on success.

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `go build` fails with "no required module provides..." | Ensure `-mod=vendor` flag is used |
| `GOWORK` errors | Set `GOWORK=off` before all go commands |
| `npm ci` fails offline | Copy `dependencies/node_modules` to `source/frontend/` |
| Docker build fails on base image | Load base images from `dependencies/images/` first |
| Migration fails | Check database exists and connection string is correct |
| Helm install fails | Verify Kubernetes cluster is accessible and namespace exists |

## Architecture Reference

For understanding the system architecture, service interactions, and design
decisions, refer to `docs/architecture/` — particularly:

- `11_ARCHITECTURE_OVERVIEW.md` — System architecture and component diagram
- `15_AI_GOVERNANCE_AND_MLOPS.md` — AI model management
- API endpoints are documented in `docs/api/openapi.yaml`
