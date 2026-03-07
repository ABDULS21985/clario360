package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

func (s *Store) CreateAgendaItem(ctx context.Context, q DBTX, item *model.AgendaItem) error {
	_, err := q.Exec(ctx, `
		INSERT INTO agenda_items (
			id, tenant_id, meeting_id, title, description, item_number,
			presenter_user_id, presenter_name, duration_minutes, order_index,
			parent_item_id, status, notes, requires_vote, vote_type, votes_for,
			votes_against, votes_abstained, vote_result, vote_notes, attachment_ids,
			category, confidential, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21,
			$22, $23, $24, $25
		)`,
		item.ID,
		item.TenantID,
		item.MeetingID,
		item.Title,
		item.Description,
		item.ItemNumber,
		nullableUUID(item.PresenterUserID),
		item.PresenterName,
		item.DurationMinutes,
		item.OrderIndex,
		nullableUUID(item.ParentItemID),
		item.Status,
		item.Notes,
		item.RequiresVote,
		item.VoteType,
		item.VotesFor,
		item.VotesAgainst,
		item.VotesAbstained,
		item.VoteResult,
		item.VoteNotes,
		item.AttachmentIDs,
		item.Category,
		item.Confidential,
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create agenda item: %w", err)
	}
	return nil
}

func (s *Store) UpdateAgendaItem(ctx context.Context, q DBTX, item *model.AgendaItem) error {
	tag, err := q.Exec(ctx, `
		UPDATE agenda_items
		SET title = $4,
		    description = $5,
		    item_number = $6,
		    presenter_user_id = $7,
		    presenter_name = $8,
		    duration_minutes = $9,
		    parent_item_id = $10,
		    status = $11,
		    notes = $12,
		    requires_vote = $13,
		    vote_type = $14,
		    votes_for = $15,
		    votes_against = $16,
		    votes_abstained = $17,
		    vote_result = $18,
		    vote_notes = $19,
		    attachment_ids = $20,
		    category = $21,
		    confidential = $22,
		    updated_at = $23
		WHERE tenant_id = $1 AND meeting_id = $2 AND id = $3`,
		item.TenantID,
		item.MeetingID,
		item.ID,
		item.Title,
		item.Description,
		item.ItemNumber,
		nullableUUID(item.PresenterUserID),
		item.PresenterName,
		item.DurationMinutes,
		nullableUUID(item.ParentItemID),
		item.Status,
		item.Notes,
		item.RequiresVote,
		item.VoteType,
		item.VotesFor,
		item.VotesAgainst,
		item.VotesAbstained,
		item.VoteResult,
		item.VoteNotes,
		item.AttachmentIDs,
		item.Category,
		item.Confidential,
		item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update agenda item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return notFoundError("agenda_item", item.ID)
	}
	return nil
}

func (s *Store) DeleteAgendaItem(ctx context.Context, q DBTX, tenantID, meetingID, itemID uuid.UUID) error {
	tag, err := q.Exec(ctx, `
		DELETE FROM agenda_items
		WHERE tenant_id = $1 AND meeting_id = $2 AND id = $3`,
		tenantID, meetingID, itemID,
	)
	if err != nil {
		return fmt.Errorf("delete agenda item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return notFoundError("agenda_item", itemID)
	}
	return nil
}

func (s *Store) GetAgendaItem(ctx context.Context, tenantID, meetingID, itemID uuid.UUID) (*model.AgendaItem, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, meeting_id, title, description, item_number,
		       presenter_user_id, presenter_name, duration_minutes, order_index,
		       parent_item_id, status, notes, requires_vote, vote_type, votes_for,
		       votes_against, votes_abstained, vote_result, vote_notes, attachment_ids,
		       category, confidential, created_at, updated_at
		FROM agenda_items
		WHERE tenant_id = $1 AND meeting_id = $2 AND id = $3`,
		tenantID, meetingID, itemID,
	)
	return scanAgendaItem(row)
}

func (s *Store) ListAgendaItems(ctx context.Context, tenantID, meetingID uuid.UUID) ([]model.AgendaItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, meeting_id, title, description, item_number,
		       presenter_user_id, presenter_name, duration_minutes, order_index,
		       parent_item_id, status, notes, requires_vote, vote_type, votes_for,
		       votes_against, votes_abstained, vote_result, vote_notes, attachment_ids,
		       category, confidential, created_at, updated_at
		FROM agenda_items
		WHERE tenant_id = $1 AND meeting_id = $2
		ORDER BY order_index ASC, created_at ASC`,
		tenantID, meetingID,
	)
	if err != nil {
		return nil, fmt.Errorf("list agenda items: %w", err)
	}
	defer rows.Close()
	out := make([]model.AgendaItem, 0)
	for rows.Next() {
		item, err := scanAgendaItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) ReorderAgendaItems(ctx context.Context, q DBTX, tenantID, meetingID uuid.UUID, itemIDs []uuid.UUID) error {
	for idx, itemID := range itemIDs {
		tag, err := q.Exec(ctx, `
			UPDATE agenda_items
			SET order_index = $4, updated_at = now()
			WHERE tenant_id = $1 AND meeting_id = $2 AND id = $3`,
			tenantID, meetingID, itemID, idx,
		)
		if err != nil {
			return fmt.Errorf("reorder agenda item %s: %w", itemID, err)
		}
		if tag.RowsAffected() == 0 {
			return notFoundError("agenda_item", itemID)
		}
	}
	return nil
}

func scanAgendaItem(scanner rowScanner) (*model.AgendaItem, error) {
	var item model.AgendaItem
	err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.MeetingID,
		&item.Title,
		&item.Description,
		&item.ItemNumber,
		&item.PresenterUserID,
		&item.PresenterName,
		&item.DurationMinutes,
		&item.OrderIndex,
		&item.ParentItemID,
		&item.Status,
		&item.Notes,
		&item.RequiresVote,
		&item.VoteType,
		&item.VotesFor,
		&item.VotesAgainst,
		&item.VotesAbstained,
		&item.VoteResult,
		&item.VoteNotes,
		&item.AttachmentIDs,
		&item.Category,
		&item.Confidential,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan agenda item: %w", err)
	}
	return &item, nil
}
