package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewCreatesIndependentRegistries(t *testing.T) {
	first := New()
	second := New()

	if first.Registry == second.Registry {
		t.Fatal("expected separate Prometheus registries per metrics instance")
	}

	first.HTTPRequestsTotal.WithLabelValues("GET", "/alerts", "200").Add(2)
	second.HTTPRequestsTotal.WithLabelValues("GET", "/alerts", "200").Inc()

	if got := testutil.ToFloat64(first.HTTPRequestsTotal.WithLabelValues("GET", "/alerts", "200")); got != 2 {
		t.Fatalf("first registry counter = %v, want 2", got)
	}

	if got := testutil.ToFloat64(second.HTTPRequestsTotal.WithLabelValues("GET", "/alerts", "200")); got != 1 {
		t.Fatalf("second registry counter = %v, want 1", got)
	}
}

func TestNewRegistersMetricsOnPrivateRegistry(t *testing.T) {
	metrics := New()
	metrics.AssetsTotal.WithLabelValues("tenant-1", "server", "high").Set(3)
	metrics.HTTPRequestsTotal.WithLabelValues("GET", "/alerts", "200").Inc()
	metrics.RiskScoreCurrent.WithLabelValues("tenant-1", "A").Set(72.5)

	families, err := metrics.Registry.Gather()
	if err != nil {
		t.Fatalf("gather registry metrics: %v", err)
	}

	names := make(map[string]struct{}, len(families))
	for _, family := range families {
		names[family.GetName()] = struct{}{}
	}

	for _, expected := range []string{
		"cyber_assets_total",
		"cyber_http_requests_total",
		"risk_score_current",
	} {
		if _, ok := names[expected]; !ok {
			t.Fatalf("expected metric %q to be registered", expected)
		}
	}
}
