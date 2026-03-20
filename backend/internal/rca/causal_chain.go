package rca

import (
	"sort"
	"strings"

	"github.com/clario360/platform/internal/cyber/mitre"
)

// ChainBuilder constructs causal chains from correlated timeline events.
type ChainBuilder struct{}

// NewChainBuilder creates a causal chain builder.
func NewChainBuilder() *ChainBuilder {
	return &ChainBuilder{}
}

// BuildFromTimeline constructs a causal chain from a set of timeline events.
// It correlates events by shared attributes (IP, user, assets) and orders by
// MITRE kill chain phase for security events, or by timestamp for other types.
func (b *ChainBuilder) BuildFromTimeline(events []TimelineEvent, analysisType AnalysisType) []CausalStep {
	if len(events) == 0 {
		return nil
	}

	switch analysisType {
	case AnalysisTypeSecurity:
		return b.buildSecurityChain(events)
	case AnalysisTypePipeline:
		return b.buildPipelineChain(events)
	case AnalysisTypeQuality:
		return b.buildQualityChain(events)
	default:
		return b.buildGenericChain(events)
	}
}

// buildSecurityChain builds a causal chain using MITRE ATT&CK kill chain ordering.
func (b *ChainBuilder) buildSecurityChain(events []TimelineEvent) []CausalStep {
	// Group events by MITRE kill chain phase
	phased := make(map[string][]TimelineEvent)
	var unphased []TimelineEvent

	for _, evt := range events {
		phase := resolveKillChainPhase(evt)
		if phase != "" {
			evt.MITREPhase = phase
			phased[phase] = append(phased[phase], evt)
		} else {
			unphased = append(unphased, evt)
		}
	}

	// Order phases by kill chain position
	orderedPhases := orderByKillChain(phased)

	var chain []CausalStep
	order := 1

	for _, phase := range orderedPhases {
		phaseEvents := phased[phase]
		// Sort events within a phase by timestamp
		sort.Slice(phaseEvents, func(i, j int) bool {
			return phaseEvents[i].Timestamp.Before(phaseEvents[j].Timestamp)
		})

		for _, evt := range phaseEvents {
			step := CausalStep{
				Order:       order,
				EventID:     evt.ID,
				EventType:   evt.Type,
				Source:      evt.Source,
				Description: evt.Summary,
				Timestamp:   evt.Timestamp,
				Severity:    evt.Severity,
				MITREPhase:  phase,
				MITRETechID: evt.MITRETechID,
				Evidence:    buildEvidence(evt),
			}
			chain = append(chain, step)
			order++
		}
	}

	// Add unphased events at the end, sorted by time
	sort.Slice(unphased, func(i, j int) bool {
		return unphased[i].Timestamp.Before(unphased[j].Timestamp)
	})
	for _, evt := range unphased {
		chain = append(chain, CausalStep{
			Order:       order,
			EventID:     evt.ID,
			EventType:   evt.Type,
			Source:      evt.Source,
			Description: evt.Summary,
			Timestamp:   evt.Timestamp,
			Severity:    evt.Severity,
			Evidence:    buildEvidence(evt),
		})
		order++
	}

	// Mark the earliest kill chain step as root cause
	if len(chain) > 0 {
		chain[0].IsRootCause = true
	}

	return chain
}

// buildPipelineChain builds a temporal causal chain for pipeline failures.
func (b *ChainBuilder) buildPipelineChain(events []TimelineEvent) []CausalStep {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	var chain []CausalStep
	rootCauseIdx := -1

	for i, evt := range events {
		step := CausalStep{
			Order:       i + 1,
			EventID:     evt.ID,
			EventType:   evt.Type,
			Source:      evt.Source,
			Description: evt.Summary,
			Timestamp:   evt.Timestamp,
			Severity:    evt.Severity,
			Evidence:    buildEvidence(evt),
		}

		// The first failure event is likely the root cause
		if rootCauseIdx == -1 && (evt.Type == "failed" || evt.Severity == "high") {
			rootCauseIdx = i
			step.IsRootCause = true
		}

		chain = append(chain, step)
	}

	// If no failure found, mark the first event
	if rootCauseIdx == -1 && len(chain) > 0 {
		chain[0].IsRootCause = true
	}

	return chain
}

