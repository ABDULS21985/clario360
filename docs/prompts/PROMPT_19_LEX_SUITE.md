# PROMPT 19 — Lex Suite: Contract Lifecycle, Clause Analysis, Legal Compliance

## Objective

Complete the Lex (Legal) Suite frontend. Currently only basic list pages exist for contracts and documents. The backend provides full contract lifecycle management (CRUD, upload, analysis, renewal, review), clause-level risk analysis, legal document management, compliance rule engines, and workflow integration — none of which are implemented in the frontend.

## Reference Architecture

- **Stack**: Next.js 14 App Router, TypeScript, Zustand, React Query, Zod, react-hook-form, Tailwind, shadcn/ui, Recharts
- **Existing types**: `types/suites.ts` — already has comprehensive Lex types (use as-is):
  - `LexContractRecord`, `LexContractSummary`, `LexContractVersion`, `LexContractDetail`
  - `LexClause`, `LexClauseType`, `LexClauseReviewStatus`
  - `LexContractRiskAnalysis`, `LexRiskFinding`, `LexComplianceFlag`
  - `LexDocument`, `LexDocumentVersion`
  - `LexComplianceRule`, `LexComplianceAlert`, `LexComplianceDashboard`, `LexComplianceScore`
  - `LexDashboard`, `LexDashboardKPIs`
  - `LexWorkflowSummary`
  - `LexExpiringContractSummary`, `LexContractRiskSummary`
- **Permissions**: `<PermissionRedirect permission="lex:read">` (or `lex:write` for mutations)
- **WebSocket topics**: `lex.contract.created`, `lex.contract.updated`, `lex.contract.expiring`, `lex.analysis.completed`, `lex.compliance.alert`, `lex.minutes.approved`

## Constants to Add (`lib/constants.ts`)

```typescript
// Lex — Dashboard
LEX_DASHBOARD: '/api/v1/lex/dashboard',

// Lex — Contracts (extend existing LEX_CONTRACTS)
LEX_CONTRACTS_EXPIRING: '/api/v1/lex/contracts/expiring',
LEX_CONTRACTS_STATS: '/api/v1/lex/contracts/stats',
LEX_CONTRACTS_SEARCH: '/api/v1/lex/contracts/search',

// Lex — Compliance (extend existing)
LEX_COMPLIANCE_RULES: '/api/v1/lex/compliance/rules',
LEX_COMPLIANCE_ALERTS: '/api/v1/lex/compliance/alerts',
LEX_COMPLIANCE_DASHBOARD: '/api/v1/lex/compliance/dashboard',
LEX_COMPLIANCE_SCORE: '/api/v1/lex/compliance/score',
LEX_COMPLIANCE_RUN: '/api/v1/lex/compliance/run',

// Lex — Workflows
LEX_WORKFLOWS: '/api/v1/lex/workflows',
```

---

## PART A — Dashboard (`lex/page.tsx`)

Replace the current basic page with a full data-driven dashboard from `GET /api/v1/lex/dashboard`.

**Layout:**
1. **KPI Cards Row** (from `LexDashboardKPIs`):
   - Active Contracts (count)
   - Expiring in 30 Days (warning badge if > 0)
   - Expiring in 7 Days (critical badge if > 0)
   - High Risk Contracts (red badge)
   - Pending Review (count)
   - Open Compliance Alerts (count)
   - Total Active Value (formatted currency)
   - Compliance Score (gauge)

2. **Two-Column Grid:**
   - Left: **Expiring Contracts** card → `expiring_contracts[]` — Table with title, party, expiry date, days remaining (color-coded)
   - Left: **High Risk Contracts** card → `high_risk_contracts[]` — Table with risk level badge, risk score, party
   - Right: **Contracts by Type** → pie chart from `contracts_by_type`
   - Right: **Contracts by Status** → horizontal bar chart from `contracts_by_status`

3. **Charts Row:**
   - **Monthly Activity** → multi-series bar chart from `monthly_activity[]` (created, activated, expired, renewed)
   - **Total Contract Value** → breakdown charts from `total_contract_value` (by type, by currency)
   - **Compliance Alerts by Status** → donut chart from `compliance_alerts_by_status`

4. **Recent Contracts** → table from `recent_contracts[]`

