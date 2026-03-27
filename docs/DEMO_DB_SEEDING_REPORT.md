# Demo DB Seeding Report

Generated from a repository scan on 2026-03-27.

## Scope

This report identifies the parts of the codebase that are good candidates for tenant-scoped demo data seeding.

Method used:

- Frontend pages and navigation were scanned to find demo-visible modules.
- Backend handlers, repositories, models, and migrations were matched to those modules.
- SQL migrations were treated as the main DB truth source.
- Repository-level schema definitions were also included where the schema is not stored in `backend/migrations/`:
  - `backend/internal/workflow/repository/schema.go`
  - `backend/internal/notification/repository/schema.go`
- The older root bootstrap migration in `backend/migrations/000001_init_schema.up.sql` appears legacy/parallel to the modular `platform_core` schema, so the modular schemas were treated as primary.

Seed status labels used below:

- `Existing`: there is already a seeder or demo seed flow in the codebase.
- `Partial`: some related seed data exists, but not the whole module.
- `Candidate`: DB-backed and demo-visible, but no dedicated seeder was found.
- `Generated`: mainly operational/runtime data; seed only if the demo specifically needs it.

## Executive Summary

High-confidence demo-seedable areas found across the codebase:

1. Platform admin and IAM
2. Onboarding and invitations
3. AI governance registry, runtime analytics, and compute benchmarking
4. Workflow definitions, instances, and human tasks
5. Notifications, webhooks, integrations, and ticket links
6. File metadata and attachments
7. Cyber suite: SOC, threat, CTEM, remediation, DSPM, UEBA, vCISO
8. Data suite: sources, models, pipelines, quality, contradictions, lineage, dark data, analytics
9. ACTA
10. LEX
11. VISUS
12. Audit trails and chain state

Areas already seeded today:

- Cyber inventory seed: `backend/cmd/seeder/main.go`
- Data suite seed: `backend/cmd/data-seeder/main.go`
- Cross-suite Prompt 59 scenario seed: `backend/cmd/prompt59-seeder/main.go`
- ACTA demo seed: `backend/internal/acta/seed.go`
- LEX demo seed: `backend/internal/lex/seed.go`
- VISUS demo seed: `backend/internal/visus/seed.go`
- AI model registry seed: `backend/internal/aigovernance/seeder/model_seeder.go`
- Tenant bootstrap seeders:
  - roles
  - settings
  - detection rules
  - VISUS KPIs/dashboard
  - LEX compliance rules
  - workflow templates

## Existing Seed Coverage

These modules already have working seed entry points and should be reused for demos before building new seeders.

| Module | Status | Current seed path | Notes |
| --- | --- | --- | --- |
| Cyber asset inventory | Existing | `backend/cmd/seeder/main.go` | Seeds assets, vulnerabilities, and asset relationships. |
| Cyber incident scenario | Existing | `backend/cmd/prompt59-seeder/main.go` | Seeds alerts, alert timeline, UEBA records, DSPM data, data sources, pipelines, and lineage for a narrative RCA demo. |
| Data sources and models | Existing | `backend/cmd/data-seeder/main.go` | Seeds `data_sources`, `sync_history`, and `data_models`. |
| ACTA core governance | Existing | `backend/internal/acta/seed.go` | Seeds committees, members, meetings, attendance, agenda, minutes, and action items. |
| LEX contract lifecycle | Existing | `backend/internal/lex/seed.go` | Seeds contracts, versions, clauses, analyses, legal documents, compliance rules, and alerts. |
| VISUS executive reporting | Existing | `backend/internal/visus/seed.go` | Seeds KPI definitions, dashboards, widgets, KPI snapshots, alerts, reports, and report snapshots. |
| AI model registry | Existing | `backend/internal/aigovernance/seeder/model_seeder.go` and `backend/migrations/platform_core/000009_seed_ai_governance_models.up.sql` | Good baseline for model registry demos. |
| Tenant bootstrap | Existing | `backend/internal/onboarding/service/seeder/*.go` | Seeds roles, settings, cyber detection rules, VISUS KPIs/dashboard, LEX compliance rules, and workflow templates. |

## Seedable Module Inventory

### Shared Platform and Admin

