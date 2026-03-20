package model

import (
	"time"

	"github.com/google/uuid"
)

type LegalDocumentType string

const (
	DocumentTypePolicy            LegalDocumentType = "policy"
	DocumentTypeRegulation        LegalDocumentType = "regulation"
	DocumentTypeTemplate          LegalDocumentType = "template"
	DocumentTypeMemo              LegalDocumentType = "memo"
	DocumentTypeOpinion           LegalDocumentType = "opinion"
	DocumentTypeFiling            LegalDocumentType = "filing"
	DocumentTypeCorrespondence    LegalDocumentType = "correspondence"
	DocumentTypeResolution        LegalDocumentType = "resolution"
	DocumentTypePowerOfAttorney   LegalDocumentType = "power_of_attorney"
	DocumentTypeOther             LegalDocumentType = "other"
)

type DocumentConfidentiality string

const (
	DocumentConfidentialityPublic       DocumentConfidentiality = "public"
	DocumentConfidentialityInternal     DocumentConfidentiality = "internal"
	DocumentConfidentialityConfidential DocumentConfidentiality = "confidential"
	DocumentConfidentialityPrivileged   DocumentConfidentiality = "privileged"
)

type DocumentStatus string

const (
	DocumentStatusDraft      DocumentStatus = "draft"
	DocumentStatusActive     DocumentStatus = "active"
	DocumentStatusArchived   DocumentStatus = "archived"
	DocumentStatusSuperseded DocumentStatus = "superseded"
)

type LegalDocument struct {
	ID              uuid.UUID               `json:"id"`
	TenantID        uuid.UUID               `json:"tenant_id"`
	Title           string                  `json:"title"`
	Type            LegalDocumentType       `json:"type"`
	Description     string                  `json:"description"`
	FileID          *uuid.UUID              `json:"file_id,omitempty"`
	FileName        *string                 `json:"file_name,omitempty"`
	FileSizeBytes   *int64                  `json:"file_size_bytes,omitempty"`
	Category        *string                 `json:"category,omitempty"`
	Confidentiality DocumentConfidentiality `json:"confidentiality"`
	ContractID      *uuid.UUID              `json:"contract_id,omitempty"`
	CurrentVersion  int                     `json:"current_version"`
	Status          DocumentStatus          `json:"status"`
	Tags            []string                `json:"tags"`
	Metadata        map[string]any          `json:"metadata"`
	CreatedBy       uuid.UUID               `json:"created_by"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	DeletedAt       *time.Time              `json:"deleted_at,omitempty"`
}

type DocumentVersion struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	DocumentID    uuid.UUID  `json:"document_id"`
	Version       int        `json:"version"`
	FileID        uuid.UUID  `json:"file_id"`
	FileName      string     `json:"file_name"`
	FileSizeBytes int64      `json:"file_size_bytes"`
	ContentHash   string     `json:"content_hash"`
	ChangeSummary *string    `json:"change_summary,omitempty"`
	UploadedBy    uuid.UUID  `json:"uploaded_by"`
	UploadedAt    time.Time  `json:"uploaded_at"`
}
