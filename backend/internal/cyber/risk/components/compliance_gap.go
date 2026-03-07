package components

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/model"
)

var criticalTechniques = []string{
	"T1059", "T1078", "T1110", "T1190", "T1021", "T1486", "T1071", "T1055", "T1003", "T1047",
	"T1053", "T1036", "T1070", "T1082", "T1105", "T1027", "T1569", "T1543", "T1140", "T1068",
}

type ComplianceGap struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewComplianceGap(db *pgxpool.Pool, logger zerolog.Logger) *ComplianceGap {
	return &ComplianceGap{db: db, logger: logger.With().Str("risk_component", "compliance").Logger()}
}

func (c *ComplianceGap) Name() string {
	return "compliance_gap_risk"
}

func (c *ComplianceGap) Weight() float64 {
	return 0.15
}

func (c *ComplianceGap) Calculate(ctx context.Context, tenantID uuid.UUID) (*model.RiskComponentResult, error) {
	totalTechniques := len(mitre.AllTechniques())
	covered := make([]string, 0)
	rows, err := c.db.Query(ctx, `
		SELECT DISTINCT technique_id
		FROM (
			SELECT unnest(mitre_technique_ids) AS technique_id
			FROM detection_rules
			WHERE tenant_id = $1 AND enabled = true AND deleted_at IS NULL
		) techniques
		WHERE technique_id IS NOT NULL AND technique_id <> ''`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list covered techniques: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var techniqueID string
		if err := rows.Scan(&techniqueID); err != nil {
			return nil, err
		}
		covered = append(covered, strings.ToUpper(techniqueID))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	coveredSet := make(map[string]struct{}, len(covered))
	for _, techniqueID := range covered {
		coveredSet[techniqueID] = struct{}{}
	}
	coverageRatio := 0.0
	if totalTechniques > 0 {
		coverageRatio = float64(len(coveredSet)) / float64(totalTechniques)
	}

	missingCritical := make([]string, 0)
	for _, techniqueID := range criticalTechniques {
		if _, ok := coveredSet[techniqueID]; !ok {
			missingCritical = append(missingCritical, techniqueID)
		}
	}
	score := math.Min(((1-coverageRatio)*80)+float64(len(missingCritical)), 100)

	return &model.RiskComponentResult{
		Score: score,
		Description: fmt.Sprintf(
			"%d of %d MITRE ATT&CK techniques have enabled detection rules. %d critical techniques lack coverage.",
			len(coveredSet), totalTechniques, len(missingCritical),
		),
		Details: map[string]interface{}{
			"covered_techniques":          len(coveredSet),
			"total_techniques":            totalTechniques,
			"coverage_ratio":              coverageRatio,
			"critical_gap_count":          len(missingCritical),
			"missing_critical_techniques": missingCritical,
		},
	}, nil
}
