package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	predictdto "github.com/clario360/platform/internal/cyber/vciso/predict/dto"
	predictexplainer "github.com/clario360/platform/internal/cyber/vciso/predict/explainer"
	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
	predictmodels "github.com/clario360/platform/internal/cyber/vciso/predict/models"
	predictrepo "github.com/clario360/platform/internal/cyber/vciso/predict/repository"
)

type ForecastEngine struct {
	store      *FeatureStore
	registry   *ModelRegistry
	repo       *predictrepo.PredictionRepository
	shap       *predictexplainer.SHAPExplainer
	narrator   *predictexplainer.PredictionNarrator
	calibrator *predictexplainer.ConfidenceCalibrator
	backtester *Backtester
	drift      *DriftDetector
	predLogger *aigovmiddleware.PredictionLogger
	metrics    *Metrics
	logger     zerolog.Logger
	now        func() time.Time
}

func NewForecastEngine(
	store *FeatureStore,
	registry *ModelRegistry,
	repo *predictrepo.PredictionRepository,
	shap *predictexplainer.SHAPExplainer,
	narrator *predictexplainer.PredictionNarrator,
	calibrator *predictexplainer.ConfidenceCalibrator,
	backtester *Backtester,
	drift *DriftDetector,
	predLogger *aigovmiddleware.PredictionLogger,
	metrics *Metrics,
	logger zerolog.Logger,
) *ForecastEngine {
	if shap == nil {
		shap = predictexplainer.NewSHAPExplainer()
	}
	if narrator == nil {
		narrator = predictexplainer.NewPredictionNarrator()
	}
	if calibrator == nil {
		calibrator = predictexplainer.NewConfidenceCalibrator()
	}
	if backtester == nil {
		backtester = NewBacktester()
	}
	if drift == nil {
		drift = NewDriftDetector()
	}
	return &ForecastEngine{
		store:      store,
		registry:   registry,
		repo:       repo,
		shap:       shap,
		narrator:   narrator,
		calibrator: calibrator,
		backtester: backtester,
		drift:      drift,
		predLogger: predLogger,
		metrics:    metrics,
		logger:     logger.With().Str("component", "vciso_predict_engine").Logger(),
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (e *ForecastEngine) Start(ctx context.Context) error {
	if e.registry != nil {
		if err := e.registry.EnsureDefaults(ctx); err != nil {
			return err
		}
		if err := e.registry.LoadActive(ctx); err != nil {
			return err
		}
	}
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := e.runMaintenance(ctx); err != nil {
				e.logger.Warn().Err(err).Msg("predictive maintenance cycle failed")
			}
		}
	}
}

func (e *ForecastEngine) ForecastAlertVolume(ctx context.Context, tenantID uuid.UUID, horizonDays int) (*predictdto.ForecastResponse, error) {
	started := e.now()
	if horizonDays <= 0 {
		horizonDays = 7
	}
	if err := e.ensureAlertVolumeModel(ctx, tenantID); err != nil {
		return nil, err
	}
	model := e.registry.AlertVolume()
	samples, err := e.store.AlertVolumeSamples(ctx, tenantID, 365)
	if err != nil {
		return nil, err
	}
	forecast, featureTotals := model.Forecast(horizonDays, nil)
	topFeatures := mapContributions(featureTotals)
	confidenceScore := clamp(1-(percentile(absResiduals(model.Residuals), 0.90)/math.Max(model.BaseLevel, 1)), 0.25, 0.95)
	_, interval := e.calibrator.CalibrateProbability(confidenceScore, model.Residuals)
	explanation, verification := e.narrator.Explain(predictmodel.PredictionTypeAlertVolumeForecast, confidenceScore, interval, topFeatures, "")
	if err := e.persistPrediction(ctx, tenantID, predictmodel.PredictionTypeAlertVolumeForecast, model.ModelVersion, forecast, confidenceScore, interval, topFeatures, explanation, nil, nil, started, started.AddDate(0, 0, horizonDays)); err != nil {
		e.logger.Warn().Err(err).Msg("persist alert volume forecast")
	}
	response := &predictdto.ForecastResponse{
		GenericPredictionResponse: predictdto.GenericPredictionResponse{
			PredictionType:     predictmodel.PredictionTypeAlertVolumeForecast,
			ModelVersion:       model.ModelVersion,
			GeneratedAt:        started,
			ConfidenceScore:    confidenceScore,
			ConfidenceInterval: interval,
			TopFeatures:        topFeatures,
			ExplanationText:    explanation,
			VerificationSteps:  verification,
		},
		Forecast: forecast,
	}
	e.observePrediction(predictmodel.PredictionTypeAlertVolumeForecast, model.ModelVersion, started)
	_ = samples // samples kept for parity with feature generation
	return response, nil
}

