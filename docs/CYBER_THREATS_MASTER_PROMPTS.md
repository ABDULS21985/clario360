# Cyber Threat Module — Master Implementation Prompts

## Recon-Aligned to Actual Backend Capabilities

> Every reference maps to an actual file, table, service, handler, or route found in `/Users/mac/clario360/backend/internal/cyber/`.

---

## Codebase Reality Summary (From Reconnaissance)

Before executing any prompt, the implementing agent MUST internalise these non-negotiable truths:

| # | Reality | Evidence |
|---|---------|----------|
| 1 | **Backend is Go + chi v5**, not Express/Gin. Router: `chi.NewRouter()`, middleware: `func(http.Handler) http.Handler`. Auth: RS256 JWT via `auth.UserFromContext()`, `auth.TenantFromContext()` | `internal/cyber/handler/routes.go`, `internal/auth/` |
| 2 | **Frontend is Next.js 14 App Router** with TypeScript, Zustand, react-query, Tailwind, shadcn/ui. Pages at `frontend/src/app/(dashboard)/cyber/`. Shared components: `DataTable`, `PageHeader`, `StatusBadge`, `SeverityIndicator`, `DetailPanel`, `SearchInput`, `useDataTable` | `frontend/src/app/(dashboard)/cyber/threats/`, `frontend/src/components/` |
| 3 | **Database is PostgreSQL** (`cyber_db`). Tables: `threats`, `threat_indicators`, `alerts`, `alert_comments`, `alert_timeline`, `security_events` (partitioned), `detection_rules`. All tenant-scoped with RLS | `migrations/cyber_db/000003_threat_detection_engine.up.sql` |
| 4 | **Threat types**: malware, phishing, apt, ransomware, ddos, insider_threat, supply_chain, zero_day, brute_force, other. **Statuses**: active, contained, eradicated, monitoring, closed. **Severities**: critical, high, medium, low | `internal/cyber/model/threat.go` |
| 5 | **Indicator types**: ip, domain, url, email, file_hash_md5, file_hash_sha1, file_hash_sha256, certificate, registry_key, user_agent, cidr. **Sources**: manual, stix_feed, osint, internal, vendor. Unique on `(tenant_id, type, value)` | `internal/cyber/model/threat.go`, migration |
| 6 | **Alert lifecycle**: new → acknowledged → investigating → in_progress → resolved/closed/false_positive/escalated/merged. Rich `Explanation` JSONB: summary, reason, evidence[], matched_conditions[], confidence_factors[], recommended_actions[], false_positive_indicators[], indicator_matches[] | `internal/cyber/model/alert.go` |
| 7 | **Detection rules**: 4 types — sigma, threshold, correlation, anomaly. Each has mitre_tactic_ids[], mitre_technique_ids[], false_positive_count, true_positive_count. Evaluators: `condition_parser.go`, `threshold_evaluator.go`, `correlation_evaluator.go`, `anomaly_evaluator.go` | `internal/cyber/detection/` |
| 8 | **MITRE ATT&CK** embedded: 14 tactics (TA0043→TA0040), ~50 techniques with sub-techniques, platforms, data sources. Coverage analysis: `mitre/coverage.go` calculates technique coverage across rules/threats | `internal/cyber/mitre/framework.go`, `mapper.go`, `coverage.go` |
| 9 | **STIX 2 parser**: Extracts indicators from STIX pattern language, handles malware/threat-actor/campaign/relationship/indicator objects. Bulk import endpoint exists at `POST /api/v1/cyber/indicators/bulk` | `internal/cyber/indicator/stix_parser.go` |
| 10 | **Enrichment pipeline**: modular `DNSEnricher`, `CVEEnricher`, `GeolocationEnricher`. CTEM engine at `internal/cyber/ctem/`. Predictive engine at `internal/cyber/vciso/predict/` with threat_forecast, campaign_detector, alert_volume_forecaster, attack_technique_trend | `internal/cyber/enrichment/`, `internal/cyber/ctem/`, `internal/cyber/vciso/predict/` |
| 11 | **Frontend patterns**: `useDataTable<T>` for paginated tables, `useApiMutation` for mutations, `apiGet`/`apiPost` for API calls, `API_ENDPOINTS` constants in `lib/constants.ts`, types in `types/cyber.ts`, WebSocket topics via realtime-store | `frontend/src/hooks/`, `frontend/src/lib/`, `frontend/src/types/cyber.ts` |
| 12 | **Existing cyber pages**: threats (list only), alerts (not built), vulnerabilities, assets, sigma rules (basic), UEBA, CTEM overview. Navigation in `config/navigation.ts` under `cyber` group | `frontend/src/app/(dashboard)/cyber/`, `frontend/src/config/navigation.ts` |

---

## Master Prompt 1 of 5: Threat Intelligence Dashboard & Threat Lifecycle Management

