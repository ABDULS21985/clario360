package expression

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Evaluator evaluates boolean expressions against a data context.
// It uses a recursive descent parser to safely evaluate workflow transition
// conditions without any access to the Go runtime.
type Evaluator struct {
	maxLength int // max expression length
	maxDepth  int // max nesting depth
}

// NewEvaluator creates a new Evaluator with safe defaults.
func NewEvaluator() *Evaluator {
	return &Evaluator{
		maxLength: 1000,
		maxDepth:  10,
	}
}

// Evaluate parses and evaluates an expression against provided data.
// data is a map structured as:
//
//	{"variables": {...}, "steps": {"stepId": {"output": {...}}}, "trigger": {"data": {...}}}
//
// Returns true/false based on the expression evaluation, or an error if
// the expression is invalid or evaluation fails.
func (e *Evaluator) Evaluate(expression string, data map[string]interface{}) (bool, error) {
	if expression == "" {
		return false, fmt.Errorf("empty expression")
	}
	if len(expression) > e.maxLength {
		return false, fmt.Errorf("expression exceeds maximum length of %d characters", e.maxLength)
	}

	tokens, err := tokenize(expression)
	if err != nil {
		return false, fmt.Errorf("tokenize error: %w", err)
	}

	parser := &parser{
		tokens:   tokens,
		pos:      0,
		maxDepth: e.maxDepth,
	}

	node, err := parser.parseExpr(0)
	if err != nil {
		return false, fmt.Errorf("parse error: %w", err)
	}

	if parser.pos < len(parser.tokens) {
		return false, fmt.Errorf("unexpected token at position %d: %q", parser.pos, parser.tokens[parser.pos].value)
	}

	result, err := evalNode(node, data)
	if err != nil {
		return false, fmt.Errorf("eval error: %w", err)
	}

	return toBool(result), nil
}

// ---------- Token types ----------

type tokenKind int

const (
	tkString  tokenKind = iota // single-quoted string literal
	tkNumber                   // integer or float
	tkBool                     // true / false
	tkNull                     // null
	tkIdent                    // identifier (part of a dotted path)
	tkDot                      // .
	tkEq                       // ==
	tkNe                       // !=
	tkGt                       // >
	tkGe                       // >=
	tkLt                       // <
	tkLe                       // <=
	tkAnd                      // &&
	tkOr                       // ||
	tkNot                      // !
	tkIn                       // in
	tkLParen                   // (
	tkRParen                   // )
	tkLBrack                   // [
	tkRBrack                   // ]
	tkComma                    // ,
)

type token struct {
	kind  tokenKind
	value string
}

// ---------- Tokenizer ----------

