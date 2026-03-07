package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/lex/dto"
	"github.com/clario360/platform/internal/lex/metrics"
	"github.com/clario360/platform/internal/lex/model"
	"github.com/clario360/platform/internal/lex/repository"
)

type ComplianceService struct {
	db         *pgxpool.Pool
	contracts  *repository.ContractRepository
	clauses    *repository.ClauseRepository
	documents  *repository.DocumentRepository
	rules      *repository.ComplianceRepository
	alerts     *repository.AlertRepository
	publisher  Publisher
	metrics    *metrics.Metrics
	topic      string
	logger     zerolog.Logger
	now        func() time.Time
}

func NewComplianceService(db *pgxpool.Pool, contracts *repository.ContractRepository, clauses *repository.ClauseRepository, documents *repository.DocumentRepository, rules *repository.ComplianceRepository, alerts *repository.AlertRepository, publisher Publisher, appMetrics *metrics.Metrics, topic string, logger zerolog.Logger) *ComplianceService {
	return &ComplianceService{
		db:         db,
		contracts:  contracts,
		clauses:    clauses,
		documents:  documents,
		rules:      rules,
		alerts:     alerts,
		publisher:  publisherOrNoop(publisher),
		metrics:    appMetrics,
		topic:      topic,
		logger:     logger.With().Str("service", "lex-compliance").Logger(),
		now:        time.Now,
	}
}

func (s *ComplianceService) ListRules(ctx context.Context, tenantID uuid.UUID) ([]model.ComplianceRule, error) {
	return s.rules.ListRules(ctx, tenantID)
}

func (s *ComplianceService) CreateRule(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateComplianceRuleRequest) (*model.ComplianceRule, error) {
	req.Normalize()
	if strings.TrimSpace(req.Name) == "" {
		return nil, validationError("rule name is required", map[string]string{"name": "required"})
	}
	rule := &model.ComplianceRule{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          req.Name,
		Description:   req.Description,
		RuleType:      req.RuleType,
		Severity:      req.Severity,
		Config:        req.Config,
		ContractTypes: req.ContractTypes,
		Enabled:       req.Enabled,
		CreatedBy:     userID,
	}
	if err := s.rules.CreateRule(ctx, s.db, rule); err != nil {
		return nil, internalError("create compliance rule", err)
	}
	return rule, nil
}

func (s *ComplianceService) UpdateRule(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateComplianceRuleRequest) (*model.ComplianceRule, error) {
	rule, err := s.rules.GetRule(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("rule not found")
		}
		return nil, internalError("load rule", err)
	}
	if req.Name != nil {
		rule.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		rule.Description = strings.TrimSpace(*req.Description)
	}
	if req.RuleType != nil {
		rule.RuleType = *req.RuleType
	}
	if req.Severity != nil {
		rule.Severity = *req.Severity
	}
	if req.Config != nil {
		rule.Config = req.Config
	}
	if req.ContractTypes != nil {
		rule.ContractTypes = req.ContractTypes
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if err := s.rules.UpdateRule(ctx, s.db, rule); err != nil {
		return nil, internalError("update rule", err)
	}
	return rule, nil
}

func (s *ComplianceService) DeleteRule(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.rules.SoftDeleteRule(ctx, tenantID, id); err != nil {
		if err == pgx.ErrNoRows {
			return notFoundError("rule not found")
		}
		return internalError("delete rule", err)
	}
	return nil
}

