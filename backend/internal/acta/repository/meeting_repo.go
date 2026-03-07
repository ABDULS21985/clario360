package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/clario360/platform/internal/acta/model"
)

func (s *Store) CreateMeeting(ctx context.Context, q DBTX, meeting *model.Meeting) error {
	metadata, err := marshalJSON(meeting.Metadata)
	if err != nil {
		return fmt.Errorf("marshal meeting metadata: %w", err)
	}
	_, err = q.Exec(ctx, `
		INSERT INTO meetings (
			id, tenant_id, committee_id, committee_name, title, description,
			meeting_number, scheduled_at, scheduled_end_at, actual_start_at, actual_end_at,
			duration_minutes, location, location_type, virtual_link, virtual_platform,
			status, cancellation_reason, quorum_required, attendee_count, present_count,
			quorum_met, agenda_item_count, action_item_count, has_minutes, minutes_status,
			workflow_instance_id, tags, metadata, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21,
			$22, $23, $24, $25, $26,
			$27, $28, $29, $30, $31, $32
		)`,
		meeting.ID,
		meeting.TenantID,
		meeting.CommitteeID,
		meeting.CommitteeName,
		meeting.Title,
		meeting.Description,
		meeting.MeetingNumber,
		meeting.ScheduledAt,
		nullableTime(meeting.ScheduledEndAt),
		nullableTime(meeting.ActualStartAt),
		nullableTime(meeting.ActualEndAt),
		meeting.DurationMinutes,
		meeting.Location,
		meeting.LocationType,
		meeting.VirtualLink,
		meeting.VirtualPlatform,
		meeting.Status,
		meeting.CancellationReason,
		meeting.QuorumRequired,
		meeting.AttendeeCount,
		meeting.PresentCount,
		meeting.QuorumMet,
		meeting.AgendaItemCount,
		meeting.ActionItemCount,
		meeting.HasMinutes,
		meeting.MinutesStatus,
		nullableUUID(meeting.WorkflowInstanceID),
		meeting.Tags,
		metadata,
		meeting.CreatedBy,
		meeting.CreatedAt,
		meeting.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create meeting: %w", err)
	}
	return nil
}

