package rules

import (
	"context"
	"encoding/json"
	"fmt"
)

type RegexChecker struct{}

type RegexConfig struct {
	Pattern string `json:"pattern"`
}

func NewRegexChecker() Checker {
	return &RegexChecker{}
}

func (c *RegexChecker) Type() string { return "regex" }

func (c *RegexChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	if dataset.Rule.ColumnName == nil {
		return nil, fmt.Errorf("regex rule requires column_name")
	}
	var cfg RegexConfig
	if err := json.Unmarshal(dataset.Rule.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode regex config: %w", err)
	}
	pattern, err := compileRegex(cfg.Pattern)
	if err != nil {
		return nil, err
	}
	failed := make([]map[string]interface{}, 0)
	for _, row := range dataset.Rows {
		if !pattern.MatchString(fmt.Sprint(row[*dataset.Rule.ColumnName])) {
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
		FailureSummary: fmt.Sprintf("%d records do not match the regex", failedCount),
	}, nil
}

