package engine

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// ---------------------------------------------------------------------------
// Tests: NewMetrics
// ---------------------------------------------------------------------------

func TestNewMetrics_NilRegisterer(t *testing.T) {
	// Should not panic — uses a throwaway registry.
	m := NewMetrics(nil)
	if m == nil {
		t.Fatal("expected non-nil Metrics")
	}
}

func TestNewMetrics_CustomRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	if m == nil {
		t.Fatal("expected non-nil Metrics")
	}

	// Verify collectors are registered by gathering.
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather failed: %v", err)
	}
	if len(families) == 0 {
		t.Error("expected at least one metric family registered")
	}
}

func TestNewMetrics_AllFieldsPopulated(t *testing.T) {
	m := NewMetrics(prometheus.NewRegistry())

	checks := []struct {
		name string
		ok   bool
	}{
		{"QueriesTotal", m.QueriesTotal != nil},
		{"CallsTotal", m.CallsTotal != nil},
		{"CallLatencySeconds", m.CallLatencySeconds != nil},
		{"TokensTotal", m.TokensTotal != nil},
		{"CostUSDTotal", m.CostUSDTotal != nil},
		{"ToolLoopIterations", m.ToolLoopIterations != nil},
		{"ToolCallsPerQuery", m.ToolCallsPerQuery != nil},
		{"GroundingResultsTotal", m.GroundingResultsTotal != nil},
		{"InjectionDetectionsTotal", m.InjectionDetectionsTotal != nil},
		{"RateLimitRejectionsTotal", m.RateLimitRejectionsTotal != nil},
		{"FallbackTotal", m.FallbackTotal != nil},
		{"ResponseLatencySeconds", m.ResponseLatencySeconds != nil},
		{"ContextTokensUsed", m.ContextTokensUsed != nil},
	}

	for _, c := range checks {
		if !c.ok {
			t.Errorf("field %s is nil", c.name)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Duplicate registration resilience
// ---------------------------------------------------------------------------

func TestNewMetricsSafe_DuplicateRegistration(t *testing.T) {
	reg := prometheus.NewRegistry()

	// First registration.
	m1, err := NewMetricsSafe(reg)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	if m1 == nil {
		t.Fatal("expected non-nil Metrics from first registration")
	}

	// Second registration — should NOT panic.
	m2, err := NewMetricsSafe(reg)
	if err != nil {
		t.Fatalf("second registration should succeed (skip duplicates), got: %v", err)
	}
	if m2 == nil {
		t.Fatal("expected non-nil Metrics from second registration")
	}
}

// ---------------------------------------------------------------------------
// Tests: NewNoopMetrics
// ---------------------------------------------------------------------------

func TestNewNoopMetrics_AllFieldsNil(t *testing.T) {
	m := NewNoopMetrics()
	if m == nil {
		t.Fatal("expected non-nil Metrics struct")
	}
	// Spot-check that fields are nil.
	if m.CallsTotal != nil {
		t.Error("expected CallsTotal to be nil in noop metrics")
	}
	if m.FallbackTotal != nil {
		t.Error("expected FallbackTotal to be nil in noop metrics")
	}
	if m.ContextTokensUsed != nil {
		t.Error("expected ContextTokensUsed to be nil in noop metrics")
	}
}

// ---------------------------------------------------------------------------
// Tests: Nil-safe helpers
// ---------------------------------------------------------------------------

func TestSafeInc_NilCounterVec(t *testing.T) {
	// Must not panic.
	safeInc(nil, "a", "b")
}

func TestSafeInc_WithCounterVec(t *testing.T) {
	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_safe_inc",
		Help: "test",
	}, []string{"label"})

	safeInc(cv, "value")

	// Verify the counter was incremented.
	metric, err := cv.GetMetricWithLabelValues("value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// We can't easily read the value without a DTO, but at least
	// verify no panic and the metric exists.
	_ = metric
}

func TestSafeAdd_NilCounterVec(t *testing.T) {
	safeAdd(nil, 5.0, "a", "b")
}

func TestSafeObserve_NilHistogram(t *testing.T) {
	safeObserve(nil, 1.5)
}

func TestSafeObserve_WithHistogram(t *testing.T) {
	h := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "test_safe_observe",
		Help:    "test",
		Buckets: prometheus.DefBuckets,
	})
	safeObserve(h, 1.5) // must not panic
}

func TestSafeObserveVec_NilHistogramVec(t *testing.T) {
	safeObserveVec(nil, 1.5, "a")
}

func TestSafeObserveVec_WithHistogramVec(t *testing.T) {
	hv := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "test_safe_observe_vec",
		Help:    "test",
		Buckets: prometheus.DefBuckets,
	}, []string{"label"})
	safeObserveVec(hv, 1.5, "value") // must not panic
}

func TestSafeCounter_Nil(t *testing.T) {
	safeCounter(nil) // must not panic
}

func TestSafeCounterAdd_Nil(t *testing.T) {
	safeCounterAdd(nil, 5.0) // must not panic
}

// ---------------------------------------------------------------------------
// Tests: Metric names use namespace/subsystem
// ---------------------------------------------------------------------------

func TestMetricNaming_UsesNamespace(t *testing.T) {
	reg := prometheus.NewRegistry()
	NewMetrics(reg)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather failed: %v", err)
	}

	for _, fam := range families {
		name := fam.GetName()
		if len(name) < len("vciso_llm_") {
			t.Errorf("metric %q missing namespace prefix", name)
			continue
		}
		if name[:10] != "vciso_llm_" {
			t.Errorf("metric %q does not start with vciso_llm_", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmark: safe helpers overhead
// ---------------------------------------------------------------------------

func BenchmarkSafeInc_Nil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		safeInc(nil, "a", "b")
	}
}

func BenchmarkSafeInc_Real(b *testing.B) {
	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bench_safe_inc",
		Help: "bench",
	}, []string{"a", "b"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		safeInc(cv, "x", "y")
	}
}

func BenchmarkSafeObserve_Nil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		safeObserve(nil, 1.5)
	}
}

func BenchmarkSafeObserve_Real(b *testing.B) {
	h := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "bench_safe_observe",
		Help:    "bench",
		Buckets: prometheus.DefBuckets,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		safeObserve(h, 1.5)
	}
}