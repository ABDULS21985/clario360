# PROMPT 22 — AI Governance Control Plane: Model Registry, Lifecycle, Drift & Shadow Testing

## Objective

Complete the AI Governance Control Plane frontend. A basic page exists at `/admin/ai-governance` and `/admin/ai-governance/[modelId]`, but the backend provides a full AI model governance platform with model registry, version management, lifecycle promotion, shadow testing, drift detection, performance monitoring, prediction logging, and explanations — most of which are not yet implemented.

## Reference Architecture

- **Stack**: Next.js 14 App Router, TypeScript, Zustand, React Query, Zod, react-hook-form, Tailwind, shadcn/ui, Recharts
- **API helpers**: `apiGet`, `apiPost`, `apiPut` from `lib/api.ts`
- **Permissions**: `<PermissionRedirect permission="admin:read">` (AI governance is an admin feature)
- **WebSocket topics**: `ai.model.registered`, `ai.model.promoted`, `ai.model.retired`, `ai.drift.detected`, `ai.shadow.completed`, `ai.performance.alert`

## Types to Create (`types/ai-governance.ts`) — NEW FILE

```typescript
// ─── Model Registry ──────────────────────────────────────────────────────────
export type AIModelType = 'classification' | 'regression' | 'anomaly_detection' | 'nlp' | 'recommendation' | 'custom';
export type AIModelStatus = 'draft' | 'staging' | 'production' | 'retired' | 'archived';
export type AIModelFramework = 'tensorflow' | 'pytorch' | 'scikit-learn' | 'xgboost' | 'custom';

export interface AIModel {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: AIModelType;
  framework: AIModelFramework;
  suite: string;  // cyber, data, acta, lex, visus
  use_case: string;
  owner_id: string;
  owner_name: string;
  current_version_id?: string;
  current_version_number?: number;
  status: AIModelStatus;
  tags: string[];
  metadata: Record<string, unknown>;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export type AIModelVersionStatus = 'draft' | 'testing' | 'staging' | 'production' | 'retired';

export interface AIModelVersion {
  id: string;
  model_id: string;
  version: number;
  status: AIModelVersionStatus;
  description: string;
  metrics: Record<string, number>;  // accuracy, precision, recall, f1, etc.
  hyperparameters: Record<string, unknown>;
  training_data_info: {
    dataset: string;
    records: number;
    features: number;
    split_ratio: string;
  };
  artifact_path?: string;
  artifact_size_bytes?: number;
  promoted_at?: string;
  promoted_by?: string;
  retired_at?: string;
  retired_by?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────
export interface LifecycleEvent {
  id: string;
  model_id: string;
  version_id?: string;
  action: 'registered' | 'version_created' | 'promoted' | 'retired' | 'rolled_back' | 'shadow_started' | 'shadow_stopped';
  from_status?: string;
  to_status?: string;
  actor_id: string;
  actor_name: string;
  reason?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

// ─── Shadow Testing ─────────────────────────────────────────────────────────
export interface ShadowComparison {
  id: string;
  model_id: string;
  production_version_id: string;
  shadow_version_id: string;
  total_predictions: number;
  agreement_rate: number;
  divergence_rate: number;
  production_accuracy?: number;
  shadow_accuracy?: number;
  latency_p50_production_ms: number;
  latency_p50_shadow_ms: number;
  latency_p99_production_ms: number;
  latency_p99_shadow_ms: number;
  started_at: string;
  stopped_at?: string;
  status: 'running' | 'stopped' | 'completed';
  created_at: string;
}

export interface ShadowDivergence {
  id: string;
  model_id: string;
  prediction_id: string;
  input_summary: string;
  production_output: string;
  shadow_output: string;
  production_confidence: number;
  shadow_confidence: number;
  is_significant: boolean;
  created_at: string;
}

// ─── Drift Detection ────────────────────────────────────────────────────────
export type DriftType = 'data_drift' | 'concept_drift' | 'prediction_drift' | 'performance_drift';
export type DriftSeverity = 'none' | 'low' | 'medium' | 'high' | 'critical';

export interface DriftReport {
  id: string;
  model_id: string;
  version_id: string;
  drift_type: DriftType;
  severity: DriftSeverity;
  score: number;  // 0-1
  threshold: number;
  features_drifted: Array<{
    feature: string;
    score: number;
    baseline_distribution: Record<string, number>;
    current_distribution: Record<string, number>;
  }>;
  recommendation: string;
  detected_at: string;
  window_start: string;
  window_end: string;
}

// ─── Performance ─────────────────────────────────────────────────────────────
export interface PerformanceMetrics {
  model_id: string;
  version_id: string;
  period: string;
  accuracy: number;
  precision: number;
  recall: number;
  f1_score: number;
  auc_roc?: number;
  latency_p50_ms: number;
  latency_p99_ms: number;
  total_predictions: number;
  error_rate: number;
  trend: Array<{
    date: string;
    accuracy: number;
    latency_p50_ms: number;
    predictions: number;
  }>;
}

// ─── Predictions & Explanations ──────────────────────────────────────────────
export interface AIPrediction {
  id: string;
  model_id: string;
  model_name: string;
  version_id: string;
  suite: string;
  entity_type: string;
  entity_id: string;
  input_summary: string;
  output: unknown;
  confidence: number;
  latency_ms: number;
  feedback?: 'correct' | 'incorrect' | null;
  created_at: string;
}

export interface AIExplanation {
  id: string;
  prediction_id: string;
  model_id: string;
  method: 'shap' | 'lime' | 'feature_importance' | 'attention';
  feature_contributions: Array<{
    feature: string;
    value: unknown;
    contribution: number;
    direction: 'positive' | 'negative';
  }>;
  summary: string;
  created_at: string;
}

// ─── Dashboard ───────────────────────────────────────────────────────────────
export interface AIGDashboard {
  total_models: number;
  models_in_production: number;
  total_predictions_today: number;
  avg_accuracy: number;
  drift_alerts: number;
  shadow_tests_running: number;
  models_by_suite: Record<string, number>;
  models_by_status: Record<string, number>;
  recent_events: LifecycleEvent[];
  performance_summary: Array<{
    model_id: string;
    model_name: string;
    accuracy: number;
    predictions_today: number;
    drift_severity: DriftSeverity;
  }>;
}
```

