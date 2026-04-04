# PROMPT 20 — Visus Suite: Executive Dashboards, KPIs, Reports & Alerts

## Objective

Complete the Visus (Executive Intelligence) Suite frontend. Currently only basic list pages exist. The backend provides a full executive dashboard builder with drag-and-drop widgets, KPI tracking with snapshot history, scheduled report generation, executive alerts, and cross-suite executive summaries — none of which are implemented in the frontend.

## Reference Architecture

- **Stack**: Next.js 14 App Router, TypeScript, Zustand, React Query, Zod, react-hook-form, Tailwind, shadcn/ui, Recharts
- **Existing types**: `types/suites.ts` — already has comprehensive Visus types (use as-is):
  - `VisusDashboard`, `VisusWidget`, `VisusWidgetPosition`, `VisusWidgetType`
  - `VisusKPIDefinition`, `VisusKPISnapshot`, `VisusKPIStatus`, `VisusKPICategory`
  - `VisusReportDefinition`, `VisusReportSnapshot`
  - `VisusExecutiveAlert`, `VisusAlertStats`, `VisusAlertSeverity`, `VisusAlertStatus`
  - `VisusExecutiveSummary`, `VisusSuiteStatus`
  - Widget data types: `VisusKpiCardWidgetData`, `VisusSeriesWidgetData`, `VisusPieWidgetData`, `VisusGaugeWidgetData`, `VisusTableWidgetData`, `VisusHeatmapWidgetData`, etc.
  - `VisusWidgetTypeDefinition` (for available widget types schema)
- **Permissions**: `<PermissionRedirect permission="visus:read">` (or `visus:write` for mutations)
- **WebSocket topics**: `visus.alert.created`, `visus.alert.updated`, `visus.kpi.threshold_breach`, `visus.report.generated`, `visus.dashboard.updated`

## Constants to Add (`lib/constants.ts`)

```typescript
// Visus — Dashboards (extend existing VISUS_DASHBOARDS)
// Uses: GET/POST /dashboards, GET/PUT/DELETE /dashboards/{id}
//       POST /dashboards/{id}/duplicate, PUT /dashboards/{id}/share
// Widgets: POST/GET/PUT/DELETE /dashboards/{id}/widgets/{wid}
//          GET /dashboards/{id}/widgets/{wid}/data
//          PUT /dashboards/{id}/widgets/layout

VISUS_WIDGET_TYPES: '/api/v1/visus/widgets/types',

// Visus — KPIs
VISUS_KPIS: '/api/v1/visus/kpis',
VISUS_KPIS_SUMMARY: '/api/v1/visus/kpis/summary',

// Visus — Alerts (extend existing)
VISUS_ALERTS: '/api/v1/visus/alerts',
VISUS_ALERTS_COUNT: '/api/v1/visus/alerts/count',
VISUS_ALERTS_STATS: '/api/v1/visus/alerts/stats',

// Visus — Reports (extend existing)
// Uses: GET/POST/PUT/DELETE /reports/{id}
//       POST /reports/{id}/generate
//       GET /reports/{id}/snapshots, /snapshots/latest, /snapshots/{snapId}

// Visus — Executive
VISUS_EXECUTIVE: '/api/v1/visus/executive',
VISUS_EXECUTIVE_SUMMARY: '/api/v1/visus/executive/summary',
VISUS_EXECUTIVE_HEALTH: '/api/v1/visus/executive/health',
```

---

## PART A — Executive Overview (`visus/page.tsx`)

Replace the current basic page with an executive command center.

### A1. Executive Summary View

Fetch `GET /api/v1/visus/executive/summary` → `VisusExecutiveSummary`.

**Layout:**
1. **Suite Health Banner** — from `suite_health` map:
   - Row of status indicators for each suite (cyber, data, acta, lex)
   - Green/Yellow/Red status with latency and last success time
   - Click any → navigate to that suite's dashboard

