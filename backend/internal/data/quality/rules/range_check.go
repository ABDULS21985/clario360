package rules

import (
	"context"
	"encoding/json"
	"fmt"
)

type RangeChecker struct{}

type RangeConfig struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

func NewRangeChecker() Checker {
	return &RangeChecker{}
}

func (c *RangeChecker) Type() string { return "range" }

func (c *RangeChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	if dataset.Rule.ColumnName == nil {
		return nil, fmt.Errorf("range rule requires column_name")
	}
	var cfg RangeConfig
	if err := json.Unmarshal(dataset.Rule.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode range config: %w", err)
	}
	failed := make([]map[string]interface{}, 0)
	for _, row := range dataset.Rows {
		number, ok := asFloat(row[*dataset.Rule.ColumnName])
		if !ok || number < cfg.Min || number > cfg.Max {
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
		FailureSummary: fmt.Sprintf("%d records are out of range", failedCount),
		Threshold:      cfg.Max,
	}, nil
}

