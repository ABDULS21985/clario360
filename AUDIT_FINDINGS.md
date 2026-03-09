# Clario360 Audit Findings

Date: 2026-03-09

## Scope

This audit covered the canonical frontend contracts, shared frontend API clients, direct frontend call sites for workflows/settings/notifications/auth, gateway route registration, shared backend response writers, websocket upgrade paths, middleware error paths, startup defaults, PM2 local topology, smoke-test automation, the external integration admin/runtime surface for Slack, Teams, Jira, ServiceNow, and generic webhooks, and the Prompt 59 closeout work for continuous DSPM, compliance tagging, and root-cause analysis.

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
| 2 | Runtime error handling drift between gateway payloads and frontend parsing | Gateway/runtime responses used nested `error` envelopes while the frontend client still only decoded flat `code`/`message` bodies | Gateway error paths, frontend shared Axios client | Fixed |
| 3 | Frontend suite/data adapters still decoded `pagination` after backend normalization to `meta` | Frontend API wrappers had retained legacy envelope types | `frontend/src/lib/suite-api.ts`, `frontend/src/lib/data-suite/api.ts`, `frontend/src/lib/data-suite/types.ts`, `frontend/src/lib/enterprise/api.ts` | Fixed |
| 4 | Settings session management was wired to a dead/non-canonical session surface | Frontend used stale endpoints and stale field assumptions | Settings page/session UI, IAM user/session handlers | Fixed |
| 5 | Gateway route coverage was incomplete for frontend-used IAM prefixes | Route registry was missing prefixes used by onboarding/invitations/AI governance pages | Gateway, onboarding/invitations/AI governance frontend flows | Fixed in source |
| 6 | Service port/default URL drift created broken startup and proxy assumptions | Gateway defaults, service defaults, and local startup scripts did not all agree on canonical ports | Gateway, workflow, cyber, data, acta, lex, visus, notification, file, local startup flows | Fixed |
| 7 | Smoke verification initially failed against stale or partial runtime processes | The already-running local gateway/PM2 services were not using the rebuilt binaries, and the previous PM2 profile was incomplete | Live local runtime only | Fixed |
| 8 | Notification websocket upgrades failed end to end | Gateway and service middleware wrappers dropped `http.Hijacker`, and the smoke probe used a non-browser websocket handshake | Gateway websocket proxy, notification websocket handler, shared middleware, smoke automation | Fixed |
| 9 | Local dev observability default kept emitting OTLP exporter errors | Base config defaulted OTLP export to `http://localhost:4317`, and PM2 inherited exporter env even when local tracing should have been disabled | Gateway and services bootstrapped from shared config in local dev | Fixed |
| 10 | Data quality dashboard and score trend endpoints returned 500 through the live gateway | Repository queries built day intervals via string concatenation, which broke pgx argument encoding and caused shared quality trend lookups to fail at runtime | `frontend` data quality page, `data-service` quality trend/dashboard handlers, file-service lifecycle cleanup paths with the same query pattern | Fixed |
| 11 | Tenant-scoped cyber/UEBA reads failed or behaved inconsistently in live runtime | Shared tenant context setters used `SET LOCAL ... = $1`, which PostgreSQL does not accept with bind parameters | Shared database tenant wrappers, cyber-service tenant-scoped queries, UEBA dashboard/risk ranking/profile surfaces | Fixed |
| 12 | User session listing returned 500 through the live gateway | IAM session queries scanned PostgreSQL `INET` values directly into Go string pointers instead of converting them to text in SQL | Settings session management, `/api/v1/users/me/sessions`, IAM session repository | Fixed |
| 13 | Notebook server listing failed hard when JupyterHub was unavailable in local/runtime environments | The notebook service treated upstream connectivity failures as fatal instead of degrading to an empty list for a discovery endpoint | Notebook dashboard and `/api/v1/notebooks/servers` | Fixed |
| 14 | VCISO executive briefing and posture summary returned 500 for empty-alert tenants | The overall MTTR aggregate query scanned nullable aggregate results into non-nullable float fields | `/api/v1/cyber/vciso/briefing`, `/api/v1/cyber/vciso/posture-summary`, shared cyber MTTR reporting | Fixed |
| 15 | Lex dashboard queries crashed on empty/default datasets | Contract repository methods appended `ORDER BY c.*` outside the `contractJSONSelect(...)` subquery, referencing an out-of-scope alias | `/api/v1/lex/dashboard`, recent contracts, lex contract list query path | Fixed |
| 16 | External integration admin functionality stopped at a minimal list page | The first pass only exposed basic CRUD, without provider readiness, install guidance, delivery log inspection, or ticket-link detail/sync surfaces | Admin integrations pages, notification-service integration APIs | Fixed |
| 17 | Slack and Jira OAuth install flow was not actually usable from the admin UI | OAuth start depended on request auth context, but the frontend authenticates API calls with bearer headers, not browser redirects | Slack/Jira integration setup, notification-service integration routes, frontend admin install actions | Fixed |
| 18 | Local OAuth readiness was opaque and runtime secrets were not surfaced clearly | PM2/startup config did not expose optional provider secrets consistently, and the integration admin surface could not tell operators which env vars were missing | `ecosystem.local.js`, `scripts/start.sh`, integration provider status API, admin integrations UI | Fixed |
| 19 | Jira and ServiceNow inbound webhook verification was only partially live-verified and unknown links could bubble into noisy failures | Signed inbound handling compiled, but unknown external tickets were not explicitly treated as ignorable no-op events in the live runtime path | Jira webhook receiver, ServiceNow webhook receiver, smoke automation | Fixed |
| 20 | Frontend production build regressed while expanding integration admin pages | App Router hook call sites across the repo still assumed non-null `usePathname`, `useSearchParams`, and related values; the build also lacked a minimal `_document` entry | Integration admin pages plus unrelated dashboard/auth pages sharing the same Next runtime assumptions | Fixed |
| 21 | Continuous DSPM capabilities existed in backend packages but were not actually live in the service runtime | `cyber-service` was not wiring the continuous DSPM engine, shadow-copy route, or pipeline-event subscription, and DSPM dashboard/data-asset responses still exposed backend/frontend field drift | `cyber-service`, `data-service` pipeline events, cyber DSPM dashboard/page | Fixed |
| 22 | Compliance-tag and shadow-copy data were not surfaced to users despite backend support | DSPM assets were returned without compliance-tag enrichment, the dashboard lacked shadow-copy access, and frontend DSPM types still assumed stale classification fields | DSPM service/handler layer, DSPM frontend table/dashboard | Fixed |
| 23 | RCA existed in backend packages but had no reachable frontend workflow | `cyber-service` did not mount RCA routes, and the alert/pipeline detail pages had no root-cause UI even though the engine and analyzers already existed | RCA engine/handler, cyber alert detail page, data pipeline detail page | Fixed |

