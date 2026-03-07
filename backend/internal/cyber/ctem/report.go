package ctem

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

func AssembleReport(
	assessment *model.CTEMAssessment,
	findings []*model.CTEMFinding,
	groups []*model.CTEMRemediationGroup,
	score model.ExposureScore,
) *model.CTEMReport {
	discovery := make(map[string]int)
	for _, finding := range findings {
		discovery[string(finding.Type)]++
	}
	return &model.CTEMReport{
		Assessment:        assessment,
		Scoping:           phaseResult(assessment, "scoping"),
		Discovery:         discovery,
		Findings:          findings,
		RemediationGroups: groups,
		ExposureScore:     score,
		ExecutiveSummary:  BuildExecutiveSummary(assessment, findings, groups, score),
		GeneratedAt:       time.Now().UTC(),
	}
}

func BuildExecutiveSummary(
	assessment *model.CTEMAssessment,
	findings []*model.CTEMFinding,
	groups []*model.CTEMRemediationGroup,
	score model.ExposureScore,
) string {
	topFindings := topCriticalFindings(findings, 3)
	topGroups := topRemediationGroups(groups, 3)
	parts := []string{
		fmt.Sprintf("Assessment %q completed with an exposure score of %.2f (%s).", assessment.Name, score.Score, score.Grade),
		fmt.Sprintf("%d findings were identified across %d scoped assets.", len(findings), assessment.ResolvedAssetCount),
	}
	if len(topFindings) > 0 {
		parts = append(parts, fmt.Sprintf("Highest priority findings: %s.", strings.Join(topFindings, "; ")))
	}
	if len(topGroups) > 0 {
		parts = append(parts, fmt.Sprintf("Recommended remediation focus: %s.", strings.Join(topGroups, "; ")))
	}
	return strings.Join(parts, " ")
}

func phaseResult(assessment *model.CTEMAssessment, phase string) json.RawMessage {
	progress, ok := assessment.Phases[phase]
	if !ok || len(progress.Result) == 0 {
		return json.RawMessage("{}")
	}
	return progress.Result
}

func topCriticalFindings(findings []*model.CTEMFinding, limit int) []string {
	sorted := append([]*model.CTEMFinding{}, findings...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].PriorityScore == sorted[j].PriorityScore {
			return severityWeight(sorted[i].Severity) > severityWeight(sorted[j].Severity)
		}
		return sorted[i].PriorityScore > sorted[j].PriorityScore
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	out := make([]string, 0, len(sorted))
	for _, finding := range sorted {
		out = append(out, fmt.Sprintf("%s (score %.2f)", finding.Title, finding.PriorityScore))
	}
	return out
}

func topRemediationGroups(groups []*model.CTEMRemediationGroup, limit int) []string {
	sorted := append([]*model.CTEMRemediationGroup{}, groups...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].MaxPriorityScore > sorted[j].MaxPriorityScore
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	out := make([]string, 0, len(sorted))
	for _, group := range sorted {
		out = append(out, fmt.Sprintf("%s (reduces score by ~%.2f)", group.Title, derefFloat(group.ScoreReduction)))
	}
	return out
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