// buildQualityChain builds a causal chain for data quality issues.
func (b *ChainBuilder) buildQualityChain(events []TimelineEvent) []CausalStep {
	return b.buildPipelineChain(events) // Same temporal logic
}

// buildGenericChain builds a simple temporal chain.
func (b *ChainBuilder) buildGenericChain(events []TimelineEvent) []CausalStep {
	return b.buildPipelineChain(events)
}

// resolveKillChainPhase maps an event to a MITRE kill chain phase.
func resolveKillChainPhase(evt TimelineEvent) string {
	if evt.MITREPhase != "" {
		return evt.MITREPhase
	}

	if evt.MITRETechID != "" {
		tech, ok := mitre.TechniqueByID(evt.MITRETechID)
		if ok && len(tech.TacticIDs) > 0 {
			// Return the first (primary) tactic
			tactic, ok := mitre.TacticByID(tech.TacticIDs[0])
			if ok {
				return tactic.ShortName
			}
		}
	}

	return ""
}

// killChainOrder defines the ordering of kill chain phases.
var killChainOrder = map[string]int{
	"reconnaissance":       1,
	"resource-development": 2,
	"initial-access":       3,
	"execution":            4,
	"persistence":          5,
	"privilege-escalation": 6,
	"defense-evasion":      7,
	"credential-access":    8,
	"discovery":            9,
	"lateral-movement":     10,
	"collection":           11,
	"command-and-control":  12,
	"exfiltration":         13,
	"impact":               14,
}

// orderByKillChain sorts phase names by their kill chain position.
func orderByKillChain(phased map[string][]TimelineEvent) []string {
	phases := make([]string, 0, len(phased))
	for phase := range phased {
		phases = append(phases, phase)
	}
	sort.Slice(phases, func(i, j int) bool {
		oi := killChainOrder[strings.ToLower(phases[i])]
		oj := killChainOrder[strings.ToLower(phases[j])]
		return oi < oj
	})
	return phases
}

func buildEvidence(evt TimelineEvent) []Evidence {
	var evidence []Evidence
	if evt.SourceIP != "" {
		evidence = append(evidence, Evidence{
			Label: "Source IP", Field: "source_ip", Value: evt.SourceIP,
			Description: "IP address associated with this event",
		})
	}
	if evt.UserID != "" {
		evidence = append(evidence, Evidence{
			Label: "User", Field: "user_id", Value: evt.UserID,
			Description: "User associated with this event",
		})
	}
	if evt.AssetID != "" {
		evidence = append(evidence, Evidence{
			Label: "Asset", Field: "asset_id", Value: evt.AssetID,
			Description: "Asset affected by this event",
		})
	}
	if evt.MITRETechID != "" {
		evidence = append(evidence, Evidence{
			Label: "MITRE Technique", Field: "mitre_technique_id", Value: evt.MITRETechID,
			Description: "MITRE ATT&CK technique identifier",
		})
	}
	return evidence
}

// CorrelateEvents groups related events by shared attributes.
func CorrelateEvents(events []TimelineEvent) map[string][]TimelineEvent {
	groups := make(map[string][]TimelineEvent)

	for _, evt := range events {
		if evt.SourceIP != "" {
			groups["ip:"+evt.SourceIP] = append(groups["ip:"+evt.SourceIP], evt)
		}
		if evt.UserID != "" {
			groups["user:"+evt.UserID] = append(groups["user:"+evt.UserID], evt)
		}
		if evt.AssetID != "" {
			groups["asset:"+evt.AssetID] = append(groups["asset:"+evt.AssetID], evt)
		}
	}

	return groups
}
