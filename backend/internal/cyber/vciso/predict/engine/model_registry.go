package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
	predictmodels "github.com/clario360/platform/internal/cyber/vciso/predict/models"
	predictrepo "github.com/clario360/platform/internal/cyber/vciso/predict/repository"
)

type ModelRegistry struct {
	repo        *predictrepo.PredictionRepository
	artifactDir string
	logger      zerolog.Logger

	alertVolume   *predictmodels.AlertVolumeForecaster
	assetRisk     *predictmodels.AssetRiskPredictor
	vulnerability *predictmodels.VulnerabilityExploitPredictor
	technique     *predictmodels.TechniqueTrendAnalyzer
	insider       *predictmodels.InsiderThreatTrajectoryModel
	campaign      *predictmodels.CampaignDetector
}

func NewModelRegistry(repo *predictrepo.PredictionRepository, artifactDir string, logger zerolog.Logger) *ModelRegistry {
	if artifactDir == "" {
		artifactDir = filepath.Join("var", "vciso-predictive-models")
	}
	return &ModelRegistry{
		repo:        repo,
		artifactDir: artifactDir,
		logger:      logger.With().Str("component", "vciso_predict_model_registry").Logger(),
	}
}

func (r *ModelRegistry) EnsureDefaults(ctx context.Context) error {
	defaults := []struct {
		modelType  predictmodel.PredictionType
		framework  predictmodel.ModelFramework
		model      any
		featureCnt int
	}{
		{predictmodel.PredictionTypeAlertVolumeForecast, predictmodel.FrameworkProphet, predictmodels.NewAlertVolumeForecaster("alert-volume-v1"), 5},
		{predictmodel.PredictionTypeAssetRisk, predictmodel.FrameworkXGBoost, predictmodels.NewAssetRiskPredictor("asset-risk-v1"), 10},
		{predictmodel.PredictionTypeVulnerabilityExploit, predictmodel.FrameworkGBM, predictmodels.NewVulnerabilityExploitPredictor("vulnerability-exploit-v1"), 9},
		{predictmodel.PredictionTypeAttackTechniqueTrend, predictmodel.FrameworkRegression, predictmodels.NewTechniqueTrendAnalyzer("technique-trend-v1"), 4},
		{predictmodel.PredictionTypeInsiderThreatTrajectory, predictmodel.FrameworkLSTM, predictmodels.NewInsiderThreatTrajectoryModel("insider-trajectory-v1"), 7},
		{predictmodel.PredictionTypeCampaignDetection, predictmodel.FrameworkDBSCAN, predictmodels.NewCampaignDetector("campaign-detector-v1"), 5},
	}
	for _, spec := range defaults {
		if r.repo == nil {
			r.setModel(spec.modelType, spec.model)
			continue
		}
		existing, err := r.repo.GetActiveModel(ctx, spec.modelType)
		if err == nil && existing != nil {
			if loadErr := r.loadModelFromArtifact(spec.modelType, existing.ModelArtifactPath); loadErr == nil {
				continue
			}
		}
		if _, err := r.Activate(ctx, spec.modelType, spec.framework, spec.model, predictmodel.BacktestMetrics{
			Accuracy: 0.80,
			Count:    1,
		}, spec.featureCnt, 1, time.Second); err != nil {
			return err
		}
	}
	return nil
}

