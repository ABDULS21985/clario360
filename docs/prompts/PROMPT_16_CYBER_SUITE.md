# PROMPT 16 — Cyber Suite: Full CRUD, Actions & Detail Views

## Objective

Complete the Cyber Suite frontend to cover **every** backend endpoint. The current implementation has list views and dashboards but is missing CRUD operations, action workflows (assign, escalate, merge alerts; create/approve/execute remediation), detail views, and several sub-feature UIs.

## Reference Architecture

- **Stack**: Next.js 14 App Router, TypeScript, Zustand, React Query, Zod, react-hook-form, Tailwind, shadcn/ui, Recharts
- **Existing types**: `frontend/src/types/cyber.ts` (already comprehensive — use as-is, extend only if needed)
- **API helpers**: `apiGet`, `apiPost`, `apiPut`, `apiPatch`, `apiDelete` from `lib/api.ts`
- **Suite helpers**: `fetchSuitePaginated`, `fetchSuiteData` from `lib/suite-api.ts`
- **Data table**: `useDataTable` hook + `DataTable` component from `components/shared/data-table/`
- **Detail page pattern**: `useQuery` + Tabs + action buttons (see `cyber/alerts/[id]/page.tsx`)
- **Permissions**: Wrap pages with `<PermissionRedirect permission="cyber:read">` (or `cyber:write` for mutations)
- **WebSocket topics for invalidation**: `alert.created`, `alert.updated`, `asset.updated`, `remediation.updated`, `threat.updated`, `rule.updated`, `ctem.updated`

## Constants to Add (`lib/constants.ts`)

```typescript
// Cyber — Dashboard (missing)
CYBER_DASHBOARD_MTTR: '/api/v1/cyber/dashboard/mttr',
CYBER_DASHBOARD_TRENDS: '/api/v1/cyber/dashboard/trends',

// Cyber — Alerts (missing)
CYBER_ALERTS_RELATED: '/api/v1/cyber/alerts', // /{id}/related

// Cyber — Vulnerabilities (missing)
CYBER_VULNERABILITIES: '/api/v1/cyber/vulnerabilities',
CYBER_VULNERABILITIES_STATS: '/api/v1/cyber/vulnerabilities/stats',
CYBER_VULNERABILITIES_TOP_CVES: '/api/v1/cyber/vulnerabilities/top-cves',

// Cyber — Threats (missing)
CYBER_THREATS_STATS: '/api/v1/cyber/threats/stats',
CYBER_INDICATORS: '/api/v1/cyber/indicators',
CYBER_INDICATORS_BULK: '/api/v1/cyber/indicators/bulk',

// Cyber — Rules (missing)
CYBER_RULES_STATS: '/api/v1/cyber/rules/stats',

// Cyber — CTEM (missing)
CYBER_CTEM_DASHBOARD: '/api/v1/cyber/ctem/dashboard',

// Cyber — DSPM (missing)
CYBER_DSPM_SCANS: '/api/v1/cyber/dspm/scans',
CYBER_DSPM_CLASSIFICATION: '/api/v1/cyber/dspm/classification',
CYBER_DSPM_EXPOSURE: '/api/v1/cyber/dspm/exposure',
CYBER_DSPM_DEPENDENCIES: '/api/v1/cyber/dspm/dependencies',

// Cyber — Risk (missing)
CYBER_RISK_TREND: '/api/v1/cyber/risk/score/trend',
CYBER_RISK_RECALCULATE: '/api/v1/cyber/risk/score/recalculate',
CYBER_RISK_TOP_RISKS: '/api/v1/cyber/risk/top-risks',
CYBER_RISK_RECOMMENDATIONS: '/api/v1/cyber/risk/recommendations',

// Cyber — vCISO (missing)
CYBER_VCISO_BRIEFING_HISTORY: '/api/v1/cyber/vciso/briefing/history',
CYBER_VCISO_RECOMMENDATIONS: '/api/v1/cyber/vciso/recommendations',
CYBER_VCISO_POSTURE_SUMMARY: '/api/v1/cyber/vciso/posture-summary',
```

