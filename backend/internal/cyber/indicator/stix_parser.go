package indicator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

var (
	stixIPPattern        = regexp.MustCompile(`(?i)\[(?:ipv4-addr|ipv6-addr):value\s*=\s*'([^']+)'\]`)
	stixCIDRPattern      = regexp.MustCompile(`(?i)\[(?:ipv4-addr|ipv6-addr):value\s+ISSUBSET\s+'([^']+)'\]`)
	stixDomainPattern    = regexp.MustCompile(`(?i)\[domain-name:value\s*=\s*'([^']+)'\]`)
	stixURLPattern       = regexp.MustCompile(`(?i)\[url:value\s*=\s*'([^']+)'\]`)
	stixEmailPattern     = regexp.MustCompile(`(?i)\[email-addr:value\s*=\s*'([^']+)'\]`)
	stixMD5Pattern       = regexp.MustCompile(`(?i)\[file:hashes\.'MD5'\s*=\s*'([^']+)'\]`)
	stixSHA1Pattern      = regexp.MustCompile(`(?i)\[file:hashes\.'SHA-1'\s*=\s*'([^']+)'\]`)
	stixSHA256Pattern    = regexp.MustCompile(`(?i)\[file:hashes\.'SHA-256'\s*=\s*'([^']+)'\]`)
	stixUserAgentPattern = regexp.MustCompile(`(?i)\[network-traffic:extensions\.'http-request-ext'\.request_header\.'User-Agent'\s*=\s*'([^']+)'\]`)
)

// ParsedBundle is the normalized output of a STIX/TAXII import.
type ParsedBundle struct {
	Threats    []ParsedThreat
	Indicators []ParsedIndicator
}

// ParsedThreat captures a threat object and its external reference ID.
type ParsedThreat struct {
	ExternalID  string
	Name        string
	Description string
	Type        model.ThreatType
	Tags        []string
}

// ParsedIndicator captures one parsed indicator and any related threat external IDs.
type ParsedIndicator struct {
	Indicator        model.ThreatIndicator
	RelatedThreatIDs []string
}

// ParseSTIXBundle parses a STIX 2 bundle into normalized threat and indicator objects.
func ParseSTIXBundle(payload json.RawMessage, defaultSource string) (*ParsedBundle, error) {
	var bundle struct {
		Type    string                   `json:"type"`
		Objects []map[string]interface{} `json:"objects"`
	}
	if err := json.Unmarshal(payload, &bundle); err != nil {
		return nil, fmt.Errorf("parse stix bundle: %w", err)
	}
	if strings.ToLower(bundle.Type) != "bundle" {
		return nil, fmt.Errorf("expected stix bundle payload")
	}

	parsed := &ParsedBundle{
		Threats:    make([]ParsedThreat, 0),
		Indicators: make([]ParsedIndicator, 0),
	}
	threatsByID := make(map[string]ParsedThreat)
	indicatorThreats := make(map[string][]string)

	for _, object := range bundle.Objects {
		objectType, _ := object["type"].(string)
		switch objectType {
		case "malware", "threat-actor", "campaign":
			threat := parseThreatObject(object)
			if threat.ExternalID == "" || threat.Name == "" {
				continue
			}
			threatsByID[threat.ExternalID] = threat
		case "relationship":
			sourceRef, _ := object["source_ref"].(string)
			targetRef, _ := object["target_ref"].(string)
			relationshipType, _ := object["relationship_type"].(string)
			if relationshipType == "indicates" {
				indicatorThreats[sourceRef] = append(indicatorThreats[sourceRef], targetRef)
			}
		}
	}

	for _, object := range bundle.Objects {
		if objectType, _ := object["type"].(string); objectType != "indicator" {
			continue
		}
		parsedIndicator, ok := parseIndicatorObject(object, defaultSource)
		if !ok {
			continue
		}
		id, _ := object["id"].(string)
		parsedIndicator.RelatedThreatIDs = append(parsedIndicator.RelatedThreatIDs, indicatorThreats[id]...)
		parsed.Indicators = append(parsed.Indicators, parsedIndicator)
	}

	for _, threat := range threatsByID {
		parsed.Threats = append(parsed.Threats, threat)
	}
	return parsed, nil
}

