package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/indicator"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

// ThreatService manages threats and indicators.
type ThreatService struct {
	threatRepo    *repository.ThreatRepository
	indicatorRepo *repository.IndicatorRepository
	producer      *events.Producer
	logger        zerolog.Logger
}

// NewThreatService creates a new ThreatService.
func NewThreatService(
	threatRepo *repository.ThreatRepository,
	indicatorRepo *repository.IndicatorRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *ThreatService {
	return &ThreatService{
		threatRepo:    threatRepo,
		indicatorRepo: indicatorRepo,
		producer:      producer,
		logger:        logger,
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
	totalPages := (total + params.PerPage - 1) / params.PerPage
	if totalPages < 1 {
		totalPages = 1
	}
	return &dto.ThreatListResponse{
		Data:       items,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
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
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.viewed", tenantID, actor, map[string]interface{}{
		"id": threatID.String(),
	})
	return threat, nil
}

// UpdateThreatStatus updates a threat status.
func (s *ThreatService) UpdateThreatStatus(ctx context.Context, tenantID, threatID uuid.UUID, actor *Actor, status model.ThreatStatus) (*model.Threat, error) {
	before, err := s.threatRepo.GetByID(ctx, tenantID, threatID)
	if err != nil {
		return nil, err
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
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.indicators_listed", tenantID, actor, map[string]interface{}{
		"id":    threatID.String(),
		"count": len(items),
	})
	return items, nil
}

// AddThreatIndicator adds an indicator to a threat.
func (s *ThreatService) AddThreatIndicator(ctx context.Context, tenantID, threatID, userID uuid.UUID, actor *Actor, req *dto.ThreatIndicatorRequest) (*model.ThreatIndicator, error) {
	if !req.Type.IsValid() || !req.Severity.IsValid() {
		return nil, repository.ErrInvalidInput
	}
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
		return nil, err
	}
	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.threat.indicator_added", tenantID, actor, map[string]interface{}{
		"threat_id":    threatID.String(),
		"indicator_id": item.ID.String(),
		"type":         item.Type,
		"value":        item.Value,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat.indicator_added", tenantID, actor, item)
	return item, nil
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

// CheckIndicators checks ad-hoc values against stored indicators.
func (s *ThreatService) CheckIndicators(ctx context.Context, tenantID uuid.UUID, actor *Actor, values []string) ([]dto.IndicatorCheckResult, error) {
	matches, err := s.indicatorRepo.CheckValues(ctx, tenantID, values)
	if err != nil {
		return nil, err
	}
	results := make([]dto.IndicatorCheckResult, 0, len(values))
	for _, value := range values {
		results = append(results, dto.IndicatorCheckResult{
			Value:      value,
			Indicators: matches[strings.ToLower(value)],
		})
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.checked", tenantID, actor, map[string]interface{}{
		"value_count": len(values),
	})
	return results, nil
}

// BulkImport imports indicators from a STIX/TAXII bundle.
func (s *ThreatService) BulkImport(ctx context.Context, tenantID, userID uuid.UUID, actor *Actor, payload []byte, source string) (int, error) {
	bundle, err := indicator.ParseSTIXBundle(payload, source)
	if err != nil {
		return 0, err
	}
	threatIDsByExternal := make(map[string]uuid.UUID)
	for _, parsedThreat := range bundle.Threats {
		threat, err := s.threatRepo.UpsertSyntheticThreat(ctx, tenantID, parsedThreat.Name, parsedThreat.Type, model.SeverityHigh, parsedThreat.Tags)
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
	totalPages := (total + params.PerPage - 1) / params.PerPage
	if totalPages < 1 {
		totalPages = 1
	}
	return &dto.IndicatorListResponse{
		Data:       items,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}
