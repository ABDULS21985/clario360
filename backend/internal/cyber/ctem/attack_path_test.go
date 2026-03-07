package ctem

import (
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestDiscoverAttackPathsSimple(t *testing.T) {
	entryID := uuid.New()
	midID := uuid.New()
	targetID := uuid.New()
	vulnID := uuid.New()

	paths := DiscoverAttackPaths(
		[]*model.Asset{
			{ID: entryID, Name: "edge", Tags: []string{"internet-facing"}, Criticality: model.CriticalityHigh},
			{ID: midID, Name: "app", Criticality: model.CriticalityMedium},
			{ID: targetID, Name: "db", Type: model.AssetTypeDatabase, Criticality: model.CriticalityCritical},
		},
		[]*model.AssetRelationship{
			{SourceAssetID: entryID, TargetAssetID: midID, RelationshipType: model.RelationshipConnectsTo},
			{SourceAssetID: midID, TargetAssetID: targetID, RelationshipType: model.RelationshipDependsOn},
		},
		map[uuid.UUID][]*model.Vulnerability{
			midID: {{ID: vulnID, Severity: "high"}},
		},
	)

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if got := len(paths[0].Hops); got != 3 {
		t.Fatalf("expected 3 hops, got %d", got)
	}
	if paths[0].Score <= 0 {
		t.Fatalf("expected positive path score, got %.2f", paths[0].Score)
	}
}

func TestDiscoverAttackPathsNoPath(t *testing.T) {
	entryID := uuid.New()
	targetID := uuid.New()

	paths := DiscoverAttackPaths(
		[]*model.Asset{
			{ID: entryID, Name: "edge", Tags: []string{"dmz"}},
			{ID: targetID, Name: "db", Type: model.AssetTypeDatabase, Criticality: model.CriticalityCritical},
		},
		nil,
		nil,
	)

	if len(paths) != 0 {
		t.Fatalf("expected 0 paths, got %d", len(paths))
	}
}

func TestDiscoverAttackPathsMaxLength(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	assets := []*model.Asset{
		{ID: ids[0], Name: "a", Tags: []string{"public"}},
		{ID: ids[1], Name: "b"},
		{ID: ids[2], Name: "c"},
		{ID: ids[3], Name: "d"},
		{ID: ids[4], Name: "e"},
		{ID: ids[5], Name: "f", Type: model.AssetTypeDatabase, Criticality: model.CriticalityCritical},
	}
	rels := []*model.AssetRelationship{
		{SourceAssetID: ids[0], TargetAssetID: ids[1], RelationshipType: model.RelationshipConnectsTo},
		{SourceAssetID: ids[1], TargetAssetID: ids[2], RelationshipType: model.RelationshipConnectsTo},
		{SourceAssetID: ids[2], TargetAssetID: ids[3], RelationshipType: model.RelationshipConnectsTo},
		{SourceAssetID: ids[3], TargetAssetID: ids[4], RelationshipType: model.RelationshipConnectsTo},
		{SourceAssetID: ids[4], TargetAssetID: ids[5], RelationshipType: model.RelationshipConnectsTo},
	}
	vulns := map[uuid.UUID][]*model.Vulnerability{
		ids[4]: {{ID: uuid.New(), Severity: "critical"}},
	}

	paths := DiscoverAttackPaths(assets, rels, vulns)

	if len(paths) != 0 {
		t.Fatalf("expected no paths because path exceeds 5 hops, got %d", len(paths))
	}
}

func TestScoreAttackPathPrefersEarlierCriticalHop(t *testing.T) {
	firstHopScore := scoreAttackPath([]AttackPathHop{
		{AssetID: uuid.New()},
		{AssetID: uuid.New(), VulnSeverity: severityPtr("critical")},
		{AssetID: uuid.New()},
	})
	lateHopScore := scoreAttackPath([]AttackPathHop{
		{AssetID: uuid.New()},
		{AssetID: uuid.New()},
		{AssetID: uuid.New()},
		{AssetID: uuid.New()},
		{AssetID: uuid.New(), VulnSeverity: severityPtr("critical")},
	})

	if firstHopScore <= lateHopScore {
		t.Fatalf("expected earlier critical hop score %.2f to exceed late-hop score %.2f", firstHopScore, lateHopScore)
	}
}

func TestDedupeAttackPathsKeepsLongestSuperset(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	d := uuid.New()
	paths := dedupeAttackPaths([]AttackPath{
		{Hops: []AttackPathHop{{AssetID: a}, {AssetID: b}, {AssetID: c}}},
		{Hops: []AttackPathHop{{AssetID: a}, {AssetID: b}, {AssetID: c}, {AssetID: d}}},
	})

	if len(paths) != 1 {
		t.Fatalf("expected 1 deduped path, got %d", len(paths))
	}
	if got := len(paths[0].Hops); got != 4 {
		t.Fatalf("expected longest path to remain, got %d hops", got)
	}
}

func severityPtr(value string) *string {
	return &value
}
