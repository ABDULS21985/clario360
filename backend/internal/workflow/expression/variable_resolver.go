package expression

import (
	"fmt"
	"strings"
)

// VariableResolver resolves ${...} placeholders in workflow step configurations.
// It recursively walks config values and substitutes variable references
// against a provided context map.
type VariableResolver struct {
	maxDepth int // max resolution depth to prevent circular references
}

// NewVariableResolver creates a new VariableResolver with safe defaults.
func NewVariableResolver() *VariableResolver {
	return &VariableResolver{
		maxDepth: 5,
	}
}

// Resolve recursively walks config and substitutes ${...} references.
// context contains: {"variables": {...}, "steps": {"id": {"output": {...}}}, "trigger": {"data": {...}}}
// It handles:
//   - string values: substitutes ${...} placeholders
//   - map values: recursively resolves all values
//   - slice values: recursively resolves all elements
//   - other types: returned as-is
func (r *VariableResolver) Resolve(config interface{}, context map[string]interface{}) (interface{}, error) {
	return r.resolveValue(config, context, 0)
}

// ResolvePath resolves a single dotted path like "variables.alert_id" or
// "steps.triage.output.is_valid" against the provided context.
func (r *VariableResolver) ResolvePath(path string, context map[string]interface{}) (interface{}, error) {
	sanitizer := NewSanitizer()
	if err := sanitizer.SanitizePath(path); err != nil {
		return nil, err
	}

	segments := strings.Split(path, ".")
	if len(segments) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	var current interface{} = context
	for _, seg := range segments {
		if seg == "" {
			return nil, fmt.Errorf("empty segment in path: %q", path)
		}
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot resolve path segment %q in path %q: not a map", seg, path)
		}
		val, exists := m[seg]
		if !exists {
			return nil, fmt.Errorf("variable not found: %q (segment %q does not exist)", path, seg)
		}
		current = val
	}
	return current, nil
}

// resolveValue is the recursive workhorse that resolves config values.
func (r *VariableResolver) resolveValue(value interface{}, context map[string]interface{}, depth int) (interface{}, error) {
	if depth > r.maxDepth {
		return nil, fmt.Errorf("maximum resolution depth of %d exceeded (possible circular reference)", r.maxDepth)
	}

	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case string:
		return r.resolveString(v, context, depth)
	case map[string]interface{}:
		return r.resolveMap(v, context, depth)
	case []interface{}:
		return r.resolveSlice(v, context, depth)
	default:
		// Non-string, non-container values (int, float, bool, etc.) are returned as-is.
		return value, nil
	}
}

// resolveString substitutes all ${...} placeholders in a string value.
// If the entire string is a single placeholder (e.g., "${variables.count}"),
// the resolved value is returned with its original type (not stringified).
// If the string contains mixed content (e.g., "Hello ${variables.name}!"),
// the resolved value is interpolated as a string.
func (r *VariableResolver) resolveString(s string, context map[string]interface{}, depth int) (interface{}, error) {
	// Quick check: does this string contain any placeholders?
	if !strings.Contains(s, "${") {
		return s, nil
	}

	// Check if the entire string is a single placeholder.
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		inner := trimmed[2 : len(trimmed)-1]
		// Make sure there are no nested ${...} inside - it should be a simple path.
		if !strings.Contains(inner, "${") && !strings.Contains(inner, "}") {
			val, err := r.ResolvePath(strings.TrimSpace(inner), context)
			if err != nil {
				return nil, err
			}
			return val, nil
		}
	}

	// Mixed content: scan and replace all ${...} placeholders.
	var result strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '$' && s[i+1] == '{' {
			// Find the closing brace.
			j := strings.Index(s[i:], "}")
			if j == -1 {
				return nil, fmt.Errorf("unterminated placeholder in string: %q", s)
			}
			path := strings.TrimSpace(s[i+2 : i+j])
			val, err := r.ResolvePath(path, context)
			if err != nil {
				return nil, err
			}
			result.WriteString(fmt.Sprintf("%v", val))
			i = i + j + 1
		} else {
			result.WriteByte(s[i])
			i++
		}
	}

	return result.String(), nil
}

// resolveMap resolves all values in a map.
func (r *VariableResolver) resolveMap(m map[string]interface{}, context map[string]interface{}, depth int) (interface{}, error) {
	resolved := make(map[string]interface{}, len(m))
	for key, val := range m {
		resolvedVal, err := r.resolveValue(val, context, depth+1)
		if err != nil {
			return nil, fmt.Errorf("resolving key %q: %w", key, err)
		}
		resolved[key] = resolvedVal
	}
	return resolved, nil
}

// resolveSlice resolves all elements in a slice.
func (r *VariableResolver) resolveSlice(slice []interface{}, context map[string]interface{}, depth int) (interface{}, error) {
	resolved := make([]interface{}, len(slice))
	for i, val := range slice {
		resolvedVal, err := r.resolveValue(val, context, depth+1)
		if err != nil {
			return nil, fmt.Errorf("resolving index %d: %w", i, err)
		}
		resolved[i] = resolvedVal
	}
	return resolved, nil
}
