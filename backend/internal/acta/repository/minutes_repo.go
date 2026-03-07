package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/clario360/platform/internal/acta/model"
)

func (s *Store) CreateMinutes(ctx context.Context, q DBTX, minutes *model.MeetingMinutes) error {
	actions, err := marshalJSONArray(minutes.AIActionItems)
	if err != nil {
		return fmt.Errorf("marshal minutes action items: %w", err)
	}
	_, err = q.Exec(ctx, `
		INSERT INTO meeting_minutes (
			id, tenant_id, meeting_id, content, ai_summary, status,
			submitted_for_review_at, submitted_by, reviewed_by, review_notes,
			approved_by, approved_at, published_at, version, previous_version_id,
			ai_action_items, ai_generated, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20
		)`,
		minutes.ID,
		minutes.TenantID,
		minutes.MeetingID,
		minutes.Content,
		minutes.AISummary,
		minutes.Status,
		nullableTime(minutes.SubmittedForReviewAt),
		nullableUUID(minutes.SubmittedBy),
		nullableUUID(minutes.ReviewedBy),
		minutes.ReviewNotes,
		nullableUUID(minutes.ApprovedBy),
		nullableTime(minutes.ApprovedAt),
		nullableTime(minutes.PublishedAt),
		minutes.Version,
		nullableUUID(minutes.PreviousVersionID),
		actions,
		minutes.AIGenerated,
		minutes.CreatedBy,
		minutes.CreatedAt,
		minutes.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create meeting minutes: %w", err)
	}
	return nil
}

func (s *Store) UpdateMinutes(ctx context.Context, q DBTX, minutes *model.MeetingMinutes) error {
	actions, err := marshalJSONArray(minutes.AIActionItems)
	if err != nil {
		return fmt.Errorf("marshal minutes action items: %w", err)
	}
	tag, err := q.Exec(ctx, `
		UPDATE meeting_minutes
		SET content = $4,
		    ai_summary = $5,
		    status = $6,
		    submitted_for_review_at = $7,
		    submitted_by = $8,
		    reviewed_by = $9,
		    review_notes = $10,
		    approved_by = $11,
		    approved_at = $12,
		    published_at = $13,
		    ai_action_items = $14,
		    ai_generated = $15,
		    updated_at = $16
		WHERE tenant_id = $1 AND meeting_id = $2 AND id = $3`,
		minutes.TenantID,
		minutes.MeetingID,
		minutes.ID,
		minutes.Content,
		minutes.AISummary,
		minutes.Status,
		nullableTime(minutes.SubmittedForReviewAt),
		nullableUUID(minutes.SubmittedBy),
		nullableUUID(minutes.ReviewedBy),
		minutes.ReviewNotes,
		nullableUUID(minutes.ApprovedBy),
		nullableTime(minutes.ApprovedAt),
		nullableTime(minutes.PublishedAt),
		actions,
		minutes.AIGenerated,
		minutes.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update meeting minutes: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return notFoundError("meeting_minutes", minutes.ID)
	}
	return nil
}

func (s *Store) GetLatestMinutes(ctx context.Context, tenantID, meetingID uuid.UUID) (*model.MeetingMinutes, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, meeting_id, content, ai_summary, status,
		       submitted_for_review_at, submitted_by, reviewed_by, review_notes,
		       approved_by, approved_at, published_at, version, previous_version_id,
		       ai_action_items, ai_generated, created_by, created_at, updated_at
		FROM meeting_minutes
		WHERE tenant_id = $1 AND meeting_id = $2
		ORDER BY version DESC
		LIMIT 1`,
		tenantID, meetingID,
	)
	return scanMinutes(row)
}

func (s *Store) ListMinutesVersions(ctx context.Context, tenantID, meetingID uuid.UUID) ([]model.MeetingMinutes, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, meeting_id, content, ai_summary, status,
		       submitted_for_review_at, submitted_by, reviewed_by, review_notes,
		       approved_by, approved_at, published_at, version, previous_version_id,
		       ai_action_items, ai_generated, created_by, created_at, updated_at
		FROM meeting_minutes
		WHERE tenant_id = $1 AND meeting_id = $2
		ORDER BY version DESC`,
		tenantID, meetingID,
	)
	if err != nil {
		return nil, fmt.Errorf("list minutes versions: %w", err)
	}
	defer rows.Close()
	out := make([]model.MeetingMinutes, 0)
	for rows.Next() {
		item, err := scanMinutes(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func scanMinutes(scanner rowScanner) (*model.MeetingMinutes, error) {
	var (
		item        model.MeetingMinutes
		actionsRaw  []byte
		aiSummary   *string
		reviewNotes *string
		submittedAt *time.Time
		submittedBy *uuid.UUID
		reviewedBy  *uuid.UUID
		approvedBy  *uuid.UUID
		approvedAt  *time.Time
		publishedAt *time.Time
		previousID  *uuid.UUID
	)
	err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.MeetingID,
		&item.Content,
		&aiSummary,
		&item.Status,
		&submittedAt,
		&submittedBy,
		&reviewedBy,
		&reviewNotes,
		&approvedBy,
		&approvedAt,
		&publishedAt,
		&item.Version,
		&previousID,
		&actionsRaw,
		&item.AIGenerated,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("scan minutes: %w", err)
	}
	item.AISummary = aiSummary
	item.SubmittedForReviewAt = submittedAt
	item.SubmittedBy = submittedBy
	item.ReviewedBy = reviewedBy
	item.ReviewNotes = reviewNotes
	item.ApprovedBy = approvedBy
	item.ApprovedAt = approvedAt
	item.PublishedAt = publishedAt
	item.PreviousVersionID = previousID
	item.AIActionItems, err = decodeExtractedActions(actionsRaw)
	if err != nil {
		return nil, fmt.Errorf("decode minutes action items: %w", err)
	}
	return &item, nil
}
