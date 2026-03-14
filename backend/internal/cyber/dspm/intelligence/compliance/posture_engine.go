package compliance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// AssetLister retrieves active data assets for a tenant.
type AssetLister interface {
	ListAllActive(ctx context.Context, tenantID uuid.UUID) ([]*cybermodel.DSPMDataAsset, error)
}

// ComplianceRepository persists and queries compliance posture records.
type ComplianceRepository interface {
	Upsert(ctx context.Context, posture *model.CompliancePosture) error
	GetByFramework(ctx context.Context, tenantID uuid.UUID, framework string) (*model.CompliancePosture, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.CompliancePosture, error)
	ListGaps(ctx context.Context, tenantID uuid.UUID, params *dto.ComplianceGapParams) ([]model.ComplianceGap, int, error)
}

// PostureEngine evaluates compliance posture across all supported frameworks
// by checking each control against all in-scope data assets.
type PostureEngine struct {
	assets  AssetLister
	repo    ComplianceRepository
	configs map[model.ComplianceFramework][]ControlMapping
	logger  zerolog.Logger
}

// NewPostureEngine creates a new compliance posture engine. It pre-loads
// all framework control configurations.
func NewPostureEngine(assets AssetLister, repo ComplianceRepository, logger zerolog.Logger) *PostureEngine {
	configs := make(map[model.ComplianceFramework][]ControlMapping)
	for _, fw := range model.AllFrameworks() {
		mappings := BuildControlMappings(fw)
		if len(mappings) > 0 {
			configs[fw] = mappings
		}
	}

	return &PostureEngine{
		assets:  assets,
		repo:    repo,
		configs: configs,
		logger:  logger.With().Str("component", "posture_engine").Logger(),
	}
}

// Evaluate runs compliance evaluation across all supported frameworks for the
// given tenant. Results are persisted and returned.
func (e *PostureEngine) Evaluate(ctx context.Context, tenantID uuid.UUID) ([]model.CompliancePosture, error) {
	e.logger.Info().Str("tenant_id", tenantID.String()).Msg("starting compliance evaluation")

	var postures []model.CompliancePosture
	for fw := range e.configs {
		posture, err := e.EvaluateFramework(ctx, tenantID, fw)
		if err != nil {
			e.logger.Error().Err(err).
				Str("framework", string(fw)).
				Msg("failed to evaluate framework")
			continue
		}
		postures = append(postures, *posture)
	}

	e.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("frameworks_evaluated", len(postures)).
		Msg("compliance evaluation complete")

	return postures, nil
}

// EvaluateFramework evaluates a single compliance framework against all active
// data assets, computing per-control scores and an overall posture score.
func (e *PostureEngine) EvaluateFramework(ctx context.Context, tenantID uuid.UUID, framework model.ComplianceFramework) (*model.CompliancePosture, error) {
	mappings, ok := e.configs[framework]
	if !ok {
		return nil, fmt.Errorf("unknown framework: %s", framework)
	}

	assets, err := e.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}

	now := time.Now().UTC()

	// Retrieve previous evaluation for trend analysis.
	var previousScore *float64
	if prev, err := e.repo.GetByFramework(ctx, tenantID, string(framework)); err == nil && prev != nil {
		s := prev.OverallScore
		previousScore = &s
	}

	posture := &model.CompliancePosture{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Framework:   framework,
		EvaluatedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	var (
		controlsCompliant    int
		controlsPartial      int
		controlsNonCompliant int
		controlsNA           int
	)

	for _, mapping := range mappings {
		detail := e.evaluateControl(mapping, assets)
		posture.ControlDetails = append(posture.ControlDetails, detail)

		switch detail.Status {
		case model.ControlCompliant:
			controlsCompliant++
		case model.ControlPartial:
			controlsPartial++
		case model.ControlNonCompliant:
			controlsNonCompliant++
		case model.ControlNotApplicable:
			controlsNA++
		}
	}

	posture.ControlsTotal = len(mappings)
	posture.ControlsCompliant = controlsCompliant
	posture.ControlsPartial = controlsPartial
	posture.ControlsNonCompliant = controlsNonCompliant
	posture.ControlsNotApplicable = controlsNA

	// Overall score: (compliant + 0.5*partial) / applicable * 100
	applicable := posture.ControlsTotal - controlsNA
	if applicable > 0 {
		posture.OverallScore = (float64(controlsCompliant) + 0.5*float64(controlsPartial)) / float64(applicable) * 100
	} else {
		posture.OverallScore = 100 // all controls are N/A
	}

	// Trend direction from previous evaluation.
	if previousScore != nil {
		posture.Score7dAgo = previousScore
		diff := posture.OverallScore - *previousScore
		switch {
		case diff > 2:
			posture.TrendDirection = model.TrendImproving
		case diff < -2:
			posture.TrendDirection = model.TrendDeclining
		default:
			posture.TrendDirection = model.TrendStable
		}
	} else {
		posture.TrendDirection = model.TrendStable
	}

	// Estimated fine exposure.
	posture.EstimatedFineExposure = e.estimateFineExposure(framework, posture)
	posture.FineCurrency = fineCurrency(framework)

	// Persist the posture.
	if err := e.repo.Upsert(ctx, posture); err != nil {
		return nil, fmt.Errorf("persisting posture: %w", err)
	}

	e.logger.Info().
		Str("framework", string(framework)).
		Float64("score", posture.OverallScore).
		Int("compliant", controlsCompliant).
		Int("partial", controlsPartial).
		Int("non_compliant", controlsNonCompliant).
		Msg("framework evaluation complete")

	return posture, nil
}