2. **Cross-Suite KPI Cards** — from `kpis[]` (VisusKPISnapshot):
   - Dynamic KPI card grid showing latest snapshot values
   - Each card: name, value (formatted by unit), trend delta, status color (normal/warning/critical)

3. **Executive Alerts** — from `alerts[]`:
   - Top 5 critical/high alerts with severity badge, description, action link
   - "View All" link → `/visus/alerts`

4. **Suite Summaries**:
   - **Cyber Security** card → from `cyber_security` data
   - **Data Intelligence** card → from `data_intelligence` data
   - **Governance** card → from `governance` data
   - **Legal** card → from `legal` data

### A2. Health Overview

`GET /api/v1/visus/executive/health` — Detailed health status:
- Suite availability status grid
- Response time chart per suite
- Error rate indicators
- Last sync timestamps

**Components:**
```
components/visus/executive/
  suite-health-banner.tsx          — Suite status row with indicators
  executive-kpi-grid.tsx           — Dynamic KPI card grid
  executive-alerts-preview.tsx     — Top alerts with action links
  suite-summary-cards.tsx          — Cross-suite summary panels
  health-overview.tsx              — Detailed health status
```

---

## PART B — Dashboard Builder (Full CRUD + Widget Management)

### B1. Dashboards List Page (`visus/dashboards/page.tsx`) — RENAME/ENHANCE existing

If the current page lists dashboards, enhance it. Otherwise create.

- **Create Dashboard** button
- Show: name, description, visibility badge, widget count, shared status, created by, dates
- Per-row actions: Open, Duplicate, Share, Edit, Delete
- Filter by visibility (private, team, organization, public)

### B2. Dashboard Create Dialog

Form fields:
- Name (required)
- Description (textarea)
- Grid Columns (number, default 12)
- Visibility (select from `VisusDashboardVisibility`)
- Tags

### B3. Dashboard Detail/Builder Page (`visus/dashboards/[id]/page.tsx`) — NEW PAGE

**This is the centerpiece — a drag-and-drop dashboard builder.**

**Header:**
- Dashboard name (editable inline)
- Edit/View mode toggle
- Share button → `PUT /api/v1/visus/dashboards/{id}/share` — Dialog with user/team multi-select
- Duplicate button → `POST /api/v1/visus/dashboards/{id}/duplicate`
- Delete button → `DELETE /api/v1/visus/dashboards/{id}`

**Widget Grid (View Mode):**
- Render widgets in a responsive grid based on `VisusWidgetPosition` (x, y, w, h)
- Each widget fetches its data from `GET /api/v1/visus/dashboards/{id}/widgets/{wid}/data`
- Auto-refresh based on `widget.refresh_interval_seconds`
- Widget types render as appropriate chart/display:

| Widget Type | Component | Data Type |
|---|---|---|
| `kpi_card` | KPI card with value, trend, target | `VisusKpiCardWidgetData` |
| `line_chart` | Line chart (Recharts) | `VisusSeriesWidgetData` |
| `bar_chart` | Bar chart (Recharts) | `VisusSeriesWidgetData` |
| `area_chart` | Area chart (Recharts) | `VisusSeriesWidgetData` |
| `pie_chart` | Pie chart (Recharts) | `VisusPieWidgetData` |
| `gauge` | Gauge chart | `VisusGaugeWidgetData` |
| `table` | Data table | `VisusTableWidgetData` |
| `alert_feed` | Alert list | `VisusAlertFeedWidgetData` |
| `text` | Rich text display | `VisusTextWidgetData` |
| `sparkline` | Mini sparkline chart | `VisusSparklineWidgetData` |
| `heatmap` | Heatmap grid | `VisusHeatmapWidgetData` |
| `status_grid` | Status indicator grid | `VisusStatusGridWidgetData` |
| `trend_indicator` | Trend arrow with delta | `VisusTrendIndicatorWidgetData` |

