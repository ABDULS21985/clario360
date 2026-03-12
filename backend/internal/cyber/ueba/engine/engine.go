package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	cybermodel "github.com/clario360/platform/internal/cyber/model"
	cyberrepo "github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/ueba/collector"
	"github.com/clario360/platform/internal/cyber/ueba/correlator"
	"github.com/clario360/platform/internal/cyber/ueba/detector"
	"github.com/clario360/platform/internal/cyber/ueba/model"
	"github.com/clario360/platform/internal/cyber/ueba/profiler"
	uebarepo "github.com/clario360/platform/internal/cyber/ueba/repository"
	"github.com/clario360/platform/internal/cyber/ueba/scorer"
	"github.com/clario360/platform/internal/events"
)

const (
	maxSchedulerJitter = 30 * time.Second
	decayRedisPrefix   = "cyber:ueba:decay:"
)

type CyberAlertCreator interface {
	Create(ctx context.Context, alert *cybermodel.Alert) (*cybermodel.Alert, error)
}

type governedPredictor interface {
	Predict(ctx context.Context, params aigovernance.PredictParams) (*aigovernance.PredictionResult, error)
}

type UEBAEngine struct {
	collector   *collector.AccessEventCollector
	profileRepo *uebarepo.ProfileRepository
	eventRepo   *uebarepo.EventRepository
	alertRepo   *uebarepo.AlertRepository
	cyberAlerts CyberAlertCreator
	predLogger  governedPredictor
	producer    *events.Producer
	configStore *ConfigStore
	redis       *redis.Client
	metrics     *UEBAMetrics
	logger      zerolog.Logger
	runCycle    func(ctx context.Context, tenantID uuid.UUID, cfg UEBAConfig) (*cycleResult, error)

	mu         sync.RWMutex
	profiler   *profiler.BehavioralProfiler
	detector   *detector.AnomalyDetector
	correlator *correlator.AnomalyCorrelator
	riskScorer *scorer.EntityRiskScorer
}

type cycleResult struct {
	CollectedEvents int
	ProfilesUpdated int
	SignalsCreated  int
	AlertsCreated   int
	EntitiesScored  int
	Confidence      float64
}

type profileState struct {
	Profile        *model.UEBAProfile
	BeforeMaturity model.ProfileMaturity
	BeforeRisk     float64
	Created        bool
}

func NewEngine(
	collector *collector.AccessEventCollector,
	profileRepo *uebarepo.ProfileRepository,
	eventRepo *uebarepo.EventRepository,
	alertRepo *uebarepo.AlertRepository,
	cyberAlerts CyberAlertCreator,
	predLogger *aigovmiddleware.PredictionLogger,
	producer *events.Producer,
	configStore *ConfigStore,
	redisClient *redis.Client,
	metrics *UEBAMetrics,
	logger zerolog.Logger,
) *UEBAEngine {
	if configStore == nil {
		configStore = NewConfigStore(context.Background(), redisClient, logger)
	}
	engine := &UEBAEngine{
		collector:   collector,
		profileRepo: profileRepo,
		eventRepo:   eventRepo,
		alertRepo:   alertRepo,
		cyberAlerts: cyberAlerts,
		predLogger:  predLogger,
		producer:    producer,
		configStore: configStore,
		redis:       redisClient,
		metrics:     metrics,
		logger:      logger.With().Str("component", "ueba-engine").Logger(),
	}
	engine.runCycle = engine.processCycle
	engine.applyConfig(configStore.Snapshot())
	return engine
}

func (e *UEBAEngine) Config() UEBAConfig {
	return e.configStore.Snapshot()
}

func (e *UEBAEngine) UpdateConfig(ctx context.Context, cfg UEBAConfig) (UEBAConfig, error) {
	updated, err := e.configStore.Update(ctx, cfg)
	if err != nil {
		return UEBAConfig{}, err
	}
	e.applyConfig(updated)
	return updated, nil
}

