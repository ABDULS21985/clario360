package lifecycle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// StaleDataFinding records a data asset that has not been scanned recently
// and may contain outdated or orphaned data.
type StaleDataFinding struct {
	AssetID        uuid.UUID `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	Classification string    `json:"classification"`
	DaysStale      int       `json:"days_stale"`
	Confidence     string    `json:"confidence"`
	Recommendation string    `json:"recommendation"`
}

// StaleDataDetector identifies data assets that have not been scanned
// within acceptable timeframes, indicating potentially orphaned or
// unmanaged data.
type StaleDataDetector struct {
	assetLister AssetLister
	logger      zerolog.Logger
}

// NewStaleDataDetector constructs a StaleDataDetector with the required dependencies.
func NewStaleDataDetector(assetLister AssetLister, logger zerolog.Logger) *StaleDataDetector {
	return &StaleDataDetector{
		assetLister: assetLister,
		logger:      logger.With().Str("component", "stale_data_detector").Logger(),
	}
}

// staleThresholdDays is the minimum number of days since last scan before an
// asset is considered stale.
const staleThresholdDays = 90

// Detect scans all active data assets for the given tenant and returns findings
// for any asset whose LastScannedAt is nil or more than 90 days in the past.
func (sd *StaleDataDetector) Detect(ctx context.Context, tenantID uuid.UUID) ([]StaleDataFinding, error) {
	assets, err := sd.assetLister.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("stale data detector: list assets: %w", err)
	}

	sd.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("asset_count", len(assets)).
		Msg("starting stale data detection")

	now := time.Now().UTC()
	var findings []StaleDataFinding

	for _, asset := range assets {

		var daysStale int
		var neverScanned bool

		if asset.LastScannedAt == nil {
			// Asset has never been scanned — use creation date for staleness calculation.
			daysStale = int(now.Sub(asset.CreatedAt).Hours() / 24)
			neverScanned = true
		} else {
			daysSinceScan := int(now.Sub(*asset.LastScannedAt).Hours() / 24)
			if daysSinceScan < staleThresholdDays {
				continue
			}
			daysStale = daysSinceScan
		}

		confidence := staleConfidence(neverScanned, daysStale)
		recommendation := staleRecommendation(asset.DataClassification, neverScanned, daysStale)

		finding := StaleDataFinding{
			AssetID:        asset.ID,
			AssetName:      asset.AssetName,
			Classification: asset.DataClassification,
			DaysStale:      daysStale,
			Confidence:     confidence,
			Recommendation: recommendation,
		}

		findings = append(findings, finding)
	}

	sd.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("findings_count", len(findings)).
		Msg("stale data detection complete")

	return findings, nil
}

// staleConfidence determines confidence in the staleness assessment.
//
//	Never scanned → high (definitively unmanaged)
//	>= 180 days   → high (strong evidence)
//	>= 90 days    → medium (above threshold)
func staleConfidence(neverScanned bool, daysStale int) string {
	if neverScanned {
		return "high"
	}
	if daysStale >= 180 {
		return "high"
	}
	return "medium"
}

// staleRecommendation generates an actionable recommendation based on the
// asset's classification and staleness severity.
func staleRecommendation(classification string, neverScanned bool, daysStale int) string {
	cls := strings.ToLower(classification)

	switch cls {
	case "public":
		if neverScanned || daysStale >= 180 {
			return "Delete unmanaged public data asset. No scan history indicates the asset is likely orphaned."
		}
		return "Re-scan public data asset to verify contents and update classification."

	case "internal":
		if neverScanned {
			return "Immediately scan and classify internal asset. Archive if no active consumers are identified."
		}
		if daysStale >= 180 {
			return "Archive internal data asset. Extended staleness indicates low operational value."
		}
		return "Re-scan internal data asset and verify access controls remain appropriate."

	case "confidential":
		if neverScanned {
			return "Urgent: scan and classify unmanaged confidential asset. Verify encryption and access controls."
		}
		return "Re-scan confidential data asset to ensure classification accuracy and compliance controls."

	case "restricted":
		if neverScanned {
			return "Critical: immediately scan restricted asset. Verify encryption, audit logging, and access restrictions."
		}
		return "Re-scan restricted data asset. Validate all compliance controls and access audit trail."

	default:
		if neverScanned {
			return "Scan unclassified data asset to determine sensitivity and apply appropriate controls."
		}
		return "Re-scan data asset to update classification and verify security posture."
	}
}
