.PHONY: build test test-cover test-short test-integration test-security test-all e2e-test \
       lint lint-fix fmt migrate-up migrate-down migrate-create migrate-status \
       seed docker-up docker-down docker-clean docker-wait docker-build \
       docker-test-up docker-test-down \
       run-all run frontend-install frontend-dev frontend-build frontend-lint frontend-test \
       generate-sdk generate-mocks validate-api proto-gen \
       loadtest helm-lint helm-template clean help

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------
GO          := go
GOFLAGS     := -race
BINARY_DIR  := backend/bin
SERVICES    := api-gateway iam-service event-bus workflow-engine audit-service \
               notification-service file-service \
               cyber-service data-service acta-service lex-service visus-service
TOOLS       := migrator data-seeder system-seeder
ALL_TARGETS := $(SERVICES) $(TOOLS)
MIGRATE     := $(GO) run -C backend ./cmd/migrator
DC          := docker compose
DC_TEST     := docker compose -f docker-compose.test.yml
HELM_CHART  := deploy/helm/clario360
SEED_SCALE  ?= large

# Service port mapping (used by run target)
PORT_api-gateway          := 8080
PORT_iam-service          := 8081
PORT_workflow-engine      := 8083
PORT_audit-service        := 8084
PORT_cyber-service        := 8085
PORT_data-service         := 8086
PORT_acta-service         := 8087
PORT_lex-service          := 8088
PORT_visus-service        := 8089
PORT_notification-service := 8090
PORT_file-service         := 8091

# ---------------------------------------------------------------------------
# Default
# ---------------------------------------------------------------------------
help: ## Show this help
	@echo "Clario 360 — Build & Development Targets"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build: ## Build all backend services
	@echo "==> Building all services..."
	@mkdir -p $(BINARY_DIR)
	@for svc in $(ALL_TARGETS); do \
		echo "  -> $$svc"; \
		$(GO) build -C backend -o bin/$$svc ./cmd/$$svc; \
	done
	@echo "==> Done."

build-%: ## Build a specific service (e.g., make build-api-gateway)
	@mkdir -p $(BINARY_DIR)
	$(GO) build -C backend -o bin/$* ./cmd/$*

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------
run-all: ## Start all backend services in parallel
	@echo "==> Starting all services..."
	@for svc in $(SERVICES); do \
		echo "  -> Starting $$svc"; \
		$(GO) run -C backend ./cmd/$$svc & \
	done; \
	echo "==> All services started. Press Ctrl+C to stop."; \
	wait

run: ## Run a specific service (e.g., make run SERVICE=iam-service)
	@if [ -z "$(SERVICE)" ]; then \
		echo "Usage: make run SERVICE=<service-name>"; \
		echo "Available services: $(SERVICES)"; \
		exit 1; \
	fi
	@echo "==> Starting $(SERVICE)..."
	$(GO) run -C backend ./cmd/$(SERVICE)

dev: docker-up ## Start all dependencies and run the API gateway
	@echo "==> Starting API gateway..."
	$(GO) run -C backend ./cmd/api-gateway

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------
test: ## Run all backend unit tests with race detector
	GOWORK=off $(GO) test $(GOFLAGS) -C backend ./...

test-cover: ## Run tests with coverage report
	GOWORK=off $(GO) test $(GOFLAGS) -C backend -coverprofile=backend/coverage.out ./...
	$(GO) tool cover -C backend -html=coverage.out -o coverage.html
	@echo "Coverage report: backend/coverage.html"

test-short: ## Run short tests only (skip integration)
	GOWORK=off $(GO) test -C backend -short ./...

test-integration: docker-test-up ## Run integration tests (requires Docker)
	@echo "==> Waiting for test dependencies..."
	@sleep 5
	GOWORK=off $(GO) test $(GOFLAGS) -C backend -tags=integration ./...
	@$(MAKE) docker-test-down

e2e-test: ## Run end-to-end tests (requires full Docker environment)
	@echo "==> Running end-to-end tests..."
	GOWORK=off $(GO) test $(GOFLAGS) -C backend -tags=e2e ./e2e_tests/...