## Fixes Applied

### Contract normalization

- Standardized shared backend paginated responses on `data` + `meta.{page,per_page,total,total_pages}` for the remaining runtime offenders (`notifications`, `cyber alerts`, `workflow instances`, and the related UEBA/workflow list DTOs).
- Standardized gateway error responses on nested `error.{code,message,details?,request_id?}` and updated the frontend shared API client to decode both nested gateway errors and still-flat downstream service responses safely.
- Removed the last frontend-side `pagination` adapters from the shared suite/data clients.
- Normalized the shared internal pagination helper usage to `per_page` + `meta` semantics.

### Frontend/backend integration hardening

- Verified that settings now use `/api/v1/users/me/sessions` and the corresponding IAM handlers exist.
- Verified that notification websocket routing exists at `/ws/v1/notifications` and that the live gateway now upgrades successfully.
- Hardened websocket auth propagation so the notification service accepts trusted gateway-forwarded identity headers while still supporting direct token auth.
- Fixed shared HTTP/tracing/metrics response-writer wrappers to preserve `Hijacker`/`Flusher` support instead of breaking websocket upgrades.
- Added an executable `scripts/smoke-test.sh` that checks:
  - service health endpoints
  - gateway unauthorized/not-found error contract
  - gateway authentication when credentials are supplied
  - representative frontend-used paginated endpoints across routed services
  - notification websocket upgrade
