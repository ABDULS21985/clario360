package model

import (
	"time"

	"github.com/google/uuid"
)

type ContractType string

const (
	ContractTypeServiceAgreement ContractType = "service_agreement"
	ContractTypeNDA              ContractType = "nda"
	ContractTypeEmployment       ContractType = "employment"
	ContractTypeVendor           ContractType = "vendor"
	ContractTypeLicense          ContractType = "license"
	ContractTypeLease            ContractType = "lease"
	ContractTypePartnership      ContractType = "partnership"
	ContractTypeConsulting       ContractType = "consulting"
	ContractTypeProcurement      ContractType = "procurement"
	ContractTypeSLA              ContractType = "sla"
	ContractTypeMOU              ContractType = "mou"
	ContractTypeAmendment        ContractType = "amendment"
	ContractTypeRenewal          ContractType = "renewal"
	ContractTypeOther            ContractType = "other"
)

type ContractStatus string

const (
	ContractStatusDraft            ContractStatus = "draft"
	ContractStatusInternalReview   ContractStatus = "internal_review"
	ContractStatusLegalReview      ContractStatus = "legal_review"
	ContractStatusNegotiation      ContractStatus = "negotiation"
	ContractStatusPendingSignature ContractStatus = "pending_signature"
	ContractStatusActive           ContractStatus = "active"
	ContractStatusSuspended        ContractStatus = "suspended"
	ContractStatusExpired          ContractStatus = "expired"
	ContractStatusTerminated       ContractStatus = "terminated"
	ContractStatusRenewed          ContractStatus = "renewed"
	ContractStatusCancelled        ContractStatus = "cancelled"
)

type AnalysisStatus string

const (
	AnalysisStatusPending   AnalysisStatus = "pending"
	AnalysisStatusAnalyzing AnalysisStatus = "analyzing"
	AnalysisStatusCompleted AnalysisStatus = "completed"
	AnalysisStatusFailed    AnalysisStatus = "failed"
)