**Widget Grid (Edit Mode):**
- Drag-and-drop widget repositioning (use `react-grid-layout` or similar)
- Resize handles on each widget
- Save layout → `PUT /api/v1/visus/dashboards/{id}/widgets/layout` — Send `{ widgets: Array<{ id: string; position: VisusWidgetPosition }> }`
- **Add Widget** button:
  1. Select type from `GET /api/v1/visus/widgets/types` → `VisusWidgetTypeDefinition[]`
  2. Configure widget (title, subtitle, config based on type schema, refresh interval)
  3. Place on grid (auto-find empty space or specify position)
  4. Save → `POST /api/v1/visus/dashboards/{id}/widgets`
- **Edit Widget** → click widget in edit mode → config dialog → `PUT .../widgets/{wid}`
- **Delete Widget** → `DELETE .../widgets/{wid}`

**API calls:**
```
GET    /api/v1/visus/dashboards                              — List
POST   /api/v1/visus/dashboards                              — Create
GET    /api/v1/visus/dashboards/{id}                         — Get with widgets
PUT    /api/v1/visus/dashboards/{id}                         — Update dashboard metadata
DELETE /api/v1/visus/dashboards/{id}                         — Delete
POST   /api/v1/visus/dashboards/{id}/duplicate               — Duplicate
PUT    /api/v1/visus/dashboards/{id}/share                   — Share { shared_with: string[] }
POST   /api/v1/visus/dashboards/{id}/widgets                 — Add widget
GET    /api/v1/visus/dashboards/{id}/widgets/{wid}           — Get widget
PUT    /api/v1/visus/dashboards/{id}/widgets/{wid}           — Update widget
DELETE /api/v1/visus/dashboards/{id}/widgets/{wid}           — Delete widget
GET    /api/v1/visus/dashboards/{id}/widgets/{wid}/data      — Widget data
PUT    /api/v1/visus/dashboards/{id}/widgets/layout           — Update layout
GET    /api/v1/visus/widgets/types                            — Available widget types
```

**Components:**
```
components/visus/dashboards/
  dashboard-columns.tsx
  dashboard-create-dialog.tsx
  dashboard-share-dialog.tsx
  dashboard-grid.tsx               — Main grid container (view + edit modes)
  dashboard-edit-toolbar.tsx       — Edit mode controls

  widgets/
    widget-renderer.tsx            — Routes widget type to correct component
    widget-wrapper.tsx             — Wrapper with title, subtitle, refresh, edit/delete controls
    widget-add-dialog.tsx          — Type selection + configuration
    widget-config-dialog.tsx       — Edit widget configuration
    widget-kpi-card.tsx
    widget-line-chart.tsx
    widget-bar-chart.tsx
    widget-area-chart.tsx
    widget-pie-chart.tsx
    widget-gauge.tsx
    widget-table.tsx
    widget-alert-feed.tsx
    widget-text.tsx
    widget-sparkline.tsx
    widget-heatmap.tsx
    widget-status-grid.tsx
    widget-trend-indicator.tsx
```

---

## PART C — KPI Management (Full CRUD + History)

### C1. KPIs Page (`visus/kpis/page.tsx`)

Enhance with:
- **KPI Summary** → `GET /api/v1/visus/kpis/summary` — Overview of all KPIs with latest values
- **KPI List** → `GET /api/v1/visus/kpis` — DataTable
  - Columns: name, category, suite, unit, last value, last status (color badge), direction, snapshot frequency, enabled (toggle)
- **Create KPI** button → dialog
- Per-row actions: Edit, Snapshot Now, View History, Delete

### C2. KPI Create/Edit Dialog

Form fields:
- Name (required), Description
- Category (select from `VisusKPICategory`)
- Suite (select from `VisusKPISuite`)
- Icon (optional, icon picker or text)
- Query Endpoint (text — the internal API endpoint to fetch value from)
- Query Params (JSON editor)
- Value Path (text — JSONPath to extract value from response)
- Unit (select from `VisusKPIUnit`)
- Format Pattern (optional text)
- Target Value (optional number)
- Warning Threshold (number)
- Critical Threshold (number)
- Direction (select: higher_is_better, lower_is_better)
- Calculation Type (select from `VisusKPICalculationType`)
- Calculation Window (optional text, e.g., "7d", "30d")
- Snapshot Frequency (select from `VisusKPISnapshotFrequency`)
- Tags

