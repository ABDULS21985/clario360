package seeder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
)

type ModelSeeder struct {
	repo   *repository.ModelRegistryRepository
	logger zerolog.Logger
}

func NewModelSeeder(pool *pgxpool.Pool, logger zerolog.Logger) *ModelSeeder {
	return &ModelSeeder{
		repo:   repository.NewModelRegistryRepository(pool, logger),
		logger: logger.With().Str("component", "ai_model_seeder").Logger(),
	}
}

func (s *ModelSeeder) Seed(ctx context.Context, tenantID, createdBy uuid.UUID) error {
	for _, spec := range defaultModels() {
		existing, err := s.repo.GetModelBySlug(ctx, tenantID, spec.Slug)
		if err != nil && err != repository.ErrNotFound {
			return err
		}
		if existing == nil {
			now := time.Now().UTC()
			existing = &aigovmodel.RegisteredModel{
				ID:          uuid.New(),
				TenantID:    tenantID,
				Name:        spec.Name,
				Slug:        spec.Slug,
				Description: spec.Description,
				ModelType:   spec.ModelType,
				Suite:       spec.Suite,
				OwnerUserID: &createdBy,
				OwnerTeam:   spec.OwnerTeam,
				RiskTier:    spec.RiskTier,
				Status:      aigovmodel.ModelStatusActive,
				Tags:        spec.Tags,
				Metadata:    mustJSON(spec.Metadata),
				CreatedBy:   createdBy,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := s.repo.CreateModel(ctx, existing); err != nil {
				return fmt.Errorf("seed ai model %s: %w", spec.Slug, err)
			}
		}
		versions, err := s.repo.ListVersions(ctx, tenantID, existing.ID)
		if err != nil {
			return err
		}
		if len(versions) > 0 {
			continue
		}
		hash, err := hashConfig(spec.ArtifactConfig)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		version := &aigovmodel.ModelVersion{
			ID:                    uuid.New(),
			TenantID:              tenantID,
			ModelID:               existing.ID,
			ModelSlug:             existing.Slug,
			ModelName:             existing.Name,
			ModelType:             existing.ModelType,
			ModelSuite:            existing.Suite,
			ModelRiskTier:         existing.RiskTier,
			VersionNumber:         1,
			Status:                aigovmodel.VersionStatusProduction,
			Description:           "Initial seeded production version",
			ArtifactType:          spec.ArtifactType,
			ArtifactConfig:        mustJSON(spec.ArtifactConfig),
			ArtifactHash:          hash,
			ExplainabilityType:    spec.ExplainabilityType,
			ExplanationTemplate:   stringPtr(spec.Template),
			TrainingMetrics:       json.RawMessage(`{}`),
			PromotedToProductionAt: &now,
			PromotedBy:            &createdBy,
			CreatedBy:             createdBy,
			CreatedAt:             now,
			UpdatedAt:             now,
		}
		if err := s.repo.CreateVersion(ctx, version); err != nil {
			return fmt.Errorf("seed ai model version %s: %w", spec.Slug, err)
		}
	}
	return nil
}

type modelSpec struct {
	Name               string
	Slug               string
	Description        string
	ModelType          aigovmodel.ModelType
	Suite              aigovmodel.ModelSuite
	RiskTier           aigovmodel.RiskTier
	ArtifactType       aigovmodel.ArtifactType
	ExplainabilityType aigovmodel.ExplainabilityType
	ArtifactConfig     map[string]any
	Template           string
	OwnerTeam          string
	Tags               []string
	Metadata           map[string]any
}

func defaultModels() []modelSpec {
	return []modelSpec{
		{
			Name:               "Threat Detection - Sigma Rule Evaluator",
			Slug:               "cyber-sigma-evaluator",
			Description:        "Deterministic Sigma-style rule evaluation for security events.",
			ModelType:          aigovmodel.ModelTypeRuleBased,
			Suite:              aigovmodel.SuiteCyber,
			RiskTier:           aigovmodel.RiskTierCritical,
			ArtifactType:       aigovmodel.ArtifactTypeRuleSet,
			ExplainabilityType: aigovmodel.ExplainabilityRuleTrace,
			ArtifactConfig: map[string]any{
				"engine": "sigma",
				"supports": []string{"selection", "condition", "timeframe"},
			},
			Template:  `Matched rules: {{join .matched_rules ", "}}.`,
			OwnerTeam: "security-operations",
			Tags:      []string{"cyber", "detection", "sigma"},
			Metadata:  map[string]any{"used_by": "detection_engine"},
		},
		{
			Name:               "Anomaly Detection - Statistical Baseline",
			Slug:               "cyber-anomaly-detector",
			Description:        "Statistical deviation detector using baseline windows and z-score thresholds.",
			ModelType:          aigovmodel.ModelTypeAnomalyDetector,
			Suite:              aigovmodel.SuiteCyber,
			RiskTier:           aigovmodel.RiskTierHigh,
			ArtifactType:       aigovmodel.ArtifactTypeStatisticalConfig,
			ExplainabilityType: aigovmodel.ExplainabilityStatisticalDeviation,
			ArtifactConfig:     map[string]any{"window": "7d", "threshold_z": 3.0},
			Template:           `Observed {{.current_value}} against baseline {{.baseline_mean}} with z-score {{signedFloat .z_score}}.`,
			OwnerTeam:          "security-operations",
			Tags:               []string{"cyber", "anomaly", "baseline"},
			Metadata:           map[string]any{"used_by": "detection_engine"},
		},
		{
			Name:               "Risk Scoring - Multi-Factor Composite",
			Slug:               "cyber-risk-scorer",
			Description:        "Weighted transparent scoring model for cyber organizational risk.",
			ModelType:          aigovmodel.ModelTypeScorer,
			Suite:              aigovmodel.SuiteCyber,
			RiskTier:           aigovmodel.RiskTierHigh,
			ArtifactType:       aigovmodel.ArtifactTypeStatisticalConfig,
			ExplainabilityType: aigovmodel.ExplainabilityFeatureImportance,
			ArtifactConfig: map[string]any{
				"components": map[string]float64{
					"vulnerability": 0.30,
					"threat":        0.25,
					"configuration": 0.20,
					"surface":       0.15,
					"compliance":    0.10,
				},
			},
			Template:  `Top contributors: {{range .contributions}}{{.Name}} {{printf "%.1f" .Share}}%% {{end}}`,
			OwnerTeam: "security-risk",
			Tags:      []string{"cyber", "risk", "scorer"},
			Metadata:  map[string]any{"used_by": "risk_service"},
		},
		{
			Name:               "Asset Auto-Classifier",
			Slug:               "cyber-asset-classifier",
			Description:        "Rule-based asset criticality classifier.",
			ModelType:          aigovmodel.ModelTypeRuleBased,
			Suite:              aigovmodel.SuiteCyber,
			RiskTier:           aigovmodel.RiskTierMedium,
			ArtifactType:       aigovmodel.ArtifactTypeRuleSet,
			ExplainabilityType: aigovmodel.ExplainabilityRuleTrace,
			ArtifactConfig:     map[string]any{"strategy": "priority_first_match"},
			Template:           `Asset criticality set because {{join .matched_rules ", "}} matched.`,
			OwnerTeam:          "security-asset-management",
			Tags:               []string{"cyber", "asset", "classification"},
			Metadata:           map[string]any{"used_by": "asset_service"},
		},
		{
			Name:               "CTEM Prioritization",
			Slug:               "cyber-ctem-prioritizer",
			Description:        "Weighted prioritization model for CTEM findings.",
			ModelType:          aigovmodel.ModelTypeScorer,
			Suite:              aigovmodel.SuiteCyber,
			RiskTier:           aigovmodel.RiskTierHigh,
			ArtifactType:       aigovmodel.ArtifactTypeStatisticalConfig,
			ExplainabilityType: aigovmodel.ExplainabilityFeatureImportance,
			ArtifactConfig:     map[string]any{"weights": map[string]float64{"impact": 0.55, "exploitability": 0.45}},
			Template:           `Priority score driven by business impact and exploitability factors.`,
			OwnerTeam:          "security-exposure-management",
			Tags:               []string{"cyber", "ctem", "prioritization"},
			Metadata:           map[string]any{"used_by": "ctem_engine"},
		},
		{
			Name:               "PII Classifier",
			Slug:               "data-pii-classifier",
			Description:        "Deterministic PII classifier for schema discovery.",
			ModelType:          aigovmodel.ModelTypeRuleBased,
			Suite:              aigovmodel.SuiteData,
			RiskTier:           aigovmodel.RiskTierHigh,
			ArtifactType:       aigovmodel.ArtifactTypeRuleSet,
			ExplainabilityType: aigovmodel.ExplainabilityRuleTrace,
			ArtifactConfig:     map[string]any{"sources": []string{"column_name", "sample_value_patterns"}},
			Template:           `Columns classified as PII because {{join .matched_rules ", "}} matched.`,
			OwnerTeam:          "data-governance",
			Tags:               []string{"data", "pii", "classification"},
			Metadata:           map[string]any{"used_by": "schema_discovery"},
		},
		{
			Name:               "Contradiction Detector",
			Slug:               "data-contradiction-detector",
			Description:        "Transparent logical contradiction detector across sources.",
			ModelType:          aigovmodel.ModelTypeRuleBased,
			Suite:              aigovmodel.SuiteData,
			RiskTier:           aigovmodel.RiskTierMedium,
			ArtifactType:       aigovmodel.ArtifactTypeRuleSet,
			ExplainabilityType: aigovmodel.ExplainabilityRuleTrace,
			ArtifactConfig:     map[string]any{"strategies": []string{"logical", "semantic", "temporal", "analytical"}},
			Template:           `Contradiction flagged because {{join .matched_conditions ", "}}.`,
			OwnerTeam:          "data-governance",
			Tags:               []string{"data", "quality", "contradiction"},
			Metadata:           map[string]any{"used_by": "contradiction_service"},
		},
		{
			Name:               "Data Quality Scorer",
			Slug:               "data-quality-scorer",
			Description:        "Transparent weighted scorer for enterprise data quality.",
			ModelType:          aigovmodel.ModelTypeScorer,
			Suite:              aigovmodel.SuiteData,
			RiskTier:           aigovmodel.RiskTierMedium,
			ArtifactType:       aigovmodel.ArtifactTypeStatisticalConfig,
			ExplainabilityType: aigovmodel.ExplainabilityFeatureImportance,
			ArtifactConfig:     map[string]any{"weights": map[string]float64{"critical": 4, "high": 3, "medium": 2, "low": 1}},
			Template:           `Quality score derived from passed and failed rules weighted by severity.`,
			OwnerTeam:          "data-quality",
			Tags:               []string{"data", "quality", "scorer"},
			Metadata:           map[string]any{"used_by": "quality_service"},
		},
		{
			Name:               "Meeting Minutes Generator",
			Slug:               "acta-minutes-generator",
			Description:        "Template-driven meeting minutes generation.",
			ModelType:          aigovmodel.ModelTypeNLPExtractor,
			Suite:              aigovmodel.SuiteActa,
			RiskTier:           aigovmodel.RiskTierMedium,
			ArtifactType:       aigovmodel.ArtifactTypeTemplateConfig,
			ExplainabilityType: aigovmodel.ExplainabilityTemplateBased,
			ArtifactConfig:     map[string]any{"template": "minutes_markdown", "summary_builder": "deterministic"},
			Template:           `Generated minutes from {{index .metadata "agenda_count"}} agenda item(s) and {{index .metadata "attendance_count"}} attendee record(s).`,
			OwnerTeam:          "governance-operations",
			Tags:               []string{"acta", "minutes", "template"},
			Metadata:           map[string]any{"used_by": "minutes_service"},
		},
		{
			Name:               "Action Item Extractor",
			Slug:               "acta-action-extractor",
			Description:        "Pattern-based action item extraction from meeting notes.",
			ModelType:          aigovmodel.ModelTypeNLPExtractor,
			Suite:              aigovmodel.SuiteActa,
			RiskTier:           aigovmodel.RiskTierLow,
			ArtifactType:       aigovmodel.ArtifactTypeRuleSet,
			ExplainabilityType: aigovmodel.ExplainabilityRuleTrace,
			ArtifactConfig:     map[string]any{"patterns": []string{"ACTION:", "will", "agreed that"}},
			Template:           `Extracted actions based on matched note patterns.`,
			OwnerTeam:          "governance-operations",
			Tags:               []string{"acta", "actions", "extractor"},
			Metadata:           map[string]any{"used_by": "minutes_generator"},
		},
		{
			Name:               "Contract Clause Extractor",
			Slug:               "lex-clause-extractor",
			Description:        "Pattern-driven clause extractor for legal documents.",
			ModelType:          aigovmodel.ModelTypeRuleBased,
			Suite:              aigovmodel.SuiteLex,
			RiskTier:           aigovmodel.RiskTierHigh,
			ArtifactType:       aigovmodel.ArtifactTypeRuleSet,
			ExplainabilityType: aigovmodel.ExplainabilityRuleTrace,
			ArtifactConfig:     map[string]any{"clause_catalog_size": 19},
			Template:           `Clause types detected from document sections using deterministic patterns.`,
			OwnerTeam:          "legal-operations",
			Tags:               []string{"lex", "clause", "extractor"},
			Metadata:           map[string]any{"used_by": "contract_analyzer"},
		},
		{
			Name:               "Contract Risk Analyzer",
			Slug:               "lex-risk-analyzer",
			Description:        "Transparent weighted risk analysis for contracts.",
			ModelType:          aigovmodel.ModelTypeScorer,
			Suite:              aigovmodel.SuiteLex,
			RiskTier:           aigovmodel.RiskTierHigh,
			ArtifactType:       aigovmodel.ArtifactTypeStatisticalConfig,
			ExplainabilityType: aigovmodel.ExplainabilityFeatureImportance,
			ArtifactConfig:     map[string]any{"factors": []string{"clause_risk", "missing_clause", "value", "expiry", "compliance"}},
			Template:           `Contract risk score built from clause risk, missing clauses, commercial value, expiry, and compliance flags.`,
			OwnerTeam:          "legal-operations",
			Tags:               []string{"lex", "risk", "analysis"},
			Metadata:           map[string]any{"used_by": "contract_analyzer"},
		},
		{
			Name:               "KPI Threshold Monitor",
			Slug:               "visus-kpi-monitor",
			Description:        "Threshold evaluator for KPI status transitions.",
			ModelType:          aigovmodel.ModelTypeStatistical,
			Suite:              aigovmodel.SuiteVisus,
			RiskTier:           aigovmodel.RiskTierMedium,
			ArtifactType:       aigovmodel.ArtifactTypeStatisticalConfig,
			ExplainabilityType: aigovmodel.ExplainabilityStatisticalDeviation,
			ArtifactConfig:     map[string]any{"logic": "directional_threshold"},
			Template:           `KPI status determined by value {{.current_value}} against configured thresholds.`,
			OwnerTeam:          "executive-reporting",
			Tags:               []string{"visus", "kpi", "monitor"},
			Metadata:           map[string]any{"used_by": "kpi_engine"},
		},
		{
			Name:               "Executive Recommendation Engine",
			Slug:               "visus-recommendation-engine",
			Description:        "Rule-based recommendation engine for executive reporting.",
			ModelType:          aigovmodel.ModelTypeRecommender,
			Suite:              aigovmodel.SuiteVisus,
			RiskTier:           aigovmodel.RiskTierLow,
			ArtifactType:       aigovmodel.ArtifactTypeRuleSet,
			ExplainabilityType: aigovmodel.ExplainabilityRuleTrace,
			ArtifactConfig:     map[string]any{"triggers": []string{"critical_kpi", "overdue_action", "expiring_contract", "coverage_gap"}},
			Template:           `Recommendations generated because {{join .matched_rules ", "}} conditions were present in the report data.`,
			OwnerTeam:          "executive-reporting",
			Tags:               []string{"visus", "recommendation", "reporting"},
			Metadata:           map[string]any{"used_by": "report_generator"},
		},
	}
}

func mustJSON(value any) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
}

func hashConfig(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", err
	}
	return aigovernance.HashJSON(decoded)
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
