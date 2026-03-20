package feeds

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ThreatFeedSignal struct {
	Source       string         `json:"source"`
	Title        string         `json:"title"`
	Severity     string         `json:"severity"`
	TechniqueIDs []string       `json:"technique_ids,omitempty"`
	Targets      []string       `json:"targets,omitempty"`
	IOCs         []string       `json:"iocs,omitempty"`
	PublishedAt  time.Time      `json:"published_at"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type ThreatFeedIngester struct{}

func NewThreatFeedIngester() *ThreatFeedIngester {
	return &ThreatFeedIngester{}
}

func (i *ThreatFeedIngester) ParseSTIX(payload []byte) ([]ThreatFeedSignal, error) {
	var document struct {
		Objects []map[string]any `json:"objects"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, fmt.Errorf("decode stix payload: %w", err)
	}
	out := make([]ThreatFeedSignal, 0, len(document.Objects))
	for _, obj := range document.Objects {
		typ, _ := obj["type"].(string)
		if typ != "indicator" && typ != "report" {
			continue
		}
		out = append(out, ThreatFeedSignal{
			Source:       "stix",
			Title:        stringField(obj, "name", "title"),
			Severity:     normalizeSeverity(stringField(obj, "severity", "labels")),
			TechniqueIDs: findTechniqueIDs(stringSliceField(obj, "labels", "kill_chain_phases")),
			Targets:      stringSliceField(obj, "sectors", "targets"),
			IOCs:         stringSliceField(obj, "indicator_types", "pattern"),
			PublishedAt:  timeField(obj, "published", "created"),
			Metadata:     obj,
		})
	}
	return out, nil
}

func (i *ThreatFeedIngester) ParseMISP(payload []byte) ([]ThreatFeedSignal, error) {
	var document struct {
		Response []struct {
			Event map[string]any `json:"Event"`
		} `json:"response"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, fmt.Errorf("decode misp payload: %w", err)
	}
	out := make([]ThreatFeedSignal, 0, len(document.Response))
	for _, item := range document.Response {
		event := item.Event
		out = append(out, ThreatFeedSignal{
			Source:       "misp",
			Title:        stringField(event, "info", "threat_level_id"),
			Severity:     normalizeSeverity(stringField(event, "threat_level_id")),
			TechniqueIDs: findTechniqueIDs(stringSliceField(event, "Tag")),
			Targets:      stringSliceField(event, "Orgc", "Tag"),
			IOCs:         stringSliceField(event, "Attribute"),
			PublishedAt:  timeField(event, "date", "timestamp"),
			Metadata:     event,
		})
	}
	return out, nil
}

func (i *ThreatFeedIngester) ParseOTX(payload []byte) ([]ThreatFeedSignal, error) {
	var document struct {
		Pulses []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, fmt.Errorf("decode otx payload: %w", err)
	}
	out := make([]ThreatFeedSignal, 0, len(document.Pulses))
	for _, pulse := range document.Pulses {
		out = append(out, ThreatFeedSignal{
			Source:       "otx",
			Title:        stringField(pulse, "name", "title"),
			Severity:     normalizeSeverity(stringField(pulse, "severity")),
			TechniqueIDs: findTechniqueIDs(stringSliceField(pulse, "tags", "references")),
			Targets:      stringSliceField(pulse, "industries"),
			IOCs:         stringSliceField(pulse, "indicators"),
			PublishedAt:  timeField(pulse, "created", "modified"),
			Metadata:     pulse,
		})
	}
	return out, nil
}

func (i *ThreatFeedIngester) ActivityLevel(items []ThreatFeedSignal, since time.Time) float64 {
	score := 0.0
	for _, item := range items {
		if item.PublishedAt.Before(since) {
			continue
		}
		score += severityWeight(item.Severity)
		if len(item.TechniqueIDs) > 0 {
			score += 0.1 * float64(len(item.TechniqueIDs))
		}
	}
	return score
}

func (i *ThreatFeedIngester) TechniqueFrequency(items []ThreatFeedSignal) map[string]float64 {
	out := map[string]float64{}
	for _, item := range items {
		weight := severityWeight(item.Severity)
		for _, technique := range item.TechniqueIDs {
			out[technique] += weight
		}
	}
	return out
}

func (i *ThreatFeedIngester) TargetFrequency(items []ThreatFeedSignal) map[string]float64 {
	out := map[string]float64{}
	for _, item := range items {
		weight := severityWeight(item.Severity)
		for _, target := range item.Targets {
			target = strings.TrimSpace(strings.ToLower(target))
			if target == "" {
				continue
			}
			out[target] += weight
		}
	}
	return out
}

func severityWeight(value string) float64 {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical", "4":
		return 1.0
	case "high", "3":
		return 0.75
	case "medium", "2":
		return 0.50
	case "low", "1":
		return 0.25
	default:
		return 0.10
	}
}

func normalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "4", "critical":
		return "critical"
	case "3", "high":
		return "high"
	case "2", "medium":
		return "medium"
	case "1", "low":
		return "low"
	default:
		return "medium"
	}
}

func stringField(m map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case map[string]any:
			for _, candidate := range []string{"name", "value"} {
				if nested, ok := typed[candidate].(string); ok && strings.TrimSpace(nested) != "" {
					return strings.TrimSpace(nested)
				}
			}
		case []any:
			for _, item := range typed {
				if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
					return strings.TrimSpace(text)
				}
			}
		}
	}
	return ""
}

func stringSliceField(m map[string]any, keys ...string) []string {
	out := make([]string, 0, 8)
	seen := map[string]struct{}{}
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			appendValue(typed)
		case []string:
			for _, item := range typed {
				appendValue(item)
			}
		case []any:
			for _, item := range typed {
				switch nested := item.(type) {
				case string:
					appendValue(nested)
				case map[string]any:
					appendValue(stringField(nested, "name", "value", "tag", "title"))
				}
			}
		case map[string]any:
			appendValue(stringField(typed, "name", "value"))
		}
	}
	sort.Strings(out)
	return out
}

func findTechniqueIDs(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToUpper(value))
		if !strings.HasPrefix(value, "T") {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func timeField(m map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			for _, layout := range []string{time.RFC3339, "2006-01-02", time.DateTime} {
				if parsed, err := time.Parse(layout, typed); err == nil {
					return parsed.UTC()
				}
			}
		case float64:
			return time.Unix(int64(typed), 0).UTC()
		}
	}
	return time.Time{}
}
