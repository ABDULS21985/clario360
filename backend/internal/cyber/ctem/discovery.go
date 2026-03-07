package ctem

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func (e *CTEMEngine) runDiscovery(ctx context.Context, assessment *model.CTEMAssessment) error {
	if err := e.findingRepo.DeleteByAssessment(ctx, assessment.TenantID, assessment.ID); err != nil {
		return err
	}

	assets, err := e.assetRepo.GetMany(ctx, assessment.TenantID, assessment.ResolvedAssetIDs)
	if err != nil {
		return err
	}

	findings := make([]*model.CTEMFinding, 0)
	vulnerabilitiesByAsset := make(map[uuid.UUID][]*model.Vulnerability)
	processed := 0
	for batchStart := 0; batchStart < len(assets); batchStart += 50 {
		if err := ctx.Err(); err != nil {
			return err
		}
		batchEnd := batchStart + 50
		if batchEnd > len(assets) {
			batchEnd = len(assets)
		}
		batch := assets[batchStart:batchEnd]
		for _, asset := range batch {
			assetVulns, err := e.discoverAssetVulnerabilities(ctx, assessment.TenantID, asset)
			if err != nil {
				return err
			}
			vulnerabilitiesByAsset[asset.ID] = assetVulns
			for _, vuln := range assetVulns {
				findings = append(findings, e.newVulnerabilityFinding(assessment, asset, vuln))
			}
			findings = append(findings, e.discoverMisconfigurations(assessment, asset, assetVulns)...)
			findings = append(findings, e.discoverExposureFindings(assessment, asset)...)
			processed++
		}
		if err := e.UpdatePhaseProgress(ctx, assessment, "discovery", processed, len(assets)); err != nil {
			return err
		}
	}

	relationships, err := e.loadScopedRelationships(ctx, assessment.TenantID, assessment.ResolvedAssetIDs)
	if err != nil {
		return err
	}
	for _, path := range DiscoverAttackPaths(assets, relationships, vulnerabilitiesByAsset) {
		findings = append(findings, e.newAttackPathFinding(assessment, path))
	}

	if err := e.findingRepo.BulkInsert(ctx, findings); err != nil {
		return err
	}

	typeCounts := make(map[string]int)
	for _, finding := range findings {
		typeCounts[string(finding.Type)]++
	}
	progress := assessment.Phases["discovery"]
	progress.ItemsProcessed = len(assets)
	progress.ItemsTotal = len(assets)
	payload, _ := json.Marshal(typeCounts)
	progress.Result = payload
	assessment.Phases["discovery"] = progress
	return e.assessmentRepo.SaveState(ctx, assessment)
}

func (e *CTEMEngine) discoverAssetVulnerabilities(ctx context.Context, tenantID uuid.UUID, asset *model.Asset) ([]*model.Vulnerability, error) {
	if asset.OS != nil && asset.OSVersion != nil {
		cpe := buildCPEStringLocal(*asset.OS, *asset.OSVersion)
		if cpe != "" {
			cves, err := e.vulnRepo.FindCVEsForAsset(ctx, cpe)
			if err != nil {
				return nil, err
			}
			for _, cve := range cves {
				if err := e.vulnRepo.UpsertFromCVE(ctx, tenantID, asset.ID, cve); err != nil {
					return nil, err
				}
			}
		}
	}

	rows, err := e.db.Query(ctx, `
		SELECT id, tenant_id, asset_id, cve_id, title, description, severity, cvss_score, cvss_vector,
		       status, discovered_at, resolved_at, source, remediation, proof, metadata, created_at, updated_at, deleted_at
		FROM vulnerabilities
		WHERE tenant_id = $1 AND asset_id = $2 AND deleted_at IS NULL AND status IN ('open','in_progress')`,
		tenantID, asset.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*model.Vulnerability, 0)
	for rows.Next() {
		vuln, err := scanVulnerabilityRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, vuln)
	}
	return items, rows.Err()
}