type Contract struct {
	ID                 uuid.UUID      `json:"id"`
	TenantID           uuid.UUID      `json:"tenant_id"`
	Title              string         `json:"title"`
	ContractNumber     *string        `json:"contract_number,omitempty"`
	Type               ContractType   `json:"type"`
	Description        string         `json:"description"`
	PartyAName         string         `json:"party_a_name"`
	PartyAEntity       *string        `json:"party_a_entity,omitempty"`
	PartyBName         string         `json:"party_b_name"`
	PartyBEntity       *string        `json:"party_b_entity,omitempty"`
	PartyBContact      *string        `json:"party_b_contact,omitempty"`
	TotalValue         *float64       `json:"total_value,omitempty"`
	Currency           string         `json:"currency"`
	PaymentTerms       *string        `json:"payment_terms,omitempty"`
	EffectiveDate      *time.Time     `json:"effective_date,omitempty"`
	ExpiryDate         *time.Time     `json:"expiry_date,omitempty"`
	RenewalDate        *time.Time     `json:"renewal_date,omitempty"`
	AutoRenew          bool           `json:"auto_renew"`
	RenewalNoticeDays  int            `json:"renewal_notice_days"`
	SignedDate         *time.Time     `json:"signed_date,omitempty"`
	Status             ContractStatus `json:"status"`
	PreviousStatus     *ContractStatus `json:"previous_status,omitempty"`
	StatusChangedAt    *time.Time     `json:"status_changed_at,omitempty"`
	StatusChangedBy    *uuid.UUID     `json:"status_changed_by,omitempty"`
	OwnerUserID        uuid.UUID      `json:"owner_user_id"`
	OwnerName          string         `json:"owner_name"`
	LegalReviewerID    *uuid.UUID     `json:"legal_reviewer_id,omitempty"`
	LegalReviewerName  *string        `json:"legal_reviewer_name,omitempty"`
	RiskScore          *float64       `json:"risk_score,omitempty"`
	RiskLevel          RiskLevel      `json:"risk_level"`
	AnalysisStatus     AnalysisStatus `json:"analysis_status"`
	LastAnalyzedAt     *time.Time     `json:"last_analyzed_at,omitempty"`
	DocumentFileID     *uuid.UUID     `json:"document_file_id,omitempty"`
	DocumentText       string         `json:"document_text"`
	CurrentVersion     int            `json:"current_version"`
	ParentContractID   *uuid.UUID     `json:"parent_contract_id,omitempty"`
	WorkflowInstanceID *uuid.UUID     `json:"workflow_instance_id,omitempty"`
	Department         *string        `json:"department,omitempty"`
	Tags               []string       `json:"tags"`
	Metadata           map[string]any `json:"metadata"`
	CreatedBy          uuid.UUID      `json:"created_by"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          *time.Time     `json:"deleted_at,omitempty"`
}

type ContractVersion struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	ContractID    uuid.UUID  `json:"contract_id"`
	Version       int        `json:"version"`
	FileID        uuid.UUID  `json:"file_id"`
	FileName      string     `json:"file_name"`
	FileSizeBytes int64      `json:"file_size_bytes"`
	ContentHash   string     `json:"content_hash"`
	ExtractedText *string    `json:"extracted_text,omitempty"`
	ChangeSummary *string    `json:"change_summary,omitempty"`
	UploadedBy    uuid.UUID  `json:"uploaded_by"`
	UploadedAt    time.Time  `json:"uploaded_at"`
}

type ContractListFilters struct {
	Page           int
	PerPage        int
	Search         string
	Status         *ContractStatus
	Statuses       []ContractStatus
	Type           *ContractType
	OwnerUserID    *uuid.UUID
	RiskLevel      *RiskLevel
	Department     string
	Tag            string
	ExpiringInDays *int
	SortColumn     string
	SortDirection  string
}

type ContractDetail struct {
	Contract        *Contract              `json:"contract"`
	Clauses         []Clause               `json:"clauses"`
	LatestAnalysis  *ContractRiskAnalysis  `json:"latest_analysis,omitempty"`
	VersionCount    int                    `json:"version_count"`
}

type ContractSummary struct {
	ID            uuid.UUID      `json:"id"`
	Title         string         `json:"title"`
	Type          ContractType   `json:"type"`
	Status        ContractStatus `json:"status"`
	PartyBName    string         `json:"party_b_name"`
	RiskLevel     RiskLevel      `json:"risk_level"`
	RiskScore     *float64       `json:"risk_score,omitempty"`
	ExpiryDate    *time.Time     `json:"expiry_date,omitempty"`
	CurrentVersion int           `json:"current_version"`
	CreatedAt     time.Time      `json:"created_at"`
}

type ExpiringContractSummary struct {
	ID              uuid.UUID      `json:"id"`
	Title           string         `json:"title"`
	Type            ContractType   `json:"type"`
	Status          ContractStatus `json:"status"`
	PartyBName      string         `json:"party_b_name"`
	ExpiryDate      time.Time      `json:"expiry_date"`
	DaysUntilExpiry int            `json:"days_until_expiry"`
	OwnerName       string         `json:"owner_name"`
	LegalReviewerName *string      `json:"legal_reviewer_name,omitempty"`
}

type ContractRiskSummary struct {
	ID         uuid.UUID      `json:"id"`
	Title      string         `json:"title"`
	Type       ContractType   `json:"type"`
	Status     ContractStatus `json:"status"`
	RiskLevel  RiskLevel      `json:"risk_level"`
	RiskScore  float64        `json:"risk_score"`
	PartyBName string         `json:"party_b_name"`
	ExpiryDate *time.Time     `json:"expiry_date,omitempty"`
}

type TotalValueBreakdown struct {
	ByType     map[string]float64 `json:"by_type"`
	ByCurrency map[string]float64 `json:"by_currency"`
}

type MonthlyContractActivity struct {
	Month      string `json:"month"`
	Created    int    `json:"created"`
	Activated  int    `json:"activated"`
	Expired    int    `json:"expired"`
	Renewed    int    `json:"renewed"`
}

type LexDashboard struct {
	KPIs                     LexKPIs                  `json:"kpis"`
	ContractsByType          map[string]int           `json:"contracts_by_type"`
	ContractsByStatus        map[string]int           `json:"contracts_by_status"`
	ExpiringContracts        []ExpiringContractSummary `json:"expiring_contracts"`
	HighRiskContracts        []ContractRiskSummary    `json:"high_risk_contracts"`
	RecentContracts          []ContractSummary        `json:"recent_contracts"`
	ComplianceAlertsByStatus map[string]int           `json:"compliance_alerts_by_status"`
	TotalContractValue       TotalValueBreakdown      `json:"total_contract_value"`
	MonthlyActivity          []MonthlyContractActivity `json:"monthly_activity"`
	CalculatedAt             time.Time                `json:"calculated_at"`
}

type LexKPIs struct {
	ActiveContracts   int     `json:"active_contracts"`
	ExpiringIn30Days  int     `json:"expiring_in_30_days"`
	ExpiringIn7Days   int     `json:"expiring_in_7_days"`
	HighRiskContracts int     `json:"high_risk_contracts"`
	PendingReview     int     `json:"pending_review"`
	OpenAlerts        int     `json:"open_compliance_alerts"`
	TotalValue        float64 `json:"total_active_value"`
	ComplianceScore   float64 `json:"compliance_score"`
}

type ContractStats struct {
	ByStatus       map[string]int `json:"by_status"`
	ByType         map[string]int `json:"by_type"`
	ByRiskLevel    map[string]int `json:"by_risk_level"`
	Expiring30Days int            `json:"expiring_30_days"`
	Expiring7Days  int            `json:"expiring_7_days"`
}

type LegalWorkflowSummary struct {
	WorkflowInstanceID uuid.UUID       `json:"workflow_instance_id"`
	ContractID         uuid.UUID       `json:"contract_id"`
	ContractTitle      string          `json:"contract_title"`
	ContractStatus     ContractStatus  `json:"contract_status"`
	WorkflowStatus     string          `json:"workflow_status"`
	CurrentStepID      *string         `json:"current_step_id,omitempty"`
	StartedAt          time.Time       `json:"started_at"`
	AssigneeID         *uuid.UUID      `json:"assignee_id,omitempty"`
	AssigneeRole       *string         `json:"assignee_role,omitempty"`
	TaskStatus         *string         `json:"task_status,omitempty"`
}
