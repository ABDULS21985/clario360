package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/indicator"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	predictfeeds "github.com/clario360/platform/internal/cyber/vciso/predict/feeds"
	"github.com/clario360/platform/internal/events"
)

// SyncErrorKind classifies sync failures so the handler can choose the right HTTP status.
type SyncErrorKind int

const (
	SyncErrNotFound  SyncErrorKind = iota // feed does not exist
	SyncErrBadConfig                      // feed configuration is invalid (e.g. missing URL)
	SyncErrUpstream                       // external feed endpoint unreachable or returned an error
	SyncErrParse                          // feed payload could not be parsed
	SyncErrInternal                       // unexpected internal error
)

// SyncError wraps a classified sync failure.
type SyncError struct {
	Kind  SyncErrorKind
	Cause error
}

func (e *SyncError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "sync failed"
}

func (e *SyncError) Unwrap() error {
	return e.Cause
}

func classifyFetchError(err error) SyncErrorKind {
	if strings.Contains(err.Error(), "feed URL is not configured") {
		return SyncErrBadConfig
	}
	return SyncErrUpstream
}

type ThreatFeedService struct {
	feedRepo      *repository.ThreatFeedRepository
	indicatorRepo *repository.IndicatorRepository
	threatRepo    *repository.ThreatRepository
	producer      *events.Producer
	ingester      *predictfeeds.ThreatFeedIngester
	logger        zerolog.Logger
}

func NewThreatFeedService(
	feedRepo *repository.ThreatFeedRepository,
	indicatorRepo *repository.IndicatorRepository,
	threatRepo *repository.ThreatRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *ThreatFeedService {
	return &ThreatFeedService{
		feedRepo:      feedRepo,
		indicatorRepo: indicatorRepo,
		threatRepo:    threatRepo,
		producer:      producer,
		ingester:      predictfeeds.NewThreatFeedIngester(),
		logger:        logger,
	}
}

func (s *ThreatFeedService) ListFeeds(ctx context.Context, tenantID uuid.UUID, page, perPage int, actor *Actor) (*dto.ThreatFeedListResponse, error) {
	items, total, err := s.feedRepo.List(ctx, tenantID, page, perPage)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item.NextSyncAt = computeThreatFeedNextSync(item)
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat_feed.listed", tenantID, actor, map[string]interface{}{
		"count": len(items),
	})
	return &dto.ThreatFeedListResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(page, perPage, total),
	}, nil
}

func (s *ThreatFeedService) CreateFeed(ctx context.Context, tenantID, userID uuid.UUID, actor *Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error) {
	if err := validateThreatFeedConfig(req); err != nil {
		return nil, err
	}
	url := optionalStringPtr(req.URL)
	item, err := s.feedRepo.Create(ctx, &model.ThreatFeedConfig{
		TenantID:          tenantID,
		Name:              strings.TrimSpace(req.Name),
		Type:              req.Type,
		URL:               url,
		AuthType:          req.AuthType,
		AuthConfig:        req.AuthConfig,
		SyncInterval:      req.SyncInterval,
		DefaultSeverity:   req.DefaultSeverity,
		DefaultConfidence: normalizeIndicatorConfidence(req.DefaultConfidence),
		DefaultTags:       normalizeStrings(req.DefaultTags),
		IndicatorTypes:    normalizeStrings(req.IndicatorTypes),
		Enabled:           req.Enabled,
		Status:            threatFeedConfigStatus(req.Enabled, nil),
		CreatedBy:         &userID,
	})
	if err != nil {
		return nil, err
	}
	item.NextSyncAt = computeThreatFeedNextSync(item)
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat_feed.created", tenantID, actor, item)
	return item, nil
}

func (s *ThreatFeedService) UpdateFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error) {
	if err := validateThreatFeedConfig(req); err != nil {
		return nil, err
	}
	before, err := s.feedRepo.GetByID(ctx, tenantID, feedID)
	if err != nil {
		return nil, err
	}
	updated, err := s.feedRepo.Update(ctx, &model.ThreatFeedConfig{
		ID:                feedID,
		TenantID:          tenantID,
		Name:              strings.TrimSpace(req.Name),
		Type:              req.Type,
		URL:               optionalStringPtr(req.URL),
		AuthType:          req.AuthType,
		AuthConfig:        req.AuthConfig,
		SyncInterval:      req.SyncInterval,
		DefaultSeverity:   req.DefaultSeverity,
		DefaultConfidence: normalizeIndicatorConfidence(req.DefaultConfidence),
		DefaultTags:       normalizeStrings(req.DefaultTags),
		IndicatorTypes:    normalizeStrings(req.IndicatorTypes),
		Enabled:           req.Enabled,
		Status:            threatFeedConfigStatus(req.Enabled, before.LastError),
	})
	if err != nil {
		return nil, err
	}
	updated.LastSyncAt = before.LastSyncAt
	updated.LastSyncStatus = before.LastSyncStatus
	updated.LastError = before.LastError
	updated.NextSyncAt = computeThreatFeedNextSync(updated)
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat_feed.updated", tenantID, actor, map[string]interface{}{
		"before": before,
		"after":  updated,
	})
	return updated, nil
}

