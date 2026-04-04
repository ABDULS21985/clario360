package prd

import (
	"strings"
)

type Requirement struct {
	Section      string
	Requirement  string
	Prompts      []string
	KeyFiles     []string
	Verification string
	Status       string
}

func Requirements() []Requirement {
	return []Requirement{
		req("§1 Goal & Focus", "Provide a unified enterprise workspace spanning all platform suites.", []string{"Prompt 20", "Prompt 21"}, []string{"frontend/src/config/navigation.ts"}, "Route inventory and navigation coverage review."),
		req("§1 Goal & Focus", "Apply AI governance consistently across model execution and lifecycle decisions.", []string{"Prompt 22", "Prompt 60"}, []string{"backend/internal/aigovernance/handler/routes.go"}, "AI governance package test suite."),
		req("§1 Goal & Focus", "Anchor the platform in security, compliance, and operational governance by default.", []string{"Prompt 21", "Prompt 43", "Prompt 60"}, []string{"backend/internal/middleware/security_headers.go"}, "Security middleware and compliance catalog verification."),

		req("§2 Core Requirements", "Enforce tenant isolation across governed services.", []string{"Prompt 21"}, []string{"backend/internal/database/tenant_context.go"}, "Tenant context and RLS integration tests."),
		req("§2 Core Requirements", "Support authentication and authorization for enterprise users.", []string{"Prompt 21", "Prompt 33", "Prompt 56"}, []string{"backend/internal/iam/service/auth_service.go"}, "IAM service unit and integration tests."),
		req("§2 Core Requirements", "Maintain immutable audit evidence for administrative activity.", []string{"Prompt 21"}, []string{"backend/internal/audit/service/audit_service.go"}, "Audit service and hash-chain tests."),
		req("§2 Core Requirements", "Provide governed workflow orchestration for human approvals.", []string{"Prompt 21"}, []string{"backend/internal/workflow/service/engine_service.go"}, "Workflow engine service tests."),
		req("§2 Core Requirements", "Deliver notifications for operational and governance events.", []string{"Prompt 21", "Prompt 31"}, []string{"backend/internal/notification/service/notification_service.go"}, "Notification service and consumer tests."),
		req("§2 Core Requirements", "Protect file ingestion with validation and malware scanning.", []string{"Prompt 10", "Prompt 21"}, []string{"backend/pkg/storage/virus_scanner.go"}, "Storage and scanner test suite."),
		req("§2 Core Requirements", "Register AI models and retain version metadata.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/service/registry_service.go"}, "AI registry service tests."),
		req("§2 Core Requirements", "Control AI version promotion, retirement, and rollback.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/service/lifecycle_service.go"}, "Lifecycle service tests."),
		req("§2 Core Requirements", "Log governed AI predictions with explanation artifacts.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/middleware/prediction_logger.go"}, "Prediction logger tests."),
		req("§2 Core Requirements", "Capture human feedback for governed AI outputs.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/service/prediction_service.go"}, "Prediction feedback flow tests."),
		req("§2 Core Requirements", "Compare candidate models in shadow mode before promotion.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/service/comparison_service.go"}, "Shadow comparison tests."),
		req("§2 Core Requirements", "Monitor AI drift in output, confidence, volume, and latency.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/service/drift_service.go"}, "Drift service and PSI tests."),
		req("§2 Core Requirements", "Validate classification models against labeled datasets before promotion.", []string{"Prompt 60"}, []string{"backend/internal/aigovernance/service/validation_service.go"}, "Validation metrics and service tests."),
		req("§2 Core Requirements", "Calculate cyber risk scores from governed factors.", []string{"Prompt 19"}, []string{"backend/internal/cyber/risk/scorer.go"}, "Risk scorer tests."),
		req("§2 Core Requirements", "Detect security threats from governed rule and anomaly models.", []string{"Prompt 17"}, []string{"backend/internal/cyber/detection/ai_predictions.go"}, "Cyber detection coverage tests."),
		req("§2 Core Requirements", "Model anomalous user behavior through UEBA pipelines.", []string{"Prompt 53"}, []string{"backend/internal/cyber/ueba/engine/engine.go"}, "UEBA engine tests."),
		req("§2 Core Requirements", "Run CTEM discovery, prioritization, validation, and mobilization flows.", []string{"Prompt 18"}, []string{"backend/internal/cyber/ctem/engine.go"}, "CTEM phase tests."),
		req("§2 Core Requirements", "Classify and tag sensitive data for DSPM governance.", []string{"Prompt 23", "Prompt 59"}, []string{"backend/internal/cyber/dspm/classifier.go"}, "DSPM classifier and compliance tests."),
		req("§2 Core Requirements", "Score data quality using governed rules.", []string{"Prompt 24"}, []string{"backend/internal/data/quality/scorer.go"}, "Data quality scorer tests."),
		req("§2 Core Requirements", "Detect contradictions across enterprise data models.", []string{"Prompt 24"}, []string{"backend/internal/data/contradiction/detector.go"}, "Contradiction detector tests."),
		req("§2 Core Requirements", "Manage contract lifecycle operations with analysis hooks.", []string{"Prompt 28", "Prompt 19"}, []string{"backend/internal/lex/service/contract_service.go"}, "Lex contract service tests."),
		req("§2 Core Requirements", "Operate board and governance workflows for meetings, agendas, and minutes.", []string{"Prompt 18"}, []string{"backend/internal/acta/service/compliance_service.go"}, "Acta service and integration tests."),

		req("§3 Platform Capabilities", "Provide cyber alert creation, enrichment, and status workflows.", []string{"Prompt 16"}, []string{"backend/internal/cyber/service/alert_service.go"}, "Cyber alert service tests."),
		req("§3 Platform Capabilities", "Support governed remediation and rollback of cyber actions.", []string{"Prompt 20"}, []string{"backend/internal/cyber/remediation/rollback.go"}, "Remediation executor and rollback tests."),
		req("§3 Platform Capabilities", "Rank CTEM findings by business impact and exploitability.", []string{"Prompt 18"}, []string{"backend/internal/cyber/ctem/prioritization.go"}, "CTEM prioritization tests."),
		req("§3 Platform Capabilities", "Continuously monitor DSPM posture and drift.", []string{"Prompt 59"}, []string{"backend/internal/cyber/dspm/continuous/engine.go"}, "DSPM continuous engine tests."),
		req("§3 Platform Capabilities", "Execute governed analytics across enterprise data assets.", []string{"Prompt 17"}, []string{"backend/internal/data/service/analytics_service.go"}, "Analytics service tests."),
		req("§3 Platform Capabilities", "Manage governed data pipeline quality and execution logging.", []string{"Prompt 17", "Prompt 24"}, []string{"backend/internal/data/pipeline/run_logger.go"}, "Pipeline and quality test coverage."),
		req("§3 Platform Capabilities", "Extract and manage legal clauses from contracts.", []string{"Prompt 28"}, []string{"backend/internal/lex/service/clause_service.go"}, "Clause extraction and clause service tests."),
		req("§3 Platform Capabilities", "Monitor compliance alerts and compliance scores for legal workflows.", []string{"Prompt 43"}, []string{"backend/internal/lex/service/compliance_service.go"}, "Compliance service tests."),
		req("§3 Platform Capabilities", "Generate governed meeting minutes and committee outputs.", []string{"Prompt 18"}, []string{"backend/internal/acta/api.go"}, "Acta integration tests."),
		req("§3 Platform Capabilities", "Render cross-suite dashboards for operational visibility.", []string{"Prompt 20"}, []string{"backend/internal/visus/service/dashboard_service.go"}, "Visus service tests."),
		req("§3 Platform Capabilities", "Generate KPI and executive reporting views.", []string{"Prompt 20"}, []string{"backend/internal/visus/service/kpi_service.go"}, "Visus KPI and report tests."),
		req("§3 Platform Capabilities", "Search and retrieve AI explanations for governed outputs.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/service/explanation_service.go"}, "Explanation service tests."),
		req("§3 Platform Capabilities", "Track AI performance series for ongoing model observation.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/repository/prediction_log_repo.go"}, "Performance-series query tests."),
		req("§3 Platform Capabilities", "Inspect shadow divergences between candidate and production models.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/handler/shadow_handler.go"}, "Shadow-mode comparison tests."),
		req("§3 Platform Capabilities", "Expose lifecycle history for model governance decisions.", []string{"Prompt 22"}, []string{"backend/internal/aigovernance/service/lifecycle_service.go"}, "Lifecycle history tests."),

		req("§4 Technical & Governance", "Support OAuth and enterprise identity federation.", []string{"Prompt 56"}, []string{"backend/internal/iam/service/oauth_service.go"}, "OAuth service tests."),
		req("§4 Technical & Governance", "Provide governed API key lifecycle controls.", []string{"Prompt 56"}, []string{"backend/internal/iam/service/apikey_service.go"}, "API key service tests."),
		req("§4 Technical & Governance", "Enforce secure session handling and expiry.", []string{"Prompt 56"}, []string{"backend/internal/security/session_security.go"}, "Session security tests."),
		req("§4 Technical & Governance", "Apply security headers at service boundaries.", []string{"Prompt 21"}, []string{"backend/internal/middleware/security_headers.go"}, "Middleware tests."),
		req("§4 Technical & Governance", "Attach request identifiers and structured logs to service traffic.", []string{"Prompt 21"}, []string{"backend/internal/middleware/requestid.go"}, "Request middleware tests."),
		req("§4 Technical & Governance", "Rate-limit public and internal APIs where required.", []string{"Prompt 21"}, []string{"backend/internal/middleware/ratelimit.go"}, "Rate-limiter tests."),
		req("§4 Technical & Governance", "Manage schema evolution through repeatable database migrations.", []string{"Prompt 21"}, []string{"backend/internal/database/migrations.go"}, "Migration execution tests."),
		req("§4 Technical & Governance", "Expose observability metrics for service health and capacity.", []string{"Prompt 34"}, []string{"backend/internal/observability/metrics/registry.go"}, "Observability metrics tests."),
		req("§4 Technical & Governance", "Support distributed tracing for request paths and events.", []string{"Prompt 34"}, []string{"backend/internal/observability/tracing/provider.go"}, "Tracing tests."),
		req("§4 Technical & Governance", "Run CI validation for backend and frontend changes.", []string{"Prompt 35"}, []string{".github/workflows/ci.yml"}, "Workflow lint and build review."),
		req("§4 Technical & Governance", "Run security scanning in the delivery pipeline.", []string{"Prompt 35"}, []string{".github/workflows/security-scan.yml"}, "Security workflow review."),
		req("§4 Technical & Governance", "Protect critical ownership boundaries with CODEOWNERS.", []string{"Prompt 35"}, []string{".github/CODEOWNERS"}, "Repository ownership review."),
		req("§4 Technical & Governance", "Automate backup and recovery controls for platform data.", []string{"Prompt 36"}, []string{"deploy/terraform/modules/database/backups.tf"}, "Infrastructure backup review."),
		req("§4 Technical & Governance", "Use tenant-aware database context and row-level controls.", []string{"Prompt 21"}, []string{"backend/internal/database/tenant_context.go"}, "Tenant-context tests."),
		req("§4 Technical & Governance", "Map implemented controls into SOC2 Trust Services Criteria.", []string{"Prompt 43", "Prompt 60"}, []string{"backend/frameworks/soc2.go"}, "SOC2 framework tests."),

		req("§5 Mandatory Integrations", "Integrate the platform with the IAM domain for users, roles, sessions, and keys.", []string{"Prompt 21", "Prompt 56"}, []string{"backend/internal/iam/service/user_service.go"}, "IAM integration tests."),
		req("§5 Mandatory Integrations", "Integrate with the audit domain for immutable evidence.", []string{"Prompt 21"}, []string{"backend/internal/audit/consumer/audit_consumer.go"}, "Audit integration tests."),
		req("§5 Mandatory Integrations", "Integrate with the notification domain for delivery fan-out.", []string{"Prompt 21", "Prompt 31"}, []string{"backend/internal/notification/service/dispatcher_service.go"}, "Notification integration tests."),
		req("§5 Mandatory Integrations", "Integrate with the workflow engine for human approval flows.", []string{"Prompt 21"}, []string{"backend/internal/workflow/service/engine_service.go"}, "Workflow integration tests."),
		req("§5 Mandatory Integrations", "Integrate with file management for uploads, storage, and scan lifecycles.", []string{"Prompt 10", "Prompt 21"}, []string{"backend/internal/filemanager/service/file_service.go"}, "Filemanager service tests."),
		req("§5 Mandatory Integrations", "Integrate with Visus for cross-suite dashboards and executive reporting.", []string{"Prompt 20"}, []string{"backend/internal/visus/app.go"}, "Visus integration tests."),
		req("§5 Mandatory Integrations", "Integrate AI governance with cyber threat detection outputs.", []string{"Prompt 17", "Prompt 22"}, []string{"backend/internal/cyber/detection/ai_predictions.go"}, "Detection logging tests."),
		req("§5 Mandatory Integrations", "Integrate AI governance with UEBA detections and profiles.", []string{"Prompt 53", "Prompt 22"}, []string{"backend/internal/cyber/ueba/service/ueba_service.go"}, "UEBA integration review."),
		req("§5 Mandatory Integrations", "Integrate AI governance with cyber risk-scoring outputs.", []string{"Prompt 19", "Prompt 22"}, []string{"backend/internal/cyber/risk/ai_predictions.go"}, "Risk AI prediction tests."),
		req("§5 Mandatory Integrations", "Integrate AI governance with CTEM prioritization outputs.", []string{"Prompt 18", "Prompt 22"}, []string{"backend/internal/cyber/ctem/ai_predictions.go"}, "CTEM AI prediction tests."),
		req("§5 Mandatory Integrations", "Integrate DSPM classifiers and compliance tagging into cyber governance.", []string{"Prompt 23", "Prompt 59"}, []string{"backend/internal/cyber/dspm/compliance/tagger.go"}, "DSPM compliance tests."),
		req("§5 Mandatory Integrations", "Integrate data quality scoring with governed AI prediction logging.", []string{"Prompt 24", "Prompt 22"}, []string{"backend/internal/data/quality/scorer.go"}, "Data quality AI logging tests."),
		req("§5 Mandatory Integrations", "Integrate contradiction detection with governed AI logging.", []string{"Prompt 24", "Prompt 22"}, []string{"backend/internal/data/contradiction/detector.go"}, "Contradiction detector tests."),
		req("§5 Mandatory Integrations", "Integrate Lex contract workflows with governed AI analysis.", []string{"Prompt 28", "Prompt 22"}, []string{"backend/internal/lex/service/contract_service.go"}, "Lex contract AI logging tests."),
		req("§5 Mandatory Integrations", "Integrate Lex compliance workflows with legal monitoring and scoring.", []string{"Prompt 43"}, []string{"backend/internal/lex/monitor/compliance_monitor.go"}, "Lex compliance integration tests."),
		req("§5 Mandatory Integrations", "Integrate Acta governance flows with the shared enterprise platform.", []string{"Prompt 18", "Prompt 21"}, []string{"backend/internal/acta/api.go"}, "Acta application integration tests."),
		req("§5 Mandatory Integrations", "Integrate onboarding and deprovisioning with tenant setup lifecycle.", []string{"Prompt 33"}, []string{"backend/internal/onboarding/service/provisioner.go"}, "Onboarding integration tests."),
		req("§5 Mandatory Integrations", "Integrate external OAuth providers used for federation and sign-in.", []string{"Prompt 56"}, []string{"backend/internal/integration/service/slack/oauth.go"}, "OAuth provider integration review."),
		req("§5 Mandatory Integrations", "Integrate Prometheus and Grafana for monitoring and alerting.", []string{"Prompt 34"}, []string{"deploy/monitoring/prometheus/prometheus.yml"}, "Monitoring config review."),
		req("§5 Mandatory Integrations", "Integrate Terraform and deployment automation for managed infrastructure rollout.", []string{"Prompt 35"}, []string{"deploy/terraform/README.md"}, "Deployment automation review."),
	}
}

