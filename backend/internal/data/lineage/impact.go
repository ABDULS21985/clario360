package lineage

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/model"
)

type ImpactAnalyzer struct {
	builder *GraphBuilder
}

func NewImpactAnalyzer(builder *GraphBuilder) *ImpactAnalyzer {
	return &ImpactAnalyzer{builder: builder}
}

func (a *ImpactAnalyzer) Analyze(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID) (*model.ImpactAnalysis, error) {
	graph, err := a.builder.BuildFullGraph(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	state := newGraphState(graph.Nodes, graph.Edges)
	centerKey := nodeKey(entityType, entityID)
	center, ok := state.nodes[centerKey]
	if !ok {
		return nil, fmt.Errorf("lineage entity not found")
	}

	type queueItem struct {
		NodeID string
		Hops   int
		Path   []string
	}

	visited := map[string]int{centerKey: 0}
	queue := []queueItem{{NodeID: centerKey, Hops: 0, Path: []string{center.Name}}}
	direct := make([]model.ImpactedEntity, 0)
	indirect := make([]model.ImpactedEntity, 0)
	affectedSuites := make([]model.AffectedSuite, 0)
	highestClassification := classificationFromNode(center)
	seenSuites := map[string]struct{}{}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if item.Hops >= 10 {
			continue
		}
		for _, edge := range state.outgoing[item.NodeID] {
			nextNode := state.nodes[edge.Target]
			nextHops := item.Hops + 1
			if previous, seen := visited[nextNode.ID]; seen && previous <= nextHops {
				continue
			}
			visited[nextNode.ID] = nextHops
			path := append(append([]string(nil), item.Path...), nextNode.Name)
			queue = append(queue, queueItem{NodeID: nextNode.ID, Hops: nextHops, Path: path})

			classification := classificationFromNode(nextNode)
			highestClassification = maxClassification(highestClassification, classification)
			impacted := model.ImpactedEntity{
				Node:               *nextNode,
				HopDistance:        nextHops,
				PathDescription:    strings.Join(path, " -> "),
				DataClassification: classification,
			}
			if nextHops == 1 {
				direct = append(direct, impacted)
			} else {
				indirect = append(indirect, impacted)
			}

			if nextNode.Type == string(model.LineageEntitySuiteConsumer) {
				suite := affectedSuiteForNode(nextNode)
				key := suite.SuiteName + "|" + suite.Capability
				if _, ok := seenSuites[key]; !ok {
					seenSuites[key] = struct{}{}
					affectedSuites = append(affectedSuites, suite)
				}
			}
		}
	}

	severity := classificationSeverity(highestClassification)
	analysis := &model.ImpactAnalysis{
		Entity:             *center,
		DirectlyAffected:   direct,
		IndirectlyAffected: indirect,
		AffectedSuites:     affectedSuites,
		TotalAffected:      len(direct) + len(indirect),
		Severity:           severity,
		Summary:            buildImpactSummary(center.Name, len(direct)+len(indirect), affectedSuites, severity),
	}
	return analysis, nil
}

func classificationFromNode(node *model.LineageNode) string {
	if node == nil || node.Metadata == nil {
		return string(model.DataClassificationPublic)
	}
	for _, key := range []string{"data_classification", "classification"} {
		if value, ok := node.Metadata[key]; ok {
			if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
				return strings.ToLower(strings.TrimSpace(str))
			}
		}
	}
	return string(model.DataClassificationPublic)
}

func affectedSuiteForNode(node *model.LineageNode) model.AffectedSuite {
	lookup := map[string]model.AffectedSuite{
		"cyber-threat-enrichment": {SuiteName: "Cybersecurity", Capability: "Threat enrichment with external data", Impact: "Would lose external enrichment inputs", Severity: "high"},
		"cyber-dspm":              {SuiteName: "Cybersecurity", Capability: "Data Security Posture Management", Impact: "Would lose DSPM-backed data visibility", Severity: "high"},
		"lex-compliance":          {SuiteName: "Legal", Capability: "Compliance data aggregation", Impact: "Would lose compliance evidence inputs", Severity: "high"},
		"visus-executive":         {SuiteName: "Executive", Capability: "Executive dashboard KPIs", Impact: "Would lose executive data refreshes", Severity: "medium"},
		"acta-board":              {SuiteName: "Governance", Capability: "Board meeting data preparation", Impact: "Would lose board reporting inputs", Severity: "medium"},
	}
	key := strings.ToLower(strings.TrimSpace(node.Name))
	if suite, ok := lookup[key]; ok {
		return suite
	}
	return model.AffectedSuite{
		SuiteName:  "Platform",
		Capability: node.Name,
		Impact:     "Would lose a downstream consumer dependency",
		Severity:   classificationSeverity(classificationFromNode(node)),
	}
}

func buildImpactSummary(entityName string, total int, suites []model.AffectedSuite, severity string) string {
	names := make([]string, 0, len(suites))
	for _, suite := range suites {
		names = append(names, suite.SuiteName)
	}
	if len(names) == 0 {
		return fmt.Sprintf("If %s becomes unavailable, %d downstream entities would be affected. Impact severity: %s.", entityName, total, severity)
	}
	return fmt.Sprintf("If %s becomes unavailable, %d downstream entities would be affected, including %s. Impact severity: %s.", entityName, total, strings.Join(names, ", "), severity)
}
