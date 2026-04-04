package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/metrics"
	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/events"
)

type ComplianceService struct {
	store     *repository.Store
	publisher Publisher
	metrics   *metrics.Metrics
	logger    zerolog.Logger
}

func NewComplianceService(store *repository.Store, publisher Publisher, metrics *metrics.Metrics, logger zerolog.Logger) *ComplianceService {
	return &ComplianceService{
		store:     store,
		publisher: publisherOrNoop(publisher),
		metrics:   metrics,
		logger:    logger.With().Str("component", "acta_compliance_service").Logger(),
	}
}

func (s *ComplianceService) RunChecks(ctx context.Context, tenantID uuid.UUID) (*model.ComplianceReport, error) {
	now := time.Now().UTC()
	committees, err := s.store.ListActiveCommittees(ctx, &tenantID)
	if err != nil {
		return nil, internalError("failed to list active committees", err)
	}
	results := make([]model.ComplianceCheck, 0)
	for _, committee := range committees {
		results = append(results, s.runMeetingFrequencyCheck(ctx, tenantID, committee, now)...)
		results = append(results, s.runQuorumChecks(ctx, tenantID, committee, now)...)
		results = append(results, s.runMinutesCompletionChecks(ctx, tenantID, committee, now)...)
		results = append(results, s.runActionTrackingCheck(ctx, tenantID, committee, now)...)
		results = append(results, s.runAttendanceRateChecks(ctx, tenantID, committee, now)...)
		results = append(results, s.runCharterReviewCheck(tenantID, committee, now)...)
		results = append(results, s.runDocumentRetentionChecks(ctx, tenantID, committee, now)...)
		results = append(results, s.runConflictChecks(ctx, tenantID, committee, now)...)
	}

	if len(results) == 0 {
		report := &model.ComplianceReport{
			TenantID:          tenantID,
			Results:           []model.ComplianceCheck{},
			ByStatus:          map[string]int{},
			ByCheckType:       map[string]int{},
			ByCommittee:       []model.CommitteeCompliance{},
			Score:             100,
			NonCompliantCount: 0,
			WarningCount:      0,
			GeneratedAt:       now,
		}
		return report, nil
	}

	if err := s.store.InsertComplianceChecks(ctx, s.store.DB(), results); err != nil {
		return nil, internalError("failed to store compliance checks", err)
	}
	report := buildComplianceReport(tenantID, committees, results, now)
	if s.metrics != nil {
		s.metrics.ComplianceScore.WithLabelValues(tenantID.String()).Set(report.Score)
		for _, result := range results {
			s.metrics.ComplianceChecksTotal.WithLabelValues(string(result.CheckType), string(result.Status)).Inc()
		}
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.compliance.checked", tenantID, nil, map[string]any{
		"tenant_id":           tenantID,
		"score":               report.Score,
		"non_compliant_count": report.NonCompliantCount,
	}, s.logger)
	return report, nil
}

func (s *ComplianceService) ListResults(ctx context.Context, tenantID uuid.UUID, filters model.ComplianceFilters) ([]model.ComplianceCheck, int, error) {
	return s.store.ListComplianceChecks(ctx, tenantID, filters)
}

func (s *ComplianceService) Score(ctx context.Context, tenantID uuid.UUID) (float64, error) {
	results, _, err := s.store.ListComplianceChecks(ctx, tenantID, model.ComplianceFilters{Page: 1, PerPage: 500})
	if err != nil {
		return 0, err
	}
	report := buildComplianceReport(tenantID, nil, results, time.Now().UTC())
	return report.Score, nil
}