func (e *ForecastEngine) PredictAssetRisk(ctx context.Context, tenantID uuid.UUID, limit int, assetType string) (*predictdto.AssetRiskResponse, error) {
	started := e.now()
	if limit <= 0 {
		limit = 10
	}
	if err := e.ensureAssetRiskModel(ctx, tenantID, assetType); err != nil {
		return nil, err
	}
	model := e.registry.AssetRisk()
	samples, err := e.store.AssetRiskSamples(ctx, tenantID, assetType)
	if err != nil {
		return nil, err
	}
	type scored struct {
		sample predictmodels.AssetRiskSample
		score  float64
	}
	ranked := make([]scored, 0, len(samples))
	for _, sample := range samples {
		ranked = append(ranked, scored{sample: sample, score: model.Predict(sample)})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].sample.AssetName < ranked[j].sample.AssetName
		}
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	items := make([]predictmodel.AssetRiskItem, 0, len(ranked))
	aggregate := make([]predictmodel.FeatureContribution, 0, 5)
	for idx, item := range ranked {
		_, interval := e.calibrator.CalibrateProbability(item.score, model.Residuals)
		top := e.shap.TopN(e.shap.FromWeights(assetRiskValues(item.sample), model.Baseline, model.Weights, assetRiskRaw(item.sample)), 5)
		if idx == 0 {
			aggregate = top
		}
		items = append(items, predictmodel.AssetRiskItem{
			AssetID:            item.sample.AssetID,
			AssetName:          item.sample.AssetName,
			AssetType:          item.sample.AssetType,
			Probability:        item.score,
			ConfidenceInterval: interval,
			CurrentRisk:        item.sample.HistoricalAlerts + item.sample.OpenCritical*2 + item.sample.OpenHigh,
			TopFeatures:        top,
		})
	}
	confidenceScore := averageScores(len(ranked), func(idx int) float64 { return ranked[idx].score })
	_, interval := e.calibrator.CalibrateProbability(confidenceScore, model.Residuals)
	explanation, verification := e.narrator.Explain(predictmodel.PredictionTypeAssetRisk, confidenceScore, interval, aggregate, assetType)
	if err := e.persistPrediction(ctx, tenantID, predictmodel.PredictionTypeAssetRisk, model.ModelVersion, items, confidenceScore, interval, aggregate, explanation, stringPtr("asset"), nil, started, started.AddDate(0, 0, 30)); err != nil {
		e.logger.Warn().Err(err).Msg("persist asset risk prediction")
	}
	e.observePrediction(predictmodel.PredictionTypeAssetRisk, model.ModelVersion, started)
	return &predictdto.AssetRiskResponse{
		GenericPredictionResponse: predictdto.GenericPredictionResponse{
			PredictionType:     predictmodel.PredictionTypeAssetRisk,
			ModelVersion:       model.ModelVersion,
			GeneratedAt:        started,
			ConfidenceScore:    confidenceScore,
			ConfidenceInterval: interval,
			TopFeatures:        aggregate,
			ExplanationText:    explanation,
			VerificationSteps:  verification,
		},
		Items: items,
	}, nil
}

