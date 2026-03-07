package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

// AnomalyEvaluator detects statistical deviations from historical baselines.
type AnomalyEvaluator struct {
	store *BaselineStore
}

type compiledAnomalyRule struct {
	Metric             string
	GroupBy            string
	Window             time.Duration
	ZScoreThreshold    float64
	MinBaselineSamples int64
	Direction          string
	RuleID             uuid.UUID
	TenantID           uuid.UUID
}

// NewAnomalyEvaluator creates an anomaly evaluator backed by the given baseline store.
func NewAnomalyEvaluator(store *BaselineStore) *AnomalyEvaluator {
	return &AnomalyEvaluator{store: store}
}

// Type returns the evaluator type name.
func (a *AnomalyEvaluator) Type() string { return string(model.RuleTypeAnomaly) }

// Compile validates and compiles anomaly rule content.
func (a *AnomalyEvaluator) Compile(content json.RawMessage) (interface{}, error) {
	var raw struct {
		Metric             string  `json:"metric"`
		GroupBy            string  `json:"group_by"`
		Window             string  `json:"window"`
		ZScoreThreshold    float64 `json:"z_score_threshold"`
		MinBaselineSamples int64   `json:"min_baseline_samples"`
		Direction          string  `json:"direction"`
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("parse anomaly content: %w", err)
	}
	if raw.Metric == "" {
		return nil, fmt.Errorf("metric is required")
	}
	if raw.GroupBy == "" {
		return nil, fmt.Errorf("group_by is required")
	}
	if err := validateFieldPath(raw.GroupBy); err != nil {
		return nil, err
	}
	window, err := time.ParseDuration(raw.Window)
	if err != nil || window <= 0 {
		return nil, fmt.Errorf("invalid window %q", raw.Window)
	}
	if raw.ZScoreThreshold <= 0 {
		return nil, fmt.Errorf("z_score_threshold must be > 0")
	}
	if raw.MinBaselineSamples < 0 {
		return nil, fmt.Errorf("min_baseline_samples must be >= 0")
	}
	switch raw.Direction {
	case "", "above", "below", "both":
	default:
		return nil, fmt.Errorf("invalid direction %q", raw.Direction)
	}
	if raw.Direction == "" {
		raw.Direction = "above"
	}
	return &compiledAnomalyRule{
		Metric:             raw.Metric,
		GroupBy:            raw.GroupBy,
		Window:             window,
		ZScoreThreshold:    raw.ZScoreThreshold,
		MinBaselineSamples: raw.MinBaselineSamples,
		Direction:          raw.Direction,
	}, nil
}

// Evaluate computes current-window values and compares them to historical baselines.
func (a *AnomalyEvaluator) Evaluate(compiled interface{}, events []model.SecurityEvent) []model.RuleMatch {
	rule, ok := compiled.(*compiledAnomalyRule)
	if !ok || rule == nil || a.store == nil {
		return nil
	}
	grouped := make(map[string][]model.SecurityEvent)
	for _, event := range events {
		groupValue, ok := resolveField(&event, rule.GroupBy)
		if !ok {
			continue
		}
		grouped[fmt.Sprintf("%v", groupValue)] = append(grouped[fmt.Sprintf("%v", groupValue)], event)
	}

	results := make([]model.RuleMatch, 0)
	ctx := context.Background()
	for groupValue, groupEvents := range grouped {
		sort.Slice(groupEvents, func(i, j int) bool {
			return groupEvents[i].Timestamp.Before(groupEvents[j].Timestamp)
		})
		windows := slidingWindows(groupEvents, rule.Window)
		for _, windowEvents := range windows {
			if len(windowEvents) == 0 {
				continue
			}
			value := computeAnomalyMetric(windowEvents, rule.Metric)
			baseline, err := a.store.GetBaseline(ctx, rule.TenantID, rule.RuleID, groupValue)
			if err != nil {
				continue
			}
			if baseline.Count < rule.MinBaselineSamples {
				_, _ = a.store.UpdateBaseline(ctx, rule.TenantID, rule.RuleID, groupValue, value)
				continue
			}

			stdDev := baseline.StdDev()
			zScore := 0.0
			isAnomalous := false
			if stdDev == 0 {
				if value != baseline.Mean {
					zScore = math.Inf(1)
					if value < baseline.Mean {
						zScore = math.Inf(-1)
					}
					isAnomalous = directionSatisfied(rule.Direction, zScore, rule.ZScoreThreshold)
				}
			} else {
				zScore = (value - baseline.Mean) / stdDev
				isAnomalous = directionSatisfied(rule.Direction, zScore, rule.ZScoreThreshold)
			}

			nextBaseline := adaptiveBaseline(baseline, value)
			_ = a.store.StoreBaseline(ctx, rule.TenantID, rule.RuleID, groupValue, nextBaseline)

			if !isAnomalous {
				continue
			}
			deviationPercent := 0.0
			if baseline.Mean != 0 {
				deviationPercent = ((value - baseline.Mean) / math.Abs(baseline.Mean)) * 100
			}
			results = append(results, model.RuleMatch{
				Events:    append([]model.SecurityEvent(nil), windowEvents...),
				Timestamp: windowEvents[len(windowEvents)-1].Timestamp,
				MatchDetails: map[string]interface{}{
					"group_value":        groupValue,
					"metric":             rule.Metric,
					"current_value":      value,
					"mean":               baseline.Mean,
					"std_dev":            stdDev,
					"z_score":            zScore,
					"deviation_percent":  deviationPercent,
					"baseline_samples":   baseline.Count,
					"direction":          rule.Direction,
				},
			})
		}
	}
	return results
}

