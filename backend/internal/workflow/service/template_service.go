package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/workflow/model"
)

// TemplateService provides access to built-in workflow templates and can
// instantiate them as tenant-specific workflow definitions.
type TemplateService struct {
	defRepo   definitionRepo
	logger    zerolog.Logger
	templates []*model.WorkflowTemplate
}

// NewTemplateService creates a new TemplateService with built-in templates pre-loaded.
func NewTemplateService(defRepo definitionRepo, logger zerolog.Logger) *TemplateService {
	svc := &TemplateService{
		defRepo: defRepo,
		logger:  logger.With().Str("service", "workflow-template").Logger(),
	}
	svc.templates = svc.buildTemplates()
	return svc
}

// ListTemplates returns built-in templates, optionally filtered by category.
func (s *TemplateService) ListTemplates(ctx context.Context, category string) ([]*model.WorkflowTemplate, error) {
	if category == "" {
		return s.templates, nil
	}

	var filtered []*model.WorkflowTemplate
	for _, t := range s.templates {
		if t.Category == category {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

// GetTemplate returns a single template by ID.
func (s *TemplateService) GetTemplate(ctx context.Context, id string) (*model.WorkflowTemplate, error) {
	for _, t := range s.templates {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, model.ErrNotFound
}

// InstantiateTemplate creates a new workflow definition from a template,
// scoped to the given tenant. Optional name and description overrides are
// applied before persisting; empty strings fall back to the template defaults.
func (s *TemplateService) InstantiateTemplate(ctx context.Context, tenantID, userID, templateID, nameOverride, descOverride string) (*model.WorkflowDefinition, error) {
	tmpl, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template %s not found: %w", templateID, err)
	}

	// Parse the template definition JSON.
	var templateDef templateDefinition
	if err := json.Unmarshal(tmpl.DefinitionJSON, &templateDef); err != nil {
		return nil, fmt.Errorf("parsing template definition: %w", err)
	}

	defName := templateDef.Name
	if nameOverride != "" {
		defName = nameOverride
	}
	defDesc := templateDef.Description
	if descOverride != "" {
		defDesc = descOverride
	}

	now := time.Now().UTC()
	def := &model.WorkflowDefinition{
		ID:            generateUUID(),
		TenantID:      tenantID,
		Name:          defName,
		Description:   defDesc,
		Version:       1,
		Status:        model.DefinitionStatusDraft,
		TriggerConfig: templateDef.TriggerConfig,
		Variables:     templateDef.Variables,
		Steps:         templateDef.Steps,
		CreatedBy:     userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if def.Variables == nil {
		def.Variables = make(map[string]model.VariableDef)
	}

	if err := s.defRepo.Create(ctx, def); err != nil {
		return nil, fmt.Errorf("creating definition from template: %w", err)
	}

	s.logger.Info().
		Str("template_id", templateID).
		Str("definition_id", def.ID).
		Str("tenant_id", tenantID).
		Str("created_by", userID).
		Msg("workflow definition created from template")

	return def, nil
}

// templateDefinition is the intermediate structure for parsing template JSON.
type templateDefinition struct {
	Name          string                        `json:"name"`
	Description   string                        `json:"description"`
	TriggerConfig model.TriggerConfig           `json:"trigger_config"`
	Variables     map[string]model.VariableDef  `json:"variables"`
	Steps         []model.StepDefinition        `json:"steps"`
}

// buildTemplates constructs the built-in template catalog.
func (s *TemplateService) buildTemplates() []*model.WorkflowTemplate {
	now := time.Now().UTC()

	return []*model.WorkflowTemplate{
		s.alertRemediationTemplate(now),
		s.contractReviewTemplate(now),
		s.boardMeetingTemplate(now),
		s.dataAccessRequestTemplate(now),
		s.changeRequestTemplate(now),
	}
}

// ---------- Template 1: Alert Remediation ----------

func (s *TemplateService) alertRemediationTemplate(now time.Time) *model.WorkflowTemplate {
	def := templateDefinition{
		Name:        "Alert Remediation",
		Description: "Cybersecurity alert triage, investigation, and remediation workflow. Automatically triages incoming alerts, assigns to analysts for investigation, and tracks remediation actions through to completion.",
		TriggerConfig: model.TriggerConfig{
			Type:  model.TriggerTypeEvent,
			Topic: "cyber.alert.events",
			Filter: map[string]interface{}{
				"severity": []interface{}{"high", "critical"},
			},
		},
		Variables: map[string]model.VariableDef{
			"alert_id": {
				Type:   "string",
				Source: "alert_id",
			},
			"severity": {
				Type:   "string",
				Source: "severity",
			},
			"alert_type": {
				Type:   "string",
				Source: "alert_type",
			},
			"affected_assets": {
				Type:   "array",
				Source: "affected_assets",
			},
			"is_valid_threat": {
				Type:    "boolean",
				Default: false,
			},
			"remediation_status": {
				Type:    "string",
				Default: "pending",
			},
		},
		Steps: []model.StepDefinition{
			{
				ID:   "triage",
				Type: model.StepTypeServiceTask,
				Name: "Auto Triage Alert",
				Config: map[string]interface{}{
					"service": "threat-intel",
					"method":  "POST",
					"url":     "/api/v1/alerts/${variables.alert_id}/triage",
					"body": map[string]interface{}{
						"alert_id": "${variables.alert_id}",
						"severity": "${variables.severity}",
					},
					"max_retries": 3,
				},
				Transitions: []model.Transition{
					{Condition: "steps.triage.output.is_valid == true", Target: "investigate"},
					{Condition: "steps.triage.output.is_valid == false", Target: "close_false_positive"},
				},
			},
			{
				ID:   "investigate",
				Type: model.StepTypeHumanTask,
				Name: "Investigate Alert",
				Config: map[string]interface{}{
					"assignee_role": "security_analyst",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "investigation_notes",
							"type":     "textarea",
							"label":    "Investigation Notes",
							"required": true,
						},
						map[string]interface{}{
							"name":     "confirmed_threat",
							"type":     "boolean",
							"label":    "Confirmed Threat?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "recommended_action",
							"type":     "select",
							"label":    "Recommended Action",
							"required": true,
							"options":  []interface{}{"isolate", "patch", "block_ip", "escalate", "monitor"},
						},
					},
					"sla_hours":       4,
					"escalation_role": "security_lead",
				},
				Transitions: []model.Transition{
					{Condition: "steps.investigate.output.confirmed_threat == true", Target: "remediate"},
					{Target: "close_false_positive"},
				},
			},
			{
				ID:   "remediate",
				Type: model.StepTypeServiceTask,
				Name: "Execute Remediation",
				Config: map[string]interface{}{
					"service": "remediation",
					"method":  "POST",
					"url":     "/api/v1/remediations",
					"body": map[string]interface{}{
						"alert_id":   "${variables.alert_id}",
						"action":     "${steps.investigate.output.recommended_action}",
						"assets":     "${variables.affected_assets}",
					},
					"max_retries": 2,
				},
				Transitions: []model.Transition{
					{Target: "verify"},
				},
			},
			{
				ID:   "verify",
				Type: model.StepTypeHumanTask,
				Name: "Verify Remediation",
				Config: map[string]interface{}{
					"assignee_role": "security_analyst",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "remediation_effective",
							"type":     "boolean",
							"label":    "Was the remediation effective?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "verification_notes",
							"type":     "textarea",
							"label":    "Verification Notes",
							"required": true,
						},
					},
					"sla_hours": 8,
				},
				Transitions: []model.Transition{
					{Target: "close_resolved"},
				},
			},
			{
				ID:   "close_false_positive",
				Type: model.StepTypeServiceTask,
				Name: "Close as False Positive",
				Config: map[string]interface{}{
					"service": "alert-manager",
					"method":  "PATCH",
					"url":     "/api/v1/alerts/${variables.alert_id}/close",
					"body": map[string]interface{}{
						"resolution": "false_positive",
					},
				},
				Transitions: []model.Transition{
					{Target: "end"},
				},
			},
			{
				ID:   "close_resolved",
				Type: model.StepTypeServiceTask,
				Name: "Close as Resolved",
				Config: map[string]interface{}{
					"service": "alert-manager",
					"method":  "PATCH",
					"url":     "/api/v1/alerts/${variables.alert_id}/close",
					"body": map[string]interface{}{
						"resolution": "resolved",
					},
				},
				Transitions: []model.Transition{
					{Target: "end"},
				},
			},
			{
				ID:   "end",
				Type: model.StepTypeEnd,
				Name: "End",
			},
		},
	}

	defJSON, _ := json.Marshal(def)
	return &model.WorkflowTemplate{
		ID:             "tmpl-alert-remediation",
		Name:           "Alert Remediation",
		Description:    "Cybersecurity alert triage, investigation, and remediation workflow with auto-triage, analyst review, and tracked remediation.",
		Category:       "cybersecurity",
		DefinitionJSON: defJSON,
		Icon:           "shield-alert",
		CreatedAt:      now,
	}
}