- Corrected the smoke-test IAM health target to `9081/healthz`, which matches the actual running service layout.
- Corrected the smoke-test websocket probe to use a valid browser-like handshake (`Origin` + valid 16-byte `Sec-WebSocket-Key`).
- Replaced the remaining string-built day interval queries with typed `int * INTERVAL '1 day'` arithmetic in the data quality trend path and matching file-service lifecycle cleanup queries.
- Extended `scripts/smoke-test.sh` to hit `/api/v1/data/quality/dashboard` and `/api/v1/data/quality/score/trend?days=30` so this regression is covered in future local sweeps.
- Replaced the shared tenant-context SQL setter with `set_config(..., true)` in the database helpers so tenant-scoped reads no longer fail on parameterized `SET LOCAL`.
- Normalized IAM session IP extraction to `host(ip_address)` so session listing returns stable text values instead of crashing on `INET` decoding.
- Hardened notebook server discovery to degrade to `[]` when JupyterHub is unavailable, which preserves the frontend contract without hiding non-network errors.
- Coalesced nullable overall MTTR aggregates to `0` for empty-alert tenants so VCISO briefing/posture endpoints return valid zeroed metrics instead of 500s.
- Fixed the remaining lex contract JSON query builders to order by the outer `t.*` alias instead of the inner `c.*` alias, which restored the dashboard and recent-contract paths.
- Normalized UEBA profile and timeline list endpoints to the canonical paginated contract and ensured empty results serialize as `data: []`, not `null`.
- Extended `scripts/smoke-test.sh` to cover the repaired live endpoints: `/api/v1/users/me/sessions`, `/api/v1/notebooks/servers`, `/api/v1/cyber/vciso/briefing`, `/api/v1/cyber/vciso/posture-summary`, `/api/v1/lex/dashboard`, `/api/v1/cyber/ueba/profiles`, `/api/v1/cyber/ueba/alerts`, and `/api/v1/cyber/ueba/profiles/{entityId}/timeline`.

### External integrations admin/runtime completion

- Added provider readiness and capability metadata to the integration API so the admin UI can distinguish OAuth-backed setup (`slack`, `jira`) from manual setup (`teams`, `servicenow`, `webhook`) and surface missing runtime env vars directly.
- Added `GET /api/v1/integrations/providers`, `GET /api/v1/integrations/ticket-links/{id}`, and integration ticket-link filtering by `integration_id`.
- Added protected OAuth bootstrap endpoints for Slack and Jira that create a short-lived install state under the authenticated tenant/user context and return a public redirect URL. This fixed the real browser flow for bearer-based frontend auth.
- Updated public Slack/Jira OAuth start handlers to accept pre-created state IDs, validate state existence, and fail with canonical `503` config errors instead of misleading `401` responses when provider credentials are absent.
- Allowed setup-pending Slack/Jira integrations to be completed incrementally without forcing full validation on every update; valid edits now automatically promote them to `active`.
- Normalized Jira and ServiceNow inbound webhook behavior so valid signed callbacks for unknown external tickets are acknowledged with `200` and logged, rather than surfacing as runtime failures.
- Expanded `scripts/smoke-test.sh` to verify:
  - integration provider inventory
  - Slack/Jira OAuth bootstrap behavior
  - temporary webhook integration CRUD/test/delivery flow
  - signed Slack Events API verification
  - Teams JWT rejection path
  - signed Jira webhook verification
  - ServiceNow shared-secret webhook verification
- Built the full tenant-admin integration frontend surface:
  - provider readiness cards
  - create/edit setup dialog with provider-specific fields
  - integration detail page with overview, delivery log, and ticket-link tabs
  - ticket-link detail page with force-sync action
  - OAuth install actions wired through the protected bootstrap endpoints
