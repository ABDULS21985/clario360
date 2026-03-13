package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
	predictmodels "github.com/clario360/platform/internal/cyber/vciso/predict/models"
)

func TestModelRegistryActivateWritesArtifact(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	registry := NewModelRegistry(nil, dir, zerolog.Nop())
	model := predictmodels.NewAlertVolumeForecaster("artifact-test")
	modelRow, err := registry.Activate(context.Background(), predictmodel.PredictionTypeAlertVolumeForecast, predictmodel.FrameworkProphet, model, predictmodel.BacktestMetrics{Accuracy: 0.8, Count: 1}, 5, 10, 0)
	if err != nil {
		t.Fatalf("activate error: %v", err)
	}
	path := filepath.Join(dir, string(predictmodel.PredictionTypeAlertVolumeForecast), "artifact-test.json")
	if modelRow.ModelArtifactPath != path {
		t.Fatalf("artifact path = %q, want %q", modelRow.ModelArtifactPath, path)
	}
}
