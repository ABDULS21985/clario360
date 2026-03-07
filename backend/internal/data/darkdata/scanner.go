package darkdata

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

const darkDataEventsTopic = "data.darkdata.events"

type RawDarkDataAsset struct {
	Name               string
	AssetType          model.DarkDataAssetType
	SourceID           *uuid.UUID
	SourceName         *string
	SchemaName         *string
	TableName          *string
	FilePath           *string
	Reason             model.DarkDataReason
	EstimatedRowCount  *int64
	EstimatedSizeBytes *int64
	ColumnCount        *int
	LastAccessedAt     *time.Time
	LastModifiedAt     *time.Time
	GovernanceStatus   model.DarkDataGovernanceStatus
	LinkedModelID      *uuid.UUID
	Columns            []string
	Metadata           map[string]any
}

type DarkDataStrategy interface {
	Name() string
	Scan(ctx context.Context, tenantID uuid.UUID) ([]RawDarkDataAsset, error)
}

type DarkDataScanner struct {
	strategies []DarkDataStrategy
	repo       *repository.DarkDataRepository
	riskScorer *DarkDataRiskScorer
	classifier *DarkDataClassifier
	producer   *events.Producer
	logger     zerolog.Logger
}

func NewScanner(strategies []DarkDataStrategy, repo *repository.DarkDataRepository, riskScorer *DarkDataRiskScorer, classifier *DarkDataClassifier, producer *events.Producer, logger zerolog.Logger) *DarkDataScanner {
	return &DarkDataScanner{
		strategies: strategies,
		repo:       repo,
		riskScorer: riskScorer,
		classifier: classifier,
		producer:   producer,
		logger:     logger,
	}
}

func (s *DarkDataScanner) RunScan(ctx context.Context, tenantID, triggeredBy uuid.UUID) (*model.DarkDataScan, error) {
	scan := &model.DarkDataScan{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Status:      model.DarkDataScanRunning,
		TriggeredBy: triggeredBy,
		StartedAt:   time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
		ByReason:    json.RawMessage(`{}`),
		ByType:      json.RawMessage(`{}`),
	}
	if err := s.repo.CreateScan(ctx, scan); err != nil {
		return nil, err
	}
	_ = s.publish(ctx, "data.darkdata.scan_started", tenantID, map[string]any{"scan_id": scan.ID, "tenant_id": tenantID})
	var runErr error
	defer func() {
		if runErr == nil {
			return
		}
		completedAt := time.Now().UTC()
		durationMs := completedAt.Sub(scan.StartedAt).Milliseconds()
		scan.Status = model.DarkDataScanFailed
		scan.CompletedAt = &completedAt
		scan.DurationMs = &durationMs
		if err := s.repo.UpdateScan(context.Background(), scan); err != nil {
			s.logger.Error().Err(err).Str("scan_id", scan.ID.String()).Msg("failed to persist failed dark data scan status")
		}
		s.logger.Error().Err(runErr).Str("scan_id", scan.ID.String()).Msg("dark data scan failed")
	}()

	discoveries := make(map[string]RawDarkDataAsset)
	reasonCounts := map[string]int{}
	typeCounts := map[string]int{}
	storageScanned := false
	for _, strategy := range s.strategies {
		if strategy.Name() == "orphaned_files" {
			storageScanned = true
		}
		items, err := strategy.Scan(ctx, tenantID)
		if err != nil {
			s.logger.Error().Err(err).Str("strategy", strategy.Name()).Str("tenant_id", tenantID.String()).Msg("dark data strategy failed")
			continue
		}
		for _, item := range items {
			key := discoveryKey(item)
			if existing, ok := discoveries[key]; ok {
				discoveries[key] = mergeDiscovery(existing, item)
				continue
			}
			discoveries[key] = item
		}
	}

	orderedKeys := make([]string, 0, len(discoveries))
	for key := range discoveries {
		orderedKeys = append(orderedKeys, key)
	}
	sort.Strings(orderedKeys)

	now := time.Now().UTC()
	totalSize := int64(0)
	piiCount := 0
	highRisk := 0
	for _, key := range orderedKeys {
		raw := discoveries[key]
		containsPII, piiTypes, classification := s.classifier.Classify(raw.Name, raw.Columns)
		asset := &model.DarkDataAsset{
			ID:                     uuid.New(),
			TenantID:               tenantID,
			ScanID:                 &scan.ID,
			Name:                   raw.Name,
			AssetType:              raw.AssetType,
			SourceID:               raw.SourceID,
			SourceName:             raw.SourceName,
			SchemaName:             raw.SchemaName,
			TableName:              raw.TableName,
			FilePath:               raw.FilePath,
			Reason:                 raw.Reason,
			EstimatedRowCount:      raw.EstimatedRowCount,
			EstimatedSizeBytes:     raw.EstimatedSizeBytes,
			ColumnCount:            raw.ColumnCount,
			ContainsPII:            containsPII,
			PIITypes:               piiTypes,
			InferredClassification: classification,
			LastAccessedAt:         raw.LastAccessedAt,
			LastModifiedAt:         raw.LastModifiedAt,
			GovernanceStatus:       raw.GovernanceStatus,
			LinkedModelID:          raw.LinkedModelID,
			Metadata:               mustMarshalJSON(raw.Metadata),
			DiscoveredAt:           now,
			CreatedAt:              now,
			UpdatedAt:              now,
		}
		if asset.GovernanceStatus == "" {
			asset.GovernanceStatus = model.DarkDataGovernanceUnmanaged
		}
		if raw.LastAccessedAt != nil {
			days := int(now.Sub(*raw.LastAccessedAt).Hours() / 24)
			asset.DaysSinceAccess = &days
		}
		score, factors := s.riskScorer.ScoreRisk(asset)
		asset.RiskScore = score
		asset.RiskFactors = factors
		if err := s.repo.UpsertAsset(ctx, asset); err != nil {
			runErr = err
			return nil, err
		}
		reasonCounts[string(asset.Reason)]++
		typeCounts[string(asset.AssetType)]++
		if asset.ContainsPII {
			piiCount++
		}
		if asset.RiskScore >= 70 {
			highRisk++
		}
		if asset.EstimatedSizeBytes != nil {
			totalSize += *asset.EstimatedSizeBytes
		}
		_ = s.publish(ctx, "data.darkdata.asset_discovered", tenantID, map[string]any{
			"id":           asset.ID,
			"name":         asset.Name,
			"reason":       asset.Reason,
			"risk_score":   asset.RiskScore,
			"contains_pii": asset.ContainsPII,
		})
	}

	scan.Status = model.DarkDataScanCompleted
	scan.SourcesScanned = inferSourcesScanned(discoveries)
	scan.StorageScanned = storageScanned
	scan.AssetsDiscovered = len(discoveries)
	scan.ByReason = mustMarshalJSON(reasonCounts)
	scan.ByType = mustMarshalJSON(typeCounts)
	scan.PIIAssetsFound = piiCount
	scan.HighRiskFound = highRisk
	scan.TotalSizeBytes = totalSize
	completedAt := time.Now().UTC()
	durationMs := completedAt.Sub(scan.StartedAt).Milliseconds()
	scan.CompletedAt = &completedAt
	scan.DurationMs = &durationMs
	if err := s.repo.UpdateScan(ctx, scan); err != nil {
		runErr = err
		return nil, err
	}
	_ = s.publish(ctx, "data.darkdata.scan_completed", tenantID, map[string]any{
		"scan_id":           scan.ID,
		"assets_discovered": scan.AssetsDiscovered,
		"pii_found":         scan.PIIAssetsFound,
		"high_risk":         scan.HighRiskFound,
	})
	return scan, nil
}