func parseThreatObject(object map[string]interface{}) ParsedThreat {
	externalID, _ := object["id"].(string)
	name, _ := object["name"].(string)
	description, _ := object["description"].(string)
	labels := stringSlice(object["labels"])
	threatType := model.ThreatTypeOther
	for _, label := range labels {
		if model.ThreatType(label).IsValid() {
			threatType = model.ThreatType(label)
			break
		}
	}
	objectType, _ := object["type"].(string)
	if objectType == "malware" && threatType == model.ThreatTypeOther {
		threatType = model.ThreatTypeMalware
	}
	return ParsedThreat{
		ExternalID:  externalID,
		Name:        name,
		Description: description,
		Type:        threatType,
		Tags:        labels,
	}
}

func parseIndicatorObject(object map[string]interface{}, defaultSource string) (ParsedIndicator, bool) {
	pattern, _ := object["pattern"].(string)
	indicatorType, value, err := parseIndicatorPattern(pattern)
	if err != nil {
		return ParsedIndicator{}, false
	}

	confidence := 0.80
	if rawConfidence, ok := object["confidence"].(float64); ok {
		confidence = rawConfidence / 100.0
		if confidence > 1 {
			confidence = 1
		}
	}
	name, _ := object["name"].(string)
	description, _ := object["description"].(string)
	if description == "" {
		description = name
	}
	labels := stringSlice(object["labels"])
	severity := severityFromLabels(labels)
	expiresAt := parseSTIXTime(object["valid_until"])

	return ParsedIndicator{
		Indicator: model.ThreatIndicator{
			Type:        indicatorType,
			Value:       value,
			Description: description,
			Severity:    severity,
			Source:      sourceOrDefault(defaultSource),
			Confidence:  confidence,
			Active:      true,
			ExpiresAt:   expiresAt,
			Tags:        labels,
		},
	}, true
}

func parseIndicatorPattern(pattern string) (model.IndicatorType, string, error) {
	pattern = strings.TrimSpace(pattern)
	switch {
	case stixIPPattern.MatchString(pattern):
		return model.IndicatorTypeIP, stixIPPattern.FindStringSubmatch(pattern)[1], nil
	case stixCIDRPattern.MatchString(pattern):
		return model.IndicatorTypeCIDR, stixCIDRPattern.FindStringSubmatch(pattern)[1], nil
	case stixDomainPattern.MatchString(pattern):
		return model.IndicatorTypeDomain, stixDomainPattern.FindStringSubmatch(pattern)[1], nil
	case stixURLPattern.MatchString(pattern):
		return model.IndicatorTypeURL, stixURLPattern.FindStringSubmatch(pattern)[1], nil
	case stixEmailPattern.MatchString(pattern):
		return model.IndicatorTypeEmail, stixEmailPattern.FindStringSubmatch(pattern)[1], nil
	case stixMD5Pattern.MatchString(pattern):
		return model.IndicatorTypeHashMD5, stixMD5Pattern.FindStringSubmatch(pattern)[1], nil
	case stixSHA1Pattern.MatchString(pattern):
		return model.IndicatorTypeHashSHA1, stixSHA1Pattern.FindStringSubmatch(pattern)[1], nil
	case stixSHA256Pattern.MatchString(pattern):
		return model.IndicatorTypeHashSHA256, stixSHA256Pattern.FindStringSubmatch(pattern)[1], nil
	case stixUserAgentPattern.MatchString(pattern):
		return model.IndicatorTypeUserAgent, stixUserAgentPattern.FindStringSubmatch(pattern)[1], nil
	default:
		return "", "", fmt.Errorf("unsupported stix pattern %q", pattern)
	}
}

func stringSlice(value interface{}) []string {
	raw, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok && text != "" {
			result = append(result, text)
		}
	}
	return result
}

func severityFromLabels(labels []string) model.Severity {
	for _, label := range labels {
		if model.Severity(label).IsValid() {
			return model.Severity(label)
		}
	}
	return model.SeverityMedium
}

func parseSTIXTime(value interface{}) *time.Time {
	text, ok := value.(string)
	if !ok || text == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return nil
	}
	return &parsed
}

func sourceOrDefault(value string) string {
	if value == "" {
		return "stix_feed"
	}
	return value
}
