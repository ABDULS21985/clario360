package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/lex/model"
)

type FileReference struct {
	FileID        uuid.UUID `json:"file_id"`
	FileName      string    `json:"file_name"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	ContentHash   string    `json:"content_hash"`
	ExtractedText string    `json:"extracted_text"`
	ChangeSummary string    `json:"change_summary"`
}

type CreateContractRequest struct {
	Title             string                 `json:"title"`
	ContractNumber    *string                `json:"contract_number,omitempty"`
	Type              model.ContractType     `json:"type"`
	Description       string                 `json:"description"`
	PartyAName        string                 `json:"party_a_name"`
	PartyAEntity      *string                `json:"party_a_entity,omitempty"`
	PartyBName        string                 `json:"party_b_name"`
	PartyBEntity      *string                `json:"party_b_entity,omitempty"`
	PartyBContact     *string                `json:"party_b_contact,omitempty"`
	TotalValue        *float64               `json:"total_value,omitempty"`
	Currency          string                 `json:"currency"`
	PaymentTerms      *string                `json:"payment_terms,omitempty"`
	EffectiveDate     *time.Time             `json:"effective_date,omitempty"`
	ExpiryDate        *time.Time             `json:"expiry_date,omitempty"`
	RenewalDate       *time.Time             `json:"renewal_date,omitempty"`
	AutoRenew         bool                   `json:"auto_renew"`
	RenewalNoticeDays int                    `json:"renewal_notice_days"`
	OwnerUserID       uuid.UUID              `json:"owner_user_id"`
	OwnerName         string                 `json:"owner_name"`
	LegalReviewerID   *uuid.UUID             `json:"legal_reviewer_id,omitempty"`
	LegalReviewerName *string                `json:"legal_reviewer_name,omitempty"`
	Department        *string                `json:"department,omitempty"`
	Tags              []string               `json:"tags"`
	Metadata          map[string]any         `json:"metadata"`
	Document          *FileReference         `json:"document,omitempty"`
}

type UpdateContractRequest struct {
	Title             *string                `json:"title,omitempty"`
	ContractNumber    *string                `json:"contract_number,omitempty"`
	Type              *model.ContractType    `json:"type,omitempty"`
	Description       *string                `json:"description,omitempty"`
	PartyAName        *string                `json:"party_a_name,omitempty"`
	PartyAEntity      *string                `json:"party_a_entity,omitempty"`
	PartyBName        *string                `json:"party_b_name,omitempty"`
	PartyBEntity      *string                `json:"party_b_entity,omitempty"`
	PartyBContact     *string                `json:"party_b_contact,omitempty"`
	TotalValue        *float64               `json:"total_value,omitempty"`
	Currency          *string                `json:"currency,omitempty"`
	PaymentTerms      *string                `json:"payment_terms,omitempty"`
	EffectiveDate     *time.Time             `json:"effective_date,omitempty"`
	ExpiryDate        *time.Time             `json:"expiry_date,omitempty"`
	RenewalDate       *time.Time             `json:"renewal_date,omitempty"`
	AutoRenew         *bool                  `json:"auto_renew,omitempty"`
	RenewalNoticeDays *int                   `json:"renewal_notice_days,omitempty"`
	SignedDate        *time.Time             `json:"signed_date,omitempty"`
	OwnerUserID       *uuid.UUID             `json:"owner_user_id,omitempty"`
	OwnerName         *string                `json:"owner_name,omitempty"`
	LegalReviewerID   *uuid.UUID             `json:"legal_reviewer_id,omitempty"`
	LegalReviewerName *string                `json:"legal_reviewer_name,omitempty"`
	Department        *string                `json:"department,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	Metadata          map[string]any         `json:"metadata,omitempty"`
}

type UploadContractDocumentRequest struct {
	FileReference
}

type UpdateContractStatusRequest struct {
	Status    model.ContractStatus `json:"status"`
	ChangedBy *uuid.UUID           `json:"changed_by,omitempty"`
}

type RenewContractRequest struct {
	NewEffectiveDate *time.Time `json:"new_effective_date,omitempty"`
	NewExpiryDate    time.Time  `json:"new_expiry_date"`
	NewValue         *float64   `json:"new_value,omitempty"`
	ChangeSummary    string     `json:"change_summary"`
}

type ReviewContractRequest struct {
	ApproverUserID   *uuid.UUID `json:"approver_user_id,omitempty"`
	ApproverRole     *string    `json:"approver_role,omitempty"`
	SLAHours         int        `json:"sla_hours"`
	Description      string     `json:"description"`
}

func (r *CreateContractRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.PartyAName = strings.TrimSpace(r.PartyAName)
	r.PartyBName = strings.TrimSpace(r.PartyBName)
	r.Currency = normalizeCurrency(r.Currency)
	r.OwnerName = strings.TrimSpace(r.OwnerName)
	r.Tags = normalizeTags(r.Tags)
	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}
	if r.RenewalNoticeDays == 0 {
		r.RenewalNoticeDays = 30
	}
}

func normalizeCurrency(raw string) string {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	if raw == "" {
		return "SAR"
	}
	return raw
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}
