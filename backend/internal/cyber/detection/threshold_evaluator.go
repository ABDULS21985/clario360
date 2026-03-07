package detection

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

// ThresholdEvaluator evaluates count/sum/distinct threshold rules over sliding windows.
type ThresholdEvaluator struct{}

type compiledThresholdRule struct {
	Field       string
	Condition   *CompiledSelection
	Threshold   float64
	Window      time.Duration
	MetricType  string
	MetricField string
}

// Type returns the evaluator type.
func (t *ThresholdEvaluator) Type() string { return string(model.RuleTypeThreshold) }

// Compile validates and compiles threshold rule content.
func (t *ThresholdEvaluator) Compile(content json.RawMessage) (interface{}, error) {
	var raw struct {
		Field     string                 `json:"field"`
		Condition map[string]interface{} `json:"condition"`
		Threshold float64                `json:"threshold"`
		Window    string                 `json:"window"`
		Metric    string                 `json:"metric"`
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("parse threshold content: %w", err)
	}
	if raw.Field == "" {
		return nil, fmt.Errorf("field is required")
	}
	if err := validateFieldPath(raw.Field); err != nil {
		return nil, err
	}
	if len(raw.Condition) == 0 {
		return nil, fmt.Errorf("condition is required")
	}
	selection, err := CompileSelection("threshold_condition", raw.Condition)
	if err != nil {
		return nil, err
	}
	if raw.Threshold <= 0 {
		return nil, fmt.Errorf("threshold must be > 0")
	}
	duration, err := time.ParseDuration(raw.Window)
	if err != nil || duration <= 0 {
		return nil, fmt.Errorf("invalid window %q", raw.Window)
	}
	metricType := "count"
	metricField := ""
	switch {
	case raw.Metric == "", raw.Metric == "count":
	case strings.HasPrefix(raw.Metric, "sum(") && strings.HasSuffix(raw.Metric, ")"):
		metricType = "sum"
		metricField = strings.TrimSuffix(strings.TrimPrefix(raw.Metric, "sum("), ")")
		if err := validateFieldPath(metricField); err != nil {
			return nil, err
		}
	case strings.HasPrefix(raw.Metric, "distinct(") && strings.HasSuffix(raw.Metric, ")"):
		metricType = "distinct"
		metricField = strings.TrimSuffix(strings.TrimPrefix(raw.Metric, "distinct("), ")")
		if err := validateFieldPath(metricField); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported metric %q", raw.Metric)
	}
	return &compiledThresholdRule{
		Field:       raw.Field,
		Condition:   selection,
		Threshold:   raw.Threshold,
		Window:      duration,
		MetricType:  metricType,
		MetricField: metricField,
	}, nil
}

// Evaluate executes the compiled threshold rule against a batch of events.
func (t *ThresholdEvaluator) Evaluate(compiled interface{}, events []model.SecurityEvent) []model.RuleMatch {
	rule, ok := compiled.(*compiledThresholdRule)
	if !ok || rule == nil {
		return nil
	}
	filtered := make(map[string][]model.SecurityEvent)
	for _, event := range events {
		matched, _ := EvaluateSelection(rule.Condition, &event)
		if !matched {
			continue
		}
		value, ok := resolveField(&event, rule.Field)
		if !ok {
			continue
		}
		filtered[fmt.Sprintf("%v", value)] = append(filtered[fmt.Sprintf("%v", value)], event)
	}
	matches := make([]model.RuleMatch, 0)
	for groupValue, groupEvents := range filtered {
		sort.Slice(groupEvents, func(i, j int) bool {
			return groupEvents[i].Timestamp.Before(groupEvents[j].Timestamp)
		})
		start := 0
		for end := range groupEvents {
			for groupEvents[end].Timestamp.Sub(groupEvents[start].Timestamp) > rule.Window {
				start++
			}
			window := groupEvents[start : end+1]
			metricValue := computeThresholdMetric(window, rule.MetricType, rule.MetricField)
			if metricValue < rule.Threshold {
				continue
			}
			matches = append(matches, model.RuleMatch{
				Events:    append([]model.SecurityEvent(nil), window...),
				Timestamp: window[len(window)-1].Timestamp,
				MatchDetails: map[string]interface{}{
					"group_value":  groupValue,
					"metric_type":  rule.MetricType,
					"metric_field": rule.MetricField,
					"metric_value": metricValue,
					"threshold":    rule.Threshold,
					"window":       rule.Window.String(),
				},
			})
			start = end + 1
		}
	}
	return matches
}

func computeThresholdMetric(events []model.SecurityEvent, metricType, metricField string) float64 {
	switch metricType {
	case "count":
		return float64(len(events))
	case "sum":
		total := 0.0
		for _, event := range events {
			value, ok := resolveField(&event, metricField)
			if !ok {
				continue
			}
			number, ok := toFloat64(value)
			if !ok {
				continue
			}
			total += number
		}
		return total
	case "distinct":
		distinct := make(map[string]struct{})
		for _, event := range events {
			value, ok := resolveField(&event, metricField)
			if !ok {
				continue
			}
			distinct[fmt.Sprintf("%v", value)] = struct{}{}
		}
		return float64(len(distinct))
	default:
		return 0
	}
}
