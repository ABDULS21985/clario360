package tracing

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracerConfig holds tracing configuration.
type TracerConfig struct {
	// Enabled controls whether tracing is active. If false, a noop TracerProvider is used.
	Enabled bool
	// Endpoint is the OTLP gRPC endpoint (e.g., "jaeger:4317").
	Endpoint string
	// ServiceName identifies the service in traces.
	ServiceName string
	// Version is the service version.
	Version string
	// Environment is "production", "staging", or "development".
	Environment string
	// SampleRate is the sampling ratio (0.0 to 1.0). Production default: 0.1 (10%).
	SampleRate float64
	// Insecure controls whether to use TLS for the OTLP exporter. True for dev.
	Insecure bool
}

// InitTracer sets up the OpenTelemetry TracerProvider.
//
// Returns the TracerProvider, a shutdown function, and any error.
// The shutdown function MUST be called on service termination to flush buffered spans.
func InitTracer(ctx context.Context, cfg TracerConfig) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if !cfg.Enabled {
		noopShutdown := func(context.Context) error { return nil }
		noopProvider := noop.NewTracerProvider()
		otel.SetTracerProvider(noopProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return nil, noopShutdown, nil
	}

	// Build resource with service identity.
	hostname, _ := os.Hostname()
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.Version),
			attribute.String("deployment.environment", cfg.Environment),
			attribute.String("host.name", hostname),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("creating trace resource: %w", err)
	}

	// Create exporter.
	var exporter sdktrace.SpanExporter
	if cfg.Endpoint != "" {
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exp, expErr := otlptracegrpc.New(ctx, opts...)
		if expErr != nil {
			return nil, nil, fmt.Errorf("creating OTLP exporter: %w", expErr)
		}
		exporter = exp
	} else {
		// No endpoint — return noop to avoid failing.
		noopShutdown := func(context.Context) error { return nil }
		noopProvider := noop.NewTracerProvider()
		otel.SetTracerProvider(noopProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return nil, noopShutdown, nil
	}

	// Create sampler.
	var sampler sdktrace.Sampler
	isDev := cfg.Environment == "development" || cfg.Environment == "dev"
	if isDev {
		sampler = sdktrace.AlwaysSample()
	} else {
		rate := cfg.SampleRate
		if rate <= 0 {
			rate = 0.1
		}
		if rate > 1.0 {
			rate = 1.0
		}
		// ParentBased ensures trace continuity: if parent is sampled, child is always sampled.
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(rate))
	}

	// Create TracerProvider.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set globals.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return tp.Shutdown(shutdownCtx)
	}

	return tp, shutdown, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
