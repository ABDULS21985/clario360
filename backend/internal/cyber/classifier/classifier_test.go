package classifier

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestClassify_DefaultRules(t *testing.T) {
	t.Run("database critical", func(t *testing.T) {
		asset := &model.Asset{Type: model.AssetTypeDatabase}
		crit, _, _ := NewAssetClassifier(zerolog.Nop()).Classify(asset)
		if crit != model.CriticalityCritical {
			t.Fatalf("expected critical, got %s", crit)
		}
	})

	t.Run("production critical", func(t *testing.T) {
		asset := &model.Asset{Type: model.AssetTypeServer, Tags: []string{"production"}}
		crit, _, _ := NewAssetClassifier(zerolog.Nop()).Classify(asset)
		if crit != model.CriticalityCritical {
			t.Fatalf("expected critical, got %s", crit)
		}
	})

	t.Run("internet facing high", func(t *testing.T) {
		asset := &model.Asset{Tags: []string{"internet-facing"}}
		crit, _, _ := NewAssetClassifier(zerolog.Nop()).Classify(asset)
		if crit != model.CriticalityHigh {
			t.Fatalf("expected high, got %s", crit)
		}
	})

	t.Run("iot high", func(t *testing.T) {
		asset := &model.Asset{Type: model.AssetTypeIoTDevice}
		crit, _, _ := NewAssetClassifier(zerolog.Nop()).Classify(asset)
		if crit != model.CriticalityHigh {
			t.Fatalf("expected high, got %s", crit)
		}
	})

	t.Run("server medium", func(t *testing.T) {
		asset := &model.Asset{Type: model.AssetTypeServer}
		crit, _, _ := NewAssetClassifier(zerolog.Nop()).Classify(asset)
		if crit != model.CriticalityMedium {
			t.Fatalf("expected medium, got %s", crit)
		}
	})

	t.Run("endpoint low", func(t *testing.T) {
		asset := &model.Asset{Type: model.AssetTypeEndpoint}
		crit, _, _ := NewAssetClassifier(zerolog.Nop()).Classify(asset)
		if crit != model.CriticalityLow {
			t.Fatalf("expected low, got %s", crit)
		}
	})

	t.Run("domain controller critical", func(t *testing.T) {
		hostname := "dc-prod-01"
		asset := &model.Asset{Hostname: &hostname}
		crit, _, _ := NewAssetClassifier(zerolog.Nop()).Classify(asset)
		if crit != model.CriticalityCritical {
			t.Fatalf("expected critical, got %s", crit)
		}
	})
}

func TestClassify_FirstMatchWinsAndCustomOverride(t *testing.T) {
	asset := &model.Asset{Type: model.AssetTypeServer, Tags: []string{"production"}}
	crit, rule, _ := NewAssetClassifier(zerolog.Nop()).Classify(asset)
	if crit != model.CriticalityCritical || rule != "production-assets-are-critical" {
		t.Fatalf("expected production rule, got %s %s", crit, rule)
	}

	custom := ClassificationRule{
		Name:     "custom-override",
		Priority: 0,
		Condition: func(asset *model.Asset) bool {
			return asset.Type == model.AssetTypeServer
		},
		Result: model.CriticalityLow,
	}
	crit, rule, _ = NewAssetClassifier(zerolog.Nop(), custom).Classify(&model.Asset{Type: model.AssetTypeServer})
	if crit != model.CriticalityLow || rule != "custom-override" {
		t.Fatalf("expected custom override, got %s %s", crit, rule)
	}
}

func TestClassifyBatch(t *testing.T) {
	openPorts, _ := json.Marshal(map[string]any{"open_ports": []int{80, 443}})
	assets := []*model.Asset{
		{ID: uuid.New(), Type: model.AssetTypeDatabase},
		{ID: uuid.New(), Type: model.AssetTypeServer},
		{ID: uuid.New(), Type: model.AssetTypeEndpoint},
		{ID: uuid.New(), Type: model.AssetTypeCloudResource, Metadata: json.RawMessage(`{"public_ip":"1.2.3.4"}`)},
		{ID: uuid.New(), Type: model.AssetTypeServer, Metadata: openPorts},
	}

	results := NewAssetClassifier(zerolog.Nop()).ClassifyBatch(assets)
	if len(results) != len(assets) {
		t.Fatalf("expected %d results, got %d", len(assets), len(results))
	}
	if results[0].Criticality != model.CriticalityCritical {
		t.Fatalf("expected first asset critical, got %s", results[0].Criticality)
	}
}
