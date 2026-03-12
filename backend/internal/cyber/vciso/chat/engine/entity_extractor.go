package engine

import (
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type EntityExtractor struct {
	now             func() time.Time
	alertIDPattern  *regexp.Regexp
	hashIDPattern   *regexp.Regexp
	ipPattern       *regexp.Regexp
	hostnamePattern *regexp.Regexp
	lastNDays       *regexp.Regexp
	sinceDate       *regexp.Regexp
	severityPattern *regexp.Regexp
	countPattern    *regexp.Regexp
	timePatterns    []timePattern
	frameworkMap    map[string]string
}

func NewEntityExtractor(now func() time.Time) *EntityExtractor {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &EntityExtractor{
		now:             now,
		alertIDPattern:  regexp.MustCompile(`(?i)\b([0-9a-f]{8}(?:-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})?)\b`),
		hashIDPattern:   regexp.MustCompile(`(?i)(#[0-9]+)`),
		ipPattern:       regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}\b`),
		hostnamePattern: regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9-]*(?:\.[a-zA-Z0-9-]+)*\b`),
		lastNDays:       regexp.MustCompile(`(?i)\blast\s+(\d+)\s+days?\b`),
		sinceDate:       regexp.MustCompile(`(?i)\bsince\s+(\d{4}-\d{2}-\d{2})\b`),
		severityPattern: regexp.MustCompile(`(?i)\b(critical|high|medium|low|info)\b`),
		countPattern:    regexp.MustCompile(`(?i)\b(?:top|last|first)\s+(\d+)\b|\b(\d+)\s+most\b`),
		timePatterns:    defaultTimePatterns(),
		frameworkMap: map[string]string{
			"iso 27001": "iso27001",
			"iso27001":  "iso27001",
			"iso":       "iso27001",
			"nca":       "nca_ecc",
			"nca ecc":   "nca_ecc",
			"sama":      "sama_csf",
			"sama csf":  "sama_csf",
			"nist":      "nist_csf",
			"nist csf":  "nist_csf",
			"soc2":      "soc2",
			"soc 2":     "soc2",
			"gdpr":      "gdpr",
			"hipaa":     "hipaa",
			"pci":       "pci_dss",
			"pci dss":   "pci_dss",
		},
	}
}

func (e *EntityExtractor) Extract(message string, intent string) map[string]string {
	normalized := normalizeMessage(message)
	out := map[string]string{}
	switch intent {
	case "alert_detail", "investigation_query", "remediation_query":
		if match := e.hashIDPattern.FindStringSubmatch(normalized); len(match) > 1 {
			out["alert_id"] = match[1]
		} else if match := e.alertIDPattern.FindStringSubmatch(normalized); len(match) > 1 {
			out["alert_id"] = match[1]
		}
	case "asset_lookup":
		if ip := e.extractIP(normalized); ip != "" {
			out["asset_ip"] = ip
		}
		if host := e.extractHostname(normalized); host != "" {
			out["asset_name"] = host
		}
	case "dashboard_build":
		out["description"] = e.extractDashboardDescription(message)
	}
	if intent == "alert_query" || intent == "trend_query" || intent == "vulnerability_query" || intent == "recommendation_query" || intent == "ueba_query" || intent == "pipeline_query" || intent == "compliance_query" {
		e.extractTimeRange(normalized, out)
		e.extractSeverity(normalized, out)
		e.extractCount(normalized, out)
		e.extractFramework(normalized, out)
	}
	return out
}

func (e *EntityExtractor) extractIP(message string) string {
	for _, candidate := range e.ipPattern.FindAllString(message, -1) {
		if addr, err := netip.ParseAddr(candidate); err == nil && addr.Is4() {
			return candidate
		}
	}
	return ""
}

func (e *EntityExtractor) extractHostname(message string) string {
	stopWords := map[string]struct{}{"the": {}, "and": {}, "show": {}, "tell": {}, "about": {}, "what": {}, "details": {}, "for": {}, "asset": {}, "server": {}, "host": {}, "device": {}, "info": {}, "status": {}}
	for _, candidate := range e.hostnamePattern.FindAllString(message, -1) {
		if len(candidate) < 3 {
			continue
		}
		if _, blocked := stopWords[candidate]; blocked {
			continue
		}
		if strings.Contains(candidate, ".") || strings.Contains(candidate, "-") || strings.ContainsAny(candidate, "0123456789") {
			return candidate
		}
	}
	return ""
}

