package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type FreshnessChecker struct{}

type FreshnessConfig struct {
	Column      string  `json:"column"`
	MaxAgeHours float64 `json:"max_age_hours"`
}

func NewFreshnessChecker() Checker {
	return &FreshnessChecker{}
}

func (c *FreshnessChecker) Type() string { return "freshness" }

func (c *FreshnessChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	var cfg FreshnessConfig
	if err := json.Unmarshal(dataset.Rule.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode freshness config: %w", err)
	}
	column := cfg.Column
	if column == "" && dataset.Rule.ColumnName != nil {
		column = *dataset.Rule.ColumnName
	}
	if column == "" {
		return nil, fmt.Errorf("freshness rule requires a column")
	}
	var latest *time.Time
	for _, row := range dataset.Rows {
		if tm, ok := asTime(row[column]); ok {
			if latest == nil || tm.After(*latest) {
				copyValue := tm
				latest = &copyValue
			}
		}
	}
	if latest == nil {
		return &CheckResult{
			Status:         "failed",
			RecordsChecked: int64(len(dataset.Rows)),
			RecordsFailed:  int64(len(dataset.Rows)),
			FailureSummary: "no timestamp values available for freshness evaluation",
		}, nil
	}
	ageHours := time.Since(*latest).Hours()
	status := "passed"
	if ageHours > cfg.MaxAgeHours {
		status = "failed"
	}
	return &CheckResult{
		Status:         status,
		RecordsChecked: int64(len(dataset.Rows)),
		RecordsPassed:  int64(len(dataset.Rows)),
		PassRate:       100,
		MetricValue:    ageHours,
		Threshold:      cfg.MaxAgeHours,
		FailureSummary: fmt.Sprintf("latest record is %.2fh old", ageHours),
	}, nil
}

