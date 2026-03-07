package analytics

import (
	"strings"

	"github.com/clario360/platform/internal/data/model"
)

func FormatResult(columns []string, rows []map[string]any, dataModel *model.DataModel, totalCount int64, masking MaskingResult, executionTimeMs int64) *model.QueryResult {
	fieldMap := make(map[string]model.ModelField, len(dataModel.SchemaDefinition))
	for _, field := range dataModel.SchemaDefinition {
		fieldMap[strings.ToLower(field.Name)] = field
	}
	columnMeta := make([]model.ColumnMeta, 0, len(columns))
	masked := make(map[string]struct{}, len(masking.ColumnsMasked))
	for _, column := range masking.ColumnsMasked {
		masked[strings.ToLower(column)] = struct{}{}
	}
	for _, column := range columns {
		field, ok := fieldMap[strings.ToLower(column)]
		meta := model.ColumnMeta{
			Name: column,
		}
		if ok {
			meta.DataType = field.DataType
			meta.Classification = string(field.Classification)
			meta.IsPII = field.PIIType != ""
		} else {
			meta.DataType = "derived"
			meta.Classification = string(model.DataClassificationPublic)
		}
		_, meta.Masked = masked[strings.ToLower(column)]
		columnMeta = append(columnMeta, meta)
	}
	return &model.QueryResult{
		Columns:    columnMeta,
		Rows:       rows,
		RowCount:   len(rows),
		TotalCount: totalCount,
		Truncated:  totalCount > int64(len(rows)),
		Metadata: model.QueryMetadata{
			ModelName:          dataModel.DisplayName,
			DataClassification: string(dataModel.DataClassification),
			PIIMaskingApplied:  len(masking.ColumnsMasked) > 0,
			ColumnsMasked:      masking.ColumnsMasked,
			ExecutionTimeMs:    executionTimeMs,
			CachedResult:       false,
		},
	}
}

