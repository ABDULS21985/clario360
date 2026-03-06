package expression

import (
	"strings"
	"testing"
)

func TestVariableResolver_SimpleVariable(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"variables": map[string]interface{}{
			"name":   "John",
			"count":  int64(42),
			"active": true,
		},
	}

	tests := []struct {
		name     string
		config   interface{}
		expected interface{}
	}{
		{
			name:     "resolve string variable",
			config:   "${variables.name}",
			expected: "John",
		},
		{
			name:     "resolve numeric variable preserves type",
			config:   "${variables.count}",
			expected: int64(42),
		},
		{
			name:     "resolve boolean variable preserves type",
			config:   "${variables.active}",
			expected: true,
		},
		{
			name:     "mixed string interpolation",
			config:   "Hello, ${variables.name}!",
			expected: "Hello, John!",
		},
		{
			name:     "multiple placeholders in string",
			config:   "${variables.name} has ${variables.count} items",
			expected: "John has 42 items",
		},
		{
			name:     "no placeholders - passthrough",
			config:   "plain string",
			expected: "plain string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.config, context)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("Resolve(%q) = %v (%T), want %v (%T)", tt.config, result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestVariableResolver_StepOutput(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"steps": map[string]interface{}{
			"triage": map[string]interface{}{
				"output": map[string]interface{}{
					"is_valid": true,
					"severity": "critical",
					"score":    int64(95),
				},
			},
		},
	}

	tests := []struct {
		name     string
		config   interface{}
		expected interface{}
	}{
		{
			name:     "step output boolean",
			config:   "${steps.triage.output.is_valid}",
			expected: true,
		},
		{
			name:     "step output string",
			config:   "${steps.triage.output.severity}",
			expected: "critical",
		},
		{
			name:     "step output number",
			config:   "${steps.triage.output.score}",
			expected: int64(95),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.config, context)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("Resolve() = %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestVariableResolver_TriggerData(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"trigger": map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "alert-123",
				"type": "security",
			},
		},
	}

	tests := []struct {
		name     string
		config   interface{}
		expected interface{}
	}{
		{
			name:     "trigger data id",
			config:   "${trigger.data.id}",
			expected: "alert-123",
		},
		{
			name:     "trigger data type",
			config:   "${trigger.data.type}",
			expected: "security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.config, context)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("Resolve() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVariableResolver_NestedMapResolution(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"variables": map[string]interface{}{
			"host":    "example.com",
			"port":    int64(8080),
			"api_key": "secret-key",
		},
	}

	config := map[string]interface{}{
		"url":     "https://${variables.host}:${variables.port}/api",
		"headers": map[string]interface{}{"Authorization": "Bearer ${variables.api_key}"},
		"timeout": int64(30),
	}

	result, err := resolver.Resolve(config, context)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	if resultMap["url"] != "https://example.com:8080/api" {
		t.Errorf("url = %v, want https://example.com:8080/api", resultMap["url"])
	}

	headers, ok := resultMap["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected headers to be a map, got %T", resultMap["headers"])
	}
	if headers["Authorization"] != "Bearer secret-key" {
		t.Errorf("Authorization = %v, want Bearer secret-key", headers["Authorization"])
	}

	if resultMap["timeout"] != int64(30) {
		t.Errorf("timeout = %v, want 30", resultMap["timeout"])
	}
}

func TestVariableResolver_SliceResolution(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"variables": map[string]interface{}{
			"item1": "alpha",
			"item2": "beta",
		},
	}

	config := []interface{}{
		"${variables.item1}",
		"static",
		"${variables.item2}",
	}

	result, err := resolver.Resolve(config, context)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	resultSlice, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected slice result, got %T", result)
	}

	if len(resultSlice) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(resultSlice))
	}
	if resultSlice[0] != "alpha" {
		t.Errorf("element 0 = %v, want alpha", resultSlice[0])
	}
	if resultSlice[1] != "static" {
		t.Errorf("element 1 = %v, want static", resultSlice[1])
	}
	if resultSlice[2] != "beta" {
		t.Errorf("element 2 = %v, want beta", resultSlice[2])
	}
}

func TestVariableResolver_MissingVariable(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"variables": map[string]interface{}{
			"name": "John",
		},
	}

	_, err := resolver.Resolve("${variables.nonexistent}", context)
	if err == nil {
		t.Fatal("expected error for missing variable")
	}
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestVariableResolver_PathTraversal(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"variables": map[string]interface{}{
			"name": "John",
		},
	}

	tests := []struct {
		name   string
		config string
	}{
		{
			name:   "double dot traversal",
			config: "${variables..name}",
		},
		{
			name:   "proto traversal",
			config: "${__proto__.name}",
		},
		{
			name:   "constructor traversal",
			config: "${constructor.name}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolver.Resolve(tt.config, context)
			if err == nil {
				t.Errorf("expected error for path traversal: %s", tt.config)
			}
		})
	}
}

func TestVariableResolver_MaxDepthExceeded(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"variables": map[string]interface{}{
			"x": "value",
		},
	}

	// Create deeply nested maps to exceed depth.
	config := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{
					"d": map[string]interface{}{
						"e": map[string]interface{}{
							"f": map[string]interface{}{
								"val": "${variables.x}",
							},
						},
					},
				},
			},
		},
	}

	_, err := resolver.Resolve(config, context)
	if err == nil {
		t.Fatal("expected error for max depth exceeded")
	}
	if !strings.Contains(err.Error(), "depth") {
		t.Errorf("expected depth-related error, got: %v", err)
	}
}

func TestVariableResolver_NonStringPreserved(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"variables": map[string]interface{}{},
	}

	tests := []struct {
		name     string
		config   interface{}
		expected interface{}
	}{
		{
			name:     "integer preserved",
			config:   int64(42),
			expected: int64(42),
		},
		{
			name:     "float preserved",
			config:   float64(3.14),
			expected: float64(3.14),
		},
		{
			name:     "bool preserved",
			config:   true,
			expected: true,
		},
		{
			name:     "nil preserved",
			config:   nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.config, context)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("Resolve() = %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestVariableResolver_ResolvePath(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"variables": map[string]interface{}{
			"alert_id": "alert-456",
		},
		"steps": map[string]interface{}{
			"triage": map[string]interface{}{
				"output": map[string]interface{}{
					"is_valid": true,
				},
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "resolve variable path",
			path:     "variables.alert_id",
			expected: "alert-456",
		},
		{
			name:     "resolve step output path",
			path:     "steps.triage.output.is_valid",
			expected: true,
		},
		{
			name:    "missing path returns error",
			path:    "variables.nonexistent",
			wantErr: true,
		},
		{
			name:    "empty path returns error",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolvePath(tt.path, context)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolvePath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("ResolvePath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestVariableResolver_UnterminatedPlaceholder(t *testing.T) {
	resolver := NewVariableResolver()

	context := map[string]interface{}{
		"variables": map[string]interface{}{
			"x": "val",
		},
	}

	_, err := resolver.Resolve("${variables.x", context)
	if err == nil {
		t.Fatal("expected error for unterminated placeholder")
	}
}
