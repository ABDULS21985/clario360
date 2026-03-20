# PRD Compliance Matrix

Generated from `backend/internal/prd/matrix.go`.

| PRD Section | Requirement | Implementation Prompt(s) | Key Files | Verification | Status |
| --- | --- | --- | --- | --- | --- |
| §1 Goal & Focus | Provide a unified enterprise workspace spanning all platform suites. | Prompt 20, Prompt 21 | frontend/src/config/navigation.ts | Route inventory and navigation coverage review. | ✅ |
| §1 Goal & Focus | Apply AI governance consistently across model execution and lifecycle decisions. | Prompt 22, Prompt 60 | backend/internal/aigovernance/handler/routes.go | AI governance package test suite. | ✅ |
| §1 Goal & Focus | Anchor the platform in security, compliance, and operational governance by default. | Prompt 21, Prompt 43, Prompt 60 | backend/internal/middleware/security_headers.go | Security middleware and compliance catalog verification. | ✅ |
| §2 Core Requirements | Enforce tenant isolation across governed services. | Prompt 21 | backend/internal/database/tenant_context.go | Tenant context and RLS integration tests. | ✅ |
| §2 Core Requirements | Support authentication and authorization for enterprise users. | Prompt 21, Prompt 33, Prompt 56 | backend/internal/iam/service/auth_service.go | IAM service unit and integration tests. | ✅ |
| §2 Core Requirements | Maintain immutable audit evidence for administrative activity. | Prompt 21 | backend/internal/audit/service/audit_service.go | Audit service and hash-chain tests. | ✅ |
| §2 Core Requirements | Provide governed workflow orchestration for human approvals. | Prompt 21 | backend/internal/workflow/service/engine_service.go | Workflow engine service tests. | ✅ |
| §2 Core Requirements | Deliver notifications for operational and governance events. | Prompt 21, Prompt 31 | backend/internal/notification/service/notification_service.go | Notification service and consumer tests. | ✅ |
| §2 Core Requirements | Protect file ingestion with validation and malware scanning. | Prompt 10, Prompt 21 | backend/pkg/storage/virus_scanner.go | Storage and scanner test suite. | ✅ |
| §2 Core Requirements | Register AI models and retain version metadata. | Prompt 22 | backend/internal/aigovernance/service/registry_service.go | AI registry service tests. | ✅ |
| §2 Core Requirements | Control AI version promotion, retirement, and rollback. | Prompt 22 | backend/internal/aigovernance/service/lifecycle_service.go | Lifecycle service tests. | ✅ |
| §2 Core Requirements | Log governed AI predictions with explanation artifacts. | Prompt 22 | backend/internal/aigovernance/middleware/prediction_logger.go | Prediction logger tests. | ✅ |
| §2 Core Requirements | Capture human feedback for governed AI outputs. | Prompt 22 | backend/internal/aigovernance/service/prediction_service.go | Prediction feedback flow tests. | ✅ |
| §2 Core Requirements | Compare candidate models in shadow mode before promotion. | Prompt 22 | backend/internal/aigovernance/service/comparison_service.go | Shadow comparison tests. | ✅ |
| §2 Core Requirements | Monitor AI drift in output, confidence, volume, and latency. | Prompt 22 | backend/internal/aigovernance/service/drift_service.go | Drift service and PSI tests. | ✅ |
| §2 Core Requirements | Validate classification models against labeled datasets before promotion. | Prompt 60 | backend/internal/aigovernance/service/validation_service.go | Validation metrics and service tests. | ✅ |
| §2 Core Requirements | Calculate cyber risk scores from governed factors. | Prompt 19 | backend/internal/cyber/risk/scorer.go | Risk scorer tests. | ✅ |
| §2 Core Requirements | Detect security threats from governed rule and anomaly models. | Prompt 17 | backend/internal/cyber/detection/ai_predictions.go | Cyber detection coverage tests. | ✅ |
| §2 Core Requirements | Model anomalous user behavior through UEBA pipelines. | Prompt 53 | backend/internal/cyber/ueba/engine/engine.go | UEBA engine tests. | ✅ |
| §2 Core Requirements | Run CTEM discovery, prioritization, validation, and mobilization flows. | Prompt 18 | backend/internal/cyber/ctem/engine.go | CTEM phase tests. | ✅ |
| §2 Core Requirements | Classify and tag sensitive data for DSPM governance. | Prompt 23, Prompt 59 | backend/internal/cyber/dspm/classifier.go | DSPM classifier and compliance tests. | ✅ |
| §2 Core Requirements | Score data quality using governed rules. | Prompt 24 | backend/internal/data/quality/scorer.go | Data quality scorer tests. | ✅ |
| §2 Core Requirements | Detect contradictions across enterprise data models. | Prompt 24 | backend/internal/data/contradiction/detector.go | Contradiction detector tests. | ✅ |
| §2 Core Requirements | Manage contract lifecycle operations with analysis hooks. | Prompt 28, Prompt 19 | backend/internal/lex/service/contract_service.go | Lex contract service tests. | ✅ |
| §2 Core Requirements | Operate board and governance workflows for meetings, agendas, and minutes. | Prompt 18 | backend/internal/acta/service/compliance_service.go | Acta service and integration tests. | ✅ |
| §3 Platform Capabilities | Provide cyber alert creation, enrichment, and status workflows. | Prompt 16 | backend/internal/cyber/service/alert_service.go | Cyber alert service tests. | ✅ |
| §3 Platform Capabilities | Support governed remediation and rollback of cyber actions. | Prompt 20 | backend/internal/cyber/remediation/rollback.go | Remediation executor and rollback tests. | ✅ |
| §3 Platform Capabilities | Rank CTEM findings by business impact and exploitability. | Prompt 18 | backend/internal/cyber/ctem/prioritization.go | CTEM prioritization tests. | ✅ |
| §3 Platform Capabilities | Continuously monitor DSPM posture and drift. | Prompt 59 | backend/internal/cyber/dspm/continuous/engine.go | DSPM continuous engine tests. | ✅ |
| §3 Platform Capabilities | Execute governed analytics across enterprise data assets. | Prompt 17 | backend/internal/data/service/analytics_service.go | Analytics service tests. | ✅ |
| §3 Platform Capabilities | Manage governed data pipeline quality and execution logging. | Prompt 17, Prompt 24 | backend/internal/data/pipeline/run_logger.go | Pipeline and quality test coverage. | ✅ |
| §3 Platform Capabilities | Extract and manage legal clauses from contracts. | Prompt 28 | backend/internal/lex/service/clause_service.go | Clause extraction and clause service tests. | ✅ |
| §3 Platform Capabilities | Monitor compliance alerts and compliance scores for legal workflows. | Prompt 43 | backend/internal/lex/service/compliance_service.go | Compliance service tests. | ✅ |
| §3 Platform Capabilities | Generate governed meeting minutes and committee outputs. | Prompt 18 | backend/internal/acta/api.go | Acta integration tests. | ✅ |
| §3 Platform Capabilities | Render cross-suite dashboards for operational visibility. | Prompt 20 | backend/internal/visus/service/dashboard_service.go | Visus service tests. | ✅ |
| §3 Platform Capabilities | Generate KPI and executive reporting views. | Prompt 20 | backend/internal/visus/service/kpi_service.go | Visus KPI and report tests. | ✅ |
| §3 Platform Capabilities | Search and retrieve AI explanations for governed outputs. | Prompt 22 | backend/internal/aigovernance/service/explanation_service.go | Explanation service tests. | ✅ |
| §3 Platform Capabilities | Track AI performance series for ongoing model observation. | Prompt 22 | backend/internal/aigovernance/repository/prediction_log_repo.go | Performance-series query tests. | ✅ |
| §3 Platform Capabilities | Inspect shadow divergences between candidate and production models. | Prompt 22 | backend/internal/aigovernance/handler/shadow_handler.go | Shadow-mode comparison tests. | ✅ |
| §3 Platform Capabilities | Expose lifecycle history for model governance decisions. | Prompt 22 | backend/internal/aigovernance/service/lifecycle_service.go | Lifecycle history tests. | ✅ |
| §4 Technical & Governance | Support OAuth and enterprise identity federation. | Prompt 56 | backend/internal/iam/service/oauth_service.go | OAuth service tests. | ✅ |
| §4 Technical & Governance | Provide governed API key lifecycle controls. | Prompt 56 | backend/internal/iam/service/apikey_service.go | API key service tests. | ✅ |
| §4 Technical & Governance | Enforce secure session handling and expiry. | Prompt 56 | backend/internal/security/session_security.go | Session security tests. | ✅ |
| §4 Technical & Governance | Apply security headers at service boundaries. | Prompt 21 | backend/internal/middleware/security_headers.go | Middleware tests. | ✅ |
| §4 Technical & Governance | Attach request identifiers and structured logs to service traffic. | Prompt 21 | backend/internal/middleware/requestid.go | Request middleware tests. | ✅ |
| §4 Technical & Governance | Rate-limit public and internal APIs where required. | Prompt 21 | backend/internal/middleware/ratelimit.go | Rate-limiter tests. | ✅ |
| §4 Technical & Governance | Manage schema evolution through repeatable database migrations. | Prompt 21 | backend/internal/database/migrations.go | Migration execution tests. | ✅ |
| §4 Technical & Governance | Expose observability metrics for service health and capacity. | Prompt 34 | backend/internal/observability/metrics/registry.go | Observability metrics tests. | ✅ |
| §4 Technical & Governance | Support distributed tracing for request paths and events. | Prompt 34 | backend/internal/observability/tracing/provider.go | Tracing tests. | ✅ |
| §4 Technical & Governance | Run CI validation for backend and frontend changes. | Prompt 35 | .github/workflows/ci.yml | Workflow lint and build review. | ✅ |
| §4 Technical & Governance | Run security scanning in the delivery pipeline. | Prompt 35 | .github/workflows/security-scan.yml | Security workflow review. | ✅ |
| §4 Technical & Governance | Protect critical ownership boundaries with CODEOWNERS. | Prompt 35 | .github/CODEOWNERS | Repository ownership review. | ✅ |
| §4 Technical & Governance | Automate backup and recovery controls for platform data. | Prompt 36 | deploy/terraform/modules/database/backups.tf | Infrastructure backup review. | ✅ |
| §4 Technical & Governance | Use tenant-aware database context and row-level controls. | Prompt 21 | backend/internal/database/tenant_context.go | Tenant-context tests. | ✅ |
| §4 Technical & Governance | Map implemented controls into SOC2 Trust Services Criteria. | Prompt 43, Prompt 60 | backend/frameworks/soc2.go | SOC2 framework tests. | ✅ |
| §5 Mandatory Integrations | Integrate the platform with the IAM domain for users, roles, sessions, and keys. | Prompt 21, Prompt 56 | backend/internal/iam/service/user_service.go | IAM integration tests. | ✅ |
| §5 Mandatory Integrations | Integrate with the audit domain for immutable evidence. | Prompt 21 | backend/internal/audit/consumer/audit_consumer.go | Audit integration tests. | ✅ |
| §5 Mandatory Integrations | Integrate with the notification domain for delivery fan-out. | Prompt 21, Prompt 31 | backend/internal/notification/service/dispatcher_service.go | Notification integration tests. | ✅ |
| §5 Mandatory Integrations | Integrate with the workflow engine for human approval flows. | Prompt 21 | backend/internal/workflow/service/engine_service.go | Workflow integration tests. | ✅ |
| §5 Mandatory Integrations | Integrate with file management for uploads, storage, and scan lifecycles. | Prompt 10, Prompt 21 | backend/internal/filemanager/service/file_service.go | Filemanager service tests. | ✅ |
| §5 Mandatory Integrations | Integrate with Visus for cross-suite dashboards and executive reporting. | Prompt 20 | backend/internal/visus/app.go | Visus integration tests. | ✅ |
| §5 Mandatory Integrations | Integrate AI governance with cyber threat detection outputs. | Prompt 17, Prompt 22 | backend/internal/cyber/detection/ai_predictions.go | Detection logging tests. | ✅ |
| §5 Mandatory Integrations | Integrate AI governance with UEBA detections and profiles. | Prompt 53, Prompt 22 | backend/internal/cyber/ueba/service/ueba_service.go | UEBA integration review. | ✅ |
| §5 Mandatory Integrations | Integrate AI governance with cyber risk-scoring outputs. | Prompt 19, Prompt 22 | backend/internal/cyber/risk/ai_predictions.go | Risk AI prediction tests. | ✅ |
| §5 Mandatory Integrations | Integrate AI governance with CTEM prioritization outputs. | Prompt 18, Prompt 22 | backend/internal/cyber/ctem/ai_predictions.go | CTEM AI prediction tests. | ✅ |
| §5 Mandatory Integrations | Integrate DSPM classifiers and compliance tagging into cyber governance. | Prompt 23, Prompt 59 | backend/internal/cyber/dspm/compliance/tagger.go | DSPM compliance tests. | ✅ |
| §5 Mandatory Integrations | Integrate data quality scoring with governed AI prediction logging. | Prompt 24, Prompt 22 | backend/internal/data/quality/scorer.go | Data quality AI logging tests. | ✅ |
| §5 Mandatory Integrations | Integrate contradiction detection with governed AI logging. | Prompt 24, Prompt 22 | backend/internal/data/contradiction/detector.go | Contradiction detector tests. | ✅ |
| §5 Mandatory Integrations | Integrate Lex contract workflows with governed AI analysis. | Prompt 28, Prompt 22 | backend/internal/lex/service/contract_service.go | Lex contract AI logging tests. | ✅ |
| §5 Mandatory Integrations | Integrate Lex compliance workflows with legal monitoring and scoring. | Prompt 43 | backend/internal/lex/monitor/compliance_monitor.go | Lex compliance integration tests. | ✅ |
| §5 Mandatory Integrations | Integrate Acta governance flows with the shared enterprise platform. | Prompt 18, Prompt 21 | backend/internal/acta/api.go | Acta application integration tests. | ✅ |
| §5 Mandatory Integrations | Integrate onboarding and deprovisioning with tenant setup lifecycle. | Prompt 33 | backend/internal/onboarding/service/provisioner.go | Onboarding integration tests. | ✅ |
| §5 Mandatory Integrations | Integrate external OAuth providers used for federation and sign-in. | Prompt 56 | backend/internal/integration/service/slack/oauth.go | OAuth provider integration review. | ✅ |
| §5 Mandatory Integrations | Integrate Prometheus and Grafana for monitoring and alerting. | Prompt 34 | deploy/monitoring/prometheus/prometheus.yml | Monitoring config review. | ✅ |
| §5 Mandatory Integrations | Integrate Terraform and deployment automation for managed infrastructure rollout. | Prompt 35 | deploy/terraform/README.md | Deployment automation review. | ✅ |