func (s *ComplianceService) RunChecks(ctx context.Context, tenantID uuid.UUID, contractIDs []uuid.UUID) (*model.ComplianceRunResult, error) {
	rules, err := s.rules.ListEnabledRules(ctx, tenantID)
	if err != nil {
		return nil, internalError("list enabled rules", err)
	}
	contracts, err := s.loadContracts(ctx, tenantID, contractIDs)
	if err != nil {
		return nil, err
	}
	createdAlerts := make([]model.ComplianceAlert, 0)
	for _, contract := range contracts {
		analysis, _ := s.contracts.GetLatestAnalysis(ctx, tenantID, contract.ID)
		clauses, _ := s.clauses.ListByContract(ctx, tenantID, contract.ID)
		for _, rule := range rules {
			if !ruleAppliesToContract(rule, contract.Type) {
				continue
			}
			alert, ok := s.evaluateRule(rule, &contract, analysis, clauses)
			if !ok {
				continue
			}
			created, err := s.alerts.CreateOrSkipDedup(ctx, s.db, alert)
			if err != nil {
				return nil, internalError("create compliance alert", err)
			}
			if created {
				createdAlerts = append(createdAlerts, *alert)
				writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.compliance.alert_created", tenantID, nil, map[string]any{
					"id":         alert.ID,
					"contract_id": alert.ContractID,
					"severity":   alert.Severity,
					"title":      alert.Title,
				}, s.logger)
			}
		}
	}
	score, err := s.GetScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := &model.ComplianceRunResult{
		TenantID:      tenantID,
		Score:         score.Score,
		AlertsCreated: len(createdAlerts),
		Alerts:        createdAlerts,
		CalculatedAt:  s.now().UTC(),
	}
	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.compliance.checked", tenantID, nil, map[string]any{
		"tenant_id":      tenantID,
		"score":          result.Score,
		"alerts_created": result.AlertsCreated,
	}, s.logger)
	return result, nil
}

func (s *ComplianceService) ListAlerts(ctx context.Context, tenantID uuid.UUID, status string, severity string, page, perPage int) ([]model.ComplianceAlert, int, error) {
	return s.alerts.List(ctx, tenantID, status, severity, page, perPage)
}

func (s *ComplianceService) GetAlert(ctx context.Context, tenantID, id uuid.UUID) (*model.ComplianceAlert, error) {
	alert, err := s.alerts.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("alert not found")
		}
		return nil, internalError("get alert", err)
	}
	return alert, nil
}

func (s *ComplianceService) UpdateAlertStatus(ctx context.Context, tenantID, id, userID uuid.UUID, req dto.UpdateAlertStatusRequest) (*model.ComplianceAlert, error) {
	req.Normalize()
	var resolvedAt *time.Time
	if req.Status == model.ComplianceAlertResolved || req.Status == model.ComplianceAlertDismissed {
		now := s.now().UTC()
		resolvedAt = &now
	}
	if err := s.alerts.UpdateStatus(ctx, s.db, tenantID, id, req.Status, &userID, resolvedAt, req.ResolutionNotes); err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("alert not found")
		}
		return nil, internalError("update alert status", err)
	}
	alert, err := s.alerts.Get(ctx, tenantID, id)
	if err != nil {
		return nil, internalError("reload alert", err)
	}
	if req.Status == model.ComplianceAlertResolved {
		writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.compliance.alert_resolved", tenantID, &userID, map[string]any{
			"id":          alert.ID,
			"resolved_by": userID,
		}, s.logger)
	}
	return alert, nil
}

func (s *ComplianceService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*model.ComplianceDashboard, error) {
	alertsByStatus, err := s.alerts.CountByStatus(ctx, tenantID)
	if err != nil {
		return nil, internalError("count alerts by status", err)
	}
	alertsBySeverity, err := s.alerts.CountBySeverity(ctx, tenantID)
	if err != nil {
		return nil, internalError("count alerts by severity", err)
	}
	rules, err := s.rules.ListRules(ctx, tenantID)
	if err != nil {
		return nil, internalError("list rules", err)
	}
	contracts, total, err := s.contracts.List(ctx, tenantID, model.ContractListFilters{Page: 1, PerPage: 200})
	if err != nil {
		return nil, internalError("list contracts", err)
	}
	_ = contracts
	score, err := s.GetScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	rulesByType := map[string]int{}
	for _, rule := range rules {
		rulesByType[string(rule.RuleType)]++
	}
	openAlerts := alertsByStatus[string(model.ComplianceAlertOpen)] +
		alertsByStatus[string(model.ComplianceAlertAcknowledged)] +
		alertsByStatus[string(model.ComplianceAlertInvestigating)]
	return &model.ComplianceDashboard{
		RulesByType:      rulesByType,
		AlertsByStatus:   alertsByStatus,
		AlertsBySeverity: alertsBySeverity,
		OpenAlerts:       openAlerts,
		ResolvedAlerts:   alertsByStatus[string(model.ComplianceAlertResolved)],
		ContractsInScope: total,
		ComplianceScore:  score.Score,
		CalculatedAt:     s.now().UTC(),
	}, nil
}

