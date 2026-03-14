package financial

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// Breach probability base rates by network exposure.
const (
	baseProbInternetFacing = 0.10
	baseProbVPN            = 0.03
	baseProbInternal       = 0.01
	baseProbDefault        = 0.02
)

// AssetLister retrieves active data assets for a tenant.
type AssetLister interface {
	ListAllActive(ctx context.Context, tenantID uuid.UUID) ([]*cybermodel.DSPMDataAsset, error)
}

// FinancialRepository persists and queries financial impact records.
type FinancialRepository interface {
	Upsert(ctx context.Context, impact *model.FinancialImpact) error
	GetByAsset(ctx context.Context, tenantID, assetID uuid.UUID) (*model.FinancialImpact, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, params *dto.FinancialImpactListParams) ([]model.FinancialImpact, int, error)
	PortfolioRisk(ctx context.Context, tenantID uuid.UUID) (*model.PortfolioRisk, error)
	TopRisks(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.FinancialImpact, error)
}

// ImpactCalculator quantifies the financial impact of potential data breaches
// across all data assets, combining per-record costs, regulatory fines, and
// probability-weighted expected losses.
type ImpactCalculator struct {
	assets    AssetLister
	repo      FinancialRepository
	costModel *BreachCostModel
	logger    zerolog.Logger
}

// NewImpactCalculator creates a new financial impact calculator.
func NewImpactCalculator(assets AssetLister, repo FinancialRepository, logger zerolog.Logger) *ImpactCalculator {
	return &ImpactCalculator{
		assets:    assets,
		repo:      repo,
		costModel: NewBreachCostModel(logger),
		logger:    logger.With().Str("component", "impact_calculator").Logger(),
	}
}

// Calculate computes financial impact for all active assets belonging to the
// tenant and persists the results.
func (c *ImpactCalculator) Calculate(ctx context.Context, tenantID uuid.UUID) error {
	c.logger.Info().Str("tenant_id", tenantID.String()).Msg("starting financial impact calculation")

	assets, err := c.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("listing assets: %w", err)
	}

	// Use a default annual revenue; in production this would come from tenant config.
	annualRevenue := extractAnnualRevenue(assets)

	calculated := 0
	for _, asset := range assets {
		impact := c.CalculateForAsset(asset, annualRevenue)
		impact.TenantID = tenantID

		if err := c.repo.Upsert(ctx, impact); err != nil {
			c.logger.Error().Err(err).
				Str("asset_id", asset.AssetID.String()).
				Msg("failed to persist financial impact")
			continue
		}
		calculated++
	}

	c.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("assets_calculated", calculated).
		Int("total_assets", len(assets)).
		Msg("financial impact calculation complete")

	return nil
}