// ---------- Template 2: Contract Review ----------

func (s *TemplateService) contractReviewTemplate(now time.Time) *model.WorkflowTemplate {
	def := templateDefinition{
		Name:        "Contract Review",
		Description: "Legal contract review workflow with document upload, legal review, compliance check, approval, and sign-off stages.",
		TriggerConfig: model.TriggerConfig{
			Type: model.TriggerTypeManual,
		},
		Variables: map[string]model.VariableDef{
			"contract_id": {
				Type: "string",
			},
			"contract_type": {
				Type:    "string",
				Default: "standard",
			},
			"contract_value": {
				Type:    "number",
				Default: float64(0),
			},
			"requesting_department": {
				Type: "string",
			},
			"vendor_name": {
				Type: "string",
			},
		},
		Steps: []model.StepDefinition{
			{
				ID:   "upload",
				Type: model.StepTypeHumanTask,
				Name: "Upload Contract Document",
				Config: map[string]interface{}{
					"assignee_role": "contract_submitter",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "document_url",
							"type":     "text",
							"label":    "Document URL",
							"required": true,
						},
						map[string]interface{}{
							"name":     "contract_summary",
							"type":     "textarea",
							"label":    "Contract Summary",
							"required": true,
						},
						map[string]interface{}{
							"name":     "contract_type",
							"type":     "select",
							"label":    "Contract Type",
							"required": true,
							"options":  []interface{}{"standard", "nda", "sow", "msa", "amendment"},
						},
						map[string]interface{}{
							"name":     "contract_value",
							"type":     "number",
							"label":    "Contract Value (USD)",
							"required": true,
						},
					},
				},
				Transitions: []model.Transition{
					{Target: "legal_review"},
				},
			},
			{
				ID:   "legal_review",
				Type: model.StepTypeHumanTask,
				Name: "Legal Review",
				Config: map[string]interface{}{
					"assignee_role": "legal_counsel",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "legal_approved",
							"type":     "boolean",
							"label":    "Legally Approved?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "legal_notes",
							"type":     "textarea",
							"label":    "Legal Review Notes",
							"required": true,
						},
						map[string]interface{}{
							"name":     "risk_level",
							"type":     "select",
							"label":    "Risk Level",
							"required": true,
							"options":  []interface{}{"low", "medium", "high"},
						},
					},
					"sla_hours":       48,
					"escalation_role": "legal_director",
				},
				Transitions: []model.Transition{
					{Condition: "steps.legal_review.output.legal_approved == true", Target: "compliance_check"},
					{Target: "revision_required"},
				},
			},
			{
				ID:   "compliance_check",
				Type: model.StepTypeServiceTask,
				Name: "Compliance Check",
				Config: map[string]interface{}{
					"service": "compliance",
					"method":  "POST",
					"url":     "/api/v1/contracts/check",
					"body": map[string]interface{}{
						"contract_id":   "${variables.contract_id}",
						"contract_type": "${steps.upload.output.contract_type}",
						"value":         "${steps.upload.output.contract_value}",
					},
				},
				Transitions: []model.Transition{
					{Condition: "steps.compliance_check.output.compliant == true", Target: "approval"},
					{Target: "revision_required"},
				},
			},
			{
				ID:   "approval",
				Type: model.StepTypeHumanTask,
				Name: "Management Approval",
				Config: map[string]interface{}{
					"assignee_role": "contract_approver",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "approved",
							"type":     "boolean",
							"label":    "Approved?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "approval_notes",
							"type":     "textarea",
							"label":    "Approval Notes",
							"required": false,
						},
					},
					"sla_hours":       24,
					"escalation_role": "department_head",
				},
				Transitions: []model.Transition{
					{Condition: "steps.approval.output.approved == true", Target: "sign_off"},
					{Target: "revision_required"},
				},
			},
			{
				ID:   "sign_off",
				Type: model.StepTypeServiceTask,
				Name: "Contract Sign-Off",
				Config: map[string]interface{}{
					"service": "document-service",
					"method":  "POST",
					"url":     "/api/v1/contracts/${variables.contract_id}/sign",
					"body": map[string]interface{}{
						"status": "executed",
					},
				},
				Transitions: []model.Transition{
					{Target: "end"},
				},
			},
			{
				ID:   "revision_required",
				Type: model.StepTypeHumanTask,
				Name: "Revision Required",
				Config: map[string]interface{}{
					"assignee_role": "contract_submitter",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "revised_document_url",
							"type":     "text",
							"label":    "Revised Document URL",
							"required": true,
						},
						map[string]interface{}{
							"name":     "revision_notes",
							"type":     "textarea",
							"label":    "Revision Notes",
							"required": true,
						},
					},
				},
				Transitions: []model.Transition{
					{Target: "legal_review"},
				},
			},
			{
				ID:   "end",
				Type: model.StepTypeEnd,
				Name: "End",
			},
		},
	}

	defJSON, _ := json.Marshal(def)
	return &model.WorkflowTemplate{
		ID:             "tmpl-contract-review",
		Name:           "Contract Review",
		Description:    "Legal contract review workflow including document upload, legal review, compliance check, management approval, and sign-off.",
		Category:       "legal",
		DefinitionJSON: defJSON,
		Icon:           "file-text",
		CreatedAt:      now,
	}
}