func (s *DarkDataScanner) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload any) error {
	if s.producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return s.producer.Publish(ctx, darkDataEventsTopic, event)
}

func discoveryKey(item RawDarkDataAsset) string {
	builder := strings.Builder{}
	builder.WriteString(string(item.AssetType))
	builder.WriteString("|")
	if item.SourceID != nil {
		builder.WriteString(item.SourceID.String())
	}
	builder.WriteString("|")
	if item.SchemaName != nil {
		builder.WriteString(strings.ToLower(*item.SchemaName))
	}
	builder.WriteString("|")
	if item.TableName != nil {
		builder.WriteString(strings.ToLower(*item.TableName))
	}
	builder.WriteString("|")
	if item.FilePath != nil {
		builder.WriteString(strings.ToLower(*item.FilePath))
	}
	builder.WriteString("|")
	builder.WriteString(string(item.Reason))
	return builder.String()
}

func mergeDiscovery(current, next RawDarkDataAsset) RawDarkDataAsset {
	if darkDataReasonPriority(next.Reason) > darkDataReasonPriority(current.Reason) {
		current.Reason = next.Reason
	}
	if current.EstimatedRowCount == nil {
		current.EstimatedRowCount = next.EstimatedRowCount
	}
	if current.EstimatedSizeBytes == nil {
		current.EstimatedSizeBytes = next.EstimatedSizeBytes
	}
	if current.ColumnCount == nil {
		current.ColumnCount = next.ColumnCount
	}
	if current.LastAccessedAt == nil {
		current.LastAccessedAt = next.LastAccessedAt
	}
	if current.LastModifiedAt == nil {
		current.LastModifiedAt = next.LastModifiedAt
	}
	current.Columns = append(current.Columns, next.Columns...)
	if current.Metadata == nil {
		current.Metadata = map[string]any{}
	}
	for key, value := range next.Metadata {
		current.Metadata[key] = value
	}
	return current
}

func darkDataReasonPriority(reason model.DarkDataReason) int {
	switch reason {
	case model.DarkDataReasonUngoverned:
		return 5
	case model.DarkDataReasonUnclassified:
		return 4
	case model.DarkDataReasonOrphanedFile:
		return 3
	case model.DarkDataReasonStale:
		return 2
	default:
		return 1
	}
}

func inferSourcesScanned(discoveries map[string]RawDarkDataAsset) int {
	seen := make(map[uuid.UUID]struct{})
	for _, item := range discoveries {
		if item.SourceID == nil {
			continue
		}
		seen[*item.SourceID] = struct{}{}
	}
	return len(seen)
}

func mustMarshalJSON(value any) json.RawMessage {
	if value == nil {
		return json.RawMessage(`{}`)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return payload
}
