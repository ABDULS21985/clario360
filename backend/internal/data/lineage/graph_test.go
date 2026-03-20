package lineage

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/model"
)

func TestFinalizeGraphDepthAndStats(t *testing.T) {
	sourceID := uuid.New()
	modelID := uuid.New()
	pipelineID := uuid.New()
	reportID := uuid.New()

	nodes := map[string]model.LineageNode{
		nodeKey(model.LineageEntityDataSource, sourceID): {ID: nodeKey(model.LineageEntityDataSource, sourceID), Type: string(model.LineageEntityDataSource), EntityID: sourceID, Name: "Source"},
		nodeKey(model.LineageEntityDataModel, modelID):   {ID: nodeKey(model.LineageEntityDataModel, modelID), Type: string(model.LineageEntityDataModel), EntityID: modelID, Name: "Model", Metadata: map[string]any{"classification": "restricted"}},
		nodeKey(model.LineageEntityPipeline, pipelineID): {ID: nodeKey(model.LineageEntityPipeline, pipelineID), Type: string(model.LineageEntityPipeline), EntityID: pipelineID, Name: "Pipeline"},
		nodeKey(model.LineageEntityReport, reportID):     {ID: nodeKey(model.LineageEntityReport, reportID), Type: string(model.LineageEntityReport), EntityID: reportID, Name: "Report"},
	}
	edges := []model.LineageEdge{
		{ID: uuid.New(), Source: nodeKey(model.LineageEntityDataSource, sourceID), Target: nodeKey(model.LineageEntityDataModel, modelID), Relationship: string(model.LineageRelationshipFeeds), Active: true, LastSeenAt: time.Now()},
		{ID: uuid.New(), Source: nodeKey(model.LineageEntityDataModel, modelID), Target: nodeKey(model.LineageEntityPipeline, pipelineID), Relationship: string(model.LineageRelationshipTransformsInto), Active: true, LastSeenAt: time.Now()},
		{ID: uuid.New(), Source: nodeKey(model.LineageEntityPipeline, pipelineID), Target: nodeKey(model.LineageEntityReport, reportID), Relationship: string(model.LineageRelationshipReportedIn), Active: true, LastSeenAt: time.Now()},
	}

	graph := finalizeGraph(nodes, edges)
	if graph.Stats.NodeCount != 4 {
		t.Fatalf("NodeCount = %d, want 4", graph.Stats.NodeCount)
	}
	if graph.Stats.EdgeCount != 3 {
		t.Fatalf("EdgeCount = %d, want 3", graph.Stats.EdgeCount)
	}
	if graph.Stats.MaxDepth != 3 {
		t.Fatalf("MaxDepth = %d, want 3", graph.Stats.MaxDepth)
	}
	if graph.Stats.SourceCount != 1 {
		t.Fatalf("SourceCount = %d, want 1", graph.Stats.SourceCount)
	}
	if graph.Stats.ConsumerCount != 1 {
		t.Fatalf("ConsumerCount = %d, want 1", graph.Stats.ConsumerCount)
	}

	var report model.LineageNode
	for _, node := range graph.Nodes {
		if node.Type == string(model.LineageEntityReport) {
			report = node
		}
	}
	if report.Depth != 3 {
		t.Fatalf("report depth = %d, want 3", report.Depth)
	}
	if !report.IsCritical {
		t.Fatalf("report should inherit criticality from restricted upstream model")
	}
}

func TestFinalizeGraphInOutDegree(t *testing.T) {
	rootID := uuid.New()
	centerID := uuid.New()
	leafAID := uuid.New()
	leafBID := uuid.New()
	leafCID := uuid.New()

	nodes := map[string]model.LineageNode{
		nodeKey(model.LineageEntityDataSource, rootID):     {ID: nodeKey(model.LineageEntityDataSource, rootID), Type: string(model.LineageEntityDataSource), EntityID: rootID, Name: "Root"},
		nodeKey(model.LineageEntityDataModel, centerID):    {ID: nodeKey(model.LineageEntityDataModel, centerID), Type: string(model.LineageEntityDataModel), EntityID: centerID, Name: "Center"},
		nodeKey(model.LineageEntityPipeline, leafAID):      {ID: nodeKey(model.LineageEntityPipeline, leafAID), Type: string(model.LineageEntityPipeline), EntityID: leafAID, Name: "LeafA"},
		nodeKey(model.LineageEntitySuiteConsumer, leafBID): {ID: nodeKey(model.LineageEntitySuiteConsumer, leafBID), Type: string(model.LineageEntitySuiteConsumer), EntityID: leafBID, Name: "LeafB"},
		nodeKey(model.LineageEntityReport, leafCID):        {ID: nodeKey(model.LineageEntityReport, leafCID), Type: string(model.LineageEntityReport), EntityID: leafCID, Name: "LeafC"},
	}
	edges := []model.LineageEdge{
		{ID: uuid.New(), Source: nodeKey(model.LineageEntityDataSource, rootID), Target: nodeKey(model.LineageEntityDataModel, centerID), Relationship: "feeds", Active: true, LastSeenAt: time.Now()},
		{ID: uuid.New(), Source: nodeKey(model.LineageEntityDataModel, centerID), Target: nodeKey(model.LineageEntityPipeline, leafAID), Relationship: "transforms_into", Active: true, LastSeenAt: time.Now()},
		{ID: uuid.New(), Source: nodeKey(model.LineageEntityDataModel, centerID), Target: nodeKey(model.LineageEntitySuiteConsumer, leafBID), Relationship: "consumed_by", Active: true, LastSeenAt: time.Now()},
		{ID: uuid.New(), Source: nodeKey(model.LineageEntityDataModel, centerID), Target: nodeKey(model.LineageEntityReport, leafCID), Relationship: "reported_in", Active: true, LastSeenAt: time.Now()},
	}

	graph := finalizeGraph(nodes, edges)
	for _, node := range graph.Nodes {
		if node.ID != nodeKey(model.LineageEntityDataModel, centerID) {
			continue
		}
		if node.InDegree != 1 {
			t.Fatalf("InDegree = %d, want 1", node.InDegree)
		}
		if node.OutDegree != 3 {
			t.Fatalf("OutDegree = %d, want 3", node.OutDegree)
		}
		return
	}
	t.Fatalf("center node not found")
}
