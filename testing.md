Prompt: Clario360 Platform-Wide Quality Audit & Hardening

You are performing a full-stack quality audit of the Clario360 platform. Your job is to systematically discover, fix, and verify integration bugs, API contract mismatches, runtime crashes, configuration gaps, routing issues, and reliability problems across the entire system.

Do not narrow the work to a single visible symptom. Treat any observed failure as a signal that similar issues may exist elsewhere. Audit broadly, verify everything, and keep going until the platform is materially hardened.

You are working in:

/Users/mac/clario360
/Users/mac/clario360/ecosystem.local.js
/Users/mac/clario360/ecosystem.config.js

Repository layout:

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

Current implementation truths you must respect:

- `ecosystem.local.js` is now the canonical full local PM2 topology. Do not treat it as a partial cyber-only profile.
- `ecosystem.config.js` is the PM2 entrypoint wrapper and should remain consistent with `ecosystem.local.js`.
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

- The frontend `ApiError` type in `frontend/src/types/api.ts` remains flat, and `frontend/src/lib/api.ts` is responsible for normalizing nested gateway errors and legacy flat backend errors into that flat client-side shape. Do not change the TypeScript contract types to chase backend drift.
- Notification websocket routing exists at `/ws/v1/notifications`. Gateway and service middleware must preserve `http.Hijacker` semantics. Browser-like websocket verification must send an allowed `Origin` header and a valid 16-byte `Sec-WebSocket-Key`.
- Local dev must keep tracing export disabled unless you are explicitly auditing telemetry wiring. Respect:
  - `OBSERVABILITY_OTLP_ENDPOINT=""`
  - `OTEL_EXPORTER_OTLP_ENDPOINT=""`
- `scripts/smoke-test.sh` already exists. Update and extend it; do not replace it with a different toolchain.
- `AUDIT_FINDINGS.md` already exists. Update it in place.

Critical rules:

- Always prefix Go commands with `GOWORK=off`.
- Run Go commands from `/Users/mac/clario360/backend`.
- Build services with `GOWORK=off go build -o <name> ./cmd/<name>/` or equivalent.
- Never assume something works because it compiles. Trace frontend usage to backend implementation and verify the contract.
- Never change the TypeScript contract types in `frontend/src/types/` to fit broken backends. Backends must conform to frontend contracts.
- Never modify `*_test.go` or `*.test.ts(x)` unless a test is factually wrong.
- Do not add speculative features. Fix real defects and harden real failure paths.
- Continue beyond the first fix. Audit for adjacent and repeated patterns.
- If you change live runtime behavior, rebuild the affected binaries and restart the PM2 stack with `pm2 restart ecosystem.config.js --update-env`, then reverify the live ports.

Execution requirements:

- Work phase by phase.
- Maintain an internal issue ledger while auditing: discovered issue, impact, root cause, affected files, fix status.
- For every bug found, fix it, then verify it.
- When you find one contract mismatch pattern in one service, actively search for the same pattern across all services.
- Favor platform-wide consistency over one-off local patches.
- Prefer updating existing shared helpers and middleware when a defect pattern is systemic.

Phase 1: Understand the contracts

Read these files completely first:

Frontend contracts and API access:
- `frontend/src/types/api.ts`
- `frontend/src/types/models.ts`
- `frontend/src/types/table.ts`
- `frontend/src/lib/api.ts`
- `frontend/src/lib/constants.ts`

Frontend data hooks:
- `frontend/src/hooks/use-data-table.ts`
- `frontend/src/hooks/use-realtime-data.ts`
- `frontend/src/hooks/use-websocket.ts`

Gateway and local runtime:
- `backend/cmd/api-gateway/main.go`
- `ecosystem.local.js`
- `scripts/smoke-test.sh`

Goal:
- Understand the canonical frontend API contracts.
- Understand how pagination, errors, auth, and realtime are expected to work.
- Build a route map of gateway path prefix → backend service.
- Understand the actual local runtime topology and health ports, not just the nominal service ports.

Phase 2: Full frontend API inventory

