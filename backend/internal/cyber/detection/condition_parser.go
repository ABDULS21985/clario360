package detection

import (
	"fmt"
	"strings"
	"unicode"
)

// BoolExpr is an evaluable boolean expression over named selections.
type BoolExpr interface {
	Evaluate(matches map[string]bool) bool
}

// AndExpr evaluates to true when both operands are true.
type AndExpr struct {
	Left, Right BoolExpr
}

// Evaluate implements BoolExpr.
func (e *AndExpr) Evaluate(matches map[string]bool) bool {
	return e.Left.Evaluate(matches) && e.Right.Evaluate(matches)
}

// OrExpr evaluates to true when either operand is true.
type OrExpr struct {
	Left, Right BoolExpr
}

// Evaluate implements BoolExpr.
func (e *OrExpr) Evaluate(matches map[string]bool) bool {
	return e.Left.Evaluate(matches) || e.Right.Evaluate(matches)
}

// NotExpr negates its operand.
type NotExpr struct {
	Operand BoolExpr
}

// Evaluate implements BoolExpr.
func (e *NotExpr) Evaluate(matches map[string]bool) bool {
	return !e.Operand.Evaluate(matches)
}

// IdentExpr resolves a named selection or filter.
type IdentExpr struct {
	Name string
}

// Evaluate implements BoolExpr.
func (e *IdentExpr) Evaluate(matches map[string]bool) bool {
	return matches[e.Name]
}

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenIdent
	tokenAnd
	tokenOr
	tokenNot
	tokenLParen
	tokenRParen
)

type token struct {
	typ   tokenType
	value string
}

type conditionParser struct {
	tokens []token
	pos    int
}

// ParseCondition parses a boolean condition such as "A and (B or not C)".
func ParseCondition(condition string) (BoolExpr, error) {
	trimmed := strings.TrimSpace(condition)
	if trimmed == "" {
		return nil, fmt.Errorf("condition cannot be empty")
	}
	tokens, err := tokenizeCondition(trimmed)
	if err != nil {
		return nil, err
	}
	p := &conditionParser{tokens: tokens}
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().typ != tokenEOF {
		return nil, fmt.Errorf("unexpected token %q", p.peek().value)
	}
	return expr, nil
}

func tokenizeCondition(input string) ([]token, error) {
	tokens := make([]token, 0, len(input)/2)
	for i := 0; i < len(input); {
		switch ch := rune(input[i]); {
		case unicode.IsSpace(ch):
			i++
		case ch == '(':
			tokens = append(tokens, token{typ: tokenLParen, value: "("})
			i++
		case ch == ')':
			tokens = append(tokens, token{typ: tokenRParen, value: ")"})
			i++
		case unicode.IsLetter(ch) || ch == '_':
			start := i
			i++
			for i < len(input) {
				r := rune(input[i])
				if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
					i++
					continue
				}
				break
			}
			word := input[start:i]
			switch strings.ToLower(word) {
			case "and":
				tokens = append(tokens, token{typ: tokenAnd, value: word})
			case "or":
				tokens = append(tokens, token{typ: tokenOr, value: word})
			case "not":
				tokens = append(tokens, token{typ: tokenNot, value: word})
			default:
				tokens = append(tokens, token{typ: tokenIdent, value: word})
			}
		default:
			return nil, fmt.Errorf("invalid character %q in condition", string(ch))
		}
	}
	tokens = append(tokens, token{typ: tokenEOF})
	return tokens, nil
}

func (p *conditionParser) parseExpr() (BoolExpr, error) {
	return p.parseOr()
}

func (p *conditionParser) parseOr() (BoolExpr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().typ == tokenOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &OrExpr{Left: left, Right: right}
	}
	return left, nil
}

func (p *conditionParser) parseAnd() (BoolExpr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().typ == tokenAnd {
		p.next()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &AndExpr{Left: left, Right: right}
	}
	return left, nil
}

func (p *conditionParser) parseNot() (BoolExpr, error) {
	if p.peek().typ == tokenNot {
		p.next()
		operand, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &NotExpr{Operand: operand}, nil
	}
	return p.parseAtom()
}

func (p *conditionParser) parseAtom() (BoolExpr, error) {
	switch tok := p.peek(); tok.typ {
	case tokenIdent:
		p.next()
		return &IdentExpr{Name: tok.value}, nil
	case tokenLParen:
		p.next()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.peek().typ != tokenRParen {
			return nil, fmt.Errorf("expected closing parenthesis")
		}
		p.next()
		return expr, nil
	default:
		return nil, fmt.Errorf("unexpected token %q", tok.value)
	}
}

func (p *conditionParser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *conditionParser) next() token {
	tok := p.peek()
	p.pos++
	return tok
}
