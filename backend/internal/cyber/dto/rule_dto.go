package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

// RuleListParams captures filters for GET /cyber/rules.
type RuleListParams struct {
	Search           *string  `form:"search"`
	Types            []string `form:"type"`
	Severities       []string `form:"severity"`
	Enabled          *bool    `form:"enabled"`
	Tag              *string  `form:"tag"`
	MITRETacticID    *string  `form:"mitre_tactic_id"`
	MITRETechniqueID *string  `form:"mitre_technique_id"`
	Page             int      `form:"page"`
	PerPage          int      `form:"per_page"`
}

// SetDefaults applies default paging.
func (p *RuleListParams) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 25
	}
}

// Validate validates rule list filters.
func (p *RuleListParams) Validate() error {
	for _, v := range p.Types {
		if !model.DetectionRuleType(v).IsValid() {
			return fmt.Errorf("invalid rule type: %q", v)
		}
	}
	for _, v := range p.Severities {
		if !model.Severity(v).IsValid() {
			return fmt.Errorf("invalid severity: %q", v)
		}
	}
	return nil
}

// RuleStatsResponse returns aggregate rule metrics for dashboard cards.
type RuleStatsResponse struct {
	Total            int                `json:"total"`
	Active           int                `json:"active"`
	ByType           []model.NamedCount `json:"by_type"`
	BySeverity       []model.NamedCount `json:"by_severity"`
	TruePositiveRate float64            `json:"true_positive_rate"`
	AlertsLast30Days int                `json:"alerts_last_30_days"`
}

// RuleListResponse returns paginated rules.
type RuleListResponse struct {
	Data []*model.DetectionRule `json:"data"`
	Meta PaginationMeta         `json:"meta"`
}

// CreateRuleRequest creates a detection rule.
type CreateRuleRequest struct {
	Name              string                  `json:"name" validate:"required,min=3,max=255"`
	Description       string                  `json:"description,omitempty" validate:"omitempty,max=4000"`
	RuleType          model.DetectionRuleType `json:"rule_type" validate:"required"`
	Severity          model.Severity          `json:"severity" validate:"required"`
	Enabled           *bool                   `json:"enabled,omitempty"`
	RuleContent       json.RawMessage         `json:"rule_content" validate:"required"`
	MITRETacticIDs    []string                `json:"mitre_tactic_ids,omitempty"`
	MITRETechniqueIDs []string                `json:"mitre_technique_ids,omitempty"`
	BaseConfidence    *float64                `json:"base_confidence,omitempty" validate:"omitempty,gte=0,lte=1"`
	Tags              []string                `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
}

// UpdateRuleRequest updates a detection rule.
type UpdateRuleRequest struct {
	Name              *string         `json:"name,omitempty" validate:"omitempty,min=3,max=255"`
	Description       *string         `json:"description,omitempty" validate:"omitempty,max=4000"`
	Severity          *model.Severity `json:"severity,omitempty"`
	Enabled           *bool           `json:"enabled,omitempty"`
	RuleContent       json.RawMessage `json:"rule_content,omitempty"`
	MITRETacticIDs    *[]string       `json:"mitre_tactic_ids,omitempty"`
	MITRETechniqueIDs *[]string       `json:"mitre_technique_ids,omitempty"`
	BaseConfidence    *float64        `json:"base_confidence,omitempty" validate:"omitempty,gte=0,lte=1"`
	Tags              *[]string       `json:"tags,omitempty"`
}

// RuleToggleRequest toggles a rule on or off.
type RuleToggleRequest struct {
	Enabled bool `json:"enabled"`
}

// RuleTestRequest dry-runs a rule against historical events.
type RuleTestRequest struct {
	DateFrom *time.Time `json:"date_from,omitempty"`
	DateTo   *time.Time `json:"date_to,omitempty"`
	Limit    int        `json:"limit,omitempty"`
}

// RuleTestResponse returns the dry-run result for a rule.
type RuleTestResponse struct {
	Matches []model.RuleMatch `json:"matches"`
	Count   int               `json:"count"`
}

// RuleFeedbackRequest records analyst TP/FP feedback.
type RuleFeedbackRequest struct {
	AlertID  uuid.UUID `json:"alert_id" validate:"required"`
	Feedback string    `json:"feedback" validate:"required,oneof=true_positive false_positive"`
}

// RuleAlertTrendPoint is a single bucket in the rule alert history series.
type RuleAlertTrendPoint struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

// RuleTopAsset captures which assets trigger a rule most often.
type RuleTopAsset struct {
	AssetID    *uuid.UUID `json:"asset_id,omitempty"`
	AssetName  string     `json:"asset_name"`
	AlertCount int        `json:"alert_count"`
}

// RulePerformanceResponse returns operational metrics for a single rule.
type RulePerformanceResponse struct {
	AlertsLast30Days     int                   `json:"alerts_last_30_days"`
	AlertsLast90Days     int                   `json:"alerts_last_90_days"`
	SeverityDistribution []model.NamedCount    `json:"severity_distribution"`
	AlertTrend           []RuleAlertTrendPoint `json:"alert_trend"`
	TopAssets            []RuleTopAsset        `json:"top_assets"`
	TruePositiveRate     float64               `json:"true_positive_rate"`
	FalsePositiveRate    float64               `json:"false_positive_rate"`
}
