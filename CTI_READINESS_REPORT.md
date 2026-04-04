# Clario 360 — CTI Implementation Readiness Report

Generated: 2026-04-03

---

## 1. Executive Summary

### Current Cyber-Service Capabilities
- **Asset Management**: Full CRUD with discovery scanning (network/cloud/agent/import), classification, enrichment, bulk operations, relationship graph
- **Vulnerability Management**: CVE tracking, CVSS scoring, remediation lifecycle, aging analytics
- **Threat Intelligence**: Threat tracking with MITRE ATT&CK mapping, IOC management, threat feeds (STIX/TAXII/MISP/CSV), bulk import/export
- **Alert Management**: Detection rule engine (Sigma/threshold/correlation/anomaly), AI-explained alerts, timeline/comments, bulk operations
- **CTEM**: 5-phase continuous threat exposure management assessments with findings, remediation groups, exposure scoring
- **DSPM**: Data classification, PII detection, access mapping, identity profiling, shadow data detection, compliance posture, financial impact, AI data governance, data lineage
- **UEBA**: User/entity behavior profiling, anomaly detection, risk scoring with decay
- **vCISO**: Executive briefings, AI chat (hybrid LLM/rule-based), risk register, policy management, compliance evidence, vendor management, maturity assessments, incident readiness, budget planning, awareness programs, predictive ML engine
- **Remediation**: Governed lifecycle (draft → approval → dry-run → execution → verification → rollback), workflow engine integration
- **Risk Analytics**: Organization risk scoring with daily snapshots, heatmap, top-risk identification, recommendations

### Scale
| Metric | Count |
|--------|-------|
| Backend Go source files | 489 |
| Backend Go test files | 106 |
| Backend Go LOC | ~133,000 |
| Frontend cyber pages/components | 268 .tsx/.ts files |
| Database migrations | 26 |
| Database tables | ~64 |
| API endpoints | 150+ |
| Kafka topics consumed | 11 |

### Database Tables That Exist Today
**Core:** assets, asset_relationships, vulnerabilities, threats, threat_indicators, detection_rules, alerts, alert_comments, alert_timeline, scan_history, cve_database, asset_activity
**Remediation:** remediation_actions, remediation_audit_trail
**CTEM:** ctem_assessments, ctem_findings, ctem_remediation_groups, exposure_score_snapshots, risk_score_history
**DSPM (12):** dspm_data_assets, dspm_scans, dspm_access_mappings, dspm_identity_profiles, dspm_access_audit, dspm_access_policies, dspm_remediations, dspm_remediation_history, dspm_data_policies, dspm_risk_exceptions, dspm_data_lineage, dspm_ai_data_usage, dspm_classification_history, dspm_compliance_posture, dspm_financial_impact
**UEBA (3):** ueba_profiles, ueba_access_events, ueba_alerts
**vCISO (19):** vciso_risks, vciso_policies, vciso_policy_exceptions, vciso_vendors, vciso_questionnaires, vciso_evidence, vciso_maturity_assessments, vciso_budget_items, vciso_awareness_programs, vciso_iam_findings, vciso_escalation_rules, vciso_playbooks, vciso_obligations, vciso_control_tests, vciso_integrations, vciso_benchmarks, vciso_control_dependencies, vciso_control_ownership, vciso_approvals
**vCISO ML (6):** vciso_predictions, vciso_prediction_models, vciso_feature_snapshots, vciso_llm_audit_log, vciso_llm_system_prompts, vciso_llm_rate_limits
**Threat Feeds (2):** threat_feed_configs, threat_feed_sync_history

### Frontend Pages That Exist Today
- SOC Dashboard, Alerts (list + detail), Assets (list + detail), Threats (list + detail), Indicators, Detection Rules, MITRE ATT&CK Coverage, CTEM Assessments, Risk Heatmap, DSPM (6 sub-pages), UEBA (dashboard + alerts + profiles), vCISO (10 sub-pages), Remediation, Threat Feeds, Analytics, Events

### API Endpoints That Exist Today
150+ REST endpoints under `/api/v1/cyber/` covering assets, alerts, threats, indicators, rules, MITRE, dashboard, risk, DSPM, CTEM, events, analytics, remediation, threat feeds, UEBA, vCISO

### Gaps Relative to CTI Requirements
1. **No dedicated CTI module** — threat intelligence is embedded in the existing threat/indicator system but lacks:
   - Global Threat Map (geographic visualization of threats)
   - Active Campaigns & Actors tracking (campaign entity with actor profiles)
   - Brand Abuse Hotspots (brand monitoring, domain impersonation detection)
   - Sector & Geographic Targeting analysis (industry/region correlation)
   - Executive Threat Dashboard (CTI-specific executive view)
2. **No geographic data on threats** — existing `threats` table has no `origin_country`, `target_countries`, or geolocation fields
3. **No campaign entity** — threats have a `campaign` text field but no dedicated campaign table with lifecycle tracking
4. **No threat actor profiles** — `threat_actor` is a plain text field, not a linked entity
5. **No brand abuse detection** — no tables or services for domain monitoring, typosquatting, phishing site tracking
6. **No sector/industry targeting analysis** — no sector classification on threats or indicators
7. **No CTI-specific Kafka topics** — would need `cyber.cti.*` topics for campaign/actor/brand events

---

## 2. Backend Architecture

### 2.1 Service Boot & Config

**File:** `backend/cmd/cyber-service/main.go`

Boot sequence:
1. Signal handling (SIGINT/SIGTERM)
2. `config.Load()` → platform config
3. `cyberconfig.Load()` → cyber-specific config
4. Port override from `CYBER_HTTP_PORT`
5. Kafka brokers/groupID from `CYBER_KAFKA_BROKERS`, `CYBER_KAFKA_GROUP_ID`
6. JWT public key from `CYBER_JWT_PUBLIC_KEY_PATH`
7. `bootstrap.Bootstrap()` → svc.Logger, svc.DBPool, svc.Redis, svc.Router
8. DB migrations from `CYBER_MIGRATIONS_PATH` or `backend/migrations/cyber_db`