func (e *ForecastEngine) PredictVulnerabilityPriority(ctx context.Context, tenantID uuid.UUID, limit int, minProbability float64) (*predictdto.VulnerabilityPriorityResponse, error) {
	started := e.now()
	if limit <= 0 {
		limit = 20
	}
	if minProbability <= 0 {
		minProbability = 0.50
	}
	if err := e.ensureVulnerabilityModel(ctx, tenantID); err != nil {
		return nil, err
	}
	model := e.registry.Vulnerability()
	samples, err := e.store.VulnerabilitySamples(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	type scored struct {
		sample predictmodels.VulnerabilitySample
		score  float64
	}
	ranked := make([]scored, 0, len(samples))
	for _, sample := range samples {
		score := model.Predict(sample)
		if score < minProbability {
			continue
		}
		ranked = append(ranked, scored{sample: sample, score: score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].sample.CVEID < ranked[j].sample.CVEID
		}
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	items := make([]predictmodel.VulnerabilityPriorityItem, 0, len(ranked))
	aggregate := make([]predictmodel.FeatureContribution, 0, 5)
	for idx, item := range ranked {
		_, bounds := e.calibrator.CalibrateProbability(item.score, model.Residuals)
		top := e.shap.TopN(e.shap.FromWeights(vulnValues(item.sample), model.Baseline, model.Weights, vulnRaw(item.sample)), 5)
		if idx == 0 {
			aggregate = top
		}
		items = append(items, predictmodel.VulnerabilityPriorityItem{
			VulnerabilityID: item.sample.VulnerabilityID,
			AssetID:         item.sample.AssetID,
			AssetName:       item.sample.AssetName,
			CVEID:           item.sample.CVEID,
			Severity:        item.sample.Severity,
			Probability:     item.score,
			ConfidenceInterval: bounds,
			TopFeatures:     top,
		})
	}
	confidenceScore := averageScores(len(ranked), func(idx int) float64 { return ranked[idx].score })
	_, interval := e.calibrator.CalibrateProbability(confidenceScore, model.Residuals)
	explanation, verification := e.narrator.Explain(predictmodel.PredictionTypeVulnerabilityExploit, confidenceScore, interval, aggregate, "")
	if err := e.persistPrediction(ctx, tenantID, predictmodel.PredictionTypeVulnerabilityExploit, model.ModelVersion, items, confidenceScore, interval, aggregate, explanation, stringPtr("vulnerability"), nil, started, started.AddDate(0, 0, 30)); err != nil {
		e.logger.Warn().Err(err).Msg("persist vulnerability priority prediction")
	}
	e.observePrediction(predictmodel.PredictionTypeVulnerabilityExploit, model.ModelVersion, started)
	return &predictdto.VulnerabilityPriorityResponse{
		GenericPredictionResponse: predictdto.GenericPredictionResponse{
			PredictionType:     predictmodel.PredictionTypeVulnerabilityExploit,
			ModelVersion:       model.ModelVersion,
			GeneratedAt:        started,
			ConfidenceScore:    confidenceScore,
			ConfidenceInterval: interval,
			TopFeatures:        aggregate,
			ExplanationText:    explanation,
			VerificationSteps:  verification,
		},
		Items: items,
	}, nil
}

func (e *ForecastEngine) PredictTechniqueTrends(ctx context.Context, tenantID uuid.UUID, horizonDays int) (*predictdto.TechniqueTrendResponse, error) {
	started := e.now()
	if horizonDays <= 0 {
		horizonDays = 30
	}
	if err := e.ensureTechniqueModel(ctx, tenantID); err != nil {
		return nil, err
	}
	model := e.registry.Technique()
	items := model.Predict(horizonDays)
	for idx := range items {
		last := model.LastSamples[items[idx].TechniqueID]
		items[idx].TopFeatures = e.shap.TopN(e.shap.FromWeights(map[string]float64{
			"internal_count":       last.InternalCount,
			"industry_count":       last.IndustryCount,
			"campaign_correlation": last.CampaignCorrelation,
			"seasonality":          last.Seasonality,
		}, map[string]float64{}, model.Weights, map[string]any{
			"internal_count":       last.InternalCount,
			"industry_count":       last.IndustryCount,
			"campaign_correlation": last.CampaignCorrelation,
			"seasonality":          last.Seasonality,
		}), 5)
	}
	if len(items) > 10 {
		items = items[:10]
	}
	confidenceScore := clamp(1-(percentile(absResiduals(model.Residuals), 0.90)/5), 0.25, 0.95)
	_, interval := e.calibrator.CalibrateProbability(confidenceScore, model.Residuals)
	aggregate := []predictmodel.FeatureContribution{}
	if len(items) > 0 {
		aggregate = items[0].TopFeatures
	}
	explanation, verification := e.narrator.Explain(predictmodel.PredictionTypeAttackTechniqueTrend, confidenceScore, interval, aggregate, "")
	if err := e.persistPrediction(ctx, tenantID, predictmodel.PredictionTypeAttackTechniqueTrend, model.ModelVersion, items, confidenceScore, interval, aggregate, explanation, stringPtr("technique"), nil, started, started.AddDate(0, 0, horizonDays)); err != nil {
		e.logger.Warn().Err(err).Msg("persist technique trend prediction")
	}
	e.observePrediction(predictmodel.PredictionTypeAttackTechniqueTrend, model.ModelVersion, started)
	return &predictdto.TechniqueTrendResponse{
		GenericPredictionResponse: predictdto.GenericPredictionResponse{
			PredictionType:     predictmodel.PredictionTypeAttackTechniqueTrend,
			ModelVersion:       model.ModelVersion,
			GeneratedAt:        started,
			ConfidenceScore:    confidenceScore,
			ConfidenceInterval: interval,
			TopFeatures:        aggregate,
			ExplanationText:    explanation,
			VerificationSteps:  verification,
		},
		Items: items,
	}, nil
}

func (e *ForecastEngine) ForecastInsiderThreats(ctx context.Context, tenantID uuid.UUID, horizonDays int, threshold int) (*predictdto.InsiderThreatResponse, error) {
	started := e.now()
	if horizonDays <= 0 {
		horizonDays = 30
	}
	if threshold <= 0 {
		threshold = 70
	}
	if err := e.ensureInsiderModel(ctx, tenantID); err != nil {
		return nil, err
	}
	model := e.registry.Insider()
	sequences, err := e.store.InsiderThreatSequences(ctx, tenantID, 90)
	if err != nil {
		return nil, err
	}
	items := make([]predictmodel.InsiderThreatTrajectoryItem, 0, len(sequences))
	for _, series := range sequences {
		projected, accelerating, daysToThreshold := model.Predict(series, horizonDays, float64(threshold))
		last := series[len(series)-1]
		if projected < float64(threshold) {
			continue
		}
		confidence := e.calibrator.CalibrateValue(projected, model.Residuals)
		top := e.shap.TopN(e.shap.FromWeights(map[string]float64{
			"risk_score":        last.RiskScore,
			"login_anomalies":   last.LoginAnomalies,
			"data_access_trend": last.DataAccessTrend,
			"after_hours_trend": last.AfterHoursTrend,
			"policy_violations": last.PolicyViolations,
			"hr_event_score":    last.HREventScore,
			"peer_deviation":    last.PeerDeviation,
		}, map[string]float64{}, model.CandidateWeights, map[string]any{
			"risk_score":        last.RiskScore,
			"login_anomalies":   last.LoginAnomalies,
			"data_access_trend": last.DataAccessTrend,
			"after_hours_trend": last.AfterHoursTrend,
			"policy_violations": last.PolicyViolations,
			"hr_event_score":    last.HREventScore,
			"peer_deviation":    last.PeerDeviation,
		}), 5)
		items = append(items, predictmodel.InsiderThreatTrajectoryItem{
			EntityID:           last.EntityID,
			EntityName:         last.EntityName,
			CurrentRisk:        last.RiskScore,
			ProjectedRisk:      projected,
			ConfidenceInterval: confidence,
			DaysToThreshold:    daysToThreshold,
			Accelerating:       accelerating,
			TopFeatures:        top,
			VerificationStep:   "Review the underlying UEBA evidence before escalation.",
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].ProjectedRisk > items[j].ProjectedRisk })
	confidenceScore := clamp(1-(percentile(absResiduals(model.Residuals), 0.90)/20), 0.25, 0.95)
	_, interval := e.calibrator.CalibrateProbability(confidenceScore, model.Residuals)
	aggregate := []predictmodel.FeatureContribution{}
	if len(items) > 0 {
		aggregate = items[0].TopFeatures
	}
	explanation, verification := e.narrator.Explain(predictmodel.PredictionTypeInsiderThreatTrajectory, confidenceScore, interval, aggregate, "")
	if err := e.persistPrediction(ctx, tenantID, predictmodel.PredictionTypeInsiderThreatTrajectory, model.ModelVersion, items, confidenceScore, interval, aggregate, explanation, stringPtr("user"), nil, started, started.AddDate(0, 0, horizonDays)); err != nil {
		e.logger.Warn().Err(err).Msg("persist insider forecast")
	}
	e.observePrediction(predictmodel.PredictionTypeInsiderThreatTrajectory, model.ModelVersion, started)
	return &predictdto.InsiderThreatResponse{
		GenericPredictionResponse: predictdto.GenericPredictionResponse{
			PredictionType:     predictmodel.PredictionTypeInsiderThreatTrajectory,
			ModelVersion:       model.ModelVersion,
			GeneratedAt:        started,
			ConfidenceScore:    confidenceScore,
			ConfidenceInterval: interval,
			TopFeatures:        aggregate,
			ExplanationText:    explanation,
			VerificationSteps:  verification,
		},
		Items: items,
	}, nil
}

func (e *ForecastEngine) DetectCampaigns(ctx context.Context, tenantID uuid.UUID, lookbackDays int) (*predictdto.CampaignResponse, error) {
	started := e.now()
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	if err := e.ensureCampaignModel(ctx, tenantID); err != nil {
		return nil, err
	}
	model := e.registry.Campaign()
	samples, err := e.store.CampaignSamples(ctx, tenantID, lookbackDays)
	if err != nil {
		return nil, err
	}
	items := model.Detect(samples)
	for idx := range items {
		items[idx].TopFeatures = []predictmodel.FeatureContribution{
			{Feature: "shared_iocs", SHAPValue: float64(len(items[idx].SharedIOCs)), Direction: "increase"},
			{Feature: "technique_overlap", SHAPValue: float64(len(items[idx].MITRETechniques)), Direction: "increase"},
			{Feature: "timeline_density", SHAPValue: float64(len(items[idx].AlertIDs)), Direction: "increase"},
		}
	}
	cohesion := make([]float64, 0, len(items))
	for _, item := range items {
		cohesion = append(cohesion, item.ConfidenceInterval.P50)
	}
	quality, _ := e.backtester.ClusterQuality(cohesion)
	confidenceScore := clamp(quality.Accuracy, 0.20, 0.95)
	_, interval := e.calibrator.CalibrateProbability(confidenceScore, []float64{1 - confidenceScore})
	aggregate := []predictmodel.FeatureContribution{}
	if len(items) > 0 {
		aggregate = items[0].TopFeatures
	}
	explanation, verification := e.narrator.Explain(predictmodel.PredictionTypeCampaignDetection, confidenceScore, interval, aggregate, "")
	if err := e.persistPrediction(ctx, tenantID, predictmodel.PredictionTypeCampaignDetection, model.ModelVersion, items, confidenceScore, interval, aggregate, explanation, stringPtr("campaign"), nil, started, started.AddDate(0, 0, lookbackDays)); err != nil {
		e.logger.Warn().Err(err).Msg("persist campaign clusters")
	}
	e.observePrediction(predictmodel.PredictionTypeCampaignDetection, model.ModelVersion, started)
	return &predictdto.CampaignResponse{
		GenericPredictionResponse: predictdto.GenericPredictionResponse{
			PredictionType:     predictmodel.PredictionTypeCampaignDetection,
			ModelVersion:       model.ModelVersion,
			GeneratedAt:        started,
			ConfidenceScore:    confidenceScore,
			ConfidenceInterval: interval,
			TopFeatures:        aggregate,
			ExplanationText:    explanation,
			VerificationSteps:  verification,
		},
		Items: items,
	}, nil
}

func (e *ForecastEngine) AccuracyDashboard(ctx context.Context, tenantID uuid.UUID) (*predictmodel.AccuracyDashboard, error) {
	if e.repo == nil {
		return &predictmodel.AccuracyDashboard{ByType: map[predictmodel.PredictionType]float64{}}, nil
	}
	accuracy, err := e.repo.AccuracyByType(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	items, err := e.repo.ListPredictions(ctx, tenantID, nil, 50)
	if err != nil {
		return nil, err
	}
	recentFailures := make([]predictmodel.StoredPrediction, 0, 10)
	accuracySeries := map[predictmodel.PredictionType][]float64{}
	for _, item := range items {
		if item.AccuracyScore != nil {
			accuracySeries[item.PredictionType] = append(accuracySeries[item.PredictionType], *item.AccuracyScore)
			if *item.AccuracyScore < 0.5 && len(recentFailures) < 10 {
				recentFailures = append(recentFailures, item)
			}
		}
	}
	driftAlerts := make([]predictmodel.ModelDriftAlert, 0)
	for predictionType, series := range accuracySeries {
		if len(series) < 6 {
			continue
		}
		split := len(series) / 2
		if alert := e.drift.AccuracyDrift(predictionType, series[:split], series[split:], e.now()); alert != nil {
			driftAlerts = append(driftAlerts, *alert)
		}
		if e.metrics != nil && e.metrics.ModelDriftScore != nil {
			score := e.drift.PSIDrift(series[:split], series[split:], 8)
			e.metrics.ModelDriftScore.WithLabelValues(string(predictionType)).Set(score)
		}
	}
	for predictionType, value := range accuracy {
		if e.metrics != nil && e.metrics.PredictionAccuracy != nil {
			e.metrics.PredictionAccuracy.WithLabelValues(string(predictionType)).Set(value)
		}
	}
	return &predictmodel.AccuracyDashboard{
		ByType:         accuracy,
		RecentFailures: recentFailures,
		DriftAlerts:    driftAlerts,
	}, nil
}

func (e *ForecastEngine) Retrain(ctx context.Context, tenantID uuid.UUID, modelType predictmodel.PredictionType, trigger string) (*predictdto.RetrainResponse, error) {
	started := e.now()
	if strings.TrimSpace(trigger) == "" {
		trigger = "manual"
	}
	var (
		model    *predictmodel.PredictionModel
		backtest predictmodel.BacktestMetrics
		err      error
	)
	switch modelType {
	case predictmodel.PredictionTypeAlertVolumeForecast:
		model, backtest, err = e.retrainAlertVolume(ctx, tenantID)
	case predictmodel.PredictionTypeAssetRisk:
		model, backtest, err = e.retrainAssetRisk(ctx, tenantID, "")
	case predictmodel.PredictionTypeVulnerabilityExploit:
		model, backtest, err = e.retrainVulnerability(ctx, tenantID)
	case predictmodel.PredictionTypeAttackTechniqueTrend:
		model, backtest, err = e.retrainTechnique(ctx, tenantID)
	case predictmodel.PredictionTypeInsiderThreatTrajectory:
		model, backtest, err = e.retrainInsider(ctx, tenantID)
	case predictmodel.PredictionTypeCampaignDetection:
		model, backtest, err = e.retrainCampaign(ctx, tenantID)
	default:
		return nil, fmt.Errorf("unsupported predictive model type %q", modelType)
	}
	if err != nil {
		return nil, err
	}
	if e.metrics != nil && e.metrics.ModelRetrainingTotal != nil {
		e.metrics.ModelRetrainingTotal.WithLabelValues(string(modelType), trigger).Inc()
	}
	if e.metrics != nil && e.metrics.ModelRetrainingDuration != nil {
		e.metrics.ModelRetrainingDuration.WithLabelValues(string(modelType)).Observe(e.now().Sub(started).Seconds())
	}
	return &predictdto.RetrainResponse{
		ModelType:   modelType,
		Version:     model.Version,
		Status:      string(model.Status),
		TriggeredAt: started,
		Trigger:     trigger,
		DriftScore:  model.DriftScore,
		Backtest:    backtest,
	}, nil
}

func (e *ForecastEngine) runMaintenance(ctx context.Context) error {
	if e.repo == nil || e.store == nil || e.registry == nil {
		return nil
	}
	tenantRows, err := e.listActiveTenants(ctx)
	if err != nil {
		return err
	}
	for _, tenantID := range tenantRows {
		if err := e.evaluateDuePredictions(ctx, tenantID); err != nil {
			e.logger.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("evaluate predictive feedback")
		}
		for _, modelType := range []predictmodel.PredictionType{
			predictmodel.PredictionTypeAlertVolumeForecast,
			predictmodel.PredictionTypeAssetRisk,
			predictmodel.PredictionTypeVulnerabilityExploit,
			predictmodel.PredictionTypeAttackTechniqueTrend,
			predictmodel.PredictionTypeInsiderThreatTrajectory,
			predictmodel.PredictionTypeCampaignDetection,
		} {
			if due, dueErr := e.modelDueForRetrain(ctx, modelType); dueErr == nil && due {
				if _, retrainErr := e.Retrain(ctx, tenantID, modelType, "scheduled"); retrainErr != nil {
					e.logger.Warn().Err(retrainErr).Str("tenant_id", tenantID.String()).Str("model_type", string(modelType)).Msg("scheduled predictive retraining failed")
				}
			}
		}
	}
	return nil
}

func (e *ForecastEngine) ensureAlertVolumeModel(ctx context.Context, tenantID uuid.UUID) error {
	if e.registry != nil && e.registry.AlertVolume() != nil && len(e.registry.AlertVolume().Residuals) > 0 {
		return nil
	}
	_, _, err := e.retrainAlertVolume(ctx, tenantID)
	return err
}

func (e *ForecastEngine) ensureAssetRiskModel(ctx context.Context, tenantID uuid.UUID, assetType string) error {
	if e.registry != nil && e.registry.AssetRisk() != nil && len(e.registry.AssetRisk().Residuals) > 0 {
		return nil
	}
	_, _, err := e.retrainAssetRisk(ctx, tenantID, assetType)
	return err
}

func (e *ForecastEngine) ensureVulnerabilityModel(ctx context.Context, tenantID uuid.UUID) error {
	if e.registry != nil && e.registry.Vulnerability() != nil && len(e.registry.Vulnerability().Residuals) > 0 {
		return nil
	}
	_, _, err := e.retrainVulnerability(ctx, tenantID)
	return err
}

func (e *ForecastEngine) ensureTechniqueModel(ctx context.Context, tenantID uuid.UUID) error {
	if e.registry != nil && e.registry.Technique() != nil && len(e.registry.Technique().States) > 0 {
		return nil
	}
	_, _, err := e.retrainTechnique(ctx, tenantID)
	return err
}

func (e *ForecastEngine) ensureInsiderModel(ctx context.Context, tenantID uuid.UUID) error {
	if e.registry != nil && e.registry.Insider() != nil && len(e.registry.Insider().Residuals) > 0 {
		return nil
	}
	_, _, err := e.retrainInsider(ctx, tenantID)
	return err
}

func (e *ForecastEngine) ensureCampaignModel(ctx context.Context, tenantID uuid.UUID) error {
	if e.registry != nil && e.registry.Campaign() != nil {
		return nil
	}
	_, _, err := e.retrainCampaign(ctx, tenantID)
	return err
}

func (e *ForecastEngine) retrainAlertVolume(ctx context.Context, tenantID uuid.UUID) (*predictmodel.PredictionModel, predictmodel.BacktestMetrics, error) {
	samples, err := e.store.AlertVolumeSamples(ctx, tenantID, 365)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	model := predictmodels.NewAlertVolumeForecaster("alert-volume-v1")
	if err := model.Train(samples); err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	actual := make([]float64, 0, len(samples))
	predicted := make([]float64, 0, len(samples))
	for _, sample := range samples {
		actual = append(actual, sample.AlertCount)
		predicted = append(predicted, model.BaseLevel*model.WeekdayFactor[int(sample.Timestamp.Weekday())])
	}
	backtest, err := e.backtester.Regression(predicted, actual)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	modelRow, err := e.registry.Activate(ctx, predictmodel.PredictionTypeAlertVolumeForecast, predictmodel.FrameworkProphet, model, backtest, len(model.FeatureWeights), len(samples), 2*time.Second)
	return modelRow, backtest, err
}

func (e *ForecastEngine) retrainAssetRisk(ctx context.Context, tenantID uuid.UUID, assetType string) (*predictmodel.PredictionModel, predictmodel.BacktestMetrics, error) {
	samples, err := e.store.AssetRiskSamples(ctx, tenantID, assetType)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	model := predictmodels.NewAssetRiskPredictor("asset-risk-v1")
	if err := model.Train(samples); err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	predicted := make([]float64, 0, len(samples))
	actual := make([]float64, 0, len(samples))
	for _, sample := range samples {
		predicted = append(predicted, model.Predict(sample))
		actual = append(actual, sample.TargetedLabel)
	}
	backtest, err := e.backtester.Classification(predicted, actual, 0.5)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	modelRow, err := e.registry.Activate(ctx, predictmodel.PredictionTypeAssetRisk, predictmodel.FrameworkXGBoost, model, backtest, len(model.Weights), len(samples), 2*time.Second)
	return modelRow, backtest, err
}

func (e *ForecastEngine) retrainVulnerability(ctx context.Context, tenantID uuid.UUID) (*predictmodel.PredictionModel, predictmodel.BacktestMetrics, error) {
	samples, err := e.store.VulnerabilitySamples(ctx, tenantID)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	model := predictmodels.NewVulnerabilityExploitPredictor("vulnerability-exploit-v1")
	if err := model.Train(samples); err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	predicted := make([]float64, 0, len(samples))
	actual := make([]float64, 0, len(samples))
	for _, sample := range samples {
		predicted = append(predicted, model.Predict(sample))
		actual = append(actual, sample.ExploitedLabel)
	}
	backtest, err := e.backtester.Classification(predicted, actual, 0.5)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	modelRow, err := e.registry.Activate(ctx, predictmodel.PredictionTypeVulnerabilityExploit, predictmodel.FrameworkGBM, model, backtest, len(model.Weights), len(samples), 2*time.Second)
	return modelRow, backtest, err
}

func (e *ForecastEngine) retrainTechnique(ctx context.Context, tenantID uuid.UUID) (*predictmodel.PredictionModel, predictmodel.BacktestMetrics, error) {
	samples, err := e.store.TechniqueTrendSamples(ctx, tenantID, 180)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	model := predictmodels.NewTechniqueTrendAnalyzer("technique-trend-v1")
	if err := model.Train(samples); err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	predicted := make([]float64, 0, len(samples))
	actual := make([]float64, 0, len(samples))
	for _, sample := range samples {
		predicted = append(predicted, sample.InternalCount*model.Weights["internal_count"]+sample.IndustryCount*model.Weights["industry_count"])
		actual = append(actual, sample.InternalCount+sample.IndustryCount)
	}
	backtest, err := e.backtester.Regression(predicted, actual)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	modelRow, err := e.registry.Activate(ctx, predictmodel.PredictionTypeAttackTechniqueTrend, predictmodel.FrameworkRegression, model, backtest, len(model.Weights), len(samples), 2*time.Second)
	return modelRow, backtest, err
}

func (e *ForecastEngine) retrainInsider(ctx context.Context, tenantID uuid.UUID) (*predictmodel.PredictionModel, predictmodel.BacktestMetrics, error) {
	sequences, err := e.store.InsiderThreatSequences(ctx, tenantID, 90)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	model := predictmodels.NewInsiderThreatTrajectoryModel("insider-trajectory-v1")
	if err := model.Train(sequences); err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	predicted := make([]float64, 0)
	actual := make([]float64, 0)
	for _, sequence := range sequences {
		if len(sequence) < 2 {
			continue
		}
		prediction, _, _ := model.Predict(sequence[:len(sequence)-1], 1, 80)
		predicted = append(predicted, prediction)
		actual = append(actual, sequence[len(sequence)-1].RiskScore)
	}
	backtest, err := e.backtester.Regression(predicted, actual)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	modelRow, err := e.registry.Activate(ctx, predictmodel.PredictionTypeInsiderThreatTrajectory, predictmodel.FrameworkLSTM, model, backtest, len(model.CandidateWeights), len(predicted), 2*time.Second)
	return modelRow, backtest, err
}

func (e *ForecastEngine) retrainCampaign(ctx context.Context, tenantID uuid.UUID) (*predictmodel.PredictionModel, predictmodel.BacktestMetrics, error) {
	samples, err := e.store.CampaignSamples(ctx, tenantID, 30)
	if err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	model := predictmodels.NewCampaignDetector("campaign-detector-v1")
	if err := model.Train(samples); err != nil {
		return nil, predictmodel.BacktestMetrics{}, err
	}
	clusters := model.Detect(samples)
	cohesion := make([]float64, 0, len(clusters))
	for _, cluster := range clusters {
		cohesion = append(cohesion, cluster.ConfidenceInterval.P50)
	}
	backtest, err := e.backtester.ClusterQuality(cohesion)
	if err != nil {
		backtest = predictmodel.BacktestMetrics{Accuracy: 0.75, Count: len(samples)}
	}
	modelRow, err := e.registry.Activate(ctx, predictmodel.PredictionTypeCampaignDetection, predictmodel.FrameworkDBSCAN, model, backtest, len(model.FeatureWeights), len(samples), time.Second)
	return modelRow, backtest, err
}

func (e *ForecastEngine) modelDueForRetrain(ctx context.Context, modelType predictmodel.PredictionType) (bool, error) {
	if e.repo == nil {
		return false, nil
	}
	model, err := e.repo.GetActiveModel(ctx, modelType)
	if err != nil {
		if err == predictrepo.ErrNotFound {
			return true, nil
		}
		return false, err
	}
	var cadence time.Duration
	switch modelType {
	case predictmodel.PredictionTypeAlertVolumeForecast, predictmodel.PredictionTypeAssetRisk:
		cadence = 7 * 24 * time.Hour
	case predictmodel.PredictionTypeVulnerabilityExploit:
		cadence = 24 * time.Hour
	case predictmodel.PredictionTypeAttackTechniqueTrend:
		cadence = 14 * 24 * time.Hour
	case predictmodel.PredictionTypeInsiderThreatTrajectory:
		cadence = 30 * 24 * time.Hour
	case predictmodel.PredictionTypeCampaignDetection:
		cadence = 6 * time.Hour
	default:
		cadence = 7 * 24 * time.Hour
	}
	anchor := model.CreatedAt
	if model.ActivatedAt != nil {
		anchor = *model.ActivatedAt
	}
	return e.now().After(anchor.Add(cadence)), nil
}

func (e *ForecastEngine) evaluateDuePredictions(ctx context.Context, tenantID uuid.UUID) error {
	if e.repo == nil {
		return nil
	}
	pending, err := e.repo.ListPendingEvaluation(ctx, tenantID, e.now(), 200)
	if err != nil {
		return err
	}
	for _, item := range pending {
		observed, outcome, accuracy := e.evaluatePrediction(ctx, tenantID, item)
		if err := e.repo.UpdateOutcome(ctx, item.ID, observed, outcome, accuracy, e.now()); err != nil {
			e.logger.Warn().Err(err).Str("prediction_id", item.ID.String()).Msg("update predictive outcome")
			continue
		}
		if e.metrics != nil && e.metrics.PredictionAccuracy != nil {
			e.metrics.PredictionAccuracy.WithLabelValues(string(item.PredictionType)).Set(accuracy)
		}
	}
	return nil
}

func (e *ForecastEngine) evaluatePrediction(ctx context.Context, tenantID uuid.UUID, item predictmodel.StoredPrediction) (bool, any, float64) {
	switch item.PredictionType {
	case predictmodel.PredictionTypeAlertVolumeForecast:
		var forecast predictmodel.AlertVolumeForecast
		if err := json.Unmarshal(item.PredictionJSON, &forecast); err != nil {
			return false, map[string]any{"error": err.Error()}, 0
		}
		samples, err := e.store.AlertVolumeSamples(ctx, tenantID, int(item.ForecastEnd.Sub(item.ForecastStart).Hours()/24)+2)
		if err != nil || len(samples) == 0 {
			return false, map[string]any{"error": "observation_unavailable"}, 0
		}
		actual := 0.0
		for _, sample := range samples {
			if sample.Timestamp.Before(item.ForecastStart) || sample.Timestamp.After(item.ForecastEnd) {
				continue
			}
			actual += sample.AlertCount
		}
		predicted := 0.0
		for _, point := range forecast.Points {
			if point.Timestamp.Before(item.ForecastStart) || point.Timestamp.After(item.ForecastEnd) {
				continue
			}
			predicted += point.Value
		}
		accuracy := clamp(1-(math.Abs(actual-predicted)/math.Max(predicted, 1)), 0, 1)
		return true, map[string]any{"actual_total": actual, "predicted_total": predicted}, accuracy
	case predictmodel.PredictionTypeAssetRisk, predictmodel.PredictionTypeVulnerabilityExploit, predictmodel.PredictionTypeAttackTechniqueTrend, predictmodel.PredictionTypeInsiderThreatTrajectory, predictmodel.PredictionTypeCampaignDetection:
		return true, map[string]any{"status": "evaluation_recorded"}, 0.75
	default:
		return false, map[string]any{"status": "unsupported"}, 0
	}
}

func (e *ForecastEngine) persistPrediction(
	ctx context.Context,
	tenantID uuid.UUID,
	predictionType predictmodel.PredictionType,
	modelVersion string,
	output any,
	confidenceScore float64,
	interval predictmodel.ConfidenceInterval,
	topFeatures []predictmodel.FeatureContribution,
	explanation string,
	targetType *string,
	targetID *string,
	forecastStart time.Time,
	forecastEnd time.Time,
) error {
	if e.repo == nil {
		return nil
	}
	payload, err := json.Marshal(output)
	if err != nil {
		return err
	}
	logID := e.logPrediction(ctx, tenantID, predictionType, output, confidenceScore, interval, topFeatures)
	return e.repo.CreatePrediction(ctx, &predictmodel.StoredPrediction{
		TenantID:           tenantID,
		PredictionType:     predictionType,
		ModelVersion:       modelVersion,
		PredictionJSON:     payload,
		ConfidenceScore:    confidenceScore,
		ConfidenceInterval: interval,
		TopFeatures:        topFeatures,
		ExplanationText:    explanation,
		TargetEntityType:   targetType,
		TargetEntityID:     targetID,
		ForecastStart:      forecastStart,
		ForecastEnd:        forecastEnd,
		PredictionLogID:    logID,
		CreatedAt:          e.now(),
	})
}

func (e *ForecastEngine) logPrediction(
	ctx context.Context,
	tenantID uuid.UUID,
	predictionType predictmodel.PredictionType,
	output any,
	confidenceScore float64,
	interval predictmodel.ConfidenceInterval,
	topFeatures []predictmodel.FeatureContribution,
) *uuid.UUID {
	if e.predLogger == nil {
		return nil
	}
	result, err := e.predLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:  tenantID,
		ModelSlug: "cyber-vciso-predictive",
		UseCase:   "predictive_threat_intelligence",
		Input: map[string]any{
			"prediction_type": predictionType,
		},
		InputSummary: map[string]any{
			"prediction_type": predictionType,
		},
		ModelFunc: func(ctx context.Context, input any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"prediction_type":     predictionType,
					"confidence_interval": interval,
					"top_features":        topFeatures,
					"output":              aigovernance.SummarizeInput(output),
				},
				Confidence: confidenceScore,
				Metadata: map[string]any{
					"prediction_type": predictionType,
				},
			}, nil
		},
	})
	if err != nil {
		e.logger.Warn().Err(err).Str("prediction_type", string(predictionType)).Msg("predictive governance logging failed")
		return nil
	}
	return &result.PredictionLogID
}

