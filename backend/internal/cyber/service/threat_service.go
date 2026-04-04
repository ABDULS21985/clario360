package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/indicator"
	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

// ThreatService manages threats and indicators.
type ThreatService struct {
	threatRepo      *repository.ThreatRepository
	indicatorRepo   *repository.IndicatorRepository
	alertRepo       *repository.AlertRepository
	enrichmentCache *EnrichmentCache
	producer        *events.Producer
	logger          zerolog.Logger
}

// NewThreatService creates a new ThreatService.
func NewThreatService(
	threatRepo *repository.ThreatRepository,
	indicatorRepo *repository.IndicatorRepository,
	alertRepo *repository.AlertRepository,
	enrichmentCache *EnrichmentCache,
	producer *events.Producer,
	logger zerolog.Logger,
) *ThreatService {
	return &ThreatService{
		threatRepo:      threatRepo,
		indicatorRepo:   indicatorRepo,
		alertRepo:       alertRepo,
		enrichmentCache: enrichmentCache,
		producer:        producer,
		logger:          logger,
	}
}

// ListThreats returns paginated threats.
func (s *ThreatService) ListThreats(ctx context.Context, tenantID uuid.UUID, params *dto.ThreatListParams, actor *Actor) (*dto.ThreatListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	items, total, err := s.threatRepo.List(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.listed", tenantID, actor, map[string]interface{}{
		"count": len(items),
	})
	return &dto.ThreatListResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(params.Page, params.PerPage, total),
	}, nil
}

// GetThreat returns a single threat with its indicators.
func (s *ThreatService) GetThreat(ctx context.Context, tenantID, threatID uuid.UUID, actor *Actor) (*model.Threat, error) {
	threat, err := s.threatRepo.GetByID(ctx, tenantID, threatID)
	if err != nil {
		return nil, err
	}
	indicators, err := s.indicatorRepo.ListByThreat(ctx, tenantID, threatID)
	if err != nil {
		return nil, err
	}
	threat.Indicators = indicators
	if threat.IndicatorCount == 0 {
		threat.IndicatorCount = len(indicators)
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.viewed", tenantID, actor, map[string]interface{}{
		"id": threatID.String(),
	})
	return threat, nil
}

// CreateThreat creates a threat and optional initial indicators.
func (s *ThreatService) CreateThreat(ctx context.Context, tenantID, userID uuid.UUID, actor *Actor, req *dto.CreateThreatRequest) (*model.Threat, error) {
	if err := validateThreatInput(req.Name, req.Type, req.Severity, req.MITRETacticIDs, req.MITRETechniqueIDs); err != nil {
		return nil, err
	}
	for _, ind := range req.Indicators {
		if !ind.Type.IsValid() || !ind.Severity.IsValid() || ind.Severity == model.SeverityInfo {
			return nil, repository.ErrInvalidInput
		}
	}

	threatModel := &model.Threat{
		TenantID:          tenantID,
		Name:              strings.TrimSpace(req.Name),
		Description:       strings.TrimSpace(req.Description),
		Type:              req.Type,
		Severity:          req.Severity,
		Status:            model.ThreatStatusActive,
		ThreatActor:       optionalStringPtr(req.ThreatActor),
		Campaign:          optionalStringPtr(req.Campaign),
		MITRETacticIDs:    normalizeStrings(req.MITRETacticIDs),
		MITRETechniqueIDs: normalizeStrings(req.MITRETechniqueIDs),
		Tags:              normalizeStrings(req.Tags),
		CreatedBy:         &userID,
	}

	created, err := s.threatRepo.Create(ctx, threatModel)
	if err != nil {
		return nil, err
	}

	for _, ind := range req.Indicators {
		indicatorModel := &model.ThreatIndicator{
			TenantID:    tenantID,
			ThreatID:    &created.ID,
			Type:        ind.Type,
			Value:       strings.TrimSpace(ind.Value),
			Description: strings.TrimSpace(ind.Description),
			Severity:    ind.Severity,
			Source:      coalesceSource(ind.Source),
			Confidence:  normalizeIndicatorConfidence(ind.Confidence),
			Active:      true,
			Tags:        normalizeStrings(ind.Tags),
			CreatedBy:   &userID,
		}
		if _, err := s.indicatorRepo.Create(ctx, indicatorModel); err != nil {
			return nil, err
		}
	}

	result, err := s.GetThreat(ctx, tenantID, created.ID, nil)
	if err != nil {
		return nil, err
	}
	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.threat.detected", tenantID, actor, map[string]interface{}{
		"id":              created.ID.String(),
		"name":            created.Name,
		"type":            created.Type,
		"severity":        created.Severity,
		"indicator_count": result.IndicatorCount,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.created", tenantID, actor, result)
	return result, nil
}