Middleware stack on `/api/v1/cyber`:
```go
r.Use(middleware.Auth(jwtMgr))           // JWT validation
r.Use(middleware.Tenant)                 // Tenant isolation
r.Use(cybermw.RateLimiter(rdb, 1200))    // 1200 req/min per tenant
```

Background goroutines:
- Kafka Consumer (11 topic subscriptions)
- Scan Scheduler
- Continuous DSPM (shadow data detection)
- Risk Snapshot Service (daily batch)
- UEBA Scheduler (behavior analytics)
- vCISO Prediction Engine (threat forecasting)

### 2.2 Package Map

```
backend/internal/cyber/
├── classifier/         # Asset criticality classification rules (3 files)
├── config/             # Cyber service config loader (1 file)
├── consumer/           # Kafka event consumers (10 files)
├── ctem/               # CTEM engine: scoping, discovery, prioritization, validation, mobilization (15 files)
├── dashboard/          # Dashboard KPI/metric calculators (6 files)
├── detection/          # Detection engine & rule management (8 files)
├── dto/                # Data transfer objects (25+ files)
├── enrichment/         # Asset enrichment pipeline: DNS, CVE, Geo (5 files)
├── handler/            # HTTP request handlers (30+ files)
├── indicator/          # IOC/STIX parsing (3 files)
├── mitre/              # MITRE ATT&CK framework data (2 files)
├── model/              # Domain models (15+ files)
├── remediation/        # Remediation execution & strategies (8 files)
├── repository/         # Database access layer (25+ files)
├── risk/               # Risk scoring engine (20 files)
├── scanner/            # Network/cloud/agent scanners (5 files)
├── service/            # Business logic services (15+ files)
├── dspm/               # Data Security Posture Management (200+ files)
│   ├── access/         #   Access control analysis
│   ├── intelligence/   #   Data intelligence
│   └── remediation/    #   DSPM remediation
├── ueba/               # User & Entity Behavior Analytics (20+ files)
└── vciso/              # Virtual CISO AI assistant (100+ files)
    ├── chat/           #   Chat interface
    ├── llm/            #   LLM integration
    └── prediction/     #   ML prediction engine
```

### 2.3 Domain Models

**Asset** (`model/asset.go`):
```go
type Asset struct {
    ID              uuid.UUID       `json:"id" db:"id"`
    TenantID        uuid.UUID       `json:"tenant_id" db:"tenant_id"`
    Name            string          `json:"name" db:"name"`
    Type            AssetType       `json:"type" db:"type"`
    IPAddress       *string         `json:"ip_address,omitempty" db:"ip_address"`
    Hostname        *string         `json:"hostname,omitempty" db:"hostname"`
    MACAddress      *string         `json:"mac_address,omitempty" db:"mac_address"`
    OS              *string         `json:"os,omitempty" db:"os"`
    OSVersion       *string         `json:"os_version,omitempty" db:"os_version"`
    Owner           *string         `json:"owner,omitempty" db:"owner"`
    Department      *string         `json:"department,omitempty" db:"department"`
    Location        *string         `json:"location,omitempty" db:"location"`
    Criticality     Criticality     `json:"criticality" db:"criticality"`
    Status          AssetStatus     `json:"status" db:"status"`
    DiscoveredAt    time.Time       `json:"discovered_at" db:"discovered_at"`
    LastSeenAt      time.Time       `json:"last_seen_at" db:"last_seen_at"`
    DiscoverySource string          `json:"discovery_source" db:"discovery_source"`
    LastScanID      *uuid.UUID      `json:"last_scan_id,omitempty" db:"last_scan_id"`
    Metadata        json.RawMessage `json:"metadata" db:"metadata"`
    Tags            []string        `json:"tags" db:"tags"`
    CreatedBy       *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
    CreatedAt       time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
    DeletedAt       *time.Time      `json:"-" db:"deleted_at"`
    // Computed
    OpenVulnerabilityCount int     `json:"open_vulnerability_count" db:"open_vulnerability_count"`
    CriticalVulnCount      int     `json:"critical_vuln_count" db:"critical_vuln_count"`
    HighVulnCount          int     `json:"high_vuln_count" db:"high_vuln_count"`
    AlertCount             int     `json:"alert_count" db:"alert_count"`
    RelationshipCount      int     `json:"relationship_count" db:"relationship_count"`
}
```

**Alert** (`model/alert.go`):
```go
type Alert struct {
    ID                  uuid.UUID          `json:"id" db:"id"`
    TenantID            uuid.UUID          `json:"tenant_id" db:"tenant_id"`
    Title               string             `json:"title" db:"title"`
    Description         string             `json:"description" db:"description"`
    Severity            Severity           `json:"severity" db:"severity"`
    Status              AlertStatus        `json:"status" db:"status"`
    Source              string             `json:"source" db:"source"`
    RuleID              *uuid.UUID         `json:"rule_id,omitempty" db:"rule_id"`
    AssetID             *uuid.UUID         `json:"asset_id,omitempty" db:"asset_id"`
    AssetIDs            []uuid.UUID        `json:"asset_ids" db:"asset_ids"`
    AssignedTo          *uuid.UUID         `json:"assigned_to,omitempty" db:"assigned_to"`
    Explanation         AlertExplanation   `json:"explanation" db:"explanation"`
    ConfidenceScore     float64            `json:"confidence_score" db:"confidence_score"`
    MITRETacticID       *string            `json:"mitre_tactic_id,omitempty" db:"mitre_tactic_id"`
    MITRETechniqueID    *string            `json:"mitre_technique_id,omitempty" db:"mitre_technique_id"`
    EventCount          int                `json:"event_count" db:"event_count"`
    FirstEventAt        time.Time          `json:"first_event_at" db:"first_event_at"`
    LastEventAt         time.Time          `json:"last_event_at" db:"last_event_at"`
    Tags                []string           `json:"tags" db:"tags"`
    Metadata            json.RawMessage    `json:"metadata" db:"metadata"`
    CreatedAt           time.Time          `json:"created_at" db:"created_at"`
    UpdatedAt           time.Time          `json:"updated_at" db:"updated_at"`
    DeletedAt           *time.Time         `json:"-" db:"deleted_at"`
}
```

