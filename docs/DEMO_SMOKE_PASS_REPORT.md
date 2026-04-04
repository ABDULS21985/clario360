# Demo Smoke Pass Report

Date: 2026-03-27

## Scope

This was a live API-backed smoke pass for the vCISO governance pages after the demo seed run.

- Service under test: local `cyber-service`
- API base used for checks: `http://[::1]:8085`
- Health check: `http://[::1]:9090/healthz` returned `200 {"status":"alive"}`
- Auth model: RS256 bearer token signed with the repo's local dev key for tenant `aaaaaaaa-0000-0000-0000-000000000001`
- Validation style: read-path checks only

This was not a browser-rendering pass. It verifies that the API routes used by the page files return authenticated, populated data. It does not validate mutations, websocket updates, or chat/LLM workflows.

## Summary

- `10/11` major vCISO governance pages passed
- All governance list/stats pages added for demo coverage returned populated data
- The one remaining blocker is the top-level vCISO overview page, whose briefing endpoint did not complete successfully

## Results

| Page | Frontend file | Endpoint checks | Outcome | Notes |
| --- | --- | --- | --- | --- |
| vCISO Overview | `frontend/src/app/(dashboard)/cyber/vciso/page.tsx` | `GET /api/v1/cyber/vciso/briefing` | Fail | Timed out in the smoke matrix, then disconnected after about `49s` on a longer run |
| Risk Register | `frontend/src/app/(dashboard)/cyber/vciso/risk-register/page.tsx` | `GET /api/v1/cyber/vciso/risks`, `GET /api/v1/cyber/vciso/risks/stats` | Pass | List returned `25` rows; stats reported `total=240` |
| Policies | `frontend/src/app/(dashboard)/cyber/vciso/policies/page.tsx` | `GET /api/v1/cyber/vciso/policies/stats`, `GET /api/v1/cyber/vciso/policies`, `GET /api/v1/cyber/vciso/policy-exceptions` | Pass | Policies returned `25` rows; policy exceptions returned `25`; stats payload returned `200` |
| Third Party | `frontend/src/app/(dashboard)/cyber/vciso/third-party/page.tsx` | `GET /api/v1/cyber/vciso/vendors/stats`, `GET /api/v1/cyber/vciso/vendors`, `GET /api/v1/cyber/vciso/questionnaires` | Pass | Vendors returned `25`; questionnaires returned `25`; vendor stats reported `total=140` |
| Evidence | `frontend/src/app/(dashboard)/cyber/vciso/evidence/page.tsx` | `GET /api/v1/cyber/vciso/evidence`, `GET /api/v1/cyber/vciso/evidence/stats` | Pass | List returned `25`; stats reported `total=320` |
| Maturity | `frontend/src/app/(dashboard)/cyber/vciso/maturity/page.tsx` | `GET /api/v1/cyber/vciso/maturity`, `GET /api/v1/cyber/vciso/benchmarks`, `GET /api/v1/cyber/vciso/budget/summary`, `GET /api/v1/cyber/vciso/budget` | Pass | Maturity returned `25`; benchmarks returned `96`; budget returned `25`; summary payload returned `200` |
| Compliance | `frontend/src/app/(dashboard)/cyber/vciso/compliance/page.tsx` | `GET /api/v1/cyber/vciso/obligations`, `GET /api/v1/cyber/vciso/control-tests`, `GET /api/v1/cyber/vciso/control-dependencies` | Pass | Each list returned `25` rows |
| Awareness | `frontend/src/app/(dashboard)/cyber/vciso/awareness/page.tsx` | `GET /api/v1/cyber/vciso/iam-findings/summary`, `GET /api/v1/cyber/vciso/awareness`, `GET /api/v1/cyber/vciso/iam-findings` | Pass | Awareness programs returned `25`; IAM findings returned `25`; summary payload returned `200` |
| Incident Readiness | `frontend/src/app/(dashboard)/cyber/vciso/incident-readiness/page.tsx` | `GET /api/v1/cyber/vciso/escalation-rules`, `GET /api/v1/cyber/vciso/playbooks` | Pass | Each list returned `25` rows |
| Integrations | `frontend/src/app/(dashboard)/cyber/vciso/integrations/page.tsx` | `GET /api/v1/cyber/vciso/integrations` | Pass | Returned `90` integrations |
| Workflows | `frontend/src/app/(dashboard)/cyber/vciso/workflows/page.tsx` | `GET /api/v1/cyber/vciso/control-ownership`, `GET /api/v1/cyber/vciso/approvals` | Pass | Each list returned `25` rows |

## Findings

### 1. Governance demo coverage is now broadly live

The seeded vCISO governance surfaces are returning populated data through the actual authenticated endpoints used by the frontend pages. This covers:

- risks and risk stats
- policies and policy exceptions
- third-party vendors and questionnaires
- evidence and evidence stats
- maturity, benchmarks, and budget
- obligations, control tests, and control dependencies
- awareness programs and IAM findings
- incident readiness playbooks and escalation rules
- integrations
- control ownership and approvals

### 2. The overview briefing route is still a blocker

`GET /api/v1/cyber/vciso/briefing` did not complete successfully in this environment.

- First matrix run: request timed out
- Longer retry: connection closed by the server after about `49.08s`

That means the top-level overview page should still be treated as not demo-ready until the briefing generator path is stabilized or optimized.

### 3. Default list responses are paginated

Most list endpoints returned `25` rows in the smoke pass, which appears to be the default page size. The stats/aggregate endpoints confirm that the seeded corpus behind those pages is materially larger than the first page returned by the API.

## Practical Demo Readiness

Ready for demo:

- risk register
- policies
- third-party
- evidence
- maturity
- compliance
- awareness
- incident readiness
- integrations
- workflows

Not yet ready for clean demo:

- vCISO overview briefing page

## Recommended Next Step

Inspect and fix the briefing path used by:

- `backend/internal/cyber/vciso/briefing.go`
- `backend/internal/cyber/service/vciso_service.go`

After that, run a real browser pass on the same pages to validate rendering, loading states, and action dialogs on top of the now-verified API data paths.
