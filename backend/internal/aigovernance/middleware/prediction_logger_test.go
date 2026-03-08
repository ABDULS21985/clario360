package middleware

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	aigovservice "github.com/clario360/platform/internal/aigovernance/service"
	aishadow "github.com/clario360/platform/internal/aigovernance/shadow"
)

func TestPredictLogsAsynchronously(t *testing.T) {
	tenantID := uuid.New()
	logger := newTestPredictionLogger(t, tenantID, false)

	result, err := logger.Predict(context.Background(), predictParams(tenantID, nil))
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}
	if result == nil {
		t.Fatal("Predict() returned nil result")
	}
	if got := len(logger.predictionCh); got != 1 {
		t.Fatalf("prediction queue length = %d, want 1", got)
	}
}

func TestPredictExecutesShadow(t *testing.T) {
	tenantID := uuid.New()
	logger := newTestPredictionLogger(t, tenantID, true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go logger.shadowWorker(ctx)

	result, err := logger.Predict(context.Background(), predictParams(tenantID, shadowModelOutput))
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}
	if result == nil {
		t.Fatal("Predict() returned nil result")
	}

	deadline := time.Now().Add(2 * time.Second)
	for len(logger.predictionCh) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(logger.predictionCh) < 2 {
		t.Fatalf("prediction queue length = %d, want at least 2 after shadow execution", len(logger.predictionCh))
	}

	logs := drainLogs(logger.predictionCh)
	shadowCount := 0
	for _, item := range logs {
		if item.IsShadow {
			shadowCount++
		}
	}
	if shadowCount != 1 {
		t.Fatalf("shadow prediction count = %d, want 1", shadowCount)
	}
}

func TestPredictNoShadow(t *testing.T) {
	tenantID := uuid.New()
	logger := newTestPredictionLogger(t, tenantID, false)

	if _, err := logger.Predict(context.Background(), predictParams(tenantID, shadowModelOutput)); err != nil {
		t.Fatalf("Predict() error = %v", err)
	}
	if got := len(logger.shadowCh); got != 0 {
		t.Fatalf("shadow queue length = %d, want 0", got)
	}
}

func TestPredictChannelFull(t *testing.T) {
	tenantID := uuid.New()
	logger := newTestPredictionLogger(t, tenantID, false)
	logger.predictionCh = make(chan *aigovmodel.PredictionLog, 1)
	logger.predictionCh <- &aigovmodel.PredictionLog{ID: uuid.New()}

	if _, err := logger.Predict(context.Background(), predictParams(tenantID, nil)); err != nil {
		t.Fatalf("Predict() error = %v", err)
	}
	if got := testutil.ToFloat64(logger.metrics.PredictionLogsDropped); got != 1 {
		t.Fatalf("PredictionLogsDropped = %.0f, want 1", got)
	}
}

func TestPredictCachedVersion(t *testing.T) {
	tenantID := uuid.New()
	logger := newTestPredictionLogger(t, tenantID, false)
	logger.registryRepo = nil

	for i := 0; i < 2; i++ {
		if _, err := logger.Predict(context.Background(), predictParams(tenantID, nil)); err != nil {
			t.Fatalf("Predict() iteration %d error = %v", i+1, err)
		}
	}
}

func TestPredictionLoggerUnder5ms(t *testing.T) {
	tenantID := uuid.New()
	logger := newTestPredictionLogger(t, tenantID, false)
	logger.predictionCh = make(chan *aigovmodel.PredictionLog, 2048)
	params := predictParams(tenantID, nil)
	params.Input = "suspicious-powershell"
	params.InputSummary = map[string]any{"rule": "suspicious-powershell"}

	const (
		batches   = 100
		batchSize = 10
	)
	for i := 0; i < 100; i++ {
		if _, err := logger.Predict(context.Background(), params); err != nil {
			t.Fatalf("warm-up Predict() error = %v", err)
		}
	}
	overheads := make([]time.Duration, 0, batches)
	for i := 0; i < batches; i++ {
		rawStart := time.Now()
		for j := 0; j < batchSize; j++ {
			if _, err := params.ModelFunc(context.Background(), params.Input); err != nil {
				t.Fatalf("raw model func error = %v", err)
			}
		}
		rawDuration := time.Since(rawStart)

		governedStart := time.Now()
		for j := 0; j < batchSize; j++ {
			if _, err := logger.Predict(context.Background(), params); err != nil {
				t.Fatalf("Predict() error = %v", err)
			}
		}
		governedDuration := time.Since(governedStart)
		overhead := (governedDuration - rawDuration) / batchSize
		if overhead < 0 {
			overhead = 0
		}
		overheads = append(overheads, overhead)
	}

	sort.Slice(overheads, func(i, j int) bool {
		return overheads[i] < overheads[j]
	})
	p99 := overheads[int(float64(len(overheads))*0.99)]
	if p99 > 5*time.Millisecond {
		t.Fatalf("prediction logger p99 overhead = %s, want <= 5ms", p99)
	}
}

