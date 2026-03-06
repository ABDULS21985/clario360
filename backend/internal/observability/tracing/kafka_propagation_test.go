package tracing

import (
	"sort"
	"testing"

	"github.com/IBM/sarama"
)

func TestSaramaHeaderCarrier_SetGet(t *testing.T) {
	headers := []sarama.RecordHeader{}
	carrier := NewSaramaHeaderCarrier(&headers)

	carrier.Set("traceparent", "00-abc123-def456-01")

	got := carrier.Get("traceparent")
	if got != "00-abc123-def456-01" {
		t.Errorf("Get(\"traceparent\") = %q, want %q", got, "00-abc123-def456-01")
	}
}

func TestSaramaHeaderCarrier_GetMissing(t *testing.T) {
	headers := []sarama.RecordHeader{}
	carrier := NewSaramaHeaderCarrier(&headers)

	got := carrier.Get("nonexistent")
	if got != "" {
		t.Errorf("Get(\"nonexistent\") = %q, want empty string", got)
	}
}

func TestSaramaHeaderCarrier_Keys(t *testing.T) {
	headers := []sarama.RecordHeader{
		{Key: []byte("traceparent"), Value: []byte("val1")},
		{Key: []byte("tracestate"), Value: []byte("val2")},
		{Key: []byte("custom-header"), Value: []byte("val3")},
	}
	carrier := NewSaramaHeaderCarrier(&headers)

	keys := carrier.Keys()
	if len(keys) != 3 {
		t.Fatalf("Keys() returned %d keys, want 3", len(keys))
	}

	sort.Strings(keys)
	expected := []string{"custom-header", "traceparent", "tracestate"}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("Keys()[%d] = %q, want %q", i, k, expected[i])
		}
	}
}

func TestSaramaHeaderCarrier_Upsert(t *testing.T) {
	headers := []sarama.RecordHeader{}
	carrier := NewSaramaHeaderCarrier(&headers)

	// Set initial value.
	carrier.Set("traceparent", "original-value")
	if got := carrier.Get("traceparent"); got != "original-value" {
		t.Fatalf("initial Get(\"traceparent\") = %q, want %q", got, "original-value")
	}

	// Update the same key.
	carrier.Set("traceparent", "updated-value")
	if got := carrier.Get("traceparent"); got != "updated-value" {
		t.Errorf("after upsert Get(\"traceparent\") = %q, want %q", got, "updated-value")
	}

	// Verify no duplicate headers were created.
	keys := carrier.Keys()
	count := 0
	for _, k := range keys {
		if k == "traceparent" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 'traceparent' header after upsert, got %d", count)
	}
}

func TestSaramaHeaderCarrier_MultipleHeaders(t *testing.T) {
	headers := []sarama.RecordHeader{}
	carrier := NewSaramaHeaderCarrier(&headers)

	carrier.Set("traceparent", "tp-value")
	carrier.Set("tracestate", "ts-value")

	if got := carrier.Get("traceparent"); got != "tp-value" {
		t.Errorf("Get(\"traceparent\") = %q, want %q", got, "tp-value")
	}
	if got := carrier.Get("tracestate"); got != "ts-value" {
		t.Errorf("Get(\"tracestate\") = %q, want %q", got, "ts-value")
	}
}
