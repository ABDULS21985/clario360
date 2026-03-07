package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/detection"
	"github.com/clario360/platform/internal/cyber/model"
)

// DetectionService is the orchestration layer between Kafka events and the detection engine.
type DetectionService struct {
	engine *detection.DetectionEngine
	logger zerolog.Logger
}

// NewDetectionService creates a new DetectionService.
func NewDetectionService(engine *detection.DetectionEngine, logger zerolog.Logger) *DetectionService {
	return &DetectionService{engine: engine, logger: logger}
}

// Start launches the background reload loop.
func (s *DetectionService) Start(ctx context.Context, refreshInterval time.Duration) {
	if s.engine == nil {
		return
	}
	s.engine.Start(ctx, refreshInterval)
}

// ReloadTenant requests an immediate rule reload for a tenant.
func (s *DetectionService) ReloadTenant(tenantID uuid.UUID) {
	if s.engine == nil {
		return
	}
	s.engine.RequestReload(tenantID)
}

// ProcessEvents evaluates a batch of already-normalized security events.
func (s *DetectionService) ProcessEvents(ctx context.Context, tenantID uuid.UUID, events []model.SecurityEvent) ([]*model.Alert, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("detection engine is not configured")
	}
	return s.engine.ProcessEvents(ctx, tenantID, events)
}

// DecodeEventBatch decodes a cloud event payload into one or more normalized security events.
func (s *DetectionService) DecodeEventBatch(eventData json.RawMessage) ([]model.SecurityEvent, error) {
	var batch []model.SecurityEvent
	if err := json.Unmarshal(eventData, &batch); err == nil {
		return batch, nil
	}
	var single model.SecurityEvent
	if err := json.Unmarshal(eventData, &single); err != nil {
		return nil, err
	}
	return []model.SecurityEvent{single}, nil
}
