package detection

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

// SigmaEvaluator evaluates Sigma-like rule content with named selections and boolean conditions.
type SigmaEvaluator struct{}

type compiledSigmaRule struct {
	Selections map[string]*CompiledSelection
	Condition  BoolExpr
	Timeframe  time.Duration
	Threshold  int
}

type sigmaEventMatch struct {
	Event            model.SecurityEvent
	MatchedSelection []string
}

// Type returns the evaluator type name.
func (s *SigmaEvaluator) Type() string { return string(model.RuleTypeSigma) }

// Compile validates and compiles Sigma-like rule content.
func (s *SigmaEvaluator) Compile(content json.RawMessage) (interface{}, error) {
	var raw struct {
		Detection map[string]json.RawMessage `json:"detection"`
		Timeframe string                     `json:"timeframe"`
		Threshold int                        `json:"threshold"`
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("parse sigma content: %w", err)
	}
	if len(raw.Detection) == 0 {
		return nil, fmt.Errorf("sigma detection block is required")
	}
	conditionRaw, ok := raw.Detection["condition"]
	if !ok {
		return nil, fmt.Errorf("sigma condition is required")
	}
	var conditionText string
	if err := json.Unmarshal(conditionRaw, &conditionText); err != nil {
		return nil, fmt.Errorf("sigma condition must be a string: %w", err)
	}
	condition, err := ParseCondition(conditionText)
	if err != nil {
		return nil, fmt.Errorf("parse condition: %w", err)
	}

	compiled := &compiledSigmaRule{
		Selections: make(map[string]*CompiledSelection, len(raw.Detection)-1),
		Condition:  condition,
		Threshold:  raw.Threshold,
	}
	if compiled.Threshold < 0 {
		return nil, fmt.Errorf("threshold must be >= 0")
	}
	if raw.Timeframe != "" {
		duration, err := time.ParseDuration(raw.Timeframe)
		if err != nil {
			return nil, fmt.Errorf("invalid timeframe %q: %w", raw.Timeframe, err)
		}
		if duration <= 0 {
			return nil, fmt.Errorf("timeframe must be positive")
		}
		compiled.Timeframe = duration
	}

	for name, selectionRaw := range raw.Detection {
		if name == "condition" {
			continue
		}
		var selection map[string]interface{}
		if err := json.Unmarshal(selectionRaw, &selection); err != nil {
			return nil, fmt.Errorf("selection %s: %w", name, err)
		}
		if len(selection) == 0 {
			return nil, fmt.Errorf("selection %s cannot be empty", name)
		}
		compiledSelection, err := CompileSelection(name, selection)
		if err != nil {
			return nil, err
		}
		compiled.Selections[name] = compiledSelection
	}
	return compiled, nil
}

// Evaluate executes the compiled Sigma rule against a batch of events.
func (s *SigmaEvaluator) Evaluate(compiled interface{}, events []model.SecurityEvent) []model.RuleMatch {
	rule, ok := compiled.(*compiledSigmaRule)
	if !ok || rule == nil {
		return nil
	}
	eventMatches := make([]sigmaEventMatch, 0)
	for _, event := range events {
		selectionResults := make(map[string]bool, len(rule.Selections))
		matchedSelectionNames := make([]string, 0, len(rule.Selections))
		matchedFields := make([]string, 0)
		for name, selection := range rule.Selections {
			matched, fields := EvaluateSelection(selection, &event)
			selectionResults[name] = matched
			if matched {
				matchedSelectionNames = append(matchedSelectionNames, name)
				matchedFields = append(matchedFields, fields...)
			}
		}
		if !rule.Condition.Evaluate(selectionResults) {
			continue
		}
		payload := event
		raw := payload.RawMap()
		raw["_matched_fields"] = matchedFields
		_ = payload.SetRawMap(raw)
		eventMatches = append(eventMatches, sigmaEventMatch{
			Event:            payload,
			MatchedSelection: matchedSelectionNames,
		})
	}

	if len(eventMatches) == 0 {
		return nil
	}
	if rule.Timeframe == 0 {
		matches := make([]model.RuleMatch, 0, len(eventMatches))
		for _, match := range eventMatches {
			matches = append(matches, model.RuleMatch{
				Events:    []model.SecurityEvent{match.Event},
				Timestamp: match.Event.Timestamp,
				MatchDetails: map[string]interface{}{
					"matched_selection_names": match.MatchedSelection,
					"matched_condition_count": len(match.MatchedSelection),
					"group_key":               match.Event.GroupKey(),
				},
			})
		}
		return matches
	}

	grouped := make(map[string][]sigmaEventMatch)
	for _, match := range eventMatches {
		grouped[match.Event.GroupKey()] = append(grouped[match.Event.GroupKey()], match)
	}
	threshold := rule.Threshold
	if threshold == 0 {
		threshold = 1
	}

	matches := make([]model.RuleMatch, 0)
	for groupKey, groupMatches := range grouped {
		sort.Slice(groupMatches, func(i, j int) bool {
			return groupMatches[i].Event.Timestamp.Before(groupMatches[j].Event.Timestamp)
		})
		start := 0
		for end := range groupMatches {
			for groupMatches[end].Event.Timestamp.Sub(groupMatches[start].Event.Timestamp) > rule.Timeframe {
				start++
			}
			window := groupMatches[start : end+1]
			if len(window) < threshold {
				continue
			}
			eventsInWindow := make([]model.SecurityEvent, 0, len(window))
			selectionUnion := make(map[string]struct{})
			for _, item := range window {
				eventsInWindow = append(eventsInWindow, item.Event)
				for _, name := range item.MatchedSelection {
					selectionUnion[name] = struct{}{}
				}
			}
			selectionNames := make([]string, 0, len(selectionUnion))
			for name := range selectionUnion {
				selectionNames = append(selectionNames, name)
			}
			sort.Strings(selectionNames)
			matches = append(matches, model.RuleMatch{
				Events:    eventsInWindow,
				Timestamp: window[len(window)-1].Event.Timestamp,
				MatchDetails: map[string]interface{}{
					"matched_selection_names": selectionNames,
					"matched_condition_count": len(selectionNames),
					"group_key":               groupKey,
					"timeframe":               rule.Timeframe.String(),
					"threshold":               threshold,
				},
			})
			start = end + 1
		}
	}
	return matches
}