func (s *ThreatFeedService) ListHistory(ctx context.Context, tenantID, feedID uuid.UUID, actor *Actor) ([]*model.ThreatFeedSyncHistory, error) {
	items, err := s.feedRepo.ListHistory(ctx, tenantID, feedID, 20)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat_feed.history_viewed", tenantID, actor, map[string]interface{}{
		"id":    feedID.String(),
		"count": len(items),
	})
	return items, nil
}

func (s *ThreatFeedService) SyncFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *Actor) (map[string]interface{}, error) {
	config, err := s.feedRepo.GetByID(ctx, tenantID, feedID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, &SyncError{Kind: SyncErrNotFound, Cause: err}
		}
		return nil, &SyncError{Kind: SyncErrInternal, Cause: err}
	}

	startedAt := time.Now().UTC()
	summary := map[string]interface{}{
		"feed_id":             feedID.String(),
		"feed_name":           config.Name,
		"indicators_parsed":   0,
		"indicators_imported": 0,
		"indicators_skipped":  0,
		"indicators_failed":   0,
	}

	preview := make([]map[string]interface{}, 0, 10)
	status := model.ThreatFeedStatusActive
	lastSyncStatus := "completed"
	var lastError *string
	var syncErr *SyncError

	appendPreview := func(item *model.ThreatIndicator) {
		if len(preview) >= 10 {
			return
		}
		preview = append(preview, map[string]interface{}{
			"id":         item.ID.String(),
			"type":       item.Type,
			"value":      item.Value,
			"severity":   item.Severity,
			"source":     item.Source,
			"created_at": item.CreatedAt,
		})
	}

	setFailed := func(kind SyncErrorKind, cause error) {
		status = model.ThreatFeedStatusError
		lastSyncStatus = "failed"
		text := cause.Error()
		lastError = &text
		syncErr = &SyncError{Kind: kind, Cause: cause}
	}

	switch config.Type {
	case model.ThreatFeedTypeManual:
		// Manual feeds are configuration-only. A sync records health without importing payload.
	case model.ThreatFeedTypeSTIX, model.ThreatFeedTypeTAXII:
		payload, err := s.fetchFeedPayload(ctx, config)
		if err != nil {
			setFailed(classifyFetchError(err), err)
			break
		}
		bundle, err := indicator.ParseSTIXBundle(payload, "stix_feed")
		if err != nil {
			setFailed(SyncErrParse, err)
			break
		}

		threatIDs := make(map[string]uuid.UUID, len(bundle.Threats))
		for _, parsedThreat := range bundle.Threats {
			threat, err := s.threatRepo.UpsertSyntheticThreat(ctx, tenantID, parsedThreat.Name, parsedThreat.Description, parsedThreat.Type, config.DefaultSeverity, parsedThreat.Tags)
			if err != nil {
				setFailed(SyncErrInternal, err)
				break
			}
			threatIDs[parsedThreat.ExternalID] = threat.ID
		}
		if syncErr != nil {
			break
		}

		for _, parsedIndicator := range bundle.Indicators {
			indicatorItem := parsedIndicator.Indicator
			if !feedAllowsIndicatorType(config, indicatorItem.Type) {
				summary["indicators_skipped"] = summary["indicators_skipped"].(int) + 1
				continue
			}
			indicatorItem.TenantID = tenantID
			indicatorItem.Source = "stix_feed"
			indicatorItem.Severity = feedIndicatorSeverity(config, indicatorItem.Severity)
			indicatorItem.Confidence = feedIndicatorConfidence(config, indicatorItem.Confidence)
			indicatorItem.Tags = mergeIndicatorTags(config.DefaultTags, indicatorItem.Tags)
			indicatorItem.Metadata = feedIndicatorMetadata(config, indicatorItem.Metadata)
			for _, relatedThreatID := range parsedIndicator.RelatedThreatIDs {
				if threatID, ok := threatIDs[relatedThreatID]; ok {
					indicatorItem.ThreatID = &threatID
					break
				}
			}
			created, err := s.indicatorRepo.Create(ctx, &indicatorItem)
			if err != nil {
				summary["indicators_failed"] = summary["indicators_failed"].(int) + 1
				continue
			}
			appendPreview(created)
			summary["indicators_imported"] = summary["indicators_imported"].(int) + 1
		}
		summary["indicators_parsed"] = len(bundle.Indicators)
	case model.ThreatFeedTypeMISP:
		payload, err := s.fetchFeedPayload(ctx, config)
		if err != nil {
			setFailed(classifyFetchError(err), err)
			break
		}
		signals, err := s.ingester.ParseMISP(payload)
		if err != nil {
			setFailed(SyncErrParse, err)
			break
		}
		summary["indicators_parsed"] = len(signals)
		for _, signal := range signals {
			for _, raw := range signal.IOCs {
				item, ok := buildFeedIndicator(raw, config, "vendor")
				if !ok {
					summary["indicators_failed"] = summary["indicators_failed"].(int) + 1
					continue
				}
				item.TenantID = tenantID
				item.Metadata = feedIndicatorMetadata(config, item.Metadata)
				created, err := s.indicatorRepo.Create(ctx, item)
				if err != nil {
					summary["indicators_failed"] = summary["indicators_failed"].(int) + 1
					continue
				}
				appendPreview(created)
				summary["indicators_imported"] = summary["indicators_imported"].(int) + 1
			}
		}
	case model.ThreatFeedTypeCSVURL:
		payload, err := s.fetchFeedPayload(ctx, config)
		if err != nil {
			setFailed(classifyFetchError(err), err)
			break
		}
		indicators, failed, err := parseFeedCSV(payload, config)
		if err != nil {
			setFailed(SyncErrParse, err)
			break
		}
		summary["indicators_parsed"] = len(indicators) + failed
		summary["indicators_failed"] = failed
		for _, item := range indicators {
			item.TenantID = tenantID
			item.Metadata = feedIndicatorMetadata(config, item.Metadata)
			created, err := s.indicatorRepo.Create(ctx, item)
			if err != nil {
				summary["indicators_failed"] = summary["indicators_failed"].(int) + 1
				continue
			}
			appendPreview(created)
			summary["indicators_imported"] = summary["indicators_imported"].(int) + 1
		}
	}

	// Always record sync history and update feed state, even on failure.
	completedAt := time.Now().UTC()
	duration := completedAt.Sub(startedAt)
	historyMeta, _ := json.Marshal(map[string]interface{}{
		"preview_indicators": preview,
	})
	history := &model.ThreatFeedSyncHistory{
		TenantID:           tenantID,
		FeedID:             feedID,
		Status:             lastSyncStatus,
		IndicatorsParsed:   summary["indicators_parsed"].(int),
		IndicatorsImported: summary["indicators_imported"].(int),
		IndicatorsSkipped:  summary["indicators_skipped"].(int),
		IndicatorsFailed:   summary["indicators_failed"].(int),
		DurationMs:         int(duration / time.Millisecond),
		ErrorMessage:       lastError,
		Metadata:           historyMeta,
		StartedAt:          startedAt,
		CompletedAt:        &completedAt,
	}
	if histErr := s.feedRepo.AppendHistory(ctx, history); histErr != nil {
		s.logger.Error().Err(histErr).Str("feed_id", feedID.String()).Msg("failed to append sync history")
	}
	if stateErr := s.feedRepo.UpdateSyncState(ctx, tenantID, feedID, status, lastSyncStatus, lastError, completedAt); stateErr != nil {
		s.logger.Error().Err(stateErr).Str("feed_id", feedID.String()).Msg("failed to update feed sync state")
	}

	if syncErr != nil {
		return nil, syncErr
	}

	summary["status"] = lastSyncStatus
	summary["completed_at"] = completedAt
	summary["duration_ms"] = history.DurationMs
	summary["preview_indicators"] = preview

	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.threat_feed.synced", tenantID, actor, summary)
	_ = publishAuditEvent(ctx, s.producer, "cyber.threat_feed.synced", tenantID, actor, summary)
	return summary, nil
}

