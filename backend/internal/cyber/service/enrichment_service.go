package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/enrichment"
	"github.com/clario360/platform/internal/cyber/metrics"
	"github.com/clario360/platform/internal/cyber/repository"
)

// EnrichmentService runs the enrichment pipeline on assets and persists updates.
type EnrichmentService struct {
	pipeline  *enrichment.Pipeline
	assetRepo *repository.AssetRepository
	metrics   *metrics.Metrics
	logger    zerolog.Logger
}

// NewEnrichmentService creates a new EnrichmentService.
func NewEnrichmentService(
	pipeline *enrichment.Pipeline,
	assetRepo *repository.AssetRepository,
	m *metrics.Metrics,
	logger zerolog.Logger,
) *EnrichmentService {
	return &EnrichmentService{
		pipeline:  pipeline,
		assetRepo: assetRepo,
		metrics:   m,
		logger:    logger,
	}
}

// EnrichAsset runs the full enrichment pipeline on a single asset and persists
// any fields that were added/modified by enrichers.
func (s *EnrichmentService) EnrichAsset(ctx context.Context, tenantID, assetID uuid.UUID) error {
	asset, err := s.assetRepo.GetByID(ctx, tenantID, assetID)
	if err != nil {
		return err
	}

	start := time.Now()
	results := s.pipeline.Run(ctx, asset)

	for _, r := range results {
		status := "success"
		if r.Error != nil {
			status = "error"
			s.metrics.EnrichmentErrors.WithLabelValues(tenantID.String(), r.EnricherName, "pipeline").Inc()
		}
		s.metrics.EnrichmentTotal.WithLabelValues(tenantID.String(), r.EnricherName, status).Inc()
		s.metrics.EnrichmentDuration.WithLabelValues(tenantID.String(), r.EnricherName).Observe(r.Duration.Seconds())
	}

	// Persist any fields the enrichers may have updated (hostname, ip_address, os, os_version, metadata)
	if hasEnrichedFields(results) {
		s.logger.Debug().
			Str("asset_id", assetID.String()).
			Dur("elapsed", time.Since(start)).
			Msg("persisting enriched asset fields")

		updateReq := &dto.UpdateAssetRequest{
			Hostname:  asset.Hostname,
			IPAddress: asset.IPAddress,
			OS:        asset.OS,
			OSVersion: asset.OSVersion,
			Metadata:  json.RawMessage(asset.Metadata),
		}
		_, err = s.assetRepo.Update(ctx, tenantID, assetID, updateReq)
		return err
	}

	return nil
}

// EnrichBatch enriches multiple assets, logging individual failures without aborting.
func (s *EnrichmentService) EnrichBatch(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) {
	for _, id := range assetIDs {
		if ctx.Err() != nil {
			return
		}
		if err := s.EnrichAsset(ctx, tenantID, id); err != nil {
			s.logger.Warn().Err(err).Str("asset_id", id.String()).Msg("enrichment failed for asset")
		}
	}
}

func hasEnrichedFields(results []enrichment.EnrichmentResult) bool {
	for _, r := range results {
		if len(r.FieldsAdded) > 0 {
			return true
		}
	}
	return false
}