```
You are a Senior Full-Stack Engineer enhancing the Cyber Threat module for an
enterprise GRC/SIEM platform (Clario360). The backend is Go + chi, frontend is
Next.js 14 + shadcn/ui + react-query.

## CRITICAL CODEBASE CONTEXT — READ FIRST

Before writing any code, run Phase 0 reconnaissance on these files:

Backend (what already exists):
- `backend/internal/cyber/handler/threat_handler.go` — all threat API handlers
- `backend/internal/cyber/handler/routes.go` — route registration
- `backend/internal/cyber/service/threat_service.go` — threat business logic
- `backend/internal/cyber/repository/threat_repo.go` — threat DB queries
- `backend/internal/cyber/model/threat.go` — Threat, ThreatIndicator, ThreatStats models
- `backend/internal/cyber/model/alert.go` — Alert model (linked to threats)
- `backend/internal/cyber/mitre/framework.go` — embedded MITRE framework

Frontend (what already exists):
- `frontend/src/app/(dashboard)/cyber/threats/page.tsx` — current threats page (list only)
- `frontend/src/app/(dashboard)/cyber/threats/_components/threat-columns.tsx` — column definitions
- `frontend/src/app/(dashboard)/cyber/threats/_components/threat-detail-panel.tsx` — side panel
- `frontend/src/app/(dashboard)/cyber/threats/_components/indicator-check-dialog.tsx` — IOC check
- `frontend/src/types/cyber.ts` — Threat, ThreatIndicator, ThreatLandscape types
- `frontend/src/lib/constants.ts` — API_ENDPOINTS (CYBER_THREATS, CYBER_INDICATORS_CHECK)
- `frontend/src/hooks/use-data-table.ts` — generic paginated table hook
- `frontend/src/components/shared/` — all shared components (kpi-card, severity-indicator, etc.)

The current threats page is READ-ONLY: list + side-panel detail + IOC check dialog.
NO threat creation, NO editing, NO dashboard, NO full detail page.

## WHAT YOU MUST BUILD

### 1. Threat Stats Dashboard Section (Top of Threats Page)

Modify: `frontend/src/app/(dashboard)/cyber/threats/page.tsx`

Add a dashboard section ABOVE the DataTable with:

**Row 1 — KPI Cards** (4 cards using existing `KpiCard` component):
- Active Threats (count, trend vs last 7 days)
- Critical/High Severity (count, red/orange accent)
- IOCs Tracked (total indicator count)
- Threats Contained This Month (count, green accent)

**Row 2 — Charts** (2 charts, 50/50 width):
- Threats by Type: bar chart (malware, phishing, apt, ransomware, etc.)
- Threats by Severity: pie/donut chart (critical=red, high=orange, medium=yellow, low=blue)

**Data source**: `GET /api/v1/cyber/threats/stats` — already exists in backend
(ThreatStats returns ByType, ByStatus, BySeverity, Total, Active counts)

Add a new API endpoint to backend for trend data if not available:
`GET /api/v1/cyber/threats/stats/trend` — returns daily threat counts for last 30 days

### 2. Threat Creation Dialog

New file: `frontend/src/app/(dashboard)/cyber/threats/_components/create-threat-dialog.tsx`

Add backend endpoint: `POST /api/v1/cyber/threats`
- Handler: `threat_handler.go` → `CreateThreat()`
- Service: `threat_service.go` → `CreateThreat(ctx, input)`
- Validation: name required, type must be valid enum, severity must be valid enum

Dialog form fields:
- **Name** (required, text input)
- **Type** (required, select: malware, phishing, apt, ransomware, ddos, insider_threat,
  supply_chain, zero_day, brute_force, other)
- **Severity** (required, select: critical, high, medium, low)
- **Description** (textarea)
- **Threat Actor** (optional text — APT group name)
- **Campaign** (optional text — campaign identifier)
- **MITRE Tactics** (multi-select from embedded framework: 14 tactics)
- **MITRE Techniques** (multi-select, filtered by selected tactics)
- **Tags** (tag input — comma separated)
- **Initial Indicators** (optional, repeatable group):
  - Type (select: ip, domain, url, email, file_hash_md5, file_hash_sha256, etc.)
  - Value (text input)
  - Severity (select)
  - Confidence (slider 0-100)

Button in PageHeader: "+ New Threat" alongside existing "Check Indicators"

### 3. Full Threat Detail Page (Replaces Side Panel for Deep Dive)

New files:
- `frontend/src/app/(dashboard)/cyber/threats/[threatId]/page.tsx`
- `frontend/src/app/(dashboard)/cyber/threats/[threatId]/_components/threat-overview.tsx`
- `frontend/src/app/(dashboard)/cyber/threats/[threatId]/_components/threat-indicators-tab.tsx`
- `frontend/src/app/(dashboard)/cyber/threats/[threatId]/_components/threat-alerts-tab.tsx`
- `frontend/src/app/(dashboard)/cyber/threats/[threatId]/_components/threat-timeline-tab.tsx`
- `frontend/src/app/(dashboard)/cyber/threats/[threatId]/_components/threat-mitre-tab.tsx`

**API**: `GET /api/v1/cyber/threats/{id}` — already exists, returns threat with indicators

**Tabbed layout** (using existing `Tabs` component from shadcn/ui):

**Tab 1 — Overview**:
- Threat header: name, type badge, severity indicator, status badge, threat actor, campaign
- Description (full text, markdown rendered)
- Key metrics row: indicator count, affected assets, linked alerts, days active
- First seen / Last seen / Contained at timeline
- Tags display
- Actions: Update Status (dropdown), Edit Threat, Delete Threat

**Tab 2 — Indicators (IOCs)**:
- DataTable of indicators for this threat
- Columns: type (badge), value (mono), severity, source, confidence (progress bar),
  first_seen, last_seen, active (toggle), expires_at
- Add Indicator button → inline form or dialog
- Bulk actions: deactivate, export CSV
- API: `GET /api/v1/cyber/threats/{id}/indicators` (exists)
- API: `POST /api/v1/cyber/threats/{id}/indicators` (exists)

**Tab 3 — Related Alerts**:
- DataTable of alerts linked to this threat (via threat_id or indicator matches)
- Columns: title, severity, status, confidence, asset, MITRE technique, created_at
- Click → navigate to alert detail (Prompt 2)
- API needed: `GET /api/v1/cyber/threats/{id}/alerts`

**Tab 4 — Activity Timeline**:
- Chronological timeline (using existing `Timeline` component) of:
  - Status changes (active → contained → eradicated)
  - New indicators added
  - New alerts correlated
  - Affected assets discovered
- Data from audit events + alert timeline

**Tab 5 — MITRE Mapping**:
- Visual display of which ATT&CK tactics/techniques this threat uses
- Mini ATT&CK matrix with highlighted cells
- Technique cards with descriptions from embedded framework

### 4. Threat Status Update

Add backend endpoint if not exists: `PUT /api/v1/cyber/threats/{id}/status` — EXISTS already
Add backend endpoint: `PUT /api/v1/cyber/threats/{id}` — for editing threat details

Frontend: Status update dropdown on detail page header with confirmation dialog:
- active → contained / monitoring
- contained → eradicated / active (reopened)
- monitoring → closed / active (reopened)
- eradicated → closed

### 5. Threat Types & API Endpoints

Add to `frontend/src/types/cyber.ts`:
```typescript
export interface ThreatStats {
  total: number;
  active: number;
  by_type: NamedCount[];
  by_status: NamedCount[];
  by_severity: NamedCount[];
}

export interface NamedCount {
  name: string;
  count: number;
}

export interface CreateThreatInput {
  name: string;
  type: ThreatType;
  severity: CyberSeverity;
  description?: string;
  threat_actor?: string;
  campaign?: string;
  mitre_tactic_ids?: string[];
  mitre_technique_ids?: string[];
  tags?: string[];
  indicators?: CreateIndicatorInput[];
}

export interface CreateIndicatorInput {
  type: IndicatorType;
  value: string;
  severity: CyberSeverity;
  confidence: number;
  source?: string;
  description?: string;
  tags?: string[];
}

export type ThreatType = 'malware' | 'phishing' | 'apt' | 'ransomware' | 'ddos' |
  'insider_threat' | 'supply_chain' | 'zero_day' | 'brute_force' | 'other';

export type IndicatorType = 'ip' | 'domain' | 'url' | 'email' | 'file_hash_md5' |
  'file_hash_sha1' | 'file_hash_sha256' | 'certificate' | 'registry_key' |
  'user_agent' | 'cidr';
