Prompt: Clario360 Endpoint Validation, Repair, and Verification

You are performing a platform-wide endpoint audit for the Clario360 platform. Your job is to discover, test, fix, and verify every meaningful HTTP and WebSocket endpoint exposed by this system, starting from the real frontend contracts and tracing all the way through the gateway, handlers, services, repositories, and runtime topology.

Do not stop at the first visible failure. Treat every broken endpoint, 5xx, contract mismatch, routing gap, auth propagation bug, pagination defect, startup drift, websocket failure, or smoke-test blind spot as evidence that similar defects may exist elsewhere. Audit broadly, fix systematically, and verify live.

You are working in:

- `/Users/mac/clario360`
- `/Users/mac/clario360/ecosystem.local.js`
- `/Users/mac/clario360/ecosystem.config.js`

Repository layout:

```text
/Users/mac/clario360/
├── frontend/                    # Next.js 14 App Router, TypeScript
│   ├── src/
│   │   ├── app/(dashboard)/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── lib/
│   │   ├── stores/
│   │   └── types/
│   └── package.json
├── backend/                     # Go module: github.com/clario360/platform
│   ├── cmd/
│   │   ├── api-gateway/         # 8080
│   │   ├── iam-service/         # 8081
│   │   ├── workflow-engine/     # 8083
│   │   ├── audit-service/       # 8084
│   │   ├── cyber-service/       # 8085
│   │   ├── data-service/        # 8086
│   │   ├── acta-service/        # 8087
│   │   ├── lex-service/         # 8088
│   │   ├── visus-service/       # 8089
│   │   ├── notification-service/# 8090
│   │   └── file-service/        # 8091
│   └── internal/
├── scripts/
│   ├── start.sh
│   ├── stop.sh
│   └── smoke-test.sh
└── migrations/
```

Current implementation truths you must respect:

- `ecosystem.local.js` is the canonical full local PM2 topology.
- `ecosystem.config.js` is the PM2 entrypoint wrapper and must remain consistent with `ecosystem.local.js`.
- Public API traffic goes through the gateway at `http://localhost:8080`.
- Current local health endpoints are:
  - gateway: `http://localhost:8080/healthz`
  - iam: `http://localhost:9081/healthz`
  - workflow: `http://localhost:8083/healthz`
  - audit: `http://localhost:8084/healthz`
  - cyber: `http://localhost:8085/healthz`
  - data: `http://localhost:8086/healthz`
  - acta: `http://localhost:8087/healthz`
  - lex: `http://localhost:8088/healthz`
  - visus: `http://localhost:8089/healthz`
  - notification: `http://localhost:8090/healthz`
  - file: `http://localhost:9091/healthz`