### C3. KPI Detail (slide-out or page)

- Current value with status color
- Trend chart → `GET /api/v1/visus/kpis/{id}/history` — Line chart of historical snapshots
- Target vs actual indicator
- Configuration details
- **Manual Snapshot** → `POST /api/v1/visus/kpis/snapshot` with `{ kpi_id }` — Force snapshot now

**API calls:**
```
POST   /api/v1/visus/kpis              — Create
GET    /api/v1/visus/kpis              — List
GET    /api/v1/visus/kpis/summary      — Summary with latest values
GET    /api/v1/visus/kpis/{id}         — Get detail with history
PUT    /api/v1/visus/kpis/{id}         — Update
DELETE /api/v1/visus/kpis/{id}         — Delete
POST   /api/v1/visus/kpis/snapshot     — Force snapshot
GET    /api/v1/visus/kpis/{id}/history — Historical snapshots
```

**Components:**
```
components/visus/kpis/
  kpi-columns.tsx
  kpi-create-dialog.tsx
  kpi-edit-form.tsx
  kpi-detail-panel.tsx
  kpi-history-chart.tsx            — Line chart of snapshots over time
  kpi-status-badge.tsx             — Color-coded status (normal/warning/critical)
  kpi-summary-grid.tsx             — Grid of KPI cards with latest values
  kpi-target-indicator.tsx         — Gauge showing actual vs target
```

---

## PART D — Report Management (CRUD + Generation)

### D1. Reports Page (`visus/reports/page.tsx`)

Enhance with:
- **Create Report** button → dialog
- DataTable: name, type, period, schedule, next_run, last_generated, total_generated, auto_send
- Per-row actions: Generate Now, View Latest, View Snapshots, Edit, Delete

### D2. Report Create/Edit Dialog

Form fields:
- Name (required), Description
- Report Type (select from `VisusReportType`: executive_summary, security_posture, data_intelligence, governance, legal, custom)
- Sections (multi-select or checklist of available report sections)
- Period (select: daily, weekly, monthly, quarterly, custom)
- Custom Period Start/End (date pickers, shown if period=custom)
- Schedule (cron expression with human-readable preview, optional)
- Recipients (multi-select of users/emails)
- Auto Send (toggle)

### D3. Report Detail/Snapshot Page

- Report definition info
- **Generate Now** → `POST /api/v1/visus/reports/{id}/generate` — Loading state, then show result
- **Snapshot List** → `GET /api/v1/visus/reports/{id}/snapshots` — DataTable with generated_at, period, format, generation time
- **Latest Snapshot** → `GET .../snapshots/latest` — Rendered report content
- **Snapshot Detail** → `GET .../snapshots/{snapId}` — Full report with:
  - Narrative text
  - Report data (rendered as sections with charts/tables based on content)
  - Sections included
  - Generation time
  - Any fetch errors per suite
  - Download option (if file_id exists)

**API calls:**
```
POST   /api/v1/visus/reports                          — Create
GET    /api/v1/visus/reports                          — List
GET    /api/v1/visus/reports/{id}                     — Get detail
PUT    /api/v1/visus/reports/{id}                     — Update
DELETE /api/v1/visus/reports/{id}                     — Delete
POST   /api/v1/visus/reports/{id}/generate            — Generate report
GET    /api/v1/visus/reports/{id}/snapshots            — List snapshots
GET    /api/v1/visus/reports/{id}/snapshots/latest     — Latest snapshot
GET    /api/v1/visus/reports/{id}/snapshots/{snapId}   — Specific snapshot
```

