# Clario360 Audit Findings

Date: 2026-03-08

## Scope

This audit covered the canonical frontend contracts, shared frontend API clients, direct frontend call sites for workflows/settings/notifications/auth, gateway route registration, shared backend response writers, middleware error paths, startup defaults, and smoke-test automation.

Primary source files reviewed during the audit included:

- `frontend/src/types/api.ts`
- `frontend/src/types/models.ts`
- `frontend/src/types/table.ts`
- `frontend/src/lib/api.ts`
- `frontend/src/lib/constants.ts`
- `frontend/src/lib/suite-api.ts`
- `frontend/src/lib/data-suite/api.ts`
- `frontend/src/lib/enterprise/api.ts`
- `frontend/src/lib/notebooks.ts`
- `frontend/src/hooks/use-data-table.ts`
- `frontend/src/hooks/use-realtime-data.ts`
- `frontend/src/hooks/use-websocket.ts`
- `frontend/src/stores/auth-store.ts`
- `frontend/src/app/(dashboard)/settings/page.tsx`
- `frontend/src/app/(dashboard)/workflows/workflows-page-client.tsx`
- `frontend/src/app/(dashboard)/workflows/[id]/workflow-instance-page-client.tsx`
- `frontend/src/app/(dashboard)/workflows/tasks/tasks-page-client.tsx`
- `frontend/src/components/workflows/task-context-panel.tsx`
- `backend/cmd/api-gateway/main.go`
- `backend/internal/gateway/config/routes.go`
- `backend/internal/gateway/config/config.go`
- `backend/internal/suiteapi/http.go`
- `backend/internal/iam/handler/user_handler.go`
- `backend/internal/iam/service/user_service.go`
- `backend/internal/security/*`
- `scripts/start.sh`
- `ecosystem.local.js`

## Gateway Route Map

Verified gateway prefix ownership from `backend/internal/gateway/config/routes.go`:

| Prefix | Backend service |
| --- | --- |
| `/api/v1/auth` | `iam-service` |
| `/api/v1/users` | `iam-service` |
| `/api/v1/roles` | `iam-service` |
| `/api/v1/tenants` | `iam-service` |
| `/api/v1/api-keys` | `iam-service` |
| `/api/v1/onboarding` | `iam-service` |
| `/api/v1/invitations` | `iam-service` |
| `/api/v1/notebooks` | `iam-service` |
| `/api/v1/ai` | `iam-service` |
| `/api/v1/workflows` | `workflow-engine` |
| `/api/v1/audit` | `audit-service` |
| `/api/v1/cyber` | `cyber-service` |
| `/api/v1/data` | `data-service` |
| `/api/v1/acta` | `acta-service` |
| `/api/v1/lex` | `lex-service` |
| `/api/v1/visus` | `visus-service` |
| `/api/v1/notifications` | `notification-service` |
| `/api/v1/files` | `file-service` |
| `/ws/v1/notifications` | `notification-service` |
| `/ws/v1/cyber` | `cyber-service` |
| `/ws/v1/visus` | `visus-service` |

## Issue Ledger

| ID | Issue summary | Root cause | Services/pages affected | Status |
| --- | --- | --- | --- | --- |
| 1 | Paginated list contract drift between canonical frontend `PaginatedResponse` and emitted/backend-adapted shapes | Shared helpers and client adapters still used `pagination`, `items`, or other non-canonical field names instead of `data` + `meta` | Suite services, data suite adapters, lex search, shared pagination helpers | Fixed |
| 2 | Error responses still returned nested `{ "error": { ... } }` from several middleware paths | Gateway/security/audit/cyber/notification/file middleware were not using the same contract as `frontend/src/lib/api.ts` | Gateway error paths, security middleware, audit/cyber/notification/file upload guards | Fixed in source |
| 3 | Frontend suite/data adapters still decoded `pagination` after backend normalization to `meta` | Frontend API wrappers had retained legacy envelope types | `frontend/src/lib/suite-api.ts`, `frontend/src/lib/data-suite/api.ts`, `frontend/src/lib/data-suite/types.ts`, `frontend/src/lib/enterprise/api.ts` | Fixed |
| 4 | Settings session management was wired to a dead/non-canonical session surface | Frontend used stale endpoints and stale field assumptions | Settings page/session UI, IAM user/session handlers | Fixed |
| 5 | Gateway route coverage was incomplete for frontend-used IAM prefixes | Route registry was missing prefixes used by onboarding/invitations/AI governance pages | Gateway, onboarding/invitations/AI governance frontend flows | Fixed in source |
| 6 | Service port/default URL drift created broken startup and proxy assumptions | Gateway defaults, service defaults, and local startup scripts did not all agree on canonical ports | Gateway, workflow, cyber, data, acta, lex, visus, notification, file, local startup flows | Fixed in source where verified; local PM2 profile remains a residual risk |
| 7 | Smoke verification initially failed against a stale live gateway even after source fixes | The already-running `8080` gateway process had not been restarted, so runtime responses did not match the fixed source tree | Live local runtime only | Source verified; runtime restart still required |

