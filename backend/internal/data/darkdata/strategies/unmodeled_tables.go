package strategies

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/darkdata"
	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type UnmodeledTablesStrategy struct {
	sourceRepo *repository.SourceRepository
	modelRepo  *repository.ModelRepository
}

func NewUnmodeledTablesStrategy(sourceRepo *repository.SourceRepository, modelRepo *repository.ModelRepository) *UnmodeledTablesStrategy {
	return &UnmodeledTablesStrategy{sourceRepo: sourceRepo, modelRepo: modelRepo}
}

func (s *UnmodeledTablesStrategy) Name() string {
	return "unmodeled_tables"
}

func (s *UnmodeledTablesStrategy) Scan(ctx context.Context, tenantID uuid.UUID) ([]darkdata.RawDarkDataAsset, error) {
	sources, _, err := s.sourceRepo.List(ctx, tenantID, dto.ListSourcesParams{Page: 1, PerPage: 5000, Status: string(model.DataSourceStatusActive)})
	if err != nil {
		return nil, err
	}
	models, _, err := s.modelRepo.List(ctx, tenantID, dto.ListModelsParams{Page: 1, PerPage: 5000})
	if err != nil {
		return nil, err
	}
	modeledTables := make(map[string]struct{})
	for _, item := range models {
		if item.SourceID == nil || item.SourceTable == nil {
			continue
		}
		modeledTables[fmt.Sprintf("%s|%s", item.SourceID.String(), strings.ToLower(*item.SourceTable))] = struct{}{}
	}

	results := make([]darkdata.RawDarkDataAsset, 0)
	for _, source := range sources {
		if source.Source.SchemaMetadata == nil {
			continue
		}
		for _, table := range source.Source.SchemaMetadata.Tables {
			key := fmt.Sprintf("%s|%s", source.Source.ID.String(), strings.ToLower(table.Name))
			if _, ok := modeledTables[key]; ok {
				continue
			}
			sourceName := source.Source.Name
			schemaName := table.SchemaName
			tableName := table.Name
			columnCount := len(table.Columns)
			columnNames := make([]string, 0, len(table.Columns))
			for _, column := range table.Columns {
				columnNames = append(columnNames, column.Name)
			}
			assetType := model.DarkDataAssetDatabaseTable
			if strings.EqualFold(table.Type, "view") {
				assetType = model.DarkDataAssetDatabaseView
			}
			results = append(results, darkdata.RawDarkDataAsset{
				Name:               fmt.Sprintf("%s.%s", source.Source.Name, table.Name),
				AssetType:          assetType,
				SourceID:           &source.Source.ID,
				SourceName:         &sourceName,
				SchemaName:         &schemaName,
				TableName:          &tableName,
				Reason:             model.DarkDataReasonUnmodeled,
				EstimatedRowCount:  &table.EstimatedRows,
				EstimatedSizeBytes: &table.SizeBytes,
				ColumnCount:        &columnCount,
				Columns:            columnNames,
				Metadata: map[string]any{
					"source_type": source.Source.Type,
					"table_type":  table.Type,
				},
			})
		}
	}
	return results, nil
}

