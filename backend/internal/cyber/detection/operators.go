package detection

import (
	"encoding/base64"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

const (
	operatorExact      = "eq"
	operatorIn         = "in"
	operatorContains   = "contains"
	operatorStartsWith = "startswith"
	operatorEndsWith   = "endswith"
	operatorRegex      = "re"
	operatorGT         = "gt"
	operatorGTE        = "gte"
	operatorLT         = "lt"
	operatorLTE        = "lte"
	operatorCIDR       = "cidr"
	operatorExists     = "exists"
	operatorAll        = "all"
	operatorBase64     = "base64"
)

var validEventFields = map[string]struct{}{
	"id":             {},
	"timestamp":      {},
	"source":         {},
	"type":           {},
	"severity":       {},
	"source_ip":      {},
	"dest_ip":        {},
	"dest_port":      {},
	"protocol":       {},
	"username":       {},
	"user":           {},
	"process":        {},
	"parent_process": {},
	"command_line":   {},
	"file_path":      {},
	"file_hash":      {},
	"asset_id":       {},
}

// CompiledFieldCondition is a pre-validated field/operator/value matcher.
type CompiledFieldCondition struct {
	FieldPath string
	Operator  string
	Value     interface{}
	Regex     *regexp.Regexp
	CIDR      *net.IPNet
}

// CompiledSelection is a named group of field conditions ANDed together.
type CompiledSelection struct {
	Name       string
	Conditions []*CompiledFieldCondition
}

// CompileSelection compiles a Sigma-like selection object into matchable criteria.
func CompileSelection(name string, selection map[string]interface{}) (*CompiledSelection, error) {
	compiled := &CompiledSelection{
		Name:       name,
		Conditions: make([]*CompiledFieldCondition, 0, len(selection)),
	}
	for rawKey, rawValue := range selection {
		fieldPath, operator, err := parseFieldOperator(rawKey)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if err := validateFieldPath(fieldPath); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		criterion := &CompiledFieldCondition{
			FieldPath: fieldPath,
			Operator:  operator,
			Value:     rawValue,
		}
		if operator == operatorRegex {
			pattern, ok := rawValue.(string)
			if !ok {
				return nil, fmt.Errorf("%s: regex operator requires string pattern", rawKey)
			}
			if len(pattern) > 2048 {
				return nil, fmt.Errorf("%s: regex pattern exceeds 2048 characters", rawKey)
			}
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid regex: %w", rawKey, err)
			}
			criterion.Regex = regex
		}
		if operator == operatorCIDR {
			cidr, ok := rawValue.(string)
			if !ok {
				return nil, fmt.Errorf("%s: cidr operator requires string value", rawKey)
			}
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid cidr %q: %w", rawKey, cidr, err)
			}
			criterion.CIDR = network
		}
		compiled.Conditions = append(compiled.Conditions, criterion)
	}
	return compiled, nil
}

// EvaluateSelection returns whether all conditions in the selection matched.
// It also returns the list of field paths that matched for explainability.
func EvaluateSelection(selection *CompiledSelection, event *model.SecurityEvent) (bool, []string) {
	matched := make([]string, 0, len(selection.Conditions))
	for _, condition := range selection.Conditions {
		ok := condition.Match(event)
		if !ok {
			return false, nil
		}
		matched = append(matched, condition.FieldPath)
	}
	return true, matched
}

// Match evaluates a compiled field condition against a single event.
func (c *CompiledFieldCondition) Match(event *model.SecurityEvent) bool {
	fieldValue, found := resolveField(event, c.FieldPath)
	if c.Operator == operatorExists {
		expected, ok := toBool(c.Value)
		if !ok {
			return false
		}
		actual := found && !isZeroValue(fieldValue)
		return actual == expected
	}
	if !found {
		return false
	}

	switch c.Operator {
	case operatorExact:
		return exactMatch(fieldValue, c.Value)
	case operatorIn:
		values, ok := toSlice(c.Value)
		if !ok {
			return false
		}
		for _, value := range values {
			if exactMatch(fieldValue, value) {
				return true
			}
		}
		return false
	case operatorContains:
		return strings.Contains(strings.ToLower(toString(fieldValue)), strings.ToLower(toString(c.Value)))
	case operatorStartsWith:
		return strings.HasPrefix(strings.ToLower(toString(fieldValue)), strings.ToLower(toString(c.Value)))
	case operatorEndsWith:
		return strings.HasSuffix(strings.ToLower(toString(fieldValue)), strings.ToLower(toString(c.Value)))
	case operatorRegex:
		text := toString(fieldValue)
		done := make(chan bool, 1)
		go func() {
			done <- c.Regex != nil && c.Regex.MatchString(text)
		}()
		select {
		case matched := <-done:
			return matched
		case <-time.After(time.Second):
			return false
		}
	case operatorGT:
		left, lok := toFloat64(fieldValue)
		right, rok := toFloat64(c.Value)
		return lok && rok && left > right
	case operatorGTE:
		left, lok := toFloat64(fieldValue)
		right, rok := toFloat64(c.Value)
		return lok && rok && left >= right
	case operatorLT:
		left, lok := toFloat64(fieldValue)
		right, rok := toFloat64(c.Value)
		return lok && rok && left < right
	case operatorLTE:
		left, lok := toFloat64(fieldValue)
		right, rok := toFloat64(c.Value)
		return lok && rok && left <= right
	case operatorCIDR:
		ip := net.ParseIP(toString(fieldValue))
		return ip != nil && c.CIDR != nil && c.CIDR.Contains(ip)
	case operatorAll:
		fieldValues, ok := toSlice(fieldValue)
		if !ok {
			return false
		}
		required, ok := toSlice(c.Value)
		if !ok {
			return false
		}
		for _, req := range required {
			foundReq := false
			for _, field := range fieldValues {
				if exactMatch(field, req) {
					foundReq = true
					break
				}
			}
			if !foundReq {
				return false
			}
		}
		return true
	case operatorBase64:
		decoded, err := base64.StdEncoding.DecodeString(toString(fieldValue))
		if err != nil {
			return false
		}
		return strings.Contains(strings.ToLower(string(decoded)), strings.ToLower(toString(c.Value)))
	default:
		return false
	}
}

