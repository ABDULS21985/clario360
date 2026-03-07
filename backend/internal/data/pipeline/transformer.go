package pipeline

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/pipeline/transforms"
)

type TransformStats = transforms.TransformStats

type TransformSummary struct {
	Steps         []TransformStats `json:"steps"`
	FilteredRows  int              `json:"filtered_rows"`
	DedupedRows   int              `json:"deduped_rows"`
	ErrorRows     int              `json:"error_rows"`
}

type Transformer struct {
	logger zerolog.Logger
}

func NewTransformer(logger zerolog.Logger) *Transformer {
	return &Transformer{logger: logger}
}

func (t *Transformer) Apply(data []map[string]interface{}, transformDefs []model.Transformation) ([]map[string]interface{}, *TransformSummary, error) {
	current := cloneRows(data)
	summary := &TransformSummary{Steps: make([]TransformStats, 0, len(transformDefs))}
	for index, step := range transformDefs {
		start := time.Now()
		next, stats, err := t.applyOne(current, step)
		if err != nil {
			return nil, nil, fmt.Errorf("transform %d (%s): %w", index, step.Type, err)
		}
		stats.Type = string(step.Type)
		stats.Duration = time.Since(start)
		summary.Steps = append(summary.Steps, *stats)
		summary.FilteredRows += stats.FilteredRows
		summary.DedupedRows += stats.DedupedRows
		summary.ErrorRows += stats.ErrorRows
		current = next
	}
	return current, summary, nil
}

func (t *Transformer) applyOne(data []map[string]interface{}, step model.Transformation) ([]map[string]interface{}, *TransformStats, error) {
	switch step.Type {
	case model.TransformationRename:
		var cfg transforms.RenameConfig
		if err := json.Unmarshal(step.Config, &cfg); err != nil {
			return nil, nil, fmt.Errorf("decode rename config: %w", err)
		}
		return transforms.ApplyRename(data, cfg)
	case model.TransformationCast:
		var cfg transforms.CastConfig
		if err := json.Unmarshal(step.Config, &cfg); err != nil {
			return nil, nil, fmt.Errorf("decode cast config: %w", err)
		}
		return transforms.ApplyCast(data, cfg)
	case model.TransformationFilter:
		var cfg transforms.FilterConfig
		if err := json.Unmarshal(step.Config, &cfg); err != nil {
			return nil, nil, fmt.Errorf("decode filter config: %w", err)
		}
		return transforms.ApplyFilter(data, cfg)
	case model.TransformationMapValues:
		var cfg transforms.MapValuesConfig
		if err := json.Unmarshal(step.Config, &cfg); err != nil {
			return nil, nil, fmt.Errorf("decode map_values config: %w", err)
		}
		return transforms.ApplyMapValues(data, cfg)
	case model.TransformationDerive:
		var cfg transforms.DeriveConfig
		if err := json.Unmarshal(step.Config, &cfg); err != nil {
			return nil, nil, fmt.Errorf("decode derive config: %w", err)
		}
		return transforms.ApplyDerive(data, cfg)
	case model.TransformationDeduplicate:
		var cfg transforms.DeduplicateConfig
		if err := json.Unmarshal(step.Config, &cfg); err != nil {
			return nil, nil, fmt.Errorf("decode deduplicate config: %w", err)
		}
		return transforms.ApplyDeduplicate(data, cfg)
	case model.TransformationAggregate:
		var cfg transforms.AggregateConfig
		if err := json.Unmarshal(step.Config, &cfg); err != nil {
			return nil, nil, fmt.Errorf("decode aggregate config: %w", err)
		}
		return transforms.ApplyAggregate(data, cfg)
	default:
		return nil, nil, fmt.Errorf("unsupported transformation type %q", step.Type)
	}
}

func cloneRows(data []map[string]interface{}) []map[string]interface{} {
	cloned := make([]map[string]interface{}, 0, len(data))
	for _, row := range data {
		copyRow := make(map[string]interface{}, len(row))
		for key, value := range row {
			copyRow[key] = value
		}
		cloned = append(cloned, copyRow)
	}
	return cloned
}