```

Add to `frontend/src/lib/constants.ts`:
```typescript
CYBER_THREAT_STATS: '/api/v1/cyber/threats/stats',
CYBER_THREAT_DETAIL: (id: string) => `/api/v1/cyber/threats/${id}`,
CYBER_THREAT_STATUS: (id: string) => `/api/v1/cyber/threats/${id}/status`,
CYBER_THREAT_INDICATORS: (id: string) => `/api/v1/cyber/threats/${id}/indicators`,
CYBER_THREAT_ALERTS: (id: string) => `/api/v1/cyber/threats/${id}/alerts`,
```

### 6. Navigation Update

Modify `frontend/src/config/navigation.ts`:
- Keep "Threat Hunting" for `/cyber/threats` (the enhanced page with dashboard + list)

### GOVERNING RULES
1. Phase 0 reconnaissance FIRST — read ALL listed files before writing code
2. Backend: Go + chi v5, PostgreSQL, `auth.TenantFromContext()` for tenant scoping
3. Frontend: Next.js 14 App Router (`params` is plain object, NOT Promise)
4. Reuse existing components: KpiCard, SeverityIndicator, StatusBadge, DataTable, Timeline,
   DetailPanel, SearchInput, BarChart, PieChart from `components/shared/`
5. Reuse existing hooks: useDataTable, useApiMutation from `hooks/`
6. All API calls through `apiGet`/`apiPost`/`apiPut` from `lib/api.ts`
7. Permission: `cyber:read` for viewing, `cyber:write` for create/edit/status change
8. WebSocket topics: `threat.detected`, `threat.updated` for real-time
9. All Go commands require `GOWORK=off` prefix
10. Build verification: `cd frontend && npm run build` must pass clean
```

---

## Master Prompt 2 of 5: Alert Management & Investigation Workspace

```
You are a Senior Full-Stack Engineer building the Alert Management and
Investigation Workspace for Clario360's Cyber module.

## CRITICAL CODEBASE CONTEXT — READ FIRST

Before writing any code, run Phase 0 reconnaissance on:

Backend (fully implemented, no frontend):
- `backend/internal/cyber/model/alert.go` — Alert model with 12 statuses,
  Explanation JSONB, ConfidenceScore, IndicatorMatches, AlertComment, AlertTimeline
- `backend/internal/cyber/handler/alert_handler.go` — alert API handlers
- `backend/internal/cyber/handler/routes.go` — route registration for alerts
- `backend/internal/cyber/service/alert_service.go` — alert business logic
- `backend/internal/cyber/repository/alert_repo.go` — alert DB queries
- `backend/internal/cyber/detection/engine.go` — how alerts are generated
- `backend/internal/cyber/model/explanation/builder.go` — alert explanation structure

Frontend (reference patterns):
- `frontend/src/app/(dashboard)/cyber/threats/` — current threat page pattern
- `frontend/src/types/cyber.ts` — existing Alert type (check if complete)
- `frontend/src/hooks/use-data-table.ts` — paginated table hook
- `frontend/src/components/shared/timeline.tsx` — timeline component
- `frontend/src/components/shared/detail-panel.tsx` — slide-out panel

The backend has FULL alert lifecycle support. The frontend has NOTHING for alerts.

## WHAT YOU MUST BUILD

### 1. Alert List Page

New files:
- `frontend/src/app/(dashboard)/cyber/alerts/page.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/_components/alert-columns.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/_components/alert-filters.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/_components/alert-stats-bar.tsx`

**Route**: `/cyber/alerts`
**Permission**: `cyber:read`

**Stats Bar** (top of page, horizontal cards):
- New Alerts (count, pulsing red dot if > 0)
- Investigating (count)
- False Positive Rate (percentage)
- Mean Time to Acknowledge (hours)
- Mean Time to Resolve (hours)

**DataTable columns**:
- Severity (SeverityIndicator)
- Alert Title (clickable → detail page)
- Status (StatusBadge with colour: new=red, acknowledged=blue, investigating=yellow,
  in_progress=orange, resolved=green, closed=gray, false_positive=purple,
  escalated=red-outline, merged=gray-outline)
- Confidence Score (progress bar 0-100%)
- MITRE Technique (badge with ID)
- Asset (name or IP)
- Rule (detection rule name)
- Created At (RelativeTime)
- Actions (acknowledge, assign, escalate)

**Filters**:
- Severity multi-select (critical, high, medium, low)
- Status multi-select (new, acknowledged, investigating, in_progress, resolved,
  closed, false_positive, escalated, merged)
- MITRE Tactic (select from 14 tactics)
- Confidence range (slider min-max)
- Date range (date picker)
- Rule type (sigma, threshold, correlation, anomaly)
- Search (full-text on title/description)

**Bulk Actions**:
- Acknowledge selected
- Assign to analyst
- Mark as false positive
- Merge selected alerts

**Real-time**: Subscribe to WebSocket topics `alert.created`, `alert.updated`

### 2. Alert Detail Page

New files:
- `frontend/src/app/(dashboard)/cyber/alerts/[alertId]/page.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/[alertId]/_components/alert-header.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/[alertId]/_components/alert-explanation.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/[alertId]/_components/alert-evidence.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/[alertId]/_components/alert-comments.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/[alertId]/_components/alert-timeline.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/[alertId]/_components/alert-related.tsx`
- `frontend/src/app/(dashboard)/cyber/alerts/[alertId]/_components/alert-actions.tsx`

**Route**: `/cyber/alerts/{alertId}`
**API**: Needs `GET /api/v1/cyber/alerts/{id}` (check if exists, add if not)

**Header Section**:
- Alert title, severity badge, status badge, confidence gauge (circular)
- MITRE technique badge with tactic category
- Affected asset link
- Assigned analyst (avatar + name)
- Action buttons: Acknowledge, Escalate, Resolve, Mark False Positive

**AI Explanation Card** (from Alert.Explanation JSONB):
- **Summary**: Plain language description of what triggered the alert
- **Reason**: Why this is considered malicious/suspicious
- **Evidence**: List of evidence items with source references
- **Matched Conditions**: What rule conditions matched
- **Confidence Factors**: What increased/decreased confidence
- **Recommended Actions**: AI-suggested next steps for the analyst
- **False Positive Indicators**: Signs that this might be benign

**Evidence Tab**:
- Raw security event(s) that triggered the alert (formatted JSON/table)
- IOC matches highlighted (which indicators matched, from which threat)
- Asset context (OS, IP, hostname, owner, criticality)
- Network context (source/dest IPs, ports, protocols)

**Investigation Comments**:
- Threaded comment system (existing `alert_comments` table)
- Add comment with rich text
- Comment history with timestamps and analyst avatars
- `POST /api/v1/cyber/alerts/{id}/comments`
- `GET /api/v1/cyber/alerts/{id}/comments`

**Alert Timeline** (using Timeline component):
- Chronological log of all status changes, assignments, comments, escalations
- Uses existing `alert_timeline` table
- `GET /api/v1/cyber/alerts/{id}/timeline`

**Related Alerts**:
- Other alerts from the same rule
- Other alerts on the same asset
- Other alerts linked to the same threat
- Correlated alerts (same MITRE technique within time window)

### 3. Alert Status Transitions

Backend API: `PUT /api/v1/cyber/alerts/{id}/status` (add if not exists)
Backend API: `PUT /api/v1/cyber/alerts/{id}/assign` (add if not exists)
Backend API: `POST /api/v1/cyber/alerts/{id}/escalate` (add if not exists)
Backend API: `PUT /api/v1/cyber/alerts/{id}/false-positive` (with reason field)

Frontend: Status transition buttons with confirmation dialogs:
- **Acknowledge**: new → acknowledged (one-click, sets analyst)
- **Start Investigation**: acknowledged → investigating
- **Escalate**: any → escalated (requires reason text)
- **Resolve**: investigating → resolved (requires resolution summary)
- **Close**: resolved → closed
- **False Positive**: any → false_positive (requires reason, updates rule FP count)
- **Reopen**: closed/false_positive/resolved → investigating

### 4. Alert Stats API

Add backend endpoint: `GET /api/v1/cyber/alerts/stats`
Returns:
```json
{
  "total": 1250,
  "by_status": [{"name": "new", "count": 45}, ...],
  "by_severity": [{"name": "critical", "count": 12}, ...],
  "by_rule_type": [{"name": "sigma", "count": 800}, ...],
  "mttr_hours": 4.5,
  "mtta_hours": 0.8,
  "false_positive_rate": 0.12
}
```

### 5. Navigation Update

Add to `frontend/src/config/navigation.ts` under cyber group:
```typescript
{ id: 'cyber-alerts', label: 'Alerts', href: '/cyber/alerts', icon: Bell }
```

Position ABOVE "Threat Hunting" (alerts are the primary SOC workflow)

### GOVERNING RULES
1. Phase 0 reconnaissance FIRST — read ALL backend alert files before writing code
2. The Explanation JSONB is the crown jewel — render it beautifully with proper hierarchy
3. Status transitions must be validated (not every transition is valid)
4. Comments support mentions (@analyst_name) with notification trigger
5. Investigation mode: when analyst acknowledges, auto-assign to them
6. Badge counts for new/unacknowledged alerts in navigation sidebar
7. Performance: alert list must handle 10k+ alerts with virtual scrolling if needed
8. All backend endpoints need proper error responses + audit events
9. Frontend: use existing components, hooks, patterns from the threats page
10. `GOWORK=off` for all Go commands, `npm run build` must pass clean
```