// UpdateThreat updates editable threat fields.
func (s *ThreatService) UpdateThreat(ctx context.Context, tenantID, threatID uuid.UUID, actor *Actor, req *dto.UpdateThreatRequest) (*model.Threat, error) {
	if err := validateThreatInput(req.Name, req.Type, req.Severity, req.MITRETacticIDs, req.MITRETechniqueIDs); err != nil {
		return nil, err
	}
	before, err := s.threatRepo.GetByID(ctx, tenantID, threatID)
	if err != nil {
		return nil, err
	}
	updated, err := s.threatRepo.Update(ctx, &model.Threat{
		ID:                threatID,
		TenantID:          tenantID,
		Name:              strings.TrimSpace(req.Name),
		Description:       strings.TrimSpace(req.Description),
		Type:              req.Type,
		Severity:          req.Severity,
		ThreatActor:       optionalStringPtr(req.ThreatActor),
		Campaign:          optionalStringPtr(req.Campaign),
		MITRETacticIDs:    normalizeStrings(req.MITRETacticIDs),
		MITRETechniqueIDs: normalizeStrings(req.MITRETechniqueIDs),
		Tags:              normalizeStrings(req.Tags),
		Metadata:          before.Metadata,
	})
	if err != nil {
		return nil, err
	}
	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.threat.updated", tenantID, actor, map[string]interface{}{
		"id":        updated.ID.String(),
		"name":      updated.Name,
		"severity":  updated.Severity,
		"old_name":  before.Name,
		"old_type":  before.Type,
		"new_type":  updated.Type,
		"old_level": before.Severity,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.updated", tenantID, actor, map[string]interface{}{
		"id":     updated.ID.String(),
		"before": before,
		"after":  updated,
	})
	return updated, nil
}

// DeleteThreat soft-deletes a threat.
func (s *ThreatService) DeleteThreat(ctx context.Context, tenantID, threatID uuid.UUID, actor *Actor) error {
	threat, err := s.threatRepo.GetByID(ctx, tenantID, threatID)
	if err != nil {
		return err
	}
	if err := s.threatRepo.Delete(ctx, tenantID, threatID); err != nil {
		return err
	}
	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.threat.updated", tenantID, actor, map[string]interface{}{
		"id":      threatID.String(),
		"deleted": true,
		"name":    threat.Name,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.deleted", tenantID, actor, map[string]interface{}{
		"id":   threatID.String(),
		"name": threat.Name,
	})
	return nil
}

// UpdateThreatStatus updates a threat status.
func (s *ThreatService) UpdateThreatStatus(ctx context.Context, tenantID, threatID uuid.UUID, actor *Actor, status model.ThreatStatus) (*model.Threat, error) {
	if !status.IsValid() {
		return nil, repository.ErrInvalidInput
	}
	before, err := s.threatRepo.GetByID(ctx, tenantID, threatID)
	if err != nil {
		return nil, err
	}
	if !canTransitionThreatStatus(before.Status, status) {
		return nil, fmt.Errorf("invalid threat status transition from %s to %s", before.Status, status)
	}
	after, err := s.threatRepo.UpdateStatus(ctx, tenantID, threatID, status)
	if err != nil {
		return nil, err
	}
	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.threat.updated", tenantID, actor, map[string]interface{}{
		"id":         threatID.String(),
		"old_status": before.Status,
		"new_status": after.Status,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.updated", tenantID, actor, map[string]interface{}{
		"id":     threatID.String(),
		"before": before.Status,
		"after":  after.Status,
	})
	return after, nil
}

// ListThreatIndicators returns indicators for a threat.
func (s *ThreatService) ListThreatIndicators(ctx context.Context, tenantID, threatID uuid.UUID, actor *Actor) ([]*model.ThreatIndicator, error) {
	items, err := s.indicatorRepo.ListByThreat(ctx, tenantID, threatID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]*model.ThreatIndicator, 0)
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.indicators_listed", tenantID, actor, map[string]interface{}{
		"id":    threatID.String(),
		"count": len(items),
	})
	return items, nil
}

