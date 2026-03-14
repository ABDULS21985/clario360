package analyzer

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// StaleAccessAnalyzer detects permissions unused beyond a configurable threshold.
// Default threshold: 90 days. A permission is stale if last_used_at is NULL or
// older than now() - thresholdDays.
type StaleAccessAnalyzer struct {
	repo   MappingProvider
	logger zerolog.Logger
}

// NewStaleAccessAnalyzer creates a new stale access detector.
func NewStaleAccessAnalyzer(repo MappingProvider, logger zerolog.Logger) *StaleAccessAnalyzer {
	return &StaleAccessAnalyzer{
		repo:   repo,
		logger: logger.With().Str("analyzer", "stale_access").Logger(),
	}
}

// Analyze finds all stale permissions for a tenant, groups by identity, and
// returns sorted by weighted risk descending.
func (a *StaleAccessAnalyzer) Analyze(ctx context.Context, tenantID uuid.UUID, thresholdDays int) ([]model.StaleAccessResult, error) {
	if thresholdDays <= 0 {
		thresholdDays = 90
	}

	mappings, err := a.repo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().UTC().Add(-time.Duration(thresholdDays) * 24 * time.Hour)

	var results []model.StaleAccessResult
	for _, m := range mappings {
		isStale := false
		var daysStale int

		if m.LastUsedAt == nil {
			isStale = true
			daysStale = int(math.Ceil(time.Since(m.DiscoveredAt).Hours() / 24))
		} else if m.LastUsedAt.Before(cutoff) {
			isStale = true
			daysStale = int(math.Ceil(time.Since(*m.LastUsedAt).Hours() / 24))
		}

		if !isStale {
			continue
		}

		results = append(results, model.StaleAccessResult{
			MappingID:          m.ID,
			IdentityType:       m.IdentityType,
			IdentityID:         m.IdentityID,
			IdentityName:       m.IdentityName,
			DataAssetID:        m.DataAssetID,
			DataAssetName:      m.DataAssetName,
			DataClassification: m.DataClassification,
			PermissionType:     m.PermissionType,
			LastUsedAt:         m.LastUsedAt,
			DaysStale:          daysStale,
			SensitivityWeight:  m.SensitivityWeight,
		})
	}

	// Sort by sensitivity weight descending (highest risk stale permissions first).
	sortByWeightDesc(results)
	return results, nil
}

// MarkStale updates is_stale on the access_mappings table for all stale entries.
func (a *StaleAccessAnalyzer) MarkStale(ctx context.Context, tenantID uuid.UUID, thresholdDays int, repo StaleFlagUpdater) (int, error) {
	if thresholdDays <= 0 {
		thresholdDays = 90
	}
	return repo.MarkStale(ctx, tenantID, thresholdDays)
}

// StaleFlagUpdater sets is_stale flags in the database.
type StaleFlagUpdater interface {
	MarkStale(ctx context.Context, tenantID uuid.UUID, thresholdDays int) (int, error)
}

func sortByWeightDesc(results []model.StaleAccessResult) {
	n := len(results)
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if results[j].SensitivityWeight > results[i].SensitivityWeight {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
