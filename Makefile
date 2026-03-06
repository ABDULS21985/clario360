.PHONY: build test lint migrate-up migrate-down proto-gen docker-up docker-down seed clean help

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------
GO          := go
GOFLAGS     := -race
BINARY_DIR  := backend/bin
SERVICES    := api-gateway iam-service event-bus workflow-engine audit-service \
               cyber-service data-service acta-service lex-service visus-service migrator
MIGRATE     := $(GO) run -C backend ./cmd/migrator
DC          := docker compose
DC_TEST     := docker compose -f docker-compose.test.yml

# ---------------------------------------------------------------------------
# Default
# ---------------------------------------------------------------------------
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build: ## Build all backend services
	@echo "==> Building all services..."
	@mkdir -p $(BINARY_DIR)
	@for svc in $(SERVICES); do \
		echo "  -> $$svc"; \
		$(GO) build -C backend -o bin/$$svc ./cmd/$$svc; \
	done
	@echo "==> Done."

build-%: ## Build a specific service (e.g., make build-api-gateway)
	@mkdir -p $(BINARY_DIR)
	$(GO) build -C backend -o bin/$* ./cmd/$*

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------
test: ## Run all backend tests with race detector
	$(GO) test $(GOFLAGS) -C backend ./...

test-cover: ## Run tests with coverage report
	$(GO) test $(GOFLAGS) -C backend -coverprofile=backend/coverage.out ./...
	$(GO) tool cover -C backend -html=coverage.out -o coverage.html
	@echo "Coverage report: backend/coverage.html"

test-short: ## Run short tests only (skip integration)
	$(GO) test -C backend -short ./...

# ---------------------------------------------------------------------------
# Lint
# ---------------------------------------------------------------------------
lint: ## Run golangci-lint
	cd backend && golangci-lint run ./...

lint-fix: ## Run golangci-lint with auto-fix
	cd backend && golangci-lint run --fix ./...

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

# ---------------------------------------------------------------------------
# Seed
# ---------------------------------------------------------------------------
seed: ## Seed the database with development data
	$(GO) run -C backend ./cmd/migrator -seed

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------
docker-up: ## Start all local dependencies (PostgreSQL, Redis, Kafka, MinIO, Keycloak)
	$(DC) up -d
	@echo "==> Waiting for services to be healthy..."
	@$(DC) ps

docker-down: ## Stop all local dependencies
	$(DC) down

docker-clean: ## Stop and remove all volumes
	$(DC) down -v

docker-test-up: ## Start test dependencies
	$(DC_TEST) up -d
	@echo "==> Test dependencies starting..."

docker-test-down: ## Stop test dependencies
	$(DC_TEST) down -v

# ---------------------------------------------------------------------------
# Frontend
# ---------------------------------------------------------------------------
frontend-install: ## Install frontend dependencies
	cd frontend && npm install

frontend-dev: ## Start frontend dev server
	cd frontend && npm run dev

frontend-build: ## Build frontend for production
	cd frontend && npm run build

frontend-lint: ## Lint frontend code
	cd frontend && npm run lint

# ---------------------------------------------------------------------------
# All-in-one
# ---------------------------------------------------------------------------
dev: docker-up ## Start all dependencies and run the API gateway
	@echo "==> Starting API gateway..."
	$(GO) run -C backend ./cmd/api-gateway

clean: ## Remove build artifacts
	rm -rf $(BINARY_DIR)
	rm -f backend/coverage.out backend/coverage.html