**Components:**
```
components/lex/dashboard/
  lex-kpi-cards.tsx
  expiring-contracts-card.tsx
  high-risk-contracts-card.tsx
  contracts-by-type-chart.tsx
  contracts-by-status-chart.tsx
  monthly-activity-chart.tsx
  contract-value-breakdown.tsx
  compliance-alerts-chart.tsx
  recent-contracts-table.tsx
```

---

## PART B — Contract Management (Full CRUD + Lifecycle)

### B1. Contracts List Page (`lex/contracts/page.tsx`)

Enhance with:
- **Create Contract** button → wizard dialog
- **Search** → `GET /api/v1/lex/contracts/search?q=...` — Full-text search
- Stats from `GET /api/v1/lex/contracts/stats`
- Expiring filter → `GET /api/v1/lex/contracts/expiring`
- Status filter tabs: draft, internal_review, legal_review, negotiation, pending_signature, active, expired, terminated
- Type filter, risk level filter, date range filter
- Per-row actions: Edit, Analyze, Upload Doc, Change Status, Renew, Delete

### B2. Contract Create Wizard

Multi-step:
1. **Basic Info**: Title, Contract Number (auto-generated option), Type (select from `LexContractType`), Description
2. **Parties**: Party A Name/Entity, Party B Name/Entity/Contact
3. **Financial**: Total Value, Currency (select), Payment Terms
4. **Dates**: Effective Date, Expiry Date, Renewal Date, Auto Renew (toggle), Renewal Notice Days
5. **Assignment**: Owner (user combobox), Legal Reviewer (user combobox), Department, Tags
6. **Document Upload**: Optional file upload → will be stored via file service

**API:** `POST /api/v1/lex/contracts`

### B3. Contract Detail Page (`lex/contracts/[id]/page.tsx`) — MAJOR ENHANCEMENT

Fetch `GET /api/v1/lex/contracts/{id}` which returns `LexContractDetail` (contract, clauses, latest_analysis, version_count).

**Header:**
- Title, contract number, status badge, risk level badge with score
- Party A ↔ Party B
- Key dates: effective, expiry (with countdown), renewal
- Total value (formatted currency)

**Action Buttons (status-dependent):**
| Status | Actions |
|---|---|
| `draft` | Edit, Upload Document, Submit for Review, Delete |
| `internal_review` | Analyze, Change Status (to legal_review), Upload New Version |
| `legal_review` | Analyze, Change Status (to negotiation/pending_signature) |
| `negotiation` | Edit, Change Status |
| `pending_signature` | Change Status (to active) |
| `active` | Renew, Suspend, Upload Amendment |
| `expired` | Renew |
| Any | Start Review → `POST /contracts/{id}/review` (creates workflow) |

**Tabs:**

#### Tab 1: Overview
- Contract details, parties, financial terms, dates
- Risk score gauge with grade
- Analysis status indicator
- Linked workflow status (if any)

#### Tab 2: Analysis & Clauses
- **Trigger Analysis** → `POST /api/v1/lex/contracts/{id}/analyze` — Button with loading state
- **Analysis Results** (from `latest_analysis: LexContractRiskAnalysis`):
  - Overall Risk: level badge + score
  - Key Findings → `key_findings[]` — Cards with severity badge, description, recommendation
  - Missing Clauses → `missing_clauses[]` — Warning list of clause types not found
  - Compliance Flags → `compliance_flags[]` — Alert cards with code, severity, description
  - Extracted Parties → `extracted_parties[]` — Table
  - Extracted Dates → `extracted_dates[]` — Timeline
  - Extracted Amounts → `extracted_amounts[]` — Table
- **Clauses** → `clauses[]` — DataTable:
  - Columns: type, title, section_reference, risk_level, risk_score, review_status, extraction_confidence
  - **Clause Detail** → Expandable row or slide-out with: content text, analysis_summary, recommendations, compliance_flags, risk_keywords
  - **Risk Summary** → `GET /api/v1/lex/contracts/{id}/clauses/risks` — Aggregated risk view
  - **Review Clause** → `PUT /api/v1/lex/contracts/{id}/clauses/{clauseId}/review` — Dialog with:
    - Review Status (select: reviewed, flagged, accepted, rejected)
    - Review Notes (textarea)

