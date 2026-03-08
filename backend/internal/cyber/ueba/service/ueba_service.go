package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ueba/dto"
	"github.com/clario360/platform/internal/cyber/ueba/engine"
	"github.com/clario360/platform/internal/cyber/ueba/model"
	uebarepo "github.com/clario360/platform/internal/cyber/ueba/repository"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
)

type UEBAService struct {
	db          *pgxpool.Pool
	engine      *engine.UEBAEngine
	metrics     *engine.UEBAMetrics
	profileRepo *uebarepo.ProfileRepository
	eventRepo   *uebarepo.EventRepository
	alertRepo   *uebarepo.AlertRepository
	producer    *events.Producer
	logger      zerolog.Logger
}

func NewUEBAService(
	db *pgxpool.Pool,
	engine *engine.UEBAEngine,
	metrics *engine.UEBAMetrics,
	profileRepo *uebarepo.ProfileRepository,
	eventRepo *uebarepo.EventRepository,
	alertRepo *uebarepo.AlertRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *UEBAService {
	return &UEBAService{
		db:          db,
		engine:      engine,
		metrics:     metrics,
		profileRepo: profileRepo,
		eventRepo:   eventRepo,
		alertRepo:   alertRepo,
		producer:    producer,
		logger:      logger.With().Str("component", "ueba-service").Logger(),
	}
}

func (s *UEBAService) ListProfiles(ctx context.Context, tenantID uuid.UUID, params *dto.ProfileListParams) (*dto.ProfileListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	offset := (params.Page - 1) * params.PerPage
	items, total, err := s.profileRepo.List(ctx, tenantID, params.PerPage, offset, params.Status)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.UEBAProfile{}
	}
	return &dto.ProfileListResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(params.Page, params.PerPage, total),
	}, nil
}

func (s *UEBAService) GetProfile(ctx context.Context, tenantID uuid.UUID, entityID string) (*dto.ProfileDetailResponse, error) {
	profile, err := s.profileRepo.GetByEntity(ctx, tenantID, entityID)
	if err != nil {
		return nil, err
	}
	heatmap, err := s.eventRepo.AggregateHeatmap(ctx, tenantID, entityID, 7)
	if err != nil {
		return nil, err
	}
	volume, err := s.eventRepo.AggregateEntityVolume(ctx, tenantID, entityID, time.Now().UTC().AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	}
	actualTables, actualIPs, err := s.loadRecentAccessSets(ctx, tenantID, entityID, time.Now().UTC().AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	}
	alerts, err := s.alertRepo.ListByEntitySince(ctx, tenantID, entityID, time.Now().UTC().AddDate(0, 0, -90))
	if err != nil {
		return nil, err
	}

	history := make([]dto.RiskHistoryPoint, 0, len(alerts))
	for i := len(alerts) - 1; i >= 0; i-- {
		alert := alerts[i]
		history = append(history, dto.RiskHistoryPoint{
			Timestamp: alert.CreatedAt,
			Score:     alert.RiskScoreAfter,
			AlertID:   alert.ID.String(),
			Severity:  alert.Severity,
			AlertType: string(alert.AlertType),
		})
	}

	return &dto.ProfileDetailResponse{
		Profile: profile,
		BaselineComparison: map[string]any{
			"access_times": map[string]any{
				"expected_peak_hours":    profile.Baseline.AccessTimes.PeakHours,
				"expected_active_hours":  profile.Baseline.AccessTimes.ActiveHoursCount,
				"actual_last_7d_heatmap": heatmap,
			},
			"data_volume": map[string]any{
				"expected_daily_bytes_mean": profile.Baseline.DataVolume.DailyBytesMean,
				"expected_daily_rows_mean":  profile.Baseline.DataVolume.DailyRowsMean,
				"actual_last_7d_volume":     volume,
			},
			"access_patterns": map[string]any{
				"expected_tables":          profile.Baseline.AccessPatterns.TablesAccessed,
				"actual_recent_tables":     actualTables,
				"expected_source_ips":      profile.Baseline.SourceIPs,
				"actual_recent_source_ips": actualIPs,
			},
			"failure_rate": map[string]any{
				"expected_failure_rate_percent": profile.Baseline.FailureRate.FailureRatePercent,
			},
		},
		RiskHistory: history,
	}, nil
}

