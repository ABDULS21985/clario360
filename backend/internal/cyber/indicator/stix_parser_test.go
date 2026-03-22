package indicator

import (
	"testing"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestParseIndicatorPatterns_Single(t *testing.T) {
	results := parseIndicatorPatterns("[ipv4-addr:value = '1.2.3.4']")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].indicatorType != model.IndicatorTypeIP || results[0].value != "1.2.3.4" {
		t.Fatalf("unexpected result: %+v", results[0])
	}
}

func TestParseIndicatorPatterns_ANDCompound(t *testing.T) {
	pattern := "[ipv4-addr:value = '10.0.0.1'] AND [domain-name:value = 'evil.com']"
	results := parseIndicatorPatterns(pattern)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].indicatorType != model.IndicatorTypeIP || results[0].value != "10.0.0.1" {
		t.Errorf("first result: %+v", results[0])
	}
	if results[1].indicatorType != model.IndicatorTypeDomain || results[1].value != "evil.com" {
		t.Errorf("second result: %+v", results[1])
	}
}

func TestParseIndicatorPatterns_ORCompound(t *testing.T) {
	pattern := "[ipv4-addr:value = '10.0.0.1'] OR [ipv4-addr:value = '10.0.0.2']"
	results := parseIndicatorPatterns(pattern)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].value != "10.0.0.1" || results[1].value != "10.0.0.2" {
		t.Errorf("unexpected values: %s, %s", results[0].value, results[1].value)
	}
}

func TestParseIndicatorPatterns_CompoundANDOR(t *testing.T) {
	pattern := "([ipv4-addr:value = '1.1.1.1'] OR [ipv4-addr:value = '2.2.2.2']) AND [domain-name:value = 'bad.com']"
	results := parseIndicatorPatterns(pattern)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	types := map[model.IndicatorType]int{}
	for _, r := range results {
		types[r.indicatorType]++
	}
	if types[model.IndicatorTypeIP] != 2 || types[model.IndicatorTypeDomain] != 1 {
		t.Errorf("unexpected type distribution: %v", types)
	}
}

func TestParseIndicatorPatterns_CIDR(t *testing.T) {
	results := parseIndicatorPatterns("[ipv4-addr:value ISSUBSET '192.168.0.0/16']")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].indicatorType != model.IndicatorTypeCIDR || results[0].value != "192.168.0.0/16" {
		t.Fatalf("unexpected: %+v", results[0])
	}
}

func TestParseIndicatorPatterns_UserAgent(t *testing.T) {
	results := parseIndicatorPatterns("[network-traffic:extensions.'http-request-ext'.request_header.'User-Agent' = 'EvilBot/1.0']")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].indicatorType != model.IndicatorTypeUserAgent || results[0].value != "EvilBot/1.0" {
		t.Fatalf("unexpected: %+v", results[0])
	}
}

func TestParseIndicatorPatterns_Dedup(t *testing.T) {
	pattern := "[ipv4-addr:value = '1.2.3.4'] AND [ipv4-addr:value = '1.2.3.4']"
	results := parseIndicatorPatterns(pattern)
	if len(results) != 1 {
		t.Fatalf("expected dedup to 1 result, got %d", len(results))
	}
}

func TestParseIndicatorPatterns_Empty(t *testing.T) {
	results := parseIndicatorPatterns("")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestParseIndicatorPatterns_Garbage(t *testing.T) {
	results := parseIndicatorPatterns("this is not a stix pattern")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestParseIndicatorPattern_BackwardCompat(t *testing.T) {
	iType, value, err := parseIndicatorPattern("[domain-name:value = 'test.com']")
	if err != nil {
		t.Fatal(err)
	}
	if iType != model.IndicatorTypeDomain || value != "test.com" {
		t.Fatalf("unexpected: %s %s", iType, value)
	}
}

func TestParseIndicatorPattern_UnsupportedReturnsError(t *testing.T) {
	_, _, err := parseIndicatorPattern("garbage")
	if err == nil {
		t.Fatal("expected error for unsupported pattern")
	}
}

func TestParseSTIXBundle_CompoundPattern(t *testing.T) {
	payload := []byte(`{
		"type": "bundle",
		"objects": [
			{
				"type": "indicator",
				"id": "indicator--1",
				"pattern": "[ipv4-addr:value = '10.0.0.1'] AND [domain-name:value = 'evil.com']",
				"name": "Compound IOC"
			}
		]
	}`)
	bundle, err := ParseSTIXBundle(payload, "osint")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Indicators) != 2 {
		t.Fatalf("expected 2 indicators from compound pattern, got %d", len(bundle.Indicators))
	}
	if bundle.Indicators[0].Indicator.Type != model.IndicatorTypeIP {
		t.Errorf("first indicator type: %s", bundle.Indicators[0].Indicator.Type)
	}
	if bundle.Indicators[1].Indicator.Type != model.IndicatorTypeDomain {
		t.Errorf("second indicator type: %s", bundle.Indicators[1].Indicator.Type)
	}
}

func TestParseSTIXBundle_Hashes(t *testing.T) {
	payload := []byte(`{
		"type": "bundle",
		"objects": [
			{
				"type": "indicator",
				"id": "indicator--2",
				"pattern": "[file:hashes.'MD5' = 'd41d8cd98f00b204e9800998ecf8427e'] AND [file:hashes.'SHA-256' = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855']",
				"name": "Multi-hash"
			}
		]
	}`)
	bundle, err := ParseSTIXBundle(payload, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Indicators) != 2 {
		t.Fatalf("expected 2, got %d", len(bundle.Indicators))
	}
}
