# PROMPT 17 — Data Suite: Full CRUD, Pipelines, Quality, Lineage & Analytics

## Objective

Complete the Data Suite frontend to cover **every** backend endpoint. Currently the frontend has list views only — no CRUD operations, no pipeline execution, no quality rule management, no contradiction resolution, no dark data governance, and no interactive analytics.

## Reference Architecture

- **Stack**: Next.js 14 App Router, TypeScript, Zustand, React Query, Zod, react-hook-form, Tailwind, shadcn/ui, Recharts
- **API helpers**: `apiGet`, `apiPost`, `apiPut`, `apiPatch`, `apiDelete` from `lib/api.ts`
- **Suite helpers**: `fetchSuitePaginated`, `fetchSuiteData` from `lib/suite-api.ts`
- **Data table**: `useDataTable` hook + `DataTable` component
- **Permissions**: `<PermissionRedirect permission="data:read">` (or `data:write` for mutations)
- **WebSocket topics**: `datasource.created`, `datasource.updated`, `pipeline.started`, `pipeline.completed`, `pipeline.failed`, `quality.completed`, `darkdata.updated`, `contradiction.found`

## Types to Add/Extend (`types/data.ts`) — NEW FILE

Create `frontend/src/types/data.ts` with types mirroring the backend. The existing `types/suites.ts` has basic stubs — create full types:

