# Clario360 AI Capabilities Matrix

_Last reviewed: 8 March 2026_

## Summary

This matrix lists the AI-related capabilities that are actually implemented in the current codebase, how they are implemented, whether they are governed through the AI governance framework, and whether there is a user-facing surface for them.

## Legend

- **Implementation Type**
  - **Rule-based**: deterministic logic using patterns, templates, thresholds, scoring rules, or heuristics.
  - **Statistical**: baseline, threshold, z-score, PSI, or other numerical evaluation.
  - **Governed wrapper**: prediction is logged/explained/versioned through the AI governance layer.
- **Governed**
  - **Yes**: routed through `aigovernance` prediction logging / explanation / lifecycle hooks.
  - **Partial**: related outputs are governed, but the full capability is not entirely routed through the framework.
  - **No**: capability exists without governance hooks.
- **User-facing**
  - **API**: exposed through backend routes or SDK.
  - **UI**: surfaced in frontend pages.
  - **Both**: both API and frontend usage exist.

## Capability Matrix

| Suite / Area | Capability | Implementation Type | Governed | User-facing | Key Evidence |
|---|---|---:|---:|---:|---|
| Acta | Minutes generation | Rule-based / template-driven | Yes | API | `backend/internal/acta/ai/minutes_generator.go`, `backend/internal/acta/service/minutes_service.go` |
| Acta | Executive summary generation | Rule-based / template-driven | Yes | API | `backend/internal/acta/ai/summary_builder.go`, `backend/internal/acta/ai/templates/summary_template.go` |
| Acta | Meeting action item extraction | Rule-based / regex heuristics | Yes | API | `backend/internal/acta/ai/action_extractor.go`, `backend/internal/acta/service/minutes_service.go` |
| Lex | Clause extraction | Rule-based / pattern matching | Partial | API | `backend/internal/lex/analyzer/clause_extractor.go`, `backend/internal/lex/app.go` |
| Lex | Contract risk scoring | Rule-based scoring | Partial | API | `backend/internal/lex/analyzer/risk_analyzer.go` |
| Lex | Missing clause detection | Rule-based policy rules | No | API | `backend/internal/lex/analyzer/missing_clause_detector.go` |
| Lex | Entity extraction (parties, dates, amounts) | Rule-based extraction | No | API | `backend/internal/lex/analyzer/entity_extractor.go` |
| Lex | Compliance flagging | Rule-based checks | No | API | `backend/internal/lex/analyzer/risk_analyzer.go`, `backend/internal/lex/analyzer/compliance_checker.go` |
| Lex | Recommendation generation | Rule-based recommendations | No | API | `backend/internal/lex/analyzer/recommendation_engine.go` |
| Cyber | Sigma rule evaluation logging | Rule-based detection + governed wrapper | Yes | API | `backend/internal/cyber/detection/ai_predictions.go` |
| Cyber | Anomaly detection logging | Statistical + governed wrapper | Yes | API | `backend/internal/cyber/detection/ai_predictions.go`, `backend/internal/cyber/detection/anomaly_evaluator.go` |
| Cyber | Organization risk scoring | Rule-based weighted scoring + governed wrapper | Yes | API | `backend/internal/cyber/risk/scorer.go`, `backend/internal/cyber/risk/ai_predictions.go` |
| Cyber | Asset criticality classification | Rule-based classification + governed wrapper | Yes | API | `backend/internal/cyber/service/asset_service_ai.go`, `backend/internal/cyber/classifier/classifier.go` |
| Cyber | Executive vCISO briefing generation | Rule-based synthesis / reporting | Partial | API | `backend/internal/cyber/vciso/briefing.go`, `backend/internal/cyber/vciso/report.go` |
| Cyber | CTEM prioritization / exposure scoring | Rule-based scoring | Partial | API | `backend/internal/cyber/ctem/prioritization.go`, `backend/internal/cyber/ctem/scoring.go` |
| Cyber | DSPM classification / posture scoring | Rule-based scoring | No | API | `backend/internal/cyber/dspm/classifier.go`, `backend/internal/cyber/dspm/scoring.go`, `backend/internal/cyber/dspm/posture.go` |
| AI Governance | Prediction logging | Governed wrapper | Yes | API / SDK / UI | `backend/internal/aigovernance/middleware/prediction_logger.go`, `deploy/jupyter/sdk/clario360_sdk/clario360/ai/predictions.py`, `frontend/src/app/(dashboard)/admin/ai-governance/page.tsx` |
| AI Governance | Structured explanations | Rule-based / explainability services | Yes | API / UI | `backend/internal/aigovernance/service/explanation_service.go`, `backend/internal/aigovernance/explainer/` |
| AI Governance | Human-readable explanation text | Rule-based templating | Yes | API / UI | `backend/internal/aigovernance/explainer/natural_language.go` |
| AI Governance | Model registry | Governed platform capability | Yes | API / SDK / UI | `backend/internal/aigovernance/service/registry_service.go`, `deploy/jupyter/sdk/clario360_sdk/clario360/ai/models.py`, `frontend/src/app/(dashboard)/admin/ai-governance/page.tsx` |
| AI Governance | Model lifecycle promotion / rollback | Governed platform capability | Yes | API / SDK / UI | `backend/internal/aigovernance/service/lifecycle_service.go`, `deploy/jupyter/sdk/clario360_sdk/clario360/ai/lifecycle.py` |
| AI Governance | Shadow testing | Governed platform capability | Yes | API / SDK / UI | `backend/internal/aigovernance/shadow/executor.go`, `backend/internal/aigovernance/service/shadow_service.go`, `frontend/src/app/(dashboard)/admin/ai-governance/page.tsx` |
| AI Governance | Drift detection / alerting | Statistical monitoring | Yes | API / UI | `backend/internal/aigovernance/drift/psi_calculator.go`, `backend/internal/aigovernance/drift/performance_monitor.go`, `backend/internal/aigovernance/service/dashboard_service.go` |
| AI Governance | Dashboard / KPI monitoring | Governed reporting capability | Yes | Both | `backend/internal/aigovernance/service/dashboard_service.go`, `frontend/src/app/(dashboard)/admin/ai-governance/page.tsx` |