func (e *CTEMEngine) discoverMisconfigurations(assessment *model.CTEMAssessment, asset *model.Asset, vulns []*model.Vulnerability) []*model.CTEMFinding {
	findings := make([]*model.CTEMFinding, 0)
	meta := decodeMetadata(asset.Metadata)
	openPorts := metadataIntSlice(meta["open_ports"])
	protocols := metadataStringSlice(meta["protocols"])
	internetFacing := containsAny(asset.Tags, "internet-facing", "dmz", "public")

	if internetFacing {
		for _, port := range openPorts {
			if port == 22 || port == 3389 || port == 445 || port == 3306 || port == 5432 {
				findings = append(findings, newFinding(
					assessment, model.CTEMFindingTypeMisconfiguration, model.CTEMFindingCategoryConfiguration, "high",
					fmt.Sprintf("Management port %d exposed on internet-facing asset %s", port, asset.Name),
					"An externally reachable management service increases the chance of remote compromise.",
					map[string]any{"port": port, "asset": asset.Name, "asset_id": asset.ID, "ip": asset.IPAddress},
					[]uuid.UUID{asset.ID}, asset.ID, nil, nil,
				))
			}
		}
	}

	if metadataBool(meta["default_credentials"]) || metadataBool(meta["weak_password"]) {
		findings = append(findings, newFinding(
			assessment, model.CTEMFindingTypeWeakCredential, model.CTEMFindingCategoryConfiguration, "critical",
			fmt.Sprintf("Weak or default credentials detected on %s", asset.Name),
			"Authentication weakness was identified in scan metadata for this asset.",
			map[string]any{"asset_id": asset.ID, "asset": asset.Name, "default_credentials": meta["default_credentials"], "weak_password": meta["weak_password"]},
			[]uuid.UUID{asset.ID}, asset.ID, nil, nil,
		))
	}

	if expiry, ok := parseMetadataTime(meta["certificate_expiry"]); ok && expiry.Before(time.Now().UTC()) {
		findings = append(findings, newFinding(
			assessment, model.CTEMFindingTypeExpiredCertificate, model.CTEMFindingCategoryOperational, "high",
			fmt.Sprintf("Expired certificate on %s", asset.Name),
			"The asset presents an expired TLS certificate which weakens trust and may indicate neglected maintenance.",
			map[string]any{"asset_id": asset.ID, "asset": asset.Name, "certificate_expiry": expiry},
			[]uuid.UUID{asset.ID}, asset.ID, nil, nil,
		))
	}

	for _, protocol := range protocols {
		if strings.EqualFold(protocol, "TLSv1.0") || strings.EqualFold(protocol, "SSLv3") || strings.EqualFold(protocol, "HTTP") {
			findings = append(findings, newFinding(
				assessment, model.CTEMFindingTypeInsecureProtocol, model.CTEMFindingCategoryConfiguration, "medium",
				fmt.Sprintf("Insecure protocol %s enabled on %s", protocol, asset.Name),
				"Legacy or unencrypted protocols expand exposure and should be disabled where possible.",
				map[string]any{"asset_id": asset.ID, "asset": asset.Name, "protocol": protocol},
				[]uuid.UUID{asset.ID}, asset.ID, nil, nil,
			))
		}
	}

	highPatchCandidates := make([]string, 0)
	vulnIDs := make([]uuid.UUID, 0)
	for _, vuln := range vulns {
		if vuln.CVEID != nil && (strings.EqualFold(vuln.Severity, "critical") || strings.EqualFold(vuln.Severity, "high")) {
			highPatchCandidates = append(highPatchCandidates, *vuln.CVEID)
			vulnIDs = append(vulnIDs, vuln.ID)
		}
	}
	if len(highPatchCandidates) > 0 {
		findings = append(findings, newFinding(
			assessment, model.CTEMFindingTypeMissingPatch, model.CTEMFindingCategoryOperational, highestSeverity(vulns),
			fmt.Sprintf("Critical security patches missing on %s", asset.Name),
			"Open high-severity CVEs indicate the asset is missing security updates that should be prioritized.",
			map[string]any{"asset_id": asset.ID, "asset": asset.Name, "cve_ids": highPatchCandidates, "os_version": asset.OSVersion},
			[]uuid.UUID{asset.ID}, asset.ID, vulnIDs, highPatchCandidates,
		))
	}

	return findings
}

