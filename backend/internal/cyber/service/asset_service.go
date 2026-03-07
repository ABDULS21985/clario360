package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

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
	assetRepo    *repository.AssetRepository
	vulnRepo     *repository.VulnerabilityRepository
	relRepo      *repository.RelationshipRepository
	scanRepo     *repository.ScanRepository
	scanRegistry *scanner.Registry
	classifier   *classifier.AssetClassifier
	enrichSvc    *EnrichmentService
	producer     *events.Producer
	metrics      *metrics.Metrics
	cfg          *cyberconfig.Config
	db           *pgxpool.Pool
	logger       zerolog.Logger
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
		s.metrics.ClassificationsTotal.WithLabelValues(tenantID.String(), ruleName).Inc()
		if crit != asset.Criticality {
			s.metrics.ClassificationChanged.WithLabelValues(tenantID.String(), string(asset.Criticality), string(crit)).Inc()
			_ = s.assetRepo.BulkUpdateCriticality(ctx, tenantID, map[uuid.UUID]model.Criticality{asset.ID: crit})
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
		"asset_id": asset.ID.String(),
		"type":     string(asset.Type),
		"name":     asset.Name,
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
	return s.assetRepo.Update(ctx, tenantID, assetID, req)
}

// DeleteAsset soft-deletes an asset.
func (s *AssetService) DeleteAsset(ctx context.Context, tenantID, assetID uuid.UUID) error {
	if err := s.assetRepo.SoftDelete(ctx, tenantID, assetID); err != nil {
		return err
	}
	s.metrics.AssetsDeleted.WithLabelValues(tenantID.String(), "unknown").Inc()
	_ = s.publishEvent(ctx, "cyber.asset.deleted", tenantID.String(), map[string]any{"asset_id": assetID.String()})
	return nil
}

// PatchTags updates tags on an asset.
func (s *AssetService) PatchTags(ctx context.Context, tenantID, assetID uuid.UUID, req *dto.TagPatchRequest) (*model.Asset, error) {
	return s.assetRepo.PatchTags(ctx, tenantID, assetID, req)
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
		results := s.classifier.ClassifyBatch(fetchedAssets)
		updates := make(map[uuid.UUID]model.Criticality)
		for _, r := range results {
			if r.Changed {
				updates[r.AssetID] = r.Criticality
			}
		}
		if len(updates) > 0 {
			_ = s.assetRepo.BulkUpdateCriticality(bgCtx, tenantID, updates)
		}
		s.enrichSvc.EnrichBatch(bgCtx, tenantID, ids)
	}()

	// Publish bulk event
	_ = s.publishEvent(ctx, "cyber.asset.bulk_created", tenantID.String(), map[string]any{
		"count": len(ids),
		"ids":   uuidsToStrings(ids),
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

	// Run scan asynchronously
	go func() {
		bgCtx := scanner.WithTenantID(context.Background(), tenantID)
		result, runErr := sc.Scan(bgCtx, cfg)
		if runErr != nil {
			result = &model.ScanResult{
				ScanID:  scan.ID,
				Status:  model.ScanStatusFailed,
				Errors:  []string{runErr.Error()},
			}
		}
		result.ScanID = scan.ID
		if err := s.scanRepo.Complete(bgCtx, tenantID, scan.ID, result); err != nil {
			s.logger.Error().Err(err).Str("scan_id", scan.ID.String()).Msg("failed to record scan completion")
		}
		status := string(result.Status)
		s.metrics.ScansTotal.WithLabelValues(tenantID.String(), req.ScanType, status).Inc()
		s.metrics.ScanDuration.WithLabelValues(tenantID.String(), req.ScanType).Observe(float64(result.DurationMs) / 1000)
	}()

	return scan, nil
}

// CancelScan cancels a running scan.
func (s *AssetService) CancelScan(ctx context.Context, tenantID, scanID uuid.UUID) error {
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
		TotalAssets:           getInt(assetStats, "total_assets"),
		ActiveAssets:          getInt(assetStats, "active_assets"),
		TotalVulnerabilities:  getInt(vulnStats, "total_vulnerabilities"),
		OpenVulnerabilities:   getInt(vulnStats, "open_vulnerabilities"),
		AssetsWithCritical:    getInt(vulnStats, "assets_with_critical_vulns"),
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

	return result, nil
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

func (s *AssetService) publishEvent(ctx context.Context, eventType, tenantID string, data any) error {
	if s.producer == nil {
		return nil
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	ev := events.NewEventRaw(eventType, "clario360/cyber-service", tenantID, dataJSON)
	return s.producer.Publish(ctx, events.Topics.AssetEvents, ev)
}
