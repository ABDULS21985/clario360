package watchers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm"
	"github.com/clario360/platform/internal/cyber/dspm/compliance"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

// PipelineWatcher monitors pipeline completions and triggers DSPM scans on new data.
type PipelineWatcher struct {
	db           *pgxpool.Pool
	repo         *repository.DSPMRepository
	alertRepo    *repository.AlertRepository
	classifier   *dspm.DSPMClassifier
	tagger       *compliance.ComplianceTagger
	producer     *events.Producer
	logger       zerolog.Logger
	cancel       context.CancelFunc
}

// NewPipelineWatcher creates a pipeline completion watcher.
func NewPipelineWatcher(
	db *pgxpool.Pool,
	repo *repository.DSPMRepository,
	alertRepo *repository.AlertRepository,
	classifier *dspm.DSPMClassifier,
	tagger *compliance.ComplianceTagger,
	producer *events.Producer,
	logger zerolog.Logger,
) *PipelineWatcher {
	return &PipelineWatcher{
		db:         db,
		repo:       repo,
		alertRepo:  alertRepo,
		classifier: classifier,
		tagger:     tagger,
		producer:   producer,
		logger:     logger.With().Str("watcher", "pipeline").Logger(),
	}
}

func (w *PipelineWatcher) Name() string { return "pipeline" }

func (w *PipelineWatcher) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)
	<-ctx.Done()
	return nil
}