func (e *UEBAEngine) applyConfig(cfg UEBAConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.profiler = profiler.NewBehavioralProfiler(cfg.EMAAlpha)
	e.detector = detector.New(detector.Config{
		MinMaturityForAlert:         cfg.MinMaturityForAlert,
		UnusualTimeMatureHighProb:   cfg.UnusualTimeMatureHighProb,
		UnusualTimeMatureMediumProb: cfg.UnusualTimeMatureMediumProb,
		UnusualTimeBaseHighProb:     cfg.UnusualTimeBaseHighProb,
		UnusualTimeBaseMediumProb:   cfg.UnusualTimeBaseMediumProb,
		UnusualVolumeMediumZ:        cfg.UnusualVolumeMediumZ,
		UnusualVolumeHighZ:          cfg.UnusualVolumeHighZ,
		UnusualVolumeCriticalZ:      cfg.UnusualVolumeCriticalZ,
		UnusualVolumeStddevMin:      cfg.UnusualVolumeStddevMin,
		FailureSpikeMediumZ:         cfg.FailureSpikeMediumZ,
		FailureSpikeHighZ:           cfg.FailureSpikeHighZ,
		FailureSpikeCriticalZ:       cfg.FailureSpikeCriticalZ,
		FailureStddevMin:            cfg.FailureStddevMin,
		FailureCriticalCount:        cfg.FailureCriticalCount,
		BulkRowsMediumMultiplier:    cfg.BulkRowsMediumMultiplier,
		BulkRowsHighMultiplier:      cfg.BulkRowsHighMultiplier,
		DDLUnusualThreshold:         cfg.DDLUnusualThreshold,
	}, nil)
	e.correlator = correlator.New(cfg.CorrelationWindow)
	e.riskScorer = scorer.New(e.profileRepo, e.alertRepo, cfg.RiskDecayRatePerDay, e.logger)
}

func (e *UEBAEngine) Run(ctx context.Context, tenantID uuid.UUID) error {
	timer := time.NewTimer(randomJitter())
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			if err := e.ProcessCycle(ctx, tenantID); err != nil && !errors.Is(err, context.Canceled) {
				e.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("ueba cycle failed")
			}
			next := e.Config().CycleInterval
			if next <= 0 {
				next = 5 * time.Minute
			}
			timer.Reset(next + randomJitter())
		}
	}
}

func (e *UEBAEngine) ProcessCycle(ctx context.Context, tenantID uuid.UUID) error {
	cfg := e.Config()
	cycleCtx, cancel := context.WithTimeout(ctx, cfg.MaxProcessingTime)
	defer cancel()

	start := time.Now()
	status := "success"
	_, err := e.runGovernedCycle(cycleCtx, tenantID, cfg)
	if err != nil {
		status = "error"
		if errors.Is(err, context.DeadlineExceeded) || cycleCtx.Err() == context.DeadlineExceeded {
			status = "timeout"
		}
	}
	if e.metrics != nil {
		tenant := tenantID.String()
		e.metrics.EngineCyclesTotal.WithLabelValues(tenant, status).Inc()
		e.metrics.EngineCycleDurationSeconds.WithLabelValues(tenant).Observe(time.Since(start).Seconds())
	}
	return err
}

func (e *UEBAEngine) runGovernedCycle(ctx context.Context, tenantID uuid.UUID, cfg UEBAConfig) (*cycleResult, error) {
	if e.predLogger == nil {
		return e.runCycle(ctx, tenantID, cfg)
	}
	result, err := e.predLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:   tenantID,
		ModelSlug:  "cyber-ueba-detector",
		UseCase:    "behavioral_anomaly_detection",
		EntityType: "tenant",
		EntityID:   &tenantID,
		Input: map[string]any{
			"cycle_interval_seconds":         int(cfg.CycleInterval.Seconds()),
			"max_events_per_cycle":           cfg.MaxEventsPerCycle,
			"max_processing_time_seconds":    int(cfg.MaxProcessingTime.Seconds()),
			"ema_alpha":                      cfg.EMAAlpha,
			"min_maturity_for_alert":         cfg.MinMaturityForAlert,
			"correlation_window_seconds":     int(cfg.CorrelationWindow.Seconds()),
			"risk_decay_rate_per_day":        cfg.RiskDecayRatePerDay,
			"batch_size":                     cfg.BatchSize,
			"precision_bias":                 "high",
			"immutability":                   "persistent_profiles_and_event_evidence",
			"cross_tenant_controls":          []string{"rls", "tenant_predicates"},
			"immature_profiles_do_not_alert": true,
		},
		InputSummary: map[string]any{
			"model_slug":               "cyber-ueba-detector",
			"tenant_id":                tenantID.String(),
			"correlation_window_hours": cfg.CorrelationWindow.Hours(),
			"max_events_per_cycle":     cfg.MaxEventsPerCycle,
		},
		ModelFunc: func(ctx context.Context, _ any) (*aigovernance.ModelOutput, error) {
			outcome, err := e.runCycle(ctx, tenantID, cfg)
			if err != nil {
				return nil, err
			}
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"events_processed":  outcome.CollectedEvents,
					"profiles_updated":  outcome.ProfilesUpdated,
					"signals_generated": outcome.SignalsCreated,
					"alerts_created":    outcome.AlertsCreated,
					"entities_scored":   outcome.EntitiesScored,
				},
				Confidence: outcome.Confidence,
				Metadata: map[string]any{
					"tenant_id": tenantID.String(),
				},
			}, nil
		},
	})
	if err != nil {
		if errors.Is(err, aigovmiddleware.ErrGovernanceUnavailable) {
			e.logger.Warn().
				Err(err).
				Str("tenant_id", tenantID.String()).
				Msg("ai governance unavailable, continuing ueba cycle without prediction logging")
			return e.runCycle(ctx, tenantID, cfg)
		}
		return nil, err
	}

	output := cycleResult{}
	if payload, ok := result.Output.(map[string]any); ok {
		output.CollectedEvents = int(numberValue(payload["events_processed"]))
		output.ProfilesUpdated = int(numberValue(payload["profiles_updated"]))
		output.SignalsCreated = int(numberValue(payload["signals_generated"]))
		output.AlertsCreated = int(numberValue(payload["alerts_created"]))
		output.EntitiesScored = int(numberValue(payload["entities_scored"]))
		output.Confidence = result.Confidence
	}
	return &output, nil
}

