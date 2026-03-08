package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/detection"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

type ruleEvaluator interface {
	Compile(json.RawMessage) (interface{}, error)
	Evaluate(interface{}, []model.SecurityEvent) []model.RuleMatch
	Type() string
}

// RuleService manages detection rules and templates.
type RuleService struct {
	ruleRepo   *repository.RuleRepository
	alertSvc   *AlertService
	producer   *events.Producer
	logger     zerolog.Logger
	evaluators map[model.DetectionRuleType]ruleEvaluator
}

// NewRuleService creates a new RuleService.
func NewRuleService(
	ruleRepo *repository.RuleRepository,
	alertSvc *AlertService,
	store *detection.BaselineStore,
	producer *events.Producer,
	logger zerolog.Logger,
) *RuleService {
	return &RuleService{
		ruleRepo: ruleRepo,
		alertSvc: alertSvc,
		producer: producer,
		logger:   logger,
		evaluators: map[model.DetectionRuleType]ruleEvaluator{
			model.RuleTypeSigma:       &detection.SigmaEvaluator{},
			model.RuleTypeThreshold:   &detection.ThresholdEvaluator{},
			model.RuleTypeCorrelation: &detection.CorrelationEvaluator{},
			model.RuleTypeAnomaly:     detection.NewAnomalyEvaluator(store),
		},
	}
}

// EnsureTemplates syncs the built-in template catalog into the database.
func (s *RuleService) EnsureTemplates(ctx context.Context) error {
	for _, template := range builtinRuleTemplates() {
		tpl := template.ToDetectionRule()
		if err := s.ruleRepo.EnsureTemplate(ctx, tpl); err != nil {
			return err
		}
	}
	return nil
}

// Stats returns total and active rule counts for a tenant.
func (s *RuleService) Stats(ctx context.Context, tenantID uuid.UUID, actor *Actor) (total, active int, err error) {
	return s.ruleRepo.Stats(ctx, tenantID)
}

// ListRules returns paginated tenant rules.
func (s *RuleService) ListRules(ctx context.Context, tenantID uuid.UUID, params *dto.RuleListParams, actor *Actor) (*dto.RuleListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	rules, total, err := s.ruleRepo.List(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.rule.listed", tenantID, actor, map[string]interface{}{
		"count": len(rules),
	})
	return &dto.RuleListResponse{
		Data: rules,
		Meta: dto.NewPaginationMeta(params.Page, params.PerPage, total),
	}, nil
}

// ListTemplates returns the system template rules.
func (s *RuleService) ListTemplates(ctx context.Context, tenantID uuid.UUID, actor *Actor) ([]*model.DetectionRule, error) {
	templates, err := s.ruleRepo.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.rule.templates_listed", tenantID, actor, map[string]interface{}{
		"count": len(templates),
	})
	return templates, nil
}

// GetRule returns a single rule.
func (s *RuleService) GetRule(ctx context.Context, tenantID, ruleID uuid.UUID, actor *Actor) (*model.DetectionRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, tenantID, ruleID)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.rule.viewed", tenantID, actor, map[string]interface{}{
		"id": ruleID.String(),
	})
	return rule, nil
}

