package expression

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenIdentifier
	tokenNumber
	tokenString
	tokenBoolean
	tokenNull
	tokenLeftParen
	tokenRightParen
	tokenComma
	tokenOperator
)

type token struct {
	typ tokenType
	val string
	pos int
}

type ExprNode interface{}

type binaryNode struct {
	op    string
	left  ExprNode
	right ExprNode
}

type unaryNode struct {
	op   string
	expr ExprNode
}

type literalNode struct {
	value any
}

type variableNode struct {
	name string
}

type functionNode struct {
	name string
	args []ExprNode
}

type parser struct {
	tokens []token
	pos    int
	depth  int
}

func tokenize(input string) ([]token, error) {
	tokens := make([]token, 0)
	for i := 0; i < len(input); {
		ch := rune(input[i])
		if unicode.IsSpace(ch) {
			i++
			continue
		}
		switch ch {
		case '(':
			tokens = append(tokens, token{typ: tokenLeftParen, val: "(", pos: i})
			i++
		case ')':
			tokens = append(tokens, token{typ: tokenRightParen, val: ")", pos: i})
			i++
		case ',':
			tokens = append(tokens, token{typ: tokenComma, val: ",", pos: i})
			i++
		case '\'', '"':
			quote := ch
			start := i
			i++
			var b strings.Builder
			for i < len(input) {
				current := rune(input[i])
				if current == quote {
					i++
					break
				}
				if current == '\\' && i+1 < len(input) {
					b.WriteByte(input[i+1])
					i += 2
					continue
				}
				b.WriteByte(input[i])
				i++
			}
			tokens = append(tokens, token{typ: tokenString, val: b.String(), pos: start})
		default:
			if unicode.IsDigit(ch) {
				start := i
				dotSeen := false
				for i < len(input) {
					current := rune(input[i])
					if current == '.' && !dotSeen {
						dotSeen = true
						i++
						continue
					}
					if !unicode.IsDigit(current) {
						break
					}
					i++
				}
				tokens = append(tokens, token{typ: tokenNumber, val: input[start:i], pos: start})
				continue
			}
			if unicode.IsLetter(ch) || ch == '_' {
				start := i
				i++
				for i < len(input) {
					current := rune(input[i])
					if !(unicode.IsLetter(current) || unicode.IsDigit(current) || current == '_' || current == '.') {
						break
					}
					i++
				}
				word := input[start:i]
				upper := strings.ToUpper(word)
				switch upper {
				case "TRUE", "FALSE":
					tokens = append(tokens, token{typ: tokenBoolean, val: strings.ToLower(word), pos: start})
				case "NULL":
					tokens = append(tokens, token{typ: tokenNull, val: "null", pos: start})
				case "AND", "OR", "NOT", "LIKE":
					tokens = append(tokens, token{typ: tokenOperator, val: upper, pos: start})
				default:
					tokens = append(tokens, token{typ: tokenIdentifier, val: word, pos: start})
				}
				continue
			}
			if i+1 < len(input) {
				pair := input[i : i+2]
				switch pair {
				case "==", "!=", ">=", "<=", "&&", "||":
					tokens = append(tokens, token{typ: tokenOperator, val: pair, pos: i})
					i += 2
					continue
				}
			}
			switch ch {
			case '+', '-', '*', '/', '>', '<', '!':
				tokens = append(tokens, token{typ: tokenOperator, val: string(ch), pos: i})
				i++
			default:
				return nil, fmt.Errorf("unexpected character %q at position %d", ch, i)
			}
		}
	}
	tokens = append(tokens, token{typ: tokenEOF, pos: len(input)})
	return tokens, nil
}

func newParser(tokens []token) *parser {
	return &parser{tokens: tokens}
}

func (p *parser) parse() (ExprNode, error) {
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.current().typ != tokenEOF {
		return nil, fmt.Errorf("unexpected token %q at position %d", p.current().val, p.current().pos)
	}
	return node, nil
}

func (p *parser) parseOr() (ExprNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.matchOperator("OR", "||") {
		op := p.previous().val
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: strings.ToUpper(op), left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (ExprNode, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.matchOperator("AND", "&&") {
		op := p.previous().val
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: strings.ToUpper(op), left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseComparison() (ExprNode, error) {
	left, err := p.parseAddition()
	if err != nil {
		return nil, err
	}
	for p.matchOperator("==", "!=", ">", ">=", "<", "<=", "LIKE") {
		op := p.previous().val
		right, err := p.parseAddition()
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: strings.ToUpper(op), left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAddition() (ExprNode, error) {
	left, err := p.parseMultiplication()
	if err != nil {
		return nil, err
	}
	for p.matchOperator("+", "-") {
		op := p.previous().val
		right, err := p.parseMultiplication()
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseMultiplication() (ExprNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.matchOperator("*", "/") {
		op := p.previous().val
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (ExprNode, error) {
	if p.matchOperator("NOT", "!", "-") {
		op := strings.ToUpper(p.previous().val)
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unaryNode{op: op, expr: expr}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (ExprNode, error) {
	if p.match(tokenNumber) {
		value, err := strconv.ParseFloat(p.previous().val, 64)
		if err != nil {
			return nil, fmt.Errorf("parse number at position %d: %w", p.previous().pos, err)
		}
		return literalNode{value: value}, nil
	}
	if p.match(tokenString) {
		return literalNode{value: p.previous().val}, nil
	}
	if p.match(tokenBoolean) {
		return literalNode{value: p.previous().val == "true"}, nil
	}
	if p.match(tokenNull) {
		return literalNode{value: nil}, nil
	}
	if p.match(tokenIdentifier) {
		name := p.previous().val
		if p.match(tokenLeftParen) {
			args := make([]ExprNode, 0)
			if !p.check(tokenRightParen) {
				for {
					arg, err := p.parseOr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if !p.match(tokenComma) {
						break
					}
				}
			}
			if !p.match(tokenRightParen) {
				return nil, fmt.Errorf("expected ')' after function arguments at position %d", p.current().pos)
			}
			return functionNode{name: strings.ToUpper(name), args: args}, nil
		}
		return variableNode{name: name}, nil
	}
	if p.match(tokenLeftParen) {
		p.depth++
		if p.depth > 50 {
			return nil, fmt.Errorf("expression exceeds maximum nesting depth")
		}
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if !p.match(tokenRightParen) {
			return nil, fmt.Errorf("expected ')' at position %d", p.current().pos)
		}
		p.depth--
		return expr, nil
	}
	return nil, fmt.Errorf("unexpected token %q at position %d", p.current().val, p.current().pos)
}

func (p *parser) match(expected tokenType) bool {
	if p.check(expected) {
		p.pos++
		return true
	}
	return false
}

func (p *parser) matchOperator(values ...string) bool {
	if p.current().typ != tokenOperator {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(p.current().val, value) {
			p.pos++
			return true
		}
	}
	return false
}

func (p *parser) check(expected tokenType) bool {
	return p.current().typ == expected
}

func (p *parser) current() token {
	return p.tokens[p.pos]
}

func (p *parser) previous() token {
	return p.tokens[p.pos-1]
}