**Threat** (`model/threat.go`):
```go
type Threat struct {
    ID                 uuid.UUID          `json:"id" db:"id"`
    TenantID           uuid.UUID          `json:"tenant_id" db:"tenant_id"`
    Name               string             `json:"name" db:"name"`
    Description        string             `json:"description" db:"description"`
    Type               ThreatType         `json:"type" db:"type"`
    Severity           Severity           `json:"severity" db:"severity"`
    Status             ThreatStatus       `json:"status" db:"status"`
    ThreatActor        *string            `json:"threat_actor,omitempty" db:"threat_actor"`
    Campaign           *string            `json:"campaign,omitempty" db:"campaign"`
    MITRETacticIDs     []string           `json:"mitre_tactic_ids" db:"mitre_tactic_ids"`
    MITRETechniqueIDs  []string           `json:"mitre_technique_ids" db:"mitre_technique_ids"`
    IndicatorCount     int                `json:"indicator_count" db:"indicator_count"`
    AffectedAssetCount int                `json:"affected_asset_count" db:"affected_asset_count"`
    AlertCount         int                `json:"alert_count" db:"alert_count"`
    FirstSeenAt        time.Time          `json:"first_seen_at" db:"first_seen_at"`
    LastSeenAt         time.Time          `json:"last_seen_at" db:"last_seen_at"`
    Tags               []string           `json:"tags" db:"tags"`
    Metadata           json.RawMessage    `json:"metadata" db:"metadata"`
    CreatedAt          time.Time          `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time          `json:"updated_at" db:"updated_at"`
    DeletedAt          *time.Time         `json:"-" db:"deleted_at"`
    Indicators         []*ThreatIndicator `json:"indicators,omitempty" db:"-"`
}
```

**ThreatIndicator** (`model/threat.go`):
```go
type ThreatIndicator struct {
    ID           uuid.UUID       `json:"id" db:"id"`
    TenantID     uuid.UUID       `json:"tenant_id" db:"tenant_id"`
    ThreatID     *uuid.UUID      `json:"threat_id,omitempty" db:"threat_id"`
    Type         IndicatorType   `json:"type" db:"type"`
    Value        string          `json:"value" db:"value"`
    Description  string          `json:"description" db:"description"`
    Severity     Severity        `json:"severity" db:"severity"`
    Source       string          `json:"source" db:"source"`
    Confidence   float64         `json:"confidence" db:"confidence"`
    Active       bool            `json:"active" db:"active"`
    FirstSeenAt  time.Time       `json:"first_seen_at" db:"first_seen_at"`
    LastSeenAt   time.Time       `json:"last_seen_at" db:"last_seen_at"`
    ExpiresAt    *time.Time      `json:"expires_at,omitempty" db:"expires_at"`
    Tags         []string        `json:"tags" db:"tags"`
    Metadata     json.RawMessage `json:"metadata" db:"metadata"`
    CreatedAt    time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}