func (w *PipelineWatcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// HandlePipelineCompleted processes a pipeline completion event.
// Called by the event consumer when a pipeline.completed event is received.
func (w *PipelineWatcher) HandlePipelineCompleted(ctx context.Context, evt *events.Event) error {
	var data EventData
	if err := json.Unmarshal(evt.Data, &data); err != nil {
		return fmt.Errorf("unmarshal pipeline event: %w", err)
	}

	if data.Status != "completed" {
		return nil
	}

	tenantID, err := uuid.Parse(evt.TenantID)
	if err != nil {
		return fmt.Errorf("parse tenant ID: %w", err)
	}

	w.logger.Info().
		Str("pipeline_id", data.PipelineID.String()).
		Str("tenant_id", tenantID.String()).
		Msg("pipeline completed, triggering DSPM scan on target assets")

	// Find target assets affected by this pipeline
	targetAssets, err := w.findTargetAssets(ctx, tenantID, data)
	if err != nil {
		w.logger.Error().Err(err).Msg("find target assets for pipeline")
		return nil // Don't fail event processing
	}

	alertsRaised := 0
	for _, asset := range targetAssets {
		classification := w.classifier.Classify(asset)

		// Get previous classification for this asset
		prevAsset, _ := w.repo.GetDataAssetByID(ctx, tenantID, asset.ID)

		// Tag with compliance frameworks
		var complianceTags []compliance.ComplianceTag
		if classification.ContainsPII {
			complianceTags = w.tagger.TagPIITypes(classification.PIITypes)
		}

		// Detect new PII that wasn't in previous scan
		if prevAsset != nil && classification.ContainsPII {
			newPII := findNewPIITypes(prevAsset.PIITypes, classification.PIITypes)
			if len(newPII) > 0 {
				alertsRaised++
				w.createNewPIIAlert(ctx, tenantID, asset, newPII, complianceTags, data.PipelineID)
			}
		}

		w.logger.Debug().
			Str("asset_id", asset.ID.String()).
			Str("classification", classification.Classification).
			Bool("contains_pii", classification.ContainsPII).
			Int("compliance_tags", len(complianceTags)).
			Msg("asset scanned after pipeline completion")
	}

	// Publish scan completed event
	if w.producer != nil {
		scanEvt, _ := events.NewEvent("cyber.dspm.continuous.scan_completed", "cyber-service", tenantID.String(), map[string]interface{}{
			"watcher":       "pipeline",
			"pipeline_id":   data.PipelineID.String(),
			"assets_scanned": len(targetAssets),
			"alerts_raised": alertsRaised,
		})
		if scanEvt != nil {
			_ = w.producer.Publish(ctx, events.Topics.DSPMEvents, scanEvt)
		}
	}

	return nil
}

// findTargetAssets finds assets that were targets of the completed pipeline.
func (w *PipelineWatcher) findTargetAssets(ctx context.Context, tenantID uuid.UUID, data EventData) ([]*model.Asset, error) {
	if data.TargetID == nil {
		return nil, nil
	}

	rows, err := w.db.Query(ctx, `
		SELECT id, tenant_id, name, type::text, host(ip_address), hostname, mac_address::text,
		       os, os_version, owner, department, location, criticality::text, status::text,
		       discovered_at, last_seen_at, discovery_source, metadata, tags, created_by, created_at, updated_at
		FROM assets
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
	`, tenantID, *data.TargetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*model.Asset
	for rows.Next() {
		asset, err := scanAssetRow(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (w *PipelineWatcher) createNewPIIAlert(ctx context.Context, tenantID uuid.UUID, asset *model.Asset, newPII []string, tags []compliance.ComplianceTag, pipelineID uuid.UUID) {
	now := time.Now().UTC()
	tagSummary := make([]string, 0, len(tags))
	for _, t := range tags {
		tagSummary = append(tagSummary, fmt.Sprintf("%s %s", t.Framework, t.Article))
	}

	metadata, _ := json.Marshal(map[string]interface{}{
		"pipeline_id":    pipelineID.String(),
		"new_pii_types":  newPII,
		"compliance_tags": tagSummary,
	})

	alert := &model.Alert{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Title:       fmt.Sprintf("New PII detected in %s after pipeline execution", asset.Name),
		Description: fmt.Sprintf("Pipeline %s wrote data containing new PII types (%v) to asset %s. Compliance frameworks affected: %v", pipelineID.String(), newPII, asset.Name, tagSummary),
		Severity:    model.SeverityHigh,
		Status:      model.AlertStatusNew,
		Source:      "dspm_continuous",
		AssetID:     &asset.ID,
		AssetIDs:    []uuid.UUID{asset.ID},
		Explanation: model.AlertExplanation{
			Summary: fmt.Sprintf("New PII types %v detected in asset %s after pipeline completion", newPII, asset.Name),
			Reason:  "Pipeline data write introduced previously unseen PII types.",
			Evidence: []model.AlertEvidence{
				{Label: "Asset", Field: "asset_name", Value: asset.Name, Description: "Data asset with new PII"},
				{Label: "PII Types", Field: "new_pii_types", Value: newPII, Description: "Newly detected PII types"},
				{Label: "Pipeline", Field: "pipeline_id", Value: pipelineID.String(), Description: "Pipeline that triggered the detection"},
			},
			RecommendedActions: []string{
				"Review the data written by the pipeline for PII content.",
				"Update data classification and apply appropriate access controls.",
				"Ensure compliance tags are reflected in data handling procedures.",
			},
		},
		ConfidenceScore: 0.9,
		EventCount:      1,
		FirstEventAt:    now,
		LastEventAt:     now,
		Tags:            []string{"dspm", "pii", "pipeline"},
		Metadata:        metadata,
	}

	if _, err := w.alertRepo.Create(ctx, alert); err != nil {
		w.logger.Error().Err(err).Str("asset_id", asset.ID.String()).Msg("create new PII alert")
	}
}

// findNewPIITypes returns PII types that are in current but not in previous.
func findNewPIITypes(previous, current []string) []string {
	prevSet := make(map[string]bool, len(previous))
	for _, p := range previous {
		prevSet[p] = true
	}
	var newTypes []string
	for _, c := range current {
		if !prevSet[c] {
			newTypes = append(newTypes, c)
		}
	}
	return newTypes
}

// scanAssetRow scans a database row into a model.Asset.
func scanAssetRow(rows interface{ Scan(dest ...interface{}) error }) (*model.Asset, error) {
	var (
		asset   model.Asset
		typeStr string
		critStr string
		statStr string
	)
	if err := rows.Scan(
		&asset.ID, &asset.TenantID, &asset.Name, &typeStr,
		&asset.IPAddress, &asset.Hostname, &asset.MACAddress,
		&asset.OS, &asset.OSVersion, &asset.Owner, &asset.Department, &asset.Location,
		&critStr, &statStr,
		&asset.DiscoveredAt, &asset.LastSeenAt, &asset.DiscoverySource,
		&asset.Metadata, &asset.Tags, &asset.CreatedBy, &asset.CreatedAt, &asset.UpdatedAt,
	); err != nil {
		return nil, err
	}
	asset.Type = model.AssetType(typeStr)
	asset.Criticality = model.Criticality(critStr)
	asset.Status = model.AssetStatus(statStr)
	if asset.Tags == nil {
		asset.Tags = []string{}
	}
	return &asset, nil
}
