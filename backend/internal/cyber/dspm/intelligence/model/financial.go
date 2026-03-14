package model

import (
	"time"

	"github.com/google/uuid"
)

// CostMethodology identifies the methodology used for breach cost estimation.
type CostMethodology string

const (
	MethodologyIBMPonemon         CostMethodology = "ibm_ponemon"
	MethodologyRegulatorySchedule CostMethodology = "regulatory_schedule"
	MethodologyCustom             CostMethodology = "custom"
)

// CostBreakdown itemizes breach cost components.
type CostBreakdown struct {
	NotificationCost   float64 `json:"notification_cost"`
	InvestigationCost  float64 `json:"investigation_cost"`
	RegulatoryFine     float64 `json:"regulatory_fine"`
	LegalCost          float64 `json:"legal_cost"`
	ReputationCost     float64 `json:"reputation_cost"`
	BusinessDisruption float64 `json:"business_disruption"`
}

// Total returns the sum of all cost components.
func (c CostBreakdown) Total() float64 {
	return c.NotificationCost + c.InvestigationCost + c.RegulatoryFine +
		c.LegalCost + c.ReputationCost + c.BusinessDisruption
}

// MethodologyDetails explains how the cost was calculated.
type MethodologyDetails struct {
	Source           string  `json:"source"`
	IndustryVertical string  `json:"industry_vertical,omitempty"`
	BaseRate         float64 `json:"base_rate"`
	AdjustmentFactor float64 `json:"adjustment_factor,omitempty"`
	Notes            string  `json:"notes,omitempty"`
}

// FinancialImpact is the per-asset financial risk quantification.
type FinancialImpact struct {
	ID                      uuid.UUID          `json:"id"`
	TenantID                uuid.UUID          `json:"tenant_id"`
	DataAssetID             uuid.UUID          `json:"data_asset_id"`
	EstimatedBreachCost     float64            `json:"estimated_breach_cost"`
	CostPerRecord           float64            `json:"cost_per_record"`
	RecordCount             int64              `json:"record_count"`
	Breakdown               CostBreakdown      `json:"cost_breakdown"`
	Methodology             CostMethodology    `json:"methodology"`
	MethodologyDetails      MethodologyDetails `json:"methodology_details"`
	ApplicableRegulations   []string           `json:"applicable_regulations"`
	MaxRegulatoryFine       float64            `json:"max_regulatory_fine"`
	BreachProbabilityAnnual float64            `json:"breach_probability_annual"`
	AnnualExpectedLoss      float64            `json:"annual_expected_loss"`
	CalculatedAt            time.Time          `json:"calculated_at"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
}

// PortfolioRisk aggregates financial risk across all assets.
type PortfolioRisk struct {
	TotalBreachCost         float64            `json:"total_breach_cost"`
	TotalAnnualExpectedLoss float64            `json:"total_annual_expected_loss"`
	MaxSingleAssetExposure  float64            `json:"max_single_asset_exposure"`
	TotalRegulatoryFines    float64            `json:"total_regulatory_fines"`
	AssetCount              int                `json:"asset_count"`
	HighRiskAssetCount      int                `json:"high_risk_asset_count"`
	AvgBreachProbability    float64            `json:"avg_breach_probability"`
	CostByClassification    map[string]float64 `json:"cost_by_classification"`
	CostByRegulation        map[string]float64 `json:"cost_by_regulation"`
}

// RegulatoryFineSchedule defines the fine structure for a regulation.
type RegulatoryFineSchedule struct {
	Framework       string  `json:"framework"`
	MaxFine         float64 `json:"max_fine"`
	MaxFineCurrency string  `json:"max_fine_currency"`
	PerViolationMin float64 `json:"per_violation_min"`
	PerViolationMax float64 `json:"per_violation_max"`
	RevenuePercent  float64 `json:"revenue_percent,omitempty"`
	Description     string  `json:"description"`
}
