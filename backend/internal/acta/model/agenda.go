package model

import (
	"time"

	"github.com/google/uuid"
)

type AgendaItemStatus string

const (
	AgendaItemStatusPending    AgendaItemStatus = "pending"
	AgendaItemStatusDiscussed  AgendaItemStatus = "discussed"
	AgendaItemStatusDeferred   AgendaItemStatus = "deferred"
	AgendaItemStatusApproved   AgendaItemStatus = "approved"
	AgendaItemStatusRejected   AgendaItemStatus = "rejected"
	AgendaItemStatusWithdrawn  AgendaItemStatus = "withdrawn"
	AgendaItemStatusForNoting  AgendaItemStatus = "for_noting"
)

type AgendaCategory string

const (
	AgendaCategoryRegular     AgendaCategory = "regular"
	AgendaCategorySpecial     AgendaCategory = "special"
	AgendaCategoryInformation AgendaCategory = "information"
	AgendaCategoryDecision    AgendaCategory = "decision"
	AgendaCategoryDiscussion  AgendaCategory = "discussion"
	AgendaCategoryRatification AgendaCategory = "ratification"
)

type VoteType string

const (
	VoteTypeUnanimous VoteType = "unanimous"
	VoteTypeMajority  VoteType = "majority"
	VoteTypeTwoThirds VoteType = "two_thirds"
	VoteTypeRollCall  VoteType = "roll_call"
)

type VoteResult string

const (
	VoteResultApproved VoteResult = "approved"
	VoteResultRejected VoteResult = "rejected"
	VoteResultDeferred VoteResult = "deferred"
	VoteResultTied     VoteResult = "tied"
)

type AgendaItem struct {
	ID               uuid.UUID        `json:"id"`
	TenantID         uuid.UUID        `json:"tenant_id"`
	MeetingID        uuid.UUID        `json:"meeting_id"`
	Title            string           `json:"title"`
	Description      string           `json:"description"`
	ItemNumber       *string          `json:"item_number,omitempty"`
	PresenterUserID  *uuid.UUID       `json:"presenter_user_id,omitempty"`
	PresenterName    *string          `json:"presenter_name,omitempty"`
	DurationMinutes  int              `json:"duration_minutes"`
	OrderIndex       int              `json:"order_index"`
	ParentItemID     *uuid.UUID       `json:"parent_item_id,omitempty"`
	Status           AgendaItemStatus `json:"status"`
	Notes            *string          `json:"notes,omitempty"`
	RequiresVote     bool             `json:"requires_vote"`
	VoteType         *VoteType        `json:"vote_type,omitempty"`
	VotesFor         *int             `json:"votes_for,omitempty"`
	VotesAgainst     *int             `json:"votes_against,omitempty"`
	VotesAbstained   *int             `json:"votes_abstained,omitempty"`
	VoteResult       *VoteResult      `json:"vote_result,omitempty"`
	VoteNotes        *string          `json:"vote_notes,omitempty"`
	AttachmentIDs    []uuid.UUID      `json:"attachment_ids"`
	Category         *AgendaCategory  `json:"category,omitempty"`
	Confidential     bool             `json:"confidential"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}