// CalculateForAsset computes the full financial impact for a single data asset.
//
// Breach probability = base_probability x exposure_factor x (1 - posture_score/100)
// Annual expected loss = total_breach_cost x breach_probability
func (c *ImpactCalculator) CalculateForAsset(asset *cybermodel.DSPMDataAsset, annualRevenue float64) *model.FinancialImpact {
	now := time.Now().UTC()

	// Calculate cost breakdown.
	breakdown := c.costModel.CalculateBreakdown(asset, annualRevenue)
	totalBreachCost := breakdown.Total()

	// Calculate per-record cost.
	perRecord := c.costModel.CostPerRecord(asset.DataClassification, asset.ContainsPII)

	// Record count.
	recordCount := estimateRecordCount(asset)

	// Breach probability calculation.
	baseProb := baseProbability(asset)
	exposureFactor := exposureAdjustment(asset)
	postureDiscount := 1.0 - (asset.PostureScore / 100.0)
	if postureDiscount < 0.05 {
		postureDiscount = 0.05 // minimum 5% residual probability
	}
	breachProb := baseProb * exposureFactor * postureDiscount

	// Clamp probability to [0, 1].
	if breachProb > 1.0 {
		breachProb = 1.0
	}
	if breachProb < 0 {
		breachProb = 0
	}

	// Annual expected loss.
	ael := totalBreachCost * breachProb

	// Determine applicable regulations.
	regulations := determineRegulations(asset)

	impact := &model.FinancialImpact{
		ID:                      uuid.New(),
		DataAssetID:             asset.AssetID,
		EstimatedBreachCost:     totalBreachCost,
		CostPerRecord:           perRecord,
		RecordCount:             recordCount,
		Breakdown:               breakdown,
		Methodology:             model.MethodologyIBMPonemon,
		MethodologyDetails:      buildMethodologyDetails(asset, perRecord),
		ApplicableRegulations:   regulations,
		MaxRegulatoryFine:       MaxFineForAsset(asset, annualRevenue),
		BreachProbabilityAnnual: breachProb,
		AnnualExpectedLoss:      ael,
		CalculatedAt:            now,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	c.logger.Debug().
		Str("asset_id", asset.AssetID.String()).
		Float64("breach_cost", totalBreachCost).
		Float64("breach_prob", breachProb).
		Float64("ael", ael).
		Msg("financial impact calculated for asset")

	return impact
}

// baseProbability returns the base annual breach probability based on
// network exposure.
func baseProbability(asset *cybermodel.DSPMDataAsset) float64 {
	if asset.NetworkExposure == nil {
		return baseProbDefault
	}
	switch strings.ToLower(*asset.NetworkExposure) {
	case "internet_facing", "internet-facing", "public":
		return baseProbInternetFacing
	case "vpn", "vpn_accessible", "vpn-accessible":
		return baseProbVPN
	case "internal", "private":
		return baseProbInternal
	default:
		return baseProbDefault
	}
}

// exposureAdjustment returns a multiplier based on asset exposure characteristics.
func exposureAdjustment(asset *cybermodel.DSPMDataAsset) float64 {
	factor := 1.0

	// No encryption at rest increases exposure.
	if asset.EncryptedAtRest != nil && !*asset.EncryptedAtRest {
		factor *= 1.5
	}

	// No encryption in transit increases exposure.
	if asset.EncryptedInTransit != nil && !*asset.EncryptedInTransit {
		factor *= 1.3
	}

	// Weak access control increases exposure.
	if asset.AccessControlType != nil {
		act := strings.ToLower(*asset.AccessControlType)
		if act == "none" || act == "" {
			factor *= 1.5
		}
	}

	// No audit logging increases exposure.
	if asset.AuditLogging != nil && !*asset.AuditLogging {
		factor *= 1.2
	}

	return factor
}

// determineRegulations returns the list of applicable regulation identifiers
// for an asset.
func determineRegulations(asset *cybermodel.DSPMDataAsset) []string {
	var regs []string

	if asset.ContainsPII {
		regs = append(regs, "GDPR")
	}
	if isHealthcareData(asset) {
		regs = append(regs, "HIPAA")
	}
	if isPaymentData(asset) {
		regs = append(regs, "PCI DSS")
	}
	if isSaudiData(asset) {
		regs = append(regs, "Saudi PDPL")
	}
	if asset.DataClassification != "public" {
		regs = append(regs, "SOC 2", "ISO 27001")
	}

	return regs
}

// buildMethodologyDetails constructs the methodology details for an impact record.
func buildMethodologyDetails(asset *cybermodel.DSPMDataAsset, baseRate float64) model.MethodologyDetails {
	industry := "general"
	if isHealthcareData(asset) {
		industry = "healthcare"
	} else if isPaymentData(asset) {
		industry = "financial"
	}

	return model.MethodologyDetails{
		Source:           "IBM/Ponemon Cost of a Data Breach Report 2024",
		IndustryVertical: industry,
		BaseRate:         baseRate,
		AdjustmentFactor: 1.0,
		Notes:            "Per-record cost adjusted by data classification and PII status; breach probability based on network exposure and security posture",
	}
}

// extractAnnualRevenue attempts to extract annual revenue from the first asset
// that has it in metadata; defaults to $10M if not found.
func extractAnnualRevenue(assets []*cybermodel.DSPMDataAsset) float64 {
	for _, asset := range assets {
		if asset.Metadata == nil {
			continue
		}
		if rev, ok := asset.Metadata["annual_revenue"]; ok {
			switch v := rev.(type) {
			case float64:
				if v > 0 {
					return v
				}
			case int:
				if v > 0 {
					return float64(v)
				}
			}
		}
	}
	// Default assumption for financial modeling.
	return 10_000_000
}
