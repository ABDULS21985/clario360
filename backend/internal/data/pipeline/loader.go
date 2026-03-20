package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type Loader struct {
	registry  *connector.ConnectorRegistry
	decryptor configDecryptor
	modelRepo *repository.ModelRepository
	sourceRepo *repository.SourceRepository
}

type LoadResult struct {
	RecordsLoaded int64
	RecordsFailed int64
	BytesWritten  int64
}

func NewLoader(registry *connector.ConnectorRegistry, decryptor configDecryptor, modelRepo *repository.ModelRepository, sourceRepo *repository.SourceRepository) *Loader {
	return &Loader{
		registry:   registry,
		decryptor:  decryptor,
		modelRepo:  modelRepo,
		sourceRepo: sourceRepo,
	}
}

func (l *Loader) Load(ctx context.Context, tenantID uuid.UUID, pipeline *model.Pipeline, rows []map[string]interface{}) (*LoadResult, error) {
	if len(rows) == 0 {
		return &LoadResult{}, nil
	}

	targetSource, targetTable, err := l.resolveTarget(ctx, tenantID, pipeline)
	if err != nil {
		return nil, err
	}
	if targetSource == nil || targetTable == "" {
		return &LoadResult{
			RecordsLoaded: int64(len(rows)),
			BytesWritten:  int64(len(mustJSON(rows))),
		}, nil
	}

	decrypted, err := l.decryptor.Decrypt(targetSource.EncryptedConfig, targetSource.Source.EncryptionKeyID)
	if err != nil {
		return nil, fmt.Errorf("decrypt target config: %w", err)
	}
	conn, err := l.registry.Create(targetSource.Source.Type, json.RawMessage(decrypted))
	if err != nil {
		return nil, fmt.Errorf("create target connector: %w", err)
	}
	defer conn.Close()

	writeRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		converted := make(map[string]any, len(row))
		for key, value := range row {
			converted[key] = value
		}
		writeRows = append(writeRows, converted)
	}

	params := connector.WriteParams{
		Strategy: string(pipeline.Config.LoadStrategy),
		MergeKeys: pipeline.Config.MergeKeys,
		Replace:   pipeline.Config.LoadStrategy == model.LoadStrategyFullReplace,
	}
	writeResult, err := conn.WriteData(ctx, targetTable, writeRows, params)
	if err != nil {
		return nil, err
	}
	return &LoadResult{
		RecordsLoaded: writeResult.RowsWritten,
		RecordsFailed: writeResult.RowsFailed,
		BytesWritten:  writeResult.BytesWritten,
	}, nil
}

func (l *Loader) resolveTarget(ctx context.Context, tenantID uuid.UUID, pipeline *model.Pipeline) (*repository.SourceRecord, string, error) {
	if pipeline.TargetID != nil {
		source, err := l.sourceRepo.Get(ctx, tenantID, *pipeline.TargetID)
		if err != nil {
			return nil, "", err
		}
		return source, pipeline.Config.TargetTable, nil
	}
	if pipeline.Config.TargetModelID != nil {
		modelItem, err := l.modelRepo.Get(ctx, tenantID, *pipeline.Config.TargetModelID)
		if err != nil {
			return nil, "", err
		}
		if modelItem.SourceID == nil || modelItem.SourceTable == nil {
			return nil, "", nil
		}
		source, err := l.sourceRepo.Get(ctx, tenantID, *modelItem.SourceID)
		if err != nil {
			return nil, "", err
		}
		return source, *modelItem.SourceTable, nil
	}
	return nil, "", nil
}
