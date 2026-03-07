package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
)

type RowCountChecker struct{}

type RowCountConfig struct {
	MinCount         int64   `json:"min_count"`
	MaxChangePercent float64 `json:"max_change_percent"`
}

func NewRowCountChecker() Checker {
	return &RowCountChecker{}
}

func (c *RowCountChecker) Type() string { return "row_count" }

func (c *RowCountChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	var cfg RowCountConfig
	if err := json.Unmarshal(dataset.Rule.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode row_count config: %w", err)
	}
	count := int64(len(dataset.Rows))
	status := "passed"
	summary := fmt.Sprintf("row count is %d", count)
	if cfg.MinCount > 0 && count < cfg.MinCount {
		status = "failed"
		summary = fmt.Sprintf("only %d rows present (minimum %d)", count, cfg.MinCount)
	}
	if dataset.PreviousResult != nil && cfg.MaxChangePercent > 0 && dataset.PreviousResult.RecordsChecked > 0 {
		change := math.Abs(float64(count-dataset.PreviousResult.RecordsChecked)) / float64(dataset.PreviousResult.RecordsChecked) * 100
		if change > cfg.MaxChangePercent {
			status = "failed"
			summary = fmt.Sprintf("row count changed by %.2f%% (%d -> %d)", change, dataset.PreviousResult.RecordsChecked, count)
		}
	}
	return &CheckResult{
		Status:         status,
		RecordsChecked: count,
		RecordsPassed:  count,
		PassRate:       100,
		MetricValue:    float64(count),
		Threshold:      float64(cfg.MinCount),
		FailureSummary: summary,
	}, nil
}