func (e *CTEMEngine) discoverExposureFindings(assessment *model.CTEMAssessment, asset *model.Asset) []*model.CTEMFinding {
	findings := make([]*model.CTEMFinding, 0, 3)
	if asset.LastSeenAt.Before(time.Now().UTC().AddDate(0, 0, -30)) {
		findings = append(findings, newFinding(
			assessment, model.CTEMFindingTypeExposure, model.CTEMFindingCategoryOperational, "medium",
			fmt.Sprintf("Asset %s has not been scanned in 30+ days", asset.Name),
			"Asset visibility is stale, which increases uncertainty about current exposure.",
			map[string]any{"asset_id": asset.ID, "last_seen_at": asset.LastSeenAt},
			[]uuid.UUID{asset.ID}, asset.ID, nil, nil,
		))
	}
	if asset.Owner == nil || strings.TrimSpace(*asset.Owner) == "" {
		findings = append(findings, newFinding(
			assessment, model.CTEMFindingTypeExposure, model.CTEMFindingCategoryOperational, "low",
			fmt.Sprintf("Asset %s has no designated owner", asset.Name),
			"Unowned assets are less likely to be remediated or maintained on time.",
			map[string]any{"asset_id": asset.ID},
			[]uuid.UUID{asset.ID}, asset.ID, nil, nil,
		))
	}
	if asset.Department == nil || strings.TrimSpace(*asset.Department) == "" {
		findings = append(findings, newFinding(
			assessment, model.CTEMFindingTypeExposure, model.CTEMFindingCategoryOperational, "low",
			fmt.Sprintf("Asset %s is not assigned to a department", asset.Name),
			"Missing accountability metadata weakens remediation routing and reporting.",
			map[string]any{"asset_id": asset.ID},
			[]uuid.UUID{asset.ID}, asset.ID, nil, nil,
		))
	}
	return findings
}

func (e *CTEMEngine) loadScopedRelationships(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) ([]*model.AssetRelationship, error) {
	rows, err := e.db.Query(ctx, `
		SELECT ar.id, ar.tenant_id, ar.source_asset_id, ar.target_asset_id, ar.relationship_type,
		       ar.metadata, ar.created_by, ar.created_at,
		       sa.name AS source_asset_name, sa.type::text AS source_asset_type, sa.criticality::text AS source_asset_criticality,
		       ta.name AS target_asset_name, ta.type::text AS target_asset_type, ta.criticality::text AS target_asset_criticality,
		       NULL::text AS direction
		FROM asset_relationships ar
		JOIN assets sa ON sa.id = ar.source_asset_id AND sa.deleted_at IS NULL
		JOIN assets ta ON ta.id = ar.target_asset_id AND ta.deleted_at IS NULL
		WHERE ar.tenant_id = $1
		  AND ar.source_asset_id = ANY($2)
		  AND ar.target_asset_id = ANY($2)`,
		tenantID, assetIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*model.AssetRelationship, 0)
	for rows.Next() {
		rel, err := scanRelationshipRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, rel)
	}
	return items, rows.Err()
}

func (e *CTEMEngine) newVulnerabilityFinding(assessment *model.CTEMAssessment, asset *model.Asset, vuln *model.Vulnerability) *model.CTEMFinding {
	evidence := map[string]any{
		"asset_id":     asset.ID,
		"asset_name":   asset.Name,
		"cve_id":       vuln.CVEID,
		"cvss_score":   vuln.CVSSScore,
		"cvss_vector":  vuln.CVSSVector,
		"source":       vuln.Source,
		"detected_at":  vuln.DetectedAt,
		"os_version":   asset.OSVersion,
	}
	cveIDs := make([]string, 0)
	if vuln.CVEID != nil {
		cveIDs = append(cveIDs, *vuln.CVEID)
	}
	return newFinding(
		assessment,
		model.CTEMFindingTypeVulnerability,
		model.CTEMFindingCategoryTechnical,
		vuln.Severity,
		fmt.Sprintf("%s on %s", vuln.Title, asset.Name),
		vuln.Description,
		evidence,
		[]uuid.UUID{asset.ID},
		asset.ID,
		[]uuid.UUID{vuln.ID},
		cveIDs,
	)
}