func (s *Store) UpdateMeeting(ctx context.Context, q DBTX, meeting *model.Meeting) error {
	metadata, err := marshalJSON(meeting.Metadata)
	if err != nil {
		return fmt.Errorf("marshal meeting metadata: %w", err)
	}
	tag, err := q.Exec(ctx, `
		UPDATE meetings
		SET committee_name = $3,
		    title = $4,
		    description = $5,
		    meeting_number = $6,
		    scheduled_at = $7,
		    scheduled_end_at = $8,
		    actual_start_at = $9,
		    actual_end_at = $10,
		    duration_minutes = $11,
		    location = $12,
		    location_type = $13,
		    virtual_link = $14,
		    virtual_platform = $15,
		    status = $16,
		    cancellation_reason = $17,
		    quorum_required = $18,
		    attendee_count = $19,
		    present_count = $20,
		    quorum_met = $21,
		    agenda_item_count = $22,
		    action_item_count = $23,
		    has_minutes = $24,
		    minutes_status = $25,
		    workflow_instance_id = $26,
		    tags = $27,
		    metadata = $28,
		    updated_at = $29
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		meeting.TenantID,
		meeting.ID,
		meeting.CommitteeName,
		meeting.Title,
		meeting.Description,
		meeting.MeetingNumber,
		meeting.ScheduledAt,
		nullableTime(meeting.ScheduledEndAt),
		nullableTime(meeting.ActualStartAt),
		nullableTime(meeting.ActualEndAt),
		meeting.DurationMinutes,
		meeting.Location,
		meeting.LocationType,
		meeting.VirtualLink,
		meeting.VirtualPlatform,
		meeting.Status,
		meeting.CancellationReason,
		meeting.QuorumRequired,
		meeting.AttendeeCount,
		meeting.PresentCount,
		meeting.QuorumMet,
		meeting.AgendaItemCount,
		meeting.ActionItemCount,
		meeting.HasMinutes,
		meeting.MinutesStatus,
		nullableUUID(meeting.WorkflowInstanceID),
		meeting.Tags,
		metadata,
		meeting.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update meeting: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return notFoundError("meeting", meeting.ID)
	}
	return nil
}

func (s *Store) GetMeeting(ctx context.Context, tenantID, meetingID uuid.UUID) (*model.Meeting, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, committee_id, committee_name, title, description,
		       meeting_number, scheduled_at, scheduled_end_at, actual_start_at, actual_end_at,
		       duration_minutes, location, location_type, virtual_link, virtual_platform,
		       status, cancellation_reason, quorum_required, attendee_count, present_count,
		       quorum_met, agenda_item_count, action_item_count, has_minutes, minutes_status,
		       workflow_instance_id, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM meetings
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, meetingID,
	)
	return scanMeeting(row)
}

func (s *Store) ListMeetings(ctx context.Context, tenantID uuid.UUID, filters model.MeetingFilters) ([]model.Meeting, int, error) {
	offset := (filters.Page - 1) * filters.PerPage
	where := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{tenantID}
	argPos := 2
	if filters.CommitteeID != nil {
		where = append(where, fmt.Sprintf("committee_id = $%d", argPos))
		args = append(args, *filters.CommitteeID)
		argPos++
	}
	if len(filters.Statuses) > 0 {
		statuses := make([]string, 0, len(filters.Statuses))
		for _, status := range filters.Statuses {
			statuses = append(statuses, string(status))
		}
		where = append(where, fmt.Sprintf("status = ANY($%d)", argPos))
		args = append(args, statuses)
		argPos++
	}
	if filters.DateFrom != nil {
		where = append(where, fmt.Sprintf("scheduled_at >= $%d", argPos))
		args = append(args, *filters.DateFrom)
		argPos++
	}
	if filters.DateTo != nil {
		where = append(where, fmt.Sprintf("scheduled_at <= $%d", argPos))
		args = append(args, *filters.DateTo)
		argPos++
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR committee_name ILIKE $%d OR description ILIKE $%d)", argPos, argPos, argPos))
		args = append(args, "%"+search+"%")
		argPos++
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM meetings WHERE "+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count meetings: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, committee_id, committee_name, title, description,
		       meeting_number, scheduled_at, scheduled_end_at, actual_start_at, actual_end_at,
		       duration_minutes, location, location_type, virtual_link, virtual_platform,
		       status, cancellation_reason, quorum_required, attendee_count, present_count,
		       quorum_met, agenda_item_count, action_item_count, has_minutes, minutes_status,
		       workflow_instance_id, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM meetings
		WHERE %s
		ORDER BY scheduled_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argPos, argPos+1,
	)
	args = append(args, filters.PerPage, offset)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list meetings: %w", err)
	}
	defer rows.Close()
	items := make([]model.Meeting, 0, filters.PerPage)
	for rows.Next() {
		item, err := scanMeeting(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (s *Store) CountMeetingConflicts(ctx context.Context, q DBTX, tenantID, committeeID uuid.UUID, startAt, endAt time.Time, excludeMeetingID *uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM meetings
		WHERE tenant_id = $1
		  AND committee_id = $2
		  AND deleted_at IS NULL
		  AND status IN ('draft', 'scheduled', 'in_progress', 'postponed')
		  AND scheduled_at < $4
		  AND COALESCE(scheduled_end_at, scheduled_at + make_interval(mins => duration_minutes)) > $3`
	args := []any{tenantID, committeeID, startAt, endAt}
	if excludeMeetingID != nil {
		query += ` AND id <> $5`
		args = append(args, *excludeMeetingID)
	}
	var count int
	if err := q.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count meeting conflicts: %w", err)
	}
	return count, nil
}

func (s *Store) NextMeetingNumber(ctx context.Context, q DBTX, tenantID, committeeID uuid.UUID) (int, error) {
	var next int
	err := q.QueryRow(ctx, `
		SELECT COALESCE(MAX(meeting_number), 0) + 1
		FROM meetings
		WHERE tenant_id = $1 AND committee_id = $2 AND deleted_at IS NULL`,
		tenantID, committeeID,
	).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("next meeting number: %w", err)
	}
	return next, nil
}

func (s *Store) UpdateMeetingAttendanceStats(ctx context.Context, q DBTX, tenantID, meetingID uuid.UUID, attendeeCount, presentCount int, quorumMet *bool) error {
	_, err := q.Exec(ctx, `
		UPDATE meetings
		SET attendee_count = $3,
		    present_count = $4,
		    quorum_met = $5,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, meetingID, attendeeCount, presentCount, quorumMet,
	)
	if err != nil {
		return fmt.Errorf("update meeting attendance stats: %w", err)
	}
	return nil
}

func (s *Store) UpdateMeetingAgendaCount(ctx context.Context, q DBTX, tenantID, meetingID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		UPDATE meetings
		SET agenda_item_count = (
			SELECT COUNT(*) FROM agenda_items WHERE tenant_id = $1 AND meeting_id = $2
		),
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, meetingID,
	)
	if err != nil {
		return fmt.Errorf("update meeting agenda count: %w", err)
	}
	return nil
}

func (s *Store) UpdateMeetingActionItemCount(ctx context.Context, q DBTX, tenantID, meetingID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		UPDATE meetings
		SET action_item_count = (
			SELECT COUNT(*) FROM action_items WHERE tenant_id = $1 AND meeting_id = $2
		),
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, meetingID,
	)
	if err != nil {
		return fmt.Errorf("update meeting action item count: %w", err)
	}
	return nil
}

func (s *Store) UpdateMeetingMinutesState(ctx context.Context, q DBTX, tenantID, meetingID uuid.UUID, hasMinutes bool, status *string) error {
	_, err := q.Exec(ctx, `
		UPDATE meetings
		SET has_minutes = $3,
		    minutes_status = $4,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, meetingID, hasMinutes, status,
	)
	if err != nil {
		return fmt.Errorf("update meeting minutes state: %w", err)
	}
	return nil
}

