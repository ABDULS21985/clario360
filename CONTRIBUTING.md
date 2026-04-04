# Contributing to Clario 360

Thank you for your interest in contributing to Clario 360. This document outlines the development workflow, coding standards, and review process.

## Getting Started

1. Follow the [Quick Start](README.md#quick-start-local-development) guide to set up your local environment
2. Ensure all tests pass before making changes: `make test`
3. Familiarize yourself with the [project structure](README.md#project-structure)

## Branch Naming

Use the following conventions:

| Prefix | Purpose | Example |
|--------|---------|---------|
| `feat/` | New feature | `feat/contract-analytics` |
| `fix/` | Bug fix | `fix/jwt-refresh-race` |
| `refactor/` | Code refactoring | `refactor/audit-service-queries` |
| `docs/` | Documentation | `docs/api-reference-update` |
| `test/` | Test additions or fixes | `test/cyber-integration` |
| `chore/` | Maintenance tasks | `chore/upgrade-go-1.23` |
| `security/` | Security fix | `security/csp-header-fix` |

Always branch from `main`:

```bash
git checkout main
git pull origin main
git checkout -b feat/your-feature-name
```

## Development Workflow

### 1. Write Code

- Follow existing patterns in the codebase
- Keep changes focused — one feature or fix per PR
- Add or update tests for any logic changes

### 2. Run Checks Locally

```bash
# Backend
make fmt          # Format Go code
make lint         # Run linter
make test         # Run unit tests

# Frontend
make frontend-lint   # Lint TypeScript/React
make frontend-test   # Run Vitest tests

# Full check
make test-all     # Run everything
```

### 3. Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:** `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `ci`, `perf`, `security`

**Scopes:** `iam`, `audit`, `cyber`, `data`, `acta`, `lex`, `visus`, `gateway`, `workflow`, `notification`, `file`, `frontend`, `infra`, `ci`

**Examples:**

```
feat(cyber): add MITRE ATT&CK mapping to threat detection
fix(gateway): prevent race condition in circuit breaker reset
docs(api): update OpenAPI spec for contract endpoints
test(lex): add integration tests for clause extraction
```

### 4. Submit a Pull Request

```bash
git push origin feat/your-feature-name
```

Then open a PR targeting `main` with:

- **Title:** Clear, concise summary (under 70 characters)
- **Description:** What changed, why, and how to test it
- **Labels:** Add relevant labels (e.g., `suite:cyber`, `priority:high`)

## Pull Request Requirements

### Before Review

- [ ] All CI checks pass (lint, test, build, security scan)
- [ ] No decrease in test coverage
- [ ] New endpoints have OpenAPI documentation
- [ ] Database changes include both up and down migrations
- [ ] No secrets, credentials, or PII in the code
- [ ] Frontend changes tested in Chrome, Firefox, and Safari

### Code Review

- **Minimum 1 approval** required for merge
- **2 approvals** required for:
  - Security-related changes
  - Database schema changes
  - Authentication/authorization changes
  - Infrastructure (Terraform, Helm) changes
- Reviewers should check for correctness, security, performance, and readability

### After Approval

- Squash merge into `main`
- Delete the feature branch after merge
- CI/CD automatically deploys to staging

## Coding Standards

### Go (Backend)

- Follow [Effective Go](https://go.dev/doc/effective_go) and standard library conventions
- Use `chi` router for HTTP handlers (not `gin`, `mux`, etc.)
- Use `slog` for structured logging via the observability package
- Use `prometheus.NewRegistry()` with `promauto.With(reg)` — never the default registry
- Middleware signature: `func(http.Handler) http.Handler`
- Extract user/tenant from context: `auth.UserFromContext(ctx)`, `auth.TenantFromContext(ctx)`
- All Go commands require `GOWORK=off` prefix when running from the `backend/` directory
- Handle errors explicitly — no `_` for error returns
- Table-driven tests preferred

### TypeScript (Frontend)

- Strict TypeScript — no `any` unless absolutely necessary
- Use Zustand for state management
- Use `react-hook-form` + Zod for form validation
- Use `shadcn/ui` components as the base UI library
- API calls through `lib/api.ts` (`apiGet`, `apiPost`, etc.)
- Auth tokens in memory only — refresh tokens in httpOnly cookies
- Test with Vitest + MSW for API mocking

### Database

- All tables must have `tenant_id` for Row-Level Security
- Include `created_at`, `updated_at` timestamps
- Use UUIDs for primary keys
- Write both up and down migrations
- Name migrations descriptively: `NNNN_create_contracts_table.up.sql`

### API Design

- RESTful resource naming (plural nouns)
- Consistent pagination: `?page=1&page_size=20`
- Standard error response format with error codes
- All endpoints documented in OpenAPI 3.1

## Testing Guidelines

### Backend

- **Unit tests** in `*_test.go` files alongside source
- **Integration tests** use `// +build integration` tag
- **Table-driven tests** for functions with multiple input/output combinations
- Mock external dependencies (database, Kafka, Redis) in unit tests
- Use `prometheus.NewRegistry()` per test to avoid registration panics

### Frontend

- **Component tests** with Vitest + React Testing Library
- **API mocking** with MSW (Mock Service Worker)
- Use `getByPlaceholderText` or `getByRole` (not `getByLabelText` due to FormField wrapper)
- Test user interactions, not implementation details

### Test Coverage

- Aim for **80%+ coverage** on business logic
- All API handlers must have tests for success and error paths
- All database queries must have integration tests

## Documentation

- Update OpenAPI spec when adding/modifying endpoints
- Add JSDoc comments for exported TypeScript functions
- Document non-obvious business logic in code comments
- Update runbooks for operational changes
- Architecture decisions go in `docs/architecture/`

## Questions?

- Open a GitHub Discussion for general questions
- Tag `@platform-team` for architecture decisions
- Reach out to security@clario360.sa for security concerns