Scan all files under `frontend/src/` for:
- `api.get`, `api.post`, `api.put`, `api.patch`, `api.delete`
- `apiGet`, `apiPost`, etc.
- `fetch(`
- `useDataTable`
- `useRealtimeData`
- `useQuery`
- `useMutation`
- WebSocket usage

For every discovered API call, produce a complete map with:
- frontend file
- HTTP method
- URL/path pattern
- expected TypeScript response type
- expected pagination/error shape if applicable
- backend service responsible, based on gateway routing

Do not sample. Inventory all of them.

Phase 3: Backend handler contract audit

For each discovered endpoint, locate the actual Go handler and audit:

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

Check for any non-standard shapes such as:
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
- top-level `total/page/per_page/total_pages`

Normalize them to the standard contract.

2. Single-resource endpoints
Check whether the frontend expects:
- raw object
or
- `{ "data": <object> }`

Make backend behavior consistent with frontend expectations.

3. Query/filter handling
Check all handlers for:
- `page`
- `per_page`
- sorting params
- search params
- multi-value filters like `status=a,b,c`

Verify:
- `per_page` is accepted consistently
- comma-separated values are parsed correctly
- invalid values return 400, not 500
- defaults are sane and documented in code

4. Error response shape
Audit each service’s error writer.

Requirements:
- Gateway-facing/public HTTP errors should use the nested `error` envelope shown above.
- The frontend normalizer in `frontend/src/lib/api.ts` currently supports both nested gateway errors and legacy flat errors, but new fixes should move backend/runtime behavior toward the canonical nested gateway-facing contract.
- Do not change frontend TypeScript types to match backend drift.

5. Auth/tenant propagation
Trace how the gateway authenticates requests and forwards identity/tenant context.
Verify each downstream service reads auth context correctly and consistently.

Phase 4: Service-wide backend consistency audit

For every service under `backend/cmd/`, inspect:
- handler layer
- service layer
- repository layer
- DTOs
- pagination helpers
- shared response helpers
- shared middleware that wraps `http.ResponseWriter`

Look for repeated defect patterns across services, including:
- inconsistent list response wrappers
- duplicated pagination DTOs with divergent JSON fields
- missing `per_page` support
- silently ignored filters
- invalid placeholder indexing in SQL builders
- nil pointer risks in handlers
- unbounded queries without limits
- tenant scoping omissions
- inconsistent status validation
- top-level response fields that don’t match frontend assumptions
- nil slices serialized as `null` instead of `[]`
- middleware wrappers that break websocket upgrades by dropping `http.Hijacker` / `http.Flusher`
- local config defaults that re-enable unwanted dev behaviors, such as OTLP export noise

When you find one pattern, search the whole repo for it and fix all occurrences that are truly broken.

Phase 5: WebSocket and notification audit

Audit realtime support end to end.

1. Gateway WebSocket proxy
Verify in `backend/cmd/api-gateway/main.go`:
- `/ws/v1/...` routes exist
- upgrade/proxy behavior is correct
- auth token handling for websocket connections is correct
- notification websocket path is routed
- gateway-side middleware does not break the client upgrade path

2. Notification service
Audit:
- `backend/cmd/notification-service/`
- `backend/internal/notification/`

Verify:
- REST endpoints used by frontend exist and are registered
- unread count endpoint exists
- list notifications endpoint exists
- mark-read/update endpoints exist if used
- websocket handler exists and upgrades properly
- websocket auth works both for direct token auth and for trusted gateway-forwarded identity headers when proxied

3. Frontend websocket client
Audit `frontend/src/hooks/use-websocket.ts` for:
- retry behavior
- cleanup on unmount
- failure handling when service is unavailable
- prevention of component tree crashes on socket failures

4. Smoke websocket verification
When verifying the websocket path with `curl`, use a browser-like handshake:
- send `Origin: http://localhost:3000` unless the allowed origin config differs
- use a valid `Sec-WebSocket-Key` such as `dGhlIHNhbXBsZSBub25jZQ==`

Fix any broken routing, missing registration, brittle client behavior, or middleware upgrade breakage.

Phase 6: Configuration and startup audit

Read `scripts/start.sh` and `ecosystem.local.js` fully.