// evaluateControl checks a single control against all in-scope assets.
func (e *PostureEngine) evaluateControl(mapping ControlMapping, assets []*cybermodel.DSPMDataAsset) model.ControlDetail {
	detail := model.ControlDetail{
		ControlID:   mapping.Definition.ControlID,
		Name:        mapping.Definition.Name,
		Description: mapping.Definition.Description,
	}

	var compliantCount, totalInScope int

	for _, asset := range assets {
		if !IsInScope(asset, mapping.Definition.Scope) {
			continue
		}
		totalInScope++

		if mapping.Check(asset) {
			compliantCount++
		} else {
			detail.Gaps = append(detail.Gaps, model.ControlGap{
				AssetID:   asset.AssetID,
				AssetName: asset.AssetName,
				Gap:       fmt.Sprintf("Asset '%s' fails control '%s': %s", asset.AssetName, mapping.Definition.ControlID, mapping.Definition.Description),
			})
		}
	}

	detail.AssetsCompliant = compliantCount
	detail.AssetsNonCompliant = totalInScope - compliantCount
	detail.AssetsTotal = totalInScope

	if totalInScope == 0 {
		detail.Status = model.ControlNotApplicable
		detail.Score = 100
	} else {
		pct := float64(compliantCount) / float64(totalInScope) * 100
		detail.Score = pct
		switch {
		case pct >= 100:
			detail.Status = model.ControlCompliant
		case pct >= 50:
			detail.Status = model.ControlPartial
		default:
			detail.Status = model.ControlNonCompliant
		}
	}

	return detail
}

// estimateFineExposure estimates the regulatory fine exposure based on
// non-compliance level and the framework's fine schedule.
func (e *PostureEngine) estimateFineExposure(framework model.ComplianceFramework, posture *model.CompliancePosture) float64 {
	// Calculate the fraction of controls that are non-compliant.
	applicable := posture.ControlsTotal - posture.ControlsNotApplicable
	if applicable == 0 {
		return 0
	}

	nonCompliantFraction := float64(posture.ControlsNonCompliant) / float64(applicable)
	partialFraction := float64(posture.ControlsPartial) / float64(applicable)

	// Weight: full non-compliance counts fully, partial at 25%.
	riskFraction := nonCompliantFraction + (partialFraction * 0.25)

	// Multiply by the framework's maximum fine.
	maxFine := frameworkMaxFine(framework)
	return maxFine * riskFraction
}

// frameworkMaxFine returns the maximum fine for a framework (in the framework's currency).
func frameworkMaxFine(framework model.ComplianceFramework) float64 {
	switch framework {
	case model.FrameworkGDPR:
		return 20_000_000
	case model.FrameworkHIPAA:
		return 2_130_000
	case model.FrameworkPCIDSS:
		return 1_200_000 // $100K/month x 12
	case model.FrameworkSaudiPDPL:
		return 5_000_000 // SAR
	case model.FrameworkSOC2:
		return 100_000
	case model.FrameworkISO27001:
		return 250_000
	default:
		return 0
	}
}

// fineCurrency returns the currency code for a framework's fine.
func fineCurrency(framework model.ComplianceFramework) string {
	switch framework {
	case model.FrameworkGDPR:
		return "EUR"
	case model.FrameworkSaudiPDPL:
		return "SAR"
	default:
		return "USD"
	}
}
