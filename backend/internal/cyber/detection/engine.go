package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/explanation"
	"github.com/clario360/platform/internal/cyber/indicator"
	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

// RuleEvaluator compiles and evaluates a rule type.
type RuleEvaluator interface {
	Compile(ruleContent json.RawMessage) (interface{}, error)
	Evaluate(compiled interface{}, events []model.SecurityEvent) []model.RuleMatch
	Type() string
}

// AlertWriter persists generated detection alerts with deduplication logic.
type AlertWriter interface {
	CreateOrMergeDetectionAlert(ctx context.Context, alert *model.Alert) (*model.Alert, bool, error)
}

// LoadedRule is the in-memory representation of a compiled detection rule.
type LoadedRule struct {
	Rule      *model.DetectionRule
	Compiled  interface{}
	Evaluator RuleEvaluator
}

// DetectionEngine evaluates events against rules and creates alerts.
type DetectionEngine struct {
	rulesMu        sync.RWMutex
	rulesByTenant  map[uuid.UUID]map[uuid.UUID]*LoadedRule
	knownTenants   map[uuid.UUID]struct{}
	knownTenantsMu sync.RWMutex

	evaluators map[model.DetectionRuleType]RuleEvaluator

	ruleRepo   *repository.RuleRepository
	assetRepo  *repository.AssetRepository
	threatRepo *repository.ThreatRepository
	alerts     AlertWriter
	indicators *indicator.Matcher
	redis      *redis.Client
	producer   *events.Producer
	logger     zerolog.Logger
	reloadCh   chan uuid.UUID
}

// NewDetectionEngine creates a new detection engine.
func NewDetectionEngine(
	ruleRepo *repository.RuleRepository,
	assetRepo *repository.AssetRepository,
	threatRepo *repository.ThreatRepository,
	alertWriter AlertWriter,
	indicators *indicator.Matcher,
	redisClient *redis.Client,
	producer *events.Producer,
	store *BaselineStore,
	logger zerolog.Logger,
) *DetectionEngine {
	return &DetectionEngine{
		rulesByTenant: make(map[uuid.UUID]map[uuid.UUID]*LoadedRule),
		knownTenants:  make(map[uuid.UUID]struct{}),
		evaluators: map[model.DetectionRuleType]RuleEvaluator{
			model.RuleTypeSigma:       &SigmaEvaluator{},
			model.RuleTypeThreshold:   &ThresholdEvaluator{},
			model.RuleTypeCorrelation: &CorrelationEvaluator{},
			model.RuleTypeAnomaly:     NewAnomalyEvaluator(store),
		},
		ruleRepo:   ruleRepo,
		assetRepo:  assetRepo,
		threatRepo: threatRepo,
		alerts:     alertWriter,
		indicators: indicators,
		redis:      redisClient,
		producer:   producer,
		logger:     logger,
		reloadCh:   make(chan uuid.UUID, 64),
	}
}

// Start launches the periodic rule reload loop.
func (e *DetectionEngine) Start(ctx context.Context, refreshInterval time.Duration) {
	if refreshInterval <= 0 {
		refreshInterval = 60 * time.Second
	}
	ticker := time.NewTicker(refreshInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case tenantID := <-e.reloadCh:
				if err := e.LoadRules(context.Background(), tenantID); err != nil {
					e.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("immediate rule reload failed")
				}
			case <-ticker.C:
				for _, tenantID := range e.snapshotKnownTenants() {
					if err := e.LoadRules(context.Background(), tenantID); err != nil {
						e.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("scheduled rule reload failed")
					}
				}
			}
		}
	}()
}

// RequestReload asks the background loop to reload a tenant's rules.
func (e *DetectionEngine) RequestReload(tenantID uuid.UUID) {
	select {
	case e.reloadCh <- tenantID:
	default:
	}
}