func (s *ComplianceService) runMeetingFrequencyCheck(ctx context.Context, tenantID uuid.UUID, committee model.Committee, now time.Time) []model.ComplianceCheck {
	start, end, applicable := lastCompletedPeriod(committee.MeetingFrequency, now)
	if !applicable {
		return []model.ComplianceCheck{newCheck(tenantID, &committee.ID, model.ComplianceCheckMeetingFrequency, "Meeting frequency", model.ComplianceStatusNotApplicable, model.ComplianceSeverityLow, "Committee meeting frequency is ad hoc and is not evaluated against a fixed cadence.", nil, nil, map[string]any{"frequency": committee.MeetingFrequency}, start, end, now)}
	}
	var completed, scheduled int
	_ = s.store.DB().QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'completed'),
			COUNT(*) FILTER (WHERE status IN ('draft', 'scheduled', 'postponed'))
		FROM meetings
		WHERE tenant_id = $1
		  AND committee_id = $2
		  AND deleted_at IS NULL
		  AND scheduled_at >= $3
		  AND scheduled_at < $4`,
		tenantID, committee.ID, start, end,
	).Scan(&completed, &scheduled)
	status := model.ComplianceStatusCompliant
	finding := ptr("Meeting cadence met for the last completed reporting period.")
	recommendation := ptr("Continue monitoring committee cadence.")
	if completed == 0 && scheduled > 0 {
		status = model.ComplianceStatusWarning
		finding = ptr("A meeting was scheduled in the last completed period but was not completed.")
		recommendation = ptr("Confirm whether the meeting was postponed or whether the committee missed its required cadence.")
	} else if completed == 0 {
		status = model.ComplianceStatusNonCompliant
		finding = ptr("No completed meeting was recorded in the last completed period.")
		recommendation = ptr("Schedule and complete the required committee meeting cadence.")
	}
	return []model.ComplianceCheck{newCheck(tenantID, &committee.ID, model.ComplianceCheckMeetingFrequency, "Meeting frequency", status, model.ComplianceSeverityHigh, "Validates the committee met during the last complete cadence period.", finding, recommendation, map[string]any{"expected": 1, "actual": completed, "scheduled": scheduled, "period_start": start, "period_end": end, "committee": committee.Name}, start, end, now)}
}

func (s *ComplianceService) runQuorumChecks(ctx context.Context, tenantID uuid.UUID, committee model.Committee, now time.Time) []model.ComplianceCheck {
	since := now.AddDate(0, -6, 0)
	rows, err := s.store.DB().Query(ctx, `
		SELECT id, scheduled_at, quorum_required, present_count, COALESCE(quorum_met, false)
		FROM meetings
		WHERE tenant_id = $1
		  AND committee_id = $2
		  AND deleted_at IS NULL
		  AND status = 'completed'
		  AND scheduled_at >= $3
		ORDER BY scheduled_at DESC`,
		tenantID, committee.ID, since,
	)
	if err != nil {
		s.logger.Error().Err(err).Str("committee_id", committee.ID.String()).Msg("failed to evaluate quorum compliance")
		return nil
	}
	defer rows.Close()

	results := make([]model.ComplianceCheck, 0)
	checked := 0
	missed := 0
	for rows.Next() {
		var meetingID uuid.UUID
		var scheduledAt time.Time
		var quorumRequired int
		var presentCount int
		var met bool
		if err := rows.Scan(&meetingID, &scheduledAt, &quorumRequired, &presentCount, &met); err != nil {
			continue
		}
		checked++
		status := model.ComplianceStatusCompliant
		finding := ptr("Meeting satisfied quorum requirements.")
		recommendation := ptr("Maintain current attendance controls.")
		if !met {
			missed++
			status = model.ComplianceStatusNonCompliant
			finding = ptr("Meeting ended without quorum.")
			recommendation = ptr("Ratify decisions in a properly constituted meeting and review attendance controls.")
		}
		results = append(results, newCheck(tenantID, &committee.ID, model.ComplianceCheckQuorumCompliance, "Meeting quorum compliance", status, model.ComplianceSeverityCritical, "Checks whether completed meetings satisfied quorum requirements.", finding, recommendation, map[string]any{"meeting_id": meetingID, "meeting_date": scheduledAt, "quorum_required": quorumRequired, "present_count": presentCount}, scheduledAt, scheduledAt, now))
	}
	aggregateStatus := model.ComplianceStatusCompliant
	aggregateFinding := ptr("All completed meetings met quorum requirements.")
	if checked == 0 {
		aggregateStatus = model.ComplianceStatusNotApplicable
		aggregateFinding = ptr("No completed meetings were available for quorum review.")
	} else if missed > 0 {
		aggregateStatus = model.ComplianceStatusWarning
		aggregateFinding = ptr("One or more meetings in the review period missed quorum.")
	}
	results = append(results, newCheck(tenantID, &committee.ID, model.ComplianceCheckQuorumCompliance, "Quorum compliance summary", aggregateStatus, model.ComplianceSeverityHigh, "Summarizes quorum performance over the last six months.", aggregateFinding, ptr("Review attendance patterns and escalation processes for recurring quorum misses."), map[string]any{"meetings_checked": checked, "quorum_met": checked - missed, "quorum_not_met": missed}, since, now, now))
	return results
}

func (s *ComplianceService) runMinutesCompletionChecks(ctx context.Context, tenantID uuid.UUID, committee model.Committee, now time.Time) []model.ComplianceCheck {
	rows, err := s.store.DB().Query(ctx, `
		SELECT m.id, m.scheduled_at, COALESCE(mm.status, '')
		FROM meetings m
		LEFT JOIN LATERAL (
			SELECT status
			FROM meeting_minutes
			WHERE tenant_id = m.tenant_id AND meeting_id = m.id
			ORDER BY version DESC
			LIMIT 1
		) mm ON true
		WHERE m.tenant_id = $1
		  AND m.committee_id = $2
		  AND m.deleted_at IS NULL
		  AND m.status = 'completed'`,
		tenantID, committee.ID,
	)
	if err != nil {
		s.logger.Error().Err(err).Str("committee_id", committee.ID.String()).Msg("failed to evaluate minutes completion")
		return nil
	}
	defer rows.Close()
	results := make([]model.ComplianceCheck, 0)
	for rows.Next() {
		var meetingID uuid.UUID
		var meetingDate time.Time
		var minutesStatus string
		if err := rows.Scan(&meetingID, &meetingDate, &minutesStatus); err != nil {
			continue
		}
		days := businessDaysBetween(meetingDate, now)
		if days <= 5 {
			continue
		}
		status := model.ComplianceStatusCompliant
		finding := ptr("Minutes were approved within the expected business-day window.")
		recommendation := ptr("Maintain the current review cadence.")
		switch minutesStatus {
		case "approved", "published":
		case "review":
			status = model.ComplianceStatusWarning
			finding = ptr("Minutes are still in review more than five business days after the meeting.")
			recommendation = ptr("Complete the chair approval workflow.")
		default:
			status = model.ComplianceStatusNonCompliant
			finding = ptr("Minutes are incomplete or not approved more than five business days after the meeting.")
			recommendation = ptr("Complete the minutes workflow and obtain chair approval.")
		}
		results = append(results, newCheck(tenantID, &committee.ID, model.ComplianceCheckMinutesCompletion, "Minutes completion", status, model.ComplianceSeverityHigh, "Checks whether meeting minutes were completed and approved within five business days.", finding, recommendation, map[string]any{"meeting_id": meetingID, "meeting_date": meetingDate, "minutes_status": minutesStatus, "days_since_meeting": days}, meetingDate, now, now))
	}
	return results
}

func (s *ComplianceService) runActionTrackingCheck(ctx context.Context, tenantID uuid.UUID, committee model.Committee, now time.Time) []model.ComplianceCheck {
	var overdueCount, totalOpen int
	var oldestOverdue *time.Time
	_ = s.store.DB().QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE due_date < CURRENT_DATE AND status IN ('pending', 'in_progress', 'overdue')),
			COUNT(*) FILTER (WHERE status IN ('pending', 'in_progress', 'deferred', 'overdue')),
			MIN(due_date) FILTER (WHERE due_date < CURRENT_DATE AND status IN ('pending', 'in_progress', 'overdue'))
		FROM action_items
		WHERE tenant_id = $1 AND committee_id = $2`,
		tenantID, committee.ID,
	).Scan(&overdueCount, &totalOpen, &oldestOverdue)
	status := model.ComplianceStatusCompliant
	finding := ptr("No overdue action items were found.")
	recommendation := ptr("Continue monitoring open action items.")
	if overdueCount >= 1 && overdueCount <= 5 {
		status = model.ComplianceStatusWarning
		finding = ptr("A small number of action items are overdue.")
		recommendation = ptr("Review ownership and extend or complete overdue action items.")
	} else if overdueCount > 5 {
		status = model.ComplianceStatusNonCompliant
		finding = ptr("More than five action items are overdue.")
		recommendation = ptr("Escalate overdue actions and review committee follow-up discipline.")
	}
	return []model.ComplianceCheck{newCheck(tenantID, &committee.ID, model.ComplianceCheckActionTracking, "Action item tracking", status, model.ComplianceSeverityMedium, "Counts overdue action items for the committee.", finding, recommendation, map[string]any{"overdue_count": overdueCount, "total_open": totalOpen, "oldest_overdue_date": oldestOverdue}, now.AddDate(0, -6, 0), now, now)}
}