## Constants to Add (`lib/constants.ts`)

```typescript
// AI Governance
AI_MODELS: '/api/v1/ai/models',
AI_PREDICTIONS: '/api/v1/ai/predictions',
AI_EXPLANATIONS: '/api/v1/ai/explanations',
AI_DASHBOARD: '/api/v1/ai/dashboard',
```

---

## PART A — AI Governance Dashboard (`admin/ai-governance/page.tsx`)

Replace/enhance existing page with full dashboard from `GET /api/v1/ai/dashboard`.

**Layout:**
1. **KPI Cards Row:**
   - Total Models (count)
   - Models in Production (count with badge)
   - Total Predictions Today (formatted)
   - Average Accuracy (percentage gauge)
   - Active Drift Alerts (warning badge)
   - Shadow Tests Running (count)

2. **Two-Column Grid:**
   - Left: **Models by Suite** → bar chart
   - Left: **Models by Status** → donut chart
   - Right: **Performance Summary** → DataTable with model name, accuracy, predictions today, drift severity badge
   - Right: **Recent Lifecycle Events** → timeline

3. **Quick Actions:**
   - Register New Model button
   - View Predictions button
   - View Drift Alerts button

**Components:**
```
components/admin/ai-governance/
  ai-dashboard-kpis.tsx
  ai-models-by-suite-chart.tsx
  ai-models-by-status-chart.tsx
  ai-performance-summary-table.tsx
  ai-lifecycle-timeline.tsx
```

---

## PART B — Model Registry (Full CRUD + Versions)

### B1. Model List (embedded in dashboard or separate tab)

- DataTable: name, type, framework, suite, status, current version, owner, created_at
- Filters: suite, type, framework, status
- **Register Model** button → create dialog
- Per-row actions: View, Edit, Promote, Retire, Delete

### B2. Register Model Dialog

Form fields:
- Name (required), Description
- Type (select from `AIModelType`)
- Framework (select from `AIModelFramework`)
- Suite (select: cyber, data, acta, lex, visus)
- Use Case (text description)
- Owner (user combobox)
- Tags

**API:** `POST /api/v1/ai/models`

### B3. Model Detail Page (`admin/ai-governance/[modelId]/page.tsx`) — ENHANCE

**Header:**
- Model name, type badge, framework badge, status badge
- Suite indicator, owner name
- Current version number

**Tabs:**

#### Tab 1: Overview
- Model details, description, use case
- Current version metrics summary
- Quick lifecycle actions (Promote, Retire, Rollback)