func (s *ComplianceService) GetScore(ctx context.Context, tenantID uuid.UUID) (*model.ComplianceScore, error) {
	openAlerts, resolvedAlerts, err := s.alerts.ScoreComponents(ctx, tenantID)
	if err != nil {
		return nil, internalError("load alert score components", err)
	}
	rules, err := s.rules.ListEnabledRules(ctx, tenantID)
	if err != nil {
		return nil, internalError("list enabled rules", err)
	}
	score := clampScore(100 - float64(openAlerts*10) + math.Min(float64(resolvedAlerts), 10) + math.Min(float64(len(rules))*2, 15))
	if s.metrics != nil {
		s.metrics.ComplianceScore.WithLabelValues(tenantID.String()).Set(score)
	}
	return &model.ComplianceScore{
		TenantID:       tenantID,
		Score:          score,
		OpenAlerts:     openAlerts,
		ResolvedAlerts: resolvedAlerts,
		RuleCoverage:   len(rules),
		CalculatedAt:   s.now().UTC(),
	}, nil
}

func (s *ComplianceService) CreateSystemAlert(ctx context.Context, alert *model.ComplianceAlert) (bool, error) {
	created, err := s.alerts.CreateOrSkipDedup(ctx, s.db, alert)
	if err != nil {
		return false, internalError("create system alert", err)
	}
	if created {
		writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.compliance.alert_created", alert.TenantID, nil, map[string]any{
			"id":          alert.ID,
			"contract_id": alert.ContractID,
			"severity":    alert.Severity,
			"title":       alert.Title,
		}, s.logger)
	}
	return created, nil
}

func (s *ComplianceService) HandleFileIntegrityEvent(ctx context.Context, tenantID uuid.UUID, fileID uuid.UUID, eventType string, description string) error {
	contracts, err := s.contracts.GetByFileID(ctx, fileID)
	if err != nil {
		return internalError("load contracts by file", err)
	}
	documents, err := s.documents.GetByFileID(ctx, fileID)
	if err != nil {
		return internalError("load documents by file", err)
	}
	for _, contract := range contracts {
		dedup := fmt.Sprintf("%s:%s", eventType, contract.ID)
		title := fmt.Sprintf("Contract document integrity event for %s", contract.Title)
		alert := &model.ComplianceAlert{
			ID:          uuid.New(),
			TenantID:    contract.TenantID,
			ContractID:  &contract.ID,
			Title:       title,
			Description: description,
			Severity:    model.ComplianceSeverityHigh,
			Status:      model.ComplianceAlertOpen,
			DedupKey:    &dedup,
			Evidence: map[string]any{
				"file_id":    fileID,
				"event_type": eventType,
			},
		}
		if _, err := s.CreateSystemAlert(ctx, alert); err != nil {
			return err
		}
	}
	for _, document := range documents {
		dedup := fmt.Sprintf("%s:%s", eventType, document.ID)
		alert := &model.ComplianceAlert{
			ID:       uuid.New(),
			TenantID: document.TenantID,
			Title:    fmt.Sprintf("Legal document integrity event for %s", document.Title),
			Description: description,
			Severity: model.ComplianceSeverityMedium,
			Status:   model.ComplianceAlertOpen,
			DedupKey: &dedup,
			Evidence: map[string]any{
				"document_id": document.ID,
				"file_id":     fileID,
				"event_type":  eventType,
			},
		}
		if _, err := s.CreateSystemAlert(ctx, alert); err != nil {
			return err
		}
	}
	return nil
}

