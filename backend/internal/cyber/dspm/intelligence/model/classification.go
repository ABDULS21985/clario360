package model

import (
	"time"

	"github.com/google/uuid"
)

// ClassificationMethod identifies how a classification was determined.
type ClassificationMethod string

const (
	ClassMethodPattern     ClassificationMethod = "pattern"
	ClassMethodContent     ClassificationMethod = "content"
	ClassMethodStatistical ClassificationMethod = "statistical"
	ClassMethodManual      ClassificationMethod = "manual"
	ClassMethodCustomRule  ClassificationMethod = "custom_rule"
)

// ClassificationChangeType categorizes the nature of a classification change.
type ClassificationChangeType string

const (
	ChangeTypeInitial          ClassificationChangeType = "initial"
	ChangeTypeEscalation       ClassificationChangeType = "escalation"
	ChangeTypeDeescalation     ClassificationChangeType = "deescalation"
	ChangeTypePIIAdded         ClassificationChangeType = "pii_added"
	ChangeTypePIIRemoved       ClassificationChangeType = "pii_removed"
	ChangeTypeReclassification ClassificationChangeType = "reclassification"
)

// PatternMatch records a single pattern match result.
type PatternMatch struct {
	PatternName string  `json:"pattern_name"`
	ColumnName  string  `json:"column_name"`
	Regex       string  `json:"regex"`
	Locale      string  `json:"locale,omitempty"`
	Weight      float64 `json:"weight"`
	MatchCount  int     `json:"match_count,omitempty"`
	SampleSize  int     `json:"sample_size,omitempty"`
}

// ContentInspectionResult captures results of content-level sampling.
type ContentInspectionResult struct {
	ColumnName    string   `json:"column_name"`
	SampleSize    int      `json:"sample_size"`
	MatchCount    int      `json:"match_count"`
	MatchRate     float64  `json:"match_rate"`
	DetectedType  string   `json:"detected_type"`
	SampleMatches []string `json:"sample_matches,omitempty"`
	Confidence    float64  `json:"confidence"`
}

// StatisticalAnalysis captures column distribution analysis.
type StatisticalAnalysis struct {
	ColumnName         string             `json:"column_name"`
	CardinalityRatio   float64            `json:"cardinality_ratio"`
	AvgValueLength     float64            `json:"avg_value_length"`
	LengthStdDev       float64            `json:"length_std_dev"`
	NullRate           float64            `json:"null_rate"`
	IsFixedLength      bool               `json:"is_fixed_length"`
	CharacterClassDist map[string]float64 `json:"character_class_dist"`
	InferredType       string             `json:"inferred_type"`
	Confidence         float64            `json:"confidence"`
}

// ClassificationEvidence records evidence for a classification decision.
type ClassificationEvidence struct {
	PatternMatches     []PatternMatch            `json:"pattern_matches,omitempty"`
	ContentResults     []ContentInspectionResult `json:"content_results,omitempty"`
	StatisticalResults []StatisticalAnalysis     `json:"statistical_results,omitempty"`
	Explanation        string                    `json:"explanation"`
}

// EnhancedClassification is the output of the multi-layer classifier.
type EnhancedClassification struct {
	AssetID               uuid.UUID              `json:"asset_id"`
	AssetName             string                 `json:"asset_name"`
	Classification        string                 `json:"classification"`
	PreviousClass         string                 `json:"previous_classification,omitempty"`
	PIITypes              []string               `json:"pii_types"`
	Confidence            float64                `json:"confidence"`
	PatternConfidence     float64                `json:"pattern_confidence"`
	ContentConfidence     float64                `json:"content_confidence"`
	StatisticalConfidence float64                `json:"statistical_confidence"`
	NeedsHumanReview      bool                   `json:"needs_human_review"`
	Evidence              ClassificationEvidence `json:"evidence"`
	DetectedBy            ClassificationMethod   `json:"detected_by"`
}

// ClassificationHistory records a classification change event.
type ClassificationHistory struct {
	ID                uuid.UUID                `json:"id"`
	TenantID          uuid.UUID                `json:"tenant_id"`
	DataAssetID       uuid.UUID                `json:"data_asset_id"`
	OldClassification string                   `json:"old_classification,omitempty"`
	NewClassification string                   `json:"new_classification"`
	OldPIITypes       []string                 `json:"old_pii_types"`
	NewPIITypes       []string                 `json:"new_pii_types"`
	ChangeType        ClassificationChangeType `json:"change_type"`
	DetectedBy        string                   `json:"detected_by"`
	Confidence        float64                  `json:"confidence"`
	Evidence          ClassificationEvidence   `json:"evidence"`
	ActorID           *uuid.UUID               `json:"actor_id,omitempty"`
	ActorType         string                   `json:"actor_type"`
	CreatedAt         time.Time                `json:"created_at"`
}

// CustomClassificationRule is a tenant-defined classification rule.
type CustomClassificationRule struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Name           string    `json:"name"`
	ColumnPatterns []string  `json:"column_patterns"`
	ValuePattern   string    `json:"value_pattern,omitempty"`
	Classification string    `json:"classification"`
	PIIType        string    `json:"pii_type,omitempty"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PIIPattern is a single PII detection pattern with metadata.
type PIIPattern struct {
	Name               string   `json:"name"`
	Regex              string   `json:"regex"`
	Weight             float64  `json:"weight"`
	Locale             string   `json:"locale,omitempty"`
	Category           string   `json:"category"`
	FalsePositiveHints []string `json:"false_positive_hints,omitempty"`
}