## Fixes Applied

### Contract normalization

- Standardized shared backend paginated responses on `data` + `meta.{page,per_page,total,total_pages}`.
- Standardized middleware and gateway error responses on top-level `code`, `message`, optional `details`, optional `request_id`.
- Removed the last frontend-side `pagination` adapters from the shared suite/data clients.
- Normalized the shared internal pagination helper to use `per_page` and `meta` semantics.

### Frontend/backend integration hardening

- Verified that settings now use `/api/v1/users/me/sessions` and the corresponding IAM handlers exist.
- Verified that notification websocket routing exists at `/ws/v1/notifications`.
- Added an executable `scripts/smoke-test.sh` that checks:
  - service health endpoints
  - gateway unauthorized/not-found error contract
  - gateway authentication when credentials are supplied
  - representative frontend-used paginated endpoints across routed services
  - notification websocket upgrade
- Corrected the smoke-test IAM health target to `9081/healthz`, which matches the actual running service layout.

### Startup and routing consistency

- Verified gateway prefix coverage for onboarding, invitations, AI governance, notifications, files, workflows, and suite routes.
- Verified that IAM health is exposed on the admin port (`9081`) rather than the main port (`8081`).
- Cross-checked the startup script’s canonical port map against service config defaults and gateway downstream URLs.

## Startup / Config Cross-Check

Verified/defaulted service ports and notable env expectations:

| Service | HTTP port source | Default | Admin/health | Key env/config notes |
| --- | --- | --- | --- | --- |
| `api-gateway` | `GW_HTTP_PORT` | `8080` | `GW_ADMIN_PORT=9080` | Uses legacy auth config (`AUTH_RSA_PRIVATE_KEY_PEM`, `AUTH_RSA_PUBLIC_KEY_PEM`, issuer/TTL vars) |
| `iam-service` | hardcoded in `main.go` | `8081` | hardcoded admin `9081` | Health is on `9081/healthz`; auth/db primarily come from legacy base config |
| `workflow-engine` | `WF_HTTP_PORT` | `8083` | admin bootstrap defaults | Service task timeout uses `WF_SERVICE_TASK_TIMEOUT_SEC` |
| `audit-service` | `AUDIT_HTTP_PORT` | `8084` | bootstrap admin port `9084` | Pool tuning uses `AUDIT_DB_MIN_CONNS` / `AUDIT_DB_MAX_CONNS` |
| `cyber-service` | `CYBER_HTTP_PORT` | `8085` | service admin defaults | Requires `CYBER_DB_URL` and `CYBER_JWT_PUBLIC_KEY_PATH` |
| `data-service` | `DATA_HTTP_PORT` | `8086` | service/admin defaults | Requires `DATA_DB_URL` and `DATA_JWT_PUBLIC_KEY_PATH`; connector timeouts use Go duration syntax |
| `acta-service` | `ACTA_HTTP_PORT` | `8087` | `ACTA_ADMIN_PORT=9087` | `ACTA_DB_URL`, `ACTA_JWT_PUBLIC_KEY_PATH`, `ACTA_DASHBOARD_CACHE_TTL` |
| `lex-service` | `LEX_HTTP_PORT` | `8088` | `LEX_ADMIN_PORT=9088` | `LEX_DB_URL`, `LEX_JWT_PUBLIC_KEY_PATH`, `LEX_DASHBOARD_CACHE_TTL` |
| `visus-service` | `VISUS_HTTP_PORT` | `8089` | `VISUS_ADMIN_PORT=9089` | `VISUS_DB_URL`, JWT pub/priv key paths, cache/service-token durations |
| `notification-service` | `NOTIF_HTTP_PORT` | `8090` | bootstrap admin port `9090` | WS/webhook tuning uses integer second env vars |
| `file-service` | `FILE_HTTP_PORT` | `8091` | service/admin defaults | Requires `FILE_DB_URL`, `FILE_JWT_PUBLIC_KEY_PATH`, `FILE_MINIO_SECRET_KEY` |

