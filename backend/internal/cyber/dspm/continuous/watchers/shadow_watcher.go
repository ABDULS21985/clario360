package watchers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/shadow"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

// ShadowWatcher detects unauthorized data duplicates on a weekly schedule.
type ShadowWatcher struct {
	detector  *shadow.Detector
	alertRepo *repository.AlertRepository
	producer  *events.Producer
	logger    zerolog.Logger
	interval  time.Duration
	threshold float64
	cancel    context.CancelFunc
}

// NewShadowWatcher creates a shadow copy detection watcher.
func NewShadowWatcher(
	detector *shadow.Detector,
	alertRepo *repository.AlertRepository,
	producer *events.Producer,
	interval time.Duration,
	threshold float64,
	logger zerolog.Logger,
) *ShadowWatcher {
	return &ShadowWatcher{
		detector:  detector,
		alertRepo: alertRepo,
		producer:  producer,
		logger:    logger.With().Str("watcher", "shadow").Logger(),
		interval:  interval,
		threshold: threshold,
	}
}

func (w *ShadowWatcher) Name() string { return "shadow" }

func (w *ShadowWatcher) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.scan(ctx); err != nil {
				w.logger.Error().Err(err).Msg("shadow scan failed")
			}
		}
	}
}

func (w *ShadowWatcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// DetectForTenant performs on-demand shadow detection for a specific tenant.
func (w *ShadowWatcher) DetectForTenant(ctx context.Context, tenantID uuid.UUID) (*shadow.DetectionResult, error) {
	return w.scanTenant(ctx, tenantID)
}

func (w *ShadowWatcher) scan(ctx context.Context) error {
	w.logger.Info().Msg("starting weekly shadow copy detection")

	// This would iterate all tenants in production
	// For now, the scheduler calls DetectForTenant per tenant
	return nil
}

func (w *ShadowWatcher) scanTenant(ctx context.Context, tenantID uuid.UUID) (*shadow.DetectionResult, error) {
	result, err := w.detector.Detect(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("shadow detection: %w", err)
	}

	alertsRaised := 0
	for _, match := range result.Matches {
		if match.HasLineage {
			// Legitimate copy tracked in lineage — skip
			continue
		}

		// Unauthorized shadow copy detected
		alertsRaised++
		w.createShadowAlert(ctx, tenantID, match)
	}

	// Publish detection event
	if w.producer != nil {
		evt, _ := events.NewEvent("cyber.dspm.shadow.scan_completed", "cyber-service", tenantID.String(), map[string]interface{}{
			"sources_count":  result.SourcesCount,
			"tables_count":   result.TablesCount,
			"matches_found":  len(result.Matches),
			"alerts_raised":  alertsRaised,
			"duration_ms":    result.Duration.Milliseconds(),
		})
		if evt != nil {
			_ = w.producer.Publish(ctx, events.Topics.DSPMEvents, evt)
		}
	}

	w.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("matches", len(result.Matches)).
		Int("alerts", alertsRaised).
		Dur("duration", result.Duration).
		Msg("shadow detection completed")

	return result, nil
}

func (w *ShadowWatcher) createShadowAlert(ctx context.Context, tenantID uuid.UUID, match shadow.ShadowMatch) {
	now := time.Now().UTC()

	metadata, _ := json.Marshal(map[string]interface{}{
		"source_asset_id": match.SourceAssetID.String(),
		"target_asset_id": match.TargetAssetID.String(),
		"source_table":    match.SourceTable,
		"target_table":    match.TargetTable,
		"match_type":      match.MatchType,
		"similarity":      match.Similarity,
		"fingerprint":     match.Fingerprint,
	})

	alert := &model.Alert{
		ID:       uuid.New(),
		TenantID: tenantID,
		Title:    fmt.Sprintf("Possible shadow copy detected: %s.%s → %s.%s", match.SourceAssetName, match.SourceTable, match.TargetAssetName, match.TargetTable),
		Description: fmt.Sprintf(
			"Table %s in %s has a %s match (%.0f%% similarity) with table %s in %s. No data lineage pipeline connects these sources, suggesting an unauthorized data copy.",
			match.SourceTable, match.SourceAssetName,
			match.MatchType, match.Similarity*100,
			match.TargetTable, match.TargetAssetName,
		),
		Severity: model.SeverityHigh,
		Status:   model.AlertStatusNew,
		Source:   "dspm_shadow",
		AssetID:  &match.SourceAssetID,
		AssetIDs: []uuid.UUID{match.SourceAssetID, match.TargetAssetID},
		Explanation: model.AlertExplanation{
			Summary: fmt.Sprintf("Unauthorized data duplicate detected between %s and %s", match.SourceAssetName, match.TargetAssetName),
			Reason:  "Schema fingerprint comparison found structurally identical tables across different data sources with no lineage connecting them.",
			Evidence: []model.AlertEvidence{
				{Label: "Source", Field: "source", Value: fmt.Sprintf("%s.%s", match.SourceAssetName, match.SourceTable), Description: "Source table"},
				{Label: "Target", Field: "target", Value: fmt.Sprintf("%s.%s", match.TargetAssetName, match.TargetTable), Description: "Duplicate table"},
				{Label: "Match Type", Field: "match_type", Value: match.MatchType, Description: "How the duplicate was detected"},
				{Label: "Similarity", Field: "similarity", Value: fmt.Sprintf("%.1f%%", match.Similarity*100), Description: "Structural similarity score"},
			},
			RecommendedActions: []string{
				"Investigate whether this data copy is authorized.",
				"If unauthorized, determine how the copy was created and by whom.",
				"Remove or properly govern the shadow copy.",
				"Add the data flow to lineage tracking if it is a legitimate copy.",
			},
		},
		ConfidenceScore: match.Similarity,
		EventCount:      1,
		FirstEventAt:    now,
		LastEventAt:     now,
		Tags:            []string{"dspm", "shadow_copy", match.MatchType},
		Metadata:        metadata,
	}

	if _, err := w.alertRepo.Create(ctx, alert); err != nil {
		w.logger.Error().Err(err).Msg("create shadow copy alert")
	}
}
