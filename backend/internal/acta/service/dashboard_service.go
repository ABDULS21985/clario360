package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/repository"
)

type DashboardService struct {
	store    *repository.Store
	cache    *redis.Client
	cacheTTL time.Duration
	logger   zerolog.Logger
}

type ActaDashboard struct {
	KPIs                  ActaKPIs                  `json:"kpis"`
	UpcomingMeetings      []model.MeetingSummary    `json:"upcoming_meetings"`
	RecentMeetings        []model.MeetingSummary    `json:"recent_meetings"`
	ActionItemsByStatus   map[string]int            `json:"action_items_by_status"`
	ActionItemsByPriority map[string]int            `json:"action_items_by_priority"`
	OverdueActionItems    []model.ActionItemSummary `json:"overdue_action_items"`
	ComplianceByCommittee []model.CommitteeCompliance `json:"compliance_by_committee"`
	ComplianceScore       float64                   `json:"compliance_score"`
	MeetingFrequencyChart []MonthlyMeetingCount     `json:"meeting_frequency_chart"`
	AttendanceRateChart   []MonthlyAttendanceRate   `json:"attendance_rate_chart"`
	RecentActivity        []AuditEntry              `json:"recent_activity"`
	CalculatedAt          time.Time                 `json:"calculated_at"`
}

type ActaKPIs struct {
	ActiveCommittees       int     `json:"active_committees"`
	UpcomingMeetings       int     `json:"upcoming_meetings_30d"`
	OpenActionItems        int     `json:"open_action_items"`
	OverdueActionItems     int     `json:"overdue_action_items"`
	ComplianceScore        float64 `json:"compliance_score"`
	MinutesPendingApproval int     `json:"minutes_pending_approval"`
	AttendanceRate         float64 `json:"attendance_rate_avg"`
}

type MonthlyMeetingCount struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

type MonthlyAttendanceRate struct {
	Month       string  `json:"month"`
	RatePercent float64 `json:"rate_percent"`
}

type AuditEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	EntityID  string    `json:"entity_id"`
}

func NewDashboardService(store *repository.Store, cache *redis.Client, cacheTTL time.Duration, logger zerolog.Logger) *DashboardService {
	return &DashboardService{
		store:    store,
		cache:    cache,
		cacheTTL: cacheTTL,
		logger:   logger.With().Str("component", "acta_dashboard_service").Logger(),
	}
}

func (s *DashboardService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*ActaDashboard, error) {
	cacheKey := "acta:dashboard:" + tenantID.String()
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
			var dashboard ActaDashboard
			if unmarshalErr := json.Unmarshal(cached, &dashboard); unmarshalErr == nil {
				return &dashboard, nil
			}
		}
	}

	dashboard := &ActaDashboard{CalculatedAt: time.Now().UTC()}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		kpis, err := s.loadKPIs(groupCtx, tenantID)
		if err == nil {
			dashboard.KPIs = *kpis
		}
		return err
	})
	group.Go(func() error {
		upcoming, err := s.store.ListUpcomingMeetings(groupCtx, tenantID, 5)
		if err == nil {
			dashboard.UpcomingMeetings = upcoming
		}
		return err
	})
	group.Go(func() error {
		recent, err := s.loadRecentMeetings(groupCtx, tenantID)
		if err == nil {
			dashboard.RecentMeetings = recent
		}
		return err
	})
	group.Go(func() error {
		byStatus, err := s.store.CountActionItemsByStatus(groupCtx, tenantID)
		if err == nil {
			dashboard.ActionItemsByStatus = byStatus
		}
		return err
	})
	group.Go(func() error {
		byPriority, err := s.store.CountActionItemsByPriority(groupCtx, tenantID)
		if err == nil {
			dashboard.ActionItemsByPriority = byPriority
		}
		return err
	})
	group.Go(func() error {
		items, err := s.loadOverdueActionItemSummaries(groupCtx, tenantID)
		if err == nil {
			dashboard.OverdueActionItems = items
		}
		return err
	})
	group.Go(func() error {
		checks, err := s.store.LatestComplianceChecksByCommittee(groupCtx, tenantID)
		if err == nil {
			dashboard.ComplianceByCommittee = buildComplianceReport(tenantID, nil, checks, time.Now().UTC()).ByCommittee
			dashboard.ComplianceScore = complianceScore(checks)
		}
		return err
	})
	group.Go(func() error {
		chart, err := s.loadMeetingFrequencyChart(groupCtx, tenantID)
		if err == nil {
			dashboard.MeetingFrequencyChart = chart
		}
		return err
	})
	group.Go(func() error {
		chart, err := s.loadAttendanceRateChart(groupCtx, tenantID)
		if err == nil {
			dashboard.AttendanceRateChart = chart
		}
		return err
	})
	group.Go(func() error {
		activity, err := s.loadRecentActivity(groupCtx, tenantID)
		if err == nil {
			dashboard.RecentActivity = activity
		}
		return err
	})
	if err := group.Wait(); err != nil {
		return nil, internalError("failed to build dashboard", err)
	}

	if s.cache != nil {
		if payload, err := json.Marshal(dashboard); err == nil {
			_ = s.cache.Set(ctx, cacheKey, payload, s.cacheTTL).Err()
		}
	}
	return dashboard, nil
}

