package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

func (s *Store) CreateAttendanceRecords(ctx context.Context, q DBTX, records []model.Attendee) error {
	for _, record := range records {
		_, err := q.Exec(ctx, `
			INSERT INTO meeting_attendance (
				id, tenant_id, meeting_id, user_id, user_name, user_email, member_role,
				status, confirmed_at, checked_in_at, checked_out_at, proxy_user_id,
				proxy_user_name, proxy_authorized_by, notes, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11, $12,
				$13, $14, $15, $16, $17
			)`,
			record.ID,
			record.TenantID,
			record.MeetingID,
			record.UserID,
			record.UserName,
			record.UserEmail,
			record.MemberRole,
			record.Status,
			nullableTime(record.ConfirmedAt),
			nullableTime(record.CheckedInAt),
			nullableTime(record.CheckedOutAt),
			nullableUUID(record.ProxyUserID),
			record.ProxyUserName,
			nullableUUID(record.ProxyAuthorizedBy),
			record.Notes,
			record.CreatedAt,
			record.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert attendance record: %w", err)
		}
	}
	return nil
}

func (s *Store) ListAttendance(ctx context.Context, tenantID, meetingID uuid.UUID) ([]model.Attendee, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, meeting_id, user_id, user_name, user_email, member_role,
		       status, confirmed_at, checked_in_at, checked_out_at, proxy_user_id,
		       proxy_user_name, proxy_authorized_by, notes, created_at, updated_at
		FROM meeting_attendance
		WHERE tenant_id = $1 AND meeting_id = $2
		ORDER BY member_role ASC, user_name ASC`,
		tenantID, meetingID,
	)
	if err != nil {
		return nil, fmt.Errorf("list attendance: %w", err)
	}
	defer rows.Close()
	out := make([]model.Attendee, 0)
	for rows.Next() {
		item, err := scanAttendee(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) UpdateAttendanceRecord(ctx context.Context, q DBTX, tenantID, meetingID uuid.UUID, attendee *model.Attendee) error {
	tag, err := q.Exec(ctx, `
		UPDATE meeting_attendance
		SET user_name = $4,
		    user_email = $5,
		    member_role = $6,
		    status = $7,
		    confirmed_at = $8,
		    checked_in_at = $9,
		    checked_out_at = $10,
		    proxy_user_id = $11,
		    proxy_user_name = $12,
		    proxy_authorized_by = $13,
		    notes = $14,
		    updated_at = $15
		WHERE tenant_id = $1 AND meeting_id = $2 AND user_id = $3`,
		tenantID,
		meetingID,
		attendee.UserID,
		attendee.UserName,
		attendee.UserEmail,
		attendee.MemberRole,
		attendee.Status,
		nullableTime(attendee.ConfirmedAt),
		nullableTime(attendee.CheckedInAt),
		nullableTime(attendee.CheckedOutAt),
		nullableUUID(attendee.ProxyUserID),
		attendee.ProxyUserName,
		nullableUUID(attendee.ProxyAuthorizedBy),
		attendee.Notes,
		attendee.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update attendance record: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("attendance record %s not found", attendee.UserID)
	}
	return nil
}

func (s *Store) CountAttendees(ctx context.Context, q DBTX, tenantID, meetingID uuid.UUID) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM meeting_attendance
		WHERE tenant_id = $1 AND meeting_id = $2`,
		tenantID, meetingID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count attendance: %w", err)
	}
	return count, nil
}

func (s *Store) CountPresentAttendance(ctx context.Context, q DBTX, tenantID, meetingID uuid.UUID) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM meeting_attendance
		WHERE tenant_id = $1
		  AND meeting_id = $2
		  AND status IN ('present', 'proxy')`,
		tenantID, meetingID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count present attendance: %w", err)
	}
	return count, nil
}

func (s *Store) MarkUncheckedAttendeesAbsent(ctx context.Context, q DBTX, tenantID, meetingID uuid.UUID, updatedAt time.Time) error {
	_, err := q.Exec(ctx, `
		UPDATE meeting_attendance
		SET status = 'absent',
		    updated_at = $3
		WHERE tenant_id = $1
		  AND meeting_id = $2
		  AND status IN ('invited', 'confirmed')`,
		tenantID, meetingID, updatedAt,
	)
	if err != nil {
		return fmt.Errorf("mark unchecked attendance absent: %w", err)
	}
	return nil
}

func (s *Store) GetAttendanceRecord(ctx context.Context, tenantID, meetingID, userID uuid.UUID) (*model.Attendee, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, meeting_id, user_id, user_name, user_email, member_role,
		       status, confirmed_at, checked_in_at, checked_out_at, proxy_user_id,
		       proxy_user_name, proxy_authorized_by, notes, created_at, updated_at
		FROM meeting_attendance
		WHERE tenant_id = $1 AND meeting_id = $2 AND user_id = $3`,
		tenantID, meetingID, userID,
	)
	return scanAttendee(row)
}

func scanAttendee(scanner rowScanner) (*model.Attendee, error) {
	var item model.Attendee
	err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.MeetingID,
		&item.UserID,
		&item.UserName,
		&item.UserEmail,
		&item.MemberRole,
		&item.Status,
		&item.ConfirmedAt,
		&item.CheckedInAt,
		&item.CheckedOutAt,
		&item.ProxyUserID,
		&item.ProxyUserName,
		&item.ProxyAuthorizedBy,
		&item.Notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan attendee: %w", err)
	}
	return &item, nil
}
