package adapters

import (
	"context"
	"time"
)

// NormalizedIndicator is the intermediate format between a raw feed and a CTI ThreatEvent.
type NormalizedIndicator struct {
	Title             string
	Description       string
	SeverityCode      string
	CategoryCode      string
	ConfidenceScore   float64
	IOCType           string
	IOCValue          string
	OriginCountryCode string
	OriginCity        string
	Latitude          float64
	Longitude         float64
	TargetSectorCode  string
	MITRETechniques   []string
	ExternalRef       string // source's own ID
	FirstSeen         time.Time
	LastSeen          time.Time
	Tags              []string
}

// FeedAdapter parses raw feed bytes into normalized indicators.
type FeedAdapter interface {
	SourceType() string
	Parse(ctx context.Context, raw []byte) ([]NormalizedIndicator, error)
}
