package metrics

import (
	"testing"
)

func TestNewMetrics_StandardMetricsRegistered(t *testing.T) {
	m := NewMetrics("testservice")

	if m.HTTP == nil {
		t.Fatal("HTTP metrics are nil")
	}
	if m.DB == nil {
		t.Fatal("DB metrics are nil")
	}
	if m.Kafka == nil {
		t.Fatal("Kafka metrics are nil")
	}

	// Verify specific HTTP metric fields are populated.
	if m.HTTP.RequestsTotal == nil {
		t.Error("HTTP.RequestsTotal is nil")
	}
	if m.HTTP.RequestDuration == nil {
		t.Error("HTTP.RequestDuration is nil")
	}
	if m.HTTP.RequestSize == nil {
		t.Error("HTTP.RequestSize is nil")
	}
	if m.HTTP.ResponseSize == nil {
		t.Error("HTTP.ResponseSize is nil")
	}
	if m.HTTP.ActiveRequests == nil {
		t.Error("HTTP.ActiveRequests is nil")
	}
	if m.HTTP.PanicsTotal == nil {
		t.Error("HTTP.PanicsTotal is nil")
	}

	// Verify specific DB metric fields are populated.
	if m.DB.QueriesTotal == nil {
		t.Error("DB.QueriesTotal is nil")
	}
	if m.DB.QueryDuration == nil {
		t.Error("DB.QueryDuration is nil")
	}
	if m.DB.ConnectionsActive == nil {
		t.Error("DB.ConnectionsActive is nil")
	}
	if m.DB.ConnectionsIdle == nil {
		t.Error("DB.ConnectionsIdle is nil")
	}
	if m.DB.ConnectionsMax == nil {
		t.Error("DB.ConnectionsMax is nil")
	}
	if m.DB.ConnectionsWait == nil {
		t.Error("DB.ConnectionsWait is nil")
	}
	if m.DB.ConnectionsWaitDur == nil {
		t.Error("DB.ConnectionsWaitDur is nil")
	}

	// Verify specific Kafka metric fields are populated.
	if m.Kafka.ProducedTotal == nil {
		t.Error("Kafka.ProducedTotal is nil")
	}
	if m.Kafka.ProduceDuration == nil {
		t.Error("Kafka.ProduceDuration is nil")
	}
	if m.Kafka.ConsumedTotal == nil {
		t.Error("Kafka.ConsumedTotal is nil")
	}
	if m.Kafka.ConsumeDuration == nil {
		t.Error("Kafka.ConsumeDuration is nil")
	}
	if m.Kafka.ConsumerLag == nil {
		t.Error("Kafka.ConsumerLag is nil")
	}
	if m.Kafka.RebalanceTotal == nil {
		t.Error("Kafka.RebalanceTotal is nil")
	}

	// Verify the registry is usable by gathering all registered metrics.
	families, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("Registry().Gather() returned error: %v", err)
	}
	if len(families) == 0 {
		t.Error("expected gathered metric families, got 0")
	}
}

func TestCustomCounter_Registration(t *testing.T) {
	m := NewMetrics("testservice")

	counter := m.Counter("requests_custom", "A custom counter", []string{"method"})
	if counter == nil {
		t.Fatal("Counter() returned nil")
	}

	// Verify the counter is functional by incrementing it.
	counter.WithLabelValues("GET").Inc()

	families, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("Registry().Gather() returned error: %v", err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "testservice_requests_custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("custom counter 'testservice_requests_custom' not found in gathered metrics")
	}
}

func TestCustomCounter_DuplicateRegistration(t *testing.T) {
	m := NewMetrics("testservice")

	counter1 := m.Counter("dup_counter", "First registration", []string{"label"})
	counter2 := m.Counter("dup_counter", "Second registration", []string{"label"})

	if counter1 != counter2 {
		t.Error("expected duplicate Counter() calls to return the same *CounterVec instance")
	}
}

func TestCustomGauge_Registration(t *testing.T) {
	m := NewMetrics("testservice")

	gauge := m.Gauge("active_connections", "Active connections gauge", []string{"pool"})
	if gauge == nil {
		t.Fatal("Gauge() returned nil")
	}

	// Verify it works.
	gauge.WithLabelValues("main").Set(42)

	families, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("Registry().Gather() returned error: %v", err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "testservice_active_connections" {
			found = true
			break
		}
	}
	if !found {
		t.Error("custom gauge 'testservice_active_connections' not found in gathered metrics")
	}
}

func TestCustomHistogram_Registration(t *testing.T) {
	m := NewMetrics("testservice")

	buckets := []float64{0.01, 0.05, 0.1, 0.5, 1.0}
	histogram := m.Histogram("request_latency", "Request latency histogram", []string{"endpoint"}, buckets)
	if histogram == nil {
		t.Fatal("Histogram() returned nil")
	}

	// Verify it works.
	histogram.WithLabelValues("/api/v1/test").Observe(0.042)

	families, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("Registry().Gather() returned error: %v", err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "testservice_request_latency" {
			found = true
			break
		}
	}
	if !found {
		t.Error("custom histogram 'testservice_request_latency' not found in gathered metrics")
	}
}