- Canonical paginated HTTP response shape is:

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 0,
    "total_pages": 1
  }
}
```

- Canonical gateway-facing error HTTP response shape is:

```json
{
  "error": {
    "code": "SOME_CODE",
    "message": "Human readable message",
    "details": {},
    "request_id": "..."
  }
}
```

- The frontend `ApiError` type in `frontend/src/types/api.ts` remains flat. `frontend/src/lib/api.ts` is responsible for normalizing nested gateway errors and legacy flat backend errors into that flat client-side shape. Do not change TypeScript contract types to chase backend drift.
- Notification websocket routing exists at `/ws/v1/notifications`.
- Gateway and downstream middleware must preserve `http.Hijacker` semantics for websocket upgrades.
- Browser-like websocket verification must send an allowed `Origin` header and a valid 16-byte `Sec-WebSocket-Key`.
- Local dev must keep tracing export disabled unless you are explicitly auditing telemetry wiring:
  - `OBSERVABILITY_OTLP_ENDPOINT=""`
  - `OTEL_EXPORTER_OTLP_ENDPOINT=""`
- `scripts/smoke-test.sh` already exists. Extend it in place.
- `AUDIT_FINDINGS.md` already exists. Update it in place.

Non-negotiable rules:

- Always prefix Go commands with `GOWORK=off`.
- Run Go commands from `/Users/mac/clario360/backend`.
- Build services with `GOWORK=off go build -o <name> ./cmd/<name>/` or equivalent.
- Never assume an endpoint works because it compiles. Trace frontend usage to backend implementation and verify live behavior.
- Never change the TypeScript contract types in `frontend/src/types/` to fit broken backends.
- Never modify `*_test.go` or `*.test.ts(x)` unless a test is factually wrong.
- Do not add speculative features. Fix real defects and harden real failure paths.
- Continue past the first fix. Search for repeated defect patterns and fix them platform-wide where truly applicable.
- If you change live runtime behavior, rebuild the affected binaries, restart PM2 with `pm2 restart ecosystem.config.js --update-env`, and reverify the live ports.

Core objective:

Test and fix all platform endpoints, including:

- all frontend-used API endpoints
- all registered gateway-routed HTTP endpoints
- all meaningful service-level REST endpoints exposed by each backend service
- all websocket endpoints
- all health and admin surfaces that are part of the local runtime contract

Work phase by phase. Maintain an internal issue ledger with:

- endpoint
- failure or risk
- impact
- root cause
- affected files
- fix status
- verification performed

Phase 1: Understand the contracts and runtime

Read these files completely first:

Frontend contracts and shared API access:

- `frontend/src/types/api.ts`
- `frontend/src/types/models.ts`
- `frontend/src/types/table.ts`
- `frontend/src/lib/api.ts`
- `frontend/src/lib/constants.ts`
- `frontend/src/lib/suite-api.ts`
- `frontend/src/lib/data-suite/api.ts`
- `frontend/src/lib/enterprise/api.ts`

Frontend data and realtime hooks:

- `frontend/src/hooks/use-data-table.ts`
- `frontend/src/hooks/use-realtime-data.ts`
- `frontend/src/hooks/use-websocket.ts`

Gateway and local runtime:

- `backend/cmd/api-gateway/main.go`
- `backend/internal/gateway/config/routes.go`
- `ecosystem.local.js`
- `scripts/start.sh`
- `scripts/smoke-test.sh`

Goal:

- understand the canonical frontend contracts
- understand how auth, pagination, errors, and realtime are supposed to work
- understand the real PM2/local runtime topology and health ports
- build a route map of gateway prefix to backend service

Phase 2: Build the full endpoint inventory

Inventory endpoints from all of these sources:

1. Frontend call sites

Scan all files under `frontend/src/` for:

- `api.get`, `api.post`, `api.put`, `api.patch`, `api.delete`
- `apiGet`, `apiPost`, `apiPut`, `apiPatch`, `apiDelete`
- `fetch(`
- `useDataTable`
- `useRealtimeData`
- `useQuery`
- `useMutation`
- direct `WebSocket` usage

2. Gateway route ownership

Read:

- `backend/internal/gateway/config/routes.go`
- `backend/cmd/api-gateway/main.go`

3. Backend service router registrations

For every service under `backend/cmd/`, trace router registration down to actual handlers and produce a complete endpoint list.

Your endpoint inventory must include:

- service
- frontend file if applicable
- backend handler file
- HTTP method
- route pattern
- whether it is public, authenticated, tenant-scoped, admin-only, or websocket
- expected request shape
- expected response shape
- whether it is list, detail, mutation, action, health, or websocket
- whether it is feasible to test automatically in local dev

Do not sample. Inventory all meaningful endpoints.

Phase 3: Execute endpoint testing

Test the inventory systematically.

For each feasible endpoint:

- call it through the gateway if it is gateway-facing
- authenticate where required
- verify status code
- verify response contract
- verify tenant/auth propagation
- verify pagination behavior where applicable
- verify invalid input handling
- verify that failures return the correct error shape

For mutations:

- use safe test data
- prefer idempotent flows where possible
- clean up created records when needed
- do not leave the environment in a broken or polluted state

For endpoints that cannot be exercised safely:

- document why
- inspect implementation deeply instead of skipping casually

Phase 4: Contract and handler audit

For every discovered endpoint, locate the actual handler and audit:

1. Paginated list endpoints

All paginated list endpoints must return exactly:

```json
{
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 0,
    "total_pages": 1
  }
}
```

Search for and normalize any non-standard shapes such as:

- `items`
- `results`
- `records`
- `tasks`
- `alerts`
- `instances`
- `definitions`
- `pagination`
- `page_size`
- `last_page`
- `pages`
- `count`
- top-level `page`, `per_page`, `total`, `total_pages`

2. Single-resource endpoints

Determine whether the frontend expects:

- raw object
- or `{ "data": <object> }`

Make backend behavior consistent with the actual frontend expectation.

3. Query/filter handling

Audit every handler for:

- `page`
- `per_page`
- search
- sort field and direction
- multi-value filters like `status=a,b,c`
- invalid values

Verify:

- `per_page` is accepted consistently
- invalid values return `400`, not `500`
- defaults are sane
- sort fields are validated
- filters are not silently ignored

4. Error response shape

Requirements:

- gateway-facing/public errors must use the nested `error` envelope
- downstream services should move toward the canonical contract, not away from it
- frontend `ApiError` types must not be changed to match broken backends

5. Auth and tenant propagation

Trace how identity is authenticated and forwarded through the gateway.

Verify each service:

- reads tenant context correctly
- rejects missing auth when required
- scopes queries by tenant where required
- does not leak cross-tenant data

Phase 5: Service-wide pattern hunt

When you find one broken pattern, search the whole repo for the same class of defect and fix all real occurrences.

Specifically hunt for:

- inconsistent response wrappers
- duplicated pagination DTO drift
- missing `per_page` support
- nil slices serialized as `null` instead of `[]`
- ignored filters
- nil pointer risks in handlers
- invalid SQL placeholder numbering
- unsafe dynamic SQL or sort injection
- unbounded queries
- broken tenant scoping
- broken websocket upgrades caused by response-writer wrappers dropping `Hijacker` or `Flusher`
- local config defaults that accidentally re-enable tracing export
- interval queries built via string concatenation instead of typed query parameters

Phase 6: WebSocket and realtime audit

Audit realtime support end to end.

1. Gateway websocket proxy

Verify in gateway code:

- `/ws/v1/...` routes exist
- proxy/upgrade behavior is correct
- auth token handling is correct
- notification websocket path is routed
- gateway middleware does not break upgrade semantics

2. Notification service

Audit:

- `backend/cmd/notification-service/`
- `backend/internal/notification/`

Verify:

- REST endpoints used by the frontend exist and are registered
- unread count exists
- list notifications exists
- mark-read and related update endpoints exist if used
- websocket handler upgrades correctly
- websocket auth works both for direct token auth and trusted gateway-forwarded identity headers

3. Frontend websocket client

Audit `frontend/src/hooks/use-websocket.ts` for:

- reconnect behavior
- cleanup on unmount
- handling when the service is unavailable
- prevention of component crashes on socket failure

4. Websocket verification

When verifying with `curl`, use a browser-like handshake:

- `Origin: http://localhost:3000`
- valid `Sec-WebSocket-Key`, for example `dGhlIHNhbXBsZSBub25jZQ==`