func (s *UEBAService) GetTimeline(ctx context.Context, tenantID uuid.UUID, entityID string, page, perPage int) (*dto.TimelineResponse, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}
	offset := (page - 1) * perPage
	items, total, err := s.eventRepo.ListTimeline(ctx, tenantID, entityID, perPage, offset)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.DataAccessEvent{}
	}
	return &dto.TimelineResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(page, perPage, total),
	}, nil
}

func (s *UEBAService) GetHeatmap(ctx context.Context, tenantID uuid.UUID, entityID string, days int) (*dto.HeatmapResponse, error) {
	if days <= 0 {
		days = 30
	}
	matrix, err := s.eventRepo.AggregateHeatmap(ctx, tenantID, entityID, days)
	if err != nil {
		return nil, err
	}
	return &dto.HeatmapResponse{
		EntityID: entityID,
		Days:     days,
		Matrix:   matrix,
	}, nil
}

func (s *UEBAService) UpdateProfileStatus(ctx context.Context, tenantID uuid.UUID, entityID string, req *dto.ProfileStatusUpdateRequest) (*model.UEBAProfile, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return s.profileRepo.UpdateStatus(ctx, tenantID, entityID, req.EntityType, req.Status, req.SuppressedUntil, req.Reason)
}

func (s *UEBAService) ListAlerts(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams) (*dto.AlertListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	offset := (params.Page - 1) * params.PerPage
	items, total, err := s.alertRepo.List(ctx, tenantID, params.PerPage, offset, params.EntityID, params.Status)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.UEBAAlert{}
	}
	return &dto.AlertListResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(params.Page, params.PerPage, total),
	}, nil
}

func (s *UEBAService) GetAlert(ctx context.Context, tenantID, alertID uuid.UUID) (*model.UEBAAlert, error) {
	return s.alertRepo.GetByID(ctx, tenantID, alertID)
}

func (s *UEBAService) UpdateAlertStatus(ctx context.Context, tenantID, alertID uuid.UUID, actorID *uuid.UUID, req *dto.AlertStatusUpdateRequest) (*model.UEBAAlert, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	resolvedBy := actorID
	if strings.ToLower(req.Status) != "resolved" && strings.ToLower(req.Status) != "false_positive" {
		resolvedBy = nil
	}
	return s.alertRepo.UpdateStatus(ctx, tenantID, alertID, req.Status, resolvedBy, req.Notes)
}

func (s *UEBAService) MarkFalsePositive(ctx context.Context, tenantID, alertID uuid.UUID, actorID *uuid.UUID, req *dto.FalsePositiveRequest) (*model.UEBAAlert, error) {
	alertRecord, err := s.alertRepo.UpdateStatus(ctx, tenantID, alertID, "false_positive", actorID, req.Notes)
	if err != nil {
		return nil, err
	}
	profile, err := s.profileRepo.GetByEntity(ctx, tenantID, alertRecord.EntityID)
	if err != nil {
		return nil, err
	}
	triggeringEvents, err := s.eventRepo.GetByIDs(ctx, tenantID, alertRecord.TriggeringEventIDs)
	if err != nil {
		return nil, err
	}
	s.retrainProfile(profile, alertRecord, triggeringEvents)
	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, err
	}
	for _, signal := range alertRecord.TriggeringSignals {
		if s.metrics != nil {
			s.metrics.FalsePositivesTotal.WithLabelValues(tenantID.String(), string(signal.SignalType)).Inc()
		}
	}
	_ = s.publishUEBAEvent(ctx, tenantID, "com.clario360.cyber.ueba.false_positive", map[string]any{
		"alert_id":  alertID.String(),
		"entity_id": alertRecord.EntityID,
		"notes":     req.Notes,
	})
	return alertRecord, nil
}

