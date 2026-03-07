package strategies

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/clario360/platform/internal/data/connector"
	cruntime "github.com/clario360/platform/internal/data/contradiction/runtime"
	"github.com/clario360/platform/internal/data/model"
)

type LogicalStrategy struct{}

func NewLogicalStrategy() interface{ Type() string; Detect(context.Context, cruntime.ModelPair, connector.Connector, connector.Connector) ([]cruntime.RawContradiction, error) } {
	return &LogicalStrategy{}
}

func (s *LogicalStrategy) Type() string { return "logical" }

func (s *LogicalStrategy) Detect(ctx context.Context, pair cruntime.ModelPair, connA, connB connector.Connector) ([]cruntime.RawContradiction, error) {
	if pair.ModelA.SourceTable == nil || pair.ModelB.SourceTable == nil {
		return nil, nil
	}
	rowsA, err := connA.FetchData(ctx, *pair.ModelA.SourceTable, connector.FetchParams{BatchSize: 10000})
	if err != nil {
		return nil, err
	}
	rowsB, err := connB.FetchData(ctx, *pair.ModelB.SourceTable, connector.FetchParams{BatchSize: 10000})
	if err != nil {
		return nil, err
	}
	comparable := comparableColumns(pair.ModelA, pair.ModelB, pair.LinkColumn)
	indexA := make(map[string]map[string]any, len(rowsA.Rows))
	for _, row := range rowsA.Rows {
		indexA[fmt.Sprint(row[pair.LinkColumn])] = row
	}
	result := make([]cruntime.RawContradiction, 0)
	for _, column := range comparable {
		samples := make([]map[string]interface{}, 0)
		mismatches := 0
		matched := 0
		for _, rowB := range rowsB.Rows {
			key := fmt.Sprint(rowB[pair.LinkColumn])
			rowA, ok := indexA[key]
			if !ok {
				continue
			}
			matched++
			if fmt.Sprint(rowA[column]) == fmt.Sprint(rowB[column]) {
				continue
			}
			mismatches++
			if len(samples) < 10 {
				samples = append(samples, map[string]interface{}{
					pair.LinkColumn: key,
					"source_a":      rowA[column],
					"source_b":      rowB[column],
				})
			}
		}
		threshold := int(math.Max(1, float64(matched)*0.05))
		if mismatches < threshold {
			continue
		}
		result = append(result, cruntime.RawContradiction{
			Type:            model.ContradictionTypeLogical,
			Title:           fmt.Sprintf("%s mismatch", column),
			Description:     fmt.Sprintf("Column %s differs across matched entities.", column),
			Column:          column,
			AffectedRecords: mismatches,
			SampleRecords:   samples,
			SourceA: model.ContradictionSource{
				SourceID:   &pair.SourceA.ID,
				SourceName: pair.SourceA.Name,
				ModelID:    &pair.ModelA.ID,
				ModelName:  pair.ModelA.DisplayName,
				TableName:  derefString(pair.ModelA.SourceTable),
				ColumnName: column,
				LastSyncedAt: pair.SourceA.LastSyncedAt,
				Status:       string(pair.SourceA.Status),
			},
			SourceB: model.ContradictionSource{
				SourceID:   &pair.SourceB.ID,
				SourceName: pair.SourceB.Name,
				ModelID:    &pair.ModelB.ID,
				ModelName:  pair.ModelB.DisplayName,
				TableName:  derefString(pair.ModelB.SourceTable),
				ColumnName: column,
				LastSyncedAt: pair.SourceB.LastSyncedAt,
				Status:       string(pair.SourceB.Status),
			},
		})
	}
	return result, nil
}

func comparableColumns(a, b *model.DataModel, linkColumn string) []string {
	fieldsA := make(map[string]model.ModelField)
	for _, field := range a.SchemaDefinition {
		fieldsA[strings.ToLower(field.Name)] = field
	}
	columns := make([]string, 0)
	for _, field := range b.SchemaDefinition {
		name := strings.ToLower(field.Name)
		match, ok := fieldsA[name]
		if !ok || strings.EqualFold(match.Name, linkColumn) {
			continue
		}
		if match.IsPrimaryKey || field.IsPrimaryKey || match.IsForeignKey || field.IsForeignKey {
			continue
		}
		if strings.Contains(name, "created_at") || strings.Contains(name, "updated_at") {
			continue
		}
		if strings.EqualFold(match.DataType, field.DataType) {
			columns = append(columns, match.Name)
		}
	}
	return columns
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