---

## Master Prompt 3 of 5: Detection Rules Management & MITRE ATT&CK Visualization

```
You are a Senior Full-Stack Engineer building the Detection Rule Management
interface and MITRE ATT&CK visualization for Clario360's Cyber module.

## CRITICAL CODEBASE CONTEXT — READ FIRST

Before writing any code, run Phase 0 reconnaissance on:

Backend (fully implemented):
- `backend/internal/cyber/model/detection_rule.go` — rule model (sigma, threshold,
  correlation, anomaly), conditions, MITRE mapping, FP/TP counters
- `backend/internal/cyber/detection/engine.go` — detection engine, rule evaluation
- `backend/internal/cyber/detection/condition_parser.go` — Sigma condition parser
- `backend/internal/cyber/detection/threshold_evaluator.go` — threshold rule evaluator
- `backend/internal/cyber/detection/correlation_evaluator.go` — correlation evaluator
- `backend/internal/cyber/detection/anomaly_evaluator.go` — anomaly detection
- `backend/internal/cyber/mitre/framework.go` — embedded ATT&CK framework (14 tactics, 50+ techniques)
- `backend/internal/cyber/mitre/coverage.go` — technique coverage analysis
- `backend/internal/cyber/mitre/mapper.go` — rule → technique mapping
- `backend/internal/cyber/handler/routes.go` — check for existing detection rule routes

Frontend (check if exists):
- `frontend/src/app/(dashboard)/cyber/sigma-rules/` — may have basic sigma page
- `frontend/src/types/cyber.ts` — check for DetectionRule types

## WHAT YOU MUST BUILD

### 1. Detection Rules List Page

New files:
- `frontend/src/app/(dashboard)/cyber/detection-rules/page.tsx`
- `frontend/src/app/(dashboard)/cyber/detection-rules/_components/rule-columns.tsx`
- `frontend/src/app/(dashboard)/cyber/detection-rules/_components/rule-stats.tsx`
- `frontend/src/app/(dashboard)/cyber/detection-rules/_components/create-rule-dialog.tsx`

**Route**: `/cyber/detection-rules`
**Permission**: `cyber:read` (view), `cyber:manage` (create/edit/toggle)

**Stats Row**:
- Total Rules (count)
- Active Rules (count, green)
- Sigma Rules / Threshold / Correlation / Anomaly (4 type counts)
- Overall True Positive Rate (percentage)

**DataTable columns**:
- Rule Name (clickable → detail)
- Type (badge: sigma=blue, threshold=green, correlation=purple, anomaly=orange)
- Severity (SeverityIndicator)
- MITRE Technique (badge)
- Status (enabled/disabled toggle)
- True Positives / False Positives (counts with ratio)
- Alerts Generated (count, last 30 days)
- Last Triggered (RelativeTime)
- Actions (edit, duplicate, disable, delete)

**Filters**:
- Type multi-select
- Severity multi-select
- MITRE Tactic filter
- Status (enabled/disabled)
- Search (name, description)

**Create Rule Button** → opens rule creation wizard (see section 3)

### 2. Detection Rule Detail Page

New files:
- `frontend/src/app/(dashboard)/cyber/detection-rules/[ruleId]/page.tsx`
- `frontend/src/app/(dashboard)/cyber/detection-rules/[ruleId]/_components/rule-overview.tsx`
- `frontend/src/app/(dashboard)/cyber/detection-rules/[ruleId]/_components/rule-logic.tsx`
- `frontend/src/app/(dashboard)/cyber/detection-rules/[ruleId]/_components/rule-performance.tsx`
- `frontend/src/app/(dashboard)/cyber/detection-rules/[ruleId]/_components/rule-alerts-tab.tsx`

**Tabbed layout**:

**Tab 1 — Overview**:
- Rule name, type badge, severity, description
- MITRE tactic/technique mapping (visual)
- Enabled/disabled toggle
- Data sources used
- Created by, created at, last modified

**Tab 2 — Detection Logic**:
- For **Sigma rules**: syntax-highlighted Sigma YAML editor (read-only view + edit mode)
  - Condition display, log sources, detection fields
  - Use a code editor component (Monaco or CodeMirror) with Sigma syntax highlighting
- For **Threshold rules**: visual threshold configuration
  - Field, operator, threshold value, time window, group-by
- For **Correlation rules**: visual correlation builder
  - Event A definition, Event B definition, correlation field, time window
- For **Anomaly rules**: anomaly parameters
  - Baseline period, sensitivity, algorithm type, features

**Tab 3 — Performance**:
- True Positive count & rate
- False Positive count & rate
- Total alerts generated (chart over time: line chart, last 30/90 days)
- Alert severity distribution
- Top triggered assets

**Tab 4 — Recent Alerts**:
- DataTable of alerts generated by this rule (paginated)
- Link to alert detail page

### 3. Rule Creation / Edit Wizard

New file: `frontend/src/app/(dashboard)/cyber/detection-rules/_components/rule-wizard.tsx`

**Step 1 — Basics**: Name, type (sigma/threshold/correlation/anomaly), severity, description
**Step 2 — Detection Logic** (dynamic based on type):
- Sigma: YAML editor with validation
- Threshold: field selector, operator, value, window, group-by
- Correlation: event definitions, correlation key, window
- Anomaly: baseline period, sensitivity, algorithm
**Step 3 — MITRE Mapping**: Select tactics/techniques from interactive matrix
**Step 4 — Review & Save**: Summary of all settings

Backend API needed:
- `POST /api/v1/cyber/detection-rules` — create rule
- `PUT /api/v1/cyber/detection-rules/{id}` — update rule
- `PUT /api/v1/cyber/detection-rules/{id}/toggle` — enable/disable
- `DELETE /api/v1/cyber/detection-rules/{id}` — soft delete
- `POST /api/v1/cyber/detection-rules/{id}/test` — test rule against recent events

### 4. MITRE ATT&CK Coverage Matrix Page

New files:
- `frontend/src/app/(dashboard)/cyber/mitre-attack/page.tsx`
- `frontend/src/app/(dashboard)/cyber/mitre-attack/_components/attack-matrix.tsx`
- `frontend/src/app/(dashboard)/cyber/mitre-attack/_components/technique-detail.tsx`
- `frontend/src/app/(dashboard)/cyber/mitre-attack/_components/coverage-stats.tsx`

**Route**: `/cyber/mitre-attack`

**MITRE ATT&CK Heat Map Matrix**:
- 14 tactic columns (Reconnaissance → Impact)
- Techniques as cells in each column
- Cell colour intensity based on:
  - Green: covered by active detection rule(s)
  - Yellow: covered but rule has high FP rate
  - Red: technique seen in active threat but no detection rule
  - Gray: not covered, no active threat
- Click cell → slide-out panel with:
  - Technique name, ID, description
  - Associated detection rules (links)
  - Associated threats (links)
  - Alerts triggered for this technique
  - Platform applicability

**Coverage Stats Bar**:
- Overall coverage: X/Y techniques covered (percentage with gauge chart)
- Coverage by tactic: mini bar chart
- Critical gap count: techniques with active threats but no rules

**Backend API needed**:
- `GET /api/v1/cyber/mitre/coverage` — technique coverage data
  (leverage existing `mitre/coverage.go`)
- `GET /api/v1/cyber/mitre/techniques` — full technique list with metadata
- `GET /api/v1/cyber/mitre/techniques/{id}` — technique detail with linked rules/threats

### 5. Navigation Update

Add to navigation.ts:
```typescript
{ id: 'cyber-detection', label: 'Detection Rules', href: '/cyber/detection-rules', icon: Shield },
{ id: 'cyber-mitre', label: 'MITRE ATT&CK', href: '/cyber/mitre-attack', icon: Grid3X3 },
```

### GOVERNING RULES
1. Phase 0 reconnaissance FIRST — read detection engine code before building UI
2. Sigma YAML editor: use Monaco Editor with custom language definition for Sigma
3. MITRE matrix must be responsive — horizontal scroll on mobile, full matrix on desktop
4. Rule testing: "Test Rule" button runs rule against last 1000 events and shows preview
5. Coverage calculation: use existing `mitre/coverage.go` logic, expose via API
6. Rule versioning: consider tracking rule edit history (optional, nice-to-have)
7. Performance: MITRE matrix with 200+ cells must render efficiently (virtualise if needed)
8. All Go endpoints need audit events and proper error handling
9. `GOWORK=off` for Go, `npm run build` must pass clean
```

