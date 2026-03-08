package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/cyber/classifier"
	cyberconfig "github.com/clario360/platform/internal/cyber/config"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/metrics"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/scanner"
	"github.com/clario360/platform/internal/events"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

// AssetService contains the business logic for asset management.
type AssetService struct {
	assetRepo        *repository.AssetRepository
	vulnRepo         *repository.VulnerabilityRepository
	relRepo          *repository.RelationshipRepository
	scanRepo         *repository.ScanRepository
	scanRegistry     *scanner.Registry
	classifier       *classifier.AssetClassifier
	enrichSvc        *EnrichmentService
	producer         *events.Producer
	metrics          *metrics.Metrics
	cfg              *cyberconfig.Config
	db               *pgxpool.Pool
	logger           zerolog.Logger
	runningScans     map[uuid.UUID]context.CancelFunc
	scanMu           sync.Mutex
	predictionLogger *aigovmiddleware.PredictionLogger
}

// NewAssetService creates a new AssetService.
func NewAssetService(
	assetRepo *repository.AssetRepository,
	vulnRepo *repository.VulnerabilityRepository,
	relRepo *repository.RelationshipRepository,
	scanRepo *repository.ScanRepository,
	scanRegistry *scanner.Registry,
	cls *classifier.AssetClassifier,
	enrichSvc *EnrichmentService,
	producer *events.Producer,
	m *metrics.Metrics,
	cfg *cyberconfig.Config,
	db *pgxpool.Pool,
	logger zerolog.Logger,
) *AssetService {
	return &AssetService{
		assetRepo:    assetRepo,
		vulnRepo:     vulnRepo,
		relRepo:      relRepo,
		scanRepo:     scanRepo,
		scanRegistry: scanRegistry,
		classifier:   cls,
		enrichSvc:    enrichSvc,
		producer:     producer,
		metrics:      m,
		cfg:          cfg,
		db:           db,
		logger:       logger,
		runningScans: make(map[uuid.UUID]context.CancelFunc),
	}
}

// CreateAsset creates a single asset, optionally classifying and enriching it.
func (s *AssetService) CreateAsset(ctx context.Context, tenantID, userID uuid.UUID, req *dto.CreateAssetRequest) (*model.Asset, error) {
	asset, err := s.assetRepo.Create(ctx, tenantID, userID, req)
	if err != nil {
		return nil, err
	}

	s.metrics.AssetsCreated.WithLabelValues(tenantID.String(), string(req.Type), "manual").Inc()

	if s.cfg.ClassifyOnCreate {
		crit, ruleName, _ := s.classifier.Classify(asset)
		s.recordAssetClassificationPrediction(ctx, tenantID, asset, crit, ruleName)
		s.metrics.ClassificationsTotal.WithLabelValues(tenantID.String(), ruleName).Inc()
		if crit != asset.Criticality {
			s.metrics.ClassificationChanged.WithLabelValues(tenantID.String(), string(asset.Criticality), string(crit)).Inc()
			_ = s.assetRepo.BulkUpdateCriticality(ctx, tenantID, map[uuid.UUID]model.Criticality{asset.ID: crit})
			_ = s.publishEvent(ctx, "cyber.asset.classified", tenantID.String(), map[string]any{
				"id":              asset.ID.String(),
				"old_criticality": string(asset.Criticality),
				"new_criticality": string(crit),
				"rule_name":       ruleName,
			})
			asset.Criticality = crit
		}
	}

	// Async enrichment
	go func() {
		bgCtx := context.Background()
		_ = s.enrichSvc.EnrichAsset(bgCtx, tenantID, asset.ID)
	}()

	// Publish event
	_ = s.publishEvent(ctx, "cyber.asset.created", tenantID.String(), map[string]any{
		"id":               asset.ID.String(),
		"name":             asset.Name,
		"type":             string(asset.Type),
		"criticality":      string(asset.Criticality),
		"discovery_source": asset.DiscoverySource,
	})

	return asset, nil
}

