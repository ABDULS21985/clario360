package feed

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/clario360/platform/internal/cyber/cti/feed/adapters"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestSTIXAdapterParse(t *testing.T) {
	raw, err := os.ReadFile(testdataPath("stix_bundle.json"))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	adapter := adapters.NewSTIXAdapter()
	indicators, err := adapter.Parse(context.Background(), raw)
	if err != nil {
		t.Fatalf("parse stix: %v", err)
	}

	if len(indicators) < 50 {
		t.Errorf("expected >= 50 indicators, got %d", len(indicators))
	}

	// Verify first indicator
	first := indicators[0]
	if first.IOCType != "ip" {
		t.Errorf("first indicator ioc_type: want ip, got %s", first.IOCType)
	}
	if first.IOCValue != "10.55.100.1" {
		t.Errorf("first indicator ioc_value: want 10.55.100.1, got %s", first.IOCValue)
	}
	if first.SeverityCode != "high" {
		t.Errorf("first indicator severity: want high, got %s", first.SeverityCode)
	}
}

func TestCSVAdapterParse(t *testing.T) {
	csv := `type,value,severity,confidence,country,title
ip,10.99.1.1,high,0.85,ru,Test IP Indicator
domain,evil-test.example.net,medium,0.70,,Test Domain
hash_sha256,abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234,critical,0.95,cn,Test Hash
`
	adapter := adapters.NewCSVAdapter()
	indicators, err := adapter.Parse(context.Background(), []byte(csv))
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(indicators) != 3 {
		t.Fatalf("expected 3 indicators, got %d", len(indicators))
	}
	if indicators[0].IOCType != "ip" || indicators[0].IOCValue != "10.99.1.1" {
		t.Errorf("indicator 0: want ip/10.99.1.1, got %s/%s", indicators[0].IOCType, indicators[0].IOCValue)
	}
	if indicators[2].SeverityCode != "critical" {
		t.Errorf("indicator 2 severity: want critical, got %s", indicators[2].SeverityCode)
	}
}

func TestGenericJSONAdapterParse(t *testing.T) {
	raw := `[{"id":"test-1","title":"Test Indicator","severity":"high","confidence":0.9,"ioc_type":"domain","ioc_value":"malware.example.net"}]`
	adapter := adapters.NewGenericJSONAdapter()
	indicators, err := adapter.Parse(context.Background(), []byte(raw))
	if err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if len(indicators) != 1 {
		t.Fatalf("expected 1 indicator, got %d", len(indicators))
	}
	if indicators[0].Title != "Test Indicator" {
		t.Errorf("title: want 'Test Indicator', got '%s'", indicators[0].Title)
	}
}

func TestNormalizerFallback(t *testing.T) {
	n := NewNormalizer()
	raw := `[{"id":"fb-1","title":"Fallback","severity":"low","confidence":0.5,"ioc_type":"ip","ioc_value":"10.1.2.3"}]`
	indicators, err := n.Normalize(context.Background(), "unknown_type", []byte(raw))
	if err != nil {
		t.Fatalf("normalize with unknown type: %v", err)
	}
	if len(indicators) != 1 {
		t.Errorf("expected 1 indicator via fallback, got %d", len(indicators))
	}
}

func TestDevGeoResolver(t *testing.T) {
	r := NewDevGeoResolver()

	loc, err := r.Resolve("10.55.100.1")
	if err != nil {
		t.Fatalf("resolve 10.55.*: %v", err)
	}
	if loc == nil || loc.Country != "ru" {
		t.Errorf("10.55.* should resolve to ru, got %v", loc)
	}

	loc, err = r.Resolve("10.39.1.1")
	if err != nil {
		t.Fatalf("resolve 10.39.*: %v", err)
	}
	if loc == nil || loc.Country != "cn" {
		t.Errorf("10.39.* should resolve to cn, got %v", loc)
	}

	loc, err = r.Resolve("192.168.1.1")
	if err != nil {
		t.Fatalf("unexpected error for unknown IP: %v", err)
	}
	if loc != nil {
		t.Errorf("unknown IP should return nil, got %v", loc)
	}
}

func TestEnricherSeverityBoost(t *testing.T) {
	e := NewEnricher(NewDevGeoResolver())

	ind := &adapters.NormalizedIndicator{
		IOCType:         "ip",
		IOCValue:        "10.55.1.1",
		SeverityCode:    "medium",
		CategoryCode:    "apt",
		ConfidenceScore: 0.9,
	}

	e.Enrich(ind)

	if ind.SeverityCode != "high" {
		t.Errorf("expected severity boost to high, got %s", ind.SeverityCode)
	}
	if ind.OriginCountryCode != "ru" {
		t.Errorf("expected geo-resolve to ru, got %s", ind.OriginCountryCode)
	}
}