func (s *ComplianceService) loadContracts(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]model.Contract, error) {
	if len(ids) == 0 {
		page := 1
		all := make([]model.Contract, 0)
		for {
			items, total, err := s.contracts.List(ctx, tenantID, model.ContractListFilters{Page: page, PerPage: 200})
			if err != nil {
				return nil, internalError("list contracts", err)
			}
			all = append(all, items...)
			if len(all) >= total || len(items) == 0 {
				return all, nil
			}
			page++
		}
	}
	out := make([]model.Contract, 0, len(ids))
	for _, id := range ids {
		contract, err := s.contracts.Get(ctx, tenantID, id)
		if err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return nil, internalError("load contract", err)
		}
		out = append(out, *contract)
	}
	return out, nil
}

func ruleAppliesToContract(rule model.ComplianceRule, contractType model.ContractType) bool {
	if len(rule.ContractTypes) == 0 {
		return true
	}
	for _, allowed := range rule.ContractTypes {
		if allowed == string(contractType) {
			return true
		}
	}
	return false
}

func (s *ComplianceService) evaluateRule(rule model.ComplianceRule, contract *model.Contract, analysis *model.ContractRiskAnalysis, clauses []model.Clause) (*model.ComplianceAlert, bool) {
	now := s.now().UTC()
	var (
		shouldAlert bool
		title       string
		description string
		evidence    = map[string]any{"rule_type": rule.RuleType}
	)
	switch rule.RuleType {
	case model.ComplianceRuleExpiryWarning:
		days := intConfig(rule.Config, "days_before", 30)
		if contract.Status == model.ContractStatusActive && contract.ExpiryDate != nil && int(contract.ExpiryDate.Sub(now).Hours()/24) <= days {
			shouldAlert = true
			title = fmt.Sprintf("Contract %q expires within %d days", contract.Title, days)
			description = "Expiry warning rule triggered."
		}
	case model.ComplianceRuleMissingClause:
		if analysis != nil && len(analysis.MissingClauses) > 0 {
			shouldAlert = true
			title = fmt.Sprintf("Contract %q is missing standard clauses", contract.Title)
			description = fmt.Sprintf("Missing clauses: %v", analysis.MissingClauses)
			evidence["missing_clauses"] = analysis.MissingClauses
		}
	case model.ComplianceRuleRiskThreshold:
		threshold := floatConfig(rule.Config, "min_score", 70)
		requiredStatus := stringConfig(rule.Config, "required_status", string(model.ContractStatusLegalReview))
		if contract.RiskScore != nil && *contract.RiskScore > threshold && string(contract.Status) != requiredStatus {
			shouldAlert = true
			title = fmt.Sprintf("High-risk contract %q is not in required review status", contract.Title)
			description = fmt.Sprintf("Risk score %.2f exceeds %.2f but status is %s", *contract.RiskScore, threshold, contract.Status)
		}
	case model.ComplianceRuleReviewOverdue:
		days := intConfig(rule.Config, "overdue_days", 7)
		if (contract.Status == model.ContractStatusInternalReview || contract.Status == model.ContractStatusLegalReview) &&
			contract.StatusChangedAt != nil && contract.StatusChangedAt.Before(now.AddDate(0, 0, -days)) {
			shouldAlert = true
			title = fmt.Sprintf("Contract %q review is overdue", contract.Title)
			description = "Contract has remained in review beyond the allowed SLA."
		}
	case model.ComplianceRuleUnsignedContract:
		if contract.Status == model.ContractStatusPendingSignature || (contract.Status == model.ContractStatusActive && contract.SignedDate == nil) {
			shouldAlert = true
			title = fmt.Sprintf("Contract %q is unsigned", contract.Title)
			description = "Signature status does not satisfy policy."
		}
	case model.ComplianceRuleValueThreshold:
		minValue := floatConfig(rule.Config, "min_value", 1_000_000)
		if contract.TotalValue != nil && *contract.TotalValue >= minValue {
			shouldAlert = true
			title = fmt.Sprintf("Contract %q exceeds value threshold", contract.Title)
			description = fmt.Sprintf("Contract value %.2f exceeds %.2f", *contract.TotalValue, minValue)
		}
	case model.ComplianceRuleJurisdictionCheck:
		if analysis != nil && hasComplianceFlag(analysis, "foreign_governing_law") {
			shouldAlert = true
			title = fmt.Sprintf("Contract %q uses foreign governing law", contract.Title)
			description = "Jurisdiction check rule triggered."
		}
	case model.ComplianceRuleDataProtectionRequired:
		if strings.Contains(strings.ToLower(contract.DocumentText), "personal data") && !hasClause(clauses, model.ClauseTypeDataProtection) {
			shouldAlert = true
			title = fmt.Sprintf("Contract %q lacks data protection terms", contract.Title)
			description = "Personal-data language found without a data protection clause."
		}
	case model.ComplianceRuleCustom:
		shouldAlert = evaluateCustomRule(rule.Config, contract)
		if shouldAlert {
			title = fmt.Sprintf("Custom compliance rule triggered for %q", contract.Title)
			description = "Custom rule conditions matched."
		}
	}
	if !shouldAlert {
		return nil, false
	}
	dedup := fmt.Sprintf("rule:%s:contract:%s", rule.ID, contract.ID)
	return &model.ComplianceAlert{
		ID:          uuid.New(),
		TenantID:    contract.TenantID,
		RuleID:      &rule.ID,
		ContractID:  &contract.ID,
		Title:       title,
		Description: description,
		Severity:    rule.Severity,
		Status:      model.ComplianceAlertOpen,
		DedupKey:    &dedup,
		Evidence:    evidence,
	}, true
}