## Routes to Add (`lib/constants.ts` → `ROUTES`)

```typescript
CYBER_VULNERABILITIES: '/cyber/vulnerabilities',
CYBER_THREATS_DETAIL: '/cyber/threats', // /{id}
```

---

## PART A — Alert Actions & Detail Enhancements

### A1. Alert Detail Page Enhancements (`cyber/alerts/[id]/page.tsx`)

Add action buttons and missing tabs to the existing alert detail page:

**Action Buttons (top-right):**
- **Assign** → `PUT /api/v1/cyber/alerts/{id}/assign` — Dialog with user combobox
- **Escalate** → `POST /api/v1/cyber/alerts/{id}/escalate` — Confirm dialog with reason textarea
- **Change Status** → `PUT /api/v1/cyber/alerts/{id}/status` — Dropdown with all `AlertStatus` values
- **Mark False Positive** → Status change to `false_positive` with reason
- **Merge** → `POST /api/v1/cyber/alerts/{id}/merge` — Multi-select of related alerts to merge into

**New Tabs:**
- **Related Alerts** tab → `GET /api/v1/cyber/alerts/{id}/related` — DataTable with link to each
- **Evidence** tab → render `alert.explanation.evidence[]` as structured cards
- **Recommended Actions** tab → render `alert.explanation.recommended_actions[]`

**Components to create:**
```
components/cyber/alerts/
  alert-assign-dialog.tsx        — User combobox, submit to PUT assign
  alert-escalate-dialog.tsx      — Reason textarea, submit to POST escalate
  alert-status-dropdown.tsx      — Status change with confirmation
  alert-merge-dialog.tsx         — Search + select related alerts, POST merge
  alert-related-table.tsx        — DataTable of related alerts
  alert-evidence-cards.tsx       — Structured evidence display
```

### A2. Alert Bulk Actions (alerts list page)

Add to the existing alerts list page:
- Multi-select rows (checkbox column)
- Bulk action toolbar: Assign, Change Status, Export
- `POST /api/v1/cyber/alerts/bulk` if the backend supports it, otherwise iterate

---

## PART B — Vulnerability Management (New Section)

### B1. Vulnerabilities Page (`cyber/vulnerabilities/page.tsx`) — NEW PAGE

Create a new page at `app/(dashboard)/cyber/vulnerabilities/page.tsx`.

**DataTable columns:** id, title, CVE ID, severity, CVSS score, status, asset name, age (days), has exploit, detected_at
**Filters:** severity, status, has_exploit (boolean), asset_type, date range
**Stats bar** at top → `GET /api/v1/cyber/vulnerabilities/stats`
**Top CVEs sidebar** → `GET /api/v1/cyber/vulnerabilities/top-cves`
**Aging chart** → `GET /api/v1/cyber/vulnerabilities/aging` (bar chart by age bucket)

**Actions per row:**
- **Update Status** → `PUT /api/v1/cyber/vulnerabilities/{id}/status`
- **View Asset** → Link to `/cyber/assets/{asset_id}`
- **Create Remediation** → Link to remediation create with pre-filled vulnerability_id

### B2. Vulnerability Detail (`cyber/vulnerabilities/[id]/page.tsx`) — NEW PAGE

Tabs: Overview, Remediation, Asset Info, Timeline
- Overview: CVE details, CVSS vector breakdown, description, proof
- Remediation: suggested fix from `remediation` field, link to create remediation action
- Asset Info: linked asset details
- Status change action button

**Components:**
```
components/cyber/vulnerabilities/
  vulnerability-columns.tsx
  vulnerability-stats-bar.tsx
  vulnerability-status-dialog.tsx
  vulnerability-detail-overview.tsx
  vulnerability-aging-chart.tsx
  top-cves-panel.tsx
```

---

## PART C — Remediation Workflow (Full Implementation)

### C1. Remediation List Page Enhancements

Enhance existing `cyber/remediation/page.tsx`:
- Add **Create Remediation** button → opens create dialog
- Add stats bar from `GET /api/v1/cyber/remediation/stats`
- Add status filter tabs (draft, pending_approval, approved, executing, verified, closed)
- Add severity/type filters

