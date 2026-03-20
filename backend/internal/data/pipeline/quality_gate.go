package pipeline

import (
	"fmt"
	"math"
	"strings"
	"time"

	dataexpr "github.com/clario360/platform/internal/data/expression"
	"github.com/clario360/platform/internal/data/model"
)

type QualityGateEvaluator struct{}

func NewQualityGateEvaluator() *QualityGateEvaluator {
	return &QualityGateEvaluator{}
}

func (g *QualityGateEvaluator) Evaluate(data []map[string]interface{}, gates []model.QualityGate, previousRun *model.PipelineRun) ([]model.QualityGateResult, error) {
	results := make([]model.QualityGateResult, 0, len(gates))
	previousCount := 0.0
	if previousRun != nil {
		previousCount = float64(previousRun.RecordsTransformed)
	}
	for _, gate := range gates {
		result, err := g.evaluateOne(data, gate, previousCount)
		if err != nil {
			return nil, fmt.Errorf("quality gate %q: %w", gate.Name, err)
		}
		results = append(results, result)
	}
	return results, nil
}

func (g *QualityGateEvaluator) evaluateOne(data []map[string]interface{}, gate model.QualityGate, previousCount float64) (model.QualityGateResult, error) {
	result := model.QualityGateResult{
		Name:        gate.Name,
		Metric:      string(gate.Metric),
		Status:      "passed",
		Severity:    defaultGateSeverity(gate.Severity),
		Operator:    gate.Operator,
		Threshold:   gate.Threshold,
		MinValue:    gate.MinValue,
		MaxValue:    gate.MaxValue,
		EvaluatedAt: time.Now().UTC(),
	}
	total := float64(len(data))
	switch gate.Metric {
	case model.QualityGateMetricNullPercentage:
		if gate.Column == "" {
			return result, fmt.Errorf("column is required")
		}
		nulls := 0.0
		for _, row := range data {
			if row[gate.Column] == nil || strings.TrimSpace(fmt.Sprint(row[gate.Column])) == "" {
				nulls++
			}
		}
		if total == 0 {
			result.MetricValue = 0
		} else {
			result.MetricValue = (nulls / total) * 100
		}
	case model.QualityGateMetricUniquePercentage:
		if gate.Column == "" {
			return result, fmt.Errorf("column is required")
		}
		seen := make(map[string]struct{})
		for _, row := range data {
			seen[fmt.Sprint(row[gate.Column])] = struct{}{}
		}
		if total == 0 {
			result.MetricValue = 100
		} else {
			result.MetricValue = (float64(len(seen)) / total) * 100
		}
	case model.QualityGateMetricRowCountChange:
		if previousCount == 0 {
			result.MetricValue = 0
		} else {
			result.MetricValue = math.Abs((total-previousCount)/previousCount) * 100
		}
	case model.QualityGateMetricMinRowCount:
		result.MetricValue = total
	case model.QualityGateMetricCustom:
		if gate.Expression == "" {
			return result, fmt.Errorf("custom gate requires expression")
		}
		compiled, err := dataexpr.Compile(gate.Expression)
		if err != nil {
			return result, err
		}
		value, err := compiled.Evaluate(map[string]interface{}{
			"row_count":          total,
			"previous_row_count": previousCount,
		})
		if err != nil {
			return result, err
		}
		switch typed := value.(type) {
		case bool:
			if typed {
				result.MetricValue = 1
			}
		case float64:
			result.MetricValue = typed
		default:
			result.MetricValue = 0
		}
	default:
		return result, fmt.Errorf("unsupported metric %q", gate.Metric)
	}

	status, message := evaluateGateThreshold(result.MetricValue, gate)
	result.Status = status
	result.Message = message
	return result, nil
}

func evaluateGateThreshold(metric float64, gate model.QualityGate) (string, string) {
	switch strings.ToLower(strings.TrimSpace(gate.Operator)) {
	case "lt", "<":
		if gate.Threshold != nil && metric < *gate.Threshold {
			return "passed", fmt.Sprintf("%.2f < %.2f", metric, *gate.Threshold)
		}
	case "lte", "<=":
		if gate.Threshold != nil && metric <= *gate.Threshold {
			return "passed", fmt.Sprintf("%.2f <= %.2f", metric, *gate.Threshold)
		}
	case "gt", ">":
		if gate.Threshold != nil && metric > *gate.Threshold {
			return "passed", fmt.Sprintf("%.2f > %.2f", metric, *gate.Threshold)
		}
	case "gte", ">=":
		if gate.Threshold != nil && metric >= *gate.Threshold {
			return "passed", fmt.Sprintf("%.2f >= %.2f", metric, *gate.Threshold)
		}
	case "eq", "==":
		if gate.Threshold != nil && metric == *gate.Threshold {
			return "passed", fmt.Sprintf("%.2f == %.2f", metric, *gate.Threshold)
		}
	case "between":
		if gate.MinValue != nil && gate.MaxValue != nil && metric >= *gate.MinValue && metric <= *gate.MaxValue {
			return "passed", fmt.Sprintf("%.2f between %.2f and %.2f", metric, *gate.MinValue, *gate.MaxValue)
		}
	default:
		if gate.Threshold == nil {
			if metric > 0 {
				return "passed", fmt.Sprintf("metric %.2f passed", metric)
			}
			return "failed", fmt.Sprintf("metric %.2f failed", metric)
		}
		if metric <= *gate.Threshold {
			return "passed", fmt.Sprintf("%.2f <= %.2f", metric, *gate.Threshold)
		}
	}
	if strings.EqualFold(gate.Severity, "warning") || strings.EqualFold(gate.Severity, "warn") {
		return "warned", fmt.Sprintf("metric %.2f did not meet threshold", metric)
	}
	return "failed", fmt.Sprintf("metric %.2f did not meet threshold", metric)
}

func defaultGateSeverity(value string) string {
	if strings.TrimSpace(value) == "" {
		return "error"
	}
	return value
}

