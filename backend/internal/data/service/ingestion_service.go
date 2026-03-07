package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type IngestionService struct {
	registry      *connector.ConnectorRegistry
	sourceRepo    *repository.SourceRepository
	syncRepo      *repository.SyncRepository
	discoveryOpts connector.DiscoveryOptions
	logger        zerolog.Logger
}

func NewIngestionService(
	registry *connector.ConnectorRegistry,
	sourceRepo *repository.SourceRepository,
	syncRepo *repository.SyncRepository,
	discoveryOpts connector.DiscoveryOptions,
	logger zerolog.Logger,
) *IngestionService {
	return &IngestionService{
		registry:      registry,
		sourceRepo:    sourceRepo,
		syncRepo:      syncRepo,
		discoveryOpts: discoveryOpts,
		logger:        logger,
	}
}

func (s *IngestionService) Run(
	ctx context.Context,
	record *repository.SourceRecord,
	decryptedConfig json.RawMessage,
	syncType model.SyncType,
	trigger model.SyncTrigger,
	userID *uuid.UUID,
) (*model.SyncHistory, error) {
	now := time.Now().UTC()
	history := repository.NewRunningSync(record.Source.TenantID, record.Source.ID, syncType, trigger, userID, now)
	if err := s.syncRepo.Create(ctx, history); err != nil {
		return nil, err
	}

	conn, err := s.registry.Create(record.Source.Type, decryptedConfig)
	if err != nil {
		history.Status = model.SyncStatusFailed
		return history, err
	}
	defer conn.Close()

	status := model.SyncStatusSuccess
	var syncErrors []string

	schema := record.Source.SchemaMetadata
	if schema == nil || syncType == model.SyncTypeSchemaOnly {
		discovered, err := conn.DiscoverSchema(ctx, s.discoveryOpts)
		if err != nil {
			status = model.SyncStatusFailed
			syncErrors = append(syncErrors, err.Error())
		} else {
			schema = discovered
			estimate, estimateErr := conn.EstimateSize(ctx)
			if estimateErr == nil {
				patch := &repository.SizeEstimatePatch{
					TableCount: estimate.TableCount,
					TotalRows:  estimate.TotalRows,
					TotalBytes: estimate.TotalBytes,
				}
				if err := s.sourceRepo.UpdateSchema(ctx, record.Source.TenantID, record.Source.ID, discovered, now, patch); err != nil {
					return nil, err
				}
			}
		}
	}

	if status != model.SyncStatusFailed && syncType != model.SyncTypeSchemaOnly && schema != nil {
		for _, table := range schema.Tables {
			if err := ctx.Err(); err != nil {
				status = model.SyncStatusCancelled
				syncErrors = append(syncErrors, err.Error())
				break
			}
			offset := int64(0)
			for {
				batch, err := conn.FetchData(ctx, qualifiedTableName(table), connector.FetchParams{
					BatchSize: 500,
					Offset:    offset,
				})
				if err != nil {
					status = model.SyncStatusPartial
					syncErrors = append(syncErrors, fmt.Sprintf("%s: %v", table.Name, err))
					break
				}
				history.TablesSynced++
				history.RowsRead += int64(batch.RowCount)
				history.RowsWritten += int64(batch.RowCount)
				history.BytesTransferred += int64(len(mustMarshal(batch.Rows)))
				if !batch.HasMore || batch.RowCount == 0 {
					break
				}
				offset += int64(batch.RowCount)
			}
		}
	}

	completedAt := time.Now().UTC()
	durationMs := completedAt.Sub(history.StartedAt).Milliseconds()
	history.CompletedAt = &completedAt
	history.DurationMs = &durationMs
	history.Status = status
	if len(syncErrors) > 0 {
		payload, _ := json.Marshal(syncErrors)
		history.Errors = payload
		history.ErrorCount = len(syncErrors)
	}
	if err := s.syncRepo.Update(ctx, history); err != nil {
		return nil, err
	}

	lastStatus := string(status)
	var lastError *string
	if len(syncErrors) > 0 {
		message := syncErrors[0]
		lastError = &message
	}
	tableCount := optionalInt(record.Source.TableCount)
	totalRows := optionalInt64(record.Source.TotalRowCount)
	totalBytes := optionalInt64(record.Source.TotalSizeBytes)
	if schema != nil {
		tableCount = &schema.TableCount
	}
	if err := s.sourceRepo.UpdateSyncState(ctx, record.Source.TenantID, record.Source.ID, repository.SyncStatePatch{
		Status:             model.DataSourceStatusActive,
		LastSyncedAt:       &completedAt,
		LastSyncStatus:     &lastStatus,
		LastSyncError:      lastError,
		LastSyncDurationMs: &durationMs,
		TableCount:         tableCount,
		TotalRows:          totalRows,
		TotalBytes:         totalBytes,
	}); err != nil {
		return nil, err
	}

	return history, nil
}

func qualifiedTableName(table model.DiscoveredTable) string {
	if table.SchemaName == "" {
		return table.Name
	}
	return table.SchemaName + "." + table.Name
}

func optionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func optionalInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func mustMarshal(value any) []byte {
	bytes, _ := json.Marshal(value)
	return bytes
}