### C2. Remediation Detail Page Enhancements (`cyber/remediation/[id]/page.tsx`)

**Tabs:**
1. **Overview** — Title, description, type, severity, affected assets, plan steps
2. **Plan** — Render `plan.steps[]` as numbered checklist, show rollback steps, risk level, estimated downtime
3. **Dry Run** — Show dry run results (`simulated_changes`, `warnings`, `blockers`, `estimated_impact`); button to trigger dry run
4. **Execution** — Show execution results (`step_results[]`, `changes_applied[]`); button to execute
5. **Verification** — Show verification checks; button to verify
6. **Audit Trail** → `GET /api/v1/cyber/remediation/{id}/audit-trail` — Timeline of all actions

**Action Buttons (context-dependent on status):**
| Current Status | Available Actions |
|---|---|
| `draft` | Edit, Submit for Approval, Delete |
| `pending_approval` | Approve, Reject, Request Revision |
| `approved` | Start Dry Run |
| `dry_run_completed` | Execute, View Dry Run Results |
| `executed` | Verify |
| `verified` | Close |
| `executed` / `verified` | Rollback (within deadline) |
| Any | View Audit Trail |

**API calls:**
```
POST   /api/v1/cyber/remediation                          — Create
PUT    /api/v1/cyber/remediation/{id}                     — Update
DELETE /api/v1/cyber/remediation/{id}                     — Delete
POST   /api/v1/cyber/remediation/{id}/submit              — Submit for approval
POST   /api/v1/cyber/remediation/{id}/approve             — Approve
POST   /api/v1/cyber/remediation/{id}/reject              — Reject
POST   /api/v1/cyber/remediation/{id}/request-revision    — Request changes
POST   /api/v1/cyber/remediation/{id}/dry-run             — Start dry run
GET    /api/v1/cyber/remediation/{id}/dry-run             — Get dry run result
POST   /api/v1/cyber/remediation/{id}/execute             — Execute
POST   /api/v1/cyber/remediation/{id}/verify              — Verify
POST   /api/v1/cyber/remediation/{id}/rollback            — Rollback
POST   /api/v1/cyber/remediation/{id}/close               — Close
GET    /api/v1/cyber/remediation/{id}/audit-trail         — Audit trail
```

### C3. Create Remediation Dialog/Page

Form fields:
- Title (text, required)
- Description (textarea)
- Type (select: patch, config_change, isolation, access_control, upgrade, custom)
- Severity (select: critical/high/medium/low)
- Alert ID (optional, combobox)
- Vulnerability ID (optional, combobox)
- Affected Asset IDs (multi-select)
- Execution Mode (select: manual, semi_auto, auto)
- Plan Steps (dynamic array of { action, description, target, expected })
- Requires Approval From (user combobox)

**Components:**
```
components/cyber/remediation/
  remediation-columns.tsx
  remediation-create-dialog.tsx
  remediation-edit-form.tsx
  remediation-status-actions.tsx     — Context-dependent action buttons
  remediation-plan-viewer.tsx        — Render plan steps
  remediation-dry-run-results.tsx    — Simulated changes, warnings, blockers
  remediation-execution-results.tsx  — Step results, applied changes
  remediation-verification.tsx       — Verification checks
  remediation-audit-trail.tsx        — Timeline of actions
  remediation-stats-bar.tsx          — Stats from /stats endpoint
```

---

## PART D — Detection Rules CRUD

### D1. Rules Page Enhancements

Enhance existing `cyber/rules/page.tsx`:
- Add **Create Rule** button → dialog or page
- Add **Edit** and **Delete** actions per row
- Add **Feedback** action (thumbs up/down for ML training) → `POST /api/v1/cyber/rules/{id}/feedback`
- Stats bar from rules stats

### D2. Rule Create/Edit Dialog

