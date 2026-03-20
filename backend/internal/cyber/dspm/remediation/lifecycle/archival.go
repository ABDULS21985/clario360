package lifecycle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ArchivalAction defines the recommended disposition for a data asset.
type ArchivalAction string

const (
	// ArchivalActionArchive recommends moving the asset to cold or archive storage.
	ArchivalActionArchive ArchivalAction = "archive"
	// ArchivalActionDelete recommends permanent deletion of the asset.
	ArchivalActionDelete ArchivalAction = "delete"
	// ArchivalActionRetain recommends keeping the asset with mandatory periodic review.
	ArchivalActionRetain ArchivalAction = "retain"
)

// ArchivalPriority indicates urgency of the archival recommendation.
type ArchivalPriority string

const (
	ArchivalPriorityCritical ArchivalPriority = "critical"
	ArchivalPriorityHigh     ArchivalPriority = "high"
	ArchivalPriorityMedium   ArchivalPriority = "medium"
	ArchivalPriorityLow      ArchivalPriority = "low"
)

// ArchivalRecommendation captures a disposition recommendation for a data asset
// that has exceeded its retention period.
type ArchivalRecommendation struct {
	AssetID        uuid.UUID `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	Classification string    `json:"classification"`
	Reason         string    `json:"reason"`
	Action         string    `json:"action"`
	Priority       string    `json:"priority"`
}

// ArchivalRecommender analyses data assets and produces disposition
// recommendations based on retention policies and data classification.
type ArchivalRecommender struct {
	assetLister AssetLister
	logger      zerolog.Logger
}

// NewArchivalRecommender constructs an ArchivalRecommender with the required dependencies.
func NewArchivalRecommender(assetLister AssetLister, logger zerolog.Logger) *ArchivalRecommender {
	return &ArchivalRecommender{
		assetLister: assetLister,
		logger:      logger.With().Str("component", "archival_recommender").Logger(),
	}
}

// Recommend evaluates all active data assets for the given tenant and returns
// archival recommendations for assets older than retentionDays. The recommended
// action depends on the asset's data classification:
//
//   - public data exceeding retention  → delete
//   - internal data exceeding retention → archive
//   - confidential/restricted data exceeding retention → retain with periodic review
func (ar *ArchivalRecommender) Recommend(ctx context.Context, tenantID uuid.UUID, retentionDays int) ([]ArchivalRecommendation, error) {
	if retentionDays <= 0 {
		return nil, fmt.Errorf("archival recommender: retentionDays must be positive, got %d", retentionDays)
	}

	assets, err := ar.assetLister.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("archival recommender: list assets: %w", err)
	}

	ar.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("asset_count", len(assets)).
		Int("retention_days", retentionDays).
		Msg("generating archival recommendations")

	now := time.Now().UTC()
	var recommendations []ArchivalRecommendation

	for _, asset := range assets {

		ageDays := int(now.Sub(asset.CreatedAt).Hours() / 24)
		if ageDays <= retentionDays {
			continue
		}

		daysOver := ageDays - retentionDays
		cls := strings.ToLower(asset.DataClassification)

		var action string
		var reason string
		var priority string

		switch cls {
		case "public":
			action = string(ArchivalActionDelete)
			reason = fmt.Sprintf(
				"Public data asset exceeded retention period by %d days. No regulatory requirement to retain; recommend permanent deletion to reduce storage costs and attack surface.",
				daysOver,
			)
			priority = archivalPriority(daysOver)

		case "internal":
			action = string(ArchivalActionArchive)
			reason = fmt.Sprintf(
				"Internal data asset exceeded retention period by %d days. Recommend archival to cold storage with metadata preservation for potential future reference.",
				daysOver,
			)
			priority = archivalPriority(daysOver)

		case "confidential":
			action = string(ArchivalActionRetain)
			reason = fmt.Sprintf(
				"Confidential data asset exceeded retention period by %d days. Regulatory and compliance obligations may require continued retention. Schedule mandatory review with data owner and legal team.",
				daysOver,
			)
			priority = archivalPriorityForSensitive(daysOver)

		case "restricted":
			action = string(ArchivalActionRetain)
			reason = fmt.Sprintf(
				"Restricted data asset exceeded retention period by %d days. Highest sensitivity classification requires executive approval before any disposition action. Schedule immediate review with CISO and legal counsel.",
				daysOver,
			)
			priority = archivalPriorityForSensitive(daysOver)

		default:
			// Unknown classification — treat conservatively as retain-with-review.
			action = string(ArchivalActionRetain)
			reason = fmt.Sprintf(
				"Data asset with unrecognized classification %q exceeded retention period by %d days. Classify the asset before taking disposition action.",
				asset.DataClassification, daysOver,
			)
			priority = string(ArchivalPriorityHigh)
		}

		rec := ArchivalRecommendation{
			AssetID:        asset.ID,
			AssetName:      asset.AssetName,
			Classification: asset.DataClassification,
			Reason:         reason,
			Action:         action,
			Priority:       priority,
		}

		recommendations = append(recommendations, rec)
	}

	ar.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("recommendations", len(recommendations)).
		Msg("archival recommendation generation complete")

	return recommendations, nil
}

// archivalPriority maps days overdue to a priority for public/internal data.
func archivalPriority(daysOver int) string {
	switch {
	case daysOver >= 180:
		return string(ArchivalPriorityHigh)
	case daysOver >= 90:
		return string(ArchivalPriorityMedium)
	default:
		return string(ArchivalPriorityLow)
	}
}

// archivalPriorityForSensitive maps days overdue to a priority for
// confidential/restricted data, which always receives elevated urgency.
func archivalPriorityForSensitive(daysOver int) string {
	switch {
	case daysOver >= 180:
		return string(ArchivalPriorityCritical)
	case daysOver >= 90:
		return string(ArchivalPriorityHigh)
	default:
		return string(ArchivalPriorityMedium)
	}
}