func computeAnomalyMetric(events []model.SecurityEvent, metric string) float64 {
	switch metric {
	case "event_count":
		return float64(len(events))
	case "unique_ips":
		values := make(map[string]struct{})
		for _, event := range events {
			if event.SourceIP != nil {
				values[*event.SourceIP] = struct{}{}
			}
			if event.DestIP != nil {
				values[*event.DestIP] = struct{}{}
			}
		}
		return float64(len(values))
	case "bytes_transferred":
		total := 0.0
		for _, event := range events {
			if value, ok := resolveField(&event, "raw.bytes_transferred"); ok {
				if n, ok := toFloat64(value); ok {
					total += n
				}
			}
		}
		return total
	case "dns_query_count":
		return float64(len(events))
	case "login_hour":
		if len(events) == 0 {
			return 0
		}
		return float64(events[len(events)-1].Timestamp.UTC().Hour())
	case "connection_interval_regularity":
		if len(events) < 3 {
			return 1
		}
		intervals := make([]float64, 0, len(events)-1)
		for i := 1; i < len(events); i++ {
			intervals = append(intervals, events[i].Timestamp.Sub(events[i-1].Timestamp).Seconds())
		}
		mean := 0.0
		for _, interval := range intervals {
			mean += interval
		}
		mean /= float64(len(intervals))
		if mean == 0 {
			return 0
		}
		variance := 0.0
		for _, interval := range intervals {
			diff := interval - mean
			variance += diff * diff
		}
		variance /= float64(len(intervals))
		return math.Sqrt(variance) / mean
	default:
		if len(events) == 0 {
			return 0
		}
		if value, ok := resolveField(&events[len(events)-1], "raw."+metric); ok {
			if n, ok := toFloat64(value); ok {
				return n
			}
		}
		return float64(len(events))
	}
}

func slidingWindows(events []model.SecurityEvent, window time.Duration) [][]model.SecurityEvent {
	if len(events) == 0 {
		return nil
	}
	result := make([][]model.SecurityEvent, 0)
	start := 0
	for end := range events {
		for events[end].Timestamp.Sub(events[start].Timestamp) > window {
			start++
		}
		result = append(result, append([]model.SecurityEvent(nil), events[start:end+1]...))
	}
	return result
}

func directionSatisfied(direction string, zScore, threshold float64) bool {
	switch direction {
	case "below":
		return zScore < -threshold
	case "both":
		return math.Abs(zScore) > threshold
	default:
		return zScore > threshold
	}
}

func adaptiveBaseline(current *Baseline, value float64) *Baseline {
	if current == nil || current.Count == 0 {
		return &Baseline{
			Mean:        value,
			Variance:    0,
			Count:       1,
			LastUpdated: time.Now().UTC(),
		}
	}
	alpha := math.Min(2.0/float64(current.Count+1), 0.05)
	next := *current
	next.Count++
	delta := value - next.Mean
	next.Mean = next.Mean + alpha*delta
	next.Variance = (1-alpha)*(next.Variance+alpha*delta*delta)
	next.LastUpdated = time.Now().UTC()
	return &next
}
