package financial

import (
	"math"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// highRiskAELThreshold is the annual expected loss threshold above which
// an asset is classified as high-risk in the portfolio.
const highRiskAELThreshold = 50_000.0

// PortfolioAnalyzer aggregates individual asset financial impacts into
// a portfolio-level risk view.
type PortfolioAnalyzer struct {
	logger zerolog.Logger
}

// NewPortfolioAnalyzer creates a new portfolio risk analyzer.
func NewPortfolioAnalyzer(logger zerolog.Logger) *PortfolioAnalyzer {
	return &PortfolioAnalyzer{
		logger: logger.With().Str("component", "portfolio_analyzer").Logger(),
	}
}

// Aggregate combines a slice of per-asset financial impacts into a single
// PortfolioRisk summary, computing totals, maximums, averages, and breakdowns
// by classification and regulation.
func (p *PortfolioAnalyzer) Aggregate(impacts []model.FinancialImpact) *model.PortfolioRisk {
	if len(impacts) == 0 {
		return &model.PortfolioRisk{
			CostByClassification: make(map[string]float64),
			CostByRegulation:     make(map[string]float64),
		}
	}

	portfolio := &model.PortfolioRisk{
		CostByClassification: make(map[string]float64),
		CostByRegulation:     make(map[string]float64),
	}

	var totalProbability float64

	for _, impact := range impacts {
		portfolio.AssetCount++

		// Accumulate total breach cost.
		portfolio.TotalBreachCost += impact.EstimatedBreachCost

		// Accumulate total annual expected loss.
		portfolio.TotalAnnualExpectedLoss += impact.AnnualExpectedLoss

		// Track maximum single-asset exposure.
		portfolio.MaxSingleAssetExposure = math.Max(
			portfolio.MaxSingleAssetExposure,
			impact.EstimatedBreachCost,
		)

		// Accumulate regulatory fines.
		portfolio.TotalRegulatoryFines += impact.MaxRegulatoryFine

		// Track high-risk assets (AEL above threshold).
		if impact.AnnualExpectedLoss >= highRiskAELThreshold {
			portfolio.HighRiskAssetCount++
		}

		// Sum probabilities for averaging.
		totalProbability += impact.BreachProbabilityAnnual

		// Break down costs by regulation.
		for _, reg := range impact.ApplicableRegulations {
			portfolio.CostByRegulation[reg] += impact.EstimatedBreachCost
		}

		// Break down costs by classification.
		// We use the methodology details' industry vertical as a proxy for
		// classification since FinancialImpact doesn't store classification directly.
		classification := impact.MethodologyDetails.IndustryVertical
		if classification == "" {
			classification = "general"
		}
		portfolio.CostByClassification[classification] += impact.EstimatedBreachCost
	}

	// Calculate average breach probability.
	if portfolio.AssetCount > 0 {
		portfolio.AvgBreachProbability = totalProbability / float64(portfolio.AssetCount)
	}

	p.logger.Info().
		Int("asset_count", portfolio.AssetCount).
		Int("high_risk_count", portfolio.HighRiskAssetCount).
		Float64("total_breach_cost", portfolio.TotalBreachCost).
		Float64("total_ael", portfolio.TotalAnnualExpectedLoss).
		Float64("max_exposure", portfolio.MaxSingleAssetExposure).
		Msg("portfolio risk aggregation complete")

	return portfolio
}