// AddThreatIndicator adds an indicator to a threat. The returned bool is true
// when an existing indicator was updated (UPSERT matched on tenant_id+type+value).
func (s *ThreatService) AddThreatIndicator(ctx context.Context, tenantID, threatID, userID uuid.UUID, actor *Actor, req *dto.ThreatIndicatorRequest) (bool, *model.ThreatIndicator, error) {
	if !req.Type.IsValid() || !req.Severity.IsValid() {
		return false, nil, repository.ErrInvalidInput
	}
	// Check if an indicator with the same key already exists before UPSERT.
	_, lookupErr := s.indicatorRepo.GetByTypeValue(ctx, tenantID, req.Type, req.Value)
	existed := lookupErr == nil
	indicatorModel := &model.ThreatIndicator{
		TenantID:    tenantID,
		ThreatID:    &threatID,
		Type:        req.Type,
		Value:       req.Value,
		Description: req.Description,
		Severity:    req.Severity,
		Source:      req.Source,
		Confidence:  req.Confidence,
		Active:      true,
		ExpiresAt:   req.ExpiresAt,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
		CreatedBy:   &userID,
	}
	item, err := s.indicatorRepo.Create(ctx, indicatorModel)
	if err != nil {
		return false, nil, err
	}
	eventName := "cyber.threat.indicator_added"
	if existed {
		eventName = "cyber.threat.indicator_updated"
	}
	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, eventName, tenantID, actor, map[string]interface{}{
		"threat_id":    threatID.String(),
		"indicator_id": item.ID.String(),
		"type":         item.Type,
		"value":        item.Value,
	})
	_ = publishAuditEvent(ctx, s.producer, eventName, tenantID, actor, item)
	return existed, item, nil
}

// UpdateIndicatorStatus toggles whether an IOC is active for matching.
func (s *ThreatService) UpdateIndicatorStatus(ctx context.Context, tenantID, indicatorID uuid.UUID, actor *Actor, active bool) (*model.ThreatIndicator, error) {
	before, err := s.indicatorRepo.GetByID(ctx, tenantID, indicatorID)
	if err != nil {
		return nil, err
	}
	after, err := s.indicatorRepo.UpdateActive(ctx, tenantID, indicatorID, active)
	if err != nil {
		return nil, err
	}
	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.threat.indicator_updated", tenantID, actor, map[string]interface{}{
		"indicator_id": indicatorID.String(),
		"threat_id":    uuidPtrString(after.ThreatID),
		"old_active":   before.Active,
		"new_active":   after.Active,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.indicator_updated", tenantID, actor, map[string]interface{}{
		"indicator_id": indicatorID.String(),
		"before":       before.Active,
		"after":        after.Active,
	})
	return after, nil
}

// ThreatStats returns aggregated threat statistics.
func (s *ThreatService) ThreatStats(ctx context.Context, tenantID uuid.UUID, actor *Actor) (*model.ThreatStats, error) {
	stats, err := s.threatRepo.Stats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.stats_viewed", tenantID, actor, map[string]interface{}{})
	return stats, nil
}

// ThreatTrend returns dashboard trend data for threats.
func (s *ThreatService) ThreatTrend(ctx context.Context, tenantID uuid.UUID, actor *Actor, days int) ([]dto.ThreatTrendPoint, error) {
	items, err := s.threatRepo.Trend(ctx, tenantID, days)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.trend_viewed", tenantID, actor, map[string]interface{}{
		"days":  days,
		"count": len(items),
	})
	return items, nil
}

// ListThreatAlerts returns alerts inferred to be related to a threat.
func (s *ThreatService) ListThreatAlerts(ctx context.Context, tenantID, threatID uuid.UUID, actor *Actor) ([]*model.Alert, error) {
	threat, indicators, err := s.loadThreatContext(ctx, tenantID, threatID)
	if err != nil {
		return nil, err
	}
	techniques, values := threatAlertContext(threat, indicators)
	items, err := s.alertRepo.FindByThreatContext(ctx, tenantID, values, techniques, 50)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]*model.Alert, 0)
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.alerts_listed", tenantID, actor, map[string]interface{}{
		"id":    threatID.String(),
		"count": len(items),
	})
	return items, nil
}