func (s *ComplianceService) runAttendanceRateChecks(ctx context.Context, tenantID uuid.UUID, committee model.Committee, now time.Time) []model.ComplianceCheck {
	since := now.AddDate(0, -6, 0)
	rows, err := s.store.DB().Query(ctx, `
		SELECT ma.user_name,
		       COUNT(*) FILTER (WHERE ma.status IN ('present', 'proxy')) AS meetings_attended,
		       COUNT(*) AS meetings_total
		FROM meeting_attendance ma
		JOIN meetings m ON m.id = ma.meeting_id
		WHERE ma.tenant_id = $1
		  AND m.committee_id = $2
		  AND m.status = 'completed'
		  AND m.deleted_at IS NULL
		  AND m.scheduled_at >= $3
		GROUP BY ma.user_name
		ORDER BY ma.user_name`,
		tenantID, committee.ID, since,
	)
	if err != nil {
		s.logger.Error().Err(err).Str("committee_id", committee.ID.String()).Msg("failed to evaluate attendance rate")
		return nil
	}
	defer rows.Close()
	results := make([]model.ComplianceCheck, 0)
	for rows.Next() {
		var name string
		var attended, total int
		if err := rows.Scan(&name, &attended, &total); err != nil {
			continue
		}
		if total == 0 {
			continue
		}
		rate := float64(attended) / float64(total) * 100
		status := model.ComplianceStatusCompliant
		if rate < 50 {
			status = model.ComplianceStatusNonCompliant
		} else if rate < 75 {
			status = model.ComplianceStatusWarning
		}
		finding := ptr("Attendance rate is within the expected threshold.")
		if status == model.ComplianceStatusWarning {
			finding = ptr("Attendance rate is below the 75% target.")
		}
		if status == model.ComplianceStatusNonCompliant {
			finding = ptr("Attendance rate is below 50%.")
		}
		results = append(results, newCheck(tenantID, &committee.ID, model.ComplianceCheckAttendanceRate, "Attendance rate", status, model.ComplianceSeverityMedium, "Measures individual attendance rates over the last six months.", finding, ptr("Follow up with the member and review participation requirements."), map[string]any{"member_name": name, "meetings_attended": attended, "meetings_total": total, "rate_percent": rate}, since, now, now))
	}
	return results
}