// LoadRules compiles and caches all enabled rules for a tenant.
func (e *DetectionEngine) LoadRules(ctx context.Context, tenantID uuid.UUID) error {
	rules, err := e.ruleRepo.ListEnabledByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	loaded := make(map[uuid.UUID]*LoadedRule, len(rules))
	for _, rule := range rules {
		if containsTag(rule.Tags, "indicator_matcher") {
			continue
		}
		evaluator, ok := e.evaluators[rule.RuleType]
		if !ok {
			e.logger.Error().Str("rule_id", rule.ID.String()).Str("rule_type", string(rule.RuleType)).Msg("unsupported rule type")
			continue
		}
		compiled, err := evaluator.Compile(rule.RuleContent)
		if err != nil {
			e.logger.Error().Err(err).Str("rule_id", rule.ID.String()).Msg("failed to compile detection rule")
			continue
		}
		attachRuleContext(tenantID, rule.ID, compiled)
		loaded[rule.ID] = &LoadedRule{
			Rule:      rule,
			Compiled:  compiled,
			Evaluator: evaluator,
		}
	}

	e.rulesMu.Lock()
	e.rulesByTenant[tenantID] = loaded
	e.rulesMu.Unlock()
	e.rememberTenant(tenantID)
	if e.indicators != nil {
		_ = e.indicators.Load(ctx, tenantID)
	}
	e.logger.Info().Int("loaded_rules", len(loaded)).Str("tenant_id", tenantID.String()).Msg("loaded detection rules")
	return nil
}

// ProcessEvents evaluates a batch of events and generates any matching alerts.
func (e *DetectionEngine) ProcessEvents(ctx context.Context, tenantID uuid.UUID, eventsBatch []model.SecurityEvent) ([]*model.Alert, error) {
	if len(eventsBatch) == 0 {
		return nil, nil
	}
	normalizedEvents := normalizeEvents(tenantID, eventsBatch)

	e.rulesMu.RLock()
	tenantRules := e.rulesByTenant[tenantID]
	e.rulesMu.RUnlock()
	if tenantRules == nil {
		if err := e.LoadRules(ctx, tenantID); err != nil {
			return nil, err
		}
		e.rulesMu.RLock()
		tenantRules = e.rulesByTenant[tenantID]
		e.rulesMu.RUnlock()
	}

	matchedRuleIDs := make(map[uuid.UUID]map[uuid.UUID]struct{})
	alerts := make([]*model.Alert, 0)

	for _, loadedRule := range tenantRules {
		matches := loadedRule.Evaluator.Evaluate(loadedRule.Compiled, normalizedEvents)
		for _, match := range matches {
			match.RuleID = loadedRule.Rule.ID
			for _, event := range match.Events {
				if matchedRuleIDs[event.ID] == nil {
					matchedRuleIDs[event.ID] = make(map[uuid.UUID]struct{})
				}
				matchedRuleIDs[event.ID][loadedRule.Rule.ID] = struct{}{}
			}
			alert, err := e.processRuleMatch(ctx, tenantID, loadedRule.Rule, match)
			if err != nil {
				e.logger.Error().Err(err).Str("rule_id", loadedRule.Rule.ID.String()).Msg("failed to process rule match")
				continue
			}
			if alert != nil {
				alerts = append(alerts, alert)
			}
			_ = e.ruleRepo.UpdateTriggered(ctx, tenantID, loadedRule.Rule.ID, match.Timestamp)
			_ = e.publishRuleTriggered(ctx, tenantID, loadedRule.Rule.ID, alert, len(match.Events))
		}
	}

	if e.indicators != nil {
		for _, event := range normalizedEvents {
			matches := e.indicators.Match(&event)
			for _, match := range matches {
				alert, err := e.processIndicatorMatch(ctx, tenantID, event, match)
				if err != nil {
					e.logger.Error().Err(err).Str("indicator_id", match.Indicator.ID.String()).Msg("failed to process indicator match")
					continue
				}
				if alert != nil {
					alerts = append(alerts, alert)
				}
			}
		}
	}

	for i := range normalizedEvents {
		ruleIDs := matchedRuleIDs[normalizedEvents[i].ID]
		if len(ruleIDs) == 0 {
			continue
		}
		normalizedEvents[i].MatchedRules = make([]uuid.UUID, 0, len(ruleIDs))
		for ruleID := range ruleIDs {
			normalizedEvents[i].MatchedRules = append(normalizedEvents[i].MatchedRules, ruleID)
		}
		sort.Slice(normalizedEvents[i].MatchedRules, func(a, b int) bool {
			return strings.Compare(normalizedEvents[i].MatchedRules[a].String(), normalizedEvents[i].MatchedRules[b].String()) < 0
		})
	}
	if err := e.ruleRepo.InsertSecurityEvents(ctx, normalizedEvents); err != nil {
		return alerts, err
	}
	return alerts, nil
}

