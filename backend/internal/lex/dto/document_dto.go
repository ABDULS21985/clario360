package dto

import (
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/lex/model"
)

type CreateLegalDocumentRequest struct {
	Title           string                        `json:"title"`
	Type            model.LegalDocumentType       `json:"type"`
	Description     string                        `json:"description"`
	Category        *string                       `json:"category,omitempty"`
	Confidentiality model.DocumentConfidentiality `json:"confidentiality"`
	ContractID      *uuid.UUID                    `json:"contract_id,omitempty"`
	Tags            []string                      `json:"tags"`
	Metadata        map[string]any                `json:"metadata"`
	Document        *FileReference                `json:"document,omitempty"`
}

type UpdateLegalDocumentRequest struct {
	Title           *string                       `json:"title,omitempty"`
	Type            *model.LegalDocumentType      `json:"type,omitempty"`
	Description     *string                       `json:"description,omitempty"`
	Category        *string                       `json:"category,omitempty"`
	Confidentiality *model.DocumentConfidentiality `json:"confidentiality,omitempty"`
	ContractID      *uuid.UUID                    `json:"contract_id,omitempty"`
	Status          *model.DocumentStatus         `json:"status,omitempty"`
	Tags            []string                      `json:"tags,omitempty"`
	Metadata        map[string]any                `json:"metadata,omitempty"`
}

type UploadDocumentVersionRequest struct {
	FileReference
}

func (r *CreateLegalDocumentRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.Tags = normalizeTags(r.Tags)
	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}
	if r.Confidentiality == "" {
		r.Confidentiality = model.DocumentConfidentialityInternal
	}
}