func TestPredictRequiresTenantID(t *testing.T) {
	logger := newTestPredictionLogger(t, uuid.New(), false)
	if _, err := logger.Predict(context.Background(), predictParams(uuid.Nil, nil)); err == nil {
		t.Fatal("expected tenant_id validation error")
	}
}

func newTestPredictionLogger(t *testing.T, tenantID uuid.UUID, withShadow bool) *PredictionLogger {
	t.Helper()

	reg := prometheus.NewRegistry()
	metrics := aigovservice.NewMetrics(reg)
	explanationSvc := aigovservice.NewExplanationService(zerolog.Nop())
	logger := &PredictionLogger{
		predictionCh:   make(chan *aigovmodel.PredictionLog, 32),
		shadowCh:       make(chan *aishadow.ExecutionTask, 8),
		explanationSvc: explanationSvc,
		metrics:        metrics,
		logger:         zerolog.Nop(),
	}
	logger.shadowExecutor = aishadow.NewExecutor(explanationSvc, zerolog.Nop())

	production := &aigovmodel.ModelVersion{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		ModelID:            uuid.New(),
		ModelSlug:          "cyber-sigma-evaluator",
		ModelSuite:         aigovmodel.SuiteCyber,
		VersionNumber:      1,
		Status:             aigovmodel.VersionStatusProduction,
		ExplainabilityType: aigovmodel.ExplainabilityRuleTrace,
	}
	logger.registryCache.Store(cacheKey(tenantID, production.ModelSlug, aigovmodel.VersionStatusProduction), &cacheEntry{
		version:   production,
		expiresAt: time.Now().UTC().Add(time.Minute),
	})
	if withShadow {
		shadow := &aigovmodel.ModelVersion{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			ModelID:            production.ModelID,
			ModelSlug:          production.ModelSlug,
			ModelSuite:         aigovmodel.SuiteCyber,
			VersionNumber:      2,
			Status:             aigovmodel.VersionStatusShadow,
			ExplainabilityType: aigovmodel.ExplainabilityRuleTrace,
		}
		logger.registryCache.Store(cacheKey(tenantID, production.ModelSlug, aigovmodel.VersionStatusShadow), &cacheEntry{
			version:   shadow,
			expiresAt: time.Now().UTC().Add(time.Minute),
		})
	}
	return logger
}

func predictParams(tenantID uuid.UUID, shadowFunc func(context.Context, any) (*aigovernance.ModelOutput, error)) aigovernance.PredictParams {
	return aigovernance.PredictParams{
		TenantID:     tenantID,
		ModelSlug:    "cyber-sigma-evaluator",
		UseCase:      "threat_detection",
		EntityType:   "detection_rule",
		Input:        map[string]any{"rule": "suspicious-powershell", "event_count": 3},
		InputSummary: map[string]any{"rule": "suspicious-powershell", "event_count": 3},
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output:     map[string]any{"matched": true},
				Confidence: 0.93,
				Metadata: map[string]any{
					"matched":            true,
					"rule_name":          "Suspicious PowerShell",
					"matched_rules":      []string{"Suspicious PowerShell"},
					"matched_conditions": []string{"process_name", "command_line"},
					"rule_weights":       map[string]any{"Suspicious PowerShell": 0.93},
				},
			}, nil
		},
		ShadowModelFunc: shadowFunc,
	}
}

func shadowModelOutput(context.Context, any) (*aigovernance.ModelOutput, error) {
	return &aigovernance.ModelOutput{
		Output:     map[string]any{"matched": true},
		Confidence: 0.91,
		Metadata: map[string]any{
			"matched":            true,
			"rule_name":          "Suspicious PowerShell",
			"matched_rules":      []string{"Suspicious PowerShell"},
			"matched_conditions": []string{"process_name"},
			"rule_weights":       map[string]any{"Suspicious PowerShell": 0.91},
		},
	}, nil
}

func cacheKey(tenantID uuid.UUID, slug string, status aigovmodel.VersionStatus) string {
	return tenantID.String() + ":" + slug + ":" + string(status)
}

func drainLogs(ch chan *aigovmodel.PredictionLog) []*aigovmodel.PredictionLog {
	out := make([]*aigovmodel.PredictionLog, 0, len(ch))
	for {
		select {
		case item := <-ch:
			if item != nil {
				out = append(out, item)
			}
		default:
			return out
		}
	}
}

func TestPredictRejectsMissingModelFunc(t *testing.T) {
	tenantID := uuid.New()
	logger := newTestPredictionLogger(t, tenantID, false)
	params := predictParams(tenantID, nil)
	params.ModelFunc = nil
	if _, err := logger.Predict(context.Background(), params); err == nil {
		t.Fatal("expected model function validation error")
	}
}