test-security: ## Run security-focused tests and static analysis
	@echo "==> Running security tests..."
	@if command -v gosec >/dev/null 2>&1; then \
		cd backend && gosec -quiet ./...; \
	else \
		echo "  [SKIP] gosec not installed — run: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
	@if command -v trivy >/dev/null 2>&1; then \
		trivy fs --security-checks vuln backend/; \
	else \
		echo "  [SKIP] trivy not installed — see https://aquasecurity.github.io/trivy/"; \
	fi
	@echo "==> Running npm audit..."
	@cd frontend && npm audit --audit-level=critical 2>/dev/null || true
	@echo "==> Security tests complete."

test-all: test test-security frontend-test ## Run all tests (unit + security + frontend)
	@echo "==> All tests complete."

loadtest: ## Run load tests (usage: make loadtest SCENARIO=smoke)
	@if [ -z "$(SCENARIO)" ]; then \
		echo "Usage: make loadtest SCENARIO=<smoke|load|stress|soak>"; \
		exit 1; \
	fi
	@if command -v k6 >/dev/null 2>&1; then \
		k6 run deploy/loadtest/$(SCENARIO).js; \
	else \
		echo "k6 not installed — see https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi

frontend-test: ## Run frontend tests
	cd frontend && npm test -- --run 2>/dev/null || cd frontend && npx vitest run

# ---------------------------------------------------------------------------
# Lint & Format
# ---------------------------------------------------------------------------
lint: ## Run golangci-lint on backend
	cd backend && golangci-lint run ./...

lint-fix: ## Run golangci-lint with auto-fix
	cd backend && golangci-lint run --fix ./...

fmt: ## Format Go source code
	@echo "==> Formatting Go code..."
	cd backend && $(GO) fmt ./...
	@echo "==> Done."

frontend-lint: ## Lint frontend code
	cd frontend && npm run lint

# ---------------------------------------------------------------------------
# Migrations
# ---------------------------------------------------------------------------
migrate-up: ## Run all database migrations
	$(MIGRATE) -direction up

migrate-down: ## Rollback last migration
	$(MIGRATE) -direction down

migrate-create: ## Create a new migration (usage: make migrate-create NAME=create_users)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	migrate create -ext sql -dir backend/migrations -seq $(NAME)