#### Tab 2: Versions
- `GET /api/v1/ai/models/{id}/versions` → DataTable:
  - Columns: version number, status, accuracy, precision, recall, f1, training dataset, artifact size, created_at
- **Create Version** → `POST /api/v1/ai/models/{id}/versions` — Form:
  - Description
  - Metrics (key-value pairs: accuracy, precision, recall, f1, etc.)
  - Hyperparameters (JSON editor)
  - Training Data Info (dataset name, records, features, split ratio)
- **Version Detail** → `GET .../versions/{vid}` — Full metrics, hyperparameters, training info
- **Promote Version** → `POST /api/v1/ai/models/{id}/versions/{vid}/promote` — Confirm dialog with reason
- **Retire Version** → `POST /api/v1/ai/models/{id}/versions/{vid}/retire` — Confirm with reason

#### Tab 3: Lifecycle History
- `GET /api/v1/ai/models/{id}/lifecycle-history` → Timeline:
  - Each event: action, from_status → to_status, actor, reason, timestamp
- **Rollback** button → `POST /api/v1/ai/models/{id}/rollback` — Rolls back to previous version

#### Tab 4: Shadow Testing
- **Start Shadow Test** → `POST /api/v1/ai/models/{id}/shadow/start` — Dialog:
  - Shadow Version (select from staging/draft versions)
  - Confirm production version will continue serving, shadow runs in parallel
- **Stop Shadow Test** → `POST /api/v1/ai/models/{id}/shadow/stop`
- **Latest Comparison** → `GET /api/v1/ai/models/{id}/shadow/comparison`:
  - Agreement rate gauge
  - Divergence rate
  - Side-by-side accuracy comparison (production vs shadow)
  - Latency comparison (p50, p99) — bar chart
  - Total predictions compared
  - Status indicator (running/stopped/completed)
- **Comparison History** → `GET .../shadow/comparison/history` — Table of past comparisons
- **Divergences** → `GET .../shadow/divergences` — DataTable:
  - Columns: input_summary, production_output, shadow_output, production_confidence, shadow_confidence, is_significant
  - Filter: significant only

#### Tab 5: Drift Detection
- **Latest Drift Report** → `GET /api/v1/ai/models/{id}/drift`:
  - Drift type, severity badge, score gauge
  - Threshold indicator
  - Feature drift table: feature name, drift score, baseline vs current distribution (mini bar charts)
  - Recommendation text
  - Detection window (start → end)
- **Drift History** → `GET .../drift/history` — Chart showing drift score over time with threshold line
  - Click any point → view that drift report

#### Tab 6: Performance
- `GET /api/v1/ai/models/{id}/performance` → `PerformanceMetrics`:
  - Metric cards: accuracy, precision, recall, F1, AUC-ROC
  - Latency chart: p50, p99 over time
  - Prediction volume chart: daily predictions
  - Error rate chart
  - Trend analysis: accuracy + latency + volume over time (multi-axis line chart)

**API calls:**
```
POST   /api/v1/ai/models                                    — Register model
GET    /api/v1/ai/models                                    — List models
GET    /api/v1/ai/models/{id}                               — Get model detail
PUT    /api/v1/ai/models/{id}                               — Update model
POST   /api/v1/ai/models/{id}/versions                      — Create version
GET    /api/v1/ai/models/{id}/versions                      — List versions
GET    /api/v1/ai/models/{id}/versions/{vid}                — Version detail
POST   /api/v1/ai/models/{id}/versions/{vid}/promote        — Promote
POST   /api/v1/ai/models/{id}/versions/{vid}/retire         — Retire
POST   /api/v1/ai/models/{id}/rollback                      — Rollback
GET    /api/v1/ai/models/{id}/lifecycle-history              — History
POST   /api/v1/ai/models/{id}/shadow/start                  — Start shadow
POST   /api/v1/ai/models/{id}/shadow/stop                   — Stop shadow
GET    /api/v1/ai/models/{id}/shadow/comparison              — Latest comparison
GET    /api/v1/ai/models/{id}/shadow/comparison/history      — Comparison history
GET    /api/v1/ai/models/{id}/shadow/divergences             — Divergences
GET    /api/v1/ai/models/{id}/drift                          — Latest drift
GET    /api/v1/ai/models/{id}/drift/history                  — Drift history
GET    /api/v1/ai/models/{id}/performance                    — Performance metrics
```

