package dashboard

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/model"
)

type MITREHeatmapCalculator struct {
	db *pgxpool.Pool
}

func NewMITREHeatmapCalculator(db *pgxpool.Pool) *MITREHeatmapCalculator {
	return &MITREHeatmapCalculator{db: db}
}

func (c *MITREHeatmapCalculator) Heatmap(ctx context.Context, tenantID uuid.UUID, days int) (model.MITREHeatmapData, error) {
	if days <= 0 {
		days = 90
	}
	since := time.Now().UTC().AddDate(0, 0, -days)
	rows, err := c.db.Query(ctx, `
		SELECT
			a.mitre_tactic_id,
			a.mitre_technique_id,
			a.mitre_technique_name,
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE a.severity = 'critical')::int,
			MAX(a.created_at)
		FROM alerts a
		WHERE a.tenant_id = $1
		  AND a.mitre_technique_id IS NOT NULL
		  AND a.created_at > $2
		  AND a.deleted_at IS NULL
		GROUP BY a.mitre_tactic_id, a.mitre_technique_id, a.mitre_technique_name
		ORDER BY COUNT(*) DESC`,
		tenantID, since,
	)
	if err != nil {
		return model.MITREHeatmapData{}, fmt.Errorf("mitre heatmap query: %w", err)
	}
	defer rows.Close()

	cellMap := map[string]model.MITREHeatmapCell{}
	maxCount := 0
	for rows.Next() {
		var (
			tacticID, techniqueID, techniqueName string
			alertCount, criticalCount            int
			lastSeen                             time.Time
		)
		if err := rows.Scan(&tacticID, &techniqueID, &techniqueName, &alertCount, &criticalCount, &lastSeen); err != nil {
			return model.MITREHeatmapData{}, err
		}
		tacticName := tacticID
		if tactic, ok := mitre.TacticByID(tacticID); ok {
			tacticName = tactic.Name
		}
		cellMap[strings.ToUpper(techniqueID)] = model.MITREHeatmapCell{
			TacticID:      tacticID,
			TacticName:    tacticName,
			TechniqueID:   techniqueID,
			TechniqueName: firstText(techniqueName, techniqueID),
			AlertCount:    alertCount,
			CriticalCount: criticalCount,
			LastSeen:      lastSeen,
		}
		if alertCount > maxCount {
			maxCount = alertCount
		}
	}
	if err := rows.Err(); err != nil {
		return model.MITREHeatmapData{}, err
	}

	coveredRows, err := c.db.Query(ctx, `
		SELECT DISTINCT unnest(mitre_technique_ids) AS technique_id
		FROM detection_rules
		WHERE tenant_id = $1 AND enabled = true AND deleted_at IS NULL`,
		tenantID,
	)
	if err != nil {
		return model.MITREHeatmapData{}, fmt.Errorf("mitre coverage query: %w", err)
	}
	defer coveredRows.Close()
	for coveredRows.Next() {
		var techniqueID string
		if err := coveredRows.Scan(&techniqueID); err != nil {
			return model.MITREHeatmapData{}, err
		}
		key := strings.ToUpper(techniqueID)
		cell, ok := cellMap[key]
		if !ok {
			cell = coverageOnlyCell(techniqueID)
		}
		cell.HasDetection = true
		cellMap[key] = cell
	}
	if err := coveredRows.Err(); err != nil {
		return model.MITREHeatmapData{}, err
	}

	cells := make([]model.MITREHeatmapCell, 0, len(cellMap))
	for _, cell := range cellMap {
		cells = append(cells, cell)
	}
	sort.SliceStable(cells, func(i, j int) bool {
		if cells[i].AlertCount == cells[j].AlertCount {
			return cells[i].TechniqueID < cells[j].TechniqueID
		}
		return cells[i].AlertCount > cells[j].AlertCount
	})
	return model.MITREHeatmapData{Cells: cells, MaxCount: maxCount}, nil
}

func coverageOnlyCell(techniqueID string) model.MITREHeatmapCell {
	techniqueName := techniqueID
	tacticID := ""
	tacticName := ""
	if technique, ok := mitre.TechniqueByID(techniqueID); ok {
		techniqueName = technique.Name
		if len(technique.TacticIDs) > 0 {
			tacticID = technique.TacticIDs[0]
			if tactic, ok := mitre.TacticByID(tacticID); ok {
				tacticName = tactic.Name
			}
		}
	}
	return model.MITREHeatmapCell{
		TacticID:      tacticID,
		TacticName:    tacticName,
		TechniqueID:   techniqueID,
		TechniqueName: techniqueName,
	}
}

func firstText(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