func (e *UEBAEngine) processCycle(ctx context.Context, tenantID uuid.UUID, cfg UEBAConfig) (*cycleResult, error) {
	tenantLabel := tenantID.String()
	result := &cycleResult{}

	collectCtx, collectCancel := context.WithTimeout(ctx, 15*time.Second)
	eventsBatch, err := e.collector.CollectSinceLastRun(collectCtx, tenantID, cfg.MaxEventsPerCycle)
	collectCancel()
	if err != nil {
		return nil, fmt.Errorf("collect ueba events: %w", err)
	}
	if len(eventsBatch) == 0 {
		if err := e.runDailyDecayIfDue(ctx, tenantID, cfg); err != nil {
			return nil, err
		}
		result.Confidence = 0.95
		return result, nil
	}

	sourceCounts := make(map[string]int)
	for _, event := range eventsBatch {
		if event == nil {
			continue
		}
		event.TenantID = tenantID
		event.EventTimestamp = event.EventTimestamp.UTC()
		sourceCounts[event.SourceType]++
	}
	if err := e.eventRepo.InsertBatch(ctx, tenantID, eventsBatch); err != nil {
		return nil, fmt.Errorf("persist ueba events: %w", err)
	}
	for sourceType, count := range sourceCounts {
		if e.metrics != nil {
			e.metrics.EventsCollectedTotal.WithLabelValues(tenantLabel, sourceType).Add(float64(count))
		}
	}
	result.CollectedEvents = len(eventsBatch)

	profilerInstance, detectorInstance, correlatorInstance, riskScorerInstance := e.components()
	if profilerInstance == nil || detectorInstance == nil || correlatorInstance == nil || riskScorerInstance == nil {
		return nil, fmt.Errorf("ueba engine components are not initialized")
	}

	states := make(map[string]*profileState)
	grouped := groupEventsByEntity(eventsBatch)

	updateCtx, updateCancel := context.WithTimeout(ctx, 20*time.Second)
	for key, entityEvents := range grouped {
		first := entityEvents[0]
		profileRecord, err := e.profileRepo.GetOrCreate(updateCtx, tenantID, first.EntityType, first.EntityID, first.EntityID, "")
		if err != nil {
			updateCancel()
			return nil, fmt.Errorf("load ueba profile %s: %w", first.EntityID, err)
		}
		state := &profileState{
			Profile:        profileRecord,
			BeforeMaturity: profileRecord.ProfileMaturity,
			BeforeRisk:     profileRecord.RiskScore,
			Created:        profileRecord.ObservationCount == 0,
		}
		for _, event := range entityEvents {
			if err := profilerInstance.UpdateProfile(profileRecord, event); err != nil {
				updateCancel()
				return nil, fmt.Errorf("update ueba profile %s: %w", first.EntityID, err)
			}
		}
		if err := e.profileRepo.Update(updateCtx, profileRecord); err != nil {
			updateCancel()
			return nil, fmt.Errorf("persist ueba profile %s: %w", first.EntityID, err)
		}
		states[key] = state
		result.ProfilesUpdated++

		if state.Created {
			_ = e.publishUEBAEvent(updateCtx, tenantID, "com.clario360.cyber.ueba.profile.created", map[string]any{
				"entity_type": profileRecord.EntityType,
				"entity_id":   profileRecord.EntityID,
			})
		}
		if state.BeforeMaturity != model.ProfileMaturityMature && profileRecord.ProfileMaturity == model.ProfileMaturityMature {
			_ = e.publishUEBAEvent(updateCtx, tenantID, "com.clario360.cyber.ueba.profile.mature", map[string]any{
				"entity_type":       profileRecord.EntityType,
				"entity_id":         profileRecord.EntityID,
				"observation_count": profileRecord.ObservationCount,
				"profile_maturity":  profileRecord.ProfileMaturity,
				"days_active":       profileRecord.DaysActive,
			})
		}
	}
	updateCancel()
	if e.metrics != nil {
		e.metrics.ProfilesUpdatedTotal.WithLabelValues(tenantLabel).Add(float64(result.ProfilesUpdated))
	}

	entitySignals := make(map[string][]model.AnomalySignal)
	confidenceSum := 0.0
	for _, event := range eventsBatch {
		state := states[entityCompositeKey(event.EntityType, event.EntityID)]
		if state == nil || state.Profile == nil {
			continue
		}
		signals := detectorInstance.DetectAnomalies(ctx, event, state.Profile)
		if len(signals) == 0 {
			continue
		}
		event.AnomalySignals = signals
		event.AnomalyCount = len(signals)
		if err := e.eventRepo.UpdateAnomalyFlags(ctx, tenantID, event.ID, signals); err != nil {
			return nil, fmt.Errorf("persist ueba anomaly flags: %w", err)
		}
		entitySignals[event.EntityID] = append(entitySignals[event.EntityID], signals...)
		for _, signal := range signals {
			result.SignalsCreated++
			confidenceSum += signal.Confidence
			if e.metrics != nil {
				e.metrics.AnomalySignalsTotal.WithLabelValues(tenantLabel, string(signal.SignalType), signal.Severity).Inc()
			}
			_ = e.publishUEBAEvent(ctx, tenantID, "com.clario360.cyber.ueba.anomaly.detected", map[string]any{
				"entity_id":   event.EntityID,
				"signal_type": signal.SignalType,
				"severity":    signal.Severity,
				"confidence":  signal.Confidence,
				"event_id":    signal.EventID.String(),
			})
		}
	}

	scoringCandidates := make(map[string][]model.UEBAAlert)
	for entityID := range entitySignals {
		historicalSignals, err := e.eventRepo.ListSignalsWithinWindow(ctx, tenantID, entityID, time.Now().UTC().Add(-cfg.CorrelationWindow))
		if err != nil {
			return nil, fmt.Errorf("load recent ueba signals: %w", err)
		}
		alerts := correlatorInstance.Correlate(ctx, tenantID, entityID, historicalSignals)
		if len(alerts) == 0 {
			continue
		}
		if e.metrics != nil {
			e.metrics.AlertsCorrelatedTotal.WithLabelValues(tenantLabel).Inc()
		}

		profileState := states[entityCompositeKey(entityTypeForEntity(states, entityID), entityID)]
		for _, alertRecord := range alerts {
			if profileState != nil && profileState.Profile != nil {
				alertRecord.EntityType = profileState.Profile.EntityType
				alertRecord.EntityName = firstNonEmpty(profileState.Profile.EntityName, profileState.Profile.EntityID)
				alertRecord.Title = fmt.Sprintf("%s - %s", alertRecord.Title, alertRecord.EntityName)
			}

			persisted, err := e.upsertCorrelatedAlert(ctx, tenantID, alertRecord)
			if err != nil {
				return nil, err
			}
			scoringCandidates[entityID] = append(scoringCandidates[entityID], *persisted)
			result.AlertsCreated++

			if e.metrics != nil {
				e.metrics.AlertsCreatedTotal.WithLabelValues(tenantLabel, string(persisted.AlertType), persisted.Severity).Inc()
			}
			_ = e.publishUEBAEvent(ctx, tenantID, "com.clario360.cyber.ueba.alert.created", map[string]any{
				"entity_id":  persisted.EntityID,
				"alert_id":   persisted.ID.String(),
				"alert_type": persisted.AlertType,
				"severity":   persisted.Severity,
				"risk_delta": persisted.RiskScoreDelta,
			})
			if persisted.CyberAlertID != nil {
				_ = e.publishUEBAEvent(ctx, tenantID, "com.clario360.cyber.ueba.alert.escalated", map[string]any{
					"entity_id":      persisted.EntityID,
					"alert_id":       persisted.ID.String(),
					"cyber_alert_id": persisted.CyberAlertID.String(),
				})
			}
		}
	}

	for entityID, alerts := range scoringCandidates {
		before := 0.0
		if state := states[entityCompositeKey(entityTypeForEntity(states, entityID), entityID)]; state != nil {
			before = state.BeforeRisk
		}
		if err := riskScorerInstance.UpdateRiskScore(ctx, tenantID, entityID, alerts); err != nil {
			return nil, fmt.Errorf("update ueba risk score: %w", err)
		}
		updatedProfile, err := e.profileRepo.GetByEntity(ctx, tenantID, entityID)
		if err != nil {
			return nil, fmt.Errorf("reload scored ueba profile: %w", err)
		}
		result.EntitiesScored++
		if e.metrics != nil {
			e.metrics.RiskScoreUpdatesTotal.WithLabelValues(tenantLabel).Inc()
			e.metrics.RiskScoreDistribution.WithLabelValues(tenantLabel).Observe(updatedProfile.RiskScore)
		}
		if updatedProfile.RiskScore != before {
			_ = e.publishUEBAEvent(ctx, tenantID, "com.clario360.cyber.ueba.risk.changed", map[string]any{
				"entity_id": entityID,
				"old_score": before,
				"new_score": updatedProfile.RiskScore,
				"old_level": riskLevelForScore(before),
				"new_level": updatedProfile.RiskLevel,
			})
			if before < 75 && updatedProfile.RiskScore >= 75 {
				_ = e.publishUEBAEvent(ctx, tenantID, "com.clario360.cyber.ueba.risk.critical", map[string]any{
					"entity_id":  entityID,
					"risk_score": updatedProfile.RiskScore,
				})
			}
		}
	}

	if err := e.runDailyDecayIfDue(ctx, tenantID, cfg); err != nil {
		return nil, err
	}

	if result.SignalsCreated == 0 {
		result.Confidence = 0.90
	} else {
		result.Confidence = clampConfidence(confidenceSum / float64(result.SignalsCreated))
	}

	if latest := latestEventTime(eventsBatch); !latest.IsZero() {
		if err := e.collector.MarkRun(ctx, tenantID, latest); err != nil {
			e.logger.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("persist ueba collector cursor")
		}
	}
	return result, nil
}

