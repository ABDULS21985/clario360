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

type AnalyticalStrategy struct{}

func NewAnalyticalStrategy() interface{ Type() string; Detect(context.Context, cruntime.ModelPair, connector.Connector, connector.Connector) ([]cruntime.RawContradiction, error) } {
	return &AnalyticalStrategy{}
}

func (s *AnalyticalStrategy) Type() string { return "analytical" }

func (s *AnalyticalStrategy) Detect(ctx context.Context, pair cruntime.ModelPair, connA, connB connector.Connector) ([]cruntime.RawContradiction, error) {
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
	result := make([]cruntime.RawContradiction, 0)
	for _, field := range commonNumericFields(pair.ModelA, pair.ModelB) {
		sumA := sumField(rowsA.Rows, field)
		sumB := sumField(rowsB.Rows, field)
		if sumA == 0 && sumB == 0 {
			continue
		}
		diffPct := math.Abs(sumA-sumB) / math.Max(math.Abs(sumA), math.Abs(sumB)) * 100
		if diffPct <= 5 {
			continue
		}
		result = append(result, cruntime.RawContradiction{
			Type:            model.ContradictionTypeAnalytical,
			Title:           fmt.Sprintf("%s aggregation mismatch", field),
			Description:     fmt.Sprintf("Aggregated values for %s differ by %.2f%%.", field, diffPct),
			Column:          field,
			AffectedRecords: 1,
			SampleRecords: []map[string]interface{}{{
				"sum_a":        sumA,
				"sum_b":        sumB,
				"difference_pct": diffPct,
			}},
			SourceA: model.ContradictionSource{SourceID: &pair.SourceA.ID, SourceName: pair.SourceA.Name, ModelID: &pair.ModelA.ID, ModelName: pair.ModelA.DisplayName},
			SourceB: model.ContradictionSource{SourceID: &pair.SourceB.ID, SourceName: pair.SourceB.Name, ModelID: &pair.ModelB.ID, ModelName: pair.ModelB.DisplayName},
			NumericDeltaPct: diffPct,
		})
	}
	return result, nil
}

func commonNumericFields(a, b *model.DataModel) []string {
	fieldsA := make(map[string]model.ModelField)
	for _, field := range a.SchemaDefinition {
		fieldsA[strings.ToLower(field.Name)] = field
	}
	values := make([]string, 0)
	for _, field := range b.SchemaDefinition {
		match, ok := fieldsA[strings.ToLower(field.Name)]
		if !ok {
			continue
		}
		if isNumericType(match.DataType) && isNumericType(field.DataType) {
			values = append(values, match.Name)
		}
	}
	return values
}

func isNumericType(value string) bool {
	switch strings.ToLower(value) {
	case "int", "integer", "float", "double", "decimal", "numeric", "number":
		return true
	default:
		return false
	}
}

func sumField(rows []map[string]any, field string) float64 {
	sum := 0.0
	for _, row := range rows {
		if value, ok := asFloat(row[field]); ok {
			sum += value
		}
	}
	return sum
}