// GetAsset returns a single asset by ID, enforcing tenant isolation.
func (s *AssetService) GetAsset(ctx context.Context, tenantID, assetID uuid.UUID) (*model.Asset, error) {
	return s.assetRepo.GetByID(ctx, tenantID, assetID)
}

// ListAssets returns a paginated, filtered asset list.
func (s *AssetService) ListAssets(ctx context.Context, tenantID uuid.UUID, params *dto.AssetListParams) (*dto.AssetListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid filter params: %w", err)
	}

	assets, total, err := s.assetRepo.List(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}

	totalPages := (total + params.PerPage - 1) / params.PerPage
	if totalPages < 1 {
		totalPages = 1
	}

	return &dto.AssetListResponse{
		Data:       assets,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

// UpdateAsset applies a partial update.
func (s *AssetService) UpdateAsset(ctx context.Context, tenantID, assetID, userID uuid.UUID, req *dto.UpdateAssetRequest) (*model.Asset, error) {
	before, err := s.assetRepo.GetByID(ctx, tenantID, assetID)
	if err != nil {
		return nil, err
	}
	updated, err := s.assetRepo.Update(ctx, tenantID, assetID, req)
	if err != nil {
		return nil, err
	}
	_ = s.publishEvent(ctx, "cyber.asset.updated", tenantID.String(), buildAssetUpdatedEvent(before, updated))
	return updated, nil
}

// DeleteAsset soft-deletes an asset.
func (s *AssetService) DeleteAsset(ctx context.Context, tenantID, assetID uuid.UUID) error {
	asset, err := s.assetRepo.GetByID(ctx, tenantID, assetID)
	if err != nil {
		return err
	}
	if err := s.assetRepo.SoftDelete(ctx, tenantID, assetID); err != nil {
		return err
	}
	s.metrics.AssetsDeleted.WithLabelValues(tenantID.String(), "unknown").Inc()
	_ = s.publishEvent(ctx, "cyber.asset.deleted", tenantID.String(), map[string]any{"id": assetID.String(), "name": asset.Name})
	return nil
}

// PatchTags updates tags on an asset.
func (s *AssetService) PatchTags(ctx context.Context, tenantID, assetID uuid.UUID, req *dto.TagPatchRequest) (*model.Asset, error) {
	asset, err := s.assetRepo.PatchTags(ctx, tenantID, assetID, req)
	if err != nil {
		return nil, err
	}
	_ = s.publishEvent(ctx, "cyber.asset.tags_updated", tenantID.String(), map[string]any{
		"id":           asset.ID.String(),
		"added_tags":   req.Add,
		"removed_tags": req.Remove,
	})
	return asset, nil
}

// BulkCreate creates up to 1000 assets from a JSON slice or CSV.
func (s *AssetService) BulkCreate(ctx context.Context, tenantID, userID uuid.UUID, reqs []dto.CreateAssetRequest) (*dto.BulkCreateResult, error) {
	if len(reqs) > 1000 {
		return nil, fmt.Errorf("bulk create limit is 1000 assets, got %d", len(reqs))
	}

	// Validate all rows first; collect all errors before writing anything
	rowErrors := make(map[int]map[string][]string)
	for i, req := range reqs {
		fieldErrs := pkgvalidator.Validate(req)
		if fieldErrs != nil {
			errs := make(map[string][]string, len(fieldErrs))
			for k, v := range fieldErrs {
				errs[k] = []string{v}
			}
			rowErrors[i] = errs
		}
	}
	if len(rowErrors) > 0 {
		return nil, &BulkValidationError{
			Code:    "BULK_VALIDATION_ERROR",
			Message: fmt.Sprintf("validation failed for %d of %d rows", len(rowErrors), len(reqs)),
			Rows:    rowErrors,
		}
	}

	// Convert DTOs to model.Asset for CopyFrom
	assets := make([]model.Asset, len(reqs))
	for i, req := range reqs {
		assets[i] = model.Asset{
			TenantID:    tenantID,
			Name:        req.Name,
			Type:        req.Type,
			IPAddress:   req.IPAddress,
			Hostname:    req.Hostname,
			MACAddress:  req.MACAddress,
			OS:          req.OS,
			OSVersion:   req.OSVersion,
			Owner:       req.Owner,
			Department:  req.Department,
			Location:    req.Location,
			Criticality: req.Criticality,
			Status:      model.AssetStatusActive,
			Metadata:    req.Metadata,
			Tags:        req.Tags,
			CreatedBy:   &userID,
		}
		if assets[i].Metadata == nil {
			assets[i].Metadata = json.RawMessage("{}")
		}
		if assets[i].Tags == nil {
			assets[i].Tags = []string{}
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	ids, err := s.assetRepo.BulkInsert(ctx, tx, tenantID, userID, assets)
	if err != nil {
		return nil, fmt.Errorf("bulk insert: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	s.metrics.AssetsBulkImported.WithLabelValues(tenantID.String()).Add(float64(len(ids)))

	// Async classification + enrichment
	go func() {
		bgCtx := context.Background()
		fetchedAssets, err := s.assetRepo.GetMany(bgCtx, tenantID, ids)
		if err != nil {
			s.logger.Warn().Err(err).Msg("bulk create: failed to fetch assets for classification")
			return
		}
		if s.cfg.ClassifyOnCreate {
			results := s.classifier.ClassifyBatch(fetchedAssets)
			assetsByID := make(map[uuid.UUID]*model.Asset, len(fetchedAssets))
			for _, asset := range fetchedAssets {
				assetsByID[asset.ID] = asset
			}
			updates := make(map[uuid.UUID]model.Criticality)
			for _, r := range results {
				if asset := assetsByID[r.AssetID]; asset != nil {
					s.recordAssetClassificationPrediction(bgCtx, tenantID, asset, r.Criticality, r.RuleName)
				}
				if r.Changed {
					updates[r.AssetID] = r.Criticality
				}
			}
			if len(updates) > 0 {
				_ = s.assetRepo.BulkUpdateCriticality(bgCtx, tenantID, updates)
			}
		}
		s.enrichSvc.EnrichBatch(bgCtx, tenantID, ids)
	}()

	// Publish bulk event
	_ = s.publishEvent(ctx, "cyber.asset.bulk_created", tenantID.String(), map[string]any{
		"count":            len(ids),
		"ids":              uuidsToStrings(ids),
		"discovery_source": "import",
	})

	return &dto.BulkCreateResult{Count: len(ids), IDs: ids}, nil
}

// BulkCreateFromCSV parses a CSV reader and delegates to BulkCreate.
func (s *AssetService) BulkCreateFromCSV(ctx context.Context, tenantID, userID uuid.UUID, r io.Reader) (*dto.BulkCreateResult, error) {
	csvReader := csv.NewReader(r)
	csvReader.TrimLeadingSpace = true

	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	required := []string{"name", "type", "criticality"}
	for _, col := range required {
		if _, ok := colIdx[col]; !ok {
			return nil, fmt.Errorf("CSV missing required column: %s", col)
		}
	}

	var reqs []dto.CreateAssetRequest
	rowNum := 1
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		rowNum++

		get := func(col string) *string {
			idx, ok := colIdx[col]
			if !ok || idx >= len(record) || strings.TrimSpace(record[idx]) == "" {
				return nil
			}
			v := strings.TrimSpace(record[idx])
			return &v
		}
		getString := func(col string) string {
			if s := get(col); s != nil {
				return *s
			}
			return ""
		}

		tagsStr := getString("tags")
		var tags []string
		if tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		req := dto.CreateAssetRequest{
			Name:        getString("name"),
			Type:        model.AssetType(getString("type")),
			Criticality: model.Criticality(getString("criticality")),
			IPAddress:   get("ip_address"),
			Hostname:    get("hostname"),
			MACAddress:  get("mac_address"),
			OS:          get("os"),
			OSVersion:   get("os_version"),
			Owner:       get("owner"),
			Department:  get("department"),
			Location:    get("location"),
			Tags:        tags,
		}
		reqs = append(reqs, req)
		if len(reqs) > 1000 {
			return nil, fmt.Errorf("CSV exceeds maximum 1000 rows")
		}
	}

	return s.BulkCreate(ctx, tenantID, userID, reqs)
}

// BulkUpdateTags adds/removes tags on multiple assets.
func (s *AssetService) BulkUpdateTags(ctx context.Context, tenantID uuid.UUID, req *dto.BulkTagRequest) error {
	ids := make([]uuid.UUID, len(req.AssetIDs))
	for i, idStr := range req.AssetIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return fmt.Errorf("invalid asset_id %q: %w", idStr, err)
		}
		ids[i] = id
	}
	return s.assetRepo.BulkUpdateTags(ctx, tenantID, ids, req.Add, req.Remove)
}

// BulkDelete soft-deletes multiple assets.
func (s *AssetService) BulkDelete(ctx context.Context, tenantID uuid.UUID, req *dto.BulkDeleteRequest) error {
	ids := make([]uuid.UUID, len(req.AssetIDs))
	for i, idStr := range req.AssetIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return fmt.Errorf("invalid asset_id %q: %w", idStr, err)
		}
		ids[i] = id
	}
	return s.assetRepo.BulkSoftDelete(ctx, tenantID, ids)
}

