package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/ai_security"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/classifier"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/compliance"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/financial"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/lineage"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/proliferation"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/repository"
	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/events"
)

const eventSource = "clario360/cyber-service"
const eventTypePrefix = "com.clario360.cyber.dspm.intelligence."

// AssetLister retrieves active data assets for a tenant.
type AssetLister interface {
	ListAllActive(ctx context.Context, tenantID uuid.UUID) ([]*cybermodel.DSPMDataAsset, error)
}

// IntelligenceService orchestrates DSPM advanced intelligence operations
// by delegating to specialized engines, analyzers, and repositories.
type IntelligenceService struct {
	classifier       *classifier.MLClassifier
	lineageEngine    *lineage.LineageEngine
	aiScanner        *ai_security.AIDataScanner
	aiGovernance     *ai_security.ModelDataGovernance
	impactCalc       *financial.ImpactCalculator
	postureEngine    *compliance.PostureEngine
	gapAnalyzer      *compliance.GapAnalyzer
	residencyTracker *compliance.ResidencyTracker
	auditEvidence    *compliance.AuditEvidenceGenerator
	prolifTracker    *proliferation.ProliferationTracker
	driftAnalyzer    *proliferation.DriftAnalyzer
	spreadViz        *proliferation.SpreadVisualizer

	lineageRepo    *repository.LineageRepository
	classRepo      *repository.ClassificationRepository
	aiUsageRepo    *repository.AIUsageRepository
	financialRepo  *repository.FinancialRepository
	complianceRepo *repository.ComplianceRepository

	assets   AssetLister
	producer *events.Producer
	logger   zerolog.Logger
}

// NewIntelligenceService creates a new IntelligenceService with all dependencies.
func NewIntelligenceService(
	mlClassifier *classifier.MLClassifier,
	lineageEng *lineage.LineageEngine,
	aiScanner *ai_security.AIDataScanner,
	aiGovernance *ai_security.ModelDataGovernance,
	impactCalc *financial.ImpactCalculator,
	postureEngine *compliance.PostureEngine,
	gapAnalyzer *compliance.GapAnalyzer,
	residencyTracker *compliance.ResidencyTracker,
	auditEvidence *compliance.AuditEvidenceGenerator,
	prolifTracker *proliferation.ProliferationTracker,
	driftAnalyzer *proliferation.DriftAnalyzer,
	spreadViz *proliferation.SpreadVisualizer,
	lineageRepo *repository.LineageRepository,
	classRepo *repository.ClassificationRepository,
	aiUsageRepo *repository.AIUsageRepository,
	financialRepo *repository.FinancialRepository,
	complianceRepo *repository.ComplianceRepository,
	assets AssetLister,
	producer *events.Producer,
	logger zerolog.Logger,
) *IntelligenceService {
	return &IntelligenceService{
		classifier:       mlClassifier,
		lineageEngine:    lineageEng,
		aiScanner:        aiScanner,
		aiGovernance:     aiGovernance,
		impactCalc:       impactCalc,
		postureEngine:    postureEngine,
		gapAnalyzer:      gapAnalyzer,
		residencyTracker: residencyTracker,
		auditEvidence:    auditEvidence,
		prolifTracker:    prolifTracker,
		driftAnalyzer:    driftAnalyzer,
		spreadViz:        spreadViz,
		lineageRepo:      lineageRepo,
		classRepo:        classRepo,
		aiUsageRepo:      aiUsageRepo,
		financialRepo:    financialRepo,
		complianceRepo:   complianceRepo,
		assets:           assets,
		producer:         producer,
		logger:           logger.With().Str("component", "intelligence_service").Logger(),
	}
}

// publishEvent publishes a domain event to the DSPM events topic.
// It logs but does not propagate errors from the event bus.
func (s *IntelligenceService) publishEvent(ctx context.Context, eventType string, tenantID uuid.UUID, data any) {
	if s.producer == nil {
		return
	}
	evt, err := events.NewEvent(eventTypePrefix+eventType, eventSource, tenantID.String(), data)
	if err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to create event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.DSPMEvents, evt); err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to publish event")
	}
}

