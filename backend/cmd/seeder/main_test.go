package main

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/classifier"
	"github.com/clario360/platform/internal/cyber/model"
)

func TestGenerateAssets_CountsByType(t *testing.T) {
	gen := newTestGeneratorFromMathRand()
	assets := gen.generateAssets(uuid.New())
	if len(assets) != assetTargetCount {
		t.Fatalf("expected %d assets, got %d", assetTargetCount, len(assets))
	}

	counts := map[model.AssetType]int{}
	for _, asset := range assets {
		counts[asset.Type]++
	}

	expected := map[model.AssetType]int{
		model.AssetTypeServer: 200, model.AssetTypeEndpoint: 150, model.AssetTypeNetworkDevice: 30,
		model.AssetTypeCloudResource: 40, model.AssetTypeIoTDevice: 20, model.AssetTypeApplication: 25,
		model.AssetTypeDatabase: 20, model.AssetTypeContainer: 15,
	}
	for assetType, want := range expected {
		if got := counts[assetType]; got != want {
			t.Fatalf("expected %s=%d, got %d", assetType, want, got)
		}
	}
}

func TestGenerateAssets_ClassifierProducesCriticalAssets(t *testing.T) {
	gen := newTestGeneratorFromMathRand()
	assets := gen.generateAssets(uuid.New())
	critical := 0
	for _, asset := range assets {
		if asset.Criticality == model.CriticalityCritical {
			critical++
		}
	}
	if critical == 0 {
		t.Fatal("expected at least one critical asset")
	}
}

func TestGenerateVulnerabilitiesAndRelationships_TargetCounts(t *testing.T) {
	gen := newTestGeneratorFromMathRand()
	assets := gen.generateAssets(uuid.New())
	vulns := gen.generateVulnerabilities(uuid.New(), assets)
	rels := gen.generateRelationships(uuid.New(), assets)

	if len(vulns) != vulnTargetCount {
		t.Fatalf("expected %d vulnerabilities, got %d", vulnTargetCount, len(vulns))
	}
	if len(rels) != relationshipTargetCount {
		t.Fatalf("expected %d relationships, got %d", relationshipTargetCount, len(rels))
	}
	for _, rel := range rels {
		if rel.SourceAssetID == rel.TargetAssetID {
			t.Fatal("expected no self-referential relationships")
		}
	}
}

func newTestGeneratorFromMathRand() *generator {
	return &generator{
		classifier: classifier.NewAssetClassifier(zerolog.Nop()),
		logger:     zerolog.Nop(),
		rng:        rand.New(rand.NewSource(defaultSeed)),
		now:        time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC),
		usedIPs:    make(map[string]struct{}),
		usedMACs:   make(map[string]struct{}),
		createdBy:  uuid.New(),
	}
}
