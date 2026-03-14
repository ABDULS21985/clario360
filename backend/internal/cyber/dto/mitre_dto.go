package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

// MITRETacticDTO is returned by the tactics endpoint.
type MITRETacticDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"short_name"`
	Description string `json:"description"`
}

// MITRETechniqueDTO is returned by the techniques endpoint.
type MITRETechniqueDTO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TacticIDs   []string `json:"tactic_ids"`
	Platforms   []string `json:"platforms"`
	DataSources []string `json:"data_sources"`
}

// MITRECoverageDTO returns rule coverage for a single technique.
type MITRECoverageDTO struct {
	TechniqueID       string     `json:"technique_id"`
	TechniqueName     string     `json:"technique_name"`
	TacticIDs         []string   `json:"tactic_ids"`
	HasDetection      bool       `json:"has_detection"`
	RuleCount         int        `json:"rule_count"`
	RuleNames         []string   `json:"rule_names"`
	CoverageState     string     `json:"coverage_state"`
	HighFPRuleCount   int        `json:"high_fp_rule_count"`
	AlertCount        int        `json:"alert_count"`
	ThreatCount       int        `json:"threat_count"`
	ActiveThreatCount int        `json:"active_threat_count"`
	LastAlertAt       *time.Time `json:"last_alert_at,omitempty"`
	Description       string     `json:"description"`
	Platforms         []string   `json:"platforms"`
}

// MITRETacticCoverageDTO is a tactic with its coverage count.
type MITRETacticCoverageDTO struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ShortName      string `json:"short_name,omitempty"`
	TechniqueCount int    `json:"technique_count"`
	CoveredCount   int    `json:"covered_count"`
}

// MITRECoverageResponseDTO is the full aggregated coverage response.
type MITRECoverageResponseDTO struct {
	Tactics           []MITRETacticCoverageDTO `json:"tactics"`
	Techniques        []MITRECoverageDTO       `json:"techniques"`
	TotalTechniques   int                      `json:"total_techniques"`
	CoveredTechniques int                      `json:"covered_techniques"`
	CoveragePercent   float64                  `json:"coverage_percent"`
	ActiveTechniques  int                      `json:"active_techniques"`
	PassiveTechniques int                      `json:"passive_techniques"`
	CriticalGapCount  int                      `json:"critical_gap_count"`
}

// MITRERuleReferenceDTO is a compact rule projection used in technique detail views.
type MITRERuleReferenceDTO struct {
	ID                 uuid.UUID               `json:"id"`
	Name               string                  `json:"name"`
	RuleType           model.DetectionRuleType `json:"rule_type"`
	Severity           model.Severity          `json:"severity"`
	Enabled            bool                    `json:"enabled"`
	TriggerCount       int                     `json:"trigger_count"`
	TruePositiveCount  int                     `json:"true_positive_count"`
	FalsePositiveCount int                     `json:"false_positive_count"`
	LastTriggeredAt    *time.Time              `json:"last_triggered_at,omitempty"`
}

// MITREThreatReferenceDTO is a compact threat projection used in technique detail views.
type MITREThreatReferenceDTO struct {
	ID         uuid.UUID          `json:"id"`
	Name       string             `json:"name"`
	Type       model.ThreatType   `json:"type"`
	Severity   model.Severity     `json:"severity"`
	Status     model.ThreatStatus `json:"status"`
	LastSeenAt time.Time          `json:"last_seen_at"`
}

// MITREAlertReferenceDTO is a compact alert projection used in technique detail views.
type MITREAlertReferenceDTO struct {
	ID              uuid.UUID         `json:"id"`
	Title           string            `json:"title"`
	Severity        model.Severity    `json:"severity"`
	Status          model.AlertStatus `json:"status"`
	ConfidenceScore float64           `json:"confidence_score"`
	AssetName       *string           `json:"asset_name,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
}

// MITRETechniqueDetailDTO enriches framework metadata with tenant-specific context.
type MITRETechniqueDetailDTO struct {
	ID                string                    `json:"id"`
	Name              string                    `json:"name"`
	Description       string                    `json:"description"`
	TacticIDs         []string                  `json:"tactic_ids"`
	Platforms         []string                  `json:"platforms"`
	DataSources       []string                  `json:"data_sources"`
	CoverageState     string                    `json:"coverage_state"`
	RuleCount         int                       `json:"rule_count"`
	AlertCount        int                       `json:"alert_count"`
	ThreatCount       int                       `json:"threat_count"`
	ActiveThreatCount int                       `json:"active_threat_count"`
	HighFPRuleCount   int                       `json:"high_fp_rule_count"`
	LastAlertAt       *time.Time                `json:"last_alert_at,omitempty"`
	LinkedRules       []MITRERuleReferenceDTO   `json:"linked_rules"`
	LinkedThreats     []MITREThreatReferenceDTO `json:"linked_threats"`
	RecentAlerts      []MITREAlertReferenceDTO  `json:"recent_alerts"`
}