func (s *UEBAService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*dto.DashboardResponse, error) {
	var kpis dto.DashboardKPIs
	err := database.RunReadWithTenant(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE status = 'active')::int,
				COUNT(*) FILTER (WHERE status = 'active' AND risk_score >= 50)::int,
				COALESCE(AVG(risk_score), 0)::double precision
			FROM ueba_profiles
			WHERE tenant_id = $1`,
			tenantID,
		).Scan(&kpis.ActiveProfiles, &kpis.HighRiskEntities, &kpis.AverageRiskScore); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			SELECT COUNT(*)::int
			FROM ueba_alerts
			WHERE tenant_id = $1 AND created_at >= now() - interval '7 days'`,
			tenantID,
		).Scan(&kpis.Alerts7D)
	})
	if err != nil {
		return nil, err
	}

	rankingProfiles, err := s.profileRepo.ListRiskRanking(ctx, tenantID, 20)
	if err != nil {
		return nil, err
	}
	distribution, err := s.alertTypeDistribution(ctx, tenantID, 30)
	if err != nil {
		return nil, err
	}
	trend, err := s.alertTrend(ctx, tenantID, 30)
	if err != nil {
		return nil, err
	}
	return &dto.DashboardResponse{
		KPIs:                  kpis,
		RiskRanking:           mapProfiles(rankingProfiles),
		AlertTypeDistribution: distribution,
		AlertTrend:            trend,
		Profiles:              mapProfiles(rankingProfiles),
	}, nil
}

func (s *UEBAService) GetRiskRanking(ctx context.Context, tenantID uuid.UUID, limit int) ([]dto.RiskRankingItem, error) {
	if limit <= 0 {
		limit = 20
	}
	items, err := s.profileRepo.ListRiskRanking(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}
	return mapProfiles(items), nil
}

func (s *UEBAService) GetConfig() dto.UEBAConfigDTO {
	return configToDTO(s.engine.Config())
}

func (s *UEBAService) UpdateConfig(ctx context.Context, req dto.UEBAConfigDTO) (dto.UEBAConfigDTO, error) {
	cfg, err := configFromDTO(req)
	if err != nil {
		return dto.UEBAConfigDTO{}, err
	}
	updated, err := s.engine.UpdateConfig(ctx, cfg)
	if err != nil {
		return dto.UEBAConfigDTO{}, err
	}
	return configToDTO(updated), nil
}