func (e *UEBAEngine) upsertCorrelatedAlert(ctx context.Context, tenantID uuid.UUID, alertRecord model.UEBAAlert) (*model.UEBAAlert, error) {
	existing, err := e.alertRepo.FindRecentOpenByType(ctx, tenantID, alertRecord.EntityID, alertRecord.AlertType, time.Now().UTC().Add(-30*time.Minute))
	if err != nil && err != cyberrepo.ErrNotFound {
		return nil, fmt.Errorf("find recent ueba alert: %w", err)
	}
	if err == cyberrepo.ErrNotFound || existing == nil {
		if severityRank(alertRecord.Severity) >= severityRank("high") {
			cyberAlert, createErr := e.createCyberAlert(ctx, tenantID, &alertRecord)
			if createErr != nil {
				return nil, createErr
			}
			if cyberAlert != nil {
				alertRecord.CyberAlertID = &cyberAlert.ID
			}
		}
		return e.alertRepo.Create(ctx, &alertRecord)
	}

	merged := mergeCorrelatedAlerts(existing, &alertRecord)
	if merged.CyberAlertID == nil && severityRank(merged.Severity) >= severityRank("high") {
		cyberAlert, createErr := e.createCyberAlert(ctx, tenantID, merged)
		if createErr != nil {
			return nil, createErr
		}
		if cyberAlert != nil {
			merged.CyberAlertID = &cyberAlert.ID
		}
	}
	updated, updateErr := e.alertRepo.UpdateCorrelation(ctx, tenantID, merged)
	if updateErr != nil {
		return nil, fmt.Errorf("update ueba correlation alert: %w", updateErr)
	}
	return updated, nil
}