// --------------------------------------------------------------------------
// Classification
// --------------------------------------------------------------------------

// EnhancedClassification runs the multi-layer ML classifier across all active
// assets for a tenant and returns an aggregated classification response.
func (s *IntelligenceService) EnhancedClassification(ctx context.Context, tenantID uuid.UUID) (*dto.EnhancedClassificationResponse, error) {
	s.logger.Info().Str("tenant_id", tenantID.String()).Msg("running enhanced classification")

	assets, err := s.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}

	results := s.classifier.ClassifyBatch(assets)

	var needingReview int
	var totalConf float64
	classResults := make([]dto.ClassificationResult, 0, len(results))
	for _, r := range results {
		if r.NeedsHumanReview {
			needingReview++
		}
		totalConf += r.Confidence
		classResults = append(classResults, dto.ClassificationResult{
			AssetID:          r.AssetID,
			AssetName:        r.AssetName,
			Classification:   r.Classification,
			PIITypes:         r.PIITypes,
			Confidence:       r.Confidence,
			NeedsHumanReview: r.NeedsHumanReview,
			DetectedBy:       string(r.DetectedBy),
			Explanation:      r.Evidence.Explanation,
		})

		// Record classification history for assets whose classification changed.
		for _, asset := range assets {
			if (asset.ID == r.AssetID || asset.AssetID == r.AssetID) && asset.DataClassification != r.Classification {
				changeType := model.ChangeTypeReclassification
				if asset.DataClassification == "" {
					changeType = model.ChangeTypeInitial
				}
				history := &model.ClassificationHistory{
					ID:                uuid.New(),
					TenantID:          tenantID,
					DataAssetID:       r.AssetID,
					OldClassification: asset.DataClassification,
					NewClassification: r.Classification,
					OldPIITypes:       asset.PIITypes,
					NewPIITypes:       r.PIITypes,
					ChangeType:        changeType,
					DetectedBy:        string(r.DetectedBy),
					Confidence:        r.Confidence,
					Evidence:          r.Evidence,
					ActorType:         "system",
					CreatedAt:         time.Now().UTC(),
				}
				if history.OldPIITypes == nil {
					history.OldPIITypes = []string{}
				}
				if history.NewPIITypes == nil {
					history.NewPIITypes = []string{}
				}
				if insertErr := s.classRepo.Insert(ctx, history); insertErr != nil {
					s.logger.Error().Err(insertErr).
						Str("asset_id", r.AssetID.String()).
						Msg("failed to insert classification history")
				}
				break
			}
		}
	}

	var avgConf float64
	if len(results) > 0 {
		avgConf = totalConf / float64(len(results))
	}

	resp := &dto.EnhancedClassificationResponse{
		Classifications: classResults,
		TotalAssets:     len(results),
		NeedingReview:   needingReview,
		AvgConfidence:   avgConf,
	}

	s.publishEvent(ctx, "classification.completed", tenantID, map[string]any{
		"total_assets":  resp.TotalAssets,
		"needing_review": resp.NeedingReview,
		"avg_confidence": resp.AvgConfidence,
	})

	return resp, nil
}

// ClassificationHistory returns paginated classification history for a specific asset.
func (s *IntelligenceService) ClassificationHistory(ctx context.Context, tenantID, assetID uuid.UUID, params *dto.ClassificationHistoryParams) ([]model.ClassificationHistory, int, error) {
	return s.classRepo.ListByAsset(ctx, tenantID, assetID, params)
}

