package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// OTXAdapter parses AlienVault OTX indicator responses and extracts indicators.
type OTXAdapter struct{}

func NewOTXAdapter() *OTXAdapter { return &OTXAdapter{} }

func (a *OTXAdapter) SourceType() string { return "otx" }

type otxEnvelope struct {
	Results   []otxIndicator `json:"results"`
	ID        string         `json:"id"`
	Indicator string         `json:"indicator"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Created   string         `json:"created"`
	Modified  string         `json:"modified"`
	Country   string         `json:"country_code"`
	City      string         `json:"city"`
	PulseInfo otxPulseInfo   `json:"pulse_info"`
}

type otxIndicator struct {
	ID          string       `json:"id"`
	Indicator   string       `json:"indicator"`
	Type        string       `json:"type"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Created     string       `json:"created"`
	Modified    string       `json:"modified"`
	Country     string       `json:"country_code"`
	City        string       `json:"city"`
	PulseInfo   otxPulseInfo `json:"pulse_info"`
}

type otxPulseInfo struct {
	Count  int        `json:"count"`
	Pulses []otxPulse `json:"pulses"`
}

type otxPulse struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Created     string   `json:"created"`
	Modified    string   `json:"modified"`
}

func (a *OTXAdapter) Parse(_ context.Context, raw []byte) ([]NormalizedIndicator, error) {
	var envelope otxEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("otx adapter: unmarshal: %w", err)
	}

	items := make([]otxIndicator, 0, len(envelope.Results)+1)
	if len(envelope.Results) > 0 {
		items = append(items, envelope.Results...)
	} else if strings.TrimSpace(envelope.Indicator) != "" {
		items = append(items, otxIndicator{
			ID:        envelope.ID,
			Indicator: envelope.Indicator,
			Type:      envelope.Type,
			Title:     envelope.Title,
			Created:   envelope.Created,
			Modified:  envelope.Modified,
			Country:   envelope.Country,
			City:      envelope.City,
			PulseInfo: envelope.PulseInfo,
		})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("otx adapter: no indicators found")
	}

	now := time.Now().UTC()
	out := make([]NormalizedIndicator, 0, len(items))
	for _, item := range items {
		iocType := mapOTXIOCType(item.Type)
		if iocType == "" || strings.TrimSpace(item.Indicator) == "" {
			continue
		}

		firstSeen := parseOTXTime(item.Created)
		if firstSeen.IsZero() {
			firstSeen = now
		}
		lastSeen := parseOTXTime(item.Modified)
		if lastSeen.IsZero() {
			lastSeen = firstSeen
		}

		tags := make([]string, 0)
		description := strings.TrimSpace(item.Description)
		category := ""
		for _, pulse := range item.PulseInfo.Pulses {
			if description == "" {
				description = strings.TrimSpace(pulse.Description)
			}
			tags = append(tags, pulse.Name)
			tags = append(tags, pulse.Tags...)
			if category == "" {
				category = inferCategory(strings.Join(pulse.Tags, " "), pulse.Tags)
			}
			if modified := parseOTXTime(pulse.Modified); !modified.IsZero() && modified.After(lastSeen) {
				lastSeen = modified
			}
		}

		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = fmt.Sprintf("OTX Indicator: %s", item.Indicator)
		}

		severity := "medium"
		if item.PulseInfo.Count > 0 || len(item.PulseInfo.Pulses) > 0 {
			severity = "high"
		}

		confidence := 0.65
		if item.PulseInfo.Count > 0 {
			confidence = 0.75
		}

		out = append(out, NormalizedIndicator{
			Title:             title,
			Description:       description,
			SeverityCode:      severity,
			CategoryCode:      category,
			ConfidenceScore:   confidence,
			IOCType:           iocType,
			IOCValue:          strings.TrimSpace(item.Indicator),
			OriginCountryCode: strings.ToLower(strings.TrimSpace(item.Country)),
			OriginCity:        strings.TrimSpace(item.City),
			ExternalRef:       strings.TrimSpace(item.ID),
			FirstSeen:         firstSeen,
			LastSeen:          lastSeen,
			Tags:              uniqueLowerStrings(tags),
		})
	}

	return out, nil
}

func mapOTXIOCType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ipv4", "ipv6", "ip":
		return "ip"
	case "domain", "hostname":
		return "domain"
	case "url":
		return "url"
	case "email":
		return "email"
	case "filehash-sha256", "sha256":
		return "hash_sha256"
	case "filehash-sha1", "sha1":
		return "hash_sha1"
	case "filehash-md5", "md5":
		return "hash_md5"
	default:
		return ""
	}
}

func parseOTXTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}