```

**Vulnerability** (`model/vulnerability.go`):
```go
type Vulnerability struct {
    ID          uuid.UUID       `json:"id" db:"id"`
    TenantID    uuid.UUID       `json:"tenant_id" db:"tenant_id"`
    AssetID     uuid.UUID       `json:"asset_id" db:"asset_id"`
    CVEID       *string         `json:"cve_id,omitempty" db:"cve_id"`
    Title       string          `json:"title" db:"title"`
    Description string          `json:"description" db:"description"`
    Severity    string          `json:"severity" db:"severity"`
    CVSSScore   *float64        `json:"cvss_score,omitempty" db:"cvss_score"`
    CVSSVector  *string         `json:"cvss_vector,omitempty" db:"cvss_vector"`
    Status      string          `json:"status" db:"status"`
    Source      string          `json:"source" db:"source"`
    Remediation *string         `json:"remediation,omitempty" db:"remediation"`
    Metadata    json.RawMessage `json:"metadata" db:"metadata"`
    CreatedAt   time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
    DeletedAt   *time.Time      `json:"-" db:"deleted_at"`
}
```

### 2.4 Repository Layer

Pattern: `*Repository` struct with `*pgxpool.Pool` + `zerolog.Logger`. All queries tenant-scoped with `AND tenant_id = $N AND deleted_at IS NULL`.

Key repositories:
- `AssetRepository` — CRUD + LEFT JOIN aggregations for vuln/alert counts, FTS on name/hostname/IP
- `VulnerabilityRepository` — CVE tracking, aging queries, CVSS scoring
- `AlertRepository` — Detection results, timeline, comments, bulk status updates
- `ThreatRepository` — Threat CRUD, IOC associations, status transitions
- `IndicatorRepository` — IOC CRUD, enrichment, match checking
- `RuleRepository` — Detection rule CRUD, templates, performance metrics
- `RemediationRepository` — Full lifecycle state machine
- `CTEMAssessmentRepository` — Assessment CRUD with phase tracking
- `DSPMRepository` — Data asset classification, access mapping, identity profiling
- `DashboardRepository` — KPI aggregation queries, trend calculations

SQL patterns: LATERAL subqueries for aggregations, GIN indexes for tags/JSONB, composite indexes for tenant+status filtering, soft deletes via `deleted_at IS NULL`.

### 2.5 Service Layer

Key services with dependencies:

```
AssetService → AssetRepo, VulnRepo, RelRepo, ScanRepo, ActivityRepo, ScanRegistry, Classifier, EnrichmentSvc, Producer, Metrics
AlertService → AlertRepo, CommentRepo, RuleRepo, Producer
ThreatService → ThreatRepo, IndicatorRepo, AlertRepo, EnrichmentCache, Producer
DetectionService → RuleRepo, AlertRepo, AssetRepo
DashboardService → DashboardRepo (KPI aggregation, trends, MTTR/MTTA)
CTEMService → CTEMRepo, AssetRepo, VulnRepo, RuleRepo (5-phase assessment engine)
RemediationService → RemediationRepo, AlertRepo, VulnRepo, WorkflowClient
RiskService → RiskRepo, AssetRepo, VulnRepo, AlertRepo, ThreatRepo
DSPMService → DSPMRepo, AssetRepo (classification, access analysis, shadow detection)
UEBAService → UEBARepo, AlertRepo (behavior profiling, anomaly detection)
VCISOService → VCISORepo, RiskService, AlertService, ThreatService (AI chat, briefings, predictions)
```

### 2.6 Route Table

All routes under `/api/v1/cyber/` with JWT auth + tenant isolation + rate limiting (1200 req/min):

| Method | Path | Handler |
|--------|------|---------|
| **Assets** | | |
| GET | /assets/stats | AssetHandler.GetStats |
| GET | /assets/count | AssetHandler.GetCount |
| POST | /assets/scan | AssetHandler.TriggerScan |
| GET | /assets/scans | AssetHandler.ListScans |
| GET | /assets/scans/{id} | AssetHandler.GetScan |
| POST | /assets/scans/{id}/cancel | AssetHandler.CancelScan |
| POST | /assets/bulk | AssetHandler.BulkCreate |
| PUT | /assets/bulk/tags | AssetHandler.BulkUpdateTags |
| DELETE | /assets/bulk | AssetHandler.BulkDelete |
| POST | /assets | AssetHandler.CreateAsset |
| GET | /assets | AssetHandler.ListAssets |
| GET | /assets/{id} | AssetHandler.GetAsset |
| PUT | /assets/{id} | AssetHandler.UpdateAsset |
| DELETE | /assets/{id} | AssetHandler.DeleteAsset |
| PATCH | /assets/{id}/tags | AssetHandler.PatchTags |
| GET | /assets/{id}/activity | AssetHandler.ListActivity |
| GET | /assets/{id}/relationships | AssetHandler.ListRelationships |
| POST | /assets/{id}/relationships | AssetHandler.CreateRelationship |
| DELETE | /assets/{id}/relationships/{relId} | AssetHandler.DeleteRelationship |
| GET | /assets/{id}/vulnerabilities | AssetHandler.ListVulnerabilities |
| POST | /assets/{id}/vulnerabilities | AssetHandler.CreateVulnerability |
| PUT | /assets/{id}/vulnerabilities/{vid} | AssetHandler.UpdateVulnerability |
| **Alerts** | | |
| PUT | /alerts/bulk/status | AlertHandler.BulkUpdateStatus |
| PUT | /alerts/bulk/assign | AlertHandler.BulkAssign |
| PUT | /alerts/bulk/false-positive | AlertHandler.BulkMarkFalsePositive |
| GET | /alerts/stats | AlertHandler.Stats |
| GET | /alerts/count | AlertHandler.Count |
| GET | /alerts | AlertHandler.ListAlerts |
| GET | /alerts/{id} | AlertHandler.GetAlert |
| PUT | /alerts/{id}/status | AlertHandler.UpdateStatus |
| PUT | /alerts/{id}/false-positive | AlertHandler.MarkFalsePositive |
| PUT | /alerts/{id}/assign | AlertHandler.Assign |
| POST | /alerts/{id}/escalate | AlertHandler.Escalate |
| POST | /alerts/{id}/comment | AlertHandler.AddComment |
| POST | /alerts/{id}/merge | AlertHandler.Merge |
| GET | /alerts/{id}/comments | AlertHandler.ListComments |
| GET | /alerts/{id}/timeline | AlertHandler.ListTimeline |
| GET | /alerts/{id}/related | AlertHandler.Related |
| **Rules** | | |
| GET | /rules/stats | RuleHandler.Stats |
| GET | /rules/templates | RuleHandler.ListTemplates |
| GET | /rules | RuleHandler.ListRules |
| POST | /rules | RuleHandler.CreateRule |
| GET | /rules/{id} | RuleHandler.GetRule |
| PUT | /rules/{id} | RuleHandler.UpdateRule |
| DELETE | /rules/{id} | RuleHandler.DeleteRule |
| PUT | /rules/{id}/toggle | RuleHandler.Toggle |
| POST | /rules/{id}/test | RuleHandler.TestRule |
| POST | /rules/{id}/feedback | RuleHandler.Feedback |
| GET | /rules/{id}/performance | RuleHandler.Performance |
| **Threats** | | |
| GET | /threats/stats | ThreatHandler.Stats |
| GET | /threats/stats/trend | ThreatHandler.Trend |
| POST | /threats | ThreatHandler.CreateThreat |
| GET | /threats | ThreatHandler.ListThreats |
| GET | /threats/{id} | ThreatHandler.GetThreat |
| PUT | /threats/{id} | ThreatHandler.UpdateThreat |
| DELETE | /threats/{id} | ThreatHandler.DeleteThreat |
| PUT | /threats/{id}/status | ThreatHandler.UpdateStatus |
| GET | /threats/{id}/indicators | ThreatHandler.ListIndicatorsForThreat |
| POST | /threats/{id}/indicators | ThreatHandler.AddIndicatorToThreat |
| GET | /threats/{id}/alerts | ThreatHandler.RelatedAlerts |
| GET | /threats/{id}/timeline | ThreatHandler.Timeline |
| **Indicators** | | |
| POST | /indicators | ThreatHandler.CreateIndicator |
| GET | /indicators | ThreatHandler.ListIndicators |
| GET | /indicators/stats | ThreatHandler.IndicatorStats |
| GET | /indicators/{id} | ThreatHandler.GetIndicator |
| PUT | /indicators/{id} | ThreatHandler.UpdateIndicator |
| DELETE | /indicators/{id} | ThreatHandler.DeleteIndicator |
| PUT | /indicators/{id}/status | ThreatHandler.UpdateIndicatorStatus |
| GET | /indicators/{id}/enrichment | ThreatHandler.IndicatorEnrichment |
| GET | /indicators/{id}/matches | ThreatHandler.IndicatorMatches |
| POST | /indicators/check | ThreatHandler.CheckIndicators |
| POST | /indicators/bulk | ThreatHandler.BulkImportIndicators |
| POST | /indicators/batch | ThreatHandler.BatchCreateIndicators |
| **MITRE** | | |
| GET | /mitre/tactics | MITREHandler.ListTactics |
| GET | /mitre/techniques | MITREHandler.ListTechniques |
| GET | /mitre/coverage | MITREHandler.GetCoverage |
| **Dashboard** | | |
| GET | /dashboard | DashboardHandler.GetDashboard |
| GET | /dashboard/kpis | DashboardHandler.GetKPIs |
| GET | /dashboard/metrics | DashboardHandler.GetMetrics |
| GET | /dashboard/timeline | DashboardHandler.GetTimeline |
| GET | /dashboard/severity | DashboardHandler.GetSeverity |
| GET | /dashboard/mttr | DashboardHandler.GetMTTR |
| GET | /dashboard/workload | DashboardHandler.GetWorkload |
| **Risk** | | |
| GET | /risk/score | RiskHandler.GetScore |
| GET | /risk/heatmap | RiskHandler.GetHeatmap |
| GET | /risk/top-risks | RiskHandler.GetTopRisks |
| GET | /risk/recommendations | RiskHandler.GetRecommendations |
| **DSPM** | | |
| GET | /dspm/data-assets | DSPMHandler.ListDataAssets |
| POST | /dspm/scans | DSPMHandler.StartScan |
| GET | /dspm/classification | DSPMHandler.GetClassification |
| GET | /dspm/exposure | DSPMHandler.GetExposure |
| **CTEM** | | |
| POST | /ctem/assessments | CTEMHandler.CreateAssessment |
| GET | /ctem/assessments | CTEMHandler.ListAssessments |
| GET | /ctem/assessments/{id} | CTEMHandler.GetAssessment |
| POST | /ctem/assessments/{id}/start | CTEMHandler.StartAssessment |
| POST | /ctem/assessments/{id}/cancel | CTEMHandler.CancelAssessment |
| GET | /ctem/assessments/{id}/findings | CTEMHandler.ListFindings |
| GET | /ctem/assessments/{id}/remediation-groups | CTEMHandler.ListRemediationGroups |
| **Events** | | |
| GET | /events/stats | EventHandler.Stats |
| GET | /events | EventHandler.ListEvents |
| GET | /events/{id} | EventHandler.GetEvent |
| **Analytics** | | |
| GET | /analytics/threat-forecast | AnalyticsHandler.ThreatForecast |
| GET | /analytics/alert-forecast | AnalyticsHandler.AlertForecast |
| GET | /analytics/campaigns | AnalyticsHandler.Campaigns |

### 2.7 Event System

**Kafka Topics (from `internal/events/topics.go`):**
```
cyber.asset.events          # Asset CRUD, classification, enrichment
cyber.vulnerability.events  # Vulnerability detection, status changes
cyber.threat.events         # Threat detection, status changes
cyber.alert.events          # Alert creation, status changes, resolution
cyber.rule.events           # Rule CRUD, toggle, performance
cyber.ctem.events           # CTEM assessment lifecycle
cyber.risk.events           # Risk score changes
cyber.remediation.events    # Remediation lifecycle
cyber.dspm.events           # DSPM scan results, classification changes
cyber.vciso.events          # vCISO briefing, recommendations
cyber.ueba.events           # UEBA anomaly detection
```

**Event format:** CloudEvents v1.0 with Clario extensions (tenantid, userid, correlationid)

**Producer config:** Idempotent, WaitForAll acks, Snappy compression, partitioned by TenantID

**Consumer subscriptions in cyber-service:**
- `cyber.asset.events` → CyberConsumer (asset lifecycle)
- `iam.events` → IAMEventConsumer (user/role changes)
- `data.source.events` → DataEventConsumer (data source changes)
- `data.dark_data.events` → DataEventConsumer (shadow data)
- `data.pipeline.events` → ContinuousDSPM (pipeline monitoring)
- `file.events` → FileEventConsumer (file operations)
- `ai.events` → CacheInvalidationConsumer (AI model changes)

---

## 3. Database Schema

### 3.1 Current Tables (64 tables across 26 migrations)

See detailed DDL extraction above in Executive Summary. Key tables organized by domain:

**Core Cyber (6 tables):** assets, asset_relationships, vulnerabilities, threats, threat_indicators, detection_rules
**Alert System (3):** alerts, alert_comments, alert_timeline
**Scans (2):** scan_history, cve_database
**Remediation (2):** remediation_actions, remediation_audit_trail
**CTEM (4):** ctem_assessments, ctem_findings, ctem_remediation_groups, exposure_score_snapshots
**Risk (1):** risk_score_history
**DSPM (15):** Full data security posture stack
**UEBA (3):** Behavior analytics with partitioned events
**vCISO (25):** Complete governance framework
**Threat Feeds (2):** Feed configs + sync history

### 3.2 RLS Policies

All tenant-scoped tables have Row-Level Security:
```sql
ALTER TABLE <table> ENABLE ROW LEVEL SECURITY;
ALTER TABLE <table> FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON <table>
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
```

Special policy for detection_rules:
```sql
CREATE POLICY template_select ON detection_rules
    FOR SELECT USING (is_template = true);