**Components:**
```
components/visus/reports/
  report-columns.tsx
  report-create-dialog.tsx
  report-edit-form.tsx
  report-generate-button.tsx
  report-snapshot-table.tsx
  report-snapshot-viewer.tsx       — Rendered report content
  report-schedule-input.tsx        — Cron with preview
  report-section-selector.tsx      — Section checklist
```

---

## PART E — Executive Alerts (Full Management)

### E1. Alerts Page (`visus/alerts/page.tsx`)

Enhance with:
- **Alert Stats** → `GET /api/v1/visus/alerts/stats` — Summary cards by category, severity, status
- **Alert Count** → `GET /api/v1/visus/alerts/count`
- DataTable → `GET /api/v1/visus/alerts`:
  - Columns: title, category badge, severity badge, status, source_suite, occurrence_count, first_seen, last_seen
  - Filters: category, severity, status, source_suite, date range
- Per-row actions: View Detail, Update Status, Dismiss

### E2. Alert Detail (slide-out panel)

- Full description
- Category, severity, status
- Source info: suite, type, entity link
- Occurrence count, first/last seen
- Linked KPI and dashboard (if any)
- Action history (viewed_at/by, actioned_at/by, dismissed_at/by)
- **Status Actions:**
  - Acknowledge → status to `acknowledged`
  - Action → status to `actioned` with action_notes
  - Dismiss → status to `dismissed` with dismiss_reason
  - Escalate → status to `escalated`

**API calls:**
```
GET    /api/v1/visus/alerts              — List
GET    /api/v1/visus/alerts/count        — Count
GET    /api/v1/visus/alerts/stats        — Statistics
GET    /api/v1/visus/alerts/{id}         — Get detail
PUT    /api/v1/visus/alerts/{id}/status  — Update status { status, action_notes?, dismiss_reason? }
```

**Components:**
```
components/visus/alerts/
  alert-columns.tsx
  alert-detail-panel.tsx
  alert-status-dialog.tsx
  alert-stats-cards.tsx
  alert-category-badge.tsx
```

---

## Sidebar Navigation

```
Visus
├── Executive Overview  (new — cross-suite summary)
├── Dashboards         (enhance — full builder with widgets)
├── KPIs               (enhance — CRUD + history + snapshots)
├── Reports            (enhance — CRUD + generation + snapshots)
└── Alerts             (enhance — full management)
```

Add route for dashboards detail:
```typescript
VISUS_DASHBOARDS: '/visus/dashboards',  // list
// Detail: /visus/dashboards/[id]       // builder
```

---

## Dependencies

Consider adding:
- `react-grid-layout` — For drag-and-drop dashboard widget grid
- Or implement a simpler CSS Grid approach with drag-and-drop using `@dnd-kit/core`

If adding a dependency is undesirable, implement a simplified grid without drag-and-drop — use a static grid with manual position configuration in the widget config dialog.

---

## Testing Requirements

- Test dashboard builder: add widget → configure → render → resize → save layout
- Test each widget type renders correctly with its data type
- Test widget data auto-refresh with configurable interval
- Test KPI CRUD: create → list → edit → snapshot → history chart
- Test report generation: create → generate → view snapshot
- Test alert management: view → acknowledge → action → dismiss
- Test executive summary: all suite health indicators render
- Test share dialog: user selection, visibility toggle
- Test dashboard duplication
- Test responsive layout: grid adapts to viewport

## Validation Checklist

- [ ] `npm run build` passes with zero errors
- [ ] Every backend endpoint under `/api/v1/visus/*` has a corresponding frontend call
- [ ] All 13 widget types render correctly with appropriate data
- [ ] Dashboard grid layout saves and loads correctly
- [ ] KPI threshold colors (normal/warning/critical) display correctly
- [ ] Report generation shows loading state and renders result
- [ ] Alert status transitions work correctly
- [ ] Executive summary shows real-time suite health
- [ ] Widget auto-refresh works at configured intervals
- [ ] Dashboard share/duplicate functions work
