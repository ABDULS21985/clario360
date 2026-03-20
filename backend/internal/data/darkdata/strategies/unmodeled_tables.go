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

const scanPageSize = 500

func (s *UnmodeledTablesStrategy) Scan(ctx context.Context, tenantID uuid.UUID) ([]darkdata.RawDarkDataAsset, error) {
	// Paginate sources so tenants with >scanPageSize active sources are handled.
	var sources []*repository.SourceRecord
	for page := 1; ; page++ {
		batch, _, err := s.sourceRepo.List(ctx, tenantID, dto.ListSourcesParams{
			Page: page, PerPage: scanPageSize,
			Statuses: []string{string(model.DataSourceStatusActive)},
		})
		if err != nil {
			return nil, err
		}
		sources = append(sources, batch...)
		if len(batch) < scanPageSize {
			break
		}
	}

	// Paginate models for the same reason.
	var allModels []*model.DataModel
	for page := 1; ; page++ {
		batch, _, err := s.modelRepo.List(ctx, tenantID, dto.ListModelsParams{Page: page, PerPage: scanPageSize})
		if err != nil {
			return nil, err
		}
		allModels = append(allModels, batch...)
		if len(batch) < scanPageSize {
			break
		}
	}
	modeledTables := make(map[string]struct{})
	for _, item := range allModels {
		if item.SourceID == nil || item.SourceTable == nil {
			continue
		}
		modeledTables[fmt.Sprintf("%s|%s", item.SourceID.String(), normalizeQualifiedTable("", *item.SourceTable))] = struct{}{}
	}

	results := make([]darkdata.RawDarkDataAsset, 0)
	for _, source := range sources {
		if source.Source.SchemaMetadata == nil {
			continue
		}
		for _, table := range source.Source.SchemaMetadata.Tables {
			key := fmt.Sprintf("%s|%s", source.Source.ID.String(), normalizeQualifiedTable(table.SchemaName, table.Name))
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

func normalizeQualifiedTable(schemaName, tableName string) string {
	tableName = strings.TrimSpace(strings.ToLower(tableName))
	schemaName = strings.TrimSpace(strings.ToLower(schemaName))
	if tableName == "" {
		return ""
	}
	if strings.Contains(tableName, ".") {
		parts := strings.SplitN(tableName, ".", 2)
		schemaName = strings.TrimSpace(parts[0])
		tableName = strings.TrimSpace(parts[1])
	}
	if schemaName == "" {
		return tableName
	}
	return schemaName + "." + tableName
}