**Components:**
```
components/admin/ai-governance/
  models/
    model-columns.tsx
    model-register-dialog.tsx
    model-edit-form.tsx
    model-overview.tsx
    model-status-badge.tsx

  versions/
    version-table.tsx
    version-create-dialog.tsx
    version-detail-panel.tsx
    version-metrics-display.tsx    — Accuracy, precision, recall, F1 cards
    version-promote-dialog.tsx
    version-retire-dialog.tsx

  lifecycle/
    lifecycle-timeline.tsx
    rollback-dialog.tsx

  shadow/
    shadow-start-dialog.tsx
    shadow-stop-button.tsx
    shadow-comparison-view.tsx     — Agreement gauge, divergence rate, latency comparison
    shadow-comparison-history.tsx
    shadow-divergence-table.tsx

  drift/
    drift-report-view.tsx          — Severity gauge, feature drift table
    drift-feature-chart.tsx        — Baseline vs current distribution
    drift-history-chart.tsx        — Drift score over time with threshold

  performance/
    performance-metrics-cards.tsx
    performance-latency-chart.tsx
    performance-volume-chart.tsx
    performance-error-rate-chart.tsx
    performance-trend-chart.tsx    — Multi-axis: accuracy + latency + volume
```

---

## PART C — Predictions & Explanations

### C1. Predictions Page (`admin/ai-governance/predictions/page.tsx`) — NEW PAGE

- DataTable → `GET /api/v1/ai/predictions`:
  - Columns: model_name, suite badge, entity_type, confidence (progress bar), latency_ms, feedback badge, created_at
  - Filters: model, suite, entity_type, confidence range, date range, feedback status
- **Prediction Detail** → click row → slide-out:
  - Full input/output display
  - Confidence gauge
  - Latency
  - **Provide Feedback** → mark as correct/incorrect (if feedback feature exists)
  - Link to explanation

### C2. Explanations Page (`admin/ai-governance/explanations/page.tsx`) — NEW PAGE

- `GET /api/v1/ai/explanations` → DataTable:
  - Columns: prediction_id, model, method, summary (truncated), created_at
- **Explanation Detail** → slide-out:
  - Method badge (SHAP, LIME, Feature Importance, Attention)
  - Feature Contributions → horizontal bar chart:
    - Green bars = positive contribution
    - Red bars = negative contribution
    - Sorted by absolute contribution
  - Summary text
  - Link to original prediction

**Components:**
```
components/admin/ai-governance/
  predictions/
    prediction-columns.tsx
    prediction-detail-panel.tsx
    prediction-feedback-button.tsx

  explanations/
    explanation-columns.tsx
    explanation-detail-panel.tsx
    explanation-feature-chart.tsx   — Horizontal bar chart of feature contributions
```

---

## Sidebar Navigation

Update Admin section:
```
Admin
├── Users
├── Roles
├── Tenants
├── API Keys
├── Audit Logs
└── AI Governance
    ├── Dashboard         (enhance)
    ├── Models            (enhance — full registry)
    ├── Predictions       (NEW)
    └── Explanations      (NEW)
```

Add pages:
```
app/(dashboard)/admin/ai-governance/page.tsx             — Dashboard (enhance)
app/(dashboard)/admin/ai-governance/[modelId]/page.tsx   — Model detail (enhance)
app/(dashboard)/admin/ai-governance/predictions/page.tsx — NEW
app/(dashboard)/admin/ai-governance/explanations/page.tsx — NEW
```

---

## Testing Requirements

- Test model registration and version creation
- Test lifecycle: register → create version → promote → shadow test → compare → promote or retire
- Test shadow testing: start → view comparison → view divergences → stop
- Test drift detection: view report → check feature drift charts → view history
- Test performance monitoring: all charts render with data
- Test prediction list with filters and detail panel
- Test explanation feature contribution chart renders correctly
- Test rollback workflow
- Test loading, error, empty states

## Validation Checklist

- [ ] `npm run build` passes with zero errors
- [ ] Every backend endpoint under `/api/v1/ai/*` has a corresponding frontend call
- [ ] Model lifecycle state machine is correctly enforced
- [ ] Shadow testing comparison renders agreement gauge and latency bars
- [ ] Drift feature charts show baseline vs current distributions
- [ ] Performance trend chart has multiple Y-axes (accuracy, latency, volume)
- [ ] Explanation feature contribution chart is bidirectional (positive/negative)
- [ ] Prediction confidence renders as progress bar with color coding
- [ ] All admin pages are gated behind admin permissions
