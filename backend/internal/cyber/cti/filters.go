package cti

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Common pagination
// ---------------------------------------------------------------------------

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func NewPaginationMeta(page, perPage, total int) PaginationMeta {
	tp := total / perPage
	if total%perPage != 0 {
		tp++
	}
	if tp < 1 {
		tp = 1
	}
	return PaginationMeta{Page: page, PerPage: perPage, Total: total, TotalPages: tp}
}

// ---------------------------------------------------------------------------
// Threat event filters
// ---------------------------------------------------------------------------

type ThreatEventFilters struct {
	Search          *string
	Severities      []string
	Categories      []string
	EventTypes      []string
	OriginCountries []string
	TargetCountries []string
	TargetSectors   []string
	IOCType         *string
	IOCValue        *string
	IsFalsePositive *bool
	MinConfidence   *float64
	MaxConfidence   *float64
	FirstSeenFrom   *time.Time
	FirstSeenTo     *time.Time
	SourceID        *string
	Sort            string
	Order           string
	Page            int
	PerPage         int
}

func (f *ThreatEventFilters) SetDefaults() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 200 {
		f.PerPage = 25
	}
	if f.Sort == "" {
		f.Sort = "first_seen_at"
	}
	if f.Order == "" {
		f.Order = "desc"
	}
}

func (f *ThreatEventFilters) Validate() error {
	for _, s := range f.Severities {
		if !isValidSeverity(s) {
			return fmt.Errorf("invalid severity: %q", s)
		}
	}
	for _, et := range f.EventTypes {
		if !isValidEventType(et) {
			return fmt.Errorf("invalid event_type: %q", et)
		}
	}
	return nil
}

func ParseThreatEventFilters(r *http.Request) ThreatEventFilters {
	f := ThreatEventFilters{
		Search:          optStr(r, "search"),
		Severities:      strSlice(r, "severity"),
		Categories:      strSlice(r, "category"),
		EventTypes:      strSlice(r, "event_type"),
		OriginCountries: strSlice(r, "origin_country"),
		TargetCountries: strSlice(r, "target_country"),
		TargetSectors:   strSlice(r, "target_sector"),
		IOCType:         optStr(r, "ioc_type"),
		IOCValue:        optStr(r, "ioc_value"),
		IsFalsePositive: optBool(r, "is_false_positive"),
		MinConfidence:   optFloat(r, "min_confidence"),
		MaxConfidence:   optFloat(r, "max_confidence"),
		FirstSeenFrom:   optTime(r, "first_seen_from"),
		FirstSeenTo:     optTime(r, "first_seen_to"),
		SourceID:        optStr(r, "source_id"),
		Sort:            r.URL.Query().Get("sort"),
		Order:           r.URL.Query().Get("order"),
		Page:            intParam(r, "page", 1),
		PerPage:         intParam(r, "per_page", 25),
	}
	f.SetDefaults()
	return f
}

// ---------------------------------------------------------------------------
// Threat actor filters
// ---------------------------------------------------------------------------

type ThreatActorFilters struct {
	Search                *string
	ActorTypes            []string
	SophisticationLevels  []string
	IsActive              *bool
	Sort                  string
	Order                 string
	Page                  int
	PerPage               int
}

func (f *ThreatActorFilters) SetDefaults() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 200 {
		f.PerPage = 25
	}
	if f.Sort == "" {
		f.Sort = "risk_score"
	}
	if f.Order == "" {
		f.Order = "desc"
	}
}

func ParseThreatActorFilters(r *http.Request) ThreatActorFilters {
	f := ThreatActorFilters{
		Search:               optStr(r, "search"),
		ActorTypes:           strSlice(r, "actor_type"),
		SophisticationLevels: strSlice(r, "sophistication"),
		IsActive:             optBool(r, "is_active"),
		Sort:                 r.URL.Query().Get("sort"),
		Order:                r.URL.Query().Get("order"),
		Page:                 intParam(r, "page", 1),
		PerPage:              intParam(r, "per_page", 25),
	}
	f.SetDefaults()
	return f
}

