package detection

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestOperatorExactAndIn(t *testing.T) {
	port := 443
	event := model.SecurityEvent{DestPort: &port, Protocol: stringPtr("TCP")}

	exact := &CompiledFieldCondition{FieldPath: "protocol", Operator: operatorExact, Value: "tcp"}
	if !exact.Match(&event) {
		t.Fatal("expected exact match to be case-insensitive")
	}

	in := &CompiledFieldCondition{FieldPath: "dest_port", Operator: operatorIn, Value: []interface{}{80, 443, 8443}}
	if !in.Match(&event) {
		t.Fatal("expected IN operator to match port 443")
	}
}

func TestOperatorContainsRegexCIDRAndExists(t *testing.T) {
	sourceIP := "10.0.1.5"
	commandLine := "powershell -enc abc"
	event := model.SecurityEvent{
		SourceIP:    &sourceIP,
		CommandLine: &commandLine,
		RawEvent:    json.RawMessage(`{"custom_field":"cmd.exe /c whoami"}`),
	}

	contains := &CompiledFieldCondition{FieldPath: "command_line", Operator: operatorContains, Value: "ENC"}
	if !contains.Match(&event) {
		t.Fatal("expected contains operator to match")
	}

	regexSel, err := CompileSelection("selection", map[string]interface{}{"raw.custom_field|re": `cmd\.exe\s+/c\s+\w+`})
	if err != nil {
		t.Fatalf("CompileSelection returned error: %v", err)
	}
	matched, _ := EvaluateSelection(regexSel, &event)
	if !matched {
		t.Fatal("expected regex operator to match raw.custom_field")
	}

	cidrSel, err := CompileSelection("selection", map[string]interface{}{"source_ip|cidr": "10.0.1.0/24"})
	if err != nil {
		t.Fatalf("CompileSelection returned error: %v", err)
	}
	matched, _ = EvaluateSelection(cidrSel, &event)
	if !matched {
		t.Fatal("expected CIDR operator to match source_ip")
	}

	exists := &CompiledFieldCondition{FieldPath: "source_ip", Operator: operatorExists, Value: true}
	if !exists.Match(&event) {
		t.Fatal("expected exists operator to report true")
	}
}

func TestResolveFieldRawAndAssetID(t *testing.T) {
	assetID := uuid.New()
	event := model.SecurityEvent{
		AssetID:  &assetID,
		RawEvent: json.RawMessage(`{"nested":{"value":"ok"}}`),
	}
	value, ok := resolveField(&event, "raw.nested.value")
	if !ok || value != "ok" {
		t.Fatalf("expected raw nested field to resolve, got %v %v", value, ok)
	}
	value, ok = resolveField(&event, "asset_id")
	if !ok || value != assetID.String() {
		t.Fatalf("expected asset_id to resolve, got %v %v", value, ok)
	}
}