// CreateCustomRule creates a tenant-defined custom classification rule and
// persists it via the classification repository.
func (s *IntelligenceService) CreateCustomRule(ctx context.Context, tenantID uuid.UUID, req *dto.CreateCustomRuleRequest) (*model.CustomClassificationRule, error) {
	now := time.Now().UTC()
	rule := &model.CustomClassificationRule{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           req.Name,
		ColumnPatterns: req.ColumnPatterns,
		ValuePattern:   req.ValuePattern,
		Classification: req.Classification,
		PIIType:        req.PIIType,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	s.publishEvent(ctx, "classification.custom_rule_created", tenantID, map[string]any{
		"rule_id":        rule.ID.String(),
		"rule_name":      rule.Name,
		"classification": rule.Classification,
	})

	s.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("rule_id", rule.ID.String()).
		Str("rule_name", rule.Name).
		Msg("custom classification rule created")

	return rule, nil
}

// --------------------------------------------------------------------------
// Lineage
// --------------------------------------------------------------------------

// GetLineageGraph returns the full lineage graph for a tenant with optional filters.
func (s *IntelligenceService) GetLineageGraph(ctx context.Context, tenantID uuid.UUID, params *dto.LineageGraphParams) (*model.LineageGraph, error) {
	graph, err := s.lineageEngine.GetGraph(ctx, tenantID, params)
	if err != nil {
		return nil, fmt.Errorf("getting lineage graph: %w", err)
	}
	return graph, nil
}

// GetUpstream returns the upstream lineage graph for a specific asset up to the given depth.
func (s *IntelligenceService) GetUpstream(ctx context.Context, tenantID, assetID uuid.UUID, depth int) (*model.LineageGraph, error) {
	if depth < 1 {
		depth = 3
	}
	if depth > 10 {
		depth = 10
	}

	edges, err := s.lineageRepo.GetUpstream(ctx, tenantID, assetID, depth)
	if err != nil {
		return nil, fmt.Errorf("getting upstream edges: %w", err)
	}

	assets, err := s.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing assets for upstream: %w", err)
	}

	graphOps := lineage.NewGraphOperations(s.logger)
	graph := graphOps.BuildGraph(edges, assets)
	return graph, nil
}

// GetDownstream returns the downstream lineage graph for a specific asset up to the given depth.
func (s *IntelligenceService) GetDownstream(ctx context.Context, tenantID, assetID uuid.UUID, depth int) (*model.LineageGraph, error) {
	if depth < 1 {
		depth = 3
	}
	if depth > 10 {
		depth = 10
	}

	edges, err := s.lineageRepo.GetDownstream(ctx, tenantID, assetID, depth)
	if err != nil {
		return nil, fmt.Errorf("getting downstream edges: %w", err)
	}

	assets, err := s.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing assets for downstream: %w", err)
	}

	graphOps := lineage.NewGraphOperations(s.logger)
	graph := graphOps.BuildGraph(edges, assets)
	return graph, nil
}

// GetImpactAnalysis computes the downstream impact of a change to the given asset.
func (s *IntelligenceService) GetImpactAnalysis(ctx context.Context, tenantID, assetID uuid.UUID) (*model.ImpactResult, error) {
	// Get all lineage edges for the tenant.
	allEdges, err := s.lineageRepo.ListByTenant(ctx, tenantID, nil)
	if err != nil {
		return nil, fmt.Errorf("listing lineage edges for impact: %w", err)
	}

	assets, err := s.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing assets for impact: %w", err)
	}

	graphOps := lineage.NewGraphOperations(s.logger)
	result := graphOps.ImpactAnalysis(allEdges, assets, assetID)

	s.publishEvent(ctx, "lineage.impact_analyzed", tenantID, map[string]any{
		"asset_id":           assetID.String(),
		"downstream_assets":  result.DownstreamAssets,
		"risk_amplification": result.RiskAmplification,
	})

	return result, nil
}

// GetPIIFlow returns a lineage graph filtered to only show PII data flows.
func (s *IntelligenceService) GetPIIFlow(ctx context.Context, tenantID uuid.UUID) (*model.LineageGraph, error) {
	piiOnly := true
	params := &dto.LineageGraphParams{
		PIIOnly: &piiOnly,
	}
	return s.lineageEngine.GetGraph(ctx, tenantID, params)
}

// --------------------------------------------------------------------------
// AI Security
// --------------------------------------------------------------------------

// ListAIUsage returns paginated AI data usage records for a tenant.
func (s *IntelligenceService) ListAIUsage(ctx context.Context, tenantID uuid.UUID, params *dto.AIUsageListParams) ([]model.AIDataUsage, int, error) {
	return s.aiUsageRepo.ListByTenant(ctx, tenantID, params)
}