func (s *Store) ListUpcomingMeetings(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.MeetingSummary, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, committee_id, committee_name, title, status, scheduled_at, duration_minutes, location, quorum_met
		FROM meetings
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND status IN ('draft', 'scheduled', 'postponed')
		  AND scheduled_at >= now()
		ORDER BY scheduled_at ASC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list upcoming meetings: %w", err)
	}
	defer rows.Close()
	out := make([]model.MeetingSummary, 0, limit)
	for rows.Next() {
		var item model.MeetingSummary
		if err := rows.Scan(&item.ID, &item.CommitteeID, &item.CommitteeName, &item.Title, &item.Status, &item.ScheduledAt, &item.DurationMinutes, &item.Location, &item.QuorumMet); err != nil {
			return nil, fmt.Errorf("scan upcoming meeting: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListCalendarMeetings(ctx context.Context, tenantID uuid.UUID, month time.Time) ([]model.MeetingSummary, error) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	rows, err := s.db.Query(ctx, `
		SELECT id, committee_id, committee_name, title, status, scheduled_at, duration_minutes, location, quorum_met
		FROM meetings
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND scheduled_at >= $2
		  AND scheduled_at < $3
		ORDER BY scheduled_at ASC`,
		tenantID, start, end,
	)
	if err != nil {
		return nil, fmt.Errorf("list calendar meetings: %w", err)
	}
	defer rows.Close()
	out := make([]model.MeetingSummary, 0)
	for rows.Next() {
		var item model.MeetingSummary
		if err := rows.Scan(&item.ID, &item.CommitteeID, &item.CommitteeName, &item.Title, &item.Status, &item.ScheduledAt, &item.DurationMinutes, &item.Location, &item.QuorumMet); err != nil {
			return nil, fmt.Errorf("scan calendar meeting: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) GetNextMeeting(ctx context.Context, tenantID, committeeID uuid.UUID, after time.Time) (*model.Meeting, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, committee_id, committee_name, title, description,
		       meeting_number, scheduled_at, scheduled_end_at, actual_start_at, actual_end_at,
		       duration_minutes, location, location_type, virtual_link, virtual_platform,
		       status, cancellation_reason, quorum_required, attendee_count, present_count,
		       quorum_met, agenda_item_count, action_item_count, has_minutes, minutes_status,
		       workflow_instance_id, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM meetings
		WHERE tenant_id = $1
		  AND committee_id = $2
		  AND deleted_at IS NULL
		  AND status IN ('draft', 'scheduled', 'postponed')
		  AND scheduled_at > $3
		ORDER BY scheduled_at ASC
		LIMIT 1`,
		tenantID, committeeID, after,
	)
	item, err := scanMeeting(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return item, err
}

func scanMeeting(scanner rowScanner) (*model.Meeting, error) {
	var (
		item              model.Meeting
		metadataRaw       []byte
		meetingNumber     *int
		scheduledEndAt    *time.Time
		actualStartAt     *time.Time
		actualEndAt       *time.Time
		location          *string
		virtualLink       *string
		virtualPlatform   *string
		cancellationReason *string
		minutesStatus     *string
		workflowInstanceID *uuid.UUID
		deletedAt         *time.Time
	)
	if err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.CommitteeID,
		&item.CommitteeName,
		&item.Title,
		&item.Description,
		&meetingNumber,
		&item.ScheduledAt,
		&scheduledEndAt,
		&actualStartAt,
		&actualEndAt,
		&item.DurationMinutes,
		&location,
		&item.LocationType,
		&virtualLink,
		&virtualPlatform,
		&item.Status,
		&cancellationReason,
		&item.QuorumRequired,
		&item.AttendeeCount,
		&item.PresentCount,
		&item.QuorumMet,
		&item.AgendaItemCount,
		&item.ActionItemCount,
		&item.HasMinutes,
		&minutesStatus,
		&workflowInstanceID,
		&item.Tags,
		&metadataRaw,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, fmt.Errorf("scan meeting: %w", err)
	}
	item.MeetingNumber = meetingNumber
	item.ScheduledEndAt = scheduledEndAt
	item.ActualStartAt = actualStartAt
	item.ActualEndAt = actualEndAt
	item.Location = location
	item.VirtualLink = virtualLink
	item.VirtualPlatform = virtualPlatform
	item.CancellationReason = cancellationReason
	item.MinutesStatus = minutesStatus
	item.WorkflowInstanceID = workflowInstanceID
	item.DeletedAt = deletedAt
	metadata, err := decodeJSONMap(metadataRaw)
	if err != nil {
		return nil, fmt.Errorf("decode meeting metadata: %w", err)
	}
	item.Metadata = metadata
	item.Attachments = attachmentMetadata(metadata)
	return &item, nil
}
