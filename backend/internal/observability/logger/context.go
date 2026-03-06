package logger

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// Context key types to avoid collisions with other packages.
type ctxKey int

const (
	// CtxKeyRequestID is the context key for the request ID.
	CtxKeyRequestID ctxKey = iota
	// CtxKeyTenantID is the context key for the tenant ID.
	CtxKeyTenantID
	// CtxKeyUserID is the context key for the user ID.
	CtxKeyUserID
	// CtxKeyCorrelationID is the context key for the correlation ID.
	CtxKeyCorrelationID
)

// FromContext enriches the base logger with request-scoped fields extracted from context.
//
// Fields extracted: request_id, tenant_id, user_id, trace_id, span_id, correlation_id.
// If a field is not present in context, it is omitted (not set to empty string).
func FromContext(ctx context.Context, base zerolog.Logger) zerolog.Logger {
	logCtx := base.With()

	if v, ok := ctx.Value(CtxKeyRequestID).(string); ok && v != "" {
		logCtx = logCtx.Str("request_id", v)
	}

	if v, ok := ctx.Value(CtxKeyTenantID).(string); ok && v != "" {
		logCtx = logCtx.Str("tenant_id", v)
	}

	if v, ok := ctx.Value(CtxKeyUserID).(string); ok && v != "" {
		logCtx = logCtx.Str("user_id", v)
	}

	if v, ok := ctx.Value(CtxKeyCorrelationID).(string); ok && v != "" {
		logCtx = logCtx.Str("correlation_id", v)
	}

	// Extract trace and span IDs from OTel span context.
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		logCtx = logCtx.Str("trace_id", spanCtx.TraceID().String())
	}
	if spanCtx.HasSpanID() {
		logCtx = logCtx.Str("span_id", spanCtx.SpanID().String())
	}

	return logCtx.Logger()
}

// FromRequest is a convenience helper that calls FromContext with r.Context().
func FromRequest(r *http.Request, base zerolog.Logger) zerolog.Logger {
	return FromContext(r.Context(), base)
}

// WithRequestID stores a request ID in the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyRequestID, id)
}

// WithTenantID stores a tenant ID in the context.
func WithTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyTenantID, id)
}

// WithUserID stores a user ID in the context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyUserID, id)
}

// WithCorrelationID stores a correlation ID in the context.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyCorrelationID, id)
}

// RequestIDFromContext retrieves the request ID from context, or "".
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyRequestID).(string)
	return v
}

// TenantIDFromContext retrieves the tenant ID from context, or "".
func TenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyTenantID).(string)
	return v
}

// UserIDFromContext retrieves the user ID from context, or "".
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyUserID).(string)
	return v
}

// CorrelationIDFromContext retrieves the correlation ID from context, or "".
func CorrelationIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyCorrelationID).(string)
	return v
}
