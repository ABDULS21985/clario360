package ctem

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

type AttackPath struct {
	Hops     []AttackPathHop `json:"hops"`
	Score    float64         `json:"score"`
	EntryID  uuid.UUID       `json:"entry_asset_id"`
	TargetID uuid.UUID       `json:"target_asset_id"`
}

type AttackPathHop struct {
	AssetID         uuid.UUID  `json:"asset_id"`
	AssetName       string     `json:"asset_name"`
	VulnerabilityID *uuid.UUID `json:"vulnerability_id,omitempty"`
	VulnSeverity    *string    `json:"vuln_severity,omitempty"`
	RelationType    string     `json:"relation_type"`
}

type graphEdge struct {
	To           uuid.UUID
	RelationType string
}

func DiscoverAttackPaths(assets []*model.Asset, relationships []*model.AssetRelationship, vulnerabilitiesByAsset map[uuid.UUID][]*model.Vulnerability) []AttackPath {
	assetIndex := make(map[uuid.UUID]*model.Asset, len(assets))
	adjacency := make(map[uuid.UUID][]graphEdge)
	entries := make([]uuid.UUID, 0)
	targets := make([]uuid.UUID, 0)

	for _, asset := range assets {
		assetIndex[asset.ID] = asset
		if containsAny(asset.Tags, "internet-facing", "dmz", "public") {
			entries = append(entries, asset.ID)
		}
		if asset.Criticality == model.CriticalityCritical || asset.Type == model.AssetTypeDatabase {
			targets = append(targets, asset.ID)
		}
	}

	for _, rel := range relationships {
		adjacency[rel.SourceAssetID] = append(adjacency[rel.SourceAssetID], graphEdge{
			To:           rel.TargetAssetID,
			RelationType: string(rel.RelationshipType),
		})
	}

	paths := make([]AttackPath, 0)
	for _, entry := range entries {
		for _, target := range targets {
			if entry == target {
				continue
			}
			paths = append(paths, findPaths(entry, target, assetIndex, adjacency, vulnerabilitiesByAsset)...)
		}
	}

	paths = dedupeAttackPaths(paths)
	sort.Slice(paths, func(i, j int) bool {
		if paths[i].Score == paths[j].Score {
			return len(paths[i].Hops) > len(paths[j].Hops)
		}
		return paths[i].Score > paths[j].Score
	})
	if len(paths) > 100 {
		paths = paths[:100]
	}
	return paths
}

func findPaths(entry, target uuid.UUID, assetIndex map[uuid.UUID]*model.Asset, adjacency map[uuid.UUID][]graphEdge, vulnerabilitiesByAsset map[uuid.UUID][]*model.Vulnerability) []AttackPath {
	type queueItem struct {
		Node uuid.UUID
		Path []AttackPathHop
	}

	initialHop := AttackPathHop{
		AssetID:   entry,
		AssetName: safeAssetName(assetIndex[entry]),
	}
	queue := []queueItem{{Node: entry, Path: []AttackPathHop{initialHop}}}
	paths := make([]AttackPath, 0)

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if len(item.Path) > 5 {
			continue
		}
		if item.Node == target {
			score := scoreAttackPath(item.Path)
			if containsCriticalOrHighHop(item.Path) {
				paths = append(paths, AttackPath{
					Hops:     item.Path,
					Score:    score,
					EntryID:  entry,
					TargetID: target,
				})
			}
			continue
		}

		visited := make(map[uuid.UUID]bool, len(item.Path))
		for _, hop := range item.Path {
			visited[hop.AssetID] = true
		}

		for _, edge := range adjacency[item.Node] {
			if visited[edge.To] {
				continue
			}
			nextHop := AttackPathHop{
				AssetID:      edge.To,
				AssetName:    safeAssetName(assetIndex[edge.To]),
				RelationType: edge.RelationType,
			}
			if vuln := highestSeverityVulnerability(vulnerabilitiesByAsset[edge.To]); vuln != nil {
				nextHop.VulnerabilityID = &vuln.ID
				severity := vuln.Severity
				nextHop.VulnSeverity = &severity
			}
			nextPath := append(append([]AttackPathHop{}, item.Path...), nextHop)
			queue = append(queue, queueItem{Node: edge.To, Path: nextPath})
		}
	}

	return paths
}

func scoreAttackPath(hops []AttackPathHop) float64 {
	total := 0.0
	for index, hop := range hops {
		if hop.VulnSeverity == nil {
			continue
		}
		total += severityWeight(*hop.VulnSeverity) * hopWeight(index)
	}
	return round2(total)
}

func containsCriticalOrHighHop(hops []AttackPathHop) bool {
	for _, hop := range hops {
		if hop.VulnSeverity == nil {
			continue
		}
		if strings.EqualFold(*hop.VulnSeverity, "critical") || strings.EqualFold(*hop.VulnSeverity, "high") {
			return true
		}
	}
	return false
}

func dedupeAttackPaths(paths []AttackPath) []AttackPath {
	type keyedPath struct {
		key  string
		path AttackPath
	}
	keyed := make([]keyedPath, 0, len(paths))
	for _, path := range paths {
		ids := make([]string, 0, len(path.Hops))
		for _, hop := range path.Hops {
			ids = append(ids, hop.AssetID.String())
		}
		keyed = append(keyed, keyedPath{key: strings.Join(ids, "->"), path: path})
	}

	sort.Slice(keyed, func(i, j int) bool {
		return len(keyed[i].path.Hops) > len(keyed[j].path.Hops)
	})

	filtered := make([]AttackPath, 0, len(keyed))
	for _, candidate := range keyed {
		keep := true
		for _, existing := range filtered {
			if isSubsetPath(candidate.path, existing) {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, candidate.path)
		}
	}
	return filtered
}

func isSubsetPath(candidate, existing AttackPath) bool {
	if len(candidate.Hops) > len(existing.Hops) {
		return false
	}
	candidateIDs := make([]string, 0, len(candidate.Hops))
	for _, hop := range candidate.Hops {
		candidateIDs = append(candidateIDs, hop.AssetID.String())
	}
	existingIDs := make([]string, 0, len(existing.Hops))
	for _, hop := range existing.Hops {
		existingIDs = append(existingIDs, hop.AssetID.String())
	}
	return strings.Contains(strings.Join(existingIDs, ","), strings.Join(candidateIDs, ","))
}

func highestSeverityVulnerability(vulns []*model.Vulnerability) *model.Vulnerability {
	if len(vulns) == 0 {
		return nil
	}
	best := vulns[0]
	for _, vuln := range vulns[1:] {
		if severityWeight(vuln.Severity) > severityWeight(best.Severity) {
			best = vuln
		}
	}
	return best
}

func safeAssetName(asset *model.Asset) string {
	if asset == nil {
		return "unknown"
	}
	return asset.Name
}

func AttackPathToJSON(path AttackPath) json.RawMessage {
	payload, _ := json.Marshal(path.Hops)
	return payload
}