---

## Master Prompt 4 of 5: IOC Management & Threat Intelligence Feed Integration

```
You are a Senior Full-Stack Engineer building the IOC (Indicator of Compromise)
Management and Threat Intelligence Feed integration UI for Clario360's Cyber module.

## CRITICAL CODEBASE CONTEXT — READ FIRST

Before writing any code, run Phase 0 reconnaissance on:

Backend (fully implemented):
- `backend/internal/cyber/model/threat.go` — ThreatIndicator model (11 types, 5 sources,
  confidence, expiration, tags, metadata)
- `backend/internal/cyber/indicator/stix_parser.go` — STIX 2 bundle parser
- `backend/internal/cyber/indicator/matcher.go` — in-memory IOC matcher with CIDR support
- `backend/internal/cyber/repository/indicator_repo.go` — indicator CRUD + batch operations
- `backend/internal/cyber/service/threat_service.go` — indicator service methods
- `backend/internal/cyber/handler/threat_handler.go` — indicator API handlers
- `backend/internal/cyber/enrichment/pipeline.go` — enrichment pipeline
- `backend/internal/cyber/enrichment/dns_enricher.go` — DNS resolution
- `backend/internal/cyber/enrichment/cve_enricher.go` — CVE matching
- `backend/internal/cyber/enrichment/geolocation_enricher.go` — IP geolocation
- `backend/internal/cyber/vciso/predict/feeds/threat_feed_ingester.go` — feed ingestion

Frontend (minimal):
- `frontend/src/app/(dashboard)/cyber/threats/_components/indicator-check-dialog.tsx` —
  the only IOC UI (paste & check)
- `frontend/src/types/cyber.ts` — ThreatIndicator, IndicatorCheckResult types

## WHAT YOU MUST BUILD

### 1. IOC Management Page

New files:
- `frontend/src/app/(dashboard)/cyber/indicators/page.tsx`
- `frontend/src/app/(dashboard)/cyber/indicators/_components/indicator-columns.tsx`
- `frontend/src/app/(dashboard)/cyber/indicators/_components/indicator-detail-panel.tsx`
- `frontend/src/app/(dashboard)/cyber/indicators/_components/add-indicator-dialog.tsx`
- `frontend/src/app/(dashboard)/cyber/indicators/_components/bulk-import-dialog.tsx`
- `frontend/src/app/(dashboard)/cyber/indicators/_components/indicator-stats.tsx`

**Route**: `/cyber/indicators`
**Permission**: `cyber:read` (view), `cyber:write` (manage)

**Stats Row**:
- Total IOCs (count)
- Active IOCs (count, green)
- Expiring Soon (count, warning — within 7 days)
- Sources breakdown: mini bar chart (manual, stix_feed, osint, internal, vendor)

**DataTable columns**:
- Type (badge, colour-coded: ip=blue, domain=purple, hash=orange, url=red,
  email=yellow, certificate=teal, registry_key=gray, user_agent=pink, cidr=indigo)
- Value (monospace, truncated with copy button)
- Severity (SeverityIndicator)
- Source (badge)
- Confidence (progress bar with % label)
- Linked Threat (threat name, clickable → threat detail)
- Active (toggle switch)
- First Seen / Last Seen (RelativeTime)
- Expires At (date, red if < 7 days)
- Actions (view, edit, deactivate, delete)

**Filters**:
- Type multi-select (11 types)
- Source multi-select (5 sources)
- Severity multi-select
- Active only toggle
- Linked to threat (yes/no)
- Confidence range slider
- Search (value, tags)

**Bulk Actions**:
- Deactivate selected
- Activate selected
- Export selected (CSV/JSON/STIX)
- Delete selected

### 2. Add Indicator Dialog

Form fields:
- **Type** (required, select from 11 types)
- **Value** (required, validated per type):
  - IP: valid IPv4/IPv6 regex
  - Domain: valid domain regex
  - URL: valid URL
  - Email: valid email
  - Hash: valid MD5 (32 hex) / SHA1 (40 hex) / SHA256 (64 hex) based on selected type
  - CIDR: valid CIDR notation
- **Severity** (required, select)
- **Source** (required, select: manual, osint, internal, vendor)
- **Confidence** (slider 0-100, default 80 for manual)
- **Description** (textarea)
- **Linked Threat** (optional, searchable select from existing threats)
- **Expires At** (optional date picker)
- **Tags** (tag input)

API: `POST /api/v1/cyber/indicators` (add if not exists as standalone)
Or `POST /api/v1/cyber/threats/{id}/indicators` for threat-linked

### 3. Bulk Import Dialog (STIX/CSV/Manual Paste)

Three import modes:

**Tab 1 — STIX Bundle**:
- File upload for `.json` STIX 2 bundle
- Preview parsed indicators before import (table showing what will be imported)
- Conflict resolution: skip duplicates (default), update existing, or fail
- API: `POST /api/v1/cyber/indicators/bulk` (exists)

**Tab 2 — CSV Import**:
- File upload for `.csv`
- Column mapping step: map CSV columns to indicator fields
- Preview first 10 rows
- Validation: highlight invalid rows in red
- API: `POST /api/v1/cyber/indicators/bulk` with format=csv

**Tab 3 — Manual Paste**:
- Enhanced version of existing IndicatorCheckDialog
- Paste multiple indicators (one per line)
- Auto-detect type (IP/domain/URL/hash)
- Set common severity/source/confidence for all
- Option to create as new indicators (not just check)

**Import Summary**: After import, show:
- Total parsed, imported, skipped (duplicate), failed
- Link to view imported indicators

### 4. Indicator Detail Panel (Slide-out)

When clicking an indicator row:
- Type + value + severity header
- Source + confidence + active status
- First/last seen + expiration
- **Enrichment section**:
  - DNS resolution (for domains/IPs)
  - Geolocation (for IPs — country, city, ASN)
  - CVE associations (for hashes — if file associated with vulnerable software)
  - Reputation score (if available)
  - WHOIS data (for domains)
- **Linked Threat**: card with threat name, type, status (click → threat detail)
- **Detection History**: recent security events that matched this indicator
  (from matcher/detection engine)
- **Tags** display + edit

Backend API needed:
- `GET /api/v1/cyber/indicators/{id}` — full indicator detail
- `GET /api/v1/cyber/indicators/{id}/enrichment` — enrichment data
- `GET /api/v1/cyber/indicators/{id}/matches` — recent detection matches

### 5. Threat Intelligence Feed Configuration

New files:
- `frontend/src/app/(dashboard)/cyber/threat-feeds/page.tsx`
- `frontend/src/app/(dashboard)/cyber/threat-feeds/_components/feed-list.tsx`
- `frontend/src/app/(dashboard)/cyber/threat-feeds/_components/add-feed-dialog.tsx`
- `frontend/src/app/(dashboard)/cyber/threat-feeds/_components/feed-detail.tsx`

**Route**: `/cyber/threat-feeds`
**Permission**: `cyber:manage`

**Feed List**:
- DataTable of configured threat intel feeds
- Columns: name, type (STIX/TAXII/MISP/CSV URL), URL, status (active/paused/error),
  last sync, indicators imported, next sync, actions
- Add Feed button

**Add/Edit Feed Dialog**:
- Name (text)
- Type (STIX/TAXII 2.1, MISP, CSV URL, Manual)
- URL (for STIX/TAXII/MISP/CSV feeds)
- Authentication (none, API key, basic auth, certificate)
- Sync interval (hourly, every 6h, daily, weekly, manual)
- Auto-import settings: default severity, default confidence, default tags
- Filter: only import specific indicator types
- Enable/disable toggle

**Feed Detail** (click feed name):
- Feed configuration summary
- Import history: table of past syncs (timestamp, indicators added, duration, status)
- Manual sync button: "Sync Now"
- Last import preview: table of recently imported indicators

Backend API needed:
- `GET /api/v1/cyber/threat-feeds` — list configured feeds
- `POST /api/v1/cyber/threat-feeds` — create feed configuration
- `PUT /api/v1/cyber/threat-feeds/{id}` — update feed config
- `POST /api/v1/cyber/threat-feeds/{id}/sync` — trigger manual sync
- `GET /api/v1/cyber/threat-feeds/{id}/history` — sync history
- Database: new table `threat_feed_configs` (name, type, url, auth, interval, settings, last_sync)

### 6. Navigation Update

Add to navigation.ts:
```typescript
{ id: 'cyber-indicators', label: 'IOC Management', href: '/cyber/indicators', icon: Fingerprint },
{ id: 'cyber-feeds', label: 'Threat Feeds', href: '/cyber/threat-feeds', icon: Rss },
```

### GOVERNING RULES
1. Phase 0 reconnaissance FIRST — read the STIX parser and indicator matcher
2. IOC value validation is CRITICAL — prevent garbage data from entering the system
3. Copy button on all IOC values (analysts copy-paste IOCs constantly)
4. STIX import must show preview before committing (never blind import)
5. Enrichment data may take time — show loading states, cache results
6. Feed sync is async — show last sync status, errors, not blocking UI
7. CSV column mapping must be flexible (different feeds have different schemas)
8. Indicator expiration: visual warning for indicators expiring within 7 days
9. `GOWORK=off` for Go, `npm run build` must pass clean
```