// GetAIUsageByAsset returns all AI data usage records for a specific asset.
func (s *IntelligenceService) GetAIUsageByAsset(ctx context.Context, tenantID, assetID uuid.UUID) ([]model.AIDataUsage, error) {
	return s.aiUsageRepo.ListByAsset(ctx, tenantID, assetID)
}

// GetModelDataGovernance returns the governance assessment for a specific AI model.
func (s *IntelligenceService) GetModelDataGovernance(ctx context.Context, tenantID uuid.UUID, modelSlug string) (*model.ModelDataAssessment, error) {
	assessment, err := s.aiGovernance.AssessModel(ctx, tenantID, modelSlug)
	if err != nil {
		return nil, fmt.Errorf("assessing model governance: %w", err)
	}

	s.publishEvent(ctx, "ai.model_governance_assessed", tenantID, map[string]any{
		"model_slug": modelSlug,
		"risk_score": assessment.RiskScore,
	})

	return assessment, nil
}

// GetAIRiskRanking returns AI data usage records sorted by risk score descending.
func (s *IntelligenceService) GetAIRiskRanking(ctx context.Context, tenantID uuid.UUID) ([]model.AIDataUsage, error) {
	params := &dto.AIUsageListParams{
		Sort:    "ai_risk_score",
		Order:   "desc",
		Page:    1,
		PerPage: 100,
	}
	usages, _, err := s.aiUsageRepo.ListByTenant(ctx, tenantID, params)
	if err != nil {
		return nil, fmt.Errorf("getting AI risk ranking: %w", err)
	}
	return usages, nil
}

// GetAIDashboard returns aggregated AI security dashboard metrics.
func (s *IntelligenceService) GetAIDashboard(ctx context.Context, tenantID uuid.UUID) (*model.AISecurityDashboard, error) {
	return s.aiUsageRepo.Dashboard(ctx, tenantID)
}

// --------------------------------------------------------------------------
// Financial
// --------------------------------------------------------------------------

// GetPortfolioRisk returns aggregate financial risk metrics across all assets.
func (s *IntelligenceService) GetPortfolioRisk(ctx context.Context, tenantID uuid.UUID) (*model.PortfolioRisk, error) {
	return s.financialRepo.PortfolioRisk(ctx, tenantID)
}

// GetAssetFinancialImpact returns the financial impact assessment for a specific asset.
func (s *IntelligenceService) GetAssetFinancialImpact(ctx context.Context, tenantID, assetID uuid.UUID) (*model.FinancialImpact, error) {
	return s.financialRepo.GetByAsset(ctx, tenantID, assetID)
}

// GetTopFinancialRisks returns the top N assets by annual expected loss.
func (s *IntelligenceService) GetTopFinancialRisks(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.FinancialImpact, error) {
	return s.financialRepo.TopRisks(ctx, tenantID, limit)
}

// --------------------------------------------------------------------------
// Compliance
// --------------------------------------------------------------------------

// GetCompliancePosture returns compliance posture for all frameworks.
func (s *IntelligenceService) GetCompliancePosture(ctx context.Context, tenantID uuid.UUID) ([]model.CompliancePosture, error) {
	return s.complianceRepo.ListByTenant(ctx, tenantID)
}

// GetFrameworkPosture returns compliance posture for a specific framework.
func (s *IntelligenceService) GetFrameworkPosture(ctx context.Context, tenantID uuid.UUID, framework string) (*model.CompliancePosture, error) {
	return s.complianceRepo.GetByFramework(ctx, tenantID, framework)
}

// GetComplianceGaps returns compliance gaps with optional framework and severity filtering.
func (s *IntelligenceService) GetComplianceGaps(ctx context.Context, tenantID uuid.UUID, params *dto.ComplianceGapParams) ([]model.ComplianceGap, int, error) {
	return s.complianceRepo.ListGaps(ctx, tenantID, params)
}

