package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

func (s *Store) CreateActionItem(ctx context.Context, q DBTX, item *model.ActionItem) error {
	metadata, err := marshalJSON(item.Metadata)
	if err != nil {
		return fmt.Errorf("marshal action item metadata: %w", err)
	}
	_, err = q.Exec(ctx, `
		INSERT INTO action_items (
			id, tenant_id, meeting_id, agenda_item_id, committee_id, title, description,
			priority, assigned_to, assignee_name, assigned_by, due_date, original_due_date,
			extended_count, extension_reason, status, completed_at, completion_notes,
			completion_evidence, follow_up_meeting_id, reviewed_at, tags, metadata,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23,
			$24, $25, $26
		)`,
		item.ID,
		item.TenantID,
		item.MeetingID,
		nullableUUID(item.AgendaItemID),
		item.CommitteeID,
		item.Title,
		item.Description,
		item.Priority,
		item.AssignedTo,
		item.AssigneeName,
		item.AssignedBy,
		item.DueDate,
		item.OriginalDueDate,
		item.ExtendedCount,
		item.ExtensionReason,
		item.Status,
		nullableTime(item.CompletedAt),
		item.CompletionNotes,
		item.CompletionEvidence,
		nullableUUID(item.FollowUpMeetingID),
		nullableTime(item.ReviewedAt),
		item.Tags,
		metadata,
		item.CreatedBy,
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create action item: %w", err)
	}
	return nil
}

func (s *Store) UpdateActionItem(ctx context.Context, q DBTX, item *model.ActionItem) error {
	metadata, err := marshalJSON(item.Metadata)
	if err != nil {
		return fmt.Errorf("marshal action item metadata: %w", err)
	}
	tag, err := q.Exec(ctx, `
		UPDATE action_items
		SET agenda_item_id = $4,
		    committee_id = $5,
		    title = $6,
		    description = $7,
		    priority = $8,
		    assigned_to = $9,
		    assignee_name = $10,
		    due_date = $11,
		    original_due_date = $12,
		    extended_count = $13,
		    extension_reason = $14,
		    status = $15,
		    completed_at = $16,
		    completion_notes = $17,
		    completion_evidence = $18,
		    follow_up_meeting_id = $19,
		    reviewed_at = $20,
		    tags = $21,
		    metadata = $22,
		    updated_at = $23
		WHERE tenant_id = $1 AND id = $2 AND meeting_id = $3`,
		item.TenantID,
		item.ID,
		item.MeetingID,
		nullableUUID(item.AgendaItemID),
		item.CommitteeID,
		item.Title,
		item.Description,
		item.Priority,
		item.AssignedTo,
		item.AssigneeName,
		item.DueDate,
		item.OriginalDueDate,
		item.ExtendedCount,
		item.ExtensionReason,
		item.Status,
		nullableTime(item.CompletedAt),
		item.CompletionNotes,
		item.CompletionEvidence,
		nullableUUID(item.FollowUpMeetingID),
		nullableTime(item.ReviewedAt),
		item.Tags,
		metadata,
		item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update action item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return notFoundError("action_item", item.ID)
	}
	return nil
}

func (s *Store) GetActionItem(ctx context.Context, tenantID, itemID uuid.UUID) (*model.ActionItem, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, meeting_id, agenda_item_id, committee_id, title, description,
		       priority, assigned_to, assignee_name, assigned_by, due_date, original_due_date,
		       extended_count, extension_reason, status, completed_at, completion_notes,
		       completion_evidence, follow_up_meeting_id, reviewed_at, tags, metadata,
		       created_by, created_at, updated_at
		FROM action_items
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, itemID,
	)
	return scanActionItem(row)
}