---

## Master Prompt 5 of 5: Security Events Viewer, CTEM Dashboard & Threat Analytics

```
You are a Senior Full-Stack Engineer building the Security Events Viewer, CTEM
Dashboard, and Threat Analytics for Clario360's Cyber module.

## CRITICAL CODEBASE CONTEXT — READ FIRST

Before writing any code, run Phase 0 reconnaissance on:

Backend (fully implemented):
- `backend/internal/cyber/model/security_event.go` — SecurityEvent model (partitioned table)
- `backend/internal/cyber/model/common.go` — shared types, event fields
- `backend/internal/cyber/ctem/` — full CTEM engine (scope, discovery, prioritization,
  validation, mobilization, reporting)
- `backend/internal/cyber/vciso/predict/threat_forecast.go` — threat forecasting
- `backend/internal/cyber/vciso/predict/campaign_detector.go` — campaign detection
- `backend/internal/cyber/vciso/predict/alert_volume_forecaster.go` — alert volume forecast
- `backend/internal/cyber/vciso/predict/feeds/attack_technique_trend.go` — technique trends
- `backend/internal/cyber/ueba/` — UEBA engine
- `backend/internal/cyber/handler/routes.go` — check for existing event/CTEM routes
- `backend/migrations/cyber_db/000003_threat_detection_engine.up.sql` — security_events table

Frontend (check what exists):
- `frontend/src/app/(dashboard)/cyber/ctem/` — check if CTEM pages exist
- `frontend/src/app/(dashboard)/cyber/ueba/` — check if UEBA pages exist
- `frontend/src/types/cyber.ts` — check for SecurityEvent, CTEM types

## WHAT YOU MUST BUILD

### 1. Security Events Viewer (Log Explorer)

New files:
- `frontend/src/app/(dashboard)/cyber/events/page.tsx`
- `frontend/src/app/(dashboard)/cyber/events/_components/event-columns.tsx`
- `frontend/src/app/(dashboard)/cyber/events/_components/event-detail-panel.tsx`
- `frontend/src/app/(dashboard)/cyber/events/_components/event-timeline.tsx`
- `frontend/src/app/(dashboard)/cyber/events/_components/event-filters.tsx`

**Route**: `/cyber/events`
**Permission**: `cyber:read`

**This is the SIEM log viewer** — the primary tool for analysts investigating incidents.

**DataTable columns**:
- Timestamp (precise, sortable, default sort desc)
- Source (log source system name)
- Event Type (categorized badge)
- Severity (SeverityIndicator)
- Source IP (monospace, clickable → IOC check)
- Dest IP (monospace, clickable → IOC check)
- Username (if available)
- Process (if available)
- Matched Rules (count badge, expandable)
- Actions (view raw, correlate)

**Filters** (advanced filter bar):
- Time range: preset buttons (last 1h, 6h, 24h, 7d, 30d) + custom date range
- Source IP / Dest IP (text input with CIDR support)
- Port range
- Protocol (TCP, UDP, ICMP, etc.)
- Event type multi-select
- Severity multi-select
- Username (text)
- Process name (text)
- Command line contains (text — for process execution hunting)
- File hash (text — for malware hunting)
- Free text search across raw_event JSONB
- Matched rule (select from detection rules)

**Event Detail Panel** (slide-out on row click):
- Formatted event header (timestamp, source, type, severity)
- **Network context**: source IP → dest IP:port (visual arrow diagram)
- **Process tree**: process → parent process (if available)
- **Command line**: full command (monospace, syntax-highlighted)
- **Raw event**: collapsible JSON viewer (pretty-printed)
- **Matched rules**: list of detection rules that matched this event
- **IOC matches**: any indicators that match fields in this event
- **Similar events**: "Find similar" button → filters to same source+type in ±1h window

**Real-time streaming toggle**:
- "Live" button: when enabled, new events stream in at top of table
- WebSocket topic: `security_event.ingested`
- Pause/resume stream

**Backend API needed**:
- `GET /api/v1/cyber/events` — paginated event list with all filters
- `GET /api/v1/cyber/events/{id}` — single event detail
- `GET /api/v1/cyber/events/stats` — event volume by source, type, severity (for charts)
- WebSocket: `security_event.ingested` topic for live streaming

### 2. CTEM (Continuous Threat Exposure Management) Dashboard

New files:
- `frontend/src/app/(dashboard)/cyber/ctem/dashboard/page.tsx`
- `frontend/src/app/(dashboard)/cyber/ctem/dashboard/_components/ctem-overview.tsx`
- `frontend/src/app/(dashboard)/cyber/ctem/dashboard/_components/exposure-score.tsx`
- `frontend/src/app/(dashboard)/cyber/ctem/dashboard/_components/attack-paths.tsx`
- `frontend/src/app/(dashboard)/cyber/ctem/dashboard/_components/remediation-tracker.tsx`
- `frontend/src/app/(dashboard)/cyber/ctem/assessments/page.tsx`
- `frontend/src/app/(dashboard)/cyber/ctem/assessments/[id]/page.tsx`

**Route**: `/cyber/ctem/dashboard`
**Permission**: `cyber:read`

**CTEM Dashboard Overview**:
- **Exposure Score Gauge**: Large circular gauge (0-100) with colour gradient
  (green ≤30, yellow 30-60, red >60)
- **KPI Row**: Total exposures, Critical exposures, Remediation rate,
  Mean time to remediate (MTTR)
- **Exposure Trend**: Area chart showing exposure score over last 90 days
- **Top Attack Paths**: List of discovered attack paths sorted by risk
  (source asset → intermediate hops → target asset with cumulative risk)
- **Remediation Progress**: Stacked bar chart (remediated vs open vs accepted risk)

**CTEM Assessment Framework** (5 phases, visual):
1. **Scope** — Define which assets/networks to assess
   - Asset group selector (from existing asset inventory)
   - Network range input
2. **Discovery** — Automated scan for exposures & attack paths
   - Run discovery button → shows progress
   - Results: list of discovered exposures with severity
3. **Prioritization** — Risk-based ranking of findings
   - Sortable table of exposures by risk score
   - CVSS score, exploitability, asset criticality, business impact
4. **Validation** — Verify findings are real (reduce false positives)
   - Analyst can mark findings as validated, false positive, or needs review
5. **Mobilization** — Create remediation tasks
   - Create remediation group
   - Assign to team/owner
   - Set due date
   - Track completion status

**Assessment List Page** (`/cyber/ctem/assessments`):
- DataTable of past/current assessments
- Columns: name, scope, status (scoping/discovering/prioritizing/validating/mobilizing/complete),
  exposure count, critical count, remediation %, created_at
- "New Assessment" button → wizard

### 3. Threat Analytics & Predictive Intelligence Dashboard

New files:
- `frontend/src/app/(dashboard)/cyber/analytics/page.tsx`
- `frontend/src/app/(dashboard)/cyber/analytics/_components/threat-forecast.tsx`
- `frontend/src/app/(dashboard)/cyber/analytics/_components/technique-trends.tsx`
- `frontend/src/app/(dashboard)/cyber/analytics/_components/campaign-detection.tsx`
- `frontend/src/app/(dashboard)/cyber/analytics/_components/alert-volume-forecast.tsx`
- `frontend/src/app/(dashboard)/cyber/analytics/_components/threat-landscape.tsx`

**Route**: `/cyber/analytics`
**Permission**: `cyber:read`

**Section 1 — Threat Landscape Overview**:
- Active threat count (KPI)
- Top MITRE tactic (badge + count)
- Top MITRE technique (badge + count)
- Threat distribution by type: donut chart
- Threats over time: area chart (last 90 days)

**Section 2 — Threat Forecast** (from predictive engine):
- "Next 30 Day Threat Forecast" card
- Predicted threat types with probability bars
- Predicted attack techniques with trend arrows (↑ increasing, ↓ decreasing, → stable)
- Confidence interval display
- API: `GET /api/v1/cyber/analytics/threat-forecast`

**Section 3 — Alert Volume Forecast**:
- Line chart: historical daily alert volume + predicted future volume (30 days)
- Confidence bands (shaded area)
- Anomaly markers (days where actual >> predicted)
- API: `GET /api/v1/cyber/analytics/alert-forecast`

**Section 4 — Attack Technique Trends**:
- Table of top 20 techniques by recent activity
- Columns: technique ID, name, count (30d), trend (sparkline), delta vs previous period
- Technique heatmap: calendar heatmap (like GitHub contribution graph) for top 5 techniques
- API: `GET /api/v1/cyber/analytics/technique-trends`

**Section 5 — Campaign Detection**:
- Active campaigns detected (card list)
- Each card: campaign name, linked threats count, IOC overlap %,
  MITRE technique overlap, start date, status
- "Investigate" button → navigates to threat detail with campaign filter
- API: `GET /api/v1/cyber/analytics/campaigns`

### 4. Cyber Threat Overview Dashboard (Landing Page)

New/modify: `frontend/src/app/(dashboard)/cyber/page.tsx`

The main `/cyber` landing page should be an executive dashboard combining:
- Active threats (count + severity breakdown)
- Open alerts (count + new vs acknowledged)
- MITRE coverage percentage (gauge)
- CTEM exposure score (gauge)
- Recent alerts (mini table, last 5)
- Threat forecast summary (next 30 days)
- Event volume (sparkline, last 24h)

Each section is a card that links to the detailed page.

### 5. Backend API Additions

For analytics:
```
GET /api/v1/cyber/analytics/threat-forecast    → from predict/threat_forecast.go
GET /api/v1/cyber/analytics/alert-forecast     → from predict/alert_volume_forecaster.go
GET /api/v1/cyber/analytics/technique-trends   → from predict/feeds/attack_technique_trend.go
GET /api/v1/cyber/analytics/campaigns          → from predict/campaign_detector.go
GET /api/v1/cyber/analytics/landscape          → aggregated threat landscape
```

For events:
```
GET  /api/v1/cyber/events                      → paginated event list
GET  /api/v1/cyber/events/{id}                 → event detail
GET  /api/v1/cyber/events/stats                → volume stats
```

For CTEM:
```
GET  /api/v1/cyber/ctem/dashboard              → exposure score + KPIs
GET  /api/v1/cyber/ctem/assessments            → assessment list
POST /api/v1/cyber/ctem/assessments            → create assessment
GET  /api/v1/cyber/ctem/assessments/{id}       → assessment detail
POST /api/v1/cyber/ctem/assessments/{id}/discover  → run discovery
GET  /api/v1/cyber/ctem/attack-paths           → discovered attack paths
```

### 6. Navigation Update

Add/restructure cyber navigation:
```typescript
// Primary cyber nav items (ordered by SOC workflow priority)
{ id: 'cyber-overview', label: 'Overview', href: '/cyber', icon: LayoutDashboard },
{ id: 'cyber-alerts', label: 'Alerts', href: '/cyber/alerts', icon: Bell },
{ id: 'cyber-events', label: 'Event Explorer', href: '/cyber/events', icon: Terminal },
{ id: 'cyber-threats', label: 'Threat Hunting', href: '/cyber/threats', icon: Search },
{ id: 'cyber-indicators', label: 'IOC Management', href: '/cyber/indicators', icon: Fingerprint },
{ id: 'cyber-detection', label: 'Detection Rules', href: '/cyber/detection-rules', icon: Shield },
{ id: 'cyber-mitre', label: 'MITRE ATT&CK', href: '/cyber/mitre-attack', icon: Grid3X3 },
{ id: 'cyber-analytics', label: 'Analytics', href: '/cyber/analytics', icon: TrendingUp },
{ id: 'cyber-ctem', label: 'CTEM', href: '/cyber/ctem/dashboard', icon: Radar },
{ id: 'cyber-feeds', label: 'Threat Feeds', href: '/cyber/threat-feeds', icon: Rss },
// Keep existing: vulnerabilities, assets, sigma-rules, ueba
```

### GOVERNING RULES
1. Phase 0 reconnaissance FIRST — read CTEM engine, predictive engine, and event model
2. Security Events page MUST handle high volume — virtual scrolling, server-side pagination only
3. Live event streaming: use existing WebSocket infrastructure, throttle to max 10 events/sec in UI
4. CTEM phases are a guided wizard workflow, not just separate pages
5. Predictive charts: always show confidence intervals, never present predictions as certainties
6. Campaign detection: link to existing threats, don't create separate campaign entities
7. Event filters must support analyst hunting workflows (IOC search, process hunting, lateral movement)
8. All chart components from existing shared library (BarChart, LineChart, AreaChart, GaugeChart, PieChart)
9. Performance: event viewer must handle millions of events via server-side search only
10. `GOWORK=off` for Go, `npm run build` must pass clean
```