func (s *ComplianceService) runCharterReviewCheck(tenantID uuid.UUID, committee model.Committee, now time.Time) []model.ComplianceCheck {
	reviewedAt := committee.UpdatedAt
	if raw, ok := committee.Metadata["charter_reviewed_at"].(string); ok && raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			reviewedAt = parsed.UTC()
		}
	}
	status := model.ComplianceStatusCompliant
	finding := ptr("Committee charter has been reviewed within the last twelve months.")
	if reviewedAt.Before(now.AddDate(-1, 0, 0)) {
		status = model.ComplianceStatusWarning
		finding = ptr("Committee charter has not been reviewed in the last twelve months.")
	}
	return []model.ComplianceCheck{newCheck(tenantID, &committee.ID, model.ComplianceCheckCharterReview, "Charter review", status, model.ComplianceSeverityLow, "Checks whether the committee charter was reviewed in the last twelve months.", finding, ptr("Review and reaffirm the committee charter."), map[string]any{"reviewed_at": reviewedAt}, now.AddDate(-1, 0, 0), now, now)}
}

func (s *ComplianceService) runDocumentRetentionChecks(ctx context.Context, tenantID uuid.UUID, committee model.Committee, now time.Time) []model.ComplianceCheck {
	rows, err := s.store.DB().Query(ctx, `
		SELECT m.id, m.scheduled_at, m.has_minutes, COALESCE(array_length(a.attachments, 1), 0) AS attachment_refs
		FROM meetings m
		LEFT JOIN (
			SELECT meeting_id, array_agg(DISTINCT unnest(attachment_ids)) AS attachments
			FROM agenda_items
			WHERE tenant_id = $1
			GROUP BY meeting_id
		) a ON a.meeting_id = m.id
		WHERE m.tenant_id = $1
		  AND m.committee_id = $2
		  AND m.status = 'completed'
		  AND m.deleted_at IS NULL`,
		tenantID, committee.ID,
	)
	if err != nil {
		s.logger.Error().Err(err).Str("committee_id", committee.ID.String()).Msg("failed to evaluate document retention")
		return nil
	}
	defer rows.Close()
	results := make([]model.ComplianceCheck, 0)
	for rows.Next() {
		var meetingID uuid.UUID
		var meetingDate time.Time
		var hasMinutes bool
		var attachmentRefs int
		if err := rows.Scan(&meetingID, &meetingDate, &hasMinutes, &attachmentRefs); err != nil {
			continue
		}
		status := model.ComplianceStatusCompliant
		finding := ptr("Required minutes and referenced documents are present in the Acta record.")
		if !hasMinutes {
			status = model.ComplianceStatusNonCompliant
			finding = ptr("Completed meeting is missing official minutes.")
		}
		results = append(results, newCheck(tenantID, &committee.ID, model.ComplianceCheckDocumentRetention, "Document retention", status, model.ComplianceSeverityHigh, "Checks whether completed meetings retain required minutes and referenced documents.", finding, ptr("Ensure meeting minutes and supporting documents are retained in line with policy."), map[string]any{"meeting_id": meetingID, "meeting_date": meetingDate, "has_minutes": hasMinutes, "attachment_references": attachmentRefs}, meetingDate, now, now))
	}
	return results
}