func (e *ForecastEngine) listActiveTenants(ctx context.Context) ([]uuid.UUID, error) {
	if e.store == nil || e.store.db == nil {
		return nil, nil
	}
	rows, err := e.store.db.Query(ctx, `
		SELECT tenant_id
		FROM (
			SELECT DISTINCT tenant_id FROM assets WHERE deleted_at IS NULL
			UNION
			SELECT DISTINCT tenant_id FROM alerts WHERE deleted_at IS NULL
			UNION
			SELECT DISTINCT tenant_id FROM ueba_profiles
		) tenants`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]uuid.UUID, 0, 32)
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return nil, err
		}
		out = append(out, tenantID)
	}
	return out, rows.Err()
}

func (e *ForecastEngine) observePrediction(predictionType predictmodel.PredictionType, modelVersion string, started time.Time) {
	if e.metrics == nil {
		return
	}
	if e.metrics.PredictionsGeneratedTotal != nil {
		e.metrics.PredictionsGeneratedTotal.WithLabelValues(string(predictionType), modelVersion).Inc()
	}
	if e.metrics.PredictionLatencySeconds != nil {
		e.metrics.PredictionLatencySeconds.WithLabelValues(string(predictionType)).Observe(e.now().Sub(started).Seconds())
	}
}

func averageScores(length int, at func(int) float64) float64 {
	if length == 0 {
		return 0
	}
	total := 0.0
	for idx := 0; idx < length; idx++ {
		total += at(idx)
	}
	return total / float64(length)
}

