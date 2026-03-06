package expression

import (
	"fmt"
	"regexp"
	"strings"
)

// variablePattern matches ${path.to.value} expressions in strings and config values.
var variablePattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// VariableResolver resolves ${...} variable references in workflow step configurations
// against a data context. The context is typically structured as:
//
//	{"variables": {...}, "steps": {...}, "trigger": {"data": {...}}}
type VariableResolver struct {
	maxDepth int
}

// NewVariableResolver creates a new VariableResolver with safe defaults.
func NewVariableResolver() *VariableResolver {
	return &VariableResolver{
		maxDepth: 20,
	}
}

// Resolve walks an arbitrary config value (string, map, slice, or primitive) and
// replaces all ${...} references with their resolved values from the context.
// If a string is entirely a single ${...} expression, the resolved value retains
// its original type (e.g., a number stays a number). If mixed with literal text,
// the result is always a string.
func (r *VariableResolver) Resolve(config interface{}, context map[string]interface{}) (interface{}, error) {
	return r.resolveValue(config, context, 0)
}

// ResolvePath resolves a single dot-separated path against the context.
// For example, ResolvePath("variables.severity", ctx) returns the value at ctx["variables"]["severity"].
func (r *VariableResolver) ResolvePath(path string, context map[string]interface{}) (interface{}, error) {
	segments := strings.Split(path, ".")
	var current interface{} = context
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot resolve path segment %q: parent is not a map", seg)
		}
		val, exists := m[seg]
		if !exists {
			return nil, fmt.Errorf("path segment %q not found in context", seg)
		}
		current = val
	}
	return current, nil
}

// resolveValue recursively resolves variable references in a value.
func (r *VariableResolver) resolveValue(val interface{}, ctx map[string]interface{}, depth int) (interface{}, error) {
	if depth > r.maxDepth {
		return nil, fmt.Errorf("variable resolution exceeded maximum depth of %d", r.maxDepth)
	}

	switch v := val.(type) {
	case string:
		return r.resolveString(v, ctx)
	case map[string]interface{}:
		return r.resolveMap(v, ctx, depth)
	case []interface{}:
		return r.resolveSlice(v, ctx, depth)
	default:
		// Primitives (int, float64, bool, nil) pass through unchanged.
		return val, nil
	}
}

// resolveString handles variable substitution within a string value.
// If the string is exactly "${path}", the resolved value keeps its native type.
// If the string contains mixed content like "prefix-${path}-suffix", the result is a string.
func (r *VariableResolver) resolveString(s string, ctx map[string]interface{}) (interface{}, error) {
	// Fast path: no variables at all.
	if !strings.Contains(s, "${") {
		return s, nil
	}

	// Check if the entire string is a single variable reference.
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		inner := trimmed[2 : len(trimmed)-1]
		// Verify there are no additional ${ inside (which would mean it's not a simple reference).
		if !strings.Contains(inner, "${") {
			resolved, err := r.ResolvePath(inner, ctx)
			if err != nil {
				return nil, fmt.Errorf("resolving variable ${%s}: %w", inner, err)
			}
			return resolved, nil
		}
	}

	// Mixed content: replace all ${...} with string representations.
	var resolveErr error
	result := variablePattern.ReplaceAllStringFunc(s, func(match string) string {
		if resolveErr != nil {
			return match
		}
		path := match[2 : len(match)-1]
		resolved, err := r.ResolvePath(path, ctx)
		if err != nil {
			resolveErr = fmt.Errorf("resolving variable ${%s}: %w", path, err)
			return match
		}
		return fmt.Sprintf("%v", resolved)
	})
	if resolveErr != nil {
		return nil, resolveErr
	}
	return result, nil
}

// resolveMap resolves all values in a map recursively.
func (r *VariableResolver) resolveMap(m map[string]interface{}, ctx map[string]interface{}, depth int) (interface{}, error) {
	resolved := make(map[string]interface{}, len(m))
	for k, v := range m {
		rv, err := r.resolveValue(v, ctx, depth+1)
		if err != nil {
			return nil, fmt.Errorf("resolving map key %q: %w", k, err)
		}
		resolved[k] = rv
	}
	return resolved, nil
}

// resolveSlice resolves all elements in a slice recursively.
func (r *VariableResolver) resolveSlice(s []interface{}, ctx map[string]interface{}, depth int) (interface{}, error) {
	resolved := make([]interface{}, len(s))
	for i, v := range s {
		rv, err := r.resolveValue(v, ctx, depth+1)
		if err != nil {
			return nil, fmt.Errorf("resolving slice index %d: %w", i, err)
		}
		resolved[i] = rv
	}
	return resolved, nil
}