`scripts/start.sh` uses the canonical service ports above. The remaining configuration gap is `ecosystem.local.js`, which is still a partial cyber-only PM2 profile and does not represent the full gateway-centered platform layout.

## Frontend API Inventory Appendix

This appendix records the frontend API surface that was audited, grouped by the actual frontend file that defines the call pattern.

| Frontend file | Method / URL patterns | Expected response contract | Responsible backend |
| --- | --- | --- | --- |
| `frontend/src/stores/auth-store.ts` | `POST /api/v1/auth/login`, `POST /api/v1/auth/verify-mfa`, `POST /api/v1/auth/logout`, `GET/PATCH /api/v1/users/me`, BFF `GET/POST/DELETE /api/auth/session` | auth endpoints return token payloads / MFA payloads; `users/me` returns raw `User` | `iam-service` via gateway; BFF handled by Next routes |
| `frontend/src/lib/notebooks.ts` | `GET /api/v1/notebooks/profiles`, `GET /api/v1/notebooks/templates`, `GET/POST/DELETE /api/v1/notebooks/servers`, `GET /api/v1/notebooks/servers/{id}/status`, `POST /api/v1/notebooks/servers/{id}/copy-template` | raw arrays/objects | `iam-service` via gateway |
| `frontend/src/hooks/use-notification-actions.ts` | `PUT /api/v1/notifications/{id}/read`, `PUT /api/v1/notifications/read-all`, `DELETE /api/v1/notifications/{id}` | 2xx empty/body ignored; failures must follow top-level error shape | `notification-service` |
| `frontend/src/hooks/use-websocket.ts` | `GET /ws/v1/notifications?token=...` upgrade | websocket upgrade, graceful reconnect/failure handling | `notification-service` via gateway |
| `frontend/src/lib/suite-api.ts` | Generic paginated GETs for suite endpoints | canonical `PaginatedResponse<T>` backed by `{ data: T[], meta: PaginationMeta }` | `acta-service`, `lex-service`, `visus-service`, `iam-service` (`/api/v1/ai`) |
| `frontend/src/lib/data-suite/api.ts` | `GET/POST/PUT/DELETE /api/v1/data/*`; `POST /api/v1/files/upload` | single-resource endpoints use `{ data: T }`; list endpoints use `{ data: T[], meta }` | `data-service`; `file-service` for upload |
| `frontend/src/lib/enterprise/api.ts` | `GET/POST/PUT/DELETE /api/v1/users`, `/api/v1/acta/*`, `/api/v1/lex/*`, `/api/v1/visus/*`, `/api/v1/ai/*`, `/api/v1/files/upload` | canonical paginated lists for list endpoints; suite single-resource endpoints return `{ data: T }` and are unwrapped client-side | `iam-service`, `acta-service`, `lex-service`, `visus-service`, `iam-service` (`/api/v1/ai`), `file-service` |
| `frontend/src/app/(dashboard)/workflows/workflows-page-client.tsx` | `GET /api/v1/workflows/instances`, `POST /api/v1/workflows/instances/{id}/retry` | list expects `PaginatedResponse<WorkflowInstance>` | `workflow-engine` |
| `frontend/src/app/(dashboard)/workflows/[id]/workflow-instance-page-client.tsx` | `GET /api/v1/workflows/instances/{id}`, `GET /api/v1/workflows/instances/{id}/history`, `POST /retry|suspend|resume` | raw object; history expects `{ steps: StepExecution[] }` | `workflow-engine` |
| `frontend/src/app/(dashboard)/workflows/tasks/tasks-page-client.tsx` | `GET /api/v1/workflows/tasks`, `POST /api/v1/workflows/tasks/{id}/claim` | list expects `PaginatedResponse<HumanTask>`; count polling uses `/api/v1/workflows/tasks/count` | `workflow-engine` |
| `frontend/src/components/workflows/task-context-panel.tsx` | `GET /api/v1/workflows/instances/{id}`, `/history`, plus related-entity lookups to `/api/v1/cyber/alerts/{id}`, `/api/v1/lex/contracts/{id}`, `/api/v1/acta/meetings/{id}` | raw objects / `{ steps: [] }` | `workflow-engine`, `cyber-service`, `lex-service`, `acta-service` |
| `frontend/src/app/(dashboard)/settings/page.tsx` and `frontend/src/app/(dashboard)/settings/_components/*` | `GET/DELETE /api/v1/users/me/sessions`, `GET/POST/DELETE /api/v1/api-keys`, `POST /api/v1/users/me/password`, `PUT /api/v1/users/me`, MFA setup/disable/verify endpoints | sessions and API keys are raw arrays; profile/password/MFA endpoints use raw objects or simple success payloads | `iam-service` |
| `frontend/src/app/(dashboard)/cyber/vciso/_components/chat-panel.tsx` | `GET /ws/v1/cyber/vciso/chat?token=...` upgrade | websocket upgrade with token auth | `cyber-service` via gateway |