func (s *Store) ListActionItems(ctx context.Context, tenantID uuid.UUID, filters model.ActionItemFilters) ([]model.ActionItem, int, error) {
	offset := (filters.Page - 1) * filters.PerPage
	where := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argPos := 2
	if filters.CommitteeID != nil {
		where = append(where, fmt.Sprintf("committee_id = $%d", argPos))
		args = append(args, *filters.CommitteeID)
		argPos++
	}
	if filters.MeetingID != nil {
		where = append(where, fmt.Sprintf("meeting_id = $%d", argPos))
		args = append(args, *filters.MeetingID)
		argPos++
	}
	if filters.AssigneeID != nil {
		where = append(where, fmt.Sprintf("assigned_to = $%d", argPos))
		args = append(args, *filters.AssigneeID)
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
	if filters.OverdueOnly {
		where = append(where, "due_date < CURRENT_DATE", "status IN ('pending', 'in_progress', 'overdue')")
	}
	whereClause := strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM action_items WHERE "+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count action items: %w", err)
	}
	query := fmt.Sprintf(`
		SELECT id, tenant_id, meeting_id, agenda_item_id, committee_id, title, description,
		       priority, assigned_to, assignee_name, assigned_by, due_date, original_due_date,
		       extended_count, extension_reason, status, completed_at, completion_notes,
		       completion_evidence, follow_up_meeting_id, reviewed_at, tags, metadata,
		       created_by, created_at, updated_at
		FROM action_items
		WHERE %s
		ORDER BY due_date ASC, created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argPos, argPos+1,
	)
	args = append(args, filters.PerPage, offset)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list action items: %w", err)
	}
	defer rows.Close()
	out := make([]model.ActionItem, 0, filters.PerPage)
	for rows.Next() {
		item, err := scanActionItem(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *item)
	}
	return out, total, rows.Err()
}

func (s *Store) ListOverdueActionItems(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.ActionItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, meeting_id, agenda_item_id, committee_id, title, description,
		       priority, assigned_to, assignee_name, assigned_by, due_date, original_due_date,
		       extended_count, extension_reason, status, completed_at, completion_notes,
		       completion_evidence, follow_up_meeting_id, reviewed_at, tags, metadata,
		       created_by, created_at, updated_at
		FROM action_items
		WHERE tenant_id = $1
		  AND due_date < CURRENT_DATE
		  AND status IN ('pending', 'in_progress', 'overdue')
		ORDER BY due_date ASC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list overdue action items: %w", err)
	}
	defer rows.Close()
	out := make([]model.ActionItem, 0, limit)
	for rows.Next() {
		item, err := scanActionItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) MarkActionItemOverdue(ctx context.Context, q DBTX, tenantID, itemID uuid.UUID, updatedAt time.Time) error {
	_, err := q.Exec(ctx, `
		UPDATE action_items
		SET status = 'overdue',
		    updated_at = $3
		WHERE tenant_id = $1 AND id = $2 AND status IN ('pending', 'in_progress')`,
		tenantID, itemID, updatedAt,
	)
	if err != nil {
		return fmt.Errorf("mark action item overdue: %w", err)
	}
	return nil
}

func (s *Store) CountActionItemsByStatus(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := s.db.Query(ctx, `SELECT status, COUNT(*) FROM action_items WHERE tenant_id = $1 GROUP BY status`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("count action items by status: %w", err)
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan action items by status: %w", err)
		}
		out[status] = count
	}
	return out, rows.Err()
}

func (s *Store) CountActionItemsByPriority(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := s.db.Query(ctx, `SELECT priority, COUNT(*) FROM action_items WHERE tenant_id = $1 GROUP BY priority`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("count action items by priority: %w", err)
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var priority string
		var count int
		if err := rows.Scan(&priority, &count); err != nil {
			return nil, fmt.Errorf("scan action items by priority: %w", err)
		}
		out[priority] = count
	}
	return out, rows.Err()
}

func scanActionItem(scanner rowScanner) (*model.ActionItem, error) {
	var (
		item         model.ActionItem
		metadataRaw   []byte
		agendaItemID  *uuid.UUID
		extensionReason *string
		completedAt   *time.Time
		completionNotes *string
		followUpMeetingID *uuid.UUID
		reviewedAt    *time.Time
	)
	err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.MeetingID,
		&agendaItemID,
		&item.CommitteeID,
		&item.Title,
		&item.Description,
		&item.Priority,
		&item.AssignedTo,
		&item.AssigneeName,
		&item.AssignedBy,
		&item.DueDate,
		&item.OriginalDueDate,
		&item.ExtendedCount,
		&extensionReason,
		&item.Status,
		&completedAt,
		&completionNotes,
		&item.CompletionEvidence,
		&followUpMeetingID,
		&reviewedAt,
		&item.Tags,
		&metadataRaw,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan action item: %w", err)
	}
	item.AgendaItemID = agendaItemID
	item.ExtensionReason = extensionReason
	item.CompletedAt = completedAt
	item.CompletionNotes = completionNotes
	item.FollowUpMeetingID = followUpMeetingID
	item.ReviewedAt = reviewedAt
	item.Metadata, err = decodeJSONMap(metadataRaw)
	if err != nil {
		return nil, fmt.Errorf("decode action item metadata: %w", err)
	}
	return &item, nil
}
