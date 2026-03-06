package tracing

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/clario360/platform/internal/auth"
)

// ChiTracingMiddleware returns chi-compatible middleware that creates spans for incoming HTTP requests.
//
// It extracts W3C TraceContext headers (traceparent, tracestate) from the incoming request,
// creates a server span, and injects platform attributes.
//
// Middleware order: register AFTER RequestID middleware and BEFORE auth middleware.
func ChiTracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract propagated trace context from incoming headers.
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagationHTTPCarrier(r.Header))

			// Determine span name: "{method} {route}".
			routePattern := "unknown"
			if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
				routePattern = rctx.RoutePattern()
			}
			spanName := r.Method + " " + routePattern

			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPMethodKey.String(r.Method),
					semconv.HTTPRouteKey.String(routePattern),
					semconv.HTTPTargetKey.String(r.URL.Path),
					semconv.HTTPSchemeKey.String(scheme(r)),
					attribute.String("net.peer.ip", r.RemoteAddr),
				),
			)
			defer span.End()

			// Inject trace context into response headers for client correlation.
			traceID := TraceIDFromContext(ctx)
			if traceID != "" {
				w.Header().Set("X-Trace-ID", traceID)
			}

			// Replace request context with span context.
			r = r.WithContext(ctx)

			// Wrap response writer to capture status code.
			tw := &tracingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(tw, r)

			// Set response attributes.
			span.SetAttributes(
				semconv.HTTPStatusCodeKey.Int(tw.statusCode),
			)

			if r.ContentLength > 0 {
				span.SetAttributes(attribute.Int64("http.request_content_length", r.ContentLength))
			}
			if tw.bytesWritten > 0 {
				span.SetAttributes(attribute.Int("http.response_content_length", tw.bytesWritten))
			}

			// Mark span as error for 5xx responses.
			if tw.statusCode >= 500 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", tw.statusCode))
				span.RecordError(fmt.Errorf("HTTP %d", tw.statusCode))
			}
		})
	}
}

// SpanEnricher returns chi middleware that enriches the active span with tenant and user attributes.
// Register AFTER auth middleware so that tenant_id and user_id are available in context.
func SpanEnricher() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := trace.SpanFromContext(r.Context())

			if tenantID := auth.TenantFromContext(r.Context()); tenantID != "" {
				span.SetAttributes(AttrTenantID.String(tenantID))
			}

			if user := auth.UserFromContext(r.Context()); user != nil {
				span.SetAttributes(AttrUserID.String(user.ID))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// tracingResponseWriter wraps http.ResponseWriter to capture status code.
type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (w *tracingResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *tracingResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

func (w *tracingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// propagationHTTPCarrier adapts http.Header to the OTel TextMapCarrier interface.
type propagationHTTPCarrier http.Header

func (c propagationHTTPCarrier) Get(key string) string {
	return http.Header(c).Get(key)
}

func (c propagationHTTPCarrier) Set(key, value string) {
	http.Header(c).Set(key, value)
}

func (c propagationHTTPCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		return fwd
	}
	return "http"
}
