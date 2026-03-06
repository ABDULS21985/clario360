package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartSpan starts a new span using the global tracer provider.
// Returns the enriched context and span.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("clario360").Start(ctx, name, opts...)
}

// RecordError records an error on the span and sets the span status to Error.
// Safe to call with a nil or noop span.
func RecordError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// AddEvent adds a named event with optional attributes to the span.
// Safe to call with a nil or noop span.
func AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SpanFromRequest returns the active span from an HTTP request's context.
func SpanFromRequest(r *http.Request) trace.Span {
	return trace.SpanFromContext(r.Context())
}

// TraceIDFromContext returns the hex-encoded trace ID from the active span,
// or "" if there is no active span.
func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}

// SpanIDFromContext returns the hex-encoded span ID from the active span,
// or "" if there is no active span.
func SpanIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if sc.HasSpanID() {
		return sc.SpanID().String()
	}
	return ""
}