// TriggerScan starts an async discovery scan.
func (s *AssetService) TriggerScan(ctx context.Context, tenantID, userID uuid.UUID, req *dto.ScanTriggerRequest) (*model.ScanHistory, error) {
	scanType := model.ScanType(req.ScanType)
	sc := s.scanRegistry.Get(scanType)
	if sc == nil {
		return nil, fmt.Errorf("scan type %q is not supported", req.ScanType)
	}

	cfg := &model.ScanConfig{
		Targets: req.Targets,
		Ports:   req.Ports,
		Options: req.Options,
	}

	scan, err := s.scanRepo.Create(ctx, tenantID, userID, scanType, cfg)
	if err != nil {
		return nil, err
	}

	scanCtx, cancel := context.WithTimeout(scanner.WithTenantID(context.Background(), tenantID), scanTimeout(req.Options))
	s.scanMu.Lock()
	s.runningScans[scan.ID] = cancel
	s.scanMu.Unlock()

	_ = s.publishEvent(ctx, "cyber.asset.scan_started", tenantID.String(), map[string]any{
		"scan_id":      scan.ID.String(),
		"scan_type":    req.ScanType,
		"target_count": len(req.Targets),
		"targets":      req.Targets,
	})

	// Run scan asynchronously
	go func() {
		defer s.unregisterRunningScan(scan.ID)
		defer cancel()

		result, runErr := sc.Scan(scanCtx, cfg)
		if runErr != nil {
			result = &model.ScanResult{
				ScanID: scan.ID,
				Status: model.ScanStatusFailed,
				Errors: []string{runErr.Error()},
			}
			if errors.Is(runErr, context.Canceled) {
				result.Status = model.ScanStatusCancelled
			}
		}
		result.ScanID = scan.ID
		completeCtx, completeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer completeCancel()
		if err := s.scanRepo.Complete(completeCtx, tenantID, scan.ID, result); err != nil {
			s.logger.Error().Err(err).Str("scan_id", scan.ID.String()).Msg("failed to record scan completion")
		}
		status := string(result.Status)
		s.metrics.ScansTotal.WithLabelValues(tenantID.String(), req.ScanType, status).Inc()
		s.metrics.ScanDuration.WithLabelValues(tenantID.String(), req.ScanType).Observe(float64(result.DurationMs) / 1000)
		switch result.Status {
		case model.ScanStatusCompleted, model.ScanStatusCancelled:
			_ = s.publishEvent(context.Background(), "cyber.asset.scan_completed", tenantID.String(), map[string]any{
				"scan_id":           scan.ID.String(),
				"assets_discovered": result.AssetsDiscovered,
				"assets_new":        result.AssetsNew,
				"assets_updated":    result.AssetsUpdated,
				"duration_ms":       result.DurationMs,
				"status":            result.Status,
			})
		default:
			_ = s.publishEvent(context.Background(), "cyber.asset.scan_failed", tenantID.String(), map[string]any{
				"scan_id": scan.ID.String(),
				"error":   strings.Join(result.Errors, "; "),
			})
		}
	}()

	return scan, nil
}