- Added local secret-file/env support for:
  - `NOTIF_SLACK_CLIENT_ID`
  - `NOTIF_SLACK_CLIENT_SECRET`
  - `NOTIF_SLACK_SIGNING_SECRET`
  - `NOTIF_ATLASSIAN_CLIENT_ID`
  - `NOTIF_ATLASSIAN_CLIENT_SECRET`
  via `.dev-secrets/*`, `scripts/start.sh`, and `ecosystem.local.js`.
- Restored frontend production-build stability by adding `frontend/src/pages/_document.tsx` and sweeping the remaining nullable Next navigation-hook assumptions that the expanded admin surface exposed.

### Continuous DSPM, compliance tagging, and RCA

- Wired `cyber-service` to instantiate and start the continuous DSPM engine, subscribe it to `data.pipeline.events`, and expose the RCA HTTP routes behind the existing auth and tenant middleware.
- Extended emitted data-pipeline lifecycle events so the continuous DSPM pipeline and transit watchers receive the source/target/table context they already expect.
- Added a live `GET /api/v1/cyber/dspm/shadow-copies` surface and connected it to the existing backend shadow detector.
- Enriched DSPM data-asset responses with compliance tags generated from the existing multi-framework compliance tagger, without mutating the canonical backend asset schema model.
- Extended the DSPM dashboard aggregates to expose the posture metrics the frontend needs for encryption, access-control, and internet-exposure views.
- Corrected the frontend DSPM types and dashboard rendering to the actual backend contract:
  - `data_classification`
  - `classification_breakdown`
  - `pii_assets_count`
  - `high_risk_assets_count`
  - posture/exposure counts
  - `metadata.compliance_tags`
- Added a reusable root-cause analysis panel component and connected it to:
  - cyber alert detail via `GET /api/v1/rca/security_alert/{incidentId}`
  - data pipeline detail via `GET /api/v1/rca/pipeline_failure/{incidentId}`
- Added a new root-cause tab to the alert and pipeline detail pages so the existing RCA engine is now represented in the frontend.
- Added a new shadow-copy section to the DSPM page so the backend shadow-detection capability is visible in the dashboard instead of remaining backend-only.

### Startup and routing consistency

- Verified gateway prefix coverage for onboarding, invitations, AI governance, notifications, files, workflows, and suite routes.
- Verified that IAM health is exposed on the admin port (`9081`) rather than the main port (`8081`).
- Cross-checked the startup script’s canonical port map against service config defaults and gateway downstream URLs.
- Replaced the old partial PM2 profile with a full gateway-centered local topology in `ecosystem.local.js`, backed by `ecosystem.config.js`.
- Explicitly disabled inherited OTLP exporter env in the local PM2/startup path so local services no longer spam trace-export errors by default.

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

`scripts/start.sh` and `ecosystem.local.js` now use the canonical service ports above, and the local PM2 profile represents the full gateway-centered platform layout.

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

## Prompt 59 Files Changed