```

### 3.3 Seed Data

**Source:** `backend/cmd/seeder/main.go`

Generates for a configurable tenant:
- **500 assets** by type: Server(200), Endpoint(150), NetworkDevice(30), CloudResource(40), IoTDevice(20), Application(25), Database(20), Container(15)
- **200 vulnerabilities** (60% open, 20% in_progress, 15% mitigated, 5% resolved) with real CVE IDs (CVE-2024-3094, CVE-2021-44228, etc.)
- **50 asset relationships** (depends_on, runs_on, connects_to, load_balances, managed_by, backs_up)
- **8 vCISO benchmarks** (NIST CSF dimensions)
- **5 control dependencies** (NIST AC-1 through RA-1)

---

## 4. Frontend Architecture

### 4.1 Page Map

19+ major pages under `/cyber/`:

| Route | Component | Primary API Endpoint |
|-------|-----------|---------------------|
| /cyber | SocDashboardPage | /api/v1/cyber/dashboard |
| /cyber/alerts | CyberAlertsPage | /api/v1/cyber/alerts |
| /cyber/alerts/[id] | AlertDetailPage | /api/v1/cyber/alerts/{id} |
| /cyber/assets | AssetsPage | /api/v1/cyber/assets |
| /cyber/threats | CyberThreatsPage | /api/v1/cyber/threats |
| /cyber/threats/[id] | ThreatDetailPage | /api/v1/cyber/threats/{id} |
| /cyber/indicators | IndicatorsPage | /api/v1/cyber/indicators |
| /cyber/rules | RulesPage | /api/v1/cyber/rules |
| /cyber/mitre | MitreCoveragePage | /api/v1/cyber/mitre/coverage |
| /cyber/ctem | CyberCtemPage | /api/v1/cyber/ctem/assessments |
| /cyber/risk-heatmap | RiskHeatmapPage | /api/v1/cyber/risk/heatmap |
| /cyber/dspm | CyberDspmPage | /api/v1/cyber/dspm/dashboard |
| /cyber/ueba | UebaDashboardPage | /api/v1/cyber/ueba/dashboard |
| /cyber/vciso | CyberVcisoPage | /api/v1/cyber/vciso/briefing |
| /cyber/remediation | RemediationPage | /api/v1/cyber/remediation |
| /cyber/threat-feeds | ThreatFeedsPage | /api/v1/cyber/threat-feeds |
| /cyber/analytics | AnalyticsPage | /api/v1/cyber/analytics/* |
| /cyber/events | EventsPage | /api/v1/cyber/events |

### 4.2 Component Library

**Shared cyber components** (`src/components/cyber/`):
- `export-menu.tsx` — CSV/JSON/PDF export with progress tracking
- `mitre-mini-heatmap.tsx` — Compact MITRE ATT&CK heatmap widget
- `root-cause-analysis-panel.tsx` — RCA visualization with causal chain
- `export-progress-dialog.tsx` — Long-running export progress

### 4.3 Type Definitions

**File:** `src/types/cyber.ts` — comprehensive TypeScript types for all cyber domain models including:
- CyberAsset, CyberAlert, AlertExplanation, Threat, ThreatIndicator, ThreatFeedConfig
- DetectionRule (Sigma/Threshold/Correlation/Anomaly content types)
- CTEMAssessment, CTEMFinding, RemediationAction
- DSPMDashboard, DataAsset, MITRECoverage
- VCISOBriefing, VCISOChatResponse
- SOCDashboard, KPICards, RiskHeatmapData
- RootCauseAnalysis

### 4.4 State Management

**Utility libraries** (`src/lib/`):
- `cyber-alerts.ts` — Alert status options, transitions, config
- `cyber-threats.ts` — Threat type/status options, transitions, severity colors
- `cyber-indicators.ts` — Indicator type detection/validation, CSV/STIX parsing, export
- `cyber-rules.ts` — Rule type options, content normalization, Sigma parsing, MITRE helpers

**WebSocket topics** subscribed:
- `cyber.alert.created`, `cyber.alert.status_changed`, `cyber.alert.assigned`, `cyber.alert.escalated`, `cyber.alert.merged`
- `cyber.threat.detected`, `cyber.threat.updated`
- `asset.created`, `asset.updated`, `asset.deleted`
- `vulnerability.created`

---

## 5. Integration Points

### 5.1 API Gateway Routing

From `backend/internal/gateway/config/routes.go`:
```go
{Prefix: "/api/v1/cyber", Service: "cyber-service", Public: false, EndpointGroup: EndpointGroupWrite}
{Prefix: "/api/v1/rca", Service: "cyber-service", Public: false, EndpointGroup: EndpointGroupWrite}
{Prefix: "/ws/v1/cyber", Service: "cyber-service", Public: false, EndpointGroup: EndpointGroupWS}
```
Service URL: `GW_SVC_URL_CYBER` env var, default `http://localhost:8085`