Form fields based on rule type:
- Name, Description, Type (sigma/threshold/correlation/anomaly), Severity
- MITRE Technique IDs (multi-select from tactics endpoint)
- Tags
- Rule Content (dynamic form based on type):
  - **Sigma**: Selections (conditions), Filters, Condition expression, Timeframe, Threshold
  - **Threshold**: Filter conditions, Group by, Metric, Threshold value, Window
  - **Anomaly**: Metric, Group by, Window, Z-score threshold, Direction
  - **Correlation**: Event types, Sequence, Group by, Window

**API calls:**
```
POST   /api/v1/cyber/rules         — Create rule
PUT    /api/v1/cyber/rules/{id}    — Update rule
DELETE /api/v1/cyber/rules/{id}    — Delete rule
POST   /api/v1/cyber/rules/{id}/feedback — Submit feedback { is_true_positive: bool }
```

**Components:**
```
components/cyber/rules/
  rule-create-dialog.tsx
  rule-edit-form.tsx
  rule-type-config.tsx          — Dynamic form per rule type
  rule-feedback-button.tsx      — TP/FP feedback
  rule-test-dialog.tsx          — Test rule results display
```

---

## PART E — Threat Intelligence CRUD

### E1. Threats Page Enhancements

Enhance existing `cyber/threats/page.tsx`:
- Add actions per row: **Update Status** → `PUT /api/v1/cyber/threats/{id}/status`

### E2. Threat Detail Page (`cyber/threats/[id]/page.tsx`) — NEW PAGE

Tabs:
1. **Overview** — Name, type, severity, status, description, first/last seen
2. **Indicators** → `GET /api/v1/cyber/threats/{id}/indicators` — DataTable with type, value, severity, confidence, first/last seen; **Add Indicator** button → `POST /api/v1/cyber/threats/{id}/indicators`
3. **Affected Assets** — List of affected assets

**Bulk Indicator Import**: Button → `POST /api/v1/cyber/indicators/bulk` — File upload or paste JSON

### E3. Indicators Page (optional standalone)

`GET /api/v1/cyber/indicators` — Full indicator list with search, filter by type/severity/source

**Components:**
```
components/cyber/threats/
  threat-columns.tsx (update)
  threat-status-dialog.tsx
  threat-detail-overview.tsx
  threat-indicators-table.tsx
  indicator-add-dialog.tsx
  indicator-bulk-import-dialog.tsx
```

---

## PART F — CTEM Assessment Detail Enhancements

### F1. CTEM Assessment Detail Page (`cyber/ctem/[id]/page.tsx`)

The page exists but needs full phase workflow implementation.

**Phase Tabs/Stepper:**
1. **Scoping** → `GET /api/v1/cyber/ctem/assessments/{id}/scope`
2. **Discovery** → `POST /api/v1/cyber/ctem/assessments/{id}/start` then `GET .../discovery`
3. **Prioritization** → `GET .../priorities`
4. **Validation** → `POST .../validate` then `GET .../validation`
5. **Mobilization** → `POST .../mobilize` then `GET .../mobilization`

**Additional tabs:**
- **Findings** → `GET .../findings` — DataTable with severity, priority_score, status, asset; status update per finding
- **Remediation Groups** → `GET .../remediation-groups` — Group list with status update & execute
- **Report** → `GET .../report` — Rendered report; **Export** → `POST .../report/export`; **Executive Summary** → `GET .../report/executive`

**Actions:**
- Start Assessment → `POST .../start`
- Cancel → `POST .../cancel`
- Compare → `GET .../compare/{otherId}` — Side-by-side comparison dialog

**CTEM List page enhancements:**
- Add **Create Assessment** → `POST /api/v1/cyber/ctem/assessments` — Form with name, description, scope config
- Add **Edit/Delete** per row

**Components:**
```
components/cyber/ctem/
  ctem-phase-stepper.tsx
  ctem-scope-viewer.tsx
  ctem-findings-table.tsx
  ctem-finding-status-dialog.tsx
  ctem-remediation-groups.tsx
  ctem-report-viewer.tsx
  ctem-compare-dialog.tsx
  ctem-create-dialog.tsx
```

---

## PART G — DSPM Enhancements

### G1. DSPM Page Enhancements