func validateThreatFeedConfig(req *dto.ThreatFeedConfigRequest) error {
	if req == nil || strings.TrimSpace(req.Name) == "" || !req.Type.IsValid() || !req.AuthType.IsValid() || !req.SyncInterval.IsValid() || !req.DefaultSeverity.IsValid() {
		return repository.ErrInvalidInput
	}
	if req.Type != model.ThreatFeedTypeManual && strings.TrimSpace(req.URL) == "" {
		return fmt.Errorf("feed URL is required")
	}
	for _, indicatorType := range req.IndicatorTypes {
		if !model.IndicatorType(strings.TrimSpace(indicatorType)).IsValid() {
			return fmt.Errorf("invalid indicator type filter: %s", indicatorType)
		}
	}
	return nil
}

func computeThreatFeedNextSync(item *model.ThreatFeedConfig) *time.Time {
	if item == nil || !item.Enabled || item.SyncInterval == model.ThreatFeedIntervalManual {
		return nil
	}
	base := time.Now().UTC()
	if item.LastSyncAt != nil {
		base = item.LastSyncAt.UTC()
	}
	var next time.Time
	switch item.SyncInterval {
	case model.ThreatFeedIntervalHourly:
		next = base.Add(time.Hour)
	case model.ThreatFeedIntervalEvery6H:
		next = base.Add(6 * time.Hour)
	case model.ThreatFeedIntervalDaily:
		next = base.Add(24 * time.Hour)
	case model.ThreatFeedIntervalWeekly:
		next = base.Add(7 * 24 * time.Hour)
	default:
		return nil
	}
	return &next
}