---

## Execution Sequence

| Step | Prompt | Key Deliverable | Depends On |
|------|--------|-----------------|------------|
| 1 | Master Prompt 1 | Threat dashboard + CRUD + full detail page with tabs | None (extends existing) |
| 2 | Master Prompt 2 | Alert management + investigation workspace | Prompt 1 (threat links) |
| 3 | Master Prompt 3 | Detection rules UI + MITRE ATT&CK matrix visualization | Prompts 1+2 (threat/alert links) |
| 4 | Master Prompt 4 | IOC management + STIX import + threat feed configuration | Prompts 1-3 (linked entities) |
| 5 | Master Prompt 5 | Event viewer + CTEM dashboard + predictive analytics + overview | Prompts 1-4 (full integration) |

Each prompt produces independently deployable features that follow existing Clario360 patterns.
Every prompt requires Phase 0 reconnaissance of specific files before writing any code.

## Enterprise Capability Baseline

After all 5 prompts are implemented, the Cyber Threat module will match or exceed capabilities found in:

| Capability | Comparable Products |
|-----------|-------------------|
| Alert Management + Investigation | Splunk SOAR, IBM QRadar, Microsoft Sentinel |
| MITRE ATT&CK Matrix | MITRE ATT&CK Navigator, CrowdStrike Falcon |
| IOC Management + STIX/TAXII | MISP, OpenCTI, Anomali ThreatStream |
| Detection Rules (Sigma) | Sigma HQ, Elastic Detection Rules |
| Security Events (SIEM) | Splunk, Elastic SIEM, Sumo Logic |
| CTEM | Tenable, Qualys, XM Cyber |
| Threat Analytics + Forecasting | Recorded Future, Mandiant Advantage |
| Threat Feeds | AlienVault OTX, VirusTotal, AbuseIPDB |