func (r *ModelRegistry) Activate(
	ctx context.Context,
	modelType predictmodel.PredictionType,
	framework predictmodel.ModelFramework,
	model any,
	backtest predictmodel.BacktestMetrics,
	featureCount int,
	trainingSamples int,
	duration time.Duration,
) (*predictmodel.PredictionModel, error) {
	version, payload, err := r.serializeModel(modelType, model)
	if err != nil {
		return nil, err
	}
	artifactPath := filepath.Join(r.artifactDir, string(modelType), version+".json")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		return nil, fmt.Errorf("create predictive artifact directory: %w", err)
	}
	if err := os.WriteFile(artifactPath, payload, 0o600); err != nil {
		return nil, fmt.Errorf("write predictive artifact: %w", err)
	}
	if r.repo == nil {
		r.setModel(modelType, model)
		return &predictmodel.PredictionModel{
			ModelType:               modelType,
			Version:                 version,
			ModelArtifactPath:       artifactPath,
			ModelFramework:          framework,
			FeatureCount:            featureCount,
			TrainingSamples:         trainingSamples,
			TrainingDurationSeconds: int(duration.Seconds()),
			Status:                  predictmodel.PredictionModelStatusActive,
			Active:                  true,
		}, nil
	}
	now := time.Now().UTC()
	modelRow := &predictmodel.PredictionModel{
		ModelType:               modelType,
		Version:                 version,
		ModelArtifactPath:       artifactPath,
		ModelFramework:          framework,
		BacktestAccuracy:        floatPtr(backtest.Accuracy),
		BacktestPrecision:       optionalFloat(backtest.Precision),
		BacktestRecall:          optionalFloat(backtest.Recall),
		BacktestF1:              optionalFloat(backtest.F1),
		BacktestMAPE:            optionalFloat(backtest.MAPE),
		FeatureCount:            featureCount,
		TrainingSamples:         trainingSamples,
		TrainingDurationSeconds: int(duration.Seconds()),
		Status:                  predictmodel.PredictionModelStatusActive,
		Active:                  true,
		ActivatedAt:             &now,
	}
	if err := r.repo.UpsertModel(ctx, modelRow); err != nil {
		return nil, err
	}
	r.setModel(modelType, model)
	return modelRow, nil
}

func (r *ModelRegistry) LoadActive(ctx context.Context) error {
	if r.repo == nil {
		return nil
	}
	for _, modelType := range []predictmodel.PredictionType{
		predictmodel.PredictionTypeAlertVolumeForecast,
		predictmodel.PredictionTypeAssetRisk,
		predictmodel.PredictionTypeVulnerabilityExploit,
		predictmodel.PredictionTypeAttackTechniqueTrend,
		predictmodel.PredictionTypeInsiderThreatTrajectory,
		predictmodel.PredictionTypeCampaignDetection,
	} {
		item, err := r.repo.GetActiveModel(ctx, modelType)
		if err != nil {
			if err == predictrepo.ErrNotFound {
				continue
			}
			return err
		}
		if err := r.loadModelFromArtifact(modelType, item.ModelArtifactPath); err != nil {
			return err
		}
	}
	return nil
}

func (r *ModelRegistry) AlertVolume() *predictmodels.AlertVolumeForecaster {
	return r.alertVolume
}

func (r *ModelRegistry) AssetRisk() *predictmodels.AssetRiskPredictor {
	return r.assetRisk
}

func (r *ModelRegistry) Vulnerability() *predictmodels.VulnerabilityExploitPredictor {
	return r.vulnerability
}

func (r *ModelRegistry) Technique() *predictmodels.TechniqueTrendAnalyzer {
	return r.technique
}

func (r *ModelRegistry) Insider() *predictmodels.InsiderThreatTrajectoryModel {
	return r.insider
}

func (r *ModelRegistry) Campaign() *predictmodels.CampaignDetector {
	return r.campaign
}

func (r *ModelRegistry) loadModelFromArtifact(modelType predictmodel.PredictionType, artifactPath string) error {
	payload, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("read predictive artifact %s: %w", artifactPath, err)
	}
	switch modelType {
	case predictmodel.PredictionTypeAlertVolumeForecast:
		model := predictmodels.NewAlertVolumeForecaster("")
		if err := model.Deserialize(payload); err != nil {
			return err
		}
		r.alertVolume = model
	case predictmodel.PredictionTypeAssetRisk:
		model := predictmodels.NewAssetRiskPredictor("")
		if err := model.Deserialize(payload); err != nil {
			return err
		}
		r.assetRisk = model
	case predictmodel.PredictionTypeVulnerabilityExploit:
		model := predictmodels.NewVulnerabilityExploitPredictor("")
		if err := model.Deserialize(payload); err != nil {
			return err
		}
		r.vulnerability = model
	case predictmodel.PredictionTypeAttackTechniqueTrend:
		model := predictmodels.NewTechniqueTrendAnalyzer("")
		if err := model.Deserialize(payload); err != nil {
			return err
		}
		r.technique = model
	case predictmodel.PredictionTypeInsiderThreatTrajectory:
		model := predictmodels.NewInsiderThreatTrajectoryModel("")
		if err := model.Deserialize(payload); err != nil {
			return err
		}
		r.insider = model
	case predictmodel.PredictionTypeCampaignDetection:
		model := predictmodels.NewCampaignDetector("")
		if err := model.Deserialize(payload); err != nil {
			return err
		}
		r.campaign = model
	default:
		return fmt.Errorf("unsupported predictive model type %q", modelType)
	}
	return nil
}