### 5.2 Middleware Stack (Gateway)

Ordered chain:
```
Recovery → RequestID → SecurityHeaders → CORS → BodyLimit → Logging → Tracing → Timeout →
(Per-route: Auth → RateLimit → ProxyHeaders)
```

### 5.3 Shared Patterns

**Pagination** (`internal/types/pagination.go`):
```go
type PaginationRequest struct {
    Page    int    `json:"page"`     // default 1
    PerPage int    `json:"per_page"` // default 20, max 100
    SortBy  string `json:"sort_by,omitempty"`
    SortDir string `json:"sort_dir,omitempty"` // "asc" | "desc"
}

type PaginatedResult[T any] struct {
    Data []T            `json:"data"`
    Meta PaginationMeta `json:"meta"`
}
```

**Tenant scoping** (`internal/types/common.go`):
```go
type TenantScoped struct {
    TenantID ID `json:"tenant_id" db:"tenant_id"`
}
```

**Soft deletes**: All domain entities use `deleted_at *time.Time` with `WHERE deleted_at IS NULL` queries.

### 5.4 Database Access

```go
// internal/database/postgres.go
func NewPostgresPool(ctx context.Context, cfg config.DatabaseConfig, logger zerolog.Logger) (*pgxpool.Pool, error)

// Cyber config: CYBER_DB_URL, CYBER_DB_MIN_CONNS (default 5), CYBER_DB_MAX_CONNS (default 20)
// Pool: 5min idle timeout, 30s health check
```