func (e *UEBAEngine) createCyberAlert(ctx context.Context, tenantID uuid.UUID, alertRecord *model.UEBAAlert) (*cybermodel.Alert, error) {
	if e.cyberAlerts == nil || alertRecord == nil {
		return nil, nil
	}

	metadata, _ := json.Marshal(map[string]any{
		"ueba_alert_type":      alertRecord.AlertType,
		"entity_type":          alertRecord.EntityType,
		"entity_id":            alertRecord.EntityID,
		"triggering_event_ids": alertRecord.TriggeringEventIDs,
		"baseline_comparison":  alertRecord.BaselineComparison,
	})
	cyberAlert := &cybermodel.Alert{
		TenantID:        tenantID,
		Title:           alertRecord.Title,
		Description:     alertRecord.Description,
		Severity:        cyberSeverity(alertRecord.Severity),
		Status:          cybermodel.AlertStatusNew,
		Source:          "ueba",
		Explanation:     buildCyberExplanation(alertRecord),
		ConfidenceScore: alertRecord.Confidence,
		EventCount:      len(alertRecord.TriggeringEventIDs),
		FirstEventAt:    alertRecord.CorrelationWindowStart,
		LastEventAt:     alertRecord.CorrelationWindowEnd,
		Tags:            []string{"ueba", string(alertRecord.AlertType)},
		Metadata:        metadata,
	}
	if len(alertRecord.MITRETechniqueIDs) > 0 {
		cyberAlert.MITRETechniqueID = &alertRecord.MITRETechniqueIDs[0]
		cyberAlert.MITRETechniqueName = &alertRecord.MITRETechniqueIDs[0]
	}
	if strings.TrimSpace(alertRecord.MITRETactic) != "" {
		tactic := alertRecord.MITRETactic
		cyberAlert.MITRETacticID = &tactic
		cyberAlert.MITRETacticName = &tactic
	}
	created, err := e.cyberAlerts.Create(ctx, cyberAlert)
	if err != nil {
		return nil, fmt.Errorf("create escalated cyber alert: %w", err)
	}
	return created, nil
}