func tokenize(expr string) ([]token, error) {
	var tokens []token
	i := 0
	runes := []rune(expr)
	n := len(runes)

	for i < n {
		ch := runes[i]

		// skip whitespace
		if unicode.IsSpace(ch) {
			i++
			continue
		}

		// single-quoted string
		if ch == '\'' {
			j := i + 1
			for j < n && runes[j] != '\'' {
				if runes[j] == '\\' {
					j++ // skip escaped char
				}
				j++
			}
			if j >= n {
				return nil, fmt.Errorf("unterminated string literal starting at position %d", i)
			}
			val := string(runes[i+1 : j])
			tokens = append(tokens, token{kind: tkString, value: val})
			i = j + 1
			continue
		}

		// two-character operators
		if i+1 < n {
			two := string(runes[i : i+2])
			switch two {
			case "==":
				tokens = append(tokens, token{kind: tkEq, value: two})
				i += 2
				continue
			case "!=":
				tokens = append(tokens, token{kind: tkNe, value: two})
				i += 2
				continue
			case ">=":
				tokens = append(tokens, token{kind: tkGe, value: two})
				i += 2
				continue
			case "<=":
				tokens = append(tokens, token{kind: tkLe, value: two})
				i += 2
				continue
			case "&&":
				tokens = append(tokens, token{kind: tkAnd, value: two})
				i += 2
				continue
			case "||":
				tokens = append(tokens, token{kind: tkOr, value: two})
				i += 2
				continue
			}
		}

		// single-character operators / punctuation
		switch ch {
		case '>':
			tokens = append(tokens, token{kind: tkGt, value: ">"})
			i++
			continue
		case '<':
			tokens = append(tokens, token{kind: tkLt, value: "<"})
			i++
			continue
		case '!':
			tokens = append(tokens, token{kind: tkNot, value: "!"})
			i++
			continue
		case '(':
			tokens = append(tokens, token{kind: tkLParen, value: "("})
			i++
			continue
		case ')':
			tokens = append(tokens, token{kind: tkRParen, value: ")"})
			i++
			continue
		case '[':
			tokens = append(tokens, token{kind: tkLBrack, value: "["})
			i++
			continue
		case ']':
			tokens = append(tokens, token{kind: tkRBrack, value: "]"})
			i++
			continue
		case ',':
			tokens = append(tokens, token{kind: tkComma, value: ","})
			i++
			continue
		case '.':
			tokens = append(tokens, token{kind: tkDot, value: "."})
			i++
			continue
		}

		// numbers (integers and floats, possibly negative when leading)
		if unicode.IsDigit(ch) || (ch == '-' && i+1 < n && unicode.IsDigit(runes[i+1])) {
			j := i
			if ch == '-' {
				j++
			}
			for j < n && (unicode.IsDigit(runes[j]) || runes[j] == '.') {
				j++
			}
			tokens = append(tokens, token{kind: tkNumber, value: string(runes[i:j])})
			i = j
			continue
		}

		// identifiers and keywords (true, false, null, in)
		if unicode.IsLetter(ch) || ch == '_' {
			j := i
			for j < n && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
				j++
			}
			word := string(runes[i:j])
			switch word {
			case "true", "false":
				tokens = append(tokens, token{kind: tkBool, value: word})
			case "null":
				tokens = append(tokens, token{kind: tkNull, value: word})
			case "in":
				tokens = append(tokens, token{kind: tkIn, value: word})
			default:
				tokens = append(tokens, token{kind: tkIdent, value: word})
			}
			i = j
			continue
		}

		return nil, fmt.Errorf("unexpected character %q at position %d", string(ch), i)
	}

	return tokens, nil
}

// ---------- AST node types ----------

type nodeKind int

const (
	ndLiteral nodeKind = iota // literal value (string, number, bool, nil)
	ndPath                    // dotted path reference
	ndArray                   // array literal [a, b, c]
	ndBinaryOp                // binary op: ==, !=, >, <, >=, <=, in, &&, ||
	ndUnaryOp                 // unary op: !
)

type astNode struct {
	kind     nodeKind
	value    interface{}   // for ndLiteral
	segments []string      // for ndPath: ["steps", "triage", "output", "is_valid"]
	op       string        // for ndBinaryOp / ndUnaryOp
	left     *astNode      // for ndBinaryOp, ndUnaryOp (operand)
	right    *astNode      // for ndBinaryOp
	elements []*astNode    // for ndArray
}

// ---------- Parser ----------

type parser struct {
	tokens   []token
	pos      int
	maxDepth int
}

func (p *parser) peek() *token {
	if p.pos >= len(p.tokens) {
		return nil
	}
	return &p.tokens[p.pos]
}

func (p *parser) advance() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) expect(kind tokenKind) (token, error) {
	t := p.peek()
	if t == nil {
		return token{}, fmt.Errorf("unexpected end of expression, expected token kind %d", kind)
	}
	if t.kind != kind {
		return token{}, fmt.Errorf("expected token kind %d but got %q", kind, t.value)
	}
	return p.advance(), nil
}

// parseExpr is the entry point: expr -> or_expr
func (p *parser) parseExpr(depth int) (*astNode, error) {
	if depth > p.maxDepth {
		return nil, fmt.Errorf("maximum nesting depth of %d exceeded", p.maxDepth)
	}
	return p.parseOr(depth)
}

// or_expr -> and_expr ( "||" and_expr )*
func (p *parser) parseOr(depth int) (*astNode, error) {
	left, err := p.parseAnd(depth)
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t == nil || t.kind != tkOr {
			break
		}
		p.advance()
		right, err := p.parseAnd(depth)
		if err != nil {
			return nil, err
		}
		left = &astNode{kind: ndBinaryOp, op: "||", left: left, right: right}
	}
	return left, nil
}