func (e *DetectionEngine) processRuleMatch(ctx context.Context, tenantID uuid.UUID, rule *model.DetectionRule, match model.RuleMatch) (*model.Alert, error) {
	asset, assetIDs := e.primaryAssetForMatch(ctx, tenantID, match)
	if e.isSuppressed(ctx, tenantID, rule.ID, asset) {
		return nil, nil
	}
	technique, tactic := mitre.MapRuleToPrimaryTechnique(rule)
	expl := explanation.BuildExplanation(rule, match, assetSlice(asset), nil)
	confidence := confidenceScore(expl)

	firstEventAt, lastEventAt := matchWindow(match.Events)
	alert := &model.Alert{
		TenantID:        tenantID,
		Title:           rule.Name,
		Description:     expl.Summary,
		Severity:        rule.Severity,
		Status:          model.AlertStatusNew,
		Source:          rule.Name,
		RuleID:          &rule.ID,
		AssetID:         assetIDPtr(asset),
		AssetIDs:        assetIDs,
		Explanation:     *expl,
		ConfidenceScore: confidence,
		EventCount:      len(match.Events),
		FirstEventAt:    firstEventAt,
		LastEventAt:     lastEventAt,
		Tags:            append([]string(nil), rule.Tags...),
		Metadata:        mustJSON(map[string]interface{}{"match_details": match.MatchDetails}),
	}
	if technique != nil {
		alert.MITRETechniqueID = stringPtr(technique.ID)
		alert.MITRETechniqueName = stringPtr(technique.Name)
	}
	if tactic != nil {
		alert.MITRETacticID = stringPtr(tactic.ID)
		alert.MITRETacticName = stringPtr(tactic.Name)
	}
	created, isNew, err := e.alerts.CreateOrMergeDetectionAlert(ctx, alert)
	if err != nil {
		return nil, err
	}
	if isNew {
		e.setRecentKey(ctx, tenantID, rule.ID, alert.AssetID, cooldownForRule(rule))
	}
	return created, nil
}

