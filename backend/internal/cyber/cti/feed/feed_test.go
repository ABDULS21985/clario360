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

func TestMISPAdapterParse(t *testing.T) {
	raw := `{"response":[{"Event":{"id":"evt-1","info":"MISP import","threat_level_id":"2","Tag":[{"name":"apt"}],"Attribute":[{"id":"attr-1","type":"ip-dst","value":"203.0.113.10","comment":"Known C2","timestamp":"1712102400","to_ids":true,"Tag":[{"name":"ransomware"}]}]}}]}`
	adapter := adapters.NewMISPAdapter()
	indicators, err := adapter.Parse(context.Background(), []byte(raw))
	if err != nil {
		t.Fatalf("parse misp: %v", err)
	}
	if len(indicators) != 1 {
		t.Fatalf("expected 1 indicator, got %d", len(indicators))
	}
	if indicators[0].IOCType != "ip" || indicators[0].IOCValue != "203.0.113.10" {
		t.Fatalf("unexpected indicator: %+v", indicators[0])
	}
	if indicators[0].SeverityCode != "high" {
		t.Fatalf("expected high severity, got %s", indicators[0].SeverityCode)
	}
}

func TestOTXAdapterParse(t *testing.T) {
	raw := `{"results":[{"id":"otx-1","indicator":"malicious.example","type":"domain","title":"OTX Pulse Indicator","created":"2026-03-01T00:00:00Z","modified":"2026-03-02T00:00:00Z","pulse_info":{"count":1,"pulses":[{"name":"Threat Pulse","description":"Credential harvesting infra","tags":["phishing"]}]}}]}`
	adapter := adapters.NewOTXAdapter()
	indicators, err := adapter.Parse(context.Background(), []byte(raw))
	if err != nil {
		t.Fatalf("parse otx: %v", err)
	}
	if len(indicators) != 1 {
		t.Fatalf("expected 1 indicator, got %d", len(indicators))
	}
	if indicators[0].IOCType != "domain" || indicators[0].IOCValue != "malicious.example" {
		t.Fatalf("unexpected indicator: %+v", indicators[0])
	}
	if indicators[0].SeverityCode != "high" {
		t.Fatalf("expected high severity, got %s", indicators[0].SeverityCode)
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