func (s *UEBAService) alertTypeDistribution(ctx context.Context, tenantID uuid.UUID, days int) ([]dto.ChartDatum, error) {
	items := make([]dto.ChartDatum, 0)
	err := database.RunReadWithTenant(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT alert_type, COUNT(*)::double precision
			FROM ueba_alerts
			WHERE tenant_id = $1 AND created_at >= $2
			GROUP BY 1
			ORDER BY 2 DESC`,
			tenantID, time.Now().UTC().AddDate(0, 0, -days),
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item dto.ChartDatum
			if err := rows.Scan(&item.Label, &item.Value); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (s *UEBAService) alertTrend(ctx context.Context, tenantID uuid.UUID, days int) ([]dto.TrendDatum, error) {
	items := make([]dto.TrendDatum, 0)
	err := database.RunReadWithTenant(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT date_trunc('day', created_at) AS bucket, alert_type, COUNT(*)::int
			FROM ueba_alerts
			WHERE tenant_id = $1 AND created_at >= $2
			GROUP BY 1, 2
			ORDER BY 1, 2`,
			tenantID, time.Now().UTC().AddDate(0, 0, -days),
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item dto.TrendDatum
			if err := rows.Scan(&item.Bucket, &item.AlertType, &item.Count); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (s *UEBAService) loadRecentAccessSets(ctx context.Context, tenantID uuid.UUID, entityID string, since time.Time) ([]string, []string, error) {
	var tables []string
	var sourceIPs []string
	err := database.RunReadWithTenant(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		tableRows, err := tx.Query(ctx, `
			SELECT DISTINCT concat_ws('.', NULLIF(schema_name, ''), NULLIF(table_name, ''))
			FROM ueba_access_events
			WHERE tenant_id = $1
			  AND entity_id = $2
			  AND event_timestamp >= $3
			  AND COALESCE(table_name, '') != ''`,
			tenantID, entityID, since,
		)
		if err != nil {
			return err
		}
		defer tableRows.Close()
		for tableRows.Next() {
			var value string
			if err := tableRows.Scan(&value); err != nil {
				return err
			}
			tables = append(tables, value)
		}
		if err := tableRows.Err(); err != nil {
			return err
		}

		ipRows, err := tx.Query(ctx, `
			SELECT DISTINCT source_ip
			FROM ueba_access_events
			WHERE tenant_id = $1
			  AND entity_id = $2
			  AND event_timestamp >= $3
			  AND COALESCE(source_ip, '') != ''`,
			tenantID, entityID, since,
		)
		if err != nil {
			return err
		}
		defer ipRows.Close()
		for ipRows.Next() {
			var value string
			if err := ipRows.Scan(&value); err != nil {
				return err
			}
			sourceIPs = append(sourceIPs, value)
		}
		return ipRows.Err()
	})
	sort.Strings(tables)
	sort.Strings(sourceIPs)
	return tables, sourceIPs, err
}

func (s *UEBAService) retrainProfile(profile *model.UEBAProfile, alertRecord *model.UEBAAlert, triggeringEvents []*model.DataAccessEvent) {
	if profile == nil || alertRecord == nil {
		return
	}
	cfg := s.engine.Config()
	profile.EnsureDefaults()

	eventsByID := make(map[uuid.UUID]*model.DataAccessEvent, len(triggeringEvents))
	for _, event := range triggeringEvents {
		if event != nil {
			eventsByID[event.ID] = event
		}
	}

	for _, signal := range alertRecord.TriggeringSignals {
		event := eventsByID[signal.EventID]
		switch signal.SignalType {
		case model.SignalTypeNewSourceIP:
			if event != nil {
				profile.Baseline.SourceIPs = upsertLRUString(profile.Baseline.SourceIPs, event.SourceIP, 50)
			}
		case model.SignalTypeNewTableAccess:
			if event != nil {
				table := qualifiedTableName(event.SchemaName, event.TableName)
				if table != "" {
					profile.Baseline.AccessPatterns.TablesAccessed = upsertTable(profile.Baseline.AccessPatterns.TablesAccessed, table, event.EventTimestamp)
				}
			}
		case model.SignalTypeUnusualTime:
			if event != nil {
				applyTimeReinforcement(profile, event.EventTimestamp, cfg.EMAAlpha)
			}
		case model.SignalTypeUnusualVolume, model.SignalTypeBulkDataAccess:
			if event != nil {
				profile.Baseline.DataVolume.MaxSingleQueryBytes = maxFloat(profile.Baseline.DataVolume.MaxSingleQueryBytes, float64(event.BytesAccessed))
				profile.Baseline.DataVolume.MaxSingleQueryRows = maxFloat(profile.Baseline.DataVolume.MaxSingleQueryRows, float64(event.RowsAccessed))
				profile.Baseline.DataVolume.DailyBytesMean = smoothTowards(profile.Baseline.DataVolume.DailyBytesMean, float64(event.BytesAccessed), 0.2)
				profile.Baseline.DataVolume.DailyRowsMean = smoothTowards(profile.Baseline.DataVolume.DailyRowsMean, float64(event.RowsAccessed), 0.2)
			}
		case model.SignalTypeFailedAccessSpike:
			profile.Baseline.FailureRate.DailyFailureCountMean = smoothTowards(profile.Baseline.FailureRate.DailyFailureCountMean, profile.Baseline.FailureRate.DailyFailureCountMean+1, 0.15)
			profile.Baseline.FailureRate.FailureRatePercent = smoothTowards(profile.Baseline.FailureRate.FailureRatePercent, profile.Baseline.FailureRate.FailureRatePercent+0.5, 0.1)
		case model.SignalTypePrivilegeEscalation:
			nudgeDDLShare(profile, 0.03)
		}
	}
}

func (s *UEBAService) publishUEBAEvent(ctx context.Context, tenantID uuid.UUID, eventType string, payload map[string]any) error {
	if s.producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "cyber-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return s.producer.Publish(ctx, events.Topics.UEBAEvents, event)
}

func configToDTO(cfg engine.UEBAConfig) dto.UEBAConfigDTO {
	return dto.UEBAConfigDTO{
		CycleInterval:               cfg.CycleInterval.String(),
		MaxEventsPerCycle:           cfg.MaxEventsPerCycle,
		MaxProcessingTime:           cfg.MaxProcessingTime.String(),
		EMAAlpha:                    cfg.EMAAlpha,
		MinMaturityForAlert:         string(cfg.MinMaturityForAlert),
		CorrelationWindow:           cfg.CorrelationWindow.String(),
		RiskDecayRatePerDay:         cfg.RiskDecayRatePerDay,
		BatchSize:                   cfg.BatchSize,
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
	}
}

func configFromDTO(req dto.UEBAConfigDTO) (engine.UEBAConfig, error) {
	cycleInterval, err := time.ParseDuration(req.CycleInterval)
	if err != nil {
		return engine.UEBAConfig{}, fmt.Errorf("invalid cycle_interval: %w", err)
	}
	maxProcessingTime, err := time.ParseDuration(req.MaxProcessingTime)
	if err != nil {
		return engine.UEBAConfig{}, fmt.Errorf("invalid max_processing_time: %w", err)
	}
	correlationWindow, err := time.ParseDuration(req.CorrelationWindow)
	if err != nil {
		return engine.UEBAConfig{}, fmt.Errorf("invalid correlation_window: %w", err)
	}
	return engine.UEBAConfig{
		CycleInterval:               cycleInterval,
		MaxEventsPerCycle:           req.MaxEventsPerCycle,
		MaxProcessingTime:           maxProcessingTime,
		EMAAlpha:                    req.EMAAlpha,
		MinMaturityForAlert:         model.ProfileMaturity(req.MinMaturityForAlert),
		CorrelationWindow:           correlationWindow,
		RiskDecayRatePerDay:         req.RiskDecayRatePerDay,
		BatchSize:                   req.BatchSize,
		UnusualTimeMatureHighProb:   req.UnusualTimeMatureHighProb,
		UnusualTimeMatureMediumProb: req.UnusualTimeMatureMediumProb,
		UnusualTimeBaseHighProb:     req.UnusualTimeBaseHighProb,
		UnusualTimeBaseMediumProb:   req.UnusualTimeBaseMediumProb,
		UnusualVolumeMediumZ:        req.UnusualVolumeMediumZ,
		UnusualVolumeHighZ:          req.UnusualVolumeHighZ,
		UnusualVolumeCriticalZ:      req.UnusualVolumeCriticalZ,
		UnusualVolumeStddevMin:      req.UnusualVolumeStddevMin,
		FailureSpikeMediumZ:         req.FailureSpikeMediumZ,
		FailureSpikeHighZ:           req.FailureSpikeHighZ,
		FailureSpikeCriticalZ:       req.FailureSpikeCriticalZ,
		FailureStddevMin:            req.FailureStddevMin,
		FailureCriticalCount:        req.FailureCriticalCount,
		BulkRowsMediumMultiplier:    req.BulkRowsMediumMultiplier,
		BulkRowsHighMultiplier:      req.BulkRowsHighMultiplier,
		DDLUnusualThreshold:         req.DDLUnusualThreshold,
	}, nil
}

func mapProfiles(items []*model.UEBAProfile) []dto.RiskRankingItem {
	out := make([]dto.RiskRankingItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, dto.RiskRankingItem{
			EntityID:        item.EntityID,
			EntityName:      firstNonEmpty(item.EntityName, item.EntityID),
			EntityType:      string(item.EntityType),
			RiskScore:       item.RiskScore,
			RiskLevel:       string(item.RiskLevel),
			AlertCount7D:    item.AlertCount7D,
			AlertCount30D:   item.AlertCount30D,
			ProfileMaturity: string(item.ProfileMaturity),
			LastSeenAt:      item.LastSeenAt,
			Status:          string(item.Status),
		})
	}
	return out
}

