package tracing

import (
	"context"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// SaramaHeaderCarrier adapts sarama.RecordHeader slice to OpenTelemetry's TextMapCarrier interface.
type SaramaHeaderCarrier struct {
	headers *[]sarama.RecordHeader
}

// NewSaramaHeaderCarrier creates a carrier for inject/extract on sarama message headers.
func NewSaramaHeaderCarrier(headers *[]sarama.RecordHeader) *SaramaHeaderCarrier {
	return &SaramaHeaderCarrier{headers: headers}
}

// Get finds a header by key and returns its value as a string.
func (c *SaramaHeaderCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set upserts a header. If the key exists, its value is updated; otherwise a new header is appended.
func (c *SaramaHeaderCarrier) Set(key, val string) {
	for i, h := range *c.headers {
		if string(h.Key) == key {
			(*c.headers)[i].Value = []byte(val)
			return
		}
	}
	*c.headers = append(*c.headers, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(val),
	})
}

// Keys returns all header key names.
func (c *SaramaHeaderCarrier) Keys() []string {
	keys := make([]string, len(*c.headers))
	for i, h := range *c.headers {
		keys[i] = string(h.Key)
	}
	return keys
}

// InjectTraceContext injects W3C TraceContext headers (traceparent, tracestate)
// into the Kafka message's headers. Called by the producer before publishing.
func InjectTraceContext(ctx context.Context, headers *[]sarama.RecordHeader) {
	carrier := NewSaramaHeaderCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// ExtractTraceContext extracts W3C TraceContext headers from Kafka message headers
// and returns a context with the propagated span context.
// If no trace headers are present, returns ctx unchanged (consumer creates a new root span).
func ExtractTraceContext(ctx context.Context, headers []*sarama.RecordHeader) context.Context {
	carrier := &readOnlySaramaCarrier{headers: headers}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// readOnlySaramaCarrier is a read-only carrier for extracting headers from consumed messages.
// sarama consumer messages use []*sarama.RecordHeader (pointer slice).
type readOnlySaramaCarrier struct {
	headers []*sarama.RecordHeader
}

func (c *readOnlySaramaCarrier) Get(key string) string {
	for _, h := range c.headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *readOnlySaramaCarrier) Set(string, string) {}

func (c *readOnlySaramaCarrier) Keys() []string {
	keys := make([]string, len(c.headers))
	for i, h := range c.headers {
		keys[i] = string(h.Key)
	}
	return keys
}

// Ensure interface compliance.
var _ propagation.TextMapCarrier = (*SaramaHeaderCarrier)(nil)
var _ propagation.TextMapCarrier = (*readOnlySaramaCarrier)(nil)