// CreateRule validates and creates a detection rule.
func (s *RuleService) CreateRule(ctx context.Context, tenantID, userID uuid.UUID, actor *Actor, req *dto.CreateRuleRequest) (*model.DetectionRule, error) {
	rule, err := s.buildRuleFromCreateRequest(tenantID, req)
	if err != nil {
		return nil, err
	}
	created, err := s.ruleRepo.Create(ctx, tenantID, userID, rule)
	if err != nil {
		return nil, err
	}
	_ = publishEvent(ctx, s.producer, events.Topics.RuleEvents, "cyber.rule.created", tenantID, actor, map[string]interface{}{
		"id":       created.ID.String(),
		"name":     created.Name,
		"type":     created.RuleType,
		"severity": created.Severity,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.rule.created", tenantID, actor, created)
	return created, nil
}

// UpdateRule validates and updates a detection rule.
func (s *RuleService) UpdateRule(ctx context.Context, tenantID, ruleID uuid.UUID, actor *Actor, req *dto.UpdateRuleRequest) (*model.DetectionRule, error) {
	existing, err := s.ruleRepo.GetByID(ctx, tenantID, ruleID)
	if err != nil {
		return nil, err
	}
	updated := *existing
	if req.Name != nil {
		updated.Name = *req.Name
	}
	if req.Description != nil {
		updated.Description = *req.Description
	}
	if req.Severity != nil {
		if !req.Severity.IsValid() {
			return nil, repository.ErrInvalidInput
		}
		updated.Severity = *req.Severity
	}
	if req.Enabled != nil {
		updated.Enabled = *req.Enabled
	}
	if len(req.RuleContent) > 0 {
		updated.RuleContent = req.RuleContent
	}
	if req.MITRETacticIDs != nil {
		updated.MITRETacticIDs = *req.MITRETacticIDs
	}
	if req.MITRETechniqueIDs != nil {
		updated.MITRETechniqueIDs = *req.MITRETechniqueIDs
	}
	if req.BaseConfidence != nil {
		updated.BaseConfidence = *req.BaseConfidence
	}
	if req.Tags != nil {
		updated.Tags = *req.Tags
	}
	if err := s.validateRule(&updated); err != nil {
		return nil, err
	}
	rule, err := s.ruleRepo.Update(ctx, tenantID, ruleID, &updated)
	if err != nil {
		return nil, err
	}
	_ = publishEvent(ctx, s.producer, events.Topics.RuleEvents, "cyber.rule.updated", tenantID, actor, map[string]interface{}{
		"id":             ruleID.String(),
		"changed_fields": changedRuleFields(existing, req),
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.rule.updated", tenantID, actor, map[string]interface{}{
		"id":     ruleID.String(),
		"before": existing,
		"after":  rule,
	})
	return rule, nil
}

// DeleteRule soft-deletes a rule.
func (s *RuleService) DeleteRule(ctx context.Context, tenantID, ruleID uuid.UUID, actor *Actor) error {
	if err := s.ruleRepo.SoftDelete(ctx, tenantID, ruleID); err != nil {
		return err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.rule.deleted", tenantID, actor, map[string]interface{}{
		"id": ruleID.String(),
	})
	return nil
}

// Toggle enables or disables a rule.
func (s *RuleService) Toggle(ctx context.Context, tenantID, ruleID uuid.UUID, actor *Actor, enabled bool) (*model.DetectionRule, error) {
	rule, err := s.ruleRepo.Toggle(ctx, tenantID, ruleID, enabled)
	if err != nil {
		return nil, err
	}
	_ = publishEvent(ctx, s.producer, events.Topics.RuleEvents, "cyber.rule.toggled", tenantID, actor, map[string]interface{}{
		"id":      ruleID.String(),
		"enabled": enabled,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.rule.toggled", tenantID, actor, map[string]interface{}{
		"id":      ruleID.String(),
		"enabled": enabled,
	})
	return rule, nil
}

// TestRule dry-runs a rule against historical security events.
func (s *RuleService) TestRule(ctx context.Context, tenantID, ruleID uuid.UUID, actor *Actor, req *dto.RuleTestRequest) (*dto.RuleTestResponse, error) {
	rule, err := s.ruleRepo.GetByID(ctx, tenantID, ruleID)
	if err != nil {
		return nil, err
	}
	evaluator, ok := s.evaluators[rule.RuleType]
	if !ok {
		return nil, repository.ErrInvalidInput
	}
	compiled, err := evaluator.Compile(rule.RuleContent)
	if err != nil {
		return nil, err
	}
	events, err := s.ruleRepo.ListSecurityEvents(ctx, tenantID, req.DateFrom, req.DateTo, req.Limit)
	if err != nil {
		return nil, err
	}
	matches := evaluator.Evaluate(compiled, events)
	for i := range matches {
		matches[i].RuleID = rule.ID
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.rule.tested", tenantID, actor, map[string]interface{}{
		"id":          ruleID.String(),
		"match_count": len(matches),
		"event_count": len(events),
	})
	return &dto.RuleTestResponse{Matches: matches, Count: len(matches)}, nil
}