func (s *ComplianceService) runConflictChecks(ctx context.Context, tenantID uuid.UUID, committee model.Committee, now time.Time) []model.ComplianceCheck {
	rows, err := s.store.DB().Query(ctx, `
		SELECT a.id, a.title, a.presenter_name, m.id
		FROM agenda_items a
		JOIN meetings m ON m.id = a.meeting_id
		JOIN meeting_attendance ma ON ma.meeting_id = m.id AND ma.user_id = a.presenter_user_id AND ma.status IN ('present', 'proxy')
		WHERE a.tenant_id = $1
		  AND m.committee_id = $2
		  AND m.status = 'completed'
		  AND a.requires_vote = true
		  AND a.presenter_user_id IS NOT NULL`,
		tenantID, committee.ID,
	)
	if err != nil {
		s.logger.Error().Err(err).Str("committee_id", committee.ID.String()).Msg("failed to evaluate conflicts of interest")
		return nil
	}
	defer rows.Close()
	results := make([]model.ComplianceCheck, 0)
	conflicts := 0
	for rows.Next() {
		var agendaID uuid.UUID
		var title string
		var presenter string
		var meetingID uuid.UUID
		if err := rows.Scan(&agendaID, &title, &presenter, &meetingID); err != nil {
			continue
		}
		conflicts++
		results = append(results, newCheck(tenantID, &committee.ID, model.ComplianceCheckConflictOfInterest, "Conflict of interest", model.ComplianceStatusWarning, model.ComplianceSeverityMedium, "Flags agenda items where the presenter was present for a vote requiring decision.", ptr("Potential conflict detected for a voting agenda item presenter."), ptr("Review conflict declarations and recusal controls."), map[string]any{"agenda_item_id": agendaID, "agenda_title": title, "presenter_name": presenter, "meeting_id": meetingID}, now.AddDate(0, -6, 0), now, now))
	}
	if conflicts == 0 {
		results = append(results, newCheck(tenantID, &committee.ID, model.ComplianceCheckConflictOfInterest, "Conflict of interest", model.ComplianceStatusCompliant, model.ComplianceSeverityLow, "Flags agenda items where the presenter was present for a vote requiring decision.", ptr("No potential presenter-voter conflicts were detected."), ptr("Continue collecting conflict declarations for decision items."), map[string]any{"conflicts_detected": 0}, now.AddDate(0, -6, 0), now, now))
	}
	return results
}

