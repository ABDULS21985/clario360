package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/darkdata"
	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

type DarkDataService struct {
	repo         *repository.DarkDataRepository
	scanner      *darkdata.DarkDataScanner
	modelService *ModelService
	sourceRepo   *repository.SourceRepository
	lineage      *LineageService
	producer     *events.Producer
	logger       zerolog.Logger
}

func NewDarkDataService(repo *repository.DarkDataRepository, scanner *darkdata.DarkDataScanner, modelService *ModelService, sourceRepo *repository.SourceRepository, lineage *LineageService, producer *events.Producer, logger zerolog.Logger) *DarkDataService {
	return &DarkDataService{
		repo:         repo,
		scanner:      scanner,
		modelService: modelService,
		sourceRepo:   sourceRepo,
		lineage:      lineage,
		producer:     producer,
		logger:       logger,
	}
}

func (s *DarkDataService) RunScan(ctx context.Context, tenantID, userID uuid.UUID) (*model.DarkDataScan, error) {
	return s.scanner.RunScan(ctx, tenantID, userID)
}

func (s *DarkDataService) ListScans(ctx context.Context, tenantID uuid.UUID, params dto.ListDarkDataScansParams) ([]*model.DarkDataScan, int, error) {
	return s.repo.ListScans(ctx, tenantID, params)
}

func (s *DarkDataService) GetScan(ctx context.Context, tenantID, id uuid.UUID) (*model.DarkDataScan, error) {
	return s.repo.GetScan(ctx, tenantID, id)
}

func (s *DarkDataService) ListAssets(ctx context.Context, tenantID uuid.UUID, params dto.ListDarkDataParams) ([]*model.DarkDataAsset, int, error) {
	return s.repo.ListAssets(ctx, tenantID, params)
}

func (s *DarkDataService) GetAsset(ctx context.Context, tenantID, id uuid.UUID) (*model.DarkDataAsset, error) {
	return s.repo.GetAsset(ctx, tenantID, id)
}

func (s *DarkDataService) UpdateStatus(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UpdateDarkDataStatusRequest) (*model.DarkDataAsset, error) {
	status := model.DarkDataGovernanceStatus(req.GovernanceStatus)
	switch status {
	case model.DarkDataGovernanceUnmanaged, model.DarkDataGovernanceUnderReview, model.DarkDataGovernanceGoverned, model.DarkDataGovernanceArchived, model.DarkDataGovernanceScheduledDeletion:
	default:
		return nil, fmt.Errorf("%w: invalid governance_status", ErrValidation)
	}
	current, err := s.repo.GetAsset(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateGovernance(ctx, tenantID, id, status, req.GovernanceNotes, &userID, current.LinkedModelID); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetAsset(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	_ = s.publish(ctx, "data.darkdata.status_changed", tenantID, map[string]any{
		"id":         id,
		"old_status": current.GovernanceStatus,
		"new_status": status,
	})
	return updated, nil
}

func (s *DarkDataService) Govern(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.GovernDarkDataRequest) (*model.DataModel, error) {
	asset, err := s.repo.GetAsset(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if asset.SourceID == nil || asset.TableName == nil {
		return nil, fmt.Errorf("%w: only source-backed table assets can be governed automatically", ErrValidation)
	}
	sourceRecord, err := s.sourceRepo.Get(ctx, tenantID, *asset.SourceID)
	if err != nil {
		return nil, err
	}
	if sourceRecord.Source.SchemaMetadata == nil {
		return nil, fmt.Errorf("%w: source schema has not been discovered", ErrValidation)
	}
	var table *model.DiscoveredTable
	for index := range sourceRecord.Source.SchemaMetadata.Tables {
		discovered := sourceRecord.Source.SchemaMetadata.Tables[index]
		if sameDiscoveredTable(discovered, asset.SchemaName, *asset.TableName) {
			table = &sourceRecord.Source.SchemaMetadata.Tables[index]
			break
		}
	}
	if table == nil {
		return nil, fmt.Errorf("%w: dark data table no longer exists in source schema", ErrValidation)
	}
	modelName := req.ModelName
	if modelName == "" {
		modelName = *asset.TableName
	}
	derivedModel, err := s.modelService.CreateDerivedModelFromTable(ctx, tenantID, userID, sourceRecord.Source, *table, modelName, req.AssignQualityRules)
	if err != nil {
		return nil, err
	}
	notes := fmt.Sprintf("Governed on %s via one-click governance flow", time.Now().UTC().Format(time.RFC3339))
	if err := s.repo.UpdateGovernance(ctx, tenantID, id, model.DarkDataGovernanceGoverned, &notes, &userID, &derivedModel.ID); err != nil {
		return nil, err
	}
	if s.lineage != nil {
		_, _ = s.lineage.Record(ctx, tenantID, dto.RecordLineageEdgeRequest{
			SourceType:   string(model.LineageEntityDataSource),
			SourceID:     *asset.SourceID,
			SourceName:   sourceRecord.Source.Name,
			TargetType:   string(model.LineageEntityDataModel),
			TargetID:     derivedModel.ID,
			TargetName:   derivedModel.DisplayName,
			Relationship: string(model.LineageRelationshipDerivedFrom),
			RecordedBy:   string(model.LineageRecordedBySystem),
		})
	}
	_ = s.publish(ctx, "data.darkdata.governed", tenantID, map[string]any{
		"dark_data_id": id,
		"model_id":     derivedModel.ID,
	})
	return derivedModel, nil
}

func (s *DarkDataService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.DarkDataStatsSummary, error) {
	return s.repo.Stats(ctx, tenantID)
}

func (s *DarkDataService) Dashboard(ctx context.Context, tenantID uuid.UUID) (map[string]any, error) {
	stats, err := s.repo.Stats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	topAssets, _, err := s.repo.ListAssets(ctx, tenantID, dto.ListDarkDataParams{Page: 1, PerPage: 10, Sort: "risk_score", Order: "desc"})
	if err != nil {
		return nil, err
	}
	scans, _, err := s.repo.ListScans(ctx, tenantID, dto.ListDarkDataScansParams{Page: 1, PerPage: 5})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"stats":        stats,
		"top_assets":   topAssets,
		"recent_scans": scans,
		"generated_at": time.Now().UTC(),
	}, nil
}

func sameDiscoveredTable(table model.DiscoveredTable, assetSchemaName *string, assetTableName string) bool {
	left := strings.TrimSpace(strings.ToLower(assetTableName))
	right := strings.TrimSpace(strings.ToLower(table.Name))
	if left == "" || right == "" {
		return false
	}
	if left == right {
		if assetSchemaName == nil || strings.TrimSpace(*assetSchemaName) == "" {
			return true
		}
		return strings.EqualFold(strings.TrimSpace(*assetSchemaName), strings.TrimSpace(table.SchemaName))
	}
	leftSchema := ""
	leftTable := left
	if strings.Contains(left, ".") {
		parts := strings.SplitN(left, ".", 2)
		leftSchema = strings.TrimSpace(parts[0])
		leftTable = strings.TrimSpace(parts[1])
	}
	rightSchema := strings.TrimSpace(strings.ToLower(table.SchemaName))
	return leftTable == right && (leftSchema == "" || leftSchema == rightSchema)
}

func (s *DarkDataService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload any) error {
	if s.producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return s.producer.Publish(ctx, "data.darkdata.events", event)
}
