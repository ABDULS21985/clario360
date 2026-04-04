package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// MISPAdapter parses MISP event exports and extracts indicators.
type MISPAdapter struct{}

func NewMISPAdapter() *MISPAdapter { return &MISPAdapter{} }

func (a *MISPAdapter) SourceType() string { return "misp" }

type mispEnvelope struct {
	Response []mispResponseItem `json:"response"`
	Event    *mispEvent         `json:"Event"`
}

type mispResponseItem struct {
	Event *mispEvent `json:"Event"`
}

type mispEvent struct {
	ID            string          `json:"id"`
	Info          string          `json:"info"`
	ThreatLevelID json.RawMessage `json:"threat_level_id"`
	Tag           []mispTag       `json:"Tag"`
	Attribute     []mispAttribute `json:"Attribute"`
}

type mispAttribute struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Value     string    `json:"value"`
	Category  string    `json:"category"`
	Comment   string    `json:"comment"`
	Timestamp string    `json:"timestamp"`
	FirstSeen string    `json:"first_seen"`
	LastSeen  string    `json:"last_seen"`
	ToIDs     bool      `json:"to_ids"`
	Tag       []mispTag `json:"Tag"`
}

type mispTag struct {
	Name string `json:"name"`
}

func (a *MISPAdapter) Parse(_ context.Context, raw []byte) ([]NormalizedIndicator, error) {
	var envelope mispEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("misp adapter: unmarshal: %w", err)
	}

	events := make([]mispEvent, 0, len(envelope.Response)+1)
	if envelope.Event != nil {
		events = append(events, *envelope.Event)
	}
	for _, item := range envelope.Response {
		if item.Event != nil {
			events = append(events, *item.Event)
		}
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("misp adapter: no events found")
	}

	now := time.Now().UTC()
	out := make([]NormalizedIndicator, 0)
	for _, event := range events {
		eventTags := mispTagNames(event.Tag)
		eventSeverity := mispSeverity(event.ThreatLevelID)
		for _, attr := range event.Attribute {
			iocType := mapMISPIOCType(attr.Type)
			if iocType == "" || strings.TrimSpace(attr.Value) == "" {
				continue
			}

			firstSeen := parseFlexibleTime(attr.FirstSeen)
			if firstSeen.IsZero() {
				firstSeen = parseFlexibleTime(attr.Timestamp)
			}
			if firstSeen.IsZero() {
				firstSeen = now
			}

			lastSeen := parseFlexibleTime(attr.LastSeen)
			if lastSeen.IsZero() {
				lastSeen = firstSeen
			}

			tags := append(eventTags, mispTagNames(attr.Tag)...)
			category := inferCategory(attr.Category, tags)
			severity := eventSeverity
			if severity == "medium" && attr.ToIDs {
				severity = "high"
			}

			title := strings.TrimSpace(attr.Comment)
			if title == "" {
				title = strings.TrimSpace(event.Info)
			}
			if title == "" {
				title = fmt.Sprintf("MISP Indicator: %s", attr.Value)
			}

			externalRef := strings.TrimSpace(attr.ID)
			if externalRef == "" {
				externalRef = strings.TrimSpace(event.ID)
			}

			out = append(out, NormalizedIndicator{
				Title:           title,
				Description:     strings.TrimSpace(attr.Comment),
				SeverityCode:    severity,
				CategoryCode:    category,
				ConfidenceScore: mispConfidence(attr.ToIDs),
				IOCType:         iocType,
				IOCValue:        strings.TrimSpace(attr.Value),
				ExternalRef:     externalRef,
				FirstSeen:       firstSeen,
				LastSeen:        lastSeen,
				Tags:            uniqueLowerStrings(tags),
			})
		}
	}

	return out, nil
}

func mapMISPIOCType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ip", "ip-src", "ip-dst":
		return "ip"
	case "domain", "hostname":
		return "domain"
	case "url", "uri":
		return "url"
	case "email", "email-src", "email-dst":
		return "email"
	case "md5":
		return "hash_md5"
	case "sha1", "sha-1":
		return "hash_sha1"
	case "sha256", "sha-256":
		return "hash_sha256"
	default:
		return ""
	}
}

func mispSeverity(raw json.RawMessage) string {
	level := strings.TrimSpace(strings.Trim(string(raw), `"`))
	switch level {
	case "1":
		return "critical"
	case "2":
		return "high"
	case "4":
		return "low"
	default:
		return "medium"
	}
}

func mispConfidence(toIDs bool) float64 {
	if toIDs {
		return 0.9
	}
	return 0.7
}

func mispTagNames(tags []mispTag) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if trimmed := strings.TrimSpace(tag.Name); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func inferCategory(seed string, tags []string) string {
	candidates := append([]string{seed}, tags...)
	for _, candidate := range candidates {
		v := strings.ToLower(candidate)
		switch {
		case strings.Contains(v, "apt"):
			return "apt"
		case strings.Contains(v, "ransom"):
			return "ransomware"
		case strings.Contains(v, "phish"):
			return "phishing"
		case strings.Contains(v, "botnet"):
			return "botnet"
		}
	}
	return ""
}

func parseFlexibleTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	if unixSeconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.Unix(unixSeconds, 0).UTC()
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}

func uniqueLowerStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