func parseFieldOperator(raw string) (string, string, error) {
	parts := strings.Split(raw, "|")
	fieldPath := parts[0]
	operator := operatorExact
	if len(parts) > 1 {
		operator = strings.ToLower(parts[1])
	}
	switch operator {
	case operatorExact, operatorIn, operatorContains, operatorStartsWith, operatorEndsWith,
		operatorRegex, operatorGT, operatorGTE, operatorLT, operatorLTE,
		operatorCIDR, operatorExists, operatorAll, operatorBase64:
		return fieldPath, operator, nil
	default:
		return "", "", fmt.Errorf("unsupported operator %q", operator)
	}
}

func validateFieldPath(fieldPath string) error {
	if strings.HasPrefix(fieldPath, "raw.") {
		if strings.Count(fieldPath, ".") > 5 {
			return fmt.Errorf("raw field traversal exceeds 5 nested levels")
		}
		return nil
	}
	if _, ok := validEventFields[fieldPath]; ok {
		return nil
	}
	return fmt.Errorf("unsupported field %q", fieldPath)
}

// resolveField resolves a known event field or raw.{path} value.
func resolveField(event *model.SecurityEvent, fieldPath string) (interface{}, bool) {
	switch fieldPath {
	case "id":
		return event.ID.String(), event.ID != [16]byte{}
	case "timestamp":
		return event.Timestamp.Unix(), !event.Timestamp.IsZero()
	case "source":
		return event.Source, event.Source != ""
	case "type":
		return event.Type, event.Type != ""
	case "severity":
		return string(event.Severity), event.Severity != ""
	case "source_ip":
		if event.SourceIP != nil {
			return *event.SourceIP, *event.SourceIP != ""
		}
		return nil, false
	case "dest_ip":
		if event.DestIP != nil {
			return *event.DestIP, *event.DestIP != ""
		}
		return nil, false
	case "dest_port":
		if event.DestPort != nil {
			return *event.DestPort, true
		}
		return nil, false
	case "protocol":
		if event.Protocol != nil {
			return *event.Protocol, *event.Protocol != ""
		}
		return nil, false
	case "username", "user":
		if event.Username != nil {
			return *event.Username, *event.Username != ""
		}
		return nil, false
	case "process":
		if event.Process != nil {
			return *event.Process, *event.Process != ""
		}
		return nil, false
	case "parent_process":
		if event.ParentProcess != nil {
			return *event.ParentProcess, *event.ParentProcess != ""
		}
		return nil, false
	case "command_line":
		if event.CommandLine != nil {
			return *event.CommandLine, *event.CommandLine != ""
		}
		return nil, false
	case "file_path":
		if event.FilePath != nil {
			return *event.FilePath, *event.FilePath != ""
		}
		return nil, false
	case "file_hash":
		if event.FileHash != nil {
			return *event.FileHash, *event.FileHash != ""
		}
		return nil, false
	case "asset_id":
		if event.AssetID != nil {
			return event.AssetID.String(), true
		}
		return nil, false
	}

	raw := event.RawMap()
	if strings.HasPrefix(fieldPath, "raw.") {
		fieldPath = strings.TrimPrefix(fieldPath, "raw.")
	}
	parts := strings.Split(fieldPath, ".")
	var current interface{} = raw
	for _, part := range parts {
		nextMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = nextMap[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func exactMatch(left, right interface{}) bool {
	leftStr := toString(left)
	rightStr := toString(right)
	if leftStr != "" || rightStr != "" {
		return strings.EqualFold(leftStr, rightStr)
	}
	leftNum, lok := toFloat64(left)
	rightNum, rok := toFloat64(right)
	if lok && rok {
		return math.Abs(leftNum-rightNum) < 1e-9
	}
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
}

func toSlice(value interface{}) ([]interface{}, bool) {
	switch typed := value.(type) {
	case []interface{}:
		return typed, true
	case []string:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out, true
	case []int:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out, true
	default:
		return nil, false
	}
}

func toString(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	case fmt.Stringer:
		return typed.String()
	case jsonNumber:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

type jsonNumber interface {
	String() string
}

func toFloat64(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case string:
		n, err := strconv.ParseFloat(typed, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func toBool(value interface{}) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		b, err := strconv.ParseBool(typed)
		return b, err == nil
	default:
		return false, false
	}
}

func isZeroValue(value interface{}) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case int:
		return typed == 0
	case int64:
		return typed == 0
	case float64:
		return typed == 0
	case []interface{}:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	default:
		return fmt.Sprintf("%v", value) == ""
	}
}
