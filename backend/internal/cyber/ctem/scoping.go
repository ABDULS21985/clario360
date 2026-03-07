package ctem

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/database"
)

type ScopingResult struct {
	AssetCount              int            `json:"asset_count"`
	ByType                  map[string]int `json:"by_type"`
	ByCriticality           map[string]int `json:"by_criticality"`
	ByDepartment            map[string]int `json:"by_department"`
	InternetFacing          int            `json:"internet_facing"`
	InternetFacingWithVulns int            `json:"internet_facing_with_vulns"`
	TotalOpenPorts          int            `json:"total_open_ports"`
	UniqueOSVersions        int            `json:"unique_os_versions"`
	AssetsWithVulns         int            `json:"assets_with_vulns"`
	AvgVulnsPerAsset        float64        `json:"avg_vulns_per_asset"`
}

func (e *CTEMEngine) runScopingPhase(ctx context.Context, assessment *model.CTEMAssessment) error {
	result, assetIDs, err := e.runScoping(ctx, assessment)
	if err != nil {
		return err
	}
	assessment.ResolvedAssetIDs = assetIDs
	assessment.ResolvedAssetCount = len(assetIDs)
	progress := assessment.Phases["scoping"]
	payload, _ := json.Marshal(result)
	progress.Result = payload
	progress.ItemsProcessed = len(assetIDs)
	progress.ItemsTotal = len(assetIDs)
	assessment.Phases["scoping"] = progress
	return e.assessmentRepo.SaveState(ctx, assessment)
}

func (e *CTEMEngine) runScoping(ctx context.Context, assessment *model.CTEMAssessment) (*ScopingResult, []uuid.UUID, error) {
	qb := database.NewQueryBuilder("SELECT a.id FROM assets a")
	qb.Where("a.tenant_id = ?", assessment.TenantID)
	qb.Where("a.deleted_at IS NULL")
	qb.Where("a.status = ?", "active")
	if len(assessment.Scope.AssetTypes) > 0 {
		qb.WhereIn("a.type", assessment.Scope.AssetTypes)
	}
	if len(assessment.Scope.AssetTags) > 0 {
		qb.WhereArrayContainsAll("a.tags", assessment.Scope.AssetTags)
	}
	if len(assessment.Scope.AssetIDs) > 0 {
		qb.WhereIn("a.id::text", uuidStrings(assessment.Scope.AssetIDs))
	}
	if len(assessment.Scope.Departments) > 0 {
		qb.WhereIn("a.department", assessment.Scope.Departments)
	}
	if len(assessment.Scope.CIDRRanges) > 0 {
		clauses := make([]string, 0, len(assessment.Scope.CIDRRanges))
		args := make([]any, 0, len(assessment.Scope.CIDRRanges))
		for _, cidr := range assessment.Scope.CIDRRanges {
			clauses = append(clauses, "a.ip_address << ?::cidr")
			args = append(args, cidr)
		}
		qb.Where("("+strings.Join(clauses, " OR ")+")", args...)
	}
	for _, id := range assessment.Scope.ExcludeAssetIDs {
		qb.Where("a.id != ?", id)
	}

	sql, args := qb.Build()
	rows, err := e.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve scoped assets: %w", err)
	}
	defer rows.Close()

	assetIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, nil, err
		}
		assetIDs = append(assetIDs, id)
	}
	if len(assetIDs) == 0 {
		return nil, nil, fmt.Errorf("scope resolved to zero assets — refine your scope criteria")
	}

	result, err := e.computeScopingMetrics(ctx, assessment.TenantID, assetIDs)
	if err != nil {
		return nil, nil, err
	}
	return result, assetIDs, nil
}

func (e *CTEMEngine) computeScopingMetrics(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) (*ScopingResult, error) {
	result := &ScopingResult{
		ByType:        make(map[string]int),
		ByCriticality: make(map[string]int),
		ByDepartment:  make(map[string]int),
	}

	row := e.db.QueryRow(ctx, `
		WITH scoped_assets AS (
			SELECT a.*,
			       COALESCE(v.open_count, 0) AS open_vuln_count
			FROM assets a
			LEFT JOIN LATERAL (
				SELECT COUNT(*)::int AS open_count
				FROM vulnerabilities v
				WHERE v.asset_id = a.id
				  AND v.deleted_at IS NULL
				  AND v.status IN ('open','in_progress')
			) v ON true
			WHERE a.tenant_id = $1
			  AND a.id = ANY($2)
			  AND a.deleted_at IS NULL
		)
		SELECT
			COUNT(*)::int AS asset_count,
			COUNT(*) FILTER (WHERE tags && ARRAY['internet-facing','dmz','public'])::int AS internet_facing,
			COUNT(*) FILTER (
				WHERE tags && ARRAY['internet-facing','dmz','public'] AND open_vuln_count > 0
			)::int AS internet_facing_with_vulns,
			COALESCE(SUM(jsonb_array_length(COALESCE(metadata->'open_ports', '[]'::jsonb))), 0)::int AS total_open_ports,
			COUNT(DISTINCT CONCAT(COALESCE(os, ''), ' ', COALESCE(os_version, ''))) FILTER (
				WHERE COALESCE(os, '') <> '' OR COALESCE(os_version, '') <> ''
			)::int AS unique_os_versions,
			COUNT(*) FILTER (WHERE open_vuln_count > 0)::int AS assets_with_vulns,
			COALESCE(AVG(open_vuln_count), 0)::float8 AS avg_vulns_per_asset
		FROM scoped_assets`,
		tenantID, assetIDs,
	)
	if err := row.Scan(
		&result.AssetCount,
		&result.InternetFacing,
		&result.InternetFacingWithVulns,
		&result.TotalOpenPorts,
		&result.UniqueOSVersions,
		&result.AssetsWithVulns,
		&result.AvgVulnsPerAsset,
	); err != nil {
		return nil, fmt.Errorf("compute scoping metrics: %w", err)
	}

	if err := e.collectNamedCounts(ctx, tenantID, assetIDs, "type", result.ByType); err != nil {
		return nil, err
	}
	if err := e.collectNamedCounts(ctx, tenantID, assetIDs, "criticality", result.ByCriticality); err != nil {
		return nil, err
	}
	if err := e.collectNamedCounts(ctx, tenantID, assetIDs, "department", result.ByDepartment); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *CTEMEngine) collectNamedCounts(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID, column string, dest map[string]int) error {
	query := fmt.Sprintf(`
		SELECT COALESCE(%s::text, 'unassigned') AS name, COUNT(*)::int AS count
		FROM assets
		WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL
		GROUP BY %s
		ORDER BY count DESC, name ASC`, column, column)
	rows, err := e.db.Query(ctx, query, tenantID, assetIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return err
		}
		dest[name] = count
	}
	return rows.Err()
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