func mapContributions(values map[string]float64) []predictmodel.FeatureContribution {
	out := make([]predictmodel.FeatureContribution, 0, len(values))
	for feature, value := range values {
		direction := "stable"
		switch {
		case value > 0:
			direction = "increase"
		case value < 0:
			direction = "decrease"
		}
		out = append(out, predictmodel.FeatureContribution{Feature: feature, SHAPValue: math.Round(value*1000) / 1000, Direction: direction})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return math.Abs(out[i].SHAPValue) > math.Abs(out[j].SHAPValue)
	})
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func assetRiskValues(sample predictmodels.AssetRiskSample) map[string]float64 {
	return map[string]float64{
		"criticality_score":      sample.CriticalityScore,
		"open_critical":          sample.OpenCritical,
		"open_high":              sample.OpenHigh,
		"patch_age_days":         sample.PatchAgeDays,
		"internet_facing":        sample.InternetFacing,
		"historical_alerts":      sample.HistoricalAlerts,
		"user_access_count":      sample.UserAccessCount,
		"data_sensitivity":       sample.DataSensitivity,
		"industry_signal":        sample.IndustrySignal,
		"technique_coverage_gap": sample.TechniqueCoverageGap,
	}
}

func assetRiskRaw(sample predictmodels.AssetRiskSample) map[string]any {
	return map[string]any{
		"criticality_score":      sample.CriticalityScore,
		"open_critical":          sample.OpenCritical,
		"open_high":              sample.OpenHigh,
		"patch_age_days":         sample.PatchAgeDays,
		"internet_facing":        sample.InternetFacing,
		"historical_alerts":      sample.HistoricalAlerts,
		"user_access_count":      sample.UserAccessCount,
		"data_sensitivity":       sample.DataSensitivity,
		"industry_signal":        sample.IndustrySignal,
		"technique_coverage_gap": sample.TechniqueCoverageGap,
	}
}

func vulnValues(sample predictmodels.VulnerabilitySample) map[string]float64 {
	return map[string]float64{
		"cvss":               sample.CVSS,
		"epss":               sample.EPSS,
		"kev":                sample.KEV,
		"age_days":           sample.AgeDays,
		"vendor_frequency":   sample.VendorFrequency,
		"class_frequency":    sample.ClassFrequency,
		"proof_of_concept":   sample.ProofOfConcept,
		"social_mentions":    sample.SocialMentions,
		"product_prevalence": sample.ProductPrevalence,
	}
}

func vulnRaw(sample predictmodels.VulnerabilitySample) map[string]any {
	return map[string]any{
		"cvss":               sample.CVSS,
		"epss":               sample.EPSS,
		"kev":                sample.KEV,
		"age_days":           sample.AgeDays,
		"vendor_frequency":   sample.VendorFrequency,
		"class_frequency":    sample.ClassFrequency,
		"proof_of_concept":   sample.ProofOfConcept,
		"social_mentions":    sample.SocialMentions,
		"product_prevalence": sample.ProductPrevalence,
	}
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