| Module | Status | Main DB entities | Why it is seedable |
| --- | --- | --- | --- |
| Tenant and IAM bootstrap | Partial | `tenants`, `users`, `roles`, `user_roles`, `sessions`, `api_keys`, `password_reset_tokens` | Core admin pages and IAM handlers exist; roles already have a seeder, but tenant/user/API-key demo data is still worth adding. |
| Tenant onboarding and invites | Partial | `tenant_onboarding`, `provisioning_steps`, `email_verifications`, `invitations` | Frontend onboarding and admin invitation flows exist; invitation/onboarding repositories are DB-backed. |
| System settings | Existing | `system_settings` | Already seeded during onboarding and useful for admin demos. |
| Platform audit logs | Candidate | `audit_logs` in `platform_core` | Admin audit page is DB-backed and useful for compliance demos. |
| Dedicated audit service | Candidate | `audit_logs`, `audit_chain_state` in `audit_db` | Good for immutable audit/evidence demonstrations. |
| AI governance model registry | Existing | `ai_models`, `ai_model_versions` | Already seeded; can be expanded with more suite-specific models. |
| AI governance runtime analytics | Candidate | `ai_prediction_logs`, `ai_shadow_comparisons`, `ai_drift_reports`, `ai_validation_results` | Admin AI governance detail pages, validation, shadow, drift, and prediction handlers all read persisted data. |
| AI governance compute and benchmarks | Candidate | `ai_inference_servers`, `ai_benchmark_suites`, `ai_benchmark_runs`, `ai_compute_cost_models` | Admin compute and benchmark pages are DB-backed and ideal for demo performance stories. |
| Workflow definitions and templates | Partial | `workflow_definitions`, `workflow_templates` | Template seeding already exists; more tenant-specific workflow definitions are easy to seed. |
| Workflow runtime | Candidate | `workflow_instances`, `workflow_step_executions`, `workflow_tasks`, `workflow_timers` | Task queue, instances, and analytics pages are all DB-backed and demo-friendly. |
| Notifications center | Candidate | `notifications`, `notification_preferences`, `notification_delivery_log`, `notification_webhooks`, `notification_templates` | User and admin notification pages are fully DB-backed. |
| Integrations and ticket sync | Candidate | `integrations`, `integration_deliveries`, `external_ticket_links` | Admin integrations UI and routes exist; schema lives in `backend/internal/notification/repository/schema.go`. |
| File metadata and attachments | Candidate | `files`, `file_access_log`, `file_quarantine_log` | Shared attachment model used across suites; useful for demos involving evidence, minutes, contracts, and uploads. |

### Cyber Suite

| Module | Status | Main DB entities | Why it is seedable |
| --- | --- | --- | --- |
| Asset inventory | Existing | `assets`, `asset_relationships`, `scan_history`, `asset_activity` | Fully demo-visible and already seeded in the cyber seeder. |
| Vulnerability management | Existing | `vulnerabilities`, `cve_database` | Backed by asset/vulnerability repos and already partially seeded. |
| Alerts and investigations | Existing | `alerts`, `alert_comments`, `alert_timeline`, `security_events` | Alert list/detail, timeline, comments, and event explorer are DB-backed; Prompt 59 already covers part of this. |
| Detection rules | Partial | `detection_rules` | Onboarding seeds default rules; a richer demo dataset with performance and feedback is still useful. |
| Threat intelligence | Candidate | `threats`, `threat_indicators` | Threat hunting and IOC pages exist, but no dedicated threat seed was found. |
| Threat feeds | Candidate | `threat_feed_configs`, `threat_feed_sync_history` | Threat feed admin screens are DB-backed and demo-friendly. |
| CTEM | Candidate | `ctem_assessments`, `ctem_findings`, `ctem_remediation_groups`, `exposure_score_snapshots` | CTEM dashboard and assessment flows are persisted and should have a demo dataset. |
| Risk scoring and heatmap | Candidate | `risk_score_history` | The risk endpoints derive from persisted scoring history and related cyber entities. |
| Remediation lifecycle | Candidate | `remediation_actions`, `remediation_audit_trail` | Full approval and execution flow exists, but no dedicated seeder was found. |
| DSPM core | Partial | `dspm_data_assets`, `dspm_scans` | Prompt 59 seeds some DSPM asset/scan data; a broader DSPM tenant seed would help. |
| DSPM access intelligence | Candidate | `dspm_access_mappings`, `dspm_identity_profiles`, `dspm_access_audit`, `dspm_access_policies` | Access intelligence has dedicated routes and models but no seed path. |
| DSPM remediation and governance | Candidate | `dspm_remediations`, `dspm_remediation_history`, `dspm_data_policies`, `dspm_risk_exceptions` | Remediation, policies, and exceptions are first-class DB modules. |
| DSPM advanced intelligence | Candidate | `dspm_data_lineage`, `dspm_ai_data_usage`, `dspm_classification_history`, `dspm_compliance_posture`, `dspm_financial_impact` | These back advanced DSPM pages such as lineage, AI usage, compliance, and financial impact. |
| UEBA | Partial | `ueba_profiles`, `ueba_access_events`, `ueba_alerts` | Prompt 59 seeds some UEBA profile/alert data, but not a full UEBA behavioral dataset. |
| vCISO briefings and chat | Candidate | `vciso_briefings`, `vciso_conversations`, `vciso_messages` | Briefing/chat flows are persisted and visible in the UI. |
| vCISO LLM governance | Candidate | `vciso_llm_audit_log`, `vciso_llm_system_prompts`, `vciso_llm_rate_limits` | Good candidate for demoing governed AI usage and prompt versioning. |
| vCISO predictive analytics | Candidate | `vciso_predictions`, `vciso_prediction_models`, `vciso_feature_snapshots` | Dedicated predictive endpoints exist but no runtime seeder was found. |
| vCISO governance program management | Partial | `vciso_risks`, `vciso_policies`, `vciso_policy_exceptions`, `vciso_vendors`, `vciso_questionnaires`, `vciso_evidence`, `vciso_maturity_assessments`, `vciso_budget_items`, `vciso_awareness_programs`, `vciso_iam_findings`, `vciso_escalation_rules`, `vciso_playbooks`, `vciso_obligations`, `vciso_control_tests`, `vciso_integrations`, `vciso_control_ownership`, `vciso_approvals`, `vciso_benchmarks`, `vciso_control_dependencies` | This area is heavily DB-backed. Migration `000016_vciso_governance.up.sql` even includes inline sample rows, but a proper tenant-aware seeder would be better. |