// ---------- Template 3: Board Meeting ----------

func (s *TemplateService) boardMeetingTemplate(now time.Time) *model.WorkflowTemplate {
	def := templateDefinition{
		Name:        "Board Meeting",
		Description: "Board meeting governance workflow covering scheduling, agenda preparation, meeting execution, minutes creation, and action item tracking.",
		TriggerConfig: model.TriggerConfig{
			Type: model.TriggerTypeManual,
		},
		Variables: map[string]model.VariableDef{
			"meeting_id": {
				Type: "string",
			},
			"meeting_type": {
				Type:    "string",
				Default: "regular",
			},
			"meeting_date": {
				Type: "string",
			},
			"board_chair": {
				Type: "string",
			},
		},
		Steps: []model.StepDefinition{
			{
				ID:   "schedule",
				Type: model.StepTypeHumanTask,
				Name: "Schedule Meeting",
				Config: map[string]interface{}{
					"assignee_role": "board_secretary",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "meeting_date",
							"type":     "date",
							"label":    "Meeting Date",
							"required": true,
						},
						map[string]interface{}{
							"name":     "meeting_type",
							"type":     "select",
							"label":    "Meeting Type",
							"required": true,
							"options":  []interface{}{"regular", "special", "annual", "emergency"},
						},
						map[string]interface{}{
							"name":     "location",
							"type":     "text",
							"label":    "Location / Video Link",
							"required": true,
						},
						map[string]interface{}{
							"name":     "invitees",
							"type":     "textarea",
							"label":    "Invitees (one per line)",
							"required": true,
						},
					},
				},
				Transitions: []model.Transition{
					{Target: "prepare_agenda"},
				},
			},
			{
				ID:   "prepare_agenda",
				Type: model.StepTypeHumanTask,
				Name: "Prepare Agenda",
				Config: map[string]interface{}{
					"assignee_role": "board_secretary",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "agenda_items",
							"type":     "textarea",
							"label":    "Agenda Items",
							"required": true,
						},
						map[string]interface{}{
							"name":     "supporting_documents",
							"type":     "textarea",
							"label":    "Supporting Document URLs (one per line)",
							"required": false,
						},
					},
					"sla_hours": 72,
				},
				Transitions: []model.Transition{
					{Target: "send_notice"},
				},
			},
			{
				ID:   "send_notice",
				Type: model.StepTypeServiceTask,
				Name: "Send Meeting Notice",
				Config: map[string]interface{}{
					"service": "notification",
					"method":  "POST",
					"url":     "/api/v1/notifications/batch",
					"body": map[string]interface{}{
						"template":   "board_meeting_notice",
						"meeting_id": "${variables.meeting_id}",
						"date":       "${steps.schedule.output.meeting_date}",
						"agenda":     "${steps.prepare_agenda.output.agenda_items}",
					},
				},
				Transitions: []model.Transition{
					{Target: "conduct_meeting"},
				},
			},
			{
				ID:   "conduct_meeting",
				Type: model.StepTypeHumanTask,
				Name: "Conduct Meeting & Record Minutes",
				Config: map[string]interface{}{
					"assignee_role": "board_secretary",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "minutes",
							"type":     "textarea",
							"label":    "Meeting Minutes",
							"required": true,
						},
						map[string]interface{}{
							"name":     "attendees",
							"type":     "textarea",
							"label":    "Attendees (one per line)",
							"required": true,
						},
						map[string]interface{}{
							"name":     "resolutions",
							"type":     "textarea",
							"label":    "Resolutions Passed",
							"required": false,
						},
						map[string]interface{}{
							"name":     "quorum_met",
							"type":     "boolean",
							"label":    "Was Quorum Met?",
							"required": true,
						},
					},
				},
				Transitions: []model.Transition{
					{Target: "approve_minutes"},
				},
			},
			{
				ID:   "approve_minutes",
				Type: model.StepTypeHumanTask,
				Name: "Approve Minutes",
				Config: map[string]interface{}{
					"assignee_role": "board_chair",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "minutes_approved",
							"type":     "boolean",
							"label":    "Minutes Approved?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "corrections",
							"type":     "textarea",
							"label":    "Corrections (if any)",
							"required": false,
						},
					},
					"sla_hours": 48,
				},
				Transitions: []model.Transition{
					{Condition: "steps.approve_minutes.output.minutes_approved == true", Target: "track_actions"},
					{Target: "conduct_meeting"},
				},
			},
			{
				ID:   "track_actions",
				Type: model.StepTypeHumanTask,
				Name: "Track Action Items",
				Config: map[string]interface{}{
					"assignee_role": "board_secretary",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "action_items",
							"type":     "textarea",
							"label":    "Action Items (assignee: description, one per line)",
							"required": false,
						},
						map[string]interface{}{
							"name":     "next_meeting_date",
							"type":     "date",
							"label":    "Next Meeting Date",
							"required": false,
						},
						map[string]interface{}{
							"name":     "all_actions_assigned",
							"type":     "boolean",
							"label":    "All Action Items Assigned?",
							"required": true,
						},
					},
				},
				Transitions: []model.Transition{
					{Target: "end"},
				},
			},
			{
				ID:   "end",
				Type: model.StepTypeEnd,
				Name: "End",
			},
		},
	}

	defJSON, _ := json.Marshal(def)
	return &model.WorkflowTemplate{
		ID:             "tmpl-board-meeting",
		Name:           "Board Meeting",
		Description:    "Board meeting governance workflow: scheduling, agenda preparation, meeting conduct, minutes approval, and action item tracking.",
		Category:       "governance",
		DefinitionJSON: defJSON,
		Icon:           "users",
		CreatedAt:      now,
	}
}