```typescript
// ─── Data Source ──────────────────────────────────────────────────────────────
export type DataSourceType = 'postgresql' | 'mysql' | 'snowflake' | 'bigquery' | 'mongodb' | 'redis' | 's3' | 'gcs' | 'azure_blob' | 'kafka' | 'elasticsearch' | 'api' | 'file' | 'custom';
export type DataSourceStatus = 'active' | 'inactive' | 'error' | 'testing' | 'discovering';

export interface DataSourceConnectionConfig {
  host?: string;
  port?: number;
  database?: string;
  username?: string;
  password?: string;  // masked in responses
  ssl_mode?: string;
  connection_string?: string;
  [key: string]: unknown;
}

export interface DataSource {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: DataSourceType;
  connection_config: DataSourceConnectionConfig;
  status: DataSourceStatus;
  last_tested_at?: string | null;
  last_sync_at?: string | null;
  sync_frequency?: string | null;
  schema_discovered: boolean;
  table_count?: number;
  record_count?: number;
  tags: string[];
  metadata: Record<string, unknown>;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface DataSourceStats {
  total: number;
  by_type: Record<string, number>;
  by_status: Record<string, number>;
  active: number;
  errored: number;
}

export interface DataSourceTypeInfo {
  type: DataSourceType;
  name: string;
  description: string;
  icon: string;
  config_schema: Record<string, unknown>;  // JSON Schema for connection config
  features: string[];
}

export interface DiscoveredSchema {
  tables: Array<{
    name: string;
    columns: Array<{ name: string; type: string; nullable: boolean }>;
    row_count?: number;
    size_bytes?: number;
  }>;
  discovered_at: string;
}

export interface SyncHistoryEntry {
  id: string;
  source_id: string;
  status: 'running' | 'completed' | 'failed';
  records_synced: number;
  errors: number;
  started_at: string;
  completed_at?: string;
  error_message?: string;
}

// ─── Pipeline ────────────────────────────────────────────────────────────────
export type PipelineStatus = 'active' | 'paused' | 'draft' | 'error' | 'archived';
export type PipelineRunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

export interface PipelineStep {
  id: string;
  name: string;
  type: 'extract' | 'transform' | 'load' | 'quality_check' | 'custom';
  config: Record<string, unknown>;
  order: number;
}

export interface DataPipeline {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  source_id?: string;
  source_name?: string;
  destination_id?: string;
  destination_name?: string;
  steps: PipelineStep[];
  schedule?: string;  // cron expression
  status: PipelineStatus;
  last_run_at?: string | null;
  last_run_status?: PipelineRunStatus;
  total_runs: number;
  success_rate?: number;
  avg_duration_seconds?: number;
  tags: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface PipelineRun {
  id: string;
  pipeline_id: string;
  status: PipelineRunStatus;
  records_processed: number;
  records_failed: number;
  started_at: string;
  completed_at?: string;
  duration_seconds?: number;
  error_message?: string;
  step_results: Array<{
    step_id: string;
    step_name: string;
    status: string;
    records_in: number;
    records_out: number;
    duration_ms: number;
    error?: string;
  }>;
}

export interface PipelineRunLog {
  timestamp: string;
  level: 'info' | 'warn' | 'error' | 'debug';
  message: string;
  step_id?: string;
}

export interface PipelineStats {
  total: number;
  by_status: Record<string, number>;
  active: number;
  paused: number;
  errored: number;
  total_runs_today: number;
  success_rate: number;
}

// ─── Data Model ──────────────────────────────────────────────────────────────
export interface DataModelField {
  name: string;
  type: string;
  description?: string;
  nullable: boolean;
  primary_key: boolean;
  foreign_key?: { table: string; column: string };
  constraints?: string[];
}

export interface DataModel {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  source_id?: string;
  source_name?: string;
  table_name?: string;
  fields: DataModelField[];
  version: number;
  record_count?: number;
  tags: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface DataModelVersion {
  version: number;
  fields: DataModelField[];
  change_summary?: string;
  created_by: string;
  created_at: string;
}

// ─── Quality ─────────────────────────────────────────────────────────────────
export type QualityRuleType = 'completeness' | 'accuracy' | 'consistency' | 'timeliness' | 'uniqueness' | 'validity' | 'custom';

export interface DataQualityRule {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: QualityRuleType;
  model_id?: string;
  model_name?: string;
  field_name?: string;
  condition: string;  // SQL-like expression
  threshold: number;  // 0-100 pass threshold
  severity: 'critical' | 'high' | 'medium' | 'low';
  enabled: boolean;
  schedule?: string;
  last_run_at?: string;
  last_result?: 'pass' | 'fail' | 'error';
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface DataQualityResult {
  id: string;
  rule_id: string;
  rule_name: string;
  rule_type: QualityRuleType;
  model_name?: string;
  passed: boolean;
  score: number;
  records_checked: number;
  records_passed: number;
  records_failed: number;
  failure_samples?: Array<Record<string, unknown>>;
  executed_at: string;
  duration_ms: number;
}

export interface DataQualityDashboard {
  overall_score: number;
  by_dimension: Record<string, number>;
  trend: Array<{ date: string; score: number }>;
  rules_total: number;
  rules_passing: number;
  rules_failing: number;
  recent_results: DataQualityResult[];
}

// ─── Contradiction ───────────────────────────────────────────────────────────
export type ContradictionStatus = 'open' | 'investigating' | 'resolved' | 'dismissed';

export interface DataContradiction {
  id: string;
  tenant_id: string;
  title: string;
  description: string;
  source_a: { source_id: string; source_name: string; field: string; value: string };
  source_b: { source_id: string; source_name: string; field: string; value: string };
  severity: 'critical' | 'high' | 'medium' | 'low';
  status: ContradictionStatus;
  resolution_notes?: string;
  resolved_by?: string;
  resolved_at?: string;
  scan_id: string;
  created_at: string;
  updated_at: string;
}

export interface ContradictionScan {
  id: string;
  status: 'running' | 'completed' | 'failed';
  contradictions_found: number;
  sources_compared: number;
  started_at: string;
  completed_at?: string;
}

export interface ContradictionStats {
  total: number;
  by_status: Record<string, number>;
  by_severity: Record<string, number>;
  open: number;
  resolved: number;
}

// ─── Lineage ─────────────────────────────────────────────────────────────────
export interface LineageNode {
  id: string;
  type: 'source' | 'model' | 'pipeline' | 'report' | 'dashboard';
  name: string;
  metadata?: Record<string, unknown>;
}

export interface LineageEdge {
  id: string;
  source_id: string;
  target_id: string;
  relationship: string;
  metadata?: Record<string, unknown>;
}

export interface LineageGraph {
  nodes: LineageNode[];
  edges: LineageEdge[];
}

export interface LineageImpactAnalysis {
  entity_type: string;
  entity_id: string;
  entity_name: string;
  downstream_count: number;
  affected_entities: Array<{
    type: string;
    id: string;
    name: string;
    distance: number;
    impact_level: 'direct' | 'indirect';
  }>;
}

// ─── Dark Data ───────────────────────────────────────────────────────────────
export type DarkDataStatus = 'unreviewed' | 'under_review' | 'governed' | 'archived' | 'deleted';
export type DarkDataStrategy = 'unmodeled_table' | 'orphaned_file' | 'stale_asset' | 'unused_field' | 'shadow_copy';

export interface DarkDataAsset {
  id: string;
  tenant_id: string;
  name: string;
  source_id?: string;
  source_name?: string;
  strategy: DarkDataStrategy;
  status: DarkDataStatus;
  description: string;
  size_bytes?: number;
  record_count?: number;
  last_accessed_at?: string;
  owner?: string;
  risk_score: number;
  recommendation: string;
  scan_id: string;
  governed_at?: string;
  governed_by?: string;
  created_at: string;
  updated_at: string;
}

export interface DarkDataStats {
  total: number;
  by_status: Record<string, number>;
  by_strategy: Record<string, number>;
  total_size_bytes: number;
  unreviewed: number;
}

export interface DarkDataDashboard {
  stats: DarkDataStats;
  by_source: Record<string, number>;
  risk_distribution: Record<string, number>;
  recent_scans: Array<{ id: string; status: string; found: number; started_at: string }>;
}

// ─── Analytics ───────────────────────────────────────────────────────────────
export interface AnalyticsQuery {
  query: string;
  model_id?: string;
  parameters?: Record<string, unknown>;
}

export interface AnalyticsResult {
  columns: Array<{ name: string; type: string }>;
  rows: Array<Record<string, unknown>>;
  row_count: number;
  execution_time_ms: number;
}

export interface SavedQuery {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  query: string;
  model_id?: string;
  tags: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
}
```

