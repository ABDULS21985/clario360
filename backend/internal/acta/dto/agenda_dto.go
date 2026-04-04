package dto

import (
	"strings"

	"github.com/google/uuid"
)

type CreateAgendaItemRequest struct {
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	ItemNumber      *string     `json:"item_number"`
	PresenterUserID *uuid.UUID  `json:"presenter_user_id"`
	PresenterName   *string     `json:"presenter_name"`
	DurationMinutes int         `json:"duration_minutes"`
	OrderIndex      *int        `json:"order_index"`
	ParentItemID    *uuid.UUID  `json:"parent_item_id"`
	RequiresVote    bool        `json:"requires_vote"`
	VoteType        *string     `json:"vote_type"`
	AttachmentIDs   []uuid.UUID `json:"attachment_ids"`
	Category        *string     `json:"category"`
	Confidential    bool        `json:"confidential"`
}

func (r *CreateAgendaItemRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
}

type UpdateAgendaItemRequest struct {
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	ItemNumber      *string     `json:"item_number"`
	PresenterUserID *uuid.UUID  `json:"presenter_user_id"`
	PresenterName   *string     `json:"presenter_name"`
	DurationMinutes int         `json:"duration_minutes"`
	ParentItemID    *uuid.UUID  `json:"parent_item_id"`
	Status          string      `json:"status"`
	RequiresVote    bool        `json:"requires_vote"`
	VoteType        *string     `json:"vote_type"`
	AttachmentIDs   []uuid.UUID `json:"attachment_ids"`
	Category        *string     `json:"category"`
	Confidential    bool        `json:"confidential"`
}

func (r *UpdateAgendaItemRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.Status = strings.TrimSpace(r.Status)
}

type ReorderAgendaRequest struct {
	ItemIDs []uuid.UUID `json:"item_ids"`
}

type UpdateAgendaNotesRequest struct {
	Notes string `json:"notes"`
}

type RecordVoteRequest struct {
	VoteType       string `json:"vote_type"`
	VotesFor       int    `json:"votes_for"`
	VotesAgainst   int    `json:"votes_against"`
	VotesAbstained int    `json:"votes_abstained"`
	Notes          string `json:"notes"`
}
