package rules

import (
	"context"
	"fmt"
)

type NotNullChecker struct{}

func NewNotNullChecker() Checker {
	return &NotNullChecker{}
}

func (c *NotNullChecker) Type() string { return "not_null" }

func (c *NotNullChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	if dataset.Rule.ColumnName == nil {
		return nil, fmt.Errorf("not_null rule requires column_name")
	}
	failed := make([]map[string]interface{}, 0)
	for _, row := range dataset.Rows {
		value := row[*dataset.Rule.ColumnName]
		if value == nil || fmt.Sprint(value) == "" {
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
		FailureSummary: fmt.Sprintf("%d records have null values", failedCount),
	}, nil
}

func statusFromCounts(failed int64) string {
	if failed == 0 {
		return "passed"
	}
	return "failed"
}

