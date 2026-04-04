# Clario360 — AI / ML Model Catalog

> **Generated**: 2026-03-13  
> **Source of truth**: [`model_seeder.go`](file:///Users/mac/clario360/backend/internal/aigovernance/seeder/model_seeder.go) + source code across the codebase

All models are registered in the **AI Governance** framework, which provides prediction logging, drift detection, shadow-mode comparison, and explainability for every model in the platform.

---

## At-a-Glance

| # | Model Name | Slug | Suite | Type | Risk Tier | Location |
|---|-----------|------|-------|------|-----------|----------|
| 1 | Sigma Rule Evaluator | `cyber-sigma-evaluator` | Cyber | Rule-Based | Critical | `detection/sigma_evaluator.go` |
| 2 | Anomaly Detector (Statistical) | `cyber-anomaly-detector` | Cyber | Anomaly Detector | High | `detection/anomaly_evaluator.go` |
| 3 | Risk Scorer (Multi-Factor) | `cyber-risk-scorer` | Cyber | Scorer | High | `cyber/service/risk_service.go` |
| 4 | UEBA Behavioral Anomaly Detector | `cyber-ueba-detector` | Cyber | Anomaly Detector | High | `cyber/ueba/engine/engine.go` |
| 5 | vCISO Intent Classifier | `cyber-vciso-classifier` | Cyber | Rule-Based | Medium | `vciso/chat/engine/intent_classifier.go` |
| 6 | vCISO LLM Engine | `cyber-vciso-llm` | Cyber | LLM (Agentic) | High | `vciso/llm/engine/llm_engine.go` |
| 7 | vCISO Predictive Threat Engine | `cyber-vciso-predictive` | Cyber | ML Classifier | High | `vciso/predict/engine/forecast_engine.go` |
| 8 | Asset Auto-Classifier | `cyber-asset-classifier` | Cyber | Rule-Based | Medium | `cyber/classifier/classifier.go` |
| 9 | CTEM Prioritization | `cyber-ctem-prioritizer` | Cyber | Scorer | High | `cyber/ctem/` |
| 10 | PII Classifier | `data-pii-classifier` | Data | Rule-Based | High | `data/darkdata/classifier.go` |
| 11 | Contradiction Detector | `data-contradiction-detector` | Data | Rule-Based | Medium | `data/contradiction/` |
| 12 | Data Quality Scorer | `data-quality-scorer` | Data | Scorer | Medium | `data/quality/` |
| 13 | Meeting Minutes Generator | `acta-minutes-generator` | Acta | NLP Extractor | Medium | `acta/ai/` |
| 14 | Action Item Extractor | `acta-action-extractor` | Acta | NLP Extractor | Low | `acta/ai/` |
| 15 | Contract Clause Extractor | `lex-clause-extractor` | Lex | Rule-Based | High | `lex/analyzer/` |
| 16 | Contract Risk Analyzer | `lex-risk-analyzer` | Lex | Scorer | High | `lex/analyzer/` |
| 17 | KPI Threshold Monitor | `visus-kpi-monitor` | Visus | Statistical | Medium | `visus/kpi/` |
| 18 | Executive Recommendation Engine | `visus-recommendation-engine` | Visus | Recommender | Low | `visus/report/` |

---

## Cyber Suite

### 1. Sigma Rule Evaluator

| Property | Value |
|----------|-------|
| **Slug** | `cyber-sigma-evaluator` |
| **Type** | Rule-Based (Deterministic) |
| **Risk Tier** | Critical |
| **Explainability** | Rule Trace |
| **Source** | [`sigma_evaluator.go`](file:///Users/mac/clario360/backend/internal/cyber/detection/sigma_evaluator.go) |

**Purpose**: Evaluates Sigma-style detection rules against security events.

**How it works**:
- Compiles named selections with field-based conditions and a boolean condition expression
- Supports `AND / OR / NOT` logic across named selections
- Sliding window grouping with timeframe + threshold parameters
- Each match returns involved events, matched selections, and group keys

**Inputs**: JSON rule content with `detection` block (named selections + condition), batch of `SecurityEvent` structs  
**Outputs**: `[]RuleMatch` — matched events, timestamp, match details (selection names, condition count, group key)

---

### 2. Anomaly Detector (Statistical Baseline)

| Property | Value |
|----------|-------|
| **Slug** | `cyber-anomaly-detector` |
| **Type** | Anomaly Detector |
| **Risk Tier** | High |
| **Explainability** | Statistical Deviation |
| **Source** | [`anomaly_evaluator.go`](file:///Users/mac/clario360/backend/internal/cyber/detection/anomaly_evaluator.go) |

**Purpose**: Detects statistical deviations from historical baselines using z-score thresholds.

**How it works**:
- Groups events by a configurable `group_by` field
- Computes per-window metric values (`event_count`, `unique_ips`, `bytes_transferred`, `dns_query_count`, `login_hour`, `connection_interval_regularity`, or custom fields)
- Compares against an adaptive baseline (EMA-smoothed mean/variance) stored in `BaselineStore`
- Direction-aware: `above`, `below`, or `both`
- Minimum baseline sample requirement before anomaly detection activates

**Inputs**: Compiled rule config (metric, group_by, window, z_score_threshold, direction), event batch  
**Outputs**: `[]RuleMatch` with z-score, deviation %, mean, std_dev, current_value, baseline_samples

---

### 3. Threshold Evaluator

| Property | Value |
|----------|-------|
| **Slug** | — (sub-evaluator of Detection Engine) |
| **Type** | Rule-Based |
| **Source** | [`threshold_evaluator.go`](file:///Users/mac/clario360/backend/internal/cyber/detection/threshold_evaluator.go) |

**Purpose**: Detects when a count, sum, or distinct metric exceeds a threshold within a sliding window.

**How it works**:
- Filters events by a compiled condition selection
- Groups by a configurable field
- Sliding window with configurable duration
- Metric types: `count`, `sum(field)`, `distinct(field)`

---

### 4. Correlation Evaluator

| Property | Value |
|----------|-------|
| **Slug** | — (sub-evaluator of Detection Engine) |
| **Type** | Rule-Based (Sequence Detector) |
| **Source** | [`correlation_evaluator.go`](file:///Users/mac/clario360/backend/internal/cyber/detection/correlation_evaluator.go) |

**Purpose**: Detects ordered multi-event attack sequences (e.g., brute-force → login success).

**How it works**:
- Defines named event types with individual conditions
- Specifies an ordered sequence of event names
- Groups by a configurable field (e.g., `source_ip`)
- Matches within a time window with optional `min_failed_count` for the first event

---

### 5. Risk Scorer (Multi-Factor Composite)

| Property | Value |
|----------|-------|
| **Slug** | `cyber-risk-scorer` |
| **Type** | Scorer |
| **Risk Tier** | High |
| **Explainability** | Feature Importance |
| **Source** | [`risk_service.go`](file:///Users/mac/clario360/backend/internal/cyber/service/risk_service.go) |

**Purpose**: Computes organizational cyber risk as a weighted composite score.

**Weights**: Vulnerability 30% · Threat 25% · Configuration 20% · Surface 15% · Compliance 10%

---

### 6. UEBA Behavioral Anomaly Detector

| Property | Value |
|----------|-------|
| **Slug** | `cyber-ueba-detector` |
| **Type** | Anomaly Detector (Behavioral) |
| **Risk Tier** | High |
| **Explainability** | Statistical Deviation |
| **Source** | [`engine.go`](file:///Users/mac/clario360/backend/internal/cyber/ueba/engine/engine.go) |

**Purpose**: End-to-end behavioral analytics engine that profiles user/entity behavior and detects anomalies across 7 signal dimensions.

**Architecture** — four-stage pipeline:

| Stage | Component | Description |
|-------|-----------|-------------|
| **1. Profiler** | [`BehavioralProfiler`](file:///Users/mac/clario360/backend/internal/cyber/ueba/profiler/profiler.go) | Builds EMA-smoothed baselines using Welford algorithm: hourly/daily access distributions, data volume, table/IP access patterns, query types, session stats, failure rates |
| **2. Detector** | [`AnomalyDetector`](file:///Users/mac/clario360/backend/internal/cyber/ueba/detector/detector.go) | Runs 7 signal detectors in parallel per event |
| **3. Correlator** | [`AnomalyCorrelator`](file:///Users/mac/clario360/backend/internal/cyber/ueba/correlator/correlator.go) | Correlates weak signals into high-confidence alert types using 5 rule-based patterns |
| **4. Risk Scorer** | [`EntityRiskScorer`](file:///Users/mac/clario360/backend/internal/cyber/ueba/scorer/risk_scorer.go) | Scores entities 0–100 with severity × recency × confidence weighting + daily exponential decay |

**7 Signal Detectors**:
1. **Unusual Time** — Activity in hours with < 2% (mature) or < 1% (baseline) historical probability
2. **Unusual Volume** — Data volume z-score > 3/4/5 for medium/high/critical
3. **New Table Access** — First-time access to a table by the entity
4. **New Source IP** — Login from a previously unseen IP (with geo resolution)
5. **Failed Access Spike** — Failure count z-score > 3/5/8
6. **Bulk Data Access** — Row count > 5×/10× daily baseline multiplier
7. **Privilege Escalation** — DDL usage exceeding 5% of observed query distribution

**5 Correlation Rules**:
1. **Data Exfiltration** (MITRE TA0010) — Unusual time + volume/bulk
2. **Credential Compromise** (TA0006) — New IP + new table or unusual time
3. **Insider Threat** (TA0004/TA0010) — Privilege escalation + bulk/restricted table
4. **Lateral Movement** (TA0008) — Failed access spike + new IP/table
5. **Reconnaissance** (TA0007) — ≥3 distinct new tables accessed

**Profile Maturity Model**: Learning (< 30 observations) → Baseline (30–90 days) → Mature (90+ days). Detectors suppress alerts during Learning phase.

---

### 7. vCISO Intent Classifier

| Property | Value |
|----------|-------|
| **Slug** | `cyber-vciso-classifier` |
| **Type** | Rule-Based (NLP) |
| **Risk Tier** | Medium |
| **Explainability** | Rule Trace |
| **Source** | [`intent_classifier.go`](file:///Users/mac/clario360/backend/internal/cyber/vciso/chat/engine/intent_classifier.go) |

**Purpose**: Classifies natural language security queries into intents and routes to deterministic execution tools.

**Pipeline**:
1. **Normalize** — Unicode NFC, lowercase, strip special chars, collapse whitespace
2. **Match** — Run pluggable matchers (regex → keyword) collecting candidates
3. **Rank** — Sort candidates by descending confidence
4. **Ambiguity** — Flag if top-2 candidates are within 10% gap
5. **Extract** — Run entity extractor (alert IDs, asset names, time ranges, severities)

**Matchers**:
- **Regex Matcher** — Compiled patterns → 0.90 confidence on match
- **Keyword Matcher** — Overlap score with min 30% overlap → confidence = 0.50 + overlap × 0.30

**Intent count**: 19 registered intents  
**Entity types extracted**: `alert_id`, `asset_name`, `asset_ip`, `time_range`, `severity`, `count`, `framework`, `description`

---

### 8. vCISO LLM Engine

| Property | Value |
|----------|-------|
| **Slug** | `cyber-vciso-llm` |
| **Type** | LLM (Agentic) |
| **Risk Tier** | High |
| **Explainability** | Reasoning Trace |
| **Source** | [`llm_engine.go`](file:///Users/mac/clario360/backend/internal/cyber/vciso/llm/engine/llm_engine.go) |

**Purpose**: LLM-powered conversational AI for complex security queries. Orchestrates governed security tools via function calling with comprehensive safety guardrails.

**13-Phase Pipeline**:

| Phase | Component | Description |
|-------|-----------|-------------|
| 1 | Rate Limiter | Token + cost-based per-tenant rate limiting |
| 2 | Conversation | Load/create conversation with context expiry |
| 3 | Injection Guard | Prompt injection detection and sanitization |
| 4 | Provider | Multi-provider resolution (OpenAI, Anthropic, Azure, Local) |
| 5 | Context Compiler | Compiles conversation history into LLM messages |
| 6 | Prompt Builder | Builds system prompt with tenant/user context |
| 7 | Tool Loop | Iterative LLM calls + tool execution (max configurable iterations) |
| 8 | Hallucination Guard | Claim extraction → evidence matching → grounding verification |
| 9 | PII Filter | Detects and redacts PII from responses |
| 10 | Synthesis | Structures final response with metadata |
| 11 | Prediction Log | AI Governance prediction logging |
| 12 | Rate Limit Consume | Deducts consumed tokens/cost |
| 13 | Persist + Audit | Persists messages, updates conversation state, writes audit log |

**Hallucination Guard** ([`hallucination_guard.go`](file:///Users/mac/clario360/backend/internal/cyber/vciso/llm/engine/hallucination_guard.go)):
- Extracts claims from LLM response (numeric, status, security, recommendation, general)
- Builds evidence set from tool call results
- Scores claims against evidence using token overlap (45%), number matching (20%), status matching (15%), identifier matching (20%)
- Safe recommendations (no numbers, no guarantees) are auto-grounded
- Critical ungrounded claims → response blocked; non-critical → softened

**Providers**: OpenAI, Anthropic, Azure OpenAI, Local models  
**Tool count**: 25+ governed security tools

---

### 9. vCISO Predictive Threat Engine

| Property | Value |
|----------|-------|
| **Slug** | `cyber-vciso-predictive` |
| **Type** | ML Classifier (Ensemble) |
| **Risk Tier** | High |
| **Explainability** | SHAP Feature Importance |
| **Source** | [`forecast_engine.go`](file:///Users/mac/clario360/backend/internal/cyber/vciso/predict/engine/forecast_engine.go) |

**Purpose**: Six explainable predictive security models with automated retraining, backtesting, drift detection, and confidence calibration.

| Sub-Model | Prediction Type | Framework | Description |
|-----------|----------------|-----------|-------------|
| **Alert Volume Forecaster** | `alert_volume_forecast` | Prophet-like | Time-series forecast with weekday factors, seasonality, and EMA trend |
| **Asset Risk Predictor** | `asset_risk` | XGBoost-like | Weighted logistic scoring of asset targeting probability |
| **Vulnerability Exploit Predictor** | `vulnerability_exploit` | GBM | Predicts exploitation probability from CVSS, EPSS, exposure, patch age |
| **Technique Trend Analyzer** | `attack_technique_trend` | Regression | Forecasts MITRE technique growth from internal + industry signals |
| **Insider Threat Trajectory** | `insider_threat_trajectory` | LSTM-like | Projects entity risk trajectories from behavioral time series |
| **Campaign Detector** | `campaign_detection` | DBSCAN-like | Clusters related alerts by shared IOCs, techniques, and timeline density |

**Common Infrastructure**:
- **Feature Store** — Centralized feature engineering per model
- **Model Registry** — Version management, activation, drift scoring
- **SHAP Explainer** — Top-N feature contributions for every prediction
- **Confidence Calibrator** — P10/P50/P90 interval generation from residuals
- **Prediction Narrator** — Natural language explanations + verification steps
- **Backtester** — Regression (MAE/RMSE/R²) and classification (accuracy/precision/recall/F1) metrics
- **Drift Detector** — PSI-based accuracy drift detection with alerting
- **Auto-Retrain** — Daily scheduled maintenance with drift-triggered retraining

---

### 10. Asset Auto-Classifier

| Property | Value |
|----------|-------|
| **Slug** | `cyber-asset-classifier` |
| **Type** | Rule-Based |
| **Risk Tier** | Medium |
| **Explainability** | Rule Trace |
| **Source** | [`classifier.go`](file:///Users/mac/clario360/backend/internal/cyber/classifier/classifier.go) |

**Purpose**: Determines asset criticality using priority-ordered classification rules.

**Algorithm**: Evaluates rules sorted by priority (lower = first); returns the first matching criticality level (Critical → High → Medium → Low). Custom rules override defaults by name.

---

### 11. DSPM Classifier + Risk Scorer

| Property | Value |
|----------|-------|
| **Slug** | — (part of DSPM subsystem) |
| **Type** | Rule-Based Classifier + Scorer |
| **Source** | [`dspm/classifier.go`](file:///Users/mac/clario360/backend/internal/cyber/dspm/classifier.go), [`dspm/scoring.go`](file:///Users/mac/clario360/backend/internal/cyber/dspm/scoring.go) |

**Purpose**: Classifies data assets by PII content and computes DSPM risk scores.

**Classifier** — Scans asset schema column names against 11 PII regex patterns (biometric, medical, bank, credit_card, SSN, salary, email, phone, name, address, birth_date) with weighted sensitivity scores:
- Weight ≥ 9 → **Restricted** (score 90)
- Weight ≥ 4 with PII → **Confidential** (score 70)
- Any PII → **Internal** (score 50)
- No PII → **Public** (score 10) or **Internal** (score 40)

**Risk Scorer** — `risk = sensitivity × exposureFactor × (1 − postureScore/100)` with three weighted factors: Sensitivity (45%), Exposure (30%), Control Gap (25%)

---

### 12. CTEM Prioritization

| Property | Value |
|----------|-------|
| **Slug** | `cyber-ctem-prioritizer` |
| **Type** | Scorer |
| **Risk Tier** | High |
| **Explainability** | Feature Importance |

**Purpose**: Weighted prioritization model for Continuous Threat Exposure Management findings.

**Weights**: Impact 55% · Exploitability 45%

---

## Data Suite

### 13. PII Classifier

| Property | Value |
|----------|-------|
| **Slug** | `data-pii-classifier` |
| **Type** | Rule-Based |
| **Risk Tier** | High |
| **Explainability** | Rule Trace |
| **Source** | [`darkdata/classifier.go`](file:///Users/mac/clario360/backend/internal/data/darkdata/classifier.go) |

**Purpose**: Deterministic PII detection for schema discovery. Classifies data assets into Public / Internal / Confidential / Restricted based on column-name pattern matching and inferred PII types.

---

### 14. Contradiction Detector

| Property | Value |
|----------|-------|
| **Slug** | `data-contradiction-detector` |
| **Type** | Rule-Based |
| **Risk Tier** | Medium |
| **Explainability** | Rule Trace |
| **Source** | `data/contradiction/` |

**Purpose**: Detects logical contradictions across data sources using four strategies: logical, semantic, temporal, and analytical.

---

### 15. Data Quality Scorer

| Property | Value |
|----------|-------|
| **Slug** | `data-quality-scorer` |
| **Type** | Scorer |
| **Risk Tier** | Medium |
| **Explainability** | Feature Importance |

**Purpose**: Computes enterprise data quality scores from passed/failed validation rules weighted by severity (Critical:4, High:3, Medium:2, Low:1).

---

## Acta Suite

### 16. Meeting Minutes Generator

| Property | Value |
|----------|-------|
| **Slug** | `acta-minutes-generator` |
| **Type** | NLP Extractor |
| **Risk Tier** | Medium |
| **Explainability** | Template-Based |
| **Source** | `acta/ai/` |

**Purpose**: Template-driven meeting minutes generation from agenda items, attendance records, and discussion notes using deterministic summarization.

---

### 17. Action Item Extractor

| Property | Value |
|----------|-------|
| **Slug** | `acta-action-extractor` |
| **Type** | NLP Extractor |
| **Risk Tier** | Low |
| **Explainability** | Rule Trace |

**Purpose**: Pattern-based extraction of action items from meeting notes using trigger patterns (`ACTION:`, `will`, `agreed that`).

---

## Lex Suite

### 18. Contract Clause Extractor

| Property | Value |
|----------|-------|
| **Slug** | `lex-clause-extractor` |
| **Type** | Rule-Based |
| **Risk Tier** | High |
| **Explainability** | Rule Trace |

**Purpose**: Pattern-driven clause extraction from legal documents. Identifies 19 clause types from document sections using deterministic pattern matching.

---

### 19. Contract Risk Analyzer

| Property | Value |
|----------|-------|
| **Slug** | `lex-risk-analyzer` |
| **Type** | Scorer |
| **Risk Tier** | High |
| **Explainability** | Feature Importance |

**Purpose**: Transparent weighted risk analysis for contracts, scoring across 5 factors: clause risk, missing clauses, commercial value, contract expiry, and compliance flags.

---

## Visus Suite

### 20. KPI Threshold Monitor

| Property | Value |
|----------|-------|
| **Slug** | `visus-kpi-monitor` |
| **Type** | Statistical |
| **Risk Tier** | Medium |
| **Explainability** | Statistical Deviation |

**Purpose**: Evaluates KPI values against directional thresholds to determine status transitions (on-track, at-risk, critical).

---

### 21. Executive Recommendation Engine

| Property | Value |
|----------|-------|
| **Slug** | `visus-recommendation-engine` |
| **Type** | Recommender |
| **Risk Tier** | Low |
| **Explainability** | Rule Trace |

**Purpose**: Rule-based recommendation engine for executive reporting. Triggers on critical KPIs, overdue actions, expiring contracts, and coverage gaps.

---

## Cross-Cutting: Detection Engine

The [`DetectionEngine`](file:///Users/mac/clario360/backend/internal/cyber/detection/engine.go) orchestrates four evaluator types plus the Indicator Matcher:

```
SecurityEvents → [Sigma | Threshold | Correlation | Anomaly] evaluators → Alert generation
                  ↓
              Indicator Matcher (IOC matching against threat intelligence)
```

All evaluations are logged through the AI Governance prediction middleware for auditability.

---

## Cross-Cutting: RCA Engine

| Property | Value |
|----------|-------|
| **Source** | [`rca/engine.go`](file:///Users/mac/clario360/backend/internal/rca/engine.go) |

The **Root Cause Analysis Engine** supports three analysis types:

| Analyzer | Type | Description |
|----------|------|-------------|
| `SecurityAlertAnalyzer` | Security | Traces cyber alerts back through causal chains |
| `PipelineFailureAnalyzer` | Pipeline | Analyzes data pipeline failures with dependency graphs |
| `QualityIssueAnalyzer` | Quality | Investigates data quality issues across sources |

Each analyzer uses: **Timeline Builder** → **Chain Builder** → **Impact Assessor** → **Recommender**

---

## Cross-Cutting: AI Governance Framework

| Component | Location | Purpose |
|-----------|----------|---------|
| **Model Registry** | `aigovernance/service/registry_service.go` | CRUD for registered models and versions |
| **Prediction Logger** | `aigovernance/middleware/prediction_logger.go` | Wraps every model call with input/output/confidence logging |
| **Drift Detector** | `aigovernance/drift/` | PSI calculator + performance monitor for production drift |
| **Shadow Mode** | `aigovernance/shadow/` | Side-by-side model comparison without production impact |
| **Explainability** | `aigovernance/explainer/` | Statistical, rule-trace, feature-importance, template, and NL explainers |
| **Validation** | `aigovernance/service/validation_service.go` | Pre-deployment model validation |
| **Lifecycle** | `aigovernance/service/lifecycle_service.go` | Model version promotion, deprecation, and rollback |
| **Dashboard** | `aigovernance/service/dashboard_service.go` | Model inventory, health metrics, and prediction analytics |