func (e *UEBAEngine) runDailyDecayIfDue(ctx context.Context, tenantID uuid.UUID, cfg UEBAConfig) error {
	key := decayRedisPrefix + tenantID.String() + ":" + time.Now().UTC().Format("2006-01-02")
	if e.redis != nil {
		ok, err := e.redis.SetNX(ctx, key, "1", 25*time.Hour).Result()
		if err != nil {
			return fmt.Errorf("check ueba daily decay gate: %w", err)
		}
		if !ok {
			return nil
		}
	}
	return e.riskScorer.RunDailyDecay(ctx, tenantID)
}

func (e *UEBAEngine) publishUEBAEvent(ctx context.Context, tenantID uuid.UUID, eventType string, payload map[string]any) error {
	if e.producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "cyber-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return e.producer.Publish(ctx, events.Topics.UEBAEvents, event)
}

func (e *UEBAEngine) components() (*profiler.BehavioralProfiler, *detector.AnomalyDetector, *correlator.AnomalyCorrelator, *scorer.EntityRiskScorer) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.profiler, e.detector, e.correlator, e.riskScorer
}

func groupEventsByEntity(eventsBatch []*model.DataAccessEvent) map[string][]*model.DataAccessEvent {
	grouped := make(map[string][]*model.DataAccessEvent)
	for _, event := range eventsBatch {
		if event == nil {
			continue
		}
		key := entityCompositeKey(event.EntityType, event.EntityID)
		grouped[key] = append(grouped[key], event)
	}
	for key := range grouped {
		sort.SliceStable(grouped[key], func(i, j int) bool {
			return grouped[key][i].EventTimestamp.Before(grouped[key][j].EventTimestamp)
		})
	}
	return grouped
}

func entityCompositeKey(entityType model.EntityType, entityID string) string {
	return string(entityType) + "|" + entityID
}

func entityTypeForEntity(states map[string]*profileState, entityID string) model.EntityType {
	for _, state := range states {
		if state != nil && state.Profile != nil && state.Profile.EntityID == entityID {
			return state.Profile.EntityType
		}
	}
	return model.EntityTypeUser
}