---

## 6. Configuration

### 6.1 Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `CYBER_HTTP_PORT` | HTTP listen port | 8085 |
| `CYBER_DB_URL` | PostgreSQL connection string | — (required) |
| `CYBER_DB_MIN_CONNS` | Min pool connections | 5 |
| `CYBER_DB_MAX_CONNS` | Max pool connections | 20 |
| `CYBER_REDIS_URL` | Redis connection string | — (required) |
| `CYBER_KAFKA_BROKERS` | Kafka broker addresses | — (required) |
| `CYBER_KAFKA_GROUP_ID` | Consumer group ID | — (required) |
| `CYBER_JWT_PUBLIC_KEY_PATH` | JWT public key PEM path | — (required) |
| `CYBER_SCAN_NETWORK_WORKERS` | Network scan concurrency | 100 |
| `CYBER_SCAN_NETWORK_TIMEOUT_SEC` | Scan probe timeout | 2 |
| `CYBER_SCAN_NETWORK_MAX_IPS` | Max IPs per scan | 65536 |
| `CYBER_ENRICHMENT_CVE_ENABLED` | CVE enrichment toggle | true |
| `CYBER_ENRICHMENT_GEO_ENABLED` | GeoIP enrichment toggle | false |
| `CYBER_CLASSIFY_ON_CREATE` | Auto-classify new assets | true |
| `CYBER_DETECTION_RULE_REFRESH_SEC` | Rule cache refresh interval | 60 |

### 6.2 Infrastructure Config

**PM2** (`ecosystem.local.js`):
```javascript
serviceApp("cyber-service", {
    CYBER_HTTP_PORT: "8085",
    CYBER_DB_URL: "postgres://clario:clario_dev_pass@localhost:5432/cyber_db?sslmode=disable",
    CYBER_DB_MIN_CONNS: "1",
    CYBER_DB_MAX_CONNS: "4",
    CYBER_REDIS_URL: "redis://127.0.0.1:6379/1",
    CYBER_KAFKA_BROKERS: "localhost:9094",
    CYBER_KAFKA_GROUP_ID: "cyber-service",
    CYBER_JWT_PUBLIC_KEY_PATH: path.join(secretsDir, "jwt-public.pem"),
})
```

**Helm:** `deploy/helm/clario360/templates/cyber-service/` with deployment, service, configmap, PDB, HPA. K8s port 8090 (different from local 8085).

**Cross-service references:**
- Notification: `NOTIF_CYBER_SERVICE_URL: "http://localhost:8085"`
- Workflow: `WF_SERVICE_URLS: "cyber=http://localhost:8085"`
- Visus: `VISUS_SUITE_CYBER_URL: "http://localhost:8085"`
- Gateway: `GW_SVC_URL_CYBER: "http://localhost:8085"`

---

## 7. Testing Infrastructure

### 7.1 Existing Tests

106 test files covering:
- Handler tests (mock services, httptest recorder)
- Service tests (mock repositories)
- DTO validation tests
- Detection engine tests (Sigma, threshold, correlation, anomaly)
- DSPM, UEBA, vCISO subsystem tests
- Remediation lifecycle tests
- CTEM assessment tests
- Risk scoring tests
- Consumer event handling tests

### 7.2 Test Patterns

**Unit test** (mock-based):
```go
type mockDashboardService struct {
    getSOCDashboardFn func(ctx context.Context, tenantID uuid.UUID) (*model.SOCDashboard, error)
}

func dashAuthCtx(tenantID, userID uuid.UUID) context.Context {
    ctx := context.Background()
    ctx = auth.WithUser(ctx, &auth.ContextUser{
        ID: userID.String(), TenantID: tenantID.String(),
        Email: "analyst@example.com", Roles: []string{"security_analyst"},
    })
    return auth.WithTenantID(ctx, tenantID.String())
}
```

**Integration test** (`backend/integration_tests/`):
```go
//go:build integration
// Guarded by TEST_RUN_INTEGRATION=1
// Uses real HTTP client against running services
// Creates test tenants with JWT tokens
```

---

## 8. CTI Implementation Recommendations

### Recommended New Tables

**cti_campaigns** — Campaign tracking entity
```
id, tenant_id, name, description, status (active/dormant/concluded),
threat_actor_id, objective, first_seen_at, last_seen_at,
target_sectors TEXT[], target_countries TEXT[], target_organizations TEXT[],
mitre_tactic_ids TEXT[], mitre_technique_ids TEXT[],
related_threat_ids UUID[], indicator_count INT, alert_count INT,
confidence DECIMAL, source TEXT, tags TEXT[], metadata JSONB,
created_by, created_at, updated_at, deleted_at
```

**cti_threat_actors** — Threat actor profiles
```
id, tenant_id, name, aliases TEXT[], description,
type (apt/cybercrime/hacktivist/state_sponsored/insider/unknown),
sophistication (novice/intermediate/advanced/expert/innovator),
origin_country TEXT, target_sectors TEXT[], target_countries TEXT[],
mitre_technique_ids TEXT[], active_campaigns INT,
first_seen_at, last_seen_at, motivation TEXT[], resource_level TEXT,
tools_used TEXT[], known_vulnerabilities TEXT[],
confidence DECIMAL, source TEXT, tags TEXT[], metadata JSONB,
created_by, created_at, updated_at, deleted_at
```