// CancelScan cancels a running scan.
func (s *AssetService) CancelScan(ctx context.Context, tenantID, scanID uuid.UUID) error {
	s.scanMu.Lock()
	cancel := s.runningScans[scanID]
	s.scanMu.Unlock()
	if cancel != nil {
		cancel()
	}
	return s.scanRepo.Cancel(ctx, tenantID, scanID)
}

// GetStats returns aggregated asset and vulnerability statistics.
func (s *AssetService) GetStats(ctx context.Context, tenantID uuid.UUID) (*dto.AssetStats, error) {
	assetStats, err := s.assetRepo.Stats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	vulnStats, err := s.vulnRepo.VulnStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	lastScan, err := s.scanRepo.LastScanAt(ctx, tenantID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to query last scan time")
	}

	result := &dto.AssetStats{
		TotalAssets:          getInt(assetStats, "total_assets"),
		ActiveAssets:         getInt(assetStats, "active_assets"),
		TotalVulnerabilities: getInt(vulnStats, "total_vulnerabilities"),
		OpenVulnerabilities:  getInt(vulnStats, "open_vulnerabilities"),
		AssetsWithCritical:   getInt(vulnStats, "assets_with_critical_vulns"),
	}

	if v, ok := assetStats["by_type"].(map[string]int); ok {
		result.ByType = v
	}
	if v, ok := assetStats["by_criticality"].(map[string]int); ok {
		result.ByCriticality = v
	}
	if v, ok := assetStats["by_status"].(map[string]int); ok {
		result.ByStatus = v
	}
	if v, ok := assetStats["by_discovery_source"].(map[string]int); ok {
		result.ByDiscoverySource = v
	}
	if v, ok := vulnStats["vulns_by_severity"].(map[string]int); ok {
		result.VulnsBySeverity = v
	}
	if lastScan != nil {
		ts := lastScan.Format("2006-01-02T15:04:05Z")
		result.LastScanAt = &ts
	}
	if v, ok := assetStats["by_department"].(map[string]int); ok {
		result.ByDepartment = v
	}
	if v, ok := assetStats["top_departments"].([]model.AssetCountByName); ok {
		result.TopDepartments = v
	}
	if v, ok := assetStats["by_os"].(map[string]int); ok {
		result.ByOS = v
	}
	if v, ok := assetStats["top_os"].([]model.AssetCountByName); ok {
		result.TopOS = v
	}

	return result, nil
}