func (r *ModelRegistry) serializeModel(modelType predictmodel.PredictionType, model any) (string, []byte, error) {
	switch modelType {
	case predictmodel.PredictionTypeAlertVolumeForecast:
		typed, ok := model.(*predictmodels.AlertVolumeForecaster)
		if !ok {
			return "", nil, fmt.Errorf("invalid alert volume model type %T", model)
		}
		payload, err := typed.Serialize()
		return typed.ModelVersion, payload, err
	case predictmodel.PredictionTypeAssetRisk:
		typed, ok := model.(*predictmodels.AssetRiskPredictor)
		if !ok {
			return "", nil, fmt.Errorf("invalid asset risk model type %T", model)
		}
		payload, err := typed.Serialize()
		return typed.ModelVersion, payload, err
	case predictmodel.PredictionTypeVulnerabilityExploit:
		typed, ok := model.(*predictmodels.VulnerabilityExploitPredictor)
		if !ok {
			return "", nil, fmt.Errorf("invalid vulnerability model type %T", model)
		}
		payload, err := typed.Serialize()
		return typed.ModelVersion, payload, err
	case predictmodel.PredictionTypeAttackTechniqueTrend:
		typed, ok := model.(*predictmodels.TechniqueTrendAnalyzer)
		if !ok {
			return "", nil, fmt.Errorf("invalid technique trend model type %T", model)
		}
		payload, err := typed.Serialize()
		return typed.ModelVersion, payload, err
	case predictmodel.PredictionTypeInsiderThreatTrajectory:
		typed, ok := model.(*predictmodels.InsiderThreatTrajectoryModel)
		if !ok {
			return "", nil, fmt.Errorf("invalid insider threat model type %T", model)
		}
		payload, err := typed.Serialize()
		return typed.ModelVersion, payload, err
	case predictmodel.PredictionTypeCampaignDetection:
		typed, ok := model.(*predictmodels.CampaignDetector)
		if !ok {
			return "", nil, fmt.Errorf("invalid campaign detector model type %T", model)
		}
		payload, err := typed.Serialize()
		return typed.ModelVersion, payload, err
	default:
		return "", nil, fmt.Errorf("unsupported predictive model type %q", modelType)
	}
}

func (r *ModelRegistry) setModel(modelType predictmodel.PredictionType, model any) {
	switch modelType {
	case predictmodel.PredictionTypeAlertVolumeForecast:
		r.alertVolume = model.(*predictmodels.AlertVolumeForecaster)
	case predictmodel.PredictionTypeAssetRisk:
		r.assetRisk = model.(*predictmodels.AssetRiskPredictor)
	case predictmodel.PredictionTypeVulnerabilityExploit:
		r.vulnerability = model.(*predictmodels.VulnerabilityExploitPredictor)
	case predictmodel.PredictionTypeAttackTechniqueTrend:
		r.technique = model.(*predictmodels.TechniqueTrendAnalyzer)
	case predictmodel.PredictionTypeInsiderThreatTrajectory:
		r.insider = model.(*predictmodels.InsiderThreatTrajectoryModel)
	case predictmodel.PredictionTypeCampaignDetection:
		r.campaign = model.(*predictmodels.CampaignDetector)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func optionalFloat(value float64) *float64 {
	if value == 0 {
		return nil
	}
	return &value
}