## Key Observations

1. The solution contains **real AI-adjacent functionality**, but most of it is **transparent rule-based or statistical logic**, not external LLM inference.
2. The strongest implemented cross-cutting AI capability is the **AI governance framework**, which provides:
   - prediction logging,
   - structured and human-readable explanations,
   - model versioning,
   - promotion / rollback,
   - shadow testing,
   - drift monitoring.
3. **Acta** is production-wired and deterministic rather than mocked; its “AI” consists of template generation and heuristic extraction.
4. **Lex** provides meaningful document analysis features, but these are predominantly deterministic analyzers rather than model-backed inference.
5. **Cyber** has the broadest operational AI surface: scoring, classification, anomaly evaluation, threat-rule evaluation, prioritization, and executive reporting.
6. There is **no active external LLM/provider integration** visible in the workspace for OpenAI, Anthropic, Bedrock, Vertex, Ollama, or embedding/chat completion APIs.

## Implemented vs. Marketed

### Clearly implemented

- Acta AI minutes generation
- Acta summary generation
- Acta action extraction
- Lex clause extraction and risk analysis
- Cyber risk scoring and governed detection logging
- AI governance registry, explainability, lifecycle, shadowing, and drift monitoring

### Marketed but currently realized as deterministic logic

- "AI minutes generation" in Acta
- "Contract analysis" and "risk scoring" in Lex
- Parts of cyber "AI-powered threat detection"
- Executive intelligence summaries / briefings

These are still legitimate implemented capabilities, but they are primarily **transparent deterministic systems** rather than black-box ML or LLM systems.

## Recommended wording for stakeholders

> Clario360 currently implements governed, explainable AI capabilities primarily through deterministic, rule-based, and statistical models across Acta, Lex, and Cyber, backed by a full AI governance framework for logging, explainability, lifecycle management, shadow testing, and drift monitoring.

## Source Pointers

- Product overview: `README.md`
- Acta AI: `backend/internal/acta/ai/`
- Lex analyzers: `backend/internal/lex/analyzer/`
- Cyber scoring / detection: `backend/internal/cyber/`
- AI governance framework: `backend/internal/aigovernance/`
- AI governance UI: `frontend/src/app/(dashboard)/admin/ai-governance/`
- Jupyter / SDK access: `deploy/jupyter/sdk/clario360_sdk/clario360/ai/`