func (s *DashboardService) loadKPIs(ctx context.Context, tenantID uuid.UUID) (*ActaKPIs, error) {
	kpis := &ActaKPIs{}
	activeCommittees, err := s.store.ListActiveCommittees(ctx, &tenantID)
	if err != nil {
		return nil, err
	}
	kpis.ActiveCommittees = len(activeCommittees)
	statusCounts, err := s.store.CountActionItemsByStatus(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	kpis.OpenActionItems = statusCounts["pending"] + statusCounts["in_progress"] + statusCounts["deferred"] + statusCounts["overdue"]
	kpis.OverdueActionItems = statusCounts["overdue"]
	if err := s.store.DB().QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status IN ('draft', 'scheduled', 'postponed')),
			COUNT(*) FILTER (WHERE has_minutes = true AND minutes_status IN ('draft', 'review', 'revision_requested')),
			COALESCE(AVG(rate), 0)
		FROM (
			SELECT CASE
				WHEN COUNT(*) = 0 THEN 0
				ELSE (COUNT(*) FILTER (WHERE ma.status IN ('present', 'proxy'))::float / COUNT(*)::float) * 100
			END AS rate,
			m.status,
			m.has_minutes,
			m.minutes_status
			FROM meetings m
			LEFT JOIN meeting_attendance ma ON ma.meeting_id = m.id
			WHERE m.tenant_id = $1
			  AND m.deleted_at IS NULL
			  AND m.scheduled_at >= now() - interval '30 days'
			GROUP BY m.id, m.status, m.has_minutes, m.minutes_status
		) summary`, tenantID).Scan(&kpis.UpcomingMeetings, &kpis.MinutesPendingApproval, &kpis.AttendanceRate); err != nil {
		return nil, fmt.Errorf("load acta kpis: %w", err)
	}
	kpis.ComplianceScore = complianceScoreFromStored(statusCounts, tenantID, s.store)
	return kpis, nil
}

func (s *DashboardService) loadRecentMeetings(ctx context.Context, tenantID uuid.UUID) ([]model.MeetingSummary, error) {
	rows, err := s.store.DB().Query(ctx, `
		SELECT id, committee_id, committee_name, title, status, scheduled_at, duration_minutes, location, quorum_met
		FROM meetings
		WHERE tenant_id = $1 AND deleted_at IS NULL AND status = 'completed'
		ORDER BY COALESCE(actual_end_at, scheduled_at) DESC
		LIMIT 5`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.MeetingSummary, 0, 5)
	for rows.Next() {
		var item model.MeetingSummary
		if err := rows.Scan(&item.ID, &item.CommitteeID, &item.CommitteeName, &item.Title, &item.Status, &item.ScheduledAt, &item.DurationMinutes, &item.Location, &item.QuorumMet); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *DashboardService) loadOverdueActionItemSummaries(ctx context.Context, tenantID uuid.UUID) ([]model.ActionItemSummary, error) {
	rows, err := s.store.DB().Query(ctx, `
		SELECT ai.id, ai.title, ai.committee_id, c.name, ai.assignee_name, ai.due_date, ai.priority, ai.status
		FROM action_items ai
		JOIN committees c ON c.id = ai.committee_id
		WHERE ai.tenant_id = $1
		  AND ai.status IN ('pending', 'in_progress', 'overdue')
		  AND ai.due_date < CURRENT_DATE
		ORDER BY ai.due_date ASC
		LIMIT 10`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.ActionItemSummary, 0, 10)
	for rows.Next() {
		var item model.ActionItemSummary
		if err := rows.Scan(&item.ID, &item.Title, &item.CommitteeID, &item.CommitteeName, &item.AssigneeName, &item.DueDate, &item.Priority, &item.Status); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *DashboardService) loadMeetingFrequencyChart(ctx context.Context, tenantID uuid.UUID) ([]MonthlyMeetingCount, error) {
	rows, err := s.store.DB().Query(ctx, `
		SELECT to_char(date_trunc('month', scheduled_at), 'YYYY-MM') AS month_key, COUNT(*)
		FROM meetings
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND scheduled_at >= date_trunc('month', now()) - interval '11 months'
		GROUP BY month_key
		ORDER BY month_key`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MonthlyMeetingCount, 0, 12)
	for rows.Next() {
		var item MonthlyMeetingCount
		if err := rows.Scan(&item.Month, &item.Count); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *DashboardService) loadAttendanceRateChart(ctx context.Context, tenantID uuid.UUID) ([]MonthlyAttendanceRate, error) {
	rows, err := s.store.DB().Query(ctx, `
		SELECT month_key, COALESCE(AVG(rate), 0)
		FROM (
			SELECT to_char(date_trunc('month', m.scheduled_at), 'YYYY-MM') AS month_key,
			       CASE WHEN COUNT(*) = 0 THEN 0
			            ELSE (COUNT(*) FILTER (WHERE ma.status IN ('present', 'proxy'))::float / COUNT(*)::float) * 100
			       END AS rate
			FROM meetings m
			LEFT JOIN meeting_attendance ma ON ma.meeting_id = m.id
			WHERE m.tenant_id = $1
			  AND m.deleted_at IS NULL
			  AND m.status = 'completed'
			  AND m.scheduled_at >= date_trunc('month', now()) - interval '11 months'
			GROUP BY m.id, month_key
		) rates
		GROUP BY month_key
		ORDER BY month_key`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MonthlyAttendanceRate, 0, 12)
	for rows.Next() {
		var item MonthlyAttendanceRate
		if err := rows.Scan(&item.Month, &item.RatePercent); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *DashboardService) loadRecentActivity(ctx context.Context, tenantID uuid.UUID) ([]AuditEntry, error) {
	rows, err := s.store.DB().Query(ctx, `
		SELECT occurred_at, event_type, message, entity_id
		FROM (
			SELECT updated_at AS occurred_at, 'meeting' AS event_type, title AS message, id::text AS entity_id
			FROM meetings
			WHERE tenant_id = $1 AND deleted_at IS NULL
			UNION ALL
			SELECT updated_at AS occurred_at, 'minutes' AS event_type, status::text AS message, id::text AS entity_id
			FROM meeting_minutes
			WHERE tenant_id = $1
			UNION ALL
			SELECT updated_at AS occurred_at, 'action_item' AS event_type, title AS message, id::text AS entity_id
			FROM action_items
			WHERE tenant_id = $1
			UNION ALL
			SELECT checked_at AS occurred_at, 'compliance' AS event_type, check_name AS message, id::text AS entity_id
			FROM compliance_checks
			WHERE tenant_id = $1
		) activity
		ORDER BY occurred_at DESC
		LIMIT 10`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AuditEntry, 0, 10)
	for rows.Next() {
		var item AuditEntry
		if err := rows.Scan(&item.Timestamp, &item.Type, &item.Message, &item.EntityID); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func complianceScoreFromStored(_ map[string]int, tenantID uuid.UUID, store *repository.Store) float64 {
	checks, _, err := store.ListComplianceChecks(context.Background(), tenantID, model.ComplianceFilters{Page: 1, PerPage: 250})
	if err != nil {
		return 0
	}
	return complianceScore(checks)
}
