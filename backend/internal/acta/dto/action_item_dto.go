package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateActionItemRequest struct {
	MeetingID      uuid.UUID      `json:"meeting_id"`
	AgendaItemID   *uuid.UUID     `json:"agenda_item_id"`
	CommitteeID    uuid.UUID      `json:"committee_id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Priority       string         `json:"priority"`
	AssignedTo     uuid.UUID      `json:"assigned_to"`
	AssigneeName   string         `json:"assignee_name"`
	DueDate        time.Time      `json:"due_date"`
	Tags           []string       `json:"tags"`
	Metadata       map[string]any `json:"metadata"`
}

func (r *CreateActionItemRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.Priority = strings.TrimSpace(r.Priority)
	r.AssigneeName = strings.TrimSpace(r.AssigneeName)
}

type UpdateActionItemRequest struct {
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Priority     string         `json:"priority"`
	AssignedTo   uuid.UUID      `json:"assigned_to"`
	AssigneeName string         `json:"assignee_name"`
	DueDate      time.Time      `json:"due_date"`
	Tags         []string       `json:"tags"`
	Metadata     map[string]any `json:"metadata"`
}

func (r *UpdateActionItemRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.Priority = strings.TrimSpace(r.Priority)
	r.AssigneeName = strings.TrimSpace(r.AssigneeName)
}

type UpdateActionItemStatusRequest struct {
	Status             string      `json:"status"`
	CompletionNotes    *string     `json:"completion_notes"`
	CompletionEvidence []uuid.UUID `json:"completion_evidence"`
}

type ExtendActionItemRequest struct {
	NewDueDate time.Time `json:"new_due_date"`
	Reason     string    `json:"reason"`
}