func buildComplianceReport(tenantID uuid.UUID, committees []model.Committee, results []model.ComplianceCheck, now time.Time) *model.ComplianceReport {
	byStatus := make(map[string]int)
	byCheckType := make(map[string]int)
	committeeLookup := make(map[uuid.UUID]string)
	for _, committee := range committees {
		committeeLookup[committee.ID] = committee.Name
	}
	committeeCounters := make(map[uuid.UUID]*model.CommitteeCompliance)
	var nonCompliant, warnings int
	for _, result := range results {
		byStatus[string(result.Status)]++
		byCheckType[string(result.CheckType)]++
		if result.Status == model.ComplianceStatusNonCompliant {
			nonCompliant++
		}
		if result.Status == model.ComplianceStatusWarning {
			warnings++
		}
		if result.CommitteeID != nil {
			entry, ok := committeeCounters[*result.CommitteeID]
			if !ok {
				entry = &model.CommitteeCompliance{
					CommitteeID:   *result.CommitteeID,
					CommitteeName: committeeLookup[*result.CommitteeID],
				}
				committeeCounters[*result.CommitteeID] = entry
			}
			if result.Status == model.ComplianceStatusWarning {
				entry.Warnings++
			}
			if result.Status == model.ComplianceStatusNonCompliant {
				entry.NonCompliant++
			}
		}
	}
	score := complianceScore(results)
	byCommittee := make([]model.CommitteeCompliance, 0, len(committeeCounters))
	for committeeID, item := range committeeCounters {
		item.Score = complianceScore(filterCommitteeResults(results, committeeID))
		byCommittee = append(byCommittee, *item)
	}
	return &model.ComplianceReport{
		TenantID:          tenantID,
		Results:           results,
		ByStatus:          byStatus,
		ByCheckType:       byCheckType,
		ByCommittee:       byCommittee,
		Score:             score,
		NonCompliantCount: nonCompliant,
		WarningCount:      warnings,
		GeneratedAt:       now,
	}
}

func complianceScore(results []model.ComplianceCheck) float64 {
	if len(results) == 0 {
		return 100
	}
	var totalWeight float64
	var passedWeight float64
	for _, result := range results {
		if result.Status == model.ComplianceStatusNotApplicable {
			continue
		}
		weight := severityWeight(string(result.Severity))
		totalWeight += weight
		if result.Status == model.ComplianceStatusCompliant {
			passedWeight += weight
		}
	}
	if totalWeight == 0 {
		return 100
	}
	return (passedWeight / totalWeight) * 100
}

func filterCommitteeResults(results []model.ComplianceCheck, committeeID uuid.UUID) []model.ComplianceCheck {
	out := make([]model.ComplianceCheck, 0)
	for _, result := range results {
		if result.CommitteeID != nil && *result.CommitteeID == committeeID {
			out = append(out, result)
		}
	}
	return out
}

func newCheck(tenantID uuid.UUID, committeeID *uuid.UUID, checkType model.ComplianceCheckType, name string, status model.ComplianceStatus, severity model.ComplianceSeverity, description string, finding, recommendation *string, evidence map[string]any, periodStart, periodEnd, now time.Time) model.ComplianceCheck {
	return model.ComplianceCheck{
		ID:             uuid.New(),
		TenantID:       tenantID,
		CommitteeID:    committeeID,
		CheckType:      checkType,
		CheckName:      name,
		Status:         status,
		Severity:       severity,
		Description:    description,
		Finding:        finding,
		Recommendation: recommendation,
		Evidence:       evidence,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		CheckedAt:      now,
		CheckedBy:      "system",
		CreatedAt:      now,
	}
}

func lastCompletedPeriod(freq model.MeetingFrequency, now time.Time) (time.Time, time.Time, bool) {
	switch freq {
	case model.MeetingFrequencyWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(weekday - 1))
		start := end.AddDate(0, 0, -7)
		return start, end, true
	case model.MeetingFrequencyBiWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(weekday - 1))
		start := end.AddDate(0, 0, -14)
		return start, end, true
	case model.MeetingFrequencyMonthly:
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		start := firstOfMonth.AddDate(0, -1, 0)
		return start, firstOfMonth, true
	case model.MeetingFrequencyQuarterly:
		month := ((int(now.Month())-1)/3)*3 + 1
		currentQuarter := time.Date(now.Year(), time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		start := currentQuarter.AddDate(0, -3, 0)
		return start, currentQuarter, true
	case model.MeetingFrequencySemiAnnual:
		startMonth := time.Month((((int(now.Month()) - 1) / 6) * 6) + 1)
		startOfHalf := time.Date(now.Year(), startMonth, 1, 0, 0, 0, 0, time.UTC)
		return startOfHalf.AddDate(0, -6, 0), startOfHalf, true
	case model.MeetingFrequencyAnnual:
		startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return startOfYear.AddDate(-1, 0, 0), startOfYear, true
	default:
		return now, now, false
	}
}