**cti_brand_abuse** — Brand abuse/impersonation tracking
```
id, tenant_id, type (typosquat/phishing_site/fake_social/dark_web_mention/credential_leak/logo_abuse/app_clone),
detected_domain TEXT, similarity_score DECIMAL,
original_domain TEXT, screenshot_url TEXT,
hosting_country TEXT, hosting_provider TEXT, hosting_ip TEXT,
status (detected/confirmed/takedown_requested/taken_down/monitoring/false_positive),
risk_level TEXT, ssl_issued_at TIMESTAMPTZ, ssl_issuer TEXT,
detection_source TEXT, takedown_requested_at TIMESTAMPTZ, taken_down_at TIMESTAMPTZ,
first_seen_at, last_seen_at,
related_indicator_ids UUID[], alert_id UUID,
tags TEXT[], metadata JSONB,
created_by, created_at, updated_at, deleted_at
```

**cti_geographic_threats** — Geographic threat aggregation
```
id, tenant_id, country_code TEXT, country_name TEXT, region TEXT,
threat_count INT, campaign_count INT, actor_count INT, indicator_count INT,
top_threat_types JSONB, top_mitre_techniques JSONB,
severity_distribution JSONB, trend_direction TEXT,
snapshot_date DATE,
created_at, updated_at
```

**cti_sector_targeting** — Sector/industry targeting analysis
```
id, tenant_id, sector TEXT, subsector TEXT,
threat_count INT, campaign_count INT, actor_count INT,
top_threat_types JSONB, top_mitre_techniques JSONB,
risk_level TEXT, trend_direction TEXT,
relevance_score DECIMAL (how relevant to this tenant's sector),
snapshot_date DATE,
created_at, updated_at
```

### Recommended New API Endpoints

Under `/api/v1/cyber/cti/`:
```
# Campaigns
POST   /cti/campaigns
GET    /cti/campaigns
GET    /cti/campaigns/{id}
PUT    /cti/campaigns/{id}
DELETE /cti/campaigns/{id}
PUT    /cti/campaigns/{id}/status
GET    /cti/campaigns/{id}/indicators
GET    /cti/campaigns/{id}/timeline

# Threat Actors
POST   /cti/actors
GET    /cti/actors
GET    /cti/actors/{id}
PUT    /cti/actors/{id}
GET    /cti/actors/{id}/campaigns
GET    /cti/actors/{id}/techniques

# Brand Abuse
POST   /cti/brand-abuse
GET    /cti/brand-abuse
GET    /cti/brand-abuse/{id}
PUT    /cti/brand-abuse/{id}/status
POST   /cti/brand-abuse/{id}/takedown
GET    /cti/brand-abuse/stats

# Geographic
GET    /cti/geo/threat-map
GET    /cti/geo/countries
GET    /cti/geo/countries/{code}
GET    /cti/geo/hotspots

# Sector Analysis
GET    /cti/sectors
GET    /cti/sectors/{sector}
GET    /cti/sectors/targeting-trends

# CTI Dashboard
GET    /cti/dashboard
GET    /cti/dashboard/global-map-data
GET    /cti/dashboard/active-campaigns
GET    /cti/dashboard/top-actors
GET    /cti/dashboard/brand-risk
```

### Recommended Kafka Topics

```
cyber.cti.campaign.events     # Campaign lifecycle
cyber.cti.actor.events        # Actor profile changes
cyber.cti.brand_abuse.events  # Brand abuse detections
cyber.cti.geo.events          # Geographic threat updates
```

### Recommended Frontend Routes

```
/cyber/cti                    # CTI Dashboard (Global Threat Map, Active Campaigns summary)
/cyber/cti/campaigns          # Campaign list
/cyber/cti/campaigns/[id]     # Campaign detail
/cyber/cti/actors             # Threat actor profiles
/cyber/cti/actors/[id]        # Actor detail with campaign history
/cyber/cti/brand-abuse        # Brand abuse monitoring
/cyber/cti/geo                # Geographic threat analysis
/cyber/cti/sectors            # Sector targeting analysis
```

### Recommended Seed Data Volume

| Entity | Count | Notes |
|--------|-------|-------|
| Campaigns | 15 | Mix of active/dormant/concluded |
| Threat Actors | 20 | APT groups, cybercrime orgs, hacktivists |
| Brand Abuse | 30 | Typosquats, phishing sites, dark web mentions |
| Geographic Threats | 50 | Country-level aggregation |
| Sector Targeting | 15 | Industry sectors with threat correlation |

### Risk Areas and Dependencies

1. **Existing `threats.campaign` field** — The existing Threat model has a `campaign *string` field. The new CTI campaign entity should be linked via a `campaign_id UUID` FK, and the text field deprecated or kept for backward compatibility.

2. **Existing `threats.threat_actor` field** — Same issue. Add `threat_actor_id UUID` FK to threat_actors table.

3. **Geographic data** — Adding `origin_country` and `target_countries` to existing threats requires a migration that doesn't break existing data (nullable fields).

4. **Feed integration** — Existing `threat_feed_configs` table already handles STIX/TAXII/MISP feeds. CTI campaigns and actors should be extractable from these feeds, requiring changes to the feed consumer/parser.

5. **Dashboard aggregation** — The CTI dashboard will need new repository queries joining across campaigns, actors, threats, indicators, and brand abuse. Consider materialized views or periodic snapshots for geographic/sector data to avoid expensive real-time aggregation.

6. **Frontend routing** — New routes under `/cyber/cti/` should follow the existing page pattern with `useDataTable`, `useRealtimeData`, and `ExportMenu` components.

7. **Permissions** — May need `cyber:cti:read` and `cyber:cti:write` permissions, or reuse existing `cyber:read`/`cyber:write`.