// and_expr -> cmp_expr ( "&&" cmp_expr )*
func (p *parser) parseAnd(depth int) (*astNode, error) {
	left, err := p.parseCmp(depth)
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t == nil || t.kind != tkAnd {
			break
		}
		p.advance()
		right, err := p.parseCmp(depth)
		if err != nil {
			return nil, err
		}
		left = &astNode{kind: ndBinaryOp, op: "&&", left: left, right: right}
	}
	return left, nil
}

// cmp_expr -> value ( ("==" | "!=" | ">" | "<" | ">=" | "<=" | "in") value )?
func (p *parser) parseCmp(depth int) (*astNode, error) {
	left, err := p.parseValue(depth)
	if err != nil {
		return nil, err
	}
	t := p.peek()
	if t == nil {
		return left, nil
	}
	switch t.kind {
	case tkEq, tkNe, tkGt, tkGe, tkLt, tkLe, tkIn:
		op := p.advance()
		right, err := p.parseValue(depth)
		if err != nil {
			return nil, err
		}
		return &astNode{kind: ndBinaryOp, op: op.value, left: left, right: right}, nil
	}
	return left, nil
}

// value -> "!" value | "(" expr ")" | path | literal | array
func (p *parser) parseValue(depth int) (*astNode, error) {
	t := p.peek()
	if t == nil {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	switch t.kind {
	case tkNot:
		p.advance()
		operand, err := p.parseValue(depth + 1)
		if err != nil {
			return nil, err
		}
		return &astNode{kind: ndUnaryOp, op: "!", left: operand}, nil

	case tkLParen:
		p.advance()
		inner, err := p.parseExpr(depth + 1)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tkRParen); err != nil {
			return nil, fmt.Errorf("missing closing parenthesis")
		}
		return inner, nil

	case tkLBrack:
		return p.parseArray(depth)

	case tkString:
		tok := p.advance()
		return &astNode{kind: ndLiteral, value: tok.value}, nil

	case tkNumber:
		tok := p.advance()
		if strings.Contains(tok.value, ".") {
			f, err := strconv.ParseFloat(tok.value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", tok.value)
			}
			return &astNode{kind: ndLiteral, value: f}, nil
		}
		n, err := strconv.ParseInt(tok.value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", tok.value)
		}
		return &astNode{kind: ndLiteral, value: n}, nil

	case tkBool:
		tok := p.advance()
		return &astNode{kind: ndLiteral, value: tok.value == "true"}, nil

	case tkNull:
		p.advance()
		return &astNode{kind: ndLiteral, value: nil}, nil

	case tkIdent:
		return p.parsePath()

	default:
		return nil, fmt.Errorf("unexpected token: %q", t.value)
	}
}

// parsePath -> identifier ("." identifier)*
func (p *parser) parsePath() (*astNode, error) {
	tok, err := p.expect(tkIdent)
	if err != nil {
		return nil, err
	}
	segments := []string{tok.value}
	for {
		t := p.peek()
		if t == nil || t.kind != tkDot {
			break
		}
		p.advance() // consume dot
		ident, err := p.expect(tkIdent)
		if err != nil {
			return nil, fmt.Errorf("expected identifier after '.'")
		}
		segments = append(segments, ident.value)
	}
	return &astNode{kind: ndPath, segments: segments}, nil
}

// parseArray -> "[" literal ("," literal)* "]"
func (p *parser) parseArray(depth int) (*astNode, error) {
	if _, err := p.expect(tkLBrack); err != nil {
		return nil, err
	}

	var elements []*astNode

	// handle empty array
	t := p.peek()
	if t != nil && t.kind == tkRBrack {
		p.advance()
		return &astNode{kind: ndArray, elements: elements}, nil
	}

	for {
		elem, err := p.parseValue(depth + 1)
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)

		t := p.peek()
		if t == nil {
			return nil, fmt.Errorf("unterminated array literal")
		}
		if t.kind == tkRBrack {
			p.advance()
			break
		}
		if t.kind != tkComma {
			return nil, fmt.Errorf("expected ',' or ']' in array, got %q", t.value)
		}
		p.advance() // consume comma
	}

	return &astNode{kind: ndArray, elements: elements}, nil
}

// ---------- Evaluator ----------

