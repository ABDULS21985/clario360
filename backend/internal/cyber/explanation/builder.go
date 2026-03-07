package explanation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/clario360/platform/internal/cyber/model"
)

// BuildExplanation constructs an explainable alert payload from a rule match.
func BuildExplanation(
	rule *model.DetectionRule,
	match model.RuleMatch,
	assets []*model.Asset,
	indicators []*model.ThreatIndicator,
) *model.AlertExplanation {
	var primaryAsset *model.Asset
	if len(assets) > 0 {
		primaryAsset = assets[0]
	}
	confidenceScore, confidenceFactors := ComputeConfidence(rule, match, primaryAsset)

	evidence := make([]model.AlertEvidence, 0, len(match.Events)*4)
	falsePositiveIndicators := make([]string, 0)
	for _, event := range match.Events {
		eventEvidence := eventEvidenceItems(event)
		evidence = append(evidence, eventEvidence...)
		falsePositiveIndicators = append(falsePositiveIndicators, DetectFalsePositiveIndicators(event, primaryAsset)...)
	}

	indicatorEvidence := make([]model.IndicatorEvidence, 0, len(indicators))
	for _, indicator := range indicators {
		indicatorEvidence = append(indicatorEvidence, model.IndicatorEvidence{
			Type:       indicator.Type,
			Value:      indicator.Value,
			Source:     indicator.Source,
			Confidence: indicator.Confidence,
		})
	}

	alert := &model.Alert{
		Severity:        ruleSeverity(rule),
		Title:           buildTitle(rule, indicators),
		ConfidenceScore: confidenceScore,
	}
	actions := GenerateRecommendedActions(alert, primaryAsset)

	matchedConditions := matchedConditions(match)
	summary := buildSummary(rule, match, primaryAsset, indicators)
	reason := buildReason(rule, matchedConditions, match)

	return &model.AlertExplanation{
		Summary:                 summary,
		Reason:                  reason,
		Evidence:                evidence,
		MatchedConditions:       matchedConditions,
		ConfidenceFactors:       confidenceFactors,
		RecommendedActions:      actions,
		FalsePositiveIndicators: uniqueStrings(falsePositiveIndicators),
		IndicatorMatches:        indicatorEvidence,
		Details: map[string]interface{}{
			"event_count":      len(match.Events),
			"confidence_score": confidenceScore,
			"match_details":    match.MatchDetails,
			"asset_count":      len(assets),
			"indicator_count":  len(indicators),
		},
	}
}

func eventEvidenceItems(event model.SecurityEvent) []model.AlertEvidence {
	evidence := []model.AlertEvidence{
		{
			Label:       "Event source",
			Field:       "source",
			Value:       event.Source,
			Description: "Originating telemetry source",
		},
		{
			Label:       "Event type",
			Field:       "type",
			Value:       event.Type,
			Description: "Normalized event type",
		},
	}
	if event.SourceIP != nil {
		evidence = append(evidence, model.AlertEvidence{
			Label:       "Source IP",
			Field:       "source_ip",
			Value:       *event.SourceIP,
			Description: "Observed source IP address",
		})
	}
	if event.DestIP != nil {
		evidence = append(evidence, model.AlertEvidence{
			Label:       "Destination IP",
			Field:       "dest_ip",
			Value:       *event.DestIP,
			Description: "Observed destination IP address",
		})
	}
	if event.DestPort != nil {
		evidence = append(evidence, model.AlertEvidence{
			Label:       "Destination port",
			Field:       "dest_port",
			Value:       *event.DestPort,
			Description: "Observed network destination port",
		})
	}
	if event.Process != nil && *event.Process != "" {
		evidence = append(evidence, model.AlertEvidence{
			Label:       "Process",
			Field:       "process",
			Value:       *event.Process,
			Description: "Observed executable or process name",
		})
	}
	if event.CommandLine != nil && *event.CommandLine != "" {
		evidence = append(evidence, model.AlertEvidence{
			Label:       "Command line",
			Field:       "command_line",
			Value:       *event.CommandLine,
			Description: "Observed command line arguments",
		})
	}
	return evidence
}

func matchedConditions(match model.RuleMatch) []string {
	values := make([]string, 0)
	switch typed := match.MatchDetails["matched_selection_names"].(type) {
	case []string:
		values = append(values, typed...)
	case []interface{}:
		for _, value := range typed {
			values = append(values, fmt.Sprintf("%v", value))
		}
	}
	sort.Strings(values)
	return uniqueStrings(values)
}

func buildSummary(rule *model.DetectionRule, match model.RuleMatch, asset *model.Asset, indicators []*model.ThreatIndicator) string {
	ruleName := "Indicator match"
	if rule != nil && rule.Name != "" {
		ruleName = rule.Name
	}
	target := "the environment"
	if asset != nil {
		target = asset.Name
	}
	if len(indicators) > 0 {
		return fmt.Sprintf("%s observed on %s with %d supporting indicator matches.", ruleName, target, len(indicators))
	}
	return fmt.Sprintf("%s matched %d event(s) affecting %s.", ruleName, len(match.Events), target)
}

func buildReason(rule *model.DetectionRule, matchedConditions []string, match model.RuleMatch) string {
	parts := make([]string, 0, 3)
	if rule != nil {
		parts = append(parts, fmt.Sprintf("Detection rule %q triggered", rule.Name))
	}
	if len(matchedConditions) > 0 {
		parts = append(parts, fmt.Sprintf("matched selections: %s", strings.Join(matchedConditions, ", ")))
	}
	if zScore, ok := match.MatchDetails["z_score"]; ok {
		parts = append(parts, fmt.Sprintf("statistical deviation z-score=%v", zScore))
	}
	if metricValue, ok := match.MatchDetails["metric_value"]; ok {
		parts = append(parts, fmt.Sprintf("threshold metric=%v", metricValue))
	}
	return strings.Join(parts, "; ")
}

func buildTitle(rule *model.DetectionRule, indicators []*model.ThreatIndicator) string {
	if rule != nil && rule.Name != "" {
		return rule.Name
	}
	if len(indicators) > 0 {
		return "Known malicious indicator connection"
	}
	return "Security detection"
}

func ruleSeverity(rule *model.DetectionRule) model.Severity {
	if rule == nil || rule.Severity == "" {
		return model.SeverityMedium
	}
	return rule.Severity
}
