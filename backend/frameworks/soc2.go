package frameworks

type ImplementationStatus string

const (
	StatusImplemented          ImplementationStatus = "implemented"
	StatusSharedResponsibility ImplementationStatus = "shared_responsibility"
)

type ImplementationRef struct {
	Prompt      string `json:"prompt"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type Control struct {
	ID                   string               `json:"id"`
	Category             string               `json:"category"`
	Title                string               `json:"title"`
	Description          string               `json:"description"`
	ImplementationStatus ImplementationStatus `json:"implementation_status"`
	ImplementationRefs   []ImplementationRef  `json:"implementation_refs"`
	Notes                string               `json:"notes,omitempty"`
}

func SOC2Controls() []Control {
	return []Control{
		control("CC1.1", "common_criteria", "Control Environment", "Audit logging establishes accountable control operation across the platform.", StatusImplemented,
			ref("Prompt 22", "internal/audit/service/audit_service.go", "Governed audit trail for privileged operations.")),
		control("CC1.2", "common_criteria", "Control Environment", "Separation of duties is enforced with role-based authorization boundaries.", StatusImplemented,
			ref("Prompt 33", "internal/iam/service/role_service.go", "Role definitions and least-privilege boundaries.")),
		control("CC1.3", "common_criteria", "Control Environment", "RBAC is enforced in service APIs and administration flows.", StatusImplemented,
			ref("Prompt 33", "internal/iam/service/auth_service.go", "Authorization and permission evaluation.")),
		control("CC1.4", "common_criteria", "Control Environment", "API access is governed through explicit key and session controls.", StatusImplemented,
			ref("Prompt 56", "internal/iam/service/apikey_service.go", "Managed API key lifecycle and revocation.")),
		control("CC1.5", "common_criteria", "Control Environment", "Session security aligns administrative conduct with governed access policy.", StatusImplemented,
			ref("Prompt 56", "internal/security/session_security.go", "Session timeout and token hardening.")),

		control("CC2.1", "communication_information", "Communication & Information", "Real-time dashboards communicate governed platform state to operators.", StatusImplemented,
			ref("Prompt 20", "internal/visus/handler/dashboard_handler.go", "Visus dashboards expose governed operational state.")),
		control("CC2.2", "communication_information", "Communication & Information", "Automated notifications disseminate security and governance events.", StatusImplemented,
			ref("Prompt 31", "internal/notification/service/notification_service.go", "Notification routing and delivery orchestration.")),
		control("CC2.3", "communication_information", "Communication & Information", "Structured reporting artifacts are generated for leadership review.", StatusImplemented,
			ref("Prompt 20", "internal/visus/report/generator.go", "Executive and dashboard reporting pipeline.")),

		control("CC3.1", "risk_assessment", "Risk Assessment", "Risk scoring is codified with reproducible component weights and factors.", StatusImplemented,
			ref("Prompt 19", "internal/cyber/risk/scorer.go", "Organizational risk scoring engine.")),
		control("CC3.2", "risk_assessment", "Risk Assessment", "CTEM coverage supports continuous exposure management and risk discovery.", StatusImplemented,
			ref("Prompt 18", "internal/cyber/ctem/engine.go", "CTEM orchestration across phases.")),
		control("CC3.3", "risk_assessment", "Risk Assessment", "Compliance assessment results feed governed risk posture evaluation.", StatusImplemented,
			ref("Prompt 43", "internal/lex/service/compliance_service.go", "Compliance scoring and alert generation.")),
		control("CC3.4", "risk_assessment", "Risk Assessment", "Exposure prioritization maps business impact and exploitability into ranked risk.", StatusImplemented,
			ref("Prompt 18", "internal/cyber/ctem/prioritization.go", "CTEM prioritization logic.")),

		control("CC4.1", "monitoring_activities", "Monitoring Activities", "Prometheus metrics and alert sources monitor service health and control effectiveness.", StatusImplemented,
			ref("Prompt 34", "../deploy/monitoring/prometheus/prometheus.yml", "Prometheus scrape and alert configuration.")),
		control("CC4.2", "monitoring_activities", "Monitoring Activities", "Security dashboards and UEBA models continuously monitor anomalous behavior.", StatusImplemented,
			ref("Prompt 53", "internal/cyber/ueba/engine/engine.go", "UEBA monitoring and anomaly scoring.")),

		control("CC5.1", "control_activities", "Control Activities", "RBAC controls regulate who may invoke governed administrative actions.", StatusImplemented,
			ref("Prompt 33", "internal/iam/service/role_service.go", "Role assignment and policy evaluation.")),
		control("CC5.2", "control_activities", "Control Activities", "Input validation is applied to inbound requests and workflow expressions.", StatusImplemented,
			ref("Prompt 35", "internal/workflow/expression/sanitizer.go", "Workflow expression sanitization and validation.")),
		control("CC5.3", "control_activities", "Control Activities", "Change management is anchored in reviewed CI/CD and protected ownership rules.", StatusImplemented,
			ref("Prompt 35", "../.github/CODEOWNERS", "Protected ownership for code changes."),
			ref("Prompt 35", "../.github/workflows/ci.yml", "CI checks before deployment.")),

		control("CC6.1", "logical_access", "Logical Access", "Logical access uses RBAC, MFA-adjacent session controls, API keys, and SSO integration points.", StatusImplemented,
			ref("Prompt 56", "internal/iam/service/auth_service.go", "Authentication and access checks."),
			ref("Prompt 56", "internal/iam/service/oauth_service.go", "OAuth and SSO support."),
			ref("Prompt 56", "internal/security/session_security.go", "Session management controls."),
			ref("Prompt 56", "internal/iam/service/apikey_service.go", "API key issuance and revocation.")),
		control("CC6.2", "credential_management", "Credential Management", "Credential storage and rotation controls cover tokens, passwords, and API secrets.", StatusImplemented,
			ref("Prompt 56", "internal/iam/service/auth_service.go", "Password and token handling."),
			ref("Prompt 56", "internal/iam/service/apikey_service.go", "Hashed API key storage.")),
		control("CC6.3", "new_user_registration", "New User Registration", "Onboarding and JIT-style identity flows are controlled through governed registration services.", StatusImplemented,
			ref("Prompt 33", "internal/onboarding/service/wizard_service.go", "Onboarding workflow orchestration."),
			ref("Prompt 56", "internal/iam/service/oauth_service.go", "Identity federation and provisioning hooks.")),
		control("CC6.4", "access_removal", "Access Removal", "Deprovisioning and session invalidation remove access when users leave scope.", StatusImplemented,
			ref("Prompt 33", "internal/onboarding/service/deprovisioner.go", "Access removal and tenant deprovisioning."),
			ref("Prompt 56", "internal/security/session_security.go", "Session invalidation behavior.")),
		control("CC6.5", "physical_access", "Physical Access", "Physical access is delegated to managed cloud and hosting providers outside the application control plane.", StatusSharedResponsibility,
			ref("Prompt 43", "../SECURITY.md", "Shared-responsibility statement for infrastructure controls."),
			note("Application code does not directly manage datacenter access controls.")),
		control("CC6.6", "external_threat_detection", "External Threat Detection", "Threat detection and UEBA pipelines identify external and insider security events.", StatusImplemented,
			ref("Prompt 17", "internal/cyber/detection/ai_predictions.go", "Threat detection model logging."),
			ref("Prompt 53", "internal/cyber/ueba/engine/engine.go", "UEBA signal generation.")),
		control("CC6.7", "transmission_encryption", "Transmission Encryption", "Security headers, OAuth, and service integrations enforce encrypted transport expectations.", StatusImplemented,
			ref("Prompt 56", "internal/middleware/security_headers.go", "Transport and browser security headers."),
			ref("Prompt 56", "internal/iam/service/oauth_service.go", "Secure OAuth flows.")),
		control("CC6.8", "malware_prevention", "Malware Prevention", "File ingestion is scanned and governed before storage and workflow use.", StatusImplemented,
			ref("Prompt 10", "pkg/storage/virus_scanner.go", "ClamAV-backed malware scanning."),
			ref("Prompt 31", "internal/filemanager/service/scan_service.go", "File scan lifecycle integration.")),

		control("CC7.1", "infrastructure_detection", "Infrastructure Detection", "Prometheus and Grafana dashboards surface infrastructure anomalies and saturation.", StatusImplemented,
			ref("Prompt 34", "../deploy/prometheus/prometheus.yml", "Infrastructure scrape configuration."),
			ref("Prompt 34", "../deploy/grafana/dashboards/service-overview.json", "Service observability dashboard.")),
		control("CC7.2", "incident_response", "Incident Response", "Security incidents feed governed alerting, remediation, and escalation workflows.", StatusImplemented,
			ref("Prompt 20", "internal/cyber/service/alert_service.go", "Alert lifecycle and escalation."),
			ref("Prompt 20", "internal/cyber/remediation/rollback.go", "Governed remediation rollback path.")),
		control("CC7.3", "incident_recovery", "Incident Recovery", "Recovery controls exist for workflows and database backup operations.", StatusImplemented,
			ref("Prompt 36", "../deploy/terraform/modules/database/backups.tf", "Database backup and retention configuration."),
			ref("Prompt 36", "internal/workflow/service/recovery_service.go", "Workflow recovery orchestration.")),
		control("CC7.4", "vulnerability_management", "Vulnerability Management", "Exposure and vulnerability posture are prioritized through CTEM and risk components.", StatusImplemented,
			ref("Prompt 16", "internal/cyber/risk/components/vulnerability_risk.go", "Vulnerability risk component."),
			ref("Prompt 18", "internal/cyber/ctem/validation.go", "CTEM validation of exploitable findings.")),

		control("CC8.1", "change_management", "Authorization of Changes", "Code and infrastructure changes require protected ownership and reviewed delivery workflows.", StatusImplemented,
			ref("Prompt 35", "../.github/CODEOWNERS", "Protected ownership for repository changes."),
			ref("Prompt 35", "../.github/workflows/deploy-production.yml", "Controlled production deployment workflow.")),
		control("CC8.2", "change_management", "Infrastructure Change Detection", "Infrastructure definitions and deployment automation make drift visible and reviewable.", StatusImplemented,
			ref("Prompt 35", "../deploy/terraform/README.md", "Terraform-managed infrastructure changes."),
			ref("Prompt 35", "../.github/workflows/deploy-staging.yml", "Staging deployment automation.")),
		control("CC8.3", "change_management", "Testing and Approval", "CI/CD enforces testing and security scanning before changes land.", StatusImplemented,
			ref("Prompt 35", "../.github/workflows/ci.yml", "Continuous integration pipeline."),
			ref("Prompt 35", "../.github/workflows/security-scan.yml", "Security scan gate.")),

		control("CC9.1", "risk_mitigation", "Risk Mitigation", "Risk mitigation is governed through remediation workflows and weighted scoring.", StatusImplemented,
			ref("Prompt 20", "internal/cyber/remediation/rollback.go", "Governed remediation controls."),
			ref("Prompt 19", "internal/cyber/risk/recommendations.go", "Risk mitigation recommendations.")),
		control("CC9.2", "vendor_risk", "Vendor Risk", "Contract analysis and compliance monitoring support vendor-risk governance.", StatusImplemented,
			ref("Prompt 28", "internal/lex/service/contract_service.go", "Contract lifecycle and analysis."),
			ref("Prompt 28", "internal/lex/analyzer/clause_extractor.go", "Clause extraction for vendor obligations.")),

		control("A1.1", "availability", "Processing Capacity", "Capacity and saturation are observable through service metrics and dashboards.", StatusImplemented,
			ref("Prompt 34", "internal/observability/metrics/registry.go", "Service capacity metrics registry."),
			ref("Prompt 34", "../deploy/grafana/dashboards/database-overview.json", "Database capacity dashboard.")),
		control("A1.2", "availability", "Infrastructure Recovery", "Recovery depends on infrastructure backup automation and service recovery paths.", StatusImplemented,
			ref("Prompt 36", "../deploy/terraform/modules/database/backups.tf", "Backup configuration."),
			ref("Prompt 36", "internal/middleware/recovery.go", "Service recovery middleware.")),

		control("C1.1", "confidentiality", "Confidential Information", "PII classification and DSPM scanning govern confidential data handling.", StatusImplemented,
			ref("Prompt 23", "internal/cyber/dspm/classifier.go", "PII and sensitive-data classification."),
			ref("Prompt 59", "internal/cyber/dspm/compliance/tagger.go", "Compliance tagging for sensitive data.")),
		control("C1.2", "confidentiality", "Confidential Information Disposal", "Deprovisioning and governed file lifecycles support confidential data disposal.", StatusImplemented,
			ref("Prompt 33", "internal/onboarding/service/deprovisioner.go", "Tenant and access deprovisioning."),
			ref("Prompt 33", "internal/filemanager/service/lifecycle_service.go", "File lifecycle and deletion controls.")),

		control("PI1.1", "processing_integrity", "Processing Accuracy", "Data quality scoring and contradiction detection validate processing accuracy.", StatusImplemented,
			ref("Prompt 24", "internal/data/quality/scorer.go", "Data quality rules and scoring."),
			ref("Prompt 24", "internal/data/contradiction/detector.go", "Contradiction detection for data integrity.")),

		control("P1.1", "privacy", "Privacy Notice", "Privacy-relevant data is tagged with compliance classifications for downstream handling.", StatusImplemented,
			ref("Prompt 59", "internal/cyber/dspm/compliance/tagger.go", "Framework-specific privacy tagging.")),
		control("P1.2", "privacy", "Choice and Consent", "Data-source and pipeline governance enables configurable handling decisions by source.", StatusImplemented,
			ref("Prompt 59", "internal/cyber/dspm/continuous/watchers/pipeline_watcher.go", "Source-aware compliance and governance checks.")),
		control("P1.3", "privacy", "Personal Information Collection", "PII classifiers label personal data collected through governed sources and scans.", StatusImplemented,
			ref("Prompt 23", "internal/cyber/dspm/classifier.go", "PII classification engine."),
			ref("Prompt 59", "internal/cyber/dspm/compliance/soc2.go", "SOC2 tagging of personal data controls.")),
	}
}

func control(id, category, title, description string, status ImplementationStatus, refs ...ImplementationRef) Control {
	item := Control{
		ID:                   id,
		Category:             category,
		Title:                title,
		Description:          description,
		ImplementationStatus: status,
		ImplementationRefs:   make([]ImplementationRef, 0, len(refs)),
	}
	for _, ref := range refs {
		if ref.Path == "" && ref.Description == "" && ref.Prompt == "" {
			continue
		}
		if ref.Path == "" && ref.Description != "" {
			item.Notes = ref.Description
			continue
		}
		item.ImplementationRefs = append(item.ImplementationRefs, ref)
	}
	return item
}

func ref(prompt, path, description string) ImplementationRef {
	return ImplementationRef{Prompt: prompt, Path: path, Description: description}
}

func note(description string) ImplementationRef {
	return ImplementationRef{Description: description}
}