- `backend/cmd/cyber-service/main.go`
- `backend/internal/cyber/model/dspm.go`
- `backend/internal/cyber/repository/dspm_repo.go`
- `backend/internal/cyber/service/dspm_service.go`
- `backend/internal/cyber/handler/handler_interfaces.go`
- `backend/internal/cyber/handler/dspm_handler.go`
- `backend/internal/cyber/handler/routes.go`
- `backend/internal/data/pipeline/engine.go`
- `frontend/src/lib/constants.ts`
- `frontend/src/types/cyber.ts`
- `frontend/src/components/cyber/root-cause-analysis-panel.tsx`
- `frontend/src/app/(dashboard)/cyber/dspm/page.tsx`
- `frontend/src/app/(dashboard)/cyber/dspm/_components/data-asset-columns.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/[id]/page.tsx`
- `frontend/src/app/(dashboard)/data/pipelines/[id]/page.tsx`

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
- `backend/internal/config/config.go`
- `backend/internal/middleware/logging.go`
- `backend/internal/notification/handler/notification_handler.go`
- `backend/internal/notification/handler/websocket_handler.go`
- `backend/internal/notification/repository/notification_repo.go`
- `backend/internal/data/repository/quality_result_repo.go`
- `backend/internal/database/tenant_context.go`
- `backend/internal/database/instrumented.go`
- `backend/internal/filemanager/repository/file_repo.go`
- `backend/internal/iam/repository/session_repo.go`
- `backend/internal/notebook/service/notebook_service.go`
- `backend/internal/workflow/dto/pagination_dto.go`
- `backend/internal/workflow/dto/instance_dto.go`
- `backend/internal/workflow/dto/definition_dto.go`
- `backend/internal/workflow/handler/helpers.go`
- `backend/internal/workflow/handler/instance_handler.go`
- `backend/internal/workflow/handler/definition_handler.go`
- `backend/internal/cyber/dto/alert_dto.go`
- `backend/internal/cyber/dashboard/mttr.go`
- `backend/internal/cyber/service/alert_service.go`
- `backend/internal/cyber/ueba/dto/alert_dto.go`
- `backend/internal/cyber/ueba/dto/profile_dto.go`
- `backend/internal/cyber/ueba/service/ueba_service.go`
- `backend/internal/lex/repository/contract_repo.go`
- `backend/internal/audit/handler/audit_handler.go`
- `backend/internal/observability/tracing/http_propagation.go`
- `backend/cmd/notification-service/main.go`
- `backend/internal/integration/dto/integration_dto.go`
- `backend/internal/integration/handler/common.go`
- `backend/internal/integration/handler/integration_handler.go`
- `backend/internal/integration/handler/jira_handler.go`
- `backend/internal/integration/handler/routes.go`
- `backend/internal/integration/handler/servicenow_handler.go`
- `backend/internal/integration/handler/slack_handler.go`
- `backend/internal/integration/repository/ticket_link_repo.go`
- `backend/internal/integration/service/integration_service.go`
- `ecosystem.local.js`
- `scripts/start.sh`
- `scripts/smoke-test.sh`
- `frontend/src/app/(dashboard)/admin/integrations/page.tsx`
- `frontend/src/app/(dashboard)/admin/integrations/[id]/page.tsx`
- `frontend/src/app/(dashboard)/admin/integrations/ticket-links/[id]/page.tsx`
- `frontend/src/app/(dashboard)/admin/integrations/_components/integration-form-dialog.tsx`
- `frontend/src/app/(dashboard)/admin/integrations/_components/delivery-log-table.tsx`
- `frontend/src/app/(dashboard)/admin/integrations/_components/ticket-links-table.tsx`
- `frontend/src/app/(dashboard)/admin/integrations/_components/integration-utils.ts`
- `frontend/src/types/integration.ts`
- `frontend/src/config/navigation.ts`
- `frontend/src/pages/_document.tsx`
- `frontend/src/hooks/use-data-table.ts`
- `frontend/src/hooks/use-breadcrumbs.ts`
- `frontend/src/components/layout/sidebar-nav-item.tsx`
- `frontend/src/components/auth/session-expired-dialog.tsx`
- `frontend/src/app/(dashboard)/notifications/notifications-page-client.tsx`
- `frontend/src/app/(dashboard)/workflows/tasks/tasks-page-client.tsx`
- `frontend/src/app/(dashboard)/data/sources/page.tsx`
- `backend/internal/observability/metrics/http_metrics.go`
- `backend/cmd/api-gateway/main.go`
- `frontend/src/lib/api.ts`
- `frontend/src/lib/suite-api.ts`
- `frontend/src/lib/data-suite/api.ts`
- `frontend/src/lib/data-suite/types.ts`
- `frontend/src/lib/enterprise/api.ts`
- `frontend/src/app/(dashboard)/cyber/ueba/_components/types.ts`
- `frontend/src/app/(dashboard)/settings/page.tsx`
- `frontend/src/__tests__/data-suite-fixtures.ts`
- `ecosystem.local.js`
- `ecosystem.config.js`
- `scripts/smoke-test.sh`
- `scripts/start.sh`

## Verification Performed

Verified successfully:

- `cd /Users/mac/clario360/backend && GOWORK=off go build ./...`
- `cd /Users/mac/clario360/frontend && npm run build`
- `cd /Users/mac/clario360/frontend && npm run type-check`
- `cd /Users/mac/clario360/backend && GOWORK=off go test ./internal/cyber/dspm/compliance ./internal/cyber/dspm/shadow ./internal/rca`
- `node -c /Users/mac/clario360/ecosystem.local.js`
- `bash -n /Users/mac/clario360/scripts/smoke-test.sh`
- `bash -n /Users/mac/clario360/scripts/start.sh`
- `pm2 restart /Users/mac/clario360/ecosystem.config.js --update-env`
- `CLARIO360_SMOKE_EMAIL=admin@clario.dev CLARIO360_SMOKE_PASSWORD='Cl@rio360Dev!' bash /Users/mac/clario360/scripts/smoke-test.sh`
  - final result: `pass=58 fail=0 skip=0`
- Direct authenticated gateway verification after PM2 restart:
  - `GET /api/v1/data/quality/score/trend?days=30` -> `200 {"data":[]}`
  - `GET /api/v1/data/quality/dashboard` -> `200 {"data":{...}}`
  - `GET /api/v1/cyber/vciso/briefing` -> `200 {"data":{...}}`
  - `GET /api/v1/cyber/vciso/posture-summary` -> `200 {"data":{...}}`
  - `GET /api/v1/lex/dashboard` -> `200 {"data":{...}}`
  - `GET /api/v1/cyber/ueba/profiles?page=1&per_page=5` -> `200 {"data":[],"meta":{...}}`
  - `GET /api/v1/cyber/ueba/profiles/test-entity/timeline?page=1&per_page=20` -> `200 {"data":[],"meta":{...}}`
  - `GET /api/v1/notebooks/servers` -> `200 []`
  - `GET /api/v1/users/me/sessions` -> `200 [...]`
  - `GET /api/v1/integrations/providers` -> `200 {"data":[...]}` with provider readiness and missing runtime config surfaced for Slack/Jira
  - `POST /api/v1/integrations/slack/oauth/session` -> canonical `503 {"error":...}` when local Slack OAuth secrets are absent
  - `POST /api/v1/integrations/jira/oauth/session` -> canonical `503 {"error":...}` when local Atlassian OAuth secrets are absent
  - `GET /api/v1/cyber/dspm/shadow-copies` -> `200 {"data":{"matches":[],"summary":...}}`
- Browser-style websocket probe through the live gateway returned `HTTP/1.1 101 Switching Protocols`
- Production `next build` completed successfully without the previous standalone trace warning about `page_client-reference-manifest.js`
- Additional Prompt 59 runtime verification:
  - targeted sequential PM2 restarts for `iam-service`, `cyber-service`, and `data-service` restored live listener availability after the PM2-wide restart left those services between bootstrap and HTTP bind
  - `GET http://localhost:8085/healthz` -> `200`
  - `GET http://localhost:8086/healthz` -> `200`
  - authenticated gateway probes confirm `0` seeded alerts and `0` seeded pipelines in the current local tenant, so live RCA route exercise was limited to route availability and frontend/runtime integration rather than a real incident replay

## Residual Risks / Follow-Up

- No blocking runtime issues remain from this audit pass.
- The local dev tenant currently has no seeded alerts or pipeline runs, so the new RCA UI and endpoints are live but were not exercised against a real incident graph in this pass.
- A full `pm2 restart ecosystem.config.js --update-env` can still leave some Go services temporarily wedged before HTTP bind during local infra churn; sequential service restarts recovered the stack without source changes, so this appears to be an existing runtime/startup coordination issue rather than a Prompt 59 regression.
- Real Slack and Jira workspace/project installs now have a working in-product bootstrap flow, but they still require operators to supply valid vendor credentials:
  - `NOTIF_SLACK_CLIENT_ID`
  - `NOTIF_SLACK_CLIENT_SECRET`
  - `NOTIF_ATLASSIAN_CLIENT_ID`
  - `NOTIF_ATLASSIAN_CLIENT_SECRET`
  Until those are configured, the platform correctly reports the providers as not ready and returns canonical `503` install-bootstrap errors.
- PM2 error logs still contain historical OTLP exporter errors from before the final env fix; they are stale log entries, not current runtime failures.