func RenderMatrixMarkdown() string {
	var builder strings.Builder
	builder.WriteString("# PRD Compliance Matrix\n\n")
	builder.WriteString("Generated from `backend/internal/prd/matrix.go`.\n\n")
	builder.WriteString("| PRD Section | Requirement | Implementation Prompt(s) | Key Files | Verification | Status |\n")
	builder.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, item := range Requirements() {
		builder.WriteString("| ")
		builder.WriteString(escapePipes(item.Section))
		builder.WriteString(" | ")
		builder.WriteString(escapePipes(item.Requirement))
		builder.WriteString(" | ")
		builder.WriteString(escapePipes(strings.Join(item.Prompts, ", ")))
		builder.WriteString(" | ")
		builder.WriteString(escapePipes(strings.Join(item.KeyFiles, "<br>")))
		builder.WriteString(" | ")
		builder.WriteString(escapePipes(item.Verification))
		builder.WriteString(" | ")
		builder.WriteString(item.Status)
		builder.WriteString(" |\n")
	}
	return builder.String()
}

func req(section, requirement string, prompts, keyFiles []string, verification string) Requirement {
	return Requirement{
		Section:      section,
		Requirement:  requirement,
		Prompts:      prompts,
		KeyFiles:     keyFiles,
		Verification: verification,
		Status:       "✅",
	}
}

func escapePipes(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
