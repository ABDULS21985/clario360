package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GenericJSONAdapter parses a JSON array of indicator objects.
type GenericJSONAdapter struct{}

func NewGenericJSONAdapter() *GenericJSONAdapter { return &GenericJSONAdapter{} }

func (a *GenericJSONAdapter) SourceType() string { return "json_generic" }

type jsonIndicator struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Severity    string   `json:"severity"`
	Category    string   `json:"category,omitempty"`
	Confidence  float64  `json:"confidence"`
	IOCType     string   `json:"ioc_type"`
	IOCValue    string   `json:"ioc_value"`
	Country     string   `json:"country,omitempty"`
	City        string   `json:"city,omitempty"`
	Sector      string   `json:"sector,omitempty"`
	MITRE       []string `json:"mitre_techniques,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	FirstSeen   string   `json:"first_seen,omitempty"`
}

func (a *GenericJSONAdapter) Parse(_ context.Context, raw []byte) ([]NormalizedIndicator, error) {
	var items []jsonIndicator
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("json adapter: unmarshal: %w", err)
	}

	now := time.Now().UTC()
	out := make([]NormalizedIndicator, 0, len(items))
	for _, it := range items {
		firstSeen := now
		if it.FirstSeen != "" {
			if t, err := time.Parse(time.RFC3339, it.FirstSeen); err == nil {
				firstSeen = t
			}
		}
		out = append(out, NormalizedIndicator{
			Title:             it.Title,
			Description:       it.Description,
			SeverityCode:      it.Severity,
			CategoryCode:      it.Category,
			ConfidenceScore:   it.Confidence,
			IOCType:           it.IOCType,
			IOCValue:          it.IOCValue,
			OriginCountryCode: it.Country,
			OriginCity:        it.City,
			TargetSectorCode:  it.Sector,
			MITRETechniques:   it.MITRE,
			ExternalRef:       it.ID,
			FirstSeen:         firstSeen,
			LastSeen:          now,
			Tags:              it.Tags,
		})
	}
	return out, nil
}