func threatFeedConfigStatus(enabled bool, lastError *string) model.ThreatFeedStatus {
	if !enabled {
		return model.ThreatFeedStatusPaused
	}
	if lastError != nil && strings.TrimSpace(*lastError) != "" {
		return model.ThreatFeedStatusError
	}
	return model.ThreatFeedStatusActive
}

func (s *ThreatFeedService) fetchFeedPayload(ctx context.Context, config *model.ThreatFeedConfig) ([]byte, error) {
	if config.URL == nil || strings.TrimSpace(*config.URL) == "" {
		return nil, fmt.Errorf("feed URL is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *config.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create feed request: %w", err)
	}
	req.Header.Set("Accept", "application/json,text/csv,*/*")
	applyThreatFeedAuth(req, config)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute feed request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read feed payload: %w", err)
	}
	return payload, nil
}

func applyThreatFeedAuth(req *http.Request, config *model.ThreatFeedConfig) {
	if config == nil {
		return
	}
	var auth map[string]interface{}
	if err := json.Unmarshal(config.AuthConfig, &auth); err != nil {
		return
	}
	switch config.AuthType {
	case model.ThreatFeedAuthAPIKey:
		key, _ := auth["api_key"].(string)
		headerName, _ := auth["header_name"].(string)
		if headerName == "" {
			headerName = "Authorization"
		}
		if key != "" {
			if strings.EqualFold(headerName, "Authorization") && !strings.HasPrefix(strings.ToLower(key), "bearer ") {
				req.Header.Set(headerName, "Bearer "+key)
			} else {
				req.Header.Set(headerName, key)
			}
		}
	case model.ThreatFeedAuthBasic:
		username, _ := auth["username"].(string)
		password, _ := auth["password"].(string)
		if username != "" {
			req.SetBasicAuth(username, password)
		}
	}
}

func feedAllowsIndicatorType(config *model.ThreatFeedConfig, indicatorType model.IndicatorType) bool {
	if config == nil || len(config.IndicatorTypes) == 0 {
		return true
	}
	for _, item := range config.IndicatorTypes {
		if strings.EqualFold(item, string(indicatorType)) {
			return true
		}
	}
	return false
}

func feedIndicatorSeverity(config *model.ThreatFeedConfig, value model.Severity) model.Severity {
	if !value.IsValid() || value == model.SeverityInfo {
		return config.DefaultSeverity
	}
	return value
}

func feedIndicatorConfidence(config *model.ThreatFeedConfig, value float64) float64 {
	value = normalizeIndicatorConfidence(value)
	if value <= 0 {
		return normalizeIndicatorConfidence(config.DefaultConfidence)
	}
	return value
}

func mergeIndicatorTags(defaults, current []string) []string {
	return normalizeStrings(append(defaults, current...))
}

