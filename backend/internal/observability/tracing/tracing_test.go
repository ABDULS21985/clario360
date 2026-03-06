package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func TestInitTracer_Disabled(t *testing.T) {
	cfg := TracerConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	_, shutdown, err := InitTracer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("InitTracer(disabled) error = %v", err)
	}

	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}

	// Shutdown should be a no-op.
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error = %v", err)
	}
}

func TestInitTracer_NoEndpoint(t *testing.T) {
	cfg := TracerConfig{
		Enabled:     true,
		Endpoint:    "",
		ServiceName: "test-service",
	}

	_, shutdown, err := InitTracer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("InitTracer(no endpoint) error = %v", err)
	}

	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error = %v", err)
	}
}

func TestTraceIDFromContext_NoSpan(t *testing.T) {
	ctx := context.Background()
	traceID := TraceIDFromContext(ctx)
	if traceID != "" {
		t.Errorf("TraceIDFromContext(no span) = %q, want empty", traceID)
	}
}

func TestSpanIDFromContext_NoSpan(t *testing.T) {
	ctx := context.Background()
	spanID := SpanIDFromContext(ctx)
	if spanID != "" {
		t.Errorf("SpanIDFromContext(no span) = %q, want empty", spanID)
	}
}

func TestRecordError_NilSpan(t *testing.T) {
	// Should not panic.
	RecordError(nil, nil)
}

func TestHTTPCarrier_Roundtrip(t *testing.T) {
	// Set up a propagator.
	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})
	otel.SetTextMapPropagator(prop)

	// Verify carrier interface works with basic set/get.
	headers := make(map[string][]string)
	carrier := propagationHTTPCarrier(headers)

	carrier.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	got := carrier.Get("traceparent")
	// http.Header canonicalizes keys
	if got == "" {
		t.Error("expected non-empty traceparent after Set")
	}

	keys := carrier.Keys()
	if len(keys) == 0 {
		t.Error("expected at least one key")
	}
}