#### Tab 3: Versions
- **Version History** → `GET /api/v1/lex/contracts/{id}/versions` — DataTable with version number, file name, file size, change summary, uploaded by, date
- **Upload New Version** → `POST /api/v1/lex/contracts/{id}/upload` — File upload with change summary
- Download link for each version

#### Tab 4: Compliance
- Contract-specific compliance alerts
- Linked compliance rules affecting this contract type

#### Tab 5: Workflow
- If `workflow_instance_id` exists, show workflow status and steps
- Link to workflow detail page

**API calls:**
```
GET    /api/v1/lex/contracts/{id}                    — Get detail (contract + clauses + analysis)
PUT    /api/v1/lex/contracts/{id}                    — Update contract
DELETE /api/v1/lex/contracts/{id}                    — Delete
POST   /api/v1/lex/contracts/{id}/upload             — Upload document
POST   /api/v1/lex/contracts/{id}/analyze            — Trigger analysis
PUT    /api/v1/lex/contracts/{id}/status             — Change status { status: LexContractStatus }
GET    /api/v1/lex/contracts/{id}/versions           — Version history
POST   /api/v1/lex/contracts/{id}/renew              — Renew contract
POST   /api/v1/lex/contracts/{id}/review             — Start review workflow
GET    /api/v1/lex/contracts/{id}/analysis           — Get latest analysis
GET    /api/v1/lex/contracts/{id}/clauses            — List clauses
GET    /api/v1/lex/contracts/{id}/clauses/risks      — Risk summary
GET    /api/v1/lex/contracts/{id}/clauses/{clauseId} — Clause detail
PUT    /api/v1/lex/contracts/{id}/clauses/{clauseId}/review — Review clause
```

**Components:**
```
components/lex/contracts/
  contract-columns.tsx
  contract-create-wizard.tsx
  contract-edit-form.tsx
  contract-overview.tsx
  contract-status-actions.tsx      — Status-dependent action buttons
  contract-risk-gauge.tsx          — Risk score gauge display
  contract-parties-display.tsx
  contract-dates-display.tsx
  contract-value-display.tsx

  analysis/
    analysis-trigger-button.tsx
    analysis-results-panel.tsx
    analysis-findings-cards.tsx     — Key findings with severity
    analysis-missing-clauses.tsx    — Warning list
    analysis-compliance-flags.tsx
    analysis-extracted-data.tsx     — Parties, dates, amounts

  clauses/
    clause-table.tsx
    clause-detail-panel.tsx        — Expandable clause with full content
    clause-review-dialog.tsx       — Review status + notes
    clause-risk-summary.tsx

  versions/
    version-history-table.tsx
    version-upload-dialog.tsx      — File upload with change summary
```

---

## PART C — Legal Document Management (Full CRUD)

### C1. Documents List Page (`lex/documents/page.tsx`)

Enhance with:
- **Create Document** button
- Filters: type, status, confidentiality, category, tags
- Per-row actions: Edit, Upload Version, View Versions, Delete

### C2. Document Create/Edit Dialog

Form fields:
- Title (required)
- Type (select from `LexDocumentType`: policy, regulation, template, memo, opinion, filing, correspondence, resolution, power_of_attorney, other)
- Description (textarea)
- Category (text)
- Confidentiality (select from `LexDocumentConfidentiality`: public, internal, confidential, privileged)
- Contract ID (optional combobox — link to contract)
- Tags
- File Upload → stored via file service

### C3. Document Detail (slide-out or page)

- Document metadata
- **Version History** → `GET /api/v1/lex/documents/{id}/versions` — Table with version, file name, size, change summary, uploaded by
- **Upload New Version** → `POST /api/v1/lex/documents/{id}/upload`
- Download links

**API calls:**
```
POST   /api/v1/lex/documents           — Create
GET    /api/v1/lex/documents           — List
GET    /api/v1/lex/documents/{id}      — Get detail
PUT    /api/v1/lex/documents/{id}      — Update
DELETE /api/v1/lex/documents/{id}      — Delete
POST   /api/v1/lex/documents/{id}/upload — Upload new version
GET    /api/v1/lex/documents/{id}/versions — Version history
```

**Components:**
```
components/lex/documents/
  document-columns.tsx
  document-create-dialog.tsx
  document-edit-form.tsx
  document-detail-panel.tsx
  document-version-table.tsx
  document-upload-dialog.tsx
  document-confidentiality-badge.tsx
```

---