// ---------------------------------------------------------------------------
// Campaign filters
// ---------------------------------------------------------------------------

type CampaignFilters struct {
	Search        *string
	Statuses      []string
	Severities    []string
	ActorID       *string
	FirstSeenFrom *time.Time
	FirstSeenTo   *time.Time
	Sort          string
	Order         string
	Page          int
	PerPage       int
}

func (f *CampaignFilters) SetDefaults() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 200 {
		f.PerPage = 25
	}
	if f.Sort == "" {
		f.Sort = "first_seen_at"
	}
	if f.Order == "" {
		f.Order = "desc"
	}
}

func ParseCampaignFilters(r *http.Request) CampaignFilters {
	f := CampaignFilters{
		Search:        optStr(r, "search"),
		Statuses:      strSlice(r, "status"),
		Severities:    strSlice(r, "severity"),
		ActorID:       optStr(r, "actor_id"),
		FirstSeenFrom: optTime(r, "first_seen_from"),
		FirstSeenTo:   optTime(r, "first_seen_to"),
		Sort:          r.URL.Query().Get("sort"),
		Order:         r.URL.Query().Get("order"),
		Page:          intParam(r, "page", 1),
		PerPage:       intParam(r, "per_page", 25),
	}
	f.SetDefaults()
	return f
}

// ---------------------------------------------------------------------------
// Brand abuse filters
// ---------------------------------------------------------------------------

type BrandAbuseFilters struct {
	BrandID         *string
	RiskLevels      []string
	AbuseTypes      []string
	TakedownStatuses []string
	Sort            string
	Order           string
	Page            int
	PerPage         int
}

func (f *BrandAbuseFilters) SetDefaults() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 200 {
		f.PerPage = 25
	}
	if f.Sort == "" {
		f.Sort = "first_detected_at"
	}
	if f.Order == "" {
		f.Order = "desc"
	}
}

func ParseBrandAbuseFilters(r *http.Request) BrandAbuseFilters {
	f := BrandAbuseFilters{
		BrandID:          optStr(r, "brand_id"),
		RiskLevels:       strSlice(r, "risk_level"),
		AbuseTypes:       strSlice(r, "abuse_type"),
		TakedownStatuses: strSlice(r, "takedown_status"),
		Sort:             r.URL.Query().Get("sort"),
		Order:            r.URL.Query().Get("order"),
		Page:             intParam(r, "page", 1),
		PerPage:          intParam(r, "per_page", 25),
	}
	f.SetDefaults()
	return f
}

// ---------------------------------------------------------------------------
// Simple list pagination
// ---------------------------------------------------------------------------

type ListParams struct {
	Page    int
	PerPage int
}

func ParseListParams(r *http.Request) ListParams {
	p := intParam(r, "page", 1)
	pp := intParam(r, "per_page", 25)
	if p < 1 {
		p = 1
	}
	if pp < 1 || pp > 200 {
		pp = 25
	}
	return ListParams{Page: p, PerPage: pp}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func optStr(r *http.Request, key string) *string {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return nil
	}
	return &v
}

func optBool(r *http.Request, key string) *bool {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return nil
	}
	b := v == "true" || v == "1"
	return &b
}

func optFloat(r *http.Request, key string) *float64 {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	return &f
}

func optTime(r *http.Request, key string) *time.Time {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil
	}
	return &t
}

func strSlice(r *http.Request, key string) []string {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func intParam(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func isValidSeverity(s string) bool {
	switch s {
	case "critical", "high", "medium", "low", "informational":
		return true
	}
	return false
}

func isValidEventType(s string) bool {
	switch s {
	case "indicator_sighting", "attack_attempt", "vulnerability_exploit", "malware_detection", "anomaly", "policy_violation":
		return true
	}
	return false
}
