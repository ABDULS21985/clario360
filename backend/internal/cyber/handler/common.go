package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/service"
)

func actorFromRequest(r *http.Request) *service.Actor {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		return nil
	}
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return nil
	}
	return &service.Actor{
		UserID:    userID,
		UserName:  user.Email,
		UserEmail: user.Email,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}
}

func parseAlertListParams(r *http.Request) (*dto.AlertListParams, error) {
	q := r.URL.Query()
	params := &dto.AlertListParams{}
	if v := q.Get("search"); v != "" {
		params.Search = &v
	}
	params.Severities = splitQueryValues(q, "severity")
	params.Statuses = splitQueryValues(q, "status")
	if v := q.Get("assigned_to"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("invalid assigned_to: %w", err)
		}
		params.AssignedTo = &id
	}
	if v := q.Get("unassigned"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid unassigned: %w", err)
		}
		params.Unassigned = &b
	}
	if v := q.Get("asset_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("invalid asset_id: %w", err)
		}
		params.AssetID = &id
	}
	if v := q.Get("rule_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("invalid rule_id: %w", err)
		}
		params.RuleID = &id
	}
	if v := q.Get("mitre_technique_id"); v != "" {
		params.MITRETechniqueID = &v
	}
	if v := q.Get("mitre_tactic_id"); v != "" {
		params.MITRETacticID = &v
	}
	if v := q.Get("rule_type"); v != "" {
		params.RuleType = &v
	}
	if v := q.Get("min_confidence"); v != "" {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid min_confidence: %w", err)
		}
		params.MinConfidence = &value
	}
	if v := q.Get("max_confidence"); v != "" {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid max_confidence: %w", err)
		}
		params.MaxConfidence = &value
	}
	params.Tags = splitQueryValues(q, "tag")
	if v := q.Get("date_from"); v != "" {
		ts, err := parseFlexibleTime(v)
		if err != nil {
			return nil, fmt.Errorf("invalid date_from: %w", err)
		}
		params.DateFrom = &ts
	}
	if v := q.Get("date_to"); v != "" {
		ts, err := parseFlexibleTime(v)
		if err != nil {
			return nil, fmt.Errorf("invalid date_to: %w", err)
		}
		params.DateTo = &ts
	}
	params.Sort = q.Get("sort")
	params.Order = q.Get("order")
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	return params, nil
}

func parseRuleListParams(r *http.Request) *dto.RuleListParams {
	q := r.URL.Query()
	params := &dto.RuleListParams{
		Search:           stringValue(q.Get("search")),
		Types:            splitQueryValues(q, "type"),
		Severities:       splitQueryValues(q, "severity"),
		Tag:              stringValue(q.Get("tag")),
		MITRETacticID:    stringValue(q.Get("mitre_tactic_id")),
		MITRETechniqueID: stringValue(q.Get("mitre_technique_id")),
		Sort:             q.Get("sort"),
		Order:            q.Get("order"),
	}
	if v := q.Get("enabled"); v != "" {
		b, _ := strconv.ParseBool(v)
		params.Enabled = &b
	}
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	return params
}

func parseThreatListParams(r *http.Request) *dto.ThreatListParams {
	q := r.URL.Query()
	params := &dto.ThreatListParams{
		Search:     stringValue(q.Get("search")),
		Types:      splitQueryValues(q, "type"),
		Statuses:   splitQueryValues(q, "status"),
		Severities: splitQueryValues(q, "severity"),
		Sort:       q.Get("sort"),
		Order:      q.Get("order"),
	}
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	return params
}

func parseIndicatorListParams(r *http.Request) (*dto.IndicatorListParams, error) {
	q := r.URL.Query()
	params := &dto.IndicatorListParams{
		Types:      splitQueryValues(q, "type"),
		Sources:    splitQueryValues(q, "source"),
		Severities: splitQueryValues(q, "severity"),
		Search:     stringValue(q.Get("search")),
		Sort:       q.Get("sort"),
		Order:      q.Get("order"),
	}
	if v := q.Get("threat_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("invalid threat_id: %w", err)
		}
		params.ThreatID = &id
	}
	if v := q.Get("active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid active: %w", err)
		}
		params.Active = &b
	}
	if v := q.Get("linked"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid linked: %w", err)
		}
		params.Linked = &b
	}
	if v := q.Get("min_confidence"); v != "" {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid min_confidence: %w", err)
		}
		params.MinConfidence = &value
	}
	if v := q.Get("max_confidence"); v != "" {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid max_confidence: %w", err)
		}
		params.MaxConfidence = &value
	}
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	return params, nil
}

func parseRiskTrendParams(r *http.Request) *dto.RiskTrendParams {
	params := &dto.RiskTrendParams{}
	if v := r.URL.Query().Get("days"); v != "" {
		params.Days, _ = strconv.Atoi(v)
	}
	params.SetDefaults()
	return params
}

func parseDashboardTrendParams(r *http.Request) *dto.DashboardTrendParams {
	params := &dto.DashboardTrendParams{}
	if v := r.URL.Query().Get("days"); v != "" {
		params.Days, _ = strconv.Atoi(v)
	}
	params.SetDefaults()
	return params
}

func parseVulnerabilityQueryParams(r *http.Request) (*dto.VulnerabilityQueryParams, error) {
	q := r.URL.Query()
	params := &dto.VulnerabilityQueryParams{
		Severities: splitQueryValues(q, "severity"),
		Statuses:   splitQueryValues(q, "status"),
		Sort:       q.Get("sort"),
		Order:      q.Get("order"),
	}
	if v := q.Get("search"); v != "" {
		params.Search = &v
	}
	if v := q.Get("asset_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("invalid asset_id: %w", err)
		}
		params.AssetID = &id
	}
	if v := q.Get("asset_type"); v != "" {
		params.AssetType = &v
	}
	if v := q.Get("cve_id"); v != "" {
		params.CVEID = &v
	}
	if v := q.Get("source"); v != "" {
		params.Source = &v
	}
	if v := q.Get("detected_after"); v != "" {
		ts, err := parseFlexibleTime(v)
		if err != nil {
			return nil, fmt.Errorf("invalid detected_after: %w", err)
		}
		params.DetectedAfter = &ts
	}
	if v := q.Get("detected_before"); v != "" {
		ts, err := parseFlexibleTime(v)
		if err != nil {
			return nil, fmt.Errorf("invalid detected_before: %w", err)
		}
		params.DetectedBefore = &ts
	}
	if v := q.Get("min_cvss"); v != "" {
		score, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid min_cvss: %w", err)
		}
		params.MinCVSS = &score
	}
	if v := q.Get("has_exploit"); v != "" {
		flag, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid has_exploit: %w", err)
		}
		params.HasExploit = &flag
	}
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	params.SetDefaults()
	return params, params.Validate()
}

func parseFlexibleTime(value string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts, nil
	}
	return time.Parse("2006-01-02", value)
}

func stringValue(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