## PART D — Compliance Management (Rules, Alerts, Dashboard)

### D1. Compliance Page (`lex/compliance/page.tsx`)

Full compliance management UI with tabs:

#### Tab 1: Dashboard
- Fetch `GET /api/v1/lex/compliance/dashboard` → `LexComplianceDashboard`
- Compliance Score → `GET /api/v1/lex/compliance/score` — Large gauge
- Rules by Type → donut chart
- Alerts by Status → bar chart
- Alerts by Severity → bar chart
- Open vs Resolved counts
- Contracts in Scope count

#### Tab 2: Rules
- DataTable → `GET /api/v1/lex/compliance/rules`
  - Columns: name, rule_type, severity, contract_types, enabled (toggle), jurisdiction
- **Create Rule** → `POST /api/v1/lex/compliance/rules`
- **Edit Rule** → `PUT /api/v1/lex/compliance/rules/{id}`
- **Delete Rule** → `DELETE /api/v1/lex/compliance/rules/{id}`
- **Run All Checks** → `POST /api/v1/lex/compliance/run` — Returns `LexComplianceRunResult` with score and alerts created

Rule Create/Edit Form:
- Name, Description
- Rule Type (select from `LexComplianceRuleType`: expiry_warning, missing_clause, risk_threshold, review_overdue, unsigned_contract, value_threshold, jurisdiction_check, data_protection_required, custom)
- Severity (select)
- Config (JSON editor or structured form based on rule type)
- Contract Types (multi-select from `LexContractType`)
- Enabled (toggle)
- Jurisdiction (optional text)
- Regulation Reference (optional text)

#### Tab 3: Alerts
- DataTable → `GET /api/v1/lex/compliance/alerts`
  - Columns: title, severity, status, contract link, rule link, created_at
  - **View Detail** → `GET /api/v1/lex/compliance/alerts/{id}` — Slide-out with full description, evidence, resolution info
  - **Update Status** → `PUT /api/v1/lex/compliance/alerts/{id}/status` — Select new status (open, acknowledged, investigating, resolved, dismissed) + optional resolution notes

**Components:**
```
components/lex/compliance/
  compliance-dashboard.tsx
  compliance-score-gauge.tsx
  compliance-rules-table.tsx
  compliance-rule-create-dialog.tsx
  compliance-rule-edit-form.tsx
  compliance-run-button.tsx
  compliance-run-results.tsx
  compliance-alerts-table.tsx
  compliance-alert-detail.tsx
  compliance-alert-status-dialog.tsx
  compliance-charts.tsx
```

---

## PART E — Legal Workflows

### E1. Workflows Section (within lex or link)

- `GET /api/v1/lex/workflows` → `LexWorkflowSummary[]` — Active legal workflows
- DataTable: contract title, contract status, workflow status, current step, assignee, started at
- Click row → navigates to workflow detail (`/workflows/{workflow_instance_id}`)

**Component:** `components/lex/workflows/lex-workflow-table.tsx`

---

## Sidebar Navigation

```
Lex
├── Dashboard          (new — full dashboard)
├── Contracts          (enhance — full CRUD + lifecycle)
├── Documents          (enhance — full CRUD + versioning)
├── Compliance         (enhance — rules, alerts, dashboard)
└── Workflows          (new — legal workflow tracker)
```

---

## Testing Requirements

- Test contract CRUD: create → list → edit → delete
- Test contract analysis: upload → analyze → view clauses → review clause
- Test contract lifecycle: draft → review → negotiate → active → renew
- Test document versioning: upload → new version → view history
- Test compliance: create rule → run checks → view alerts → resolve alert
- Test risk visualization: gauge renders correctly, risk badges color-coded
- Test expiring contracts: countdown display, warning badges
- Test loading, error, empty states on all pages

## Validation Checklist

- [ ] `npm run build` passes with zero errors
- [ ] Every backend endpoint under `/api/v1/lex/*` has a corresponding frontend call
- [ ] Contract status transitions are correctly enforced
- [ ] Risk analysis results render all sections (findings, clauses, flags, extracted data)
- [ ] Clause review workflow works end-to-end
- [ ] Compliance score gauge renders correctly
- [ ] File upload works for contract documents and new versions
- [ ] Currency formatting is correct for contract values
- [ ] Expiry countdown is accurate and color-coded