### Data Suite

| Module | Status | Main DB entities | Why it is seedable |
| --- | --- | --- | --- |
| Data sources and sync history | Existing | `data_sources`, `sync_history` | Already covered by `data-seeder`. |
| Data models | Existing | `data_models` | Already covered by `data-seeder`. |
| Data catalogs | Candidate | `data_catalogs` | Present in the schema but not currently covered by the seeders. |
| Pipelines and run history | Partial | `pipelines`, `pipeline_runs`, `pipeline_run_logs` | Prompt 59 seeds pipelines/runs; general tenant-level pipeline seeding would improve demos. |
| Data quality | Candidate | `data_quality_rules`, `data_quality_results` | Quality dashboards and handlers are persisted but not explicitly seeded. |
| Contradictions | Candidate | `contradictions`, `contradiction_scans` | Contradiction pages are DB-backed and need demo records. |
| Data lineage | Partial | `data_lineage`, `data_lineage_edges` | Prompt 59 seeds some lineage edges; broader lineage graph seeding is still a good target. |
| Dark data | Candidate | `dark_data_scans`, `dark_data_assets` | Dark data pages are DB-backed, but no dedicated seeder was found. |
| Analytics workbench | Candidate | `saved_queries`, `analytics_audit_log` | Saved query and analytics audit features are persisted and are strong demo modules. |

### ACTA

| Module | Status | Main DB entities | Why it is seedable |
| --- | --- | --- | --- |
| Committees and members | Existing | `committees`, `committee_members` | Already seeded. |
| Meetings and attendance | Existing | `meetings`, `meeting_attendance` | Already seeded. |
| Agenda management | Existing | `agenda_items` | Already seeded. |
| Minutes lifecycle | Existing | `meeting_minutes` | Already seeded. |
| Action items | Existing | `action_items` | Already seeded. |
| Compliance checks | Candidate | `compliance_checks` | Compliance pages/repositories exist, but the ACTA demo seed does not populate compliance checks yet. |

### LEX

| Module | Status | Main DB entities | Why it is seedable |
| --- | --- | --- | --- |
| Contracts and versions | Existing | `contracts`, `contract_versions` | Already seeded. |
| Clause extraction and analysis | Existing | `contract_clauses`, `contract_analyses` | Already seeded. |
| Legal documents | Existing | `legal_documents`, `document_versions` | Already seeded. |
| Compliance monitoring | Existing | `compliance_rules`, `compliance_alerts` | Seeded via LEX demo seed and onboarding compliance-rule seeder. |
| Expiry notifications | Candidate | `expiry_notifications` | Expiry warnings are DB-backed but not explicitly seeded yet. |