func (s *AssetService) SetPredictionLogger(predictionLogger *aigovmiddleware.PredictionLogger) {
	s.predictionLogger = predictionLogger
}

// BulkValidationError is a typed error for bulk validation failures.
type BulkValidationError struct {
	Code    string
	Message string
	Rows    map[int]map[string][]string
}

func (e *BulkValidationError) Error() string { return e.Message }

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(int64); ok {
		return int(v)
	}
	return 0
}

func uuidsToStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// ListRelationships returns all relationships for an asset.
func (s *AssetService) ListRelationships(ctx context.Context, tenantID, assetID uuid.UUID) (any, error) {
	rels, err := s.relRepo.ListForAsset(ctx, tenantID, assetID)
	if err != nil {
		return nil, err
	}
	response := map[string][]map[string]any{
		"outgoing": {},
		"incoming": {},
	}
	for _, rel := range rels {
		if rel.SourceAssetID == assetID {
			response["outgoing"] = append(response["outgoing"], map[string]any{
				"id":   rel.ID,
				"type": rel.RelationshipType,
				"target": map[string]any{
					"id":          rel.TargetAssetID,
					"name":        rel.TargetAssetName,
					"type":        rel.TargetAssetType,
					"criticality": rel.TargetAssetCriticality,
				},
				"metadata":   rel.Metadata,
				"created_at": rel.CreatedAt,
			})
		} else {
			response["incoming"] = append(response["incoming"], map[string]any{
				"id":   rel.ID,
				"type": rel.RelationshipType,
				"source": map[string]any{
					"id":          rel.SourceAssetID,
					"name":        rel.SourceAssetName,
					"type":        rel.SourceAssetType,
					"criticality": rel.SourceAssetCriticality,
				},
				"metadata":   rel.Metadata,
				"created_at": rel.CreatedAt,
			})
		}
	}
	return response, nil
}