## Constants to Add (`lib/constants.ts`)

```typescript
// Data — Source Types
DATA_SOURCE_TYPES: '/api/v1/data/source-types',
DATA_SOURCES_TEST_CONFIG: '/api/v1/data/sources/test-config',

// Data — Pipeline Runs
DATA_PIPELINES_ACTIVE: '/api/v1/data/pipelines/active',

// Data — Model
DATA_MODELS_DERIVE: '/api/v1/data/models/derive',

// Data — Contradictions
DATA_CONTRADICTIONS_SCAN: '/api/v1/data/contradictions/scan',
DATA_CONTRADICTIONS_SCANS: '/api/v1/data/contradictions/scans',
DATA_CONTRADICTIONS_DASHBOARD: '/api/v1/data/contradictions/dashboard',

// Data — Dark Data
DATA_DARK_DATA_SCAN: '/api/v1/data/dark-data/scan',
DATA_DARK_DATA_SCANS: '/api/v1/data/dark-data/scans',
DATA_DARK_DATA_DASHBOARD: '/api/v1/data/dark-data/dashboard',

// Data — Lineage (entity-level)
DATA_LINEAGE_SEARCH: '/api/v1/data/lineage/search',
```

---

## PART A — Data Source Management (Full CRUD)

### A1. Source List Page Enhancements (`data/sources/page.tsx`)

- **Create Source** button → opens multi-step wizard dialog
- Stats bar from `GET /api/v1/data/sources/stats`
- Filter tabs by status (active, inactive, error)
- Filter by type
- Per-row actions: Edit, Test Connection, Discover Schema, Sync, Delete

### A2. Create Source Wizard Dialog

Multi-step flow:
1. **Select Type** → `GET /api/v1/data/source-types` — Grid of source type cards with icons
2. **Configure Connection** → Dynamic form based on `source-types/{type}` config schema
3. **Test Connection** → `POST /api/v1/data/sources/test-config` — Show success/error
4. **Save** → `POST /api/v1/data/sources`

### A3. Source Detail Page (`data/sources/[id]/page.tsx`)

**Tabs:**
1. **Overview** — Name, type, status, connection info (masked), stats
2. **Schema** → `GET /api/v1/data/sources/{id}/schema` — Tree view of tables/columns; **Discover** button → `POST .../discover`
3. **Sync History** → `GET /api/v1/data/sources/{id}/sync-history` — DataTable; **Sync Now** button → `POST .../sync`
4. **Statistics** → `GET /api/v1/data/sources/{id}/stats` — Charts (record counts, last sync times)
5. **Settings** — Edit connection config, sync schedule

**Actions:**
- Test Connection → `POST /api/v1/data/sources/{id}/test`
- Change Status → `PATCH /api/v1/data/sources/{id}/status`
- Delete → `DELETE /api/v1/data/sources/{id}`

