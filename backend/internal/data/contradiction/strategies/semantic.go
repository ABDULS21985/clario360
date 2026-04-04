package strategies

import (
	"context"
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/data/connector"
	cruntime "github.com/clario360/platform/internal/data/contradiction/runtime"
	"github.com/clario360/platform/internal/data/model"
)

type SemanticStrategy struct{}

func NewSemanticStrategy() interface{ Type() string; Detect(context.Context, cruntime.ModelPair, connector.Connector, connector.Connector) ([]cruntime.RawContradiction, error) } {
	return &SemanticStrategy{}
}

func (s *SemanticStrategy) Type() string { return "semantic" }

func (s *SemanticStrategy) Detect(ctx context.Context, pair cruntime.ModelPair, connA, connB connector.Connector) ([]cruntime.RawContradiction, error) {
	result := make([]cruntime.RawContradiction, 0)
	if pair.ModelA.SourceTable != nil {
		rows, err := connA.FetchData(ctx, *pair.ModelA.SourceTable, connector.FetchParams{BatchSize: 10000})
		if err != nil {
			return nil, err
		}
		result = append(result, detectSemanticForModel(pair.ModelA, pair.SourceA, rows.Rows)...)
	}
	if pair.ModelB.SourceTable != nil {
		rows, err := connB.FetchData(ctx, *pair.ModelB.SourceTable, connector.FetchParams{BatchSize: 10000})
		if err != nil {
			return nil, err
		}
		result = append(result, detectSemanticForModel(pair.ModelB, pair.SourceB, rows.Rows)...)
	}
	return result, nil
}

func detectSemanticForModel(modelItem *model.DataModel, source *model.DataSource, rows []map[string]any) []cruntime.RawContradiction {
	result := make([]cruntime.RawContradiction, 0)
	var semanticSamples []map[string]interface{}
	for _, row := range rows {
		if start, ok := parseTime(row["start_date"]); ok {
			if end, ok := parseTime(row["end_date"]); ok && start.After(end) {
				semanticSamples = append(semanticSamples, map[string]interface{}{"start_date": start, "end_date": end})
			}
		}
	}
	if len(semanticSamples) > 0 {
		result = append(result, cruntime.RawContradiction{
			Type:            model.ContradictionTypeSemantic,
			Title:           "Invalid date ordering",
			Description:     "start_date occurs after end_date.",
			Column:          "start_date",
			AffectedRecords: len(semanticSamples),
			SampleRecords:   semanticSamples,
			SourceA: model.ContradictionSource{
				SourceID:   &source.ID,
				SourceName: source.Name,
				ModelID:    &modelItem.ID,
				ModelName:  modelItem.DisplayName,
			},
			SourceB: model.ContradictionSource{
				SourceID:   &source.ID,
				SourceName: source.Name,
				ModelID:    &modelItem.ID,
				ModelName:  modelItem.DisplayName,
			},
		})
	}
	for _, field := range modelItem.SchemaDefinition {
		name := strings.ToLower(field.Name)
		if strings.Contains(name, "percent") || strings.Contains(name, "rate") {
			samples := make([]map[string]interface{}, 0)
			for _, row := range rows {
				if value, ok := asFloat(row[field.Name]); ok && (value < 0 || value > 100) {
					samples = append(samples, map[string]interface{}{field.Name: value})
				}
			}
			if len(samples) > 0 {
				result = append(result, cruntime.RawContradiction{
					Type:            model.ContradictionTypeSemantic,
					Title:           fmt.Sprintf("%s out of range", field.Name),
					Description:     "percentage-like value falls outside 0-100.",
					Column:          field.Name,
					AffectedRecords: len(samples),
					SampleRecords:   samples,
					SourceA:         model.ContradictionSource{SourceID: &source.ID, SourceName: source.Name, ModelID: &modelItem.ID, ModelName: modelItem.DisplayName},
					SourceB:         model.ContradictionSource{SourceID: &source.ID, SourceName: source.Name, ModelID: &modelItem.ID, ModelName: modelItem.DisplayName},
				})
			}
		}
	}
	return result
}