func upsertLRUString(existing []string, value string, capacity int) []string {
	if strings.TrimSpace(value) == "" {
		return existing
	}
	out := make([]string, 0, capacity)
	out = append(out, value)
	for _, item := range existing {
		if strings.EqualFold(item, value) {
			continue
		}
		out = append(out, item)
		if len(out) == capacity {
			break
		}
	}
	return out
}

func upsertTable(existing []model.FrequencyEntry, table string, ts time.Time) []model.FrequencyEntry {
	for i := range existing {
		if strings.EqualFold(existing[i].Name, table) {
			existing[i].Frequency = maxFloat(existing[i].Frequency, 0.25)
			existing[i].LastAccessed = ts
			return trimTables(existing)
		}
	}
	existing = append(existing, model.FrequencyEntry{
		Name:         table,
		Frequency:    0.25,
		LastAccessed: ts,
	})
	return trimTables(existing)
}

func trimTables(existing []model.FrequencyEntry) []model.FrequencyEntry {
	sort.SliceStable(existing, func(i, j int) bool {
		if existing[i].Frequency == existing[j].Frequency {
			return existing[i].LastAccessed.After(existing[j].LastAccessed)
		}
		return existing[i].Frequency > existing[j].Frequency
	})
	if len(existing) > 20 {
		existing = existing[:20]
	}
	return existing
}