**Components:**
```
components/data/sources/
  source-columns.tsx
  source-type-grid.tsx            — Grid of connectable source types
  source-config-form.tsx          — Dynamic form based on source type
  source-create-wizard.tsx        — Multi-step creation wizard
  source-detail-overview.tsx
  source-schema-tree.tsx          — Table/column tree view
  source-sync-history-table.tsx
  source-stats-panel.tsx
  source-status-badge.tsx
  source-test-button.tsx          — Connection test with result indicator
```

---

## PART B — Pipeline Management (Full CRUD + Execution)

### B1. Pipeline List Page Enhancements (`data/pipelines/page.tsx`)

- **Create Pipeline** button
- Stats bar from `GET /api/v1/data/pipelines/stats`
- **Active Pipelines** indicator → `GET /api/v1/data/pipelines/active`
- Status filter tabs (active, paused, draft, error)
- Per-row actions: Run, Pause/Resume, Edit, Delete
- Row status indicator showing last run status

### B2. Pipeline Create/Edit Page or Dialog

Form fields:
- Name, Description
- Source (combobox → data sources)
- Destination (combobox → data sources)
- Steps (dynamic array builder):
  - Step type (extract/transform/load/quality_check/custom)
  - Step name
  - Configuration (JSON editor or structured form)
- Schedule (cron expression with human-readable preview)
- Tags

### B3. Pipeline Detail Page (`data/pipelines/[id]/page.tsx`)

**Tabs:**
1. **Overview** — Name, source → destination flow diagram, schedule, stats
2. **Steps** — Visual pipeline step flow (source → step1 → step2 → … → destination)
3. **Run History** → `GET /api/v1/data/pipelines/{id}/runs` — DataTable with status, duration, records processed
4. **Run Detail** → `GET .../runs/{runId}` — Step-by-step results, record counts per step
5. **Logs** → `GET .../runs/{runId}/logs` — Scrollable log viewer with level filtering

**Actions:**
- Run Now → `POST /api/v1/data/pipelines/{id}/run`
- Pause → `POST .../pause`
- Resume → `POST .../resume`
- Edit → `PUT /api/v1/data/pipelines/{id}`
- Delete → `DELETE /api/v1/data/pipelines/{id}`

**Components:**
```
components/data/pipelines/
  pipeline-columns.tsx
  pipeline-create-form.tsx
  pipeline-step-builder.tsx       — Dynamic step array with type-based config
  pipeline-flow-diagram.tsx       — Visual source → steps → destination
  pipeline-run-table.tsx
  pipeline-run-detail.tsx
  pipeline-log-viewer.tsx         — Scrollable logs with level filter
  pipeline-schedule-input.tsx     — Cron expression with preview
  pipeline-status-badge.tsx
```

---

## PART C — Data Model Management (CRUD + Versioning)

### C1. Model List Page Enhancements (`data/models/page.tsx`)

- **Create Model** button
- **Derive from Source** button → `POST /api/v1/data/models/derive`
- Per-row actions: Edit, Validate, View Versions, View Lineage, Delete

### C2. Model Detail Page (`data/models/[id]/page.tsx`)

**Tabs:**
1. **Schema** — Field table (name, type, nullable, PK, FK, constraints) — editable
2. **Versions** → `GET /api/v1/data/models/{id}/versions` — Version history with diff
3. **Lineage** → `GET /api/v1/data/models/{id}/lineage` — Mini lineage graph for this model
4. **Validate** → `POST /api/v1/data/models/{id}/validate` — Validation results

**API calls:**
```
POST   /api/v1/data/models              — Create
PUT    /api/v1/data/models/{id}         — Update (new version)
DELETE /api/v1/data/models/{id}         — Delete
POST   /api/v1/data/models/derive       — Derive from source
POST   /api/v1/data/models/{id}/validate — Validate schema
GET    /api/v1/data/models/{id}/versions — Version history
GET    /api/v1/data/models/{id}/lineage  — Model lineage
```

**Components:**
```
components/data/models/
  model-columns.tsx
  model-create-dialog.tsx
  model-derive-dialog.tsx         — Select source, discover, derive model
  model-schema-editor.tsx         — Editable field table
  model-version-history.tsx       — Version list with diffs
  model-lineage-mini.tsx          — Small lineage graph for one model
  model-validate-results.tsx
```

---

## PART D — Data Quality Management (Rule CRUD + Execution)