// CreateRelationship creates a directed relationship between two assets.
func (s *AssetService) CreateRelationship(ctx context.Context, tenantID, assetID, userID uuid.UUID, req *dto.CreateRelationshipRequest) (any, error) {
	rel, err := s.relRepo.Create(ctx, tenantID, assetID, userID, req)
	if err != nil {
		return nil, err
	}
	_ = s.publishEvent(ctx, "cyber.asset.relationship_created", tenantID.String(), map[string]any{
		"id":        rel.ID.String(),
		"source_id": rel.SourceAssetID.String(),
		"target_id": rel.TargetAssetID.String(),
		"type":      rel.RelationshipType,
	})
	return rel, nil
}

// DeleteRelationship removes a relationship.
func (s *AssetService) DeleteRelationship(ctx context.Context, tenantID, assetID, relID uuid.UUID) error {
	rel, err := s.relRepo.GetByID(ctx, tenantID, relID)
	if err != nil {
		return err
	}
	if err := s.relRepo.Delete(ctx, tenantID, assetID, relID); err != nil {
		return err
	}
	_ = s.publishEvent(ctx, "cyber.asset.relationship_deleted", tenantID.String(), map[string]any{
		"id":        rel.ID.String(),
		"source_id": rel.SourceAssetID.String(),
		"target_id": rel.TargetAssetID.String(),
		"type":      rel.RelationshipType,
	})
	return nil
}

// ListVulnerabilities returns paginated vulns for an asset.
func (s *AssetService) ListVulnerabilities(ctx context.Context, tenantID, assetID uuid.UUID, params *dto.VulnerabilityListParams) (any, int, error) {
	return s.vulnRepo.List(ctx, tenantID, assetID, params)
}

// CreateVulnerability adds a vulnerability to an asset.
func (s *AssetService) CreateVulnerability(ctx context.Context, tenantID, assetID, userID uuid.UUID, req *dto.CreateVulnerabilityRequest) (any, error) {
	vuln, err := s.vulnRepo.Create(ctx, tenantID, assetID, userID, req)
	if err != nil {
		return nil, err
	}
	_ = s.publishEvent(ctx, "cyber.vulnerability.created", tenantID.String(), map[string]any{
		"id":       vuln.ID.String(),
		"asset_id": vuln.AssetID.String(),
		"cve_id":   vuln.CVEID,
		"severity": vuln.Severity,
		"source":   vuln.Source,
	})
	return vuln, nil
}

// UpdateVulnerability updates a vulnerability's status.
func (s *AssetService) UpdateVulnerability(ctx context.Context, tenantID, assetID, vulnID uuid.UUID, req *dto.UpdateVulnerabilityRequest) (any, error) {
	before, err := s.vulnRepo.GetByID(ctx, tenantID, vulnID)
	if err != nil {
		return nil, err
	}
	vuln, err := s.vulnRepo.UpdateStatus(ctx, tenantID, assetID, vulnID, req)
	if err != nil {
		return nil, err
	}
	_ = s.publishEvent(ctx, "cyber.vulnerability.updated", tenantID.String(), map[string]any{
		"id":         vuln.ID.String(),
		"asset_id":   vuln.AssetID.String(),
		"old_status": before.Status,
		"new_status": vuln.Status,
	})
	return vuln, nil
}

// ListScans returns paginated scan history.
func (s *AssetService) ListScans(ctx context.Context, tenantID uuid.UUID, params *dto.ScanListParams) (any, int, error) {
	return s.scanRepo.List(ctx, tenantID, params)
}