// ---------- Template 4: Data Access Request ----------

func (s *TemplateService) dataAccessRequestTemplate(now time.Time) *model.WorkflowTemplate {
	def := templateDefinition{
		Name:        "Data Access Request",
		Description: "Data access governance workflow with request submission, data steward review, security review, approval, and automated provisioning.",
		TriggerConfig: model.TriggerConfig{
			Type: model.TriggerTypeManual,
		},
		Variables: map[string]model.VariableDef{
			"request_id": {
				Type: "string",
			},
			"requester_id": {
				Type: "string",
			},
			"dataset_name": {
				Type: "string",
			},
			"access_level": {
				Type:    "string",
				Default: "read",
			},
			"business_justification": {
				Type: "string",
			},
			"data_classification": {
				Type:    "string",
				Default: "internal",
			},
		},
		Steps: []model.StepDefinition{
			{
				ID:   "submit_request",
				Type: model.StepTypeHumanTask,
				Name: "Submit Data Access Request",
				Config: map[string]interface{}{
					"assignee_role": "data_requester",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "dataset_name",
							"type":     "text",
							"label":    "Dataset Name",
							"required": true,
						},
						map[string]interface{}{
							"name":     "access_level",
							"type":     "select",
							"label":    "Access Level",
							"required": true,
							"options":  []interface{}{"read", "write", "admin"},
						},
						map[string]interface{}{
							"name":     "business_justification",
							"type":     "textarea",
							"label":    "Business Justification",
							"required": true,
						},
						map[string]interface{}{
							"name":     "duration_days",
							"type":     "number",
							"label":    "Access Duration (days)",
							"required": true,
						},
					},
				},
				Transitions: []model.Transition{
					{Target: "steward_review"},
				},
			},
			{
				ID:   "steward_review",
				Type: model.StepTypeHumanTask,
				Name: "Data Steward Review",
				Config: map[string]interface{}{
					"assignee_role": "data_steward",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "steward_approved",
							"type":     "boolean",
							"label":    "Approved by Data Steward?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "data_classification",
							"type":     "select",
							"label":    "Data Classification",
							"required": true,
							"options":  []interface{}{"public", "internal", "confidential", "restricted"},
						},
						map[string]interface{}{
							"name":     "steward_notes",
							"type":     "textarea",
							"label":    "Review Notes",
							"required": false,
						},
					},
					"sla_hours":       24,
					"escalation_role": "data_governance_lead",
				},
				Transitions: []model.Transition{
					{Condition: "steps.steward_review.output.steward_approved == true", Target: "security_review"},
					{Target: "request_denied"},
				},
			},
			{
				ID:   "security_review",
				Type: model.StepTypeServiceTask,
				Name: "Security Review",
				Config: map[string]interface{}{
					"service": "security",
					"method":  "POST",
					"url":     "/api/v1/access-reviews",
					"body": map[string]interface{}{
						"requester_id":        "${variables.requester_id}",
						"dataset":             "${steps.submit_request.output.dataset_name}",
						"access_level":        "${steps.submit_request.output.access_level}",
						"data_classification": "${steps.steward_review.output.data_classification}",
					},
				},
				Transitions: []model.Transition{
					{Condition: "steps.security_review.output.security_cleared == true", Target: "final_approval"},
					{Target: "request_denied"},
				},
			},
			{
				ID:   "final_approval",
				Type: model.StepTypeHumanTask,
				Name: "Final Approval",
				Config: map[string]interface{}{
					"assignee_role": "data_governance_lead",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "final_approved",
							"type":     "boolean",
							"label":    "Final Approval Granted?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "conditions",
							"type":     "textarea",
							"label":    "Access Conditions / Restrictions",
							"required": false,
						},
					},
					"sla_hours": 24,
				},
				Transitions: []model.Transition{
					{Condition: "steps.final_approval.output.final_approved == true", Target: "provision_access"},
					{Target: "request_denied"},
				},
			},
			{
				ID:   "provision_access",
				Type: model.StepTypeServiceTask,
				Name: "Provision Access",
				Config: map[string]interface{}{
					"service": "iam",
					"method":  "POST",
					"url":     "/api/v1/access-grants",
					"body": map[string]interface{}{
						"user_id":      "${variables.requester_id}",
						"dataset":      "${steps.submit_request.output.dataset_name}",
						"access_level": "${steps.submit_request.output.access_level}",
						"duration":     "${steps.submit_request.output.duration_days}",
					},
					"max_retries": 3,
				},
				Transitions: []model.Transition{
					{Target: "end"},
				},
			},
			{
				ID:   "request_denied",
				Type: model.StepTypeServiceTask,
				Name: "Notify Request Denied",
				Config: map[string]interface{}{
					"service": "notification",
					"method":  "POST",
					"url":     "/api/v1/notifications",
					"body": map[string]interface{}{
						"template":  "data_access_denied",
						"user_id":   "${variables.requester_id}",
						"dataset":   "${steps.submit_request.output.dataset_name}",
					},
				},
				Transitions: []model.Transition{
					{Target: "end"},
				},
			},
			{
				ID:   "end",
				Type: model.StepTypeEnd,
				Name: "End",
			},
		},
	}

	defJSON, _ := json.Marshal(def)
	return &model.WorkflowTemplate{
		ID:             "tmpl-data-access-request",
		Name:           "Data Access Request",
		Description:    "Data access governance workflow with steward review, security review, approval, and automated provisioning.",
		Category:       "data",
		DefinitionJSON: defJSON,
		Icon:           "database",
		CreatedAt:      now,
	}
}