func (e *DetectionEngine) processIndicatorMatch(ctx context.Context, tenantID uuid.UUID, event model.SecurityEvent, match *model.IndicatorMatch) (*model.Alert, error) {
	asset := e.lookupAsset(ctx, tenantID, event.AssetID)
	assetIDs := make([]uuid.UUID, 0, 1)
	if event.AssetID != nil {
		assetIDs = append(assetIDs, *event.AssetID)
	}
	matchDetails := map[string]interface{}{
		"indicator_field":      match.Field,
		"indicator_value":      match.Value,
		"indicator_age_hours":  indicatorAgeHours(match.Indicator),
	}
	alertExplanation := explanation.BuildExplanation(nil, model.RuleMatch{
		Events:       []model.SecurityEvent{event},
		MatchDetails: matchDetails,
		Timestamp:    event.Timestamp,
	}, assetSlice(asset), []*model.ThreatIndicator{match.Indicator})

	alert := &model.Alert{
		TenantID:        tenantID,
		Title:           fmt.Sprintf("Known malicious indicator match: %s", match.Value),
		Description:     alertExplanation.Summary,
		Severity:        match.Indicator.Severity,
		Status:          model.AlertStatusNew,
		Source:          "indicator_matcher",
		AssetID:         event.AssetID,
		AssetIDs:        assetIDs,
		Explanation:     *alertExplanation,
		ConfidenceScore: confidenceScore(alertExplanation),
		EventCount:      1,
		FirstEventAt:    event.Timestamp,
		LastEventAt:     event.Timestamp,
		Tags:            append([]string{"indicator_match"}, match.Indicator.Tags...),
		Metadata:        mustJSON(map[string]interface{}{"indicator_id": match.Indicator.ID.String(), "field": match.Field}),
	}

	if match.Indicator.ThreatID != nil {
		_ = e.threatRepo.RecordObservation(ctx, tenantID, *match.Indicator.ThreatID, assetIDs)
	} else if e.threatRepo != nil {
		threat, err := e.threatRepo.UpsertSyntheticThreat(ctx, tenantID, "Observed malicious indicator activity", model.ThreatTypeOther, match.Indicator.Severity, []string{"indicator_match"})
		if err == nil {
			_ = e.threatRepo.RecordObservation(ctx, tenantID, threat.ID, assetIDs)
		}
	}
	created, _, err := e.alerts.CreateOrMergeDetectionAlert(ctx, alert)
	return created, err
}

func (e *DetectionEngine) primaryAssetForMatch(ctx context.Context, tenantID uuid.UUID, match model.RuleMatch) (*model.Asset, []uuid.UUID) {
	assetIDs := make([]uuid.UUID, 0, len(match.Events))
	seen := make(map[uuid.UUID]struct{})
	var primaryAssetID *uuid.UUID
	for _, event := range match.Events {
		if event.AssetID == nil {
			continue
		}
		if primaryAssetID == nil {
			primaryAssetID = event.AssetID
		}
		if _, ok := seen[*event.AssetID]; ok {
			continue
		}
		seen[*event.AssetID] = struct{}{}
		assetIDs = append(assetIDs, *event.AssetID)
	}
	return e.lookupAsset(ctx, tenantID, primaryAssetID), assetIDs
}

func (e *DetectionEngine) lookupAsset(ctx context.Context, tenantID uuid.UUID, assetID *uuid.UUID) *model.Asset {
	if assetID == nil || e.assetRepo == nil {
		return nil
	}
	asset, err := e.assetRepo.GetByID(ctx, tenantID, *assetID)
	if err != nil {
		return nil
	}
	return asset
}

func (e *DetectionEngine) isSuppressed(ctx context.Context, tenantID, ruleID uuid.UUID, asset *model.Asset) bool {
	if e.redis == nil || asset == nil {
		return false
	}
	key := dedupKey(tenantID, ruleID, &asset.ID)
	exists, err := e.redis.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return exists > 0
}