For each service, inspect its `main.go`, config loader, and env parsing. Build a verified table of:
- HTTP port env var
- health/admin port env var
- DB env vars
- JWT/public/private key env vars
- required secrets
- duration env vars
- defaults

Cross-check against `scripts/start.sh`, `ecosystem.local.js`, and actual service expectations.

Validate:
- env var names match exactly
- ports do not conflict
- JWT keys are passed in the expected format
- duration values use valid Go duration syntax
- required secrets/keys satisfy service validation rules
- health checks point to the correct live port
  - especially IAM on `9081`
  - especially file-service on `9091`
- local PM2 config keeps OTLP export disabled unless explicitly enabled for the audit

Fix broken env names, inconsistent config parsing, startup mismatches, or stale local topology assumptions.

Phase 7: Database and repository audit

For each repository layer:

1. Pool/connection hygiene
Verify:
- only one pool per service
- `rows.Close()` is always deferred
- transactions always rollback on failure paths
- pool settings are reasonable

2. Query correctness
Audit:
- placeholder numbering
- dynamic WHERE clauses
- `IN (...)` placeholder generation
- `ANY($N)` argument typing
- pagination offset/limit logic
- sort field validation to avoid SQL injection

3. Index coverage
Compare frequent query filters against migrations.
Flag and fix missing indexes on important columns such as:
- `tenant_id`
- `status`
- `created_at`
- foreign keys used in joins
- high-cardinality filters used in list endpoints

Only add indexes where justified by real query patterns.

Phase 8: Frontend defensive hardening

Audit all frontend code under `frontend/src/` for brittle API assumptions.

1. Optional chaining / null safety
Search for unsafe access patterns including:
- `.meta.total`
- `.meta.page`
- `.data.length`
- `.pagination.*`
- nested property access on async query results

Fix with proper guards where necessary.

2. Loading and error states
Verify components and hooks do not dereference undefined data during loading/error states.

3. Render-time side effects
Search for `setState` or other side effects during render.
Move them into effects or handlers.

4. Error boundaries / graceful failure
Ensure page-level crashes degrade into an error state, empty state, or retryable UI instead of a white screen.

5. Repeated contract assumptions
If one hook assumes a response shape unsafely, search for similar assumptions everywhere and fix them systematically.

Phase 9: Build and compile verification

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

Do not stop at partial success. Fix all build and type errors introduced or revealed by the audit.

Phase 10: Smoke-test automation

Update `scripts/smoke-test.sh` so it:

- checks service health endpoints
- authenticates through the gateway
- captures an access token
- exercises all discovered frontend-used API endpoints where feasible
- validates 2xx responses where expected
- validates paginated endpoints return:
  - `data` as an array
  - `meta.page` as a number
  - `meta.per_page` as a number
  - `meta.total` as a number
  - `meta.total_pages` as a number
- validates error endpoints return the canonical nested `error` shape
- tests WebSocket upgrade for notifications using a valid browser-like handshake
- prints a clear pass/fail summary

The script must remain self-contained and rely only on:

- bash
- curl
- jq
- python3

Phase 11: Documentation of findings

Update `/Users/mac/clario360/AUDIT_FINDINGS.md` with:

- issue summary
- root cause
- services/pages affected
- files changed
- verification performed
- any residual risks or follow-up items

Prioritize fixes in this order:

1. Crashes and panics
2. Broken routing / missing service registration
3. Wrong API response shapes
4. Auth / tenant propagation failures
5. Config / startup mismatches
6. Query correctness and data consistency bugs
7. Frontend hardening gaps
8. Performance and reliability issues

Deliverables:

- all discovered defects fixed where feasible
- all affected services compiling
- frontend type-check and production build passing
- `scripts/smoke-test.sh` updated and passing
- `AUDIT_FINDINGS.md` updated
- local PM2 stack aligned with `ecosystem.local.js`
- concise final report summarizing:
  - what was audited
  - what was fixed
  - what was verified
  - what remains blocked, if anything

Do not stop after addressing one obvious symptom. Audit the platform deeply and fix repeated classes of issues across the board.