// ---------- Template 5: Change Request ----------

func (s *TemplateService) changeRequestTemplate(now time.Time) *model.WorkflowTemplate {
	def := templateDefinition{
		Name:        "Change Request",
		Description: "Enterprise change management workflow covering submission, impact assessment, CAB review, approval, implementation, and verification.",
		TriggerConfig: model.TriggerConfig{
			Type: model.TriggerTypeManual,
		},
		Variables: map[string]model.VariableDef{
			"change_id": {
				Type: "string",
			},
			"change_type": {
				Type:    "string",
				Default: "standard",
			},
			"priority": {
				Type:    "string",
				Default: "medium",
			},
			"submitter_id": {
				Type: "string",
			},
			"affected_systems": {
				Type:    "array",
				Default: []interface{}{},
			},
		},
		Steps: []model.StepDefinition{
			{
				ID:   "submit",
				Type: model.StepTypeHumanTask,
				Name: "Submit Change Request",
				Config: map[string]interface{}{
					"assignee_role": "change_submitter",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "change_title",
							"type":     "text",
							"label":    "Change Title",
							"required": true,
						},
						map[string]interface{}{
							"name":     "change_description",
							"type":     "textarea",
							"label":    "Change Description",
							"required": true,
						},
						map[string]interface{}{
							"name":     "change_type",
							"type":     "select",
							"label":    "Change Type",
							"required": true,
							"options":  []interface{}{"standard", "normal", "emergency"},
						},
						map[string]interface{}{
							"name":     "priority",
							"type":     "select",
							"label":    "Priority",
							"required": true,
							"options":  []interface{}{"low", "medium", "high", "critical"},
						},
						map[string]interface{}{
							"name":     "affected_systems",
							"type":     "textarea",
							"label":    "Affected Systems (one per line)",
							"required": true,
						},
						map[string]interface{}{
							"name":     "implementation_plan",
							"type":     "textarea",
							"label":    "Implementation Plan",
							"required": true,
						},
						map[string]interface{}{
							"name":     "rollback_plan",
							"type":     "textarea",
							"label":    "Rollback Plan",
							"required": true,
						},
					},
				},
				Transitions: []model.Transition{
					{Target: "impact_assessment"},
				},
			},
			{
				ID:   "impact_assessment",
				Type: model.StepTypeServiceTask,
				Name: "Impact Assessment",
				Config: map[string]interface{}{
					"service": "cmdb",
					"method":  "POST",
					"url":     "/api/v1/impact-analysis",
					"body": map[string]interface{}{
						"change_id":        "${variables.change_id}",
						"affected_systems": "${steps.submit.output.affected_systems}",
						"change_type":      "${steps.submit.output.change_type}",
					},
				},
				Transitions: []model.Transition{
					{Target: "cab_review"},
				},
			},
			{
				ID:   "cab_review",
				Type: model.StepTypeHumanTask,
				Name: "CAB Review",
				Config: map[string]interface{}{
					"assignee_role": "change_advisory_board",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "cab_approved",
							"type":     "boolean",
							"label":    "CAB Approved?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "risk_rating",
							"type":     "select",
							"label":    "Risk Rating",
							"required": true,
							"options":  []interface{}{"low", "medium", "high", "very_high"},
						},
						map[string]interface{}{
							"name":     "cab_comments",
							"type":     "textarea",
							"label":    "CAB Comments",
							"required": false,
						},
						map[string]interface{}{
							"name":     "scheduled_window",
							"type":     "text",
							"label":    "Approved Implementation Window",
							"required": false,
						},
					},
					"sla_hours":       24,
					"escalation_role": "change_manager",
				},
				Transitions: []model.Transition{
					{Condition: "steps.cab_review.output.cab_approved == true", Target: "change_approval"},
					{Target: "change_rejected"},
				},
			},
			{
				ID:   "change_approval",
				Type: model.StepTypeHumanTask,
				Name: "Change Manager Approval",
				Config: map[string]interface{}{
					"assignee_role": "change_manager",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "manager_approved",
							"type":     "boolean",
							"label":    "Approved for Implementation?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "approval_notes",
							"type":     "textarea",
							"label":    "Approval Notes",
							"required": false,
						},
					},
					"sla_hours": 8,
				},
				Transitions: []model.Transition{
					{Condition: "steps.change_approval.output.manager_approved == true", Target: "implementation"},
					{Target: "change_rejected"},
				},
			},
			{
				ID:   "implementation",
				Type: model.StepTypeHumanTask,
				Name: "Implementation",
				Config: map[string]interface{}{
					"assignee_role": "change_implementer",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "implementation_notes",
							"type":     "textarea",
							"label":    "Implementation Notes",
							"required": true,
						},
						map[string]interface{}{
							"name":     "implementation_successful",
							"type":     "boolean",
							"label":    "Implementation Successful?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "rollback_required",
							"type":     "boolean",
							"label":    "Was Rollback Required?",
							"required": true,
						},
					},
				},
				Transitions: []model.Transition{
					{Condition: "steps.implementation.output.implementation_successful == true", Target: "verification"},
					{Target: "change_failed"},
				},
			},
			{
				ID:   "verification",
				Type: model.StepTypeHumanTask,
				Name: "Post-Implementation Verification",
				Config: map[string]interface{}{
					"assignee_role": "change_verifier",
					"form_fields": []interface{}{
						map[string]interface{}{
							"name":     "verification_passed",
							"type":     "boolean",
							"label":    "Verification Passed?",
							"required": true,
						},
						map[string]interface{}{
							"name":     "verification_notes",
							"type":     "textarea",
							"label":    "Verification Notes",
							"required": true,
						},
						map[string]interface{}{
							"name":     "monitoring_period_days",
							"type":     "number",
							"label":    "Monitoring Period (days)",
							"required": true,
							"default":  float64(7),
						},
					},
					"sla_hours": 24,
				},
				Transitions: []model.Transition{
					{Condition: "steps.verification.output.verification_passed == true", Target: "close_success"},
					{Target: "change_failed"},
				},
			},
			{
				ID:   "close_success",
				Type: model.StepTypeServiceTask,
				Name: "Close Change - Successful",
				Config: map[string]interface{}{
					"service": "cmdb",
					"method":  "PATCH",
					"url":     "/api/v1/changes/${variables.change_id}",
					"body": map[string]interface{}{
						"status":     "closed_successful",
						"resolution": "Change implemented and verified successfully",
					},
				},
				Transitions: []model.Transition{
					{Target: "end"},
				},
			},
			{
				ID:   "change_rejected",
				Type: model.StepTypeServiceTask,
				Name: "Close Change - Rejected",
				Config: map[string]interface{}{
					"service": "cmdb",
					"method":  "PATCH",
					"url":     "/api/v1/changes/${variables.change_id}",
					"body": map[string]interface{}{
						"status": "rejected",
					},
				},
				Transitions: []model.Transition{
					{Target: "end"},
				},
			},
			{
				ID:   "change_failed",
				Type: model.StepTypeServiceTask,
				Name: "Close Change - Failed",
				Config: map[string]interface{}{
					"service": "cmdb",
					"method":  "PATCH",
					"url":     "/api/v1/changes/${variables.change_id}",
					"body": map[string]interface{}{
						"status": "failed",
					},
				},
				Transitions: []model.Transition{
					{Target: "end"},
				},
			},
			{
				ID:   "end",
				Type: model.StepTypeEnd,
				Name: "End",
			},
		},
	}

	defJSON, _ := json.Marshal(def)
	return &model.WorkflowTemplate{
		ID:             "tmpl-change-request",
		Name:           "Change Request",
		Description:    "Enterprise change management workflow with submission, impact assessment, CAB review, approval, implementation, and post-change verification.",
		Category:       "enterprise",
		DefinitionJSON: defJSON,
		Icon:           "git-pull-request",
		CreatedAt:      now,
	}
}