Phase 7: Startup and configuration audit

Read `scripts/start.sh` and `ecosystem.local.js` fully.

For each service, verify:

- HTTP port env var
- admin/health port env var
- DB env vars
- JWT key env vars
- required secrets
- timeout/duration env vars
- defaults

Cross-check actual code expectations against:

- `scripts/start.sh`
- `ecosystem.local.js`
- gateway downstream URLs
- live local health endpoints

Validate:

- env var names match exactly
- ports do not conflict
- duration strings are valid Go durations where expected
- health checks hit the correct port
- IAM health uses `9081`
- file-service health uses `9091`
- local PM2 config keeps OTLP export disabled unless explicitly auditing telemetry

Phase 8: Database and repository audit

For each repository layer:

1. Connection hygiene

Verify:

- one pool per service
- `rows.Close()` is deferred
- transactions roll back on failure paths
- pool settings are sane

2. Query correctness

Audit:

- placeholder numbering
- dynamic `WHERE` construction
- `IN (...)` and `ANY($N)` handling
- pagination offset/limit logic
- sort validation
- typed interval/date query handling

3. Index coverage

Compare real query patterns against migrations.
Add indexes only where justified by actual endpoint/query usage.

Focus on columns such as:

- `tenant_id`
- `status`
- `created_at`
- foreign keys used in joins
- high-cardinality list filters

Phase 9: Frontend defensive hardening

Audit all frontend endpoint consumers for brittle assumptions.

Search for:

- unsafe `.meta.total`
- unsafe `.meta.page`
- unsafe `.data.length`
- legacy `.pagination.*`
- nested property access on async results without guards
- render-time side effects
- loading/error-state crashes

Fix repeated frontend assumptions systematically.

Phase 10: Build, runtime, and smoke verification

After fixes:

Backend:

```bash
cd /Users/mac/clario360/backend
GOWORK=off go build ./...
```

Frontend:

```bash
cd /Users/mac/clario360/frontend
npm run type-check
npm run build
```

Runtime:

- rebuild any affected service binaries
- restart PM2 with `pm2 restart ecosystem.config.js --update-env`
- reverify health endpoints
- reverify the specific endpoints that were fixed

Smoke automation:

Update `scripts/smoke-test.sh` so it:

- checks all local health endpoints
- authenticates through the gateway
- captures an access token
- exercises representative endpoints from every service
- covers all critical frontend-used endpoints
- validates paginated endpoints return `data` plus canonical `meta`
- validates error endpoints return the canonical nested `error` shape
- tests websocket upgrade for notifications with a valid browser-like handshake
- prints a clear pass/fail summary

The smoke script must remain self-contained and use only:

- bash
- curl
- jq
- python3

Phase 11: Documentation

Update `/Users/mac/clario360/AUDIT_FINDINGS.md` with:

- issue summary
- root cause
- affected endpoints and services
- files changed
- verification performed
- residual risks or follow-up items

Deliverables:

- all discovered endpoint defects fixed where feasible
- all affected backend services compiling
- frontend type-check passing
- frontend production build passing
- `scripts/smoke-test.sh` updated and passing
- `AUDIT_FINDINGS.md` updated
- local PM2 stack aligned with `ecosystem.local.js`
- concise final report covering:
  - what endpoints were inventoried
  - what failed
  - what was fixed
  - what was verified live
  - what remains blocked, if anything

Priority order:

1. Crashes and panics
2. Broken routing or missing handler registration
3. Wrong API response shapes
4. Auth or tenant propagation failures
5. Configuration or startup mismatches
6. Query correctness and data consistency bugs
7. Frontend hardening gaps
8. Performance and reliability issues

Final instruction:

Do not stop after addressing one endpoint or one service. Audit the platform endpoint surface deeply, fix repeated classes of defects, rebuild, restart, and verify the live runtime behavior end to end.