// ListThreatTimeline builds a merged threat activity stream.
func (s *ThreatService) ListThreatTimeline(ctx context.Context, tenantID, threatID uuid.UUID, actor *Actor) ([]dto.ThreatTimelineEntry, error) {
	threat, indicators, err := s.loadThreatContext(ctx, tenantID, threatID)
	if err != nil {
		return nil, err
	}

	entries := make([]dto.ThreatTimelineEntry, 0, len(indicators)+8)
	entries = append(entries, dto.ThreatTimelineEntry{
		ID:          "threat-created",
		Kind:        "threat_created",
		Title:       "Threat created",
		Description: threat.Name,
		Timestamp:   threat.CreatedAt,
		Variant:     severityVariant(threat.Severity),
	})
	if threat.ContainedAt != nil {
		entries = append(entries, dto.ThreatTimelineEntry{
			ID:          "threat-contained",
			Kind:        "status_changed",
			Title:       "Threat contained",
			Description: fmt.Sprintf("Threat status moved to %s", model.ThreatStatusContained),
			Timestamp:   *threat.ContainedAt,
			Variant:     "success",
		})
	}
	if threat.Status != model.ThreatStatusActive && threat.UpdatedAt.After(threat.CreatedAt) {
		entries = append(entries, dto.ThreatTimelineEntry{
			ID:          "threat-status-current",
			Kind:        "status_changed",
			Title:       "Threat lifecycle updated",
			Description: fmt.Sprintf("Current status is %s", threat.Status),
			Timestamp:   threat.UpdatedAt,
			Variant:     statusVariant(threat.Status),
		})
	}

	for _, ind := range indicators {
		entries = append(entries, dto.ThreatTimelineEntry{
			ID:          "indicator-" + ind.ID.String(),
			Kind:        "indicator_added",
			Title:       fmt.Sprintf("%s indicator added", strings.ToUpper(string(ind.Type))),
			Description: ind.Value,
			Timestamp:   ind.CreatedAt,
			Variant:     severityVariant(ind.Severity),
		})
	}

	relatedAlerts, err := s.ListThreatAlerts(ctx, tenantID, threatID, nil)
	if err != nil {
		return nil, err
	}
	for _, alert := range relatedAlerts {
		entries = append(entries, dto.ThreatTimelineEntry{
			ID:          "alert-" + alert.ID.String(),
			Kind:        "alert_correlated",
			Title:       "Alert correlated",
			Description: alert.Title,
			Timestamp:   alert.CreatedAt,
			Variant:     severityVariant(alert.Severity),
		})
		timeline, err := s.alertRepo.ListTimeline(ctx, tenantID, alert.ID)
		if err != nil {
			return nil, err
		}
		for _, item := range timeline {
			entries = append(entries, dto.ThreatTimelineEntry{
				ID:          "alert-timeline-" + item.ID.String(),
				Kind:        "alert_" + item.Action,
				Title:       "Related alert activity",
				Description: fmt.Sprintf("%s: %s", alert.Title, item.Description),
				Timestamp:   item.CreatedAt,
				Variant:     "default",
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Timestamp.Equal(entries[j].Timestamp) {
			return entries[i].ID > entries[j].ID
		}
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})
	if len(entries) > 100 {
		entries = entries[:100]
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.timeline_viewed", tenantID, actor, map[string]interface{}{
		"id":    threatID.String(),
		"count": len(entries),
	})
	return entries, nil
}

// CheckIndicators checks ad-hoc values against stored indicators.
func (s *ThreatService) CheckIndicators(ctx context.Context, tenantID uuid.UUID, actor *Actor, values []string) ([]dto.IndicatorCheckResult, error) {
	matches, err := s.indicatorRepo.CheckValues(ctx, tenantID, values)
	if err != nil {
		return nil, err
	}
	results := make([]dto.IndicatorCheckResult, 0, len(values))
	for _, value := range values {
		matched := matches[strings.ToLower(value)]
		if matched == nil {
			matched = make([]*model.ThreatIndicator, 0)
		}
		results = append(results, dto.IndicatorCheckResult{
			Value:      value,
			Indicators: matched,
		})
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.checked", tenantID, actor, map[string]interface{}{
		"value_count": len(values),
	})
	return results, nil
}

// BulkImport imports indicators from a STIX/TAXII bundle.
// conflictMode controls behaviour when an indicator with the same (tenant, type, value) already exists:
//
//	"update" (default) – upsert, overwriting mutable fields.
//	"skip"             – silently skip duplicates.
//	"fail"             – return an error on the first duplicate.
func (s *ThreatService) BulkImport(ctx context.Context, tenantID, userID uuid.UUID, actor *Actor, payload []byte, source, conflictMode string) (int, error) {
	if conflictMode == "" {
		conflictMode = "update"
	}

	bundle, err := indicator.ParseSTIXBundle(payload, source)
	if err != nil {
		return 0, err
	}
	threatIDsByExternal := make(map[string]uuid.UUID)
	for _, parsedThreat := range bundle.Threats {
		threat, err := s.threatRepo.UpsertSyntheticThreat(ctx, tenantID, parsedThreat.Name, parsedThreat.Description, parsedThreat.Type, model.SeverityHigh, parsedThreat.Tags)
		if err != nil {
			return 0, err
		}
		threatIDsByExternal[parsedThreat.ExternalID] = threat.ID
	}

	created := 0
	for _, parsedIndicator := range bundle.Indicators {
		indicatorModel := parsedIndicator.Indicator
		indicatorModel.TenantID = tenantID
		indicatorModel.CreatedBy = &userID
		for _, externalID := range parsedIndicator.RelatedThreatIDs {
			if threatID, ok := threatIDsByExternal[externalID]; ok {
				indicatorModel.ThreatID = &threatID
				break
			}
		}

		if conflictMode == "skip" || conflictMode == "fail" {
			existing, _ := s.indicatorRepo.GetByTypeValue(ctx, tenantID, indicatorModel.Type, indicatorModel.Value)
			if existing != nil {
				if conflictMode == "fail" {
					return created, fmt.Errorf("duplicate indicator: type=%s value=%s", indicatorModel.Type, indicatorModel.Value)
				}
				// skip
				continue
			}
		}

		if _, err := s.indicatorRepo.Create(ctx, &indicatorModel); err != nil {
			return created, err
		}
		created++
	}
	if created > 0 {
		_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.bulk_imported", tenantID, actor, map[string]interface{}{
			"count": created,
		})
	}
	return created, nil
}