migrate-status: ## Show migration status for all databases
	@echo "==> Migration status"
	@for db_dir in backend/migrations/*/; do \
		db=$$(basename $$db_dir); \
		count=$$(ls -1 $$db_dir/*.sql 2>/dev/null | wc -l | tr -d ' '); \
		echo "  $$db: $$count migration files"; \
	done

# ---------------------------------------------------------------------------
# Seed
# ---------------------------------------------------------------------------
seed: ## Seed the database with development data (override with SEED_SCALE=small|large|massive)
	$(GO) run -C backend ./cmd/system-seeder -scale $(SEED_SCALE)

# ---------------------------------------------------------------------------
# Code Generation
# ---------------------------------------------------------------------------
generate-sdk: ## Generate TypeScript types from OpenAPI spec
	@echo "==> Generating TypeScript SDK from OpenAPI spec..."
	@if [ -f docs/api/openapi.yaml ]; then \
		npx openapi-typescript docs/api/openapi.yaml -o frontend/src/types/api-generated.ts; \
		echo "==> Generated: frontend/src/types/api-generated.ts"; \
	else \
		echo "  [SKIP] docs/api/openapi.yaml not found"; \
	fi

generate-mocks: ## Generate Go mocks for testing
	@echo "==> Generating Go mocks..."
	@if command -v mockgen >/dev/null 2>&1; then \
		cd backend && go generate ./...; \
	else \
		echo "  [SKIP] mockgen not installed — run: go install go.uber.org/mock/mockgen@latest"; \
	fi

validate-api: ## Validate OpenAPI specification
	@echo "==> Validating OpenAPI spec..."
	@if [ -f docs/api/openapi.yaml ]; then \
		npx @redocly/cli lint docs/api/openapi.yaml; \
	else \
		echo "  [SKIP] docs/api/openapi.yaml not found"; \
	fi

proto-gen: ## Generate protobuf Go code from .proto files
	@echo "==> Generating protobuf code..."
	@find backend/proto -name '*.proto' -print0 2>/dev/null | xargs -0 -r protoc \
		--go_out=backend --go_opt=paths=source_relative \
		--go-grpc_out=backend --go-grpc_opt=paths=source_relative \
		-I backend/proto
	@echo "==> Done."

# ---------------------------------------------------------------------------
# Docker — Local Development
# ---------------------------------------------------------------------------
docker-up: ## Start all local dependencies (PostgreSQL, Redis, Kafka, MinIO)
	$(DC) up -d
	@echo "==> Waiting for services to be healthy..."
	@$(DC) ps

docker-down: ## Stop all local dependencies
	$(DC) down

docker-clean: ## Stop and remove all volumes (WARNING: destroys data)
	$(DC) down -v

docker-wait: ## Wait for all Docker services to be healthy
	@echo "==> Waiting for services to be healthy..."
	@timeout=60; elapsed=0; \
	while [ $$elapsed -lt $$timeout ]; do \
		healthy=$$($(DC) ps --format json 2>/dev/null | grep -c '"healthy"' || echo 0); \
		total=$$($(DC) ps -q 2>/dev/null | wc -l | tr -d ' '); \
		echo "  $$healthy/$$total services healthy ($$elapsed""s)"; \
		if [ "$$healthy" -ge "$$total" ] && [ "$$total" -gt 0 ]; then \
			echo "==> All services healthy!"; \
			exit 0; \
		fi; \
		sleep 5; \
		elapsed=$$((elapsed + 5)); \
	done; \
	echo "==> WARNING: Timed out waiting for healthy services."; \
	$(DC) ps; \
	exit 1

docker-test-up: ## Start test dependencies
	$(DC_TEST) up -d
	@echo "==> Test dependencies starting..."

docker-test-down: ## Stop test dependencies
	$(DC_TEST) down -v

# ---------------------------------------------------------------------------
# Docker — Build Images
# ---------------------------------------------------------------------------
docker-build: ## Build Docker images for all services
	@echo "==> Building Docker images..."
	@for svc in $(SERVICES); do \
		echo "  -> clario360/$$svc"; \
		docker build -f deploy/docker/Dockerfile.backend \
			--build-arg SERVICE=$$svc \
			-t clario360/$$svc:latest \
			backend/; \
	done
	@echo "  -> clario360/frontend"
	@docker build -f deploy/docker/Dockerfile.frontend \
		-t clario360/frontend:latest \
		frontend/
	@echo "==> Done."

docker-build-%: ## Build Docker image for a specific service
	docker build -f deploy/docker/Dockerfile.backend \
		--build-arg SERVICE=$* \
		-t clario360/$*:latest \
		backend/

# ---------------------------------------------------------------------------
# Frontend
# ---------------------------------------------------------------------------
frontend-install: ## Install frontend dependencies
	cd frontend && npm install

frontend-dev: ## Start frontend dev server
	cd frontend && npm run dev

frontend-build: ## Build frontend for production
	cd frontend && npm run build

# ---------------------------------------------------------------------------
# Helm
# ---------------------------------------------------------------------------
helm-lint: ## Lint Helm chart
	@if [ -d "$(HELM_CHART)" ]; then \
		helm lint $(HELM_CHART); \
	else \
		echo "  [SKIP] Helm chart not found at $(HELM_CHART)"; \
	fi

helm-template: ## Render Helm templates locally
	@if [ -d "$(HELM_CHART)" ]; then \
		helm template clario360 $(HELM_CHART); \
	else \
		echo "  [SKIP] Helm chart not found at $(HELM_CHART)"; \
	fi

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------
clean: ## Remove build artifacts
	rm -rf $(BINARY_DIR)
	rm -f backend/coverage.out backend/coverage.html
