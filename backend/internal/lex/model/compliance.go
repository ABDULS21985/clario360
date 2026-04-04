package model

import (
	"time"

	"github.com/google/uuid"
)

type ComplianceRuleType string

const (
	ComplianceRuleExpiryWarning         ComplianceRuleType = "expiry_warning"
	ComplianceRuleMissingClause         ComplianceRuleType = "missing_clause"
	ComplianceRuleRiskThreshold         ComplianceRuleType = "risk_threshold"
	ComplianceRuleReviewOverdue         ComplianceRuleType = "review_overdue"
	ComplianceRuleUnsignedContract      ComplianceRuleType = "unsigned_contract"
	ComplianceRuleValueThreshold        ComplianceRuleType = "value_threshold"
	ComplianceRuleJurisdictionCheck     ComplianceRuleType = "jurisdiction_check"
	ComplianceRuleDataProtectionRequired ComplianceRuleType = "data_protection_required"
	ComplianceRuleCustom                ComplianceRuleType = "custom"
)

type ComplianceSeverity string

const (
	ComplianceSeverityCritical ComplianceSeverity = "critical"
	ComplianceSeverityHigh     ComplianceSeverity = "high"
	ComplianceSeverityMedium   ComplianceSeverity = "medium"
	ComplianceSeverityLow      ComplianceSeverity = "low"
)

type ComplianceAlertStatus string

const (
	ComplianceAlertOpen          ComplianceAlertStatus = "open"
	ComplianceAlertAcknowledged  ComplianceAlertStatus = "acknowledged"
	ComplianceAlertInvestigating ComplianceAlertStatus = "investigating"
	ComplianceAlertResolved      ComplianceAlertStatus = "resolved"
	ComplianceAlertDismissed     ComplianceAlertStatus = "dismissed"
)

type ComplianceRule struct {
	ID            uuid.UUID          `json:"id"`
	TenantID      uuid.UUID          `json:"tenant_id"`
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	RuleType      ComplianceRuleType `json:"rule_type"`
	Severity      ComplianceSeverity `json:"severity"`
	Config        map[string]any     `json:"config"`
	ContractTypes []string           `json:"contract_types"`
	Enabled       bool               `json:"enabled"`
	CreatedBy     uuid.UUID          `json:"created_by"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	DeletedAt     *time.Time         `json:"deleted_at,omitempty"`
}

type ComplianceAlert struct {
	ID              uuid.UUID             `json:"id"`
	TenantID        uuid.UUID             `json:"tenant_id"`
	RuleID          *uuid.UUID            `json:"rule_id,omitempty"`
	ContractID      *uuid.UUID            `json:"contract_id,omitempty"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	Severity        ComplianceSeverity    `json:"severity"`
	Status          ComplianceAlertStatus `json:"status"`
	ResolvedBy      *uuid.UUID            `json:"resolved_by,omitempty"`
	ResolvedAt      *time.Time            `json:"resolved_at,omitempty"`
	ResolutionNotes *string               `json:"resolution_notes,omitempty"`
	DedupKey        *string               `json:"dedup_key,omitempty"`
	Evidence        map[string]any        `json:"evidence"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

type ComplianceDashboard struct {
	RulesByType     map[string]int `json:"rules_by_type"`
	AlertsByStatus  map[string]int `json:"alerts_by_status"`
	AlertsBySeverity map[string]int `json:"alerts_by_severity"`
	OpenAlerts      int            `json:"open_alerts"`
	ResolvedAlerts  int            `json:"resolved_alerts"`
	ContractsInScope int           `json:"contracts_in_scope"`
	ComplianceScore float64        `json:"compliance_score"`
	CalculatedAt    time.Time      `json:"calculated_at"`
}

type ComplianceScore struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	Score        float64   `json:"score"`
	OpenAlerts   int       `json:"open_alerts"`
	ResolvedAlerts int     `json:"resolved_alerts"`
	RuleCoverage int       `json:"rule_coverage"`
	CalculatedAt time.Time `json:"calculated_at"`
}

type ComplianceRunResult struct {
	TenantID     uuid.UUID           `json:"tenant_id"`
	Score        float64             `json:"score"`
	AlertsCreated int                `json:"alerts_created"`
	Alerts       []ComplianceAlert   `json:"alerts"`
	CalculatedAt time.Time           `json:"calculated_at"`
}