### D1. Quality Page Enhancements (`data/quality/page.tsx`)

The page already shows dashboard, score, and results. Add:
- **Rules tab** with full CRUD:
  - Create Rule → `POST /api/v1/data/quality/rules`
  - Edit Rule → `PUT /api/v1/data/quality/rules/{id}`
  - Delete Rule → `DELETE /api/v1/data/quality/rules/{id}`
  - Run Rule → `POST /api/v1/data/quality/rules/{id}/run`
  - Toggle enabled/disabled
- **Quality Score Trend** chart → already fetched, ensure it renders as area chart
- **Result Detail** → `GET /api/v1/data/quality/results/{id}` — Show failure samples

### D2. Quality Rule Create/Edit Dialog

Form fields:
- Name, Description
- Type (completeness/accuracy/consistency/timeliness/uniqueness/validity/custom)
- Model (combobox)
- Field Name (combobox populated from model schema)
- Condition (SQL-like expression textarea)
- Threshold (0-100 slider)
- Severity (select)
- Schedule (cron input)

**Components:**
```
components/data/quality/
  quality-rule-columns.tsx
  quality-rule-create-dialog.tsx
  quality-rule-edit-form.tsx
  quality-result-detail.tsx       — Failure samples table
  quality-score-trend-chart.tsx
  quality-dimension-chart.tsx     — Radar or bar chart by dimension
```

---

## PART E — Contradiction Detection (Scan + Resolution)

### E1. Contradictions Page Enhancements (`data/contradictions/page.tsx`)

- **Scan for Contradictions** button → `POST /api/v1/data/contradictions/scan`
- Stats bar from `GET /api/v1/data/contradictions/stats`
- Dashboard → `GET /api/v1/data/contradictions/dashboard`
- **Scan History** → `GET /api/v1/data/contradictions/scans` — Table with scan status
- Per-contradiction actions:
  - Update Status → `PUT /api/v1/data/contradictions/{id}/status`
  - Resolve → `POST /api/v1/data/contradictions/{id}/resolve` — Dialog with resolution notes, choose which source is correct

### E2. Contradiction Detail View (slide-out panel or page)

Show:
- Source A value vs Source B value (side-by-side comparison)
- Severity, status, created_at
- Resolution form (notes, resolve action)

**Components:**
```
components/data/contradictions/
  contradiction-columns.tsx
  contradiction-scan-button.tsx
  contradiction-scan-history.tsx
  contradiction-detail-panel.tsx  — Side-by-side source comparison
  contradiction-resolve-dialog.tsx
  contradiction-stats-bar.tsx
```

---

## PART F — Data Lineage (Interactive Graph + Impact Analysis)

### F1. Lineage Page Enhancements (`data/lineage/page.tsx`)

The page already shows a basic graph. Enhance with:
- **Interactive graph** (consider react-flow or vis.js) — nodes for sources, models, pipelines; edges for relationships
- **Click node** → side panel with entity details
- **Entity Lineage** → `GET /api/v1/data/lineage/graph/{entityType}/{entityId}` — Focused graph
- **Upstream/Downstream** → `GET .../upstream/{entityType}/{entityId}` and `.../downstream/...`
- **Impact Analysis** → `GET .../impact/{entityType}/{entityId}` — Shows all downstream affected entities
- **Search** → `GET /api/v1/data/lineage/search` — Search nodes by name
- **Record Lineage** → `POST /api/v1/data/lineage/record` — Admin ability to manually add edges
- **Delete Edge** → `DELETE /api/v1/data/lineage/edges/{id}`

**Components:**
```
components/data/lineage/
  lineage-graph.tsx               — Interactive DAG visualization
  lineage-node.tsx                — Rendered node with type icon
  lineage-edge.tsx                — Directed edge with label
  lineage-entity-panel.tsx        — Side panel for selected entity
  lineage-impact-dialog.tsx       — Impact analysis results
  lineage-search-bar.tsx
  lineage-record-dialog.tsx       — Manually add lineage edge
```

---

## PART G — Dark Data Discovery (Scan + Governance)

### G1. Dark Data Page Enhancements (`data/dark-data/page.tsx`)

