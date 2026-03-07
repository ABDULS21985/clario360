package expression

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

type CompiledExpression struct {
	ast ExprNode
}

type evalState struct {
	start time.Time
	vars  map[string]any
}

func Compile(expression string) (*CompiledExpression, error) {
	if strings.TrimSpace(expression) == "" {
		return nil, fmt.Errorf("expression is required")
	}
	tokens, err := tokenize(expression)
	if err != nil {
		return nil, err
	}
	parser := newParser(tokens)
	ast, err := parser.parse()
	if err != nil {
		return nil, err
	}
	return &CompiledExpression{ast: ast}, nil
}

func (e *CompiledExpression) Evaluate(vars map[string]interface{}) (interface{}, error) {
	state := &evalState{
		start: time.Now(),
		vars:  normalizeVars(vars),
	}
	return state.eval(e.ast)
}

func (s *evalState) eval(node ExprNode) (any, error) {
	if time.Since(s.start) > 100*time.Millisecond {
		return nil, fmt.Errorf("expression evaluation timed out")
	}
	switch typed := node.(type) {
	case literalNode:
		return typed.value, nil
	case variableNode:
		value, ok := s.vars[strings.ToLower(typed.name)]
		if !ok {
			return nil, nil
		}
		return value, nil
	case unaryNode:
		value, err := s.eval(typed.expr)
		if err != nil {
			return nil, err
		}
		switch typed.op {
		case "NOT", "!":
			if value == nil {
				return nil, nil
			}
			boolValue, ok := toBool(value)
			if !ok {
				return nil, fmt.Errorf("NOT expects boolean input")
			}
			return !boolValue, nil
		case "-":
			if value == nil {
				return nil, nil
			}
			number, ok := toFloat(value)
			if !ok {
				return nil, fmt.Errorf("unary minus expects numeric input")
			}
			return -number, nil
		default:
			return nil, fmt.Errorf("unsupported unary operator %q", typed.op)
		}
	case binaryNode:
		left, err := s.eval(typed.left)
		if err != nil {
			return nil, err
		}
		right, err := s.eval(typed.right)
		if err != nil {
			return nil, err
		}
		return evaluateBinary(typed.op, left, right)
	case functionNode:
		fn, ok := builtins()[typed.name]
		if !ok {
			return nil, fmt.Errorf("unknown function %s", typed.name)
		}
		args := make([]any, 0, len(typed.args))
		for _, arg := range typed.args {
			value, err := s.eval(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}
		return fn(args)
	default:
		return nil, fmt.Errorf("unsupported expression node")
	}
}

func evaluateBinary(op string, left, right any) (any, error) {
	switch op {
	case "AND", "&&":
		if left == nil || right == nil {
			return nil, nil
		}
		l, lok := toBool(left)
		r, rok := toBool(right)
		if !lok || !rok {
			return nil, fmt.Errorf("AND expects boolean operands")
		}
		return l && r, nil
	case "OR", "||":
		if left == nil || right == nil {
			return nil, nil
		}
		l, lok := toBool(left)
		r, rok := toBool(right)
		if !lok || !rok {
			return nil, fmt.Errorf("OR expects boolean operands")
		}
		return l || r, nil
	case "+":
		if left == nil || right == nil {
			return nil, nil
		}
		if l, lok := toFloat(left); lok {
			if r, rok := toFloat(right); rok {
				return l + r, nil
			}
		}
		value := toString(left) + toString(right)
		if len(value) > 10000 {
			return nil, fmt.Errorf("expression output exceeds maximum string length")
		}
		return value, nil
	case "-":
		return numericOp(left, right, func(a, b float64) float64 { return a - b })
	case "*":
		return numericOp(left, right, func(a, b float64) float64 { return a * b })
	case "/":
		if left == nil || right == nil {
			return nil, nil
		}
		l, lok := toFloat(left)
		r, rok := toFloat(right)
		if !lok || !rok {
			return nil, fmt.Errorf("division expects numeric operands")
		}
		if r == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return l / r, nil
	case "==":
		if left == nil || right == nil {
			return nil, nil
		}
		return compare(left, right) == 0, nil
	case "!=":
		if left == nil || right == nil {
			return nil, nil
		}
		return compare(left, right) != 0, nil
	case ">":
		if left == nil || right == nil {
			return nil, nil
		}
		return compare(left, right) > 0, nil
	case ">=":
		if left == nil || right == nil {
			return nil, nil
		}
		return compare(left, right) >= 0, nil
	case "<":
		if left == nil || right == nil {
			return nil, nil
		}
		return compare(left, right) < 0, nil
	case "<=":
		if left == nil || right == nil {
			return nil, nil
		}
		return compare(left, right) <= 0, nil
	case "LIKE":
		if left == nil || right == nil {
			return nil, nil
		}
		return like(toString(left), toString(right)), nil
	default:
		return nil, fmt.Errorf("unsupported operator %s", op)
	}
}

func numericOp(left, right any, fn func(a, b float64) float64) (any, error) {
	if left == nil || right == nil {
		return nil, nil
	}
	l, lok := toFloat(left)
	r, rok := toFloat(right)
	if !lok || !rok {
		return nil, fmt.Errorf("operator expects numeric operands")
	}
	return fn(l, r), nil
}

func compare(left, right any) int {
	if lt, ok := toTime(left); ok {
		if rt, ok := toTime(right); ok {
			switch {
			case lt.Before(rt):
				return -1
			case lt.After(rt):
				return 1
			default:
				return 0
			}
		}
	}
	if lf, ok := toFloat(left); ok {
		if rf, ok := toFloat(right); ok {
			switch {
			case math.Abs(lf-rf) < 1e-9:
				return 0
			case lf < rf:
				return -1
			default:
				return 1
			}
		}
	}
	ls := toString(left)
	rs := toString(right)
	switch {
	case ls < rs:
		return -1
	case ls > rs:
		return 1
	default:
		return 0
	}
}

func like(value, pattern string) bool {
	replaced := regexp.QuoteMeta(pattern)
	replaced = strings.ReplaceAll(replaced, "%", ".*")
	replaced = strings.ReplaceAll(replaced, "_", ".")
	matched, _ := regexp.MatchString("^"+replaced+"$", value)
	return matched
}

func normalizeVars(vars map[string]interface{}) map[string]any {
	normalized := make(map[string]any, len(vars))
	for key, value := range vars {
		normalized[strings.ToLower(key)] = value
	}
	return normalized
}
