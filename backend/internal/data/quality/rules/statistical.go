package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
)

type StatisticalChecker struct{}

type StatisticalConfig struct {
	Column          string  `json:"column"`
	ZScoreThreshold float64 `json:"z_score_threshold"`
}

func NewStatisticalChecker() Checker {
	return &StatisticalChecker{}
}

func (c *StatisticalChecker) Type() string { return "statistical" }

func (c *StatisticalChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	var cfg StatisticalConfig
	if err := json.Unmarshal(dataset.Rule.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode statistical config: %w", err)
	}
	column := cfg.Column
	if column == "" && dataset.Rule.ColumnName != nil {
		column = *dataset.Rule.ColumnName
	}
	if column == "" {
		return nil, fmt.Errorf("statistical rule requires a column")
	}
	numbers := make([]float64, 0, len(dataset.Rows))
	for _, row := range dataset.Rows {
		if number, ok := asFloat(row[column]); ok {
			numbers = append(numbers, number)
		}
	}
	if len(numbers) == 0 {
		return &CheckResult{Status: "warning", FailureSummary: "no numeric values available"}, nil
	}
	sum := 0.0
	for _, number := range numbers {
		sum += number
	}
	mean := sum / float64(len(numbers))
	variance := 0.0
	for _, number := range numbers {
		variance += math.Pow(number-mean, 2)
	}
	stddev := math.Sqrt(variance / float64(len(numbers)))
	failed := make([]map[string]interface{}, 0)
	for _, row := range dataset.Rows {
		number, ok := asFloat(row[column])
		if !ok || stddev == 0 {
			continue
		}
		if math.Abs((number-mean)/stddev) > cfg.ZScoreThreshold {
			failed = append(failed, row)
		}
	}
	checked := int64(len(dataset.Rows))
	failedCount := int64(len(failed))
	status := "passed"
	if failedCount > 0 {
		status = "warning"
	}
	return &CheckResult{
		Status:         status,
		RecordsChecked: checked,
		RecordsPassed:  checked - failedCount,
		RecordsFailed:  failedCount,
		PassRate:       passRate(checked, failedCount),
		FailureSamples: limitedSamples(failed, 10),
		FailureSummary: fmt.Sprintf("%d outliers detected", failedCount),
		MetricValue:    failedCountFloat(failedCount),
		Threshold:      cfg.ZScoreThreshold,
	}, nil
}

func failedCountFloat(value int64) float64 {
	return float64(value)
}