func applyTimeReinforcement(profile *model.UEBAProfile, ts time.Time, alpha float64) {
	hour := ts.Hour()
	weekday := remapWeekday(ts.Weekday())
	for i := range profile.Baseline.AccessTimes.HourlyDistribution {
		target := 0.0
		if i == hour {
			target = 1
		}
		profile.Baseline.AccessTimes.HourlyDistribution[i] = smoothTowards(profile.Baseline.AccessTimes.HourlyDistribution[i], target, alpha)
	}
	for i := range profile.Baseline.AccessTimes.DailyDistribution {
		target := 0.0
		if i == weekday {
			target = 1
		}
		profile.Baseline.AccessTimes.DailyDistribution[i] = smoothTowards(profile.Baseline.AccessTimes.DailyDistribution[i], target, alpha)
	}
	normalizeSlice(profile.Baseline.AccessTimes.HourlyDistribution[:])
	normalizeSlice(profile.Baseline.AccessTimes.DailyDistribution[:])
}

func nudgeDDLShare(profile *model.UEBAProfile, delta float64) {
	if profile.Baseline.AccessPatterns.QueryTypes == nil {
		profile.Baseline.AccessPatterns.QueryTypes = map[string]float64{}
	}
	profile.Baseline.AccessPatterns.QueryTypes["ddl"] += delta
	total := 0.0
	for _, key := range []string{"select", "insert", "update", "delete", "ddl"} {
		total += profile.Baseline.AccessPatterns.QueryTypes[key]
	}
	if total == 0 {
		return
	}
	for _, key := range []string{"select", "insert", "update", "delete", "ddl"} {
		profile.Baseline.AccessPatterns.QueryTypes[key] /= total
	}
}

func remapWeekday(day time.Weekday) int {
	switch day {
	case time.Sunday:
		return 6
	default:
		return int(day - 1)
	}
}

func normalizeSlice(values []float64) {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	if sum == 0 {
		return
	}
	for i := range values {
		values[i] /= sum
	}
}

func smoothTowards(current, target, alpha float64) float64 {
	return alpha*target + (1-alpha)*current
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func qualifiedTableName(schemaName, tableName string) string {
	schemaName = strings.TrimSpace(schemaName)
	tableName = strings.TrimSpace(tableName)
	switch {
	case schemaName == "" && tableName == "":
		return ""
	case schemaName == "":
		return tableName
	case tableName == "":
		return schemaName
	default:
		return schemaName + "." + tableName
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