func latestEventTime(eventsBatch []*model.DataAccessEvent) time.Time {
	latest := time.Time{}
	for _, event := range eventsBatch {
		if event != nil && event.EventTimestamp.After(latest) {
			latest = event.EventTimestamp
		}
	}
	return latest
}

func mergeCorrelatedAlerts(existing, incoming *model.UEBAAlert) *model.UEBAAlert {
	merged := *existing
	if severityRank(incoming.Severity) > severityRank(merged.Severity) {
		merged.Severity = incoming.Severity
	}
	if incoming.Confidence > merged.Confidence {
		merged.Confidence = incoming.Confidence
	}
	merged.TriggeringSignals = append(append([]model.AnomalySignal{}, existing.TriggeringSignals...), incoming.TriggeringSignals...)
	merged.TriggeringEventIDs = dedupUUIDs(append(append([]uuid.UUID{}, existing.TriggeringEventIDs...), incoming.TriggeringEventIDs...))
	merged.CorrelatedSignalCount = len(merged.TriggeringSignals)
	if incoming.CorrelationWindowStart.Before(merged.CorrelationWindowStart) {
		merged.CorrelationWindowStart = incoming.CorrelationWindowStart
	}
	if incoming.CorrelationWindowEnd.After(merged.CorrelationWindowEnd) {
		merged.CorrelationWindowEnd = incoming.CorrelationWindowEnd
	}
	merged.MITRETechniqueIDs = dedupStrings(append(append([]string{}, existing.MITRETechniqueIDs...), incoming.MITRETechniqueIDs...))
	if strings.TrimSpace(merged.MITRETactic) == "" {
		merged.MITRETactic = incoming.MITRETactic
	}
	merged.BaselineComparison = incoming.BaselineComparison
	merged.Description = incoming.Description
	return &merged
}

func buildCyberExplanation(alertRecord *model.UEBAAlert) cybermodel.AlertExplanation {
	evidence := make([]cybermodel.AlertEvidence, 0, len(alertRecord.TriggeringSignals))
	conditions := make([]string, 0, len(alertRecord.TriggeringSignals))
	falsePositiveIndicators := []string{
		"Recent travel, VPN changes, or sanctioned maintenance windows can mimic new IP and unusual time signals.",
		"Authorized bulk exports, data migrations, or break-glass admin actions can produce UEBA anomalies without malicious intent.",
	}
	recommendedActions := []string{
		"Validate whether the actor initiated the activity and confirm business justification.",
		"Review the linked access events and compare them to the entity's recent baseline.",
		"Rotate credentials or restrict access if the behavior is not explained by approved work.",
	}
	for _, signal := range alertRecord.TriggeringSignals {
		conditions = append(conditions, fmt.Sprintf("%s: %s", signal.SignalType, signal.ActualValue))
		evidence = append(evidence, cybermodel.AlertEvidence{
			Label:       string(signal.SignalType),
			Field:       "event_id",
			Value:       signal.EventID.String(),
			Description: signal.Description,
		})
	}
	return cybermodel.AlertExplanation{
		Summary:                 alertRecord.Title,
		Reason:                  alertRecord.Description,
		Evidence:                evidence,
		MatchedConditions:       conditions,
		RecommendedActions:      recommendedActions,
		FalsePositiveIndicators: falsePositiveIndicators,
		Details: map[string]any{
			"triggering_event_ids": alertRecord.TriggeringEventIDs,
			"baseline_comparison":  alertRecord.BaselineComparison,
			"alert_type":           alertRecord.AlertType,
		},
	}
}

func cyberSeverity(value string) cybermodel.Severity {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return cybermodel.SeverityCritical
	case "high":
		return cybermodel.SeverityHigh
	case "medium":
		return cybermodel.SeverityMedium
	case "low":
		return cybermodel.SeverityLow
	default:
		return cybermodel.SeverityInfo
	}
}

func riskLevelForScore(score float64) model.RiskLevel {
	switch {
	case score >= 75:
		return model.RiskLevelCritical
	case score >= 50:
		return model.RiskLevelHigh
	case score >= 25:
		return model.RiskLevelMedium
	default:
		return model.RiskLevelLow
	}
}

func clampConfidence(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 0.99:
		return 0.99
	default:
		return value
	}
}

func numberValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func randomJitter() time.Duration {
	return time.Duration(rand.Int63n(int64(maxSchedulerJitter)))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func dedupUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func dedupStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func severityRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