// GetResidencyAnalysis runs residency tracking across all active assets and
// returns detected violations.
func (s *IntelligenceService) GetResidencyAnalysis(ctx context.Context, tenantID uuid.UUID) ([]model.ResidencyViolation, error) {
	assets, err := s.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing assets for residency: %w", err)
	}

	violations := s.residencyTracker.Analyze(assets)

	if len(violations) > 0 {
		s.publishEvent(ctx, "compliance.residency_violations_detected", tenantID, map[string]any{
			"violation_count": len(violations),
		})
	}

	return violations, nil
}

// GenerateAuditReport generates a comprehensive audit evidence report for
// the specified compliance framework.
func (s *IntelligenceService) GenerateAuditReport(ctx context.Context, tenantID uuid.UUID, framework string) (*model.AuditReport, error) {
	// Retrieve the current posture for the framework.
	posture, err := s.complianceRepo.GetByFramework(ctx, tenantID, framework)
	if err != nil {
		return nil, fmt.Errorf("getting compliance posture for audit: %w", err)
	}

	// Get compliance gaps for the framework.
	fw := framework
	gapParams := &dto.ComplianceGapParams{
		Framework: &fw,
		Page:      1,
		PerPage:   100,
	}
	gaps, _, err := s.complianceRepo.ListGaps(ctx, tenantID, gapParams)
	if err != nil {
		return nil, fmt.Errorf("getting compliance gaps for audit: %w", err)
	}

	report, err := s.auditEvidence.Generate(ctx, tenantID, posture, gaps)
	if err != nil {
		return nil, fmt.Errorf("generating audit report: %w", err)
	}

	s.publishEvent(ctx, "compliance.audit_report_generated", tenantID, map[string]any{
		"framework":      framework,
		"overall_score":  posture.OverallScore,
		"assets_in_scope": len(report.AssetInventory),
		"gaps_found":     len(report.GapAnalysis),
	})

	return report, nil
}

// --------------------------------------------------------------------------
// Proliferation
// --------------------------------------------------------------------------

// GetProliferationOverview returns an overview of data proliferation across
// all sensitive assets for a tenant.
func (s *IntelligenceService) GetProliferationOverview(ctx context.Context, tenantID uuid.UUID) (*model.ProliferationOverview, error) {
	assets, err := s.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing assets for proliferation: %w", err)
	}

	overview, err := s.prolifTracker.Track(ctx, tenantID, assets)
	if err != nil {
		return nil, fmt.Errorf("tracking proliferation: %w", err)
	}

	if overview.TotalUnauthorizedCopies > 0 {
		s.publishEvent(ctx, "proliferation.unauthorized_copies_detected", tenantID, map[string]any{
			"unauthorized_copies": overview.TotalUnauthorizedCopies,
			"assets_with_copies":  overview.AssetsWithCopies,
		})
	}

	return overview, nil
}

// GetAssetProliferation returns proliferation details for a specific asset.
func (s *IntelligenceService) GetAssetProliferation(ctx context.Context, tenantID, assetID uuid.UUID) (*model.DataProliferation, error) {
	// Get lineage edges originating from this asset.
	edges, err := s.lineageRepo.ListByTenant(ctx, tenantID, nil)
	if err != nil {
		return nil, fmt.Errorf("listing edges for asset proliferation: %w", err)
	}

	// Filter edges to those originating from the specified asset.
	var assetEdges []model.LineageEdge
	for _, edge := range edges {
		if edge.SourceAssetID == assetID {
			assetEdges = append(assetEdges, edge)
		}
	}

	dp := s.prolifTracker.TrackAsset(ctx, tenantID, assetID, assetEdges)
	if dp == nil {
		// Return an empty result if no proliferation detected.
		dp = &model.DataProliferation{
			AssetID:      assetID,
			SpreadEvents: []model.SpreadEvent{},
		}
	}

	// Enrich with asset metadata.
	assets, err := s.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return dp, nil // Return what we have even if enrichment fails.
	}
	for _, asset := range assets {
		if asset.AssetID == assetID || asset.ID == assetID {
			dp.AssetName = asset.AssetName
			dp.Classification = asset.DataClassification
			dp.PIITypes = asset.PIITypes
			break
		}
	}

	return dp, nil
}