func hasClause(clauses []model.Clause, clauseType model.ClauseType) bool {
	for _, clause := range clauses {
		if clause.ClauseType == clauseType {
			return true
		}
	}
	return false
}

func hasComplianceFlag(analysis *model.ContractRiskAnalysis, code string) bool {
	if analysis == nil {
		return false
	}
	for _, flag := range analysis.ComplianceFlags {
		if flag.Code == code {
			return true
		}
	}
	return false
}

func intConfig(config map[string]any, key string, fallback int) int {
	if config == nil {
		return fallback
	}
	switch value := config[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case string:
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func floatConfig(config map[string]any, key string, fallback float64) float64 {
	if config == nil {
		return fallback
	}
	switch value := config[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case string:
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func stringConfig(config map[string]any, key string, fallback string) string {
	if config == nil {
		return fallback
	}
	if value, ok := config[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func evaluateCustomRule(config map[string]any, contract *model.Contract) bool {
	field := stringConfig(config, "field", "")
	operator := stringConfig(config, "operator", "eq")
	value := config["value"]
	switch field {
	case "risk_score":
		if contract.RiskScore == nil {
			return false
		}
		return compareNumeric(*contract.RiskScore, operator, value)
	case "total_value":
		if contract.TotalValue == nil {
			return false
		}
		return compareNumeric(*contract.TotalValue, operator, value)
	case "status":
		if wanted, ok := value.(string); ok {
			switch operator {
			case "neq":
				return string(contract.Status) != wanted
			default:
				return string(contract.Status) == wanted
			}
		}
	}
	return false
}

func compareNumeric(current float64, operator string, value any) bool {
	target := floatConfig(map[string]any{"v": value}, "v", 0)
	switch operator {
	case "gt":
		return current > target
	case "gte":
		return current >= target
	case "lt":
		return current < target
	case "lte":
		return current <= target
	case "neq":
		return current != target
	default:
		return current == target
	}
}
