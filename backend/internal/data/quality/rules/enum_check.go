package rules

import (
	"context"
	"encoding/json"
	"fmt"
)

type EnumChecker struct{}

type EnumConfig struct {
	AllowedValues []string `json:"allowed_values"`
}

func NewEnumChecker() Checker {
	return &EnumChecker{}
}

func (c *EnumChecker) Type() string { return "enum" }

func (c *EnumChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	if dataset.Rule.ColumnName == nil {
		return nil, fmt.Errorf("enum rule requires column_name")
	}
	var cfg EnumConfig
	if err := json.Unmarshal(dataset.Rule.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode enum config: %w", err)
	}
	allowed := make(map[string]struct{}, len(cfg.AllowedValues))
	for _, value := range cfg.AllowedValues {
		allowed[value] = struct{}{}
	}
	failed := make([]map[string]interface{}, 0)
	for _, row := range dataset.Rows {
		if _, ok := allowed[fmt.Sprint(row[*dataset.Rule.ColumnName])]; !ok {
			failed = append(failed, row)
		}
	}
	checked := int64(len(dataset.Rows))
	failedCount := int64(len(failed))
	return &CheckResult{
		Status:         statusFromCounts(failedCount),
		RecordsChecked: checked,
		RecordsPassed:  checked - failedCount,
		RecordsFailed:  failedCount,
		PassRate:       passRate(checked, failedCount),
		FailureSamples: limitedSamples(failed, 10),
		FailureSummary: fmt.Sprintf("%d records use values outside the allowed set", failedCount),
	}, nil
}