// ListIndicators returns paginated indicators.
func (s *ThreatService) ListIndicators(ctx context.Context, tenantID uuid.UUID, params *dto.IndicatorListParams, actor *Actor) (*dto.IndicatorListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	items, total, err := s.indicatorRepo.List(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.listed", tenantID, actor, map[string]interface{}{
		"count": len(items),
	})
	return &dto.IndicatorListResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(params.Page, params.PerPage, total),
	}, nil
}

func validateThreatInput(name string, threatType model.ThreatType, severity model.Severity, tacticIDs, techniqueIDs []string) error {
	if strings.TrimSpace(name) == "" || !threatType.IsValid() || !severity.IsValid() || severity == model.SeverityInfo {
		return repository.ErrInvalidInput
	}
	for _, tacticID := range tacticIDs {
		if _, ok := mitre.TacticByID(strings.TrimSpace(tacticID)); !ok {
			return fmt.Errorf("invalid MITRE tactic id: %s", tacticID)
		}
	}
	for _, techniqueID := range techniqueIDs {
		if _, ok := mitre.TechniqueByID(strings.TrimSpace(techniqueID)); !ok {
			return fmt.Errorf("invalid MITRE technique id: %s", techniqueID)
		}
	}
	return nil
}

func optionalStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func normalizeIndicatorConfidence(value float64) float64 {
	if value > 1 {
		value = value / 100
	}
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func coalesceSource(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "manual"
	}
	return trimmed
}

func canTransitionThreatStatus(current, next model.ThreatStatus) bool {
	if current == next {
		return true
	}
	switch current {
	case model.ThreatStatusActive:
		return next == model.ThreatStatusContained || next == model.ThreatStatusMonitoring
	case model.ThreatStatusContained:
		return next == model.ThreatStatusEradicated || next == model.ThreatStatusActive
	case model.ThreatStatusMonitoring:
		return next == model.ThreatStatusClosed || next == model.ThreatStatusActive
	case model.ThreatStatusEradicated:
		return next == model.ThreatStatusClosed
	default:
		return false
	}
}

func threatAlertContext(threat *model.Threat, indicators []*model.ThreatIndicator) ([]string, []string) {
	techniques := normalizeStrings(threat.MITRETechniqueIDs)
	values := make([]string, 0, len(indicators))
	seen := make(map[string]struct{}, len(indicators))
	for _, ind := range indicators {
		key := strings.ToLower(strings.TrimSpace(ind.Value))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		values = append(values, key)
	}
	return techniques, values
}

func (s *ThreatService) loadThreatContext(ctx context.Context, tenantID, threatID uuid.UUID) (*model.Threat, []*model.ThreatIndicator, error) {
	threat, err := s.threatRepo.GetByID(ctx, tenantID, threatID)
	if err != nil {
		return nil, nil, err
	}
	indicators, err := s.indicatorRepo.ListByThreat(ctx, tenantID, threatID)
	if err != nil {
		return nil, nil, err
	}
	return threat, indicators, nil
}

func severityVariant(severity model.Severity) string {
	switch severity {
	case model.SeverityCritical, model.SeverityHigh:
		return "error"
	case model.SeverityMedium:
		return "warning"
	default:
		return "default"
	}
}

func statusVariant(status model.ThreatStatus) string {
	switch status {
	case model.ThreatStatusContained, model.ThreatStatusEradicated, model.ThreatStatusClosed:
		return "success"
	case model.ThreatStatusMonitoring:
		return "warning"
	default:
		return "default"
	}
}
