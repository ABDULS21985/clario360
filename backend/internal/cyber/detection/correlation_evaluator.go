package detection

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

// CorrelationEvaluator detects ordered multi-event attack sequences.
type CorrelationEvaluator struct{}

type correlationEventDefinition struct {
	Name      string                 `json:"name"`
	Condition map[string]interface{} `json:"condition"`
}

type compiledCorrelationRule struct {
	Selections      map[string]*CompiledSelection
	Sequence        []string
	GroupBy         string
	Window          time.Duration
	MinFailedCount  int
}

type correlatedEvent struct {
	Name  string
	Event model.SecurityEvent
}

// Type returns the evaluator type.
func (c *CorrelationEvaluator) Type() string { return string(model.RuleTypeCorrelation) }

// Compile validates and compiles a correlation rule.
func (c *CorrelationEvaluator) Compile(content json.RawMessage) (interface{}, error) {
	var raw struct {
		Events         []correlationEventDefinition `json:"events"`
		Sequence       []string                     `json:"sequence"`
		GroupBy        string                       `json:"group_by"`
		Window         string                       `json:"window"`
		MinFailedCount int                          `json:"min_failed_count"`
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("parse correlation content: %w", err)
	}
	if len(raw.Events) == 0 {
		return nil, fmt.Errorf("events are required")
	}
	if len(raw.Sequence) < 2 {
		return nil, fmt.Errorf("sequence requires at least two named events")
	}
	if raw.GroupBy == "" {
		return nil, fmt.Errorf("group_by is required")
	}
	if err := validateFieldPath(raw.GroupBy); err != nil {
		return nil, err
	}
	duration, err := time.ParseDuration(raw.Window)
	if err != nil || duration <= 0 {
		return nil, fmt.Errorf("invalid window %q", raw.Window)
	}
	compiled := &compiledCorrelationRule{
		Selections:     make(map[string]*CompiledSelection, len(raw.Events)),
		Sequence:       raw.Sequence,
		GroupBy:        raw.GroupBy,
		Window:         duration,
		MinFailedCount: raw.MinFailedCount,
	}
	for _, definition := range raw.Events {
		if definition.Name == "" {
			return nil, fmt.Errorf("correlation event name is required")
		}
		if len(definition.Condition) == 0 {
			return nil, fmt.Errorf("condition is required for %s", definition.Name)
		}
		selection, err := CompileSelection(definition.Name, definition.Condition)
		if err != nil {
			return nil, err
		}
		compiled.Selections[definition.Name] = selection
	}
	for _, sequenceName := range raw.Sequence {
		if _, ok := compiled.Selections[sequenceName]; !ok {
			return nil, fmt.Errorf("sequence references undefined event %q", sequenceName)
		}
	}
	return compiled, nil
}

// Evaluate executes the compiled correlation rule.
func (c *CorrelationEvaluator) Evaluate(compiled interface{}, events []model.SecurityEvent) []model.RuleMatch {
	rule, ok := compiled.(*compiledCorrelationRule)
	if !ok || rule == nil {
		return nil
	}
	grouped := make(map[string][]correlatedEvent)
	for _, event := range events {
		groupValue, ok := resolveField(&event, rule.GroupBy)
		if !ok {
			continue
		}
		groupKey := fmt.Sprintf("%v", groupValue)
		for name, selection := range rule.Selections {
			matched, _ := EvaluateSelection(selection, &event)
			if matched {
				grouped[groupKey] = append(grouped[groupKey], correlatedEvent{Name: name, Event: event})
			}
		}
	}

	matches := make([]model.RuleMatch, 0)
	for groupKey, items := range grouped {
		sort.Slice(items, func(i, j int) bool {
			return items[i].Event.Timestamp.Before(items[j].Event.Timestamp)
		})
		for start := 0; start < len(items); start++ {
			if items[start].Name != rule.Sequence[0] {
				continue
			}
			collected := []model.SecurityEvent{items[start].Event}
			windowStart := items[start].Event.Timestamp
			sequenceIndex := 0
			firstEventCount := 1
			matched := false
			for next := start + 1; next < len(items); next++ {
				if items[next].Event.Timestamp.Sub(windowStart) > rule.Window {
					break
				}
				if items[next].Name == rule.Sequence[0] && sequenceIndex == 0 {
					firstEventCount++
					collected = append(collected, items[next].Event)
					continue
				}
				nextExpected := rule.Sequence[sequenceIndex+1]
				if items[next].Name != nextExpected {
					continue
				}
				if sequenceIndex == 0 && rule.MinFailedCount > 0 && firstEventCount < rule.MinFailedCount {
					continue
				}
				collected = append(collected, items[next].Event)
				sequenceIndex++
				if sequenceIndex == len(rule.Sequence)-1 {
					matches = append(matches, model.RuleMatch{
						Events:    append([]model.SecurityEvent(nil), collected...),
						Timestamp: items[next].Event.Timestamp,
						MatchDetails: map[string]interface{}{
							"group_value":       groupKey,
							"sequence":          rule.Sequence,
							"min_failed_count":  rule.MinFailedCount,
							"matched_step_count": len(rule.Sequence),
							"window":            rule.Window.String(),
						},
					})
					start = next
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}
	}
	return matches
}