func (e *DetectionEngine) setRecentKey(ctx context.Context, tenantID, ruleID uuid.UUID, assetID *uuid.UUID, ttl time.Duration) {
	if e.redis == nil || assetID == nil {
		return
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	key := dedupKey(tenantID, ruleID, assetID)
	_ = e.redis.Set(ctx, key, "1", ttl).Err()
}

func dedupKey(tenantID, ruleID uuid.UUID, assetID *uuid.UUID) string {
	if assetID == nil {
		return fmt.Sprintf("detect:recent:%s:%s:global", tenantID, ruleID)
	}
	return fmt.Sprintf("detect:recent:%s:%s:%s", tenantID, ruleID, assetID.String())
}

func cooldownForRule(rule *model.DetectionRule) time.Duration {
	var payload map[string]interface{}
	if err := json.Unmarshal(rule.RuleContent, &payload); err != nil {
		return 5 * time.Minute
	}
	if rawCooldown, ok := payload["cooldown"].(string); ok {
		if duration, err := time.ParseDuration(rawCooldown); err == nil && duration > 0 {
			return duration
		}
	}
	return 5 * time.Minute
}

func matchWindow(events []model.SecurityEvent) (time.Time, time.Time) {
	if len(events) == 0 {
		now := time.Now().UTC()
		return now, now
	}
	first := events[0].Timestamp
	last := events[0].Timestamp
	for _, event := range events[1:] {
		if event.Timestamp.Before(first) {
			first = event.Timestamp
		}
		if event.Timestamp.After(last) {
			last = event.Timestamp
		}
	}
	return first, last
}

func normalizeEvents(tenantID uuid.UUID, input []model.SecurityEvent) []model.SecurityEvent {
	events := make([]model.SecurityEvent, 0, len(input))
	now := time.Now().UTC()
	for _, event := range input {
		if event.ID == uuid.Nil {
			event.ID = uuid.New()
		}
		event.TenantID = tenantID
		if event.Timestamp.IsZero() {
			event.Timestamp = now
		}
		if event.ProcessedAt.IsZero() {
			event.ProcessedAt = now
		}
		if len(event.RawEvent) == 0 {
			event.RawEvent = json.RawMessage("{}")
		}
		events = append(events, event)
	}
	return events
}

func attachRuleContext(tenantID, ruleID uuid.UUID, compiled interface{}) {
	switch typed := compiled.(type) {
	case *compiledAnomalyRule:
		typed.TenantID = tenantID
		typed.RuleID = ruleID
	}
}

func containsTag(tags []string, value string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, value) {
			return true
		}
	}
	return false
}

func confidenceScore(expl *model.AlertExplanation) float64 {
	if expl == nil || expl.Details == nil {
		return 0.70
	}
	if value, ok := expl.Details["confidence_score"].(float64); ok {
		return value
	}
	return 0.70
}

func assetSlice(asset *model.Asset) []*model.Asset {
	if asset == nil {
		return nil
	}
	return []*model.Asset{asset}
}

func assetIDPtr(asset *model.Asset) *uuid.UUID {
	if asset == nil {
		return nil
	}
	id := asset.ID
	return &id
}

func indicatorAgeHours(indicator *model.ThreatIndicator) float64 {
	if indicator == nil {
		return 0
	}
	return time.Since(indicator.LastSeenAt).Hours()
}

func alertID(alert *model.Alert) *uuid.UUID {
	if alert == nil {
		return nil
	}
	return &alert.ID
}

func uuidPtrString(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	out := value.String()
	return &out
}

func stringPtr(value string) *string {
	return &value
}

func mustJSON(payload interface{}) json.RawMessage {
	if payload == nil {
		return json.RawMessage("{}")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return json.RawMessage("{}")
	}
	return encoded
}

func (e *DetectionEngine) publishRuleTriggered(ctx context.Context, tenantID, ruleID uuid.UUID, alert *model.Alert, eventCount int) error {
	if e.producer == nil {
		return nil
	}
	event, err := events.NewEvent("cyber.rule.triggered", "cyber-service", tenantID.String(), map[string]interface{}{
		"id":          ruleID.String(),
		"alert_id":    uuidPtrString(alertID(alert)),
		"event_count": eventCount,
	})
	if err != nil {
		return err
	}
	return e.producer.Publish(ctx, events.Topics.AlertEvents, event)
}

func (e *DetectionEngine) rememberTenant(tenantID uuid.UUID) {
	e.knownTenantsMu.Lock()
	defer e.knownTenantsMu.Unlock()
	e.knownTenants[tenantID] = struct{}{}
}

func (e *DetectionEngine) snapshotKnownTenants() []uuid.UUID {
	e.knownTenantsMu.RLock()
	defer e.knownTenantsMu.RUnlock()
	out := make([]uuid.UUID, 0, len(e.knownTenants))
	for tenantID := range e.knownTenants {
		out = append(out, tenantID)
	}
	return out
}