// SubmitFeedback records true-positive or false-positive feedback against a rule.
func (s *RuleService) SubmitFeedback(ctx context.Context, tenantID uuid.UUID, actor *Actor, req *dto.RuleFeedbackRequest) (*model.DetectionRule, error) {
	alert, err := s.alertSvc.GetAlert(ctx, tenantID, req.AlertID, actor)
	if err != nil {
		return nil, err
	}
	if alert.RuleID == nil {
		return nil, repository.ErrInvalidInput
	}

	status := model.AlertStatusInvestigating
	if req.Feedback == "false_positive" {
		status = model.AlertStatusFalsePositive
	}
	_, err = s.alertSvc.UpdateStatus(ctx, tenantID, req.AlertID, actor, &dto.AlertStatusUpdateRequest{
		Status: status,
		Notes:  stringPointer(fmt.Sprintf("Analyst feedback recorded as %s", req.Feedback)),
	})
	if err != nil {
		return nil, err
	}

	rule, err := s.ruleRepo.UpdateFeedbackCounters(ctx, tenantID, *alert.RuleID, req.Feedback)
	if err != nil {
		return nil, err
	}

	totalFeedback := rule.TruePositiveCount + rule.FalsePositiveCount
	fpRate := rule.FPRate()
	if totalFeedback >= 100 && fpRate > 0.50 {
		rule, err = s.ruleRepo.Toggle(ctx, tenantID, rule.ID, false)
		if err != nil {
			return nil, err
		}
		_, _ = s.alertSvc.AddComment(ctx, tenantID, req.AlertID, actor, &dto.AlertCommentRequest{
			Content: fmt.Sprintf("Rule auto-disabled due to %.1f%% false positive rate.", fpRate*100),
		})
		_ = publishEvent(ctx, s.producer, events.Topics.RuleEvents, "cyber.rule.auto_disabled", tenantID, actor, map[string]interface{}{
			"id":      rule.ID.String(),
			"reason":  "high false positive rate",
			"fp_rate": fpRate,
		})
	}

	_ = publishAuditEvent(ctx, s.producer, "cyber.rule.feedback_recorded", tenantID, actor, map[string]interface{}{
		"id":       rule.ID.String(),
		"feedback": req.Feedback,
	})
	return rule, nil
}

// Coverage returns the ATT&CK coverage map for the tenant's enabled rules.
func (s *RuleService) Coverage(ctx context.Context, tenantID uuid.UUID, actor *Actor) ([]mitre.TechniqueCoverage, error) {
	rules, err := s.ruleRepo.ListEnabledByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	coverage := mitre.BuildCoverage(rules)
	_ = publishAuditEvent(ctx, s.producer, "cyber.mitre.coverage_viewed", tenantID, actor, map[string]interface{}{
		"count": len(coverage),
	})
	return coverage, nil
}

func (s *RuleService) buildRuleFromCreateRequest(tenantID uuid.UUID, req *dto.CreateRuleRequest) (*model.DetectionRule, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	baseConfidence := 0.70
	if req.BaseConfidence != nil {
		baseConfidence = *req.BaseConfidence
	}
	rule := &model.DetectionRule{
		TenantID:          &tenantID,
		Name:              req.Name,
		Description:       req.Description,
		RuleType:          req.RuleType,
		Severity:          req.Severity,
		Enabled:           enabled,
		RuleContent:       req.RuleContent,
		MITRETacticIDs:    req.MITRETacticIDs,
		MITRETechniqueIDs: req.MITRETechniqueIDs,
		BaseConfidence:    baseConfidence,
		Tags:              req.Tags,
	}
	if err := s.validateRule(rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *RuleService) validateRule(rule *model.DetectionRule) error {
	if !rule.RuleType.IsValid() {
		return repository.ErrInvalidInput
	}
	if !rule.Severity.IsValid() {
		return repository.ErrInvalidInput
	}
	evaluator, ok := s.evaluators[rule.RuleType]
	if !ok {
		return repository.ErrInvalidInput
	}
	_, err := evaluator.Compile(rule.RuleContent)
	return err
}

func changedRuleFields(existing *model.DetectionRule, req *dto.UpdateRuleRequest) []string {
	fields := make([]string, 0, 8)
	if req.Name != nil && *req.Name != existing.Name {
		fields = append(fields, "name")
	}
	if req.Description != nil && *req.Description != existing.Description {
		fields = append(fields, "description")
	}
	if req.Severity != nil && *req.Severity != existing.Severity {
		fields = append(fields, "severity")
	}
	if req.Enabled != nil && *req.Enabled != existing.Enabled {
		fields = append(fields, "enabled")
	}
	if len(req.RuleContent) > 0 {
		fields = append(fields, "rule_content")
	}
	if req.MITRETacticIDs != nil {
		fields = append(fields, "mitre_tactic_ids")
	}
	if req.MITRETechniqueIDs != nil {
		fields = append(fields, "mitre_technique_ids")
	}
	if req.BaseConfidence != nil && *req.BaseConfidence != existing.BaseConfidence {
		fields = append(fields, "base_confidence")
	}
	if req.Tags != nil {
		fields = append(fields, "tags")
	}
	return fields
}

func stringPointer(value string) *string {
	return &value
}