func evalNode(node *astNode, data map[string]interface{}) (interface{}, error) {
	switch node.kind {
	case ndLiteral:
		return node.value, nil

	case ndPath:
		return resolvePath(node.segments, data)

	case ndArray:
		var result []interface{}
		for _, elem := range node.elements {
			val, err := evalNode(elem, data)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
		return result, nil

	case ndUnaryOp:
		if node.op == "!" {
			operand, err := evalNode(node.left, data)
			if err != nil {
				return nil, err
			}
			return !toBool(operand), nil
		}
		return nil, fmt.Errorf("unknown unary operator: %s", node.op)

	case ndBinaryOp:
		return evalBinaryOp(node, data)

	default:
		return nil, fmt.Errorf("unknown node kind: %d", node.kind)
	}
}

func evalBinaryOp(node *astNode, data map[string]interface{}) (interface{}, error) {
	// short-circuit for && and ||
	if node.op == "&&" {
		leftVal, err := evalNode(node.left, data)
		if err != nil {
			return nil, err
		}
		if !toBool(leftVal) {
			return false, nil
		}
		rightVal, err := evalNode(node.right, data)
		if err != nil {
			return nil, err
		}
		return toBool(rightVal), nil
	}
	if node.op == "||" {
		leftVal, err := evalNode(node.left, data)
		if err != nil {
			return nil, err
		}
		if toBool(leftVal) {
			return true, nil
		}
		rightVal, err := evalNode(node.right, data)
		if err != nil {
			return nil, err
		}
		return toBool(rightVal), nil
	}

	leftVal, err := evalNode(node.left, data)
	if err != nil {
		return nil, err
	}
	rightVal, err := evalNode(node.right, data)
	if err != nil {
		return nil, err
	}

	switch node.op {
	case "==":
		return compareEqual(leftVal, rightVal), nil
	case "!=":
		return !compareEqual(leftVal, rightVal), nil
	case ">":
		return compareOrdered(leftVal, rightVal, ">")
	case ">=":
		return compareOrdered(leftVal, rightVal, ">=")
	case "<":
		return compareOrdered(leftVal, rightVal, "<")
	case "<=":
		return compareOrdered(leftVal, rightVal, "<=")
	case "in":
		return evalIn(leftVal, rightVal)
	default:
		return nil, fmt.Errorf("unknown operator: %s", node.op)
	}
}

// resolvePath walks the data map using the dotted path segments.
func resolvePath(segments []string, data map[string]interface{}) (interface{}, error) {
	var current interface{} = data
	for _, seg := range segments {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot resolve path segment %q: not a map", seg)
		}
		val, exists := m[seg]
		if !exists {
			return nil, fmt.Errorf("path segment %q not found", seg)
		}
		current = val
	}
	return current, nil
}

// toBool converts a value to a boolean for logical evaluation.
func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case int64:
		return val != 0
	case int:
		return val != 0
	case float64:
		return val != 0
	default:
		return true
	}
}

// compareEqual does a deep equality comparison, coercing numeric types.
func compareEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// normalize numbers to float64 for comparison
	af, aIsNum := toFloat64(a)
	bf, bIsNum := toFloat64(b)
	if aIsNum && bIsNum {
		return af == bf
	}

	// compare booleans
	ab, aIsBool := a.(bool)
	bb, bIsBool := b.(bool)
	if aIsBool && bIsBool {
		return ab == bb
	}

	// compare strings
	as, aIsStr := a.(string)
	bs, bIsStr := b.(string)
	if aIsStr && bIsStr {
		return as == bs
	}

	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// compareOrdered compares two numeric values with the given operator.
func compareOrdered(a, b interface{}, op string) (bool, error) {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk {
		return false, fmt.Errorf("cannot compare non-numeric values with %s", op)
	}
	switch op {
	case ">":
		return af > bf, nil
	case ">=":
		return af >= bf, nil
	case "<":
		return af < bf, nil
	case "<=":
		return af <= bf, nil
	default:
		return false, fmt.Errorf("unknown comparison operator: %s", op)
	}
}

// evalIn checks if leftVal is contained in rightVal (which must be a slice).
func evalIn(leftVal, rightVal interface{}) (bool, error) {
	arr, ok := rightVal.([]interface{})
	if !ok {
		return false, fmt.Errorf("right-hand side of 'in' must be an array")
	}
	for _, elem := range arr {
		if compareEqual(leftVal, elem) {
			return true, nil
		}
	}
	return false, nil
}

// toFloat64 attempts to convert a value to float64.
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int64:
		return float64(val), true
	case int:
		return float64(val), true
	case float32:
		return float64(val), true
	default:
		return 0, false
	}
}