func (e *CTEMEngine) newAttackPathFinding(assessment *model.CTEMAssessment, path AttackPath) *model.CTEMFinding {
	assetIDs := make([]uuid.UUID, 0, len(path.Hops))
	severity := "high"
	title := "Attack path between exposed entry point and critical target"
	if len(path.Hops) > 0 {
		title = fmt.Sprintf("Attack path from %s to %s", path.Hops[0].AssetName, path.Hops[len(path.Hops)-1].AssetName)
	}
	for _, hop := range path.Hops {
		assetIDs = append(assetIDs, hop.AssetID)
		if hop.VulnSeverity != nil && strings.EqualFold(*hop.VulnSeverity, "critical") {
			severity = "critical"
		}
	}
	finding := newFinding(
		assessment,
		model.CTEMFindingTypeAttackPath,
		model.CTEMFindingCategoryArchitectural,
		severity,
		title,
		"Relationship analysis found a traversable path from an entry asset to a high-value target through vulnerable intermediaries.",
		map[string]any{"entry_asset_id": path.EntryID, "target_asset_id": path.TargetID, "score": path.Score},
		assetIDs,
		path.TargetID,
		nil,
		nil,
	)
	pathJSON := AttackPathToJSON(path)
	finding.AttackPath = pathJSON
	length := len(path.Hops)
	finding.AttackPathLength = &length
	finding.PriorityScore = path.Score
	return finding
}

func newFinding(
	assessment *model.CTEMAssessment,
	findingType model.CTEMFindingType,
	category model.CTEMFindingCategory,
	severity string,
	title, description string,
	evidence map[string]any,
	affectedAssetIDs []uuid.UUID,
	primaryAssetID uuid.UUID,
	vulnerabilityIDs []uuid.UUID,
	cveIDs []string,
) *model.CTEMFinding {
	now := time.Now().UTC()
	evidenceJSON, _ := json.Marshal(evidence)
	metadataJSON, _ := json.Marshal(map[string]any{
		"generated_by": "ctem.discovery",
	})
	primary := primaryAssetID
	return &model.CTEMFinding{
		ID:                   uuid.New(),
		TenantID:             assessment.TenantID,
		AssessmentID:         assessment.ID,
		Type:                 findingType,
		Category:             category,
		Severity:             severity,
		Title:                title,
		Description:          description,
		Evidence:             evidenceJSON,
		AffectedAssetIDs:     affectedAssetIDs,
		AffectedAssetCount:   len(affectedAssetIDs),
		PrimaryAssetID:       &primary,
		VulnerabilityIDs:     vulnerabilityIDs,
		CVEIDs:               cveIDs,
		ValidationStatus:     model.CTEMValidationPending,
		CompensatingControls: []string{},
		Status:               model.CTEMFindingStatusOpen,
		Metadata:             metadataJSON,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func decodeMetadata(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return map[string]any{}
	}
	return value
}

func metadataIntSlice(value any) []int {
	rawSlice, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(rawSlice))
	for _, item := range rawSlice {
		switch typed := item.(type) {
		case float64:
			out = append(out, int(typed))
		case int:
			out = append(out, typed)
		}
	}
	return out
}

func metadataStringSlice(value any) []string {
	rawSlice, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(rawSlice))
	for _, item := range rawSlice {
		if typed, ok := item.(string); ok {
			out = append(out, typed)
		}
	}
	return out
}

func metadataBool(value any) bool {
	boolean, ok := value.(bool)
	return ok && boolean
}

func parseMetadataTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case string:
		for _, layout := range []string{time.RFC3339, "2006-01-02"} {
			parsed, err := time.Parse(layout, typed)
			if err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func highestSeverity(vulns []*model.Vulnerability) string {
	best := "medium"
	for _, vuln := range vulns {
		if severityWeight(vuln.Severity) > severityWeight(best) {
			best = vuln.Severity
		}
	}
	return best
}

func scanVulnerabilityRow(row interface{ Scan(dest ...any) error }) (*model.Vulnerability, error) {
	var item model.Vulnerability
	err := row.Scan(
		&item.ID, &item.TenantID, &item.AssetID, &item.CVEID, &item.Title, &item.Description, &item.Severity,
		&item.CVSSScore, &item.CVSSVector, &item.Status, &item.DetectedAt, &item.ResolvedAt, &item.Source,
		&item.Remediation, &item.Proof, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	if item.Metadata == nil {
		item.Metadata = json.RawMessage("{}")
	}
	return &item, nil
}

func scanRelationshipRow(row interface{ Scan(dest ...any) error }) (*model.AssetRelationship, error) {
	var rel model.AssetRelationship
	var relType string
	var sourceType, targetType *string
	var sourceCriticality, targetCriticality *string
	err := row.Scan(
		&rel.ID, &rel.TenantID, &rel.SourceAssetID, &rel.TargetAssetID, &relType, &rel.Metadata, &rel.CreatedBy, &rel.CreatedAt,
		&rel.SourceAssetName, &sourceType, &sourceCriticality, &rel.TargetAssetName, &targetType, &targetCriticality, &rel.Direction,
	)
	if err != nil {
		return nil, err
	}
	rel.RelationshipType = model.RelationshipType(relType)
	if sourceType != nil {
		value := model.AssetType(*sourceType)
		rel.SourceAssetType = &value
	}
	if sourceCriticality != nil {
		value := model.Criticality(*sourceCriticality)
		rel.SourceAssetCriticality = &value
	}
	if targetType != nil {
		value := model.AssetType(*targetType)
		rel.TargetAssetType = &value
	}
	if targetCriticality != nil {
		value := model.Criticality(*targetCriticality)
		rel.TargetAssetCriticality = &value
	}
	return &rel, nil
}

var versionCleanerLocal = regexp.MustCompile(`[^0-9A-Za-z._-]+`)

func buildCPEStringLocal(osName, osVersion string) string {
	normalizedOS := strings.ToLower(strings.TrimSpace(osName))
	normalizedVersion := strings.ToLower(strings.TrimSpace(osVersion))

	type vendorProduct struct {
		vendor  string
		product string
		part    string
	}

	var mapping vendorProduct
	switch {
	case strings.Contains(normalizedVersion, "ubuntu") || strings.Contains(normalizedOS, "ubuntu") || normalizedOS == "linux":
		mapping = vendorProduct{vendor: "canonical", product: "ubuntu_linux", part: "o"}
	case strings.Contains(normalizedVersion, "centos") || strings.Contains(normalizedOS, "centos"):
		mapping = vendorProduct{vendor: "centos", product: "centos", part: "o"}
	case strings.Contains(normalizedVersion, "debian") || strings.Contains(normalizedOS, "debian"):
		mapping = vendorProduct{vendor: "debian", product: "debian_linux", part: "o"}
	case strings.Contains(normalizedVersion, "red hat") || strings.Contains(normalizedVersion, "rhel") || strings.Contains(normalizedOS, "rhel"):
		mapping = vendorProduct{vendor: "redhat", product: "enterprise_linux", part: "o"}
	case strings.Contains(normalizedOS, "windows"):
		mapping = vendorProduct{vendor: "microsoft", product: "windows_server", part: "o"}
	default:
		return ""
	}

	version := extractVersionLocal(normalizedVersion)
	if version == "" {
		version = "*"
	}

	return fmt.Sprintf("cpe:2.3:%s:%s:%s:%s:*:*:*:*:*:*:*",
		mapping.part, mapping.vendor, mapping.product, version,
	)
}

func extractVersionLocal(input string) string {
	if input == "" {
		return ""
	}
	fields := strings.Fields(input)
	for _, field := range fields {
		cleaned := versionCleanerLocal.ReplaceAllString(field, "")
		if cleaned != "" && strings.ContainsAny(cleaned, "0123456789") {
			return cleaned
		}
	}
	return versionCleanerLocal.ReplaceAllString(input, "")
}
