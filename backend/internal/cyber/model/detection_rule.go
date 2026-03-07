package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DetectionRule models one persisted detection rule or rule template.
type DetectionRule struct {
	ID                 uuid.UUID         `json:"id" db:"id"`
	TenantID           *uuid.UUID        `json:"tenant_id,omitempty" db:"tenant_id"`
	Name               string            `json:"name" db:"name"`
	Description        string            `json:"description" db:"description"`
	RuleType           DetectionRuleType `json:"rule_type" db:"rule_type"`
	Severity           Severity          `json:"severity" db:"severity"`
	Enabled            bool              `json:"enabled" db:"enabled"`
	RuleContent        json.RawMessage   `json:"rule_content" db:"rule_content"`
	MITRETacticIDs     []string          `json:"mitre_tactic_ids" db:"mitre_tactic_ids"`
	MITRETechniqueIDs  []string          `json:"mitre_technique_ids" db:"mitre_technique_ids"`
	BaseConfidence     float64           `json:"base_confidence" db:"base_confidence"`
	FalsePositiveCount int               `json:"false_positive_count" db:"false_positive_count"`
	TruePositiveCount  int               `json:"true_positive_count" db:"true_positive_count"`
	LastTriggeredAt    *time.Time        `json:"last_triggered_at,omitempty" db:"last_triggered_at"`
	TriggerCount       int               `json:"trigger_count" db:"trigger_count"`
	Tags               []string          `json:"tags" db:"tags"`
	IsTemplate         bool              `json:"is_template" db:"is_template"`
	TemplateID         *string           `json:"template_id,omitempty" db:"template_id"`
	CreatedBy          *uuid.UUID        `json:"created_by,omitempty" db:"created_by"`
	CreatedAt          time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt          *time.Time        `json:"-" db:"deleted_at"`
}

// FPRate returns the observed false positive rate for the rule.
func (r *DetectionRule) FPRate() float64 {
	total := r.FalsePositiveCount + r.TruePositiveCount
	if total == 0 {
		return 0
	}
	return float64(r.FalsePositiveCount) / float64(total)
}

// RuleMatch captures the events and details that caused a rule to match.
type RuleMatch struct {
	RuleID       uuid.UUID              `json:"rule_id"`
	Events       []SecurityEvent        `json:"events"`
	MatchDetails map[string]interface{} `json:"match_details"`
	Timestamp    time.Time              `json:"timestamp"`
}

// RuleTemplate is returned by the templates endpoint and used to bootstrap system rules.
type RuleTemplate struct {
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	RuleType    DetectionRuleType `json:"rule_type"`
	Severity    Severity          `json:"severity"`
	Content     json.RawMessage   `json:"rule_content"`
	Tactics     []string          `json:"mitre_tactic_ids"`
	Techniques  []string          `json:"mitre_technique_ids"`
	Tags        []string          `json:"tags"`
}