Add to existing `cyber/dspm/page.tsx`:
- **Data Asset Detail** → Click row → slide-out detail panel or link to `cyber/dspm/[id]`
- **Classification breakdown** → `GET /api/v1/cyber/dspm/classification` — Pie chart
- **Exposure report** → `GET /api/v1/cyber/dspm/exposure` — Risk cards
- **Dependencies** → `GET /api/v1/cyber/dspm/dependencies` — Graph or list view
- **Scan history** → `GET /api/v1/cyber/dspm/scans` — Table with scan details

---

## PART H — Risk & vCISO Enhancements

### H1. Risk Heatmap Page Enhancements

Add to existing `cyber/risk-heatmap/page.tsx`:
- **Risk Score Trend** → `GET /api/v1/cyber/risk/score/trend` — Line chart
- **Top Risks** → `GET /api/v1/cyber/risk/top-risks` — Ranked card list
- **Recommendations** → `GET /api/v1/cyber/risk/recommendations` — Actionable list with priority, effort, estimated reduction
- **Recalculate** button → `GET /api/v1/cyber/risk/score/recalculate`

### H2. vCISO Page Enhancements

Add to existing `cyber/vciso/page.tsx`:
- **Briefing History** → `GET /api/v1/cyber/vciso/briefing/history` — Timeline of past briefings
- **Recommendations** → `GET /api/v1/cyber/vciso/recommendations` — Prioritized list
- **Posture Summary** → `GET /api/v1/cyber/vciso/posture-summary` — Component scores chart
- **Generate Report** → `POST /api/v1/cyber/vciso/report` — Download/view generated report

---

## PART I — Asset CRUD

Add to existing `cyber/assets/page.tsx` and detail page:
- **Create Asset** → `POST /api/v1/cyber/assets` — Form with all CyberAsset fields
- **Edit Asset** → `PUT /api/v1/cyber/assets/{id}` — Edit dialog/page
- **Delete Asset** → `DELETE /api/v1/cyber/assets/{id}` — Confirm dialog
- **Update Tags** → `PATCH /api/v1/cyber/assets/{id}/tags`
- **Add Relationship** → `POST /api/v1/cyber/assets/{id}/relationships` — Dialog with target asset combobox, relationship type
- **Remove Relationship** → `DELETE /api/v1/cyber/assets/{id}/relationships/{relationshipId}`

---

## PART J — Dashboard Enhancements

Add to existing `cyber/page.tsx`:
- **MTTR card** → `GET /api/v1/cyber/dashboard/mttr`
- **Trend Analysis** → `GET /api/v1/cyber/dashboard/trends` — Multi-series line chart

---

## Sidebar Navigation

Add to the Cyber section in the sidebar:
```
Cyber
├── Dashboard          (existing)
├── Alerts             (existing)
├── Vulnerabilities    (NEW)
├── Assets             (existing)
├── Threats            (existing)
├── Detection Rules    (existing)
├── Remediation        (existing)
├── CTEM               (existing)
├── DSPM               (existing)
├── MITRE ATT&CK       (existing)
├── Risk Heatmap       (existing)
└── vCISO              (existing)
```

---

## Testing Requirements

- All new components must have Vitest + React Testing Library tests
- Use MSW handlers for API mocking
- Test CRUD flows: create → list shows new item → edit → delete
- Test action workflows: submit → approve → execute → verify for remediation
- Test error states: API errors show toast, form validation errors show inline
- Test loading states: skeletons render while fetching
- Test permission gates: pages redirect without `cyber:write` for mutation actions

## Validation Checklist

- [ ] `npm run build` passes with zero errors (TypeScript clean)
- [ ] All new pages render without hydration errors
- [ ] Every backend endpoint under `/api/v1/cyber/*` has a corresponding frontend call
- [ ] DataTable pagination, sorting, filtering work on all list pages
- [ ] Real-time updates via WebSocket invalidate relevant queries
- [ ] All forms validate with Zod before submission
- [ ] All dialogs have loading states during API calls
- [ ] Mobile responsive (sidebar collapses, tables scroll horizontally)