The inventory above covers the direct frontend endpoint definitions and the shared API wrappers that fan out into the dashboard pages. Dynamic entity-specific URLs were audited through those wrapper definitions and direct call sites.

## Files Changed

Key files changed during this audit:

- `backend/internal/suiteapi/http.go`
- `backend/internal/gateway/config/routes.go`
- `backend/internal/gateway/config/config.go`
- `backend/internal/gateway/middleware/proxy_auth.go`
- `backend/internal/gateway/proxy/reverse_proxy.go`
- `backend/internal/gateway/proxy/websocket_proxy.go`
- `backend/internal/iam/dto/user_dto.go`
- `backend/internal/iam/handler/user_handler.go`
- `backend/internal/iam/service/user_service.go`
- `backend/internal/audit/middleware/rate_limiter.go`
- `backend/internal/audit/middleware/tenant_guard.go`
- `backend/internal/cyber/handler/asset_handler.go`
- `backend/internal/cyber/middleware/rate_limiter.go`
- `backend/internal/filemanager/middleware/upload_guard.go`
- `backend/internal/notification/middleware/rate_limiter.go`
- `backend/internal/security/csrf.go`
- `backend/internal/security/rate_limit_api.go`
- `backend/internal/security/rate_limit_auth.go`
- `backend/internal/security/sanitizer_middleware.go`
- `backend/internal/security/session_security.go`
- `backend/internal/types/pagination.go`
- `frontend/src/lib/suite-api.ts`
- `frontend/src/lib/data-suite/api.ts`
- `frontend/src/lib/data-suite/types.ts`
- `frontend/src/lib/enterprise/api.ts`
- `frontend/src/app/(dashboard)/settings/page.tsx`
- `frontend/src/__tests__/data-suite-fixtures.ts`
- `scripts/smoke-test.sh`

## Verification Performed

Verified successfully:

- `cd /Users/mac/clario360/backend && GOWORK=off go build ./...`
- `cd /Users/mac/clario360/frontend && npm run build`
- `cd /Users/mac/clario360/frontend && npm run type-check` after `.next/types` had been regenerated by the build
- `bash -n /Users/mac/clario360/scripts/smoke-test.sh`
- `bash /Users/mac/clario360/scripts/smoke-test.sh`
  - passed health checks for `gateway`, `iam`, `workflow`, `audit`, `cyber`, `notification`
  - failed health checks for `data`, `acta`, `lex`, `visus`, `file` because those services were not running in the current local stack
  - correctly skipped authenticated endpoint coverage because smoke credentials were not configured
- Temporary source-based gateway validation on `http://localhost:18080`
  - `GET /api/v1/definitely-missing` returned top-level `{code,message,request_id}`
  - `GET /api/v1/users/me` without auth returned top-level `{code,message,request_id}`

## Residual Risks / Follow-Up

- The live gateway already running on `http://localhost:8080` during smoke verification was stale and still returned legacy nested error payloads. Source was verified separately on a fresh gateway process; the running stack still needs a restart to pick up the fix.
- `data-service`, `acta-service`, `lex-service`, `visus-service`, and `file-service` were not running locally during the smoke sweep, so authenticated end-to-end validation across the full platform could not complete in this session.
- `CLARIO360_SMOKE_EMAIL` and `CLARIO360_SMOKE_PASSWORD` were not set, so the authenticated portion of the smoke script was intentionally skipped.
- `ecosystem.local.js` remains a partial cyber-only PM2 profile and still does not represent the full gateway-centered local topology. It should be either expanded to include the gateway and canonical ports or treated as a specialized one-off profile.
- Next.js production builds continue to emit a standalone trace warning while still exiting successfully:
  - missing copy source for `.next/server/app/(dashboard)/page_client-reference-manifest.js`
  - this did not block the build, but it should be investigated before relying on standalone output packaging.