func feedIndicatorMetadata(config *model.ThreatFeedConfig, metadata json.RawMessage) json.RawMessage {
	payload := map[string]interface{}{}
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &payload)
	}
	payload["feed_id"] = config.ID.String()
	payload["feed_name"] = config.Name
	payload["feed_type"] = config.Type
	merged, _ := json.Marshal(payload)
	return merged
}

func buildFeedIndicator(raw string, config *model.ThreatFeedConfig, source string) (*model.ThreatIndicator, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, false
	}
	indicatorType, ok := detectIndicatorType(value)
	if !ok || !feedAllowsIndicatorType(config, indicatorType) {
		return nil, false
	}
	return &model.ThreatIndicator{
		Type:       indicatorType,
		Value:      value,
		Severity:   config.DefaultSeverity,
		Source:     source,
		Confidence: normalizeIndicatorConfidence(config.DefaultConfidence),
		Active:     true,
		Tags:       normalizeStrings(config.DefaultTags),
	}, true
}

func parseFeedCSV(payload []byte, config *model.ThreatFeedConfig) ([]*model.ThreatIndicator, int, error) {
	reader := csv.NewReader(strings.NewReader(string(payload)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, 0, fmt.Errorf("parse feed csv: %w", err)
	}
	if len(records) < 2 {
		return []*model.ThreatIndicator{}, 0, nil
	}

	headers := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		headers[strings.ToLower(strings.TrimSpace(header))] = index
	}

	items := make([]*model.ThreatIndicator, 0, len(records)-1)
	failed := 0
	for _, record := range records[1:] {
		value := csvField(record, headers, "value", "indicator", "ioc")
		if value == "" {
			failed++
			continue
		}
		indicatorType := model.IndicatorType(csvField(record, headers, "type", "indicator_type"))
		if !indicatorType.IsValid() {
			detected, ok := detectIndicatorType(value)
			if !ok {
				failed++
				continue
			}
			indicatorType = detected
		}
		if !feedAllowsIndicatorType(config, indicatorType) {
			continue
		}
		severity := model.Severity(csvField(record, headers, "severity"))
		if !severity.IsValid() || severity == model.SeverityInfo {
			severity = config.DefaultSeverity
		}
		confidence := normalizeIndicatorConfidence(config.DefaultConfidence)
		if rawConfidence := csvField(record, headers, "confidence"); rawConfidence != "" {
			if parsed, err := strconv.ParseFloat(rawConfidence, 64); err == nil {
				confidence = normalizeIndicatorConfidence(parsed)
			}
		}
		source := csvField(record, headers, "source")
		if source == "" {
			source = "vendor"
		}
		item := &model.ThreatIndicator{
			Type:        indicatorType,
			Value:       strings.TrimSpace(value),
			Description: csvField(record, headers, "description"),
			Severity:    severity,
			Source:      source,
			Confidence:  confidence,
			Active:      true,
			Tags:        normalizeStrings(append(config.DefaultTags, splitCSVTags(csvField(record, headers, "tags"))...)),
		}
		items = append(items, item)
	}
	return items, failed, nil
}

func csvField(record []string, headers map[string]int, keys ...string) string {
	for _, key := range keys {
		index, ok := headers[key]
		if ok && index >= 0 && index < len(record) {
			return strings.TrimSpace(record[index])
		}
	}
	return ""
}

func splitCSVTags(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func detectIndicatorType(value string) (model.IndicatorType, bool) {
	switch {
	case strings.Contains(value, "/"):
		if _, _, err := net.ParseCIDR(value); err == nil {
			return model.IndicatorTypeCIDR, true
		}
	case net.ParseIP(value) != nil:
		return model.IndicatorTypeIP, true
	case strings.Contains(value, "@"):
		if err := validateIndicatorValue(model.IndicatorTypeEmail, value); err == nil {
			return model.IndicatorTypeEmail, true
		}
	case strings.HasPrefix(strings.ToLower(value), "http://") || strings.HasPrefix(strings.ToLower(value), "https://"):
		if err := validateIndicatorValue(model.IndicatorTypeURL, value); err == nil {
			return model.IndicatorTypeURL, true
		}
	case md5Pattern.MatchString(value):
		return model.IndicatorTypeHashMD5, true
	case sha1Pattern.MatchString(value):
		return model.IndicatorTypeHashSHA1, true
	case sha256Pattern.MatchString(value):
		return model.IndicatorTypeHashSHA256, true
	case domainPattern.MatchString(value):
		return model.IndicatorTypeDomain, true
	}
	return "", false
}
