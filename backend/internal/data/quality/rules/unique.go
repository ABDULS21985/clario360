package rules

import (
	"context"
	"fmt"
)

type UniqueChecker struct{}

func NewUniqueChecker() Checker {
	return &UniqueChecker{}
}

func (c *UniqueChecker) Type() string { return "unique" }

func (c *UniqueChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	if dataset.Rule.ColumnName == nil {
		return nil, fmt.Errorf("unique rule requires column_name")
	}
	counts := make(map[string]int64)
	failed := make([]map[string]interface{}, 0)
	for _, row := range dataset.Rows {
		key := fmt.Sprint(row[*dataset.Rule.ColumnName])
		counts[key]++
	}
	for _, row := range dataset.Rows {
		key := fmt.Sprint(row[*dataset.Rule.ColumnName])
		if counts[key] > 1 {
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
		FailureSummary: fmt.Sprintf("%d duplicate records found", failedCount),
	}, nil
}

