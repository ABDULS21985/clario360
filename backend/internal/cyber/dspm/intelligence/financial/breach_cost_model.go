package financial

import (
	"strings"

	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// Per-record costs based on IBM/Ponemon Cost of a Data Breach Report.
const (
	costPerRecordHealthcare = 429.0
	costPerRecordFinancial  = 388.0
	costPerRecordGeneralPII = 169.0
	costPerRecordInternal   = 25.0
	costPerRecordPublic     = 0.0

	// Additional per-record cost components.
	notificationCostPerRecord   = 2.50
	investigationBaseFixed      = 50_000.0
	investigationCostPerRecord  = 0.50
	legalCostPerRecord          = 1.50
	reputationRevenueMultiplier = 0.03

	// Business disruption cost factors by network exposure.
	disruptionInternetFacing = 100_000.0
	disruptionVPN            = 30_000.0
	disruptionInternal       = 10_000.0
	disruptionDefault        = 15_000.0
)

// BreachCostModel calculates data breach costs using the IBM/Ponemon methodology
// with industry-specific per-record rates and multi-component cost breakdowns.
type BreachCostModel struct {
	logger zerolog.Logger
}

// NewBreachCostModel creates a new breach cost model instance.
func NewBreachCostModel(logger zerolog.Logger) *BreachCostModel {
	return &BreachCostModel{
		logger: logger.With().Str("component", "breach_cost_model").Logger(),
	}
}

// CostPerRecord returns the estimated per-record breach cost based on the
// data classification and PII status. Rates are from the IBM/Ponemon Cost
// of a Data Breach Report, adjusted for industry verticals.
func (m *BreachCostModel) CostPerRecord(classification string, containsPII bool) float64 {
	cl := strings.ToLower(classification)

	// If not PII and not a regulated classification, use internal/public rates.
	if !containsPII {
		switch cl {
		case "restricted", "confidential":
			return costPerRecordInternal
		case "internal":
			return costPerRecordInternal
		default:
			return costPerRecordPublic
		}
	}

	// PII-containing data: use industry-standard rates.
	// The classification helps distinguish healthcare/financial from general PII.
	switch cl {
	case "restricted":
		// Restricted PII often indicates healthcare or highly regulated data.
		return costPerRecordHealthcare
	case "confidential":
		// Confidential PII often indicates financial or sensitive business data.
		return costPerRecordFinancial
	case "internal":
		return costPerRecordGeneralPII
	case "public":
		return costPerRecordPublic
	default:
		return costPerRecordGeneralPII
	}
}

// CalculateBreakdown produces a detailed cost breakdown for a potential data
// breach of the given asset. Components include notification, investigation,
// regulatory fines, legal costs, reputation damage, and business disruption.
func (m *BreachCostModel) CalculateBreakdown(asset *cybermodel.DSPMDataAsset, annualRevenue float64) model.CostBreakdown {
	recordCount := estimateRecordCount(asset)
	records := float64(recordCount)

	breakdown := model.CostBreakdown{}

	// 1. Notification cost: $2.50 per record (postage, credit monitoring, call centers).
	breakdown.NotificationCost = notificationCostPerRecord * records

	// 2. Investigation cost: $50K base + $0.50 per record (forensics, IR team, tools).
	breakdown.InvestigationCost = investigationBaseFixed + (investigationCostPerRecord * records)

	// 3. Regulatory fines: maximum across applicable frameworks.
	breakdown.RegulatoryFine = MaxFineForAsset(asset, annualRevenue)

	// 4. Legal cost: $1.50 per record (lawsuits, settlements, counsel).
	breakdown.LegalCost = legalCostPerRecord * records

	// 5. Reputation cost: 3% of annual revenue.
	if annualRevenue > 0 {
		breakdown.ReputationCost = annualRevenue * reputationRevenueMultiplier
	}

	// 6. Business disruption: based on network exposure level.
	breakdown.BusinessDisruption = businessDisruptionCost(asset)

	m.logger.Debug().
		Str("asset_id", asset.AssetID.String()).
		Int64("record_count", recordCount).
		Float64("total_cost", breakdown.Total()).
		Msg("breach cost breakdown calculated")

	return breakdown
}

// estimateRecordCount returns the estimated record count for an asset,
// defaulting to 10,000 if not available.
func estimateRecordCount(asset *cybermodel.DSPMDataAsset) int64 {
	if asset.EstimatedRecordCount != nil && *asset.EstimatedRecordCount > 0 {
		return *asset.EstimatedRecordCount
	}
	// Default estimate when actual count is unknown.
	return 10_000
}

// businessDisruptionCost returns the estimated business disruption cost
// based on the asset's network exposure level.
func businessDisruptionCost(asset *cybermodel.DSPMDataAsset) float64 {
	if asset.NetworkExposure == nil {
		return disruptionDefault
	}

	switch strings.ToLower(*asset.NetworkExposure) {
	case "internet_facing", "internet-facing", "public":
		return disruptionInternetFacing
	case "vpn", "vpn_accessible", "vpn-accessible":
		return disruptionVPN
	case "internal", "private":
		return disruptionInternal
	default:
		return disruptionDefault
	}
}