// GetScan returns a single scan record.
func (s *AssetService) GetScan(ctx context.Context, tenantID, scanID uuid.UUID) (any, error) {
	return s.scanRepo.GetByID(ctx, tenantID, scanID)
}

// CountAssets returns a simple count of assets with optional filters.
func (s *AssetService) CountAssets(ctx context.Context, tenantID uuid.UUID, params *dto.AssetListParams) (int, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return 0, fmt.Errorf("invalid filter params: %w", err)
	}
	return s.assetRepo.CountByParams(ctx, tenantID, params)
}

// EnrichBatch delegates batch enrichment to the EnrichmentService.
func (s *AssetService) EnrichBatch(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) {
	s.enrichSvc.EnrichBatch(ctx, tenantID, ids)
}

func (s *AssetService) publishEvent(ctx context.Context, eventType, tenantID string, data any) error {
	if s.producer == nil {
		return nil
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	ev := events.NewEventRaw(eventType, "clario360/cyber-service", tenantID, dataJSON)
	topic := events.Topics.AssetEvents
	if strings.HasPrefix(eventType, "cyber.vulnerability.") {
		topic = events.Topics.VulnerabilityEvents
	}
	return s.producer.Publish(ctx, topic, ev)
}

func (s *AssetService) unregisterRunningScan(scanID uuid.UUID) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	delete(s.runningScans, scanID)
}

func scanTimeout(options map[string]any) time.Duration {
	const defaultTimeout = 30 * time.Minute
	if options == nil {
		return defaultTimeout
	}

	switch value := options["timeout_seconds"].(type) {
	case float64:
		if value > 0 {
			return time.Duration(value) * time.Second
		}
	case int:
		if value > 0 {
			return time.Duration(value) * time.Second
		}
	case int64:
		if value > 0 {
			return time.Duration(value) * time.Second
		}
	case string:
		if seconds, err := time.ParseDuration(value + "s"); err == nil && seconds > 0 {
			return seconds
		}
	}

	return defaultTimeout
}

func buildAssetUpdatedEvent(before, after *model.Asset) map[string]any {
	changedFields := make([]string, 0)
	oldValues := make(map[string]any)
	newValues := make(map[string]any)

	recordChange := func(field string, oldValue, newValue any) {
		normalizedOld := normalizeAssetEventValue(oldValue)
		normalizedNew := normalizeAssetEventValue(newValue)
		if fmt.Sprintf("%v", normalizedOld) == fmt.Sprintf("%v", normalizedNew) {
			return
		}
		changedFields = append(changedFields, field)
		oldValues[field] = normalizedOld
		newValues[field] = normalizedNew
	}

	recordChange("name", before.Name, after.Name)
	recordChange("type", before.Type, after.Type)
	recordChange("ip_address", before.IPAddress, after.IPAddress)
	recordChange("hostname", before.Hostname, after.Hostname)
	recordChange("mac_address", before.MACAddress, after.MACAddress)
	recordChange("os", before.OS, after.OS)
	recordChange("os_version", before.OSVersion, after.OSVersion)
	recordChange("owner", before.Owner, after.Owner)
	recordChange("department", before.Department, after.Department)
	recordChange("location", before.Location, after.Location)
	recordChange("criticality", before.Criticality, after.Criticality)
	recordChange("status", before.Status, after.Status)
	recordChange("tags", strings.Join(before.Tags, ","), strings.Join(after.Tags, ","))
	recordChange("metadata", string(before.Metadata), string(after.Metadata))

	return map[string]any{
		"id":             after.ID.String(),
		"changed_fields": changedFields,
		"old_values":     oldValues,
		"new_values":     newValues,
	}
}

func normalizeAssetEventValue(value any) any {
	switch v := value.(type) {
	case *string:
		if v == nil {
			return nil
		}
		return *v
	case *int:
		if v == nil {
			return nil
		}
		return *v
	case *bool:
		if v == nil {
			return nil
		}
		return *v
	case *uuid.UUID:
		if v == nil {
			return nil
		}
		return v.String()
	default:
		return v
	}
}