func (e *EntityExtractor) extractTimeRange(message string, out map[string]string) {
	now := e.now()
	start := now.AddDate(0, 0, -7)
	end := now
	found := false
	for _, pattern := range e.timePatterns {
		if !pattern.Pattern.MatchString(message) {
			continue
		}
		found = true
		switch pattern.Name {
		case "today":
			start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case "yesterday":
			day := now.AddDate(0, 0, -1)
			start = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, now.Location())
			end = time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, now.Location())
		case "this_week":
			offset := (int(now.Weekday()) + 6) % 7
			weekStart := now.AddDate(0, 0, -offset)
			start = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, now.Location())
		case "last_week":
			offset := (int(now.Weekday()) + 6) % 7
			weekStart := now.AddDate(0, 0, -offset-7)
			start = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, now.Location())
			end = start.AddDate(0, 0, 7).Add(-time.Second)
		case "last_7_days":
			start = now.AddDate(0, 0, -7)
		case "this_month":
			start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		case "last_month":
			month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -1, 0)
			start = month
			end = start.AddDate(0, 1, 0).Add(-time.Second)
		case "last_30_days":
			start = now.AddDate(0, 0, -30)
		case "last_24_hours":
			start = now.Add(-24 * time.Hour)
		case "last_hour":
			start = now.Add(-time.Hour)
		}
		break
	}
	if match := e.lastNDays.FindStringSubmatch(message); len(match) > 1 {
		if days, err := strconv.Atoi(match[1]); err == nil && days > 0 {
			if days > 365 {
				days = 365
			}
			start = now.AddDate(0, 0, -days)
			found = true
		}
	}
	if match := e.sinceDate.FindStringSubmatch(message); len(match) > 1 {
		if ts, err := time.Parse("2006-01-02", match[1]); err == nil {
			start = ts
			found = true
		}
	}
	if !found {
		start = now.AddDate(0, 0, -7)
		end = now
	}
	out["start_time"] = start.UTC().Format(time.RFC3339)
	out["end_time"] = end.UTC().Format(time.RFC3339)
}

func (e *EntityExtractor) extractSeverity(message string, out map[string]string) {
	if strings.Contains(message, "all severities") || strings.Contains(message, "any severity") || strings.Contains(message, "all alerts") {
		return
	}
	seen := map[string]struct{}{}
	values := make([]string, 0, 2)
	for _, match := range e.severityPattern.FindAllStringSubmatch(message, -1) {
		value := strings.ToLower(match[1])
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	if len(values) > 0 {
		out["severity"] = strings.Join(values, ",")
	}
}

func (e *EntityExtractor) extractCount(message string, out map[string]string) {
	match := e.countPattern.FindStringSubmatch(message)
	if len(match) == 0 {
		out["count"] = "5"
		return
	}
	value := match[1]
	if value == "" {
		value = match[2]
	}
	count, err := strconv.Atoi(value)
	if err != nil {
		out["count"] = "5"
		return
	}
	if count > 100 {
		count = 100
	}
	if count <= 0 {
		count = 5
	}
	out["count"] = strconv.Itoa(count)
}

func (e *EntityExtractor) extractFramework(message string, out map[string]string) {
	for key, value := range e.frameworkMap {
		if strings.Contains(message, key) {
			out["framework"] = value
			return
		}
	}
}

func (e *EntityExtractor) extractDashboardDescription(message string) string {
	normalized := strings.TrimSpace(message)
	replacements := []string{"build me a", "build a", "create a", "make a", "generate a", "set up a", "dashboard", "view", "panel", "showing", "with"}
	lower := strings.ToLower(normalized)
	for _, replacement := range replacements {
		lower = strings.Replace(lower, replacement, "", 1)
	}
	lower = strings.Join(strings.Fields(lower), " ")
	if lower == "" {
		return "security overview"
	}
	return lower
}


