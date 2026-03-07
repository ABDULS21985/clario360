package lineage

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/model"
)

type graphState struct {
	nodes    map[string]*model.LineageNode
	edges    []*model.LineageEdge
	outgoing map[string][]*model.LineageEdge
	incoming map[string][]*model.LineageEdge
}

func newGraphState(nodes []model.LineageNode, edges []model.LineageEdge) *graphState {
	state := &graphState{
		nodes:    make(map[string]*model.LineageNode, len(nodes)),
		edges:    make([]*model.LineageEdge, 0, len(edges)),
		outgoing: make(map[string][]*model.LineageEdge),
		incoming: make(map[string][]*model.LineageEdge),
	}
	for i := range nodes {
		node := nodes[i]
		state.nodes[node.ID] = &node
	}
	for i := range edges {
		edge := edges[i]
		state.edges = append(state.edges, &edge)
		state.outgoing[edge.Source] = append(state.outgoing[edge.Source], &edge)
		state.incoming[edge.Target] = append(state.incoming[edge.Target], &edge)
	}
	return state
}

func nodeKey(entityType model.LineageEntityType, entityID uuid.UUID) string {
	return fmt.Sprintf("%s:%s", entityType, entityID)
}

func parseNodeKey(value string) (model.LineageEntityType, uuid.UUID, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", uuid.Nil, fmt.Errorf("invalid lineage node key")
	}
	entityType := model.LineageEntityType(parts[0])
	if !entityType.IsValid() {
		return "", uuid.Nil, fmt.Errorf("invalid lineage entity type")
	}
	entityID, err := uuid.Parse(parts[1])
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("invalid lineage entity id")
	}
	return entityType, entityID, nil
}

func sortedNodeIDs(nodes map[string]*model.LineageNode) []string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func mergeStringSlices(values ...[]string) []string {
	seen := make(map[string]struct{})
	merged := make([]string, 0)
	for _, list := range values {
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			merged = append(merged, item)
		}
	}
	sort.Strings(merged)
	return merged
}

func classificationSeverity(classification string) string {
	switch strings.ToLower(strings.TrimSpace(classification)) {
	case string(model.DataClassificationRestricted):
		return "critical"
	case string(model.DataClassificationConfidential):
		return "high"
	case string(model.DataClassificationInternal):
		return "medium"
	default:
		return "low"
	}
}

func classificationRank(classification string) int {
	switch strings.ToLower(strings.TrimSpace(classification)) {
	case string(model.DataClassificationRestricted):
		return 4
	case string(model.DataClassificationConfidential):
		return 3
	case string(model.DataClassificationInternal):
		return 2
	default:
		return 1
	}
}

func maxClassification(values ...string) string {
	best := string(model.DataClassificationPublic)
	bestRank := 1
	for _, value := range values {
		rank := classificationRank(value)
		if rank > bestRank {
			bestRank = rank
			best = strings.ToLower(strings.TrimSpace(value))
		}
	}
	return best
}