- **Scan for Dark Data** button → `POST /api/v1/data/dark-data/scan`
- Stats bar from `GET /api/v1/data/dark-data/stats`
- Dashboard → `GET /api/v1/data/dark-data/dashboard`
- **Scan History** → `GET /api/v1/data/dark-data/scans` — Table
- Per-asset actions:
  - Update Status → `PUT /api/v1/data/dark-data/{id}/status`
  - Govern → `POST /api/v1/data/dark-data/{id}/govern` — Moves to governed status, creates data model

### G2. Dark Data Detail (slide-out or dialog)

Show: name, source, strategy type, size, record count, last accessed, risk score, recommendation
Action: Govern button with confirmation

**Components:**
```
components/data/dark-data/
  dark-data-columns.tsx
  dark-data-scan-button.tsx
  dark-data-scan-history.tsx
  dark-data-detail-panel.tsx
  dark-data-govern-dialog.tsx
  dark-data-stats-bar.tsx
  dark-data-dashboard-charts.tsx  — Risk distribution, by source, by strategy
```

---

## PART H — Analytics (Query Builder + Saved Queries)

### H1. Analytics Page Enhancements (`data/analytics/page.tsx`)

Full interactive analytics workspace:

- **Query Editor** — Code editor (Monaco or CodeMirror) with SQL syntax highlighting
- **Model Explorer** → `POST /api/v1/data/analytics/explore/{modelId}` — Browse model data
- **Explain** → `POST /api/v1/data/analytics/explain` — Natural language explanation of data
- **Execute Query** → `POST /api/v1/data/analytics/query` — Results table + execution stats
- **Results Visualization** — Toggle between table view and auto-chart (bar/line based on data shape)
- **Save Query** → `POST /api/v1/data/analytics/saved` — Dialog with name, description, tags
- **Saved Queries** → `GET /api/v1/data/analytics/saved` — Sidebar list
  - Edit → `PUT /api/v1/data/analytics/saved/{id}`
  - Delete → `DELETE /api/v1/data/analytics/saved/{id}`
  - Run → `POST /api/v1/data/analytics/saved/{id}/run`
  - Load into editor
- **Audit Log** → `GET /api/v1/data/analytics/audit` — Query execution history

**Components:**
```
components/data/analytics/
  query-editor.tsx                — SQL editor with syntax highlighting
  query-results-table.tsx         — Dynamic column results table
  query-results-chart.tsx         — Auto-visualization of results
  saved-queries-sidebar.tsx       — List of saved queries
  save-query-dialog.tsx
  model-explorer-panel.tsx        — Browse model schema for reference
  analytics-audit-log.tsx
```

---

## PART I — Dashboard Enhancements

Enhance `data/page.tsx` (dashboard):
- Fetch `GET /api/v1/data/dashboard` and render all sections
- KPI cards: Total sources, Active pipelines, Quality score, Dark data count
- Charts: Quality trend, Pipeline success rate, Source status distribution
- Recent activity: Latest pipeline runs, quality failures, new contradictions

---

## Sidebar Navigation

```
Data
├── Dashboard          (existing — enhance)
├── Sources            (existing — add CRUD)
├── Pipelines          (existing — add CRUD + execution)
├── Models             (existing — add CRUD + versioning)
├── Quality            (existing — add rule CRUD)
├── Contradictions     (existing — add scan + resolution)
├── Lineage            (existing — add interactive graph)
├── Dark Data          (existing — add scan + governance)
└── Analytics          (existing — add full query workspace)
```

---

## Testing Requirements

- All new components must have Vitest + React Testing Library tests
- Use MSW handlers for API mocking
- Test CRUD flows for sources, pipelines, models, quality rules, saved queries
- Test pipeline execution flow: run → monitor → view results → view logs
- Test connection testing flow: configure → test → success/error feedback
- Test contradiction resolution flow
- Test dark data governance flow
- Test analytics query execution and result rendering
- Test loading, error, and empty states

## Validation Checklist

- [ ] `npm run build` passes with zero errors (TypeScript clean)
- [ ] All new pages render without hydration errors
- [ ] Every backend endpoint under `/api/v1/data/*` has a corresponding frontend call
- [ ] DataTable pagination, sorting, filtering work on all list pages
- [ ] Real-time updates via WebSocket invalidate relevant queries
- [ ] All forms validate with Zod before submission
- [ ] Connection test feedback is immediate and clear
- [ ] Pipeline step builder supports all step types
- [ ] Lineage graph is interactive and performant with 100+ nodes
- [ ] Analytics query editor has syntax highlighting and auto-complete