### VISUS

| Module | Status | Main DB entities | Why it is seedable |
| --- | --- | --- | --- |
| Dashboards and widgets | Existing | `visus_dashboards`, `visus_widgets` | Already seeded. |
| KPI definitions and snapshots | Existing | `visus_kpi_definitions`, `visus_kpi_snapshots` | Already seeded. |
| Executive alerts | Existing | `visus_executive_alerts` | Already seeded. |
| Reports and snapshots | Existing | `visus_report_definitions`, `visus_report_snapshots` | Already seeded. |
| Suite cache | Generated | `visus_suite_cache` | Useful only if the demo depends on cached cross-suite rollups. Usually better generated from live suite data. |

## Best Next Seed Candidates

If the goal is to maximize demo value quickly, the next seeders I would build in this order are:

1. Workflow runtime demo seed
   - Definitions, instances, human tasks, approvals, and timers
   - This unlocks `/workflows`, `/workflows/tasks`, and admin workflow pages
2. Notifications + webhooks + integrations seed
   - Notifications, user preferences, webhook delivery history, integration configs, external ticket links
   - This unlocks `/notifications`, `/admin/notifications`, `/admin/integrations`
3. AI governance runtime seed
   - Prediction logs, validation results, shadow comparisons, drift reports, inference servers, benchmark suites/runs
   - This unlocks the deeper AI governance pages beyond model registry
4. Data quality / contradictions / dark data / analytics seed
   - These are visible modules with no full demo dataset yet
5. Cyber advanced seed pack
   - Threat intel, CTEM, remediation lifecycle, threat feeds, UEBA access events
6. DSPM advanced seed pack
   - Access intelligence, remediations, policies, exceptions, compliance posture, financial impact
7. vCISO governance tenant seed
   - Risks, policies, vendors, evidence, maturity, budget, approvals, playbooks, integrations
8. File attachment seed
   - Shared attachment/evidence files across ACTA, LEX, cyber, and vCISO evidence views
9. ACTA compliance checks seed
10. LEX expiry-notification seed

## Modules That Are Not Strong First-Class DB Seed Targets

These areas exist in the product but are not primarily backed by their own application tables in this repo:

- Notebook workspace
  - Profiles and templates are code-defined in `backend/internal/notebook/model/notebook.go`
  - Server lifecycle appears to rely on an external notebook hub plus emitted events, not first-party DB tables
- Purely computed dashboards
  - Some overview/risk/summary endpoints are derived from other persisted tables rather than needing their own seeded table

## Key Evidence Files

Primary files used to build this report:

- Frontend navigation and pages
  - `frontend/src/config/navigation.ts`
  - `frontend/src/app/**/page.tsx`
- Existing seed flows
  - `backend/cmd/seeder/main.go`
  - `backend/cmd/data-seeder/main.go`
  - `backend/cmd/prompt59-seeder/main.go`
  - `backend/internal/acta/seed.go`
  - `backend/internal/lex/seed.go`
  - `backend/internal/visus/seed.go`
  - `backend/internal/aigovernance/seeder/model_seeder.go`
  - `backend/internal/onboarding/service/seeder/*.go`
- Modular schema sources
  - `backend/migrations/platform_core/*.sql`
  - `backend/migrations/audit_db/*.sql`
  - `backend/migrations/cyber_db/*.sql`
  - `backend/migrations/data_db/*.sql`
  - `backend/migrations/acta_db/*.sql`
  - `backend/migrations/lex_db/*.sql`
  - `backend/migrations/notification_db/*.sql`
  - `backend/migrations/visus_db/*.sql`
  - `backend/internal/workflow/repository/schema.go`
  - `backend/internal/notification/repository/schema.go`
  - `backend/migrations/000010_create_file_storage_tables.up.sql`

## Conclusion

The codebase already supports a broad demo seeding strategy.

The fastest path is:

- reuse the existing ACTA, LEX, VISUS, cyber, data, onboarding, and AI model seeders
- then add new seeders for workflow runtime, notifications/integrations, AI governance runtime analytics, and the unseeded data/cyber advanced modules

That combination would cover nearly every major dashboard and detail page in the product with realistic tenant-scoped demo data.
